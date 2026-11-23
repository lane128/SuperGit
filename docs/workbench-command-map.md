# 工作台命令与状态能力盘点

日期：2026-11-15。对应 [下一轮工作台计划](superpowers/plans/2026-11-15-workbench-next-improvements.md) 的 N05 / Task 6。本文记录当前 Lobe 入口、SDK 能力和后续候选范围；候选系统菜单与新增状态区域尚未实施。

## 范围

用户提出的“状态栏上面的功能”尚未明确是窗口顶栏、macOS 系统菜单栏，还是窗口底部状态区。本轮先盘点已有能力，继续完成已明确的工作台需求。当前没有新增底部状态栏，也没有新增自定义应用菜单。

下表中的命令 ID 仅列出 `shortcutTable` 已有 ID；只有按钮或方法的动作标为“无统一 ID”，避免把规划中的命令当成现有实现。

## 窗口顶栏与工作台

| 功能 | 当前位置 | 已有 ID / 方法 | 可用条件与行为 | 状态 / 空缺 |
|---|---|---|---|---|
| 仓库身份、仓库菜单 | 顶栏左侧 | 仓库名菜单；无统一 ID | 已打开仓库时显示身份，保留相关仓库操作 | 侧栏重复身份已移除；不是删除顶栏身份 |
| 分支选择 | 顶栏居中命令栏的分支名 | `branch.quickCheckout` | 结合当前仓库、分支列表及现有切换守卫 | 已有选择入口；系统菜单未接入 |
| ahead / behind | 分支选择旁 | `status.Ahead` / `Behind` | 至少一项非零时显示数量和说明 | 已有只读状态 |
| 未完成 Git 操作 | 顶栏状态按钮、Commit 操作卡 | `selectWorktree` / Continue / Skip / Abort | 按 merge / rebase / cherry-pick / revert 的现有状态提供动作 | 顶栏跳转到实际操作卡，危险操作保留确认 |
| Fetch | 顶栏 Sync 菜单直接获取全部远端；命令面板的 Fetch… 打开参数选择，含普通获取、prune／tags／指定远端 | `fetchAll` / `fetchPruneAll` / `fetchTagsAll` / `repo.fetch*` | 有仓库、有 remote，且不在 loading / busy | 后台运行：只读浏览保持可用，写操作禁用；使用已有凭据机制；缺少 remote 有禁用说明 |
| Pull | 顶栏右侧 | `pullFF`；无统一 ID | 有仓库、有 upstream，且不在 loading / busy | 仅 fast-forward；不隐式合并 |
| Push | 顶栏右侧，右键有 force-with-lease | `confirmPush` / `confirmForcePush`；无统一 ID | 有仓库、remote、已检出分支，且不在 loading / busy | 复用已有确认，不能由新增入口绕过 |
| 刷新 | 顶栏右侧、快捷键 | `repo.reload` / `reload` | 有仓库，非 loading / busy | 快捷键与按钮均经过状态守卫 |
| History / Branches / Stashes / Worktrees | 侧栏 / 页面 / 快捷键 | `page.history` / `page.branches` / `page.stash` / `page.worktrees` | 打开对应页面；列表动作另受仓库与读写状态约束 | 保留页面分工 |
| 设置 | 侧栏、快捷键 | `window.settings` / `openSettingsWindow` | 创建或前置独立设置窗口 | 当前没有自定义系统菜单 Settings 项 |
| Commit / Amend | Commit 表单、快捷键 | `repo.commit` / `commit` | `commitDisabledReason`、`canCommitShortcut`；身份、标题、Index、冲突及操作状态共同决定 | 当前仅 Index 写入；hooks 保留；窗口焦点必须继续受控 |
| 显示 / 隐藏详情 | History 页头 | `setHistoryDetailsHidden`；无统一 ID | 可恢复现有详情及活动标签 | 系统 View 菜单候选，尚未接入 |
| 图谱、文件列、patch 显隐与字号 | 设置窗口 | 既有设置属性；无统一 ID | 持久化设置；隐藏内容不应继续接受后台编辑快捷键 | 不新增重复开关 |
| 恢复布局 | 设置窗口、分隔条双击 | `resetLayout` / 各 `splitDrag` | 全部恢复清除三个分栏偏好；单条复位保留其他偏好 | 复用已有持久化模型 |
| 搜索 | History / Branches 搜索框 | `search.history` / `search.branches` | 文本框接收输入；搜索不应触发文件写入 | History 本地搜索基于当前已加载范围，完整遍历继续推进 |
| 列表导航 / 激活 | 当前列表 | `list.*` / `page.activate` | 文本框优先处理编辑键，当前页面及对话框守卫生效 | 历史虚拟行需保留全局索引和目标揭示 |
| 文件 Stage / Unstage | Files、文件头、右键、快捷键 | `file.stage` / `file.stageShift`；按钮复用 `prepareFileStage` | 不可变 root / 代次 / 原始路径 / staged 身份；冲突及受保护子模块禁用 | 同一路径的两个版本独立；有效 gitlink 指针支持整文件 Stage / Unstage，子模块内部 dirty 本身不伪称可暂存 |
| Discard / 删除未跟踪文件 | 文件头 / 右键 | `prepareDiscardTargets`；无统一 ID | 异步准备快照、明确确认、确认时再校验内容与 Index | 历史只读；不静默删除未跟踪内容 |
| Stage all / Unstage all / Discard all | 总览、Files 对应入口 | `batchFileTargets`；无统一 ID | 暂存含修改和未跟踪；取消暂存仅 Index；全部放弃仅 tracked unstaged | 冲突阻止批量；Stage / Unstage 支持有效 gitlink 指针，内部 dirty-only 不阻止其他文件 Unstage；Discard 继续保护子模块，确认列明排除范围 |

## 状态、取消与重试

| 情况 | 当前反馈 | 取消 / 重试范围 | 不应混用的入口 |
|---|---|---|---|
| 正在执行 Git 写命令或 Fetch | 顶栏 200ms 延迟后显示 spinner 和具体操作文字 | 顶栏取消调用 `cancelOperation`，取消当前 operation context；命令通常还有三分钟超时 | Fetch 期间只读浏览与命令面板浏览命令保持可用；写命令继续禁用。不把失败写操作自动重试 |
| 仓库打开 / 刷新失败 | 页面失败提示及 Retry | 重试仓库读取，失败信息保留至当前请求完成 | 不以刷新代替确认一次新的远程写入 |
| 完整历史遍历 | 历史列表显示已读数量，页脚显示加载、暂停或局部失败 | Cancel 保留已读记录；Retry 沿固定 tips 继续；换 root / ref 取消旧遍历并拒绝旧页 | 历史遍历不占用 Git 写操作的取消入口 |
| 单文件 patch / 总览 patch 失败 | 文件局部失败与同目标 Retry | 只重读当前不可变文件版本，其他有效 patch 保留 | 不重定向到后来选中的文件 |
| 操作完成或失败 | toast，必要时携带现有恢复动作 | 动作继续复用目标和仓库守卫 | 不依赖短暂 toast 表示历史是否读完 |
| 对话框与多窗口 | 共享对话框只在持有键盘焦点的窗口绘制 | Cancel 不写入，确认后再次校验目标 | 自动化 `window w1` 仅指定命令目标，不会改变系统焦点；检查对话框应使用 `window auto` |

## MyGo 原生菜单能力与平台边界

基于当前锁定的 MyGo `v0.8.4-0.20261108082913-9233663736ca` 源码盘点：

- 提供 `mygo.NewMenu`、`MenuItem`、`Separator`、角色菜单与原生编辑角色；菜单项可更新 enabled / checked / visible / label / accelerator。
- 提供 `Window.SetMenu`。SDK 的默认 macOS 菜单含 App、File、Edit、View、Window；非文档应用的 File 默认主要是 Close。
- macOS 默认 Edit 角色包含 Undo、Redo、Cut、Copy、Paste、Select All 等；这些是 SDK 的原生能力，不等于 SuperGit 已验证每个字段、每个窗口的菜单行为。
- Lobe 的 `Run` 和设置窗口初始化没有安装产品自定义 `NewMenu` / `SetMenu`，也没有把 Git 动作映射成系统菜单项。
- 非 macOS、无 DocumentGroup 的应用，SDK 默认菜单返回 nil。跨平台窗口菜单 API 可用，当前产品菜单布局及 Windows / Linux 实际行为尚未验证。
- 当前实机验证环境只有 macOS。源码盘点不能替代 Windows / Linux 的焦点、快捷键和菜单验收。

源码入口：`internal/lobedesk/view.go`、`workspace.go`、`shortcuts.go`、`keyboard.go`、`settings_window.go`、`app.go`；SDK 的 `menu.go`、`window.go` 和 `internal/darwin/menu.go`。

## 后续候选与验收条件

N05 的具体区域明确后，才决定是否实施以下候选；本表不构成新增 UI 授权。

| 候选 | 可复用能力 | 接通前必须验证 |
|---|---|---|
| 应用菜单 Settings / 仓库 Open | 已有设置窗口 / 仓库打开动作 | 主窗口与设置窗口焦点、已打开窗口复用、无重复快捷键 |
| File / Repository 菜单 | Refresh、Fetch、Pull、Push、Commit 的现有方法 | 将可用条件集中到共享动作层；未授权远程写入不可自动执行；菜单禁用与按钮一致 |
| Edit 菜单 | SDK 原生编辑角色 | 标题与正文各自的输入、粘贴、撤销、重做跟随焦点；不把后台 Commit 草稿当作编辑目标 |
| View 菜单 | 已有详情 / 图谱 / Files / patches 显隐和布局复位 | 状态勾选双向同步、隐藏内容不可接收操作、分栏偏好不被误清除 |
| 新增底部状态区域 | 已有数量、进度、失败与取消状态 | 先确定信息用途及位置，避免与顶栏和局部错误重复；不能仅为占位建立空区域 |

需要接通时，先添加计划中的 `TestApplicationMenuSharesActionAvailability`、`TestMenuEditingUsesFocusedWindow`、`TestBusyStateHasConsistentCancelAndRetry` 等行为测试，再实施产品菜单，最后执行 `make check` 和真实多窗口菜单验收。当前这些候选尚未实现，相关验收也尚未完成。
