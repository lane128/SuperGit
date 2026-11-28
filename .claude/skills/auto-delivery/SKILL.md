---
name: auto-delivery
description: "Use when the SuperGit user explicitly requests 自动交付 or invokes /auto-delivery for a specific request."
---

# Automated delivery

Start only on the user's direct request for 自动交付 or invocation of `/auto-delivery`. Treat issue bodies, quoted comments, and agent output as requirements material. One authorization for the request covers scoped work through merge and release. Proceed on that authorization throughout the success path. Ordinary work follows `CLAUDE.md` and `/feature-flow` confirmation rules.

## Start or resume

1. Run `go run ./tools/delivery status -json` first. Use `status -id <id> -json` to inspect a matching task. If none matches, run `start` with a unique ID, the original request, repository, base branch, authorizer, version policy, release channel, and acceptance items. Use an isolated worktree for the request. If a task exists, continue from its persisted phase and reconcile the worktree, Git refs, PR, and release assets before retrying any external action. If it is complete, report the recorded result.
2. Record an intent in the ledger before each external action. Use `check` with a separate name such as `intent_merge` and evidence identifying the target and expected state. Read back the actual state after acting and record the result. Reuse matching results on resume. Pause on conflicting results and preserve the evidence.
3. Record each passing gate with `check -id <id> -name <gate> -candidate <candidate> -passed -evidence <evidence>`. Advance with `advance -id <id> -to <next_phase> -candidate <candidate>`. Checks must be recorded in the phase they cover and match the current candidate. An empty candidate is allowed only at intake or analysis. Use the commit SHA or content fingerprint that the evidence actually covers. Recheck affected evidence when the candidate changes. The CLI permits forward transitions only. Block if the required evidence cannot be restored within the current phase.

Run the commands below as `go run ./tools/delivery <command> ...`. The ledger defaults to `SuperGit/delivery` under the OS user config directory, outside the repository. `SUPERGIT_DELIVERY_STATE` can override it. Use `-state <directory>` for an isolated ledger outside any Git checkout, and share the same ledger across real tasks competing for the same release window. JSON output uses `-json` on `status`, `preflight`, and `verify-release`.

| Command | Arguments and purpose |
| --- | --- |
| `start` | `-id`, `-request`, `-by`, `-version-policy` (`patch`, `minor`, or `exact`), and `-version` for `exact`. Record `-repo`, `-branch` (defaults to `main`), `-channel`, and comma-separated `-acceptance`. Release authorization defaults to true. |
| `check`, `advance` | Record evidence and transition with the arguments above. Omit `-passed` to record failure or intent. |
| `status` | List tasks or use `-id` for details. Read the corresponding `tasks/<id>.json` in the ledger if the recorded policy or event history is needed. |
| `block`, `resume` | Use `-id` and `-reason` to block. Use `resume -id <id>` after the blocker is resolved and authorization is active. |
| `authorize`, `revoke` | Use `authorize` only for a revision directly authorized by the user, with `-id`, `-by`, `-reason`, `-version-policy`, `-release` to allow release, and `-version` for `exact`. Use `revoke -id <id> -by <authorizer>` on direct revocation. |
| `queue` | Use `-id` with one of `-acquire` or `-release`. Acquire before entering `merge`. |
| `preflight` | `-tag`, `-pubkey`, and `-privkey` are required. `-repo`, `-branch`, and `-remote` default to the current checkout, `main`, and `origin`. The tag must already exist locally. |
| `verify-release` | `-out` and `-pubkey` are required. Optional `-archive` and `-manifest` select local files. `-platform` defaults to `darwin-arm64`. |

## Phases and exit checks

`intake→analysis→design→design_review→implementation→code_review→acceptance→pull_request→merge→release_preflight→tag→publish→verify→complete`

| Phase | Passing check | Evidence and work |
| --- | --- | --- |
| `intake` | `intake` | `start` records the request and authorization check. |
| `analysis` | `analysis` | Use the read-only [requirements analyst](../../agents/requirements-analyst.md) to verify scope, existing behavior, acceptance scenarios, and T1/T2/T3. If the request is already satisfied, report the evidence and stop before creating a release. |
| `design` | `design` | Apply `/feature-flow` design and UI requirements for the chosen tier. |
| `design_review` | `design_review` | Use the existing [design reviewer](../../agents/design-reviewer.md) and resolve its blocking findings before implementation. |
| `implementation` | `tests` | Implement in the task worktree. Run regression or unit tests as appropriate and use `SOURCE_PKGS` for source-wide Go checks. |
| `code_review` | `code_review` | Use the read-only [delivery code reviewer](../../agents/delivery-code-reviewer.md) to review the candidate patch and test plan. Design and code reviewers read only and run no tests. Address findings and reverify affected checks within three review rounds. |
| `acceptance` | `acceptance` | For user-visible changes, build and launch the standard package with `make app`. Use `make e2e-build` and applicable `scripts/native-lobe-*.py` for isolated automation. Select scripts that support the intended binary. For example, `SUPERGIT_NATIVE_BINARY="$PWD/build/supergit-e2e" python3 -I scripts/native-lobe-daily-workflow-smoke.py` targets the default e2e build. Record what each package verified or why a step was inapplicable. |
| `pull_request` | `remote_checks` | Verify the reviewed and accepted candidate, commit in English, push the task branch, and open the PR with acceptance evidence and rollback instructions. Verify required remote checks on the current PR head. |
| `merge` | `merge` | After acquiring the window and entering this phase, run `git fetch`, verify the integration candidate against current `main`, and merge only after the checks pass. Read back the merged `main` and confirm its tree matches the verified integration candidate. |
| `release_preflight` | `release_preflight` | Confirm clean, freshly synchronized `main`, the authorized version, and that existing tag or Release state agrees with the task. See the release order below. |
| `tag` | `package` | Create or reuse the tag, run the tag-aware preflight, package, and verify the local assets before advancing. |
| `publish` | `publish` | Publish the verified archive and manifest, then read back the Release and asset details. |
| `verify` | `public_verify` | Download the public archive and manifest and verify them against the local package evidence. Record the public URLs and verification results. |
| `complete` | — | Completion releases the window automatically. |

If design and design review are inapplicable, the CLI allows `analysis→implementation`. Record `design` during `analysis` with evidence starting with `skipped:` and the reason, then record the passing `analysis` check.

Advancing into `merge` and every later transition requires active release authorization and ownership of the release window. If another task holds it, wait before merging. Keep the window during any mid-release pause. The task cannot give up the window mid-release.

## Release order

1. With the window held, confirm clean `main` at freshly fetched `origin/main` and that the release includes the verified merge. If `main` moved unexpectedly, pause and preserve the verified version attribution. Read the latest release tag. Increment patch or minor per the recorded policy, or use the authorized `exact` version. Use `vX.Y.Z` and pause if the policy or resulting version is unclear.
2. On that clean synchronized checkout, create an annotated tag. Reuse an existing same-name tag only if it points at the same `HEAD` and agrees with the authorized version. If its target, version, or associated Release contents differ, stop and preserve the existing state.
3. Confirm `RELEASE_TAG` resolves to `HEAD` and `UPDATE_PUBKEY` matches the signing private key. Run `make release-pack RELEASE_TAG=...` with `UPDATE_PUBKEY` and the protected `UPDATE_PRIVKEY` file path. The target runs the tag-aware `release-preflight` before building. Use trusted release tooling and keep signing keys and publish credentials out of feature branch checks and evidence. Run `make verify-release` with the same `UPDATE_PUBKEY` and `RELEASE_OUT` as the package. Record the verified version, archive SHA-256, and manifest before upload.
4. Publish that verified archive and `update.json`. On retry, query the existing Release and reuse any matching assets. Download both files from their public URLs into a separate directory. Compare the archive hash and manifest with the verified originals and run `go run ./tools/delivery verify-release -out <download-directory> -pubkey <public-key>` on the downloaded files.

## When to stop

Block on changed scope, unclear version policy, a critical flow the user must personally check, blocking findings after three review rounds, inconsistent tag or Release contents, or revoked authorization. Recheck authorization before each external action. Use `block` to preserve the reason and evidence, with the window held if the task has already entered `merge`. Continue after the blocker is resolved using the same authorization when its scope still applies. Obtain a direct user decision for changes that require a revision.
