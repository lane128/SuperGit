# Fetch 后台化与一步获取

Status: Implemented and accepted (2026-11-23)
Date: 2026-11-23
Related: 用户反馈（fetch 期间界面不可操作、命令面板 fetch 多选 origin/upstream）；`8f41ee5`、`82d28a6`；[palette-fetch 计划](../superpowers/plans/2026-11-18-palette-fetch.md)

设计审查结论（2026-11-23）：已处理 6 个 must-fix——后台标签页关闭/替换、`selectForKey` 写路径误放行、refresh/retry 静默无效、`history.chooseRef` 选择无效、新命令未登记、受影响测试与 smoke 脚本；并补充停止会话的取消/完成清理。实现后独立审查再处理：返回 fetch 标签页时 `loading` 卡住、完成时打断命令面板/引用选择输入、空闲标签关闭确认与按钮规则不一致、smoke 脚本旧远端索引、fetch 刷新后 Amend 仍指向旧提交。最终采用“完成即刷新 + 面板同仓库保留输入”，避免延迟刷新与命令内预览竞争。

## Background and goal

两个问题：

1. **Fetch 期间几乎无法浏览。** `fetchRemotes` 走 `performDone` → `beginOperation`，把 `a.busy` 置为 true。`a.busy` 同时被约 150 处当作“排他操作”使用，于是 fetch 期间连只读操作也被禁用：提交详情里点文件、看 diff（`commit_file_selection.go` 的 `historicalFileOriginCurrent`）、工作区文件选择（`workspace_selection.go`）、文件/提交前后导航（`detail_focus.go`、`detail_tab_shortcuts.go`、`keyboard.go`、`history_navigation.go`）、仓库标签页切换（`repository_tabs.go`）、命令面板（`command_palette.go`）全部失效。这和应用里“fetch 是用户会走开去等的操作”的注释意图矛盾。
2. **命令面板 Fetch 多出两次选择。** `8f41ee5` 把命令面板 Fetch 扩成 6 个选项；选择单远端变体且远端多于一个时，再进入远端选择页（`palette_parameters.go` 的 `paletteFetchRemote`）。2026-11-18 的 palette-fetch 计划原本写明“Fetch all remotes、不暗示目标选择步骤”，这次交互是后续改动引入的。

目标：

- Fetch（手动与自动）保持后台运行：只读浏览全部可用，写 Git 的操作继续禁用。
- 命令面板 `Fetch` 一步获取全部远端，与顶栏 Fetch 一致；prune、tags 变体改为独立可搜索命令。
- 保留标题栏/分支栏的 fetch 进度与成功/失败反馈。普通工作台在替换 checkout 前没有独立的 fetch 取消入口（既有行为），本次不新增。

## Non-goals

- 不改变 `git fetch` 语义、prune 默认值、自动 fetch 的间隔与开关。
- 不允许提交、推送、检出、stash、合并等写操作与 fetch 并发。
- 不改变命令面板其他命令的参数流程；不新增快捷键。
- 不重做 fetch 进度 UI（仍使用标题栏与分支栏的现有进度），也不新增 fetch 取消入口。
- 不在本次范围内改变“只 fetch 某个远端”的普通路径；定向获取仅通过独立的 `Fetch from Remote…` 命令提供（见第 5 节）。

## Approach

### 1. Fetch 后台化：保留 `busy`，只对只读路径放行

不改变“fetch 期间 `a.busy = true`”的现有排他语义，因此所有写操作的门禁零改动。新增一个语义明确的判断：

```go
// exclusiveBusy reports whether a foreground operation is running. A fetch is
// background: it keeps busy set so writes stay gated, but reads stay open.
func (a *app) exclusiveBusy() bool { return a.busy && !a.fetching }
```

把**只读**入口从 `a.busy` 改为 `a.exclusiveBusy()`（fetch 期间为 false，其他操作期间仍为 true）：

| 能力 | 位置 |
| --- | --- |
| 提交详情点文件 / 打开 diff | `commit_file_selection.go` `historicalFileOriginCurrent` |
| 工作区文件选择、文件组选择 | `workspace_selection.go` `selectWorkspaceFile`、`selectWorkspaceGroup` |
| 文件 Tab 焦点与切换 | `detail_tab_shortcuts.go` `canFocusDetailTab` |
| 上下一个文件 | `detail_focus.go` `navigateHistoryFile` |
| 打开选中文件（外部程序） | `keyboard.go` `canOpenSelectedFile` |
| 列表方向键/翻页选择 | `keyboard.go` `moveSelection` |
| 历史前进/后退 | `history_navigation.go` `historyNavigationBlocked`、`canNavigateHistory` |
| 父/子/HEAD 提交跳转 | `parent_navigation.go`、`child_navigation.go`、`head_navigation.go` |
| 比较基准选择 | `revision_compare.go` `comparisonMenuItems`、`comparisonMenuAction` |
| 历史引用范围选择 | `palette_history_ref.go` 选择回调 |
| 聚焦提交信息输入框（只编辑草稿） | `commit_message_focus.go` `canFocusCommitMessage` |
| 提交右键中的作者/日期/周/月筛选 | `menu.go` 四个 `filter-*` 项 |
| 命令面板中的浏览/UI 命令 | 见下一节 |
| 仓库标签页切换 | 见下一节 |

明确**不**在放行范围：`workspace.go` 的 `selectForKey`（Enter/Space 会顺带暂存，属于写路径）、手动刷新与读取失败重试（最终走 `readRepositoryAtRoot`，它仍检查 `busy`，放行会变成静默无效）、`local_watch` 后台探测、Sync 菜单、分支选择器、快检、仓库切换。写操作继续只检查 `a.busy`，fetch 期间保持禁用，无需逐项改动。`performDoneAs`、直接调用 `beginOperation` 的路径也不改，因此并发写有兜底。

### 2. 仓库标签页：可切换，不可关闭/新建/换仓库

`repositoryTabs` 拆分两个判断：

- `canSwitch()`：允许 fetch 期间切换已打开的标签页；阻止 loading、仓库打开中、选择器、对话框、`exclusiveBusy()`，保留 `localWatch.closed` 检查。
- `canChange()`：维持现状（含 `busy`），继续约束新建标签页、打开/替换仓库、仓库选择器。

标签条使用 `canSwitch()` 决定 tab 行/激活/循环切换/横向滚动/拖拽排序；新建标签页、切换仓库按钮仍使用 `canChange()`。关闭按钮与会话选择仍要求 `canChange()`，并额外按**目标会话**判断 `!target.busy && !target.fetching && !target.loading`；这样活动标签 fetch 时不会出现“按钮可点、确认框却禁用”的不一致，也不能在切走后关闭正在 fetch 的后台标签页。

后台会话的生命周期补强：

- `stopRepositorySession` 调用 `cancelOperation()`，让被关闭/替换的会话停止正在跑的 fetch。
- fetch 完成回调在 `localWatch.closed` 时只清理 `busy`/`fetching`，不再发 toast 或 `reload()`。
- fetch 完成时立即 reload；命令面板的“命令列表”阶段与历史引用选择器在同仓库刷新时保留查询与输入，只刷新行内容。完成时的 reload 沿用激活语义检查 HEAD，HEAD/分支变化会取消 Amend 并保留草稿。
- 激活一个仍在 fetch 或正在读取的标签页时不再启动第二次校验；已有的 fetch/reload 负责刷新，并保持 `activationRefresh` 语义。
- `reload()` 在 `localWatch.closed` 时直接返回，避免已停止的会话继续读仓库。
- `canClose` 与标签页关闭菜单使用同一个目标会话判断：`busy`/`fetching`/`loading` 任一为真都不可关闭；fetch 期间仍不可关闭该标签页。

### 3. 命令面板：可打开，按命令区分只读/写入

`canOpenCommandPalette` 不再依赖某条命令的可用性，而是统一检查 `exclusiveBusy()`、loading、history reload、选择器/引用面板/对话框，因此 fetch 期间可打开。

`commandPaletteUnavailable` 保持“排他 busy → 等待”，另加：

```go
if a.fetching && !fetchSafePaletteCommand(id) {
    return a.t("commandPaletteWait")
}
```

`fetchSafePaletteCommand(id)` 是“fetch 期间安全”的白名单（不一定只读，但不写 Git）：`page.*`、`history.head`、`history.chooseRef`、`history.show/hide*`、`history.navigate*`、`commit.copySHA`、`commit.viewTree`、`file.history`、`file.blame`、`diff.ignoreWhitespace`、`view.*`、`window.settings`、`window.about`、`workspace.focusCommitMessage`、`repo.containingFolder`、`repo.terminal`、`repo.sublimeText`。这些命令中 `history.hide/show*`、`view.*`、`diff.ignoreWhitespace` 会写 `settings.json`，但不写 Git。

不放行：`repo.open`、`repo.quickSwitch`、`repo.reopenClosedTab`、`repo.clone`、`repo.reload`（最终走被 `busy` 阻塞的 `readRepositoryAtRoot`，放行会成为静默无效）、`app.checkForUpdates`、`open_submodule`、`repo.*fetch*`，以及 branch/tag/remote/stash/workspace/LFS/Git Flow 等写命令；它们显示现有 “请等待当前任务完成”。

`finishPaletteStep` 的 `a.busy` 改为 `a.exclusiveBusy()`，让 fetch 期间也能完成浏览参数步骤；写步骤会在 `commandPaletteUnavailable` 复查处被拒绝。

只读命令的实际入口也要同步放行，否则会出现“行可点但动作无效”：`palette_history_ref.go` 的选择回调与 `commit_message_focus.go` 改用 `exclusiveBusy()`；`menu.go` 四个筛选项同样改用 `exclusiveBusy()`。

### 4. 命令面板 Fetch：一步获取全部远端

- `repo.fetch` 的文案改为 **Fetch all remotes / 获取全部远端**（去掉省略号），`Run` 直接调用现有 `fetchAll()`；删除 `paletteFetchOptions` 与 `fetchOptions` 选项表和 `commandPalette.fetchOption`。
- 新增两个独立命令，仍走现有全部远端流程：
  - `repo.fetchPrune`：Fetch all remotes with prune / 获取全部远端并清理，调用 `fetchPruneAll()`。
  - `repo.fetchTags`：Fetch all remotes with tags / 获取全部远端和标签，调用新增的 `fetchTagsAll()`。
- `startFetch` 改为接收 `gitrepo.FetchMode`，三个调用点分别传 `FetchDefault`、`FetchWithPrune`、`FetchWithTags`。
- 新命令必须同时登记到 `commandPaletteTable()` 与 `paletteCommand()`；`commandPaletteUnavailable` 里现有的 `repo.fetch` 特判（`repositoryOpenPending`/`tabSnapshotUnavailable` 与无远端原因）扩展到三个 fetch 命令。
- `fetchRemotes` 的守卫补上 `repositoryOpenPending()`，与 `performDoneAs` 的其他调用保持一致。
- 参考命令名 `Fetch…` 仍作为英文搜索别名保留（`referenceCommandNames`），命令目录测试按别名解析 `repo.fetch`。
- 删除仅服务于选项页的 i18n 键（`paletteFetchOptions`、六个 hint）；新增 prune/tags 命令文案与成功提示。

### 5. 保留定向远端获取

参考命令目录里另有“指定 remote 的 Fetch”。新增独立命令 `Fetch from Remote…` / 从远端获取…：运行后进入现有 `paletteFetchRemote` 远端选择页，只使用 `FetchDefault`。普通 Fetch 不经过该页。该命令在 fetch 期间不可用。

## Alternatives and trade-offs

- **方案 A（未采用）：fetch 不设置 busy。** 需要把约 150 处写门禁逐一改成 `busy || fetching`，改动面大且容易漏；漏掉时写操作会静默失败或并发执行。方案 B 保留 `busy`，只放行明确的只读入口，风险更小。
- **方案 C（未采用）：fetch 完全隔离在独立 goroutine/状态。** 能彻底解耦，但会重做操作生命周期、取消和进度，超出本次范围。
- **方案 D（未采用）：保留 6 个 fetch 变体。** 与“一步获取”的用户选择不符；prune/tags 单独命令已覆盖常用变体。
- **定向远端获取**：作为可选命令保留能力，但普通 Fetch 不再被迫选择远端。

## Impact

- Modules / files:
  - `internal/lobedesk/app.go`：`exclusiveBusy`、`startFetch` 模式参数、`fetchTagsAll`、`fetchRemotes` 守卫、完成回调与 `reload()` 的停止会话处理。
  - `internal/lobedesk/palette_fetch.go`、`palette_parameters.go`、`palette_workflow.go`、`command_palette.go`：一步 Fetch、prune/tags 命令登记与特判、安全命令白名单、`finishPaletteStep`、删除选项/远端阶段（可选命令除外）。
  - `internal/lobedesk/repository_tabs.go`、`repository_tabs_drag.go`：`canSwitch` 与标签条/拖拽分流、目标会话关闭判断、`stopRepositorySession` 取消操作。
  - 只读门禁文件：`commit_file_selection.go`、`workspace_selection.go`、`detail_focus.go`、`detail_tab_shortcuts.go`、`keyboard.go`、`history_navigation.go`、`parent_navigation.go`、`child_navigation.go`、`head_navigation.go`、`revision_compare.go`、`palette_history_ref.go`、`commit_message_focus.go`、`menu.go`。
  - `internal/lobedesk/i18n.go`：fetch 命令与提示文案。
  - 测试与文档：`palette_fetch_test.go`、`palette_parameters_test.go`、`palette_catalog_test.go`、新增 fetch 后台/标签页测试；`scripts/native-lobe-toolbar-sync-smoke.py`；`docs/shortcuts.md`、`docs/features/command-palette.md`、`docs/workbench-command-map.md`。
- Compatibility:
  - 不改变 Git 数据、仓库配置或设置文件格式。
  - 命令 ID 保留 `repo.fetch`；新增 `repo.fetchPrune`、`repo.fetchTags`。依赖“Fetch…”文案的测试与文档同步更新。
  - 删除命令面板的 6 变体与远端选择页；单个远端的定向获取保留为独立命令 `Fetch from Remote…`。这是有意的行为变更。
- Data / update signing / permissions:
  - 不涉及签名、权限或凭据，也不新增持久化格式。
  - fetch 期间放行的部分 UI 命令会更新现有 `settings.json`（侧栏、主题、忽略空白、隐藏分支），但不写 Git 数据。

## Test plan

- Unit tests:
  - `exclusiveBusy` 在 `busy && fetching` 时为 false、`busy && !fetching` 时为 true。
  - fetch 期间只读入口可用：提交文件选择、工作区文件选择、文件/提交导航、提交信息聚焦、历史引用选择、命令面板安全命令；每个白名单命令至少验证一条实际可执行路径。
  - fetch 期间写入口仍禁用：文件行 Enter/Space 不暂存、`performDoneAs` 返回 false、命令面板写命令带 `commandPaletteWait`、快检/分支选择器/Sync 菜单/手动刷新与重试仍禁用。
  - 标签页：fetch 期间可切换；**非活动** fetch 标签页不可关闭；关闭/新建/打开仓库/仓库选择器仍被拒绝；`moveTab` 在 fetch 期间可重排。
  - 停止会话：关闭或替换正在 fetch 的会话时会取消操作；完成回调见到 `localWatch.closed` 后只清理标志，不发提示、不 reload。
  - fetch 生命周期：开始时 `busy && fetching`，完成/失败后两者都清零；重复 Fetch 不启动第二个；进度条仍可见；切走再切回仍能恢复完成状态。
- Regression tests:
  - 命令面板 `Fetch` 一步执行：Enter 后直接 `fetching`，不再出现选项页/远端页；仅一个远端和多个远端都成立（改写 `TestPaletteFetchOwnedInputs`、`TestPaletteFetchFailureFeedback`）。
  - `repo.fetchPrune`、`repo.fetchTags` 使用正确的 `FetchMode`（用本地 file 远端断言 `.git/FETCH_HEAD`、跟踪引用与 prune/tags 结果）。
  - 删除旧 `TestPaletteFetchParameterGeneration`；把 `TestPaletteFetchRemoteSurvivesBackgroundRefresh` 改写为“面板打开时后台刷新等待”，不再依赖远端选择页。
  - 更新 `TestPaletteFetchCatalogAndGuards`、`TestCommandPaletteReferenceCatalog`（`Fetch…` 按 `referenceCommandNames` 别名解析）。
  - 真实 fetch 进行中（本地 file 远端、延迟的 `uploadpack`）验证只读操作可用、写操作禁用。
  - 更新 `scripts/native-lobe-toolbar-sync-smoke.py`：移除 Fetch Options/六个变体与远端选择，改为一步 Fetch + `fetch prune`/`fetch tags` 命令。
- e2e / automation:
  - `make app` + 隔离临时仓库（origin + upstream，`remote.origin.uploadpack` 延迟约 8 秒）。
  - 点击 Sync → Fetch，在 fetch 进行中：切换仓库标签页、点提交、点文件、切文件 Tab、打开命令面板执行只读命令；以上均响应。
  - 同一时段点提交/推送/检出/Sync 菜单：保持禁用或显示等待原因。
  - ⌘P → `Fetch all remotes` 回车：不再出现选项页或 origin/upstream 选择；progress 在标题栏/分支栏出现。
  - 搜索 `fetch prune`、`fetch tags` 并执行，观察成功提示。
- Uncovered areas and reasons:
  - 真实网络认证、SSH 凭据与远端服务端行为；本次仍使用本地 file 远端。
  - 系统剪贴板、外部编辑器与原生窗口焦点；本次验收覆盖应用内交互。

## Acceptance steps

1. Build: `make app`
2. 准备临时仓库与两个本地裸远端；为远端设置延迟的 `uploadpack` 以稳定观察 fetch 过程。
3. Steps:
   - 顶栏 Sync → Fetch（或 ⌘P → Fetch all remotes）；在 fetch 运行期间依次操作提交列表、文件列表、文件 Tab、仓库标签页、命令面板只读命令，并尝试提交/推送/检出。
   - ⌘P 搜索 Fetch、fetch prune、fetch tags，确认各自一步执行。
   - 完成与失败各验证一次标题栏/分支栏状态与提示。
4. Expected results:
   - Fetch 期间只读操作全部响应，写操作保持禁用；标签页可切换；fetch 完成后后台会话刷新。
   - 命令面板 Fetch 无选项页、无远端选择；prune/tags 为独立命令。
   - 进度、取消、成功/失败提示与原来一致。
5. Evidence location: `docs/acceptance/2026-11-23/`

## Risks and rollback

- Risks:
  - 只读门禁改动过宽，可能意外放行写操作。缓解：只把明确安全的入口改为 `exclusiveBusy()`；写路径继续依赖 `busy`；`performDoneAs` 与直接 `beginOperation` 仍是兜底。
  - 命令面板白名单遗漏写命令，导致 fetch 期间出现可点但静默失败的条目。缓解：非白名单一律返回 `commandPaletteWait`；复查全部 palette ID，并为每条白名单命令补可执行路径测试。
  - 后台标签页生命周期：切走后关闭/替换会话会取消 fetch；完成回调与 `reload()` 必须识别 `localWatch.closed`，否则已停止会话继续读仓库或弹提示。
  - 切到另一个标签页后可以在那里写 Git，而原标签页仍在 fetch。不同仓库无冲突；同一仓库的不同工作树由 Git 自身的 ref 锁兜底，失败会按正常 Git 错误提示，不做跨会话串行化。
  - 自动 fetch 与手动 fetch 的互斥仍依赖 `busy`，行为不变；自动 fetch 在面板/对话框打开时仍等待。
- Rollback: `git revert <commit>` 并 `make app` 重建。

## 确认结果

- 保留独立命令 **Fetch from Remote… / 从远端获取…**（第 5 节）：普通 Fetch 仍是一步获取全部远端，只有显式选择该命令时才出现远端选择页。
