# Unstage All Shortcut Implementation Plan

**Goal:** Match Sublime Merge's Cmd+Shift+R / Ctrl+Shift+R Unstage All binding using the existing guarded batch operation.

**Architecture:** Add one workspace entry to shortcutTable, sharing the existing unstageAll label with the settings card. Availability and execution both check that working-tree Files are visible, detail navigation is allowed, no chooser/dialog is active, and batchFileTargets("unstage") accepts a nonempty scope. Keep existing batch validation and completion refresh.

**Constraints:** Preserve all existing uncommitted work. No commit/install/release. No new settings or UI. Keep original native acceptance failures documented. Native keyboard acceptance is blocked by failure to obtain verified native window focus; deterministic UI Tester plus real Git is separate evidence.

**Reference:** Installed Build 2125 Default - Merge keymaps bind super+shift+r (OSX:339), ctrl+shift+r (Windows:316/Linux:317) to unstage_all. Stage All Modified is a different operation; do not wire Cmd+Shift+A to the current Stage All implementation.

## Task 1: Dispatch and verify Unstage All

- [x] Write tests using registerShortcuts and ui.Tester.Key. Compare real Index/working files/draft/HEAD after callback refresh for existing and unborn repositories.
- [x] Cover busy/loading/nil/empty/history commit/other page/hidden detail/chooser/ref picker/dialog/protected entries and stale execution availability; assert no operation starts.
- [x] Watch focused tests fail before adding the binding. After correcting a nonexistent test-only fileQuery field to existing filesCollapsed, RED showed missing binding and no operation for committed/unborn.
- [x] Add workspace.unstageAll, Cmd|Shift+R, platform-specific display text and a predicate rechecked in Run. Reuse unstageAll.
- [x] Run focused tests, including settings-card/table checks. Focused GREEN 1.266 s; expanded real keyboard/Git/refresh/guard and batch/settings race passed 3.242 s. Independent product/design/testing reviews found no blocker; their reviews were read-only, not independent test execution.
- [x] Correct shortcut documentation and independently re-read installed reference keymap/command resources. Windows/Linux amd64 CGO-disabled builds passed, compile coverage only.
- [x] Complete make check task bi1yyx528, exit 0. Coordinator read build/alignment-unstage-all-shortcut-check.log: ordinary lobedesk 82.449 s, race lobedesk 348.258 s, vet completed without errors. git diff --check also passed. Native runtime remains unverified because window focus cannot be obtained in the existing acceptance environment.

## Rulings and evidence

Ruling: Implement on the existing feature branch alongside the current uncommitted integration rather than copy the incomplete tree into another implementation worktree. The three independent review roles retain their own worktrees and read main's absolute files. No commit is authorized.

Ruling: The prior native focus failure is bounded and remains open; it does not block deterministic verification of this independent shortcut. This does not establish complete native or cross-platform runtime parity.

Final: minor (deferred): Existing Unstage All button tooltip does not advertise the new shortcut; Settings already lists it from the shared table. No new tooltip copy was added in this narrow increment.

Final: input-focus follow-up closed by batch_shortcut_commit_fields_test.go in the Stage tracked increment: actual Summary/Body × both batch keys now verify successful Git/refresh, full draft and focus preservation, plus ordinary Select All/Copy. Related current focused race passed 3.732 s. This remains deterministic UI testing, not native runtime acceptance. Shared notice-details dialog coverage still relies on hasPendingDialog rather than an additional dedicated shortcut test.

Review scope: product inspected the final binding/predicate/tests; design verified shared-table placement and label consistency; testing checked real dispatch and Git invariant coverage. No production edits followed the focused race run or the start of make check.
