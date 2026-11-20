# Message Selection Copy

**Status:** Implemented; focused race, independent review and final whole ordinary/race/vet gate complete. No commit, installation or publication.

**Goal:** Keep selected-text copying available directly in a historical message's URL/SHA context menu, while retaining normal selection, target capture and plain-message editing menus.

## Reference and scope

Installed Sublime Merge `Commit Message.sublime-menu` declares `copy` beside `navigate_to_context_commit` and `open_context_url`. This establishes the menu capability, not native selection, enablement or targeting parity. Product, design and testing roles worked from independent worktrees. The product recommendation informed the implementation; it was not human approval or native evidence.

Existing app keyboard copying, subject copying and complete-message copying already worked. This adds the missing selected-text menu entry. It introduces no shortcut, button, setting, editable message or Git operation.

## Decisions and implementation

- `commit_info.go` retains its selectable summary container and passes that container to both subject and body message menus.
- `commit_message_copy.go` reads the SDK container's `SelectedTexts`, slices using rune offsets and joins selected paragraphs with one newline, matching ordinary text copy. It preserves Unicode, whitespace and selected paragraph boundaries.
- `commit_message_links.go` prepends a localized Copy Selected Text item and separator to an existing URL/SHA target menu. It captures the text with the target snapshot while the menu is closed and keeps it frozen while open. Source refresh does not replace the text the reader chose.
- Empty selection disables the item and cannot dispatch an empty/whole-message fallback. Right-click inside a selection preserves it; outside a selection follows the SDK's existing caret-collapse behavior. Escape closes without copying or a notice.
- Copy uses existing `copyText`, including its localized success/error feedback. It cannot navigate, open a URL, write Git or change a draft.
- **Ruling:** messages with no URL/SHA targets keep the existing SDK Copy / Select All menu. Replacing that menu with a copy-only menu would remove an existing operation. This narrows the product role's optional suggestion that the new custom copy entry could also appear without targets.

## Verification

The partial-selection tests first failed because the menu entry was absent, in both languages and clipboard success/failure cases. Empty/outside selection, Escape and subject-to-body selection tests also failed on the missing entry before implementation.

After implementation, actual headless events verify a partial Unicode selection matches ordinary Cmd/Ctrl+C, refresh plus source change retains the original captured payload, exactly one write occurs, and a failed write produces one error without false success. Cross-subject/body copying matches the SDK's baseline exactly, including the paragraph newline. Disabled and cancelled actions do not write or post notices.

A narrow independent review found no concrete implementation regression. It identified missing interaction coverage of the target-free SDK fallback. `TestMessageWithoutTargetsKeepsSDKCopyAndSelectAll` now opens that actual default menu, executes Copy and Select All, and checks the clipboard and absence of app target actions. The review did not rerun tests itself.

Related confirmed issues were repaired before this increment: message text/icons now use `scaledFont(14)`; explicit commit/worktree selections invalidate old message request/menu ownership before early returns. Tests cover reselecting the same source, A→B→A and A→worktree→A before resolution consumption and while paging. Existing successful jump and Back/Forward tests remain green. Details and the preceding 521-input gate are in `2026-11-18-message-commit-navigation-scope.md`.

Latest combined message, selection-copy, history-navigation and action-copy race passed **38.939 s**. Captures passed in **3.155 s** using an ignored overlay. Main read all eight updated 480×360 captures at maximum 1.15 scale: `build/message-target-{en,zh}-{light,dark}.png` and their `-disabled.png` variants. Full labels wrap, stay inside the viewport and distinguish copy, external opening and commit navigation; guarded SHA rows are disabled while URLs remain available. The height regression exercises actual wrapped layout across all four language/theme combinations; it is not an independent font-metric assertion. Max-scale mixed-menu keyboard navigation is covered by the retained production regression.

## Final gate and limitations

Accepted the frozen-source wrapper task `b7vcjcrzi`, with prefix `build/alignment-message-selection-copy` and default `make check`, after reading its complete log, verification JSON and actual task output. All ordinary tests passed (`internal/lobedesk` **184.412 s**), all race tests passed (`internal/lobedesk` **510.955 s**), and `go vet ./...` passed. All **526** source inputs remained unchanged, all **8** packages matched, and both check and diff-check exits were **0**. The preceding 521-input gate predates the two corrections and selection-copy entry; this gate covers the corrected production snapshot. Later Select All prototypes live only in ignored overlay files and are not a production Select All implementation.

Headless menu/clipboard simulation is not native system clipboard or OS-event acceptance. Native Sublime behavior, browser integration and Windows/Linux native runtime remain unverified. All fixtures use owned disposable repositories and no network or credentials.
