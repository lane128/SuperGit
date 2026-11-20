# Quick Branch Checkout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Cmd+B/Ctrl+B to search and check out an existing local branch while preserving the existing repository protections and commit-scoped picker.

**Architecture:** A dedicated root/epoch/generation-scoped picker consumes the complete local a.branches list and existing search/cursor/row primitives. It closes before invoking switchBranchAt(name, displayedSHA), retaining the backend clean check and displayed-target SHA check. The existing commit-scoped checkout dialog and its selection semantics remain separate.

**Tech Stack:** Go, MyGo/Lobe DialogBase, shared picker helpers, owned Git fixtures and UI Tester.

**Spec:** Ongoing authorized Sublime Merge alignment and product/design/testing reports of 2026-11-18. Installed resources map CmdB/CtrlB to show_command_palette with command checkout_branch. Resources do not establish default candidate scope, ordering or reference native focus behavior. This increment is a local-branch adaptation; remote/tag/create-branch variants remain outside it.

## Global Constraints

- Preserve existing uncommitted work and generated_images. No commit/install/release/publication or dependency/config repair.
- Wait for command-palette full check bzrrsksc8 and read its complete output before source/test edits. Keep final source inputs stable until then.
- CmdB/CtrlB opens one local-branch picker; no stacking during another main dialog, chooser/ref popup, busy/loading/history reload. Shared shortcut metadata supplies the new entry and Settings label.
- Candidate scope is all local branches, never branchesAt(selectedSHA), remote refs or tags. Keep existing commit-scoped checkout behavior and expected-SHA validation unchanged.
- Current branch and branches occupied in another worktree remain visible, with reasons and no selectable action. Dirty/conflict/active-operation status must not start checkout; backend clean and Git ownership checks remain authoritative for external races.
- Search, arrows, Enter, disabled/empty feedback, scrolling, active descendant and Esc/backdrop focus restoration follow the existing picker/command-palette behavior. No redundant row tooltip that consumes the first Escape.
- One styled DialogBase panel must fit the window with 16 px gutters; focus search after overlay captures its opener. No new animation or visual system.
- Capture root, loadSeq and generation; stale callbacks after repository reload/switch or close/reopen cannot act. Revalidate target name and displayed full SHA before closing; close before calling checked checkout.
- Preserve draft/amend and all Git/working bytes on cancel or rejection. Successful checkout may change only the owned fixture as expected; no implicit stash, force, discard or detached checkout.
- All acceptance uses owned disposable repositories, synthetic identities and isolated config. No real project checkout, user remotes or OS permission workaround. Cross-builds are compile coverage only.

## Design decisions

Ruling: use a separate global local-branch picker state instead of weakening the existing commit-scoped state. Shared helpers can be reused without changing the SHA scope of existing history actions. Cost: a small amount of distinct lifecycle code.

Ruling: call existing switchBranchAt with the candidate's displayed SHA rather than unchecked switchBranch. External target movement must produce the existing changed-target rejection rather than checking out an undisplayed commit.

Ruling: retain the generic command palette's existing nine-action catalog in this increment. CmdB/CtrlB directly enters the checkout picker as an adaptation of the resource's preselected command. Remote/create/tag commands and command-palette chaining need separate catalog decisions.

## Review Focus

- A candidate whose tip differs from the selected historical commit must still appear, while old commit-scoped picker filtering remains correct.
- Dirty state, current branch, other-worktree ownership and busy transitions must give understandable feedback and preserve local state.
- External branch target movement and root/epoch/generation changes must invalidate the exact old choice, without falling back to another branch.
- Opening/cancelling from actual Summary/Body/search must preserve draft and input ownership; one delayed Escape must close the picker.
- Disabled-only/empty results, long branch names, Chinese/English and narrow light/dark windows must remain navigable and contained.

### Task 1: Guarded local branch search

**Files:** Create internal/lobedesk/quick_checkout.go and quick_checkout_test.go. Modify app.go (state), shortcuts.go (metadata), dialog_scope.go (ownership), view.go (render), i18n.go (necessary labels/reasons); refactor only the existing branch-row shell if sharing is useful without changing commit-scoped behavior. Update docs/shortcuts.md and evidence ledger.

**Interfaces:** Consume a.branches []gitrepo.Branch, a.worktrees []gitrepo.Worktree, status, loadSeq, switchBranchAt(name, sha string), filterBranchesAt and picker helpers. Introduce quickCheckoutState, canOpenQuickCheckout() bool, openQuickCheckout(), quickCheckoutDialog(*ui.Context), quickCheckoutUnavailable(gitrepo.Branch) string and runQuickCheckout(name, sha string, generation uint64).

- [x] RED: actual controls/dispatcher for CmdB/CtrlB, all-local scope, filtering/arrows/exact Enter, current/occupied/dirty/empty cases, stale callbacks and cancelled original input. Observe focused test failure before implementation.
- [x] Implement scoped picker, shared metadata/render/modal ownership, availability reasons and close-before-checked-action lifecycle.
- [x] GREEN: actual picker plus existing commit-scoped checkout, command-palette, shortcut and focus regressions. Owned Git fixture verifies exact successful branch/HEAD and unchanged rejected/cancelled state.
- [x] Independent scoped implementation review; reproduce and fix material findings with regression evidence.
- [ ] Related race, final Windows/Linux compile, bounded owned native keyboard attempt and representative rendered layout; inspect evidence, preserve any native blocker honestly.
- [x] Full make check on stable final source; read complete output, check formatting and record results/limits. Deliver uncommitted result.

## Progress

Initial planning reports: three independent-worktree roles reported: product conclusions delivered by acbd1c30b2ad96e23; design conclusions delivered by a75014b919100a2f1 (launch labels and report role names were crossed); testing ae6cfc58a7db69046 independently ran TestCommandPalette once and passed in 2.100 s. Design used emil-design-eng and impeccable, read current native capture and source, and did not run GUI or edit configuration. Main verified SwitchBranchChecked validates displayed SHA and backend clean before git switch --no-guess --no-overwrite-ignore. Command-palette full check bzrrsksc8 passed exit 0; full log read and source hashes matched before implementation (ordinary lobedesk 92.484 s, race 359.438 s, vet clean).

- Renewed three-role cycle: product conclusions a9a0451f67b06f2d7, design ace52594ca1175b6c (Emil + Impeccable), testing a85c80e2e889a21a5; launch labels/report role names crossed, so attribution follows report contents. Product confirmed dirty state can open for inspection while execution is disabled. Design confirmed single DialogBase, focus after opener capture, visible reasons and no repeated tooltip.
- Ruling: dirty/conflict/active-operation state permits viewing/searching the local catalog but disables execution, with distinct visible reasons. Busy/loading/history reload/chooser/ref popup/other dialogs block opening. Reopening starts an empty search; backend failure closes the picker and reports through the existing toast/reload flow. Cost: failed actions require reopening and searching again.
- RED build/alignment-quick-checkout-red.log: missing quick-checkout state/methods. Added separate state/render/helper file and shared CmdB/CtrlB metadata. Existing commit-scoped picker unchanged; generic command catalog still has nine entries. Modal ownership helpers distinguish the two search pickers from action dialogs; retained window dispatcher blocks both pickers.
- First implementation compile exposed an incorrect assumed statusErr field; inspected existing reload flow and used actual loadErr. No backend behavior change. Expanded focused GREEN build/alignment-quick-checkout-green.log passed 3.909 s; related existing checkout/palette/commit-focus/shortcut tests build/alignment-quick-checkout-related.log passed 7.102 s.
- Owned Git tests verify exact target branch/HEAD after actual search+Enter and asynchronous reload; external dirty bytes, branch-tip movement and other-worktree occupancy reject while preserving HEAD/Index/working bytes, draft/amend and feedback. Guard matrix includes current/occupied/dirty/conflict/operation/busy/loading/history/chooser/ref/dialog/root/epoch/generation/tip/closed and all three status/branch/worktree read-error paths. Actual Summary, Body and History search restore typing after Esc/backdrop; delayed focus-visible single Escape and empty Enter covered.
- Independent testing-role static implementation review found no specific blocker; suggested missing conflict/read-error/closed-callback coverage was added. No independent test rerun claimed. Final related race build/alignment-quick-checkout-race.log passed 13.730 s. Windows/Linux amd64 CGO-disabled builds passed, compile coverage only.
- Eight rendered captures in build/quick-checkout-captures (English/Chinese × light/dark × 360×360/900×640) and control/dialog bounds passed. Main and design role inspected narrow English light and wide Chinese dark; no usability rendering blocker or required polish. Long names truncate visually and retain full accessible labels; no repeated row tooltip.
- Native ordinary e2e binary build/supergit-lobe-quick-checkout-native; driver build/.native-focus/quick-checkout.py; owned local main/topic fixture and isolated HOME/config. Attempt build/quick-checkout-native-cufnn0ut blocked during bounded five-second natural window-focus prerequisite before any CmdB/draft input. Process stopped exit 0; full build/alignment-quick-checkout-native.log read. Not native success or proof of shortcut failure; no refocus workaround or blind rerun. Combined native validation step stays unchecked.
- Full make check bqnt91plz passed exit 0: ordinary lobedesk 88.679 s, race 351.605 s, vet clean. Complete build/alignment-quick-checkout-check.log read; all internal/cmd hashes matched build/alignment-quick-checkout-final-sources.sha256 after completion. Diff formatting passed. Existing native focus issue remains open; combined native-acceptance checkbox stays unchecked.
- Follow-up independent testing role a98f20113cdd73ef7 executed TestQuickCheckout once, passed 3.505 s exit 0. No reproducible behavior defect found; native precondition block explicitly retained. No running full-check output inspected or duplicate full suite started.
- No commit/install/release, real repository checkout, remote access, configuration repair or durable product-rule write.
