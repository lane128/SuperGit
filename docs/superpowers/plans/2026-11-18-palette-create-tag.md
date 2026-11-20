# Palette Lightweight Tag Plan

> Continue loaded executing-plans/TDD/verification guidance. Main implements; product/design/testing review independently in worktrees. No commit, installation or publication.

**Goal:** Expose existing local lightweight tag creation through one explicit selected-commit command-palette action.

**Architecture:** A command-only tag.create descriptor captures the full SHA from the visible selected history row. Reuse promptCreateTag and the existing palette-only prompt handoff. Show local/lightweight scope and captured short SHA. Root/load-epoch and readiness checks plus once-only acceptance protect retained callbacks. No automatic HEAD fallback, new shortcut, annotation wizard, target picker, push or delete.

**Evidence:** Installed Default.sublime-commands:136 declares create_tag; Commit.sublime-menu:40 supplies $commit. Reference default target/type remains unknown. Existing SuperGit menu passes the chosen commit SHA to separate lightweight/annotated prompts. This increment is an explicit existing-operation adaptation. Prior history-range full check passed normal/race/vet with all 460 source inputs unchanged.

## Decisions and constraints

- History page plus a selected visible commit row is required; workspace selection, other pages and a filtered-out selection cannot silently choose HEAD. Hiding the details pane alone does not hide the selected row or invalidate its context.
- File-history commits remain eligible when visibly selected. The operation adds a local tag and does not clear history scopes.
- Capture the full SHA shown by the prompt. Another eligible commit becoming selected does not replace that captured target. Repository epoch/root or readiness changes reject acceptance; name/duplicate/target validation remains backend-owned.
- Dirty work/staged/untracked files do not require cleaning for tagging. Preserve HEAD, index/work/stash/old refs and commit drafts; intentionally add only the named lightweight ref.
- Escape/backdrop/Cancel must retain original input selection and draft/amend. Existing menu/annotated prompts stay unchanged.
- Final checks use owned disposable Git repositories only. Never mutate real project tags/remotes or bypass native permissions. Windows/Linux compilation is not native runtime acceptance.

## Task

- [x] Observe missing-command RED using actual palette-to-prompt cancellation tests.
- [x] Add descriptor, context guards, bilingual scope/start labels, captured-target wrapper and canonical catalog 15→16.
- [x] Verify three openers/two aliases/Enter-click/three dismissals, whitespace, no write before confirmation, stale scope and one-consumption guards.
- [x] Real owned Enter/click creation targets old selected commit (not HEAD), remains lightweight, rejects invalid/duplicate names, fixes target across selection change and preserves unrelated Git content.
- [x] Render narrow bilingual/light/dark captures; read scoped product/design/testing implementation reviews. Initial related/race/vet and Windows/Linux builds passed before the cancellation fix.
- [x] Finish the reviewed cancellation fix pass, including reproduced same-class branch/Stash callbacks, related/race/vet. Final Windows/Linux amd64 build rerun passed.
- [x] Freeze source inputs, run/read full make check, verify final hashes and formatting; update docs.

## Status

Implemented palette_create_tag.go plus handoff/scope/owned tests. Actual roles product a3f65d8ed877c38a7, design a7ac01dd68cb8676d (Emil/Impeccable), testing a89e163bf654aae72 completed preparation and scoped implementation reviews. Product found no confirmed issue. Design actually read four 360px bilingual/light/dark prompt images and found no blocker; main viewed English light/Chinese dark prompts and the English light entry. Entry long text truncates normally; form fully states local/lightweight scope and captured target. Screenshots do not prove focus or native parity.

Missing-command RED preceded production. Initial 36 handoff/cancel/whitespace combinations plus catalog passed in 0.817 s. Owned/scoped/narrow tests passed in 3.435 s: Enter/click creation at an older selected commit, another eligible commit selected after opening without changing the captured target, Unicode/hierarchical name, lightweight object type commit, invalid/duplicate failures, exact unrelated refs (including stash), HEAD/raw Index/staged-work-untracked status/bytes and draft/amend preservation. Guard tests allow hidden details/file-history visible commits/dirty state, reject invalid page/selection/read/modal contexts, and test stale callbacks with once-only warning or silent different-root refusal. Hook rejection is not separately exercised by the new UI tests.

Testing review identified cancellation retaining a live callback. Main reproduced an actual owned write after cancelling the tag prompt (build/alignment-palette-create-tag-cancel-red.log). Added a paletteCancel callback to the existing prompt state, invoked on palette cancellation and stale-root prompt disposal, and used it to revoke tag acceptance. Tag cancellation plus shared regressions passed 10.559 s. Same-class investigation then reproduced all six branch/Stash × Escape/backdrop/Cancel cases (build/alignment-palette-prompt-cancel-red.log). Connected those two existing wrappers to the same cancellation hook; no new user-visible function or changed successful Git operation. Final related normal 11.026 s / targeted race 34.775 s / vet clean (build/alignment-palette-prompt-cancel-green.log). Final Windows/Linux builds passed. Reviews were not rerun after this bounded fix pass; the author verified RED→GREEN and the final full suite passed.

Ruling: extend the cancellation hook to existing branch/Stash palette prompts after owned reproductions established the same defect. Reason: the shared lifecycle must revoke those accepted callbacks too. Cost: touches their cancellation state; covered by dedicated six-case owned regression, shared prompt tests and the final passing full suite.

Full make check bn2mvkmf2 completed exit 0: ordinary lobedesk tests 126.121 s, race 423.627 s, and go vet ./... clean. Complete build/alignment-palette-create-tag-full-check.log was read. All 464 inputs in build/alignment-palette-create-tag-final-sources.json match current consumed source hashes, with no added, removed or missing inputs; result saved in build/alignment-palette-create-tag-source-verification.json. Final git diff --check passed. Implementation and automated acceptance are complete; native runtime and reference default target/type remain unverified. No real project Git mutation, commit, installation, native permission retry or publication.
