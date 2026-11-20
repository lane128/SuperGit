# Copy Author Name

**Status:** Implemented; focused race and independent static review passed. Whole ordinary suite and vet passed for the 505-input name-only snapshot. The follow-up adding full identity/email is recorded separately in copy-author-identity. No commit, installation or publication.

**Goal:** Copy a selected historical commit's full author name directly from its summary, including when that name is visually truncated.

## Reference and scope

Main independently read `/Applications/Sublime Merge.app/Contents/MacOS/Packages/Default.sublime-package`, `Commit Author.sublime-menu`. It declares `Copy Author Name`, `copy_to_clipboard`, `text: $author`. The resource also declares author-plus-email and email-only actions; this increment implements only the name, which the existing Commit model already supplies. These declarations are data evidence, not native interaction verification.

A product-role comparison selected this bounded read-only gap. A separate design review of the history-reference chooser suggested clearer scope wording; that is a future candidate and is not included here, preserving the priority of concrete reference capabilities over general polish.

## Implementation

- `internal/lobedesk/commit_author_menu.go` adds an author-text context menu and captures the complete displayed author while the menu is closed. Refreshing the summary cannot silently retarget an already-open menu.
- `commit_info.go` attaches it to the existing author text without adding a new button or shortcut. Empty author disables the action.
- `i18n.go` adds Copy Author Name / 复制作者姓名. The action reuses `copyText` for existing success/failure feedback and never starts a Git operation or refreshes the repository.

## Acceptance evidence

`commit_author_menu_test.go` first failed because the context action did not exist. Actual right-click/menu-click tests then passed for English/Chinese, full Unicode/space/long names, a summary refresh while the menu is open, unchanged selected SHA and title/body draft, and clipboard failure. The clipboard boundary is injected; this is not a native OS clipboard runtime claim.

Focused race including author-copy, existing commit-message selection, narrow parent links and action-copy consistency passed 1.729 s. A fresh independent static test-role review found no reportable miscopy or summary interaction defect; it did not rerun tests.

Whole ordinary suite and vet passed as `bem92ey4a`, prefix `build/alignment-copy-author-name` (lobedesk 146.115 s; other packages cached). Main read complete log, verification JSON and task output: all 505 source inputs unchanged, eight packages matched and diff formatting passed. This accepts the name-only snapshot before the separate full identity/email follow-up. No new whole UI race was run for this bounded read-only menu; earlier undo full-race evidence remains separate. Native clipboard and native reference behavior remain unverified.
