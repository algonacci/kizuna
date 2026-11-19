package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *app) commitLabel(w *worktree) string {
	if w.Detached {
		return "Commit to detached HEAD"
	}
	name := []rune(branchLabel(*w))
	if len(name) > 30 {
		name = append(name[:29], '…')
	}
	return "Commit to " + string(name)
}

func (a *app) changesSidebar(c *ui.Context) {
	w := a.current()
	if w == nil {
		return
	}
	_, height := c.Size()
	compact := height < 680
	rowHeight := float32(32)
	if compact {
		rowHeight = 28
	}
	selected := len(a.selectedChanges())
	ui.Column(c).FillWidth().Grow(1).MinHeight(0).Children(func() {
		ui.Row(c).FillWidth().Height(36).Shrink(0).Padding(0, 14).Gap(10).BorderWidth(0, 0, 1, 0).BorderColor(softLine).Children(func() {
			all := len(w.Changes) > 0 && selected == len(w.Changes)
			ui.Checkbox(c, &all, "").Label("Select all").Disabled(a.busy || len(w.Changes) == 0).OnChange(func() {
				for _, ch := range w.Changes {
					a.checked[ch.Path] = all
				}
			})
			summary := fmt.Sprintf("%d changed files", len(w.Changes))
			switch {
			case len(w.Changes) == 0:
				summary = "No changed files"
			case len(w.Changes) == 1:
				summary = "1 changed file"
			case len(w.Changes) > 0 && selected != len(w.Changes):
				summary = fmt.Sprintf("%d of %d files selected", selected, len(w.Changes))
			}
			ui.Text(c, summary).FontSize(11.5).TextColor(muted).SingleLine().Grow(1).Basis(0).MinWidth(0)
		})
		ui.Scroll(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(4, 0).Children(func() {
				for _, ch := range w.Changes {
					c.Key("file:" + ch.Path)
					row := ui.Row(c).FillWidth().Height(rowHeight).Padding(0, 14).Gap(10)
					if a.file == ch.Path {
						row.Background(accent.Alpha(0.14))
					}
					row.Children(func() {
						checked := a.checked[ch.Path]
						ui.Checkbox(c, &checked, "").Label("Include: " + ch.Path).Disabled(a.busy).OnChange(func() { a.checked[ch.Path] = checked })
						path := ch.Path
						if ch.From != "" {
							path = ch.From + " → " + ch.Path
						}
						b := ui.ButtonBase(c).Grow(1).Basis(0).MinWidth(0).FillHeight().Gap(8).Justify(ui.Start).Label("Change: " + ch.Path).Tooltip(path).TextColor(text)
						b.OnClick(func() { a.selectFile(ch.Path) }).Children(func() {
							ui.Row(c).Grow(1).Basis(0).MinWidth(0).Gap(6).Children(func() {
								ui.Text(c, filepath.Base(ch.Path)).FontSize(12.5).SingleLine().Shrink(0).MaxWidth(200)
								if dir := filepath.Dir(ch.Path); dir != "." {
									ui.Text(c, dir).FontSize(11).TextColor(faint).SingleLine().Grow(1).Basis(0).MinWidth(0)
								}
							})
							statusMark(c, ch.Code)
						})
					})
				}
			})
		})
		padding, gap, descriptionHeight := float32(12), float32(8), float32(64)
		if compact {
			padding, gap, descriptionHeight = 10, 6, 36
		}
		ui.Column(c).FillWidth().Shrink(0).Padding(padding).Gap(gap).BorderWidth(1, 0, 0, 0).BorderColor(softLine).Children(func() {
			disabled := a.busy || w.Bare || w.Prunable || w.StatusError != ""
			ui.TextInput(c, &a.summary).Bind(&a.summaryInput).Label("Commit summary").Placeholder("Summary (required)").FillWidth().Disabled(disabled)
			ui.TextArea(c, &a.description).Label("Commit description").Placeholder("Description").FillWidth().Height(descriptionHeight).MinHeight(descriptionHeight).Disabled(disabled)
			commit := ui.PrimaryButton(c, a.commitLabel(w)).FillWidth().Disabled(disabled || strings.TrimSpace(a.summary) == "" || selected == 0).OnClick(a.commitSelected)
			switch {
			case len(w.Changes) > 0 && selected == 0:
				commit.Tooltip("Check at least one file")
			case strings.TrimSpace(a.summary) == "":
				commit.Tooltip("Write a summary first")
			}
		})
	})
}

func (a *app) diffView(c *ui.Context) {
	w := a.current()
	if w == nil {
		return
	}
	if a.file == "" {
		a.emptyState(c, w)
		return
	}
	ui.Column(c).Fill().Children(func() {
		ui.Row(c).FillWidth().Height(36).Shrink(0).Padding(0, 16).Gap(10).Background(panel).BorderWidth(0, 0, 1, 0).BorderColor(softLine).Children(func() {
			ui.Text(c, a.file).FontSize(12).FontWeight(600).SingleLine().Grow(1).Basis(0).MinWidth(0).Tooltip(a.file)
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
						bg = addBg
					case strings.HasPrefix(s, "-"):
						color = red
						bg = delBg
					case strings.HasPrefix(s, "@@"):
						color = accent
						bg = hunkBg
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

// emptyState is the right pane when the worktree has nothing to commit:
// it says so, and offers the next steps.
func (a *app) emptyState(c *ui.Context, w *worktree) {
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().AlignItems(ui.Center).Padding(48, 24).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(520).Gap(18).Children(func() {
				label, color := statusLabel(*w)
				title, body := "No local changes", "There are no uncommitted changes in "+filepath.Base(w.Path)+". Here are some next steps."
				if w.Bare || w.Prunable || w.StatusError != "" {
					title, body = label, "Kizuna cannot read this worktree's files."
					if w.StatusError != "" {
						body = w.StatusError
					} else if w.Prunable {
						body = w.PruneReason
					}
				} else {
					color = text
				}
				ui.Column(c).FillWidth().Gap(6).Children(func() {
					ui.Text(c, title).FontSize(20).FontWeight(700).TextColor(color)
					ui.Text(c, body).FontSize(12.5).TextColor(muted).Wrap().FillWidth()
				})
				usable := !a.busy && !w.Prunable
				k := primaryKey()
				ui.Column(c).FillWidth().Background(surface).Radius(8).Border(1, line).Children(func() {
					suggestion(c, "Open in terminal", "Build, test or run Git commands in this worktree.", "Open terminal", "", usable && !w.Bare, func() { a.openSelected(true) })
					suggestion(c, "Show in file manager", w.Path, "Open folder", "", usable, func() { a.openSelected(false) })
					suggestion(c, "Work on another branch", "Check out a branch side by side in a new worktree.", "New worktree", k+"N", !a.busy, a.startCreate)
				})
			})
		})
	})
}

func suggestion(c *ui.Context, title, detail, button, shortcut string, enabled bool, fn func()) {
	ui.Row(c).FillWidth().MinHeight(60).Padding(10, 14).Gap(12).BorderWidth(0, 0, 1, 0).BorderColor(softLine).Children(func() {
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(3).Children(func() {
			ui.Text(c, title).FontSize(12.5).FontWeight(600)
			ui.Text(c, detail).FontSize(11.5).TextColor(muted).SingleLine().FillWidth().Tooltip(detail)
		})
		if shortcut != "" {
			ui.Text(c, shortcut).FontSize(11).TextColor(faint).Shrink(0)
		}
		ui.Button(c, button).Shrink(0).Disabled(!enabled).OnClick(fn)
	})
}

func (a *app) cloneView(c *ui.Context) {
	dialogTitle(c, "Clone repository", "Uses your existing Git credentials and SSH agent.")
	ui.Column(c).FillWidth().Gap(6).Children(func() {
		ui.Text(c, "Repository URL").FontSize(11.5).TextColor(muted)
		ui.TextInput(c, &a.cloneURL).Label("Input: REPOSITORY URL").Placeholder("https://github.com/owner/repo.git").FillWidth().AutoFocus().Disabled(a.busy).OnChange(a.updateClonePath)
	})
	ui.Column(c).FillWidth().Gap(6).Children(func() {
		ui.Text(c, "Local path").FontSize(11.5).TextColor(muted)
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			ui.TextInput(c, &a.clonePath).Label("Input: CLONE DESTINATION").Placeholder("Paste a URL to fill this in").Grow(1).Basis(0).MinWidth(0).Disabled(a.busy)
			ui.Button(c, "Choose…").Label("Choose clone folder").Tooltip("Choose the parent folder; the repository name is added automatically").Shrink(0).Disabled(a.busy).OnClick(a.chooseCloneFolder)
		})
		ui.Text(c, "Parent folder: "+a.cloneParent).FontSize(11).TextColor(faint).SingleLine().FillWidth().Tooltip(a.cloneParent)
	})
	dialogButtons(c, func() {
		ui.Button(c, "Cancel").Disabled(a.busy).OnClick(func() { a.cloning = false })
		ui.PrimaryButton(c, "Clone and open").Disabled(a.busy || a.cloneURL == "" || a.clonePath == "").OnClick(a.cloneRepository)
	})
}
