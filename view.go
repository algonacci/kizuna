package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
)

// The window follows GitHub Desktop: a top bar of three sections (current
// repository, current worktree, one remote action), a sidebar with the
// Changes and Worktrees tabs, and the diff or worktree detail on the right.
// Less frequent actions live in the two dropdowns, the native menu and
// keyboard shortcuts.
func (a *app) view(c *ui.Context) {
	defer a.navigationHelp(c)
	defer a.quitDialog(c)
	defer a.formDialogs(c)
	dark := c.Theme().Dark // the system's, before SetTheme replaces it
	switch a.theme {
	case themeLight:
		dark = false
	case themeDark:
		dark = true
	}
	if dark {
		usePalette(darkPalette)
	} else {
		usePalette(lightPalette)
	}
	c.SetTheme(appTheme())
	c.OnShortcut(ui.Cmd, ui.KeySlash, func() { a.keyboardHelp = true })
	c.OnShortcut(ui.Cmd, ui.KeyO, a.pickRepository)
	c.OnShortcut(ui.Cmd, ui.KeyN, a.startCreate)
	c.OnShortcut(ui.Cmd, ui.KeyR, a.refresh)
	c.OnShortcut(ui.Cmd, ui.KeyW, a.closeInner)
	c.OnShortcut(ui.Cmd|ui.Shift, ui.KeyT, func() { a.remoteAction("fetch") })
	c.OnShortcut(ui.Cmd|ui.Shift, ui.KeyP, func() { a.remoteAction("pull") })
	c.OnShortcut(ui.Cmd, ui.KeyP, func() { a.remoteAction("push") })
	ui.Column(c).Fill().Background(base).Children(func() {
		a.toolbar(c)
		if a.repo == nil {
			a.welcome(c)
		} else {
			width, _ := c.Size()
			sidebar := float32(300)
			if width < 900 {
				sidebar = 264
			}
			ui.Row(c).Grow(1).MinHeight(0).FillWidth().AlignItems(ui.Stretch).Children(func() {
				ui.Column(c).Width(sidebar).Shrink(0).MinHeight(0).Background(panel).BorderWidth(0, 1, 0, 0).BorderColor(softLine).Children(func() {
					a.sidebarTabs(c)
					if a.managing {
						a.worktreeList(c)
					} else {
						a.changesSidebar(c)
					}
				})
				ui.Column(c).Grow(1).Basis(0).MinWidth(0).MinHeight(0).Background(canvas).Children(func() {
					if a.managing {
						ui.Scroll(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
							if w := a.current(); w != nil {
								a.detail(c, w)
							}
						})
					} else {
						a.diffView(c)
					}
				})
			})
		}
		a.statusBar(c)
	})
}

func (a *app) statusBar(c *ui.Context) {
	ui.Row(c).FillWidth().Height(26).Shrink(0).Padding(0, 14).Gap(10).Background(panel).BorderWidth(1, 0, 0, 0).BorderColor(softLine).Children(func() {
		color := muted
		if a.errorMessage {
			color = red
		} else if a.busy {
			color = accent
		}
		ui.Box(c).Size(6, 6).Radius(3).Background(color).Shrink(0)
		ui.Text(c, a.message).FontSize(11).TextColor(color).SingleLine().Grow(1).Basis(0).MinWidth(0).Tooltip(a.message)
		if w := a.current(); w != nil {
			ui.Text(c, w.Path).FontSize(11).TextColor(faint).SingleLine().MaxWidth(360).Shrink(1).MinWidth(0).Tooltip(w.Path)
		}
	})
}

func (a *app) toolbar(c *ui.Context) {
	inset := c.TitleBar()
	width, _ := c.Size()
	logo := a.repo == nil || width >= 1000
	ui.Row(c).FillWidth().Height(max(inset.Height, 52)).Shrink(0).Padding(0, inset.Right+6, 0, inset.Left).Background(panel).BorderWidth(0, 0, 1, 0).BorderColor(softLine).DragWindow().AlignItems(ui.Stretch).Children(func() {
		if logo {
			ui.Row(c).Shrink(0).Padding(0, 14).Gap(9).Children(func() {
				ui.Box(c).Size(26, 26).Radius(8).LinearGradient(brandGradient).Center().Children(func() { ui.Icon(c, icBranch).FontSize(15).TextColor(ui.Hex("#ffffff")) })
				if a.repo == nil || width >= 1180 {
					ui.Text(c, "Kizuna").FontSize(13.5).FontWeight(700)
				}
			})
		}
		if a.repo != nil {
			// Three equal sections share what the logo and theme button leave.
			used := float32(72 + inset.Left + inset.Right)
			if logo {
				used += 54
				if width >= 1180 {
					used += 60
				}
			}
			section := min(280, (float32(width)-used)/3)
			a.repositoryButton(c, section)
			a.worktreeButton(c, section)
			a.remoteButton(c, section)
		}
		ui.Box(c).Grow(1)
		a.themeButton(c)
	})
}

// themeButton cycles the appearance, as Nimbo's title bar button does.
func (a *app) themeButton(c *ui.Context) {
	ic, label := icMonitor, "System"
	switch a.theme {
	case themeLight:
		ic, label = icSun, "Light"
	case themeDark:
		ic, label = icMoon, "Dark"
	}
	b := ui.ButtonBase(c).Width(58).Shrink(0).Column().Center().Gap(4).Radius(8).Margin(6, 0).
		Label("Appearance: " + label).Tooltip("Appearance: System, Light or Dark").TextColor(muted).Transition(hoverFade)
	if b.Hovered() {
		b.Background(raised).TextColor(text)
	}
	b.OnClick(a.cycleTheme).Children(func() {
		ui.Icon(c, ic).FontSize(16)
		ui.Text(c, label).FontSize(10.5).FontWeight(500)
	})
}

func (a *app) repositoryName(path string) string {
	if id := a.repoIdentities[path]; id != "" {
		return id
	}
	name := filepath.Base(path)
	if name == ".git" {
		name = filepath.Base(filepath.Dir(path))
	}
	return name
}

// repositoryLabel names a recent repository with its folder, so that two
// checkouts of the same remote stay distinguishable.
func (a *app) repositoryLabel(path string) string {
	return a.repositoryName(path) + "  ·  " + filepath.Dir(path)
}

func (a *app) repositoryButton(c *ui.Context, width float32) {
	b := barButton(c, icRepo, "Current repository", a.repositoryName(a.repo.Path), "Repository selector", width, a.repoMenu, true, true)
	b.OnClick(func() { a.repoMenu = !a.repoMenu })
	ui.PopoverBase(c, b, &a.repoMenu, func(p ui.Element) {
		popoverPanel(p, 380)
		menuCaption(c, "REPOSITORIES")
		for _, path := range a.recent {
			c.Key("repo:" + path)
			menuRow(c, icRepo, a.repositoryName(path), filepath.Dir(path), "", a.repositoryLabel(path), path == a.repo.Path, !a.busy, func() {
				a.repoMenu = false
				if path != a.repo.Path {
					a.openRepository(path)
				}
			})
		}
		if len(a.repo.Remotes) > 1 {
			menuSeparator(c)
			menuCaption(c, "REMOTE FOR FETCH, PULL AND PUSH")
			for _, r := range a.repo.Remotes {
				c.Key("remote:" + r)
				menuRow(c, icRemote, r, a.repo.RemoteURLs[r], "", "Remote: "+r, r == a.remote, !a.busy, func() { a.remote = r; a.repoMenu = false })
			}
		}
		menuSeparator(c)
		k := primaryKey()
		menuRow(c, icFolder, "Add repository…", "", k+"O", "Add repository", false, !a.busy, func() { a.repoMenu = false; a.pickRepository() })
		menuRow(c, icClone, "Clone repository…", "", "", "Clone repository", false, !a.busy, func() { a.repoMenu = false; a.startClone() })
		menuRow(c, icRefresh, "Refresh", "", k+"R", "Refresh", false, !a.busy, func() { a.repoMenu = false; a.refresh() })
	})
}

func (a *app) worktreeButton(c *ui.Context, width float32) {
	w := a.current()
	title := "No worktree"
	if w != nil {
		title = branchLabel(*w)
	}
	b := barButton(c, icBranch, "Current worktree", title, "Worktree selector", width, a.treeMenu, true, true)
	b.OnClick(func() { a.treeMenu = !a.treeMenu })
	ui.PopoverBase(c, b, &a.treeMenu, func(p ui.Element) {
		popoverPanel(p, 400)
		menuCaption(c, "WORKTREES")
		ui.Scroll(c).FillWidth().MaxHeight(340).Children(func() {
			ui.Column(c).FillWidth().Gap(2).Children(func() {
				for _, t := range a.repo.Trees {
					c.Key("tree:" + t.Path)
					label, _ := statusLabel(t)
					menuRow(c, icBranch, branchLabel(t), filepath.Base(t.Path)+"  ·  "+label, "", branchLabel(t)+"  ·  "+t.Path, t.Path == a.selected, !a.busy, func() {
						a.treeMenu = false
						a.selectTree(t.Path)
					})
				}
			})
		})
		menuSeparator(c)
		k := primaryKey()
		usable := !a.busy && w != nil && !w.Prunable
		menuRow(c, icPlus, "New worktree…", "", k+"N", "New worktree", false, !a.busy, func() { a.treeMenu = false; a.startCreate() })
		menuRow(c, icTree, "Manage worktrees", "", "", "Manage worktrees", false, true, func() { a.treeMenu = false; a.showWorktrees() })
		menuRow(c, icFolder, "Open folder", "", "", "Open folder", false, usable, func() { a.treeMenu = false; a.openSelected(false) })
		menuRow(c, icTerminal, "Open terminal", "", "", "Open terminal", false, usable && !w.Bare, func() { a.treeMenu = false; a.openSelected(true) })
	})
}

// remoteState picks the one remote action that matters now, as GitHub
// Desktop's Fetch/Pull/Push button does.
func (a *app) remoteState() (ic *ui.SVG, title, detail, action string, enabled bool) {
	w := a.current()
	if w == nil || a.remote == "" {
		return icFetch, "No remote", "Add one with git remote add", "", false
	}
	r := a.remote
	switch {
	case w.Bare || w.Prunable || w.Detached:
		ic, title, detail, action = icFetch, "Fetch "+r, "Update remote branches", "fetch"
	case w.Upstream == "":
		ic, title, detail, action = icPush, "Publish branch", "Push to "+r+" and track it", "push"
	case w.Behind > 0:
		ic, title, action = icPull, "Pull "+r, "pull"
		detail = fmt.Sprintf("%d commits behind", w.Behind)
		if w.Ahead > 0 {
			detail += fmt.Sprintf(" · %d ahead", w.Ahead)
		}
	case w.Ahead > 0:
		ic, title, detail, action = icPush, "Push "+r, fmt.Sprintf("%d commits to push", w.Ahead), "push"
	default:
		ic, title, detail, action = icFetch, "Fetch "+r, "Check for new commits", "fetch"
	}
	if a.busy && strings.HasPrefix(a.operation, "Git ") {
		detail = "Working…"
	}
	return ic, title, detail, action, !a.busy && !(w.Prunable && action != "fetch")
}

func (a *app) remoteButton(c *ui.Context, width float32) {
	ic, title, detail, action, enabled := a.remoteState()
	b := barButton(c, ic, detail, title, title, width, false, enabled, false)
	if w := a.current(); w != nil && w.Upstream != "" {
		b.Tooltip(fmt.Sprintf("%s · ↑%d ↓%d", w.Upstream, w.Ahead, w.Behind))
	}
	b.OnClick(func() { a.remoteAction(action) })
}

func (a *app) sidebarTabs(c *ui.Context) {
	w := a.current()
	changes, trees := 0, len(a.repo.Trees)
	if w != nil {
		changes = len(w.Changes)
	}
	ui.Row(c).FillWidth().Height(40).Shrink(0).BorderWidth(0, 0, 1, 0).BorderColor(softLine).AlignItems(ui.Stretch).Children(func() {
		for _, tab := range []struct {
			label  string
			count  int
			active bool
			fn     func()
		}{
			{"Changes", changes, !a.managing, a.showChanges},
			{"Worktrees", trees, a.managing, a.showWorktrees},
		} {
			b := ui.ButtonBase(c).Grow(1).Basis(0).Gap(7).Center().Label(tab.label).TextColor(muted).Transition(hoverFade).BorderWidth(0, 0, 2, 0).BorderColor(ui.RGBA(0, 0, 0, 0))
			if tab.active {
				b.TextColor(text).BorderColor(accent)
			} else if b.Hovered() {
				b.TextColor(text)
			}
			b.OnClick(tab.fn).Children(func() {
				ui.Text(c, tab.label).FontSize(12.5).FontWeight(600)
				if tab.count > 0 {
					ui.Text(c, fmt.Sprint(tab.count)).FontSize(10.5).TextColor(muted).Padding(1, 6).Radius(8).Background(raised)
				}
			})
		}
	})
}

func (a *app) welcome(c *ui.Context) {
	ui.Scroll(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
		ui.Column(c).FillWidth().AlignItems(ui.Center).Padding(56, 24, 32, 24).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(460).Gap(18).Children(func() {
				ui.Column(c).Gap(6).Children(func() {
					ui.Text(c, "Open a repository").FontSize(22).FontWeight(700)
					ui.Text(c, "Review changes, commit, and work on branches side by side in worktrees.").FontSize(13).TextColor(muted).Wrap()
				})
				ui.Row(c).Gap(8).Children(func() {
					ui.PrimaryButton(c, "Add repository").Disabled(a.busy).OnClick(a.pickRepository)
					ui.Button(c, "Clone repository").Disabled(a.busy).OnClick(a.startClone)
				})
				if len(a.recent) > 0 {
					ui.Column(c).FillWidth().Gap(2).Padding(10, 0, 0, 0).Children(func() {
						caption(c, "RECENT")
						ui.Box(c).Height(4)
						for i, path := range a.recent {
							if i >= 6 {
								break
							}
							c.Key("recent:" + path)
							menuRow(c, icRepo, a.repositoryName(path), filepath.Dir(path), "", "Recent: "+path, false, !a.busy, func() { a.openRepository(path) })
						}
					})
				}
			})
		})
	})
}

func (a *app) worktreeList(c *ui.Context) {
	ui.Column(c).FillWidth().Grow(1).MinHeight(0).Children(func() {
		if len(a.repo.Trees) > 6 || a.search != "" {
			ui.Column(c).FillWidth().Padding(10, 10, 4, 10).Children(func() {
				ui.TextInput(c, &a.search).Placeholder("Filter worktrees").Label("Filter worktrees").FillWidth()
			})
		}
		ui.Scroll(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(6).Gap(2).Children(func() {
				count := 0
				for _, w := range a.repo.Trees {
					if a.search != "" && !strings.Contains(strings.ToLower(w.Path+" "+w.Branch), strings.ToLower(a.search)) {
						continue
					}
					count++
					c.Key(w.Path)
					b := ui.ButtonBase(c).FillWidth().Height(50).Padding(0, 10).Gap(10).Radius(6).Justify(ui.Start).Label("Worktree: " + w.Path).Tooltip(w.Path).Disabled(a.busy).TextColor(text)
					if a.selected == w.Path {
						b.Background(accent.Alpha(0.13))
					} else if b.Hovered() {
						b.Background(raised)
					}
					b.OnClick(func() { a.selectTree(w.Path); a.managing = true }).Children(func() {
						ui.Icon(c, icBranch).FontSize(14).TextColor(muted).Shrink(0)
						ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(3).Children(func() {
							ui.Text(c, branchLabel(w)).FontSize(12.5).FontWeight(600).SingleLine().FillWidth()
							ui.Text(c, filepath.Base(w.Path)).FontSize(10.5).TextColor(faint).SingleLine().FillWidth()
						})
						if w.Locked {
							ui.Icon(c, icLock).FontSize(12).TextColor(amber).Shrink(0).Tooltip("Locked")
						}
						treeDot(c, w)
					})
				}
				if count == 0 {
					ui.Text(c, "No matching worktrees").FontSize(12).TextColor(muted).Padding(12)
				}
			})
		})
		ui.Column(c).FillWidth().Padding(10).BorderWidth(1, 0, 0, 0).BorderColor(softLine).Children(func() {
			ui.Button(c, "New worktree…").Label("New worktree").FillWidth().Disabled(a.busy).OnClick(a.startCreate)
		})
	})
}

// treeDot shows a worktree's state in a few pixels: a count of changed
// files, or a coloured dot.
func treeDot(c *ui.Context, w worktree) {
	label, color := statusLabel(w)
	if len(w.Changes) > 0 && w.StatusError == "" {
		ui.Text(c, fmt.Sprint(len(w.Changes))).FontSize(10.5).FontWeight(600).TextColor(color).Shrink(0).Tooltip(label)
		return
	}
	ui.Box(c).Size(7, 7).Radius(4).Background(color).Shrink(0).Tooltip(label)
}

func branchLabel(w worktree) string {
	if w.Bare {
		return "Bare repository"
	}
	if w.Detached {
		return "Detached HEAD"
	}
	if w.Branch == "" {
		return "No branch"
	}
	return w.Branch
}
func statusLabel(w worktree) (string, ui.Color) {
	if w.Prunable {
		return "Missing · prunable", amber
	}
	if w.Bare {
		return "Repository storage", muted
	}
	if w.StatusError != "" {
		return "Status unavailable", red
	}
	if len(w.Changes) == 1 {
		return "1 changed file", amber
	}
	if len(w.Changes) > 0 {
		return fmt.Sprintf("%d changed files", len(w.Changes)), amber
	}
	return "Clean working tree", green
}

func infoRow(c *ui.Context, label string, fn func()) {
	ui.Row(c).FillWidth().MinHeight(30).Gap(12).Padding(0, 14).BorderWidth(0, 0, 1, 0).BorderColor(softLine).Children(func() {
		ui.Text(c, label).FontSize(11.5).TextColor(muted).Width(96).Shrink(0)
		ui.Row(c).Grow(1).Basis(0).MinWidth(0).Gap(8).Children(fn)
	})
}

func (a *app) detail(c *ui.Context, w *worktree) {
	ui.Column(c).FillWidth().MaxWidth(820).Padding(24, 28).Gap(18).Children(func() {
		ui.Row(c).FillWidth().Gap(12).Children(func() {
			ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(5).Children(func() {
				ui.Text(c, branchLabel(*w)).FontSize(20).FontWeight(700).SingleLine().FillWidth().Tooltip(branchLabel(*w))
				ui.Text(c, w.Path).FontSize(12).TextColor(muted).SingleLine().FillWidth().Tooltip(w.Path)
			})
			if !w.Main && !w.Bare {
				ui.Row(c).Gap(8).Shrink(0).Children(func() {
					label, action := "Lock", "lock"
					if w.Locked {
						label, action = "Unlock", "unlock"
					}
					ui.Button(c, label).Disabled(a.busy).OnClick(func() { a.mutate(action) })
					ui.Button(c, "Remove…").TextColor(red).Disabled(a.busy || w.Locked).Tooltip("Locked worktrees cannot be removed").OnClick(func() { a.removing = true })
				})
			}
		})
		ui.Column(c).FillWidth().Background(surface).Radius(8).Border(1, line).Children(func() {
			infoRow(c, "Status", func() {
				label, color := statusLabel(*w)
				ui.Text(c, label).FontSize(12).TextColor(color)
				if w.Main {
					ui.Text(c, "· primary worktree").FontSize(12).TextColor(muted)
				}
			})
			infoRow(c, "Upstream", func() {
				switch {
				case w.Upstream != "":
					ui.Text(c, w.Upstream).FontSize(12).SingleLine().Shrink(1).MinWidth(0)
					ui.Text(c, fmt.Sprintf("↑%d  ↓%d", w.Ahead, w.Behind)).FontSize(11.5).TextColor(muted).Shrink(0)
				case w.Bare:
					ui.Text(c, "—").FontSize(12).TextColor(muted)
				default:
					ui.Text(c, "Not published").FontSize(12).TextColor(muted)
				}
			})
			if w.Head != "" {
				infoRow(c, "Commit", func() { ui.Text(c, w.Head[:min(10, len(w.Head))]).Font("monospace").FontSize(11.5).TextColor(accent) })
			}
			if w.Locked {
				infoRow(c, "Locked", func() {
					reason := w.LockReason
					if reason == "" {
						reason = "No reason given"
					}
					ui.Text(c, reason).FontSize(12).TextColor(amber).SingleLine().FillWidth()
				})
			}
			if w.Prunable {
				infoRow(c, "Missing", func() { ui.Text(c, w.PruneReason).FontSize(12).TextColor(amber).SingleLine().FillWidth() })
			}
			if w.StatusError != "" {
				infoRow(c, "Error", func() { ui.Text(c, w.StatusError).FontSize(12).TextColor(red).Wrap().FillWidth() })
			}
		})
		if len(w.Changes) > 0 {
			ui.Column(c).FillWidth().Gap(6).Children(func() {
				caption(c, "CHANGED FILES")
				for i, ch := range w.Changes {
					if i >= 8 {
						ui.Text(c, fmt.Sprintf("and %d more in Changes", len(w.Changes)-8)).FontSize(11.5).TextColor(muted)
						break
					}
					c.Key("change:" + ch.Path)
					ui.Row(c).FillWidth().Gap(10).Height(22).Children(func() {
						statusMark(c, ch.Code)
						ui.Text(c, ch.Path).FontSize(12).SingleLine().Grow(1).Basis(0).MinWidth(0).Tooltip(ch.Path)
					})
				}
			})
		}
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			caption(c, "RECENT COMMITS")
			if len(w.Commits) == 0 {
				ui.Text(c, "No commits to display.").TextColor(muted).FontSize(12)
			}
			for _, commit := range w.Commits {
				ui.Row(c).FillWidth().Gap(12).Height(22).Children(func() {
					ui.Text(c, commit.Hash).Font("monospace").FontSize(11).TextColor(accent).Width(64).Shrink(0)
					ui.Text(c, commit.Subject).FontSize(12).SingleLine().Grow(1).Basis(0).MinWidth(0).Tooltip(commit.Subject)
				})
			}
		})
	})
}

// formDialogs draws the Clone, New worktree and Remove dialogs. They stay
// open while their Git command runs.
// A modal holds the window's shortcuts, so each dialog takes Primary+W
// itself.
func (a *app) formDialogs(c *ui.Context) {
	dialog(c, &a.cloning, 540, a.busy, func() { c.OnShortcut(ui.Cmd, ui.KeyW, a.closeInner); a.cloneView(c) })
	if a.repo != nil {
		dialog(c, &a.creating, 540, a.busy, func() { c.OnShortcut(ui.Cmd, ui.KeyW, a.closeInner); a.createView(c) })
		removing := a.removing && a.managing
		dialog(c, &removing, 440, a.busy, func() { c.OnShortcut(ui.Cmd, ui.KeyW, a.closeInner); a.removeView(c) })
		if !removing {
			a.removing = false
		}
	}
}

func field(c *ui.Context, label string, value *string, placeholder string, disabled bool) {
	ui.Column(c).FillWidth().Gap(6).Children(func() {
		ui.Text(c, label).FontSize(11.5).TextColor(muted)
		ui.TextInput(c, value).Label("Input: " + label).Placeholder(placeholder).FillWidth().Disabled(disabled)
	})
}
func (a *app) createView(c *ui.Context) {
	dialogTitle(c, "New worktree", "Check out a branch in its own directory, next to your other worktrees.")
	ui.Column(c).FillWidth().Gap(6).Children(func() {
		ui.Text(c, "Mode").FontSize(11.5).TextColor(muted)
		ui.Select(c, &a.draft.Mode, []string{"New branch", "Existing branch", "Detached HEAD"}).Label("Worktree mode").FillWidth().Disabled(a.busy)
	})
	if a.draft.Mode == "Existing branch" {
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			ui.Text(c, "Local branch").FontSize(11.5).TextColor(muted)
			if len(a.repo.Branches) > 0 {
				ui.Select(c, &a.draft.Branch, a.repo.Branches).Label("Local branch").FillWidth().Disabled(a.busy)
			} else {
				ui.Text(c, "No local branches yet").FontSize(12).TextColor(faint)
			}
		})
	} else {
		if a.draft.Mode == "New branch" {
			field(c, "Branch name", &a.draft.Branch, "feature/my-next-idea", a.busy)
		}
		field(c, "Starting point", &a.draft.Base, "HEAD, main, origin/main, or a commit", a.busy)
	}
	field(c, "Directory", &a.draft.Path, "/absolute/path/to/worktree", a.busy)
	ui.Text(c, "Use a new or empty directory. A branch can be checked out in only one worktree at a time.").FontSize(11.5).TextColor(faint).Wrap()
	dialogButtons(c, func() {
		ui.Button(c, "Cancel").Disabled(a.busy).OnClick(func() { a.creating = false })
		ui.PrimaryButton(c, "Create worktree").Disabled(a.busy || a.draft.Path == "" || (a.draft.Mode != "Detached HEAD" && a.draft.Branch == "")).OnClick(a.create)
	})
}

func (a *app) removeView(c *ui.Context) {
	name := "this worktree"
	if w := a.current(); w != nil {
		name = filepath.Base(w.Path)
	}
	dialogTitle(c, "Remove "+name+"?", "The worktree directory is deleted; its branch is kept. Git refuses when there are uncommitted or untracked files.")
	dialogButtons(c, func() {
		ui.Button(c, "Cancel").AutoFocus().Disabled(a.busy).OnClick(func() { a.removing = false })
		ui.Button(c, "Remove worktree").TextColor(red).Disabled(a.busy).OnClick(func() { a.mutate("remove") })
	})
}
