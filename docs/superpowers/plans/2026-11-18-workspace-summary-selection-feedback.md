# Summary Selection Feedback

**Goal:** Explicit workspace list selection remains visible when the preview changes to Summary; primary filename weight continues to describe the current file preview.

**Scope:** One row-rendering condition plus a focused actual-tab regression. No selection model, Git action, keyboard binding, toolbar or setting change. No commit, installation or publication.

## Decisions and evidence

The product review confirms the existing rule: selected file membership is independent of active preview tabs. Main retains that rule. Testing initially proposed clearing selection when reopening a file tab, then explicitly withdrew that expectation after reading the new regression; no user approval is inferred from either report.

Design identified that Summary hid both selected background and primary weight while retaining the explicit set. A new owned UI Tester test actually clicks Summary and fails because b.go loses AX Selected even though membership remains. The fix permits explicit current-repository membership to stay selected in Summary, while retaining the old behavior for a legacy preview-only cursor. Filename weight remains guarded by the visible file preview.

The regression then clicks an existing c.go file tab outside the selected set. It verifies c is previewed without joining the set, while b/same selection membership and the anchor survive. It separately verifies legacy preview-only Summary has no selected row. This pins product behavior without claiming native Sublime equivalence.

## Tasks

- [x] Reproduce Summary feedback failure using actual tab clicks (build/alignment-workspace-summary-feedback-red.log).
- [x] Integrate minimal rendering fix only after accepting the prior 478-input full make check.
- [x] Related tests passed 4.990 s; targeted race passed 9.756 s, package-wide vet clean. Final test includes an added legacy cursor assertion; this assertion ran under targeted race.
- [x] Second design/test review found no important implementation or product-rule issue; both explicitly distinguish AX Selected from actual background appearance.
- [x] Six actual large-font Summary captures at 220/360/500 in applied light/dark passed 0.559 s. Capture asserts Summary is active and correct row AX membership. Main viewed 220 dark and 500 light: selected backgrounds remain, and no file has primary-preview weight. Captures are under build/workspace-summary-captures; screenshot evidence does not prove native interaction.
- [x] Complete whole ordinary test suite with frozen source/package verification (task b4zcb5gsi, prefix build/alignment-workspace-summary). Whole go test ./... passed exit 0, lobedesk 123.444 s. Complete log, verification JSON and background output were read: all 479 source inputs unchanged, no added/removed files, eight-package list matches and diff format clean.

## Verification scope

The preceding 478-input make check accepted keyboard continuity (ordinary, full race, vet, source/package/format). This final change only adjusts a rendering condition and adds one test; verification is the whole ordinary suite, targeted UI/tab race checks and package-wide vet. Full race is not repeated for this rendering-only follow-up, and the preceding full race is not presented as covering it. Windows/Linux runtime interaction and exact Sublime gestures remain unverified.
