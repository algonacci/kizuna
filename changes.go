package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const previewLimit = 512 * 1024

func cloneRepository(ctx context.Context, url, destination string) error {
	if url == "" || strings.HasPrefix(url, "-") {
		return fmt.Errorf("Enter a repository URL or local source path")
	}
	if !filepath.IsAbs(destination) {
		return fmt.Errorf("Choose an absolute destination directory")
	}
	_, err := gitInput(ctx, os.TempDir(), "", false, "clone", "--", url, destination)
	return err
}

// Retain a bounded preview even when Git produces a very large patch.
type cappedOutput struct {
	buffer    bytes.Buffer
	truncated bool
}

func (b *cappedOutput) String() string { return b.buffer.String() }

func (b *cappedOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := previewLimit - b.buffer.Len()
	if len(p) > remaining {
		p = p[:remaining]
		b.truncated = true
	}
	_, err := b.buffer.Write(p)
	return n, err
}
func gitInput(ctx context.Context, dir, input string, diff bool, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"--no-pager", "--literal-pathspecs", "-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	cmd.Stdin = strings.NewReader(input)
	var out, stderr cappedOutput
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err != nil {
		e, ok := err.(*exec.ExitError)
		if !diff || !ok || e.ExitCode() != 1 {
			return "", fmt.Errorf("git %s: %v\n%s", args[0], err, strings.TrimSpace(stderr.String()))
		}
	}
	result := out.String()
	if out.truncated {
		result += "\n… Preview truncated at 512 KiB.\n"
	}
	return result, nil
}

func fileDiff(ctx context.Context, dir string, ch change) (string, error) {
	if ch.Code == "??" {
		path := filepath.Join(dir, ch.Path)
		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			return "+ symlink → " + target, err
		}
		if !info.Mode().IsRegular() {
			return "Preview unavailable for this file type.", nil
		}
		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, previewLimit+1))
		if err != nil {
			return "", err
		}
		if bytes.IndexByte(data, 0) >= 0 {
			return "Binary file — no text preview.", nil
		}
		truncated := len(data) > previewLimit
		if truncated {
			data = data[:previewLimit]
		}
		var result strings.Builder
		result.WriteString("New file: " + ch.Path + "\n")
		for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
			result.WriteString("+" + line + "\n")
		}
		if truncated {
			result.WriteString("… Preview truncated at 512 KiB.\n")
		}
		return result.String(), nil
	}
	base := "HEAD"
	if _, err := git(ctx, dir, "rev-parse", "--verify", "HEAD"); err != nil {
		empty, err := gitInput(ctx, dir, "", false, "hash-object", "-t", "tree", "--stdin")
		if err != nil {
			return "", err
		}
		base = strings.TrimSpace(empty)
	}
	args := []string{"diff", "--no-ext-diff", "--no-textconv", "--no-color", "--no-renames", base, "--", ch.Path}
	if ch.From != "" {
		args = append(args, ch.From)
	}
	result, err := gitInput(ctx, dir, "", false, args...)
	if err == nil && result == "" {
		result = "No net changes relative to HEAD (the index may contain a different staged version)."
	}
	return result, err
}

// Commit full selected files; --only leaves unrelated staged changes in the index.
func commitFiles(ctx context.Context, dir, summary, description string, changes []change) error {
	if strings.TrimSpace(summary) == "" {
		return fmt.Errorf("Enter a commit summary")
	}
	if len(changes) == 0 {
		return fmt.Errorf("Select at least one changed file")
	}
	conflicts, err := git(ctx, dir, "ls-files", "--unmerged")
	if err != nil {
		return err
	}
	if conflicts != "" {
		return fmt.Errorf("Resolve merge conflicts before committing")
	}
	for _, state := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD", "rebase-merge", "rebase-apply"} {
		path, e := git(ctx, dir, "rev-parse", "--git-path", state)
		if e != nil {
			return e
		}
		path = strings.TrimSuffix(path, "\n")
		if !filepath.IsAbs(path) {
			path = filepath.Join(dir, path)
		}
		if _, e = os.Stat(path); e == nil {
			return fmt.Errorf("Finish the active merge/rebase/cherry-pick in your terminal first")
		}
	}
	var paths []string
	seen := map[string]bool{}
	for _, ch := range changes {
		for _, p := range []string{ch.Path, ch.From} {
			if p != "" && !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		}
	}
	input := strings.Join(paths, "\x00") + "\x00"
	var stagePaths []string
	for _, p := range paths {
		_, statErr := os.Lstat(filepath.Join(dir, p))
		tracked, e := gitInput(ctx, dir, "", false, "ls-files", "-z", "--", p)
		if e != nil {
			return e
		}
		// A staged deletion/rename source is already absent from both disk and index.
		if statErr == nil || tracked != "" {
			stagePaths = append(stagePaths, p)
		}
	}
	if len(stagePaths) > 0 {
		if _, err = gitInput(ctx, dir, strings.Join(stagePaths, "\x00")+"\x00", false, "add", "-A", "--pathspec-from-file=-", "--pathspec-file-nul"); err != nil {
			return err
		}
	}
	message := strings.TrimSpace(summary)
	if strings.TrimSpace(description) != "" {
		message += "\n\n" + strings.TrimSpace(description)
	}
	_, err = gitInput(ctx, dir, input, false, "commit", "--only", "-m", message, "--pathspec-from-file=-", "--pathspec-file-nul")
	return err
}

func syncRemote(ctx context.Context, dir, action, remote string) error {
	if remote == "" || strings.HasPrefix(remote, "-") {
		return fmt.Errorf("Select a remote first")
	}
	remotes, err := git(ctx, dir, "remote")
	if err != nil {
		return err
	}
	found := false
	for _, r := range strings.Fields(remotes) {
		if r == remote {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("Remote %q is not configured", remote)
	}
	var args []string
	if action == "fetch" {
		args = []string{"fetch", remote}
	} else {
		branch, err := git(ctx, dir, "symbolic-ref", "--quiet", "--short", "HEAD")
		if err != nil {
			return fmt.Errorf("Switch to a branch before %s", action)
		}
		branch = strings.TrimSpace(branch)
		upstreamRemote, _ := git(ctx, dir, "config", "--get", "branch."+branch+".remote")
		mergeRef, _ := git(ctx, dir, "config", "--get", "branch."+branch+".merge")
		target := "refs/heads/" + branch
		if strings.TrimSpace(upstreamRemote) == remote && strings.HasPrefix(strings.TrimSpace(mergeRef), "refs/heads/") {
			target = strings.TrimSpace(mergeRef)
		}
		switch action {
		case "pull":
			args = []string{"-c", "pull.rebase=false", "pull", "--ff-only", "--no-autostash", remote, target}
		case "push":
			args = []string{"push", "--set-upstream", remote, "HEAD:" + target}
		default:
			return fmt.Errorf("Unknown remote action %q", action)
		}
	}
	_, err = gitInput(ctx, dir, "", false, args...)
	return err
}
