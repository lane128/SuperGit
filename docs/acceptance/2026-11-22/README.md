# SuperGit 功能与界面验收记录

**修复复验通过。** 初验发现的两项问题已在本地工作区构建中关闭。双版本文件在两个方向各连续刷新两次后，标签、补丁和操作均保持正确。980 × 640 pt 窗口首次打开历史菜单时，19 项均完整可见。系统目录选择器的确认、外部编辑器真机启动与生产凭据交互仍待人工验收。

原始截图、PDF、问题清单和证据包保存在当次验收的本地目录。下文记录验收步骤、结果和复验入口。

## 验收对象

| 项目 | 本轮对象与依据 |
| --- | --- |
| 日期与平台 | 2026-11-22，上海时间。macOS 27.2.1（26B131），Apple Silicon，Go 1.27.3，Git 2.54.0。 |
| 被测版本 | 默认入口 `cmd/supergit`，工作树快照时间 11:26:15，基准提交 `7f6663a1e3efef4db3d9cd1b8443e4b4da5ed0ce`。当时已有未提交改动，输入文件 2,706 个，快照 SHA-256 为 `37091defe87f2268a6bcd4c79983dffac84882a3492bc7ecc19cef2a693064e9`。快照清单见证据包 `metadata/snapshot.json`。 |
| 原生测试构建 | 从当时的工作树隔离构建标准包及带 MyGo 自动化入口的默认工作台。隔离源目录无 Git 元数据，标准包版本回落至 0.1.0（1）。真实窗口原图均通过应用内原生自动化获取。 |
| 安装版 | `/Applications/SuperGit.app` 为 0.1.0（25），程序 SHA-256 为 `65bc343ee4c059374a74a08ba31df9b4f709b81237ee3e9580c35cba7cbb4538`。与测试前工作区 build 中的既有包一致，和本轮从工作树新构建的二进制不同。严格验签、启动及用隔离仓库开窗均通过，Git 状态一致。系统拒绝直接截取该窗口，故其记录限于上述操作。 |
| 数据与远端 | 新建测试仓库、独立偏好与自有本地 bare 远端。LFS 锁使用环回测试服务。写入后按场景核对提交树、父提交、引用、Index 条目或工作区字节。 |
| 平台边界 | 本轮原生交互覆盖默认 macOS 工作台。旧版回退界面的 Go 用例随项目运行，其原生 UI、Windows 与 Linux 没有本轮桌面结论。 |

## 初验发现

### SG-01　刷新后从未暂存跳到已暂存

**结果为未通过。** 若用户继续使用右侧操作，看到的版本和可选的操作都已经改变。此次复现中的刷新没有改变 HEAD、Index 条目或测试文件内容。

1. 为 `src/a.go` 准备三种内容。HEAD、Index 和工作区分别保存历史版本、`index version` 和 `working version`。
2. 依次点开文件列表的未暂存行和已暂存行，在 Files 中折叠 Modified Files。
3. 点击 `a.go · Unstaged` 标签，确认显示 `+working version` 和 Stage file。
4. 按 ⌘R 并等待读取结束。控件快照中选中项变为 `a.go · Staged`，内容为 `+index version`，操作变为 Unstage file。同一隔离仓库连做两次，结果一致。

**复验条件**为活动版本、补丁内容与动作目标均保持一致。附加检查可对调两个版本的选择顺序，并在展开与折叠的文件分组下重复。证据在 PDF 第 2 页、图册第 9 组及 `logs/current-probe-version-refresh.log`。

### SG-02　窄窗菜单末项初始未完整显示

**结果为可达，保留界面观察。** 在 980 × 640 pt、有 35 条分支和两个远端的测试窗口打开历史菜单。19 项菜单的末项 Reset 起点为 629 pt，高 32 pt，底边超出窗口 21 pt。向下移动焦点后该项位于 604–636 pt，回车打开正确确认，取消后 Git 状态一致。

若需关闭该观察项，应明确长菜单采用的滚动提示，并复核首次打开和键盘、滚动到末端的体验。原始坐标见 `metadata/lobe-menu-access-wrm8wz0i/clipped.json`。同目录 `results.json` 的最终可达结果须结合初始坐标查看。证据在 PDF 第 3 页及图册第 9 组。

## 初验逐项记录

“原生通过”表示在真实 macOS 测试窗口完成所列路径。“原生＋自动化”表示同项场景含应用窗口、组件或 Git 层的不同验证。“部分验证”列出尚缺的系统结果。证据名以 `Test` 开头时可在 `自动化用例.csv` 精确搜索，其他名字对应 `logs/名称.log`。

### 仓库、编辑与工作区

| 编号 | 功能与结果 | 实际检查 | 证据名 |
| --- | --- | --- | --- |
| 01 | 打开与多仓库标签<br>**原生通过** | 从欢迎页和已有工作台接收文件夹。同一规范路径复用原标签，重启后标签顺序和两份提交草稿一致。 | `standard-bundle`<br>`TestRepositoryTabDragPersistsOrderWithoutSwitching`<br>`TestReopenClosedRepositoryRestoresPositionWithFreshState` |
| 02 | 系统目录选择器<br>**部分验证** | 看到了只选目录的 macOS 面板，取消后回到欢迎页。确认选择时驱动超时，未取得从面板返回工作台的端到端证据。 | `current-probe-picker-input`<br>`TestOpenRepositoryShortcutLoadsChosenDirectory` |
| 03 | 新建与克隆<br>**原生通过** | 在含中文和空格的路径建库，回车暂存文本与二进制，首个提交为零父节点。克隆失败可重试，已有目标目录的数据保留。 | `current-probe-first-commit`<br>`current-lobe-clone-smoke` |
| 04 | 提交与拆分改动<br>**原生通过** | 分别暂存 ALPHA、BETA，撤回 ALPHA 后只提交 BETA，再提交剩余内容。两次提交的树与预期相同。 | `standard-bundle`<br>`current-lobe-hunk-enter-smoke` |
| 05 | 补丁、图片与 Amend<br>**原生＋自动化** | 切换统一与并排补丁。原生图片前后分别为 200 × 120 与 240 × 150。Amend 实际改写 HEAD，父提交不变，提交中包含准确的暂存内容。 | `current-probe-media-amend`<br>`TestAmendPrefillsCompleteHeadMessageAfterReload`<br>`TestSideBySideStagesAndUnstagesOnlyTheRightCell` |
| 06 | 丢弃、批量与按行操作<br>**原生＋自动化** | 原生按行暂存、撤回后核对 Index。自动化验证取消丢弃、拒绝陈旧选区和恢复删除文件；千文件视图验证首末条目。 | `current-lobe-long-patch-baseline_--worktree_--stage-unstage_--stage-line_--selection-bar`<br>`current-lobe-overview-sections-smoke`<br>`TestFileDiscardShortcutFrozenTargetsAndStaleContent` |
| 07 | 刷新同路径双版本<br>**未通过** | 工作区与暂存区的同名标签可同时打开。折叠已修改分组后，在工作区标签刷新，两次都跳到暂存版本。详见 SG-01。 | `current-probe-version-refresh` |
| 08 | 本地变化与仓库导航<br>**原生通过** | 从外部修改工作区、Index 和分支后，界面读取新状态。父仓库、子模块、链接工作树往返两轮，逐一核对各自 HEAD 与草稿。 | `current-local-watch-smoke`<br>`current-probe-navigation` |

### 历史、远端与 Git 操作

| 编号 | 功能与结果 | 实际检查 | 证据名 |
| --- | --- | --- | --- |
| 09 | 提交图与历史浏览<br>**原生通过** | 243 条提交的历史用例核对图中顺序、Stash 和侧栏定位。筛选、选引用、搜索、读失败重试和 Blame 切换均已执行。 | `lobe-unified-history`<br>`current-probe-menu` |
| 10 | 摘要、父提交与复制<br>**原生＋自动化** | 900 × 700 窗口的哈希、作者、日期值左边缘对齐，两条父提交链接均可跳转。身份、日期及剪贴板失败反馈由自动化覆盖。 | `summary`<br>`TestHistorySummaryMetadataColumns`<br>`TestCommitAuthorIdentityMenuCopiesCapturedAuthor`<br>`TestCommitDateMenuCopiesCapturedDisplayValue` |
| 11 | 分支、引用与标签<br>**原生＋自动化** | 切换并筛选 35 条长分支，验证非当前分支推送、改名及删除。附注标签推送后保留其对象，删除本地标签时远端仍在。 | `current-lobe-reference-smoke`<br>`standard-bundle` |
| 12 | 远端与同步<br>**原生通过** | 验证 Fetch 选项、Push 确认与取消、快进 Pull、分叉拒绝、过期租约拒绝及 prune。在自有 bare 远端核对实际引用。 | `standard-bundle`<br>`current-lobe-sync-smoke`<br>`current-lobe-force-prune-smoke` |
| 13 | 合并与历史改写<br>**原生通过** | Merge 核对两个父提交，Rebase 核对新父提交；Cherry-pick 和 Revert 确认页核对目标与产物，并检查取消路径。 | `current-lobe-reference-smoke`<br>`current-lobe-integrate-smoke` |
| 14 | Reset、Undo 与操作恢复<br>**原生通过** | 分别执行 soft、mixed、hard Reset 和 Undo。外部先推进 HEAD 后拒绝旧目标，另验进行中操作的继续、跳过与中止保护。 | `current-lobe-reset-smoke`<br>`current-lobe-reset-smoke_--stale`<br>`current-lobe-operation-smoke`<br>`current-lobe-operation-stale-smoke` |
| 15 | 冲突编辑与两侧选择<br>**原生通过** | 文本结果写入 Index 并完成双父提交。残留冲突标记会提示并保留未合并状态。另测二进制、删除冲突及 Rebase 两侧取舍。 | `current-probe-editor`<br>`current-lobe-conflict-side-smoke_--shape_binary`<br>`current-lobe-conflict-side-smoke_--shape_deletion`<br>`current-lobe-conflict-side-smoke_--rebase-continue` |
| 16 | Stash 生命周期<br>**原生通过** | 对含暂存、未暂存与未跟踪文件的仓库执行保存、应用、弹出与删除。重叠失败保留 Stash，三方冲突保留 Index 中的各阶段。 | `standard-bundle`<br>`current-lobe-operation-smoke` |

### 设置、扩展与系统集成

| 编号 | 功能与结果 | 实际检查 | 证据名 |
| --- | --- | --- | --- |
| 17 | 工作树与子模块<br>**原生通过** | 创建、切换和移除工作树，锁定、脏或含忽略文件时拒绝移除。更新子模块到父仓库记录的版本，脏子模块保持原内容。 | `current-probe-worktree`<br>`standard-bundle`<br>`current-probe-navigation` |
| 18 | 命令面板与扩展<br>**原生＋自动化** | 清点 75 项目录及 About。抽测目录树、LFS 跟踪与 Git Flow 初始化、配置。Flow 工作流和 LFS 锁使用隔离远端或本地服务补测。 | `standard-bundle`<br>`TestPaletteExtensionFlowWorkflows`<br>`lfs-supplement` |
| 19 | 设置与多窗口<br>**原生通过** | 覆盖外观、历史、Git、快捷键和关于页。检查重置确认、偏好写入失败与恢复，分别从主窗口和设置窗口触发确认。 | `current-lobe-settings-layout-smoke`<br>`current-lobe-settings-layout-smoke_--persistence`<br>`current-lobe-window-dialog-smoke` |
| 20 | 语言、主题与分栏<br>**原生通过** | 18 组窗口组合各验五种状态，检查当前文件标签和关闭按钮。三个分栏分隔线可拖动、重启恢复，文件列双击可恢复默认。 | `current-probe-final-matrix`<br>`current-probe-divider` |
| 21 | 菜单与键盘<br>**有观察项** | 文件行及片段标题的回车操作有效。980 × 640 窗口历史菜单末项初始超出 21 pt，经向下导航后可见，回车打开确认。详见 SG-02。 | `current-probe-menu`<br>`current-lobe-hunk-enter-smoke`<br>`current-probe-first-commit` |
| 22 | 规模与异步输入<br>**原生通过** | 32,002 条提交完成加载。另测 10,003 个跟踪文件、8,000 行长补丁。在受控慢读期间，滚动、搜索、取消、重试连续三轮完成。 | `current-lobe-large-history-performance`<br>`current-probe-interleaved_--input-pressure`<br>`current-lobe-performance-baseline_--with-submodule_--batch-files_1000` |
| 23 | 外部程序与身份认证<br>**部分验证** | 外部程序的路径参数、失败反馈已有自动化。Finder 目录投递已走通。Sublime Text 实际启动、生产 SSH／HTTPS 凭据交互仍待真机确认。 | `TestExternalOpenPreservesPathsAsArguments`<br>`TestExternalOpenFailureKeepsBoundedDiagnosticOutput`<br>`standard-bundle` |
| 24 | 构建与安装包<br>**分层验证** | 从工作树构建默认应用并验签。安装的 0.1.0（25）通过签名、窗口启动及仓库状态检查。完整交互与截图使用工作树测试构建。 | `production-build`<br>`production-signature`<br>`current-installed`<br>`sdk-snapshot` |

## 界面与视觉验收

每种原生窗口组合均在最终加载状态保留长提交说明、横向溢出的 13 个文件标签、空提交、未暂存和已暂存五张图。测试点验活动文件标签和关闭按钮、提交草稿、已暂存／未暂存的选中态与内容，并为每组核对 Git 指纹。矩阵图册包含 18 组共 90 张。尺寸为逻辑点，原始 PNG 的像素尺寸单列在图像索引中。

| 内容宽 × 高 | 字号 | 语言及主题组合 | 原生组合数 |
| --- | --- | --- | ---: |
| 980、1320、1440 × 640 | 100% | 每个宽度下的中文、英文与浅色、深色全部组合 | 12 |
| 980 × 640 | 115% | 中文浅色、英文深色 | 2 |
| 1440 × 640 | 115% | 中文深色、英文浅色 | 2 |
| 900 × 640 | 90% | 中文浅色、英文深色 | 2 |
| 1600 × 640 | 90%、100%、115% | 两种语言、两种主题，每种组合均渲染 | 12 组离屏摘要，单独计数 |

另测了 900 × 700 pt 下窄窗摘要的字段与两个父提交、设置五个分类、通知错误详情、三处分隔线和多窗口确认单归属。原生窗口最大可用宽度为 1440 pt，1600 pt 由相同 UI 的离屏渲染器补测。另有 6 张冲突编辑器离屏图。未执行 VoiceOver 人工朗读及真人视觉偏好评审。

## 自动化运行结果

| 验证入口 | 实际结果 |
| --- | --- |
| `go test -count=1 -json github.com/egoist/mygo/ui ./...` | 退出 0。项目 1,483 个顶层测试通过，LFS 锁测试因环境未发现 git-lfs 跳过，随后用 Git LFS 3.8.1 单独运行通过。项目合计 1,484 个。 |
| UI SDK 常规用例 | 1,315 个顶层测试通过，2 个跳过。分别为 `TestEditorHelper` 与 `TestListMovesWithoutChoosing`。 |
| `go test -count=1 -race -json -timeout 20m ./...` | 退出 0。项目 1,483 项通过，LFS 锁用例在可用 LFS 环境下单独以 `-race` 运行通过。 |
| UI SDK 竞态用例 | 1,313 项通过、2 项跳过。此次命令另排除 `TestDiffFrameAllocs` 与 `TestPathContains` 两个分配测试，二者的常规用例通过。 |
| `go vet github.com/egoist/mygo/ui ./...` | 退出 0。 |
| `make app e2e-build`、严格验签、MyGo 输入快照 | 标准包与测试程序构建成功，标准包严格验签通过。MyGo 输入快照的 1,969 个文件通过校验。 |
| `make native-check` | 6 组均通过，覆盖仓库标签、工具栏和同步、日常提交流程、Stash、冲突两侧选择及子模块。 |
| 命令面板 | 在原生选择器中核对 75 个参考命令及 About。各命令组的实际操作、自动化和外部依赖证据在证据包 `命令覆盖.csv` 中区分。 |

### 规模与并发样本

| 场景 | 本机实测 |
| --- | --- |
| 大历史 | 32,002 条提交，64 条分支，额外 512 个引用。从启动到完整载入 0.566 秒。可视区同时有 11 条历史行，搜索往返 0.026 秒。此项为单次样本。 |
| 大仓库与子模块 | 12,025 条提交，10,003 个跟踪文件，含 1,000 个跟踪文件的子模块。批量改动、恢复、刷新及子模块状态连续三轮完成。 |
| 超长补丁 | 8,000 行变更，三轮跳到补丁尾部、选择、按行暂存和撤回并核对内容。 |
| 宽历史交互 | 使用测试读取闸门保证滚动、搜索发生在加载中，三轮均完成取消、恢复与重试。本项验证交互时序。 |

### 失败记录的归属

- 脚本起始页、入口或控件发生变化。旧脚本会等待 Changes 页、初始 Commit message、内联 Pop 按钮，或预期跨库切换清空各仓库草稿。按现有 History 入口、Stash 菜单及标签草稿模型重测，保留原脚本退出码和补测脚本。
- 文件分隔线的默认宽度随其余两列的占用变化。补测确认双击后存储的文件列宽度清除，重启后的计算值一致。
- 初测请求 1600 pt 原生窗口时被系统收窄，加载速度也使旧慢读脚本错过并发时段。分别用离屏渲染和受控读取重测。空提交截图以最终空态文字出现为准。
- 系统选目录的两次尝试均能验证取消。第一次确认动作未被系统面板接受，第二次没有取得完成反馈。随后尝试用桌面控制连接独立验收窗口，控制会话过期，无法点按确认。目录投递和目录选定回调均通过，原生确认仍标记为部分验证。
- 可重复的双版本刷新失败归入 SG-01，窄窗菜单首次显露归入 SG-02。两者均附原始截图、控件快照和后续状态检查。

## 证据包使用

解压后打开 `index.html` 可分组查看 **247 张原始 PNG**。第 9 组为上述两项发现，第 10 组为原生窗口矩阵，第 11 组为离屏图。索引中列出每张图的像素宽高、原始路径和 SHA-256。

- `自动化用例.csv` 逐条记录项目与 SDK 的通过、跳过及子用例，支持按包、运行类型和名称筛选。本文计数只取顶层测试。
- `运行记录.csv` 和 `logs/` 保留运行入口、原始日志、失败与重跑记录。临时目录名属于本轮运行。命令覆盖表逐项区分目录存在与同组功能行为的证据。
- `images/` 保存原图及相应状态快照。`probes/` 保存与本轮有关的原生补测脚本。`metadata/` 保存工作树指纹、安装版记录、关键原始测量与试验数据。
- 复验 SG-01 可按上文四步直接在隔离仓库操作。自动化脚本为 `probes/native-probe-version-refresh.py`，采用当前仓库的原生自动化构建与测试 Git 仓库。

## 放行条件

SG-01 与 SG-02 已完成修复复验。可用 [文件版本刷新脚本](../../../scripts/native-detail-tabs-smoke.py) 的 `--refresh-only` 模式和[历史菜单脚本](../../../scripts/native-lobe-menu-access-smoke.py) 重测。发布前仍需在目标安装包上人工完成系统目录确认与所需的外部编辑器、凭据流程。
