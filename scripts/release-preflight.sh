#!/bin/sh
set -eu

: "${RELEASE_TAG:?RELEASE_TAG is required}"
: "${UPDATE_PUBKEY:?UPDATE_PUBKEY is required}"
: "${UPDATE_PRIVKEY:?UPDATE_PRIVKEY must point to the signing key file}"

exec "${GO:-go}" run ./tools/delivery preflight \
    -tag "$RELEASE_TAG" -pubkey "$UPDATE_PUBKEY" -privkey "$UPDATE_PRIVKEY"
