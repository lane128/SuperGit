# README screenshots

These images show the native SuperGit interface using isolated test repositories. Names, email addresses (`example.invalid`), commit messages, and file contents belong to those fixtures. They are illustrative data, not real project activity. Both language editions of the README use the same images.

Each PNG is resized proportionally to 1600 pixels wide from its original native screenshot. No controls, labels, or application content have been composited or replaced. The four images total approximately 834 KiB.

| Image | What it shows | Source capture |
| --- | --- | --- |
| `history.png` | Rounded graph connections, split reference capsules, Stash, and commit details in the light theme; the sidebar is hidden in this capture. | `native-history-visual-styles.py`, `light-standard/rounded-segmented.png` |
| `repository-tabs.png` | Two repository tabs, the sidebar, a working change, and a repository-specific commit draft. | `native-repository-tabs-smoke.py`, `tabs.png` |
| `partial-staging.png` | Staged and unstaged tabs for one file, with just the selected addition staged. | `native-lobe-daily-workflow-smoke.py`, `02b-inspect-both-versions-staged.png` |
| `history-settings-dark.png` | Independent graph and reference styles in the dark settings window. | `native-history-visual-styles.py`, `dark-narrow/settings-compact-stripe.png` |

Captured on 2026-11-25 from the history-style acceptance build and standard-app release smoke run. See the [history-style acceptance record](../acceptance/2026-11-25/history-visual-styles.md) and [release record](../acceptance/2026-11-25/release-1.0.0.md) for context. Capture paths under `/tmp/` and `build/` are local evidence; the copies in this directory are the durable README assets.

When refreshing a screenshot, use an isolated fixture/config, check the image for private data, keep the feature description consistent with what is visible, and update this source table. Do not capture personal repository content or publish design mockups as product screenshots.
