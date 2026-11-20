# Second Parent Navigation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Alt+Shift+Down to select and reveal the second parent of a visible historical selection, only while the history commit list has keyboard focus.

**Architecture:** Extend the shared shortcut table with an explicit control-local binding, registered on the existing focusable commit-list container. Reuse first-parent target validation through a parent-index helper, and reuse jumpToCommit without changes to pagination, cancellation or native focus.

**Tech Stack:** Go, MyGo/Lobe, existing history widgets and Git read APIs.

**Spec:** Scope below, derived from current user-authorized alignment and product/design/testing reports. Build2125 Default - Merge resources OSX:80, Windows:85, Linux:87 map Alt+Shift+Down to navigate_to_parent {parent:1}, context commit_table. Resources do not prove runtime focus or filter semantics.

## Global Constraints

- Preserve uncommitted work and generated_images. No commit, install, release or external publication.
- Source of truth is /Users/adam/SuperGit; role worktrees omit current integrated edits.
- No child navigation, stack, setting, layout or added focus restoration. No changes to existing first-parent scope.
- A visible selected historical commit with at least two parents supplies Parents[1] full SHA. Missing second parent/root/worktree/hidden selection no-op.
- Apply first-parent busy/loading/historyReloading/chooser/ref-popup/main-dialog guards to request and Run. Preserve typing in actual search and commit fields.
- Second-parent key is local to the commit list, including focused rows; details/file list/tab focus cannot invoke it. Keep key metadata in shortcutTable so Settings lists the implemented binding.
- Use existing selection/reveal and pending lifecycle, including retry feedback and cancellation after new selection. Existing parent buttons retain all explicit parents.
- Main waits for first-parent full check before source changes. Owned synthetic fixtures only for native/Git assertions.

## Review Focus

- Original row retains focus after navigation: consecutive parent keys/arrows must use current selected SHA; Enter/Space must not reselect original row.
- Focus in historical details and file list must not trigger the local binding.
- Three or more parents must choose exactly index 1, never final/adjacent/first parent.
- New local metadata must not accidentally register a window key; Settings still lists the same binding.
- Guard transitions and lost target during paging must retain existing cancellation/retry behavior.

### Task 1: Local second-parent entry

**Files:** Modify `internal/lobedesk/{parent_navigation.go,shortcuts.go,history.go,i18n.go}`; create `internal/lobedesk/second_parent_navigation_test.go`; update `docs/shortcuts.md` and this ledger.

**Interfaces:** Consume existing `parentNavigationTarget() string`, `navigateToParent()`, `jumpToCommit(string)` and `appShortcut`. Introduce indexed `parentNavigationTargetAt(index int) string` used by first-parent wrapper and `navigateToSecondParent()`. Add `Local bool` to appShortcut and skip it in window registration; the commit list binds the table's local second-parent entry and calls its Run. Keep Field semantics unchanged.

- [x] RED: real historyPage plus shared window dispatcher; focus commit row and assert exact second full SHA and reveal. Cover three-parent source with nonadjacent target, single-parent/root/worktree/hidden/guards, details/search outside scope, and continuation keys. Run `go test ./internal/lobedesk -run '^TestSecondParent' -count=1`; expect missing binding/navigation assertions.
- [x] Implement indexed target helper, local table metadata/binding and localized action name, without widening the focus range. Match parent:1, Alt|Shift + Down, and execution-time guards.
- [x] GREEN: focused tests plus first-parent, pending-jump, table/card and detail-focus regressions. Actual container/row focus must participate; a direct Run test alone does not prove scope.
- [x] One scoped independent review; fix verified blockers with regression evidence.
- [x] Relevant race, Windows/Linux compile and owned macOS native merge navigation with ready details/Git fingerprints; inspect capture.
- [x] Full make check after production source stabilizes; read complete output and diff formatting; update docs truthfully.
- [x] Deliver uncommitted result, limitations and product rulings. No commit step authorized.

## Rulings and progress

Ruling: adopt list-local scope instead of product's broader non-input historical-page suggestion, because installed reference explicitly says commit_table and existing focusable list provides a local ownership boundary. If this interpretation is narrower than reference runtime, users can still use explicit parent buttons; do not represent native reference equivalence.

First-parent full check bg454u0tm passed exit 0 (ordinary lobedesk 92.741 s, race 363.178 s, vet clean); full log read before second-parent source changes.

- RED: `build/alignment-second-parent-red.log` reproduced missing binding with actual history-page/row focus and scope tests.
- Implemented `parentNavigationTargetAt(index)` and second-parent wrapper; first-parent remains index 0. Added appShortcut.Local metadata, excluded from window registration; the existing focusable commit-list container registers the shared second-parent key/action. Added localized action label.
- Focused GREEN: `build/alignment-second-parent-green.log`, 1.323 s. Actual row selects index 1 of three parents, target visible, Enter/Space retain selected parent and Up follows selection rather than original focused row. Real details tab, file row and search plus window-only rendering cannot invoke the key. Missing parents/worktree/hidden selection and retained action guards covered.
- Related race: `build/alignment-second-parent-race.log`, 18.921 s, exit 0. Independent testing-role implementation review found no specific blocker; static review, not independent rerun.
- Windows/Linux amd64 CGO-disabled builds passed, compile coverage only.
- Native ordinary e2e binary `build/supergit-lobe-second-parent-native`; driver `build/.native-focus/second-parent-navigation.py`; results `build/second-parent-native-5yodbll3`, log `build/alignment-second-parent-native.log`. Owned real two-branch merge fixture verifies exact second SHA, details focus blocks key, target details finish loading, Enter/Space retain selection, missing second parent no-op, draft/Git/working bytes unchanged. Screenshot inspected; process stopped 0. Natural window focus checked; no native refocus workaround. Scope is macOS English/light/1100x720; no target-OS native, native long paging or reference GUI acceptance.
- Full make check `b8m1xnnmk` passed exit 0: ordinary lobedesk 85.956 s, race 353.046 s, vet clean. Full `build/alignment-second-parent-check.log` read; production sources stable throughout. Diff formatting passed.
- No commit/install/release or durable product rule recorded.

## 2026-11-18 renewed role review and paging coverage

Three fresh independent-worktree product/design/testing roles reviewed the current implementation. They retained list-local scope, exact `Parents[1]`, no first-parent fallback, existing selection/reveal/route feedback and execution-time guards. Reports are static; they did not run the current app. Main's targeted frozen-source race wrapper `b7by0r0xo` passed in **5.372 s**, with **526** unchanged inputs, **8** matching packages and zero check/diff-check exits. This was a targeted run, not another whole `make check`.

The testing role identified a combined coverage gap: real merge relationship and actual keyboard invocation were tested separately; the second-parent key had no direct real-Git paging/cancel regression. Added `internal/lobedesk/second_parent_paging_test.go` with `TestSecondParentActualKeyPagesRealMerge`. Its owned repository has distinct first/second branches and an actual merge with Git-derived ordered parent SHAs. A separate tip has 240 descendants of the second parent, forcing them to precede that target topologically; the test asserts the merge is loaded and the second parent is not in the first 200 rows. An earlier scratch fixture assumed Git would delay a short side branch based on dates; that assumption failed before invocation and was replaced with the explicit topological requirement.

Actual row click plus Alt+Shift+Down must keep the source and avoid a route until paging completes, then reveal exactly the second parent and support Back to the merge / Forward to the parent. A second case changes selection before delivering the held page callback; the old jump must remain cancelled and create no route. Both preserve HEAD, symbolic ref, all refs, index entries, status, owned worktree bytes and draft/amend. The overlay experiment passed **1.536 s**. An isolated mutation changing only `navigateToSecondParent` to use parent index 0 failed both cases at the invocation assertion; no production implementation was modified for that experiment. After integration, second-parent, first-parent, pending-jump and history-navigation race tests passed **6.572 s**. A narrow independent testing-role review read only the added regression and found no blocker in real input, merge parent order, unloaded target, held callback cancellation, exact route or Git/draft preservation assertions; it did not independently rerun tests.

Final frozen-source whole ordinary/race/vet wrapper `bywb96pls` completed with prefix `build/alignment-second-parent-paging`. Main read the full log, verification JSON and actual task output: ordinary tests passed (`internal/lobedesk` **144.964 s**), race tests passed (`internal/lobedesk` **471.003 s**), and vet passed. All **527** inputs remained unchanged, all **8** packages matched, and check/diff-check exits were **0**. This accepts the added regression snapshot; subsequent historical file-selection changes require separate verification. New coverage is headless; the earlier recorded native test does not prove native long paging or Windows/Linux runtime. No commit/install/release.
