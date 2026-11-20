# Repository Chooser Selection Resume

**Status:** Isolated headless verification complete; no production source change. The second-parent full check `bywb96pls` remains pending and its source inputs stay frozen. No commit, installation or publication.

**Goal:** Verify that cancelling Open Repository preserves the exact commit-field selection and lets the next typed character replace that selection, without an extra focus click.

## Role decisions

A fresh product/design/testing batch used independent worktrees and current main-checkout files. Design used emil-design-eng and impeccable. Design and testing selected the existing Cmd/Ctrl+O flow as a bounded verification candidate, rather than introducing an interface or shortcut. Existing integration tests inject directory chooser cancellation and check focus/context; they do not exercise a selected Unicode substring followed by resumed typing.

The product role initially recommended file multi-selection based on an older alignment-log investigation. Current `workspace_selection.go` and `workspace_multiselect_test.go` already implement and exercise that capability; the role withdrew the missing-feature claim after checking the current files. Historical investigation text must not be treated as current capability inventory.

## Experiment and evidence

Ignored overlay files:

- `build/_keyboard/repository_open_selection_resume_test.go`
- `build/_keyboard/repository-open-selection-resume-overlay.json`

`TestPrototypeOpenRepositoryCancelResumesExactSelection` uses the actual summary and description controls separately. Keyboard events select the substring `作者` inside `prefix 作者 suffix`; ordinary copy first verifies the exact selection. Cmd/Ctrl+O invokes the existing action with an injected empty directory chooser result. Without clicking or refocusing, ordinary copy must still return `作者`, then typing `X` must replace only that substring in the original field and preserve the other field. The existing repository identity, loading state, HEAD, symbolic ref, refs, index entries and owned worktree bytes are checked.

Focused race command:

```sh
go test -race -overlay build/_keyboard/repository-open-selection-resume-overlay.json ./internal/lobedesk -run '^TestPrototypeOpenRepositoryCancelResumesExactSelection$' -count=1 -v
```

Passed both field cases, package time **1.903 s**. No production bug was reproduced. The overlay was not added to the production package while the existing whole check is running, and that check will not validate this ignored experiment.

## Remaining evidence boundary

Injected cancellation proves application input behavior in the headless tester. It does not prove native directory-panel focus or selection restoration, actual OS events, or Windows/Linux runtime. Those remain separate acceptance items. Do not infer that a production repair is needed from this missing native evidence, or claim native reference equivalence from a passing headless experiment.
