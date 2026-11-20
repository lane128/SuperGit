# Command Palette Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a searchable keyboard entry for existing navigation and repository-opening actions, preserving drafts, modal ownership and current action guards.

**Architecture:** An explicit whitelist consumes IDs, localized labels, key display and Run functions from shortcutTable. Reuse the existing picker search/cursor/rows and MyGo Modal; track root and load epoch so stale selections cannot act on a different repository. Closing precedes execution and guard revalidation.

**Tech Stack:** Go, MyGo/Lobe, existing picker helpers and UI Tester.

**Spec:** Authorized ongoing Sublime Merge alignment plus product/design/testing role reports of 2026-11-18. Installed Build 2125 maps Cmd/Ctrl+P and Cmd/Ctrl+Shift+P to show_command_palette without arguments. Cmd/Ctrl+B preselects checkout_branch and is outside this increment. Bundle resources do not prove native layout, ordering, matching or focus behavior.

## Global Constraints

- Preserve all unrelated uncommitted work and generated_images. No commit/install/release/publication or new dependencies/configuration.
- Wait for the running F7 full check before production/test edits. Native F7 acceptance remains blocked independently of deterministic results.
- First catalog: repo.open, window.settings, page.history, page.branches, page.stash, page.worktrees, repo.reload, history.head, workspace.focusCommitMessage. No Field/Local/list/default Enter/Diff/file-tab keys, write actions or alias duplication.
- Cmd/Ctrl+P and Cmd/Ctrl+Shift+P share the palette entry and Settings metadata. Existing dialogs, chooser/ref popup, busy/loading/history reload prevent opening. Reopening does not stack or reset a live query.
- Match localized labels and stable command IDs case-insensitively by query words. Preserve fixed catalog order. Disabled actions remain visible with localized reasons; arrows skip them; empty/no-available results cannot execute.
- Palette query starts empty on each open; the actual search field owns arrows/Enter and typing. Esc/backdrop cancels and relies on the SDK's conditional original-focus restoration. Target actions own any explicit focus request after execution.
- Modal participates in hasPendingDialog(toastMain). Suppress background window registrations while open and guard retained callbacks. Existing lower-level operation-completion behavior remains unchanged.
- Root/load-epoch change invalidates the palette. A stale row callback must not execute after close/reopen or repository/context change. Current readiness and command availability are checked again after closing.
- Use existing theme tokens, picker row geometry, active-descendant semantics and scroll reveal. Modal width/height constrained to the current window with 16 px gutters; no new visual system or keyboard-opening animation.
- Owned fixtures only for tests/native acceptance. Cross-builds prove compilation only. No native focus workaround or blind rerun after a failed prerequisite.

## Design decisions

Ruling: use a small explicit catalog backed by existing shortcut metadata, instead of automatic shortcut import or a separate complete command framework. This prevents control-local keys becoming global actions; broader Git actions remain follow-up work.

Ruling: use the existing Modal/picker combination instead of a new overlay implementation. It already owns focus trapping/restoration, search navigation and scrolling; exact reference visuals remain unobserved.

Ruling: show unavailable rows with a reason and skip them during keyboard selection. An empty repository can still offer Open Repository and Settings; repository-dependent pages/navigation remain disabled. A ready repository enables the other catalog actions subject to existing guards and HEAD availability.

## Review Focus

- Opening from real Summary/Body/search must preserve editor contents; cancelling must resume typing in the original surviving field.
- Arrows, Enter, Escape, page shortcuts, commit and batch keys must not reach background controls while the palette is active, including retained window callbacks.
- Root/load epoch changes and close/reopen must invalidate old callbacks; a disabled/removed row must never execute another command as a fallback.
- Execution closes the modal before calling an existing action; commit-message focus must reach the intended real Summary field and ordinary navigation must remain usable.
- Chinese/English labels, no results, disabled-only results, long labels, narrow windows and light/dark tokens must remain usable without overflow.

### Task 1: Searchable guarded palette

**Files:** Create internal/lobedesk/command_palette.go and command_palette_test.go. Modify app.go (state), shortcuts.go (entry/dispatch isolation), dialog_scope.go (modal ownership), view.go (render), i18n.go (palette-only copy). Update docs/shortcuts.md and this plan with actual evidence.

**Interfaces:** Consume shortcutTable() []appShortcut, pickerSearchBar, stepPickerCursor, placePicker, repairPicker, pickerRowBase, pickerRows, pickerReveal, ui.Modal, hasPendingDialog(toastOwner), existing Run actions. Introduce commandPaletteState, openCommandPalette(), commandPaletteDialog(*ui.Context), commandPaletteEntries(), commandPaletteUnavailable(id string) string and runPaletteCommand(id string, generation uint64). Keep localized label/key/action data in the shared table and whitelist only stable IDs.

- [x] RED: actual controls plus shared dispatcher exercise both aliases, filtering/arrows/Enter, disabled/no-results behavior, cancelling to real draft fields, background-key isolation and stale generation/root/epoch callbacks. Run go test ./internal/lobedesk -run '^TestCommandPalette' -count=1 and observe missing implementation failure.
- [x] Implement catalog, lifecycle, guarded dispatch, modal/search/rows, localized copy and sizing using existing visual tokens.
- [x] GREEN: focused palette tests plus existing dialog/shortcut/commit-focus/picker regressions. Verify close-before-action, exact selected action and unchanged drafts/Git state.
- [x] One independent implementation review; reproduce/fix material findings with regression evidence. No unrelated design/config repair.
- [x] Relevant race, Windows/Linux compile and current native attempt with bounded focus prerequisite; inspect rendered light/dark/narrow evidence where available and record any blocked acceptance honestly.
- [x] Run/read full make check with stable production sources, diff formatting, and update shortcut/alignment evidence. Deliver uncommitted result with remaining parity gaps.

## Progress

Planning: all three read-only reports received. Main source inspection confirms existing Modal conditionally restores a surviving opener, while picker provides search-owned arrows and row reveal. F7 full check b7tk489dk passed exit 0 and complete output was read before palette production/test edits (ordinary lobedesk 86.675 s, race 361.430 s, vet clean).

- RED: build/alignment-command-palette-red.log, missing palette implementation. Initial real-control run exposed cancellation losing the opener; SDK Overlay captures opener after children build, so search.Focus moved after that capture. Captured failure in build/alignment-command-palette-focus-red.log; focused suite then passed.
- Ruling: use ui.DialogBase, the same underlying modal primitive, rather than wrapping ui.Modal with another styled panel. Narrow 360 px renders and horizontal bounds reproduced a doubled panel overflowing the window. One constrained panel fixes the cause and retains modal Escape/backdrop/focus behavior. Cost: SuperGit chooses its own existing-token styling, without proving reference visuals.
- Render matrix: eight light/dark × English/Chinese × 360×360/900×640 captures in build/palette-captures. Bounds test checks horizontal control containment and dialog/search vertical containment; scrolling option geometry may extend below its clipped viewport. Representative narrow light and Chinese dark images inspected by main and independent design role. No operational rendering blocker; optional dimmer disabled titles deferred because adjacent reasons already explain availability.
- Independent testing role static review found no blocker; its actual Summary/History-search and backdrop cancellation coverage suggestions were added. Actual Summary, Body and History search preserve drafts and resume typing after cancellation. Target Summary focus is one-shot and Tab escapes. Retained window action isolation is directly tested through dispatcher wrapper, not an SDK captured-callback replay.
- Native attempt build/command-palette-native-mia_c64j misread accessibility active-descendant focus. SDK access.go explicitly reports the selected option; corrected driver to prove actual query typing instead of expecting an AX focused marker on search. No product fix for this script assertion.
- Native build/command-palette-native-i0_ny3rm then proved one Escape closed a repeated-label tooltip instead of the palette. Existing full check bgo53w6lg stopped; no final full-check claim from that run. TestCommandPaletteDelayedEscapeClosesOnce reproduces the issue with an active window, Tab keyboard-focus visibility and delayed frame (build/alignment-command-palette-escape-red.log).
- Ruling: remove only palette row tooltips that repeat the already visible command name. Accessible full labels and disabled reasons remain; no other picker/tooltip behavior changed. The new regression passed with final related race, build/alignment-command-palette-race.log. Initial test without visible keyboard focus did not reproduce; final test uses the SDK's actual focus-visible precondition.
- Final native build/command-palette-native-25zo22ih passed both aliases, actual query updates, background page shortcut isolation, single-Escape Body restoration/typing, target Summary focus and preserved draft/HEAD/refs/Index/status/working bytes. Screenshots and full result log inspected; owned process stopped exit 0. Driver build/.native-focus/command-palette.py; ordinary e2e binary build/supergit-lobe-palette-native. Coverage is macOS English/light 1100×720; no native Windows/Linux, all catalog action handoffs or reference GUI equivalence claim.
- Final Windows/Linux amd64 CGO-disabled builds and diff formatting passed. Full make check bzrrsksc8 passed exit 0: ordinary lobedesk 92.484 s, race 359.438 s, vet clean. Complete build/alignment-command-palette-check.log read; all recorded internal/cmd source hashes matched build/alignment-command-palette-final-sources.sha256 after completion, and diff formatting passed. F7's earlier native focus block and original intermittent New Repository Enter/focus cause remain open.
- No commit/install/release, external publication, real-remote action or durable product-rule recording.
