package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *app) repositoryLabel(path string) string {
	name := filepath.Base(path)
	if name == ".git" {
		name = filepath.Base(filepath.Dir(path))
	}
	if id := a.repoIdentities[path]; id != "" {
		name = id
	}
	return name + "  ·  " + filepath.Dir(path)
}
func (a *app) repositoryBar(c *ui.Context) {
	ui.Column(c).FillWidth().Padding(18, 14, 12, 14).Gap(9).Children(func() {
		caption(c, "REPOSITORY")
		labels := []string{}
		selected := "Add a repository to get started"
		for _, path := range a.recent {
			labels = append(labels, a.repositoryLabel(path))
			if a.repo != nil && a.repo.Path == path {
				selected = a.repositoryLabel(path)
			}
		}
		display := "Choose a repository"
		if a.repo != nil {
			display = a.repo.identity("origin")
		}
		picker(c, &selected, labels, "Repository selector", display, !a.busy && len(labels) > 0, func() {
			for _, path := range a.recent {
				if a.repositoryLabel(path) == selected {
					a.openRepository(path)
					return
				}
			}
		})
		if a.repo != nil {
			ui.Text(c, filepath.Dir(a.repo.Path)).FontSize(10.5).TextColor(faint).SingleLine().FillWidth().Tooltip(a.repo.Path)
			ui.Row(c).FillWidth().Gap(4).Padding(10, 0, 0, 0).Children(func() {
				for _, tab := range []struct {
					label  string
					ic     *ui.SVG
					active bool
					fn     func()
				}{
					{"Changes", icChanges, !a.managing, func() { a.managing = false; a.creating = false; a.cloning = false }},
					{"Worktrees", icTree, a.managing, func() { a.managing = true; a.creating = false; a.cloning = false }},
				} {
					b := ui.ButtonBase(c).Grow(1).Height(32).Radius(6).Gap(7).Center().Label(tab.label).Disabled(a.busy).TextColor(muted)
					if tab.active {
						b.Background(raised).TextColor(text)
					} else if b.Hovered() {
						b.Background(softLine)
					}
					b.OnClick(tab.fn).Children(func() { ui.Icon(c, tab.ic).FontSize(14); ui.Text(c, tab.label).FontSize(11.5) })
				}
			})
		}
	})
}

func (a *app) contextBar(c *ui.Context) {
	ui.Row(c).FillWidth().Height(68).Padding(0, 18).Gap(12).Children(func() {
		ui.Icon(c, icBranch).FontSize(18).TextColor(accent)
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			caption(c, "WORKTREE")
			labels := []string{}
			selected := "No worktree"
			if a.repo != nil {
				for _, w := range a.repo.Trees {
					label := branchLabel(w) + "  ·  " + w.Path
					labels = append(labels, label)
					if w.Path == a.selected {
						selected = label
					}
				}
			}
			display := "No worktree"
			if w := a.current(); w != nil {
				display = branchLabel(*w)
			}
			picker(c, &selected, labels, "Worktree selector", display, !a.busy && a.repo != nil, func() {
				if a.repo != nil {
					for _, w := range a.repo.Trees {
						if branchLabel(w)+"  ·  "+w.Path == selected {
							a.selectTree(w.Path)
							return
						}
					}
				}
			})
		})
		ui.Column(c).Width(110).Shrink(0).MinWidth(0).Gap(4).Children(func() {
			caption(c, "REMOTE")
			picker(c, &a.remote, a.repo.Remotes, "Remote selector", a.remote, !a.busy && len(a.repo.Remotes) > 0, func() {})
		})
		if w := a.current(); w != nil {
			ui.Text(c, fmt.Sprintf("↑ %d  ↓ %d", w.Ahead, w.Behind)).FontSize(11).TextColor(muted).Tooltip(w.Upstream).Shrink(0)
		}
	})
}

func (a *app) changesSidebar(c *ui.Context) {
	w := a.current()
	if w == nil {
		return
	}
	_, height := c.Size()
	compact := height < 700
	ui.Column(c).FillWidth().Grow(1).MinHeight(0).Children(func() {
		ui.Row(c).FillWidth().Padding(10, 14).Gap(6).BorderWidth(1, 0, 1, 0).BorderColor(softLine).Children(func() {
			all := len(w.Changes) > 0 && len(a.selectedChanges()) == len(w.Changes)
			ui.Checkbox(c, &all, "").Label("Select all").Disabled(a.busy || len(w.Changes) == 0).OnChange(func() {
				for _, ch := range w.Changes {
					a.checked[ch.Path] = all
				}
			})
			ui.Text(c, fmt.Sprintf("%d changed files", len(w.Changes))).FontSize(11.5).TextColor(muted).Grow(1)
		})
		ui.Scroll(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(6).Gap(2).Children(func() {
				for _, ch := range w.Changes {
					c.Key("file:" + ch.Path)
					row := ui.Row(c).FillWidth().Height(46).Padding(0, 8).Gap(9).Radius(6)
					if a.file == ch.Path {
						row.Background(accent.Alpha(0.12))
					}
					row.Children(func() {
						checked := a.checked[ch.Path]
						ui.Checkbox(c, &checked, "").Label("Include: " + ch.Path).Disabled(a.busy).OnChange(func() { a.checked[ch.Path] = checked })
						b := ui.ButtonBase(c).Grow(1).MinWidth(0).FillHeight().Justify(ui.Start).Label("Change: " + ch.Path).Tooltip(ch.Path)
						if b.Hovered() {
							b.Background(raised)
						}
						b.OnClick(func() { a.selectFile(ch.Path) }).Children(func() {
							ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
								ui.Text(c, filepath.Base(ch.Path)).FontSize(12).SingleLine().FillWidth()
								if dir := filepath.Dir(ch.Path); dir != "." {
									ui.Text(c, dir).FontSize(10).TextColor(faint).SingleLine().FillWidth()
								}
							})
							badge(c, strings.TrimSpace(ch.Code), amber)
						})
					})
				}
				if len(w.Changes) == 0 {
					ui.Text(c, "No local changes").FontSize(12).TextColor(muted).Padding(16)
				}
			})
		})
		padding, gap := float32(14), float32(10)
		if compact {
			padding, gap = 10, 6
		}
		ui.Column(c).FillWidth().Padding(padding).Gap(gap).BorderWidth(1, 0, 0, 0).BorderColor(softLine).Children(func() {
			ui.Row(c).FillWidth().Gap(6).Children(func() {
				caption(c, "COMMIT")
				ui.Box(c).Grow(1)
				ui.Text(c, fmt.Sprintf("%d files selected", len(a.selectedChanges()))).FontSize(10).TextColor(faint)
			})
			ui.TextInput(c, &a.summary).Label("Commit summary").Placeholder("Summary (required)").FillWidth().Disabled(a.busy)
			descriptionHeight := float32(66)
			if compact {
				descriptionHeight = 36
			}
			ui.TextArea(c, &a.description).Label("Commit description").Placeholder("Description").FillWidth().Height(descriptionHeight).MinHeight(descriptionHeight).Disabled(a.busy)
			ui.PrimaryButton(c, "Commit selected files").FillWidth().Disabled(a.busy || w.Bare || w.Prunable || w.StatusError != "" || strings.TrimSpace(a.summary) == "" || len(a.selectedChanges()) == 0).OnClick(a.commitSelected)
			ui.Row(c).FillWidth().Gap(6).Children(func() {
				ui.Icon(c, icBranch).FontSize(11).TextColor(faint)
				ui.Text(c, branchLabel(*w)).FontSize(10.5).TextColor(muted).SingleLine().Grow(1).MinWidth(0).Tooltip(w.Path)
			})
		})
	})
}

func (a *app) diffView(c *ui.Context) {
	w := a.current()
	if w == nil {
		return
	}
	if a.file == "" {
		ui.Column(c).Fill().Center().Padding(32).Gap(16).Children(func() {
			ui.Icon(c, icBranch).FontSize(48).TextColor(accent)
			label, color := statusLabel(*w)
			ui.Text(c, label).FontSize(24).Bold().TextColor(color)
			if w.StatusError != "" {
				ui.Text(c, w.StatusError).FontSize(12).TextColor(red)
			}
			ui.Text(c, w.Path).FontSize(12).TextColor(muted).SingleLine().FillWidth().Tooltip(w.Path)
			ui.Text(c, "Pick another worktree, pull the latest commits, or start something new.").FontSize(13).TextColor(muted)
		})
		return
	}
	ui.Column(c).Fill().Children(func() {
		ui.Row(c).FillWidth().Height(48).Padding(0, 18).Gap(10).BorderWidth(0, 0, 1, 0).BorderColor(softLine).Children(func() {
			ui.Icon(c, icChanges).FontSize(16).TextColor(muted)
			ui.Text(c, a.file).FontSize(12).FontWeight(600).SingleLine().Grow(1).MinWidth(0).Tooltip(a.file)
			added, removed := 0, 0
			for _, s := range strings.Split(a.diff, "\n") {
				if strings.HasPrefix(s, "+") && !strings.HasPrefix(s, "+++") {
					added++
				}
				if strings.HasPrefix(s, "-") && !strings.HasPrefix(s, "---") {
					removed++
				}
			}
			ui.Text(c, fmt.Sprintf("+%d", added)).Font("monospace").FontSize(11).TextColor(green)
			ui.Text(c, fmt.Sprintf("−%d", removed)).Font("monospace").FontSize(11).TextColor(red)
		})
		c.Key("diff:" + a.selected + ":" + a.file)
		ui.Scroll(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(8, 0).Children(func() {
				if a.diffLoading {
					ui.Text(c, "Loading diff…").TextColor(muted).Padding(20)
					return
				}
				lines := strings.Split(a.diff, "\n")
				oldLine, newLine := 0, 0
				inHunk := false
				for i, s := range lines {
					if strings.HasPrefix(s, "diff --git ") {
						inHunk = false
						continue
					}
					if !inHunk && (strings.HasPrefix(s, "index ") || strings.HasPrefix(s, "--- ") || strings.HasPrefix(s, "+++ ")) {
						continue
					}
					if i >= 2500 {
						ui.Text(c, "Preview limited to 2,500 lines.").Padding(16).TextColor(amber)
						break
					}
					color, bg := text, canvas
					oldNumber, newNumber := "", ""
					if strings.HasPrefix(s, "@@ ") {
						parts := strings.Fields(s)
						if len(parts) >= 3 {
							fmt.Sscanf(strings.Split(parts[1], ",")[0], "-%d", &oldLine)
							fmt.Sscanf(strings.Split(parts[2], ",")[0], "+%d", &newLine)
							inHunk = true
						}
					} else if strings.HasPrefix(s, "diff ") {
						inHunk = false
					} else if inHunk {
						switch {
						case strings.HasPrefix(s, "+"):
							newNumber = fmt.Sprint(newLine)
							newLine++
						case strings.HasPrefix(s, "-"):
							oldNumber = fmt.Sprint(oldLine)
							oldLine++
						case strings.HasPrefix(s, " "):
							oldNumber, newNumber = fmt.Sprint(oldLine), fmt.Sprint(newLine)
							oldLine++
							newLine++
						}
					}
					if len(s) > 4000 {
						s = s[:4000] + " … [long line truncated]"
					}
					switch {
					case strings.HasPrefix(s, "+++") || strings.HasPrefix(s, "---"):
						color = muted
					case strings.HasPrefix(s, "+"):
						color = green
						bg = ui.Hex("#152a24")
					case strings.HasPrefix(s, "-"):
						color = red
						bg = ui.Hex("#301e25")
					case strings.HasPrefix(s, "@@"):
						color = accent
						bg = raised
					case strings.HasPrefix(s, "diff ") || strings.HasPrefix(s, "index "):
						color = muted
					}
					// Wrap long lines instead of clipping code off the edge of the viewport.
					ui.Row(c).FillWidth().MinHeight(22).Padding(2, 16).Gap(12).Background(bg).AlignItems(ui.Start).Children(func() {
						ui.Text(c, oldNumber).Font("monospace").FontSize(10).TextColor(muted).Width(28).Shrink(0)
						ui.Text(c, newNumber).Font("monospace").FontSize(10).TextColor(muted).Width(28).Shrink(0)
						ui.Text(c, s).Font("monospace").FontSize(12).TextColor(color).Grow(1).MinWidth(0)
					})
				}
			})
		})
	})
}

func (a *app) cloneView(c *ui.Context) {
	ui.Text(c, "Bring a repository home.").FontSize(27).Bold()
	ui.Text(c, "Clone a remote repository, then work across branches in their own directories.").TextColor(muted)
	ui.Column(c).FillWidth().MaxWidth(620).Gap(20).Children(func() {
		ui.Column(c).FillWidth().Gap(7).Children(func() {
			caption(c, "REPOSITORY URL")
			ui.TextInput(c, &a.cloneURL).Label("Input: REPOSITORY URL").Placeholder("https://github.com/owner/repo.git").FillWidth().Disabled(a.busy).OnChange(a.updateClonePath)
		})
		ui.Column(c).FillWidth().Gap(7).Children(func() {
			caption(c, "CLONE DESTINATION")
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				ui.TextInput(c, &a.clonePath).Label("Input: CLONE DESTINATION").Placeholder("Paste a repository URL to fill the destination").Grow(1).MinWidth(0).Disabled(a.busy)
				ui.Button(c, "Choose…").Label("Choose clone folder").Tooltip("Choose the parent folder; the repository name is added automatically").Shrink(0).Disabled(a.busy).OnClick(a.chooseCloneFolder)
			})
			ui.Text(c, "Parent folder: "+a.cloneParent).FontSize(11).TextColor(muted).SingleLine().FillWidth().Tooltip(a.cloneParent)
		})
		ui.Text(c, "Uses your existing Git credentials and SSH agent.").FontSize(12).TextColor(muted)
		ui.Row(c).Gap(8).Children(func() {
			ui.PrimaryButton(c, "Clone and open").Disabled(a.busy || a.cloneURL == "" || a.clonePath == "").OnClick(a.cloneRepository)
			ui.Button(c, "Cancel").Disabled(a.busy).OnClick(func() { a.cloning = false })
		})
	})
}
