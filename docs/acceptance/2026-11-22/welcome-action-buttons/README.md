# Acceptance Record — Welcome Page Equal-Width Action Buttons

- **Date**: 2026-11-22
- **Feature Design Doc**: [welcome-action-buttons.md](../../../features/welcome-action-buttons.md)
- **Tested Build**: `SuperGit 0.1.0 (34)` (working tree on top of `3474768`, built via `make app` and `make e2e-build`)
- **Platform**: macOS (Apple Silicon)

## Verification Summary

### Automated Tests (`go test -cover ./...` & `go vet ./...`)
- `gofmt -l internal cmd tools`: clean (no diff)
- `go vet ./...`: PASS
- `go test -cover ./...`: PASS (exit code 0)
  - `supergit/internal/lobedesk`: PASS (242.317s, **85.4% coverage**)
  - `supergit/internal/gitrepo`: PASS (132.601s, 80.4% coverage)
  - All remaining packages (`appicon`, `desktop`, `updater`, `appversion`, `releasepack`): PASS
- New & updated regression tests:
  - `TestWelcomeAndMenuActionCopyWithoutEllipsis` (EN/ZH exact strings without `…` across Welcome page and sidebar `repoMenu`)
  - `TestRepositoryFileMenuOffersNewAndClone` (EN/ZH macOS File menu exact strings without `…`)
  - `TestWelcomeActionButtonsEqualWidthAndAlignment` (EN/ZH × `1280×800` & `980×640` × Small/Standard/Large scales + `loadErr` & recent list & Large @ `980×640`)
  - `TestWelcomeActionButtonsRenderIcons` (helper intrinsic width + rendered `emptyPage` pixel comparison vs text-only baseline)

### Native Real-App Acceptance (`build/supergit-e2e` + `build/mygo-automation`)
All 6 native GPU scenarios passed in isolated `$HOME` and fixture repository environments (details in [results.json](./results.json); `.png` screenshots listed below are stored locally and gitignored):

| Scenario | Theme / Lang / Scale / Size | Measured Button Width × Height | Vertical Gap | Evidence |
| --- | --- | --- | --- | --- |
| EN Light Standard + Dialogs | Light · EN · 1.0× · `1280×800` | `220.0 × 29.0` (all 3, `X=625.5`) | `8.0` | [welcome-en-light-1280x800.png](./welcome-en-light-1280x800.png) |
| EN Dark Standard | Dark · EN · 1.0× · `1280×800` | `220.0 × 29.0` (all 3, `X=625.5`) | `8.0` | [welcome-en-dark-1280x800.png](./welcome-en-dark-1280x800.png) |
| ZH Light Standard (min window) | Light · ZH · 1.0× · `980×640` | `220.0 × 29.0` (all 3, `X=475.5`) | `8.0` | [welcome-zh-light-980x640.png](./welcome-zh-light-980x640.png) |
| ZH Dark Large (min window) | Dark · ZH · 1.15× · `980×640` | `253.0 × 33.35` (all 3, `X=459.0`) | `8.0` | [welcome-zh-dark-980x640-large.png](./welcome-zh-dark-980x640-large.png) |
| EN Light Small (min window) | Light · EN · 0.9× · `980×640` | `198.0 × 26.10` (all 3, `X=486.5`) | `8.0` | [welcome-en-light-980x640-small.png](./welcome-en-light-980x640-small.png) |
| Sidebar `repoMenu` | Light · EN · 1.0× · `1100×720` | `New Repository`, `Clone Repository` (no `…`) | — | [repo-menu-en-light.png](./repo-menu-en-light.png) |

Interactive checks verified in native app:
- Clicking `New Repository` opens the New repository modal; `Cancel` dismisses cleanly.
- Clicking `Clone Repository` opens the Clone repository modal; `Cancel` dismisses cleanly.
- Opening an isolated Git fixture repository and clicking the sidebar repo header exposes `New Repository` and `Clone Repository` without trailing ellipses.

## UI Checklist Result
- [x] No unjustified purple/blue gradients, glassmorphism, glow, or random emoji.
- [x] Spacing and font sizes come from a consistent scale (`scaledFont(220)`, `Gap(8)`, `Gap(6)` icon-text gap, `workbenchButtonBaseHeight = 29`, `FontSize(12)`, `Icon FontSize(14)`).
- [x] Icon style and stroke width are consistent (Lucide `FolderOpen`, `FolderPlus`, `Download`).
- [x] Real content, no placeholder copy.
- [x] Interactive states (hover, press, focus, disabled) inherited uniformly from `styleWorkbenchButton`.
- [x] Empty, error, and recent-list layouts verified (unit + native).
- [x] Contrast and hit targets meet accessibility requirements in both Light and Dark themes.
- [x] Motion limited to 100ms color transitions in `styleWorkbenchButton`.
- [x] Single primary green CTA (`Open Repository` / `打开仓库`); secondary actions use neutral surface buttons.
- [x] Light and Dark themes visually inspected and verified at `1280×800` and `980×640`.

## Uncovered / Manual Verification Notes
- Native macOS top menu bar (`File → New Repository / Clone Repository`) is verified programmatically via `TestRepositoryFileMenuOffersNewAndClone` (both English and Simplified Chinese) against `mygo.App.Menu()`, not by OS-level menu bar screenshot.
- Native system folder picker sheet (`NSOpenPanel`) triggered by `Open Repository` and physical Finder drag-and-drop onto the Welcome canvas are left for optional manual spot-check in `build/SuperGit.app`.
