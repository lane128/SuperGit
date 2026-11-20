# Workspace File Multiselect Implementation Plan

**Goal:** Select files within one workspace group and stage/unstage precisely those files through the row menu or Enter/Space.

**Architecture:** Keep preview identity separate from a repository-scoped selected set and range anchor. Resolve selected raw path/version identities into existing fileActionTarget snapshots; validate every target before one StagePaths/UnstagePaths operation. Existing All controls and single-row icons retain their scopes.

**Tech Stack:** Go, MyGo UI Tester, Lobe, disposable local Git fixtures.

## Decisions and constraints

These are implementation decisions, not verified Sublime gesture semantics. Official release notes establish file multiselect and local resources establish selected-file stage/unstage commands. Product, design and testing reviewed independently in isolated worktrees. Main implements in the existing integration checkout. No commit, installation or publication.

- Normal click replaces selection. Cmd/Ctrl click toggles within one group; another group replaces it. Shift (including Cmd+Shift) replaces selection with the inclusive anchor range within that group.
- Identity is raw path plus staged state. Modified, Untracked and Staged cannot mix. Cmd+A is owned by focused file rows, covers their expanded group including virtualized rows, and leaves text-input selection alone.
- Last added/range endpoint is previewed. Removing the primary previews the first remaining displayed row; removing the last clears preview. Right-click selected preserves the set and previews that row; right-click unselected replaces selection.
- Selected row background indicates membership; bold filename indicates the primary preview. Group count reports selection. No toolbar/settings or additional outline.
- Menu stage/unstage and focused selected-row Enter/Space use the set. Single-row icons and other menu actions remain single-file. All controls remain All. No selected discard.
- Same-repository refresh intersects surviving visible identities, without following a file into another group. Repository switch clears selection. Folding clears multiselection; existing single-preview fold restoration remains compatible.
- All targets must still match repository root, generation and status. Busy/dialog/line-selection guards continue to apply. No silent skipping of protected selected files.

## Review focus

Same path with distinct HEAD/index/work bytes; an unselected tracked and untracked control; stale menu snapshots after root/generation changes; input/dialog keyboard ownership; refresh/fold and primary removal.

## Tasks

- [x] Write real modifier-click → Enter owned-repository regression; observe failure against current single-selection behavior.
- [x] Add focused selection model/lifecycle, row gestures and selection counts. Keep primary preview and selection membership separate.
- [x] Add selected-file stage/unstage snapshot execution and menu/Enter integration; reuse status validation and literal path handling.
- [x] Verify range/select-all/group isolation, exact index/work bytes, primary removal/right-click behavior and stale/protected action guards.
- [x] Independent implementation review, targeted/race/vet and Windows/Linux compile checks.
- [x] Freeze source inputs and package list, run full make check, read complete output and verify fingerprints/diff format. Record actual limitations.

## Progress

Three role reports received. Product chose isolated visible groups, deterministic primary/anchor behavior and refresh intersection. Design recommended explicit selected count and action scope; main uses existing filename weight for primary distinction instead of a new outline. Testing identified meaningful behavioral tests and exact Git-byte controls.

Implemented workspace_selection.go plus row/menu/lifecycle integration and localized counts/scope copy. Original modifier-click/Enter regression failed because same.go retained its old Index version, then passed after implementation. Product review found nonprimary removal unexpectedly changed preview and refresh could activate a file tab from Summary; both were reproduced by actual UI/owned-repository regressions and fixed. Main additionally reproduced active-tab reconciliation choosing an unrelated already-open file after the primary disappeared, then kept the preview inside the surviving set while preserving Summary.

Design review found ambiguous single-file discard/stash labels and missing accessible selection count. A menu regression and eight applied-theme/language/width layout cases failed before fixes. Multiselect menus now explicitly say this file for destructive/stash actions; group descriptions carry total and selected counts, and lists declare Multiselectable. Original no-selection fold hint and read-only row-menu preview behavior remain covered by existing regressions.

Testing's second pass found two evidence gaps, not production defects. Strengthened Shift fixture now contains four staged files with non-HEAD Index content: b/c/same are the actual selected range, z is a meaningful untouched staged control. Protection tests construct the target set after marking a member protected, then prove the normal member is not staged via menu or Enter. Conflict/submodule cases here are explicit status fixtures, not claims of newly exercised real conflict/submodule creation. Additional tests open a real menu before root/generation changes, click a row icon while multiselected, and mutate actual owned Git status before worker revalidation. Exact Index/worktree assertions distinguish selected-only actions from All.

Final related tests passed 11.101 s and targeted race 8.026 s; complete accepted-focused and accepted-race logs were read. Package-wide vet passed earlier in this increment; final full gate includes vet again. Final Windows/Linux amd64 test compilation passed; compilation is not native acceptance. Six actual large-font captures at 220/360/500 in applied light/dark passed 0.399 s, with row AX selection assertions. Main viewed 220 dark and 500 light; design independently viewed 220 dark, 360 light and 500 dark and found no new important issue within those samples. Menu/keyboard/native accessibility behavior is not established by screenshots.

Full make check for this increment completed exit 0 (task bx5z1hzg4): ordinary lobedesk 118.152 s, race 424.476 s, vet clean. Complete log, verification JSON and background output were read. All 477 source inputs and the eight-package set matched, with no changed/added/removed files and clean diff format. Captures used a separate explicit overlay under ignored build/_keyboard, excluded from ordinary Go package discovery. No commit, installation or publication.

A subsequent requested three-role review found a keyboard continuity gap: normal arrow navigation cleared the range anchor, so Shift-click became a single click. Actual UI Tester reproduced the defect despite this successful full gate. A bounded follow-up is recorded in [Workspace Selection Continuity](2026-11-18-workspace-selection-continuity.md); its changes require their own final acceptance and are not covered by the 477-input gate.
