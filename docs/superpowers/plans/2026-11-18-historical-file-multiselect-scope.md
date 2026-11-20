# Historical File Multiselect Scope

**Status:** Historical file multiselect implemented and its stable-source whole check accepted. Actual-input regressions, related selection/detail/workspace race tests, full ordinary/race tests and vet pass. Independent implementation review found a cached-patch reset defect; its real-Git reproduction and a separate Summary-refresh regression passed after fixes. Native reference gestures and Windows/Linux runtime remain unverified. No commit, installation or publication.

**Goal:** Bring the historical commit's changed-file list to the existing workspace's selection conventions while preserving one primary difference preview and existing single-file action scope.

## Evidence and role boundaries

A fresh product/design/testing batch used independent worktrees and current main-checkout paths. Design used emil-design-eng and impeccable. Product's bounded lookup did not find the historical entry and therefore did not claim absence. Design and testing located `history.go`'s actual list and click handler; main independently read that code and `app.go:selectCommitFile`.

Current workspace selection has an independent file collection and modifier handling. Historical `commitFileList` instead derives each row's selected state from the active detail tab and calls `selectCommitFile(path)` without modifiers. That helper opens a single file tab. Diff-line selections and multiple open detail tabs do not establish file-list multi-selection.

The earlier official Files-tab multi-select capability evidence remains applicable; native modifier gestures, exact Files-tab ownership and range behavior have not been verified. Adopting current SuperGit workspace gestures is an explicit consistency decision, not observed Sublime runtime parity. A testing report mentioned a legacy `internal/desktop` fixture; it is not used for current Lobe acceptance.

## Actual-input RED

Ignored files `build/_keyboard/historical_file_multiselect_test.go` and `historical-file-multiselect-overlay.json` exercise the actual Lobe list with A/B/C. Ordinary click A then Cmd/Ctrl-click C should retain A and C; ordinary click A then Shift-click C should select A/B/C. In both cases C is the primary preview. Current code fails both cases: A is unselected, and range selection also leaves B unselected. The test ran in **0.479 s** and failed on these exact selection assertions. It is an intentional acceptance prototype for missing behavior, not a newly introduced production regression or a passing feature test.

## Bounded implementation decisions

- Reuse the existing workspace ordinary-click, Cmd/Ctrl-toggle and Shift-range conventions; use exact raw paths and current displayed file order. Keep one primary file for the difference preview.
- Keep collection identity separate from open file tabs and diff-line selections. Bind historical membership and anchor to repository plus selected commit; reconcile against current changed-file identities after refresh and reject old-source callbacks.
- When a primary file is removed from membership, keep an already-selected preview if possible, otherwise select the first remaining file in display order. Empty membership returns to summary rather than presenting an arbitrary selected row.
- Reuse current row background and accessible selected state; mark the list as multiselectable. Preserve existing file-tab opening, scrolling and keyboard focus behavior.
- Existing context actions remain explicitly single-file. Right-click within the collection may retain membership while targeting the clicked file; right-click outside selects that file. Do not silently turn Undo, export or file history into selected-set operations.
- No new setting, layout, shortcut, batch mutation or dependency change in this increment. Historical multi-selection must not affect the separate workspace selection.

## Implementation and verification

- `commit_file_selection.go` owns historical membership, range anchor and callback source validation. State lives in the existing repository/commit detail context; workspace selection remains independent.
- `history.go` passes actual click modifiers, exposes accessible multiselection and separates selected-row backgrounds from the primary preview's text emphasis. Right-click keeps existing single-file action targets.
- `commit_file_multiselect_test.go` and `commit_file_multiselect_scope_test.go` cover ordinary/toggle/range clicks, removal, Summary, refresh, repository/commit transitions, raw rename paths, stale callbacks and actual Copy Path menu payload.
- Independent review found that clearing the last selected file reset cached patches. `commit_file_multiselect_preview_test.go` reproduces the visible loss using an owned Git repository and the actual patch view. Empty membership now clears only preview/line selection and preserves loaded patches.
- A separate actual-input regression reproduced refresh leaving Summary after a selected file disappeared. Reconciliation now preserves Summary and any still-valid independent preview tab.
- Both new regressions passed together in **0.905 s**. The related historical-file, commit-file, detail-tab, Changes and workspace tests passed with `-race` in **55.954 s** after the fixes.
- Final stable-source gate: **accepted**, task `bt7cqatrp`, prefix `build/alignment-historical-file-multiselect`. Main read the complete log, verification JSON and task output after completion: full `go test ./...` passed (Lobe **199.068 s**), full `go test -race ./...` passed (Lobe **555.118 s**), and `go vet ./...` passed. All **531** source inputs remained unchanged, with no additions/removals and **8** matching packages; check and diff-check exit codes were both **0**. Other packages reused Go's valid cached results. The ignored input-edge overlay was verified separately and was not included in this whole gate. The earlier second-parent gate is not reused for these changes.
- Native reference selection and Windows/Linux runtime remain separate acceptance evidence; headless RED/GREEN cannot establish them.

## Follow-up role review

A renewed product/design/testing batch reviewed the current integration files from independent worktrees. Product and testing confirmed no new blocking defect within this increment. Testing identified unexercised paths: reverse Shift with anchor removal, selection and context actions across virtualized rows, and actual keyboard navigation after multiselection. These are coverage boundaries, not confirmed failures. Main subsequently ran `TestHistoricalActualInputEdgesOverlay` using ignored scratch files and a Go overlay: reverse Shift with anchor removal and actual Up-key navigation after multiselection both passed in **0.506 s**. The latter verifies the clicked source row has focus, then checks destination focus, singleton membership and active preview. This is list-level headless evidence, not full-page or native acceptance; the cross-virtualization path remains unverified. The role's own attempted execution was rejected by isolation checks and produced no test result; main's separate run supplies this evidence.

Design applied emil-design-eng and impeccable guidance; its local Impeccable engine was unavailable, so no engine setup or download was performed. Main verified two source observations: historical rows expose membership through `AccessSelected` but no explicit current-preview description, and the shared row renderer hides all directory text below 260 points, allowing same-name files to look alike. Full paths remain in accessible labels/tooltips. Neither actual assistive-technology output nor the visual severity has been tested. These remain follow-up accessibility and shared-row layout candidates; this frozen snapshot is not expanded to change shared row behavior. Sublime native parity for either detail is unverified.
