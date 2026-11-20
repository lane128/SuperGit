# Stage Tracked Changes Implementation Plan

**Goal:** Implement Sublime Merge's stage_all_modified key, Cmd+Shift+A / Ctrl+Shift+A, without staging untracked files.

**Reference:** Installed Build 2125 Default - Merge keymaps OSX:337, Windows:314, Linux:315; command caption Stage All Modified / git add -u. Existing Stage All includes untracked files and stays unchanged.

**Design:** One shortcutTable workspace action, label 暂存已跟踪文件的改动 / Stage tracked changes. Reuse the Unstage All shortcut's visible-workspace/dialog/chooser/navigation guards and batch protection, select only unstaged entries already tracked by the Index, and revalidate before writing. Backend batches git add -u -- <literal paths> under its write lock. No new button, settings or dialogs. Index-added files count as tracked; pure untracked files do not. An unstaged filesystem rename stages the tracked source deletion, leaving its untracked destination out of the Index, matching git add -u.

## Task 1: Implement and verify

- [x] Backend RED test compares tracked staging with a git add -u oracle in committed/unborn repositories. Initial missing-method RED was followed by a behavioral regression check: temporarily using ordinary add made both cases fail with different tree hashes; restoring -u passed. The file-to-directory fixture excludes untracked children and preserves working bytes.
- [x] UI RED tests use real shortcutTable dispatch and real Git, completion refresh, dialog/page guards and retained draft. Initial missing binding/no-operation failures were observed. The independent GIT_INDEX_FILE oracle leaves the tested Index intact, including intent-to-add.
- [x] Implement tracked-only batch mode and backend path method; shared shortcut availability and execution recheck; localize action/busy copy and action-copy coverage. Existing Stage All remains unchanged.
- [x] Focused race passed: gitrepo 1.830 s, lobedesk 3.824 s, including prior Unstage All and batch regressions. Product/design/testing scoped reviews found no blocking defect. Testing found a mistaken test fold key; corrected it to existing filesSectionUnstaged/filesSectionStaged constants, then current StageTracked focused race passed 2.383 s. A later actual commit-field test found the separately documented dispatch defect below, superseding the first full check.
- [x] Windows/Linux amd64 CGO-disabled builds passed; compile coverage only. git diff --check passed. Shortcut documentation updated.
- [x] Final full make check bvnvnybyb passed exit 0; coordinator read build/alignment-stage-tracked-editor-final-check.log: ordinary lobedesk 88.562 s, race lobedesk 364.769 s, vet completed without errors. Earlier brru9ade5 was stopped because the actual commit-field regression below required a production fix; it is not a final pass. Native runtime remains blocked by the separately documented window-focus environment issue.

## Rulings

Continue in the existing feature integration branch, preserving all prior uncommitted work. Review roles remain in independent worktrees and read main's absolute paths. No commit, install or release is authorized. Do not change the existing Stage All semantics or reclassify prior native failures.

Test preparation correction: the backend Git helper returns a trailing newline for write-tree; read-tree needed a trimmed object ID. Correcting that fixture error did not change the production implementation or weaken its tree comparison.

Coverage boundaries: changed/deleted gitlink eligibility is checked through the existing parent-Index policy, with new scope model tests for conflict, dirty-only child and changed-pointer eligibility; this increment does not add a real child-repository write scenario. Actual Summary/Body dispatch is now verified as described below. Product/design/testing reviews were read-only source/test reviews, not independent execution. Native keyboard and Windows/Linux runtime acceptance remain unverified.

## Commit-field dispatch regression and repair

New batch_shortcut_commit_fields_test.go renders actual commitMessageFields and focuses Summary/Body for both stage and unstage. RED reproduced two stage failures while unstage passed. SDK editor_navigation.go:27–61 strips Shift when deciding ownership of Cmd+A, and input.go:976–994 handles editor keys before window shortcuts. The existing window binding alone therefore did not reach the action from either editor.

commit_fields.go now routes exact stageModified/unstageAll keys through the same shortcut table and Available/Run checks in OnTextKey. Summary preserves its existing Enter and paste undo/redo handler; Body uses the narrow batch handler. Returning true ends that key before editor/window fallback, avoiding duplicate execution. Plain Cmd+A is not intercepted. When an action is unavailable, normal editor fallback remains; guards promise no Git write, not unchanged selection.

Actual Summary/Body × stage/unstage GREEN plus related shortcut/editing tests passed race 5.549 s. Added explicit plain Select All/Copy assertions for each actual field, preserving the complete draft with trailing blank lines; current focused race passed 3.732 s. Testing role's second review focused only on handler/dispatch/test changes and found no blocker. Windows/Linux builds passed again after the repair. No production changes followed the new full-check start; the later change only strengthened test assertions.
