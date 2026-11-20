# File Discard Shortcut Scope

**Status:** Modified-row Backspace and selected-subset discard confirmation implemented; the late-preparation repair and its stable-source full gate are accepted. Main read the full log, verification JSON and task output for `bmpzbp8g0`: ordinary tests passed (Lobe 152.412 s), race tests passed (Lobe 490.992 s), vet passed, all 535 inputs remained unchanged, 8 packages matched, and check/diff-check exits were 0. The prior historical-file gate is not reused for this increment. No commit, installation or publication.

## Reference evidence

Main read the installed `/Applications/Sublime Merge.app/Contents/MacOS/Packages/Default - Merge.sublime-package`. The Linux, OSX and Windows keymaps declare ordinary `backspace` as `discard_file` with `path: $path`, `is_staged: false` and control `table_of_contents table_of_contents_tree` (command lines 205, 222 and 203 respectively). Their separate Shift+Backspace commands are restricted to `is_diff`. These declarations do not establish native multiselection, confirmation or focus semantics.

## Proposed bounded behavior

Only a focused Modified workspace file row owns the ordinary Backspace command. Exclude Untracked, Staged, historical files, Diff, text inputs and window-level focus; Shift+Backspace is outside this increment. Reuse the existing preparation, confirmation and prepared-content execution safeguards. No new preference or setting.

Following current workspace action conventions, a focused row inside the valid selection targets that collection; a row outside it targets only itself. Each keypress freezes its target snapshot. A later keypress during preparation supersedes the earlier request; changing selection alone never substitutes another target set into the captured request. This is a SuperGit scope decision, not verified reference runtime parity. Reject an entire target collection containing a conflict or submodule, without silently skipping entries. Unrelated protected entries should not enlarge the target scope.

## Confirmed integration constraint

Main verified `workspace_selection.go:203` supplies selection-aware stage targets. However `prepareDiscardTargets` has only single-file and whole-repository modes. Its `batch=false` confirmation names only the first target; `batch=true` requires equality with the whole-repository discard set and validates that broader scope. A selected subset must not be passed through either mode unchanged. Collection-aware confirmation wording and target validation are necessary if the proposed collection behavior is implemented. This is a missing integration capability, not an observed defect in existing callers.

## Required verification

- Actual title/body Backspace keeps text editing and focus, with no confirmation.
- Actual focused-row input owns the target even when another file is previewed. Selection membership follows the explicit scope above.
- Shift+Backspace and excluded surfaces do not trigger the action.
- Preparation and cancellation preserve exact workspace bytes, Index and HEAD in an owned disposable repository.
- Refresh or repository changes invalidate pending preparation and confirmation.
- External workspace or Index edits with unchanged status codes are rejected by prepared-content checks.
- Conflicts, submodules and protected paths prevent execution for the target collection.
- Confirmation must communicate the actual collection and use the existing cancellation/focus conventions. Design review did not finish verifying the dialog implementation: explicitly test initial cancellation focus, Esc with zero writes, Enter following the focused action, cancellation returning to the original row, repeated Backspace during preparation, and mixed staged/unstaged content wording. These are acceptance requirements, not confirmed current defects.

Existing useful fixtures: `file_row_enter_test.go` for real Git and actual row focus; `diff_file_actions_test.go` for `waitDiscardPreparation`, cancellation and stale preparation; `internal/gitrepo/discard_prepared_test.go` for checks before writes. The role reports were source reviews, not executed acceptance tests.

## Implementation and evidence

`file_discard_shortcut.go` registers a row-local key and guards the captured repository/refresh source and current interaction scope. `workspace.go` wires it to existing file rows. `file_actions.go` preserves existing single/all callers and adds selected-subset confirmation and execution validation; `i18n.go` supplies explicit selected-count scope in Chinese and English.

Two actual-key tests first failed because Backspace never opened confirmation. After implementation they passed. Extended `file_discard_shortcut_test.go` cases cover exact selected file effects, preserved HEAD/Index/unselected/untracked bytes, cancellation and draft preservation, excluded surfaces, singleton focus outside the collection, frozen targets despite later selection changes, refresh and external content/Index rejection. A modal test verifies initial Cancel focus, Enter/Esc cancellation and restored row focus. A two-key test holds preparation callbacks and executes them in reverse order after changing bytes, proving only the latest prepared content remains actionable. The text-input check uses a real SDK field; full commit-form and native OS input acceptance remain separate.

Independent scoped implementation review reported no confirmed new defect and identified the missing two-key callback-order test, which main subsequently added. Related tests passed with race detection in **10.399 s**; a strengthened exact confirmation-message assertion will also run in the full gate. No production dialog focus change was necessary.

## Late preparation follow-up

A renewed role review identified preparation-time surface changes as a coverage gap. Main reproduced a real defect with an ignored overlay: after an actual Backspace, holding the prepared callback and then opening the repository chooser, selecting historical content or switching page caused the old request to open confirmation on the new surface. All three cases failed before the fix; no file writes occurred. The initial full check was explicitly stopped and is not accepted as a completed gate.

`workspaceDiscardSurfaceAllowed` now supplies the same interaction guard to both shortcut entry and the selected-subset preparation callback. `file_discard_surface_test.go` permanently covers the three reproduced paths. Related discard/workspace race tests passed in **10.991 s**. A separate owned-Git case in `file_discard_deleted_test.go` verifies a selected deleted tracked file is restored alongside a modified selected file, preserving the Index and an unselected file. That case plus the surface regressions passed with race detection in **2.582 s**. Single-file Chinese/English confirmation now explicitly states unstaged changes and Index preservation.

Replacement full gate prefix: `build/alignment-file-discard-shortcut-surface`, **accepted** after reading the complete outputs for `bmpzbp8g0`; ordinary/race/vet and source/diff verification passed. Other packages reused valid Go test cache entries. Verification of destructive behavior uses owned temporary repositories; native acceptance remains a separate user check before release. No claim of full Sublime runtime parity or Windows/Linux runtime acceptance.
