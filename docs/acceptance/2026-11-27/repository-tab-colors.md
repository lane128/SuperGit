# Repository tab colors acceptance

Status: original feature accepted and released as 1.0.2 on 2026-11-27. The later Sublime 1A follow-up was installed locally and accepted, then released as 1.0.3; see [release 1.0.3](release-1.0.3.md).

Source: feature/repository-tab-colors, uncommitted changes on d26d29f. This is a local feature build, not a release. User approved the design with “按这个方案实现”.

## Scope

Approved B solid cards: equal 168–192 DIP widths, 36 DIP height, final 13 DIP / 500 typography in both selection states, fixed blue 2 DIP selected underline, seven coordinated backgrounds, borderless 24 DIP close target and compact color-dot context menu. Typography trials and the final choice are recorded below. Per-canonical-checkout atomic preference files live in repository-colors beside settings; aliases and Git data retain their existing formats.

## Automated evidence

- Storage validation, missing/corrupt values, explicit blue/default reset, inactive assignment, stale root/closed target, busy/loading guards, failure notification and reopen/session persistence are covered in repository_color_test.go.
- Rendered card geometry, stable selection, borderless close and fixed line/background pixels are covered for seven presets × two themes × two scales in repository_tab_style_test.go.
- Existing tab alias, draft protection, keyboard navigation, drag/reorder, CJK/duplicate labels and overflow tests are included in verification.
- A keyboard regression reproduced Escape being consumed by the workbench global shortcut; popup-local Escape now dismisses and restores tab focus.
- Full coverage and vet use ./cmd/... ./internal/... ./tools/... (all tracked source packages). Generated native fixture sources under build/ are excluded without deleting them.

## Verification results

- `go vet ./cmd/... ./internal/... ./tools/...`: PASS.
- One full `go test -cover ./cmd/... ./internal/... ./tools/...` run: all functional tests passed; two new copy guards failed (English “Tab” in Chinese copy, and an unregistered failure-message action). Both were corrected; the targeted copy tests and all repository-tab/color/card regressions then passed in 17.033s. No redundant full rerun after these copy/test-only fixes. Measured coverage: lobedesk 85.9%, gitrepo 81.1%.
- Scoped independent review: PASS, no must-fix findings. Initial isolation prevented the reviewer from running git against the main checkout; a generated tracked patch plus complete inventory closed that coverage gap. New source files and their direct dependencies were inspected separately. Reviewer ran no tests and made no edits.
- `make app && make e2e-build`: PASS, including local bundle code-signature verification.
- `python3 scripts/native-repository-tab-colors.py --out build/repository-tab-colors-20261127-final`: PASS, 8/8 cases. The driver uses disposable repositories and isolated profiles. Each case checks inactive pointer/keyboard recolor, all seven radio dots, save/reset, restart restoration, same-background and gray-selected states, and overflow panning without navigation. Every repository byte fingerprint remained unchanged.
- Native PNG sampling: 24/24 selected-line samples have the exact expected light/dark RGB, across initial, same-background and gray-selected screenshots.
- Standard `build/SuperGit.app/Contents/MacOS/supergit` launched separately with a dedicated Chinese profile at `build/repository-tab-colors-preview/`; process observed running and app log empty. The window remains open for user inspection. Automation screenshots use the matching e2e executable, not the standard bundle.

Matrix: light/dark × 900×640/1320×840 × 100%/115% font scale. All eight palette captures were visually inspected together, with individual dark/narrow/scaled, gray-selected and overflow captures checked at larger size.

Local, gitignored evidence:

- [Native results](../../../build/repository-tab-colors-20261127-final/results.json)
- [Eight-case palette matrix](../../../build/repository-tab-colors-20261127-final/palette-matrix.png)
- [Native line pixel checks](../../../build/repository-tab-colors-20261127-final/native-line-checks.json)
- [Light palette](../../../build/repository-tab-colors-20261127-final/light-900-1/palette.png)
- [Dark palette at 115%](../../../build/repository-tab-colors-20261127-final/dark-900-1.15/palette.png)
- [Gray selected card](../../../build/repository-tab-colors-20261127-final/light-1320-1/gray-selected.png)
- [Dark overflow](../../../build/repository-tab-colors-20261127-final/dark-1320-1.15/overflow.png)

Harness corrections: the first attempt clicked while startup restoration still disabled tabs; readiness now waits for New Tab to become enabled. The second sent menu-open and Escape together before a rendering boundary; those inputs now settle separately. The final eight-case run passed after both corrections.

Native VoiceOver narration, titlebar dragging and the full error/welcome-state matrix were not separately driven in this run. Existing automated welcome/loading/guard/drag tests passed; these remain useful points for the user's final hands-on check.

## Visual and interaction review

| Before | After | Why |
| --- | --- | --- |
| Content-sized tabs with selection padding | Equal-width cards with stationary labels and close targets | Short names remain easy to select; selection cannot move the target |
| Selected outline and tinted surface | Fixed blue bottom line on the assigned background | Repository identity and current selection stay distinct |
| Close inherits normal-button border and sheen | Plain close target with hover/press fill and dashed keyboard focus | Removes the unwanted outline while preserving feedback |
| Standard vertical context actions | Seven color dots in a native horizontal radio group, followed by alias/close | Compact palette with checked state and directional keyboard support |

Native menu convention: arrows move within/between groups; Enter/Space chooses; Escape and Tab dismiss. The HTML preview's Tab traversal is replaced by native menu behavior. No motion was added to frequent keyboard actions.

## T2 follow-up: card depth

User feedback: tabs still looked flat and needed visible depth. Added a light top-edge highlight and shallow bottom shadow to every repository card, with a stronger shadow/highlight for the selected card. Kept the existing palette, selected blue line, geometry and borderless/shadowless close control. Increased the scroll viewport's vertical clearance so it does not cut off the card shadows.

- Regression `TestRepositoryCardsHaveStrongerDepthWhenSelected`: RED on the flat implementation, then GREEN in both themes. Checks actual rendered top highlight, bottom shadow and stronger selected depth.
- `gofmt`, `go vet ./internal/lobedesk`, and `go test ./internal/lobedesk -run 'TestRepositoryCards|TestRepositoryTab' -count=1`: PASS (15.316s).
- Inline diff self-review completed before native verification. T2: skipped a new design document, HTML variants, independent reviewer and full coverage rerun.
- `make app && make e2e-build`: PASS. Re-ran the existing native acceptance harness: 8/8 theme/width/scale combinations PASS.
- Inspected actual light/dark captures: top highlights and bottom shadows are visible; selected depth is stronger without moving labels or close targets. Standard app rebuilt and reopened with the existing isolated Chinese preview profile (PID 21470 observed at launch).
- Current evidence: [Results](../../../build/repository-tab-depth-20261127/results.json), [light cards](../../../build/repository-tab-depth-20261127/light-900-1/cards.png), [dark cards at 115%](../../../build/repository-tab-depth-20261127/dark-900-1.15/cards.png). Earlier screenshots above show the superseded flat treatment.

## T2 follow-up: 14 DIP captions

User asked what 14 DIP would look like and then requested it. Caption type is now 14 DIP in both selection states; weights, icon size, backgrounds, underline and geometry are unchanged. Caption elision measures the rendered size, so longer names elide instead of clipping.

- New regression `TestRepositoryTabCaptionsMeasureTheCurrentFontSize`: picks a name that fits a 13 DIP caption but not the current budget, then asserts the caption is elided and the full name preserved. Passed with the 14 DIP sizing.
- `gofmt`, `go vet ./internal/lobedesk`, and `go test ./internal/lobedesk -run 'TestRepositoryTabCaptionsMeasureTheCurrentFontSize|TestRepositoryTab' -count=1`: PASS (15.339s).
- Inline diff self-review completed before native verification; the `display` import remains in use for the opening label.
- `make app && make e2e-build` then the native matrix: 8/8 PASS. Inspected light and dark captures at 100%/115%: labels remain inside cards, long CJK names elide with the middle ellipsis, underline and close targets stay aligned.
- Standard app rebuilt and reopened with the existing isolated Chinese preview profile (PID 33303 observed at launch). Evidence: [results](../../../build/repository-tab-font14-20261127/results.json), [light at 115%](../../../build/repository-tab-font14-20261127/light-900-1.15/cards.png), [dark at 115%](../../../build/repository-tab-font14-20261127/dark-900-1.15/cards.png).

## T2 follow-up: regular weight

User asked for regular weight. Both selection states now use 14 DIP at weight 400; the fixed blue line and ink carry selection. Caption measurement uses the same regular weight, so names that fit are no longer elided for a bold variant.

- Updated `TestRepositoryTabCaptionsMeasureTheCurrentFontSize` to regular weight and added `TestRepositoryTabCaptionsUseRegularWeight`: a name that fits at 400 but overflows at 600 must keep its full caption. Both passed (0.468s). The first regular-weight fixture failed to find a qualifying name; mixing wide and narrow glyphs corrected the fixture.
- `go vet ./internal/lobedesk` and `git diff --check`: PASS. Inline diff self-review completed before native verification. This T2 adjustment skips a new design document, independent review and full coverage rerun.
- `make app && make e2e-build`: PASS, including bundle signature verification. Native matrix: 8/8 PASS across light/dark, 900/1320 widths and 100%/115% font scales. Inspected light/dark 115% captures: regular labels fit, long names elide, card depth and selected line remain visible.
- Reopened the standard app with the existing isolated Chinese preview profile (PID 40742 at launch). Evidence: [results](../../../build/repository-tab-regular-20261127/results.json), [light at 115%](../../../build/repository-tab-regular-20261127/light-900-1.15/cards.png), [dark at 115%](../../../build/repository-tab-regular-20261127/dark-900-1.15/cards.png).

## T2 follow-up: 13 DIP at weight 500

User asked for 13px/500 after trying the regular-weight build. Caption type is now 13 DIP at weight 500 in both selection states; the fixed blue line and ink still carry selection. Measurement uses the same size and weight.

- Rewrote the caption tests for the new metrics: `TestRepositoryTabCaptionsMeasureTheCurrentFontSize` now picks a name that fits at 13 DIP/500 but not at 14 DIP/400 and asserts it keeps its full caption (RED under the previous measurement, GREEN after); `TestRepositoryTabCaptionsUseNormalWeight` asserts a name that fits at 500 but overflows at 600 stays whole.
- `gofmt`, `go vet ./internal/lobedesk`, `go test ./internal/lobedesk -run 'TestRepositoryTabCaptions|TestRepositoryTab' -count=1`: PASS (15.288s).
- Design doc and preview HTML updated to 13px/500.
- `make app && make e2e-build`: PASS, including bundle signature verification. Native matrix: 8/8 PASS across light/dark, 900/1320 widths and 100%/115% font scales. Inspected light/dark 1320 at 115%: 13px captions sit inside the cards with the existing depth and selected line unchanged.
- Rebuilt app relaunched from `build/SuperGit.app` with the user's normal profile (PID 58161 at launch). Evidence: [results](../../../build/repository-tab-font13-20261127/results.json), [light at 115%](../../../build/repository-tab-font13-20261127/light-1320-1.15/cards.png), [dark at 115%](../../../build/repository-tab-font13-20261127/dark-1320-1.15/cards.png), [light at 900 wide](../../../build/repository-tab-font13-20261127/light-900-1.15/cards.png).
- At the user's request the same build was installed locally: `/Applications/SuperGit.app` now holds this build (binary SHA-256 matches `build/SuperGit.app`, codesign verify PASS, PID 66375 at launch). The previous installed 1.0.1 bundle is preserved at `build/SuperGit-1.0.1-installed-backup.app` for rollback.

## Acceptance, release authorization and rollback

After local installation of the final 13 DIP / 500 build, the user requested: “提交改动&push, 然后 merge feature 分支到 main 中，然后发布新版本&打tag”. This closed Stop 2 and authorized the requested Git and release operations. Feature commit `14f811a` was pushed, fast-forwarded into `main` and tagged `v1.0.2`; the version was packaged and published as the latest release. Full verification and publication results are recorded in [release 1.0.2](release-1.0.2.md).

Reverting the feature commit restores the previous tabs; older versions safely ignore repository-colors files. The prior installed application is backed up as recorded above. Published release/tag deletion or replacement requires separate authorization.

## T2 follow-up (post-1.0.2): 38 DIP edge-to-edge tabs, visible edges, one hover surface

User feedback after trying the installed 1.0.2 build: the tabs were too tall for the window (“Tab 高度相对于整体设计是不是偏高了？”), the whole strip had grown (“现在是整个标签栏都变高了吧”), and vertical padding was not wanted at all (“为什么 Tab 一定要上下留白？重新设计”). A design preview with three strip heights led to “用 38 吧”, followed by “tab 左右边界没区分吗？浅色看不出来间隔”, and finally “为什么悬浮到 tab 上 和 tab 关闭按钮上还是两个效果？不应该是整个 tab 有 overflow 效果吗？感觉很奇怪”. This is post-release work on top of `main` at `9592ce7`; no new release was requested.

What changed:

- Strip height returns to the app's original 38 DIP, including its existing 1px bottom separator. The `max(52, ...)` override is gone; cards fill the 37 DIP above the separator with no vertical padding. Viewport, scroll row and card all use `repositoryTabHeight` (= `titleBarHeight - 1`); the tab target is one DIP shorter for the card's top outline.
- Floating shadows and the inset sheen are removed. Cards use a 6 DIP top-only radius, a 1px outline on the top and side edges tinted from the card's own background, and the fixed 2 DIP selection line now spans the full card width.
- Hover and press are painted once on the card (`row.Hovered()` over its descendants, pressed from either child). The label target and the close target no longer paint their own fills; both keep their dashed keyboard focus rings. Hovering the close button now lights the whole tab exactly like hovering the label.
- Caption measurement subtracts the new 2 DIP side outlines.

Tests:

- New `TestRepositoryTabHoverCoversTheWholeCard` (light and dark): idle card uniform, label hover paints the close area too, close hover produces the same card as label hover. Proven RED against the previous per-button fills (“hovering the label left the close area unpainted”) and GREEN after the change.
- New `TestRepositoryTabsFillTheStripWithVisibleEdges` (light and dark): target sits at y=1 with `repositoryTabHeight-1` height, left/top edges differ from both the card and the strip, and the 6 DIP gap shows the strip ground.
- `TestRepositoryCardsKeepGeometryAndBorderlessClose` updated to the new height; the superseded `TestRepositoryCardsHaveStrongerDepthWhenSelected` was replaced.

Verification:

- `gofmt` clean, `go vet ./internal/lobedesk`: PASS.
- Targeted `go test ./internal/lobedesk -run 'TestRepositoryTab|TestRepositoryCards|TestRepositoryTabs' -count=1`: PASS.
- Full `go test -count=1 ./internal/lobedesk`: PASS (255.680s). After adding dark-theme coverage to the hover and edge tests, both were rerun on the restored implementation: PASS.
- `make app && make e2e-build`: PASS. Native matrix `scripts/native-repository-tab-colors.py`: 8/8 PASS across light/dark, 900/1320 widths and 100%/115% font scales.
- Real-app hover check through the e2e automation: hovering the label and hovering the close button produce the same whole-card tint in light; dark and light captures inspected together. The close button shows no separate block.
- Inline `git diff` self-review completed before acceptance. This T2 change skips a new design document, the independent reviewer and a coverage rerun; the design doc geometry section and the preview HTML were updated.

Local evidence (gitignored): [matrix results](../../../build/repository-tab-38-20261127/results.json), [light cards](../../../build/repository-tab-38-20261127/light-900-1/cards.png), [dark at 115%](../../../build/repository-tab-38-20261127/dark-900-1.15/cards.png), [idle](../../../build/repository-tab-38-hover/idle.png), [label hover](../../../build/repository-tab-38-hover/hover-label.png), [close hover](../../../build/repository-tab-38-hover/hover-close.png), [preview](../../../design/repo-tabs-height-preview.html).

## T2 follow-up: Sublime-inspired 1A integrated chrome

The user rejected the separate colors for strip, toolbar and tabs, requested three integrated alternatives, supplied a Sublime Merge reference, and chose “按照 1A 推荐方案实现”. The approved [1A preview](../../../design/repo-tabs-sublime-v1.html) replaces the selected underline and colored backgrounds with neutral surfaces and colored solid folder icons.

What changed:

- Strip remains 38 DIP. Tabs are 35 DIP, bottom-aligned, with no gap; selected tabs have a 1 DIP top/side edge and no bottom edge.
- Selected tabs and the toolbar share a surface (`#FFFFFF` light / `#1C2128` dark). The selected tab covers the strip divider, joining the toolbar seamlessly.
- Unselected tabs sit on the strip ground (`#E5E9EF` / `#0C1016`). A fine vertical separator distinguishes adjacent unselected tabs without doubling the selected outline.
- Removed selected blue lines, colored tab backgrounds and hover/press background fills. Hover changes foreground emphasis only, including the close control.
- Assigned repository colors render in solid folder icons; text stays neutral at 13 DIP / 500. Default icons stay neutral. Menu swatches match icon hues; identifiers and persistence are unchanged.
- New Tab uses a frameless plus-and-label treatment. Dirty dots, close targets, keyboard focus, aliases, reorder and overflow behavior remain available.

Verification:

- `gofmt` clean; `go vet ./internal/lobedesk` and `go vet ./cmd/... ./internal/... ./tools/...`: PASS.
- `go test -count=1 -v -run 'TestRepositoryTab|TestRepositoryTabs|TestRepositoryColor|TestRepositoryBlank|TestTitleBar|TestTheTitleBar' ./internal/lobedesk`: PASS (23.403s).
- New rendered regressions cover the selected tab/toolbar seam and idle separators; unchanged backgrounds during label/close hover in both selection states; neutral surfaces, colored filled icons, stable geometry and 24×24 DIP close targets across all presets, both themes and 100%/115% scales. Superseded shadow/underline tests were replaced to match the approved design.
- Inline diff self-review completed before native acceptance; no blocking finding. T2 skips a new design pipeline, independent reviewer and redundant full-suite/coverage rerun because shared core behavior is unchanged.
- `make app && make e2e-build`: PASS, including bundle code-signature verification. Bundle metadata remains 1.0.2 (67); this is an uncommitted local build on `9592ce7`, not a new release.
- `python3 scripts/native-repository-tab-colors.py --out build/repository-tab-sublime-1a-20261127`: PASS, 8/8 cases across light/dark × 900/1320 widths × 100%/115% scales. Inactive pointer/keyboard recolor, seven radio dots, save/reset, restart persistence and overflow without navigation passed; all disposable repository byte fingerprints stayed unchanged.
- Visually inspected light 900 at 115%, dark 1320 at 115%, light 1320 overflow and dark 900 at 115% palette captures. The selected surface joins the toolbar, colored icons stay distinct, long names elide and the menu remains within the window. Initial captures can include background diff-loading indicators because the harness waits for tab readiness, not history-content completion.
- Standard `build/SuperGit.app` launched separately against the disposable `light-900-1/api` repository; process PID 20274 observed running. Screenshots come from the matching e2e executable. Standard bundle launch uses the normal application profile; the eight-case harness uses isolated profiles.

Local evidence (gitignored): [results](../../../build/repository-tab-sublime-1a-20261127/results.json), [light at 115%](../../../build/repository-tab-sublime-1a-20261127/light-900-1.15/cards.png), [dark at 115%](../../../build/repository-tab-sublime-1a-20261127/dark-1320-1.15/cards.png), [light overflow](../../../build/repository-tab-sublime-1a-20261127/light-1320-1/overflow.png), [dark palette](../../../build/repository-tab-sublime-1a-20261127/dark-900-1.15/palette.png).

An optional combined screenshot sheet was not generated because the local Python environment lacks Pillow; individual native captures were inspected directly. Native VoiceOver narration and titlebar dragging were not separately exercised. Hover background behavior is verified by rendered unit tests; this matrix does not independently compare native hover pixels.

Rollback: restore the scoped tab/toolbar presentation changes; existing repository-color preference identifiers need no migration or deletion. This follow-up was committed as `6602a3f`, tagged `v1.0.3`, and published; published assets and tags require separate authorization to change.
