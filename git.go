package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type change struct{ Code, Path, From string }
type commit struct{ Hash, Subject string }
type worktree struct {
	Path, Head, Branch, LockReason, PruneReason string
	Main, Bare, Detached, Locked, Prunable      bool
	Changes                                     []change
	Commits                                     []commit
	Upstream                                    string
	Ahead, Behind                               int
	StatusError                                 string
}
type repository struct {
	Path     string
	Trees    []worktree
	Branches []string
}

// Arguments are always passed directly to Git; paths and refs never enter a shell.
func git(ctx context.Context, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	argv := append([]string{"--no-pager", "-C", dir}, args...)
	cmd := exec.CommandContext(ctx, "git", argv...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if e, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s", strings.TrimSpace(string(e.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}

func parseWorktrees(raw string) ([]worktree, error) {
	var trees []worktree
	var current *worktree
	for _, field := range strings.Split(raw, "\x00") {
		key, value, _ := strings.Cut(field, " ")
		if key == "worktree" {
			trees = append(trees, worktree{Path: value, Main: len(trees) == 0})
			current = &trees[len(trees)-1]
			continue
		}
		if current == nil {
			continue
		}
		switch key {
		case "HEAD":
			current.Head = value
		case "branch":
			current.Branch = strings.TrimPrefix(value, "refs/heads/")
		case "bare":
			current.Bare = true
		case "detached":
			current.Detached = true
		case "locked":
			current.Locked, current.LockReason = true, value
		case "prunable":
			current.Prunable, current.PruneReason = true, value
		}
	}
	if len(trees) == 0 {
		return nil, fmt.Errorf("Git returned no worktrees")
	}
	return trees, nil
}

func parseChanges(raw string) []change {
	var result []change
	fields := strings.Split(raw, "\x00")
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if len(f) < 4 {
			continue
		}
		ch := change{Code: f[:2], Path: f[3:]}
		if strings.ContainsAny(ch.Code, "RC") && i+1 < len(fields) {
			i++
			ch.From = fields[i]
		}
		result = append(result, ch)
	}
	return result
}

func loadRepository(ctx context.Context, path string) (*repository, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	absolute, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, err
	}
	raw, err := git(ctx, absolute, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return nil, err
	}
	trees, err := parseWorktrees(raw)
	if err != nil {
		return nil, err
	}
	// The common directory also works when opening a linked or bare worktree.
	common, err := git(ctx, absolute, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return nil, err
	}
	r := &repository{Path: strings.TrimSuffix(common, "\n"), Trees: trees}
	refs, err := git(ctx, r.Path, "for-each-ref", "--format=%(refname:short)", "refs/heads/")
	if err != nil {
		return nil, err
	}
	for _, ref := range strings.Split(strings.TrimSpace(refs), "\n") {
		if ref != "" {
			r.Branches = append(r.Branches, ref)
		}
	}
	for i := range r.Trees {
		w := &r.Trees[i]
		if w.Bare || w.Prunable {
			continue
		}
		status, e := git(ctx, w.Path, "status", "--porcelain=v1", "-z", "--untracked-files=normal")
		if e != nil {
			w.StatusError = e.Error()
			continue
		}
		w.Changes = parseChanges(status)
		history, e := git(ctx, w.Path, "log", "-8", "--format=%h%x00%s%x00")
		if e == nil {
			entries := strings.Split(history, "\x00")
			for j := 0; j+1 < len(entries); j += 2 {
				w.Commits = append(w.Commits, commit{strings.TrimSpace(entries[j]), entries[j+1]})
			}
		}
		upstream, e := git(ctx, w.Path, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
		if e == nil {
			w.Upstream = strings.TrimSpace(upstream)
			counts, e := git(ctx, w.Path, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
			if e == nil {
				nums := strings.Fields(counts)
				if len(nums) == 2 {
					w.Ahead, _ = strconv.Atoi(nums[0])
					w.Behind, _ = strconv.Atoi(nums[1])
				}
			}
		}
	}
	return r, nil
}

type createRequest struct{ Mode, Branch, Base, Path string }

func createWorktree(ctx context.Context, repo string, req createRequest) error {
	if !filepath.IsAbs(req.Path) {
		return fmt.Errorf("Choose an absolute destination path")
	}
	if req.Base == "" {
		req.Base = "HEAD"
	}
	if strings.HasPrefix(req.Base, "-") {
		return fmt.Errorf("Invalid starting ref")
	}
	args := []string{"worktree", "add"}
	switch req.Mode {
	case "New branch":
		if _, err := git(ctx, repo, "check-ref-format", "--branch", req.Branch); err != nil {
			return fmt.Errorf("Invalid branch name: %w", err)
		}
		args = append(args, "-b", req.Branch, "--", req.Path, req.Base)
	case "Existing branch":
		if _, err := git(ctx, repo, "check-ref-format", "--branch", req.Branch); err != nil {
			return fmt.Errorf("Invalid branch name: %w", err)
		}
		if _, err := git(ctx, repo, "show-ref", "--verify", "--", "refs/heads/"+req.Branch); err != nil {
			return fmt.Errorf("Choose an existing local branch")
		}
		args = append(args, "--", req.Path, req.Branch)
	case "Detached HEAD":
		args = append(args, "--detach", "--", req.Path, req.Base)
	default:
		return fmt.Errorf("Choose a worktree mode")
	}
	_, err := git(ctx, repo, args...)
	return err
}

func changeWorktree(ctx context.Context, repo, path, action string) error {
	raw, err := git(ctx, repo, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return err
	}
	trees, err := parseWorktrees(raw)
	if err != nil {
		return err
	}
	for _, w := range trees {
		if w.Path != path {
			continue
		}
		if w.Main || w.Bare {
			return fmt.Errorf("The primary worktree cannot be %s", action)
		}
		var args []string
		switch action {
		case "remove":
			args = []string{"worktree", "remove", "--", path}
		case "lock":
			args = []string{"worktree", "lock", "--reason", "Protected by Kizuna", "--", path}
		case "unlock":
			args = []string{"worktree", "unlock", "--", path}
		default:
			return fmt.Errorf("Unknown worktree action: %s", action)
		}
		_, err = git(ctx, repo, args...)
		return err
	}
	return fmt.Errorf("Worktree no longer exists; refresh the repository")
}
