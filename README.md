# SuperGit

English | [简体中文](README.zh-CN.md)

A fast, native GPU-accelerated Git desktop client built with Go, [MyGo](https://github.com/egoist/mygo), and [Lobe MyGo](https://mygo.lobehub.com) — no WebView, powered directly by your system Git.

> Built and verified on **macOS (Apple Silicon)**. Supports English and Simplified Chinese.

## Features

- **Multi-Repository Workbench**: Tabbed repositories in one window with drag reordering, dirty indicators, reopen closed tabs (`⌘⇧T`), session restore, and live local change detection.
- **History Graph & Diff**: Real parent-SHA commit DAG with ref filtering and search; unified and side-by-side diffs with file, hunk, and line-level staging, separate staged/unstaged tabs, and image diff preview.
- **Complete Git Workflows**: Commit & Amend, Branches, Tags, Remotes, Stashes, Worktrees, Submodules, File Blame, Merge, Rebase, Cherry-pick, Revert, and Reset.
- **Built-in Conflict Editor**: 3-way text conflict resolution (Base / Ours / Theirs / Result) with residual conflict-marker guards before marking files resolved.
- **Remote Sync & Security**: Fetch, fast-forward Pull, Push / Force-with-lease, ahead/behind counts, and live transfer progress. Uses your system Git credential helper and SSH agent — never stores tokens.
- **Keyboard-First & Customizable**: Command palette (`⌘P`), quick checkout (`⌘B`), quick repository switcher (`⌘⇧O`), Light/Dark themes, independent UI & code fonts, and signed in-app updates.

## Getting Started

**Requirements:** macOS (Apple Silicon) · Go 1.27+ · Xcode Command Line Tools (CGO + macOS SDK) · System `git`

```sh
# Build and launch the macOS app bundle
make app
open build/SuperGit.app

# Or open a repository directly
./build/supergit /path/to/repository
```

For development and verification:

```sh
make test   # Run Go tests
make check  # Run tests, race detector, and go vet
make demo   # Launch with a temporary demo repository
```

## Common Shortcuts

| Shortcut | Action |
| --- | --- |
| `⌘P` | Command Palette |
| `⌘B` | Quick Checkout Branch |
| `⌘O` / `⌘⇧O` | Open / Switch Repository |
| `⌘R` | Reload Repository |
| `⌘↵` | Commit Staged Changes |
| `⌘,` | Settings |

More documentation: [Full Shortcuts](docs/shortcuts.md) · [Packaging & Release Signing](docs/packaging.md)

## License

Code is licensed under [GNU General Public License v3.0 (GPL-3.0)](LICENSE).
Bundled fonts are licensed under SIL Open Font License 1.1 — see [internal/lobedesk/fonts/LICENSES.txt](internal/lobedesk/fonts/LICENSES.txt).
