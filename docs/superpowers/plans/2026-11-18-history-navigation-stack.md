# Commit Browsing Back and Forward Implementation Plan

> **For agentic workers:** Use the already-loaded executing-plans and test-driven-development guidance. The main agent implements; a separate reviewer checks the bounded final change. No commit or release without explicit user authorization.

**Goal:** Let users return along successful explicit commit jumps within one ordinary fixed-tip history scope, preserving the selected SHA, search context, drafts and Git state.

**Architecture:** Keep a scoped browsing trail and a separate pending explicit navigation transaction. Parent, second-parent, chosen child and same-range HEAD navigation opt into recording; ordinary selectCommit callers never create trail entries. Final selection can complete a transaction after paging, but manual selection and stale repository/history generations cancel it. Replay selects already-recorded commits without widening the reference range or entering file history.

**Tech Stack:** Go, existing MyGo shortcut table/input guards, history paging callbacks, actual UI Tester controls and disposable Git repositories.

**Spec:** User-authorized Sublime Merge alignment; product a62a39d85e27d5b80, design a93ddc99a88e6b7c4 and testing af31fa01cc7d32f49 reports, recorded in 2026-11-17-sublime-merge-alignment.md. Main independently verified installed Build 2125 Default - Merge.sublime-package keymaps: OSX lines 77–78 super+left/right, Windows lines 82–83 and Linux lines 84–85 alt+left/right invoke navigate_back/forward. Reference GUI trail semantics remain unobserved; the behavior below is a SuperGit adaptation within the authorized task.

## Global Constraints

- Child-navigation full check b1drvyh4n passed before this increment: ordinary lobedesk 101.369 s, race 366.112 s, vet clean, complete output and source fingerprints verified. Preserve all unrelated dirty work and generated_images/.
- No commit, push, installation, packaging, publication, real-project Git write, real remote or permission bypass.
- First scope is same repository, ordinary history, same load epoch and history reload/fixed-tip generation. Repository/ref/file-history transitions and refresh invalidate the trail. Plain text search changes do not change the fixed-tip traversal.
- Record SHA plus exact historyQuery. Only successful explicit first-parent, second-parent, child choice or same-range HEAD jumps create entries. Same SHA, cancelled picker, failed/unresolved jump, ordinary arrows/clicks and automatic reconciliation do not add entries.
- A manual selection away from the current trail anchor invalidates that trail and cancels any pending transaction; the next explicit jump starts a fresh trail. Search edits preserve a trail only while the current anchor remains the selected visible historical commit.
- Back/Forward restore the recorded query and reveal the recorded commit only after validating its membership in the current ordinary history source. Never call showCommit to expand refs, or exit file history during replay. Hidden-by-query targets become visible through their recorded query.
- Back after A → parent P → child B returns P then A; Forward returns P then B. New successful explicit navigation after Back truncates the old forward tail.
- A new user search edit cancels an in-flight explicit jump even if the source remains visible. Late page data may append, but cannot clear the new query, change selection or record the old target. Same-SHA HEAD keeps existing clear-query/reveal behavior without adding a position, and synchronizes the current trail anchor query to empty so Back/Forward cannot restore stale text.
- A pending pagination jump has a captured source, target, query, root/load/reload identity. Do not move the trail cursor until the exact target is selected. New selection/scope or failed/cancelled traversal cancels the transaction; stale callbacks cannot add entries or revive state. Existing pendingJumpSHA behavior remains authoritative for paging.
- Scope-changing HEAD or parent navigation keeps existing behavior and invalidates the local trail. Same-range recording must never block those existing commands.
- Shared metadata supplies macOS Cmd+Left/Right and Windows/Linux Alt+Left/Right. Copy the established first-parent typing policy, then set platform modifiers. Real text fields retain editing keys; unavailable directions do nothing; busy/loading/chooser/ref/modal states block execution and retained callbacks.
- No new toolbar, setting, animation, generic-palette command or tooltip. Reuse existing localized notices for read failure; add only necessary bilingual Settings action labels. Draft/amend, refs, HEAD, Index, status and work bytes are invariant.

## Review Focus

- Distinguish actual browsing trail from inverse parent/child relationships on a fork.
- Search restoration must preserve a visible selected anchor and subsequent list operations; cached per-frame history rows cannot override replay.
- Manual selection, worktree selection, automatic reconciliation and reload callbacks must never accidentally create records.
- A delayed parent paging completion cannot record after another explicit selection, root/ref/file switch, pause, failure or refresh.
- Same-range HEAD is recordable; scope-expanding HEAD and existing cross-file parent navigation keep their current semantics and clear the trail.

## Main Decisions

Ruling: scoped trail instead of full page/session history — fits the existing relationship commands without implementing cross-repository drafts, tabs or scroll restoration. Cost: crossing a scope or refreshing discards Back/Forward positions.

Ruling: record explicit successful navigation rather than every selection — avoids polluting history with ordinary list movement and asynchronous reconciliation. Cost: ordinary mouse/arrow browsing is not recoverable with this first increment; document the scope accurately.

Ruling: save SHA and query rather than SHA alone — jumpToCommit clears text search, so query restoration is necessary to return to the actual search context. Cost: recorded positions must be revalidated before restoring their query.

Ruling: manual anchor changes invalidate the trail — keeps Back/Forward behavior deterministic when ordinary selection is outside the recorded route. Cost: users cannot resume the older trail after manually choosing an unrelated commit.

## Task 1: Scoped trail, replay and lifecycle

**Files:** create internal/lobedesk/history_navigation.go and history_navigation_test.go; add focused app state. Integrate explicit navigation callers in parent_navigation.go, child_navigation.go, head_navigation.go and commit_info.go (all existing parent SHA buttons, including third and further parents). Adapt only the needed completion/cancellation hooks in history.go and app.go.

**Interfaces:** Consume selectedSHA, worktreeSelected, historyQuery, historySource(), historyReload, loadSeq, pendingJumpSHA and jumpToCommit. Define private position/scope/transaction types and entry methods for explicit jump, successful selection, cancellation, canReplay(direction) and replay(direction). Finalize names after inspecting all pendingJumpSHA completion/cancellation sites; do not introduce recording in generic selection.

- [x] Write behavioral RED: fork A → P → B, Back P/A, Forward P/B, exact query restoration and reveal; manual selection reset; Back then new successful jump truncates Forward.
- [x] Write RED for no-op/cancel/failure, chosen child generation, first/second-parent keys and all existing parent SHA buttons (including third/further parent), and same-range HEAD; out-of-scope HEAD/file parent preserve existing behavior while clearing trail.
- [x] Write RED: A→P, edit query while P remains visible, same-SHA HEAD clears query without duplicate P; Back restores A's query, Forward restores P with empty query. Delayed A→P plus new query matching A cancels old selection/recording even though the anchor remains visible.
- [x] Write RED for delayed pagination completion, repeated input, newer selection/root/epoch/reload/ref/file changes, pause/error and stale callback. Assert no premature cursor/record update.
- [x] Implement minimal private scoped trail and explicit transaction integration. Keep read-only navigation and existing pending paging intact.
- [x] Run focused tests and existing child/parent/HEAD/history paging regressions; read complete outputs.

## Task 2: Platform shortcuts and actual-control acceptance

**Files:** history_navigation.go/test.go, shortcuts.go and necessary i18n.go entries; docs/shortcuts.md.

- [x] RED: platform metadata and actual-control Back/Forward; unavailable directions, busy/loading/pending dialogs/chooser/ref and retained callbacks remain inert. Runtime key dispatch tested on macOS; Windows/Linux compile only.
- [x] Actual Summary/Body/history-search inputs retain ownership without navigating; replay invalidates stale frame rows and subsequent list movement uses the new selected commit. Precise caret movement and new replay→Enter/Space chains are not separately tested; existing row handlers remain in place.
- [x] Self-owned local Git fork/merge navigation preserves HEAD, symbolic HEAD, refs, Index, status, working bytes and draft/amend.
- [x] Implement shared-table bindings and bilingual labels with existing input policy.
- [x] Run all related regressions; inspect minimum-width bilingual/theme Settings copy. Four 560×480 captures with font scale 1.15; no new layout.

## Task 3: Final review and acceptance

- [x] One independent bounded implementation review; reproduce confirmed material defects before fixing. Testing review ae9e72eeaa00190fe found two issues, both reproduced and fixed with regressions. No second broad review.
- [x] Windows/Linux amd64 compilation and bounded owned macOS native attempt. Natural-focus prerequisite blocked before input; keep this distinct from successful deterministic tests and compilation.
- [x] Freeze final source fingerprints; run full make check, read complete ordinary/race/vet output, verify consumed sources and git diff --check. Update shortcut/e2e/alignment evidence. No commit/install/release.

## Progress

Three-role investigation and a fresh focused plan review are complete. New review roles: product a8bcd58cc9b9864b0, design aa6d94f41271aa99c (Emil/Impeccable used), testing adea66e6c4c20c8ca, each in an independent worktree reading the actual integrated source. Main verified the parent-button call site and incorporated the reported plan gaps: all parent buttons share recording, any query edit cancels pending navigation, and same-SHA HEAD synchronizes the current anchor query without adding an entry. Settings labels should say “沿跳转记录后退／前进” / “Back/Forward through commit jumps”. Invalidated records are silent; actual read failure retains existing feedback without promising that a reload preserves the old trail.

Child-navigation full check b1drvyh4n passed and source fingerprints matched before implementation. A further explicit user request created product a684fc9c499cebf3f, design a6f9056beb6c57354 (Emil/Impeccable applied) and testing ae9e72eeaa00190fe in independent worktrees. Main implemented in the integrated workspace; all role reviews were read-only. Product found no new scope blocker and requested precise final documentation. Design found no interaction/naming blocker and requested minimum-width English label evidence, now captured.

Implemented history_navigation.go plus focused tests. Existing relationship shortcuts and every parent SHA button use the opt-in wrapper. jumpToCommit final selection settles the explicit transaction; ordinary selectCommit calls only invalidate, never record. Ordinary page completions use resumePendingCommitJump. SearchBar edits cancel pending transactions immediately; scope reset clears them. Shortcut descriptors inherit the parent typing policy and choose the reference platform modifiers; bilingual Settings names remain explicit about jump records.

Evidence:
- Missing-API RED: build/alignment-history-navigation-red.log. Initial relevant navigation GREEN 1.765 s, build/alignment-history-navigation-initial-green.log.
- Shortcut RED: build/alignment-history-navigation-shortcut-red.log, actual Back dispatch initially missing.
- Independent testing review found edit→clear search could revive pending navigation, and a late page could navigate behind the newly opened command palette. Both reproduced in build/alignment-history-navigation-review-red.log before fixes. Added immediate SearchBar cancellation and resumed/final interaction rechecks. Corresponding regression tests passed in the related GREEN run.
- Final related GREEN 3.375 s, build/alignment-history-navigation-final-related.log, complete output read. Includes actual Summary/Body/history search ownership, actual parent button, old frame rows, retained callbacks, real 420-commit page success/cancellation, and real merge/fork route with Git/draft invariance.
- Four Settings renders at 560×480 and font scale 1.15: build/history-navigation-captures/. Coordinator inspected settings-en-light.png and settings-zh-dark.png; both new full action names are visible. Boundary checks cover both locales/themes. No new layout/animation/tooltip.
- Windows/Linux amd64 CGO-disabled builds and ordinary macOS e2e build passed; compile evidence is not target-OS runtime acceptance.
- Native attempt build/history-navigation-native-bje6hqjh/ failed its five-second natural focus prerequisite before any input, owned process stopped exit 0. Full build/alignment-history-navigation-native.log read. Driver build/.native-focus/history-navigation.py; no permission bypass or blind retry.
- Final full make check bt4g4lvck completed exit 0. Complete build/alignment-history-navigation-full-check.log read: ordinary lobedesk 104.375 s, race lobedesk 370.253 s, vet clean. All 445 fingerprinted inputs match build/alignment-history-navigation-final-sources.json; build/alignment-history-navigation-source-verification.json has changed=[]. git diff --check passed. Source fingerprint scope is Git-listed main workspace Go/module files and Makefile, excluding ignored agent worktrees. Native focus blockage remains open and separate from this full pass.

The roles' recommendations are advisory; no role report authorizes commit/install/release. No changes to real user repositories, installed app or permissions.
