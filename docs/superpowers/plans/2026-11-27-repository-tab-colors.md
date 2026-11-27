# Repository Tab Colors Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans in this session. User approved implementation; do not ask again for execution method. Commits remain deferred until acceptance.

**Goal:** Implement the approved solid repository cards, fixed selection underline and compact persisted color picker.

**Architecture:** Keep repository styling separate from file tabs. Store a stable preset ID in a per-canonical-checkout JSON file next to aliases, using the existing atomic writer. Build the custom palette with MyGo ContextMenuBase so dismissal, focus, placement and ordinary menu actions retain native behavior.

**Tech Stack:** Go, MyGo UI, lobe-mygo theme tokens, existing UI tester/native automation.

**Spec:** docs/features/repository-tabs-redesign.md; approved by “按这个方案实现”.

## Global Constraints

- T3: regression tests first, independent scoped review before make app acceptance.
- 36px cards, 6px radius, 168px minimum/192px ideal equal width, 6px gaps; 13px text at 500/600 weight; 24px close with 14px X.
- Selection uses #1764C0 light / #86B7F9 dark, 2px underline inset 12px. Background does not change on selection.
- Seven stable IDs: default, blue, green, amber, rose, violet, gray. Use exact theme colors in the spec.
- No Git writes, alias format changes, dependency changes, release operations or private credentials.

## Review Focus

- A stale menu after close/replacement must not write to a different checkout (Task 1).
- Rename failure must retain prior color and report an error on the visible workbench (Task 1).
- Canonical symlink roots and separate worktrees must share/separate preferences correctly (Task 1).
- Keyboard dismissal and selection must restore tab focus without accidental navigation (Task 3).
- Busy/loading states, clipped close buttons and scaled overflow must preserve current protections (Tasks 2–3).

### Task 1: Per-checkout preference

**Files:** create internal/lobedesk/repository_color.go and repository_color_test.go; modify app.go and repository_session.go.

**Interfaces:** repositoryColorPath(root string) (string, error); readRepositoryColor(root string) string; (*repositoryTabs).setRepositoryColor(target *app, root, color string). app.repositoryColor stores a normalized preset; repo.Root is already canonical from Git open.

- [x] Add failing tests for seven IDs, missing/corrupt/unknown JSON fallback, distinct default/blue, independent roots, failed atomic rename, stale target, inactive assignment, and reopen/session restoration.
- [x] Run `go test ./internal/lobedesk -run 'TestRepositoryColor' -count=1`; confirm failure.
- [x] Implement validated per-root storage with `writeLocalJSON`, updating memory only after success. Load alongside aliases in normal open and session restore.
- [x] Re-run targeted tests; require PASS.

### Task 2: Cards and fixed underline

**Files:** create repository_tab_style.go and repository_tab_style_test.go; modify repository_tabs.go.

**Interfaces:** repositoryTabColors(dark bool, preset string, selected bool) (background, ink, line ui.Color); paintRepositoryTab(tab ui.Element, tk *theme.Tokens, preset string, selected bool) ui.Color.

- [x] Add rendered pixel/geometry tests for minimum/equal widths, all backgrounds with both selection states/themes, 100%/115% scale, and borderless/shadowless close slots.
- [x] Run targeted tests to reproduce old geometry/close decoration failures.
- [x] Apply approved tokens, fixed underline, font weight, stable card dimensions; use a plain ButtonBase for close and preserve dirty/disabled/clipping behavior. Preserve horizontal overflow, arrows and drag reorder.
- [x] Run tab/style/overflow/drag/label tests; require PASS.

### Task 3: Compact menu

**Files:** create repository_tab_menu.go; extend repository_color_test.go and repository_tab_context_test.go; modify i18n.go and repository_tabs.go.

**Interfaces:** (*repositoryTabs).tabMenu(c *ui.Context, target ui.Element, a, session *app) (closeTab, editAlias bool).

- [x] Add failing tests for inactive recolor without activation/draft mutation, seven dots in one row, keyboard selection/dismissal, outside dismissal and busy/stale-target protection.
- [x] Implement ContextMenuBase with 18px dots in 28px targets, internal checkmark/default reset, labels/tooltips, existing close/alias actions, guarded writes. Keep popup placement/focus behavior owned by MyGo.
- [x] Run context/color tests; require PASS.

### Task 4: Verification and acceptance

- [x] Run gofmt, go vet and one full coverage suite excluding pre-existing generated build fixtures, record commands/results.
- [x] Obtain scoped read-only independent review of diff and direct dependencies without rerunning tests; resolve findings and run proportionate checks.
- [x] Build/launch make app against isolated disposable repositories. Verify light/dark at 900×640, 1320×840, 115% scale; color save/reset/restart, inactive target, keyboard, dirty/long/overflow cases.
- [x] Record results/screenshots/limitations in docs/acceptance/2026-11-27/repository-tab-colors.md and present the reviewed build for user acceptance (Stop 2).
