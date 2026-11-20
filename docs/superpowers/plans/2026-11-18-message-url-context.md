# Message URL Context Actions

**Status:** Implemented; reviewed parsing and long-menu issues repaired. Full make check (ordinary/race/vet) accepted the final wrapped-menu snapshot with 515 unchanged inputs and eight matching packages. No commit, installation or publication.

**Goal:** Open an explicitly chosen HTTP(S) address from a historical commit subject or body while preserving existing plain-text reading and selection.

## Reference and decisions

Installed Sublime Merge Build 2125 `Default.sublime-package/Commit Message.sublime-menu` declares `open_context_url`, without arguments or caption. It does not establish pointer/selection targeting, protocol support, punctuation rules or native opening behavior. This change implements an explicit context-menu adaptation, not a claim of identical reference interaction.

Three independent worktree roles investigated product scope, design using Emil Design Engineering and Impeccable, and SDK opener/testing APIs. Direct inline links are unsuitable with the pinned SDK: a scratch headless test dragging from inside `ui.Link` produced an URL-open request instead of safe selection. A separate context-menu prototype preserved complete text drag-copy and opened exactly the explicitly selected URL. Both used SDK headless state, never a real browser. Product's separate links-list proposal was not added.

## Implementation

- Preserve `display.Text` and the existing Selectable scope. Attach URL context menus to the subject and body independently; no extra link list, settings or primary-click behavior.
- Recognize explicit HTTP(S) URLs with a parsed nonempty hostname. Exclude other schemes, malformed URLs and scheme text embedded in a word; deduplicate exact strings in source order.
- Preserve bare URL bytes, including trailing path/query/fragment punctuation and ASCII/Unicode apostrophes. Only remove an explicit enclosing quote/bracket immediately preceding the URL and its corresponding closing wrapper (including punctuation after that wrapper). Ambiguous prose punctuation remains part of a bare URL rather than silently changing a potentially legal target. These are bounded application rules, not established Sublime parsing parity.
- Menu labels are the actual URLs, with external-link icons. Each row opens only its own captured target, retaining the list while open. A refreshed message cannot retarget an existing menu; reopening captures the current message.
- Use `c.OpenURLThen` rather than shell commands or file-path openers. Failure queues a localized error and redraw; success means opener acceptance only and does not claim page loading. No Git work, selection mutation, draft changes or clipboard overwrite.

## Verification

The parser API was initially missing; after adding it, actual right-click tests failed because the message menu was absent. After integration, both languages passed actual right-click selection, opening the second URL, refresh while open, injected opener failure and retry against the refreshed message. Drag-copy preserves the whole original paragraph and emits no opening request.

A new punctuation regression first failed because query/fragment trailing punctuation was stripped; parsing was corrected to preserve that data. URL fixtures include balanced path parentheses, deduplication, Unicode spacing/paths, encoded bytes, query/fragment strings, invalid host/port, embedded schemes and unsupported protocols. A long URL in the subject is tested within a 480×360 menu viewport.

Independent review identified legal path punctuation and apostrophe truncation; the second pass identified punctuation outside enclosing quotes. Main reproduced each issue as failing literal URL assertions and fixed it. The original Unicode prose-period fixture now expects the full bare URL including that period, consistent with the explicit decision to preserve ambiguous legal target data; no assertion was removed. Additional regressions retain ASCII and Unicode apostrophes, bare punctuation, quoted URLs followed by prose punctuation, uppercase schemes and percent-encoded bytes.

Final combined message, long-subject, author and date focused race passed 5.750 s. Main generated and read `build/message-link-long-menu.png`: long menu rows stay within a 480×360 viewport and elide visually; the complete address remains readable/selectable in the unchanged original message. This is headless rendering, not native browser evidence.

The first whole ordinary suite failed as `blphdm6bk`, prefix `build/alignment-message-url-context`: `TestEveryCopyOfAnActionUsesOneVerb` reported `openMessageLinkFailed` missing its action group. Main read complete log, source verification and task output: 514 inputs unchanged, eight packages matched, diff clean, check exit 1; lobedesk failed after 144.286 s, so vet did not run. Repaired by registering the open-link action group and using the expected English verb (`Open link failed: %s`); the focused action-copy check passed. This is a repaired failure, not an accepted whole-suite run.

## Wrapped-menu follow-up

Fresh design and test roles identified that two long URLs with a common prefix could look identical after SingleLine elision. Main captured the concrete two-address case and confirmed it. The message menu now uses `ContextMenuBase` with wrapped, complete URL labels and a viewport-bounded scroll region; it retains full accessibility labels and the existing menu lifecycle. `build/message-link-two-long-urls.png` now shows both complete targets and their distinct `q=one#part` / `q=two#part` endings. Main and the independent design reviewer read the updated image; the second static review found no concrete additional issue.

The strengthened test checks both long rows are multiline, separate and inside the viewport, then chooses the second exact target. Another test scrolls eight long items to the final URL and opens that exact target. An isolated mutation restoring SingleLine failed the long-row regression as expected. A negative UI test right-clicks non-HTTP content and verifies no URL open action/request.

`commit_message_selection_menu_test.go` preserves a partial selection when right-clicking inside that selection, after Escape and after opening a URL. Initial fixture right-clicked outside the prefix selection and lost it; SDK `textSelectionMenu` deliberately moves the caret outside an existing selection. Corrected the fixture to test the intended inside-selection contract; did not change production selection semantics. Final combined summary/menu/selection/action-copy race passed 6.582 s.

Full make check completed as `b18cf980a`, prefix `build/alignment-message-url-wrapped`. Main read the complete log, verification JSON and task output: 515 inputs unchanged, eight packages matched, check exit 0 and diff exit 0. Ordinary lobedesk tests passed in 143.939 s; full race gitrepo/lobedesk passed in 90.296 s / 477.297 s; vet passed. This accepts the wrapped-menu implementation at that snapshot. Native browser integration and exact reference semantics remain unverified.

The product role identified message SHA context navigation as a subsequent candidate. `navigate_to_context_commit` has no declared format/target parameters in installed resources; existing jumpToCommit is not itself the missing capability. No SHA parser, new shortcut or navigation entry was added in this follow-up.
