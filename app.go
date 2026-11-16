package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type host interface {
	Update(func())
	PickFolder() (string, error)
	OpenPath(string) error
	OpenTerminal(string) error
}
type app struct {
	host                      host
	ctx                       context.Context
	cancel                    context.CancelFunc
	repo                      *repository
	recent                    []string
	configPath                string
	selected, search, message string
	busy                      bool
	errorMessage              bool
	creating, removing        bool
	draft                     createRequest
}

func newApp(h host) *app {
	ctx, cancel := context.WithCancel(context.Background())
	return &app{host: h, ctx: ctx, cancel: cancel, message: "Open a repository to get started"}
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
			a.recent = recent
			if !same || a.current() == nil {
				a.selected = r.Trees[0].Path
			}
			a.creating = false
			a.removing = false
			a.message = fmt.Sprintf("%d worktrees · up to date", len(r.Trees))
			a.errorMessage = false
			a.report(saveErr)
		})
	}()
}
func (a *app) refresh() {
	if a.repo != nil && !a.creating && !a.removing {
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
}
func (a *app) startCreate() {
	if a.busy || a.repo == nil {
		return
	}
	base := "HEAD"
	parent := filepath.Dir(a.repo.Trees[0].Path)
	if w := a.current(); w != nil && w.Head != "" {
		base = w.Head
	}
	a.draft = createRequest{Mode: "New branch", Base: base, Path: filepath.Join(parent, repoName(a.repo)+"-worktree")}
	a.creating = true
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
