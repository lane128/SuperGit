# Child Commit Navigation Implementation Plan

**Goal:** Add Alt+Up child navigation within the current fixed-tip ordinary history scope, preserving draft, Git state and established input guards.

**Architecture:** Discover direct children by scanning Parents in the complete topological prefix a.commits, never the UI-filtered rows. A visible selected source must itself be in that prefix. One child jumps via jumpToCommit; multiple children use a root/load/reload/source/generation-scoped picker. No child in this range gives scoped feedback. File history and selections outside the proven prefix give recovery feedback rather than guessing.

**Tech Stack:** Go, MyGo/Lobe DialogBase and shared picker helpers, existing read-only history navigation and UI Tester/owned Git fixtures.

**Spec:** User-authorized Sublime Merge alignment; product a86836d639e9efb4d/a570e7daf36c82b34, design a24a7db33f1f652ca/ad3096d9031830e70, testing a8f8fa07f938f88f9/abfcf0bac39f0dbf7 reports. Installed three-platform resources map Alt+Up to navigate_to_child without args/context. Reference multi-child ordering, scope and GUI focus behavior remain unknown.

## Global Constraints

- Palette handoff full check bp9bney37 passed: ordinary lobedesk 101.216 s, race 363.225 s, vet clean; complete output read and all source hashes/diff formatting verified before child edits.
- Preserve all dirty/untracked work and generated_images. No commit/install/release/publication, real user repository Git write, real remote or permission bypass.
- Scope is current ordinary fixed tips. A loaded source in a.commits proves its in-scope direct children are already in the topo-order prefix, even if older pages remain, pause or tail-read error occurs. This does not establish all repository descendants.
- Search/UI reachability must not hide candidate relationships. Visible selected historical source is required; details fallback, workspace row or hidden selected source cannot initiate navigation.
- File --follow history does not carry this proof. Preserve current selection/filter and explain that ordinary history is required; no implicit range expansion or extra Git traversal.
- One candidate direct jump, multiple explicit choice ordered as existing history, zero feedback says current history range. No force/checkout/write action or new setting. Existing jump clears only text search to reveal the chosen commit while keeping ref range.
- Inherit first-parent shortcut's typing protection and modal/busy/loading/chooser/ref guards. Do not forward Alt+Up from real editors. Shared shortcut metadata supplies Settings label. Generic palette remains ten commands.
- Picker has one constrained DialogBase, 16px gutters, existing theme/rows/scroll helpers, one-shot focus after opener capture and no duplicate tooltips. Esc/backdrop preserves input/draft. Root/load epoch/reload/source/generation must invalidate old choice callbacks; revalidate direct relation before selecting.

## Review Focus

- Hidden child C and visible parent P under search: relationship comes from full prefix; jump reveals C.
- Merge second-parent edges, multiple tips, root with children and HEAD with in-scope children.
- Prefix versus file-history/fallback selection, remaining pages, paused/error tails and ref scope changes.
- Actual inputs retain editing behavior; multiple-choice search/arrows/Enter and final cancel restore focus without background key leakage.
- Stale picker callbacks after new selection/root/reload/epoch/reopen cannot navigate.

### Task 1: Scoped discovery and shortcut

**Files:** new internal/lobedesk/child_navigation.go and child_navigation_test.go; shortcut metadata.

**Interfaces:** Consume app.commits, historyRows(), selectedSHA, historyReload, loadSeq and jumpToCommit(sha). Produce childNavigationCandidates(source string) []gitrepo.Commit, canNavigateToChild() bool, navigateToChild(). Derive binding from first-parent metadata to preserve its established typing guard; change ID/key/label/availability/action only.

- [x] RED/GREEN: full-prefix discovery despite UI search, direct jump and query reveal, merge edge/zero/paused tail/file-history/hidden source/guards; actual input editing key does not invoke navigation.

### Task 2: Explicit multiple-child picker

**Files:** child_navigation.go or focused child_navigation_picker.go; app.go state, dialog_scope.go ownership, view.go rendering, necessary bilingual i18n.

**Interfaces:** Produce childNavigationState with open/root/epoch/reload/source/generation/query/cursor/focus fields; childNavigationDialog(c), chooseChildNavigation(sha,generation), discardStaleChildNavigation().

- [x] RED/GREEN: actual multiple choices/search/arrows/Enter, Esc/backdrop/draft/input preservation, stale source/root/epoch/reload/generation/edge, modal isolation and narrow bilingual layout. Owned Git navigation preserves refs/index/files and draft.

### Task 3: Acceptance

- [x] Independent scoped review, reproduce/fix material findings (maximum three rounds).
- [x] Related regressions and final Windows/Linux compilation; rendered evidence as needed, bounded owned native attempt with existing natural-focus prerequisite.
- [x] Full make check on stable final source; complete logs/hashes/diff validation and docs. Native blockage stays separate from deterministic/full-check success.

## Decisions

Ruling: current fixed-tip scope rather than all local refs — consistent with displayed history and avoids a separate expensive traversal. Cost: off-scope children are intentionally not offered; feedback states the scope, no claim of reference GUI equivalence.

Ruling: loaded source in the actual ordinary topo-order prefix is sufficient, not whole-history completion — every child of that source in the same traversal precedes it. Cost: relies on existing stream ordering; test hidden and second-parent children and reject file history/fallback.

Ruling: file-history navigation gives an ordinary-history recovery message rather than guessing or changing user scope — --follow can omit real edges. Cost: Alt+Up is unavailable there until the user returns to ordinary history.

## Progress

Implemented scoped child discovery, Alt+Up binding inherited from the first-parent input policy, and one explicit multiple-child picker. Reports are advisory; explicit scope choices above are main-agent decisions within authorized alignment work.

- Initial missing-API RED: build/alignment-child-navigation-red.log; initial focused GREEN predates the later isolation fix.
- Retained background callback regression reproduced “background page shortcut leaked through child picker” in build/alignment-child-navigation-isolation-red.log. Registration and dispatch now both guard the child picker. Related navigation/palette/checkout regressions passed, 18.582 s, build/alignment-child-navigation-final-related.log.
- Empty search regression reproduced the wrong command-specific copy; child search now uses the existing commit-specific searchNoMatch message. Actual filtering and Enter recovery pass.
- Independent read-only implementation review abfcf0bac39f0dbf7 found no confirmed new defect; it did not run tests. Its multi-tip paging, backdrop and focus-restoration coverage gaps were subsequently filled. Escape/backdrop restores the original real Body and typing appends; eight locale/theme/size renders verify gutters and arrow/Enter choice. Real pinned multi-tip pages include both children before the selected source while older pages remain, and exclude a ref added after traversal started.
- Scoped zero/file-history feedback regression initially inspected pointer formatting instead of toast description. Corrected the assertion to compare the actual notice description; no production change. Final focused result is build/alignment-child-navigation-final-focused.log.
- Eight captures: build/child-navigation-captures/. Coordinator viewed all eight captures; the subsequent design role also inspected all eight and found no confirmed important design defect. Windows/Linux amd64 CGO-disabled builds and ordinary macOS e2e build passed; these do not prove target-OS runtime behavior.
- Native attempt: build/child-navigation-native-ejqi6uh5/, driver build/.native-focus/child-navigation.py, log build/alignment-child-navigation-native.log. Natural focus prerequisite failed before any draft input or Alt+Up. Owned process stopped exit 0; no bypass/retry or native success claim.
- Final full make check b1drvyh4n completed exit 0. Complete log build/alignment-child-navigation-full-check.log read: ordinary lobedesk 101.369 s, race lobedesk 366.112 s, vet clean. All 3,244 fingerprinted Go/module files matched build/alignment-child-navigation-final-sources.json; result build/alignment-child-navigation-source-verification.json has changed=[]. git diff --check passed. Native focus blockage remains open and separate from this full pass. No commit/install/release.
