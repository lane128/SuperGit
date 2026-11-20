# Palette Checkout Handoff Implementation Plan

**Goal:** Expose existing local/cached-remote quick checkout through the generic command palette, preserving original input focus, draft and execution guards.

**Architecture:** Add branch.quickCheckout to the explicit command catalog. Render palette and checkout mutually exclusively through one helper shared by production and integration tests. Give both DialogBase calls the same explicit backdrop key before construction, preserving one modal session and original opener across the handoff. Palette search takes focus once after opener capture. Existing checkout operations remain unchanged.

**Tech Stack:** Go, MyGo/Lobe, existing shortcut metadata and owned Git/UI Tester fixtures.

**Spec:** User-authorized Sublime Merge alignment, preceding product a77e9dd971872581a/design ae32293e88f6375dd/testing ae3682eeb13c4e203 reports in 2026-11-17-sublime-merge-alignment.md. Installed resources establish generic checkout_branch plus CmdB preselection; reference GUI cancellation/focus semantics remain unobserved.

## Global Constraints

- Wait for remote-checkout final full check b8nkpzqwa and verify complete log/source hashes before source/test edits. Existing source is dirty and uncommitted; preserve unrelated work/generated_images.
- One new discoverable entry, same existing branch.quickCheckout identity and CmdB/Ctrl+B action. No new Git behavior, tags, arbitrary-start creation, fetch, settings or shortcut.
- Generic catalog becomes ten unique commands; shortcut aliases remain deduplicated. Field/Local/list/edit actions stay excluded.
- Palette self-modal must not disable the checkout entry. Use palette eligibility for display; close palette before rechecking action Available and opening quick checkout. Retain busy/loading/history reload/chooser/ref/other-dialog/repository guards and root/epoch/generation validation.
- Dirty/conflict/operation/read error can still enter checkout for inspection; checkout rows explain execution restrictions as before.
- Checkout root Escape/backdrop exits to the surviving original input; no return-to-palette stack. Existing target/name Back/Escape semantics remain. Preserve draft/amend and Git/working state on cancellation/rejection.
- Shared explicit overlay key must only be constructed once per view build. Do not insert a blank modal frame or attempt OS focus workarounds.
- Native GUI remains serialized and acceptance uses owned local-only fixtures; no user repository checkout, real remotes, permission bypass, commit, install or release.

## Review Focus

- Body focus survives two picker contents, actual typing proves logical focus after cancellation.
- Retained callbacks after root A→B, load epoch or generation changes cannot open an old checkout or close a new palette.
- Only palette itself is excluded from display eligibility; other dialog/busy states still block execution.
- Shared overlay key never duplicates in a single build; production and integration tests use the same mutual-render helper.
- Exact checkout target validation remains authoritative after entering via the new command.

### Task 1: Catalog and single-session handoff

**Files:** command_palette.go, quick_checkout.go, view.go, i18n.go under internal/lobedesk; introduce command_picker.go if useful for shared mutually exclusive rendering; new palette_checkout_test.go and minimal existing command_palette_test.go count update.

**Interfaces:** Consume appShortcut ID branch.quickCheckout, paletteCommand(id), commandPaletteUnavailable(id), runPaletteCommand(id,generation), quickCheckoutDialog(c), canOpenQuickCheckout()/openQuickCheckout(). Produce commandPickerDialogs(c *ui.Context) for mutually exclusive palette/checkout construction, used by view and integration tests. Extend commandPaletteState with one-shot search focus initialized on open.

- [x] RED: actual Body → both palette aliases → search checkout → Enter/click → checkout search → Escape → Body typing, draft/amend intact; ensure background key isolation, single entry and enabled row while only palette is open. Observe failure before implementation.
- [x] GREEN: add catalog entry, one-time focus, shared key and mutual-render helper. Keep existing post-close revalidation and stale protection; add real root A→B/epoch/generation/busy/dialog cases.
- [x] Owned Git: adapt existing quickCheckoutFixture success/external dirty/tip/worktree rejection tests to start via actual palette entry; verify exact branch/SHA, unchanged rejected baseline and draft.

### Task 2: Acceptance

- [x] Independent scoped review of final handoff implementation; reproduce and fix material findings, respecting maximum three review rounds.
- [x] Relevant UI/checkout regressions and final cross-build; inspect bilingual/narrow layouts if new command row changes rendering.
- [ ] Native handoff acceptance: bounded attempt stopped at natural focus prerequisite before input; remains unverified.
- [x] Full make check with stable final source hashes, complete output read, diff formatting and docs update.

## Progress

Planning only. Three prior roles establish the scope and SDK-based overlay proposal. Proposal requires actual control verification; no claim of runtime correctness. Latest narrow role cycle reports all received; delivered report roles differ from launch labels, so attribution follows contents: product ac8cc2fb154197ee0 confirms existing localized label/ID search and guard copy support the entry; design a79f0d703fadb7fd9 confirms mutually exclusive production rendering, shared explicit key, one-shot palette focus and identical integration-test rendering; testing ad6f7b3afc11b891e locates existing count assertion at command_palette_test.go:242 and owned four-scenario test at quick_checkout_test.go:272. No new blocker or executed acceptance claimed. Parent still owns full-check b8nkpzqwa, regardless of a role's inability to see that task. Implementation now complete; final full-check acceptance pending.

- Prerequisite remote-checkout check b8nkpzqwa passed exit 0. Complete log read; ordinary tests valid cache, race lobedesk 360.739 s, vet clean. All recorded internal/cmd hashes matched and diff formatting passed before handoff edits.
- RED build/alignment-palette-checkout-red.log: missing commandPickerDialogs. Added shared mutual-render helper, explicit common key immediately before each DialogBase, one-shot palette search focus and branch.quickCheckout catalog entry. New command label uses existing shortcut metadata with localized ellipsis; dialog title is unchanged. Catalog test now asserts ten unique commands and includes checkout, still excludes write/direct-edit actions and aliases.
- Actual integration tester uses the same commandPickerDialogs as view.go. Both aliases and Enter/click handoff, single dialog, typing into checkout query, background key isolation, Escape and backdrop cancellation back to actual Body typing, draft/amend, real root A→B/epoch/generation/busy/chooser/ref/dialog guards pass. Raw shortcut Available still blocks while palette is open; entry eligibility ignores only its own modal. Existing post-close availability/action recheck remains intact.
- Owned quickCheckoutFixture through palette covers exact branch/SHA success and external dirty/tip/other-worktree rejection with preserved baseline, feedback and draft/amend. Final related run build/alignment-palette-checkout-green.log passed 17.860 s, including existing palette/local/remote/commit-scoped checkout regressions.
- Independent testing review ad6f7b3afc11b891e read implementation and production/test integration, found no important defect; optional backdrop cancellation coverage added and passed. No second review round or speculative feature added.
- Final Windows/Linux amd64 CGO-disabled compilation passed. Eight full palette and eight checkout-filtered bilingual/theme/size renders and bounds passed in build/palette-checkout-captures. Main inspected narrow English light and wide Chinese dark filtered entry. Compile/layout evidence does not establish native runtime behavior.
- Native ordinary e2e binary build/supergit-lobe-palette-checkout-native and owned-only driver build/.native-focus/palette-checkout.py. Attempt build/palette-checkout-native-n1ht4y3x failed five-second natural native focus prerequisite before draft or palette input; process stopped 0. Complete build/alignment-palette-checkout-native.log read. No workaround or blind rerun; native handoff remains unverified.
- Final make check bp9bney37 passed exit 0: ordinary lobedesk 101.216 s, race lobedesk 363.225 s, vet clean. Complete build/alignment-palette-checkout-check.log read; all internal/cmd hashes matched build/alignment-palette-checkout-final-sources.sha256 (verification output build/alignment-palette-checkout-source-verification.log), and git diff --check passed. No source edits between snapshot and verification. Native acceptance remains separately blocked.
- Ruling: use separate quickCheckoutCommand label for the command ellipsis so the existing checkout dialog title stays appropriate. Shared shortcut metadata still supplies palette/Settings labels. Cost: one additional translation key.
- No commit/install/release/publication, user repository Git writes, external remote or OS permission bypass.
