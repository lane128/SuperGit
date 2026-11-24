# Detail tabs redesign acceptance — 2026-11-24

Status: implementation, independent review, automated native acceptance, and local installation complete; no push, publication, or release performed.

Design: [detail-tabs-redesign.md](../../features/detail-tabs-redesign.md). The user approved the hollow-outline design with “按照这个版本修改”. T3 workflow; design approval preceded product edits, and independent code review preceded native acceptance.

## Delivered behavior

- Content row: 36 DIP at standard text size, 41.4 DIP at 1.15 scale; separate 6 DIP rail, with a 3 DIP resting thumb and full-height hover/drag feedback.
- Status badge: transparent interior, 1 DIP colored border, 4 DIP radius, 18 DIP square scaled with text, 11 DIP/800 letter from the bundled Inter Bold face (`SuperGit Status Text`, fixed independently of code/UI font settings). A/M/D remain green/amber/red, with the existing T/!/R/C/?/S semantics preserved.
- A/M/D ink and border meet 7:1 against selected/idle/hover/pressed surfaces; other statuses meet at least 4.5:1. Local colors use the existing contrast helper and do not change global tokens.
- Flat selected background, 2 DIP accent underline, neutral hover/press, independent keyboard focus ring. Repository tabs retain their existing appearance.
- Names use measured middle elision at UAX29 grapheme boundaries. The existing typesetting dependency supplies segmentation; no dependency version was added or upgraded.
- Stable 600-weight width measurement; names render at 400/600. The stage suffix and 24 DIP close slot do not shrink. History/worktree maximum widths are 240/280 scaled DIP, capped by the actual viewport.
- Full path, stage and localized status meaning remain in the tooltip/accessibility description. Existing accessible tab names and identity semantics remain intact.
- Native scroll container still handles wheel input, clamping and reveal. Custom rail preserves focus and accounts for press/move events arriving in one frame.

## Build identity and isolation

- Base HEAD: `838e4d3ad9a4200db90eca8319b167b4f08a6545`, plus the pending detail-tab changes.
- `make app`: `build/SuperGit.app`, version `0.1.0 (45)`, ad-hoc signature verification passed.
- Standard executable SHA-256: `f4483c0ee9df420ef636887b4ade16951eb145d5adf968cd667d784a6fa9cbfd`.
- `make e2e-build`: `build/supergit-e2e`, same product source with the automation plugin.
- Automation executable SHA-256: `db5fa65fe51067a9bccb72a933cdc68c798d64f362bcd8cd443821f67daa6b4f`.
- Each native case created a new fixture repository and HOME, disabled auto-fetch/update preferences, and isolated Git configuration. HEAD, index, porcelain status and working-file hashes matched before and after viewing, dragging, switching and closing tabs.
- The standard bundle opened a visible 1320×840 native window under a separate HOME, then was installed to `/Applications/SuperGit.app` at the user's explicit request. The prior installed `0.1.0 (43)` bundle was quit and preserved at `build/SuperGit.app-0.1.0-43-20261124-101538`; the installed `0.1.0 (45)` signature and binary hash match the accepted build.
- After installation, `open -a /Applications/SuperGit.app` launched PID 91944, and CoreGraphics reported an on-screen native window at 1320×840. The app was left running for the user.

## Verification results

| Check | Result |
| --- | --- |
| Initial regression run against old UI | Failed as expected: clipped active tab at 230 DIP, no hollow badge/status meaning, wrong selected surface and no thumb in the independent rail. |
| Targeted detail/name/focus/shortcut regressions | Passed after implementation. |
| `go vet ./...` | Passed on final product code. |
| `go test -cover ./...` | Other packages passed; `internal/lobedesk` failed only `TestEllipsisLiteralsLiveInI18nOrTheElisionHelper`, which rejected the new helper's placement. Reported package coverage: 85.7%. |
| Ellipsis ownership correction | Moved the new elision helper unchanged into `model.go`, alongside existing text helpers. Kept the invariant test unchanged. Targeted invariant and behavior tests passed. |
| Review regression tests | Focus-loss test and batched-grab-offset tests failed before their respective fixes and passed afterward. |
| Final targeted suite | Passed, 4.943 seconds; includes all detail tab/name/track/focus/file-tab/history-detail regressions and ellipsis/label invariants. |
| Independent code review | Two rounds. Both confirmed scrollbar findings fixed; no remaining actionable findings. No tests run by reviewer. |
| Native 1320×840 matrix | 8/8 passed: SuperGit/Macchiato × light/dark × 1/1.15 scale. |
| Native 900×840 matrix | 8/8 passed; asserts an actual 230 DIP detail viewport after splitter drag, including large English Unstaged. |
| Standard bundle launch | Passed; CoreGraphics detected an on-screen native SuperGit window. |
| `git diff --check` and Python syntax | Passed. |

The full coverage command is retained as an initial failed run, not represented as a green final full run. After the local helper relocation and scrollbar review fixes, `go vet` and the affected regression suite were rerun; the expensive full package suite was not repeated. No existing shared model behavior changed.

Commands:

```sh
go vet ./...
go test -cover ./...
go test ./internal/lobedesk -run 'Test.*(DetailTab|DetailTrack|DetailName|DetailFocus|FileTab|HistoryDetail|WorkspaceSelectionSummary|Ellipsis|ActionLabels|MenuLabels)' -count=1
make app
make e2e-build
python3 scripts/native-detail-tabs-redesign.py
python3 scripts/native-detail-tabs-redesign.py --window-width 900
```

## Native checks and visual inspection

The native script opens real A/D/M files and both staged/unstaged versions, then 12 long filenames and a combining-mark/ZWJ filename. It drags the history divider, verifies the active tab and full close slot fit the actual scroll viewport, hits the top edge of the 6 DIP rail, drags to the start without changing the diff target, scrolls with a horizontal wheel, closes the active version by pointer coordinates, and checks focus restoration plus Tab/Shift+Tab. Repository fingerprints remain unchanged.

At the minimum window with 1.15 scale, measured bounds were:

- Detail viewport: `(670, 122, 230, 41.4)` DIP.
- Active Unstaged tab button: `(670, 122, 206, 41.4)` DIP.
- Reserved trailing close width: 24 DIP, ending at x=900.
- Rail: y=163.4 through 169.4 DIP.

Inspected native captures of SuperGit light deleted content, SuperGit dark added content, Macchiato dark large-font Unstaged at both window widths, and the light large-font close successor with visible focus ring and full-path tooltip. Hollow badge interiors, colored letters, middle elision, stable suffix and bottom line match the approved direction. No extra animation was added to repeated keyboard or drag actions.

Interaction review using the loaded `emil-design-eng` guidance:

| Before | After | Why |
| --- | --- | --- |
| Custom rail press transferred focus to the diff pane | `KeepFocus()` retains the prior tab/editor | Keyboard navigation continues after scrolling. |
| Latest pointer position became the grab origin | Subtract the initial drag delta | Coalesced press/move events preserve the full drag distance. |
| Revealing only Close could leave the name offscreen | Reveal the viewport-capped whole tab | Both name and close action remain visible when switching backward. |

The `impeccable` craft checks were applied to the actual native captures: hierarchy, text/background contrast, transparent badge treatment, focus/hover/pressed states, narrow layout and correspondence to the approved comp. No new visual direction or settings were introduced.

## Evidence

All build paths are local gitignored artifacts; no external publication occurred.

| Evidence | Path |
| --- | --- |
| Initial regression output | `build/detail-tabs-red.log`, `build/detail-tabs-track-red.log` |
| Full coverage output including failed invariant | `build/detail-tabs-full.log` |
| Final targeted regression output | `build/detail-tabs-final-targeted.log` |
| Standard-width matrix log | `build/detail-tabs-native-matrix.log` |
| Standard-width screenshots, bounds, action logs, results | `build/detail-tabs-native-3w9tntji/` |
| Minimum-width matrix log | `build/detail-tabs-native-minimum.log` |
| Actual 230 DIP screenshots, bounds, action logs, results | `build/detail-tabs-native-hjjylvid/` |
| Standard bundle window probe | `build/detail-tabs-standard-b45v2c_l/window-info.json` |
| Installed bundle window probe | `build/installed-window-info.json` |
| Previous installed bundle backup | `build/SuperGit.app-0.1.0-43-20261124-101538` |
| Reusable native acceptance script | `scripts/native-detail-tabs-redesign.py` |

The first native pilot expected an accessible splitter role, but this application's handles are pointer-only. The final script derives the divider coordinate from the actual Files pane bounds. This was a harness correction, with no product change. Its failed log remains `build/detail-tabs-native-pilot.log`.

## Review boundaries and remaining human acceptance

- The independent reviewer ran in an isolated worktree. A shared-checkout Git command was rejected by its isolation guard; review instead compared direct reads of the exact pending source against the isolated baseline. No permissions were changed.
- The standard bundle has no automation plugin, and system accessibility/screen-capture permissions were unavailable. Its launch/visible-window check is separate from the detailed screenshots and interactions, which used the same-source e2e executable. The installed bundle was launched and its native window detected, but its controls were not driven through system accessibility.
- Native matrix language was English. Chinese sizing and localized descriptions were covered by headless native-UI regression tests, not a separate Chinese window matrix.
- Batched drag input is covered by the actual grab-offset helper tests and static tracing of MyGo's input ordering. The native automation driver sends pointer actions separately; it does not reproduce all events within one OS render frame.
- No Git write flows, settings migrations, release/update flows or multi-display behavior changed or were claimed as tested here.
- The user requested local installation of this accepted build. Commit and push remain unperformed and require an explicit request; this record does not authorize a release or external publication.

## Rollback

After a future revert, restore only this detail-tab implementation and its tests/docs: restore the former tab row and native scrollbar presentation. Tab identity, open-file state, stage selection, repository data and preferences require no migration or cleanup.
