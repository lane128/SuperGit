# File History Recovery Verification

> Continue loaded testing/verification guidance. Main adds owned regression evidence; three independent worktree roles review. No commit, installation or publication.

**Goal:** Verify existing deleted-path rename tracking and exact fixed-tip recovery without adding UI behavior.

**Scope:** Test-only changes unless an actual failure establishes a production defect. All repositories are disposable owned local fixtures. Existing first-successful-page pinning semantics remain unchanged; no promise that an unaccepted first read preserves a tip.

## Tasks

- [x] Strengthen pause/error UI Retry regressions: capture original tips and SHA sequence, move owned HEAD after interruption, compare every result SHA and tip after completion.
- [x] Add small-page deleted renamed-path test matching fixed-tip git log --follow and excluding unrelated commits.
- [x] Run affected tests; obtain scoped product/design/testing review. Investigate any real failure before changing production.
- [x] Run related/race/vet and full make check against frozen inputs; inspect complete log/hash/file-set/format and document limits.

## Status

Test-only implementation adds a deleted renamed literal path with Unicode and a pathspec-like prefix. Five expected fixed-tip SHA values from independent git log --follow are compared with bounded pages of two; first-page acceptance is followed by recreating the deleted path at a newer HEAD. Old-path ancestors must remain and unrelated commits must be excluded. Existing UI pause/error tests now copy original tips, create a new commit changing the target file, click actual Retry, and compare every SHA and unchanged tips while excluding new HEAD.

Actual product ae0977fa5614dc3f2 and testing ac5197bd5df2a4aa5 reviewed the added assertions without executing tests; design a144e115f372eb7ca previously reviewed existing range wording under Emil/Impeccable guidance and proposed no necessary UI change. Testing suggestions were adopted: assert completion leaves fileHistoryMore false and fail immediately for empty pages with HasMore true. No production edits or established production defect.

Initial logs contradicted a mistakenly reported passing result: backend fixture git rm interpreted :(glob) as pathspec magic; UI tests passed. The report was corrected, rm/add now use --literal-pathspecs. A subsequent backend run exposed an untrimmed rev-parse newline in the explicit revision, now trimmed. These are fixture failures, not production RED evidence. Corrected focused backend test passed 0.634 s. Final related backend/UI tests passed 1.586 s/2.599 s; targeted backend/UI race passed 1.693 s/1.936 s; vet clean. Complete corrected-focused and accepted-related logs were read. Full make check ended exit 2: ordinary lobedesk 136.440 s and race 443.822 s passed, but vet found the subsequent keyboard test temporarily authored under build/ as an independent package (undefined app). Complete log was read. This was introduced by main during the full run; Git-ignored build/ did not exclude that Go package, despite the frozen Git-listed input set. The temporary test has now been moved to internal/lobedesk. Reacceptance completed in the subsequent full check: exit 0, ordinary lobedesk 134.007 s, race 441.496 s, vet clean. Complete build/alignment-new-repository-keyboard-full-check.log was read; all 471 frozen inputs match with no changed/added/removed/missing files, eight-package set matches (build/alignment-new-repository-keyboard-source-verification.json), final git diff --check passed. File-history recovery automated acceptance is complete. The later async-creation and rename-menu tests are now integrated and undergoing a separate combined full gate; this acceptance does not claim those additions were included.
