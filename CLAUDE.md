## Skill routing

When the user's request matches an available skill, invoke it via the Skill tool. Route only to skills in the session's available-skills list; answer directly for quick questions or small scoped edits.

Key routing rules:
- Product ideas/brainstorming → invoke /office-hours
- QA/testing site behavior → invoke /qa
- Code review/diff check → invoke /review
- Ship/deploy/PR → invoke /ship

Do not suggest or auto-invoke other gstack skills: they run when the user types them, or when a running skill hands off to them (`gstack-config set proactive all` routes every skill).

## Development Workflow (Required)

For feature work and any change to behavior, UI, data handling, update signing, or permissions: invoke `/feature-flow` first, produce each phase's artifact, and wait for human confirmation. **Do not start coding before the design is approved.**

Non-negotiable gates:

1. Write the design doc to `docs/features/<slug>.md` (template: `.claude/skills/feature-flow/templates/design.md`). For UI changes, also add a directly openable HTML mockup under `design/`.
2. New behavior requires unit tests; bug fixes require a regression test that reproduces the original problem.
3. Run `make test` before committing; run `make check` for cross-package, low-level, or build-related changes.
4. User-visible changes must be verified by actually launching the app (`make app`) and recorded with evidence in `docs/acceptance/<YYYY-MM-DD>/`.
5. Git operations (commit, push, tag, reset, clean) do not require user confirmation; state the change, verification, and rollback, and keep commit messages in English.
6. Never commit private keys, `.env` files, or acceptance credentials; `update.key` must never enter the repository.

Lightweight path (comments, text-only copy changes, single-file changes with no behavior change, added tests): may skip the design doc and design review, but must still follow 3, 5, and 6, and explicitly state which steps were skipped.

### UI changes

For any user-visible UI change:

- Invoke `frontend-design` before coding new UI or a redesign, and consult `ui-ux-pro-max` for concrete style, color, typography, and UX choices.
- For HTML mockups under `design/`, hand off from `feature-flow` to `design-shotgun` (variants) and `design-html` (final).
- After implementation, run an anti-AI-slop audit: `design-review` on HTML mockups, or `impeccable` on the native app. Use `emil-design-eng` when interaction or motion changed.
- Acceptance must include light/dark screenshots at target window sizes and a completed UI checklist in the design doc (see `feature-flow`).
