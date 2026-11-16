package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func testGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "LC_ALL=C")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}
func repoFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "project with spaces")
	if err = os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	testGit(t, path, "init", "-b", "main")
	testGit(t, path, "config", "user.name", "Kizuna Test")
	testGit(t, path, "config", "user.email", "kizuna@example.invalid")
	testGit(t, path, "config", "commit.gpgsign", "false")
	testGit(t, path, "config", "core.hooksPath", filepath.Join(root, "no-hooks"))
	if err = os.WriteFile(filepath.Join(path, "hello.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	testGit(t, path, "add", ".")
	testGit(t, path, "commit", "-m", "Initial commit")
	return path
}
func mustLoad(t *testing.T, path string) *repository {
	t.Helper()
	r, err := loadRepository(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func TestWorktreeLifecycleRealGit(t *testing.T) {
	ctx := context.Background()
	path := repoFixture(t)
	r := mustLoad(t, path)
	if len(r.Trees) != 1 || r.Trees[0].Branch != "main" || len(r.Trees[0].Commits) != 1 {
		t.Fatalf("initial state: %+v", r)
	}
	// NUL output preserves paths which Git's line-based porcelain would quote.
	dest := filepath.Join(filepath.Dir(path), "feature with spaces\nand newline")
	if err := createWorktree(ctx, r.Path, createRequest{"New branch", "feature/one", "HEAD", dest}); err != nil {
		t.Fatal(err)
	}
	linked := mustLoad(t, dest)
	if linked.Path != r.Path || len(linked.Trees) != 2 || linked.Trees[1].Path != dest {
		t.Fatalf("linked repo: %+v", linked)
	}
	if err := changeWorktree(ctx, r.Path, dest, "lock"); err != nil {
		t.Fatal(err)
	}
	if !mustLoad(t, path).Trees[1].Locked {
		t.Fatal("lock missing")
	}
	if err := changeWorktree(ctx, r.Path, dest, "remove"); err == nil {
		t.Fatal("removed locked tree")
	}
	if err := changeWorktree(ctx, r.Path, dest, "unlock"); err != nil {
		t.Fatal(err)
	}
	dirty := filepath.Join(dest, "untracked\nfile.txt")
	if err := os.WriteFile(dirty, []byte("keep me"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := mustLoad(t, path).Trees[1].Changes; len(got) != 1 || got[0].Path != "untracked\nfile.txt" {
		t.Fatalf("changes: %+v", got)
	}
	if err := changeWorktree(ctx, r.Path, dest, "remove"); err == nil {
		t.Fatal("removed dirty tree")
	}
	if _, err := os.Stat(dirty); err != nil {
		t.Fatal("dirty data lost", err)
	}
	if err := os.Remove(dirty); err != nil {
		t.Fatal(err)
	}
	if err := changeWorktree(ctx, r.Path, dest, "remove"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("worktree not removed")
	}
	if !strings.Contains(testGit(t, path, "branch", "--list"), "feature/one") {
		t.Fatal("branch deleted on removal")
	}
	if err := changeWorktree(ctx, r.Path, path, "remove"); err == nil {
		t.Fatal("primary removal allowed")
	}
	// Reuse the retained branch, then verify checked-out branches are rejected.
	if err := createWorktree(ctx, r.Path, createRequest{"Existing branch", "feature/one", "", dest}); err != nil {
		t.Fatal(err)
	}
	if err := createWorktree(ctx, r.Path, createRequest{"Existing branch", "feature/one", "", dest + "-duplicate"}); err == nil {
		t.Fatal("duplicate checkout allowed")
	}
	detached := dest + "-detached"
	if err := createWorktree(ctx, r.Path, createRequest{"Detached HEAD", "", "HEAD", detached}); err != nil {
		t.Fatal(err)
	}
	if !mustLoad(t, detached).Trees[2].Detached {
		t.Fatal("not detached")
	}
}
func TestRenameStatusAndInvalidRequests(t *testing.T) {
	path := repoFixture(t)
	r := mustLoad(t, path)
	ctx := context.Background()
	testGit(t, path, "mv", "hello.txt", "renamed file.txt")
	changes := mustLoad(t, path).Trees[0].Changes
	if len(changes) != 1 || changes[0].Path != "renamed file.txt" || changes[0].From != "hello.txt" {
		t.Fatalf("rename: %+v", changes)
	}
	for _, req := range []createRequest{
		{"New branch", "--force", "HEAD", filepath.Join(t.TempDir(), "a")},
		{"New branch", "good", "--orphan", filepath.Join(t.TempDir(), "b")},
		{"Existing branch", "missing", "", filepath.Join(t.TempDir(), "c")},
		{"Detached HEAD", "", "HEAD", "relative/path"},
	} {
		if err := createWorktree(ctx, r.Path, req); err == nil {
			t.Fatalf("accepted invalid request: %+v", req)
		}
	}
}
func TestBareRepoAndPrunableTree(t *testing.T) {
	path := repoFixture(t)
	bare := filepath.Join(t.TempDir(), "bare.git")
	testGit(t, path, "clone", "--bare", path, bare)
	r := mustLoad(t, bare)
	if !r.Trees[0].Bare {
		t.Fatal("bare repo not recognized")
	}
	dest := filepath.Join(t.TempDir(), "linked")
	if err := createWorktree(context.Background(), r.Path, createRequest{"Existing branch", "main", "", dest}); err != nil {
		t.Fatal(err)
	}
	r = mustLoad(t, dest)
	if len(r.Trees) != 2 || r.Trees[1].Branch != "main" {
		t.Fatalf("bare linked: %+v", r)
	}
	if err := os.RemoveAll(dest); err != nil {
		t.Fatal(err)
	}
	if !mustLoad(t, bare).Trees[1].Prunable {
		t.Fatal("missing worktree not flagged")
	}
}
