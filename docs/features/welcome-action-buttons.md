# Welcome Page Equal-Width Action Buttons

Status: Approved
Date: 2026-11-22
Related: Welcome page (`internal/lobedesk/view.go`, `internal/lobedesk/button_style.go`, `internal/lobedesk/i18n.go`)

## Background and goal

On the Welcome tab (`emptyPage` in `internal/lobedesk/view.go`), the three repository entry buttons (`Open Repository…`, `New repository…`, `Clone repository…`) are placed directly into a centered column without a shared width container. Because `workbenchButton` sizes itself to its label plus horizontal padding, the three stacked buttons render with three noticeably different widths.

In addition, all three labels end with a trailing ellipsis (`…`), which reads like unintended text truncation on standalone Welcome buttons, and English casing is inconsistent (`Open Repository…` vs `New repository…` / `Clone repository…`).

User-confirmed direction:
1. Render the three Welcome page action buttons in a **vertical equal-width stack** (`scaledFont(220)` wide, `Gap(8)`).
2. Include **14px Lucide icons** alongside each button label (`FolderOpen`, `FolderPlus`, `Download`).
3. Remove trailing ellipses (`…`) from `chooseFolder` (Welcome page), `newRepo` (Welcome page, macOS File menu, and sidebar repository context menu), and `cloneRepo` (Welcome page, macOS File menu, and sidebar repository context menu), and unify English Title Case (`Open Repository`, `New Repository`, `Clone Repository`) and Chinese copy (`打开仓库`, `创建仓库`, `克隆仓库`).

## Non-goals

- Do not redesign the overall Welcome page structure, hero icon, drag-and-drop zone, or Recent repositories list.
- Do not change the button height (`29` DIP), border radius (`6` DIP), colors, or focus ring tokens in `button_style.go`.
- Do not modify `openRepo` (`"Open a Git repository"`, used for the Welcome heading and macOS File menu) or sentence-case dialog/hint/command-palette keys (`newRepoTitle`, `cloneRepoTitle`, `createdOpenFailedHint`, `clonedOpenFailedHint`).
- Do not change ellipsis rules on other unrelated menu/dialog label pairs covered by `dialogLabelPairs` in `label_test.go`.
- Do not add new Welcome page actions or keyboard shortcuts.

## Approach

- Interactive HTML mockup (local-only, gitignored under `design/`): [design/welcome-action-buttons.html](../../design/welcome-action-buttons.html)
  - Open directly from the repository root: `open design/welcome-action-buttons.html`
  - Defaults to the user-confirmed **With icons** vertical equal-width design (`220px`, `Gap 8px`, clean labels without `…`).
  - Supports Light/Dark themes, English/Chinese toggle, Before vs After comparison, `1280×800` and `980×640` window sizes, and Default / Error / Drag-over / Busy (disabled) / No-recent states.

### Layout & styling changes (`internal/lobedesk/button_style.go`, `internal/lobedesk/view.go`)

1. Add a helper in `internal/lobedesk/button_style.go`:
   ```go
   func workbenchIconLabelButton(c *ui.Context, icon *ui.SVG, label string, role buttonRole, disabled bool) ui.Element {
       b := styleWorkbenchButton(c, ui.ButtonBase(c), role, disabled, false)
       b.Children(func() {
           ui.Icon(c, icon).FontSize(scaledFont(14)).Shrink(0)
           ui.Text(c, label).SingleLine()
       })
       return b
   }
   ```
   `styleWorkbenchButton` already configures `Height(scaledFont(workbenchButtonBaseHeight))`, `PaddingX(10)`, `Gap(6)`, `Radius(6)`, `FontSize(scaledFont(12))`, `FontWeight(500)`, colors, border, inset sheen, focus ring, and `Transition`. `ui.ButtonBase(c)` centers its row children horizontally and vertically, and accessibility uses the text child (`label`) so `AccessByRole(ui.RoleButton, label)` continues to match cleanly (if needed during implementation, `.Label(label)` will be set explicitly).

2. In `internal/lobedesk/view.go` (`emptyPage`), wrap the three action buttons in a centered fixed-width column:
   - Container: `ui.Column(c).Width(scaledFont(220)).Gap(8)` (no extra top margin, keeping the existing `12`-DIP zone gap so the error-to-button gap stays within the `0..20` DIP invariant tested in `repository_welcome_recent_test.go`)
   - Buttons (each chaining `.FillWidth()`):
     1. `workbenchIconLabelButton(c, lucide.FolderOpen(), a.t("chooseFolder"), buttonPrimary, blocked).FillWidth()`
     2. `workbenchIconLabelButton(c, lucide.FolderPlus(), a.t("newRepo"), buttonNormal, !a.canShowCloneRepository()).FillWidth()`
     3. `workbenchIconLabelButton(c, lucide.Download(), a.t("cloneRepo"), buttonNormal, !a.canShowCloneRepository()).FillWidth()`

Why `scaledFont(220)` and `Gap(8)`:
- `220` DIP at 100% scale (scaling proportionally via `scaledFont` to `198` DIP at Small `0.9×`, `220` DIP at Standard `1.0×`, and `253` DIP at Large `1.15×`) comfortably holds the 14px icon + 6px gap + English/Chinese label with balanced horizontal padding, and forms a clear visual hierarchy above the `420`-DIP `Recent repositories` block.
- `Gap(8)` groups the three action buttons into a cohesive action cluster, visually distinct from the `12`-DIP spacing between the heading, hint, button group, and Recent repositories section.

### Copy & i18n changes (`internal/lobedesk/i18n.go`)

Update the three existing keys directly (no duplicate keys needed since menus also drop `…`):
- `chooseFolder` (Welcome primary button): EN `"Open Repository"`, ZH `"打开仓库"`
- `newRepo` (Welcome button, macOS File menu, sidebar `repoMenu`): EN `"New Repository"`, ZH `"创建仓库"`
- `cloneRepo` (Welcome button, macOS File menu, sidebar `repoMenu`): EN `"Clone Repository"`, ZH `"克隆仓库"`

Verification against existing i18n invariants:
- `i18n_action_copy_test.go`: `"create repository"` requires Chinese copy to contain `"创建"` (`"创建仓库"` passes); `"clone"` requires Chinese copy to contain `"克隆"` (`"克隆仓库"` passes). Neither key has a `busy*` or `*Failed` prefix/suffix.
- `label_test.go`: neither `newRepo` nor `cloneRepo` is in `dialogLabelPairs`.

## UI checklist

- [x] No unjustified purple/blue gradients, glassmorphism, glow, or random emoji.
- [x] Spacing and font sizes come from a consistent scale (`scaledFont(220)`, `Gap(8)`, `Gap(6)` icon-text gap, `workbenchButtonBaseHeight = 29`, `FontSize(12)`, `Icon FontSize(14)`, `Radius(6)`).
- [x] Icon style and stroke width are consistent (Lucide `FolderOpen`, `FolderPlus`, `Download` at `scaledFont(14)`).
- [x] Real content, not lorem ipsum or placeholder marketing copy.
- [x] Every interactive element has hover, press, focus, disabled, and loading states (inherited from `styleWorkbenchButton`).
- [x] Empty, error, and loading states are designed (verified in mockup: Default, Error, Drag over, Busy/disabled, No recent).
- [x] Contrast and hit targets meet accessibility requirements (Plan A+ Primer tokens; `220×29` DIP hit targets; white icon/text on green primary button).
- [x] Motion has a purpose and respects reduced-motion settings (100ms color transition only; `prefers-reduced-motion` honored in mockup).
- [x] Each view or section has one primary action (`Open Repository` green primary; `New Repository` and `Clone Repository` neutral normal).
- [x] Light and dark themes are consistent at the target window sizes (`1280×800` and `980×640`).

## Alternatives and trade-offs

1. **Text-only buttons vs With icons** — User selected **With icons** for clearer visual differentiation of Open / New / Clone in the vertical stack.
2. **Keeping `…` in macOS menus via separate keys vs removing `…` uniformly** — User selected **removing `…` uniformly** so Welcome buttons, File menu, and repository context menu use identical clean copy without maintaining duplicate i18n keys.

## Impact

- Modules / files:
  - `internal/lobedesk/button_style.go` (add `workbenchIconLabelButton`)
  - `internal/lobedesk/view.go` (`emptyPage` fixed-width button container + icon-label buttons)
  - `internal/lobedesk/i18n.go` (update `chooseFolder`, `newRepo`, `cloneRepo` in EN and ZH)
  - `internal/lobedesk/repository_blank_journey_test.go` (update hardcoded `const openWelcomeRepository = "Open Repository"`)
  - `internal/lobedesk/repository_menu_darwin_test.go` (assert exact EN/ZH File menu labels without `…`)
  - `internal/lobedesk/welcome_action_buttons_test.go` (new regression tests covering Welcome button layout, icons, font scales, exact EN/ZH copy, and sidebar menu consistency)
  - `scripts/native-repository-tabs-smoke.py`, `scripts/native-lobe-clone-smoke.py`, `scripts/native-lobe-new-repository-smoke.py` (update Welcome & menu button selectors to the new labels without `…`)
- Compatibility: No config, persisted session, or Git behavior changes.
- Data / update signing / permissions: None.

## Test plan

- Unit tests:
  - Existing `TestEveryCopyOfAnActionUsesOneVerb`, `TestActionLabelsCarryTheirEllipsisInI18n`, `TestEllipsisLiteralsLiveInI18nOrTheElisionHelper`.
  - Existing `TestRepositoryFileMenuOffersNewAndClone`, `new_repository_test.go`, `new_repository_keyboard_test.go`, `clone_repository_test.go`, `repository_blank_journey_test.go`, `repository_welcome_recent_test.go`, `repository_open_integration_test.go`.
- Regression tests (new, in `internal/lobedesk/welcome_action_buttons_test.go`):
  1. **Exact copy & menu parity (`TestWelcomeAndMenuActionCopyWithoutEllipsis`)**:
     - Asserts exact strings in both `localeEnglish` (`"Open Repository"`, `"New Repository"`, `"Clone Repository"`) and `localeChinese` (`"打开仓库"`, `"创建仓库"`, `"克隆仓库"`), and asserts none end with `"…"`.
     - Opens the sidebar `repoMenu` in both `localeEnglish` and `localeChinese` and asserts the menu items match the exact strings without `"…"`.
     - macOS File menu parity in both `localeEnglish` and `localeChinese` (`"New Repository"` / `"Clone Repository"` and `"创建仓库"` / `"克隆仓库"` without `"…"`) is verified by `TestRepositoryFileMenuOffersNewAndClone` in `internal/lobedesk/repository_menu_darwin_test.go`.
  2. **Equal width, alignment, font scales, and error+recent edge case (`TestWelcomeActionButtonsEqualWidthAndAlignment`)**:
     - Restores `setUIFontScale` in `t.Cleanup` so the process-wide atomic never leaks between tests.
     - Renders `emptyPage` across all combinations of:
       - Locales: `localeEnglish`, `localeChinese`
       - Window sizes: `1280×800`, `980×640`
       - Font scales: `fontScaleSmall` (`0.9×`), `fontScaleStandard` (`1.0×`), `fontScaleLarge` (`1.15×`)
     - Locates all three buttons via `AccessByRole(ui.RoleButton, exactLabel)`, asserts identical `Bounds.W` (matching `scaledFont(220)`), identical `Bounds.X`, vertical ordering with `8`-DIP gap, and in-bounds placement.
     - Also verifies the combined worst-case layout (`loadErr != ""` + populated recent repositories list + `fontScaleLarge` at `980×640`) keeps all three buttons equal-width, clickable, and preserves the `0..20`-DIP error-to-button gap.
  3. **Icon presence verification (`TestWelcomeActionButtonsRenderIcons`)**:
     - Compares the intrinsic (unconstrained) width of `workbenchIconLabelButton(c, icon, label, ...)` against `workbenchButton(c, label, ...)` for each of the three Welcome buttons (`FolderOpen`, `FolderPlus`, `Download`), asserting the icon-label button is wider by at least `scaledFont(14) + 6` (14px icon + 6px gap).
     - Also compares rendered interior pixels of each `emptyPage` button against a same-width text-only `workbenchButton` baseline to ensure `emptyPage` itself paints icons on all three buttons.
- e2e / automation:
  - Run `make test` (`go test ./...`) and `go test -cover ./internal/lobedesk/...`.
- Uncovered areas and reasons:
  - Pixel-level Lucide glyph crispness and macOS native menu bar rendering are verified during real-app acceptance (`make app`).

## Acceptance steps

1. Build: `make app` and launch `open build/SuperGit.app`
2. Steps:
   - Open a new Welcome tab (`+ New Tab`).
   - Inspect in Light mode and Dark mode at both default size (~1280×800) and minimum size (980×640).
   - Switch UI language between English and Simplified Chinese in Settings (`⌘,`) and inspect the Welcome tab.
   - Switch UI text size across Small, Standard, and Large in Settings (`⌘,`) and verify the three buttons scale cleanly without truncation or overflow.
   - Trigger an open error (e.g., drop a non-Git folder) and verify the error state spacing above the buttons; verify disabled styling while busy/opening.
   - Verify hovering, Tab focus ring, and clicking each button still triggers the folder picker, New Repository dialog, and Clone Repository dialog.
   - Check macOS top `File` menu and an open repository's sidebar context menu to verify `New Repository` / `Clone Repository` (and `创建仓库` / `克隆仓库`) appear cleanly without `…`.
3. Expected results:
   - All three Welcome buttons have identical width (`220` DIP at 100% scale), cleanly aligned left/right edges, and crisp 14px icons (`FolderOpen`, `FolderPlus`, `Download`).
   - Labels read `Open Repository` / `New Repository` / `Clone Repository` (EN) and `打开仓库` / `创建仓库` / `克隆仓库` (ZH) with no trailing `…` on either the Welcome page or menus.
4. Evidence location: `docs/acceptance/2026-11-22/welcome-action-buttons/`

## Risks and rollback

- Risks:
  - Low risk. Only affects Welcome page button layout/icons and three i18n labels (`chooseFolder`, `newRepo`, `cloneRepo`); no Git commands, persistence, or dialog logic change.
  - At Large font scale (`1.15×`), `scaledFont(220)` scales to `253` DIP, well within the `980`-DIP minimum window width (`980 - 184 sidebar - 64 padding = 732` DIP available).
- Rollback:
  - `git revert <commit>` restores the previous auto-width layout, text-only buttons, and ellipsis copy without side effects.
