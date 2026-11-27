# Repository tabs: fixed selection line and custom backgrounds

Status: Approved — user requested implementation (“按这个方案实现”) on 2026-11-27
Date: 2026-11-27
Tier: T3 — approved implementation with per-checkout local persistence

## Background and goal

The user selected the B card foundation, requested right-click tab colors, rejected the large palette and selected outer border, then selected B's bottom-line treatment. Latest direction: use one fixed selection color and coordinate customizable backgrounds with it.

The original app's short tabs have no minimum width; close controls inherit a normal-button border and inset shadow. Preserve stable card geometry and remove those close decorations while refining selection and background color as separate concepts.

## Scope and non-goals

Included: minimum width, card typography, fixed selected underline, coordinated background presets, compact right-click dots, default reset, theme adaptation, accessibility and proposed local persistence.

Product implementation was authorized by the user's “按这个方案实现” approval on 2026-11-27. No arbitrary color picker, selection-color setting, global palette settings, Git writes, status-color meanings, release work, or remote preference sharing. Existing aliases, drafts, drag reorder, context actions, close rules, welcome/error states and busy/modal guards remain intact.

## Current preview

[Interactive B preview](../../design/repo-tabs-color-preview.html) now shows only the chosen B direction. Below it, seven independent static selected-state samples show the fixed line against every background; they are component examples, not simultaneous selected tabs in one strip.

The interactive strip supports mixed colors, all one color, default/long names, and nine-tab overflow; 900px/1320px widths; system/light/dark appearance. Right-click or Shift+F10 opens the dot menu. The page's menu demo and scene controls are preview tools only.

Theme and width changes retain in-memory assignments. Reload/reset/scene changes restore fixtures. No real settings or repositories are modified. Earlier comparisons remain at `design/repo-tabs-comparison.html`; their style specifications and earlier test results are superseded.

## Geometry and typography

- Card: 36px high, 6px radius, zero border/outer outline. Equal width, ideal 192px, minimum 168px; overflow horizontally thereafter. Follow-up T2 refinement (2026-11-27, user requested more depth): a shallow bottom shadow and top-edge highlight on every card, stronger on the selected card. Keep the background and geometry unchanged; the close control still has no shadow. The scroll viewport provides five vertical DIPs around the card for shadow rendering.
- Strip: 52px high, 6px card gap, 3px horizontal track padding.
- System font 13px at weight 500 in both selection states (review follow-ups passed through 14px and regular 400 before the user settled on 13px/500 on 2026-11-27). Selection is carried by the fixed line and ink alone; it never changes label weight, size, width, background, or position. Caption elision measures this size and weight, so neither can clip names.
- Repository icon 15px. Close target 24×24px, X 14px; no border or inset shadow.
- Dirty indicator remains a 6px dot in the close slot, replaced by X on close hover/focus. It has no relationship to assigned background colors.
- Keyboard focus is an inset dashed indicator only when keyboard focus is visible. Pointer selection never adds a focus or selection border.

## Fixed selection color

Define a dedicated `selection-ink` token independent of all repository palette entries:

| Theme | Selected line | Selected text/icon | Unselected text/icon |
| --- | --- | --- | --- |
| Light | `#1764C0` | `#202B38` | `#526175` |
| Dark | `#86B7F9` | `#E9EFF7` | `#B7C5D6` |

The line is 2px high, inset 12px from either side, at the bottom of the card with rounded ends. It always uses the same token within a theme: choosing green, pink, default or blue does not recolor it. Theme changes adapt its luminance while preserving its blue identity.

Selected text and icons use neutral primary ink; other tabs use neutral secondary ink. A tab's background is identical before and after selection. The visible line, weight and accessible selected state identify the current repository, including when all tabs share one background.

## Coordinated backgrounds

Use low-saturation backgrounds so the selection blue stays prominent. Menu dot colors may be stronger to make the preset identifiable, but dot color is never the line color source. Palette values live in `repo-tabs-color-preview.html`; implementation should reuse semantic background/ink/selection roles rather than derive colors ad hoc.

| Preset identifier | Display name | Light background | Dark background |
| --- | --- | --- | --- |
| default | 默认 | `#EDF1F5` | `#202B39` |
| blue | 蓝色 | `#E7EEF8` | `#233249` |
| green | 绿色 | `#E7F0ED` | `#273A35` |
| amber | 米黄 | `#F5EFDF` | `#3B3529` |
| rose | 玫灰 | `#F3E9EE` | `#3C2D38` |
| violet | 淡紫 | `#EEEBF5` | `#332F45` |
| gray | 灰色 | `#E9EDF2` | `#303743` |

Default remains a distinct stored preference rather than an alias for explicit blue. Current display-name refinements do not change preset identifiers.

## Compact right-click menu

- One “标签颜色” label (localized from the preview's “Tab 颜色” to follow the project's Chinese terminology checks), then a single row: default plus six colors. Each dot is 18px within a 28px click target.
- Current choice has an internal checkmark; default shows ∅ when unselected. Color names appear in tooltips/accessibility labels, not caption tiles. No explanatory paragraph, grid, or separate reset row.
- Selecting a dot applies immediately and closes the menu. Default dot restores default. Right-clicking/recoloring an inactive tab does not activate it or change dirty state/order/drafts.
- Keyboard access: Shift+F10/context-menu key, arrows within/between native menu groups, Enter/Space to choose, Escape with focus restored to the target. Tab follows the native menu convention and dismisses; the HTML preview permits Tab traversal. Clicking outside dismisses without mutation. Keep the popup within the viewport.
- Preserve native context alias/close actions. Prototype demonstrates close but not the existing alias dialog.
- Native welcome/unloaded tabs must not offer persisted repository-color changes. Preserve existing busy/loading/modal guards. Do not reuse `askAlias` unchanged: it activates the target.

## Proposed persistence and implementation boundaries

The approved implementation follows `internal/lobedesk/repository_alias.go`: a separate local preference file per SHA-256 canonical checkout root, atomically saved using `writeLocalJSON`. Use a separate `repository-colors` directory beside existing settings; do not change alias formats, `.git/config`, or worktree files.

Store a validated stable preset identifier, not a display label, RGB value or selection color. Missing/unknown/malformed values fall back to default. Reset can atomically write the default identifier without deleting existing user files. A failed save reports a localized error, retains the previous assignment and leaves the active tab unchanged.

Aliases and canonical symlink paths share the same checkout preference. Same-named different roots and different worktrees may have different colors. Reopening/restarting retains colors; alias edits do not change them. Per-checkout files prevent unrelated repository preferences from overwriting one another. Update loaded same-checkout instances via existing mechanisms; cross-process changes may take effect on reopening, with last successful write winning. No broad settings refactor is needed.

Likely modules: `repository_tabs.go` (geometry, fixed underline, context target, clipping); `button_style.go` (scoped tab/close paint); a repository-color helper and tests; initialization/session preference loading; localization/accessibility strings. Git backend, signing and permissions remain unchanged.

## Verification

### Current design checks

Completed in ego-browser:

- Seven presets × two themes × selected/unselected = 28 states. Line RGB stayed identical across presets within each theme; selected/unselected background stayed identical. Border/outline remained zero, typography reviewed at 14px/400 and settled at 13px/500, later given shallow depth.
- Minimum text contrast: **5.32:1**. Minimum fixed-line/background contrast: **4.89:1**, above the 3:1 graphical indicator target.
- Four scenes × two widths × two themes = 16 layouts passed width, overflow and borderless-close checks.
- Actual right-click changed the active tab to amber without changing the line RGB; inactive recolor retained current repository; default reset and seven-dot single-row layout passed.
- Initial exact `2px` computed-height assertion encountered browser-zoom rounding (`1.99653px`). The corrected assertion allows 0.02px rounding and passed; authored height remains 2px.
- Light/dark screenshots visually inspected. Evidence: `design/repo-tabs-fixed-line-checks.json`, `design/repo-tabs-fixed-line-light.png`, `design/repo-tabs-fixed-line-dark.png`.

These are HTML checks only. Go tests and native acceptance are recorded separately during the approved implementation.

### Regression and native acceptance after approval

1. Add failing tests for short/long/CJK/duplicate captions, stable geometry at 100%/115%, fixed line independent of every preset, background unchanged by selection, neutral text and close decorations.
2. Cover preset validation, missing/corrupt preferences, canonical root identity, aliases, worktrees, reopen/reset, atomic save failures and stale menu targets.
3. Exercise inactive-tab recolor without navigation, existing alias/close/context behavior, dirty status, drafts, keyboard navigation/reveal, reorder and busy/loading/modal guards.
4. Run `gofmt`, `go vet`, targeted tests and one T3 `go test -cover ./...` using a checkout without generated `build/` fixtures. Add `make check` if shared low-level/config utilities change. Independent scoped code review must precede native acceptance.
5. Launch `make app` against disposable repositories. Verify both themes at 900×640 and 1320×840 plus 115% scale. Check fine-line visibility, same-color/gray cases, clipped controls, new-tab/overflow access, focus, titlebar dragging and welcome/error states.
6. Verify restart persistence, aliases, save failure and deterministic native guard fixtures. Record evidence under `docs/acceptance/<acceptance-date>/` and present the reviewed build for user acceptance.

## Risks, review and rollback

A thin line can become unclear through scaling or clipping; native verification must reproduce the chosen line thickness and confirm it stays visible. Colors must not regain control of the selection token or become dirty/error indicators. Local preference writes require failure and stale-target coverage.

Self-review covers scope, failure modes, compatibility, reproducibility and persistence. Independent code review remains a pre-acceptance gate. The user approved B's fixed-color underline and palette on 2026-11-27 (“按这个方案实现”), satisfying `.claude/skills/feature-flow/SKILL.md` Phase 1 Stop 1.

Rollback future implementation by reverting scoped tab/color changes. Optional preference files can remain ignored by older versions; no alias migration, Git restoration or destructive cleanup is required.
