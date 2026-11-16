package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func awaitDiff(t *testing.T, a *app, h *fakeHost) {
	t.Helper()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for a.diffLoading {
		select {
		case fn := <-h.queue:
			fn()
		case <-timer.C:
			t.Fatal("diff timed out")
		}
	}
}
func TestChangesSelectionCommitAndDiffUI(t *testing.T) {
	a, h := appFixture(t)
	root := repoFixture(t)
	writeFile(t, root, "hello.txt", "updated from UI\n")
	writeFile(t, root, "unchecked.txt", "keep me out\n")
	testGit(t, root, "add", "unchecked.txt")
	a.openRepository(root)
	await(t, a, h)
	awaitDiff(t, a, h)
	tt := ui.NewTester(a.view, 1180, 780)
	click(t, tt, "Change: hello.txt")
	awaitDiff(t, a, h)
	if !strings.Contains(a.diff, "+updated from UI") {
		t.Fatal(a.diff)
	}
	click(t, tt, "Include: unchecked.txt")
	if a.checked["unchecked.txt"] {
		t.Fatal("checkbox did not exclude file")
	}
	click(t, tt, "Commit summary")
	tt.Type("Commit from native UI")
	click(t, tt, "Commit selected files")
	await(t, a, h)
	awaitDiff(t, a, h)
	if a.errorMessage {
		t.Fatal(a.message)
	}
	if a.summary != "" || len(a.current().Changes) != 1 || a.current().Changes[0].Path != "unchecked.txt" {
		t.Fatalf("post commit: %+v", a.current())
	}
	if got := strings.TrimSpace(testGit(t, root, "diff", "--cached", "--name-only")); got != "unchecked.txt" {
		t.Fatal(got)
	}
	if a.checked["unchecked.txt"] {
		t.Fatal("refresh lost unchecked selection")
	}
	// Queued previews for a previous file must not overwrite the latest selection.
	writeFile(t, root, "hello.txt", "one more edit\n")
	a.refresh()
	await(t, a, h)
	a.selectFile("hello.txt")
	a.selectFile("unchecked.txt")
	awaitDiff(t, a, h)
	if a.file != "unchecked.txt" || strings.Contains(a.diff, "one more edit") {
		t.Fatal("stale preview accepted")
	}
	for _, size := range [][2]int{{820, 600}, {1440, 900}, {1180, 780}} {
		tt.SetSize(size[0], size[1])
		tt.Frame()
		click(t, tt, "Change: unchecked.txt")
		awaitDiff(t, a, h)
		if a.file != "unchecked.txt" {
			t.Fatal("file action failed after resize")
		}
	}
}

func TestCloneAndRemoteActionsUI(t *testing.T) {
	source := repoFixture(t)
	remote := filepath.Join(t.TempDir(), "remote.git")
	testGit(t, source, "clone", "--bare", source, remote)
	a, h := appFixture(t)
	tt := ui.NewTester(a.view, 1180, 780)
	click(t, tt, "Clone repository")
	a.cloneURL = remote
	a.clonePath = filepath.Join(t.TempDir(), "cloned")
	click(t, tt, "Clone and open")
	await(t, a, h)
	if a.repo == nil || a.cloning || a.errorMessage {
		t.Fatal("clone UI", a.message)
	}
	root := a.current().Path
	testGit(t, root, "config", "user.name", "Test")
	testGit(t, root, "config", "user.email", "test@example.invalid")
	testGit(t, root, "config", "commit.gpgsign", "false")
	writeFile(t, root, "hello.txt", "native push\n")
	a.refresh()
	await(t, a, h)
	awaitDiff(t, a, h)
	a.summary = "Commit before push"
	a.commitSelected()
	await(t, a, h)
	click(t, tt, "Push")
	await(t, a, h)
	if a.errorMessage {
		t.Fatal(a.message)
	}
	if testGit(t, root, "rev-parse", "HEAD") != testGit(t, remote, "rev-parse", "main") {
		t.Fatal("push button did not push")
	}
	click(t, tt, "Fetch")
	await(t, a, h)
	if a.errorMessage {
		t.Fatal(a.message)
	}
	click(t, tt, "Pull")
	await(t, a, h)
	if a.errorMessage {
		t.Fatal(a.message)
	}
	// Add another repository then use the dropdown to switch back.
	h.folder = source
	click(t, tt, "Add repository")
	await(t, a, h)
	tt.Frame()
	if err := tt.Click("Repository selector"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.recent) != 2 {
		t.Fatalf("recent: %v", a.recent)
	}
	previous := a.recent[1]
	if err := tt.Click(a.repositoryLabel(previous)); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	await(t, a, h)
	if a.repo.Path != previous {
		t.Fatal("repository dropdown did not switch", a.repo.Path)
	}
	if _, err := os.Stat(filepath.Join(root, "hello.txt")); err != nil {
		t.Fatal(err)
	}
}
