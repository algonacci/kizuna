# Kizuna

A native Git worktree client. Parallel branches, one home.

Built with **Go 1.27.3 + MyGo v0.7.4**, following the native desktop conventions
in `second-brain/Go-MyGo-GUI-Desktop`.

The interface follows Nimbo's labeled native icon toolbar and Tomori's quiet
chrome: repository selection and Changes/Worktrees navigation in the left sidebar,
a compact worktree/remote context row, and a dedicated diff panel. Repository and
worktree pickers show short names while keeping full local paths in their menus.

## Features

- Open a Git repository, linked worktree, or bare repository.
- Switch repositories from a dropdown showing remote `owner/repo` (including
  organizations such as `Omniflow-id`) and local path. Add local repositories or
  clone HTTPS/SSH URLs into a new directory.
  Clone defaults to `~/Documents/GitHub/<repository-name>` from the URL; **Choose…**
  selects a different parent folder. The complete destination remains editable.
- Changes-first workspace: check files on the left, review their unified diff in
  the main pane, and enter a commit summary/description below the file list.
- Commit the full current contents of selected files while preserving unrelated
  staged changes. Rename sources and deleted files are included correctly.
- Fetch, fast-forward pull, and push/publish branches using a selected remote.
- Browse and filter worktrees with branch, HEAD, changed files, and recent commits.
- Show upstream ahead/behind counts using locally available refs.
- Create worktrees with a new branch, an existing local branch, or detached HEAD.
- Open a worktree in the file manager or external terminal.
- Lock/unlock linked worktrees and remove them while retaining their branches.
- Remember the last ten repositories; restore the latest on launch.
- Refresh manually or when the window regains focus.

Removal uses Git's normal protections: locked, modified, and untracked worktrees
are refused. Kizuna never force-removes a worktree. Primary worktrees cannot be
removed. Missing worktrees are labeled as prunable. The **Worktrees** button opens
the worktree management view with lock/unlock, removal, status, and recent commits;
Open folder, Open terminal, and New worktree stay available in the toolbar.

## Changes and remotes

Diffs compare the working directory against HEAD, including staged and unstaged
changes together. New text files get an added-lines preview; binary files show a
notice. Previews are bounded to 512 KiB and 2,500 displayed lines. Long individual
lines are truncated at 4,000 bytes. Diff gutters show old/new file line numbers.

Checkboxes select **whole files**, not hunks or the existing staged snapshot.
Commit stages selected files and uses `git commit --only`; unchecked staged files
remain staged. If a hook rejects a commit, the error is shown, the draft is retained,
and selected files may remain staged. Finish active merges/rebases/cherry-picks in
your terminal before committing here.

Fetch updates remote refs. Pull uses `--ff-only --no-autostash`; diverging histories
surface a Git error. Push publishes the current branch with upstream tracking and
honors an existing differently named upstream on the selected remote. There is no
force push. Existing Git credential helpers, SSH configuration, and agent are used;
Kizuna does not provide a sign-in flow. Git errors appear in the status area.
Network operations have a two-minute timeout. Pruning, moving worktrees, partial-hunk
staging, and an integrated merge/conflict editor are not implemented.

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
resizing through MyGo's headless tester with a queued host. Additional real-Git tests
cover selected commits, preservation of unchecked staged files, initial commits,
rename/deletion, literal filenames, bounded diffs, clone, fetch, pull, publishing,
upstream target names, and non-fast-forward rejection using local bare remotes.
UI tests also exercise checkbox selection, commit, diff selection, repository
dropdown switching, clone, fetch, pull, and push. Screenshots cover
welcome, changes/diff, compact layout, worktree management, create, and clone. These tests do
not substitute for testing native file dialogs and terminal launching on the OS.

## Structure

| File | Responsibility |
| --- | --- |
| `git.go` | Context-bound Git commands, NUL-delimited parsing, worktree operations |
| `changes.go` | Bounded diffs, selected-file commits, clone, fetch/pull/push |
| `repository_identity.go` | Remote URL to owner/repository identity |
| `app.go` | App state, asynchronous jobs, queued UI updates |
| `view.go`, `changes_view.go`, `theme.go` | Native interface and palette |
| `main.go` | Window, menu, folder dialog, desktop integration |
| `config.go` | Recent repositories with atomic file replacement |

Git commands use argument arrays rather than shell interpolation. Repository
operations are serialized while background jobs run; view rendering performs no
filesystem or Git I/O. Application shutdown cancels active Git commands.
