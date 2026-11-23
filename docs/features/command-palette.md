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

仓库工具栏中央是一个固定 360 pt 宽的命令栏。左端的命令图标和 `⌘P` / `⌘⇧P`（其他平台 Ctrl）打开全部命令。点分支名或领先／落后状态，面板直接进入 Checkout，空查询按退格回到全部命令。切换长短分支时命令栏位置和宽度固定，长名称从中间省略，悬停可读全名。输入中文名称、英文名称或稳定命令 ID；支持连续文本、多词和按顺序跳字母的缩写。连续匹配优先于分散匹配，相同分数保持目录顺序。空查询按当前语言的命令名排序。搜索结果中实际命中的标题字符以粗体显示，使用命令 ID 也能找到对应入口。

↑/↓ 选择可用命令，Enter 或点击执行。需要参数的命令进入现有参数输入或选择器，保留命令标题和原有校验；在参数为空时按 Backspace 返回上一步。多步命令会恢复已输入的参数和选中行，从第一步退回完整目录。Esc 或点击面板外取消，恢复原输入焦点和提交草稿。不可用命令显示原因、不能执行；空结果显示明确提示。仓库或加载版本改变后旧操作失效。

### 命令目录

逐项对照本机 Sublime Merge Build 2125 的 `Default.sublime-package/Default.sublime-commands`，75 个默认命令均能在 SuperGit 搜索并进入对应操作。另有首次原生截图中可见的 About。英文名称与参考目录逐项对应，六个带 Git 命令提示的条目也显示对应提示。SuperGit 保留已有的打开仓库、克隆、聚焦提交信息等快捷操作。中英文界面都可以用英文原名搜索。

| 类别 | 数量 | 可执行操作 |
| --- | ---: | --- |
| 分支 | 11 | Checkout、从 HEAD／标签／远端创建分支、重命名、删除本地或远端分支、合并、变基、设置或清除上游 |
| 同步与远端 | 6 | Fetch、Pull、Push，以及添加、重命名和删除远端 |
| 标签 | 4 | 创建、删除本地标签、删除指定远端上的标签、推送标签 |
| 工作区与 Stash | 9 | 暂存已修改文件、暂存已修改及未跟踪文件、取消暂存、清理未跟踪文件、应用补丁、保存／弹出／删除／清空 Stash |
| 历史与文件 | 11 | 逐行追溯、文件历史、浏览文件树，跳转至提交／分支／标签／Stash，隐藏分支、仅显示某个分支、恢复一个或全部隐藏分支 |
| 视图、设置与仓库 | 9 | 切换侧边栏、忽略空白、默认展开合并、选择主题、偏好设置、切换仓库、打开仓库文件夹、在终端或 Sublime Text 中打开仓库 |
| 子模块 | 2 | 添加子模块、打开已初始化的子模块 |
| Git LFS | 3 | 编辑跟踪模式、锁定文件、解锁文件 |
| Git Flow | 20 | 初始化和配置，开始或变基 Feature／Bugfix／Release／Hotfix／Support，完成或发布前四类主题分支 |

完整英文清单随回归测试保存。[Build 2125 命令清单](../../internal/lobedesk/testdata/sublime-merge-2125-palette.txt)

在 Pull 中选快进、合并或变基，再选远端及其分支。Push 提供普通推送、建立上游和带租约的强制推送，确认框显示源和目标。删除远端标签会读取实际推送目标上的标签，再校验选中的对象。Git Flow 的 Finish 只完成本地分支，Publish 单独执行远端推送。

View Tree 按所选提交读取目录、子目录和文件，Back 返回所在目录。文本预览上限为 1 MiB。二进制文件显示大小，子模块显示目标提交。隐藏某分支会从历史遍历中移除其独有提交，其他可见引用可达的提交继续显示。

编辑 LFS 跟踪模式直接修改对应的 `.gitattributes`。锁定与解锁需要 Git LFS 和支持锁服务的远端。Git Flow 需要 AVH，macOS 也会检测本机 Sublime Merge 附带的版本。外部编辑器入口需要已安装 Sublime Text。当前仓库状态或本机依赖未满足时，面板会解释原因。

清理未跟踪文件会先列出受影响路径，确认时重新核对内容。Git 忽略的文件及嵌套仓库受保护。暂存、删除分支或标签、推送等操作复用对应 Git 校验，仓库切换或引用移动会使旧选择失效。

### Fetch

搜 `fetch`（或「获取」）显示一个 `Fetch…` 入口。回车或点击进入参数选择，再次确认所选方式后才启动获取。

参数页默认选中 `Fetch`（后台获取全部远端）。还可选 `Fetch --prune`（同时清理远端已删除分支的本地跟踪引用）、`Fetch --tags`（同时获取全部标签）或 `Fetch from Remote…`（继续选择一个远端）。也可搜 `fetch prune`、`fetch tags` 或 `fetch remote` 直达各项。在空输入框按退格返回上一层，Esc 关闭面板。顶栏 Sync 菜单中的 `Fetch` 仍直接获取全部远端。

Fetch 在后台运行：进度仍显示在标题栏与分支栏，可随时取消；期间提交、推送、检出等写操作保持禁用，但历史、文件、差异、仓库标签页和命令面板的浏览命令继续可用。选择远端期间，后台仓库刷新和定时 Fetch 会等到面板关闭后继续，避免正在输入的远端名被刷新清空。

### Merge Branch

选择 Merge Branch 或「合并分支」后，直接输入分支名。列表用 `L` 标出本地分支，用 `R` 标出远端跟踪分支。当前检出的分支作为合并目的地，其他分支作为候选。方向键移动选中项时，面板底部同步显示完整来源引用与当前分支。按 Enter 或点击候选打开 SuperGit 的合并确认，确认框显示来源和提交短 SHA。取消保留原 HEAD。仓库处于分离 HEAD、有未提交改动或有进行中的 Git 操作时，面板说明禁用原因。

Merge Branch 参数页的布局依据用户提供的 Sublime Merge 截图。Build 2125 的内置命令目录确认了 `fetch` 和 `merge_branch` 的入口；SuperGit 在 Fetch 参数页提供全部远端的默认方式与指定远端选项。本轮 Computer Use 打开原应用的仓库选择器后超时，独立实例也未获得控制许可。上方首次实测表覆盖通用命令与 Checkout，本轮尚缺 Merge 参考参数页的现场复核。

实现选择：保留现有共享 command picker 和 DialogBase 的焦点／弹层管理，统一顶部位置；增加小型模糊匹配器，不引入依赖。没有自动导入全部快捷键，避免把局部编辑快捷键变成全局命令。

## 验收标准

- 工具栏与两个快捷键均能打开并直接输入；面板顶部位置不随搜索结果多少而跳动。
- `crbr` 能找到 Create Branch；中英文查询、无结果、禁用项、方向键与精确执行正确。
- 参数为空退格可返回命令列表；普通非空参数退格仍为文本编辑。
- 取消和参数往返保留原 Summary／Body 草稿及焦点；背景快捷键被隔离。
- 窄窗口、中英文明暗主题不溢出；复用的命令交接与过期回调回归通过。
- 原生 Computer Use 验收记录和自动化实际结果在下方补充。

## Fetch 与 Merge Branch 的 macOS 验收

在 900 × 640 和 1320 × 840 pt 窗口、115% 字号下，分别运行 `scripts/native-lobe-toolbar-sync-smoke.py`。每轮启动独立的 SuperGit 原生进程，只连接该轮新建的两个本地裸仓库。两种尺寸都通过下列检查。

- 在同一窗口检出长短两个分支，包括领先 102 和落后 1 的状态。命令栏宽度保持 360 pt，中心与窗口中心相差不超过 1 pt。
- 对四个 Fetch 命令逐个造出新的远端分支、过期的跟踪引用，以及一个常规抓取不会随分支取得的标签。获取全部远端的三个命令更新两个远端，清理与标签各自按模式生效；`Fetch from Remote…` 只更新所选远端。逐个核对新引用、清理结果和标签，当前 HEAD 和工作树都与操作前一致。
- 在合并页输入 `trial-2`，用方向键依次选中本地、origin、upstream 三个来源，核对每一步底部显示的完整引用。取消确认保留原 HEAD。重新选择有新提交的远端目标并确认，当前分支实际前进到选中的 SHA。
- 从 Sync 菜单执行 Fetch、Pull 和 Push。分别核对跟踪引用前进、工作树快进、裸仓库收到指定提交。取消 Push 保留裸仓库原引用。

脚本会输出该轮截图和原生操作日志的目录。`fetch-search.png` 记录 Fetch 搜索结果，`fetch-options.png` 记录获取参数页，`merge-preview.png` 记录合并参数页，`sync-menu.png` 记录同步菜单。

## 完整目录的验收记录

2026-11-20 在 macOS Apple Silicon 上用 SuperGit 原生窗口验收。脚本为每轮创建独立的工作树、两个裸远端和设置目录。900 × 640 和 1320 × 840 pt 都使用 115% 字号。验收入口为 `scripts/native-lobe-toolbar-sync-smoke.py`，其界面断言通过窗口的无障碍树读取，Git 结果由独立进程读取。

| 核验对象 | 检查方式 |
| --- | --- |
| 完整目录 | 与本机安装包逐行比对 75 项默认标题。两种窗口尺寸都从实际列表中读到每一项且只出现一次，另外读到 About。回归测试也验证中文界面能按英文原名查找。 |
| 键盘与分步输入 | 输入 `crbr` 命中 Create Branch。输入完整的 `stash.save` 进入保存范围。添加远端时从地址栏退格返回仍保留名称，再次进入并提交后用 `git remote get-url` 核对实际配置。 |
| 文件、暂存与 Stash | 在 View Tree 打开所选提交的文件并返回目录。分别暂存已跟踪和全部文件，取消暂存后检查索引与文件内容。使用空说明保存 Stash，核对工作区和 Stash 列表。 |
| Git Flow 与 LFS | 在本机可用的 AVH 上从窗口初始化、选择生产和开发分支并读取 Configure。通过 LFS 面板添加模式，用 `git check-attr` 核对结果。独立测试还对五类分支运行已支持的 Flow 操作，并使用真实 Git LFS 3.7.1 客户端对实现锁协议的本地测试服务执行锁定和解锁。 |
| 文件与引用保护 | 临时仓库回归覆盖同名分支与标签、移动的引用、远端更换推送地址、变化的 Stash 列表、清理前变更内容、忽略文件、嵌套仓库和被替换为符号链接的目录。 |
| 文件历史与外观 | 验证隐藏分支后按可见引用遍历与分页、固定提交的文件内容、不同差异视图的空白过滤，以及系统／浅色／深色外观与五套配色的独立持久化。 |

在 macOS 上复跑原生验收可使用以下命令。脚本把截图与日志路径输出到终端。

```sh
make e2e-build
SUPERGIT_NATIVE_BINARY="$PWD/build/supergit-e2e" SUPERGIT_WINDOW_SIZE=900x640 python3 -I scripts/native-lobe-toolbar-sync-smoke.py
SUPERGIT_NATIVE_BINARY="$PWD/build/supergit-e2e" SUPERGIT_WINDOW_SIZE=1320x840 python3 -I scripts/native-lobe-toolbar-sync-smoke.py
```

每轮生成 `command-catalog.png`、`push-options.png`、`git-flow-configure.png`、`tree-file.png` 等截图、完整目录 `command-catalog.txt` 以及操作日志。上表记录的 UI 核验对象是 SuperGit。对 Sublime Merge 的观察见本文开头；本轮 Computer Use 读取原版仓库窗口仍返回 `-10005`，未能逐条现场复核它的参数页和默认选项。

## 首次验收记录

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

首次实现使用现有主题与分步表单，当时尚未加入同一输入行的命令标签。后续版本在 Fetch、合并、Checkout 和单参数命令中增加了行内标签。后续目录扩展增加了匹配字母加粗。搜索采用 SuperGit 的确定性相关度排序。Windows/Linux 未做原生验收。

独立只读审查已完成，未发现阻断问题。实现与文档均保留在 `/Users/adam/SuperGit`，未提交。审查使用的空 worktree `/Users/adam/.codex/worktrees/command-palette-review/SuperGit` 归档时被桌面应用以“primary checkout cannot be removed”拒绝；没有绕过保护删除它，原工作区的改动保持原位。
