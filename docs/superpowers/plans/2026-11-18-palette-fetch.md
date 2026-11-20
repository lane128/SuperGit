# Palette Fetch Implementation Plan

> **For agentic workers:** Continue already-loaded executing-plans/TDD/verification guidance. Main implements on the existing feature checkout; product/design/testing review read-only in independent worktrees. No commit, installation or release.

**Goal:** Expose the existing ordinary fetch-all action through one searchable command, without inventing a Fetch shortcut.

**Architecture:** Keep shortcutTable exclusively for actual bindings. Add a palette command-table helper combining its metadata with one command-only appShortcut descriptor, used for listing and execution. Reuse fetchAll/startFetch unchanged through a guarded wrapper. Add a no-remote unavailable reason to the palette.

**Tech Stack:** Go, MyGo/Lobe actual UI Tester, disposable local file-transport Git fixtures.

**Spec:** User-authorized ongoing Sublime Merge alignment. Product a141e43266569d659 read Default.sublime-package Default.sublime-commands:54–57 (Fetch…→fetch, no args); Remote.sublime-menu:3–8 directs a palette to fetch with a named remote. F6 is spelling-check in Default, and Default - Merge has no F6/fetch mapping. These declarations do not establish all-remotes semantics or GUI execution. Reusing SuperGit's existing all-remotes action is a bounded adaptation. Design ab75020fcaaf268e1 applied Emil/Impeccable; testing a424893a683dd047e supplied acceptance boundaries.

## Global Constraints

- Prior F8 full check b28m1a8c6 passed exit 0 (ordinary 109.990 s, race 382.611 s, vet clean). Complete log read, all 450 frozen source hashes matched and diff formatting passed before implementation.
- Preserve unrelated dirty work/generated_images. No commit, push, install, publication, real remotes or OS permission bypass.
- Exactly one canonical command repo.fetch; catalog 11→12. No F6 binding, settings shortcut row, prune command, remote selector or layout change.
- Label the immediate action Fetch all remotes / 获取全部远端, without ellipsis; the target scope must be explicit. Do not imply a target-selection step or complete reference parity.
- No repo, no remotes, busy/loading/history reload, chooser/ref picker or another main modal blocks execution; palette itself is ignored only while evaluating its row. Recheck normal guards after closing.
- Preserve original surviving input and selection, draft/amend; no forced focus on completion. Existing fetching/progress/cancel/success/error feedback remains unchanged.
- Fetch may update remote-tracking refs, FETCH_HEAD and objects; local branches/HEAD/raw Index/staged/unstaged/untracked bytes remain unchanged. Ordinary Fetch does not prune by request, but existing Git configuration still applies.

## Review Focus

- A command without a key must never register a zero/unknown shortcut or show an empty Settings shortcut row.
- No-remote reason must appear in the actual disabled row; retained callbacks cannot bypass root/epoch/generation or modal guards.
- Opening/cancelling the palette writes no Git; actual Enter/click closes the palette before Fetch starts, and duplicate invocation must not restart it.
- Fetch reload must preserve surviving original editor selection, draft and amend; actual replacement with X proves selection, not just appended typing.
- Multiple remotes and failure must retain the existing sequential fetch semantics; no new prune policy or claims of atomic rollback across remotes.

## Task 1: Existing Fetch in the palette

**Files:** command_palette.go (command table, whitelist, no-remote reason), palette_fetch.go (descriptor and guarded wrapper), i18n.go (explicit bilingual label), palette_fetch_test.go; update count assertions in command_palette_test.go/palette_repository_switch_test.go and docs/shortcuts.md/e2e-automation.md.

**Interfaces:** commandPaletteTable() []appShortcut combines shortcutTable() with one descriptor. canFetchCommand() bool and fetchFromCommand() wrap existing fetchAll(). commandPaletteEntries() and runPaletteCommand() consume the same descriptor source. Reuse existing commandPaletteUnavailable and pending-dialog ownership.

- [x] Observe missing-command catalog RED before implementation; verify actual filtered row and no synthetic F6/empty-key binding after implementation.
- [x] Add descriptor/table and label, guard missing remotes both in row and execution; preserve existing fetch implementation.
- [x] Test actual Summary/Body/history input → palette → Enter/click with local file remotes, real completion, original selected-text restoration, draft/amend and exact local invariants. Assert tracking refs/FETCH_HEAD refresh, multiple remotes and ordinary non-prune behavior under controlled config.
- [x] Cover actual disabled no-repo/no-remote/dynamic guard rows, cancellation/no Git writes, stale root/epoch/generation and retained descriptor execution guards; failure feedback uses existing path, with the exposed shared completion race corrected below.
- [x] Read focused results, capture added entry at 360×360 en/zh light/dark, and obtain scoped role review; no blocking review finding.
- [x] Run related checks, Windows/Linux compile, full make check; read full output and verify final consumed source hashes/diff formatting.

## Decisions

Ruling: expose the existing all-remotes operation with an explicit scope label rather than imitate the reference ellipsis. Reason: the existing action executes immediately and the resources do not prove reference candidate semantics. Cost: no directed target picker or full Fetch interaction parity is claimed.

Ruling: use a palette descriptor table extending shortcut metadata rather than add a fake zero-key shortcut or broaden appShortcut registration flags. Reason: command-only actions should not affect actual key registration or Settings. Cost: both palette listing and dispatch must consume this helper, covered by actual activation tests.

Ruling: continue implementation on the existing feature checkout after verified prior checks; role agents remain read-only and isolated. Reason: no source consumer remains pending and this increment touches the integrated uncommitted application. Cost: only scoped files may be changed and final full verification is required.

Ruling: fix the shared performDoneAs cancellation-classification race exposed by the owned Fetch failure test. Reason: deferred worker cleanup cancelled the context before the queued UI callback inspected it, misreporting ordinary errors as cancellation. Cost: feedback classification changes for all callers of this shared helper; deterministic failure/active-cancel/success coverage, related operation regressions and full checks are required. Capture operationCancelled(ctx, err) before enqueue; leave operation execution unchanged.

## Progress

Observed missing-command RED at build/alignment-palette-fetch-red.log. The first focused run exposed two issues. Body cases had an extra test Enter after palette closure, legitimately replacing the restored selected Body with a newline; diagnostic output established unchanged local Git and the exact replacement, so that extra input was removed from the test. Failure feedback exposed the shared completion race described above. A deterministic independent RED at build/alignment-operation-completion-red.log waits for operationCtx.Done before executing queued UI work, proving the old error→cancelled classification. The minimal capture-before-enqueue correction then passed failure, real cancellation and success, alongside all Fetch tests (7.130 s).

Product and testing implementation reviews found no blocker; testing independently inspected the shared correction and invariant coverage. Design actually Read all four build/palette-fetch-captures/fetch-360x360-{en,zh}-{light,dark}.png renders and found no scoped issue. Roles remained read-only, with no GUI contention or independent test rerun. Complete related log read: build/alignment-palette-fetch-related.log (normal 18.926 s, targeted race 10.686 s, vet clean). Windows/Linux compilation passed exit 0. Removed only temporary diagnostic logging before freezing final inputs; full make check btr3evnb3 passed exit 0 (ordinary lobedesk 117.324 s, race 390.915 s, vet clean). Complete output read at build/alignment-palette-fetch-full-check.log; all 453 frozen source inputs matched, recorded in build/alignment-palette-fetch-source-verification.json, and git diff --check passed before Stash source changes.

No new native attempt: unchanged natural-window-focus prerequisite remains unresolved. No real remotes, project push, commit, installation or release. Resource declaration does not establish reference GUI target-selection or focus semantics.
