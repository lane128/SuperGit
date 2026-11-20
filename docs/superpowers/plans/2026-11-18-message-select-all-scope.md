# Message Select All Scope

**Status:** Reference and SDK investigation only. No new Select All menu entry or production source changes. Selection-copy whole check `b7vcjcrzi` completed successfully with 526 unchanged source inputs, eight matching packages and zero check/diff-check exits; see `2026-11-18-message-selection-copy.md`.

**Goal:** Restore a real Select All operation in the historical-message URL/SHA context menu, using the existing text-selection system and preserving focus, cross-paragraph selection and ordinary copying.

## Role findings and reference

A fresh product/design/testing batch used independent worktrees and absolute current-checkout files. Design used emil-design-eng and impeccable. Installed Sublime Merge's Commit Message menu declares `select_all`. This confirms the menu capability, not native targeting, selection range or enablement rules. Cut/paste are not required for read-only historical messages.

The app's target-free text already offers SDK Copy / Select All. Target-bearing text has the custom wrapped copy/URL/SHA menu without Select All. Existing keyboard Select All is also a selection capability; the missing piece is its custom menu entry, not a new copy-full-message action.

New text operations belong beside Copy Selected Text and before the existing separator. Escape must leave selection/focus intact; outside click should respect the newly clicked target; stale source changes must not select a newer message or restore old focus. Select All should be available for nonempty selectable content even when no text is currently selected. Copy's empty-selection disable rule does not apply to Select All.

The summary's current selectable container includes subject/body and selectable metadata. Existing ordinary Select All follows that scope. Before an implementation is called aligned, pin its exact range against the existing SDK command; do not quietly alter global selection boundaries or claim native message-only range parity. Tests must check no selection leaks into adjacent columns or the draft editor.

## SDK investigation

Pinned MyGo exposes `Menu.EditItems()`, which attaches standard editing actions to SDK-built system/drawn menus. The existing custom wrapped menu uses `ContextMenuBase` / `MenuParts`, whose public methods expose items, groups, popup and submenu parts but no corresponding editing-command item or direct selectable-container Select All method was found in the inspected public API.

`Element.SetTextSelection(start,end)` is documented and implemented for an input's local editor; it does not modify the engine's multi-paragraph selectable-container selection. Main tested this instead of assuming the shared method name covered both systems.

`build/_keyboard/message_select_all_api_test.go` is an isolated overlay experiment. First, real Cmd/Ctrl+A then Cmd/Ctrl+C selects/copies both paragraphs including Unicode exactly. After collapsing the selection, invoking the public setters on the container and first paragraph, focusing that paragraph and copying leaves the sentinel clipboard unchanged. The experiment fails with `copied "sentinel" want "Subject 作者\nBody https://example.invalid/docs"`. This is evidence that this attempted SDK route is unsuitable, not an application regression or a completed menu feature.

Do not implement a copy-all flag as Select All, simulate a production keypress, use private unsafe access, or regress long URL wrapping by assuming the native menu builder has equivalent row layout. Investigate a supported SDK editing-action integration before choosing an implementation. No dependency change or external request has been made.

### Supported drawn-menu prototype

The latest design review identified `MenuItem.Content` as a public custom-content route inside `PopoverMenu`; `Menu.EditItems()` provides the actual editing commands. Main inspected the drawn-menu implementation: its panel enables vertical scrolling and highlighted rows scroll into view. Thus the absence of a public maximum-height setter does not establish a scrolling limitation.

`build/_keyboard/message_standard_selection_test.go`, loaded only through `message-standard-selection-overlay.json`, exercises this route without changing production or dependencies. `TestPrototypeStandardMessageSelectAll` passed in **0.395 s**, and its focused race run passed in **1.639 s**. A subsequent `git diff --check` passed. An actual menu click selects two paragraphs containing Chinese, an emoji sequence and a combining mark; the action leaves a sentinel clipboard unchanged, then ordinary copying exactly matches the SDK keyboard baseline. Adjacent selectable-column text is excluded. Twelve long URLs use custom wrapped content; End brings the last row wholly inside 480×360 and Enter chooses its exact full URL. This proves the basic supported integration, not full application migration or native acceptance.

Before migration, still verify stale source/menu ownership, frozen selected-text copying, actual maximum font scale and both themes/languages, existing icons and popup treatment, cancellation/focus and long offscreen message selection. Standard `EditItems` dispatches editing commands inside the SDK rather than through the application's chosen-item callback, so these guards need explicit experiments. A dependency upgrade or patch is not currently justified by the basic selection/wrapping/scrolling requirements.

### Source lifecycle experiment

A fresh product/design/testing batch used three independent worktrees to assess integration contracts. Main ran `TestPrototypeMessageMenuSourceLifecycle` through ignored `message-menu-lifecycle-overlay.json`; all three observational cases passed in **0.515 s**, and their focused race run passed in **1.522 s**. The unguarded control keeps the menu attached to the same element when its message changes: its old Select All entry selects the **replacement** message. Two guarded variants close the stale menu and leave a sentinel clipboard untouched: omit the old `PopoverMenu` binding after source invalidation, or replace the selectable scope and paragraph keys. This establishes a public lifecycle guard mechanism; it does not establish application integration, source ABA handling, cancellation focus or frozen-copy compatibility.

The remaining design question was whether standard `EditItems` can supply Select All while preserving the current captured Copy Selected Text action without a second Copy entry. The design role's public-API inspection found no such integration in the pinned SDK: `EditItems()` has no parameters or return value, read-only commands include Copy / separator / Select All, generated items have no public enumeration/removal/customization handle, and `MenuParts` has no editing-command bridge. Standard editing dispatch bypasses the application's captured-copy handler. Do not silently replace the current tested frozen-copy semantics, introduce duplicate actions, or call this experiment a production fix.

### Integration ruling and remaining gate

Product and testing independently recommend retaining one captured Copy Selected Text action, with current Unicode payload, one clipboard write and truthful success/error feedback. This is an implementation ruling preserving the existing tested behavior, not human approval of an SDK change. An old menu may still copy its captured text, while an old Select All must never select replacement content or steal focus. Closing the entire stale menu is therefore not automatically an equivalent replacement for the existing copy contract.

The basic public standard-menu route is sufficient for standalone selection, wrapping and scrolling but does **not** yet satisfy this application's combined menu contract. Production Select All remains unimplemented. A future SDK integration should expose an independently usable standard selection command or editing-command menu item; it must avoid private access, simulated keystrokes and duplicate Copy. No dependency change, SDK cache edit or external request was made.

The testing role narrowed the eventual regression gate to five behavior groups: (1) frozen Unicode/cross-paragraph copy after refresh, including clipboard failure, (2) Escape preserving copy and outside clicks transferring actual typing focus, (3) empty/outside selection disabling copy but enabling Select All for nonempty content, (4) same-SHA text replacement, A→B→A and explicit reselect invalidating old selection commands, and (5) exact actual `commitInfoContent` scope including current selectable metadata and built offscreen first/last markers, excluding adjacent columns and draft. Existing parser/navigation/full-suite tests remain necessary after a production integration; prototypes are not substitutes.

## Required eventual acceptance

- Actual custom-menu click selects the intended complete scope; subsequent ordinary Cmd/Ctrl+C matches the SDK baseline, including Unicode and paragraph newlines.
- Long scrollable content includes built offscreen text at both ends; selection stays inside the intended summary scope.
- Enter/keyboard choice, Escape, outside click, focus restoration and stale source/menu cases behave consistently; no URL opening, Git mutation, draft change or stale jump.
- Existing target-free SDK Copy / Select All, frozen selected-text copy and long-target menu behavior remain valid.

Only headless experiments and resource/code inspection have been performed. Native system events, native reference selection and Windows/Linux runtime remain unverified. No commit, installation or publication.
