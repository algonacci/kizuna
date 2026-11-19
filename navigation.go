package main

import "github.com/egoist/mygo/ui"

func (a *app) navigationHelp(c *ui.Context) {
	ui.DialogBase(c, &a.keyboardHelp, func(_, panel ui.Element) {
		panel.Width(440).Padding(20).Gap(12).Background(c.Theme().Surface)
		ui.Text(c, "Keyboard shortcuts").FontWeight(700)
		k := primaryKey()
		ui.Text(c, k+"O — Open repository\n"+k+"N — New worktree\n"+k+"R — Refresh\n"+k+"W — Close the open form, else quit Kizuna\n"+k+"/ — This help\nTab / Shift+Tab — Move focus\nEsc — Close help\nRepository menu — Fetch, pull, push, open folder/terminal").Wrap()
		ui.Button(c, "Close shortcuts").AutoFocus().OnClick(func() { a.keyboardHelp = false })
	})
}
