---
name: feature-flow
description: "SuperGit feature development workflow: clarify scope, write and review a design doc, plan, implement with TDD, run tests, perform real-app acceptance, then commit. Use for new features and any change to user-visible behavior, UI, data handling, update/signing, or permissions."
argument-hint: "[feature-slug or description]"
---

# Feature Flow

SuperGit's feature development workflow. It turns "design → review → code → test → real-app acceptance → commit" into phases with artifacts, instead of relying on memory.

## Scope and levels

Use the full workflow for: new features, user-visible behavior or UI changes, data handling, update/signing, permissions, and release flow.

Use the lightweight path for: comments, text-only copy changes, single-file changes with no behavior change, and added tests. The lightweight path may skip phases 1–3, but must state why and still follow phases 5 and 7.

When unsure which path applies, use the full workflow or ask the user first.

## Phase 0: Clarify

Before writing the design doc, confirm:

- Background and goal: what problem this solves and who uses it.
- Non-goals: what this change explicitly does not do.
- Acceptance criteria: what the user can observe.
- Constraints: platform, compatibility, data, update signing, window sizes.

Ask when information is missing; do not fill requirements with assumptions.

## Phase 1: Design doc

1. Pick a kebab-case `<slug>`, copy `.claude/skills/feature-flow/templates/design.md` to `docs/features/<slug>.md`, and fill it in.
2. UI changes:
   - Invoke `frontend-design` for new UI or a redesign, and consult `ui-ux-pro-max` for concrete style, color, typography, and UX choices.
   - Hand off to `design-shotgun` for variants, then `design-html` for the final mockup under `design/` (reuse existing scenes and theme variables). Provide the file path and how to open it.
   - Record the chosen direction and the UI checklist result in the design doc.
3. Set the design doc status to "Pending review".
4. **Stop.** Send the design doc and mockup paths to the user and wait for approval. Do not write implementation code before the design is approved.

## Phase 2: Design review

Check the following (dispatch a read-only subagent if `.claude/agents/design-reviewer.md` exists; otherwise self-review against this checklist and ask the user to review):

- Are the goal and non-goals clear, and is there out-of-scope work?
- Are failure modes, compatibility, and rollback covered?
- Can the test plan actually verify the goal?
- Can a third party reproduce the acceptance steps?
- Does this touch high-risk flows such as update signing, permissions, deletion, or payments?
- For UI changes: has the HTML mockup passed `design-review` with no unresolved AI-slop findings, and does the design doc record the UI checklist result?

Resolve every "must fix" item and set the design doc status to "Approved".

**Stop** and wait for the user to approve moving to implementation.

## Phase 3: Plan

List the files and modules to change, implementation order, test checklist, and rollback. If the scope expands noticeably, discuss it first.

## Phase 4: Implementation

- Use TDD: write a failing test first, then implement.
- Bug fixes require a regression test that reproduces the original problem.
- Change only what the design doc covers; if new requirements appear, update the design doc and get it re-approved first.
- Keep Go files gofmt-clean; do not make unrelated refactors.
- For UI changes, keep spacing, type, color, and states aligned with the approved mockup; do not introduce one-off values or new visual patterns without updating the design doc.

## Phase 5: Verification

- Routine changes: `make test`.
- Cross-package, low-level, or build-related changes: `make check`.
- Report `go test -cover ./<pkg>/...` for touched packages; there is no global percentage threshold, but new logic must be tested.
- Report any failure honestly; do not move to acceptance while verification is failing.

## Phase 6: Real-app acceptance

- Run `make app`, launch `build/SuperGit.app`, and follow the design doc's acceptance steps using an isolated test repository.
- For UI changes, also check light/dark themes and the target window sizes.
- For e2e work: run `make e2e-build`, then the relevant scripts.
- For UI changes: run `impeccable` on the native app (or `design-review` on the HTML mockup) as the anti-AI-slop audit, and use `emil-design-eng` when interaction or motion changed. Resolve findings before reporting acceptance.
- Capture light/dark screenshots at the target window sizes and record the UI checklist result in the design doc and the acceptance record.
- Put evidence (screenshots, logs, result JSON) in `docs/acceptance/<YYYY-MM-DD>/` and record in its `README.md`: tested version, steps, results, and uncovered items.
- **Stop.** Send the acceptance results and evidence paths to the user and wait for approval.

## Phase 7: Commit

1. Run `/review` (or an independent review) first and resolve any must-fix findings.
2. Prepare an English commit message:

   ```
   <type>(<scope>): <summary>

   Why: <problem and cause>
   Verified: <commands run and results>
   Rollback: <how to roll back>
   ```

3. State the change, verification, and rollback, then commit and push once review passes.
4. Do not commit private keys, `.env` files, acceptance credentials, build artifacts, or unrelated files.

## UI checklist

A UI change is not ready for acceptance until every item passes:

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

At the end of each phase, state: what was done, where the artifacts are, what was skipped, and what the user needs to confirm next. Do not skip any "Stop" checkpoint.
