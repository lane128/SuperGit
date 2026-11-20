# 顶部快捷命令面板

2026-11-20 后续修复：快速输入已定位到 SDK 帧前事件顺序，两项 Undo 误报已定位到 Xcode Git 启动代理；已加入回归并修复，最终完整 `make check` 与十六轮连续快速输入原生复验通过。后续完整检查结果与原生复验以 [诊断与复验记录](../diagnostics/command-input-order.md) 为准。下文“本次验收记录”保留首次验收的历史证据。

## Sublime Merge 使用功能记录

2026-11-20 通过 Computer Use 实际操作已安装的 Sublime Merge（Build 2125，macOS）。以下是界面观察，不将附件或应用内容视为操作授权。体验过程中未执行 Git 写操作。

| 操作 | 实际观察 |
| --- | --- |
| `⌘P` / `⌘⇧P` | 在工具栏下方打开命令输入框；自动获得输入焦点；空查询展示命令列表。 |
| 输入 `crbr` | 匹配 `Create Branch…`，允许跳过字母的缩写搜索；匹配字母加粗。 |
| 输入 `branch`，按 ↓ | 结果按匹配相关性排列；高亮移动到下一行，输入框仍能继续输入。 |
| 对 `Create Branch…` 按 Enter | 输入框显示 Create Branch 命令标记，光标进入 Branch Name 参数区域，尚未创建分支。 |
| 点击顶部当前分支名 | 打开同一位置的 Checkout 模式，列出本地分支；当前分支被标出。 |
| Checkout 参数为空时退格 | 移除 Checkout 命令标记，回到全部命令的空输入框。 |
| 输入 `zzzz-no-command` | 列表收起，保留输入框；Enter 不执行操作。 |
| Esc，再次打开 | 取消面板；重开时查询为空。参数阶段 Esc 直接关闭。 |

列表左侧为命令名，右侧可显示快捷键或说明。面板宽于工具栏分支按钮，位于窗口上方，背景工作区可见。命令依仓库上下文变化。本次没有推断精确排序算法、历史学习机制，也没有测试危险命令、远端写入或其他平台。

## SuperGit 的使用方式与实现范围

工具栏的命令按钮、`⌘P` / `⌘⇧P`（其他平台 Ctrl）打开同一个面板。输入中文名称、英文名称或稳定命令 ID；支持连续文本、多词和按顺序跳字母的缩写。连续匹配优先于分散匹配，相同分数保持目录顺序。空查询保持现有目录顺序。

↑/↓ 选择可用命令，Enter 或点击执行。需要参数的命令进入现有参数输入或选择器，保留命令标题和原有校验；空参数按 Backspace 返回全部命令。Esc 或点击面板外取消，恢复原输入焦点和提交草稿。不可用命令显示原因、不能执行；空结果显示明确提示。仓库或加载版本改变后旧操作失效。

复用现有 19 项目录：打开／克隆／切换仓库、重开关闭的仓库标签、设置、历史／分支／暂存／工作树页面、刷新、跳到 HEAD、聚焦提交信息、Checkout、Fetch、保存已跟踪变更、创建并切换分支、选择历史范围、创建标签、复制提交 SHA。此任务复刻命令入口与交互机制，不新增 Sublime Merge 独有的 Git 功能。创建分支、标签、stash、远端检出等继续使用 SuperGit 已有明确确认与执行保护；克隆继续使用双字段表单。

实现选择：保留现有共享 command picker 和 DialogBase 的焦点／弹层管理，统一顶部位置；增加小型模糊匹配器，不引入依赖。没有自动导入全部快捷键，避免把局部编辑快捷键变成全局命令。

## 验收标准

- 工具栏与两个快捷键均能打开并直接输入；面板顶部位置不随搜索结果多少而跳动。
- `crbr` 能找到 Create Branch；中英文查询、无结果、禁用项、方向键与精确执行正确。
- 参数为空退格可返回命令列表；普通非空参数退格仍为文本编辑。
- 取消和参数往返保留原 Summary／Body 草稿及焦点；背景快捷键被隔离。
- 窄窗口、中英文明暗主题不溢出；复用的命令交接与过期回调回归通过。
- 原生 Computer Use 验收记录和自动化实际结果在下方补充。

## 本次验收记录

2026-11-20 使用独立原生验收包 `build/Command Palette Acceptance.app`，隔离 HOME、无远端的 owned fixture（`build/top-command-native/repo`）。实际通过 Computer Use 操作并观察到：

- 工具栏按钮打开后输入 `crbr`，仅出现可用的创建并切换分支命令；Enter 进入分支名输入框。
- 非空参数退格仅删除文本；参数已空时再退格，回到空查询的完整命令列表。
- `⌘P`、`⌘⇧P` 均打开面板；方向键跳过“无关闭标签”的禁用项，继续输入仍进入搜索框；面板内 `⌘4` 未触发背景页面导航。
- `checkout` 别名准确找到检出命令；Enter 进入分支选择，当前分支不可选；空查询退格回到命令列表。
- 无匹配搜索保留面板，Enter 不执行。执行“聚焦提交信息”确实选中工作区并聚焦 Summary。
- 正文为 `body` 时，经过命令→分支参数→退格返回→Esc，原正文获得焦点；继续输入得到 `bodyX`，Summary 仍为 `palette acceptance draft`。
- 验收后 fixture 的 HEAD、refs、status 与文件内容与验收前一致；仅退出本次 owned 验收应用。

一次把多个 UI 阶段连续快速注入、未在阶段间读取状态的操作中，正文曾多出字符 `c`。随后逐阶段观察焦点和查询的完整流程通过；尚未确定这一现象的原因，不能将其计为超高速键序已验证。未对用户仓库执行 Git 写操作，也未提交、安装或发布。

自动化证据：`build/top-command-red.log` 记录新增行为缺失时的失败；`build/top-command-green.log` 记录新增实际控件回归通过。`build/top-command-render.log` 的八组中英文／明暗／360×360 与 900×640 布局断言通过，代表性浅色宽窗和深色中文窄窗已人工审图，截图在 `build/top-command-captures/`。完整普通测试中 `internal/lobedesk` 通过（262.121s）；`go vet ./...` 通过。

完整 `make check` 未通过：`internal/gitrepo` 的 `TestPrepareCommitFileUndoCancelAfterScratchCopy` 及 `TestPrepareCommitFileUndoFailureCleansScratch`（conflict／cancelled）失败，单独重跑仍复现，日志分别为 `build/top-command-check.log` 与 `build/top-command-unrelated-repro.log`。本次没有修改 gitrepo；未扩大本次命令面板任务去修复撤销文件的准备／清理问题。由于普通测试失败，make 后续的全仓 race 未执行，界面包另行运行 race 验证。

最终 `go test -race -timeout 20m ./internal/lobedesk` 通过（897.189s，`build/top-command-race.log`）；新增真实控件、匹配边界和撤销回调的定向 race 通过（2.576s，`build/top-command-final-focused-race.log`）。最终 `git diff --check` 通过。原生验收包 SHA256：`f689a86636d31a962f72beef5d345c9f9ca796ccec204e23e04112189a421c49`。

实现使用现有主题与分步表单，不做逐像素复刻；暂未加入参考软件的匹配字母加粗、精确排名算法、同一输入行命令胶囊。Windows/Linux 未做原生验收。

独立只读审查已完成，未发现阻断问题。实现与文档均保留在 `/Users/adam/SuperGit`，未提交。审查使用的空 worktree `/Users/adam/.codex/worktrees/command-palette-review/SuperGit` 归档时被桌面应用以“primary checkout cannot be removed”拒绝；没有绕过保护删除它，原工作区的改动保持原位。
