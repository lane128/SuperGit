# History Pagination Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans inline. The user has authorized continuous product implementation; no project commits or publication.

**Goal:** Browse beyond 2,000 commits through explicit 300-commit pages, with accurate DAG/SHA selection, bounded Git output per read and no duplicated or skipped commits when refs move between pages.

**Architecture:** Capture immutable commit tips from all refs and detached HEAD. Read topologically ordered pages against those OIDs over stdin, with one lookahead record to identify the end. Append in an independent guarded/cancellable read lane. Refresh rereads the loaded depth against a fresh tip set in bounded batches; changing repository, refreshing or writing cancels and rejects pending page reads.

**Tech Stack:** Go, system Git, MyGo v0.8.3, macOS native automation.

**Spec:** docs/superpowers/specs/2026-11-11-product-workflows-design.md (history continuation, accurate parents and async isolation).

## Global Constraints

- Work inline in existing non-Git directory; do not initialize Git, commit, publish, or use user repositories for tests.
- Keep Git argv typed, literal paths, output bounds, cancellation and repository-generation guards.
- Preserve existing search over loaded commits, selected SHA/file Diff, focus, draft and workspace state.
- ListState resets only during view construction; callbacks must not invalidate the committed accessibility tree.
- One narrow independent read-only source review; no parallel implementers.

## Review Focus

- Refs/HEAD move between pages: use pinned tips so page order and membership remain consistent until refresh.
- Multi-parent history, annotated tags, detached HEAD and unborn repositories: no duplicate records, omitted tips or default-HEAD fallback for an empty pinned set.
- Refresh/write/repository switch with a pending page: cancel and reject stale results without appending to a new history generation.
- Search excludes the current selection, or appended data moves a row: preserve SHA and pending list input, reject old presentation callbacks.
- Read failure or terminal short page: retain existing history/Diff and enable retry only when appropriate; never infer completion from a failed command.

## Tasks

- [x] Backend RED→GREEN: HistoryPage(ctx, tips, offset, limit) returns copied tips, commits and HasMore; ReadHistory(ctx, limit) aggregates at most 300 records per command; History remains a convenience wrapper. Real Git tests cover pinned ref movement, merge ordering, annotated tags, detached/unborn, exact-multiple/short final pages, invalid arguments and cancellation. A fast-import fixture exceeds 2,000 commits.
- [x] UI RED→GREEN: snapshot carries tips/HasMore; independent page loading with current ticket, history revision and offset validation; immutable append updates historyRevision and loaded depth. Refresh/open/write cancel page reads. Keep list focus, selected SHA and commit Diff; show loading/end states on the existing continuation button. State tests cover busy/end guards, stale/duplicate/invalid pages, cancellation, selection and unchanged preview.
- [x] Native: update large fixture checks for 300→2,100→2,400 paging and exact old SHA/Diff, continued paging after external refs move, refresh at retained depth, exhaustion on a small fixture and preservation of keyboard behavior. Run full Go/race/vet/app and complete native regression, inspect screenshots/logs, one independent review, update README and schedule next iteration.

Pre-flight: backend page metadata is consumed by snapshot and append; both must use the same captured tip set and exact loaded offset. Existing immutable history presentation already maps filtered rows to original snapshot indices. Existing selection/detail lane stays separate from page reads.

Ruling: refresh retains the loaded depth by rereading it in bounded batches — preserves the user's browsing depth and updates reference labels — cost: deep refresh rewalks history and is slower than a persistent Git traversal; commands remain in the background and cancellable.
Ruling: retain explicit continuation and loaded-only search — matches the existing UI and avoids loading an entire large repository automatically — cost: finding very old commits may require multiple continuation clicks; global Git search is a separate iteration.

## Evidence

Backend and UI state APIs first failed their new tests on missing APIs, then passed. Real backend tests cover a 2,105-commit fast-import history, ref movement between pages, copied pinned tips, ordered merge/tag/detached traversal, unborn versus explicitly pinned empty history, exact terminal page, invalid ranges/OIDs and cancellation. UI tests cover independent loading, preserved SHA/Diff/draft, stale repository/revision/offset/ticket/tips, busy/end guards, duplicate/malformed pages, read failure and retry. A second RED→GREEN regression requires finishing a page to release its context and reject replayed callbacks.

Output-size investigation: a 1,000-commit fixture with 5,000-byte subjects exceeded the 4 MiB command cap when a refresh attempted a 2,000-record batch, despite 300-record pages fitting. The new aggregate-output regression failed with the real limit error, then passed after ReadHistory was limited to 300 records per batch. This matches append and refresh output sizes. The full Go/app run before this extra regression passed; a fresh complete run is still required after the batching fix.

First native run reached 2,400 commits and passed scrolling. New deep-selection test then timed out waiting for the numbered file's content because Perf commit 10000 is a merge and its first changed file is feature.txt. Failure snapshot and screenshot showed the correct merge SHA, two parents and feature Diff. Corrected the test to explicitly select files/00000.txt, preserving the existing first-file behavior. Fresh `make check app` after the batching fix passed full Go tests, race, vet, normal app build and local ad-hoc signature verification (`build/history-pagination-check.log`, task bcb18n3sh, exit 0). Complete `make native-check` rebuilt e2e tools and passed all 49 scenarios (`build/history-pagination-native-final.log`, task be4z57p4z, exit 0): workflows 10, Diff 6, watch 6, history 10, keyboard 6, workspace files 7, commit files 4. All seven application logs are empty. Inspected final `history-paged-deep-selection.png` (2,400 loaded, exact merge SHA and selected numbered-file Diff), `history-paged-after-refresh.png` (2,700 loaded, exact last SHA and Version 9309 Diff) and `commit-files-keyboard-empty.png` (disabled terminal-history button, no stale empty-commit files/Diff). Native command timing from 300 through 2,400 was about 0.534 s, including automation and waits; this is not frame time. Independent read-only source reviewer a3ef1582194404778 is running; no result yet. Fresh large-fixture benchmark completed (task b1oo6ih2w, exit 0; `build/history-pagination-benchmark.log`): five-run means, including Git subprocesses, History300 61 ms, History2000 295 ms, Status 152 ms, ChangeStamp 196 ms, full 2,000-row snapshot 699 ms. README now records these current values and the cost of batched depth refresh rather than retaining the old single-command history timings. Page append avoids rereading the complete snapshot. Independent read-only reviewer a3ef1582194404778 reported two concrete findings, both fixed with RED→GREEN regressions in `history_refresh_test.go`:
- A failed snapshot history read installed empty rows and `HasMore=false`, erasing loaded history and mislabeling it exhausted. New tests failed on discarded rows and on the initial-failure label. `snapshot.historyErr` now distinguishes failure; `installSnapshot` retains prior rows/tips/more on failure; the button reads 「历史读取失败，请刷新」 and paging is disabled until a successful refresh; an initial failure no longer shows the not-yet-committed message.
- Search `OnChange` still zeroed `historyList` after construction. The new callback regression failed; `historySearchChanged` now requests `historyListReset`, consumed at the start of the history view, matching the other lists.
Post-fix `make check app` passed full tests, race, vet, normal build and ad-hoc signature verification (`build/history-pagination-check.log`, task box05eoxu, exit 0). Post-fix `make native-check` passed all 49 scenarios with empty application logs (`build/history-pagination-native-final.log`, task bzebngf5m, exit 0): workflows 10, Diff 6, watch 6, history 10, keyboard 6, workspace files 7, commit files 4. Re-inspected `history-paged-deep-selection.png` (2,400 loaded, exact merge SHA, explicit files/00000.txt Diff) and `history-paged-after-refresh.png` (2,700 loaded, last row SHA and Version 9309 Diff); `commit-files-keyboard-empty.png` confirms the disabled terminal label on a small history. README now states failed-refresh retention and view-time search reset.

Final review handling: reviewer findings were source-level and both reproduced; no additional review round was requested. Reviewer did not run tests/GUI or judge native focus, rapid input, performance or repository preservation; post-fix full checks and native runs above provide that evidence. Remaining limitations are documented rather than silently resolved: Git `--skip` rewalks prefix history so deep paging/refresh grows slower and memory grows with loaded rows; each command is bounded at 300 rows plus lookahead but a single huge commit record can still exceed the 4 MiB output cap and is now reported without destroying loaded rows; there is no streaming traversal, history eviction or global Git-side search yet. No project commit/publish, no user repository writes.
