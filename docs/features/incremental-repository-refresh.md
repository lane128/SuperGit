# Incremental repository refresh

Status: Approved
Date: 2026-11-24
Related: SoulApp opening and Fetch performance diagnosis

## Background and goal

SoulApp has 75 configured submodules. A single instrumented native-app run took approximately 14 seconds to reach automatic Fetch; Fetch took 1.52 seconds, followed by approximately 13 seconds of full refresh. This is one warm-machine observation, not a statistical benchmark. Parent status, watcher baseline and per-submodule reads overlap in information but execute separately.

Goal: publish a usable parent-repository view before detailed submodule traversal finishes; avoid unconditional worktree/submodule scans after Fetch; retain correct dirty/conflict/operation state and invalidate stale asynchronous results.

## Non-goals

No remote protocol changes, persistent cache, new page, wholesale visual redesign, unlimited concurrency, or changes to Git write semantics. Do not silently ignore submodule changes in the authoritative parent status. Do not shorten network timeouts to conceal slowness.

## Approach

### 1. Separate first usable snapshot from supplemental details

First publish a coherent parent snapshot containing checkout identity, HEAD, authoritative parent status, operation state and initial bounded history/refs. Parent status continues to include submodule changes. History and independent reads may overlap, but detailed child status and watcher baseline are not prerequisites for publishing the parent snapshot.

Load the submodule inventory and gitlinks in bulk: one configuration read, one index read and one HEAD tree read, correctly handling unborn HEAD, unmerged index stages and missing/uninitialized modules. Reuse existing parsing semantics. Detailed child branch, dirty state and changed-file counts arrive asynchronously. Existing unknown counts must be shown as loading/unknown, never as zero/clean. A failed child read marks that child unavailable and retains any last known value with stale indication.

Start with a maximum of four child workers, benchmark two and four before fixing the default. Bound concurrency per checkout across initial load and watcher jobs, not independently for each producer. Coalesce duplicate child requests. A slow child cannot block all children or the parent view.

### 2. Refresh according to what changed

Use explicit refresh intents, backed by the existing load sequence/context guards rather than an unrelated second scheduler:

- Fetch completion: reread refs, ahead/behind and affected history/decorations; no unconditional full child scan, parent status scan or baseline rebuild.
- Working tree/index notification: authoritative parent status and affected child details; expand scope when identity cannot be determined.
- Checkout, merge, rebase, reset and other worktree mutations: retain existing conservative full invalidation until individually proven safe.
- Manual Refresh: request a full reconciliation, including all child details; coalesce overlapping requests.

Fetch failure also reconciles refs because Git can update some refs before failing. Multiple-remote fetches coalesce refresh work. Capture HEAD/checkout identity before and after remote work; external HEAD/index/operation changes expand refresh scope. Do not derive ahead/behind from an obsolete HEAD.

Fetch progress ends when network work ends. Any remaining ref/history reconciliation has its own bounded loading indication. Preserve current write-operation exclusion and preconditions; enabling safe read-only navigation must not accidentally enable conflicting writes.

### 3. Watcher consistency and cancellation

Do not merely remove the watcher baseline read: initialize monitoring without a blind interval. Install observation before initial reads where feasible; retain a dirty generation/pending event through initial publication and reconcile after baseline initialization. Existing periodic observation remains a fallback for missing/coalesced filesystem events. Watcher initialization must not overwrite a newer pending change with an old clean baseline.

Avoid rebuilding the expensive submodule baseline for ref-only changes. Reuse status text from the same parent observation. Child dirtiness remains observable for tracked, untracked, staged and HEAD changes, including external operations during Fetch.

Every result carries checkout identity and refresh generation; superseded work is cancelled, and late results cannot publish. Independent parent/ref/child generations must prevent a ref refresh from discarding a needed child update or an old child result from overwriting a new status. Preserve tab lifetime cancellation.

### Review resolutions: baseline, lifecycle and reproducibility

Split the existing combined ChangeStamp into separately comparable ref and local-state observations (local state includes index, worktree, operation and child state). A refs reconciliation advances only its captured ref baseline; it must never acknowledge pending local changes. A ref-only observation schedules ref reconciliation rather than full reopening. Observe at least two subsequent polling cycles in regression tests to prove Fetch does not merely defer full scanning. Do not introduce filesystem event infrastructure in this change: retain periodic polling. Capture local identity before and after initial reads, retain pending generations, and schedule reconciliation if observations differ. Recheck HEAD/index/operation before parent publication; discard inconsistent results and retry with bounded attempts, then show an explicit changing/unavailable state rather than publishing mismatched data. Polling cannot provide an atomic filesystem snapshot, so preserve operation-time validation for writes.

Success, error, timeout and user cancellation can all leave updated refs. Reconciliation runs from the live tab context, not the cancelled Fetch context, with its own cancellation and generation. Closing the tab cancels reconciliation; replacement refresh takes ownership of the pending reconciliation intent. Preserve the original operation outcome for user feedback and always settle its progress flags. Test a two-remote Fetch where the first updates refs and the second blocks and is cancelled: first-remote refs become visible, cancellation is reported correctly, stale callbacks are rejected and loading converges.

Planned executable acceptance entry point: `python3 scripts/native-lobe-incremental-refresh-performance.py` (to be added during implementation). Default mode creates an isolated temporary 75-child repository and local bare remotes, with staged, unstaged, untracked, uninitialized and conflicting child cases; it never constructs fixtures inside SoulApp. A fixture-only Git wrapper blocks a chosen child status on a release signal. The script asserts that the parent view displays the expected branch and dirty state and accepts a read-only navigation action before releasing the child. Timestamp first usable view at this successful UI assertion, measured from process launch. Timestamp post-Fetch reconciliation from Fetch process exit until the expected fetched ref/history and ahead/behind are visible and refresh progress has settled. Collect subprocess command counts through two further watcher polls. An explicit `--repo /Users/adam/SoulApp` mode performs only reads and Fetch, preserves user preferences and never creates commits or changes worktree/index contents.

Deterministic fixture gates (bounded command counts, correct state, responsive parent view under a blocked child, no delayed full refresh, cancellation convergence) are mandatory. The <=4-second and <=1-second medians are machine-specific performance acceptance targets, not timing assertions in unit CI; report misses for user acceptance rather than silently relaxing them. Run five warm samples and report each sample, median and maximum. Add initial-publication HEAD/index/operation race tests alongside Fetch race tests. These script names and hooks are planned artifacts, not existing or executed validation.

Design review completed read-only on 2026-11-24. The three required revisions above address combined watcher stamps, cancellation lifecycle and executable acceptance criteria; the initial-read consistency and polling-scope suggestions are also incorporated. User approved implementation on 2026-11-24: 按照推荐方案执行优化。

### 4. Delivery order

First implement bulk gitlink reads plus bounded child inspection and command-count regression tests. Then add incremental Fetch refresh with external-change reconciliation. Finally separate first snapshot publication from supplemental child/watcher initialization. Each step preserves a full-refresh fallback; do not introduce persistent cache or a public performance setting.

## Alternatives and trade-offs

- Only parallelize the current loop: smaller change but keeps repeated subprocesses and Fetch refresh cost; useful as part of bulk reads, insufficient alone.
- Ignore submodules in parent status: faster but can falsely show a clean checkout; rejected for authoritative status.
- Persistent cache: introduces invalidation/migration complexity; unnecessary for the measured bottleneck.
- Full refresh after every operation: simplest correctness model, retained for explicit manual refresh and worktree mutations, excessive after Fetch.

## Impact

- Modules / files: internal/gitrepo/submodule.go (bulk inventory/gitlinks and bounded details), internal/gitrepo/observe.go (baseline work), internal/lobedesk/app.go (publication and operation completion), internal/lobedesk/local_watch.go (coalescing/baseline), directly related submodule views and tests. Confirm exact seams during implementation.
- Compatibility: preserve repository/tab identity, detached/unborn HEAD, uninitialized and conflicted submodules, external working-tree changes and cancellation semantics.
- Data / update signing / permissions: no persisted schema or signing changes; no SSH changes; real-repository validation uses reads and Fetch only, never destructive fixture setup in SoulApp.

## Test plan

- Unit tests: bulk gitlink parity; index conflict stages; unborn HEAD; unusual paths; missing child; partial errors; worker concurrency and cancellation.
- Regression tests: 75-child fixture proves parent gitlink reads remain constant-count; blocked child cannot prevent parent publication; Fetch does not unconditionally trigger complete child inspection; delayed results cannot cross tab/refresh generations.
- Consistency tests: external HEAD/index/operation changes during Fetch; partial Fetch failure; multiple remotes; staged/untracked child changes; watcher event during initial baseline; overlapping explicit refresh; failed child never rendered clean.
- e2e / automation: isolated local-remote fixture exercises actual Fetch and UI loading states, followed by authorized read/Fetch checks on SoulApp. Capture command counts and phase timestamps, first usable UI and settled detail time, not only process elapsed time.
- T3 verification: go test -cover ./... once, make check for cross-package changes, independent scoped code review before acceptance. No tests run for this design-only phase.
- Uncovered areas: network-server latency cannot be guaranteed; cold-disk and very large nested submodule configurations need separate characterization.

## Acceptance steps

1. Build: make app; use e2e-build when automation requires it.
2. Open 75-child fixture and SoulApp; verify first usable parent view, progressive detail accuracy, manual Refresh and Fetch; switch tabs during reads; make external changes in isolated fixture.
3. Expected results: command-count and correctness gates above must pass. On the measured SoulApp machine, propose median first usable view <=4 seconds and median post-Fetch reconciliation <=1 second across five warm runs, with full details measured separately. These are targets, not existing results; report all runs and max, and explain resource contention. A deliberately delayed child must not hold first view. Correctness gates cannot be relaxed to reach timing targets.
4. Check loading/error states in light/dark and normal/narrow windows; no major layout changes or HTML mockup planned.
5. Evidence location: docs/acceptance/2026-11-24/ (use actual implementation acceptance date if later).

## Risks and rollback

- Risks: stale snapshots, missed filesystem events during baseline installation, excess disk contention, misleading clean/count states, premature write enablement after Fetch.
- Rollback: revert the implementation commits through normal revert; preserve the conservative full refresh path during staged implementation. No data migration required.
