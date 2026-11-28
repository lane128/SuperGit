#!/bin/sh
set -eu

: "${RELEASE_TAG:?RELEASE_TAG is required}"
: "${UPDATE_PUBKEY:?UPDATE_PUBKEY is required}"
: "${UPDATE_PRIVKEY:?UPDATE_PRIVKEY must point to the signing key file}"
: "${RELEASE_OUT:?RELEASE_OUT is required}"

exec "${GO:-go}" run ./tools/releasepack \
    -app "${APP:-build/SuperGit.app}" -tag "$RELEASE_TAG" \
    -out "$RELEASE_OUT" -priv "$UPDATE_PRIVKEY" \
    -base-url "${RELEASE_BASE_URL:-}" \
    -notes-zh "${RELEASE_NOTES_ZH:-}" -notes-en "${RELEASE_NOTES_EN:-}"
