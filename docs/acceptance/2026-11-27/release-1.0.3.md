# SuperGit 1.0.3 release

Status: Published 2026-11-27 as the latest release. All pre-publication gates passed.

## Authorization and source

On 2026-11-27, after installing and trying the integrated-tab build, the user requested commit/push, then “发个新版本 & 打tag”.

- Feature commit: `6602a3f16edef69f9fe1549c8f0276b6041ff0a7` (`feat(lobedesk): integrate repository tabs with the toolbar`).
- Tagged from a clean `main` checkout synchronized with `origin/main`; no feature branch was involved.
- Annotated tag: `v1.0.3`, dereferencing to the commit above (verified with `git ls-remote origin 'refs/tags/v1.0.3^{}'`).
- Immediately before tagging and before packaging: branch `main`, empty `git status --porcelain`, `HEAD == origin/main == v1.0.3^{}`.
- Binary build metadata reports `vcs.revision=6602a3f…` and `vcs.modified=false`.

## Contents

Repository tabs now use an integrated Sublime-style chrome: the selected tab joins the toolbar, the selection underline and hover background fills are removed, and repository colors render as solid folder icons with neutral captions. See [feature acceptance](repository-tab-colors.md) and the [design record](../../features/repository-tabs-redesign.md).

## Validation

- `gofmt` clean; `go vet ./cmd/... ./internal/... ./tools/...`: PASS.
- Targeted repository tab tests: PASS (23.069s locally, ran again before commit). Race run of the same selection: PASS (76.634s). No redundant full-suite or coverage rerun; shared core logic was unchanged.
- Native matrix `scripts/native-repository-tab-colors.py`: 8/8 PASS (light/dark × 900/1320 widths × 100%/115% scale), repository fingerprints unchanged.
- First packaging attempt omitted `UPDATE_PUBKEY`, which left the embedded update key empty. It was repackaged from the same tagged checkout with the existing public key from `~/.config/supergit-release/update.pub`; the embedded `DefaultPublicKeyBase64` then matched that file exactly.
- `make release-pack`: PASS. Bundle reports 1.0.3 (68); `codesign --verify --deep --strict` passes on both the built app and the app extracted from the archive.
- Independent verification used a temporary program at `build/verifyrelease/` (gitignored) calling `internal/updater`: manifest URL/size/SHA-256, Ed25519 signature, ZIP safety, extraction, and bundle metadata: PASS.
- Private key permissions remained 0600. No private key bytes were printed, staged or committed.

## Artifacts

| Item | Value |
| --- | --- |
| Archive | `SuperGit-1.0.3-darwin-arm64.zip` |
| Size | 15,456,404 bytes |
| SHA-256 | `9ff23a4af69a7f7e6db5a7fe20f55d05f6696a2a7c9682f5e797efe157cec0d9` |
| Bundle | 1.0.3 (68) |
| Other assets | `update.json`, `SHA256SUMS.txt` |
| Release URL | https://github.com/lane128/SuperGit/releases/tag/v1.0.3 |

Local evidence is gitignored: `build/release-1.0.3/`, `build/release-1.0.3-public/`, `build/release-1.0.3-smoke/`, `build/verifyrelease/`.

## Publication

The release was published with `--latest`; `GET /releases/latest` then returned `v1.0.3`. All three assets were downloaded again from the public `releases/download/v1.0.3/` URLs and were byte-identical to the locally verified files. The downloaded manifest and archive passed the same independent signature, size and ZIP-safety verification.

## Limits and rollback

The release targets macOS Apple Silicon with the existing ad-hoc signature and Ed25519 updater verification; Apple Developer ID signing and notarization remain unavailable. VoiceOver narration and titlebar dragging were not separately exercised.

Revert `6602a3f` to restore the previous tabs. Older versions ignore the separate color preferences and render them as neutral icons. Published release assets and the tag are not modified or deleted without separate authorization.
