# Palette Clone Implementation Plan

> Continue loaded executing-plans/TDD/verification guidance. Main implements; three independent worktree roles review. No commit, installation or publication.

**Goal:** Make the existing Clone form reachable through one command-only repo.clone entry, including an empty workspace.

**Architecture:** Extend the canonical command catalog and no-repository exemption. Reuse Clone validation and operation handling. If actual handoff tests require it, give palette-originated Clone the shared command-picker DialogBase session while ordinary Clone keeps its renderer. Preserve original focus/selection and drafts on cancellation.

**Evidence:** Installed Sublime Merge Main.sublime-menu declares display_clone_page. Palette equivalence and native focus semantics remain unknown. Prior tag full check passed normal/race/vet against 464 unchanged source inputs. Prior role investigations are in the alignment ledger.

## Constraints and review focus

- No new shortcut, Clone backend behavior, New Repository bundle, directory browser or settings.
- No network or real-project Git writes; use disposable owned local fixtures only when needed.
- No-repository and existing-repository activation must both work; busy/loading/modal/stale guards remain effective.
- One dialog, actual source-input typing, no Enter-through submission; cancelled handoff restores original editor selection and draft/amend.
- Repeated aliases cannot reopen/reset Clone. Narrow English/Chinese light/dark controls must remain usable.
- Existing running-operation cancellation and successful-open behavior must remain covered by existing regression tests; new entry tests do not prove real transfer cancellation.

## Tasks

- [x] Write and observe missing-entry RED for actual aliases/activation/cancellation.
- [x] Register command and implement minimal modal/focus handoff required by actual test evidence.
- [x] Verify guards, selected-text restoration, repeated activation and narrow bilingual/theme bounds.
- [x] Read independent product/design/testing reviews and address confirmed issues with RED→GREEN.
- [x] Run related/race/vet and Windows/Linux compilation; freeze inputs and run complete make check, inspect full log/hash/file-set/format results.
- [x] Update shortcuts, e2e and alignment records with actual evidence and limits.

## Status

Implemented repo.clone without a new shortcut, catalog 16→17 and no-repository exemption. Actual role mapping from reports: product a151e662de69dc6ec, design a367c1ad07ae958cc (Emil/Impeccable), testing a11c92f88fc20a086. Independent preparation and implementation reviews completed. Product found no confirmed issue. Testing confirmed meaningful selected-text restoration assertions and requested commit-shortcut isolation/error-state bounds; both were added. Reading-budget remarks do not impose a product source-line limit.

Missing-entry RED is in build/alignment-palette-clone-red.log. A minimal direct existing-Modal handoff then failed actual original-editor selection restoration (build/alignment-palette-clone-handoff-red.log). Palette-originated Clone now retains the shared command-picker DialogBase key and captures opener before initial source focus; ordinary menu/empty-state Clone still uses its Modal, with shared form content and existing backend/operation behavior. The 24 no-repository/existing-repository × two aliases × Enter/click × three dismissals passed in 4.604 s, including source typing, repeated aliases and actual X replacement of the original selected Body text.

Owned local fixtures cover successful actual Clone and exact base.txt tree, existing-target sentinel preservation, and queued-success cancellation through the palette form. Guards cover busy/loading/history/chooser/reference/prompt/Clone modal and root/epoch/generation changes. Four 360×360 en/zh light/dark captures have bounds and actual source/path typing assertions. Related initial tests passed 7.997 s; race 21.454 s and vet clean.

An added narrow-error regression observed Cancel/Clone below the bottom gutter (build/alignment-palette-clone-error-layout.log). Palette-only error scroll height now adapts to available height; ordinary Clone keeps its 100px error region. Error bounds and added Cmd+Enter isolation passed with final related tests in 13.620 s (build/alignment-palette-clone-review-green.log). Design initially reported a pale enabled button from captures taken at the disabled→ready transition start. Captures now assert enabled state and advance the headless clock 150ms after filling fields. Design reread all four updated images and withdrew that finding; no production button style change.

Ruling: use a shared palette session for Clone and constrain its error area in narrow windows, after actual focus-loss and offscreen-button RED cases. Reason: the new entry must retain editor selection and accessible cancellation. Cost: palette Clone has a distinct container/error height, so existing Clone tests plus shared-picker tests and full suite are required.

Final targeted race 23.450 s, vet clean and Windows/Linux amd64 compilation passed (bx6iec9zo exit 0; complete build/alignment-palette-clone-final-related.log read). Full make check bzistppsr completed exit 0: ordinary lobedesk tests 133.887 s, race 436.683 s, and go vet ./... clean. Complete build/alignment-palette-clone-full-check.log was read. All 467 inputs frozen in build/alignment-palette-clone-final-sources.json match, with no added, removed or missing inputs; verification is saved in build/alignment-palette-clone-source-verification.json. Final git diff --check passed. Implementation and automated acceptance are complete. Tests do not establish real in-transfer cancellation, native runtime acceptance or Sublime palette equivalence. No real-project Git writes, commit, installation or publication.
