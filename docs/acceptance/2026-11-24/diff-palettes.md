# Diff palettes acceptance — 2026-11-24

Status: implemented; automated checks and native interactions completed with the limitations below. Installed locally at the user's explicit request. Final human acceptance pending. No commit, push, or release.

Design: [Diff color redesign](../../features/diff-color-redesign.md). T3; the user approved A/B/C with a Settings selector and Delta layered as the default.

## Delivered

- Settings → Appearance → Interface → Diff colors offers Git classic, Delta layered, and Vivid. Chinese labels are Diff 配色、Git 经典、Delta 层次、鲜明红绿.
- Each scheme has explicit light/dark addition and deletion colors. Theme mode selects the matching half; interface colors and status badges retain their existing tokens.
- Missing/unknown saved values fall back to Delta. The existing settings save, failure-warning, reset, and repository-tab preference-copy paths include `diffPalette`.
- Workspace, overview, history, and revision comparison resolve the same semantic colors. Unified/split rows share their renderer. Hunk headers have a blue-gray background and readable text.
- Selection uses an inset outline without replacing the red/green background. Gutter checks remain visible; history gutters now expose the same checked accessibility state. A semantic side line occupies existing gutter space.
- Search uses opaque yellow backgrounds with explicit foreground colors. Only the active occurrence is underlined. No word-level diff emphasis or new Git actions were added.

## Automated verification

- `gofmt` and `git diff --check`: passed.
- `go vet ./internal/... ./cmd/... ./tools/...`: passed.
- One `go test -cover ./internal/... ./cmd/... ./tools/...` run completed. All packages other than `internal/lobedesk` passed. That package reported 85.8% coverage and two failing test names: the search pixel test still expected the old amber color (eight subcases), and the Chinese-term guard had not yet registered the approved Diff/Delta names.
- Updated only the old search color predicate, retaining the long-line extent/visibility assertions; registered Diff/Delta in the existing term list and documented them in `docs/git-terms.md`.
- Final targeted rerun passed:

```sh
go test ./internal/lobedesk -run '^(TestDiffPalette|TestDiffFind|TestDiffMetaText|TestGutterInkClearsDiff|TestChineseStringsOnlyKeepTheListedEnglishTerms|TestEveryKeyUsedByTheCodeExists)' -count=1
```

- The full coverage run was not repeated after those assertion/catalog/documentation corrections, following the repository's targeted-rerun rule. Its retained log is a failed initial run, not a green full run.
- New regressions cover absent/unknown/default values, persistence, reset, tab preference copying, English/Chinese settings selection at 848/560 widths, save failure and recovery, all five interface themes × light/dark × three diff schemes, and selected history-row pixels/accessibility. Add/delete text clears 7:1; gutters and other text clear 4.5:1; selection edges clear 3:1.
- Initial focused tests also caught insufficient dark hunk gutter contrast and a test fixture using `Selected` instead of the toolkit's button `Checked` semantics. The local diff gutter resolver now adjusts contrast in either mode; the test applies the actual app theme.
- Explicit package roots exclude generated fixture repositories under `build/`. No SDK/race `make check` matrix was run: this change adds a preference and local rendering, without cross-package architecture, low-level, or build-system modifications.

## Review

A scoped read-only independent reviewer inspected the new resolver/tests and directly related settings, tab, workspace, history, overview, revision-comparison, split-row, and search paths. The old search pixel expectation was the one concrete finding; the reviewer verified its correction. No unresolved product-code findings were reported in the inspected scope.

Isolation guards prevented source-tree Git diff operations from the review worktree. The reviewer compared permitted source and baseline Reads; this was not a complete independent diff inventory. The main session checked the actual rendering/persistence/wiring diffs and final terminology/test follow-ups inline before acceptance. The reviewer did not run tests or modify files.

## Native verification

`make app e2e-build` passed, including ad-hoc codesign verification. Standard bundle: `build/SuperGit.app`, version 0.1.0 (47), based on `d71b934` plus this working tree. Executable SHA-256:

```text
367d0ffed479512f36324586e308fd56b34e97103047511ac1a303b6dddeb316
```

The standard bundle executable was launched with an isolated HOME/repository. CoreGraphics confirmed a visible 1320×840 native window. The process was stopped after that check; the installed application was not replaced.

Detailed interaction/screenshots used the same product source with the e2e automation plugin:

```sh
python3 -I scripts/native-diff-palettes.py
python3 -I scripts/native-diff-palettes.py --width 900
```

- 12 palette/mode/size cases passed scripted assertions: three schemes × light/dark × two configurations.
- Wide configuration: SuperGit interface palette, 1320×840 main window, 848×680 Settings, font scale 1.0.
- Narrow configuration: Catppuccin Macchiato interface palette, 900×840 main window, restored 560×680 Settings, font scale 1.15.
- Both configurations exercised default Delta, choosing each palette, saving, working overview, a file tab, selected addition, ordinary/active search, and history. Restart retained Vivid in all four mode/size runs.
- The narrow run additionally toggled history split/unified mode and searched the tail of a long line. At this detail-pane width the opposite split column requires horizontal scrolling; the captured split view shows the left column.
- HEAD, index, working diff, and fixture bytes were identical before/after every run. No staging/discard/write action was invoked through the UI.
- Visually inspected default light Settings, light Git classic history, dark Delta selected/search, dark Vivid history, narrow dark Settings, narrow light split view, and narrow dark long-line search. Palette fills are distinct; selected rows retain their semantics; search has legible explicit foregrounds; the settings selector stays within the narrow form.

Automation-script corrections were limited to the driver: selected menu items include a trailing check mark in their accessible label; Settings must close before native main-window keyboard interactions; closing with `--settle` attempts to snapshot a destroyed window, so closure is checked via the window list; the line-selection bar rebuild changes row identities, so the script focuses the visible code paragraph before scoped Find. No product code changed for these driver issues.

## Evidence

All logs/images are local gitignored artifacts.

| Evidence | Path |
| --- | --- |
| Initial full coverage run | `build/diff-palette-tests.log` |
| Final targeted regressions | `build/diff-palette-targeted.log` |
| Build/signature output | `build/diff-palette-build.log` |
| Wide native matrix | `build/diff-palettes-4i33kdh0/` |
| Narrow native matrix | `build/diff-palettes-hcjqpnub/` |
| Native run summaries | `build/diff-palette-native.log`, `build/diff-palette-native-narrow.log` |
| Standard bundle visible-window probe | `build/diff-palette-standard-0s5pctmi/window.json` |

## Installed build

At the user's explicit request ("安装到本机看看效果") after the checks above, the previous installed bundle (build 45, executable `7939e806…`) was quit normally, backed up to `build/SuperGit.app-before-diff-palettes-20261124-144458`, and replaced with `ditto` from the verified `build/SuperGit.app` (build 47). The backup and the installed copy both pass `codesign --verify`; installed and source executables share SHA-256 `367d0ffe…`, and the packaged font license is byte-identical. The installed app was launched with `open -n` (PID 77309) and CoreGraphics confirmed a visible 1320×840 window; it was left running for the user. Window evidence: `build/installed-diff-palettes-window.json`.

## Known limitations and human acceptance

- In the 900-wide, 1.15-scale history view, auto-reveal of the long-line `END` search match leaves approximately 6 DIP of the paragraph's tail beyond the right viewport edge. The new semantic background covers the visible line; automatic search positioning is not fully edge-safe. This is in the unchanged reveal/measurement path and was not expanded into a positioning fix in this palette task. Evidence: `dark-900/vivid-long-search.{png,txt}` under the narrow matrix. This limitation must not be described as a fully visible long-line match.
- Standard-app AX/screen-capture permissions were unavailable. Standard bundle evidence confirms a visible window; detailed controls/screenshots belong to the e2e executable.
- Revision comparison, restore defaults, failed save, tab propagation, and Chinese controls have automated Go coverage; this native matrix did not separately drive all of them. No Windows/Linux rendering check or Git-write UI acceptance was performed.
- Human visual acceptance remains pending under feature-flow Phase 3 Stop 2. No commit/push/install is authorized by this record.

Rollback after a future commit should revert only the palette implementation/tests and related documentation. Existing saved `diffPalette` can remain as an ignored field in older versions; do not delete user preferences or repository data.
