# File tab keyboard navigation

Goal: connect installed Sublime Merge Build 2125 prev_file_tab / next_file_tab resource keys to existing detail tab navigation. Default - Merge.sublime-package contains OSX:254–259 (Cmd/Ctrl+PageUp/Down and Cmd+Shift+[/]), Windows:235–236 and Linux:237–238 (Ctrl+PageUp/Down), without explicit contexts/args. Ctrl+Tab maps to a different next_tab command and is outside this increment. Resource evidence does not establish actual reference GUI cycle/focus behavior.

Product decision: preserve the existing visible detail tab order including permanent Commit/Summary, cycle both directions, and do nothing when no file tabs are visible. Only Changes and visible History details participate; never switch pages or create tabs. Versioned file keys remain distinct. Focus and reveal the activated tab once, retain ordinary editing text, draft/amend, per-tab scroll and current repository/commit identity. Reuse activateDetailTab/navigateDetailTab and the existing tab reveal path. The visible permanent first tab is a deliberate SuperGit adaptation, not proven reference behavior.

Guard request and retained Run, then recheck focus consumption: repository present, no busy/loading/history refresh, native chooser, reference popup or pending main-window dialog. Hidden historical file diffs do not participate. Scope focus requests to detail context and repository load epoch, discard requests after blocking state/context/epoch changes even if tab controls were absent. Do not introduce native window refocus, new settings or changed layout.

Steps:
- [x] Actual keyboard RED for the missing shared-table bindings.
- [x] Implement platform aliases, translated labels and one-shot guarded tab focus; retain existing local arrow keys.
- [x] Verify versioned tabs/summary/full-cycle, actual Summary/Body and History search dispatch, preserved draft, Tab escape, historical context/scroll, hidden/empty/single-file and guard/stale-request cases.
- [x] Scoped role review, relevant race tests, cross-platform compilation, native acceptance and diff formatting.
- [x] Full make check passed exit 0 (`bhcb92z8e`): ordinary lobedesk 93.914 s, race 359.481 s, vet clean. Full `build/alignment-file-tabs-check.log` read. No production source changes during this run; only documentation and owned native driver work.

Evidence and decisions:
- Initial bindings RED: `build/alignment-file-tabs-red.log`. First implementation exposed SDK ownership of modified PageUp/PageDown inside both single- and multiline editors (`build/alignment-file-tabs-search-red.log`). Added narrow `OnTextKey` forwarding from actual Summary, Body and History SearchBar to the shared file-tab bindings; consume only matching, available actions.
- Product/design static reviews found no blocker. Testing review found that hiding historical Diff between request and consumption could redirect pending file-tab focus to Summary. Reproduced both before/after visibility normalization (`build/alignment-file-tabs-hidden-red.log`), then scoped pending focus to its original target and visible file state. Root/commit/load epoch and blocking state continue to cancel it.
- Related race command: `go test -race ./internal/lobedesk -run '^TestFileTab|^TestDetailTabsKeep|^TestDetailNavigation|^TestEvery.*Shortcut|^TestTheSettingsCard|^TestBatchShortcut' -count=1` passed in 20.033 s (`build/alignment-file-tabs-race.log`). A prior new scroll-test run panicked because the test wrote a nil map; initialized the fixture map and reran successfully.
- Windows/Linux amd64 CGO-disabled builds passed; this is compilation evidence, not native target-OS acceptance.
- macOS ordinary current e2e binary: `build/supergit-lobe-tabs-native`. Acceptance `build/file-tabs-native-sdnhbzw5/`, driver `build/.native-focus/file-tabs.py`, log `build/alignment-file-tabs-native.log`. All three alias pairs work from actual Summary/Body; single-file cycles include Commit; Tab escapes; draft, refs, Index and working bytes unchanged. Screenshot inspected, process stopped 0. Native focus was a checked prerequisite; no window-refocus workaround. Initial driver used an incorrect `option` role for the file opener and stopped before testing any alias; corrected to the actual Open in file tab button.
- Historical search dispatch has real-widget coverage, not a separate native acceptance run. Historical scroll coverage uses an unmounted content panel to avoid viewport clamping; overflow reveal reuses existing production path and was statically reviewed, not independently accepted with an overflowing native strip.

No commit, installation or release is authorized. Previous New Repository full-chain comparison passed once in current ordinary build; its original intermittent focus/Enter cause remains unresolved.
