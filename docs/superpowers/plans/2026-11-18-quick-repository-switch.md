# Quick Repository Switch Implementation Plan

> **For agentic workers:** Use the already-loaded executing-plans/TDD/verification guidance. Main implements, roles review scoped work in independent worktrees. No commit/install/release without explicit user authorization.

**Goal:** Add a searchable keyboard switcher for other repositories successfully opened in this application session.

**Architecture:** Reuse recentRepositories and openRecentRepository(root string), with one root/load/generation-scoped DialogBase and shared picker helpers. Opening/searching/moving/cancelling never opens a repository. A confirmed candidate closes its own modal before rechecking and calling the existing asynchronous reset=true opening flow.

**Tech Stack:** Go, MyGo/Lobe existing picker/search/row/focus helpers, shared shortcut table, actual UI Tester and two self-owned local Git fixtures.

**Spec:** User-authorized Sublime Merge alignment. Product abbf4384d15f5d49f, design ad78f26bab4de4f6b (Emil/Impeccable) and testing af0bd9e6d1e31f3e8 reports. Installed resources declare quick_switch_repository: macOS Cmd+Shift+O and Cmd+Ctrl+P, Windows Ctrl+Shift+O, Linux Ctrl+Shift+O and Ctrl+Alt+P. Main menu and command catalog contain Switch Repository…; these facts do not establish reference candidate sources, ordering, matching, drafts or focus semantics.

## Global Constraints

- History-navigation full check bt4g4lvck passed before this increment: ordinary lobedesk 104.375 s, race 370.253 s, vet clean, complete output read, all 445 frozen inputs match and diff formatting clean. Preserve all unrelated dirty work and generated_images/.
- No real-project Git writes, real remotes, OS permission bypass, persistence/settings changes, commit, installation or publication.
- Source is the existing application-session MRU: at most eight roots including current; exclude current root, leaving at most seven alternatives. Preserve MRU order, no new candidate source or storage. Filtering uses case-insensitive full-path substring matching; basename matches naturally.
- Current root is excluded rather than shown disabled. Missing paths remain selectable so existing open/error handling is authoritative; never prune or reorder MRU merely because a search, cancel or failure occurs.
- Primary row shows repository name, secondary path distinguishes equal names; accessible label includes full path. Highlighted long path may wrap inline. No repeated tooltip, extra toolbar, animation or new setting.
- First increment is dedicated shortcuts only. Existing title menu stays usable; do not add menu→picker or generic-palette→picker handoff in this increment. Keep generic palette count ten.
- Shortcut metadata provides Settings entries for the resource-backed platform keys. Ordinary input fields can invoke the switcher; picker search owns arrows/Enter. Busy/loading/history reload, native chooser, ref picker or another main modal blocks opening and execution.
- Opening scope captures current root/loadSeq plus monotonically increasing picker generation. Root change, same-root reload or reopen invalidates old callbacks. Confirm only an exact candidate still in current MRU and different from current root; close picker, then recheck remaining guards before openRecentRepository.
- One DialogBase with 16px gutters, existing theme/rows/scroll limits. Capture original opener before one-shot search focus. Tab can leave search. One Esc closes even with text; backdrop closes; both restore original surviving input and leave its text/selection/draft alone. Reopen with empty query.
- Distinguish no other session roots from no matching search. Empty Enter is inert. Search/cursor changes do not switch repositories.
- Open/status failure preserves current root, draft/amend and MRU, but existing open initialization resets traversal/navigation/scroll. Do not promise full UI restoration. Failure feedback identifies the attempted target and that the current repository remains.
- Accepted reset=true switch follows existing draft/amend/detail clearing and MRU promotion, without draft recovery or a new confirmation. A target's pending Git operation may prefill its existing operation message. Switching back does not restore the old draft.

## Review Focus

- New picker must not block its own confirmation through hasPendingDialog; close before existing open entry.
- Same basename at two roots, Unicode/spaces and long paths must remain distinguishable at narrow widths.
- Missing directory, removed .git or status failure must preserve the current draft/root/MRU with accurate failure scope.
- Old candidate callbacks must not reopen after root/load/generation changes or act on a reopened picker.
- Actual Summary/Body/history-search cancellation must restore the original editor and typing; no per-frame focus or first-Esc tooltip interception.

## Main Decisions

Ruling: exclude current root — matches existing title-menu semantics and leaves only actionable choices. Cost: the picker is not a list of all eight MRU entries; wording must say other recent repositories.

Ruling: dedicated shortcut entry first — provides the resource-backed quick-switch task without another modal handoff lifecycle. Cost: Switch Repository… will not yet be in the generic command catalog.

Ruling: reuse reset/error semantics instead of introducing draft persistence or rollback — keeps this increment limited to convenient access to an existing operation. Cost: success clears old ordinary drafts, and failed opening can reset browsing positions; document and test the exact boundary.

## Task 1: Candidate picker, lifecycle and shortcut

**Files:** create internal/lobedesk/quick_repository_switch.go and quick_repository_switch_test.go; modify app.go state, dialog_scope.go modal ownership, view.go rendering, shortcuts.go metadata and required bilingual i18n.go.

**Interfaces:** Consume recentRepositories, paletteRoot(), loadSeq, openRecentRepository(root string), existing picker helpers and DialogBase. Produce a private picker state and candidate/open/choose/stale/dialog methods; finalize exact names from neighboring quick_checkout implementation.

- [x] Write RED for session-only MRU order, excluding current, full-path search/equal names, no roots/no matches and inert Enter; shared key dispatch and guards.
- [ ] Write RED using actual Summary/Body/history search: typing in picker, arrows/Enter, one Esc/backdrop/opener continuation, Tab and reopen; stale root/epoch/generation and retained background callbacks.
- [x] Implement one existing-style scoped modal, bilingual necessary copy and table bindings, close-before-recheck execution. Do not change generic command catalog.
- [x] Run focused picker/recent-repository/palette/modal regressions and inspect rendered en/zh, light/dark, narrow/wide identity/empty/long-path states.

## Task 2: Owned switching, review and final acceptance

- [ ] Use two disposable local Git fixtures with distinct same-name paths and staged/unstaged/untracked content. Confirm real A→B→A switching, success reset/MRU behavior, missing/open/status failures, stale open completion and failure target feedback. Compare each fixture's HEAD/symbolic HEAD/refs/Index/status/file bytes and original draft/amend where applicable.
- [x] Independent bounded implementation review, reproduce material findings before repair; maximum three rounds, subsequent rounds only changed source.
- [x] Windows/Linux compilation and one bounded macOS native attempt with natural-focus prerequisite; preserve any native block separately from deterministic success.
- [ ] Freeze source fingerprints, run full make check, read complete output, verify final consumed inputs and diff formatting, update shortcut/e2e/alignment evidence. No commit/install/release.

## Progress

Implemented the scoped session-MRU picker, platform shortcuts, main-dialog ownership, retained-callback isolation and target-specific failure context in the existing open flow. Initial missing-API RED was observed; initial focused GREEN passed 2.099 s. Related picker/recent/palette/child/browsing regressions passed 20.032 s. Expanded focused tests passed 3.296 s: actual Summary/Body/history-search selected text survives cancellation, platform aliases and reopen, dirty A→B→A with staged/unstaged/untracked bytes unchanged, missing target/removed .git/Status error, and superseded failure callbacks. The corrupt-index fixture explicitly proves Open succeeds before Status fails.

Implementation review roles: product a7253cdc8994ccda7, design ac55955c5ae2f7134 (Emil/Impeccable), testing a5c0c46620ec10703, all isolated-worktree/read-only. No confirmed production defect; testing identified coverage gaps, then main added the selected-text and failure-boundary cases. Design actually viewed all 12 narrow captures and four wide paths captures from the 24-render matrix: same-name Unicode/spaced long paths and both empty states are readable in en/zh and light/dark. Windows/Linux compilation and macOS e2e build passed. Full make check bnvg8upkt passed exit 0: ordinary lobedesk 106.514 s, race 372.188 s and vet clean. Complete output was read, all 447 frozen inputs match, and git diff --check passed before the next increment's source edits. Evidence: build/alignment-repository-switch-full-check.log and build/alignment-repository-switch-source-verification.json.

Remaining limits: real fixtures use distinct temporary root names; same-basename identity is verified separately by the actual rendered picker. Tab followed by Escape is exercised, but departure of input focus after Tab is not independently asserted. Target-operation message prefill follows existing code and is not newly exercised. Reference GUI candidate/sort/focus/draft semantics remain unobserved. Role recommendations do not authorize commit or release.
