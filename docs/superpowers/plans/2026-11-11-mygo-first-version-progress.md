# SDD ledger — plan: docs/superpowers/plans/2026-11-11-mygo-first-version.md

Execution: native implementation in this session. User `/goal` explicitly requests MyGo implementation and continuing without a handoff question.

Ruling: No Git worktree or commits — /Users/adam/SuperGit is not a Git repository; preserve all existing design files and implement in place. Cost if wrong: changes are uncommitted files in the requested workspace.
Ruling: Use MyGo native GPU UI with system Git for the first runnable build — actual Go 1.27.3 and Xcode are present; bundled Git packaging follows in a later distribution step. Cost if wrong: users must have Git installed for v0.1.
Pre-flight Task 1 → Task 2: typed repository methods are the only Git boundary; main thread receives immutable result messages, no Git commands in render functions.
Pre-flight Task 2 → Task 3: desktop.Run and cmd/supergit build the app; launch validation uses only a task-owned demo repository.

Task 1: complete — real Git integration tests RED (missing implementation) → GREEN. Added regressions for literal pathspecs, rename unstage, malformed .gitmodules and inherited index override. `go test ./internal/gitrepo` passes.
Task 2: complete — asynchronous guards and graph tests RED → GREEN. `go test ./...` and native builds pass. Workspace and history use separate diff results.
Task 3: complete — `go test -race ./...` and `go vet ./...` pass; local arm64 app and .app bundle built. MyGo native automation verified file/hunk stage, hunk reverse, index-only commit, parent/child navigation, uninitialized module disabled in secondary worktree, branches/worktrees and light/dark history. Screenshots captured only from this app under build/; native logs empty.

Final review: one read-only reviewer, no parallel implementers and no source edits during review. Workspace is not Git, so no worktree or review commit range. No Critical findings; three Important staging findings reproduced and fixed in one pass. No deferred minors.
Final: fixed unborn unstage after additional worktree edits — TestUnbornUnstageAfterFurtherEditsPreservesWorktree RED→GREEN.
Final: fixed staging edits to an already staged rename — TestStageFurtherEditsToStagedRename RED→GREEN, including newline destination.
Final: fixed hunk staging with diff.noprefix/custom prefixes — TestHunkStagingOverridesDiffPrefixConfiguration RED→GREEN, including reverse.
Final: fixed selection after a file moves between groups — TestFileSelectionFollowsStageAndUnstage RED→GREEN; corrected old/new direction labels.
Final verification: `make check app e2e-build` → all tests, race, vet and both native builds PASS. Final native UI also verified hunk stage/reverse with demo diff.noprefix=true and selection automatically following both group moves. Native hook-rejection test preserved draft and Index. An initial fill/press sequence reached a disabled pre-render button, so automation was corrected to wait for the updated frame; the actual submission then reported the hook rejection. The task-generated hook was removed afterwards.
Final: Ruling: Existing design/ HTML remains a separate prototype — native behavior is documented in README; cost if wrong: prototype may suggest unavailable native features.
Final: Ruling: Windows/Linux are outside current validation — deliver macOS arm64 preview; cost if wrong: no supported cross-platform binary yet.
Final: Ruling: Signing/notarization/distribution deferred — user authorized local preview, not a release; cost if wrong: this .app is for local use and requires system Git.

Actual-use evidence: native form commit `Verify native staged-only commit` in task-owned temporary demo included README.md and first hunk only. Second hunk, new file, changed submodule pointer and module dirty remained outside that commit. File-level stage followed by reverse hunk returned Index to empty and preserved working modifications. Native automation initially waited for a submodule-page label while on workspace and timed out; navigating to the submodule page then correctly showed uninitialized/disabled entry.
