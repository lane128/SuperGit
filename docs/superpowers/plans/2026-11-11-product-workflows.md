# Product Workflows Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans. User explicitly requests continuous development; execute in this session without another handoff question. No project commits or publishing.

**Goal:** Expand the MyGo native preview into a usable daily Git workflow client, with explicit write confirmations and submodule-aware sync.
**Architecture:** typed gitrepo operations, main-thread state, background serialized mutations, native forms and dialogs.
**Tech Stack:** Go 1.27.3 / MyGo v0.8.3 / system Git / macOS.
**Spec:** docs/superpowers/specs/2026-11-11-product-workflows-design.md

## Global Constraints

Preserve existing files. No shell Git execution, literal filenames, no implicit discard/force push/remove, Index-only commits and real hooks. Network and destructive actions execute only through explicit UI confirmation. Tests use new temporary local repositories/remotes. No project Git checkout exists; work in place without parallel implementers or commits.

## Review Focus

- Stash selected SHA must remain the target when stack indices change.
- Dirty trees, unfinished operations and worktrees must not lose content during checkout/integration/remove.
- Push names both ends, never force, and checks submodule reachability.
- Conflict save must not follow symlinks or overwrite a file from a different repository/operation.
- Error refresh must preserve valid working status even if auxiliary data fails.

### Task 1: Branch and remote boundary
- [x] Real fixture tests RED for create/switch/rename/delete/track, dirty refusal, bare-remote clone/fetch/ff-only pull/push, unknown option names rejected.
- [x] Typed operations GREEN; explicit destination refspec and submodule push check.

### Task 2: Stash, integration, conflicts and multi-workspace boundary
- [x] Real fixture tests RED for stash SHA identity, include untracked, restore conflicts, integration operation state and conflict stages, worktree remove guards, submodule init/update guards.
- [x] Typed operations GREEN. No force operations; conflict data bounded and operation-specific.

### Task 3: Native workflow UI
- [x] Add typed action forms with immutable captured targets and dialogs; test confirmation state before wiring.
- [x] Branch/remote/stash/clone/init/worktree/submodule/integration/conflict routes with errors, disabled/busy states, cancellation of background network operations and logs.
- [x] History filtering/load more, persistent preferences/recent repository navigation.

### Task 4: Verify and deliver
- [x] Full test/race/vet/build; native automation exercises representative real workflows.
- [x] Independent read-only review of new operations, fix important findings with regression tests.
- [x] Update README, product capability matrix and progress ledger; reopen rebuilt app on isolated demo. Report remaining product work precisely.
