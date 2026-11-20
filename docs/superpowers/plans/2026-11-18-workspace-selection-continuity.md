# Workspace Selection Continuity

**Goal:** Ordinary workspace keyboard navigation establishes the same single-selection/range anchor as a normal file click, while busy navigation preserves both preview and keyboard focus.

**Scope:** Two workspace navigation entry points and one early busy guard. Commit-file navigation keeps its existing path. No new settings, toolbar, selected discard, commit, installation or publication. These are implementation continuity decisions, not verified Sublime gesture specifications.

## Evidence and tasks

- [x] Fresh product/design/testing roles reviewed the actual integration checkout from isolated worktrees. Product identified arrow-to-Shift-click losing the range anchor by tracing selectFile; testing identified fold/input ownership gaps; design found no important issue in the three specified actual screenshot samples.
- [x] Reproduce actual click b.go → KeyDown to c.go → Shift click same.go. Original production selects only same.go, confirming the anchor discontinuity (build/alignment-workspace-continuity-probe.log). Input Enter/Space/Cmd+A controls passed in the same probe.
- [x] Prototype only workspace navigation calls in detail_focus.go and keyboard.go via explicit overlay; use selectWorkspaceFile(..., 0, false) to establish the current single selection and anchor. Related candidate tests passed 3.835 s, race 5.841 s; complete logs read.
- [x] Independent second review found no blocking issue in the two-line candidate, but suggested a busy focus boundary. Actual busy KeyDown reproduced focus moving away from the unchanged preview. Add an early workspace-only busy guard before navigation can queue focus. Candidate continuity tests passed 1.250 s.
- [x] Integrate both navigation calls, the busy guard and workspace_selection_continuity_test.go after accepting the prior 477-input full gate. Integrated related tests passed 3.786 s; targeted race passed 6.180 s, package-wide vet was clean, and Windows/Linux amd64 compilation passed. Complete integrated and race logs were read (build/alignment-workspace-continuity-integrated.log and -integrated-race.log); the vet log is empty with successful command exit.
- [x] Complete final frozen-source make check, read complete output and verify source/package/diff fingerprints.

## Verification boundaries

The new tests exercise actual row KeyDown followed by Shift-click, the Changes window-level moveSelection callback followed by actual Shift-click, and actual text-input Enter/Space/Cmd+A with exact Index/worktree controls. The window fallback is a direct production callback test, not a claim of actual global key dispatch in a native app.

Single/multi fold compatibility is tested through foldFilesSection/expandFilesSection directly. It asserts restoration of the original single preview without reviving the previous multi-set/anchor. Header click dispatch and settings persistence are deliberately outside this test; it does not write user settings. Busy focus uses actual keyboard input, settled frames and the accessibility focus state.

The prior successful 477-input make check covered the initial multiselect implementation, not these follow-up changes. Final full check task bnaucuwu3 completed exit 0: ordinary lobedesk 117.448 s, race 441.391 s, vet clean. Complete log, verification JSON and background output were read. All 478 source inputs remained unchanged with no added/removed files, eight-package set matched and diff format was clean (build/alignment-workspace-continuity-source-verification.json). Keyboard continuity automated acceptance is complete. Compilation is not native Windows/Linux acceptance. Native parity with Sublime remains unclaimed. A later Summary selection-feedback fix has separate verification in [Summary Selection Feedback](2026-11-18-workspace-summary-selection-feedback.md).
