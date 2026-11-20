# Rename Row Direction

**Goal:** Make existing rename rows identify source, destination filename and destination directory in a readable direction.

**Scope:** Shared fileRowPath renderer for commit and workspace rows. No Git command, menu, history range, shortcut or new setting. Ordinary rows retain filename/directory layout; rows below 260 retain filename priority. No commit, installation or publication.

## Evidence and design

Product a608b712675a30838, design a18b2d23e3d836db3 (Emil/Impeccable) and testing ae2f39e97d9a3c0ec independently reviewed current source and actual settled UI Tester captures. Existing wide rows put the new basename first and old-path arrow at the far right, omit the current directory, and make same-name directory moves unclear. Initial unlaid-out images were withdrawn; only settled captures establish the issue.

Ruling: put the bounded historical source before the primary current basename, retain destination directory at the right, and separate the arrow so source ellipsis cannot hide direction. This follows the existing basename-plus-directory convention and preserves filename priority for long paths. Source group max30%, rename directory max20%; ordinary directory max40% remains. Narrow rows still omit auxiliary paths. The full-current-path accessible row label remains; visual sublabels stay hidden from accessibility.

## Tasks

- [x] Confirm four wide light/dark failures and two narrow passing controls against original renderer (build/alignment-rename-row-confirmed-red.log).
- [x] Prototype minimal renderer using explicit Go overlay without changing the running 473-input full gate.
- [x] Review actual 500/360/220 large-font light/dark captures; fix source ellipsis hiding arrow. Final product/design image reviews found no remaining issue in the inspected samples.
- [x] Expand tests to commit/workspace, root/long paths, window widths 259/260 plus 220/360/500, applied theme, stable frames and accessible labels (120 subcases). Correct root locator collision using documented Tester.Find semantics; root row label and drawn name must both exist.
- [x] Finish related/race/vet/platform compilation and scoped test review.
- [x] After prior full acceptance, integrate file_row.go and file_row_rename_test.go, run complete final gate and inspect logs/hash/package/file-set/format.

## Progress

Original four wide-case RED confirmed. First candidate GREEN 0.428 s, related 2.988 s/race 2.473 s/vet clean. Final separate-arrow GREEN/capture 0.459 s. Expanded focused regression passed 0.539 s. Build/rename-row-arrow-captures/ contains final candidate images; design read three representative images and product the 360px image. The expanded test first hit a locator error for root-level names (Find returns same-named row label); full failure log and SDK method were inspected before changing the test. It is not a new production failure.

Prior 473-input full check is now accepted: ordinary lobedesk 134.767 s, race 444.587 s, vet clean, complete log/JSON read, unchanged source/file set, matching eight-package set, diff format passed. It covered the async-creation and real rename-menu tests but not this visual candidate. Final reviewed related tests passed 2.795 s, race 2.559 s, vet and Windows/Linux amd64 compilation passed; complete log read. Final scoped test review found no implementation defect. Review boundaries are explicit: the 120 subcases use window widths near the threshold, while the existing fileRowShowsFolder unit test covers the exact threshold; root-name existence uses row-label plus drawn-name count, not independently measured glyph bounds; frame stability checks filename X/W, not every rectangle. Percentage caps are implementation choices, not separately asserted product requirements.

After the 473-input full gate was accepted, main integrated the reviewed fileRowPath change and file_row_rename_test.go. A byte-for-byte assertion binds the production renderer to the reviewed candidate. Integrated related tests passed 1.310 s; actual-source capture check passed 0.236 s, complete prefix logs read. Main viewed the integrated 360px dark capture; all six integrated PNG hashes match the independently reviewed final candidate images. The production renderer also matches the reviewed candidate byte for byte. Final full make check completed exit 0: ordinary lobedesk 134.353 s, race 443.171 s, go vet ./... clean. Complete build/alignment-rename-row-full-check.log and background verification output were read. All 474 frozen source inputs match with no changed/added/removed/missing files, eight-package list matches (build/alignment-rename-row-source-verification.json), and final git diff --check passed. Implementation and automated acceptance are complete. Platform compilation is not native runtime acceptance, and neither screenshots nor resources establish Sublime parity.
