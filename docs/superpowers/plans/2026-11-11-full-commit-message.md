# Full Commit Message Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans inline under the existing continuous product authorization. Work in the current non-Git workspace; no project commits/publication.

**Goal:** Read the selected commit's full message, including multiline body and trailers, in the existing metadata scroll area without delaying file Diff or displaying a previous commit's body.

**Architecture:** A typed SHA-only CommitMessage API reads UTF-8 text through bounded system Git. A separate cancellable immutable-message lane follows selected SHA and repository generation; changing files or refreshing a snapshot does not cancel or replace this message. View displays the actual first line as title and remaining message in the existing metadata scroll region, with loading/error/retry states.

**Tech Stack:** Go, system Git, MyGo v0.8.3, native macOS automation.

**Spec:** docs/superpowers/specs/2026-11-11-product-workflows-design.md (accurate commit details, read isolation); existing complete-product prototype history detail flow.

## Global Constraints

- No user repository writes, external services, project commit/push or publication. Tests use fresh temporary repositories.
- Keep typed argv, OID validation, Git output/time limits and cancellation.
- No callback ListState resets, no shared file-Diff lane, no changes to commit/write behavior.
- Preserve body line breaks, Unicode, indentation, trailers and trailing newline from Git; do not silently truncate. Display text only, no active links or markup.
- Bound message display to 1 MiB; an oversized or invalid UTF-8 result is a reported read failure and leaves files/Diff usable.
- One narrow independent read-only review after implementation.

## Review Focus

- Rapid commit switching and repository changes reject/cancel obsolete bodies; retrying an error must target the current SHA.
- File navigation and automatic refresh retain an already read immutable message and focus.
- A write while immutable metadata reads must not leave a permanent loading indicator or bind body to a new commit.
- Single-line/empty messages, multiline first paragraph, Unicode and trailers preserve content without duplicating a folded title.
- Oversized/read-failed messages show a retry path while file list/Diff stay usable; long normal bodies scroll within metadata rather than expanding the files viewport.

## Tasks

- [x] Backend RED→GREEN: CommitMessage(ctx, sha) validates immutable OID, explicitly emits UTF-8 without signature output, preserves full message framing, rejects invalid/oversized text and respects cancellation. Real Git tests cover root/merge/empty commit, unusual whitespace/trailers, non-UTF8 source encoding and size/error cases.
- [x] State/view RED→GREEN: selected-message state with independent lane, repository cancellation, stale SHA/ticket protection, current-SHA retry and body/title splitting. Open cancels message read and preserves valid old details if opening fails; accepted open clears state. Show message loading/error/retry and body inside existing 180 DIP metadata scroll. Snapshot refresh and file selection keep immutable body.
- [x] Native and full verification: add isolated message cases to the existing commit-file native fixture (body/trailers, scroll, rapid selection, refresh, oversized failure and current retry), verify exact SHA/Diff and no repository writes. Full Go/race/vet/app and native regressions, inspect screenshots/logs, one read-only review, README, evidence and next dynamic iteration.

Pre-flight: message state is independent of commit-files/Diff state; both are keyed by selected SHA. Metadata visibility comes from the current snapshot's selected commit; the message must never replace that identity. Repository-open generation switch already rejects all old reads.

Ruling: use a separate immutable-read lane for messages — file browsing and background refresh must not interrupt a selected commit's body — cost: one additional short Git process per commit selection. A same-repository write may finish concurrently with a read of the already selected immutable SHA; neither changes the other's target.
Ruling: retain the existing 180 DIP metadata viewport and make its full message scrollable — keeps the changed-files and Diff panes usable — cost: long messages require scrolling to reach later metadata/body content.

## Evidence

Three new backend tests first failed on missing CommitMessage, then passed against real Git: original multi-paragraph/indented/trailer text and trailing newlines; captured SHA after HEAD moves; root/merge/empty messages; explicit UTF-8 conversion despite ISO-8859-1 repository output settings; invalid/missing OIDs, canceled context, invalid UTF-8 raw commit object and 1 MiB bound. Four initial desktop state/display tests first failed on missing APIs, then passed: superseded/repository cancellation, stale SHA/ticket/generation/closed rejection, same-repository write concurrency, snapshot retention, error/current-SHA retry and exact title/body splitting.

Implementation: message reads run in independent `commit-message` lane, with a child of repository context and no coupling to commit-file/Diff cancellation. Open cancels before changing generation; successful open clears message state. Metadata title uses the original first line when loaded, with the remaining message in the existing scroll region. Loading/error/retry are local to the selected message and do not hide file Diff.

Native investigation: first body assertion incorrectly used the automation snapshot's 120-character truncated label. Adding --full alone still failed because -i removes plain text elements; inspected failure snapshots/screenshots, CLI skill and actual interactive filtering implementation. controls() now uses unfiltered --full snapshots to assert the entire message. This was a harness visibility issue; the body had already appeared in the actual window. Corrected native run passed all seven commit-file/message groups (`build/full-commit-message-native.log`, task b4n8bmwwo, exit 0). Inspected `commit-message-full-body.png`: bottom lines and both trailers visible while file viewport/Diff remain fixed; `commit-message-read-error.png`: explicit 1 MiB error, enabled current-message retry and accurate file Diff.

Independent source reviewer ae8ea1ec5446d772f performed one narrow read-only pass (12 calls), identifying a parent-navigation guard gap: during a write the enabled parent button could change selected SHA while the new message read was blocked, leaving it unread. Added TestCommitMessageParentNavigationRespectsWriteAndOpenGuards (missing guard API RED→GREEN); centralized canSelectCommit in selectCommit and parent button, rejecting writing/opening/closed/no-repo/out-of-range while allowing read-only navigation during snapshot loading. No second review round. Reviewer declined native text rendering, scrolling/focus and actual overlapping reads; main-session native checks provide that evidence, without a second independent runtime pass.

Fresh post-review `make check app` passed full Go tests, race, vet, normal app build and local ad-hoc signature verification (`build/full-commit-message-check.log`, task bw647nn8n, exit 0). Final `make native-check` completed successfully (`build/full-commit-message-native-final.log`, task bto1cyovw, exit 0); inspected the full output and counted 52 PASS groups: workflows 10, Diff 6, watch 6, history 10, keyboard 6, workspace files 7, commit files/messages 7. All seven final application logs (`workflows`, `diff`, `watch`, `history`, `keyboard`, `files`, `commit_files`) are 0 bytes. Reopened both screenshots produced by the final run: full-body tail/trailers remain visible in bounded metadata scroll with files and correct Diff below; oversized-message error and current-SHA retry remain visible with correct Diff. README covers full-message behavior, encoding, bound, retry and parent navigation guard. This iteration is complete; no project commit/publish or user repository writes. Continue the existing dynamic product-development loop.
