# First Parent Navigation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Alt+Down navigation to the visible selected commit's first parent, while preserving later user selections and giving actionable feedback when missing history cannot load.

**Architecture:** Reuse the shared shortcut table, `jumpToCommit`, selection/reveal and existing asynchronous history paging. Add a small `parent_navigation.go` wrapper for shortcut target/guards; fix only confirmed pending-jump lifecycle defects in the existing navigation path.

**Tech Stack:** Go, MyGo UI, Lobe components, existing Git repository API.

**Spec:** This document's scope below, derived from the ongoing user-authorized Sublime Merge alignment and product/design/testing reports on 2026-11-18. Default - Merge resource entries OSX:79, Windows:84, Linux:86 map Alt+Down to navigate_to_parent. Resource evidence does not prove runtime focus/filter or default parent semantics.

## Global Constraints

- Preserve all unrelated uncommitted work and generated_images. No commit, install or publication authorized.
- Main implements; independent-worktree roles review actual /Users/adam/SuperGit files by absolute path.
- Use owned disposable fixtures and synthetic identities for Git/UI acceptance; compare refs, HEAD, Index and work bytes.
- Keep current layout, dependency versions, existing per-parent buttons and reference range.
- First parent means Parents[0], a SuperGit adaptation. No child/second-parent shortcut or navigation stack.
- Historical visible selection only; root/no selection/worktree/other page no action. SkipWhenTyping preserves all actual editor keys, with no new OnTextKey forwarding.
- Busy/loading/historyReloading/chooser/ref popup/main dialogs block request and retained Run. Ordinary background pagination may continue.
- Missing target under a blocked full-history read must warn to retry and must not leave inert pending state. Already loaded targets remain navigable. File-history internal pause may be exited through existing reset/resume behavior.
- Selection/reveal follow existing jump behavior, including clearing query and leaving file history only when required. Keep hidden details hidden; do not introduce native window focus or new focus restoration behavior for old buttons.

## Review Focus

- Hidden old selection must not supply a shortcut target after filtering.
- Multiple merge parents: shortcut uses first full SHA, buttons keep explicit parent targets.
- Paging request followed by another user selection must not override it later.
- File-history pause must allow exiting to full history; ordinary pause/error must not silently stall missing targets.
- Real history-search/commit editors retain editing behavior and draft; successful read-only navigation preserves Git state.

### Task 1: Parent navigation entry and pending lifecycle

**Files:** Create `internal/lobedesk/parent_navigation.go`, `internal/lobedesk/parent_navigation_test.go`; modify `shortcuts.go`, `history.go`, `app.go`, `i18n.go`; update `docs/shortcuts.md`, this plan and alignment ledger.

**Interfaces:** Consume `historyRows() []gitrepo.Commit`, `jumpToCommit(string)`, `selectCommit(string)`, `hasPendingDialog(toastMain)`, shared appShortcut. Produce `parentNavigationTarget() string` and `navigateToParent()`; use `SkipWhenTyping: true`, `ui.Alt`, `ui.KeyDown` in shared binding with execution-time guard recheck.

- [x] Write focused regressions for paused/error missing-target jump and user reselect cancelling pending, and actual shared-table Alt+Down for ordinary/merge/root/hidden selection, blocked states, input ownership and Git preservation. Assert concrete selected full SHA, reveal, cleared pending, preserved query for unavailable keys, and precise first-parent target with unrelated adjacent row.
- [x] Run focused tests before fixes: `go test ./internal/lobedesk -run 'TestParentNavigation|TestPendingJump' -count=1`; expect missing binding or explicit stale/paused pending assertion failure, not a fixture/build error.
- [x] Implement wrapper and shared binding; localize action/retry text. Fix demonstrated lifecycle defects at their shared source: a new explicit selection cancels older pending jump; missing target cannot queue behind ordinary paused/error history. Keep file-history exit/resume and existing pagination intact.
- [x] Run focused tests and related HEAD/history/file-tab navigation regressions; expect exit 0. Add actual widget input and asynchronous file-history completion coverage where existing helpers support deterministic checks.
- [x] Review final increment once with a role after implementation; fix verified blockers with RED/GREEN regression, without expanding scope.
- [x] Run related race tests, Windows/Linux compile coverage, owned macOS native Alt+Down acceptance where natural native focus permits; read results and inspect screenshot.
- [x] Run full make check after source stabilizes, read complete result, update evidence and diff formatting. Await the preceding file-tab full check before modifying production source so its evidence retains a stable input tree.
- [x] Deliver uncommitted changes with evidence and limits. Commit step intentionally omitted because user authorization is absent.

## Progress

All three reports support first-parent-only scope. Prior file-tab full make check completed exit 0 (ordinary lobedesk 93.914 s, race 359.481 s, vet clean); complete log read before parent source changes.

- RED confirmed missing shortcut, paused/page/read failure retaining pending, and stale request after newer commit selection (`build/alignment-parent-navigation-red.log`). Additional RED confirmed explicit worktree selection and asynchronous page failure (`build/alignment-parent-worktree-red.log`, `build/alignment-parent-page-error-red.log`).
- Implemented visible-history first-parent wrapper, shared Alt+Down with SkipWhenTyping and rechecked guards, localized labels and retry feedback. Existing jump path now refuses inert missing-target requests, clears pending on page failure, and cancels pending for explicit commit/worktree reselect including early-return paths.
- Focused GREEN `build/alignment-parent-navigation-green.log`: 1.339 s. Includes actual History search typing, first merge parent rather than adjacent/second parent, guards, 420-commit real fixture with source only in file history and target beyond loaded full-history page, final selected parent/detail completion, preserved draft/amend and complete Git/working-byte fingerprint.
- Related race `build/alignment-parent-navigation-race.log`: 18.524 s, exit 0. Windows/Linux amd64 CGO-disabled compilation passed; native platform runtime remains untested.
- Independent implementation review by testing role found no verified blocker; review was read-only, not an independent test execution. Checked reconcile/pending order, file-history resume and explicit parent buttons.
- Ordinary macOS e2e binary `build/supergit-lobe-parent-native`; native run `build/parent-navigation-native-5y_3s5pv`, log `build/alignment-parent-navigation-native-ready.log`, driver `build/.native-focus/parent-navigation.py`. Actual Alt+Down selected parent from child; parent SHA/details and file list loaded; root repeated key no-op; Cmd9 restored retained draft; dirty Index/refs/HEAD/status/working bytes unchanged. Screenshot inspected, process stopped 0. Initial sx3i38f1 run also passed selection/Git assertions but screenshot caught pending detail reads; added bounded readiness assertions and repeated successfully without production source changes.
- Native row focus remains at the invoking child while selection/reveal and details move to parent, following existing navigation behavior. No new focus restoration or native refocus. Reference GUI focus semantics remain unknown. Native scope: English/light/1100x720, simple single-parent and root only; merge and paging are deterministic regressions.
- Full make check `bg454u0tm` passed exit 0: ordinary lobedesk 92.741 s, race 363.178 s, vet clean; complete `build/alignment-parent-navigation-check.log` read. No production changes after its start. Diff formatting passed.
- No commit/install/release. No durable product rule recorded.
