## Skill routing

When the user's request matches an available skill, invoke it via the Skill tool. Route only to skills in the session's available-skills list; answer directly for quick questions or small scoped edits.

Key routing rules:
- Product ideas/brainstorming → invoke /office-hours
- QA/testing site behavior → invoke /qa
- Code review/diff check → invoke /review
- Ship/deploy/PR → invoke /ship

Do not suggest or auto-invoke other gstack skills: they run when the user types them, or when a running skill hands off to them (`gstack-config set proactive all` routes every skill).

## Development Workflow

Classify each change into one of three tiers before starting, and state the chosen tier (and any skipped steps) explicitly. When unsure between two tiers, pick the higher tier or ask the user.

### Tiers

- **T1 (Trivial)** — docs, comments, text-only copy edits, single-file non-behavioral refactors, test-only additions.
  - Skip design doc, HTML mockup, and real-app acceptance.
  - Verify with `gofmt`, `go vet`, and targeted tests (`go test -run ... ./<pkg>`).
  - Perform an inline `git diff` self-check before committing.
- **T2 (Lightweight feature / localized UI polish / localized bug fix)** — scoped layout/style/icon tweaks, small interaction adjustments, or localized bug fixes that do **not** touch Git write paths, conflict handling, persistence/migrations, update signing, permissions, or cross-package architecture.
  - Skip the full design doc and multi-skill UI pipeline. Briefly state the plan and impact before editing; create a lightweight HTML preview under `design/` only when visual comparison is needed.
  - Add unit/regression tests; run `gofmt`, `go vet`, and targeted tests (run full `make test` only when shared code is affected).
  - Perform an inline `git diff` self-check **before** real-app verification (no heavyweight review subagent).
  - Verify visually in `make app` (light/dark themes and target window sizes for UI changes); `docs/acceptance/` write-ups are optional unless requested.
- **T3 (Full workflow via `/feature-flow`)** — new features, whole-page or major UI redesigns, Git write operations, conflict handling, persistence/config migrations, update signing, permissions, release flow, or cross-package refactors.
  - Invoke `/feature-flow` and follow its 4 phases: Design & Review → Implement, Test & Code Review → Real-app Acceptance → Commit & Push.
  - **Do not start coding before the design is approved.**

### Non-negotiable gates

1. **Tests & avoiding redundant full reruns**:
   - New behavior requires unit tests; bug fixes require a regression test reproducing the original issue.
   - When coverage is needed, run `go test -cover ./cmd/... ./internal/... ./tools/...` once using the Makefile's `SOURCE_PKGS` list. Skip a separate `make test` run because `internal/lobedesk` takes ~4 min and `-cover` does not share its cache. Use `make check` for cross-package, low-level, or build-related changes.
   - After review fixes touching only tests, docs, or trivial styling, run `go vet` + targeted tests instead of re-running the full `internal/lobedesk` suite.
2. **Code review before acceptance**:
   - Complete code review (inline `git diff` check for T1/T2; scoped read-only independent review for T3) **before** presenting the build for final real-app acceptance, so approved builds commit cleanly without post-acceptance edits.
   - Scoped T3 independent reviewers inspect only `git diff` and directly related files, focus on bugs/regressions/security, and do not re-run tests.
3. **Real-app verification**:
   - User-visible T2/T3 changes must be verified by launching `make app` (`build/SuperGit.app`). T3 changes record evidence in `docs/acceptance/<YYYY-MM-DD>/`.
4. **Git operations**:
   - Ordinary `commit` and `push` (fast-forward on the current branch) do not require extra confirmation once verification/acceptance passes; always state the change, verification, and rollback, and keep commit messages in English.
   - Destructive Git operations (`reset`, `clean`, force-push) always require explicit user confirmation first. In ordinary mode, `tag` and creating a Release also require explicit confirmation.
5. **Secrets**:
   - Never commit private keys, `.env` files, or acceptance credentials; `update.key` must never enter the repository.

### Automated delivery

Only a direct request for 自动交付 or invocation of `/auto-delivery` starts automated delivery. One authorization per request covers merge and release. `tools/delivery` persists task state outside the repository. Ordinary work keeps the confirmation rules above. Use the Makefile's `SOURCE_PKGS` for Go checks across the source tree.

### Release packaging order

Feature completion, main integration, and version packaging are separate sequential gates:

1. Finish and verify the feature on its feature branch.
2. Merge the feature branch into `main` and synchronize `origin/main`.
3. Create the release tag and build the version **only from a clean `main` checkout** that matches `origin/main`.

Never create a release tag or package a version from a feature worktree or an unmerged/unintegrated branch. Before tagging, confirm all of the following in the build checkout:

```sh
test "$(git branch --show-current)" = main
test -z "$(git status --porcelain)"
test "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)"
```

If local `main` has diverged, reconcile it explicitly first; do not work around it by tagging or building from the feature branch. Record the source commit and these checks in the acceptance/release record.

### UI changes

- **T3 new pages or major redesigns**: invoke `frontend-design` / `ui-ux-pro-max` as needed, produce an HTML mockup under `design/` (`design-shotgun` / `design-html` when exploring variants), and run `design-review` / `impeccable` (plus `emil-design-eng` if interaction/motion changed) before final acceptance. Record light/dark verification and the UI checklist in the design doc and acceptance record.
- **T2 localized UI polish**: skip the heavyweight multi-skill stack unless requested; match existing tokens/patterns and check light/dark themes and target window sizes in `make app`.
