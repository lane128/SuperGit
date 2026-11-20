# Push Shortcut Implementation Plan

> **For agentic workers:** Apply already-loaded executing-plans/TDD/verification guidance. Main implements and runs tests; scoped role review uses independent worktrees. No commit, installation or release.

**Goal:** Add F8 access to the existing ordinary Push confirmation without bypassing its target/source checks or changing force-push behavior.

**Architecture:** Follow the existing F7 shortcut pattern: one shared-table binding, a canPushShortcut predicate and an execution method that rechecks it before confirmPush(). Keep existing confirmation and PushChecked execution unchanged.

**Tech Stack:** Go, MyGo/Lobe shortcut table and actual UI Tester; disposable local repositories and file-transport bare remotes only.

**Spec:** User-authorized ongoing Sublime Merge alignment. Product afb44f8bdbea9334c verified installed Default - Merge.sublime-package keymaps: OSX:322, Linux:300 and Windows:299 declare F8→show_command_palette with args.command=push, without context. This establishes the target action and a reference intermediate palette declaration. Reusing SuperGit's existing confirmation is a bounded adaptation, not proof of complete intermediate-UI equivalence. Design a7846807b773b69bf and testing a1172df05143f5f92 reviewed this boundary.

## Global Constraints

- Preserve the main workspace's corrected handoff full check b47hpf9li and verify its output/frozen inputs before integration. F8 may be implemented in the separate snapshot worktree .claude/worktrees/f8-push-entry while the main workspace remains untouched.
- Preserve unrelated dirty work/generated_images. No actual project pushes, credentials, real remotes, OS permission bypass, commit, install or publication.
- Only F8, ordinary Push. No Shift+F8, force, new settings, target-picking flow or command-catalog expansion.
- F8 opens the existing source/remote/destination confirmation; it must not write Git until explicit confirmation. Retain PushChecked's exact branch/SHA/remote URL/pushURL validation and existing success/error/upstream feedback.
- No repo, missing branch or HEAD (detached/unborn), no remotes, busy/loading/history reload, chooser/ref picker or any main modal blocks both registration and execution. Repeated F8 must not stack confirmation or start a push.
- Ordinary Summary/Body/history search may invoke the shortcut. Cancel restores surviving input text/selection and draft; do not add focus forcing, animation or infer native Enter confirmation from resources.
- No proof of complete Sublime Merge interaction parity; intermediate reference palette behavior is still unobserved.

## Review Focus

- confirmPush itself does not supply all shortcut guards; a retained window callback must recheck before opening confirmation.
- No-upstream uses the existing first-remote/same-name policy and sets tracking only through the accepted confirmation.
- Source SHA, remote URL or pushURL changes after confirmation must remain rejected; changed upstream must not silently redirect the captured target.
- Shortcut invocation and cancellation cannot change local HEAD/refs/Index/work bytes or remote refs.
- Actual input-focus tests must use real key events and selected-text continuation, not direct action calls alone.

## Task 1: F8 guarded entry

**Files:** create internal/lobedesk/push_shortcut.go and push_shortcut_test.go; modify shortcuts.go and docs/shortcuts.md. Reuse existing i18n push label unless a necessary label is absent. Existing app.go PushChecked invocation and force-push implementation stay unchanged.

**Interfaces:** Consume confirmPush(), branchPushTarget(), existing pending-dialog ownership and shortcut metadata. Produce canPushShortcut() bool and pushFromShortcut(), bound as repo.pushF8 to ui.KeyF8 with no text-input suppression and no catalog entry.

- [x] Write and observe actual F8 RED from Summary/Body/history search; assert one confirmation, no busy/write before confirmation, repeated invocation inert and cancel preserves original selected text/draft.
- [x] Write guard/retained callback coverage including nil repo, detached/unborn, no remote, busy/loading/reload, chooser/ref and dialogs.
- [x] Implement the minimal predicate/rechecked wrapper and shared binding; use the existing ordinary confirmation.
- [x] Reuse owned push fixtures for actual F8→confirm success with/without upstream and cancellation; prove exact remote OID and unchanged local staged/unstaged/untracked bytes. Reuse existing mutation rejection tests for captured target/source invariants where production flow is unchanged.
- [x] Run focused push/confirmation/input regressions, then independent bounded testing/design review. Reproduce material defects before repair; no unrelated UI expansion.
- [x] Freeze inputs, run full make check, read full result and verify fingerprints/formatting. Windows/Linux compile only establishes build portability. Preserve the known native-focus limitation without blindly retrying it.

## Decisions

Ruling: F8 enters the existing explicit confirmation, adapting the reference target action while retaining the application's guarded publication flow. Cost: the reference's directed palette step is not reproduced, so this increment cannot be called complete interaction parity.

## Progress

Previous handoff full check b26k7tfu7 ended at vet due to a test-fixture shallow copy of Repository containing sync.Mutex. After correction, b47hpf9li passed exit 0: ordinary lobedesk 106.982 s, race 383.543 s and vet clean. Complete output read; all 448 frozen corrected inputs matched and git diff --check passed before F8 integration.

F8 was implemented in an EnterWorktree-managed isolated snapshot at .claude/worktrees/f8-push-entry while the main workspace stayed frozen. Relevant baseline passed 2.839 s with dependencies resolved from cache (GOPROXY=off); 448 base source inputs fingerprinted. Missing-binding RED was observed before adding canPushShortcut()/pushFromShortcut() and one shared-table binding. First focused run exposed invalid reset/child fixtures; adding their actual captured scope (head/branch and non-worktree source selection) yielded GREEN 2.014 s without a production repair. Nine real-input cancellation combinations and two real local-file-remote operations passed; related tests passed 6.077 s, lower-layer checked Push 2.989 s, vet clean.

Implementation review roles: product a80e13808a57d51d4, design ad4fd403283b9af22 (Emil/Impeccable), testing a7a1229832f4e1653, all isolated-worktree/read-only. No confirmed production defect. Testing identified raw Index/config/cancelled-ref assertion gaps; main added raw Index bytes and full config invariance before confirmation/cancellation plus localRefs comparison after cancel. Strengthened focused tests passed 2.006 s, targeted race 3.805 s, vet clean. Design accepted actual selected-text restoration and explicit confirmation behavior, reusing unchanged confirmation visuals without claiming native execution.

Exited the implementation worktree with keep, preserving its evidence and files. A main-source fingerprint write attempted while isolated was refused by the worktree boundary; main returned through ExitWorktree before verifying main hashes or integrating, with no bypass. Integrated only the two new F8-owned Go files and its shortcut-table entry; added main notice-details modal coverage. Integrated related tests passed 5.760 s, checked-push layer 2.867 s and vet clean. Windows/Linux compilation passed. Full integrated make check b28m1a8c6 passed exit 0: ordinary lobedesk 109.990 s, race 382.611 s, vet clean. Complete output read at build/alignment-f8-full-check.log; all 450 frozen inputs matched, recorded in build/alignment-f8-source-verification.json, and git diff --check passed before any Fetch source changes.

Ruling: implement in an exact-source isolated snapshot while the main full check runs, then integrate only the owned F8 files after verifying the prior check. Cost: snapshot and integrated evidence are separate, so full checks run again after integration.

Ruling: reuse existing lower-layer source/remote/pushURL/non-fast-forward rejection tests rather than duplicate them across every input scenario. Cost: F8 tests prove actual entry, cancellation and accepted ordinary operations; mutation refusal relies on the unchanged shared execution layer.

No new native attempt: the existing natural-window-focus precondition remains unverified, no blind retry or permission bypass. Ordinary Push success/failure/upstream-partial-failure feedback is reused; this increment does not add a new feedback UI or claim directed-palette parity. No project push, commit, installation or release.

On the renewed explicit request created independent-worktree product a1cac3b6cbdc07d8e, design a4a82093f744288c9 (Emil/Impeccable) and testing a26754183055a99a5. Their scoped read-only reports are complete and agree on the existing ordinary confirmation, repeat suppression, original selected-text cancellation and local-file-remote acceptance. No roles ran GUI/tests or edited source. Main read the existing F7 entry, ordinary confirmation and checked-push tests; internal/gitrepo/push_checked_test.go already covers source, remote URL, pushURL and missing-source rejection and ordinary non-fast-forward protection. The new shortcut tests now exercise actual entry and accepted confirmation, rather than duplicate those lower-layer tests. New Repository investigation deliberately remains unresolved because no reference command-directory declaration was found, CmdN precedence is unobserved and native File-menu infrastructure would broaden the increment. All push acceptance is confined to disposable file-transport fixtures, not actual project publication.
