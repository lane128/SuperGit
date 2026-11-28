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

Run the commands below as `go run ./tools/delivery <command> ...`. When the workflow needs to distinguish exit `2` (usage) from exit `1` (operation failure), build once with `go build -o "$tmp/delivery" ./tools/delivery` where `tmp` is a temporary directory, then invoke `"$tmp/delivery"` directly. `go run` collapses nonzero program exit codes.

The ledger defaults to `SuperGit/delivery` under the OS user config directory, outside the repository. `SUPERGIT_DELIVERY_STATE` can override it. Use `-state <directory>` for an isolated ledger outside any Git checkout, and share the same ledger across real tasks competing for the same release window. JSON output uses `-json` on `status`, `preflight`, and `verify-release`.

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
| `analysis` | `analysis` | Use the read-only [requirements analyst](../../agents/requirements-analyst.md) to verify scope, existing behavior, acceptance scenarios, and T1/T2/T3. Record an already-satisfied request as described below. |
| `design` | `design` | Apply `/feature-flow` design and UI requirements for the chosen tier. |
| `design_review` | `design_review` | Use the existing [design reviewer](../../agents/design-reviewer.md). Resolve blocking findings and record design approval under the request's authorization before implementation. |
| `implementation` | `tests` | Implement in the task worktree. Run regression or unit tests as appropriate and use `SOURCE_PKGS` for source-wide Go checks. |
| `code_review` | `code_review` and `tests` | Use the read-only [delivery code reviewer](../../agents/delivery-code-reviewer.md) to review the candidate patch and test plan. Before acceptance, both checks must pass for the current candidate and be recorded during `code_review`. A review-driven commit requires the coordinator to rerun tests and obtain a fresh review for that candidate. If the candidate is unchanged, record the still-applicable test evidence again in `code_review`. Design and code reviewers read only and run no tests. |
| `acceptance` | `acceptance` | For user-visible changes, build and launch the standard package with `make app`. Use `make e2e-build` and applicable `scripts/native-lobe-*.py` for isolated automation. Select scripts that support the intended binary. For example, `SUPERGIT_NATIVE_BINARY="$PWD/build/supergit-e2e" python3 -I scripts/native-lobe-daily-workflow-smoke.py` targets the default e2e build. Record what each package verified or why a step was inapplicable. |
| `pull_request` | `remote_checks` | Verify the reviewed and accepted candidate, commit in English, push the task branch, and open the PR with acceptance evidence and rollback instructions. Verify required remote checks on the current PR head. |
| `merge` | `merge` | After acquiring the window, run `git fetch` and verify the integration candidate against current `main`. Pass the attribution check below before merging. Read back merged `main` and confirm its tree matches the verified integration candidate. |
| `release_preflight` | `release_preflight` | Confirm clean, freshly synchronized `main`; record the release selection and reconcile existing tag and Release state. |
| `tag` | `package` | Create or reuse the selected tag, build with the public key, and hand off to the protected executor for packaging, signing, and verification. |
| `publish` | `publish` | The protected executor checks the remote tag and publishes the verified assets. Read back the Release and asset details. |
| `verify` | `public_verify` | Download the public archive and manifest and verify them against the local package evidence. Record the public URLs and verification results. |
| `complete` | — | Completion releases the window automatically. |

For an already-satisfied request, record a passing `already_satisfied` check in `analysis` with the version or commit and the evidence that it meets the acceptance criteria. Block the task with the reason `already satisfied; no release needed` and report the evidence. On resume, recognize the check in `status -id <id> -json` and report the existing result without re-asking or releasing.

For automated T3 design approval, record a passing `design_review` check while in `design_review`, with the approved design's candidate fingerprint and evidence linking the design revision, independent review result, resolved findings, and recorded per-request authorization. On resume, reuse the recorded approval for that design and scope. If the task is still in `design_review`, advance from the recorded check when it matches. If the task has advanced, use the approval evidence to confirm the same design revision. A changed design requires review of the new revision and a scope change requires a user decision.

If design and design review are inapplicable, the CLI allows `analysis→implementation`. Record `design` during `analysis` with evidence starting with `skipped:` and the reason, then record the passing `analysis` check.

## Merge and attribution

Advancing into `merge` and every later transition requires active release authorization and ownership of the release window. If another task holds it, wait before merging. Keep the window during any mid-release pause. The task cannot give up the window mid-release.

Before merging, confirm the protected release executor is available. Identify the last publicly verified release tag and commit from public verification evidence, then diff that commit against the integration candidate and inspect the intervening commits. Attribute every product change to this request or a previously authorized task with recorded evidence. List documentation-only changes separately. If the baseline cannot be verified or any product change falls outside recorded authorization, block before merging and record the affected files and commits. If `main` or the candidate changes, rebuild the integration candidate and repeat the verification and attribution check.

## Release selection and protected handoff

1. With the window held in `release_preflight`, confirm that `main` is clean and `HEAD` equals both `origin/main` and `FETCH_HEAD` after fetching the release branch. Confirm that `HEAD` contains the verified merge and that the resulting tree matches the verified integration candidate. On the first selection, read the latest release tag and increment patch or minor according to the authorization, or use the authorized `exact` version. Record the selected `vX.Y.Z`, the target commit, and the last publicly verified baseline before tagging.
   ```sh
   go run ./tools/delivery check -id "$task_id" -name release_selection \
     -candidate "$sha" -passed -evidence "tag=$release_tag commit=$sha last_release=$last_release"
   ```
2. On resume, use `status -id <id> -json` and read `checks.release_selection`. Reuse its passed `release_preflight` selection and confirm that the check candidate, recorded commit, and release checkout agree. Once recorded, the version is fixed for the request. If reconciliation fails or a task has already entered `tag` without a selection, block and preserve the evidence. Create an annotated tag on the verified checkout only if the selected tag is absent. Reuse an existing same-name tag only when it resolves to the recorded commit and agrees with the authorized version. Stop on inconsistent tag or Release state.
3. Build and check the candidate app at the recorded commit with only `UPDATE_PUBKEY` available for signing configuration, for example `make app UPDATE_PUBKEY="$pubkey"`. Give the protected executor the app bundle with an artifact checksum, the public key, the selected tag and commit, and the expected release asset URL. The executor verifies the handoff and runs pinned packaging, signing, verification, and upload tools without executing candidate code or its Makefile. `UPDATE_PRIVKEY` and publish credentials stay in the executor, outside the candidate checkout, its Makefile environment, and task evidence. If the executor is unavailable, block and report the missing prerequisite.
4. The repository's combined `make release-pack RELEASE_TAG=...` target builds `app` as well as signing. For the automated handoff, the executor confirms the signing key matches `UPDATE_PUBKEY` and packages the handed-off bundle with its pinned `releasepack` binary. It then runs `make verify-release` from pinned release tooling with the same `UPDATE_PUBKEY` and `RELEASE_OUT`. After verification, compare `CFBundleShortVersionString` in the verified extracted app and `update.json`'s `version` to the selected tag without `v`. Check that the archive filename and the `darwin-arm64` asset URL match the planned release for that tag, including `SuperGit-<version>-darwin-arm64.zip`. Block on a difference even if the verifier passed. Record the verification result, archive SHA-256, and manifest before upload.
5. Before upload, read the selected remote tag from the trusted remote. If it is absent, the protected executor publishes the selected annotated tag and reads it back. Compare the peeled `refs/tags/<tag>^{}` from `git ls-remote` to the recorded commit and block on a mismatch. The protected executor uploads the verified archive and `update.json`. On retry, query the existing Release and reuse matching assets only. Download both files from their public URLs into a separate directory, compare the archive hash and manifest with the verified originals, and run the pinned `delivery verify-release -out <download-directory> -pubkey <public-key>` on the downloads.

## When to stop

Block on changed scope, unclear version policy, a critical flow the user must personally check, blocking findings after three review rounds, inconsistent tag or Release contents, or revoked authorization. Recheck authorization before each external action. Use `block` to preserve the reason and evidence, with the window held if the task has already entered `merge`. Continue after the blocker is resolved using the same authorization when its scope still applies. Obtain a direct user decision for changes that require a revision.
