package main

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// closeFixture opens a fixture repository with HOME and the config dir
// pointed at temporary folders, so nothing reads or writes the user's own.
func closeFixture(t *testing.T) (*app, *fakeHost, *ui.Tester) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	a, h := appFixture(t)
	h.folder = repoFixture(t)
	tt := ui.NewTester(a.view, 1180, 780)
	click(t, tt, "Add repository")
	await(t, a, h)
	if a.repo == nil {
		t.Fatal(a.message)
	}
	tt.Frame()
	return a, h, tt
}

func pressW(tt *ui.Tester) {
	tt.Key(ui.Cmd, ui.KeyW)
	tt.Frame()
}

// TestCloseIdleQuits: with nothing open and nothing running, Primary+W
// quits at once, without asking.
func TestCloseIdleQuits(t *testing.T) {
	a, h, tt := closeFixture(t)
	pressW(tt)
	if a.confirmQuit || !h.quit || !a.allowClose {
		t.Fatalf("confirm %v, quit %v, allowClose %v", a.confirmQuit, h.quit, a.allowClose)
	}
}

// TestCloseFormsFirst: Primary+W closes the New worktree and Clone
// dialogs, an open dropdown and the remove prompt as their Cancel buttons
// (or Esc) do, before quitting.
func TestCloseFormsFirst(t *testing.T) {
	a, h, tt := closeFixture(t)
	menu(t, tt, "Worktree selector", "New worktree")
	pressW(tt)
	if a.creating || h.quit || a.confirmQuit {
		t.Fatalf("New worktree form: creating %v, quit %v", a.creating, h.quit)
	}
	menu(t, tt, "Repository selector", "Clone repository")
	pressW(tt)
	if a.cloning || h.quit || a.confirmQuit {
		t.Fatalf("Clone form: cloning %v, quit %v", a.cloning, h.quit)
	}
	click(t, tt, "Repository selector")
	pressW(tt)
	if a.repoMenu || h.quit || a.confirmQuit {
		t.Fatalf("dropdown: open %v, quit %v", a.repoMenu, h.quit)
	}
	click(t, tt, "Worktrees")
	a.removing = true // Remove… is offered only for linked worktrees
	tt.Frame()
	pressW(tt)
	if a.removing || h.quit || !a.managing {
		t.Fatalf("remove prompt: removing %v, quit %v", a.removing, h.quit)
	}
	pressW(tt)
	if !h.quit {
		t.Fatal("nothing left open, but Primary+W did not quit")
	}
}

// TestQuitAsksWhileGitRuns: quitting while a Git command runs asks; Cancel
// keeps it running, and the window's close (requestQuit) asks the same way.
func TestQuitAsksWhileGitRuns(t *testing.T) {
	a, h, tt := closeFixture(t)
	release := make(chan struct{})
	a.runGitAction("Git push · origin…", false, func() error { <-release; return nil })
	tt.Frame()

	pressW(tt)
	if h.quit || a.allowClose || !a.confirmQuit || !tt.HasText("Quit Kizuna") {
		t.Fatalf("quit without asking: quit %v, confirm %v", h.quit, a.confirmQuit)
	}
	// Primary+W again behind the question does nothing.
	pressW(tt)
	if h.quit || !a.confirmQuit {
		t.Fatal("Primary+W behind the question acted")
	}
	click(t, tt, "Cancel")
	if h.quit || a.allowClose || a.confirmQuit || !a.busy || a.operation == "" {
		t.Fatalf("Cancel: quit %v, confirm %v, busy %v", h.quit, a.confirmQuit, a.busy)
	}

	a.requestQuit() // the window's close button, Cmd+Q, the window manager
	tt.Frame()
	click(t, tt, "Quit Kizuna")
	if !h.quit || !a.allowClose {
		t.Fatalf("confirm: quit %v, allowClose %v", h.quit, a.allowClose)
	}
	close(release)
	await(t, a, h)
}

// TestQuitAsksForCommitMessage: a typed but uncommitted commit message is
// asked about; Cancel keeps it and gives the keyboard back to it.
func TestQuitAsksForCommitMessage(t *testing.T) {
	a, h, tt := closeFixture(t)
	writeFile(t, a.current().Path, "hello.txt", "changed\n")
	a.refresh()
	await(t, a, h)
	tt.Frame()
	click(t, tt, "Commit summary")
	tt.Type("Half written")
	tt.Frame()
	pressW(tt)
	if h.quit || !a.confirmQuit || !tt.HasText("The commit message you typed has not been committed and will be lost.") {
		t.Fatalf("quit without asking: quit %v, confirm %v", h.quit, a.confirmQuit)
	}
	click(t, tt, "Cancel")
	tt.Frame()
	tt.Type("!")
	tt.Frame()
	if h.quit || a.summary != "Half written!" {
		t.Fatalf("Cancel: quit %v, summary %q", h.quit, a.summary)
	}
	pressW(tt)
	click(t, tt, "Quit Kizuna")
	if !h.quit {
		t.Fatal("confirm did not quit")
	}
}

// TestCloseIgnoredBehindHelp: Primary+W, or File > Close, does nothing
// while the keyboard help dialog is open.
func TestCloseIgnoredBehindHelp(t *testing.T) {
	a, h, tt := closeFixture(t)
	tt.Key(ui.Cmd, ui.KeySlash)
	tt.Frame()
	if !a.keyboardHelp {
		t.Fatal("help did not open")
	}
	pressW(tt)
	a.closeInner() // File > Close, through host.Update
	tt.Frame()
	if h.quit || a.confirmQuit || !a.keyboardHelp {
		t.Fatalf("acted behind the dialog: quit %v, confirm %v", h.quit, a.confirmQuit)
	}
}
