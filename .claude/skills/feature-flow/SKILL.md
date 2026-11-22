---
name: feature-flow
description: "SuperGit feature development workflow (T3 full flow, plus T1/T2 tier routing): clarify & design with review, implement with TDD & code review before acceptance, run real-app acceptance, then fast commit & push. Use for new features, major UI redesigns, Git write paths, conflict handling, data/persistence, update/signing, permissions, or cross-package refactors."
argument-hint: "[feature-slug or description]"
---

# Feature Flow

SuperGit's development workflow. It matches process weight to change risk (T1/T2/T3) and puts **code review before real-app acceptance** so approved builds commit in seconds without post-acceptance churn.

## Scope and tiers

Choose a tier before starting and state it explicitly (along with any skipped steps). When unsure, pick the higher tier or ask the user.

- **T1 (Trivial)** — docs, comments, text-only copy edits, single-file non-behavioral refactors, test-only additions.
  - Skip Phases 1 and 3. Edit → `gofmt` + `go vet` + targeted tests → inline `git diff` self-check → Phase 4 commit.
- **T2 (Lightweight feature / localized UI polish / localized bug fix)** — scoped layout/style/icon tweaks, small interaction fixes, or localized bug fixes that do **not** touch Git write paths, conflict handling, persistence/migrations, update signing, permissions, or cross-package architecture.
  - Skip the long design doc and heavyweight UI skill stack. Briefly state the plan/impact first (add a lightweight HTML preview under `design/` only if visual comparison is needed).
  - Run Phase 2 with targeted unit/regression tests + `go vet` + inline `git diff` self-check (no heavyweight review subagent; avoid full `internal/lobedesk` reruns unless shared core logic changed).
  - Run Phase 3 as a quick `make app` check (light/dark themes and target window sizes for UI changes); `docs/acceptance/` write-ups are optional unless requested.
  - Finish with Phase 4 commit & push.
- **T3 (Full workflow)** — new features, whole-page or major UI redesigns, Git write operations, conflict handling, persistence/config migrations, update signing, permissions, release flow, or cross-package refactors.
  - Follow Phases 1 → 4 below without skipping Stop checkpoints.

---

## Phase 1: Clarify, design & design review (T3)

1. **Clarify** missing requirements before drafting (do not fill gaps with assumptions):
   - Background, goal, and non-goals.
   - Observable acceptance criteria.
   - Constraints: platform, compatibility, data/persistence, update signing, window sizes.
2. **Draft design doc**:
   - Pick a kebab-case `<slug>`, copy `.claude/skills/feature-flow/templates/design.md` to `docs/features/<slug>.md`, and fill it in (including file/module plan, test plan, acceptance steps, and rollback).
   - For **major UI changes / new views**: invoke `frontend-design` / `ui-ux-pro-max` as needed, produce an HTML mockup under `design/` (`design-shotgun` for variants, `design-html` for final, `design-review` to check for AI-slop), and record the chosen direction and UI checklist in the design doc.
3. **Design review (combined into this phase)**:
   - Dispatch the read-only `.claude/agents/design-reviewer.md` subagent (or self-review against its checklist if unavailable) to check scope, failure modes, compatibility, testability, reproducibility, and risky flows.
   - Resolve all must-fix items and set status to "Pending approval".
4. **Stop 1.** Send the design doc (and mockup path if applicable) to the user and wait for a single approval before coding. Once approved, set status to "Approved".

---

## Phase 2: Implement, test & code review (T2 / T3)

Complete implementation, automated tests, and code review **before** real-app acceptance so the user tests the exact code that will be committed.

1. **Implement (TDD)**:
   - Write a failing test first, then implement; bug fixes require a regression test reproducing the original issue.
   - Keep changes strictly within scope; if scope expands noticeably, discuss (and update the T3 design doc) first.
   - Keep Go files `gofmt`-clean; match approved mockup tokens/spacing for UI changes.
2. **Verify (single-pass, avoid redundant reruns)**:
   - **T2**: `gofmt`, `go vet ./...`, and targeted tests (`go test -run ... ./<pkg>`); run full `make test` only if shared/core code changed.
   - **T3**: run `go test -cover ./...` once (do **not** run `make test` and then `go test -cover ./...` separately — `internal/lobedesk` takes ~4 min per full run and `-cover` does not share cache). For cross-package, low-level, or build-related changes, run `make check`.
   - Report any failure honestly; do not move on while verification is failing.
3. **Code review (before acceptance)**:
   - **T1 / T2**: fast inline `git diff` self-check in the main session (correctness, edge cases, stale copy/references, formatting, secrets). Do **not** spawn a review subagent.
   - **T3**: run a scoped read-only independent review (e.g., `/review` or a bounded read-only subagent). Guardrails: inspect **only `git diff` and directly related files**, focus on bugs, regressions, security, and architectural risk, and **do not re-run tests** inside the reviewer.
   - Resolve must-fix findings. After small review fixes touching only tests, docs, or trivial styling, verify with `go vet` + targeted tests rather than re-running the full `internal/lobedesk` suite; re-run full tests only if core/shared behavior changed.

---

## Phase 3: Real-app acceptance (T2 / T3)

1. Build and launch `make app` (`build/SuperGit.app`) against an isolated test repository.
2. **T2**: verify the changed behavior directly (light/dark themes and target window sizes for UI changes). `docs/acceptance/` write-ups are optional unless requested.
3. **T3**:
   - Follow the design doc's acceptance steps (plus `make e2e-build` and relevant scripts for e2e work).
   - For major UI changes: run `impeccable` (and `emil-design-eng` if interaction/motion changed) and check light/dark screenshots at target window sizes against the UI checklist.
   - Record evidence (logs, result JSON, version, steps, results, uncovered items) in `docs/acceptance/<YYYY-MM-DD>/README.md` (note when `.png` screenshots or `design/` mockups are local gitignored artifacts).
4. **Stop 2.** Present the acceptance results (and evidence paths for T3) to the user and wait for approval.

---

## Phase 4: Commit & push (fast close-out)

Because tests and code review already completed in Phase 2, this step should finish in seconds:

1. Run a quick hygiene check: `git status --short`, `git diff --check`, and confirm no secrets, private keys (`update.key`), `.env` files, acceptance credentials, build artifacts, or unrelated files are staged.
2. Prepare an English commit message:

   ```
   <type>(<scope>): <summary>

   Why: <problem and cause>
   Verified: <commands run and results>
   Rollback: <how to roll back>
   ```

3. State the change, verification, and rollback, then commit and push (fast-forward on the current branch) once acceptance passes.
4. **Always ask for explicit confirmation** before any destructive or release Git operation (`tag`, `reset`, `clean`, force-push, or publishing a Release).

---

## UI checklist

For T3 UI changes (and as a lightweight sanity check for T2 UI polish):

- No unjustified purple/blue gradients, glassmorphism, glow, or random emoji.
- Spacing and font sizes come from a consistent scale; no one-off values.
- Icon style and stroke width are consistent.
- Real content, not lorem ipsum or placeholder marketing copy.
- Every interactive element has hover, press, focus, disabled, and loading states.
- Empty, error, and loading states are designed.
- Contrast and hit targets meet accessibility requirements.
- Motion has a purpose and respects reduced-motion settings.
- Each view or section has one primary action.
- Light and dark themes are consistent at the target window sizes.

## Output requirements

At the end of each phase, state: what was done, where the artifacts are, what was skipped, and what the user needs to confirm next. Do not skip any "Stop" checkpoint on T3 work.
