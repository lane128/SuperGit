# Commit Diff Baseline Plan

> Continue loaded execution/TDD/verification guidance. Main implements; three independent worktree roles review. No commit, installation or publication.

**Goal:** Explain the baseline of the existing historical commit diff in its current pane.

**Scope:** Add a quiet bilingual line above the historical file-diff viewport. Show first-parent short SHA → selected-commit short SHA for commits with parents; identify a root as having no parent without claiming an explicit empty-tree backend argument. Ordinary and merge commits retain the existing first-parent diff. No parent selector, two-commit comparison, new settings, navigation or Git command change.

## Tasks

- [x] Observe missing baseline RED in actual historical diff-pane rendering.
- [x] Add minimal baseline renderer and labels, omitting unknown commit context and hidden/worktree diff paths.
- [x] Check owned root/ordinary/merge/empty content consistency, narrow bilingual/theme bounds and retained draft/target state.
- [x] Read product/design/testing implementation reviews and address confirmed findings.
- [x] Run related/race/vet/platform builds; freeze inputs and run complete make check; inspect full logs, hashes, file set and formatting.
- [x] Record actual acceptance and limits in alignment/e2e docs.

## Status

Implemented a secondary, two-line bilingual explanation above the existing historical diff viewport in historyDiffFile. Ordinary/merge shows first-parent short SHA → selected-commit short SHA; root says it has no parent. Unknown context, hidden patch and workspace paths do not gain a historical baseline. No Git command or interaction changed. Reference arbitrary-two-commit behavior remains unproven. Prior search-feedback full check passed against 468 unchanged inputs.

Actual roles product af0bab94b3cc9dfa4, design acd0a646b9a57eb74 (Emil/Impeccable), testing a750adfc8184d3217 completed preparation and implementation reviews. Product found no scope/semantic issue. Design actually read four representative 300px/large-font empty-diff captures in build/commit-diff-baseline-captures/ and found no clipping or visual blocker. Twelve captures cover ordinary/merge/root × two locales × two themes; tests also cover unknown and hidden metadata contexts.

Missing-label RED observed twelve missing ordinary/merge/root labels (build/alignment-commit-diff-baseline-red.log); first rendering GREEN 0.400 s. Owned Git root/ordinary/distinct-tree merge/empty commits bind actual ShowAll content to rendered labels, assert first/second parent order, preserve HEAD/refs/raw Index/status/staged/work/untracked and draft/amend. Related tests passed 3.074 s; related race 12.561 s, vet and Windows/Linux amd64 compilation passed before final test-only assertions.

Testing review requested stronger unknown/hidden exclusions and late-selection loading evidence. Added all relevant baseline exclusions, hidden-message assertion and an actual selectCommit asynchronous read regression: queue old completions, select newer commit, deliver newer then older completions. Baseline/loading/patch stay paired with the newer target and drafts remain; a 300px nonempty file header is visible beside the baseline. Final related tests 3.363 s and targeted baseline race 2.241 s passed (build/alignment-commit-diff-baseline-final-related.log). No confirmed unresolved defect from scoped reviews.

Ruling: place the explanation outside the historical patch viewport rather than inside commitDiffView's virtualized content. Reason: preserve existing row heights, scroll offsets and per-file headers. Cost: the historical diff pane uses one extra small text row; narrow large-font bounds and existing viewport tests are included in acceptance. Root text states no parent instead of claiming an explicit empty-tree argument, matching the inspected backend.

Full make check bos2ak80l completed exit 0: ordinary lobedesk tests 132.707 s, race 434.768 s, go vet ./... clean. Complete build/alignment-commit-diff-baseline-full-check.log was read. All 470 inputs frozen in build/alignment-commit-diff-baseline-final-sources.json match with no added, removed or missing files; verification saved in build/alignment-commit-diff-baseline-source-verification.json. Final git diff --check passed. Implementation and automated acceptance are complete. No native runtime or Sublime baseline-label equivalence claimed; no real-project Git writes, commit, installation or publication.
