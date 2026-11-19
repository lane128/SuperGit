# Lobe 候选主线能力差距表

对象为 `internal/lobedesk`，对应 TODO R11。自 2026-11-19，默认 `make app` 经 `cmd/supergit` 构建标签式工作台。`make desktop-app` 可构建旧界面回退包。表中的原生证据按对应批次记录，写操作使用临时仓库、本地裸远端和独立偏好。

## 2026-11-19 默认入口核对

- `make app` 构建支持同窗口仓库标签的 `build/SuperGit.app`。本轮 `make native-check` 从这套打包配方构建测试包，日常暂存 / 提交 / 同步、Stash、Merge 冲突取边和子模块闭环均通过。此前旧桌面入口中从最近记录切换仓库会替换当前仓库，现可用 `make desktop-app` 构建该回退界面。原生目录选择框可打开，选择和取消仍待验收。
- Lobe 的最近仓库现在跨重启保留。macOS 隔离预览在 980×700 和 1320×700 两种宽度通过 Finder 目录交付、同窗口双仓库标签鼠标切换、退出恢复标签顺序与活动仓库、关闭所有标签后重启并从 Welcome 重开最近仓库。两个临时仓库的 Git 状态和文件字节保持一致。本批原生自动化发现了 `File → Open Recent` 的完整路径条目，但窗口未获得焦点，尚未验证重启后从该菜单实际打开仓库。
- 最近路径与启动标签分开保存。只有在 Lobe 最近列表不存在时，才从旧版 `preferences.json` 读取 `Recent`。读取保留原文件，导入路径只出现在最近列表，需要用户选择后才打开。现有 Lobe 列表优先，包括空列表。
- History Graph 在分支线超出可见宽度时显示横向轨道。macOS 在 980×640 和 1320×840 验证横向平移及重复点击当前提交。再次点击后选中节点完整回到视野，提交列表与右侧详情仍显示原提交。单线历史无轨道。深色和浅色另有 Go 渲染回归覆盖。
- 工作区与 History 已接入统一 / 左右 Diff 切换。macOS 预览构建在 980×640 和 1320×840 验证总览与文件标签、右列单行暂存 / 取消暂存，以及切回统一视图。
- 文本冲突编辑已接通。`TestConflictEditorWorkbenchJourney` 使用真实 Git 仓库验证编辑、保存与暂存流程。下表的旧原生记录仍只覆盖外部编辑或整文件取边。

## 能力与证据

| 流程 | 后端 | 当前 Lobe UI | 原生证据 | 尚缺 / 可用退路 |
| --- | --- | --- | --- | --- |
| 打开仓库 | `gitrepo.Open` | `app.go:chooseRepository` 原生目录面板；启动路径和 `openRepository` | detail-tabs 脚本启动与仓库切换；local-watch 脚本启动。系统面板选目录未由这两套脚本操作 | 面板选目录 → 打开仍需人工核对；已有命令行路径可用 |
| 最近仓库 / 启动恢复 | 不需要新的 Git 写 API | Welcome 和 macOS File → Open Recent 共用最多 12 条最近记录。打开的标签顺序及活动标签独立恢复 | 最新隔离预览通过双仓库鼠标切换与重启后从 Welcome 重开。原生菜单重启路径待验收 | Go 回归覆盖旧路径导入、同时写入保护和重试。显式 CLI 路径临时打开，保留正常的会话与最近记录 |
| Clone / Init | `gitrepo/remotes.go:Clone` / `Init` | Lobe 已有 Clone 和新建仓库表单 | Clone 本地闭环及新建仓库 → 暂存 → 首次提交已有 Lobe 原生记录，见 [e2e 自动化说明](e2e-automation.md) | 面板选目录与真实远端认证需单独验证 |
| 文件 / 批量暂存、取消暂存 | `Stage` / `UnstagePaths` 等 | Files、文件头、总览、菜单与快捷键 | 当前 detail-tabs：指定文件、批量、取消、固定分组；相关 Go 回归验证部分暂存 | 行 / 片段操作有现成入口及旧 Lobe 验证记录；本轮未重复完整行级原生链 |
| 完整提交说明 / hooks / Index-only Commit | `Commit` | 分离的单行摘要和多行正文、有效身份、提交按钮 | 当前 detail-tabs 对实际 `%B`、作者、HEAD / Index / 工作区及拒绝 hook 核对 | 有当前原生证据；用户真实认证 / 签名环境不在本地 fixture 结论内 |
| Amend 完整 HEAD 说明 | `AmendCommit` / `CommitMessage` | Amend 开关；此前仅预填 `HeadSubject`，会遗漏正文 | 原有原生用例手工填写新正文，不能证明原正文保留 | 本轮真实 reload + UI checkbox 回归复现正文丢失；已改读固定 HEAD SHA 的完整消息；全量测试 / race / vet、普通与 e2e 构建及完整 detail-tabs 原生冒烟通过；新增断言验证 HEAD 摘要和两段正文实际预填，完整回写与作者保护断言仍通过 |
| 添加远端 | `Repository.AddRemote` | 侧栏 Remotes 分组内的「添加远端…」及分组右键菜单。重复名称错误显示在表单内，输入可继续编辑 | 980×640 原生双仓库验证取消、添加、重名提示及更名重试。仅当前仓库配置发生预期变化，两个仓库的 HEAD、引用、索引与工作文件保持 | 添加动作只保存配置。Go 回归另验证显式 Fetch 生成跟踪引用；真实认证待验收 |
| Fetch / prune | `Fetch` / `FetchPrune` | 顶栏及右键菜单；手动 / 定时获取 | 当前 `native-lobe-sync-smoke.py` 验证 Fetch 后远端跟踪 ref 前进，HEAD / Index / 工作字节不变 | 新增 `native-lobe-force-prune-smoke.py` 验证顶栏 Shift+F10 → prune：只移除已删除远端分支的跟踪引用，同名本地分支保留，远端 refs / 本地 HEAD / Index / 脏字节 / upstream 不变；不覆盖真实网络或任意 fetch 映射 |
| Pull / Push / upstream | `PullFF` / `PushChecked` / `PushForceChecked`；旧 API 保留 | Pull 仅快进；Push 固定源提交 / 远端；首次设置 upstream，已有 upstream 优先；Force 使用确认时显式租约 | 当前 sync：Push 取消 / 首推 / upstream → Fetch → FF Pull；分叉 Pull 拒绝。reference：非当前分支 Push 取消 / 发布 / upstream，不切 checkout | 当前 sync 已用两个本地裸远端通过：main 的 upstream 为 backup/publish/nested，Push 取消 / 确认只影响所示目标、origin 不变，Fetch 读取双方新引用，HEAD / Index / 暂存和未暂存字节保留。新增 force-prune 脚本由顶栏 focus + Shift+F10 打开应用内菜单，通过 Force 取消 / 分叉替换，以及外部远端推进并 fetch 后旧显式租约拒绝；本地脏状态及 upstream 保留，详情含恢复说明和 Git stale info。此路径可自动化；远端选择器、任意远端名规则及真实网络认证不在结论内 |
| 分支 / 标签引用管理 | checked Rename / Delete / PushTag / DeleteRemoteBranch | Branches 详情 footer，确认显示原 SHA，删除与推送固定原对象 / 远端 | 当前 `native-lobe-reference-smoke.py`：重命名保留 tip / upstream、本地分支删除、附注标签推送 / 本地删除、远端分支租约删除；取消均不写目标，主 HEAD / Index / 字节保留 | stale 对象 / 地址变化有真实 Git 与 Go UI 回归，本脚本未验证原生竞态；同 OID 重建与外部远端配置竞态边界保留 |
| Stash | `SaveStash`、Apply / Pop / Drop | `stash.go` 页面、保存及含未跟踪选项、应用 / 弹出 / 确认删除；按 SHA 读取 | 当前 `native-lobe-stash-smoke.py`：含未跟踪保存、预览、Apply 精确恢复 Index / 工作字节且保留条目、Drop 取消 / 确认、Pop 成功及重叠拒绝保留 stash | 重叠拒绝保留 HEAD / Index / 原有 tracked 字节，但 Git 仍可能恢复未跟踪文件；已用纯 Git 对照核实。新版已按当前语言说明行动及可能恢复未跟踪文件，并保留原始 Git 错误：真实 Apply / Pop × 中英文测试和新增英文原生断言、完整检查通过；长正文已改为紧凑提示 / 滚动完整详情，新增原生高度 <180 DIP、原文可读与 Esc 关闭断言通过；分支 footer 等剩余布局边界仍归 R04，当前脚本另通过 Apply / Pop 真实三方冲突：核对 Index 三阶段、双方冲突文本、HEAD / stash 保留、恢复说明及 Git stderr；外部解决并暂存后字节正确。应用内冲突编辑仍未验收。错误详情不包含 Git stdout；纯 Git 对照确认本机 CONFLICT 位于 stdout、stderr 为 Index was not unstashed。旧 `native-stash-smoke.py` 不作为新版证据 |
| 冲突解决 / Continue / Abort / Skip | `Integrate` / `Continue` / `Abort` / `Skip` / 取边 | 工作区操作卡、冲突文件菜单；merge 经提交框完成，其他操作 Continue；Abort / Skip 确认 | 当前 `native-lobe-operation-smoke.py` 共 11 组通过：cherry-pick / Rebase / Revert 冲突下 Continue 禁用、Abort / Skip 取消与确认、外部解决后 Continue 的完整消息 / 父提交 / Index / 字节保护；Merge 未解决时 Commit 禁用，Abort 取消 / 确认，解决后提交框写入完整说明并生成正确双父提交 | reference 脚本另通过 footer 发起无冲突 Merge / Rebase 的取消 / 确认，核对 Merge 双父提交与 Rebase 目标父提交 / 完整源说明。新增 native-lobe-integrate-smoke.py 通过历史行菜单发起无冲突 Cherry-pick / Revert 的取消与确认：新提交父 SHA、完整说明 / 原 SHA、精确字节、topic 引用和无残留操作均核对。新增 `native-lobe-conflict-side-smoke.py` 两组通过 Merge 文件菜单 Use ours / Use theirs：按原 Index 第 2 / 3 阶段核对精确暂存字节、取边前后引用不变，随后 UI Commit 生成正确双父提交 / 完整消息，topic 与无关二进制保留。后续 `--rebase` 两组通过实际阶段身份、精确暂存字节、Rebase 专用成功提示及 UI Abort 恢复原分支 / refs / 干净状态；文件菜单与 Diff 标题菜单已修正中英文 Ours 为变基基底、Theirs 为重放提交。结果 `build/lobe-conflict-side-yr7qi2k_/results.json`，同轮 Merge 完整提交回归通过。总览 / 文件标签另补中英文只读预览、文本 / 二进制外部处理与删除型冲突取空侧后果提示，16 组渲染 / 高度回归及全量检查通过。`--shape binary/deletion` 后续共 8 组原生通过：Merge / Rebase 两种取边核对精确阶段字节或暂存删除，Merge 完整提交、Rebase Abort 恢复原状态；完整日志 `build/lobe-conflict-shapes-check.log`。该证据覆盖整文件取边；后续 `--rebase-continue` 文本两组已通过 UI 取边后 Continue，核对 ours 空重放省略、theirs 单父重放及原消息 / 作者日期、精确字节、topic 与无关二进制保护，结果 `build/lobe-conflict-side-o40jntnx/results.json`。二进制 / 删除型各两组 Continue 后续全部通过，核对相同父提交 / 元数据规则及精确二进制或删除结果，结果 `build/lobe-conflict-side-oziemhcb/results.json`、`build/lobe-conflict-side-_kqq593z/results.json`。三方手工编辑和其他操作取边仍未验收。新增 `native-lobe-operation-stale-smoke.py` 两组通过同类型同 HEAD、不同目标提交的 cherry-pick 替换：旧 Abort / Skip 确认拒绝，替换操作 / MERGE_MSG / refs / Index / 状态 / 文件字节完整保留，结果 `build/lobe-operation-stale-gwtngzfl/results.json`。后续三组完整重跑通过，新增原生旧 Continue 卡片：原已解决操作启用 Continue，外部替换并解决后直接点击原控件，恢复提示拒绝且替换状态保持，结果 `build/lobe-operation-stale-9oyofyxg/results.json`。未覆盖确认期间刷新、其他类型替换或外部进程原子互斥。原 operation-card 的外部解决不算 UI 编辑通过。残留标记规则待 D4 / R01；无完整三方编辑器 |
| Worktree | `Worktrees` / `AddWorktree` / `RemoveWorktree` | `workspaces.go` 列表、创建表单、切换、保护性移除、保留分支 | 当前 `native-lobe-worktree-smoke.py`：表单创建含空格路径 / 新分支 → 切换及返回 → 取消移除 → 脏目录拒绝 → 干净移除；先核对保留分支，再坐标点击 toast Delete 并验证仅原分支删除、当前仍为 main、主 HEAD / Index / 文件不变 | 后续完整脚本 8 组通过：已加载锁定时菜单禁用、确认后新锁定时拒绝原目标、普通 status 干净但有 ignored 二进制文件时拒绝且保留准确字节 / HEAD / Index。系统目录面板回填仍需人工；ignored 两类保护原因后续已用携带原路径的类型化错误映射为当前语言，真实 Worktree / cherry-pick 回归、全部 Go / race / vet / 构建及原生 Worktree / 完整工作台通过 |
| Submodule | `Submodules` / `UpdateSubmodule`，后者 `update --init --checkout` 并执行安全检查 | Worktrees 页子模块区，更新、打开已初始化模块、显示路径；有效 gitlink Stage / Unstage | 当前 `native-lobe-submodule-smoke.py`：未初始化模块 Update 初始化、偏移 child HEAD 更新回 Index SHA、内部 dirty 拒绝、打开子仓库并显示独立历史；parent HEAD / Index 不变 | 子仓库 Worktrees 路径已修复：主记录经 Git 解析到真实 checkout，从子模块主目录和 linked worktree 读取均有真实仓库回归；新增原生路径 / Current 断言及完整回归通过。旧错误跨仓库保留已修复；最新原生五组进一步保存切换前真实错误，检查 child Worktrees / History 全部辅助角色中无父错误标题 / 正文，同时父子 HEAD / refs / Index 条目 / 状态及配置 / 脏字节保持，结果 `build/lobe-submodule-psyuv4uc/results.json`；dirty 更新拒绝后续已按当前语言显示恢复说明，保留后端原文本；真实两路径中英文回归 RED→GREEN、全量检查和原生子模块 / 完整详情标签通过，结果 `build/lobe-submodule-2vnzuocy/results.json`，英文正文及切 root 退出截图已核对。长错误布局与确认归属仍归 R04。ignore 一致性待 D4 / R05；不支持递归发布承诺 |
| Reset / Undo | `ResetToChecked` / `UndoCommitChecked` | 历史行应用内菜单，Reset 模式表单与 Hard 二次确认；HEAD 的 Undo 确认 | 当前 `native-lobe-reset-smoke.py` 四组：Soft / Mixed / Hard / Undo，菜单由 Shift+F10 打开；取消保持完整原状态，实际执行核对 HEAD / branch / Index / 工作字节及未跟踪二进制保护，Hard 二次确认另核对取消 | 当前正常路径及 --stale 四组外部 HEAD 前进拒绝已验，Hard 在二次确认后推进；外部新 HEAD / refs / 原始 Index / 字节均保留并有重试说明。同 SHA 切分支、进行中操作替换及跨进程原子互斥不在原生结论内；并非真实用户仓库或跨平台验收 |
| 偏好持久化 | Lobe `settings.go` 原子写入与读取 | 设置中代码字号可跟随界面，也可固定为 11–19 pt。旧版主题和代码字号只补充 Lobe 未保存的选项 | 560×480 macOS 设置窗口通过中英文鼠标调整和两种模式的重启恢复。Go 回归覆盖迁移优先级、旧文件字节和 19 pt 长行范围 | 偏好读取优先级及旧文件保留通过 Go 回归，旧桌面入口通过独立原生检查。证据见 R13 |

## 下一步验收顺序

1. Amend 完整消息预填的全量与实际原生门槛已通过；本表与试用记录已更新。
2. 为已有 Lobe 入口补独立、本地远端的同步 / Stash / 操作卡 / Worktree / Submodule 原生闭环；沿用现有自动化方式，不能把旧 desktop 的路由和按钮名称机械替换。
3. 将每项通过与失败写入 [试用记录](lobe-trial-log.md)，记录源码状态、执行命令、日志及边界。真实系统目录面板与用户远端关键流程保留人工验收。
4. R13 的默认打包与回退已验证。R11 继续记录人工打开目录、菜单焦点、真实远端认证等剩余验收。

## 核对入口

- 当前 UI：`internal/lobedesk/app.go`、`workspace.go`、`menu.go`、`stash.go`、`workspaces.go`、`settings.go`。
- 后端：`internal/gitrepo/remotes.go`、`integration.go`、`stash.go`、`workspaces.go`、`submodule.go`、`commit_message.go`。
- 当前 Lobe 原生脚本：`scripts/native-detail-tabs-smoke.py`、`native-local-watch-smoke.py`、`native-workbench-layout-smoke.py`、`native-lobe-sync-smoke.py`、`native-lobe-reference-smoke.py`、`native-lobe-stash-smoke.py`、`native-lobe-worktree-smoke.py`、`native-lobe-submodule-smoke.py`、`native-lobe-operation-smoke.py`、`native-lobe-window-dialog-smoke.py`、`native-lobe-reset-smoke.py`、`native-lobe-force-prune-smoke.py`、`native-lobe-integrate-smoke.py`。
- 标准包入口为 `make native-check`。旧桌面回退入口为 `make desktop-native-check`，用 `scripts/check-native-workflows.py` 启动 `build/supergit-desktop-e2e`。`native-workflow-smoke.py` 与 `native-stash-smoke.py` 也针对旧界面。
- 历史记录：[Lobe 预览进度](lobe-preview-progress.md)；命令 / 可用条件：[命令与状态盘点](workbench-command-map.md)。
