# Remote Quick Checkout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend quick checkout to cached remote branches with explicit local targets and displayed-SHA revalidation, preserving drafts and existing local/commit-scoped checkout.

**Architecture:** Keep one DialogBase and one root/epoch/generation lifecycle across search, tracker-target selection and local naming. Remote rows retain full remote/ref identity. New TrackBranchChecked revalidates the displayed remote SHA under the repository-instance lock before existing protected checkout; tracker choices reuse checked local switching.

**Tech Stack:** Go, MyGo/Lobe, existing picker/field controls, Git CLI and isolated owned fixtures.

**Spec:** Ongoing user-authorized Sublime Merge capability alignment; product/design/testing reports from the two remote-checkout role cycles, recorded in 2026-11-17-sublime-merge-alignment.md. Installed CmdB/CtrlB resource preselects checkout_branch, with additional remote/create variants; candidate scope, ordering and reference GUI behavior remain unobserved.

## Global Constraints

- Preserve all existing uncommitted changes/generated_images. No commit/install/release/publication, real project checkout or remote access.
- Local quick-checkout full make check bqnt91plz passed and complete output/source hashes verified before edits: ordinary lobedesk 88.679 s, race 351.605 s, vet clean. Its native focus block remains open.
- Cached remote refs only, excluding symbolic HEAD; no automatic fetch. Full remote/ref identity must survive display, filtering and execution, including same short name on different remotes.
- Zero trackers: enter creation form. One or more exact-Upstream trackers: explicit local-target choices plus create-new choice, no automatic first tracker. Current/occupied/dirty/conflict/operation/read-error reasons remain visible and execution is guarded.
- Creating a local branch always requires an explicit editable local name and a button naming creation plus checkout. Existing local-name collision cannot overwrite/reuse/change upstream. Local target switching binds full displayed local SHA and exact upstream identity.
- Keep search query/cursor through the later steps. Back/Escape in naming/target step returns to previous step; search-step Escape/backdrop exits and restores original surviving input. One overlay retains opener; step inputs focus once, with Tab available to Back/Create controls.
- Root/load epoch/generation and selected remote name/SHA invalidation must reject stale callbacks. Close before starting Git operation. No implicit detached/stash/force or destructive rollback.
- Checked backend compares cached displayed SHA with current local remote-tracking ref, not server state; instance lock does not serialize independent Git processes, and check/switch is not externally atomic.
- Existing legacy remote-menu and commit-scoped picker behavior remains outside this extension unless a shared change is necessary and separately tested. Generic command palette catalog remains nine items.
- Existing Operate tokens; one constrained DialogBase with 16 px gutters, no repeated tooltip, no new setting/animation/config repair. Owned fixtures and isolated synthetic config only; compile coverage is not target-OS native acceptance.

## Decisions

Ruling: use one overlay with explicit search/target/name steps rather than closing it to open askPrompt. Existing prompt has root-only lifecycle and no return-query protocol; keeping the opener and state is simpler and avoids stacking/focus races.

Ruling: use explicit target rows instead of silently reusing trackingBranch's first match. Existing tracker choice may have a different local tip from the remote tip; display and validate the actual local destination without moving it or changing upstream.

Ruling: new checked API preserves legacy TrackBranch behavior via a shared private implementation; only the new quick-checkout path opts into expected remote SHA. Existing separate menu behavior is not silently broadened.

## Review Focus

- Two remotes with the same short branch name and unrelated same-name local branch must never redirect a choice.
- Multiple matching trackers must require an exact explicit local target; no upstream change or tip reset.
- External remote movement/deletion, root/epoch/generation changes and local/upstream changes after choosing must fail without unintended local mutation.
- Returning from naming/target choices preserves query/highlight; field focus is one-shot and final cancellation restores the original real input.
- Invalid/colliding local names, dirty/conflict/operation/occupied states, no results and narrow bilingual/theme layouts remain understandable; failure is reported by existing operation feedback.

### Task 1: Checked remote branch creation

**Files:** Modify internal/gitrepo/branches.go; create internal/gitrepo/track_branch_checked_test.go.

**Interfaces:** Produce Repository.TrackBranchChecked(ctx context.Context, name, remoteRef, expectedSHA string) error and a recognizable changed-target error; reuse existing branchName, clean, resolve and protected switch arguments. Preserve TrackBranch(ctx,name,remoteRef) callers.

- [x] RED/GREEN: owned bare/clone fixture verifies exact HEAD/upstream/config on success; invalid/moved/deleted expected remote SHA, existing local-name collision, dirty and ignored collision reject with original state preserved. Existing legacy TrackBranch regression remains green.

### Task 2: Explicit remote choices and local naming

**Files:** Extend internal/lobedesk/quick_checkout.go; split remote step helpers into quick_checkout_remote.go if needed; add quick_checkout_remote_test.go. Update necessary localized copy and shortcut docs only.

**Interfaces:** Consume a.remoteBranches []gitrepo.Branch and a.branches exact Upstream matches. Extend quickCheckoutState with step, remote identity, target/local name and once-focus state. Produce remote choice/name handlers that consume generation and displayed SHA, then close before performAs calls TrackBranchChecked or checked local switch.

- [x] RED/GREEN: actual controls cover zero/one/multiple trackers, two remotes/same short names, editable name/collision, Back/Escape query/cursor/original focus, Tab escape, stale callbacks and no background keys. Owned Git UI tests verify exact remote-created branch/upstream and unchanged rejected state/draft; retain local/commit-scoped regressions.

### Task 3: Review and acceptance

- [x] Independent scoped implementation review; reproduce/fix material findings with regression evidence.
- [ ] Related race, final Windows/Linux builds, bilingual/theme/narrow captures and bounded native attempt; retain blocked acceptance honestly, no blind rerun.
- [x] Full make check on final stable input, complete output/source hash verification and diff formatting; record evidence and deliver uncommitted result.

## Progress

Planning: product a0d87fd409e73a02b recommends explicit tracker choices and editable collision-free local naming. Design af69dc2f4b6b52c65 recommends one DialogBase and retained return state/focus. Testing aeeb00ef6eb413460 identifies expected-SHA rejection, name/config preservation, dirty/conflict/ignored collisions and instance-lock limits. Main verified existing legacy first-tracker and unchecked remote-creation paths.

- Backend RED: build/alignment-remote-checkout-backend-red.log missing TrackBranchChecked. Shared private trackBranch preserves legacy TrackBranch; checked API validates OID and current local remote-tracking tip, uses ErrSwitchTargetChanged and existing clean/protected switch. GREEN 3.935 s includes checked success/rejections and legacy clone/fetch/push/pull. Owned bare/clone tests preserve HEAD/ref/config/index/working bytes on invalid/empty/moved/deleted SHA, existing destination and dirty/staged/untracked/ignored collision.
- UI RED build/alignment-remote-checkout-ui-red.log reproduced missing remote row/explicit tracker choices. Implemented quick_checkout_remote.go and single-overlay search/targets/name steps. Search combines all locals and cached non-symbolic remotes; RemoteBranches already excludes symbolic refs, so literal feature/HEAD is preserved. New branch naming never silently reuses a local; explicit tracker execution rereads remote SHA and exact local name/SHA/upstream, then uses SwitchBranchChecked. Creation calls TrackBranchChecked after closing. Existing menu/commit-scoped behavior stays separate.
- Independent review aeeb00ef6eb413460 found target search repeatedly taking Tab focus and Back from naming resetting create selection to first tracker. Reproduced both in build/alignment-remote-checkout-review-red.log; fixed one-shot search focus and saved target cursor. Corrected Tab test to traverse the two legitimate option stops before Back; build/alignment-remote-checkout-tab-focus.log shows actual traversal. Form-button Escape was separately RED (build/alignment-remote-checkout-tab-red.log); panel shortcut now returns within the overlay. No second review round required for covered fixes.
- Additional RED/GREEN: full-name filtering preserves nested feature/HEAD (identity-red.log), tracker choice rejects external remote-tip movement (tracker-red.log). Expanded actual controls cover zero/one/multiple trackers, same short remote name, collision feedback, generation/root/epoch/busy/read error, external moved/deleted remote/dirty bytes, exact tracker destination without resetting its tip, name editing and final Body typing after cancel. Related run build/alignment-remote-checkout-related.log passed: gitrepo 4.443 s, lobedesk 11.685 s.
- Windows/Linux amd64 CGO-disabled builds and ordinary macOS e2e binary passed; compile coverage only. Twenty-four step/theme/language/size renders and bounds passed (build/alignment-remote-checkout-captures.log, build/remote-checkout-captures). Main inspected narrow English light naming and wide Chinese dark targets. Design af69dc2f4b6b52c65 inspected all 24 with no operational layout blocker.
- Design minors deferred: make create-target row's next-step affordance clearer; label name explicitly Local branch name; improve dark selected-text contrast (existing input theme). No new visual system or unrequested global SDK change.
- Native driver build/.native-focus/remote-quick-checkout.py uses an owned local-only remote fixture and isolated HOME. Attempt build/remote-checkout-native-aanrrd_f stopped during five-second natural native-focus prerequisite before draft or CmdB input; process exit 0. Complete build/alignment-remote-checkout-native.log read. Native interaction is unverified; no workaround or blind rerun. Combined native checkbox remains open.
- Initial make check b43taomnt was stopped to verify a suspected mouse-Back focus path before finalizing. Actual click/return/typing test passed without any product-code change (build/alignment-remote-checkout-mouse-back-red.log contains PASS despite its initial filename); no reproduced mouse defect. Retained TestQuickCheckoutRemoteMouseBackRestoresSearch. Interrupted output preserved in build/alignment-remote-checkout-interrupted-check.log; it is not final acceptance evidence. That check bcjiobs90 was also stopped after the follow-up role cycle produced two reproducible user-path defects (see below). Partial output is preserved in build/alignment-remote-checkout-before-review-fixes-check.log, not final evidence. Check b7yvc5abb failed: the generated overlay reproduction Go file in build/ was discovered as a separate supergit/build package and could not resolve lobedesk test helpers. Full failure log read and preserved as build/alignment-remote-checkout-scratch-package-failure.log; ordinary lobedesk passed 98.187 s, but race/vet did not run, so this is not full acceptance. Moved only the task-generated scratch copy to build/.review-fixtures/ and updated its overlay path; permanent regressions remain in internal/lobedesk. go list ./... confirms no supergit/build package. No production/test input under internal/cmd changed. Final rerun b8nkpzqwa passed exit 0: ordinary tests reused valid cache, race lobedesk 360.739 s, vet clean. Complete build/alignment-remote-checkout-check.log read; every recorded internal/cmd source hash matched (build/alignment-remote-checkout-source-verification.log) and git diff --check passed before next-feature source edits. Native focus remains blocked, so combined native checkbox remains open.
- Ruling: existing backend RemoteBranches filters symbolic refs, so trust that result rather than rejecting all names ending /HEAD — otherwise legitimate nested branches disappear. Cost: UI relies on the established backend contract.
- Ruling: Tab may stop at selectable options before Back; assert eventual reachable Back rather than the next Tab always landing there — follows actual accessible control order. Cost: traversal has more than one stop.
- Follow-up three-role cycle: product afe1dbabaae3cd4ba retains current remote decisions and recommends palette-to-checkout handoff next. Design a22b6a4d7b41b9d47 identifies invalid names losing the form; testing a40e0ef449fdfd7c9 identifies indistinguishable local refs/heads/origin/topic versus remote refs/remotes/origin/topic. Both actual-control reproductions failed in build/alignment-remote-checkout-followup-red.log via a Go overlay without changing running full-check sources. Then stopped the incomplete older full check and fixed both.
- New permanent quick_checkout_review_regression_test.go verifies candidate identities differ and eleven invalid name forms keep inline feedback, editable value and draft, then accept correction in place. Rows visibly/accessibly distinguish local/remote category. UI validates ref-format restrictions before closing; backend Git check-ref-format remains authoritative. Related GREEN build/alignment-remote-checkout-followup-green.log: gitrepo 4.498 s, lobedesk 13.769 s. Final Windows/Linux builds passed. Regenerated 24 captures/bounds; main inspected narrow English light search and Chinese dark targets with category text. No fresh native run after these UI changes; earlier focus prerequisite block remains the only native evidence.
- No commit/install/release, real project checkout, remote network access or durable product-rule write.
