# Copy Author Identity

**Status:** Implemented; full make check accepted the 508-input implementation snapshot. Additional test-only boundary coverage passed focused race; whole ordinary suite and vet for that follow-up are running. No commit, installation or publication.

**Goal:** Complete the existing author's context menu with Copy Author and Copy Author Email, preserving the original author's exact name and email.

## Reference and product decision

The installed `Default.sublime-package/Commit Author.sublime-menu` declares this order: Copy Author (`$author <$email>`), Copy Author Name (`$author`), Copy Author Email (`$email`). This is resource evidence, not native runtime verification. Product and design roles retained that order and existing clipboard feedback; no additional settings or controls. Chinese labels: 复制作者、复制作者姓名、复制作者邮箱. Missing email disables identity/email items, missing name disables name-only copy. Outputs have no added trailing newline. Never use committer identity as a substitute.

## Implementation

- `gitrepo.Commit` now carries `AuthorEmail`.
- `history.go` centralizes the actual log format and seven-field record width. `%ae` follows `%an`; parser date/refs/local-day indexes move accordingly. Ordinary, page, stream and rename-following file-page readers use the same format; stream framing, file-page skip and offset overflow bounds use the shared count.
- Existing date-parser fixtures include the new email column while preserving their date/day assertions. This is the internal record contract update, not weakened validation.
- `commit_author_menu.go` captures name and email as one identity while closed, so refresh cannot mix an old name with a new email. `commit_info.go` passes both existing selected-history metadata fields. Reuses `copyText` without Git work, staging, commit, refresh or draft changes.
- Branch/tag metadata producers were inspected but are outside this scope: the current summary obtains selected commits through history rows/commits/refreshSelectedCommit, not branch/tag tip metadata.

## Verification

New backend and UI cases first failed because author email was absent. A real owned Git fixture uses Unicode/spaced author name, mixed-case email and a different committer. It checks exact author name/email and usable date/day across History, FileHistory, single-row HistoryPage, single-row HistoryStream and single-row rename-following FileHistoryPage; includes root, merge and rename commits.

Actual UI clicks check full identity/email output and keeping the captured pair after summary refresh. English/Chinese missing-email menu tests verify only name remains enabled and clicking disabled email cannot copy a placeholder. Prior full-name, clipboard-failure, message selection and narrow-summary parent-link tests remain in the focused checks.

Final targeted history-reader race passed 7.248 s; author menu/summary race passed 2.009 s. Windows/Linux amd64 CGO-disabled backend test compilation passed; this is compilation only. A fresh independent static review checked all producers/parser/framing/skip indexes and the captured menu identity and found no reportable issue. It did not rerun tests.

Full ordinary suite, full race and vet passed as `bq19041ft`, prefix `build/alignment-copy-author-identity`. Main read complete log, source verification and task output: all 508 inputs unchanged, eight packages matched, check/diff exits zero. Ordinary gitrepo 82.606 s and lobedesk 148.753 s; full race gitrepo 90.729 s and lobedesk 485.649 s. This accepts the complete identity/email implementation snapshot. Native system clipboard and native reference interaction are still unverified.

## Boundary follow-up

The independent testing role found a deterministic coverage gap, not a demonstrated production bug: arbitrary write boundaries and empty email/refs were not exercised together. `history_identity_chunks_test.go` now checks every two-chunk cut and one-byte writes, including splitting UTF-8 names, root/merge parents, empty email and refs in one record, and skipping a complete record across chunks while retaining the next. The expected width is explicitly seven. An isolated overlay that restored six-field stream framing failed this test as expected; no production source was mutated. Final isolated race passed 1.315 s.

The design role requested checking the long-name hit target with missing-email items. An isolated 480×360 experiment at actual maximum font scale 1.15 passed race 1.645 s, for English and Chinese. The existing disabled-menu regression now includes that long author, date, maximum font and visible hit-target assertion. This is headless UI interaction, not native runtime acceptance.

Integrated history identity/framing race passed 1.537 s; integrated author-menu race passed 1.864 s. Only one test file was added and one existing test strengthened after the 508-input gate; production code stayed unchanged. Whole ordinary suite and vet passed as `bazhybjl3`, prefix `build/alignment-author-boundary-final`: gitrepo 99.805 s, lobedesk 167.883 s. Main read full log, verification and task output: 509 inputs unchanged, eight packages matched, check/diff exits zero. No repeated whole UI race is claimed for this test-only follow-up.

A second focused test review found two remaining coverage gaps: empty email/refs were only retained, never skipped, and only the disabled email item was clicked. The framing regression now tests both record orders, and the UI regression independently reopens and clicks both disabled identity/email actions. Focused race passed (history 1.194 s; combined author/date menus 2.674 s). These later test changes belong to the subsequent copy-date verification snapshot, not the accepted 509-input snapshot.
