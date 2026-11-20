# Fast Forward Pull Shortcut Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add F7 as a guarded keyboard entry to SuperGit's existing explicitly fast-forward-only Pull action.

**Architecture:** Reuse pullFF, performAs and Repository.PullFF without changing transport or integration strategy. Shared table supplies the key and Settings label; a small manual-shortcut predicate/wrapper checks current UI state and known upstream availability before dispatch.

**Tech Stack:** Go, MyGo/Lobe, existing Git operations and owned local bare-repository fixtures.

**Spec:** This document's scope derives from ongoing authorized Sublime alignment and three role reports. Installed resource F7 maps to pull with no arguments on all three platforms. Resources do not establish fast-forward-only semantics; SuperGit continues to expose its existing fast-forward-only action. Shift+F7 opens a reference command palette, outside this scope.

## Global Constraints

- Preserve unrelated dirty work and generated_images; no commit/install/release/publication.
- Do not alter PullFF to merge/rebase, auto-stash, force or overwrite; existing clean-worktree check and Git errors remain authoritative.
- F7 only in this increment. No command palette or modifier aliases, remote settings or new confirmation flow.
- Repository present and ready, upstream present, no busy/loading/history reload/chooser/ref popup/main-window dialog; request and retained Run both recheck. Known current-branch UpstreamGone must not be bypassed.
- Plain F7 may work in actual Summary/Body/History search without consuming text, moving focus or losing draft; unavailable/modified keys leave normal handling intact.
- Existing fast-forward operation busy/progress/failure/success and reload flow remains; repeated input cannot reenter while busy.
- Scope existing toolbar/menu upstream differences explicitly: investigate before changing shared behavior. Do not silently broaden eligibility to detached/no-upstream states or turn rejected remote metadata into new network behavior.
- All mutating acceptance uses newly owned bare/clone fixtures, synthetic identity, isolated HOME/config and local transports. Never operate on the real project repo or contact user remotes.
- Keep production inputs stable until second-parent full check completes. Windows/Linux cross-build is compilation only.

## Review Focus

- Retained shortcut callback invoked after a dialog/chooser or busy transition must not start Pull.
- Input focus must survive both shortcut dispatch and asynchronous reload; inspect actual field rather than substitute.
- Missing/deleted upstream and detached state must not bypass menu/operation constraints.
- Divergence, dirty worktree and conflicts must preserve local HEAD, Index and working bytes; fetched tracking refs may legitimately update where fetch occurred.
- Fast-forward success must update exact expected commit/tree once; repeated F7 must not launch concurrent operations.

### Task 1: F7 entry and owned integration acceptance

**Files:** Create `internal/lobedesk/pull_shortcut.go`, `internal/lobedesk/pull_shortcut_test.go`; modify `shortcuts.go`; modify existing shared Pull guard only if reproduced bypass makes it necessary. Update shortcut docs and evidence ledger.

**Interfaces:** Consume `pullFF()`, `hasPendingDialog(toastMain)`, `status.Upstream`, `branches` and existing operation callbacks. Produce `canPullShortcut() bool`, `pullFromShortcut()` and shared `repo.pullF7` binding using bare ui.KeyF7 and existing pull label.

- [x] RED: missing binding, actual editor dispatch, retained Run guard matrix including known UpstreamGone, no reentry; verify precise operation label/busy state and draft/focus preservation. Run `go test ./internal/lobedesk -run '^TestPullShortcut' -count=1`; expect missing-key/action assertion failure.
- [x] Implement predicate/wrapper and binding with request/Run rechecks, preserving current PullFF semantics and existing failure copy. Ledger any necessary shared-entry change with reproducing evidence.
- [x] GREEN: real owned local bare plus peer clone fast-forward via actual F7 and asynchronous callbacks; assert exact target HEAD/tree, no unrelated changes, preserved draft; divergence/dirty/guard cases must not rewrite local state. Reuse existing backend coverage where behavior is unchanged instead of duplicating every backend test.
- [x] Scoped independent implementation review; fix verified blockers with RED/GREEN evidence.
- [ ] Related race and Windows/Linux compile; owned macOS native F7 on current ordinary binary with actual ready editor and Git state assertions, inspect screenshot and stop owned process.
- [x] Run/read full make check on stable source; update evidence and diff formatting; deliver uncommitted result and limits.

## Rulings and progress

Ruling: F7 means SuperGit's existing fast-forward-only Pull; reference pull mode remains unproven, so this is a keyboard alias adaptation and not complete semantic parity. If reference allows merge/rebase by default, broader Pull-mode alignment remains an explicit future product decision.

Planning only. Product/design/test reports read-only. Main confirmed existing backend first requires clean worktree/upstream, fetches configured remote then runs merge --ff-only --no-edit --no-autostash --no-overwrite-ignore. Toolbar uses status.Upstream; current-branch menu additionally hides UpstreamGone. Existing perform entry protects repo/busy/loading, so new shortcut must cover modal/chooser state itself. Second-parent full check b8m1xnnmk passed exit 0 (ordinary lobedesk 85.956 s, race 353.046 s, vet clean); full log read before Pull source edits. Pull shortcut execution evidence:

- RED `build/alignment-pull-shortcut-red.log`: missing shared binding for guard matrix and owned fast-forward/dirty/divergent scenarios.
- Implemented only pull_shortcut.go plus shared bare F7 binding, existing pull label; no toolbar/menu/backend changes. Predicate checks repo, current branch/upstream, known current UpstreamGone, busy/loading/history reload/chooser/ref-popup/main dialogs and Run rechecks before calling existing pullFF.
- Initial focused GREEN 2.509 s (`build/alignment-pull-shortcut-green.log`). Expanded final related race passed 7.850 s (`build/alignment-pull-shortcut-race.log`): actual Summary, Body and History search preserve draft/focus/query; owned bare+clone fast-forward exact HEAD/clean tree; dirty and divergent rejections preserve local HEAD/Index/work bytes; modified keys and operation non-reentry; notification-details dialog added to guard matrix.
- Independent implementation review found no blocker. Its actual Summary/search and notice-details coverage suggestions were added; retained Run recheck is tested directly, not by replaying a captured SDK window callback.
- Windows/Linux amd64 CGO-disabled builds passed, compile coverage only.
- Native attempt `build/pull-shortcut-native-8a86jz1d/`, driver `build/.native-focus/pull-shortcut.py`, log `build/alignment-pull-shortcut-native.log`: current ordinary binary `build/supergit-lobe-pull-native` launched the owned local-remote fixture, but repeated windows snapshots lacked native focused state during bounded five-second prerequisite. Stopped exit 0 before filling editors or sending any F7. This is blocked native acceptance, not a shortcut defect or success. No unconditional refocus/permission workaround or blind rerun. Existing intermittent native focus diagnosis remains open.
- Full make check `b7tk489dk` passed exit 0; complete `build/alignment-pull-shortcut-check.log` read before palette source edits. Ordinary lobedesk 86.675 s, race 361.430 s, vet clean; production source stable throughout. Native keyboard success remains unverified; the combined native-acceptance step above stays unchecked.
- Ruling: leave existing toolbar/menu guards unchanged, because shortcut can enforce stricter modal/detached/known-deleted-upstream availability without changing existing visible entry behavior. This preserves legacy differences for a separate scoped decision; no broad guard refactor is required for the alias.
- No commit/install/release, real-remote writes, config repair or durable product rule recorded.
