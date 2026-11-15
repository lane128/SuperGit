#!/bin/sh
# Creates only a new, private temporary directory. Existing repositories are never used.
set -eu
export GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null GIT_TERMINAL_PROMPT=0
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_COMMON_DIR
base=$(mktemp -d "${TMPDIR:-/tmp}/supergit-demo.XXXXXX")
root="$base/workspace"
child="$base/module-source"
init_repo() {
    git init -q -b main "$1"
    git -C "$1" config user.name 'Demo Developer'
    git -C "$1" config user.email 'demo@example.invalid'
}
init_repo "$root"
init_repo "$child"
printf 'package core\n\nconst Version = "0.1.0"\n' > "$child/core.go"
git -C "$child" add -- core.go
git -C "$child" commit -q -m 'Create core module'
mkdir -p "$root/src"
for n in $(seq 1 48); do printf 'line %02d: workspace refresh baseline\n' "$n"; done > "$root/src/workspace.txt"
printf '# Demo workspace\n\nA local repository for trying SuperGit.\n' > "$root/README.md"
git -C "$root" add -- .
git -C "$root" commit -q -m 'Create Git workspace'
git -C "$root" checkout -q -b feature/hunk-staging
printf 'Preview a patch before staging it.\n' > "$root/src/staging.txt"
git -C "$root" add -- src/staging.txt
git -C "$root" commit -q -m 'Add patch staging workflow'
git -C "$root" checkout -q main
printf 'native window, background git operations\n' > "$root/src/native.txt"
git -C "$root" add -- src/native.txt
git -C "$root" commit -q -m 'Connect native workspace'
git -C "$root" merge -q --no-ff feature/hunk-staging -m 'Merge patch staging workflow'
git -C "$root" -c protocol.file.allow=always submodule add -q "$child" 'modules/core kit'
git -C "$root" commit -q -am 'Track core as a submodule'
git -C "$root/modules/core kit" config user.name 'Demo Developer'
git -C "$root/modules/core kit" config user.email 'demo@example.invalid'
printf 'package core\n\nconst Version = "0.2.0"\n' > "$root/modules/core kit/core.go"
git -C "$root/modules/core kit" commit -q -am 'Improve core refresh performance'
printf '// Local module change, not included in parent commits.\n' >> "$root/modules/core kit/core.go"
git -C "$root" worktree add -q -b review "$base/review"
printf '# Demo workspace\n\nA local repository for trying SuperGit.\n\nStaged documentation update.\n' > "$root/README.md"
git -C "$root" add -- README.md
for n in $(seq 1 48); do
    case "$n" in
        3) printf 'line 03: keep the UI responsive during refresh\n' ;;
        41) printf 'line 41: stage each hunk independently\n' ;;
        *) printf 'line %02d: workspace refresh baseline\n' "$n" ;;
    esac
done > "$root/src/workspace.txt"
printf 'Try staging this new file.\n' > "$root/新文件.txt"
printf '%s\n' "$root"
