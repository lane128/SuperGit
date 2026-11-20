# Commit File Patch Export Implementation Plan

**Goal:** Export one historical file's complete diff as an applicable patch through the existing file context menu and a native save dialog.

**Architecture:** A dedicated streaming backend writes raw patch bytes to an io.Writer. The UI captures immutable repository/commit/path identity, selects a destination, then writes an adjacent temporary file and replaces the destination only after successful generation and close. Export is read-only with respect to Git and must not trigger a repository reload.

**Tech Stack:** Go, existing runOutput, MyGo Dialog.Save, Lobe context menu and owned temporary Git fixtures.

## Scope and evidence

Local Sublime Commit File.sublime-menu declares create_patch with from/commit/path arguments. Its runtime commit-range behavior is not verified. Current SuperGit historical file menu has no export action. Product/design/testing roles independently inspected the actual main checkout and found no existing export/save implementation in their bounded reads.

Main confirmed Repository.Show (diff.go:326) uses run, whose stdout limit is 4 MiB (repository.go:17,44); it does not request binary patch payload. Reusing preview text would risk incomplete/non-applicable export. runOutput already streams to an io.Writer with literal pathspecs, repository routing, cancellation and bounded stderr. The SDK provides mygo.Dialog.Save and SaveDialogOptions; chooseRepository demonstrates the existing injectable native-dialog boundary and repositoryChooserOpen guard.

Implementation decision: export the selected commit relative to its first parent, or the empty tree for a root. Export only the selected changed file, including both raw old/new paths for a rename. Do not add arbitrary two-commit selection, multiselect export, patch application, new settings or toolbar. Native Sublime parity remains unclaimed. No commit, installation or publication.

## Task 1: Streaming patch backend

Create internal/gitrepo/commit_file_patch.go and commit_file_patch_test.go.

Interface: WriteCommitFilePatch(ctx context.Context, sha, path string, dst io.Writer) error.

- [x] Write tests first for root/ordinary/merge first-parent export, a rename with Unicode/space/tab raw paths, an unselected changed-file control, cancellation/writer failure and binary output exceeding the preview limit. Run focused tests and observe missing behavior before implementation.
- [ ] Validate SHA/path, resolve the actual changed-file metadata and first parent. Reject paths absent from the selected commit. Stream patch with binary/full-index output, explicit a/ and b/ prefixes, rename handling, no external diff/textconv/color and literal paths. Use runOutput, not run/parseDiff.
- [ ] Apply exported bytes in separate owned repositories at the proper base and compare exact target bytes/rename results. Assert source HEAD/index/work bytes unchanged. No real project or remote mutations.
- [ ] Run backend focused/race checks and inspect complete results.

## Task 2: Historical file-menu save flow

Create internal/lobedesk/commit_file_patch.go and commit_file_patch_test.go. Modify the existing app dialog injection field, historical fileDetailMenu and i18n entries only as needed.

Interfaces: captured target holds repository root, loadSeq, selected SHA and raw changed-file metadata; UI dispatcher validates target before native dialog and again after it returns. Inject save-dialog selection for headless tests, following openDirectoryDialog.

- [ ] Test actual menu activation, destination/cancel/error paths, stale root/generation/commit identity and exact output scope before production implementation. The injected chooser returns only owned temporary paths.
- [ ] Add a clearly singular historical-file export action. Native dialog message identifies the raw path and first-parent/root baseline. Busy/loading/chooser/dialog/commit-file-read guards disable the action.
- [ ] On cancel do nothing. On accepted current target, use the existing cancellable operation lifecycle with an export-specific busy label. Capture repository/commit identity, never read mutable UI fields on the worker.
- [ ] Generate to an adjacent temporary file; close successfully before replacing the destination. Remove only task-created temporary files after errors/cancellation. Existing destination bytes survive failed generation. A native overwrite choice supplies destination authorization; no silent truncation before patch generation succeeds.
- [ ] Queue scoped success/error feedback on main completion without refreshing Git or clearing drafts/selections. Verify source HEAD/index/worktree and UI draft remain unchanged when destination is outside the owned repository.

## Task 3: Review and acceptance

- [ ] Product reviews singular scope/baseline and cancellation. Design uses Emil/Impeccable for menu/message scope without extra UI. Testing independently checks exact patch applicability and stale/save-failure evidence. All role worktrees remain isolated; review actual integration files.
- [ ] Run related tests, targeted race, vet and Windows/Linux compile checks. Native save-dialog interaction remains a separate platform acceptance boundary.
- [ ] Freeze source inputs and package list, run complete make check, read full output and verify source/package/diff fingerprints before claiming completion.

## Progress

Implementation is present in both backend and historical file menu. Initial backend tests failed because streaming export was absent; initial actual-context-menu UI tests failed because the action was absent. Backend and UI patch regressions subsequently passed. Related targeted race passed (gitrepo 4.084 s, lobedesk 12.417 s); Windows/Linux amd64 CGO-disabled test compilation passed. Native save-dialog interaction and cross-platform runtime replacement remain unverified.

Product and design reviewed the current main-checkout save flow/copy without important findings. Independent testing identified possible scope leakage through a renamed file's old path. Main checked actual Git metadata: reusing the old name as a regular file is represented as modified + added; reusing it as a directory produces the real R100 + unrelated child-addition case. The latter regression was observed failing with the rename-only filter removed, then passed with the filter restored. Text modification/deletion, root, first-parent merge, raw rename paths, >4 MiB binary payload and literal pathspec behavior are covered by actual patch application in owned repositories.

UI tests use actual row right-click and menu-item clicks with an injected native save boundary. They cover successful replacement/application, exact committed bytes rather than index/work bytes, source HEAD/index/work preservation, no reload/draft reset, chooser cancellation/error, root/generation/SHA/metadata changes during the chooser and after opening the menu, pre-cancellation, invalid Git object, adjacent-temp creation failure and replacement failure. Failed generation preserves existing destination bytes and removes task-created temporary files. A real filesystem close failure is not injected or claimed as exercised.

Fresh independent testing review (adcd95918163325ad) read actual main-checkout files and found no new deterministic defect. It confirmed the repaired rename-directory regression and existing save/stale-target coverage. It identified an additional evidence boundary: cancellation tests cancel before generation starts; they do not prove the complete UI path when a temporary file already contains partial output. Cancellation after streaming begins, alongside native save/overwrite/focus recovery and platform replacement runtime, remains unverified. Source stays frozen during the full-check retry.

Fresh design review (ac93bd2d61da3fcae) identified a confirmed P2 cross-platform scope disclosure defect: MyGo Windows ShowSaveDialog (internal/windows/dialog.go:110) and Linux ShowSaveDialog (internal/linux/dialog.go:101) never consume SaveDialogOptions.Message. Main independently read both implementations. Thus Windows/Linux users currently cannot see the complete source path, rename identities and first-parent/root baseline in the native save dialog. This is an implementation gap, not merely missing runtime evidence. Resolve before declaring this increment complete: provide an app-owned, fully readable scope summary before choosing a destination on those platforms, preserve the captured target and cancellation/stale guards, and test the actual summary controls. Existing generic confirmDialog limits its message to five rows; do not reuse that truncation for arbitrarily long paths. Keep macOS native scope message. Source remains frozen until the running full check has completed; its eventual green result alone will not resolve this P2.

The Windows SDK additionally returns empty path + nil error for both cancellation and some native Show/GetResult failures. The app cannot distinguish these through the current API; do not claim injected chooser errors prove native failure reporting. Record as a dependency/runtime limitation rather than modifying the SDK cache.

Fresh product investigation (abb87135fc36142b4) recommends historical single-file reverse application as the next candidate after export acceptance. Local Sublime resources declare revert_file/cherry_pick_file, while current app has only whole-commit operations. Exact reference range, destination and auto-commit semantics remain unverified. This recommendation is not approval to execute a destructive operation on a real repository. No reverse-file implementation has started.

The first frozen-source make check (bm4d7mngg, prefix build/alignment-commit-file-patch) failed ordinary tests at TestEveryCopyOfAnActionUsesOneVerb: the new export busy/failure keys were not registered in actionCopyGroups. All 483 source inputs stayed unchanged, eight packages matched and diff formatting passed; race/vet did not run because make stopped. Added an export action group covering menu/title/busy/failure without weakening the checker. Related action-copy/ellipsis/export tests passed 2.929 s.

The frozen-source make check retry completed successfully (bq6mep412, prefix build/alignment-commit-file-patch-final). Main read its complete log, verification JSON and task output: ordinary lobedesk 119.396 s, full race lobedesk 432.545 s, vet clean; all 483 source inputs unchanged, eight-package list matched and diff formatting passed. This accepts the original export implementation, before the cross-platform scope follow-up below. No commit, installation or publication.

## Cross-platform scope follow-up

Added commit_file_patch_scope.go and commit_file_patch_scope_test.go. Historical menu now dispatches through runtime.GOOS: macOS opens its existing native save flow; Windows/Linux first show an app-owned scrollable scope summary with complete rename paths, commit and baseline. The summary reuses confirmation ownership, cancellation and action cleanup, with a dedicated layout rather than the ordinary five-row message. Other confirmation rendering stays unchanged. Continue uses the original captured target and the existing export guard; stale generation/SHA/root cannot open a chooser.

New tests first failed in an isolated Go overlay because the scope capability was absent. Candidate then passed actual cancel/continue controls for windows/linux/darwin platform inputs, stale target changes and complete long rename text. The long-text test now actually scrolls to MaxY and verifies that the footer button bounds do not move, replacing the weaker height-only assertion. Existing menu save fixtures render/continue the summary where applicable.

Independent product/design accepted the bounded flow/layout. Testing suggested an existing confirmation could be relabeled; main verified that commitFilePatchReady rejects pending dialogs before askConfirm, and added TestCommitFilePatchScopeKeepsExistingConfirmation, which passes without production changes. This proposed defect was not confirmed. The original export-return guard remains exercised by its existing tests; no extra review round was needed.

After the 483-input check completed, the candidate was integrated. Related export/i18n tests passed 3.456 s; targeted race including shared confirmation/dialog tests passed 13.599 s. Windows/Linux amd64 CGO-disabled test compilation passed. Native platform interaction remains unverified.

The frozen-source full make check for the integrated scope follow-up completed successfully (bbend1ar3, prefix build/alignment-commit-file-patch-scope). Main read its complete log, verification JSON and task output: ordinary lobedesk 120.491 s, full race lobedesk 429.614 s, vet clean. All 485 source inputs stayed unchanged, eight packages matched and diff formatting passed. This is the full gate for the final production implementation; it supersedes the earlier 483-input implementation gate.

### Additional frozen-source verification

Fresh product/design/testing roles completed a bounded pass. Design proposed four representative capture combinations. Main corrected the suggested 150% scale to the actual supported maximum fontScaleLarge=1.15 (fontscale.go); setUIFontScale rejects larger values. Headless overlay capture test passed 0.532 s, producing seven top/bottom images for English/light 1000×700 standard, Chinese/dark 700×500 standard with long rename, English/dark 480×360 large, and Chinese/light 480×360 large with long rename. Main viewed a representative image from every case. Captures assert applied light/dark, applied font scale, visible nonoverlapping buttons, actual scrolling to MaxY and fixed footer bounds. Images are under build/patch-scope-captures. This is headless appearance evidence, not native platform interaction.

The testing role proposed deterministic partial-write cancellation through an owned controlled Git-process fixture. Main implemented a Darwin-only test in an isolated overlay (build/_keyboard/patch_partial_cancel_test.go; overlay patch-partial-cancel-overlay.json). Metadata commands still invoke real Git; only the final --binary patch process writes a known prefix directly to the adjacent temporary file, signals a FIFO acknowledgement, then blocks. After reading the acknowledgement, the test verifies the actual partial-file bytes before invoking a.opCancel. Completion is synchronized through the existing main-thread callback channel. Three repeated race runs passed in 2.426 s: original destination bytes survived, task temporary file disappeared, busy/opCancel cleared and feedback reported cancellation rather than success/error. No sleeps, polling or production hooks were added.

This closes the controlled-process partial-write cancellation evidence gap. It does not claim a native cancel-button click or cancellation timing in an actual large Git diff process; actual Git patch applicability remains independently covered. These additional tests/captures initially lived only in ignored scratch while the 485-input production gate was frozen. After that gate completed, the cancellation regression was retained as internal/lobedesk/commit_file_patch_cancel_darwin_test.go. It uses syscall.Mkfifo rather than an external mkfifo command, remains Darwin-only, and adds no production hooks. Three integrated race repetitions passed 2.793 s; package-wide vet passed.

The only change after the accepted 485-input full gate is this new test file. The whole ordinary test suite with 486 frozen source inputs completed successfully (bzfr3lehu, prefix build/alignment-commit-file-patch-cancel-final): lobedesk 120.917 s. Main read the complete log, verification JSON and task output: all 486 source inputs unchanged, eight-package list matched and diff formatting passed. Full race was not repeated for this test-only addition: the 485-input whole-race result and the retained test's three targeted race repetitions are separate evidence. This completes automated acceptance for patch export; native save/overwrite/focus recovery and real Windows/Linux runtime remain separate unverified boundaries.
