package main

import "github.com/egoist/mygo/ui"

func (a *app) navigationHelp(c *ui.Context) {
	dialog(c, &a.keyboardHelp, 440, false, func() {
		dialogTitle(c, "Keyboard shortcuts", "")
		k := primaryKey()
		ui.Column(c).FillWidth().Gap(2).Children(func() {
			for _, row := range [][2]string{
				{k + "O", "Add repository"},
				{k + "N", "New worktree"},
				{k + "R", "Refresh"},
				{k + "Shift+T", "Fetch"},
				{k + "Shift+P", "Pull"},
				{k + "P", "Push"},
				{k + "W", "Close the open dialog, else quit Kizuna"},
				{k + "/", "This help"},
				{"Tab / Shift+Tab", "Move focus"},
				{"Esc", "Close a dialog or dropdown"},
			} {
				ui.Row(c).FillWidth().Height(24).Gap(12).Children(func() {
					ui.Text(c, row[0]).FontSize(11.5).TextColor(muted).Width(120).Shrink(0)
					ui.Text(c, row[1]).FontSize(12.5).SingleLine().Grow(1).Basis(0).MinWidth(0)
				})
			}
		})
		ui.Text(c, "Add, clone and refresh are in the repository dropdown; new worktree, folder and terminal in the worktree dropdown.").FontSize(11.5).TextColor(faint).Wrap().FillWidth()
		dialogButtons(c, func() { ui.Button(c, "Close shortcuts").AutoFocus().OnClick(func() { a.keyboardHelp = false }) })
	})
}
