# Palette History Reference Range Plan

> Continue the session's executing-plans, TDD and verification guidance. Main implements; three roles review in independent worktrees. No commit or publication.

**Goal:** Make the existing history reference-range picker accessible through one explicitly named command-palette entry.

**Architecture:** Add command-only history.chooseRef metadata, no key binding. Reuse the existing reference-search/list renderer inside the command picker's existing DialogBase session; retain the original input opener through the shared overlay key. Keep the header Popover entry and existing apply semantics. Capture repository root/load epoch, reject file-history mode, and discard stale sessions.

**Evidence:** Installed reference declarations toggle_ref_filter/toggle_files_filter do not establish equivalence to this range selector. This is an existing-capability entry adaptation, not a claim of shortcut or runtime parity. Prior product/design/testing investigations are recorded in the alignment ledger. Branch creation prerequisite passed full tests/race/vet and all 458 final input hashes.

## Constraints and decisions

- Preserve unrelated uncommitted work and generated_images. Owned disposable fixtures only; no real-project Git mutation, installation or release.
- Entry text: 选择历史引用范围… / Choose history reference range…; necessary context says it only affects the history list.
- File-history mode disables the entry with an exit-file-history explanation. Busy/read/modal guards are shared with the palette.
- Opening/cancellation preserves page, history scope/search/selection/scroll, commit draft/amend and Git state; confirmation alone uses existing applyHistoryRef.
- Reuse SDK opener/selection restoration before adding manual focus state. A shared renderer avoids independent picker behavior. The header Popover keeps its existing interaction.
- New modal width/list height are bounded by available space. Fixed header dimensions are not a reproduced defect and are outside this increment.
- Full source freeze, complete log review and final hashes required for acceptance. Cross-builds prove compilation, not native runtime.

## Task

- [x] Observe missing-entry RED with actual palette handoff tests.
- [x] Add descriptor, bilingual labels/reasons and scoped state. Catalog 14→15; shortcut registration unchanged.
- [x] Extract reusable reference picker content; retain header behavior and use one DialogBase session for the new entry.
- [x] Verify Summary/Body/history-search × both aliases × Enter/click × Escape/backdrop cancellation and selected-text replacement. Ensure first Enter does not also apply a ref.
- [x] Verify stale root/epoch/file-history guards, busy/read/dialog guards, no-match, error Retry, empty cached refs and actual confirmation with owned Git invariants. An empty cached snapshot is not full unborn native acceptance.
- [x] Render bilingual/light/dark narrow entry/picker captures, read scoped product/design/testing review, run related tests/race/vet and initial Windows/Linux builds. Final guard Windows/Linux amd64 build rerun also passed.
- [x] Freeze final inputs, run full make check, read complete log, verify final hashes and diff formatting; update docs.

## Status

Implementation preparation. Latest role launch labels crossed; actual reports map product a1d6b4dc2005b5590, design ac3898167ee52fbb0 (Emil/Impeccable), testing a1f4a30575a146249. All completed scoped read-only preparation reports. Product emphasizes file-history isolation and cancellation without apply. Testing emphasizes opener restoration and first Enter not passing through. Design emphasizes explicit history-only scope, preserved applied/highlighted distinction, and visible Retry/selection at narrow width. These are acceptance criteria, not implementation authorization from agents.

Implemented history.chooseRef in palette_history_ref.go, sharing historyRefPickerContent with the original header Popover. The new phase uses commandPickerOverlayKey/DialogBase; existing SDK focus/selection restoration suffices, so no manual selection state was added. New modal bounds adapt to the window; header dimensions/interaction remain unchanged. Actual catalog has fifteen canonical entries, no new key bindings.

Observed missing-entry RED, then 24 actual input/alias/activation/cancellation combinations and original header regressions passed in 1.008 s. Owned Enter/click confirmation, captures, catalog and Retry checks passed in 1.375 s. Disposable fixtures contain two main commits, an older feature ref, a saved stash, staged/unstaged tracked content and an untracked file; selecting feature loads its one-commit history while exact HEAD/full refs/raw Index/work/untracked/status snapshot and draft/amend remain unchanged. Empty cached refs still offer All refs; this is not a claim of complete unborn runtime coverage.

Product's implementation review found no confirmed behavior issue. Testing suggested deletion and stale-focus coverage: added deleted filtered candidate rejection plus root/epoch/file-mode stale dismissal with selected-text replacement in the surviving Summary field; passed in 1.214 s. Old internal callback retention is not directly exposed by the UI Tester, so this is current UI event coverage, not an independent stale closure invocation. Shared renderer rechecks current ref availability before applying.

Design actually read four bilingual/light/dark error-state captures; no layout blocker. Main viewed English light error, English dark normal and Chinese light normal images. Raw Git diagnostic localization was discussed and left as a minor future summary improvement; retain current Git original and Retry rather than change the existing diagnostic contract. This does not establish native focus or reference runtime parity.

Main found missing shared modal ownership and reproduced TestPaletteHistoryRefModalOwnership RED. Registered the phase in shared dialog guards and retained-window-shortcut dispatch, then picker/catalog/header tests passed in 6.108 s, including actual page-shortcut rejection, retained callback blocking and ownership release after cancellation. Related pre-fix checks passed normal 16.689 s / targeted race 33.264 s / vet clean; final full check includes the modal fix and all final tests. Final Windows/Linux builds passed.

Full make check bi667xs46 passed exit 0: normal lobedesk tests 121.691 s, race 413.632 s, vet clean. Complete build/alignment-palette-history-ref-full-check.log was read. All 460 inputs in build/alignment-palette-history-ref-final-sources.json matched with no added/removed inputs; build/alignment-palette-history-ref-source-verification.json records the result. Final git diff --check passed. Implementation and automated acceptance for this increment are complete; native runtime and reference filter equivalence remain unverified. No commit, installation, publication, native permission retry or real-project Git mutation.
