package main

import "github.com/egoist/mygo/ui"

func (a *app) navigationHelp(c *ui.Context) {
	ui.DialogBase(c, &a.keyboardHelp, func(_, panel ui.Element) {
		panel.Width(440).Padding(20).Gap(12).Background(c.Theme().Surface)
		ui.Text(c, "Keyboard shortcuts").FontWeight(700)
		ui.Text(c, "Cmd/Ctrl+O — Open repository\nCmd/Ctrl+N — New worktree\nCmd/Ctrl+R — Refresh\nCmd/Ctrl+/ — This help\nTab / Shift+Tab — Move focus\nEsc — Close help\nRepository menu — Fetch, pull, push, open folder/terminal").Wrap()
		ui.Button(c, "Close shortcuts").AutoFocus().OnClick(func() { a.keyboardHelp = false })
	})
}
