# Historical Preview Errors Implementation Plan

**Goal:** Distinguish failed historical file-list/Diff reads from valid empty results, with retries targeting the currently selected immutable SHA/path.

**Spec:** docs/superpowers/specs/2026-11-11-product-workflows-design.md (accurate immutable commit details and isolated reads).

**Architecture:** Keep the existing cancellable commit lane; add explicit file-list and Diff completion/error state. Typed request identities include repository ticket, selected SHA and path. File-list retry preserves commit message; Diff retry preserves file list. Reset errors on target replacement, reject stale callbacks, and expose retry after an interrupted read rather than reporting an empty result.

**Constraints:** Current non-Git workspace; no project commit, push, publication or user repository writes. Use temporary local fixtures. No callback ListState resets. Preserve existing native keyboard/focus behavior and independent commit-message lane. One narrow read-only reviewer after implementation.

**Review focus:** Failed reads must never claim empty success. Retry must retain exact SHA/path. Obsolete callbacks, including duplicate completion, cannot alter current state. Writing/opening blocks new reads; canceled unfinished reads remain recoverable. Empty valid commits remain keyboard accessible. No errors leak between repositories or files.

- [x] State RED→GREEN: completion/identity/error tests; interrupted reads and guards; preserve message and unrelated workspace state.
- [x] UI and native verification: show file-list and Diff loading/failure/retry states; actual missing-tree and missing-blob fixtures fail then recover when objects are supplied, with HEAD/Index/working data unchanged.
- [x] Independent source review, fix important findings, full Go/race/vet/app and native regression; inspect final logs/screenshots; update README and schedule next iteration.

## Evidence and rulings

Root cause: selectCommit/showCommitFile clear preview before reading; their error branches only call record. diffView treats empty preview as successful no-text Diff and commitFileListView has no loading/error/empty distinction. Existing commit lane cancellation/generation isolation is retained.

Ruling: scope this iteration to immutable historical preview failures — workspace Diff and stash failures use other state/lifecycles and remain separate work. Cost: their existing generic failure displays are not improved here.

Four new desktop state tests first failed on missing preview state/APIs, then passed. They cover file-list failure versus successful empty list; Diff failure/partial-data rejection and recovery at a raw unusual path; SHA/repository/ticket/path/closed/interrupted stale rejection, duplicate completion, cancellation and retry; blocked starts during writing/opening/closed/no-repo/no-SHA/unknown path. A retry preserves independent commit message and file list as appropriate. Completed reads release contexts; new targets clear errors.

`make check app` passed full Go tests, race, vet, normal app build and local ad-hoc signature verification (`build/history-preview-check.log`, bglbagbcm, exit 0). Dedicated real-window run passed all 9 commit-file/message/preview groups (`build/history-preview-native.log`, bl5obxenh, exit 0). Inspected `commit-files-read-error.png`, `commit-diff-read-error.png`, `commit-preview-recovered.png`: local failures and retry controls are visible; restoring fixture objects recovers the same SHA/path and accurate Diff. Missing tree also makes Git's separate message read fail; that existing independent message retry is correctly retained, not silently retried by file-list recovery. HEAD, Index and working bytes stay unchanged. README updated. Independent read-only reviewer a09757b2f7d7eb59b completed one source pass and reported no concrete defects across failure/empty distinctions, retry SHA/path, stale/duplicate results, interruption, repository/write/open/snapshot interactions and list lifetime. Reviewer ran no tests or GUI, and explicitly left actual accessibility lifetime/focus to runtime validation; the main session's dedicated native run and final full regression cover focus/navigation. No second review round. Final full regression completed (`build/history-preview-native-final.log`, bu59g8xjy, exit 0): inspected all output and counted 54 PASS groups (workflows 10, Diff 6, watch 6, history 10, keyboard 6, workspace files 7, commit files/messages/preview errors 9). All seven final application logs are 0 bytes. Reopened final-run file-list failure, Diff failure and recovered-preview screenshots; verified explicit errors/retry controls, usable metadata/files and recovered exact-path Diff. Iteration complete; continue the existing dynamic product-development loop. No project commits, publication or user repository writes.
