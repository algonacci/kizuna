package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
)

type host interface {
	Update(func())
	PickFolder() (string, error)
	PickCloneFolder(string) (string, error)
	OpenPath(string) error
	OpenTerminal(string) error
	// Quit closes the window once requestQuit found nothing to ask about.
	Quit()
	// SetTheme makes native menus and tooltips follow the chosen mode.
	SetTheme(mode string)
}
type app struct {
	keyboardHelp               bool
	host                       host
	ctx                        context.Context
	cancel                     context.CancelFunc
	repo                       *repository
	recent                     []string
	configPath                 string
	selected, search, message  string
	busy                       bool
	errorMessage               bool
	creating, removing         bool
	draft                      createRequest
	managing                   bool
	file, diff, remote         string
	diffLoading                bool
	diffGeneration             uint64
	diffCancel                 context.CancelFunc
	checked                    map[string]bool
	summary, description       string
	cloning                    bool
	cloneURL, clonePath        string
	cloneParent, cloneAutoPath string
	repoIdentities             map[string]string
	// operation labels a Git command in flight that quitting would cut off
	// (commit, fetch/pull/push, clone, worktree changes); reading a
	// repository or waiting on a folder picker is not one.
	operation string
	// confirmQuit shows the quit question; allowClose lets the window close.
	confirmQuit, allowClose bool
	// repoMenu and treeMenu open the top bar's two dropdowns.
	repoMenu, treeMenu bool
	// theme is the appearance mode: system, light or dark.
	theme        string
	summaryInput ui.Handle
}

func newApp(h host) *app {
	ctx, cancel := context.WithCancel(context.Background())
	return &app{host: h, ctx: ctx, cancel: cancel, checked: make(map[string]bool), repoIdentities: make(map[string]string), message: "Open a repository to get started", theme: themeSystem}
}
func (a *app) current() *worktree {
	if a.repo != nil {
		for i := range a.repo.Trees {
			if a.repo.Trees[i].Path == a.selected {
				return &a.repo.Trees[i]
			}
		}
	}
	return nil
}
func (a *app) report(err error) {
	if err != nil {
		a.message = err.Error()
		a.errorMessage = true
	}
}

func (a *app) pickRepository() {
	if a.busy {
		return
	}
	a.busy = true
	go func() {
		path, err := a.host.PickFolder()
		a.host.Update(func() {
			a.busy = false
			a.report(err)
			if err == nil && path != "" {
				a.openRepository(path)
			}
		})
	}()
}

func (a *app) openRepository(path string) {
	if a.busy {
		return
	}
	a.busy = true
	a.errorMessage = false
	a.message = "Reading worktrees…"
	recent := append([]string(nil), a.recent...)
	config := a.configPath
	go func() {
		r, err := loadRepository(a.ctx, path)
		var saveErr error
		if err == nil {
			recent = remember(recent, r.Path)
			saveErr = writeRecent(config, recent)
		}
		a.host.Update(func() {
			a.busy = false
			if err != nil {
				a.report(err)
				return
			}
			same := a.repo != nil && a.repo.Path == r.Path
			a.repo = r
			a.repoIdentities[r.Path] = r.identity("origin")
			a.recent = recent
			if !same || a.current() == nil {
				a.selected = r.Trees[0].Path
			}
			a.creating = false
			a.removing = false
			a.cloning = false
			a.message = fmt.Sprintf("%d worktrees · up to date", len(r.Trees))
			a.errorMessage = false
			a.report(saveErr)
			a.syncChanges(!same)
			a.loadRecentIdentities()
		})
	}()
}

func (a *app) loadRecentIdentities() {
	var missing []string
	for _, path := range a.recent {
		if a.repoIdentities[path] == "" {
			missing = append(missing, path)
		}
	}
	if len(missing) == 0 {
		return
	}
	go func() {
		identities := make(map[string]string)
		for _, path := range missing {
			url, err := git(a.ctx, path, "remote", "get-url", "origin")
			if err == nil {
				identities[path] = remoteIdentity(strings.TrimSpace(url))
			}
			if identities[path] == "" {
				remotes, _ := git(a.ctx, path, "remote")
				for _, remote := range strings.Fields(remotes) {
					url, e := git(a.ctx, path, "remote", "get-url", remote)
					if e == nil {
						if id := remoteIdentity(strings.TrimSpace(url)); id != "" {
							identities[path] = id
							break
						}
					}
				}
			}
		}
		a.host.Update(func() {
			for path, id := range identities {
				if id != "" {
					a.repoIdentities[path] = id
				}
			}
		})
	}()
}
func (a *app) refresh() {
	if a.repo != nil && !a.creating && !a.removing && !a.cloning {
		a.openRepository(a.repo.Path)
	}
}
func (a *app) selectTree(path string) {
	if a.busy {
		return
	}
	a.selected = path
	a.creating = false
	a.removing = false
	a.managing = false
	a.summary, a.description = "", ""
	a.syncChanges(true)
}

// cycleTheme steps System → Light → Dark and remembers the choice.
func (a *app) cycleTheme() {
	a.theme = nextTheme(a.theme)
	a.host.SetTheme(a.theme)
	path, s := settingsPath(a.configPath), settings{Theme: a.theme}
	go func() {
		if err := writeSettings(path, s); err != nil {
			a.host.Update(func() { a.report(fmt.Errorf("Could not save appearance: %w", err)) })
		}
	}()
}

// showChanges and showWorktrees switch the sidebar tab.
func (a *app) showChanges() { a.managing, a.removing = false, false }
func (a *app) showWorktrees() {
	if a.repo != nil {
		a.managing = true
	}
}
func (a *app) startCreate() {
	if a.busy || a.repo == nil {
		return
	}
	a.repoMenu, a.treeMenu = false, false
	base := "HEAD"
	parent := filepath.Dir(a.repo.Trees[0].Path)
	if w := a.current(); w != nil && w.Head != "" {
		base = w.Head
	}
	a.draft = createRequest{Mode: "New branch", Base: base, Path: filepath.Join(parent, repoName(a.repo)+"-worktree")}
	a.creating = true
	a.cloning = false
	a.removing = false
}
func repoName(r *repository) string {
	if r == nil {
		return ""
	}
	if filepath.Base(r.Path) == ".git" {
		return filepath.Base(filepath.Dir(r.Path))
	}
	return filepath.Base(r.Path)
}
func (a *app) create() {
	if a.busy || a.repo == nil {
		return
	}
	req := a.draft
	req.Branch = strings.TrimSpace(req.Branch)
	req.Base = strings.TrimSpace(req.Base)
	repo := a.repo.Path
	a.runMutation("Creating worktree…", req.Path, func() error { return createWorktree(a.ctx, repo, req) })
}
func (a *app) mutate(action string) {
	w := a.current()
	if a.busy || w == nil {
		return
	}
	repo, path := a.repo.Path, w.Path
	a.runMutation("Running git worktree "+action+"…", path, func() error { return changeWorktree(a.ctx, repo, path, action) })
}
func (a *app) runMutation(label, selected string, fn func() error) {
	a.busy = true
	a.operation = label
	a.message = label
	a.errorMessage = false
	repo := a.repo.Path
	go func() {
		err := fn()
		var r *repository
		var reloadErr error
		if err == nil {
			r, reloadErr = loadRepository(a.ctx, repo)
		}
		a.host.Update(func() {
			a.busy = false
			a.operation = ""
			if err != nil {
				a.report(err)
				return
			}
			a.creating = false
			a.removing = false
			a.message = "Worktrees updated"
			a.errorMessage = false
			if reloadErr != nil {
				a.report(fmt.Errorf("Action completed, but refresh failed: %w", reloadErr))
				return
			}
			a.repo = r
			a.selected = selected
			if a.current() == nil {
				a.selected = r.Trees[0].Path
			}
			a.syncChanges(false)
		})
	}()
}

func (a *app) syncChanges(reset bool) {
	if reset {
		a.checked = make(map[string]bool)
		a.file = ""
		a.summary, a.description = "", ""
		a.remote = ""
	}
	w := a.current()
	if w == nil {
		a.selectFile("")
		return
	}
	next := make(map[string]bool)
	found := false
	for _, ch := range w.Changes {
		value, exists := a.checked[ch.Path]
		if !exists {
			value = true
		}
		next[ch.Path] = value
		if ch.Path == a.file {
			found = true
		}
	}
	a.checked = next
	if !found {
		a.file = ""
		if len(w.Changes) > 0 {
			a.file = w.Changes[0].Path
		}
	}
	valid := false
	for _, r := range a.repo.Remotes {
		if r == a.remote {
			valid = true
		}
	}
	if !valid {
		a.remote = ""
		for _, r := range a.repo.Remotes {
			if r == w.Remote {
				a.remote = r
				break
			}
		}
		if a.remote == "" {
			for _, r := range a.repo.Remotes {
				if r == "origin" {
					a.remote = r
					break
				}
			}
		}
		if a.remote == "" && len(a.repo.Remotes) > 0 {
			a.remote = a.repo.Remotes[0]
		}
	}
	a.selectFile(a.file)
}

func (a *app) selectFile(path string) {
	if a.diffCancel != nil {
		a.diffCancel()
	}
	a.diffGeneration++
	a.file, a.diff, a.diffLoading = path, "", false
	w := a.current()
	if w == nil || path == "" {
		return
	}
	var selected *change
	for _, ch := range w.Changes {
		if ch.Path == path {
			copy := ch
			selected = &copy
			break
		}
	}
	if selected == nil {
		return
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.diffCancel = cancel
	gen, dir, ch := a.diffGeneration, w.Path, *selected
	a.diffLoading = true
	go func() {
		preview, err := fileDiff(ctx, dir, ch)
		a.host.Update(func() {
			if gen != a.diffGeneration {
				return
			}
			a.diffLoading = false
			if err != nil {
				a.diff = "Could not load diff: " + err.Error()
			} else {
				a.diff = preview
			}
		})
	}()
}

func (a *app) selectedChanges() []change {
	var changes []change
	if w := a.current(); w != nil {
		for _, ch := range w.Changes {
			if a.checked[ch.Path] {
				changes = append(changes, ch)
			}
		}
	}
	return changes
}

// Refresh after failures too: hooks, transport errors and pulls can change disk state.
func (a *app) runGitAction(label string, clearCommit bool, fn func() error) {
	if a.busy || a.repo == nil {
		return
	}
	a.busy = true
	a.operation = label
	a.message = label
	a.errorMessage = false
	repo := a.repo.Path
	go func() {
		err := fn()
		r, reloadErr := loadRepository(a.ctx, repo)
		a.host.Update(func() {
			a.busy = false
			a.operation = ""
			if err == nil && clearCommit {
				a.summary, a.description = "", ""
			}
			if reloadErr == nil {
				a.repo = r
				a.syncChanges(false)
			}
			a.message = "Done · repository refreshed"
			a.errorMessage = false
			if err != nil {
				a.report(err)
			} else if reloadErr != nil {
				a.report(fmt.Errorf("Action completed; refresh failed: %w", reloadErr))
			}
		})
	}()
}
func (a *app) commitSelected() {
	w := a.current()
	if a.busy || w == nil {
		return
	}
	dir, summary, description, changes := w.Path, a.summary, a.description, a.selectedChanges()
	a.runGitAction("Committing selected files…", true, func() error { return commitFiles(a.ctx, dir, summary, description, changes) })
}
func (a *app) remoteAction(action string) {
	w := a.current()
	if a.busy || w == nil || action == "" || a.remote == "" {
		return
	}
	dir, remote := w.Path, a.remote
	a.runGitAction("Git "+action+" · "+remote+"…", false, func() error { return syncRemote(a.ctx, dir, action, remote) })
}

func (a *app) startClone() {
	if a.busy {
		return
	}
	a.repoMenu, a.treeMenu = false, false
	a.cloning = true
	a.creating = false
	a.removing = false
	if a.cloneParent == "" {
		if home, err := os.UserHomeDir(); err == nil {
			a.cloneParent = filepath.Join(home, "Documents", "GitHub")
		} else {
			a.report(err)
		}
	}
	a.updateClonePath()
}

func (a *app) updateClonePath() {
	if a.clonePath != "" && a.clonePath != a.cloneAutoPath {
		return
	}
	name := cloneFolderName(a.cloneURL)
	a.clonePath = ""
	if name != "" && a.cloneParent != "" {
		a.clonePath = filepath.Join(a.cloneParent, name)
	}
	a.cloneAutoPath = a.clonePath
}

func (a *app) chooseCloneFolder() {
	if a.busy {
		return
	}
	a.busy = true
	parent := a.cloneParent
	go func() {
		path, err := a.host.PickCloneFolder(parent)
		a.host.Update(func() {
			a.busy = false
			a.report(err)
			if err != nil || path == "" {
				return
			}
			a.cloneParent = path
			a.clonePath = ""
			a.cloneAutoPath = ""
			a.updateClonePath()
		})
	}()
}
func (a *app) cloneRepository() {
	if a.busy {
		return
	}
	url, destination := strings.TrimSpace(a.cloneURL), a.clonePath
	a.busy = true
	a.errorMessage = false
	a.message = "Cloning repository…"
	a.operation = a.message
	go func() {
		err := cloneRepository(a.ctx, url, destination)
		a.host.Update(func() {
			a.busy = false
			a.operation = ""
			if err != nil {
				a.report(err)
				return
			}
			a.cloneURL, a.clonePath = "", ""
			a.cloneAutoPath = ""
			a.cloning = false
			a.openRepository(destination)
		})
	}()
}
func (a *app) openSelected(terminal bool) {
	w := a.current()
	if a.busy || w == nil {
		return
	}
	path := w.Path
	go func() {
		var err error
		if terminal {
			err = a.host.OpenTerminal(path)
		} else {
			err = a.host.OpenPath(path)
		}
		if err != nil {
			a.host.Update(func() { a.report(err) })
		}
	}()
}
