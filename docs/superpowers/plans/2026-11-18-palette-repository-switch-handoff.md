# Palette Repository Switch Handoff Implementation Plan

> **For agentic workers:** Apply already-loaded executing-plans, TDD and verification guidance. Main implements; current independent-worktree roles review scoped changes. No commit, installation or release without explicit user authorization.

**Goal:** Make the existing session repository switcher discoverable through one canonical command in the generic command palette.

**Architecture:** Add repo.quickSwitch to the catalog, reuse its existing shortcut execution and failure/reset semantics, and render the target through commandPickerDialogs with the shared commandPickerOverlayKey. As with palette→checkout, keep one modal session and original opener while replacing the palette content.

**Tech Stack:** Go, MyGo/Lobe existing dialog/picker helpers and actual UI Tester.

**Spec:** The user's ongoing instruction to align interactions and capabilities with installed Sublime Merge. Product a6f84787ea5e75666 and design a2ee12398866bce84 independently read Switch Repository…→quick_switch_repository in Main.sublime-menu and Default.sublime-commands. Resources establish command/menu presence only; reference GUI restoration semantics remain unobserved. Testing a4514075a9b17ce83 is investigating the bounded handoff. Role recommendations are advisory; authorization comes from the user's ongoing development request.

## Global Constraints

- Finish prior quick-switch full check bnvg8upkt and verify frozen inputs before changing source.
- Preserve all unrelated uncommitted work and generated_images; no real-project Git writes or real remote operations.
- Exactly one canonical catalog addition: 10→11; quick-switch alias remains only a shortcut, not a duplicate command.
- Reuse session MRU, candidate filtering, missing-target errors, success reset and draft semantics. No new settings, persistence, toolbar or title-menu changes.
- One overlay construction per frame, shared commandPickerOverlayKey, one-shot focus after DialogBase captures opener. Do not reconstruct an opener from the disappearing palette search.
- One Escape/backdrop closes the full session, preserving surviving Summary/Body/history-search text and selection. Reopen starts empty; no back-to-palette step or animation.
- Existing root/loadSeq/generation and busy/loading/history reload/chooser/ref/modal guards apply both when listing and when executing. No-repository command is disabled; current repository with no alternatives can open the existing empty state.
- No commit/install/release, OS permission bypass or claim of full GUI parity. Existing native-focus evidence is distinct from deterministic UI evidence.

## Review Focus

- The standalone quick-switch dialog must move into mutually exclusive rendering without breaking dedicated shortcut invocation or original input restoration.
- A retained palette callback must not act after root/load/generation changes or close a newer session.
- Search focus must not be reapplied every frame; Tab departure needs a direct focus assertion.
- Command execution must close the palette before evaluating the existing switcher modal guard.
- Closing after handoff must restore the original input selection, not focus the defunct palette search; neither opening nor cancelling may switch repositories.

## Task 1: Catalog entry and shared modal handoff

**Files:** modify internal/lobedesk/command_palette.go, command_picker.go, quick_repository_switch.go, view.go and i18n.go; add palette_repository_switch_test.go; update catalog-count tests and docs/shortcuts.md. Existing recent-repository actions remain unchanged.

**Interfaces:** Consume quickRepositorySwitchShortcuts(), openQuickRepositorySwitch(), commandPickerDialogs(c *ui.Context), commandPickerOverlayKey and runPaletteCommand(id string, generation uint64). Reuse the canonical binding's ID, Available and Run; add a command-specific bilingual ellipsis label following quickCheckoutCommand.

- [x] Write actual UI RED for both palette aliases and Enter/click handoff from Summary/Body/history search; assert one dialog, input routing, one Escape/backdrop and original selected-text continuation.
- [x] Write catalog-count and actual handoff RED before implementation. Add no-repo/empty-session/retained-callback and direct Tab-focus regression evidence; existing shared guard tests passed without requiring production repairs.
- [x] Add canonical whitelist entry and necessary label; include stale quick-switch clearing and exclusive target rendering in commandPickerDialogs; add shared key to switcher and remove independent main view render.
- [x] Run focused palette/checkout/repository-switch/modal/input regressions. Reuse prior full-path visual evidence if consumed target UI stays identical; capture added command at narrow bilingual/light/dark sizes.
- [x] Get scoped independent testing/design review. Reproduce material defects before fixing, subsequent review limited to new changes; at most three review rounds.
- [x] Freeze final inputs, run make check, read complete output and verify fingerprints/diff formatting. Cross-compile Windows/Linux. Report native limitations accurately; do not retry an unchanged focus prerequisite blindly.

## Decisions

Ruling: add the palette entry as the next bounded alignment increment, superseding only the earlier dedicated-shortcut-first delivery boundary. The user's ongoing development request covers the existing capability's reference-backed entry. Cost: generic command count and its docs/tests must change together.

Ruling: preserve direct-switch reset/failure semantics and original single-session cancellation. Cost: no draft persistence or reference-GUI equivalence is implied.

## Progress

Prior quick-switch full check bnvg8upkt passed before handoff source changes: complete output read (ordinary lobedesk 106.514 s, race 372.188 s, vet clean), all 447 consumed inputs match and diff formatting clean.

Implemented one canonical catalog entry with bilingual ellipsis label; shared key and exclusive commandPickerDialogs routing now cover the repository switcher. Removed its independent main view rendering. Existing switch/reset/error/data flow is unchanged. Catalog expectation deliberately updates to 11 and excludes alias duplication.

Observed actual missing-entry/count RED in build/alignment-palette-repository-switch-red.log. First integration build exposed a joined-line syntax error in view.go; corrected that line before successful focused GREEN (0.655 s). Related suite passed 24.030 s, including 24 original-input combinations (3 openers × 2 aliases × 2 activation modes × 2 cancellation modes), direct logical text-focus→option-focus Tab assertion, true repository switch success/failure, catalog and guard regressions. Four 360×360 command-entry captures at build/palette-repository-switch-captures cover en/zh and light/dark. Windows/Linux compilation passed.

Current roles' actual mapping, confirmed from their received task reports: product aad41bb484391b3ef, design a8c8454bf89345dcc (Emil/Impeccable applied), testing ad005cee4b311ed0d. Launch-return labels initially mismatched; peer addresses were corrected explicitly. Product found no blocker. Design actually read all four added-entry captures and found no visible issue, reusing prior switcher visual evidence without re-auditing its 24 captures. Testing found no confirmed production defect and identified evidence gaps; main strengthened the existing dedicated shortcut tests to render through commandPickerDialogs, added actual no-repo disabled/empty-state/close assertions, dynamic row-guard assertions, nonempty root supersession and empty reopen through both entry types. Final strengthened related suite passed 23.698 s; complete output read at build/alignment-palette-repository-switch-final-related.log. Initial final make check b26k7tfu7 completed with exit 2: ordinary lobedesk 108.550 s and race 379.054 s passed, then vet rejected the new nonempty-root test's copy of gitrepo.Repository because it contains sync.Mutex. Main read the complete output, replaced that shallow copy with a fresh Repository{Root: ...}, and searched related tests for the same pattern (none found). Focused handoff tests passed 1.938 s and go vet ./... passed after the correction. Only palette_repository_switch_test.go changed from the original 448 frozen inputs. Corrected full make check b47hpf9li passed exit 0: ordinary lobedesk 106.982 s, race 383.543 s and vet clean. Complete output read at build/alignment-palette-repository-corrected-full-check.log; all 448 frozen corrected inputs matched before F8 integration, with evidence at build/alignment-palette-repository-source-verification.json, and git diff --check passed.

No new native attempt: the immediately preceding bounded macOS attempt failed its natural-focus prerequisite before input, with application exit 0. This increment's deterministic interaction evidence does not establish native acceptance or reference GUI focus parity. No OS permission bypass, commit, installation or release.

Ruling: reuse the prior switcher's accepted path/empty-state visual evidence; newly render only the added command entry. Cost: static captures do not prove native focus behavior, which remains explicitly unverified.

Ruling: address scoped review coverage gaps without another broad review, because production logic did not change after the review and only tests were strengthened. Cost: no independent rerun of those added tests; the main session runs and reads the complete relevant suite.
