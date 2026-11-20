# Palette Stash Implementation Plan

> **For agentic workers:** Continue already-loaded executing-plans/TDD/verification guidance. Main owns implementation and tests; independent product/design/testing worktrees review only. No commit, installation or publication.

**Goal:** Make existing tracked-only Stash creation discoverable through the command palette, with its scope explicit and its existing message prompt retained.

**Architecture:** Add one command-only descriptor to commandPaletteTable, not shortcutTable. Guard and recheck before promptSaveStash(false); retain the existing tracked/untracked and single-file implementations. Test actual palette→prompt ownership before changing shared modal infrastructure.

**Tech Stack:** Go, MyGo/Lobe actual UI Tester, owned disposable Git repositories only.

**Spec:** User-authorized ongoing Sublime Merge alignment. Actual role reports identify product aa8154b3caeaaca2e, design aaddc13651fd918ff (Emil/Impeccable) and testing a1febcb84efac96e8 in independent worktrees. Concurrent launch-return labels crossed the received roles; use these report-confirmed roles for continued review. Both product/design shared installed Default.sublime-package Default.sublime-commands:76–77 (Stash…→stash, no args), Main.sublime-menu:80–81 (stash and directed show_command_palette command:stash). Merge OSX keymap:335 declares super+s→stash, :342 super+s→save in merge_mode; Default OSX:16 declares save async=true. Resource presence does not prove runtime precedence or default untracked policy.

## Global Constraints

- Fetch full make check btr3evnb3 passed exit 0 (ordinary lobedesk 117.324 s, race 390.915 s, vet clean). Complete log read, all 453 frozen source inputs matched and formatting clean before Stash source edits.
- Preserve unrelated uncommitted work/generated_images. No commit, real-project stash, remote writes, install, release or permission bypass.
- One canonical command for existing tracked-only creation. No CmdS, Pop/Drop/Restore additions, new option, native-menu framework or layout redesign.
- Explicit bilingual tracked scope with ellipsis because the existing message prompt requires another step. Reuse the existing include-untracked and per-file menu entries unchanged.
- No repo, no tracked changes, conflict/operation, busy/loading/history reload, chooser/ref or other main dialog blocks the new command; actual prompt execution must retain repository identity checks. Avoid treating untracked-only status as saveable tracked changes.
- Existing prompt message trimming/nonempty requirement and failure clearing are retained; no new prompt draft-persistence behavior.
- Creation changes index/work contents by design, but must preserve original staged/work versions in exactly one new stash, prior stash identity and local HEAD/branches. Untracked/ignored files remain for tracked-only mode. Cancellation must preserve everything.
- No complete Sublime Merge parity claim. Native natural-window-focus prerequisite remains unresolved; deterministic UI evidence is distinct.

## Review Focus

- Actual production render order has promptDialog before commandPickerDialogs; verify handoff captures the original opener/selection rather than defunct palette search.
- Existing generic prompt uses width 400; render at 360×360 before assuming narrow-window support. Only change shared modal code if a bounded failure is demonstrated.
- Existing promptSaveStash(false) checks any status entries, so untracked-only may be Git no-op; first reproduce, then choose the smallest guard necessary for the new entry. Do not silently change include-untracked semantics.
- A stale palette/root/epoch/generation callback cannot close a newer session or start a write; prompt acceptance after repository replacement cannot write elsewhere.
- Repeated activation/confirmation must not create multiple stashes; empty/whitespace-only message follows the existing disabled/Enter behavior.

## Task 1: Tracked Stash entry and prompt handoff

**Files:** create palette_stash.go and palette_stash_test.go; extend the command descriptor source/whitelist and i18n labels/reasons; update catalog count assertions (12→13) and documentation. Existing stash.go/menu.go are reused unless a demonstrated blocker requires a scoped repair.

**Interfaces:** canSaveTrackedStashCommand() bool, saveTrackedStashFromCommand(), tracked-stash command descriptor consumed by commandPaletteTable(); existing promptSaveStash(false), askPrompt(), promptDialog() and commandPaletteUnavailable().

- [x] Verify Fetch final acceptance and frozen inputs before source edits.
- [x] Observe missing-command RED; test actual filtered command row and both palette aliases from Summary/Body/history.
- [x] Add one descriptor and necessary bilingual scope label/reasons with registration-time and execution-time guards; no fake key binding.
- [x] Test Enter/click → exactly one prompt, correct prompt input focus, Tab departure, empty/whitespace message inert; Escape/backdrop/Cancel restores surviving original selected text and draft/amend. Use production dialog order and actual selected-text replacement.
- [x] Use an owned repository with staged/unstaged tracked edits plus untracked and ignored files. Cancellation/guard refusal changes nothing; accepted Save makes exactly one stash with original index/work blob OIDs, cleans tracked paths to HEAD and preserves HEAD/branches/untracked/ignored/prior stash. Cover repeated acceptance and stale callbacks.
- [x] Reproduce untracked-only boundary before any shared action repair. Render new entry/prompt at 360×360 en/zh light/dark; investigate actual clipping and lost opener before choosing a bounded fix.
- [x] Run focused/related checks and scoped independent role review; compile Windows/Linux.
- [x] Run full make check, read complete result and verify final consumed hashes/formatting.

## Decisions

Ruling: prefer the proven command-catalog entry over CmdS for this increment. Reason: overlapping resource save declarations do not establish actual GUI priority. Cost: dedicated shortcut parity remains unverified.

Ruling: explicitly name tracked-only scope and reuse the existing message prompt. Reason: reference no-args does not establish untracked default, whereas this existing action has a clear scope. Cost: this is a documented product adaptation, not proof of exact reference Stash flow.

## Status

The renewed three-role cycle used actual product aaba5fd43a4358559, design a9fc5d2cda2ea285a (Emil/Impeccable) and testing a3a143099fa1e28bf, all read-only independent worktrees. Actual roles were confirmed after concurrent launch-return labels crossed. Prior preparation roles and verified installed resource evidence were reused rather than re-investigated.

Owned Go probe reproduced existing SaveStash(false) returning nil with only untracked changes and stash count 0→0, unchanged HEAD/status/untracked bytes. Evidence build/alignment-stash-owned-probe.json; task-only probe archived as build/stash-owned-probe/main.go.txt so go test ./... cannot discover an unfrozen scratch package. The new entry disables this case; existing menu/include-untracked/single-path semantics were not changed.

Observed missing-entry RED (build/alignment-palette-stash-red.log). Added descriptor/whitelist and explicit bilingual tracked-only label, catalog 12→13. Command table moved from palette_fetch.go into command_palette.go and composes the Fetch/Stash descriptors; actual shortcut metadata/Settings remain unchanged. First integrated run then demonstrated lost original selected input in every cancellation case and 400px prompt controls outside narrow gutters (build/alignment-palette-stash-first.log). A busy guard assertion also incorrectly required pre-existing busy=false; corrected the fixture assertion to compare original busy state.

Ruling: add a dedicated renderer only for this command's prompt handoff, selected by prompt.paletteHandoff in the mutually exclusive command picker. Reason: actual tests demonstrated generic Modal would lose opener and clip; sharing commandPickerOverlayKey/DialogBase retains the original session, limits width to min(400, width-32) and focuses once after opener capture. Cost: small duplication of generic prompt UI; original generic/menu prompt paths remain and require regression coverage. Added only two prompt flags and generic renderer skip for this handoff. Actual 36 cancellation combinations (3 openers × 2 palette aliases × Enter/click × Escape/backdrop/Cancel), one prompt, empty input and Tab passed after the correction (1.966 s).

Product identified silent refusal if the repository reloads while the prompt is open. TestPaletteStashChangedSessionFeedback observed RED, then same-root epoch changes report localized state-changed guidance and changed guards report their specific cause; different roots remain silent to avoid polluting another repository. Testing found runtime-guard coverage gaps; added actual post-open busy/loading/conflict/operation/untracked-only acceptance and whitespace-message assertions. Owned saves now compare exact saved index/work blob OIDs rather than trimmed Git show text. Final focused GREEN passed 2.074 s; related tests 17.414 s, targeted race 11.048 s and vet clean (complete build/alignment-palette-stash-related.log read). Windows/Linux compile passed exit 0.

Ruling: explain refusal within the initiating repository while rejecting another root silently. Reason: Save must not disappear without an outcome after a same-repository refresh, but an old callback must not add notices to a different repository. Cost: no recovery of the dismissed prompt message; this preserves existing prompt-draft semantics. Product/test peers were informed after the bounded fix; no broad second review or independent rerun claimed.

Design actually Read all eight build/palette-stash-captures/stash-{entry,prompt}-360x360-{en,zh}-{light,dark}.png images and found no visual blocker. Existing light-theme disabled Save text was noted as faint, deferred with the unchanged shared button style. Product/test implementation reviews and the real-input tests supply different evidence; screenshots alone do not prove focus.

Full make check b1dnknvnu passed exit 0 (ordinary lobedesk 117.133 s, race 399.629 s, vet clean). Complete output read at build/alignment-palette-stash-full-check.log; all 455 frozen sources matched in build/alignment-palette-stash-source-verification.json and git diff --check passed before branch source changes. No native retry, real-project stash, commit, installation or publication.
