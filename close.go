package main

import (
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"
)

// primaryKey names the platform's command modifier as the keyboard shows it.
func primaryKey() string {
	if runtime.GOOS == "darwin" {
		return "⌘"
	}
	return "Ctrl+"
}

// closeInner is Primary+W (and File > Close): it closes the innermost open
// thing — the remove prompt, the New worktree form, the Clone form — as
// their Cancel buttons do, and otherwise asks to quit. Behind a dialog it
// does nothing; Esc closes dialogs.
func (a *app) closeInner() {
	if a.keyboardHelp || a.confirmQuit || a.allowClose {
		return
	}
	switch {
	case a.cloning:
		if !a.busy {
			a.cloning = false
		}
	case a.creating:
		if !a.busy {
			a.creating = false
		}
	case a.removing && a.managing:
		if !a.busy {
			a.removing = false
		}
	default:
		a.requestQuit()
	}
}

// quitReasons says what quitting now would lose: a Git operation cut off
// mid-way, or a commit message not yet committed.
func (a *app) quitReasons() []string {
	var reasons []string
	if a.operation != "" {
		reasons = append(reasons, "A Git operation is running ("+strings.TrimSuffix(a.operation, "…")+"). Quitting cancels it mid-way; a cancelled commit, push or clone can leave the repository partly changed.")
	}
	if a.repo != nil && strings.TrimSpace(a.summary+a.description) != "" {
		reasons = append(reasons, "The commit message you typed has not been committed and will be lost.")
	}
	return reasons
}

// requestQuit is every way of closing the window (Primary+W with nothing
// left open, the window's close button, Cmd+Q, the window manager): it asks
// first when something would be lost.
func (a *app) requestQuit() {
	if a.allowClose || a.confirmQuit {
		return
	}
	if len(a.quitReasons()) == 0 {
		a.quit()
		return
	}
	a.keyboardHelp = false
	a.confirmQuit = true
}

// cancelQuit dismisses the quit question and gives the keyboard back to
// the commit message, when that is what was asked about and it is on
// screen; otherwise the focus goes back where the dialog found it.
func (a *app) cancelQuit() {
	a.confirmQuit = false
	if a.repo != nil && !a.managing && !a.creating && !a.cloning && strings.TrimSpace(a.summary+a.description) != "" {
		a.summaryInput.Focus()
	}
}

// quit closes the window, now that nothing needs asking.
func (a *app) quit() {
	a.confirmQuit = false
	a.allowClose = true
	a.host.Quit()
}

func (a *app) quitDialog(c *ui.Context) {
	ui.DialogBase(c, &a.confirmQuit, func(_, panel ui.Element) {
		panel.Width(460).Padding(20).Gap(12).Background(c.Theme().Surface).Radius(10).Border(1, line)
		ui.Text(c, "Quit Kizuna?").FontWeight(700)
		reasons := a.quitReasons()
		if len(reasons) == 0 {
			reasons = []string{"Nothing is running any more."}
		}
		for _, reason := range reasons {
			ui.Text(c, reason).FontSize(12.5).Wrap()
		}
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			ui.Box(c).Grow(1)
			ui.Button(c, "Cancel").AutoFocus().OnClick(a.cancelQuit)
			ui.Button(c, "Quit Kizuna").TextColor(red).OnClick(a.quit)
		})
	})
}
