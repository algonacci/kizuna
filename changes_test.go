package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}
func changesFor(t *testing.T, root string) []change {
	t.Helper()
	return mustLoad(t, root).Trees[0].Changes
}

func TestCommitSelectedPreservesUnselectedIndex(t *testing.T) {
	root := repoFixture(t)
	ctx := context.Background()
	writeFile(t, root, "hello.txt", "selected contents\n")
	writeFile(t, root, "other.txt", "staged but unchecked\n")
	testGit(t, root, "add", "other.txt")
	if err := commitFiles(ctx, root, "Selected only", "Detailed message", []change{{Code: " M", Path: "hello.txt"}}); err != nil {
		t.Fatal(err)
	}
	if got := testGit(t, root, "show", "--format=", "--name-only", "HEAD"); strings.TrimSpace(got) != "hello.txt" {
		t.Fatal("committed unchecked file", got)
	}
	if got := testGit(t, root, "diff", "--cached", "--name-only"); strings.TrimSpace(got) != "other.txt" {
		t.Fatal("lost unrelated staged change", got)
	}
	if got := testGit(t, root, "log", "-1", "--format=%B"); !strings.Contains(got, "Detailed message") {
		t.Fatal(got)
	}
	// Literal pathspecs must not interpret wildcard and :(...) filenames.
	writeFile(t, root, "[one].txt", "literal filename\n")
	writeFile(t, root, ":(glob)star*.txt", "literal magic\n")
	writeFile(t, root, "new\nfile.txt", "newline\n")
	selected := []change{{Code: "??", Path: "[one].txt"}, {Code: "??", Path: ":(glob)star*.txt"}, {Code: "??", Path: "new\nfile.txt"}}
	if err := commitFiles(ctx, root, "Literal names", "", selected); err != nil {
		t.Fatal(err)
	}
	for _, ch := range selected {
		testGit(t, root, "cat-file", "-e", "HEAD:"+ch.Path)
	}
	if got := testGit(t, root, "diff", "--cached", "--name-only"); strings.TrimSpace(got) != "other.txt" {
		t.Fatal(got)
	}
}

func TestCommitRenameDeletionAndInitialCommit(t *testing.T) {
	ctx := context.Background()
	root := repoFixture(t)
	testGit(t, root, "mv", "hello.txt", "renamed.txt")
	if err := commitFiles(ctx, root, "Rename", "", changesFor(t, root)); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "renamed.txt")); err != nil {
		t.Fatal(err)
	}
	preview, err := fileDiff(ctx, root, changesFor(t, root)[0])
	if err != nil || !strings.Contains(preview, "-hello") {
		t.Fatalf("deletion preview: %s %v", preview, err)
	}
	if err := commitFiles(ctx, root, "Delete", "", changesFor(t, root)); err != nil {
		t.Fatal(err)
	}
	empty := t.TempDir()
	testGit(t, empty, "init", "-b", "main")
	testGit(t, empty, "config", "user.name", "Test")
	testGit(t, empty, "config", "user.email", "test@example.invalid")
	testGit(t, empty, "config", "commit.gpgsign", "false")
	writeFile(t, empty, "first.txt", "first commit\n")
	testGit(t, empty, "add", "first.txt")
	changes := changesFor(t, empty)
	preview, err = fileDiff(ctx, empty, changes[0])
	if err != nil || !strings.Contains(preview, "+first commit") {
		t.Fatalf("unborn preview: %s %v", preview, err)
	}
	if err := commitFiles(ctx, empty, "Initial", "", changes); err != nil {
		t.Fatal(err)
	}
}

func TestDiffTrackedUntrackedBinaryAndBounds(t *testing.T) {
	ctx := context.Background()
	root := repoFixture(t)
	writeFile(t, root, "hello.txt", "modified\n")
	preview, err := fileDiff(ctx, root, change{Code: " M", Path: "hello.txt"})
	if err != nil || !strings.Contains(preview, "-hello") || !strings.Contains(preview, "+modified") {
		t.Fatalf("tracked: %s %v", preview, err)
	}
	writeFile(t, root, "nested/new.txt", "new content\n")
	preview, err = fileDiff(ctx, root, change{Code: "??", Path: "nested/new.txt"})
	if err != nil || !strings.Contains(preview, "+new content") {
		t.Fatalf("untracked: %s %v", preview, err)
	}
	writeFile(t, root, "binary.bin", "abc\x00def")
	preview, err = fileDiff(ctx, root, change{Code: "??", Path: "binary.bin"})
	if err != nil || !strings.Contains(preview, "Binary") {
		t.Fatal(preview, err)
	}
	writeFile(t, root, "large.txt", strings.Repeat("xxxxxxxxxxxxxxx\n", previewLimit/16+100))
	preview, err = fileDiff(ctx, root, change{Code: "??", Path: "large.txt"})
	if err != nil || !strings.Contains(preview, "truncated") {
		t.Fatal("unbounded preview", err)
	}
	testGit(t, root, "add", "large.txt")
	preview, err = fileDiff(ctx, root, change{Code: "A ", Path: "large.txt"})
	if err != nil || !strings.Contains(preview, "truncated") {
		t.Fatal("unbounded git diff", err)
	}
}

func TestCloneFetchPullPushLocalRemote(t *testing.T) {
	ctx := context.Background()
	source := repoFixture(t)
	remote := filepath.Join(t.TempDir(), "remote.git")
	testGit(t, source, "clone", "--bare", source, remote)
	first := filepath.Join(t.TempDir(), "first clone")
	second := filepath.Join(t.TempDir(), "second clone")
	for _, dest := range []string{first, second} {
		if err := cloneRepository(ctx, remote, dest); err != nil {
			t.Fatal(err)
		}
		testGit(t, dest, "config", "user.name", "Test")
		testGit(t, dest, "config", "user.email", "test@example.invalid")
		testGit(t, dest, "config", "commit.gpgsign", "false")
	}
	writeFile(t, first, "hello.txt", "pushed from first\n")
	if err := commitFiles(ctx, first, "Change from first", "", changesFor(t, first)); err != nil {
		t.Fatal(err)
	}
	if err := syncRemote(ctx, first, "push", "origin"); err != nil {
		t.Fatal(err)
	}
	if err := syncRemote(ctx, second, "fetch", "origin"); err != nil {
		t.Fatal(err)
	}
	if w := mustLoad(t, second).Trees[0]; w.Behind != 1 {
		t.Fatalf("fetch did not update behind: %+v", w)
	}
	if err := syncRemote(ctx, second, "pull", "origin"); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(second, "hello.txt")); string(data) != "pushed from first\n" {
		t.Fatalf("pull: %s", data)
	}
	// Publish a new branch, including upstream configuration.
	testGit(t, first, "switch", "-c", "feature/publish")
	if err := syncRemote(ctx, first, "push", "origin"); err != nil {
		t.Fatal(err)
	}
	if got := testGit(t, first, "rev-parse", "--abbrev-ref", "@{upstream}"); strings.TrimSpace(got) != "origin/feature/publish" {
		t.Fatal(got)
	}
	// Branch tracking a differently named remote branch must push to that target.
	testGit(t, first, "branch", "-m", "local-name")
	writeFile(t, first, "tracked.txt", "tracked differently\n")
	if err := commitFiles(ctx, first, "Different local name", "", changesFor(t, first)); err != nil {
		t.Fatal(err)
	}
	if err := syncRemote(ctx, first, "push", "origin"); err != nil {
		t.Fatal(err)
	}
	if got := testGit(t, first, "rev-parse", "HEAD"); got != testGit(t, remote, "rev-parse", "refs/heads/feature/publish") {
		t.Fatal("wrong push target")
	}
	// Non-fast-forward is surfaced, never force pushed or silently merged.
	testGit(t, first, "switch", "main")
	writeFile(t, first, "a.txt", "first diverges\n")
	testGit(t, first, "add", "a.txt")
	testGit(t, first, "commit", "-m", "First diverges")
	writeFile(t, second, "b.txt", "second diverges\n")
	testGit(t, second, "add", "b.txt")
	testGit(t, second, "commit", "-m", "Second diverges")
	if err := syncRemote(ctx, first, "push", "origin"); err != nil {
		t.Fatal(err)
	}
	if err := syncRemote(ctx, second, "push", "origin"); err == nil {
		t.Fatal("accepted non-fast-forward push")
	}
	before := testGit(t, second, "rev-parse", "HEAD")
	if err := syncRemote(ctx, second, "pull", "origin"); err == nil {
		t.Fatal("merged diverged branch")
	}
	if before != testGit(t, second, "rev-parse", "HEAD") {
		t.Fatal("failed pull changed HEAD")
	}
	if err := cloneRepository(ctx, remote, first); err == nil {
		t.Fatal("clone overwrote existing directory")
	}
}

func TestRepositoryIdentity(t *testing.T) {
	cases := map[string]string{
		"https://github.com/algonacci/kizuna.git":       "algonacci/kizuna",
		"git@github.com:Omniflow-id/erp.git":            "Omniflow-id/erp",
		"ssh://git@github.com/Omniflow-id/erp.git":      "Omniflow-id/erp",
		"https://gitlab.com/group/subgroup/project.git": "group/subgroup/project",
		"/local/path/repo.git":                          "", "file:///local/path/repo.git": "",
	}
	for raw, want := range cases {
		if got := remoteIdentity(raw); got != want {
			t.Fatalf("%s = %s, want %s", raw, got, want)
		}
	}
}
