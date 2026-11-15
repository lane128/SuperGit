#!/bin/sh
set -eu
export GIT_CONFIG_NOSYSTEM=1
export GIT_CONFIG_GLOBAL=/dev/null
export GIT_TERMINAL_PROMPT=0
root=$(mktemp -d "${TMPDIR:-/tmp}/supergit-workflows.XXXXXX")
git init --bare -b main "$root/remote.git" >/dev/null
git init -b main "$root/workspace" >/dev/null
git -C "$root/workspace" config user.name 'Preview Developer'
git -C "$root/workspace" config user.email preview@example.invalid
printf 'original\n' > "$root/workspace/conflict.txt"
git -C "$root/workspace" add conflict.txt
git -C "$root/workspace" commit -m 'Initial workspace' >/dev/null
git -C "$root/workspace" switch -c feature >/dev/null 2>&1
printf 'feature version\n' > "$root/workspace/conflict.txt"
git -C "$root/workspace" commit -am 'Feature changes' >/dev/null
git -C "$root/workspace" switch main >/dev/null 2>&1
printf 'main version\n' > "$root/workspace/conflict.txt"
git -C "$root/workspace" commit -am 'Main changes' >/dev/null
git -C "$root/workspace" remote add origin "$root/remote.git"
git -C "$root/workspace" push -u origin main feature >/dev/null 2>&1
printf 'draft kept by Stash\n' > "$root/workspace/draft.txt"
printf '%s\n' "$root"
