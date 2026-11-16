# Kizuna

A native Git worktree client. Parallel branches, one home.

Built with **Go 1.27.3 + MyGo v0.7.4**, following the native desktop conventions
in `second-brain/Go-MyGo-GUI-Desktop`.

## Features

- Open a Git repository, linked worktree, or bare repository.
- Browse and filter worktrees with branch, HEAD, changed files, and recent commits.
- Show upstream ahead/behind counts using locally available refs.
- Create worktrees with a new branch, an existing local branch, or detached HEAD.
- Open a worktree in the file manager or external terminal.
- Lock/unlock linked worktrees and remove them while retaining their branches.
- Remember the last ten repositories; restore the latest on launch.
- Refresh manually or when the window regains focus.

Removal uses Git's normal protections: locked, modified, and untracked worktrees
are refused. Kizuna never force-removes a worktree. Primary worktrees cannot be
removed. Missing worktrees are labeled as prunable; pruning, moving, fetching,
staging, and committing are not part of this initial version.

## Run

Install Go 1.27.3 and Git, then:

```sh
go tool mygo dev
# Or open a particular repository:
go run . /absolute/path/to/repository
```

Keyboard shortcuts: **Cmd/Ctrl+O** opens a repository, **Cmd/Ctrl+N** creates a
worktree, and **Cmd/Ctrl+R** refreshes the current repository.

Recent repositories are stored in `kizuna/recent.json` under `os.UserConfigDir()`
(`~/Library/Application Support` on macOS).

## Build

```sh
go tool mygo build
open build/darwin-arm64/Kizuna.app
```

The macOS build produces an application bundle and DMG in `build/darwin-arm64/`.
macOS arm64 is the build-verified target. Linux and Windows native runtime behavior
has not been verified in this repository.

## Verification

```sh
go test ./...
go test -race ./...
go vet ./...
go tool mygo vet .
CGO_ENABLED=0 go test ./...
KIZUNA_SNAPSHOT=build/screenshots go test -run TestSnapshots .
```

Integration tests use temporary Git repositories for create/lock/unlock/remove,
dirty-file protection, retained branches, detached HEAD, bare repositories,
missing directories, staged renames, and paths containing spaces/newlines.
UI tests drive creation, selection-dependent actions, removal cancellation, and
resizing through MyGo's headless tester with a queued host. Screenshots cover
welcome, worktree detail, compact layout, and the creation form. These tests do
not substitute for testing native file dialogs and terminal launching on the OS.

## Structure

| File | Responsibility |
| --- | --- |
| `git.go` | Context-bound Git commands, NUL-delimited parsing, worktree operations |
| `app.go` | App state, asynchronous jobs, queued UI updates |
| `view.go`, `theme.go` | Native interface and palette |
| `main.go` | Window, menu, folder dialog, desktop integration |
| `config.go` | Recent repositories with atomic file replacement |

Git commands use argument arrays rather than shell interpolation. Repository
operations are serialized while background jobs run; view rendering performs no
filesystem or Git I/O. Application shutdown cancels active Git commands.
