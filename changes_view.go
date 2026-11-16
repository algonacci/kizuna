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
	ui.Row(c).FillWidth().MinHeight(76).Padding(12, 16).Gap(12).Background(panel).BorderWidth(1, 0, 1, 0).BorderColor(line).Children(func() {
		ui.Column(c).Width(264).Shrink(0).Gap(5).Children(func() {
			caption(c, "REPOSITORY")
			labels := []string{}
			selected := "Add a repository to get started"
			for _, path := range a.recent {
				labels = append(labels, a.repositoryLabel(path))
				if a.repo != nil && a.repo.Path == path {
					selected = a.repositoryLabel(path)
				}
			}
			ui.Select(c, &selected, labels).Label("Repository selector").FillWidth().Disabled(a.busy || len(labels) == 0).OnChange(func() {
				for _, path := range a.recent {
					if a.repositoryLabel(path) == selected {
						a.openRepository(path)
						return
					}
				}
			})
		})
		ui.Column(c).Grow(1).MinWidth(0).Gap(5).Children(func() {
			caption(c, "CURRENT WORKTREE")
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
			ui.Select(c, &selected, labels).Label("Worktree selector").FillWidth().Disabled(a.busy || a.repo == nil).OnChange(func() {
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
		ui.Button(c, "Add…").Label("Add repository").Tooltip("Add a local repository").Disabled(a.busy).OnClick(a.pickRepository)
		ui.Button(c, "Clone…").Label("Clone repository").Disabled(a.busy).OnClick(a.startClone)
	})
	if a.repo != nil && !a.cloning {
		ui.Row(c).FillWidth().Height(46).Padding(0, 16).Gap(8).Background(panel).Children(func() {
			caption(c, "REMOTE")
			ui.Select(c, &a.remote, a.repo.Remotes).Label("Remote selector").Width(145).Disabled(a.busy || len(a.repo.Remotes) == 0)
			w := a.current()
			disabled := a.busy || w == nil || w.Bare || w.Prunable || a.remote == ""
			ui.Button(c, "Fetch").Disabled(disabled).OnClick(func() { a.remoteAction("fetch") })
			ui.Button(c, "Pull").Disabled(disabled || (w != nil && w.Detached)).OnClick(func() { a.remoteAction("pull") })
			ui.PrimaryButton(c, "Push").Disabled(disabled || (w != nil && w.Detached)).OnClick(func() { a.remoteAction("push") })
			ui.Box(c).Grow(1)
			if w != nil {
				label := a.repo.identity(a.remote)
				ui.Text(c, label).FontSize(12).TextColor(accent).SingleLine().MinWidth(0).Tooltip(a.repo.RemoteURLs[a.remote])
			}
		})
	}
}

func (a *app) changesSidebar(c *ui.Context) {
	w := a.current()
	if w == nil {
		return
	}
	_, height := c.Size()
	compact := height < 700
	ui.Column(c).Width(300).Shrink(0).MinHeight(0).Background(panel).BorderWidth(0, 1, 0, 0).BorderColor(line).Children(func() {
		ui.Row(c).FillWidth().Height(48).Padding(0, 16).Gap(10).BorderWidth(0, 0, 1, 0).BorderColor(line).Children(func() {
			ui.Text(c, "Changes").Bold().Grow(1)
			badge(c, fmt.Sprint(len(w.Changes)), muted)
		})
		ui.Row(c).FillWidth().Padding(8, 12).Gap(6).Children(func() {
			all := len(w.Changes) > 0 && len(a.selectedChanges()) == len(w.Changes)
			ui.Checkbox(c, &all, "Select all").Disabled(a.busy || len(w.Changes) == 0).OnChange(func() {
				for _, ch := range w.Changes {
					a.checked[ch.Path] = all
				}
			})
			ui.Text(c, fmt.Sprintf("%d selected", len(a.selectedChanges()))).FontSize(11).TextColor(muted)
		})
		ui.Scroll(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Gap(1).Children(func() {
				for _, ch := range w.Changes {
					c.Key("file:" + ch.Path)
					row := ui.Row(c).FillWidth().Height(40).Padding(0, 12).Gap(8)
					if a.file == ch.Path {
						row.Background(raised)
					}
					row.Children(func() {
						checked := a.checked[ch.Path]
						ui.Checkbox(c, &checked, "").Label("Include: " + ch.Path).Disabled(a.busy).OnChange(func() { a.checked[ch.Path] = checked })
						b := ui.ButtonBase(c).Grow(1).MinWidth(0).FillHeight().Justify(ui.Start).Label("Change: " + ch.Path).Tooltip(ch.Path)
						if b.Hovered() {
							b.Background(raised)
						}
						b.OnClick(func() { a.selectFile(ch.Path) }).Children(func() {
							ui.Text(c, ch.Path).FontSize(12).SingleLine().Grow(1).MinWidth(0)
							ui.Text(c, strings.TrimSpace(ch.Code)).FontSize(11).TextColor(amber).Shrink(0)
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
		ui.Column(c).FillWidth().Padding(padding).Gap(gap).BorderWidth(1, 0, 0, 0).BorderColor(line).Children(func() {
			ui.Text(c, "Commit to "+branchLabel(*w)).FontSize(12).Bold().SingleLine().FillWidth().Tooltip(branchLabel(*w))
			ui.TextInput(c, &a.summary).Label("Commit summary").Placeholder("Summary (required)").FillWidth().Disabled(a.busy)
			descriptionHeight := float32(66)
			if compact {
				descriptionHeight = 36
			}
			ui.TextArea(c, &a.description).Label("Commit description").Placeholder("Description").FillWidth().Height(descriptionHeight).MinHeight(descriptionHeight).Disabled(a.busy)
			ui.PrimaryButton(c, "Commit selected files").FillWidth().Disabled(a.busy || w.Bare || w.Prunable || w.StatusError != "" || strings.TrimSpace(a.summary) == "" || len(a.selectedChanges()) == 0).OnClick(a.commitSelected)
			ui.Text(c, "Commits the full contents of checked files.").FontSize(10).TextColor(muted)
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
		ui.Row(c).FillWidth().Height(48).Padding(0, 18).Gap(12).BorderWidth(0, 0, 1, 0).BorderColor(line).Children(func() {
			ui.Text(c, a.file).FontSize(13).Bold().SingleLine().Grow(1).MinWidth(0).Tooltip(a.file)
			badge(c, "WORKING TREE ↔ HEAD", muted)
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
					if i >= 2500 {
						ui.Text(c, "Preview limited to 2,500 lines.").Padding(16).TextColor(amber)
						break
					}
					color, bg := text, base
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
		field(c, "REPOSITORY URL", &a.cloneURL, "https://github.com/owner/repo.git or git@github.com:owner/repo.git", a.busy)
		field(c, "CLONE DESTINATION", &a.clonePath, "/absolute/path/to/new-folder", a.busy)
		ui.Text(c, "Uses your existing Git credentials and SSH agent.").FontSize(12).TextColor(muted)
		ui.Row(c).Gap(8).Children(func() {
			ui.PrimaryButton(c, "Clone and open").Disabled(a.busy || a.cloneURL == "" || a.clonePath == "").OnClick(a.cloneRepository)
			ui.Button(c, "Cancel").Disabled(a.busy).OnClick(func() { a.cloning = false })
		})
	})
}
