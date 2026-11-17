---
name: supergit-ui-design
description: Use when designing, implementing, reviewing, or verifying SuperGit macOS interfaces in Go/MyGo, including themes, buttons, settings, dialogs, Toast, workbench layout, Diff, and history graphs. Follow the project's current UI handoff and Git interaction contracts.
---

# SuperGit UI 设计与实现

## 读取当前规范

从 SuperGit 项目根目录打开 `design/workbench-redesign/README.md`，再读 `design/workbench-redesign/UI-Spec.md` 中与本次任务对应的章节。以当前规范的规格、实施范围和验收条件为准。需要比对视觉时，按交接入口查阅 `SuperGit-Design-Reference.pdf` 对应页码。若项目不在当前工作区，先定位包含这些文件的 SuperGit 项目。

| 改动 | 规范章节 |
| --- | --- |
| 配色、字体、按钮、焦点 | 基础规格、按钮与控件 |
| 设置、确认表单与反馈 | 设置窗口、弹窗、Toast、内联提示与进度 |
| 工作台、暂存状态、Diff、历史 | 工作台与 Diff、历史线图 |

## 实现或审查

1. 对照 `UI-Spec.md` 的实施范围和源码映射，确认画稿中的目标设计与现有功能。只实现用户请求的范围。跨层改动时先处理主题和共用控件，再接入各页面。
2. 以 `cmd/supergit-lobe` 为预览入口，在 `internal/lobedesk` 复用 Go/MyGo 的现有主题、控件与状态。按规范实现无描边填充按钮、深浅主题、禁用原因和键盘焦点。控件尺寸与色值从项目规范取用。
3. 复用 `internal/gitrepo` 和已有命令守卫。涉及 Git 写入时保留仓库、路径、暂存版本或 SHA 的目标校验。历史 Diff 保持只读，Toast 和弹窗保留发起窗口归属及过期目标保护。当前动作与已注册快捷键使用同一可用性判断。
4. 把三栏重排、最近仓库、提交快速预览与历史折叠视为可独立验收的需求。按照规范和用户当前授权确定接入范围。

## 验收

参考 `UI-Spec.md` 的验收表和 `docs/e2e-automation.md`。运行与改动范围相符的 Go 测试。实际界面变更优先覆盖浅深色、中英文、最小窗口、键盘焦点和对应状态。涉及 Git 写入时，在隔离临时仓库核对实际目标与结果。说明已执行的检查及原生验证情况。
