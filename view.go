package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *app) view(c *ui.Context) {
	defer a.navigationHelp(c)
	c.OnShortcut(ui.Cmd, ui.KeySlash, func() { a.keyboardHelp = true })
	c.SetTheme(appTheme())
	c.OnShortcut(ui.Cmd, ui.KeyO, a.pickRepository)
	c.OnShortcut(ui.Cmd, ui.KeyN, a.startCreate)
	c.OnShortcut(ui.Cmd, ui.KeyR, a.refresh)
	ui.Column(c).Fill().Background(base).Children(func() {
		a.toolbar(c)
		ui.Row(c).Grow(1).MinHeight(0).FillWidth().AlignItems(ui.Stretch).Children(func() {
			ui.Column(c).Width(286).Shrink(0).MinHeight(0).Background(panel).BorderWidth(0, 1, 0, 0).BorderColor(softLine).Children(func() {
				a.repositoryBar(c)
				if a.repo == nil || a.managing {
					a.sidebar(c)
				} else {
					a.changesSidebar(c)
				}
			})
			ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Children(func() {
				if a.repo != nil {
					a.contextBar(c)
				}
				if a.cloning {
					ui.Scroll(c).Fill().Children(func() { ui.Column(c).FillWidth().Padding(28).Gap(22).Children(func() { a.cloneView(c) }) })
				} else if a.repo == nil {
					a.welcome(c)
				} else if a.creating || a.managing {
					ui.Scroll(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
						ui.Column(c).FillWidth().Padding(28).Gap(22).Children(func() {
							if a.creating {
								a.createView(c)
							} else if w := a.current(); w != nil {
								a.detail(c, w)
							}
						})
					})
				} else {
					ui.Column(c).FillWidth().Grow(1).MinHeight(0).Padding(12, 14, 14, 14).Children(func() {
						ui.Column(c).Fill().Background(canvas).Border(1, softLine).Radius(10).Children(func() { a.diffView(c) })
					})
				}
			})
		})
		ui.Row(c).FillWidth().MinHeight(26).Padding(5, 14).Gap(10).Background(panel).BorderWidth(1, 0, 0, 0).BorderColor(softLine).Children(func() {
			color := muted
			if a.errorMessage {
				color = red
			} else if a.busy {
				color = accent
			}
			ui.Box(c).Size(5, 5).Radius(3).Background(color)
			ui.Text(c, a.message).FontSize(10.5).TextColor(color).Grow(1).MinWidth(0)
			if a.busy {
				ui.Text(c, "WORKING").FontSize(10).TextColor(accent).Shrink(0)
			}
		})
	})
}

func (a *app) sidebar(c *ui.Context) {
	ui.Column(c).FillWidth().Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).FillWidth().Padding(14).Gap(10).Children(func() {
			caption(c, "WORKTREES")
			if a.repo != nil {
				ui.TextInput(c, &a.search).Placeholder("Filter worktrees…").Label("Filter worktrees").FillWidth()
			}
		})
		ui.Scroll(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(0, 10).Gap(5).Children(func() {
				if a.repo != nil {
					count := 0
					for _, w := range a.repo.Trees {
						if a.search != "" && !strings.Contains(strings.ToLower(w.Path+" "+w.Branch), strings.ToLower(a.search)) {
							continue
						}
						count++
						c.Key(w.Path)
						b := ui.ButtonBase(c).FillWidth().MinHeight(76).Padding(12).Radius(7).Justify(ui.Start).Label("Worktree: " + w.Path).Tooltip(w.Path).Disabled(a.busy)
						if a.selected == w.Path && !a.creating {
							b.Background(accent.Alpha(0.10)).BorderWidth(0, 0, 0, 2).BorderColor(accent)
						} else if b.Hovered() {
							b.Background(raised)
						}
						b.OnClick(func() { a.selectTree(w.Path); a.managing = true }).Children(func() {
							ui.Column(c).Grow(1).MinWidth(0).Gap(7).Children(func() {
								ui.Row(c).FillWidth().Gap(8).Children(func() {
									ui.Icon(c, icBranch).FontSize(15).TextColor(accent)
									ui.Text(c, branchLabel(w)).FontSize(13).SingleLine().Grow(1).MinWidth(0)
									if w.Locked {
										ui.Text(c, "L").FontSize(10).TextColor(amber).Tooltip("Locked")
									}
								})
								ui.Text(c, filepath.Base(w.Path)).FontSize(11).TextColor(muted).SingleLine().FillWidth()
								label, color := statusLabel(w)
								ui.Text(c, label).FontSize(11).TextColor(color)
							})
						})
					}
					if count == 0 {
						ui.Text(c, "No matching worktrees").FontSize(12).TextColor(muted).Padding(10)
					}
				}
			})
		})
		if len(a.recent) > 0 {
			ui.Column(c).FillWidth().Padding(18).Gap(8).BorderWidth(1, 0, 0, 0).BorderColor(line).Children(func() {
				caption(c, "RECENT REPOSITORIES")
				for i, path := range a.recent {
					if i >= 4 {
						break
					}
					name := filepath.Base(path)
					if name == ".git" {
						name = filepath.Base(filepath.Dir(path))
					}
					if identity := a.repoIdentities[path]; identity != "" {
						name = identity
					}
					c.Key("recent:" + path)
					ui.Button(c, name).FillWidth().Label("Recent: " + path).Tooltip(path).Disabled(a.busy).OnClick(func() { a.openRepository(path) })
				}
			})
		}
	})
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
	if len(w.Changes) > 0 {
		return fmt.Sprintf("%d changed files", len(w.Changes)), amber
	}
	return "Clean working tree", green
}
func (a *app) toolbar(c *ui.Context) {
	inset := c.TitleBar()
	ui.Row(c).FillWidth().Height(max(inset.Height, 68)).Padding(0, inset.Right+10, 0, inset.Left+12).Gap(2).Background(panel).BorderWidth(0, 0, 1, 0).BorderColor(softLine).DragWindow().Children(func() {
		ui.Row(c).Gap(8).Padding(0, 10, 0, 0).Children(func() {
			ui.Box(c).Size(28, 28).Radius(9).LinearGradient(ui.LinearGradient{From: ui.Hex("#779ef5"), To: ui.Hex("#a28aef"), Angle: 135}).Center().Children(func() { ui.Icon(c, icBranch).FontSize(17).TextColor(ui.Hex("#ffffff")) })
			width, _ := c.Size()
			if width >= 1000 {
				ui.Text(c, "Kizuna").FontSize(14).FontWeight(700)
			}
		})
		toolSeparator(c)
		tool(c, icFolder, "Add repo", "Add repository", !a.busy, false, a.pickRepository)
		tool(c, icClone, "Clone", "Clone repository", !a.busy, a.cloning, a.startClone)
		toolSeparator(c)
		tool(c, icPlus, "New tree", "+ New worktree", !a.busy && a.repo != nil, a.creating, a.startCreate)
		tool(c, icRefresh, "Refresh", "Refresh", !a.busy && a.repo != nil && !a.creating && !a.removing, false, a.refresh)
		ui.Box(c).Grow(1)
		w := a.current()
		enabled := !a.busy && w != nil && !w.Bare && !w.Prunable && a.remote != ""
		tool(c, icFetch, "Fetch", "Fetch", enabled, false, func() { a.remoteAction("fetch") })
		tool(c, icPull, "Pull", "Pull", enabled && (w == nil || !w.Detached), false, func() { a.remoteAction("pull") })
		tool(c, icPush, "Push", "Push", enabled && (w == nil || !w.Detached), w != nil && w.Ahead > 0, func() { a.remoteAction("push") })
		toolSeparator(c)
		tool(c, icFolder, "Folder", "Open folder", !a.busy && w != nil && !w.Prunable, false, func() { a.openSelected(false) })
		tool(c, icTerminal, "Terminal", "Open terminal", !a.busy && w != nil && !w.Prunable && !w.Bare, false, func() { a.openSelected(true) })
	})
}
func (a *app) welcome(c *ui.Context) {
	ui.Column(c).Fill().Center().Padding(40).Gap(18).Children(func() {
		ui.Icon(c, icBranch).FontSize(66).TextColor(accent)
		ui.Text(c, "Parallel branches. One home.").FontSize(27).Bold()
		ui.Text(c, "Keep a fix, a feature, and your main branch open at once.\nKizuna gives every worktree a place of its own.").TextColor(muted).FontSize(14)
		ui.PrimaryButton(c, "Open Git repository").Margin(12, 0).Disabled(a.busy).OnClick(a.pickRepository)
		caption(c, "LOCAL FIRST   /   NATIVE UI   /   POWERED BY GIT")
	})
}
func (a *app) detail(c *ui.Context, w *worktree) {
	ui.Column(c).FillWidth().Gap(12).Children(func() {
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			badge(c, "WORKTREE", accent)
			if w.Main {
				badge(c, "PRIMARY", muted)
			}
			if w.Locked {
				badge(c, "LOCKED", amber)
			}
			if w.Detached {
				badge(c, "DETACHED", amber)
			}
		})
		ui.Text(c, branchLabel(*w)).FontSize(28).Bold().SingleLine().FillWidth().MinWidth(0).Tooltip(branchLabel(*w))
		ui.Text(c, w.Path).FontSize(12).TextColor(muted).SingleLine().FillWidth().MinWidth(0).Tooltip(w.Path)
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			ui.Box(c).Grow(1)
			if !w.Main && !w.Bare {
				label, action := "Lock", "lock"
				if w.Locked {
					label, action = "Unlock", "unlock"
				}
				ui.Button(c, label).Disabled(a.busy).OnClick(func() { a.mutate(action) })
				ui.Button(c, "Remove…").TextColor(red).Disabled(a.busy || w.Locked).OnClick(func() { a.removing = true })
			}
		})
	})
	if a.removing {
		ui.Column(c).FillWidth().Padding(18).Gap(10).Background(raised).Radius(8).Border(1, red).Children(func() {
			ui.Text(c, "Remove this worktree?").Bold().TextColor(red)
			ui.Text(c, "The worktree directory will be deleted. Its branch is kept. Git refuses removal when there are uncommitted or untracked files.").FontSize(12)
			ui.Row(c).Gap(8).Children(func() {
				ui.Button(c, "Cancel").Disabled(a.busy).OnClick(func() { a.removing = false })
				ui.Button(c, "Remove worktree").TextColor(red).Disabled(a.busy).OnClick(func() { a.mutate("remove") })
			})
		})
	}
	ui.Column(c).FillWidth().Padding(18).Gap(10).Background(panel).Radius(8).Border(1, line).Children(func() {
		label, color := statusLabel(*w)
		ui.Row(c).FillWidth().Gap(10).Children(func() {
			ui.Text(c, "● "+label).TextColor(color).Grow(1)
			if w.Head != "" {
				badge(c, w.Head[:min(10, len(w.Head))], muted)
			}
		})
		if w.Upstream != "" {
			ui.Text(c, fmt.Sprintf("%s   ·   %d ahead  /  %d behind", w.Upstream, w.Ahead, w.Behind)).FontSize(12).TextColor(muted)
		} else if !w.Bare {
			caption(c, "No upstream configured")
		}
		if w.Locked {
			ui.Text(c, "Lock: "+w.LockReason).FontSize(12).TextColor(amber)
		}
		if w.Prunable {
			ui.Text(c, w.PruneReason).FontSize(12).TextColor(amber)
		}
		if w.StatusError != "" {
			ui.Text(c, w.StatusError).FontSize(12).TextColor(red)
		}
	})
	caption(c, "WORKING DIRECTORY")
	if len(w.Changes) == 0 {
		label := "All clear. Your working directory has no changes."
		if w.Bare || w.Prunable || w.StatusError != "" {
			label = "Working directory status is unavailable."
		}
		ui.Text(c, label).TextColor(muted).FontSize(13)
	} else {
		ui.Column(c).FillWidth().Gap(2).Children(func() {
			for i, ch := range w.Changes {
				if i >= 100 {
					caption(c, fmt.Sprintf("+ %d more files", len(w.Changes)-100))
					break
				}
				c.Key("change:" + ch.Path)
				ui.Row(c).FillWidth().Padding(9, 12).Gap(12).Background(panel).Radius(4).Children(func() {
					ui.Text(c, ch.Code).FontSize(11).TextColor(amber).Width(26).Shrink(0)
					path := ch.Path
					if ch.From != "" {
						path = ch.From + " → " + ch.Path
					}
					ui.Text(c, path).FontSize(12).SingleLine().Grow(1).MinWidth(0).Tooltip(path)
				})
			}
		})
	}
	caption(c, "RECENT COMMITS")
	if len(w.Commits) == 0 {
		ui.Text(c, "No commits to display.").TextColor(muted).FontSize(12)
	}
	ui.Column(c).FillWidth().Gap(12).Children(func() {
		for _, commit := range w.Commits {
			ui.Row(c).FillWidth().Gap(12).Children(func() {
				ui.Text(c, commit.Hash).FontSize(11).TextColor(accent).Width(70).Shrink(0)
				ui.Text(c, commit.Subject).FontSize(12).SingleLine().Grow(1).MinWidth(0).Tooltip(commit.Subject)
			})
		}
	})
}
func field(c *ui.Context, label string, value *string, placeholder string, disabled bool) {
	ui.Column(c).FillWidth().Gap(7).Children(func() {
		caption(c, label)
		ui.TextInput(c, value).Label("Input: " + label).Placeholder(placeholder).FillWidth().Disabled(disabled)
	})
}
func (a *app) createView(c *ui.Context) {
	ui.Text(c, "A new place to work.").FontSize(27).Bold()
	ui.Text(c, "Check out a branch in its own directory, alongside your existing worktrees.").TextColor(muted)
	ui.Column(c).FillWidth().MaxWidth(620).Gap(20).Children(func() {
		caption(c, "MODE")
		ui.Select(c, &a.draft.Mode, []string{"New branch", "Existing branch", "Detached HEAD"}).Label("Worktree mode").FillWidth().Disabled(a.busy)
		if a.draft.Mode == "Existing branch" {
			caption(c, "LOCAL BRANCH")
			if len(a.repo.Branches) > 0 {
				ui.Select(c, &a.draft.Branch, a.repo.Branches).Label("Local branch").FillWidth().Disabled(a.busy)
			} else {
				caption(c, "No local branches yet")
			}
		} else {
			if a.draft.Mode == "New branch" {
				field(c, "BRANCH NAME", &a.draft.Branch, "feature/my-next-idea", a.busy)
			}
			field(c, "STARTING REF", &a.draft.Base, "HEAD, main, origin/main, or a commit", a.busy)
		}
		field(c, "DESTINATION DIRECTORY", &a.draft.Path, "/absolute/path/to/worktree", a.busy)
		ui.Text(c, "Use a new or empty directory. A branch can only be checked out in one worktree at a time.").FontSize(12).TextColor(muted)
		ui.Row(c).Gap(10).Children(func() {
			ui.PrimaryButton(c, "Create worktree").Disabled(a.busy || a.draft.Path == "" || (a.draft.Mode != "Detached HEAD" && a.draft.Branch == "")).OnClick(a.create)
			ui.Button(c, "Cancel").Disabled(a.busy).OnClick(func() { a.creating = false })
		})
	})
}
