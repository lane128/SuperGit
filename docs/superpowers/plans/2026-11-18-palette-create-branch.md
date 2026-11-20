# Palette Create Branch Implementation Plan

> **For agentic workers:** Continue already-loaded executing-plans/TDD/verification guidance. Main implements; product/design/testing review in independent read-only worktrees. No commit, installation or publication.

**Goal:** Expose the existing branch-page create-and-switch operation in the command palette, explicitly naming its current-HEAD start and checkout effect.

**Architecture:** One command-only descriptor in commandPaletteTable; capture the full current HEAD SHA before promptNewBranchAt(sha,true). Recheck root/load epoch/UI HEAD/branch and readiness before accepting. Reuse the proven single-session palette prompt mechanism without altering ordinary menu prompts or create-only behavior.

**Tech Stack:** Go, MyGo/Lobe UI Tester; disposable owned local Git fixtures.

**Spec:** Ongoing user-authorized Sublime Merge alignment. Prior product a930ad2a33accf1cf verified Default.sublime-commands:9–10 create_branch with no args; Main.sublime-menu:108–112 and Action.sublime-menu:3–7 direct show_command_palette to create_branch; Commit.sublime-menu:32–36 supplies from:$commit, Branch Folder.sublime-menu:3–7 supplies prefix:$branch, Merge OSX keymap:47–48 routes the same command. No declaration specifies checkout or proves the default start. Existing branches.go passes status.Head to promptNewBranch, which explicitly calls promptNewBranchAt(start,true). This increment preserves that existing operation with explicit wording, not a claim of reference default equivalence.

## Global Constraints

- Stash prerequisite was satisfied before branch edits: b1dnknvnu passed normal tests/race/vet, complete output was read, all 455 frozen source hashes matched and git diff --check passed.
- Preserve unrelated dirty work/generated_images. No real-project branch creation/checkout, commit, push, installation, release or permission bypass.
- Exactly one canonical create-and-switch command; catalog 13→14. No new shortcut, arbitrary start selector, setting, create-only action, native-menu architecture, rename/delete/tag changes.
- Name the scope in both languages with ellipsis. Show captured short SHA as read-only context and use an explicit create-and-switch acceptance label.
- No repo/HEAD, dirty tracked or untracked status, conflicts/operation, busy/loading/history reload, chooser/ref or another main modal blocks entry and acceptance; reuse existing backend clean checks as the final data boundary.
- Capture full SHA, not a symbolic branch name. Subsequent symbolic ref movement cannot silently change the creation target. Root/load epoch/UI HEAD or branch changes invalidate the initiating session.
- Cancellation preserves original Summary/Body/history selection, draft/amend and all Git state. Accepted operation intentionally creates the new ref and switches branch; source commits, index/work files and unrelated refs remain unchanged.
- Existing menu prompt and create-only semantics are unchanged. No reference GUI priority/focus equivalence or cross-platform native acceptance claim.

## Review Focus

- Do not silently expose checkout=true under an ambiguous create-only action label.
- The captured SHA shown in the prompt must be the exact start passed to CreateBranch; source display and execution cannot resolve different symbolic names.
- The Stash prompt renderer is currently named/sited for one handoff. Share only its proven modal/session mechanics; avoid changing Stash content or focus while adding branch-specific source context.
- The existing generic prompt root guard does not establish epoch/A→B→A or once-only acceptance; test the new command's actual callback lifecycle.
- Invalid or existing branch names must surface existing errors without replacing/renaming a branch or mutating files; require no ambiguous success after refusal.

## Task 1: Existing create-and-switch operation in palette

**Files:** create palette_create_branch.go and palette_create_branch_test.go; extend command_palette.go descriptor/whitelist/reasons; share the palette prompt renderer through a focused generic helper if necessary, keeping Stash regressions green; extend promptState only for necessary read-only source context; add bilingual labels and update catalog/documentation expectations.

**Interfaces:** branch creation descriptor consumed by commandPaletteTable; canCreateBranchCommand() bool and createBranchFromCommand(); existing promptNewBranchAt(capturedSHA,true), CreateBranch(), prompt root guard and commandPickerOverlayKey. A shared palette prompt renderer must retain Stash's one-overlay/one-focus behavior.

- [x] Verify completed Stash log, exact frozen source hashes and formatting before source edits.
- [x] Observe missing-entry RED, then actual Summary/Body/history × both aliases × Enter/click handoff; assert explicit source SHA/checkout intent, one prompt and empty/whitespace inert.
- [x] Add the one descriptor/label and guarded wrapper. Capture full HEAD SHA at entry; recheck captured session before acceptance, explain same-root refusal and avoid notices in a different repository.
- [x] Test actual Escape/backdrop/Cancel restores original selected text/draft/amend with zero Git writes, Tab leaves input, and 360×360 bilingual/light/dark form fits.
- [x] Use an owned clean repository to accept via Enter/explicit button: new ref and symbolic HEAD point to requested branch at captured SHA, original branch/commit/index/work/unrelated refs unchanged. Repeat acceptance cannot create an extra ref. Source-symbolic-name movement never changes the captured target.
- [x] Verify dirty tracked/untracked, no HEAD, conflict/operation and runtime changed root/epoch/UI HEAD/branch guards. Invalid/duplicate names use existing error feedback and preserve the owned state.
- [x] Run focused/related checks including Stash/generic prompts; read scoped role review and new renders. Cross-compile Windows/Linux, run full make check, read complete output and verify final consumed inputs/formatting.

## Decisions

Ruling: preserve the branch-page checkout=true action and make it explicit in the command/form. Reason: current behavior is established, while reference no-args does not establish a different default. Cost: reference start/checkout parity remains unproven; existing create-only menu is retained.

Ruling: pin the creation source to the full displayed SHA rather than repeatedly resolve a symbolic name. Reason: the displayed start and created ref must agree. Cost: this guarantees captured target identity, not an assertion that external Git cannot change the current repository between UI frames.

## Status

Implemented one canonical branch.create command, catalog 13→14, with bilingual current-HEAD/create-and-switch labels, captured full SHA and once-only session/readiness checks. Extracted palette_prompt.go from the proven Stash handoff; ordinary menu prompts and backend CreateBranch remain unchanged.

Actual roles product a2328e9bfe226d3d9, design a4c507b9f35b21a9f (Emil/Impeccable), testing aa9a62f05b8b60ccd completed independent read-only preparation and implementation reviews. Product/testing found no clear behavior defect. Testing suggested exact preservation of all pre-existing refs and once-only refusal notices; main strengthened assertions and added an unrelated tag fixture. The pinned scenario now compares the exact external main advance through the full pre-create refs snapshot. These additions passed, without claiming a peer rerun.

Design read eight 360×360 bilingual/light/dark entry/prompt images. English entry truncation is acceptable because the form explicitly states the effect. The reported light ready-button contrast was traced to captures at the disabled-to-ready color transition's initial instant: Tester.Frame does not advance its clock. The narrow test now asserts the action is enabled and advances 150 ms before capture; main inspected the resulting green/white English light button. No production style change was needed. Actual focus/cancellation/Tab evidence comes from UI tests, not screenshots. Existing Stash disabled-style observation remains outside this increment.

Missing-command RED: build/alignment-palette-create-branch-red.log. Initial branch/Stash GREEN: 5.153 s. Capture run: 0.269 s. Related normal 18.478 s and targeted race 19.632 s, vet clean: build/alignment-palette-create-branch-related.log. Final strengthened branch tests and settled captures: 3.850 s, build/alignment-palette-create-branch-review-check.log. Windows/Linux amd64 compilation passed; native runtime remains unverified.

First full make check biz9d85a6 failed TestChineseUsesOneVerbForCreating: the new Chinese command used 新建 while the existing catalog requires 创建. Complete failure output was read; race/vet did not run after this normal-test failure. Changed only the new Chinese command to 从 HEAD 创建并切换分支…; the existing rule plus branch tests/captures passed in 3.770 s (build/alignment-palette-create-branch-copy-check.log). Tests were not weakened. Design independently re-read all four settled prompt captures and withdrew the ready-button contrast finding.

Corrected full make check bmv4njj44 passed with exit 0: normal lobedesk tests 122.132 s, race 407.226 s, vet clean. Complete log build/alignment-palette-create-branch-final-check.log was read. All 458 consumed inputs in build/alignment-palette-create-branch-verified-sources.json matched, with no added/removed source inputs; build/alignment-palette-create-branch-source-verification.json records the result. Final git diff --check passed. This increment's implementation and automated acceptance are complete; reference default start/checkout semantics and native runtime acceptance remain unverified. No commit, installation, publication or real-project branch operation.
