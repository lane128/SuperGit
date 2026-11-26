# Workspace Quick Actions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship titlebar one-click Stash and checked workspace cleanup in SuperGit 1.0.1.

**Architecture:** Keep existing Stash menu flows in a split button next to the branch (approved option A). Add a repository-level prepared cleanup object that binds HEAD, branch, index, filesystem and ignore state; a scoped asynchronous dialog previews and executes it. Preserve existing per-file operations.

**Tech Stack:** Go 1.27.3, system Git, MyGo/Lobe native UI, macOS Apple Silicon, existing Python native automation.

**Spec:** docs/features/workspace-quick-actions.md

## Global Constraints

- Preserve ignored untracked files, branch identity and HEAD/history.
- Default cleanup preserves HEAD-absent additions, including staged additions and rename destinations.
- Deletion is opt-in per opening, previewed, checked again before writing; no recursive deletion or unscoped hard reset.
- Refuse unsupported/unprovable states before writes; report partial failures faithfully.
- No new dependencies, settings, global shortcuts, or signing keys.
- User authorized implementation, self-test, main merge and release 1.0.1 on 2026-11-27; execute these after gates without re-asking. Native destructive tests use newly created fixtures only.
- Review precedes final native acceptance. Full checks after integrating independent worktrees; avoid redundant full suites.

## Review Focus

- Tracked file replaced by a directory with ignored children must never lose those children (backend tests).
- Same status letters with changed bytes or same HEAD on a different branch invalidate confirmation (backend tests).
- Cancellation/repository switching during asynchronous preparation must not resurrect or execute a stale dialog (UI tests).
- Linked worktrees share refs/stash; identify this operation by unique message token, not latest reference (backend tests).
- Narrow window and long branch labels must keep both controls reachable; menu keeps existing named flows (UI/native tests).

---

### Task 1: Repository checked clean and quick Stash

**Files:** Create `internal/gitrepo/workspace_clean.go`, `workspace_clean_test.go`, `workspace_stash.go`, `workspace_stash_test.go`; reuse helpers from `palette_workspace.go`, `discard_prepared.go`, `stash.go` as needed.

**Interfaces:**
- Produces `WorkspaceCleanPreview` with exported `Head string`, `RestorePaths []string`, `NewPaths []string`, `DeleteNew bool`, `CanExecute bool`; private immutable validation snapshot.
- Produces `(*Repository).PrepareWorkspaceClean(context.Context, bool) (WorkspaceCleanPreview, error)`.
- Produces `(*Repository).CleanWorkspaceChecked(context.Context, WorkspaceCleanPreview) error`.
- Produces `(*Repository).SaveWorkspaceStash(context.Context) error`; generates unique automatic message and verifies its own result.

- [ ] Write real-Git regression tests: mixed staged/unstaged/untracked, staged additions, renames, ignored paths, symlinks, file/directory replacements, empty directories; verify default keeps additions, opt-in yields clean status and preserved ignored bytes.
- [ ] Run `go test ./internal/gitrepo -run 'TestWorkspace(Clean|Stash)'` and observe missing API failures.
- [ ] Implement snapshot/preflight, ordered checked execution and quick Stash using existing safe helpers; include index flags, sparse checkout, conflicts/operations, nested repositories/submodules and ignored-rule collisions.
- [ ] Add stale content/index/branch/HEAD/ignore/inode tests, refused-state no-write fingerprints, partial failure/cancellation and Stash restoration tests.
- [ ] Run targeted tests and `go vet ./internal/gitrepo`; return only task changes for integration.

### Task 2: Titlebar controls and scoped clean dialog

**Files:** Create `internal/lobedesk/workspace_quick_actions.go`, `workspace_quick_actions_test.go`; modify `view.go`, `stash.go`, `app.go`, `dialog_scope.go`, existing close/keyboard/reset and i18n files.

**Interfaces:** Consumes Task 1 APIs. Produces app handlers `quickWorkspaceStash()`, `openWorkspaceClean()`, `prepareWorkspaceClean(bool)`, `confirmWorkspaceClean()` and scoped dialog state.

- [ ] Add failing UI/state tests for titlebar placement, split menu, default false reset, loading/empty eligibility and repository-scoped late-result rejection.
- [ ] Render 24px archive split button and cleanup next to branch, remove duplicate right Stash menu, keep file toolbar unchanged.
- [ ] Render 520px dialog with full scrollable restore/new path groups, per-open checkbox, explicit ignored preservation and irreversible warning; cancel focus, Esc, loading guard, busy guard, localization.
- [ ] Bind repository pointer/root/generation, pending-dialog watcher suppression and repository-switch cleanup. Option changes prepare fresh previews. Execute exact approved snapshot via existing operation plumbing, refreshing on errors too.
- [ ] Run targeted UI tests and `go vet ./internal/lobedesk` after Task 1 integration.

### Task 3: Integrated verification and native acceptance

**Files:** Create `scripts/native-lobe-workspace-quick-actions.py`; record `docs/acceptance/2026-11-27/README.md` and evidence.

- [ ] Add isolated fixture native script using existing automation patterns. Assert backend state and ignored bytes after Stash/restore, cancellation, default cleanup, opt-in deletion and stale confirmation.
- [ ] Run gofmt, `go test -cover ./...` once and `make check`; store logs outside package trees. Fix actual failures with targeted reruns, then repeat affected full checks only where necessary.
- [ ] Scoped independent read-only review of changed and directly related files; no duplicate test runs by reviewer. Resolve must-fix findings.
- [ ] Build `make e2e-build`, `GOFLAGS=-tags=e2e make app`; launch actual `build/SuperGit.app` with isolated fixtures. Run native acceptance including light/dark and 1320×840/900×640 titlebar, screenshot evidence and keyboard flows.
- [ ] Record exact results/limitations; no claim that browser mockups constitute native acceptance.

### Task 4: Merge and release 1.0.1

**Files:** Release notes/acceptance record; existing Makefile, tools/appversion and releasepack drive versioning.

- [ ] Check status/diff/secret hygiene; run required `/verify` immediately before feature commit. English commit includes Why, Verified, Rollback.
- [ ] Integrate feature into main without overwriting unrelated user work; push main after final acceptance.
- [ ] Confirm existing v1.0.1/Release does not exist, use existing Ed25519 key from `~/.config/supergit-release/update.key` without printing key bytes. Tag approved commit v1.0.1 and build signed release with matching public key.
- [ ] Verify version 1.0.1, codesign, manifest signature/archive checksum and release notes, then publish GitHub release assets and mark latest.
- [ ] Download public release manifest/archive and verify checksum/version/signature, record release URL and rollback instructions. Do not delete old versions or replace user's installed application unless asked.
