# Repository tab colors acceptance

Status: accepted after local installation on 2026-11-27. User explicitly authorized commit, push, merge into main, a new version tag and release.

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
