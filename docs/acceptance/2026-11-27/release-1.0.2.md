# SuperGit 1.0.2 release

Status: Published 2026-11-27 as the latest release. All pre-publication gates passed.

## Authorization and source

On 2026-11-27, after installing and trying the repository-tab build, the user explicitly requested commit/push, merging the feature branch into main, tagging and publishing a new version. Version 1.0.2 follows the latest published 1.0.1.

- Feature commit: `14f811a44350aab6feaeedd52fb4a9fcb5d53ab7` (`feat(lobedesk): add colored repository tab cards`).
- `feature/repository-tab-colors` was pushed and merged into `main` with `--ff-only`; `origin/main` was synchronized before tagging or packaging.
- Annotated tag: `v1.0.2`, pointing to the feature commit above.
- Immediately before tagging and again before packaging: branch was `main`, `git status --porcelain` was empty, and `HEAD == origin/main == v1.0.2^{}` (tag equality checked after creating it).
- Binary build metadata confirms the source revision above and `vcs.modified=false`; no e2e tag was used.
- A fetch of historical tags reported that local `v1.0.0` differs from the remote. It was not overwritten. Main was fetched independently with `--no-tags` and release checks used the new tag explicitly.

## Contents

Repository tabs now use equal-width cards with depth, a fixed blue selection line, seven per-checkout color choices and consistent 13 DIP / 500 captions. See [feature acceptance](repository-tab-colors.md) for design review, unit tests, native light/dark/width/scale coverage and local installation evidence.

## Validation

- Pre-commit `go vet ./cmd/... ./internal/... ./tools/...` and `go build -o /dev/null ./cmd/... ./internal/... ./tools/...`: PASS.
- Final focused tab/color/card/copy regression run: PASS (19.513s). `tools/releasepack`, `tools/appversion` and `internal/updater` tests: PASS.
- The generic `/verify` wrapper was NOT_READY: its `./...` checks traversed ignored native-test fixture source under `build/`, and its test selector did not detect a Go runner. Actual source-package checks above were run separately. No tests, checker policy or native evidence were removed or weakened to get green.
- Full race check `go test -race -timeout 20m ./cmd/... ./internal/... ./tools/...`: PASS. Seven packages with tests passed (lobedesk 841.751s, gitrepo 156.946s, desktop, appicon, updater, releasepack, appversion); no failures and no data races. Three packages have no test files.
- `make release-pack` ran from the clean synchronized main checkout, using the existing update public/private key pair: PASS.
- `codesign --verify --deep --strict` passed on both the built app and the app extracted from the release archive.
- Independent archive verification checked version, download URL, archive size, SHA-256, Ed25519 signature and ZIP safety: PASS.
- Both the published 1.0.1 manifest and the new 1.0.2 manifest validate with the existing public key, confirming update-key continuity.
- Extracted release metadata is 1.0.2 (build 66); executable bytes match the packaged build and its Go build metadata contains the intended update public key.
- The extracted release started with an isolated Chinese profile and disposable repositories, stayed running for five seconds with an empty log, then was closed. Earlier 8/8 native cases covered the same feature source with the e2e executable.
- Private key permissions were 0600. No private key bytes were printed, staged or committed.

## Artifacts

| Item | Value |
| --- | --- |
| Archive | `SuperGit-1.0.2-darwin-arm64.zip` |
| Size | 15,454,519 bytes |
| SHA-256 | `d8acb5313c392c89106e2957a25bd2a80f51a2b0e1dfc7e849e8c2e25010e29d` |
| Bundle | 1.0.2 (66) |
| Other assets | `update.json`, `SHA256SUMS.txt` |
| Release URL | https://github.com/lane128/SuperGit/releases/tag/v1.0.2 |

Local evidence is gitignored: `build/release-1.0.2/`, `build/release-1.0.2-smoke/`, and `.superpowers/sdd/2026-11-27-repository-tab-colors/release-source-checks.json`. The full race log is `/tmp/supergit-1.0.2-race.log`.

## Publication

The draft release was published with `--latest`. `GET /releases/latest` then returned `v1.0.2` (not draft, not prerelease). All three assets were downloaded again from the public `releases/download/v1.0.2/` URLs: each was byte-identical to the locally verified file. `verify-release.go` re-validated the downloaded manifest and archive (URL, size, SHA-256, Ed25519 signature, ZIP safety), and the archive was extracted again to confirm `codesign --verify --deep --strict` passes and the bundle reports 1.0.2 (66).

The release was created only after the race check passed. Nothing in the published assets was changed after publication.

## Limits and rollback

The release targets macOS Apple Silicon. It uses the existing ad-hoc macOS signature and Ed25519 updater verification; Apple Developer ID signing and notarization remain unavailable. VoiceOver narration and titlebar dragging were not separately tested in this feature acceptance.

Revert feature commit `14f811a` to restore the previous tabs. Older versions ignore the separate color preferences. The previous local installation is preserved at `build/SuperGit-1.0.1-installed-backup.app`. Deleting or replacing published release assets/tags is a separate operation requiring authorization.
