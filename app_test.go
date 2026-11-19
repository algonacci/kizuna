package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

type fakeHost struct {
	queue  chan func()
	folder string
	opened chan string
	quit   bool
}

func (h *fakeHost) Update(fn func())                       { h.queue <- fn }
func (h *fakeHost) PickFolder() (string, error)            { return h.folder, nil }
func (h *fakeHost) PickCloneFolder(string) (string, error) { return h.folder, nil }
func (h *fakeHost) OpenPath(path string) error             { h.opened <- "folder:" + path; return nil }
func (h *fakeHost) OpenTerminal(path string) error         { h.opened <- "terminal:" + path; return nil }
func (h *fakeHost) Quit()                                  { h.quit = true }
func appFixture(t *testing.T) (*app, *fakeHost) {
	t.Helper()
	h := &fakeHost{queue: make(chan func(), 64), opened: make(chan string, 8)}
	a := newApp(h)
	a.configPath = filepath.Join(t.TempDir(), "recent.json")
	t.Cleanup(a.cancel)
	return a, h
}
func await(t *testing.T, a *app, h *fakeHost) {
	t.Helper()
	timer := time.NewTimer(20 * time.Second)
	defer timer.Stop()
	for a.busy {
		select {
		case fn := <-h.queue:
			fn()
		case <-timer.C:
			t.Fatal("timed out waiting for queued UI update")
		}
	}
}
func click(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	tt.Frame()
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
}

func TestNativeViewWorktreeFlow(t *testing.T) {
	a, h := appFixture(t)
	h.folder = repoFixture(t)
	tt := ui.NewTester(a.view, 1180, 780)
	click(t, tt, "Open Git repository")
	await(t, a, h)
	if a.repo == nil {
		t.Fatal(a.message)
	}
	original := a.selected
	click(t, tt, "+ New worktree")
	click(t, tt, "Input: BRANCH NAME")
	tt.Type("feature/from-ui")
	a.draft.Path = filepath.Join(filepath.Dir(h.folder), "ui-worktree")
	click(t, tt, "Create worktree")
	await(t, a, h)
	if a.errorMessage || len(a.repo.Trees) != 2 || a.current().Branch != "feature/from-ui" {
		t.Fatalf("creation: %s %+v", a.message, a.repo)
	}
	created := a.selected
	click(t, tt, "Worktree selector")
	click(t, tt, "main  ·  "+original)
	if a.selected != original {
		t.Fatal("worktree dropdown did not select main")
	}
	click(t, tt, "Worktree selector")
	click(t, tt, "feature/from-ui  ·  "+created)
	if a.selected != created {
		t.Fatal("worktree dropdown did not restore created worktree")
	}
	click(t, tt, "Open folder")
	select {
	case path := <-h.opened:
		if path != "folder:"+a.selected {
			t.Fatal(path)
		}
	case <-time.After(time.Second):
		t.Fatal("folder action missing")
	}
	click(t, tt, "Worktrees")
	click(t, tt, "Lock")
	await(t, a, h)
	if !a.current().Locked {
		t.Fatal("UI lock failed")
	}
	click(t, tt, "Unlock")
	await(t, a, h)
	click(t, tt, "Remove…")
	click(t, tt, "Cancel")
	if a.removing || len(a.repo.Trees) != 2 {
		t.Fatal("remove cancellation failed")
	}
	click(t, tt, "Remove…")
	click(t, tt, "Remove worktree")
	await(t, a, h)
	if a.errorMessage || len(a.repo.Trees) != 1 || a.selected != original {
		t.Fatal("remove failed", a.message)
	}
	if got := readRecent(a.configPath); len(got) != 1 || got[0] != a.repo.Path {
		t.Fatalf("history: %v", got)
	}
	for _, size := range [][2]int{{820, 600}, {1440, 900}, {1180, 780}} {
		tt.SetSize(size[0], size[1])
		tt.Frame()
		click(t, tt, "Refresh")
		await(t, a, h)
	}
}

func TestQueuedLoadAndFailedCreatePreserveState(t *testing.T) {
	a, h := appFixture(t)
	path := repoFixture(t)
	a.openRepository(path)
	if a.repo != nil || !a.busy {
		t.Fatal("load not asynchronous")
	}
	a.openRepository("/not/a/repository") // Serialized while the first load is in flight.
	await(t, a, h)
	if a.repo == nil {
		t.Fatal(a.message)
	}
	a.startCreate()
	a.draft.Branch = "main"
	a.create()
	await(t, a, h)
	if !a.errorMessage || !a.creating || len(a.repo.Trees) != 1 {
		t.Fatal("failed create lost draft", a.message)
	}
	a.creating = false
	a.openRepository(t.TempDir())
	await(t, a, h)
	if !a.errorMessage || a.repo == nil || !strings.Contains(a.repo.Path, filepath.Base(path)) {
		t.Fatal("failed open lost existing repo")
	}
}

func TestWelcomeCloneAction(t *testing.T) {
	a, _ := appFixture(t)
	tt := ui.NewTester(a.view, 820, 600)
	click(t, tt, "Clone repository")
	if !a.cloning {
		t.Fatal("welcome clone action did not open the clone form")
	}
	click(t, tt, "Cancel")
	if a.cloning {
		t.Fatal("clone form did not close")
	}
}

func TestSnapshots(t *testing.T) {
	out := os.Getenv("KIZUNA_SNAPSHOT")
	if out == "" {
		t.Skip("set KIZUNA_SNAPSHOT to render UI review images")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	a, _ := appFixture(t)
	tt := ui.NewTester(a.view, 1180, 780)
	save := func(name string) {
		t.Helper()
		tt.Frame()
		f, err := os.Create(filepath.Join(out, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err = png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}
	save("welcome")
	a.keyboardHelp = true
	save("shortcuts")
	a.keyboardHelp = false
	a.repo = &repository{Path: "/Users/eric/Projects/kizuna/.git", Branches: []string{"main", "feature/worktree-client"}, Trees: []worktree{
		{Path: "/Users/eric/Projects/kizuna", Branch: "main", Head: "309631ba129d", Main: true},
		{Path: "/Users/eric/Projects/kizuna-client", Branch: "feature/worktree-client", Head: "e7a4d01927ba", Upstream: "origin/feature/worktree-client", Ahead: 2, Changes: []change{{Code: " M", Path: "app.go"}, {Code: "??", Path: "view.go"}, {Code: "A ", Path: "resources/icon.png"}}, Commits: []commit{{"e7a4d01", "Build a native home for your worktrees"}, {"93b02ce", "Add safe Git worktree operations"}, {"309631b", "Initial commit"}}},
		{Path: "/Users/eric/Projects/kizuna-hotfix", Branch: "fix/startup", Head: "309631ba129d", Locked: true, LockReason: "Protected by Kizuna"},
	}}
	a.selected = a.repo.Trees[1].Path
	a.recent = []string{a.repo.Path}
	a.repo.Remotes = []string{"origin"}
	a.repo.RemoteURLs = map[string]string{"origin": "git@github.com:algonacci/kizuna.git"}
	a.repoIdentities[a.repo.Path] = "algonacci/kizuna"
	a.remote = "origin"
	a.file = "app.go"
	a.checked = map[string]bool{"app.go": true, "view.go": true, "resources/icon.png": false}
	a.summary = "Add changes-first workspace"
	a.diff = "diff --git a/app.go b/app.go\nindex 309631b..e7a4d01 100644\n--- a/app.go\n+++ b/app.go\n@@ -16,6 +16,10 @@ type app struct {\n     host       host\n     repo       *repository\n-    selected   string\n+    selected   string\n+    file       string\n+    diff       string\n+    checked    map[string]bool\n }\n\n@@ -109,4 +113,5 @@ func (a *app) selectTree(path string) {\n     a.selected = path\n+    a.syncChanges(true)\n }"
	a.message = "3 worktrees · up to date"
	save("workspace")
	tt.SetSize(820, 600)
	save("compact")
	tt.SetSize(1180, 780)
	a.managing = true
	save("worktrees")
	a.managing = false
	a.startCreate()
	a.draft.Branch = "feature/next-idea"
	save("create")
	a.startClone()
	a.cloneURL = "https://github.com/Omniflow-id/example.git"
	a.clonePath = "/Users/eric/Projects/example"
	save("clone")
}
