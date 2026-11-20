# F5 Refresh Shortcut Implementation Plan

**Goal:** Add Sublime Merge's F5 refresh key while preserving the existing Cmd+R / Ctrl+R binding.

**Reference evidence:** Coordinator independently read installed Default - Merge.sublime-package: OSX:103, Windows:102, Linux:104 each bind f5 to refresh with no explicit context/args. This is resource evidence, not hands-on reference GUI acceptance.

**Scope:** Add a shortcutTable entry with no modifiers and ui.KeyF5, label shortcutReload and the existing Global group. Both existing repo.reload and the new alias use one manual-refresh predicate and execution wrapper: repository exists, not busy/loading, no native chooser and no pending main-window dialog. Recheck at execution. Do not change low-level reload/openRepository or operation-completion refreshes. Any page and ordinary input focus remain allowed; do not add settings, retries or other reference aliases.

## Task 1

- [x] Add real registerShortcuts/tt.Key tests for each key, a genuine external file change and accepted callback refresh. Actual multiline Body is focused on a non-workspace page; tests check one loadSeq increment, canonical same root, updated status, unchanged draft/focus, HEAD/refs/Index/working bytes.
- [x] Verify nil/busy/loading/chooser/main dialogs, second key during loading, direct Run recheck and incorrect F5 modifiers do not start another read. Shared settings-card/table checks are included.
- [x] RED observed missing F5 and missing shared availability. A separate existing-refresh state assertion exposed only /var vs /private/var fixture aliases: diagnostic output confirmed draft and focus stayed unchanged. Corrected the test to compare EvalSymlinks root, preserving its same-repository check. Added shared shortcut-only predicate/wrapper and platform-correct old key label. Focused race GREEN 7.640 s, including actual commit-field batch regressions.
- [x] Product/design/testing scoped read-only reviews found no blocker. These reviews did not independently execute tests. Windows/Linux amd64 CGO-disabled builds passed; compile coverage only. Shortcut documentation updated.
- [x] Full make check bg3b92hp0 passed exit 0: go test ./... (lobedesk 82.772 s), go test -race ./... (lobedesk 344.657 s), go vet ./... clean. Coordinator read the complete log build/alignment-refresh-shortcut-check.log.

## Execution boundary

Implementation began on 2026-11-18 only after Stage tracked editor repair full check bvnvnybyb passed exit 0 (ordinary lobedesk 88.562 s, race 364.769 s, vet clean). Native keyboard acceptance remains subject to the separately recorded inability to obtain verified native window focus. macOS hardware may require Fn to deliver F5; media-key interception is not evidence of application dispatch failure. No commit, installation or release is authorized.

Coverage precision: loadSeq increments exactly once at the tested key start and remains unchanged on a second key during loading; the test does not separately assert final seq after all asynchronous work. Shared hasPendingDialog supplies notice-details protection without a dedicated new notice-details fixture. No production changes followed the full-check start.
