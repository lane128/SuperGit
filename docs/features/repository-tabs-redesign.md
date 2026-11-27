# Repository tabs: integrated chrome and colored icons

Status: Approved — user requested “按照 1A 推荐方案实现” on 2026-11-27; native verification recorded in the acceptance document.
Date: 2026-11-27
Tier: T2 visual follow-up to the released T3 repository-color feature

## Background and goal

The original B design used fixed blue selection lines and configurable colored card backgrounds. After trying the installed build, the user requested a 38 DIP strip, clearer tab boundaries and consistent hover behavior. They then rejected the three disconnected surfaces formed by the strip, tabs and toolbar.

The current approved direction follows the supplied Sublime Merge reference: integrate the selected tab with the toolbar, remove the selected underline and hover background changes, and move repository colors to solid folder icons. The user chose variant 1A, with neutral text. This specification supersedes the earlier fixed-line and colored-background designs; their implementation and verification history remains in [the acceptance record](../acceptance/2026-11-27/repository-tab-colors.md).

## Scope and non-goals

This follow-up changes tab/strip/toolbar presentation, tab geometry, icon treatment and the New Tab button's appearance. It preserves existing aliases, drafts, drag reorder, context actions, close rules, welcome/error states, busy/modal guards and per-checkout color preferences.

No arbitrary color picker, additional settings, Git writes, persistence migration, status-color meanings, release work or remote preference sharing. Implementation approval does not authorize a new commit or release.

## Current preview

Approved comparison: [Sublime-inspired 1A/1B/1C](../../design/repo-tabs-sublime-v1.html). Variant 1A uses solid colored folder icons and neutral labels. Earlier comparisons remain in `design/repo-tabs-integrated-redesign.html` and `design/repo-tabs-color-schemes.html` for reference. Preview files and screenshots are local, gitignored artifacts.

## Geometry and typography

- Strip: 38 DIP high (or the platform titlebar minimum), including a 1 DIP divider painted beneath its children. Horizontal track padding is 2 DIP per side; tabs have no gap.
- Tab: 35 DIP high, aligned to the bottom of the strip, leaving 3 DIP above at the standard strip height. Top corners use a 6 DIP radius; bottom corners are square. A 1 DIP top/side border reserves the same space in both selection states. There is no bottom border.
- Equal widths: ideal 192 DIP, minimum 168 DIP; horizontal overflow beyond that. Caption width accounts for both side borders.
- Selected tab covers the strip divider and uses the toolbar surface, creating an uninterrupted connection. Its top/side border identifies the selected shape.
- Unselected tabs are transparent on the strip. A centered 1 DIP vertical separator appears between consecutive unselected tabs; it is omitted beside the selected tab's existing edge.
- System font: 13 DIP at weight 500 in both selection states. Selection never changes label weight, size, width or position. Caption measurement uses the same metrics.
- Repository icon: 15 DIP. Assigned colors use a solid folder silhouette; the default folder and welcome icon retain their existing neutral outlines.
- Close target: 24×24 DIP, X at 14 DIP, without a background, border or shadow. Dirty state uses a 6 DIP dot in the same slot, replaced by X while the tab is hovered or the close control is focused.
- Keyboard focus remains an inset dashed indicator. New Tab is a frameless plus-and-text button with its accessible label and keyboard focus behavior preserved.

## Integrated chrome colors

The strip is a recessed track; the selected tab and toolbar form one surface. Repository color never changes either background.

| Role | Light | Dark |
| --- | --- | --- |
| Strip / unselected tabs | `#E5E9EF` | `#0C1016` |
| Selected tab / toolbar | `#FFFFFF` | `#1C2128` |
| Strip divider | `#D4DBE3` | `#252C36` |
| Selected top/side edge | `#D2D9E2` | `#2E3642` |
| Selected text | `#1F2328` | `#E6EDF3` |
| Unselected text | `#5C6672` | `#8B949E` |
| Hovered unselected text | `#1F2328` | `#DCE4ED` |

These values apply to the SuperGit palette. Other palettes fall back to their existing semantic layout, elevated-surface, border and text tokens.

There is no selection underline, shadow, hover fill or press fill. Hover increases foreground emphasis across the tab, including its close control. Selection remains available through the native accessible selected state as well as the tab shape and connected surface.

## Repository icon colors

The same hue is used by the folder icon and context-menu swatch. Unselected, unhovered icons use 88% opacity; selected or hovered icons use full opacity. Labels remain neutral.

| Preset identifier | Display name | Light icon | Dark icon |
| --- | --- | --- | --- |
| default | 默认 | Neutral text ink | Neutral text ink |
| blue | 蓝色 | `#0969DA` | `#58A6FF` |
| green | 绿色 | `#1A7F37` | `#3FB950` |
| amber | 米黄 | `#9A6700` | `#E3B341` |
| rose | 玫灰 | `#BF3989` | `#F778BA` |
| violet | 淡紫 | `#7C3AED` | `#BC8CFF` |
| gray | 灰色 | `#57606A` | `#9CA6B2` |

Default remains a distinct stored preference rather than an alias for blue. Preset identifiers and stored data do not change.

## Compact right-click menu

- One localized “标签颜色” label, followed by default plus six colors. Each dot is 18 DIP within a 28 DIP click target.
- Current choice has an internal checkmark; default shows ∅ when unselected. Color names remain available as tooltips/accessibility labels.
- Choosing a dot applies immediately and closes the menu. Recoloring an inactive tab does not activate it or change dirty state, order or drafts.
- Keyboard access: Shift+F10/context-menu key, arrows within/between native groups, Enter/Space to choose, Escape with focus restored to the target. Tab dismisses under the native menu convention. Clicking outside dismisses without mutation.
- Preserve alias/close actions and busy/loading/modal guards. Welcome/unloaded tabs do not offer persisted repository-color changes.

## Persistence and implementation boundaries

Existing persistence follows `internal/lobedesk/repository_alias.go`: one local preference file per SHA-256 canonical checkout root, atomically saved using `writeLocalJSON`. The `repository-colors` directory sits beside settings; aliases, `.git/config` and worktree files are unchanged.

Store a validated stable preset identifier, never a display label or RGB value. Missing, unknown or malformed values fall back to default. Reset atomically writes the default identifier. Failed saves report a localized error, retain the previous assignment and leave the active tab unchanged.

Canonical symlink paths share a checkout preference. Same-named roots and separate worktrees may use different colors. Reopening/restarting retains preferences; aliases do not alter them. Loaded same-checkout instances update through existing mechanisms; cross-process changes may take effect on reopening, with the last successful write winning.

The T2 follow-up touches `repository_tab_style.go`, `repository_tabs.go`, `repository_tab_menu.go`, `view.go` and rendered style tests. Git backend, signing, permissions and persistence code are unchanged.

## Verification

Current regression coverage:

1. Selected tab surface joins the toolbar without a bottom divider; top/side borders and separators render in both themes.
2. Hovering either the label or close control leaves both background regions unchanged, for selected and unselected tabs.
3. All seven presets × two themes × two font scales retain neutral backgrounds and stable geometry. Assigned colors render in the solid folder icon; close targets remain 24×24 DIP.
4. Existing repository tab, color, overflow, caption, titlebar and keyboard tests remain applicable.
5. Run `gofmt`, `go vet` and targeted tests; complete inline diff review before native acceptance. T2 skips a new full design pipeline, independent reviewer and redundant full-suite run.
6. Build `make app` and `make e2e-build`; exercise light/dark at 900×640 and 1320×840 with 100%/115% font scale. Inspect actual native captures for selected seams, icons, long-name elision, menu and overflow controls.

Latest results and local evidence are in [the acceptance record](../acceptance/2026-11-27/repository-tab-colors.md). Earlier fixed-line contrast and pixel checks describe superseded designs and are not evidence for 1A.

## Risks and rollback

The selected shape and seam must stay visible when tabs scroll or text scale changes. Repository icon colors must remain independent of selection and dirty/error state. Keyboard focus must remain visible without adding pointer-hover backgrounds.

Rollback the scoped presentation changes to restore the previous appearance. Repository-color preferences need no migration or cleanup; older versions continue to interpret the same identifiers. A new release requires separate authorization.
