# Workspace quick actions / 1.0.1 acceptance

Status: In progress — not release approval evidence yet.

Scope: approved option A, one-click whole-workspace Stash and checked cleanup beside the titlebar branch. User requested implementation, self-test, main merge and 1.0.1 release on 2026-11-27.

## Build and environment

- Implementation base: `fe2604e` (fresh origin/main worktree).
- Platform: macOS Apple Silicon; Go/MyGo native application.
- HTML preview remains a local ignored artifact in the original checkout's `design/`; it does not replace native acceptance.

## Verification ledger

- Existing baseline: `go test ./internal/lobedesk -run 'TestTopCommandPaletteToolbarAndPosition|TestRepositoryConfirmation' -count=1` passed.
- UI red: new workspace UI tests failed on missing `workspaceCleanState` / `WorkspaceCleanPreview` / handlers before implementation.
- Backend integration: checked cleanup and quick Stash tests cover mixed changes, ignored paths, staged additions, intent-to-add boundaries, file/directory replacement, symlinks, nested repositories/submodules, stale snapshots, cancellation and partial failure.
- Whole-branch independent read-only review completed. Both P2 findings were reproduced and fixed: tracked files beneath ignored directory ancestors, and cleanup reachability with only empty directories. `TestWorkspaceCleanTrackedFileInsideIgnoredDirectory` and `TestWorkspaceCleanEmptyDirectoryOnly` failed before their fixes and passed afterward.
- Final related regression run: `go test ./internal/gitrepo ./internal/lobedesk -run 'TestWorkspace(Clean|Quick)' -count=1` passed (23.890s / 3.578s).
- Earlier `go test -cover ./...` measured gitrepo 81.1%, lobedesk 85.9%, but failed on the missing action-copy group and an equal-timestamp rename-history fixture. Fixed both test-side issues (the original three-commit assertion remains; the identity test then passed 20 consecutive runs) and did not rerun full coverage before the final `make check`.
- Full `make check` passed on the final code: ordinary tests, `-race` (lobedesk 857.4s), MyGo SDK/snapshot checks and `go vet`, exit 0. Log: `.superpowers/sdd/2026-11-27-workspace-quick-actions/check.log` (local ignored). Feature commit `01fb89d`; the tag and GitHub release are recorded in the release section.
- Native acceptance: **18/18 passed** across English light 1320×840, English dark 900×640, and Chinese light 900×640. Each run verifies all six scenarios below. Local screenshots and raw logs: `build/acceptance-workspace-actions/{light-1320,dark-900,zh-light-900}/`.
- Initial native run passed four scenarios, then failed because the script expected `checkbox` while automation exposes `checkBox`. Corrected the script's role spelling; all three complete runs then passed. No product change was needed.
- Visual inspection: titlebar actions sit beside branch; named Stash menu remains available; default checkbox is clear, paths scroll, warning and Cancel/confirm controls fit the narrow window in both themes. No new motion or page redesign.
- Acceptance build: uncommitted feature on `fe2604e`, `make app` with e2e tag, ad-hoc signature verified. Bundle fallback version was 0.1.0 (62) because the local v1.0.0 tag belongs to an older rewritten history. Release must be rebuilt from v1.0.1 and verified separately.
- User requested self-test followed by merge/release; no separate human hands-on acceptance occurred. Destructive actions were exercised only in script-created disposable repositories. Conflicts, sparse checkout, ignored collisions, linked-worktree races and empty-directory edge cases are covered by Go regression tests, not all by native UI automation.

## Native reproduction

```sh
make e2e-build
GOFLAGS=-tags=e2e make app
python3 -I scripts/native-lobe-workspace-quick-actions.py --app "$PWD/build/SuperGit.app" --theme light --size 1320x840 --output build/acceptance-workspace-actions/light-1320
python3 -I scripts/native-lobe-workspace-quick-actions.py --app "$PWD/build/SuperGit.app" --theme dark --size 900x640 --output build/acceptance-workspace-actions/dark-900
python3 -I scripts/native-lobe-workspace-quick-actions.py --app "$PWD/build/SuperGit.app" --theme light --size 900x640 --language zh --output build/acceptance-workspace-actions/zh-light-900
```

The script creates its own temporary repository and preferences, launches the actual bundle with `open -n`, and compares HEAD, branch, index and file fingerprints. It checks Stash and existing Apply, cancellation, default preservation, opt-in deletion, symlink-target preservation and stale-preview refusal. It never accepts an existing checkout as its destructive target. [Recorded results](workspace-actions-results.json); PNGs and raw logs remain local gitignored evidence.

The six existing standard native workflows also passed against the same application source: repository tabs, toolbar/sync, daily workflow, Stash, conflict sides, and submodules. The scripts behind `make native-check` were invoked directly with `SUPERGIT_NATIVE_BUILD_DIR=build/_quick-actions-regression` and the built bundle binary, keeping fixture source files under a Go-ignored directory while full Go checks ran. The toolbar test covered a long Unicode branch at 900px and 115% font scale, named Stash, command palette, and local-remote Fetch/Pull/Push. Legacy desktop native acceptance was not repeated; the feature changes the Lobe workbench.

The daily workflow initially failed twice because its snapshot reads could refresh `.git/index` and invalidate a selected patch. An added inode/mtime assertion reproduced that read side effect. Setting `GIT_OPTIONAL_LOCKS=0` for the script removed the side effect, and the complete current-build workflow passed with the assertion retained. No production selection or watcher code was changed. Final daily evidence: `build/_quick-actions-regression/lobe-daily-cx5qa_r3/`; conflict and submodule logs are sibling directories.

## Release

Released 2026-11-27: commit `8ec5f27` is remote `main`; annotated tag `v1.0.1` points at it. GitHub Release: https://github.com/lane128/SuperGit/releases/tag/v1.0.1 (marked latest).

| Asset | Value |
| --- | --- |
| `SuperGit-1.0.1-darwin-arm64.zip` | 15,441,386 bytes |
| SHA-256 | `f7492bf9c22a926dd2a6676b8309ffc18da82f2a61e8b3ca6f3080eba0ea48aa` |
| Bundle version | 1.0.1 (build 63) |

The archive was built without the e2e tag, embedded the existing update public key, passed `codesign --verify --deep --strict`, and `verify-release.go` validated the manifest URL, size, SHA-256 and Ed25519 signature. The existing public key also verifies the published 1.0.0 manifest, confirming key continuity. Private key permissions were checked (600); key bytes are not recorded here or in logs. A fresh download from `releases/latest` is verified in the release log.

Rollback: revert `8ec5f27`; code rollback cannot restore explicitly discarded user changes. Removing a release/tag or replacing the published binary is a separate destructive operation.
