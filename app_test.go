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
	themes []string
}

func (h *fakeHost) Update(fn func())                       { h.queue <- fn }
func (h *fakeHost) PickFolder() (string, error)            { return h.folder, nil }
func (h *fakeHost) PickCloneFolder(string) (string, error) { return h.folder, nil }
func (h *fakeHost) OpenPath(path string) error             { h.opened <- "folder:" + path; return nil }
func (h *fakeHost) OpenTerminal(path string) error         { h.opened <- "terminal:" + path; return nil }
func (h *fakeHost) Quit()                                  { h.quit = true }
func (h *fakeHost) SetTheme(mode string)                   { h.themes = append(h.themes, mode) }
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

// menu opens a top bar dropdown (its trigger's label) and clicks an item.
func menu(t *testing.T, tt *ui.Tester, trigger, item string) {
	t.Helper()
	click(t, tt, trigger)
	click(t, tt, item)
}

func TestNativeViewWorktreeFlow(t *testing.T) {
	a, h := appFixture(t)
	h.folder = repoFixture(t)
	tt := ui.NewTester(a.view, 1180, 780)
	click(t, tt, "Add repository")
	await(t, a, h)
	if a.repo == nil {
		t.Fatal(a.message)
	}
	original := a.selected
	menu(t, tt, "Worktree selector", "New worktree")
	click(t, tt, "Input: Branch name")
	tt.Type("feature/from-ui")
	a.draft.Path = filepath.Join(filepath.Dir(h.folder), "ui-worktree")
	click(t, tt, "Create worktree")
	await(t, a, h)
	if a.errorMessage || len(a.repo.Trees) != 2 || a.current().Branch != "feature/from-ui" {
		t.Fatalf("creation: %s %+v", a.message, a.repo)
	}
	created := a.selected
	menu(t, tt, "Worktree selector", "main  ·  "+original)
	if a.selected != original {
		t.Fatal("worktree dropdown did not select main")
	}
	menu(t, tt, "Worktree selector", "feature/from-ui  ·  "+created)
	if a.selected != created {
		t.Fatal("worktree dropdown did not restore created worktree")
	}
	menu(t, tt, "Worktree selector", "Open folder")
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
	for _, size := range [][2]int{{760, 520}, {1440, 900}, {1180, 780}} {
		tt.SetSize(size[0], size[1])
		tt.Frame()
		menu(t, tt, "Repository selector", "Refresh")
		await(t, a, h)
		if a.repoMenu {
			t.Fatal("dropdown stayed open after choosing Refresh")
		}
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

func TestCompactChangesKeepFilesAndCommitReachable(t *testing.T) {
	a, _ := appFixture(t)
	a.repo = &repository{Path: "/fixture", Trees: []worktree{{Path: "/fixture", Branch: "main", Main: true, Changes: []change{
		{Path: "app.go", Code: " M"}, {Path: "view.go", Code: "??"}, {Path: "resources/icon.png", Code: "A "},
	}}}}
	a.selected = "/fixture"
	a.file = "app.go"
	a.summary = "Review selected changes"
	a.checked = map[string]bool{"app.go": true}
	for _, size := range [][2]float32{{760, 520}, {820, 600}} {
		tt := ui.NewTester(a.view, int(size[0]), int(size[1]))
		for _, label := range []string{"Change: app.go", "Change: view.go", "Change: resources/icon.png", "Commit summary", "Commit description", "Commit to main", "Repository selector", "Worktree selector", "No remote"} {
			box, ok := tt.Find(label)
			if !ok || box.W <= 0 || box.H <= 0 || box.X < 0 || box.Y < 0 || box.X+box.W > size[0] || box.Y+box.H > size[1] {
				t.Fatalf("%s not reachable at %v: %+v", label, size, box)
			}
		}
	}
}

// TestThemeCyclesAndPersists: the title bar button steps System → Light →
// Dark → System, tells the host (native menus, tooltips) and saves the
// choice beside recent.json, which keeps its own format.
func TestThemeCyclesAndPersists(t *testing.T) {
	a, h := appFixture(t)
	if err := writeRecent(a.configPath, []string{"/fixture"}); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(a.view, 1180, 780)
	path := settingsPath(a.configPath)
	if got := readSettings(path).Theme; got != themeSystem || a.theme != themeSystem {
		t.Fatalf("default mode: app %q, saved %q", a.theme, got)
	}
	for _, want := range []string{themeLight, themeDark, themeSystem} {
		label := map[string]string{themeSystem: "Appearance: System", themeLight: "Appearance: Light", themeDark: "Appearance: Dark"}[a.theme]
		click(t, tt, label)
		if a.theme != want || h.themes[len(h.themes)-1] != want {
			t.Fatalf("cycled to %q (host %v), want %q", a.theme, h.themes, want)
		}
		deadline := time.Now().Add(5 * time.Second)
		for readSettings(path).Theme != want {
			if time.Now().After(deadline) {
				t.Fatalf("saved %q, want %q", readSettings(path).Theme, want)
			}
			time.Sleep(10 * time.Millisecond)
		}
		if isDark != (want == themeDark) && want != themeSystem {
			t.Fatalf("palette dark=%v for %q", isDark, want)
		}
	}
	if got := readRecent(a.configPath); len(got) != 1 || got[0] != "/fixture" {
		t.Fatalf("recent.json changed: %v", got)
	}
}

// TestRemoteButtonFollowsBranchState: the one remote button offers what
// matters now, as GitHub Desktop's does.
func TestRemoteButtonFollowsBranchState(t *testing.T) {
	a, _ := appFixture(t)
	a.repo = &repository{Path: "/fixture", Remotes: []string{"origin"}, Trees: []worktree{{Path: "/fixture", Branch: "main", Upstream: "origin/main"}}}
	a.selected, a.remote = "/fixture", "origin"
	w := &a.repo.Trees[0]
	for _, tc := range []struct {
		ahead, behind int
		upstream      string
		detached      bool
		title, action string
	}{
		{0, 0, "origin/main", false, "Fetch origin", "fetch"},
		{2, 0, "origin/main", false, "Push origin", "push"},
		{0, 3, "origin/main", false, "Pull origin", "pull"},
		{1, 3, "origin/main", false, "Pull origin", "pull"},
		{0, 0, "", false, "Publish branch", "push"},
		{0, 0, "", true, "Fetch origin", "fetch"},
	} {
		w.Ahead, w.Behind, w.Upstream, w.Detached = tc.ahead, tc.behind, tc.upstream, tc.detached
		_, title, _, action, enabled := a.remoteState()
		if title != tc.title || action != tc.action || !enabled {
			t.Errorf("%+v: got %q %q enabled=%v", tc, title, action, enabled)
		}
	}
	a.remote = ""
	if _, title, _, action, enabled := a.remoteState(); enabled || action != "" || title != "No remote" {
		t.Errorf("no remote: %q %q %v", title, action, enabled)
	}
	a.remoteAction("") // the disabled button's action does nothing
	if a.busy {
		t.Fatal("empty remote action started work")
	}
}

// TestDialogsShareBackdropAndStayWhileBusy: Clone is a modal dialog over a
// dimmed backdrop; clicking the backdrop closes it, except while its Git
// command runs.
func TestDialogsShareBackdropAndStayWhileBusy(t *testing.T) {
	a, _ := appFixture(t)
	tt := ui.NewTester(a.view, 1180, 780)
	click(t, tt, "Clone repository")
	if !a.cloning || !tt.HasText("Clone and open") {
		t.Fatal("clone dialog did not open")
	}
	// The page behind is base (#f3f4f7 light, #0e1118 dark); the backdrop
	// must darken it by a quarter or more in both modes.
	for mode, page := range map[string]uint8{themeLight: 0xf3, themeDark: 0x0e} {
		a.theme = mode
		tt.Frame()
		if corner := tt.Image().RGBAAt(1180-40, 780-120); corner.G > page-page/4 {
			t.Fatalf("%s: backdrop does not dim the window: %v", mode, corner)
		}
	}
	a.busy = true
	tt.ClickAt(20, 400)
	tt.Frame()
	if !a.cloning {
		t.Fatal("backdrop click closed the dialog while busy")
	}
	a.busy = false
	tt.Frame()
	tt.ClickAt(20, 400)
	tt.Frame()
	if a.cloning {
		t.Fatal("backdrop click did not close the dialog")
	}
}

func TestSnapshots(t *testing.T) {
	root := os.Getenv("KIZUNA_SNAPSHOT")
	if root == "" {
		t.Skip("set KIZUNA_SNAPSHOT to render UI review images")
	}
	for _, mode := range []string{themeDark, themeLight} {
		t.Run(mode, func(t *testing.T) { snapshots(t, filepath.Join(root, mode), mode) })
	}
}

func snapshots(t *testing.T, out, mode string) {
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	a, _ := appFixture(t)
	a.theme = mode
	tt := ui.NewTester(a.view, 1280, 800)
	save := func(name string) {
		t.Helper()
		tt.Frame()
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
	both := func(name string) {
		t.Helper()
		tt.SetSize(1280, 800)
		save(name)
		tt.SetSize(760, 520)
		save(name + "-compact")
		tt.SetSize(1280, 800)
	}
	a.recent = []string{"/Users/eric/Projects/kizuna/.git", "/Users/eric/Projects/omniflow-api", "/Users/eric/Projects/recast"}
	a.repoIdentities["/Users/eric/Projects/kizuna/.git"] = "algonacci/kizuna"
	a.repoIdentities["/Users/eric/Projects/omniflow-api"] = "Omniflow-id/omniflow-api"
	both("welcome")
	a.keyboardHelp = true
	save("shortcuts")
	a.keyboardHelp = false
	a.startClone()
	a.cloneURL = "https://github.com/Omniflow-id/example.git"
	a.clonePath = "/Users/eric/Documents/GitHub/example"
	both("clone")
	a.cloning = false
	a.repo = &repository{Path: "/Users/eric/Projects/kizuna/.git", Branches: []string{"main", "feature/worktree-client"}, Trees: []worktree{
		{Path: "/Users/eric/Projects/kizuna", Branch: "main", Head: "309631ba129d", Main: true, Upstream: "origin/main", Commits: []commit{{"309631b", "Initial commit"}}},
		{Path: "/Users/eric/Projects/kizuna-client", Branch: "feature/worktree-client", Head: "e7a4d01927ba", Upstream: "origin/feature/worktree-client", Ahead: 2, Changes: []change{{Code: " M", Path: "app.go"}, {Code: "??", Path: "view.go"}, {Code: "A ", Path: "resources/icon.png"}, {Code: " D", Path: "docs/old-notes.md"}}, Commits: []commit{{"e7a4d01", "Build a native home for your worktrees"}, {"93b02ce", "Add safe Git worktree operations"}, {"309631b", "Initial commit"}}},
		{Path: "/Users/eric/Projects/kizuna-hotfix", Branch: "fix/startup", Head: "309631ba129d", Locked: true, LockReason: "Protected by Kizuna"},
	}}
	a.selected = a.repo.Trees[1].Path
	a.repo.Remotes = []string{"origin"}
	a.repo.RemoteURLs = map[string]string{"origin": "git@github.com:algonacci/kizuna.git"}
	a.remote = "origin"
	a.file = "app.go"
	a.checked = map[string]bool{"app.go": true, "view.go": true, "resources/icon.png": false, "docs/old-notes.md": true}
	a.summary = "Add changes-first workspace"
	a.diff = "diff --git a/app.go b/app.go\nindex 309631b..e7a4d01 100644\n--- a/app.go\n+++ b/app.go\n@@ -16,6 +16,10 @@ type app struct {\n     host       host\n     repo       *repository\n-    selected   string\n+    selected   string\n+    file       string\n+    diff       string\n+    checked    map[string]bool\n }\n\n@@ -109,4 +113,5 @@ func (a *app) selectTree(path string) {\n     a.selected = path\n+    a.syncChanges(true)\n }"
	a.message = "3 worktrees · up to date"
	both("workspace")
	a.repoMenu = true
	save("repository-menu")
	a.repoMenu = false
	a.treeMenu = true
	both("worktree-menu")
	a.treeMenu = false
	a.selected = a.repo.Trees[0].Path
	a.file, a.diff = "", ""
	a.repo.Trees[0].Behind = 3
	both("no-changes")
	a.repo.Trees[0].Behind = 0
	a.selected = a.repo.Trees[1].Path
	a.managing = true
	both("worktrees")
	a.removing = true
	save("remove")
	a.removing = false
	a.managing = false
	a.selected = a.repo.Trees[1].Path
	a.file = "app.go"
	a.startCreate()
	a.draft.Branch = "feature/next-idea"
	both("create")
	a.creating = false
	a.operation = "Git push · origin…"
	a.confirmQuit = true
	save("quit")
}
