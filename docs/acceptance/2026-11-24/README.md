# Settings redesign acceptance — 2026-11-24

Also on this date: [Detail tabs redesign acceptance](detail-tabs.md) and [Files state markers acceptance](files-state-markers.md), recorded separately to preserve this Settings evidence.

Status: implementation and automated acceptance complete; final human acceptance pending. No commit, push, installation, or release was performed for this redesign.

Design: [settings-redesign.md](../../features/settings-redesign.md). The user approved the design with “按照这个版本修改”. This is a T3 change. The preceding uncommitted sidebar/navigation work remains in the checkout and is not replaced by this change.

## Build identity and isolation

- Base HEAD: `2c2d4ef62c084c3063c94d9e337bd5d9b2795e20` plus the working-tree changes.
- Standard build: `make app`, `build/SuperGit.app`, version `0.1.0 (43)`.
- Standard executable SHA-256: `073304a2896beeeb774fc535554ea7393d302e35f7137eb461599d5bc0b5b2c7`.
- Build/signature log: `build/settings-redesign-build.log`; ad-hoc signature verification passed.
- Automation build: `build/settings-native-KdPTTI/SuperGit.app/Contents/MacOS/supergit`, built from the same product source with `e2e`. The minimum-size cases use an isolated Go overlay changing only initial dimensions to 560×480.
- Every layout/persistence/geometry scenario used a temporary HOME and a local fixture repository with isolated Git configuration. Initial automatic fetch/check/download preferences were disabled. Preferences and window state were stored under the fixture HOME, not the user's profile.
- Reset and save-failure scenarios only affected generated fixture preferences. HEAD, refs, index, status, and working-file bytes were compared before/after.
- The deterministic updater acceptance uses a separate overlay and action recorder. It verifies presentation and callback selection, not download, signature validation, installation, or restart.

## Implementation delivered

- Settings default 848×680 DIP; minimum 560×480; maximum 848×860; maximize/full-screen disabled in window options.
- New geometry key `settings-compact-v1`; legacy `settings` geometry remains untouched.
- Navigation 160 DIP; content begins 24 DIP after navigation, with 24 DIP on the right. Content is 640 DIP at default width and 352 DIP at minimum width.
- Form rows stack below 688 DIP. Appearance, History, Git, Shortcuts, and About share section hierarchy and grouped rows.
- Existing settings semantics retained: font selection and size, detail visibility guard, automatic fetch timing, shortcut catalog, update states/actions, and reset confirmation.
- Reset is visually separated from version/update settings. Long settings paths remain selectable and wrap.

## Automated verification

| Check | Actual result |
| --- | --- |
| `go test -cover ./...` | All packages except `internal/lobedesk` passed; that package had one failing test, `TestAppearanceHelpShowsCompleteTextAtMinimumWidth`, in both locales. It still looked for the old paragraph replaced by the approved design. Coverage reported 85.6% for that package. |
| Corrected help regression | Updated the expected paragraph to `settingsFontHelp`, retaining the complete-height/truncation assertion. Targeted rerun passed. |
| `go vet ./...` | Passed. |
| Settings/font/layout targeted tests | Passed; command and log below. |
| Independent scoped review | No remaining verified findings after fixing the reopen wait label. Reviewer did no tests/builds. Its isolation guard denied shared-checkout Git access, so it compared direct source reads against the worktree baseline rather than using live `git diff`. |
| Inline diff check | Verification-script fixes and test-only follow-ups self-reviewed before final acceptance reporting. |
| Native layout matrix | 32/32 passed: two palettes × two locales × light/dark × 1.0/1.15 text scale × default/minimum size. Each scenario visited all five pages. |
| Geometry | 4/4 passed: legacy oversized, maximized, full-screen, and valid new 700×600 geometry. |
| Persistence/reset | 2/2 passed: English and Chinese failed save, warning dismissal, recovery, reset defaults, language preservation and repository unchanged. |
| Updater presentation/action matrix | 20/20 passed: 10 snapshots × English light/default and Chinese dark/minimum, at 1.15 text scale. |
| Standard bundle launch | Passed: actual `build/SuperGit.app` executable opened a visible 1320×840 native repository window in an isolated HOME/repository. |

The full coverage run was not repeated after the test-only correction, following the project's rule to use `go vet` plus targeted tests after such changes. The original failed full-run log is retained and is not represented as a green full run.

Commands:

```sh
go test -cover ./...
go vet ./...
go test ./internal/lobedesk -run 'Test(AppearanceHelp|Settings|CodeFont|UIFont|AppearanceCard|SettingsPath)' -count=1

SUPERGIT_NATIVE_BINARY="$PWD/build/settings-native-KdPTTI/SuperGit.app/Contents/MacOS/supergit" \
  python3 scripts/native-lobe-settings-layout-smoke.py
# Repeat with --minimum-window, --redesign, and --persistence.
# Repeat default/minimum with SUPERGIT_SETTINGS_PALETTE=catppuccin-macchiato.

python3 build/check-settings-update-states.py
```

The update-state helper is a local, gitignored acceptance artifact. It retains its generated overlay and build log. Its snapshots cover idle, checking, upToDate, available, downloading with known and unknown total, verifying, readyToRestart, applying, and error. Tests verify the disabled checking action, last-check time, percentage only for known totals, action keys, and every enabled callback's recorded destination. Applying retains the existing default status/action behavior.

## Native evidence

All paths below are repository-relative local artifacts; `build/` and the HTML mockup are gitignored and were not published.

| Evidence | Location |
| --- | --- |
| Full coverage output, including initial failure | `build/settings-redesign-coverage.log` |
| Targeted regression output | `build/settings-redesign-targeted.log` |
| Final targeted follow-up and vet | `build/settings-redesign-final-checks.log` |
| SuperGit default-size matrix | `build/lobe-settings-layout-hegwg4o8/` |
| SuperGit minimum-size matrix | `build/lobe-settings-layout-btn0gdwl/` |
| Macchiato default-size matrix | `build/lobe-settings-layout-nweooj2r/` |
| Macchiato minimum-size matrix | `build/lobe-settings-layout-3xxx4dot/` |
| Macchiato combined log | `build/settings-redesign-macchiato.log` |
| Geometry fixtures/results | `build/lobe-settings-layout-vw16m7c8/` and `build/settings-redesign-geometry.log` |
| Persistence fixtures/results | `build/lobe-settings-layout-pm5u7hog/` and `build/settings-redesign-persistence.log` |
| Update-state overlay/screenshots/actions | `build/settings-update-states-s_fdcz2a/` and `build/settings-redesign-update-states.log` |
| Standard bundle visible-window evidence | `build/settings-release-launch-7d0stgnb/window-info.txt` |
| Approved HTML design render evidence | `build/settings-design-review/` |

Geometry cases verify a real native close via `menu File "Close Window"`, then read the saved JSON, compare legacy/sentinel entries, assert the new dimensions and non-maximized/non-full-screen flags, reopen Settings, and verify the layout again. The 700×600 scenario restores seeded valid compact geometry; it does not simulate a live resize drag.

Two automation issues were diagnosed and corrected without changing product behavior: UI `key cmd+w` does not execute the AppKit menu accelerator, so the script now invokes the native menu and checks the window disappeared; save warnings intentionally persist until dismissed, so the script dismisses the localized Close button before coordinate-clicking Reset. The old failed logs remain for traceability.

## Visual inspection

Inspected native screenshots from all five default light pages, default Macchiato dark Appearance, minimum dark Appearance/About, minimum Macchiato light Git, successful reset, and update downloading/ready/error states.

- Navigation/content spacing matches 160 + 24 DIP, with no wide horizontal empty band between them.
- Wide form rows align labels and controls; narrow rows keep segmented groups together below labels.
- Sections use existing theme colors, borders and font scales. No decorative gradients, shadows or new visual language were introduced.
- Long paths and update errors wrap; bottom actions remain reachable through scrolling.
- Disabled prune/download controls include their dependency explanation.
- Reset stays separate and opens confirmation by coordinate click; cancellation preserves saved preferences.
- Shorter pages retain vertical breathing room because switching categories keeps a stable window size, as approved.

## Limits and remaining human acceptance

- The terminal reports `AXIsProcessTrusted() == false` and `CGPreflightScreenCaptureAccess() == false`. The standard bundle was launched and its visible window detected, but its Settings controls were not driven or captured through system accessibility. Detailed Settings screenshots and interactions above belong to the e2e build; they are not screenshots of the non-e2e executable.
- Automation has no live resize/maximize/full-screen command. Window-option regression tests verify constraints, and native fixtures verify legacy isolation/restoration. Dragging wider on a 1600/2000 DIP desktop and clicking native window controls remain for human verification.
- Multi-display disconnect/reconnect, offscreen recovery, and a desktop work area smaller than the minimum were not tested. No native multi-display claim is made.
- Actual update network/download/install flows were not exercised. The approved change is presentation-only; updater service code was not altered by the overlay or the production change.
- Keyboard focus, section scroll restoration, menu state, font boundaries, and detail visibility guards have Go regression coverage. They were not all separately repeated as manual system-level interactions.

Final acceptance is pending under `.claude/skills/feature-flow/SKILL.md`, Phase 3 Stop 2. No commit/push is authorized by this record.

## Rollback

Revert only the Settings redesign implementation and corresponding tests after a future commit, restoring the prior layout and `settings` geometry key. Keep the unused `settings-compact-v1` entry; do not delete user preferences or geometry. Earlier sidebar History removal is a separate change and must not be included implicitly in that rollback.
