# Message Commit Navigation Scope

**Status:** Bounded message-SHA action implemented. The 521-input snapshot passed full ordinary/race/vet. Subsequent confirmed font-scaling and source-round-trip issues are now repaired with integrated RED→GREEN regressions. Latest combined focused race passed 38.939 s; a final gate including these corrections and selected-text copying is recorded in `2026-11-18-message-selection-copy.md`. The preceding wrapped-URL snapshot was accepted separately (`b18cf980a`: 515 unchanged inputs, eight packages, ordinary/race/vet and diff clean).

**Goal:** Prepare the smallest explicit historical-message commit-navigation action without weakening existing navigation guards or plain-text selection.

## Reference and role findings

Installed Sublime Merge `Commit Message.sublime-menu` declares `navigate_to_context_commit` without arguments. Complete/abbreviated SHA rules, right-click positioning, ref-scope changes and native failure handling remain unverified. Three new independent-worktree roles reviewed product scope, design using Emil Design Engineering and Impeccable, and existing UI navigation guard/test coverage. Their recommendations are implementation inputs, not human approval or proof of native parity.

Candidate first increment: only independent complete 40-character hexadecimal SHA tokens, case-normalized and deduplicated in source order; do not accept short SHAs, refs or arbitrary revision expressions. Do not offer an extra commit action for SHA text inside an HTTP(S) URL. Retain selectable message text and the existing wrapped target menu. Typed entries must distinguish application navigation (“Jump to Commit · full SHA”) from external URLs.

## Existing APIs and required guard work

- `Repository.ResolveRef(ctx, ref) (string, error)` delegates to `git rev-parse --verify --end-of-options ref^{commit}`. Validate token format first; this API itself accepts refs and revision expressions.
- `historyNavigationBlocked()` covers busy/loading/history reload, repository chooser, ref menu and main-window dialogs, but does not validate repository identity, page or source.
- `jumpToCommit(sha)` can clear search, leave file history and continue paging. Its entry only checks nonnil repo/nonempty SHA; it is not a complete guarded message-menu entry.
- `jumpWithHistory(sha)` records explicit jumps and existing pending cancellation, primarily within the same full-history scope. `showCommit(sha)` can widen a ref filter to all refs; do not mix these two semantics without explicit implementation/test decisions.
- Parent shortcuts re-read the selected commit's parents. They must not be called for an independently captured message SHA.

A future typed menu action needs its own snapshot: repository pointer/root, `loadSeq`, `historyReload`, source SHA, target SHA and menu generation. Revalidate before read-only resolution and again before applying its asynchronous result. A root string alone does not detect reopening the same path. Current source must still be a visible historical selection, and busy/loading/dialog state must not silently redirect or clear a newer query. A refreshed/opened menu must not allow an older captured callback to affect it.

Missing object and “not found in the current history traversal” are different outcomes. A valid commit object can be dangling or outside the pinned traversal. Preserve this distinction; do not label traversal EOF as “belongs to another repository.” Multiple candidates need explicit separate choices and full targets in feedback.

## Isolated backend evidence

`build/_keyboard/message_commit_resolution_test.go`, overlaid as a virtual gitrepo test, exercised `ResolveRef` in owned disposable repositories. It resolves an existing full commit SHA; rejects a blob, a full SHA present only in another fixture repository, an all-zero missing object and a pre-cancelled request. The original repository deliberately contains staged, unstaged and untracked changes. It verifies exact HEAD, raw status, index bytes and worktree/untracked file contents remain unchanged. Focused race passed 1.603 s. This verifies the existing backend building block only, not a message UI action.

## Required UI acceptance before implementation is called complete

1. Actual message right-click and target choice: exact captured SHA, loaded and paged targets, explicit navigation history behavior, no changes to drafts or Git state.
2. Open-menu invalidation: repository switch/reopen, source/visibility change, generation/reload change and menu replacement reject old actions before any query, selection or navigation-record side effect.
3. Busy/loading/dialog changes between opening, selection and read completion reject or cancel safely; traversal pause/error/end produce accurate feedback.

## Subsequent scope decision and isolated recognition

Main adopts keeping the current ref traversal for the first message action, following the product role's scope analysis. `showCommit` can widen all refs merely because the target is not yet loaded, even if it belongs to the current ref; it is not a “search current scope first” wrapper. Explicit message jumps should use the existing navigation transaction within the current scope, preserving its forward/back behavior. Normal traversal EOF after successful object resolution must report that the object exists but was not located in this traversal, with the complete SHA; pause/read failure must not be called EOF or object absence. No new automatic ref-switch setting is proposed.

`build/_keyboard/message_sha_recognition_test.go` is an isolated prototype, not production parsing. It verifies complete 40-character tokens, canonical case/deduplication, source order, rejection of longer hexadecimal/identifier substrings, Unicode identifier adjacency, short hashes and revision suffixes, and omission of SHA text inside HTTP(S) URLs and other URI-like tokens. Focused prototype race passed 1.507 s. The app still has no message-SHA action.

A fresh design review required keyboard-active targets to remain visible in long menus. `build/_keyboard/message_menu_keyboard_test.go` tests the current production URL menu through an overlay: focus message text, Shift+F10 opens, arrows move to the eighth long address and it is focused within the viewport, Enter opens that exact address, Escape closes a reopened menu without another request. Focused race passed 3.365 s. This validates the existing menu building block, not future typed-SHA behavior or native OS events.

The testing role detailed two actual UI acceptance windows: mutate source/repository/modal state after obtaining the menu item but before dispatching its click, and delay the real repository-resolution main-thread completion while changing ownership. Reuse `browsingApp`, `quickCheckoutFixture`, callback channels and `historyCompletion`; use a full 40-character independently chosen target, not the source or its parent. The role's sample shorthand reference must not expand the complete-SHA scope.

At the end of investigation, prototype Go files remained under ignored `build/_keyboard`; frozen production inputs for `b18cf980a` had not changed. Those prototypes alone did not establish implementation acceptance. Subsequent production work is recorded below.

## Implemented increment and decisions

- `commit_message_commits.go` recognizes standalone complete 40-hex SHA-1 values, canonicalizes case and deduplicates in source order. Short/longer hashes, identifier substrings, Unicode adjacency, URI-embedded values and revision suffixes/ranges are excluded. SHA-256 and abbreviated hashes remain outside this increment.
- `commit_message_links.go` retains plain selectable subject/body text and uses one frozen typed snapshot. Existing URL choices appear first, followed by SHA choices in their respective source order. SHA rows display localized “Jump to Commit · full SHA” with a commit icon; URL rows retain complete addresses and an external-link icon. Rows wrap and scroll inside the viewport. No new shortcut or setting.
- `messageCommitOrigin` captures repository pointer/root, load epoch, history reload, source SHA/query, ref/pinned scope and menu generation. Both actual menu dispatch and asynchronous read completion validate ownership, visible ordinary-history selection and blockers. A newer request cannot be cleared by an older completion. A successful read queued before context cancellation is rejected when consumed.
- Resolution uses the existing read-only `ResolveRef`; it never checks out or rewrites repository data. Successful resolution enters `jumpWithHistoryTarget` in the current ref traversal and carries the message origin through paging. Scope/source/menu changes cancel the pending message transaction. Existing relationship navigation keeps its prior entry semantics.
- Full-target feedback distinguishes an unresolvable commit object, an existing object outside the completed traversal, and a paused/failed history read. An unsuccessful owned message jump restores the source search; stale callbacks do not restore or clear a newer search. Actual Cancel during paging restores the source search and reports the complete target before resetting traversal.

## Regression and review evidence

The first actual menu tests failed because the SHA action was absent, then passed in both languages. Loaded targets differ from source and parent and are real ancestors in the main fixture; the outside-traversal case uses a separate dangling commit. The loaded case verifies exact HEAD, raw status, index bytes, staged/worktree/untracked data and draft retention, and Back restores source SHA/query.

Ownership tests cover stale open menus and delayed real ResolveRef completions for repository replacement/reopen, changed root, epoch/reload, source/visibility/query/page/file/worktree changes, busy/loading/reloading, chooser/ref menu/dialog and menu replacement. They assert no request, query/selection change, pending navigation, route, browser request or stale notice. Replacement-menu acceptance is tested against an outstanding asynchronous request; a newly opened menu is itself a new user action.

Real 420-commit fixtures exercise an unloaded non-parent target, paging to its exact row and Back. A genuine read failure is obtained by temporarily renaming only the disposable fixture's `.git` after object resolution and before history reading; metadata is restored. Tests reject stale read feedback after page/source/query/dialog/menu/ref/pinned-scope changes. Parser regressions first exposed `SHA:file` and `mailto:SHA` false positives; exclusions were tightened.

One fresh independent worktree reviewer found two concrete defects: revision ranges (`A..B`, `A...B`) offered the first SHA, and Cancel during active paging discarded the source search/complete-target feedback. Main reproduced both as RED tests and repaired them. A further cancellation test exposed a queued successful ResolveRef result being applied after its context was cancelled; main reproduced and repaired it in the same fix pass. No unresolved review findings remain from that review; final whole-suite acceptance is still required.

`TestMessageCommitMenuKeyboardScrollAndCancel` uses Shift+F10, arrows, Escape and Enter on a mixed menu with eight long URLs and two unique SHAs, checks deduplication and keeps the focused last target inside a 480×360 viewport. Final combined message, URL/selection, history navigation and action-copy focused race passed 25.754 s. Main generated and read `build/message-target-en-light.png` and `build/message-target-zh-dark.png` at 480×360 and max 1.15 font scale; complete URL and SHA labels are distinguishable and within the viewport. Captures use an ignored overlay, not a production test package or native OS evidence.

## Whole-check gate

`python3 -I build/_keyboard/check_workspace_multiselect.py build/alignment-message-commit-navigation` completed as `b6zyfdhpl` with default `make check`. Main read complete log, fingerprint JSON and task output: 521 unchanged inputs, eight matching packages, check/diff exits 0; ordinary lobedesk 143.723 s and full race lobedesk 548.466 s, other packages cached or no tests, vet passed. An attempted stop found it already completed; it was not interrupted. This accepts its test snapshot but does not close subsequently discovered defects. The earlier 515-input gate does not accept these new SHA changes. Native reference semantics, system events and Windows/Linux native runtime remain unverified. No commit, installation or publication.

## Subsequent three-role audit and corrections

A fresh product/design/testing batch reviewed current absolute main-checkout files from independent worktrees while `b6zyfdhpl` remained frozen. Design used emil-design-eng and impeccable. They did not edit production or run a duplicate whole suite.

- **Font scaling (P2):** message rows and icons still use fixed 14 rather than `scaledFont(14)`. The prior max-font captures enlarged the application text, not this menu's text. Main's ignored overlay `TestMessageTargetMenuFollowsReaderFontScale` reproduced equal wrapped row heights at standard and large presets (`90.66376` each). Pending fix: apply the existing reader-scale convention to menu text/icons, retain a behavior regression and refresh captures/viewport acceptance.
- **Source-selection round trip (P2):** after ResolveRef has completed but before its main callback runs, explicitly selecting B then A makes the old origin snapshot equal again; the stale result navigates to T and creates a two-position route. Main reproduced this using actual tester selection-button events and the real `selectCommit`/`selectWorktree` entry points, for both another-commit and worktree round trips. `TestMessageCommitSelectionRoundTripCancelsResolution` is an ignored RED overlay. Pending fix: make explicit selection invalidate outstanding message requests and menu ownership, rather than checking current source values alone. The reviewer scratch file was no longer present after its read-only worktree cleanup; main's reproduction is retained under `build/_keyboard`.

The product role recommends a subsequent small alignment increment: add Copy Selected Text to the message target menu, with no selection disabled. Installed reference resources declare `copy` beside navigation/open actions, while existing app keyboard copy and full-message/subject copy already exist. This is a missing menu entry, not a missing clipboard capability or established native selection parity. Main inspected SDK `SelectedTexts`: it must be called on the selectable container, returns rune-indexed paragraph ranges, and matches ordinary copy with newline joins. An isolated `TestPrototypeMessageSelectionSnapshot` verified partial Unicode selection, disabled empty selection and capture retained across refresh under focused race (1.531 s). No production copy entry has been added; clipboard failures and full app integration remain to be tested.

Both confirmed corrections were implemented after the 521-input gate. `selectCommit` and `selectWorktree` invalidate request/menu ownership before early returns, retaining the existing navigation-state protection for successful jumps and Back/Forward. The integrated selection test initially assumed every paging fixture source had a parent and panicked; it now selects an actual independent loaded row instead. All resolved/paging source round trips, same-source reselect and old-origin rejection pass. Menu text/icons use `scaledFont(14)`; the four-combination layout regression and max-scale mixed-menu keyboard regression pass. New selected-text copy code, its independent review and the final combined gate are recorded in `2026-11-18-message-selection-copy.md`. Historical pending-fix bullets above describe the discovery state; they are not current unresolved findings.
