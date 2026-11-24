# Files state markers acceptance — 2026-11-24

Status: implemented, automated/native checks passed, installed locally at the user's explicit request. Final human acceptance pending; no commit, push, or release.

Design: [Files state markers](../../features/files-state-markers.md). T3; user approved A leading hollow badges with a fixed bundled sans-serif face and requested installation after implementation.

## Delivered

- Working-tree, commit-history, and revision-comparison file rows use a leading 18 DIP hollow status badge, preserving the existing 29 DIP row height and trailing staging controls.
- A/M/D/?/R/C/T/!/S retain existing status mapping and semantics. Conflict rows show one ! and do not gain a staging control.
- Files and detail tabs share the bundled Inter Bold face, registered separately as `SuperGit Status Text`; changing the code/UI font does not select the badge face.
- This fixes the font source across platforms, but does not promise pixel-identical OS rasterization. Only macOS was run in this acceptance.
- `internal/lobedesk/fonts/OFL.txt` includes copyright notices and full OFL 1.1 text. `make app` copies it to `Contents/Resources/Font-Licenses.txt` before signing.

## Verification

- `go test -cover ./internal/... ./cmd/...`: passed; `internal/lobedesk` coverage 85.7%. Full output: `build/files-markers-coverage.log`.
- `go vet ./internal/... ./cmd/...`: passed.
- After the license/test follow-up, gofmt plus the targeted command below passed. No production Go behavior changed after the full coverage run, so that expensive run was not repeated.
- Explicit package roots exclude temporary Git fixture repositories containing non-Go `.go` contents under `build/`, which otherwise break `./...`. The tools packages and the complete `make check` race/SDK matrix were not rerun for this localized UI/font change and packaging-resource addition.
- New tests cover font registration, file/tab pixel invariance under code-font changes, marker placement, conflicts without staging controls, narrow rows (130/110/96 DIP), history/comparison markers, preserved row semantics, and contrast across five palettes in light/dark mode. A/M/D/T/! require 7:1; other codes require 4.5:1.
- Independent scoped read-only review found no runtime defect; its must-fix was missing full font-license text and an inaccurate Inter attribution year. Both were corrected. Follow-up packaging diff and license contents were checked inline before final acceptance; no second independent review was run.
- `make app`: passed, ad-hoc codesign verification passed. The packaged license is byte-identical to the source.
- Native matrix: 16/16 passed, SuperGit/Macchiato × light/dark × 1.0/1.15 scale × 1320/900 window widths. Checks include A/M/D/?/! presence, scaled badge/row bounds, conflict staging controls absent, Stage/Unstage actions present, and commit-history markers.
- Each native fixture compares HEAD, index, porcelain status, and working-file hashes before/after viewing; unchanged in every case.
- Inspected screenshots: default light Files, narrow Macchiato dark at 1.15 scale, and SuperGit dark history. Badges are legible, aligned ahead of paths, and separated from staging controls.

```sh
go test ./internal/lobedesk -run 'Test(StatusMarker|FileRowsShowLeadingStatusMarkers|ConflictRowKeepsMarkerWithoutStageControl|HistoryAndComparisonRowsShowStatusMarkers|DetailTabStatusBadgeIgnores|FileRowSeparates)'
python3 -I scripts/native-files-state-markers.py
python3 -I scripts/native-files-state-markers.py --window-width 900
```

The initial automation failures were fixture/script issues: the script first inspected History before selecting the changes row; its staged-addition file had incorrectly been included in the base commit; and the large-font text-order check included text from the adjacent History pane. The script now opens the intended list, creates a real staged addition, and restricts text checks to the file row's horizontal bounds. No product change was required for these failures.

## Native evidence and installed build

All evidence below is local under gitignored `build/`.

| Evidence | Location |
| --- | --- |
| 1320-width matrix, screenshots, snapshots, results | `build/files-markers-native-z0lnw98_/` |
| 900-width matrix, screenshots, snapshots, results | `build/files-markers-native-tui77u3o/` |
| Standard bundle window probe and log | `build/files-markers-standard-_v1dpvkb/` |
| Installed bundle window probe | `build/files-markers-installed-window.json` |
| Installation/backup paths | `build/files-markers-install.json` |
| Prior installed bundle backup | `build/SuperGit.app-before-files-markers-20261124-120620` |

- Standard and installed bundle: `0.1.0 (45)`. The version stays unchanged because the work is uncommitted; distinguish this build by its executable hash.
- Executable SHA-256: `7939e80699ef4f1b2c5e26a847b5b70bdba112831e06a9128fa7ec509d4ba3ba`.
- Standard `build/SuperGit.app` executable launched with isolated HOME and fixture repository; CoreGraphics confirmed a visible 1320×840 window.
- Old installed app quit normally, then was moved intact to the backup above. New bundle copied with `ditto`; codesign, binary comparison, and license comparison passed.
- `/Applications/SuperGit.app` launched with PID 16976; CoreGraphics confirmed its visible 1320×840 window. It was left running for the user.

## Limits and rollback

- Native interaction/screenshots use the same product source built with the e2e automation plugin. The standard/installed builds have visible-window evidence, but no screenshot or AX-driven interaction: this session lacks system accessibility and screen-capture permissions.
- Revision-comparison badges and submodule/rename states have unit coverage; this native matrix did not exercise revision comparison, submodule protection, copy/type-change Git fixtures, or staging keyboard actions. It performed no Git write actions through the UI.
- No Windows/Linux build or rendering verification was performed.
- Existing uncommitted detail-tab work remains in the installed build and checkout.
- To roll back locally, quit the app and restore the preserved bundle. No user repository or preference deletion is needed. Source rollback should remove only this feature and its related tests/license packaging, preserving earlier detail-tab work.

Final human acceptance is pending under feature-flow Phase 3 Stop 2. Installation was explicitly authorized by the user; this record does not authorize commit/push/release.
