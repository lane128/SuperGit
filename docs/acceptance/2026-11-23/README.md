# Fetch 后台化与一步获取 — 验收记录

**结果：通过。** Fetch 期间提交/文件浏览、命令面板只读命令、仓库标签页切换均可用；写命令保持禁用并显示等待原因。命令面板 Fetch 一步执行，无选项页与远端页。fetch 完成后两个远端的新引用出现在界面中。

## 验收对象

| 项目 | 内容 |
| --- | --- |
| 日期与平台 | 2026-11-23，macOS 27.2（26B131），Apple Silicon。 |
| 工作树 | 分支 `design/workbench-ux`，基准提交 `a499088`，含本次未提交改动。 |
| 构建 | `make app`（`build/SuperGit.app`，0.1.0 (36)，ad-hoc 签名通过）与 `make lobe-e2e-build`（同一工作树的默认工作台，带 MyGo 自动化入口）。 |
| 测试仓库 | `/tmp/supergit-acceptance-2026-11-23/repo`：3 个提交、`origin` + `upstream` 两个本地裸远端；`remote.origin.uploadpack` 指向 `slow-upload-pack.sh`，每次协商前延迟 30 秒。第二个仓库 `other` 用于标签页切换。 |
| 窗口 | 真实工作台 1320 × 840 pt；脚本化 smoke 另跑 900 × 640 与 1320 × 840 pt，115% 字号。 |
| 自动化入口 | `build/mygo-automation` 通过无障碍树读取与操作原生窗口；Git 结果由独立 `git` 进程核对。 |

## 自动化验证

| 命令 | 结果 |
| --- | --- |
| `gofmt -l internal/lobedesk` | 无输出 |
| `go vet ./...` | 通过 |
| `go test -cover ./...` | 全部通过；`internal/lobedesk` 85.4%，`internal/gitrepo` 80.4%，`internal/updater` 57.0%，`internal/desktop` 30.9% |
| `python3 -I scripts/native-lobe-toolbar-sync-smoke.py`（900 × 640） | PASS，证据 `build/toolbar-sync-cyue3rh1/` |
| 同上（1320 × 840） | PASS，证据 `build/toolbar-sync-kvr07tph/` |

Smoke 覆盖四个 Fetch 命令：`Fetch all remotes`、`Fetch all remotes with prune`、`Fetch all remotes with tags`、`Fetch from Remote…`，分别在两个本地远端核对新引用、prune 结果、标签范围与 HEAD/工作树不变；同时覆盖 Merge 预览、Push 确认、Pull 与取消。

## 真实工作台：Fetch 期间浏览

证据目录：本目录。脚本为临时文件 `/tmp/sg-fetch-accept/run.py`，不在仓库内。

| 步骤 | 结果 | 证据 |
| --- | --- | --- |
| ⌘P 输入 `repo.fetch` | 只显示 `Fetch all remotes`；回车后立即开始 fetch，没有 Fetch Options 输入框或远程选择页；Sync 菜单禁用，进度条出现在分支栏下。 | `fetch-one-step.png`、`fetch-command-list.txt`、`during-fetch-start.txt` |
| fetch 进行中点提交 `Add alpha file` | 提交详情切换到 `dec7222…`，进度条仍在。 | `during-fetch-commit.txt` |
| 点文件 `alpha.txt` | Diff 显示 `+alpha line one`，进度条仍在。 | `during-fetch-file.png`、`during-fetch-file.png.txt` |
| ⌘P 执行只读命令 `page.branches` | 切到 Branches 页，进度条仍在。 | `during-fetch-palette-page.txt` |
| ⌘P 搜索 `repo.push` | 行禁用，显示 “Wait for the current task to finish.”；此时远端引用尚未出现。 | `during-fetch-write-gate.png`、`during-fetch-write-gate.txt` |
| 等待完成 | 进度消失、Sync 恢复可用，`origin/acceptance-origin` 与 `upstream/acceptance-upstream` 均写入；Branches 页列出两条新远端分支，出现 “Fetched from remotes” 提示。 | `fetch-complete.png`、`after-fetch-refs.txt` |
| 完成后 ⌘P → Merge Branch | 远端引用可被选择（`origin/acceptance-origin`），确认完成后的 reload 已进入界面状态。 | `after-fetch-refs.txt` |

## 真实标签页：Fetch 期间切换

为避免系统目录选择器，使用从 `build/SuperGit.app` 复制、替换为 e2e 工作台二进制并重新 ad-hoc 签名的临时 bundle；两个仓库通过 LaunchServices 文件夹打开进入各自标签页。脚本为 `/tmp/sg-fetch-accept/tabs.py`。

| 步骤 | 结果 | 证据 |
| --- | --- | --- |
| 打开 `repo` 与 `other` 两个标签页 | 两个标签页均加载完成且空闲。 | `fetch-tabs.log` |
| 在 `repo` 启动 Fetch 后切到 `other` | 切换成功；`other` 显示自己的历史，不继承 `repo` 的进度。 | `during-fetch-other-tab.txt` |
| 切回 `repo` | 标签页恢复选中，后台 fetch 的进度条继续显示。 | `during-fetch-tab-return.png` |
| 等待完成 | 后台标签的 fetch 完成并 refresh，两个远端引用写入。 | `fetch-tabs-complete.png` |

## 未覆盖与说明

- **取消入口**：普通工作台（有分支栏）没有独立的 fetch 取消按钮；`Cancel` 只在无分支栏的空状态渲染。这是本次改动前就存在的行为，不属于本次范围；设计文档已按实际能力更正。
- **失败反馈**：本轮原生验收未单独制造 fetch 失败；失败路径由 `TestPaletteFetchFailureFeedback` 等单元测试覆盖。
- **网络与凭据**：仅使用本地 file 远端，未覆盖 SSH/HTTPS 认证与远端服务端行为。
- **主题**：本次不修改视觉 token 或布局，验收截图使用浅色主题；写入禁用状态和进度条沿用既有组件。
- **窗口尺寸**：真实工作台验收使用 1320 × 840；行为相关的 900 × 640 布局由 smoke 覆盖。

## 复现

```sh
make app lobe-e2e-build
python3 -I scripts/native-lobe-toolbar-sync-smoke.py
SUPERGIT_WINDOW_SIZE=900x640 python3 -I scripts/native-lobe-toolbar-sync-smoke.py
SUPERGIT_WINDOW_SIZE=1320x840 python3 -I scripts/native-lobe-toolbar-sync-smoke.py
```

隔离仓库与延迟 `uploadpack` 的搭建见 `docs/features/fetch-background-one-step.md` 的 Acceptance steps。
