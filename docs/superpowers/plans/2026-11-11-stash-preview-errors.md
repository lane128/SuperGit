# Stash Preview Errors Implementation Plan

**Goal:** Distinguish Stash preview loading, failure, successful empty Diff and no selection; retry the selected SHA without changing any Stash or repository data.

**Spec:** docs/superpowers/specs/2026-11-11-product-workflows-design.md (Stash SHA identity, read-only inspection, isolated background reads).

**Architecture:** Keep the `stash` lane with typed ticket/SHA requests and a cancellable child of repository context. Every new read clears preview/error and marks incomplete; completion releases context and rejects stale/duplicate results. Open/write cancels reads; failed open/interrupted read stays retryable. Commit callbacks only request list reset, consumed at the start of stashes view, preserving MyGo accessibility lifetime. Snapshot refresh retains selected immutable preview; backend membership validation reports missing Stash rather than reading a shifted reflog entry.

**Constraints:** No user repositories, project commits, remote writes or publication. New temporary local fixture only. Existing apply/pop/drop confirmation and SHA guards remain. One independent narrow read-only source review; no parallel implementers or Git initialization in this non-Git workspace.

**Review focus:** Retry never targets stash@{n} after reflog shift. Failure discards partial data. Selection/open/write cancels old process context and rejects old result. No callback ListState zeroing; pending reset is applied only at view entry. Errors clear on replacement; empty success remains distinct from no selection and failure. Interrupted reads cannot become successful empty preview.

- [x] State RED→GREEN: loading/error/empty/retry, cancellation/stale/duplicate guards, open/write interruption and old list-state lifetime.
- [x] UI/native: local error/current-SHA retry, explicit empty/no selection, source guards; isolated --stash native fixture verifies oversized failure, reflog shift, rapid replacement and selected entry removal, with HEAD/Index/working/stash data checked.
- [x] One source review, full Go/race/vet/app/native checks, final logs/screenshots, README and next loop.

## Evidence and rulings

Root cause: selectStash clears Diff and only records an error; view renders an empty patch for no selection/failure/success alike. It also zeroes stashList inside a button callback and uses repository context without a per-selection cancel. Other lists already defer resets until view entry.

Ruling: retain independent Stash preview across snapshots, even if reflog positions move — retry uses SHA and backend verifies membership. Cost: removed selected entries remain displayed by their captured SHA until an explicit retry/selection, at which point missing membership is reported; preview is read-only and write buttons belong to current list rows.

Four new state tests first failed on missing APIs/state, then passed: failure/partial-data discard/current-SHA retry/successful empty Diff, cancellation/replacement/duplicate rejection, stale repository/ticket/SHA/closed/interrupted protection, busy-start guards, repository lifetime, completed-preview retention and snapshot/reflog shift isolation. A borrowed ListState comparison confirms callbacks preserve the committed frame and request a later reset.

`make check app` passed full Go tests, race, vet, normal app build and local ad-hoc signature verification (`build/stash-preview-errors-check.log`, bsxeke5a3, exit 0). Dedicated `--stash` native run passed all four groups (`build/stash-preview-errors-native.log`, bdf6w65k9, exit 0): explicit no-selection/empty success, real >4 MiB failure/retry, shifted reflog and rapid selection, removed selected SHA refusal. Browsing/retry preserve HEAD, Index, tracked fixture bytes, clean status and expected Stash list; external fixture setup changes are explicit. Inspected final empty, failure, replacement and removed-SHA screenshots. Added --stash to the maintained native runner/Makefile and README.

Independent read-only reviewer aee0d46e27998d5d6 completed one narrow source pass with no concrete defects. No second round. Reviewer declined actual process termination, native accessibility/timing and rendered behavior; main-session native checks cover visible errors/retry and rapid replacement, while cancellation and async open/write interruption are verified by state helpers/source, not a dedicated process-timing or real failed-open race test. Ruling: retain that verification boundary — helpers directly exercise the cancellation/identity contract, and native workflows exercise shared hooks — cost: precise runtime termination/open-write race behavior is not independently measured in this iteration. Full native regression completed (`build/stash-preview-errors-native-final.log`, bs4y9uuvr, exit 0); inspected all output and counted 61 PASS groups: workflows 10, Diff 6, watch 6, history 10, keyboard 6, workspace files/errors 10, commit files/messages/errors 9, Stash preview 4. All eight final application logs are 0 bytes. Reopened final-run empty, oversized failure, replacement and removed-SHA screenshots: distinct empty result, current-SHA retry and missing-entry errors are visible; replacement displays the accurate small Diff with no old local error. Iteration complete; continue existing dynamic product loop. No project commits, push, publication or user repository writes.
