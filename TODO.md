# SuperGit TODO

已完成项见 [DONE.md](DONE.md)。产品与设计规划更新于 2026-11-15；2026-11-16 已核对并勾选原始清单中 10 条已有实现与验收证据的重复待办，保留原问题背景。下一步以「开发规划与需求」为入口，原始条目用于追溯背景与验证证据；未确认的产品选择和未验证边界继续保持未完成。

**已确认的目标布局（2026-11-15）**：定位栏｜提交列表（含工作区首行）｜全高文件列表｜「提交 / 摘要 + 多文件」标签式详情。采用 `design/workbench-preview.html` 最新预览；标签式详情主体已在 Lobe 开发入口落地，验收与保留尾项见下方 T1–T6。

**当前原生主界面状态（2026-11-15）**：Lobe 历史页采用上述四列布局。工作区首标签「提交」包含提交表单、进行中操作及全部改动；历史首标签「摘要」包含完整说明与全部补丁。文件可打开为独立、可关闭的标签，同路径的暂存 / 未暂存版本分开；草稿与各标签滚动位置保留。刷新以仍存在的活动标签身份确定补丁和操作目标，包括文件分组折叠时。既有三个详情偏好分别控制历史信息块、文件列与历史补丁；首标签保留，隐藏详情可恢复。macOS 原生交互、16 组截图矩阵与独立代码审查已执行；审查发现的重要刷新身份问题已修复；2026-11-16 已补齐文件列表单一 Tab 入口、聚焦行操作、可见区读取优先级和工作区补丁缓存复用。默认打包仍使用旧桌面入口，D1 迁移门槛继续有效。

**下一轮反馈改进（2026-11-16）**：见 [工作台下一步改动计划](docs/superpowers/plans/2026-11-15-workbench-next-improvements.md)。三处分栏、虚线工作区节点、固定三组、提交字段与作者身份、文件头及批量操作、完整历史与侧栏已实现。真实 Git 原生冒烟包含主历史 2,405 条、按文件历史 2,403 条完整计数和最早提交定位；全量测试 / race / vet 与普通 / e2e 构建通过。本轮完整 16 组显示矩阵、真实分栏重启和独立审查已完成；两项重要发现经回归修复并再次通过全量及原生检查。2026-11-16 已补齐缩窄详情区后的活动标签与关闭按钮自动显露，真实缩窗回归及完整显示矩阵通过。N05 新菜单 / 状态区域仍待范围明确。当前安装包未更新；本段不关闭下面原计划的保留尾项。

读法：
- **待你选择**：需要你选 A/B/C 或给一条规则，确认前保持现状。
- **待开发**：方向已定、可以直接实现或补测试的条目。
- **观察与讨论**：已有结论的观察项，等实际使用反馈、上游变化或解锁环境后再动。

---

## 开发规划与需求（产品 / 设计，2026-11-15）

### 产品目标与规划边界

- **已有背景**：用户使用 Sublime Merge，关注操作流畅、现代界面与可定制性；submodule 是核心要求。技术路线为 Go + MyGo 原生 GPU UI、系统 Git、无 WebView，macOS 优先，最终面向 Windows / Linux。依据：`docs/superpowers/specs/2026-11-11-mygo-first-version-design.md` 与 `2026-11-11-product-workflows-design.md`。
- **已经明确的主流程**：历史页作为主界面，工作区实时改动作为首行，与真实提交 DAG 同屏；查看提交或工作区 → 查看文件 → 阅读补丁 → 暂存 → 提交，分支切换与同步在标题栏。
- **建议的首批使用者**：熟悉基本 Git、日常处理多分支及子模块的个人开发者。首版先让他们可靠完成日常开发，不以 Git 教学、托管平台协作或全功能 IDE 为目标；用户定位仍可调整。
- **本次交付范围**：整理需求与开发顺序，不实施功能、不切换打包入口、不 commit 或发布。以下「建议」表示产品提案，不等于用户已确认；新增用户可见能力与设置仍需确认后实施。
- **设计原则建议**：可辨认的仓库与操作目标、选择与可见内容一致、焦点决定键盘作用范围、失败有局部恢复入口、宽屏优先增加补丁阅读空间。参考 Sublime Merge 的工作流，不逐项复制装饰和菜单。

### 现状核对：区分实现、测试与交付

| 层次 | 已确认的现状 | 对排期的影响 |
| --- | --- | --- |
| Git 后端 | `internal/gitrepo` 已有暂存、提交、分支、同步、Stash、整合操作、worktree / submodule 等能力；两套 UI 共用 | 优先复用已有能力，不能把后端已有视为新版 UI 已接通 |
| 打包版 | `make app` 经 `cmd/supergit` 构建 `internal/lobedesk`。`make desktop-app` 构建旧桌面回退包 | 默认入口切换及原生证据见 R13 |
| 新版界面 | `cmd/supergit-lobe` → `internal/lobedesk`；已有历史与工作区合并、设置、文件 / 行 / 片段操作、分支和同步等大量演进 | 已超出 README 中早期「仅浏览、暂存、提交」的预览描述，但仍不能据此宣称功能全部对齐 |
| 本地实时更新 | 2026-11-16 已接入独立本地检测，结合 `ChangeStamp` 与改动文件元数据，前台 / 后台及聚焦检查 | 自动 Fetch 关闭时的外部编辑、暂存、提交、切分支与子模块 HEAD 已原生验收；细节与元数据观察边界见 R10 |
| 自动化 | `make check` 检查全部 Go 包，`make native-check` 验证标准包，`make desktop-native-check` 验证回退入口 | 本轮两个 macOS 原生入口均通过，Windows / Linux 实机验收待完成 |
| 视觉证据 | 已查看 `build/narrow-shots/zh-light-worktree.png`、`en-dark.png`；980×640 下能容纳四个区域，摘要在前时文件仍呈选中态 | 本轮依据已有截图与代码做规划，没有重新启动或实测 App |
| 跨平台 | 已验证记录集中在 macOS Apple Silicon；新版外部打开 / 文件定位 / 终端仍直接执行 macOS `open` | 需要平台适配与目标系统实测，不能由 Go 可编译推断跨平台可交付 |

### 主线决策与方案对照

**D1 已确认（2026-11-15）**：Lobe 达标后接管。后续界面开发以 `internal/lobedesk` 为主线，共用 `internal/gitrepo`；旧版 desktop 暂时保留为当前打包版与行为对照。完成能力差距核对、新版原生闭环验证与可回退的偏好迁移后，再切换打包入口；此决定不包含删除旧版代码或发布授权。这里的 Lobe 指基于 MyGo 的 Lobe MyGo 原生界面组件库，新版仍使用 Go + MyGo，无 WebView。

1. **已选：以 Lobe 作为下一版候选主线，达到能力与验证门槛后再切换打包**。符合近期工作区合并和界面投入方向；先补日常闭环、复用 Git 后端，再迁移交付入口。代价是需明确补齐旧版独有能力并迁移偏好。
2. **备选：继续以 desktop 为主线，将新版交互移回旧版**。保留现有交付与测试路径，但要重新实现近期布局、设置与交互，重复成本较大。
3. **不建议长期双线同规格演进**。短期可保留旧版做行为对照，长期会让能力、文档与自动化继续分叉。

2026-11-19 已切换默认打包入口，R13 记录验收结果。旧界面可独立构建作回退与行为对照。R11 继续跟踪待验收能力。

### 里程碑与完成门槛

优先级：**P0** 为可能导致错误写入、错误目标或核心操作受阻的问题；**P1** 为日常流程与主要可用性；**P2** 为效率增强与视觉优化。同一阶段先完成可独立验收的小批次，再进入下一批；暂不承诺日期，主线与布局确定后再按实际差距估工。

| 阶段 | 结果 | 依赖与退出条件 |
| --- | --- | --- |
| M0 · 正确性与状态 | 冲突、选区、焦点、错误恢复符合可见状态 | 不依赖布局大改；P0 场景与相关回归通过，不存在无提示的关键操作失效 |
| M1 · 主工作台收敛 | 已确认的标签式详情落地，文件列表与多文件浏览形成稳定的鼠标 / 键盘流程 | D2 已确认，按 T1–T6 执行；D3 / D4 只阻塞各自相关需求；980 / 1320 / 1600 宽均可用，长路径、多标签和大字号通过实际检查 |
| M2 · 候选主线与日常试用 | 用户打开的安装版与正在评审的界面一致，可日常开发 | 先确认 D1 / D5；主线能力差距表逐项验收，新版有完整原生闭环，偏好迁移可回退 |
| M3 · 跨平台预览 | 明确目标系统上可安装、可操作、可恢复 | 先确认 D6；每个宣称支持的平台都有原生构建、工作流与平台行为的实测证据 |
| M4 · 进阶与交付品质 | 根据真实试用补效率能力并准备分发 | 不阻塞 M0–M2；新增能力单独确认，发布另行授权 |

### M0：优先处理的需求

- [ ] **R01 / P0 · 冲突解决前的保护**。已核对 `lobedesk/menu.go:resolveConflict` 直接执行 `Stage`，`gitrepo/diff.go` 只识别普通 `@@`。建议先提供明确的冲突只读状态与外部解决路径，存在残留冲突标记时阻止直接标记完成并说明原因；后续再补 combined diff / 三方展示。**验收**：普通文本未清标记时不直接进入已解决状态；已清标记可暂存；删除冲突、二进制和 Rebase 的 Ours / Theirs 含义有明确提示；失败保留进行中状态。残留标记允许覆盖还是一律阻止，由 D4 确认，不能把启发式检查说成能识别所有冲突。2026-11-16：已修正文件菜单与 Diff 标题菜单的 Rebase 中英文取边含义及成功提示：Ours 为变基基底，Theirs 为重放提交；真实阶段语义回归 RED→GREEN，全量 Go / race / vet、两种构建、Rebase 两种取边后 Abort 恢复、Merge 两种取边后完整提交及完整详情标签原生回归通过。证据见 `build/lobe-rebase-side-check.log` 和 `build/lobe-conflict-side-yr7qi2k_/results.json`。后续补齐总览 / 文件标签的中英文只读预览与外部处理提示，UD / DU / DD 明确取不存在一侧会删除并暂存路径；含提示的展开节使用实际布局高度。16 组渲染与高度回归 RED→GREEN，全量 Go / race / vet、两种构建、带提示断言的 Merge / Rebase 及完整详情标签原生通过，日志 `build/lobe-conflict-preview-check.log`。真实二进制 / 删除型冲突已补 Merge / Rebase × Ours / Theirs 共 8 组原生通过：精确阶段字节或暂存删除、引用与无关文件保护、Merge 完整提交及 Rebase Abort 原状态恢复均核对，日志 `build/lobe-conflict-shapes-check.log`。Rebase 取边后 Continue 后续已补文本 / 二进制 / 删除 × 两侧共六组原生通过：精确字节或删除、ours 空重放省略、theirs 单父重放及原消息 / 作者日期、topic / 无关文件保护，日志 `build/lobe-rebase-side-continue.log`、`build/lobe-rebase-side-continue-shapes.log`。D4 标记策略仍待决定。
- [x] **R02 / P1 · 不可见选区不能阻断文件操作**。纳入标签计划 T4：切换标签、关闭标签、隐藏详情或切换目标时清理旧补丁的行选区。摘要 / 提交标签本身包含补丁，不能只凭标签名判断选区是否可见；行操作必须固定实际文件、暂存分组与差异版本。**验收**：切换后 Space / Enter 不被旧选区静默阻断，复制不读到其他文件的行；摘要中可见补丁的操作仍正确，折叠文件后不操作隐藏行；输入框的方向键、换行和复制不被接管。
- [x] **R03 / P1 · 文件列表独立焦点**。工作区 / 历史 Files 整体一个 Tab 入口，Tab 离开，Shift+Tab 返回此前聚焦的行；↑/↓、Home/End、PageUp/PageDown 沿文件身份移动并跳过标题，←/→ 可到达分组标题与行内按钮。Space 暂存 / 取消暂存、Enter 外部打开固定到聚焦行的原始路径与暂存分组，防止同路径双版本错操作；输入框与弹层保留键盘归属。2026-11-16：单 Tab 停靠和真实 Index 目标回归 RED→GREEN，全量测试 / race / vet、普通 / e2e 构建与完整原生冒烟通过；新增原生断言分别验证工作区与历史 Files 的 Tab 离开 / Shift+Tab 恢复原行。整批审查另复现历史 Files 滚动后记忆行被裁剪；现保留一个原路径 / 上下文记忆行并在焦点返回时显露，1,000 文件回归 RED→GREEN，全量 Go / 构建及完整原生工作台复测通过。
- [ ] **R04 / P1 · 反馈与确认归属**。合并刷新无反馈、错误 toast 遮按钮、跨窗口对话框三个问题。**验收**：慢刷新有读取状态，失败可重试且旧数据标明过期；错误不遮主要操作，首行给当前语言的可执行说明、保留 Git 原文；确认固定到发起窗口、仓库和对象，后来的确认不静默覆盖前一个。不能未经核对直接把已完成的通知 owner 修复再做一遍。2026-11-16 新版同步验收已复现长 Git 错误 toast 占据大片补丁区域，证据 `build/lobe-sync-smoke/final.png`；同时保留此前未明原因的历史选择跳变调查。Stash 原生验收另确认英文界面仍显示中文后端错误，且重叠失败可能已恢复未跟踪文件，当前反馈未说明该部分恢复边界；证据 `build/lobe-stash-smoke/final.png`。2026-11-16 已修复主窗口旧 toast 跨仓库残留：队列记录 root，切 root 清除已显示通知、丢弃旧队列并阻止旧按钮执行；同根刷新保留提示，设置窗口通知仍由各自上下文显示。三项 UI 回归 RED→GREEN，全部 Go 测试 / race / vet、两种构建及子模块 / 完整工作台原生回归通过；新增原生断言确认父仓库错误不进入子仓库。确认框后续已绑定发起窗口与仓库根：后来请求保留原确认并显示中英文重试提示，切 root 丢弃旧确认且旧回调不能执行，关闭设置窗口清理遗留确认；同根刷新、与仓库无关的设置重置确认保留。相关 UI 回归 RED→GREEN，全部 Go 测试 / race / vet、两种构建、原生操作卡 / 完整工作台回归通过。其他自定义弹层仍随焦点，确认对象随仓库状态变化的逐项审计、长错误展示仍未完成；未新增原生双窗口焦点切换验收。Stash 反馈后续已修复：后端保留原始 Git 错误，新版按当前语言说明处理方式与可能已恢复未跟踪文件；真实 Apply / Pop × 中英文回归、全部 Go 测试 / race / vet、两种构建、原生 Stash / 完整工作台回归通过。长正文后续改为紧凑提示与“查看详情”：原窗口内滚动展示完整内容，支持复制和 Esc 关闭，切 root 清理旧详情；设置最小窗口尺寸有边界回归。全部 Go 检查 / 构建通过；原生首次因脚本缺少辅助函数中断，修正后 Stash / 完整工作台重跑通过，新增高度 <180 DIP、打开详情保留原文和 Esc 关闭断言。已查看 `build/lobe-stash-smoke/compact-error.png` 与 `error-details.png`。长标题与关闭窗口生命周期后续已修复：完整原标题可在详情读取 / 精确复制，关闭设置窗口清理旧详情，重开不继承；相关回归 RED→GREEN。UI 回归另确认 980×640 / 字号 1.15 下长连续 Unicode 文本与复制 / 关闭按钮的边界。全部 Go 检查 / 两种构建及原生 Stash / 完整工作台再次通过；未声称原生大字号验收。分支 footer 遮挡后续已修复：按实际按钮区高度上移提示；980 / 1320 × 普通 / 1.15 字号四组点击回归 RED→GREEN。全部 Go 检查和两种构建通过，原生脚本补足选中 main 的步骤后，同步 / 完整工作台重跑通过；错误显示期间坐标点击 Copy branch name 成功并验证 HEAD / Index 不变，截图 `build/lobe-sync-smoke/branch-footer.png` 已查看。通用输入弹层后续已保护：保存 Stash、分支 / 标签命名与重命名输入固定主窗口和原 root，后来请求保留当前草稿并提示重试，切 root 后旧回调失效；同根刷新和多步输入保留。相关回归 RED→GREEN，全部 Go 检查 / 两种构建 / 原生 Stash 与完整工作台通过；未新增原生双窗口焦点验收。Reset 独立表单后续已绑定主窗口与原 root，后来请求不替换选定提交，切 root 后不启动旧操作；同根刷新保留选择，Hard Reset 二次确认与忙碌 / 取消保护保留。相关回归 RED→GREEN，全部 Go 检查 / 两种构建 / 原生操作卡与完整工作台通过，尚无新增原生 Reset 实际执行验收。提交的 checkout 分支选择器后续已固定主窗口与原 root，后来请求保留选定提交、搜索和键盘位置，切 root 后旧 Enter 不启动切分支；相关回归 RED→GREEN，全量 Go 检查 / 两种构建 / 完整原生工作台回归通过。Worktree 创建表单后续已固定主窗口和原 root，保留当前草稿，旧提交 / 目录选择返回 / 异步失败恢复不会进入新仓库或覆盖新表单；同根刷新及原有创建 / 失败重试 / 忙碌保留输入回归通过。相关回归 RED→GREEN，全量 Go 检查 / 两种构建 / 原生 Worktree 与完整工作台回归通过；未新增系统目录选择或双窗口原生验收。Blame 后续已固定主窗口与原 root，后来请求保留当前文件 / 版本 / 选项，切 root 或关闭后旧读取结果失效，旧行回调不能作用到重新打开的对象；同根刷新保留。相关回归 RED→GREEN，全量 Go 检查 / 两种构建 / 完整原生工作台通过；没有新增 Blame 专项或双窗口原生验收。普通刷新反馈后续已补齐：超过 200ms 后顶栏显示当前语言的加载文字和旋转图标，完成退出，读取期间 Refresh 禁用且不显示仅用于写入操作的取消按钮；中英文 / 延迟 / 操作优先级回归 RED→GREEN。既有失败后的旧数据标记、实际 Retry 启动新读取和 Retrying 禁用回归已核对通过；全量 Go 检查 / 两种构建 / 完整原生工作台通过，未新增刻意延迟读取的原生验收。跨类型弹层后续已保护：同窗口先开的交互弹层保留目标与输入，后来请求显示中英文重试提示；七类弹层 42 组先后打开组合及窗口独立性检查通过。Worktree 异步失败恢复等待当前弹层结束，草稿可重新打开；已请求的错误详情保留原文并延后显示。相关回归 RED→GREEN，全量 Go 检查 / 两种构建 / Stash、Worktree、操作卡、完整工作台原生回归通过；未新增原生并发弹层或双窗口验收。Reset 同根基线后续已保护：打开时捕获 HEAD / 分支，Hard 二次确认保留同一基线，执行前在应用内写锁下读取实际 Git 状态；外部提交 / 同 SHA 切分支 / detached × 三种模式九组拒绝回归及正常模式通过。真实 UI Hard 二次确认后外部切分支、不刷新 UI 的回归 RED→GREEN，拒绝后 HEAD / Index / 工作字节保留并有本地化重试提示。全量首轮被中文 Reset 术语检查拦下，改为“重置”后，全量 Go 检查 / 两种构建 / 原生操作卡与完整工作台重跑通过；尚无新增原生 Reset 执行验收，不保证与外部 Git 进程原子互斥。本地分支普通 / 强制删除后续已固定名称＋SHA，菜单 / footer 保留所选对象，确认显示目标提交，执行前复核真实分支 tip；外部移动 / 不同 tip 重建时拒绝旧确认，缺失时保留 Git 错误并给本地化重试提示。真实 Git 与应用确认回归 RED→GREEN，正常删除与 Git 未合并 / 当前 / linked 工作树保护保留；全量 Go 检查 / 两种构建 / 原生同步与完整工作台通过。未新增原生删除执行验收；同名同 SHA 重建不可区分，不保证外部进程原子互斥。标签删除后续固定原始 ObjectSHA，确认显示原对象；附注说明变化但提交不变时仍拒绝旧确认。Git 旧 OID 条件删除配合 --no-deref，只删除选中的标签引用，符号引用目标保留；失败给当前语言恢复说明并保留原始 Git 错误。真实 Git / 应用确认回归及全量 Go 检查 / 两种构建 / 原生同步与完整工作台通过；未新增原生标签删除执行验收，同名同 OID 重建不可区分。提交关联的分支选择后续也固定选定 SHA：菜单和选择器执行前复核分支目标，外部移动 / 不同 tip 重建后拒绝旧选择；真实 Enter / Git 回归 RED→GREEN，正常切换及 dirty / linked 工作树保护通过，全量 Go 检查 / 两种构建 / 完整原生工作台通过。普通分支切换保留跟随分支语义，检查不保证外部进程原子性，未新增原生选择器切换验收。Abort / Skip 后续捕获原操作 Kind / HEAD / Token，执行前复核完整身份；同类型同 HEAD 替换及确认后刷新不能重定向旧确认。真实 Git / UI 回归 RED→GREEN，拒绝保留新操作、Index 和冲突字节，正常中止 / 跳过保留；全量 Go 检查 / 两种构建 / 原生操作卡与完整工作台通过。未新增原生操作替换场景，外部 Git 原子互斥不在保证内。Continue 后续同样复核操作卡的 Kind / HEAD / Token，拒绝旧卡片继续后来开始的新操作；原操作解决冲突并暂存后仍可正常完成。真实 Git / UI 回归 RED→GREEN，全量 Go 检查 / 两种构建 / 原生操作卡与完整工作台通过；未新增确认步骤或原生替换操作验收。分支重命名后续固定菜单 / footer 所选名称＋SHA，输入期间移动或不同 tip 重建时拒绝；正常 Git 重命名保留 HEAD / 跟踪配置迁移，已有名称不覆盖。真实 Git / UI 回归 RED→GREEN，全量 Go 检查 / 两种构建 / 完整原生工作台通过；未新增原生重命名执行验收，同名同 SHA 重建和外部进程原子边界保留。整合确认后续固定双方目标：分支菜单 / footer 与历史提交入口传入选定 SHA，Merge / Rebase 的分支标签附原 SHA，四类整合操作执行前核对当前 HEAD / 分支。目标分支后来移动仍只使用原提交，当前侧变化则拒绝；十二组后端拒绝 / 四组正常操作及真实 footer 两侧变化回归通过，全量 Go 检查 / 两种构建 / 原生操作卡与完整工作台通过。未新增原生 Merge / Rebase / Revert 发起验收，外部进程原子互斥不在保证内。Undo 后续捕获并显示原提交 SHA，执行前复核原 HEAD / 分支并要求没有进行中操作；旧确认不会撤销后来提交或软重置未完成的 cherry-pick。真实菜单 / Git 回归 RED→GREEN，正常撤销保留 Index 和工作文件、根提交拒绝；全量 Go 检查 / 两种构建 / 原生操作卡与完整工作台通过。未新增原生 Undo 执行验收。Worktree 移除后续固定原路径 / HEAD / 分支 / 锁定状态，前进、切分支或同路径改挂另一分支时拒绝旧确认；真实 Git / UI 回归 RED→GREEN，替代目录和登记保留，正常移除保留分支及既有内容保护。全量 Go 检查 / 两种构建 / 原生 Worktree 与完整工作台通过。未新增原生路径替换验收，同路径 / 分支 / HEAD / 状态重建不可区分。Worktree 移除成功提示的删除分支按钮后续固定名称＋原 SHA，以 root 保持同仓库刷新后的可用性，切仓库失效；真实成功提示动作回归 RED→GREEN，原分支可删除、后来 tip 保留。全量 Go 检查 / 两种构建 / 原生 Worktree 与完整工作台通过；未新增原生成功按钮删除分支验收。通知队列后续按 root 过滤，保留独立设置结果与顺序、同 root 有效提示，修复成功切仓库时同时清空两窗口队列；真实读取回归 RED→GREEN，全量 Go 检查 / 两种构建 / 原生子模块与完整工作台通过。生产审计确认当前没有带动作按钮的错误提示；未来新增该类错误需单独核对紧凑展示与动作保留。标签推送后续固定原始对象 SHA 与远端有效 fetch / push 地址指纹，确认显示原对象；标签或远端变化时拒绝旧确认，源 refspec 使用原对象 OID，保留附注与非强制保护。真实 Git / UI 回归 RED→GREEN，全量 Go 检查 / 两种构建 / 原生同步与完整工作台通过。未新增原生标签推送执行验收，外部 Git 配置的检查与写入非跨进程原子操作。远端分支删除后续固定原 SHA 与远端配置，使用显式 expected-OID 删除租约；确认后更新跟踪引用或 UI 不能使旧确认删除新 tip，正常删除仍清理跟踪引用。真实 Git / UI 回归 RED→GREEN，全量 Go 检查 / 两种构建 / 原生同步与完整工作台通过；未新增原生远端删除执行验收，同 SHA 重建与远端配置跨进程竞态边界保留。普通 / 非当前分支推送后续固定源名称＋SHA、远端与目标，使用原 OID 推送并保留非强制和子模块保护；首次上游设置支持自定义 fetch 映射，配置失败明确说明远端推送已完成。真实 Git / UI 回归 RED→GREEN，全量 Go 检查 / 两种构建通过；补齐文案动作登记并更新原生旧文案断言后，同步与完整工作台原生重跑通过。强制推送后续固定源 OID、原远端和显式预期远端 SHA / 不存在基线；确认后 fetch / UI 刷新不改变租约，读取远端引用失败不打开确认。真实 Git / UI 回归 RED→GREEN，原租约正常推送及不存在基线保护通过，全量 Go 检查 / 两种构建 / 原生同步与完整工作台通过；未新增原生强制推送执行验收。子模块更新后续固定原暂存区指针与检出身份，菜单打开后 Index / checkout 变化时拒绝旧动作；真实 Git / 菜单回归 RED→GREEN，正常初始化 / 恢复与内容保护通过。中文术语修正后，全量 Go 检查 / 两种构建 / 原生子模块、Worktree 与完整工作台通过；未新增原生 stale-Index 验收，子模块 URL 身份和跨进程原子边界保留。Worktree 可编辑起点按提交时命名引用解析的行为已审计保留。后续原生引用脚本已补非当前 Push、重命名、本地分支 / 标签删除、标签推送、远端租约删除及无冲突 Merge / Rebase 发起；Worktree 成功提示 Delete 已经坐标点击验收。共享 dirty 保护后续改为可识别错误，新版按当前语言给提交 / Stash 恢复说明，旧版原始错误文本保留；真实 Git 回归 RED→GREEN，全量 Go / race / vet、两种构建、Worktree 与完整工作台原生回归通过，英文 dirty 拒绝已查看截图。忽略文件保护后续也完成当前语言映射：noIgnored / protectIgnored 携带原路径与覆盖类型，保留后端旧文本，真实 Worktree / cherry-pick 中英文回归 RED→GREEN，全量与原生 Worktree / 完整工作台通过。整批审查后续复现并修复刷新观察吞掉普通保存、深历史刷新清除 SHA / 标签，以及长补丁拖选中断；四项连同 Files 记忆行均经 RED→GREEN，全量 Go / race / vet、两种构建、三轮万提交原生刷新保留 SHA 及完整工作台通过。深历史修复同时覆盖读取回调与逐帧搜索选择逻辑；此前偶发跳变尚无同源证明，原调查不据此关闭。设置重置确认的双窗口探针后续复现错窗：入口按当前键盘焦点选择 owner，现固定为设置窗口。回归有效 RED→GREEN，确认相关测试、全量 Go / race / vet、两种构建、双窗口两种请求先后顺序与取消、完整工作台原生回归通过；截图和完整仓库指纹已核对。追加原生并发检查通过：主窗口 New branch 输入草稿与设置重置确认同时保留，取消设置确认后草稿精确不变，再取消命名不写引用；双窗口截图与完整指纹已核对。自动化目标与内容输入不激活系统窗口，真实 OS 焦点切换不在本轮结论内。后续实际保存 / 重置原生调查复现保存失败提示随系统焦点错窗；已补中英文实际设置卡片保存失败、重置成功 / 失败归属回归 RED→GREEN，设置页写入与重置显式指定设置窗口，主窗口偏好保存保留主窗口归属。最终全量 Go 38.131s / race 161.385s / vet、两种构建通过；中英文原生保存失败保留旧配置及当次主题、恢复保存、实际重置默认值且保留语言、主窗口不出现设置提示及完整 Git 指纹保护通过，完整详情标签原生十组通过。日志 `build/lobe-settings-notice-owner-check.log` / `build/lobe-settings-persistence-followup.log`，结果 `build/lobe-settings-layout-terpjqit/results.json` 已核对。原生设置窗口未取得系统键盘焦点，通知按既有规则延迟展示，两种语言的 `save_warning_visual_verified` / `reset_success_visual_verified` 均为 false；不把队列归属回归或配置写入通过当作系统焦点下提示可见验收。原生 Reset / Undo 后续四组通过：Shift+F10 打开应用内历史菜单，Soft / Mixed / Hard / Undo 均核对取消不变及实际 HEAD / Index / 工作字节，Hard 二次确认取消不写、确认后恢复目标字节，未跟踪二进制文件均保留；截图与结果已核对。另四组 --stale 原生检查通过：确认打开后外部推进 HEAD，三种 Reset 与 Undo 拒绝旧操作并给当前语言重试说明，完整外部状态不变；Hard 在二次确认之后推进。不据此承诺外部进程原子互斥，也未覆盖同 SHA 切分支或进行中操作替换。Force / prune 后续原生通过：顶栏 Shift+F10 应用内菜单可达，独立本地裸远端验证取消、分叉强制替换，以及远端推进且 fetch 后旧显式租约拒绝；详情保留当前语言说明与 Git stale info。prune 只删过期跟踪引用，同名本地分支与脏状态保留。独立回归另复现读取 / 写入期间确认被消耗，现保留原对象与回调、主确认按钮等待可执行、取消可用，独立设置确认不阻塞；有效 RED→GREEN，全量 Go / race / vet、两种构建、Force/prune、双窗口、旧 HEAD 四组与完整工作台原生回归通过。最初原生超时来自未打开紧凑错误详情，不能据此证明确认丢失。新增 `native-lobe-operation-stale-smoke.py` 在 980×640 独立仓库 / HOME 下通过 Abort / Skip 确认后同类型同 HEAD、不同目标提交的 cherry-pick 替换拒绝；恢复说明正确，替换操作标记 / MERGE_MSG / refs / 分支 / Index / 状态 / Unicode 冲突字节及无关二进制均保持。结果 `build/lobe-operation-stale-gwtngzfl/results.json`，Skip 拒绝截图已核对。模态框原生辅助树不暴露底层 Refresh，确认期间刷新仍仅有 Go 回归；后续旧 Continue 卡片原生验证通过：原操作先解决以启用 Continue，外部替换并解决不同目标后直接点击原控件，恢复提示拒绝，替换状态完整保留；Abort / Skip 同轮通过，日志 `build/lobe-operation-stale-continue.log`、结果 `build/lobe-operation-stale-9oyofyxg/results.json` 和 Continue 拒绝截图已核对。子模块内部 dirty 更新拒绝后续也已接入当前语言映射：保留后端原始中文 Error 文本，以可识别错误给英文 / 中文恢复说明。真实 checked / 普通更新 × 两语言回归 RED→GREEN，核对父子 HEAD / refs / Index 条目及脏 Unicode 字节保持，包装错误的原始详情保留。全量 Go / race / vet、两种构建、子模块原生五组及完整详情标签十组通过；日志 `build/lobe-submodule-dirty-full-check.log` / `build/lobe-submodule-dirty-native.log` / `build/lobe-submodule-dirty-detail.log`，结果 `build/lobe-submodule-2vnzuocy/results.json` 已核对，英文恢复说明完整可读及切 root 退出截图已查看。只修复该拒绝原因的语言，不改更新保护或默认依赖。**当前剩余**：真实双窗口 / 并发弹层系统焦点、其他 stale 对象原生竞态，以及未明原因的历史选择跳变；跨进程原子互斥、同身份重建和子模块 URL 边界仍按已记录限制处理。上述早期“未新增”描述是当时的验证范围；最新能力以 R11 与能力差距表为准。不能据此勾选 R04。
- [ ] **R05 / P1 · 子模块状态与差异一致**。已核对 Status 强制 `--ignore-submodules=none`，Diff 未同步。建议 Diff 同样显示实际子模块变化，保持核心使用场景的一致性。**验收**：默认配置与 `ignore=all` 下，指针移动 / 内部 dirty 的行状态与详情一致；仅内部 dirty 不伪称父仓库可提交这些文件；两种配置有真实仓库回归。该策略属于 D4，确认后联动文件级与行级路径。

### M1：主工作台需求

- [x] **R06 / P1 · 落实已确认的标签式四列布局**。D2 以最新预览为准：定位栏｜提交列表｜全高文件列表｜「提交 / 摘要 + 多文件」标签式详情，替代文件与摘要上下分区。工作区的「提交」标签显示提交框、进行中操作与全部改动；历史的「摘要」标签显示说明 / 元数据及全部文件补丁；单文件标签只显示对应文件版本的补丁。**验收**：打开 / 切换 / 关闭文件标签可用；同路径不同暂存分组不混淆；切换标签不丢草稿，滚动位置按目标保留，过期结果不能串页；空仓库、空提交、读取失败和隐藏详情均有正确反馈。按下方 T1–T6 开发并验证；保留 980×640 最小窗口，不新增上下分区拖动器。
- [ ] **R07 / P1 · 工作区身份、进入与选择规则**。推荐首行明确命名「工作区」，以次级计数显示改动；有未提交改动或进行中操作时默认进入工作区，干净仓库默认 HEAD。**验收**：不靠颜色也能认出工作区；空仓库可进入首次提交；刷新保留草稿与当前上下文；历史筛选或读取失败不能使有效工作区无入口。启动默认规则由 D3 确认；文件高亮按已确认的标签方案执行，详见 T2。
- [ ] **R08 / P1 · 分支选择与提交菜单可达**。2026-11-16 已原生验证 980×640 下 35 个长 Unicode 分支、两个本地远端、无结果 Enter、鼠标选择与聚焦 Enter checkout，以及 origin/topic / backup/topic 搜索隔离。截图发现并修复长分支撑破顶栏、标题与搜索重叠；中英文 × 标准 / 1.15 字号的布局回归 RED→GREEN，全量测试 / race / vet、两种构建、引用动作和完整工作台原生回归通过。新增原生坐标断言检查顶栏与搜索不越界、不重叠；新增 `native-lobe-menu-access-smoke.py` 在 980×640、35 个长 Unicode 分支及两个本地远端下，验证历史菜单 18 项边界均在窗内、Down 逐项到达、分支 picker / 底部 Reset 坐标激活和 Reset 的 Enter 激活，取消后 Git 状态不变。后续同脚本通过作者 / 日期 / 近七天 / 本月四种坐标过滤：可见查询与结果正确，搜索 Enter 不改变 checkout；无结果与真实读取失败明确区分，失败保留旧记录与 stale 提示，恢复 fixture 后坐标 Retry 成功。证据 `build/lobe-menu-access-sfbuiw3v/results.json`。最新同脚本通过全部 11 个表单 / 确认入口的坐标打开与取消，加四种过滤共 15 项鼠标激活；结果 `build/lobe-menu-access-vmz51dcg/results.json`。三项复制命令尚未实际点击（避免覆盖当前系统剪贴板），R08 保持未完成。推荐复用已有可搜索、可滚动 picker，减少长菜单中的逐分支动作；常用动作按「浏览 / 分支整合 / 编辑 / 复制」组织，危险动作明确点名源与目标。**验收**：30+ 分支、长名、多远端、640 高窗口下，全部动作鼠标与键盘都可达；失败与无结果区分、可重试；筛选历史不会意外 checkout；搜索 Enter 不作用于已被筛掉的旧分支。需自建组件时先评估 MyGo 能力，避免复制整套菜单样式。
- [ ] **R09 / P2 · 视觉一致性收尾**。2026-11-16 已修复超长当前分支导致的顶栏与 Branches 页溢出，分支名中间省略并保留完整 tooltip / 辅助描述，页副标题按剩余宽度收缩。980×640 英文浅色、1320 标准浅色及 980 大字号深色原生几何断言通过；说明栏改为随换行展开，窄面板回归与完整验证通过，最终大字号截图已查看。随后重跑当前工作台 16 组 × 6 场景显示矩阵通过；最新确认保护修复后再次完整通过，96 张截图和快照在 `build/narrow-shots-82hkcttz/`、日志 `build/lobe-latest-visual-matrix.log`。核对列宽、活动标签 / 关闭入口、提交可用状态与 Git 指纹，抽查窄窗大字号表单 / 长标签和宽屏长正文代表图；该矩阵不覆盖应用全部页面 / 交互状态，不据此关闭 R09。设置页后续补独立验收：最小 560×480 下，中英文 × 浅深色 × 标准 / 1.15 字号共 8 组布局回归通过，关键选项不横向越界、底部重置可滚到且坐标打开设置窗口所属确认。原生默认 820×720 同样 8 组通过，24 张外观 / 底部 / 确认截图在 `build/lobe-settings-layout-40ndm5ll/`；核对所有交互控件横向边界、重置 / 取消坐标激活、Git 与偏好文件精确不变。代表大字号截图已查看；后续 `--minimum-window` 用隔离 Go overlay 仅将设置窗口初始尺寸设为已有 560×480 下限，独立二进制八组原生检查全部通过：实际滚动视口宽度、交互控件横向边界、底部 Reset / Cancel 坐标激活及 Git / 偏好文件保持。结果 `build/lobe-settings-layout-4qt1xdmo/results.json`、日志 `build/lobe-settings-minimum-native.log`，大字号外观 / 底部 / 确认代表截图已查看。没有改变产品源码或普通二进制；这是原生最小初始尺寸验收，不是实际拖动缩窗证据，也未覆盖设置所有悬停 / 焦点状态。按依赖次序处理：文件选择与摘要可见状态一致 → 设置命名 → 长路径与长引用可辨认 → 大字号和宽屏预算 → 胶囊省略 / 排序等细节。**验收**：选中态、悬停态、键盘焦点可区分；完整路径和引用可获取；中文 / 英文 × 浅色 / 深色，标准和大字号下无关键控件裁切，辅助文本对比度沿用既有 ≥4.5:1 检查。头像、悬停卡片、额外节点样式暂不作为通过门槛。

### M2：日常试用与主线交付需求

- [x] **R10 / P1 · 接通本地改动自动检测**。2026-11-16 已接通独立于自动 Fetch 的本地检测：复用 `gitrepo.ChangeStamp`，补充当前改动文件的大小、mtime、模式及符号链接目标观察；前台约 3 秒、后台约 15 秒，聚焦立即检查。刷新 / 写入 / 关闭取消旧探测，校验仓库与代次，失败退避并去重提示；刷新保留草稿与焦点，并更新补丁缓存代次。生命周期及真实 Git 回归 RED→GREEN；全量测试 / race / vet、两种构建与两套原生冒烟通过。`native-local-watch-smoke.py` 关闭自动 Fetch、全程不按 Refresh，验证再次保存、未选中 Unicode 文件、外部暂存 / 提交 / 切分支、子模块 HEAD 更新及 HEAD / Index 保护。文件内容观察依赖元数据；刻意保持相同大小、时间戳和模式的字节改写不保证识别。
- [ ] **R11 / P1 · 候选主线能力差距表**。2026-11-16 已建立 [实际能力差距表](docs/lobe-capability-gaps.md)，区分后端、Lobe 入口、当前 / 历史原生证据与未验收。确认 Clone / Init / 最近仓库尚无新版入口，旧 desktop 原生脚本不能证明新版闭环；另复现 Amend 只预填标题导致正文遗漏，已修复完整消息预填，真实 reload 回归 RED→GREEN，全量测试 / race / vet、两种构建及新增预填断言的完整原生冒烟通过。新增 `native-lobe-sync-smoke.py` 已原生通过 Push 取消 / 首推设置 upstream、Fetch 保护本地状态、FF Pull 及分叉拒绝；后续扩展两个本地裸远端，通过 main → backup/publish/nested 的取消 / 发布、origin 不变、Fetch 双方新引用，保留 HEAD / branch / Index / 暂存二进制和未暂存 Unicode 字节；日志与截图见试用记录。新增 `native-lobe-stash-smoke.py` 通过含未跟踪保存、Apply、Pop、Drop 取消 / 确认及重叠拒绝保护；Git 失败可能恢复未跟踪文件的边界已独立复现并记录。后续同脚本通过 Apply / Pop 实际三方冲突、stash / HEAD 保留、Index 三阶段与双方文本、恢复说明 / Git stderr，以及外部解决暂存后的精确字节；应用内编辑仍未验收。新增 `native-lobe-worktree-smoke.py` 原生通过创建 / 切换 / 返回、移除取消、脏目录拒绝和干净移除后保留分支；后续已通过 toast Delete 的坐标点击，确认只删除原分支且 main HEAD / Index / Unicode 字节不变。完整 Worktree 脚本现 8 组通过，新增锁定菜单禁用、确认后新锁定目标拒绝、ignored 二进制文件拒绝及精确字节 / HEAD / Index 保留；ignored 拒绝原因后续已完成当前语言映射，保留原路径和后端旧错误文本，真实 Worktree / cherry-pick 回归 RED→GREEN、全量 Go / race / vet / 构建、原生 Worktree / 完整工作台通过。新增 `native-lobe-submodule-smoke.py` 通过 Update 初始化 / 回到 Index SHA / dirty 拒绝及打开 child 独立历史；child Worktrees 路径已修复并通过主 / linked 子模块真实仓库回归、原生 Current / 真实目录断言及全量门槛；跨 root 错误提示已复核：最新原生五组通过，先保存父仓库真实更新拒绝，再在 child Worktrees / History 中检查错误容器与正文均不残留，父子 HEAD / refs / Index 条目 / 状态及 .gitmodules / 脏文件字节保持；结果 `build/lobe-submodule-psyuv4uc/results.json`、日志 `build/lobe-submodule-root-followup.log`，三张前后截图已查看。泛用通知归属回归也通过；首次探针遇到自动重读令菜单禁用，补就绪后重新取菜单目标而未改产品。这里只覆盖已完成错误后的切换，不声称运行中竞态或 OS 焦点全覆盖。新增 `native-lobe-operation-smoke.py` 通过真实 cherry-pick 冲突的 Continue 禁用、Abort / Skip 取消确认及外部解决后的 Continue 完整消息 / Index 保护；后续已扩展 Rebase / Merge / Revert 独立冲突操作卡，完整 11 组通过，核对禁用、取消确认、Abort / Skip 恢复、Continue 的完整消息 / 父提交及 Merge 提交框生成双父提交；冲突发起和解决由 fixture 执行，应用内编辑仍未验收。新增 `native-lobe-reference-smoke.py` 经详情 footer 坐标点击，原生通过非当前分支 Push 取消 / 发布 / upstream、分支重命名保留 tip / tracking、本地分支删除、附注标签推送 / 删除和远端分支租约删除；每段核对主 HEAD / Index / Unicode 字节、取消与目标范围。首轮同名本地 / 远端行歧义修正后完整重跑通过，最终截图已查看。Force / prune 后续通过顶栏 focus + Shift+F10 的应用内菜单原生验收：取消与分叉替换、外部远端推进且 fetch 后显式旧租约拒绝、错误详情保留本地化说明和 Git 原文；prune 只删除过期跟踪引用，同名本地分支及完整脏状态保留。原系统右键自动化限制不适用于这条已验证路径。同一引用脚本另通过 footer 发起无冲突 Merge / Rebase 的取消 / 确认，核对双父提交、目标父提交、完整源消息与双方文件；后续 native-lobe-integrate-smoke.py 已通过历史行菜单发起 Cherry-pick / Revert 的取消与确认：新提交父 SHA、完整源说明 / Revert 原 SHA、准确文件字节及 topic 引用保留均核对；只覆盖无冲突正常发起，应用内冲突编辑仍未补齐。新增 native-lobe-reset-smoke.py 原生通过 Soft / Mixed / Hard Reset 与 Undo：历史行 Shift+F10 应用内菜单可用；取消与 Hard 二次确认、HEAD / Index / 工作字节及未跟踪二进制保护均核对。新增 `native-lobe-conflict-side-smoke.py` 已通过 Merge 冲突文件菜单 Ours / Theirs 两组：精确采用原 Index 第 2 / 3 阶段字节并暂存，取边不移动引用，随后 UI Commit 核对双父提交 / 完整消息 / topic 与无关二进制保护；结果 `build/lobe-conflict-side-i1wtl9ae/results.json`。后续 Rebase 真实取边语义和 Abort 恢复通过；二进制 / 删除型冲突的 Merge / Rebase × Ours / Theirs 共 8 组也通过，核对精确阶段字节或暂存删除、Merge 完整提交与 Rebase 原分支恢复。总览 / 文件标签已补中英文外部处理和删除后果说明，代表原生提示滚动可见截图已核对，日志见 `build/lobe-conflict-shapes-check.log`、`build/lobe-conflict-hints-ready.log`。Rebase 取边后 Continue 后续文本 / 二进制 / 删除 × 两侧共六组通过，核对空重放省略或正确父提交 / 原消息 / 作者、main / topic 身份、精确 Index / 工作字节或删除及无残留操作；日志 `build/lobe-rebase-side-continue.log`、`build/lobe-rebase-side-continue-shapes.log`。仅覆盖整文件取边和 Continue，不等于三方手工编辑或其他操作取边。R11 保持未完成。按 UI 实际操作核对：打开 / 最近仓库 / Clone / Init、暂存与完整提交说明（含 Amend）、Fetch / Pull / Push 与 upstream、Stash、冲突继续 / 中止、worktree、submodule、偏好持久化。分别记录「后端已有 / UI 已接通 / 原生已验证 / 尚缺」，不复制旧版能力表充数。**验收**：试用必需流程逐项通过；无法支持的流程显示准确边界和可用退路；Amend 对 HEAD 的完整说明处理先核对，不能只凭标题预填测试判定通过。
- [ ] **R12 / P1 · 仓库入口与恢复**。D5 确认后补最近仓库与启动规则；建议先列最近仓库，避免新增自动重开开关。**验收**：启动空页有打开入口，近期路径可辨认；命令行指定路径优先；移动、离线或无权限的目录给原因与重新定位入口，不把暂时不可用当作删除依据；不自动改写仓库或联网。Clone / Init 若纳入本批，只复用已有后端并使用明确的新目录。
- [x] **R13 / P1 · 打包、偏好与测试对齐**。2026-11-19，`make app` 默认构建标签式工作台，`make desktop-app` 构建独立旧桌面回退包。新版读取旧偏好仅补缺失项，保留原文件。`make native-check` 用实际标准打包配方验证打开 → 外部编辑刷新 → 暂存 → 提交 → 同步，以及 Merge 冲突取边、Stash 和子模块。`make desktop-native-check` 九组通过，两套验证使用隔离偏好。[试用记录](docs/lobe-trial-log.md)保留证据及人工验证边界。
- [x] **R14 / P1 · 文档同步与试用反馈**。2026-11-16 已同步 README、快捷键和原生验证对象；`docs/packaging.md` 明确默认 desktop / Lobe 入口、独立偏好、旧安装包不会自动更新、实际版本号注入与签名边界。原观察区车道间距已改为当前 20 / 最低 6 DIP。新增 [Lobe 试用记录](docs/lobe-trial-log.md)，记录 macOS 27.2.1 arm64、未提交源码状态、实际脚本场景 / 结果及一次未明原因的选择跳变复现。文档与隔离原生试用记录已交付；尚无新增用户日常开发、真实远端或跨平台会话验收，不据此进入分发准备。

### M3 / M4：跨平台与后续范围

- [ ] **R15 / P1 · 跨平台支持矩阵**。D6 明确 Windows / Linux 的顺序、最低系统、架构和 Linux 显示环境；先探测 MyGo 原生依赖与构建可行性，再承诺日期。适配默认打开、文件定位、终端、窗口 / 菜单、快捷键修饰键、配置路径及 Git 发现。**验收**：每个目标系统都实际验证 Unicode / 空格 / 特殊路径、CRLF、Git 认证、hooks、子进程取消、外观与高 DPI；单独标记无法验证的场景，不默认 Unix 取消逻辑在 Windows 等价。
- [ ] **R16 / P2 · 大仓库体验预算**。2026-11-16 已采集 [本机后端读取基线](docs/performance-baseline-2026-11-16.md)：Apple M4 / 16 GiB，10,001 文件 / 12,024 提交 / 25 分支，以及独立小仓库 / 子模块样本；记录三组耗时、Go 分配量和复现方式，运行后核对 fixture 规模 / HEAD / 干净状态并清理本次临时目录。History 300 / 2,000 中位约 56.9 / 278.9ms，包含 Git 子进程；不是 UI 帧耗时，也不是峰值 RSS。后续当前 Lobe 三轮原生基线通过：首批历史中位 0.453s、Cancel 0.016s、Retry 至完整历史 2.312s，未变刷新至 ready / 完整历史 1.724 / 3.503s；初测 End 至含万文件的初始提交约 2.418s，应用采样最大 RSS 中位 1,915,712 KiB。分阶段采样与独立回归确认屏外 Files 行 / 补丁节全部构建；已改为视口裁剪，滚动范围 / End 焦点 / 跨节光标回归 RED→GREEN。全量检查、构建、原生工作台及 16 组显示矩阵通过；同内容 fixture 复测万文件选中阶段 RSS 中位从 1,772,432 降至 240,960 KiB，End 约 0.694s，39 个 Files 行 / 14 个节标题而非全部构建。结果包含自动化往返，RSS 非精确峰值，非帧耗时。整批审查四项交互修复后，全量 / 构建和原生复测通过；万文件 End 中位 0.690s，刷新保留原 SHA 后完整遍历中位 2.638s，整段应用采样最大 RSS 中位 250,752 KiB。早期刷新跳回 HEAD，不能直接当成正确保留上下文时的性能；旧数字和新样本均保留。单 tracked 文件外部保存 / 恢复的自动刷新后三轮通过：保存至改动状态中位 3.256s，恢复至干净状态中位 2.843s，HEAD / Index / refs 和最终文件字节保留；工作树选择至补丁文本可访问中位 34ms，不是正文屏内可见或帧延迟。长单文件历史补丁后续补 4,000 行全替换 / 约 8,000 条增删行的三轮原生样本：首屏 49 个行文本控件，选中至首行可访问中位 80ms，End / Home 两端导航约 0.699 / 0.697s，视口与仓库指纹断言通过；未采长补丁 RSS。宽 DAG 后续补 32 并行分支 / 32,002 提交的三轮原生样本：加载仍进行时 End 选择并显露，Cancel 中位 21ms，Retry 完整遍历约 12.179s；完整历史仅构建 27 个行控件，根提交图谱与仓库指纹通过。万文件父仓库＋已初始化 1,000 文件子模块联合样本后续三轮通过：父 10,003 tracked 条目 / 12,025 提交，子文件保存到父改动状态中位 5.894s，恢复到父干净中位 2.828s；两层完整指纹保持，整段应用采样最大 RSS 中位 263,232 KiB。批量增量样本后续三轮通过：同时改写 1,000 父文件＋一个子文件，保存至 1,001 条改动状态中位 6.106s，恢复至干净 2.492s；精确路径、两层身份和全部原字节恢复通过。工作区 Files 每轮全部构建 1,001 行，快照往返中位 139ms，批量阶段应用采样最大 RSS 中位 494,800 KiB；包含历史重读 / 自动化 / 恢复，不能归因于行控件独占或称为帧时间。工作区 Files 后续已裁剪：2,000 行构建回归 RED→GREEN，保留完整组内位置、End 跨到 staged 版本和屏外记忆焦点；全量 Go / race / vet、两种构建、三轮批量复测及完整详情标签原生通过，批量 Files 每轮 38 行，应用采样最大 RSS 中位 399,856 KiB。首次原生首轮曾在恢复后 60s 无响应，随后同源码三轮未复现；完整失败记录保留且脚本补超时栈留证，根因仍待定位，不以重跑通过关闭。后续最新构建定向调查在第二轮同阶段再次超时，已保存 `build/lobe-performance-ojrzcpxj/app-2.log` Go 栈及阶段 JSON；未见应用 Git 读取 / 锁循环，主线程在原生事件循环调用，尚不足以区分原生或客户端等待。后续双侧采样与实时树确认 Git / 模型已恢复但界面帧停止，观察器与自动化请求正常；临时依赖诊断进一步定位 MyGo display link 标记仍为运行而长时间不交付帧，RequestFrame 因判断顺序跳过已有恢复路径。已保存[小范围依赖候选与复现入口](docs/diagnostics/mygo-display-link-recovery.md)：原实现精确 RED，候选 Darwin 包、应用 Go / race / vet、三轮同规模批量恢复和全部详情标签烟测通过；普通 e2e 已恢复，默认依赖未接入候选，停帧项继续开放，不把候选通过称为默认应用已修复。详见性能基线。长补丁 / 宽 DAG 的 RSS 后续各三轮补采通过：应用采样最大 RSS 中位分别 201,408 / 715,024 KiB，Git 指纹和原有导航断言保持；宽 DAG 较高值出现在根导航阶段，未做分配归因，不称为精确峰值。工作区长补丁后续三轮通过：总览首屏 20 个、单文件标签 49 个增删控件，End / Home 与完整指纹保持；单文件标签原先构建全部 8,000 行，已裁剪并通过跨视口 gutter 拖选回归 RED→GREEN、全量 Go / race / vet、两种构建、历史长补丁与完整详情标签原生。工作区整段应用采样最大 RSS 中位 207,312 KiB。当时仅只读；后续 `--stage-line` 三轮通过 8000 条增删行导航后单删除行暂存，Index 精确只去掉原版本首行、工作区字节 / HEAD / refs 不变，只有目标文件 MM，结束后自建夹具完整指纹恢复。结果 `build/lobe-long-patch-l50k34dn/results.json`、暂存截图与完整日志已核对；首轮探针因未显式键盘 focus 未打开菜单，修正探针后通过。随后 `--stage-unstage` 三轮从精确 Staged 标签对该删除行执行 Unstage line，在任何外部清理前完整指纹已恢复；结果 `build/lobe-long-patch-8k4qu0s7/results.json`，最终截图 Staged Files 0 / Line unstaged。追加 40 行跨视口删除选区对照：普通依赖下原生输入在 37–38 行附近停止推进；无原生窗口回归与完整 Go 测试通过。隔离帧恢复候选＋状态诊断 overlay 三轮通过精确 40 行暂存 / 取消暂存，取消暂存前未做外部清理即恢复完整 Git 指纹，结果 `build/lobe-long-patch-ja37ajwx/results.json`；默认依赖未改变，不能计作普通应用多选验收。持久入口 `scripts/check-mygo-display-link-recovery.py --native-long-range` 后续实际通过：精确原实现 RED、候选完整 Darwin GREEN、无诊断 overlay 的三轮 40 行精确往返均通过，结果 `build/lobe-long-patch-u19cb976/results.json`，第三轮截图已查看，普通 e2e 自动恢复；候选验证子项完成，默认依赖的多行原生验收仍开放，详情见依赖诊断文档。普通依赖的 `--stage-unstage --line-kind add` 后续三轮通过长补丁末条添加行往返：Index 精确为原 after 全文加选中的 working 末行（未选删除行保留为上下文），working / HEAD / refs 保持，取消暂存后外部清理前完整指纹恢复；结果 `build/lobe-long-patch-_4zuou9o/results.json`，暂存 / 取消暂存截图已查看。首次新探针 End 在无行选区时回退到历史，显式重新选择当前 diff 行后通过，未改生产代码。普通依赖的 `--discard-undo` 后续对单删除行 / 单添加行各三轮通过：确认前完整指纹不变、确认后工作区精确只恢复 / 移除所选行，Index / HEAD / refs / 状态保护，原生 Undo 后、外部清理前完整指纹恢复；结果 `build/lobe-long-patch-e_6_gg3_/results.json` 与 `build/lobe-long-patch-fymi2qa2/results.json`，两种丢弃 / 撤销截图已查看。删除行首次探针错误预期恢复在文件末尾，依据实际替换配对顺序与快照修正后通过；不是产品修复。隔离候选的 40 行添加选区后续三轮通过持久入口 `--native-long-range --range-line-kind add`：Index 在原行对应位置精确插入选中的末 40 行，Staged 从新行号 3962 选到 4040（跳过上下文），取消暂存后完整指纹恢复；结果 `build/lobe-long-patch-gv7wvl0w/results.json`，暂存 / 取消暂存截图已查看，普通 e2e 自动恢复，默认依赖不变。此为候选验证，不关闭默认依赖的多行停帧项；多行丢弃及系统文本跨屏拖选仍属未验证边界。加载期后续三轮通过 12 次逐次 settle 滚轮、搜索逐字输入与清空，并确认操作全程历史仍读取；滚动位移、精确输入值、匹配行及最终指纹通过。该样本不等于硬件高频轮事件或提交草稿 / 输入法编辑验收。多 hunk 后续补九处分散替换样本：真实 Git 核对 9 hunk / 18 增删行，历史与工作区各三轮通过，原始第 4,000 行显露和完整指纹保持；仅只读导航，不代表密集 hunk 或部分写入验收。宽单行后续补 ASCII 宽行：历史与工作区各三轮，工作区文件标签横向滚动后行尾坐标及截图在屏内，完整指纹保持；自动刷新会重载补丁，探针重新选中当前 gutter 后测 Home，未声称保留旧行选区。工作区总览的就绪补丁节与折叠标题后续已按视口裁剪：1,000 节展开 / 折叠、完整滚动范围与远处选中行定位回归 RED→GREEN；全量 Go / race / vet、两种构建和完整详情标签原生通过。独立 1,000 脏文件原生样本在补丁就绪后首屏仅构建 3 个节标题、末尾 6 个，滚到最后补丁再返回及 HEAD / refs / 原始 Index / 全部字节保护通过；结果 `build/lobe-overview-sections-ny31jbxn/results.json`。加载 / 错误 / 空补丁占位节后续已按实际列宽及字号测量高度并复用裁剪：1,000 节 × 中英文 × 标准 / 1.15 字号 × 两种列宽共 24 组回归通过，核对普通布局节高、完整滚动范围、滚到第 900 节及返回、实时列宽切换和可见加载优先级。全量 Go / race / vet、两种构建通过；千文件原生复测仍为首屏 3 个 / 末尾 6 个标题、实际末尾补丁与返回及完整指纹通过，结果 `build/lobe-overview-sections-lv8fimhd/results.json`。占位状态的裁剪结论来自实际布局回归，该原生脚本不强制制造全部占位状态；冲突说明和尚无有效列宽的冷启动首帧保持普通布局，未测新帧耗时 / RSS。完整详情标签回归随后十组通过，日志 `build/lobe-placeholder-detail-ready.log`；探针依据实际禁用状态补上 Cmd+Enter 前等待 Commit 就绪，保留 hooks 与精确写入断言。首次屏内文件点击后意外切换历史 / 页面的原因未确认，完整证据保留于 `build/lobe-placeholder-detail-failure-v1ixczhw/`，不以重跑通过关闭。Unicode 宽行已修复并验收混合中文 / 全角 / ASCII 与 1.15 字号工作区文件标签：实测字形回归覆盖估计过窄 / 过宽，实际行布局回归复现代码区少分配约 14–16 DIP，预留统一为实际 138 DIP 后通过；全量 Go / race / vet、两种构建、完整详情标签原生及新三轮宽行通过，截图中完整尾标记无省略号，完整仓库指纹保持。结果 `build/lobe-long-patch-i6oljx2b/results.json`，日志 `build/lobe-unicode-row-final-check.log`；20,000 DIP 上限及任意 Unicode 字形精度仍有限制，证据见性能基线。历史 Summary / 历史文件标签后续混合 Unicode + 1.15 字号横向行尾各三轮通过，右端坐标在视口内且截图完整显示 wide-tail-3999，End / Home 与完整指纹通过；结果 `build/lobe-long-patch-c0si5seo/results.json`。工作区总览扩展流程三次被导航 / 横向回滚后的读位变化打断；前两次辅助树显示自动重读，第三次临时 overlay 也确认末行定位后观察器触发仓库重读，只是瞬态未被辅助树采到。诊断后普通构建已恢复，探针将行尾测量与当前补丁端点恢复分开，等待就绪后获取当前 gutter，不要求保留重读前选区；普通固定依赖下四种视图完整复验已通过：工作区 / 历史各三轮，工作区总览、工作区文件标签、历史 Summary、历史文件标签均核对横向位移 / 行尾边界 / End / Home，完整 Git 指纹不变。结果 `build/lobe-long-patch-zg00bb_l/results.json`、`build/lobe-long-patch-0iowt7s3/results.json` 已完整核对，四种视图第三轮截图均完整显示 wide-tail-3999，无省略号；日志 `build/lobe-unicode-projections-current-target.log`。途中另一次探针失败定位为重读后使用旧 gutter 标签，目标选择改为就绪后的当前树；不证明自动重读保留旧选择或读位。未改产品行为或默认依赖。加载 / 错误 / 空补丁占位节裁剪与该混合 Unicode / 1.15 字号四视图横向行尾子项完成；冲突说明裁剪、冷启动首帧构建量、扩展子模块联合负载、Unicode 极长行 / 任意字形精度、密集多 hunk 与部分写入、更广加载期交互及内存归因仍为未验证边界，按既定范围推进，R16 保持未完成。在主线确定后，分别测打开、增量刷新、深历史分页、宽分支图与长补丁；复用万文件 / 万提交 fixture，增加子模块与多分支样本。**验收**：记录硬件、仓库规模、耗时与内存；Git 不阻塞主线程；滚动、选中和取消仍响应；性能目标依据基线制定，不把 Git 命令耗时称为帧耗时。
- [ ] **R17 / P2 · 进阶能力候选**。文件树、命令面板、导航栈、交互式 Rebase、完整三方合并、递归子模块发布、签名详情、快捷键自定义与持久操作记录，按试用反馈逐项立项；不默认全部纳入首版。不增加新的强制操作或删除快捷键。
- [ ] **R18 / P1（分发前）· 交付门槛**。目标平台原生冒烟通过、已知限制明确、macOS 开发者签名 / 公证与其他平台安装方案准备完成，确认升级与回退路径；自动更新另行评估。发布前由用户亲自核对权限、删除及真实远端关键流程，并单独授权发布。

### 集中待确认的产品决策

此处是建议汇总，未答复时维持现状，不阻塞无依赖的修复。其余微调继续保留在后面的原清单，避免一次要求用户决定几十个 A/B/C。

| ID | 真正需要决定的问题 | 建议与取舍 | 解锁需求 |
| --- | --- | --- | --- |
| D1（已执行） | Lobe 达标后接管 | 默认 `make app` 已切换至 Lobe，旧桌面回退包保留 | R11、R13、M2 |
| D2（已确认，更新） | 四列，提交 / 摘要与多文件整合为标签式详情 | 按最新预览：第三列为全高文件列表；第四列首标签为提交 / 摘要并包含全部补丁，后续为可关闭的单文件标签；替代此前上下分区方案 | R06、T1–T6、相关设置命名 |
| D3 | 打开仓库默认进入哪里？ | 推荐有改动 / 进行中操作选工作区，否则 HEAD；也可保持始终第一条提交，行为改动较小。进入该目标后的首标签规则已在 D2 / T2 中确定，不与启动默认目标混为一项 | R07、R09 |
| D4 | 两条正确性规则：冲突残留标记、子模块 ignore | 推荐冲突残留阻止直接标记完成，特殊合法文本另设明确流程；子模块始终显示实际状态与 Diff。前者比确认后放行更保守，后者偏离用户 ignore 配置但信息一致 | R01、R05 |
| D5 | 最近仓库是否自动重开？ | 推荐先做最近列表、由用户选择；自动重开更省一步，但要处理不可用路径、草稿恢复与自动联网的关系 | R12 |
| D6 | 跨平台试用顺序与平台基线？ | 推荐先完成 macOS 日常试用，同时做 Windows / Linux 构建探测；探测结果和用户实际平台决定下一平台，不凭空选发行版或架构 | R15、M3 |

### 已确认布局说明（D2，2026-11-15 更新；原生主体已实现）

用户已要求按最新标签式预览制定改进计划。本节正式替换此前的「文件与摘要上下分区」方案。设计依据为本机交互预览 `design/workbench-preview.html`，参考用户提供的 Summary 与多个文件标签截图。预览已验证标签切换、关闭 / 重新打开、草稿保留与工作区 / 历史切换；这些证据仅适用于 HTML 原型。

```text
定位栏 │ 提交列表 │ 全高文件列表 │ [提交 / 摘要] [文件 A ×] [文件 B ×]
       │ 工作区   │ 按状态分组   │ 当前标签内容
       │ 历史线图 │              │ 首标签：提交信息 + 全部文件补丁
       │          │              │ 文件标签：单文件补丁
```

- **已确认范围**：Lobe 主线、四列结构、第三列只放文件、右侧标签与正文连为一个区域；首标签工作区叫「提交」、历史叫「摘要」，不可关闭；文件标签可打开、切换、关闭和重新打开；切换保留提交草稿。
- **总览行为**：工作区首标签为提交框、进行中操作与全部改动，区分已暂存 / 未暂存；历史首标签为完整提交说明、元数据与全部文件补丁。按文件折叠 / 展开，并保留二进制、删除、重命名、空差异、gitlink 和读取失败的准确表达。
- **单文件行为**：文件列表点击后打开或激活对应标签，重复点击不增加副本；同名不同目录用完整路径区分，同一路径的已暂存 / 未暂存用分组身份区分。单文件标签只画该目标的内容，不再滚到整份提交补丁的某个位置充当单文件视图。
- **方案取舍**：上下分区方案被本次决定替代；五列并排不采用。原生已有的「摘要 + 一个文件」升级为「含全部补丁的首标签 + 多文件标签」，不能只调整标签外观就标为完成。
- **实施默认规则（本计划提出）**：首次进入一个目标显示首标签，不自动为全部文件创建标签；点击文件按打开顺序增加标签。关闭活动标签后优先激活右侧邻居、没有则左侧、最后回首标签；关闭非活动标签不切换当前内容。示例预览为了展示样式预置了文件标签，正式实现按需打开。
- **上下文与恢复**：状态以仓库根、工作区 / 固定提交 SHA 区分；工作区文件额外带暂存分组。首期只保留当前仓库的工作区和最近一个历史提交的标签状态；选择不同历史 SHA 清理旧历史视图，切仓库清理全部标签，不写入偏好文件。工作区草稿沿用现有生命周期，不因切标签或关闭文件丢失；不顺带承诺跨仓库草稿恢复。
- **刷新规则**：目标和 Diff 内容不变时保留标签与各自滚动；内容变化清理旧行选区并重新读取。确认文件离开对应分组或从提交文件列表消失后移除对应标签；读取失败不等同消失，应保留错误与重试。取消暂存 / 暂存后不得悄悄让旧标签指向另一分组。切 SHA、切仓库、刷新和写入均拒绝过期读取。
- **宽度与范围**：沿用 980×640 最小窗口，第三列 180–300px 自适应、最右正文至少 300px；980 / 1320 / 1600 宽均验证。标签过多时可横向滚动，当前标签始终可见；完整路径可获取。文件树、新增快捷键、标签拖动排序、跨重启恢复和新增设置开关不纳入本批；原有横向拖动需求仍独立待定。

### 标签式工作台改进计划（R06，T1–T6）

> **执行说明**：已获授权开始实施，使用 `superpowers:executing-plans` 按任务推进；如另行选择多 Agent，遵守独立 worktree 与合并后完整验证的规则。每项先建立能复现对应行为的回归，再实现并验证；commit 与发布仍需用户单独明确授权，不能按任务自动提交。
>
> **实施进度（2026-11-15）**：T1 / T2 / T4 / T5 完成；T3 总览、单文件投影、双版本读取、局部错误重试与准确行级操作已实现，动态可见区读取优先级和跨视图补丁复用仍作为性能尾项保留。T6 本批验收完成：2000 行虚拟化测试；原生双版本 / 同名路径 / 草稿 / 多标签 / SHA 与仓库切换 / 设置显隐 / 文件焦点 / 指定目标暂存 / 刷新 / 超限重试；16 组、64 张截图及全部组列宽核对。代表截图已检查，未逐张人工检查 64 张。独立审查发现的重要问题「折叠分组后刷新错用文件 / 暂存版本」经双用例 RED→GREEN 和原生回归修复；轻微问题「文件列表逐行 Tab 停靠」保留在 R03。`make check`（全量 Go / race / vet）、`make lobe-build`、`make lobe-e2e-build` 通过。证据与边界见 [DONE.md](DONE.md) 和 [原生验证说明](docs/e2e-automation.md)。本批不代表全部路线图或跨平台发布门槛完成。

**Goal:** 在 Lobe 原生主界面落实已确认的「提交 / 摘要 + 多文件标签」方案，并保留正确的 Git 操作目标与草稿。

**Architecture:** 复用 `internal/gitrepo`、现有文件列表和补丁渲染；独立管理标签身份、活动目标及滚动状态。历史复用 `ShowAll` 与 `commitSections`，总览与单文件共用读取结果；工作区总览新增有界后台读取，标签切换与 Git 写入分别管理。

**Tech Stack:** 当前 `go.mod` 的 Go 1.27.3、MyGo 与 Lobe MyGo 原生 UI；系统 Git；现有 Go / race / vet 与 MyGo 原生自动化。

**Spec:** 本文件上方「已确认布局说明」与 `design/workbench-preview.html`；原型的示例数据、预置标签和浏览器小屏重排不直接复制为产品行为。

#### 全局约束与重点风险

- 只改 Lobe 界面及必要的共享读取辅助；不迁移 `make app`，不重做 desktop，不改变 hooks、提交仅含 Index、特殊路径、部分暂存与现有危险操作确认规则。
- UI 主线程不执行 Git；只读任务有界、可被目标变化作废；错误可重试，不把失败画成空文件或空提交。不通过放大现有输出上限掩盖问题。
- 标签、滚动与折叠是阅读状态；关闭标签不删除文件、不改 Index。行选区按具体补丁版本管理，正文草稿独立于标签组件身份。
- 首要回归：同路径双分组（T1 / T3）；切提交 / 切仓库后的过期结果（T1 / T3）；单文件失败而其他文件正常（T3）；长名与大量标签（T2 / T6）；隐藏或关闭标签后的快捷键与草稿（T4 / T5）。

#### 文件与职责

| 文件 | 计划职责 |
| --- | --- |
| 新建 `internal/lobedesk/detail_tabs.go`、`detail_tabs_test.go` | 标签身份、顺序、激活、关闭、上下文协调；不执行 Git |
| 修改 `internal/lobedesk/history_panes.go`、`history_panes_test.go` | 标签栏、全高文件列、总览 / 单文件内容分派、附着式选中外观 |
| 新建 `internal/lobedesk/detail_overview.go`、`detail_overview_test.go`、`worktree_overview.go`、`worktree_overview_test.go` | 提交 / 摘要总览与工作区多文件读取状态；渲染与可取消串行队列分文件维护 |
| 修改 `internal/lobedesk/app.go`、`commit_diff.go`、`history.go`、`workspace.go` | 接入选择与刷新、保留分文件读取结果、补丁范围、暂存身份与局部重试 |
| 新建 `internal/lobedesk/detail_focus.go`、`detail_focus_test.go`；修改 `keyboard.go`、`lines.go` 及相关选择测试；新建 `overview_lines.go`、`overview_lines_test.go` | 可见内容、焦点与快捷键守卫；清理旧行选择；总览操作固定差异目标 |
| 修改 `internal/lobedesk/settings.go`、`settings_page.go`、`i18n.go` 及相关测试 | 旧偏好映射、设置命名、双语文案与安全回退 |
| 新建 `internal/lobedesk/patch_virtual.go`、`patch_virtual_test.go` | 历史 / 工作区总览的可见行控件虚拟化，保留完整滚动高度与屏幕外键盘目标显露 |
| 修改 `scripts/native-narrow-shots.py`；新增 `scripts/native-detail-tabs-smoke.py` | Lobe 新布局截图与隔离原生交互验证，沿用 `docs/e2e-automation.md` 方法 |
| 修改 `docs/selection-model.md`、`docs/shortcuts.md`、README 与本文件 | 同步实际完成后的行为与验证对象，不提前宣称新版已打包 |

以下接口按任务逐步落地；执行时若与既有结构冲突，先在任务内统一更新调用方与测试，不能留下两套标签状态。

#### T1 / P1：标签身份与生命周期（先行基础）

**接口**：在 `detail_tabs.go` 定义 `detailContextKey{Root, CommitSHA string; Worktree bool}`、`detailFileKey{Path string; Staged bool}`、`detailTabKey{Summary bool; File detailFileKey}`、`detailTabsState{OpenFiles []detailFileKey; Active detailTabKey}`。提供 `(*detailTabsState).Open(file detailFileKey)`、`Activate(tab detailTabKey) bool`、`Close(file detailFileKey)`；`Activate` 对不存在的文件标签返回 false。总览键为 `detailTabKey{Summary:true}`；在 `app.go` 持有工作区与当前历史上下文及其阅读状态。

- [x] 增加 `TestDetailTabsDeduplicateByPathAndStage`：同 path / stage 打开两次只有一项，相同 basename 不同路径、同 path 不同 stage 分别存在。
- [x] 增加 `TestClosingDetailTabsChoosesTheExpectedNeighbor` 与 `TestDetailContextsDoNotLeakAcrossTargets`：覆盖活动 / 非活动关闭、最后一项回首标签、仓库 / SHA 切换、工作区往返历史。随后实现上述纯状态接口，并接入 `selectWorktree` / `selectCommit` / 打开仓库重置路径。
- [x] 运行 `go test ./internal/lobedesk -run 'TestDetailTabs|TestClosingDetailTabs|TestDetailContexts' -count=1`，要求全部通过；重复打开与跨分组测试应能识别仅按 basename / path 去重的错误实现。2026-11-15：通过；补充快速 SHA 往返、不经渲染的生命周期回归，全量 `go test ./...` 通过。

#### T2 / P1：全高文件列与多文件标签（依赖 T1）

**接口**：在 `history_panes.go` 接入 `(*app).openDetailFile(file detailFileKey)`、`activateDetailTab(tab detailTabKey)`、`closeDetailFile(file detailFileKey)`，统一调用 T1；`selectFile(path string, staged bool)` 与 `selectCommitFile(path string)` 保留现有入口签名。每个标签的滚动状态按上下文和目标独立持有。

- [x] 扩展 `history_panes_test.go`：`TestFileTabsOpenActivateAndClose` 断言重复点击激活已有标签；`TestSummaryDoesNotPretendASingleFileIsSelected` 断言首标签不把预读的第一文件画成当前打开文件；`TestDetailTabScrollIsKeptPerTarget` 覆盖 A → B → A 的滚动恢复。
- [x] 实现附着式标签栏：首标签不可关闭、文件标签带关闭按钮和完整路径提示；窄栏同路径双版本标明「已暂存 / 未暂存」。保留稳定控件身份，移除固定 `detailTabSummary / detailTabFile` 单一状态的旧分派，避免模型与渲染不一致。首标签始终可回到，多标签溢出支持滚动与活动项显露。
- [x] 文件列表维持全高；点击文件打开标签而非跳到整份 Diff 的位置。关闭标签后按规则同步文件高亮；首标签可保留列表的键盘游标，但不宣布某文件补丁正在显示。
- [x] 运行 `go test ./internal/lobedesk -run 'TestFileTabs|TestSummaryDoesNot|TestDetailTabScroll|TestTheFileTab|TestThePatchKeeps' -count=1`；原生布局确认无上下分割器、第三列不再承载提交框。2026-11-15：专项回归与 16 组原生截图矩阵通过，已检查代表截图。

#### T3 / P1：首标签总览与准确单文件补丁（依赖 T1 / T2）

**接口**：在 `detail_overview.go` 提供 `(*app).detailOverview(c *ui.Context)` 与 `loadWorktreeOverview()`；工作区条目用 `worktreeOverviewEntry{Key detailFileKey; Diff gitrepo.Diff; Loading bool; Err string}` 保存局部状态。历史保留 `[]gitrepo.FileDiff` 或等价的原始分文件数据，`buildCommitDiff` 继续作为聚合入口；单文件视图只投影该 path 的全部相关段，不能漏掉同一路径类型变更的第二段。

- [x] 增加 `TestHistorySummaryIncludesMessageAndAllPatches`、`TestFileTabOnlyDrawsItsOwnPatch`：说明与全部补丁出现在摘要；单文件不包含其他路径；同路径的类型变更两段都保留；历史只读且 merge 沿用第一父提交比较语义。
- [x] 工作区总览复用提交框与 `operationCard`，按已有分组显示顺序展示文件块；支持单文件与全部折叠 / 展开。读取在可取消的后台串行队列执行，不因逐帧绘制反复启动 Git；总览未完成时按块显示加载，不呈现为全部成功。
- [x] **T3 性能尾项**：2026-11-16 已实现工作区单文件 / 总览成功补丁缓存复用，按仓库、刷新代次、原始路径与暂存分组隔离，保留预算 16 MiB；显式重试仍读取 Git，刷新后不复用旧代次。队列动态优先读取视口内、展开且等待中的文件块，保留其他待办、串行读取、取消与固定目标重试；不在绘制中运行 Git。双向缓存、预算淘汰、空补丁、身份隔离、队列重排和实际 UI 滚动回归 RED→GREEN，最终全量测试 / race / vet、两种构建与完整原生冒烟通过。历史单文件原本已投影已加载提交补丁。缓存阶段一次原生历史选择跳变超时，原因未定；同一二进制重跑及本次最终回归均通过，调查记录保留，不把该偶发问题宣称修复。
- [x] 增加 `TestOverviewSeparatesIndexAndWorktree`、`TestOverviewReadFailureIsLocalAndRetryable`、`TestStaleOverviewCannotReplaceTheCurrentTarget`：真实 fixture 覆盖同文件双版本、某一文件超限而其他文件成功、固定目标重试、快速切 SHA / 仓库及暂存后的旧回调；失败不误判文件消失。
- [x] 接入刷新协调：同目标同内容保持滚动 / 折叠；写入后重读并清旧选区；确认条目消失才移除对应标签。增加 `TestRefreshedTabsKeepOnlyTheirOriginalVersion`，验证暂存迁组后不将旧标签悄悄重定向。
- [x] 运行 `go test ./internal/lobedesk ./internal/gitrepo -run 'TestHistorySummary|TestFileTabOnly|TestOverview|TestStaleOverview|TestRefreshedTabs|TestBuildCommitDiff' -count=1`；空仓库 / 删除 / 重命名 / 子模块沿用后端回归及文件级退路，二进制元信息有摘要控件测试，空提交 / 输出超限有原生验证。未逐种进行图形验收，不扩大此结论。

#### T4 / P0：选区、焦点与提交快捷键（依赖 T2 / T3；合并 R02 / 关联 R03）

**接口**：标签切换统一清理离开的行选区；调整现有 `canCommitShortcut() bool`、`worktreeDiffShown() bool`、`commitDiffShown() bool` 与文件列表焦点分发。可见性由活动标签内的实际内容及补丁目标判断，不再等价于 `detailTabShown() == detailTabFile`。

- [x] 增加 `TestDetailTabSwitchClearsInvisibleSelection`、`TestOverviewLineActionKeepsItsFileAndStage`：总览中点 B 的片段必须操作 B 的准确暂存版本，不使用后台残留的 `selectedPath`；选区不能跨文件 / 分组；折叠后不能复制或暂存不可见行。旧 Diff 校验继续有效。
- [x] 扩展 `TestCommitShortcutReachesTheHistoryCommitBox`：只有工作区「提交」标签可见、无弹层 / 表单且仓库可写时提交；文件标签或历史摘要中的 ⌘↩ 不提交后台草稿。增加 `TestDraftSurvivesClosingAFileTab` 覆盖标题、正文、Amend 状态不因标签组件重建丢失。
- [x] 标签栏作为一个 Tab 停靠区域，焦点在标签上时 Left / Right、Home / End 导航，关闭按钮可读屏辨认；文件列表焦点时 Up / Down 只走文件，输入框保留自身按键。新增 `TestDetailTabsKeepKeyboardOwnership`，覆盖引用弹层、详情隐藏与连续关闭后焦点回到有效控件。
- [x] 运行 `go test ./internal/lobedesk -run 'TestDetailTabSwitch|TestOverviewLineAction|TestCommitShortcut|TestDraftSurvives|TestDetailTabsKeepKeyboard|TestSpaceOnTheFocused' -count=1`；使用真实隔离仓库核对写入路径、HEAD、Index 与工作区字节，历史浏览不得写入。

#### T5 / P1：偏好兼容、设置与文案（依赖 T2 / T3）

**接口与映射（实施默认）**：保留现有 JSON 字段：`DetailsInfo` 控制历史摘要中的提交信息块；`DetailsChanges` 控制文件列；`DetailsDiff` 控制历史总览补丁与单文件标签。首标签保留，工作区提交框及改动入口沿用现有始终可用规则；`HideDetails` 仍收起整个详情区。历史信息与补丁同时为 false 时回退为显示信息块，避免空详情；打开文件标签和滚动状态不持久化，不新增开关或改写真实测试外的用户配置。

- [x] 在 settings 测试增加 `TestLegacyDetailSettingsMapToTabbedLayout`：旧字段、缺字段、全隐藏与文件列隐藏均可恢复；关闭补丁显示时回首标签并清理隐藏选区，重新启用可正常打开文件。已有偏好 round-trip 保留无关设置。
- [x] 设置与页面中英文统一为实际区域：「提交信息 / 文件列表 / 补丁」，说明补丁影响总览与单文件视图，工作区首标签为「提交」，历史为「摘要」。更新 `detailsCard`、`detailsLayoutHelp`、keep-one 提示、辅助功能名称及相关术语测试；不把设置恢复默认误记为清理仓库数据。
- [x] 运行 `go test ./internal/lobedesk -run 'TestLegacyDetailSettings|TestSettings|TestChineseStrings|TestEveryCopy|TestTheFileTab' -count=1`；同步 `docs/selection-model.md`、`docs/shortcuts.md` 与 README 的实际新行为，保留旧版 / 新版验证入口区分。2026-11-15：专项测试与 `make check`、`make lobe-build` 通过；图形验收仍属 T6。

#### T6 / P1：原生回归与交付核对（依赖 T1–T5）

- [x] 新增 `scripts/native-detail-tabs-smoke.py`，只用新建 fixture 与独立偏好：工作区首标签 → 打开 A / B → 暂存 / 未暂存双版本 → 关闭活动 / 非活动标签 → 返回提交草稿 → 历史摘要全部补丁 → 单文件 → 快速切提交 / 仓库 → 失败重试 → 刷新及暂存后协调。浏览阶段核对无仓库写入，写入阶段核对仅明确目标改变。
- [x] 扩展 `scripts/native-narrow-shots.py`：980 / 1320 / 1600 宽 × 640 高，中文 / 英文、浅色 / 深色；标准字号与大字号 1.15 的关键组合。覆盖长正文、同名不同目录、10 个以上文件标签、空提交、长路径；980 下文件栏 ≥180、正文 ≥300，无控件遮挡，活动标签和关闭入口可到达。大总览验证可滚动，不能一次创建全部不可见行的 UI 节点。
- [x] 运行 `make check`、`make lobe-build`、`make lobe-e2e-build`；图形会话解锁后运行 `python3 -I scripts/native-detail-tabs-smoke.py` 和 `python3 -I scripts/native-narrow-shots.py`。原生与脚本日志均需记录实际结果，锁屏或缺图形环境时明确未验证，不能用 HTML 原型检查替代。
- [x] 根据变更做一次独立审查，重点检查 T1–T5 的风险与既有行级操作回归；最多三轮，后续聚焦新增修改。把已完成项与证据归档 DONE，更新本文件「当前原生状态」；仍保留 Lobe 达标后才切换打包的 D1 门槛。

**执行顺序**：T1 → T2 → T3 → T4 → T5 → T6。T1 / T2 可形成第一批可验证成果，但总览、单文件隔离和行操作保护未完成前，不宣布 R06 已交付。T4 与 R01 的冲突写入保护均是相关流程的完成门槛；D3 启动规则、D4 子模块策略、自动检测 R10 与主线打包 R13 不因本计划而被默认批准。

### 验证与维护方式

- 本批已从规划进入原生实现：记录实际 Go / race / vet、构建、原生脚本与截图证据；未运行的项目明确保留，未经复现的问题仍标为「待验证」。T3 调度 / 缓存尾项、R03 列表 Tab 停靠和跨平台发布门槛未计为完成。
- 后续每个需求完成时记录实现入口、实际验证、已知限制；Go 检查用现有 `make check`，界面变更使用对应主线原生构建和隔离 fixture。锁屏时不把单元测试替代焦点、窗口与鼠标实测。
- 已完成项归档到 DONE 前先保留验证证据；迁移失败、丢草稿、写入目标错误、冲突误暂存等问题阻止对应阶段通过。已明确的 Git 保护规则不在此重新放宽。
- **近期工作顺序**：已确认的标签式工作台按 T1 → T2 → T3 → T4 → T5 → T6 推进，R02 与文件焦点衔接并入 T4；R01 冲突保护单独先调查、待 D4 确认规则后修复，R05 同理。随后接通本地自动检测 R10，再完成主线能力核对与打包迁移门槛；不把待确认需求混入标签布局实现。

---

## 原始需求与问题清单（保留上下文）

以下 A/B/C 与观察项保留原记录，不表示全部进入近期开发。优先级、推荐方向和依赖以上方规划为准；冲突处先复查代码再更新，不把历史讨论直接当作现状。

## 用户提出需求

- [x] 工作区不需要了，用户的实时改动和 Sublime Merge 一样，graph + 工作区合并。用户主界面就是当前 commit + 改动 一起显示（15:19 完成）。历史页成为唯一主界面，首行工作树 + 提交线图同屏；独立工作区页、它的三栏布局和 `workspaceSplit` 设置一起删除。分支切换与 Fetch/Pull/Push 移入标题栏（`TestTheTitleBarKeepsTheBranchSwitchAndRemoteActions`），暂存全部／取消暂存全部移入文件栏头部，补丁标签页重新带回文件头（暂存／取消暂存、冲突解决、文件菜单）。⌘1–⌘4 重排为历史/分支/Stash/工作树；Space、⇧↵、Enter、⌘C、Esc、⇧↑↓ 在历史页的工作树行生效；命令行选择可在历史页的工作树补丁里用方向键走动（`TestTheWorkingTreeDiffLinesWalkOnTheHistoryPage`）。「撤销上次提交…」随最后提交卡片消失，改放到 HEAD 提交的右键菜单（`TestOnlyTheHeadCommitOffersUndo`）。
- [ ] log graph 竖直线的间距已收紧（14:57：`graphLaneGap` 32→20，仍保证最大节点 18.2 不与邻道相碰；窄窗口按列表宽度的 26% 继续压缩）。若还觉得宽，请回复目标间距或能否接受线条弯曲更陡。
- [x] 「文件列」「摘要」独立区域。**D2 已实现并验收（2026-11-16 对账）**：全高文件列 + 右侧「提交 / 摘要与多文件」标签；首标签含信息与全部补丁，文件标签聚焦单文件。T1–T6、`TestFileTabOnlyDrawsItsOwnPatch` 及完整详情标签原生回归已有证据；最新十组通过见 `build/lobe-submodule-dirty-detail.log`。此前上下分区方案不再执行。详见 R06、T1–T6；本条关闭重复实现待办，不关闭 R09 / R11 / R16 的保留门槛。
- [x] 左侧「分支」「远程」等展开标题整行可点击（14:57 完成）。分组标题改为一个 `RoleDisclosure` 按钮，带 `Expanded` 状态；整行（箭头 + 名称 + 计数）点击都折叠/展开，箭头只是装饰。标题不再跳转页面：分支/标签列表行用于筛选历史，Stash/工作树行本身会进入对应页面，页面仍可经 ⌘3–⌘5 到达。测试：`TestTheWholeGroupHeadingFoldsItsList`（标题中部与右侧都能折叠、读屏 Expanded 跟随、页面不变）、`TestWithoutARepositoryTheSidebarOpensNoPage` 同步更新；把整行点击改回只认箭头后转红。
- [ ] 右键提交菜单参考 Sublime Merge（图片见 `/Users/adam/Downloads/已粘贴 2026-11-15 下午2.38.39.png`）。14:57 已做：合并/变基/重置文案点名两端，例如「将 origin/feat/x 合并到 main…」「将 main 变基到 origin/feat/x…」「将 main 重置到此提交…」；来源优先取远端分支，其次本地分支，再退回短 SHA；分离 HEAD 时保留通用文案。测试：`TestTheCommitMenuNamesTheBranchItActsOn`。**待定**：Sublime Merge 还有「隐藏分支 / 只显示该分支 / 恢复隐藏分支」一组（需要先有「隐藏引用」概念）与 Create/Edit 子菜单（现有标签/分支/复制项已是等价入口）；菜单分组顺序是否也照搬（分支操作在最上）等待确认。

## 一、待你选择

### 界面与布局

- [x] **文件栏与 diff 栏拖动分界**。已按后续确认的三处分栏计划实现，使用 `splitDrag` 并持久化 `filesSplit`；默认 300 DIP，窄窗按 180 / 300 的文件 / 补丁最小宽度约束让步。2026-11-16 最新原生验证通过三处分栏坐标拖动及重启恢复；另双击 Files 分隔线清除偏好、恢复默认宽度，其他分栏保持，第二次重启保留重置结果。Git HEAD / Index / 文件字节不变；证据 `scripts/native-workbench-layout-smoke.py`、`build/workbench-layout-smoke/result.json`。此条与后续已实现计划重复，按实际验收关闭。
- [ ] **工作区行如何自报身份**。现状：列表首行标题是「3 个改动」，副标题「提交改动」是祈使句，没有「工作区」字样或图标，与提交行同构；点它打开的是提交框。方案：A 保持；B 标题改「工作区」，改动数移到右侧小 Tag；C 保留标题，副标题改「3 个改动 · 未提交」。验收：不看底色也能认出首行，读屏名含「工作区」。
- [x] **工作区行要不要节点标记**。2026-11-16 对账：已由后续 N02 反馈确认并实现工作区标记 / 通向实际 HEAD 的虚线，关闭线图时隐藏。`TestWorkingChangesUseDashedHeadConnection`、`TestWorkingChangesPaintDashedLane`、`TestHiddenHeadDoesNotConnectToFirstVisibleCommit` 及后续显示矩阵已有验证，最新全量见 `build/lobe-submodule-dirty-full-check.log`。2026-11-16 安装体验反馈后进一步修正：当 HEAD 为第一条提交时，工作区方块与 HEAD 同列，使用竖直虚线；只有 HEAD 位于更下方时才保留独立侧边通道。新增几何 / 绘制回归 RED→GREEN，完整 Go / race / vet / 两种构建通过，原生截图 `build/lobe-worktree-vertical-ak4walif/vertical-head.png` 已核对。修正版已更新至 Applications 测试包 version 3，需退出重开。本条不决定 R07 / D3 的工作区名称与启动选择。**原问题背景（已由 N02 取代）**：列表首行的工作区只有两行文字（改动数 +「提交改动」），没有节点或连线。方案：A 保持；B 画一个虚线空心节点，只在线图打开时出现。验收（若 B）：该行节点不与任何车道连线相交，关闭线图时不画。
- [x] **文件栏的键盘操作**。2026-11-16 对账：文件列表已独立处理方向键 / Enter / Space，文件版本焦点与提交列表分离；`TestDetailTabsKeepKeyboardOwnership`、`TestGroupedFileSpaceUsesFocusedVersion`、`TestHistoryFileListHasOneTabStop` 及原生单 Tab / Shift+Tab 记忆入口、精确暂存均通过。最新全量 / 原生十组见 `build/lobe-submodule-dirty-full-check.log` / `build/lobe-submodule-dirty-detail.log`。**原问题背景（已解决，15:19）**：`openSelectedFile` / `toggleSelectedStage` 已改为认历史页的工作树行，Enter 打开、Space/⇧↵ 暂存或取消暂存已经可用；但文件栏本身还不是可聚焦列表，点选文件后按 ↑/↓ 仍移动提交选择并换掉整个文件列表。方案：A 把文件栏做成可聚焦列表（焦点框与提交列表一致），聚焦时上下键走文件行；B 明确文档化为鼠标操作并去掉行内快捷键。验收（若 A）：Tab 进入文件栏后上下键换文件、Enter 打开、Space 暂存或取消暂存，且不影响提交列表的方向键。
- [ ] **Changes / Tree 视图切换**。现状：文件栏只有平铺列表，Sublime Merge 有目录树切换；300 宽下深路径信息量低。方案：A 暂不做（文件栏还会随详情变窄）；B 给文件栏加平铺/树切换（需要按目录建行、折叠状态与键盘顺序）。验收（若 B）：切换后同一文件的选中状态保持，树模式下列表可折叠。
- [x] **首标签与文件高亮一致性（方案已定，纳入 T2）**。2026-11-16 对账：T2 已完成，首标签不冒充单文件选择，文件高亮跟随活动文件标签；列表键盘游标另存。证据见 T2 的回归、完整详情标签原生标签 / 历史 / 双版本组，最新十组日志 `build/lobe-submodule-dirty-detail.log`。**原问题背景（已解决）**：摘要在前时预读的第一文件仍被画成选中且读屏报 checked。新规则：首次进入目标显示提交 / 摘要标签，不把预读文件宣告为当前单文件；点击文件后才打开 / 激活对应标签，文件行与活动文件标签同步。列表键盘游标与正在显示的补丁身份分开；历史与工作区一起验证。
- [x] **旧设置导致空详情（纳入 T5）**。2026-11-16 对账：T5 已完成，`TestLegacyDetailSettingsMapToTabbedLayout` 核对旧字段映射、全部历史内容隐藏时的信息回退、隐藏后清理行选区与重新开启文件标签；完整详情标签原生旧偏好显隐组通过，最新日志 `build/lobe-submodule-dirty-detail.log`。已实现规则：保留不可关闭的提交 / 摘要首标签；历史提交信息与补丁都被隐藏时安全回退为显示信息块，避免空标签条。用户明确隐藏整个详情区时仍按现有行为收起，可从原入口恢复；不新增自动收起设置。
- [ ] **Stash 详情是否加「隐藏详情」**。现状：历史页可用页头按钮收起整个详情区并持久化（`hideDetails`）；Stash 只能拖分割线，窄窗口下差异挤占列表（工作区页已在第 266 轮并入历史页，问题只剩 Stash）。方案：A 照历史页加可见按钮 + 快捷键并持久化；B 只加快捷键；C 不做。验收（任意方案）：隐藏后列表占满宽度且滚动位置不变，重启后状态保留，恢复后差异的滚动位置也在。
- [ ] **分支页是否加「隐藏详情」**。现状：分支页只能把详情拖到最小 240px。方案：A 加按钮 + 快捷键 + 持久化；B 只加快捷键；C 不做。验收（若 A）：隐藏后列表占满宽度，重启后保留，恢复后详情滚动位置不丢。
- [x] **工作区页合并**（15:19 完成）。独立工作区页删除，历史页首行承载工作树；暂存操作、分支切换与远端动作全部有归宿，侧栏、⌘1–⌘4、菜单与文档同步。详见「用户提出需求」第一条。
- [ ] **打包版是否补设置面板**。现状：「设置」窗口与 ⌘, 只存在于 Lobe 预览版（`internal/lobedesk`）；打包版 `.app` 用的 `internal/desktop` 没有设置页。方案：A 保持（设置只做在 Lobe 版）；B 给打包版补同规格设置窗口（需先决定两套 UI 的取舍，工作量大）。验收（若 B）：`.app` 里 ⌘, 打开设置窗口，主题/字号/线图等选项与 Lobe 版一致。
- [ ] **标题栏的分支切换器仍是普通长菜单**（产品审查 2026-11-15 提出）。位置：`workspace.go:113-151`、标题栏 `view.go:102-108`、可复用的搜索弹层 `history_ref.go:252-360`；相关既有条目见「提交菜单的长度与分组」与「标题栏拥挤」。现状：`overlay.DropdownMenu` 直接列本地分支，无搜索、不滚动、只有本地分支；触发文本 `SingleLine()` 不设上限，长名会挤压 Fetch/Pull/Push；读取失败与「没有分支」都显示成不可点击的文字。方案：A 复用引用选择器的搜索弹层（本地/远端分组、可过滤、可滚动，远端走「检出并跟踪」确认）；B 保留菜单，只加宽度上限与远端分支分组（仍受菜单不滚动限制）；C 做命令面板式切换（大）。验收（若 A）：30 个分支时 640 高窗口底部可达、输入即过滤、长名有 tooltip、读取失败可重试。
- [ ] **启动时没有上次仓库与最近列表**（产品审查 2026-11-15 提出）。位置：`app.go:415-427`（只处理命令行路径）、`view.go:635-655`（空页只有「选择文件夹」）、`settings.go:19-45`（已持久化 20 个仓库根的折叠状态，但没有最近仓库）。现状：不带路径启动停在空页，侧栏「打开 Git 仓库」只在已打开仓库后出现；多仓库用户每次都要重新找路径。方案：A 启动自动重开上次仓库（如 Sublime Merge）；B 只列最近 5–10 个仓库、不自动打开（如 GitHub Desktop）；C 两者都做，自动重开可在设置里关掉。验收（任意方案）：不存在或已删除的最近路径自动剔除；空页与侧栏共用同一列表；不写入仓库内容。
- [ ] **宽窗口下提交列表占宽、补丁几乎不增长**（设计复查 2026-11-15 提出）。位置：`history.go`（`historySplitDefault = 0.45`）。现状：1320 宽时列表约 508px、补丁约 321px；980 宽补丁约 300px。补丁是阅读主区，但窗口变宽时几乎不跟着变宽。方案：A 保持固定比例；B 给列表设像素上限（例如 ≤420px）或按像素推导分界，剩余全部给详情。验收（若 B）：1320/1600 宽下补丁随窗口增长而变宽，列表不低于最小宽；980 宽布局与现在一致。

### 键盘焦点与对话框

- [ ] **Tab 停靠点模型**（差异行、列表行、设置页共用一个问题）。现状：差异区每行行号是一个 Tab 停靠点（500 行补丁就有 500 个）；列表每行也都是停靠点；设置页每个分段控件各占一个停靠点，读屏听不出分组，且命令面板触发按钮没有名称。方案：A 保持（菜单键/Shift+F10 已验证可用，差异行没有整行暂存快捷键）；B 差异列与列表各用 `FocusGroup(ui.Vertical)`，整列一个停靠点、方向键在行间移动；C 行号 `TabStop(false)`，差异区整体聚焦、菜单键作用于当前选中行，并给选中行补暂存快捷键。设置页的分组标题与命令按钮名称可以直接补，不必等 A/B/C。验收（若 B/C）：Tab 次数明显减少，方向键与现有行选择不冲突，读屏能听出分组。
- [x] **待定对话框的跨窗口归属**。2026-11-16 对账：确认按发起窗口 / 仓库根持有，同窗口后来请求不覆盖原对象，主窗口与设置窗口可分别保留确认；`confirm_scope_test.go`、`dialog_conflict_test.go` 及 R04 已记录的双窗口请求 / 取消原生验证通过，最新全量见 `build/lobe-submodule-dirty-full-check.log`。关闭本条状态归属问题；真实 OS 焦点切换与其他对象竞态继续由 R04 跟踪。**原问题背景（已解决）**：对话框状态在主窗口与设置窗口之间共享，切换窗口后可能画在另一个窗口；`askConfirm` 可能覆盖已打开的确认。方案：A 对话框状态按窗口持有；B 同一时间只允许一个对话框并按窗口归属绘制。验收：两窗口各自打开确认时互不覆盖。

### 历史列表与线图

- [ ] **远程分支胶囊的省略规则**。现状：固定 20 字符中省略（详情 40），`origin/remove-donate-links` 显示成 `origin/r…onate-links`，远端名占掉一半宽度。方案：A 保持；B 省略时优先保留分支名尾部（`origin/…donate-links`）；C 只有一个远端时省掉 `origin/` 前缀，多远端才保留（需要判断远端数量）。验收：三种引用的常见长名在 1320 与 980 宽下都不出现两个省略号；悬停仍给完整名字。
- [ ] **引用胶囊的显示上限**。现状：一行最多 3 枚（`rowRefs`），其余进 `+N`；窗口更宽时其实能放下 4 枚。方案：A 固定 3；B 按可用宽度动态放（需要文本测量，成本同远程胶囊 B 方案）。验收（若 B）：1320 宽能放 4 枚时不出 `+N`，窄窗口回落到 3 枚。
- [ ] **引用胶囊的排序一致性**。现状：行上「按上下文挑、按 Git 顺序摆」，≤3 个时保持 Git 顺序；详情面板是「行上三枚 + 折叠项」，被提升的引用排在折叠项之前，因此详情不完全是 Git 顺序。方案：A 保持；B 两处都严格按 Git 顺序（详情单独排序）；C 两处都按上下文优先级（把 `rowRefs` 改回按优先级排序，并改 `TestRowRefsKeepsGitOrderWhileChoosingByContext`）。验收：同一提交在行上、详情、引用选择器里的顺序有明确文档与测试。
- [ ] **胶囊宽度与大字号**。现状：历史行上限 150px、详情 320px，随字号比例放大；大字号窄窗口下提交标题更早被截断（先截标题是既有取舍）；紧凑行 + 大字号 + 同一提交两个引用时，标题只剩十几个字符。方案：A 保持；B 窄窗口下收紧胶囊上限；C 紧凑行只显示一枚胶囊。验收：980 宽 + 1.15 字号下标题至少保留 12 个字符或按选定规则截断。
- [ ] **详情里的作者列宽**。现状：作者名过长时在日期前中略，悬停看全名。方案：A 保持；B 把标签列从 54 收到 44（「作者/Author」都放得下），多让出 10px。验收：980 宽下常见英文名（≤20 字符）不被中略。
- [ ] **元数据行的可见性**。现状：长正文提交下，作者/时间/父提交要滚动摘要标签才能看到。方案：A 保持（分块滚动）；B 把元数据移出滚动区固定显示。验收（若 B）：信息块滚动时元数据不移动，1320/980 两档布局都不挤。
- [ ] **重走历史期间详情显示旧提交**。现状：列表已是加载占位，右侧详情仍显示上一个选中提交。方案：A 保持（新引用仍包含该提交时不会闪）；B 详情也换占位。验收：重走期间列表与详情状态一致，或明确显示「正在加载」。
- [ ] **重走占位的闪现**。现状：占位没有延迟阈值，极快仓库上可能不足一帧。方案：A 保持；B 显示前等 150ms。验收：慢仓库能看到占位，快仓库不闪。
- [ ] **提交菜单的长度与分组**。现状：13 项加每个分支一项，常为 19 行；菜单面板不滚动，640 高窗口下底部「重置」被裁掉，键盘 End/↑↓ 可到、滚轮无效。方案：A 保持（靠键盘可达，给 MyGo 报 issue 等上游）；B 自建可滚动菜单（复制 MyGo 行样式，有主题漂移风险）；C 把「复制」「筛选」收成子菜单或缩短标签（组件不支持嵌套，需自建）。另外**菜单里的远端分支**：现状只列本地分支，`origin/x` 仍要去分支页「检出并跟踪」；若要加，先定分组规则（按远端分组或收进「检出远端分支…」子菜单）。验收：640 高窗口最常用的检出/复制/重置在首屏可达；多远端不把菜单撑长。
- [ ] **历史列表的签名徽章**。现状：签名状态只在提交详情显示，列表行没有标记（逐条验签会拖慢长列表）。方案：A 保持；B 只给已加载页里少数已签名提交做标记并缓存结果。验收（若 B）：打开 200 条历史不增加可感知的加载时间。
- [ ] **设计参考里的五项视觉增强**。① HEAD 的空心环节点（现状与普通提交同为实心点，合并提交是空心方块）；② 折叠历史的计数节点（现状按页加载，没有折叠）；③ 作者头像（现状纯文字，`display.Avatar` 已在侧栏用过）；④ 行悬停卡片（现状无卡片，详情在右侧）；⑤ 提交数徽章（现状页头是「N 条提交」纯文字）。每项都是 A 保持现状 / B 按参考图实现；B 的代价分别是：① 环内底色要跟随行地面；② 需要 `git rev-list --count` 或分页返回剩余数；③ 作者列 110→130 且不增加网络请求；④ 需要判断标题是否被截断（MyGo 无测量 API）；⑤ 中英文与字号三档下不压标题。
- [ ] **类型变更是否合并两个同名文件头**。现状：同一路径从 gitlink 换成普通文件（或反向）时，提交差异按 Git 的两段 patch 各画一个文件头，名字相同；Blame 已按最终形态判断。方案：A 保持（与 Git 输出一致）；B 合并成一个文件头。验收（若 B）：两段内容仍完整可见，暂存/撤销语义不变。
- [ ] **提交差异的折叠状态是否跨提交保留**。现状：折叠按文件路径记录且不随提交切换清空：在提交 A 折叠了 `pkg/gui/gui.go`，选提交 B 时同名文件仍是折叠的。方案：A 保留现状（连续浏览同一批文件顺手，但可能让人以为提交 B 缺文件）；B 每次换提交全部展开（在 `resetCommitDiff` 里清空 `commitCollapsed`）。验收（若 B）：换提交后所有文件默认展开；同一提交内折叠/展开仍恢复选区。
- [ ] **历史搜索无结果时的详情**。现状：列表显示「没有匹配的提交」、副标题 0 条，右侧仍显示上一个提交。方案：A 保持（清空搜索即恢复）；B 详情换「筛选结果为空」占位并保留原选中；C 清空选中。验收（若 B）：无匹配时详情不再显示旧提交，清空搜索后恢复原选中，而不是回到第一条。
- [ ] **分支页首次打开是否自动选中当前分支**。现状：分支页打开且尚无选中时，右侧仍显示「从左侧选择…」，即使列表里已有当前分支。方案：A 页面首次进入时选中当前分支（无当前分支则第一行），之后按现有规则保留；B 保持现状（必须先点一行）。验收（若 A）：⌘3 进入即看到当前分支详情；↑/↓、搜索过滤、Tab 行为不变。
- [ ] **当前分支详情的「切换」按钮是否隐藏**。现状：所有禁用按钮都有原因提示，但「切换」对当前分支永远禁用。方案：A 隐藏（合并/变基/删除保留提示）；B 六项全保留。验收（若 A）：当前分支详情少一枚死按钮，其余禁用项仍有说明。
- [ ] **复制名称与「在历史中查看」的入口**。现状：三类详情都能复制名称，「在历史中查看」只有标签的 Enter/双击有。方案：A 详情头部名字旁加两个图标（复制、历史），footer 只留状态变更动作（顺带缓解标签 4 按钮的 3+1 换行）；B 只进右键菜单/快捷键；C 保持现状。验收（若 A）：1320/980 下 footer 不超过两行，图标带 tooltip。
- [ ] **blame 的代码文本选择**。现状：整行点击跳转到该提交，代码文本无法拖选复制；右键菜单可复制整行。方案：A 保持（行点击优先）；B 把跳转限制在 SHA / 作者 / 时间列，代码列可拖选，代价是点代码不再跳转。验收（若 B）：代码列拖选与 Cmd+C 可用；点 SHA/作者/时间仍跳转；整行高亮与悬停反馈不变。
- [ ] **子模块改动的差异画法**。现状：`git diff` 使用 `--submodule=short`，子模块改动显示为普通补丁（`160000` 模式 + `Subproject commit <sha>`），与普通文件同一套渲染，Blame 也能识别并置灰。方案：A 保持（与 Git 默认输出一致）；B 改回 `--submodule=log` 并自己解析 `Submodule sub a..b:` 段，需要额外解析器，Blame 判断也要另找信号。验收（若 B）：子模块改动的文件头、路径、可展开行与前后普通文件一致，且 Blame 仍正确置灰。
- [ ] **子模块忽略配置下 Status 与 Diff 不一致**。现状：`status.go` 强制 `--ignore-submodules=none`，`diff.go` 未加；设置 `submodule.<name>.ignore=all` 后，文件行显示 `M` 但 Diff 为空。方案：A `Diff`/`lines.go` 同样加 `--ignore-submodules=none`，保持「状态可见就有差异」；B Status 尊重用户配置，行也不显示改动。验收：两种配置下左右两侧一致。
- [ ] **无法 blame 的文件是否给退路**。现状：提交里新增的文件、子模块指针、删除的文件，Blame 菜单项置灰并写明原因。方案：A 保持；B 允许追溯父提交或删除前最后版本的该文件；C 菜单里直接加「追溯上一版本」项。验收（若 B/C）：菜单文案说明追溯的是哪个版本，失败时说清原因。

### 文件行与差异

- [ ] **大提交差异超过 4 MiB 时无法显示（2026-11-16 测试体验反馈，留待讨论）**。触发：历史页查看首次提交，Files 显示 1,079 个文件，摘要中的差异区域报「Git 输出超过 4 MiB 显示上限；操作可能已经完成，请刷新状态确认」，文件列表与提交说明仍可显示。当前实现：`internal/lobedesk/app.go` 一次调用 `ShowAll` 读取整次提交的全部补丁，`internal/gitrepo/repository.go` 对单次 Git 标准输出设 `4 << 20` 字节上限，超过后返回错误，`internal/gitrepo/diff.go` 不再解析该次补丁；历史文件标签也依赖这份整体结果。此处为只读查看，不表示提交失败或仓库损坏，同样内容下 Retry 不能消除固定上限。待讨论：大提交采用按文件 / 分批 / 流式加载，还是保留整体限制并提供明确的局部查看入口；单文件也超限时如何处理；内存与响应预算如何确定。文案需区分只读差异超限与写操作结果，避免在查看历史时提示「操作可能已经完成」。验收方向（待方案确认）：用总 diff 超过 4 MiB 的提交复现，文件列表与提交说明仍可用，按选定方案查看差异并明确未加载范围，不将截断补丁当作完整内容；小提交与文件标签不回归。本条仅记录，未选定方案、未修改实现或上调限制。

- [ ] **文件行的路径写法**。现状：文件行显示「文件名 + 文件夹」（文件夹靠右）；Git 的写法是「文件夹/文件名」。方案：A 保持；B 改成按完整路径显示并按宽度中略（目前没有文本测量，只能按字符数近似）。验收（若 B）：长路径在 980 宽下中略到保留文件名，悬停看全路径。
- [ ] **文件夹列的对齐**。现状：文件行里的文件夹靠右直接跟行尾，文件多且长短不一时右侧参差。方案：A 保持；B 固定文件夹列宽（例如 40%），会牺牲长路径显示。验收：文件列表中文件夹左缘对齐，文件名不被挤掉。
- [ ] **详情里的复制按钮常显**。现状：摘要标签里的复制按钮常显，内容较长时每屏多一个小图标；Sublime Merge 是悬停才出。方案：A 保持；B 改为悬停显示（键盘聚焦时也显示）。验收：悬停/聚焦时复制按钮出现，不改变 SHA 行布局。
- [ ] **差异代码区的右键归属**。现状：在代码文字上右键弹出系统「复制 / 全选」菜单并截断应用行菜单，且不选中该行；行号手柄与行尾空白处右键则正常。方案：A 保持（手柄/行尾为整行入口，代码区保留文本菜单）；B 让代码区右键也选中并显示应用菜单，代价是文本右键菜单消失；C 只在代码区未选中文本时让应用菜单优先。验收（若 B/C）：右键后该行被选中、菜单完整；拖选与 ⌘C 不受影响。
- [ ] **行尾暂存图标抢 Enter**。现状：点过工作区文件行的「暂存/取消暂存」图标后焦点在图标上，Enter 再次触发暂存，而不是页面级的「打开文件」；只影响图标区域，行本身已正确。方案：A 保持（Enter 重复聚焦控件的动作是按钮通用行为）；B 让图标不抢 Enter（MyGo 没有不可聚焦按钮的开关，需要改结构或等上游）。验收（若 B）：点图标后按 Enter 打开文件，点图标本身仍只暂存一次。
- [ ] **冲突文件的 combined diff 没有被解析**（产品审查 2026-11-15 提出）。位置：`internal/gitrepo/diff.go:14`（`hunkRE` 只认 `@@`）、`:25-50`（未识别的行都记为 meta）、`internal/lobedesk/workspace.go:424-462`（只按 Kind 上色）、`history_panes.go:272-290`（冲突详情头只给「标记为解决」）、`menu.go:306-323,912-915`（Ours/Theirs 在菜单里；「标记为解决」直接 `git add`，不检查残留标记）。现状：`diff --cc` 的 `@@@ -a -b +c @@@` 与双列前缀一条 hunk 都建不出来，冲突文件整段显示为灰色 meta，没有行号与增删底色；「标记为解决」在冲突标记仍在时也会暂存，容易把标记提交进去。方案：A 先修解析，用行号与颜色区分两侧（中）；B 再把冲突块渲染成双栏 Ours/Theirs 预览（大）；C 维持外部编辑器解决，只加头部三个动作并在残留标记时确认（小）。建议先 C 再 A。验收：冲突文件按块显示两侧行号与颜色；残留标记时暂存前有确认。

### 设置与行为

- [ ] **打包版是否加「跟随系统」主题模式**。现状：打包版的明暗由侧栏手动开关持久化，Dock 图标跟随这个选择；macOS 系统外观变化不会改变 Dock，而 Lobe 预览版在「跟随系统」主题下会。方案：A 保持（Dock 跟随 App 主题）；B 给打包版增加「跟随系统 / 浅色 / 深色」三选一，默认跟随系统；C 只让 Dock 跟随系统、界面仍手动（会出现界面与 Dock 不一致，不推荐）。验收（若 B）：切换系统外观时界面与 Dock 同步变化，手动选择后不再跟随。
- [ ] **自动 fetch 的开关语义**。现状：一个「启用自动 fetch」同时管「打开仓库时 fetch 一次」与「按间隔定时 fetch」。方案：A 保持；B 拆成两个独立开关（打开时 / 定时间隔）。验收（若 B）：两个开关互不影响，关闭定时后打开仓库仍可选是否 fetch。
- [ ] **手动 fetch 与自动计时**。现状：手动 fetch 会重置定时器，避免刚手动获取又被定时器重复。方案：A 保持；B 手动不影响计时。验收：切换间隔、手动 fetch、自动触发三者的下一次时间可预期。
- [ ] **合并/变基是否需要二次确认**。现状：分支与远端分支的「合并/变基」会先弹确认（与提交菜单一致）；部分读者觉得多一次点击。方案：A 保持；B 改回直接执行（危险操作仍确认）。验收：确认框只出现在选定的操作上。
- [ ] **A/M 文件是否同时出现在两块**。现状：索引与工作区都有改动的文件会同时出现在「已暂存」和「未暂存」，这是 Git GUI 通行做法（选中状态靠区块位置区分）。方案：A 保持；B 只在未暂存显示并加标记。验收（若 B）：两块计数与实际改动数一致，不重复。
- [x] **设置里的详情开关叫什么（纳入 T5，以 T5 的命名与兼容映射为准）**。2026-11-16 对账：T5 已完成实际区域命名、中英文帮助与保留一个内容块的理由；`TestLegacyDetailSettingsMapToTabbedLayout`、`TestDetailsKeepsOneBlockWithAReason`、原生旧偏好显隐组和设置窗口八组矩阵已有验证。最新全量 / 原生见 `build/lobe-submodule-dirty-full-check.log` / `build/lobe-submodule-dirty-detail.log`。**旧问题背景（2026-11-15）**：位置：`settings_page.go:149-152`、`i18n.go:232-236`（「提交信息 / 文件改动 / 差异」与「勾选要显示的分栏」），界面上对应的是「摘要」标签、文件栏和以文件名命名的补丁标签（`i18n.go:294-297`、`history_panes.go:68-85`）。现状：读设置的人要在「差异」和「文件」之间自行映射，「分栏」也与实际结构不符（其中两项是同一栏里的标签页），keep-one 提示只提两项。方案：A 改用界面名（摘要 / 文件栏 / 补丁标签；英文 Summary / Files / Diff），help 与 keep-one 提示同步；B 保留描述性名称，在括号里指向界面名。验收：三项设置与页面实际区域一一对应，中英文都能对上，keep-one 提示覆盖三项。
- [ ] **作者过滤是否要精确匹配**。现状：`Ada` 会同时命中 `Ada Lovelace`（子串匹配）。方案：A 保持；B 给搜索框加引号语法（`"Ada Lovelace"`）或独立作者过滤状态。验收（若 B）：精确匹配与子串匹配可区分，日期/引用搜索不受影响。
- [ ] **取消后的体验**。现状：取消没有「重试」动作，要手动再点一次；操作开始后的前 200ms 没有取消按钮，卡死在这段时间只能等延迟结束。方案：A 保持；B 在失败/取消提示里加「重试」（触发同一个操作，不绕过确认），并让超过 1 秒仍未结束的操作提前显示取消按钮。验收（若 B）：慢操作 1 秒内可取消，快操作不闪；重试不绕过确认。

### 键位与交互

- [ ] **快捷键补齐范围**（对照 `docs/shortcuts.md` 与 Sublime Merge）。现状：常用键位已实现，未实现的按风险分三档：A 低风险（⌘O 打开仓库、⌘0 跳到 HEAD、⌘⇧F 引用筛选、⌘⇧A/⌘⇧R 暂存全部/取消暂存全部）；B 中等（⌘←/⌘→ 导航栈、⌥↑ 父提交）；C 需产品决定（⌘S 给 Stash 会占用「保存」直觉；⌘P 命令面板工作量大但对键盘可达性提升最大）。请回复做哪几档，或指定键位。做 B/C 前先抽「快捷键注册表」（已完成，第 264 轮；新键位直接加进 `shortcutTable()` 即同时出现在设置页卡片里）。
- [x] **看不见的行选择会让 Space/Enter 静默失效（纳入 R02 / T4，按实际可见补丁判断）**。2026-11-16 对账：R02 / T4 已完成可见补丁键盘归属、隐藏详情清理选择，标签 / 文件 / 历史焦点互不串用；`TestHiddenCommitDetailsReleaseTheLineKeys`、`TestDetailTabsKeepKeyboardOwnership` 与 T4 所列工作区守卫回归通过，最新全量和完整详情标签原生十组见 `build/lobe-submodule-dirty-full-check.log` / `build/lobe-submodule-dirty-detail.log`。**旧问题背景（2026-11-15）**：位置：`history_panes.go:171-207`（切换标签只改 `detailTab`）、`history.go:251-261`（隐藏详情只翻 bool）、`keyboard.go:37-45,56-63`（`canOpenSelectedFile` / `canToggleSelectedStage` 只看 `len(lineSel) > 0`）。现状：在工作树补丁里选了几行，再点「摘要」或隐藏详情，选中仍在但看不见；此时 Space 暂存、Enter 打开文件都返回 false 且没有任何提示，按 Esc 或重新点一行才恢复；回到文件标签还能看到旧的高亮。`docs/selection-model.md:30` 写的是切换摘要或隐藏详情时清空行选择，代码没有照做。方案：A 按文档清空（切走文件标签与隐藏详情时清 `lineSel`，提交补丁的 `commitSel` 同样处理）；B 保留选择但让守卫看可见性（`worktreeDiffShown() && len(lineSel) > 0`），文档改成「隐藏时保留选择」。验收：任意方案下文档与行为一致，Space/Enter 不会在无提示的情况下失效。
- [ ] **分支搜索框 Enter 的行为**。2026-11-16 源码核对：搜索框 `OnSearch` 先 `syncBranchSelection()`，再激活当前可见引用；原选择被筛掉时会移至首个可见结果，无匹配时清空选择，激活入口也复查可见性。因此旧记录「可能沿用已被筛掉的分支」不再适用于当前实现。本地分支会切换、远端分支打开检出提示、标签打开其提交；搜索框内 Enter 也执行这些动作，已补充 `docs/shortcuts.md`。本轮只修正文档，未执行真实分支切换验收。行为方案仍待确认：A 保持；B Enter 只提交筛选并把焦点移到列表；C 搜索触发切换时先确认。
- [ ] **⌘F 后输入会追加到已有搜索词**。现状：MyGo `field.SearchBar` 的 `ShortKey` 只调用 `Focus()`，模块自己的搜索栏聚焦后 `selectAll()` 但项目侧拿不到接口；已有 `rebase` 时再输入得到 `rebaseX`。方案：A 给 MyGo 提 issue（`pkg/field/searchbar.go` 在 `focusNow` 时选中全部），等上游；B 项目内重写搜索栏（复制样式，有漂移风险）。验收（若 A 落地）：分支/历史/检出选择/引用选择器四个搜索框行为一致。
- [ ] **搜索框 Esc 的行为**。2026-11-16 UI Tester 四类搜索复查：先实际输入 `x`，确认查询由 `needle` 变为 `needlex` 且输入框获得焦点，再连续按两次 Esc。历史 / 分支搜索均保留查询及焦点；检出分支选择器第一次 Esc 关闭并清空查询；引用选择器第一次 Esc 关闭但保留查询。当前依赖的 `SearchBar` 没有 Esc 清空处理，两个选择器还受弹层关闭逻辑影响。证据：`build/lobe-search-escape-typing-observation.log`；使用 build 内临时 Go overlay，无产品代码 / 依赖修改。原生隔离仓库复查也已完成输入与连续两次 Esc 的四组观察，HEAD / 当前分支 / refs / Index / 工作区字节保持不变（`build/lobe-search-escape-ybnmusvu/results.json`，关闭后截图已核对）。历史 / 分支查询保持；检出弹窗关闭；引用弹窗第一次 Esc 后仍存在、第二次后消失。后续限定 3 秒的快照轮询仍观察到第一次 Esc 未关闭；原生重开时查询为空，与 `history_ref.go` 每次打开清空查询的实现一致，关闭状态中暂存的 query 不代表下次会恢复。重开后单次 Esc 的关闭断言也失败（`build/lobe-search-escape-native-reopen.log`），该次后续检出重开未执行。随后在独立依赖副本中加入按键分发日志（不改模块缓存 / 项目依赖 / 普通二进制），确认首个 Esc 被 `needle` tooltip（ID `12560610992988495487`）接收，第二个才由引用 popup（ID `9700942433234273728`）接收；快照与日志 ID 一致，原因是更上层提示优先关闭，并非引用弹层未收到按键后仍不处理。证据：`build/lobe-search-escape-t7habby4/app.log`、`refs-before.snapshot.txt`、`results.json`。该日志版四组观察、两个选择器重开查询为空及 Git 状态保持均通过。额外“移开指针 / 点击输入框后无 tooltip”的普通包检查未通过此前提，不能作为无提示状态的单次 Esc 验收；框架提示也能由键盘焦点触发，不应统称为悬停。当前行为已记录到 `docs/shortcuts.md`，未验证实际 OS 激活焦点。旧记录「SearchBar 自带 Esc 清空」不能作为当前行为依据。方案仍待确认：A 统一为 Esc 清空查询并保留焦点，再按一次离开搜索；B 保持框架行为。
- [ ] **深色主题的 tooltip 是浅色底**。现状：MyGo `ui.Tooltip` 用 `t.inverse()` 取色，深色主题下是白底深字，与菜单/卡片明显不同。方案：A 保持（跟随系统提示）；B 报给 MyGo；C 项目自绘（不建议）。验收（若 B 落地）：深色下提示与卡片同色系，浅色下仍可读。
- [x] **错误 toast 遮挡详情 footer**。2026-11-16：主窗口提示按实际 footer 高度让出按钮区；长错误显示当前语言的行动入口，完整原文在可滚动、可复制的详情中保留。980 / 1320 × 普通 / 1.15 字号的真实点击回归 RED→GREEN；全部 Go 检查 / 构建通过，原生 Pull 失败提示保持显示时，分支页坐标点击 Copy branch name 成功且 HEAD / Index 不变，截图已核对。R04 的其他弹层 / 对象状态保护仍待完成。
- [x] **刷新仓库期间没有进度反馈**。2026-11-16 对账：R04 已实现读取超过 200ms 后显示当前语言的加载文字 / 旋转图标、完成退出，读取中禁用 Refresh，不显示写操作取消按钮。`TestSlowRepositoryReadHasVisibleTitleBarProgress`、`TestRepositoryReadProgressWaitsAndKeepsOperationPriority` 回归及最新全量检查通过（`build/lobe-submodule-dirty-full-check.log`）；没有新增刻意延迟读取的原生验收。本条原功能缺失已解决，其他反馈 / 响应边界仍归 R04 / R16。**原问题背景**：⌘R 后旧页面停留、刷新按钮变灰，标题栏无文字；期间 Space/暂存被静默忽略。方案：A 标题栏在 `loading` 时显示现有 `loadingRepo` 文案 + 转圈，但不显示无作用的取消按钮；B 让加载可取消并复用取消按钮；C 保持。注意 `progressText` 非空时标题栏会画取消按钮，直接复用会出现无效按钮。

### 待你澄清或确认

- [x] **「file changed 一列」的最终形态（已确认，纳入 T2）**。2026-11-16 对账：D2 / T2 与后续 N01 / N06 已实现，四列、全高 Files、固定状态分组、对应文件标签及可调列宽均有布局 / 原生证据，最新完整详情标签十组见 `build/lobe-submodule-dirty-detail.log`。已实现规则：第三列为全高的独立变更文件列表；工作区按状态分组、历史按提交列文件，点击后在第四列打开对应文件标签。不再等待选择「每条提交旁显示文件列」等其他形态。
- [ ] **「深色主题下浅色卡片单独算墨色」具体指哪里**。现状：深色模式下 `BgLayout/BgBase/BgContainer` 都来自方案的暗色面，`deepInk` 对暗面直接返回原色，当前界面没有「深色模式里的浅色卡片」。请给页面名或截图（例如提示条、图标底板、差异里的新增/删除块）。
- [ ] **测试遗留的配置文件是否删除**。`~/Library/Application Support/supergit/settings.json`（2026-11-14 由测试写入）与 `preferences.json`（Recent 全是临时目录）看起来都是测试遗留物。删除需要你明确同意；删除前我会先列出内容。

---

## 二、待开发

### 交互与状态修复

- [x] **失败的重载被说成「打开失败」，且没有测试**。位置：`contentCard` 只在 `repo == nil` 时显示 `loadErr`，已打开仓库后重载失败会保持旧列表、没有错误行和重试，toast 却说「无法打开仓库」。做法：拆出结果应用函数，重载失败时在列表上显示错误与「重试」，保留旧列表并标记其已过期；补文案、旧列表标记与重试恢复的测试。（已完成 2026-11-15 12:13：`reloadFailedBanner` + `reloadFailed`/`reloadStaleHint`，toast 区分首次打开、刷新与切换失败；`TestAFailedReloadSaysTheContentsAreStale`、`TestAFirstOpenFailureIsNotCalledARefresh`、`TestReadFailuresNameWhatTheReaderWasDoing`）
- [x] **引用选择器错误态的 Tab 可以穿到背后页面**。现状：Tab 到「重试」后，方向键会移动背后的历史列表。做法：弹层根上固定消费 Up/Down/Home/End，或在 `historyRefOpen` 时屏蔽列表移动；补「错误态 Tab→Down 不改变 `selectedSHA`」测试。（已完成 2026-11-15 12:13：`moveSelection` 在 `historyRefOpen` 时直接返回，方向键与 Home/End/PageUp/PageDown 都走这一入口；`TestTheRefPickerKeepsTheHistoryArrows`）
- [x] **重试按钮在 busy 时静默无效**。做法：`busy || loading` 时禁用并显示「正在重试…」，重复点击不重复启动；补测试。（已完成 2026-11-15 12:13：`loading` 时显示「正在重试…」，仅 `busy` 时保持「重试」但禁用，避免把 fetch/暂存误报成重试；`TestRetryIsDisabledWhileAReadIsRunning`、`TestRetryDoesNotCallAnotherOperationARetry`）
- [x] **分支/标签读取失败分不清是哪一组，且与页头计数矛盾**。现状：两处错误都堆在本地分支上方，标签标题消失，页头仍写「0 个标签」。做法：让 `failedWithRetry` 带标题，标签错误放进标签区，失败时计数显示「—」；补测试。（已完成 2026-11-15 12:13：错误移入各自分组、标签区保留标题；页头把失败组写为「标签读取失败 / local branches unavailable」，比「—」更明确；`TestAFailedTagReadKeepsItsOwnSection`）
- [x] **工作树创建失败会关闭表单并清空输入**。做法：失败时保留表单与错误，成功后再清空；补测试。（已完成 2026-11-15 12:13：`submitWorktreeForm` 在回调收到错误或操作未能启动时恢复表单；`TestAFailedWorktreeCreationKeepsTheTypedValues`。真实 git 失败路径仍缺 e2e）
- [x] **提交补丁读取失败后没有重试入口**（产品复查 2026-11-15 提出）。位置：`history.go:831-833`、`app.go:1029-1030,1124-1125`。现状：文件列表已加载、只有 `ShowAll` 补丁失败时，差异区只显示错误原文；再点同一提交因 `commitFilesLoaded` 直接返回，只能换提交或整页刷新。做法：差异错误行补「重试」，调用 `loadCommitDetail(true)`；补「files 成功、diff 失败时按钮存在，点击后 `commitSeq` 递增」测试。（已完成 2026-11-15 12:35：补丁错误行加 Retry；文件列表本身失败时 diff 栏只说原因、不再重复同名按钮；`TestAFailedCommitPatchOffersRetry`、`TestAFailedFileListDoesNotRepeatItsRetryInThePatch`）
- [x] **切引用失败后仍显示旧计数与旧提交**（产品复查 2026-11-15 提出，12:56 完成）。新增 `failHistoryReload`：失败的 walk 清空 `commits/historyTips/historyMore`、选中、提交详情与文件、递增 `commitSeq`（丢弃在途详情回调）、清行选择；页头副标题改「提交读取失败」，详情回到占位。这里没有沿用「以下为上次内容」：旧行属于被替换的 walk，新引用筛选下展示它们会自相矛盾，所以直接丢弃。开始新 walk 时先清 `historyErr`，错误行不会与「正在重读」占位并存。测试：`TestAFailedRefSwitchDropsTheOldWalk`（清空旧行/选中/在途详情，页头不再计数，详情回占位；回退清理逻辑后转红）。
- [x] **工作树与子模块共用一个 `workspacesErr`，且渲染是 else-if**（产品复查 2026-11-15 提出，12:56 完成）。拆成 `worktreesErr` / `submodulesErr`，两条读取各自赋值；错误与「重试」放进对应分区，空状态只在自己分区未失败时出现；分区计数显示「读取失败」，页头副标题按组写「工作树读取失败 · N 个子模块」（英文 `worktrees unavailable`）。测试：`TestWorktreesPageShowsNoEmptyStateAfterAFailedRead`（工作树失败不再隐藏子模块空状态）、`TestAWorkspaceReadFailureStaysInItsSection`（子模块失败时工作树行与计数仍在，副标题点名失败组）。
- [x] **Stash 读取失败仍显示旧补丁与「0 个 Stash」**（产品复查 2026-11-15 提出）。位置：`stash.go:43,65-74,227-256`。现状：失败后旧补丁与选中仍在，页头计数归零。做法：失败时清空补丁/选中，页头计数显示「读取失败」，补测试。（已完成 2026-11-15 12:35：页头改为「Stash 读取失败」；正常失败路径下 `reconcileSelection` 已清选中与旧补丁，已核对。`TestAFailedStashReadNamesTheFailure`。`Stashes` 返回「部分结果 + error」时的显式清理仍缺，见下方）
- [x] **文件栏与 diff 栏的分界线画反了**（设计复查 2026-11-15 提出）。现状：MyGo 边框顺序为上右下左，文件栏 `(0,0,0,1)` 画在左缘，与 7px 分隔线并成双细线；diff 栏 `(0,1,0,0)` 画在窗口最右，两栏正文之间反而没有线。做法：文件栏改画右边框，diff 栏去掉右边框。（已完成 2026-11-15 12:13：`history_panes.go` 两处 `BorderWidth` 调换）
- [x] **980 最小窗口把 diff 挤到 207px**（设计复查 2026-11-15 提出）。位置：`history.go:23-27`、`splitter.go:36-48`、`history_panes.go:34`。现状：980 扣侧栏 184 后约 796；列表最小 320 + 详情最小 520 + 7 超宽，回落 50/50：列表约 394、文件栏 180、diff 仅约 207，低于文件栏计算里的 `historyDiffPaneMin=300`；窗口到 1031 才恢复。做法：详情最小改 480（=180+300），`details` 改用 `(row-splitterWidth)*(1-share)`；验收 980/1031/1320 三档下 diff ≥300。（已完成 2026-11-15 12:25：列表最小 320→300、详情最小 520→480、文件栏宽度改按剩余宽度计算；`TestThePatchKeepsItsWidthAtTheSmallestWindow` 覆盖三档，旧公式下 980/1031 分别只剩 214.5/240）
- [x] **diff 标签条的活动标签像悬浮胶囊**（设计复查 2026-11-15 提出）。位置：`history_panes.go:137-148,155-172`。现状：38px 灰条里放 26px 白圆角标签，上下各露 6px 灰，底部另有一条 1px 线，标签与内容断开。做法：标签满高或条/标签同高，只留上方圆角，底边线只画在不活动区域；补截图验收。（已完成 2026-11-15 12:25：标签满高 38、只留上方圆角；去掉整条底边线，让活动标签与正文接上。截图复查仍待窄窗口组合）
- [x] **文件标签只有 basename，没有路径线索**（设计复查 2026-11-15 提出）。位置：`history_panes.go:143-145,167-171`。现状：目录被丢弃，`MaxWidth(220)` 截断后无 tooltip；文件栏隐藏时同名文件难辨。做法：标签加完整路径 tooltip，或显示「文件名 + 次级目录」；验收同名文件与长路径在 980 宽下可区分。（已完成 2026-11-15 12:25：标签保留 basename，`Tooltip` 给完整路径；`TestTheFileTabNamesItsFullPath`）
- [x] **侧栏当前分支标记会被截掉**（设计复查 2026-11-15 提出）。位置：`view.go:434-448,387-409`。现状：当前分支靠名字末尾追加「 ·」，长名省略时标记先消失，ref 文本也没有 tooltip。做法：标记改成独立的 `Shrink(0)` 元素（10px 勾/点），名字加 Tooltip；验收 184px 下 40 字符分支名仍见标记，悬停见全名。（已完成 2026-11-15 12:25：当前分支用独立绿色勾标记，名字不再带后缀；短名不弹 tooltip，长名与页头引用名才有完整名提示；读屏描述补「当前」。`TestTheCurrentBranchMarkIsNotPartOfItsName`）
- [x] **设计复查第二批（12:35）**：浅色主题侧栏选中态改用 `selectionBg`（原先与底色相同，悬停反而更像选中）；diff 两个标签补 `RoleTabList/RoleTab` 与 `AccessSelected`；文件标签改为占满剩余宽度并去掉固定 220 上限，300 宽下不再硬裁；提交文件行与工作区文件行补完整路径 tooltip；页头引用选择按钮长名限宽 220 并有 tooltip。测试：`TestTheFileTabNamesItsFullPath`（含 300 宽）、`TestACommitFileRowTooltipsItsFullPath`、`TestTheCurrentBranchMarkIsNotPartOfItsName`。
- [x] **历史错误态仍可翻页与走旧行**（产品复查补充）：页脚「加载更多」在 `historyErr` 时不再出现，`loadMoreHistory` 与 `moveCommitSelection` 都提前返回；`TestAFailedHistoryListIsNotPagedOrWalked`。
- [x] **980 宽提交行的图谱与胶囊没有宽度预算**（设计复查 2026-11-15 提出，12:46 完成）。图谱按列表宽度用 `FitWidth` 压到约 26%（8 lane 在 309 宽下变 80px，车道间距由 32 收到 8，最低 6）；引用胶囊按宽度 3/2/1 递减，其余折进 `+N`；单枚胶囊上限从 150 收到 24%（最低 76）；窄行把 HEAD 词并入分支胶囊的勾标记（tooltip 仍说明 HEAD 指向），标题截断时才有 tooltip，行内间距 6→4。980/8 lane/HEAD+3 ref 实测标题约 110px、图谱 80px、胶囊不出界、`+3` 可见。测试：`TestTheNarrowCommitRowKeepsRoomForTheSubject`、`TestANarrowRowCapsItsRefs`、`TestRefSlotsFollowTheColumnWidth`、`TestTheSubjectTooltipOnlyAppearsWhenItWouldClip`、`TestANarrowRowFoldsRefsByRank`、`TestANarrowGraphPacksItsLanes`；`make check` 全过。
- [x] **点「重试」后横幅立即消失，读取期间无反馈**（产品复查 2026-11-15 提出，13:09 完成）。`openRepository` 开始时不再清 `loadErr`，只有读取成功或换成新错误时才更新；读取期间横幅保留，按钮变为禁用的「正在重试…」，成功后才消失。测试：`TestARetryKeepsItsBannerWhileTheReadRuns`（banner、标题、禁用态）；保留逻辑本身是一行状态改动，未做异步的「点击后立即断言」测试以避免与后台读取竞态。
- [x] **忙碌时重置会静默关闭对话框**（产品复查 2026-11-15 提出，13:23 完成）。软/混合/硬模式按钮在 `busy || loading || repo == nil` 时禁用；只有 `performReset` 确认操作已启动才关闭对话框，未启动时保留模式与提交上下文。硬重置仍先关对话框再问确认。测试：`TestResetModesAreDisabledWhileTheRepositoryIsBusy`（禁用态与点击不关窗）、`TestResetKeepsItsDialogWhenNothingStarts`。
- [x] **同名「重试」可能同时出现**（产品复查 2026-11-15 提出，13:09 完成）。抽出 `failedRead`：有 `loadErr` 横幅时列表错误只显示文字，不再重复整页重试；分支/标签/工作树/子模块的重试按组命名（「重试本地分支」「重试远端分支」「重试标签」「重试工作树」「重试子模块」）。测试：`TestTheReloadBannerOwnsThePagesOnlyRetry`（横幅 + 分支错误只有 1 个「重试」；回退门控后转红）、`TestBranchReadFailuresOfferRetry`（两组失败显示两个不同名称，无裸「重试」）、`TestBothWorkspaceSectionsOfferTheirOwnRetry`。
- [x] **Stash 返回「部分结果 + error」时的旧详情清理**（产品复查 2026-11-15 提出，13:23 完成）。新增 `clearStashDetail`：列表读取失败时清 `selectedStash/stashDiff/stashDiffErr` 并递增 `stashSeq`，在途补丁回调被丢弃；`reconcileSelection` 不再在 `stashesErr` 时自动选部分结果的首行，详情回到占位。单条补丁失败的重试路径不变。测试：`TestAFailedStashListDropsTheOldPatch`（清空、序列递增、不自动选中、旧补丁消失；回退自动选中守卫后转红）。
- [x] **取消后看不到仓库处于哪个操作中**（13:54 完成）。operation 状态移入标题栏，任何页面刷新后都显示「进行中：rebase」等；按钮可点击跳到工作区（那里有继续/跳过/中止）。工作区页头原有同名标签移除，避免与标题栏重复；operation 卡片仍留在提交栏。测试：`TestTheTitleBarNamesAnOperationLeftRunning`（标签、tooltip 指向工作区、点击切换页面）。
- [x] **粗线 + 小节点的合并方框头轻**（13:40 完成）。合并方框下限改为 `graphNodeDot(node,line)+1`（即线宽 + 2），粗线 + 小节点下由半边长 4（8×8）增到 5（10×10），方框描边也从固定 2 改为 `max(2, line)`，不再比 3px 车道轻。测试：`TestGraphNodesStayVisibleInEveryLineWidth` 补「方框 ≥ 线宽+2 且 ≥ 圆点+1」与粗线小节点断言；回退下限后转红。
- [x] **工作区分区折叠没有持久化**（14:25 完成）。折叠状态随每次点击写入设置文件：`workspaceFolded` 是按仓库根的最近使用列表（`root` + `sections`），每项只记已折的分区、按页面顺序排列；再次点击展开即从文件移除该项。保存时把当前仓库移到列表最前，其余保持原顺序，最多保留 20 个仓库，超出的旧仓库自动丢弃。打开仓库时按 `repo.Root` 取回自己的折叠集，另一个仓库互不影响；设置页「重置」会展开所有分区并清空这份列表。手改文件里的未知分区名、重复分区、空仓库项会被丢弃，同一仓库出现两次时以靠前（较新）的为准。测试：`TestFoldedWorkspaceSectionsSurviveARestart`（折叠→重读设置→恢复、另一仓库为空、展开后不留空项、重置清空）、`TestFoldedRepositoriesStayApartAndBounded`（顺序、空项、20 上限）、`TestFoldedSettingsOnlyKeepWhatTheWorkspaceDraws`；把 `rememberFilesFolds` 改为空操作后重启测试转红。`make check` 全过。
- [x] **日期过滤只有「这一天」入口**（14:10 完成）。提交右键菜单在「筛选这一天」后补「筛选最近七天 / 筛选本月」，两者与作者/日期过滤同组、同规则：写入搜索框而不是隐藏状态，搜索框自带的 × 一键清除。最近七天写成 `2026-11-09..2026-11-15`（今天是窗口右端），本月写成 `2026-11`（月份前缀本来就是每天的匹配串）。`filterCommits` 支持 `from..to` 闭区间，端点可以是日 / 月 / 年，任一端可省略（`2026-11..` 表示从该月起）；不像日期的文本仍按子串搜索，日期读不出（Day 为空）的提交不进入区间结果。测试：`TestADayRangeNarrowsTheHistory`（含开区间、单日、非法文本回退）、`TestTheMenuFiltersTheLastWeekAndThisMonth`（入口写入的搜索词与过滤结果一致、无日期提交时仍可用）；回退区间解析后两测试转红。`make check` 全过。
- [x] **设置页命令面板触发按钮没有名称**（13:40 完成）。核对后指设置页「配色」下拉与标题栏「分支」下拉的触发按钮：可见内容是文字/圆点，但触发元素没有 Role/Label，读屏只读到当前值。两者补 `Role(按钮)`、`Label`（配色/分支）与当前值 `Description`，可见布局不变。测试：`TestThePaletteTriggerNamesItself`、`TestTheBranchSelectorNamesItself`。
- [x] **快捷键注册表**（14:43 完成）。新增 `internal/lobedesk/shortcuts.go`：`shortcutTable()` 是窗口级键位的唯一来源，每项带 `ID / Mod / Key / Keys（卡片显示的键名）/ Group / Label（i18n）/ SkipWhenTyping / Field / Available / Run`。`registerShortcuts` 只遍历这张表注册；`Field`（⌘F 由搜索框自己绑定）不占用窗口；`Available` 为假的条目干脆不注册（无行选择时的 ⌘C 仍交给系统复制）。侧栏的「⌘1 / ⌘2 / ⌘,」改读同一张表，不再手写。原来 `view.go` 的 21 个 `OnWindowShortcut` 全部迁入，行为逐项保持（页面切换、⌘R 忙碌守卫、⌘↩ 对话框守卫、方向键/Home/End/Page、Space/⇧↵ 输入框守卫、↵ 三页默认动作、⇧↑↓ 行选择、Esc、⌘C）。测试：`TestEveryWindowShortcutComesFromTheTable`（扫描包内源码，`shortcuts.go` 之外出现 `OnWindowShortcut(` 即失败）、`TestEveryShortcutIsListedAndRegistered`（ID/键名/双语词条/分组/动作齐全，字段绑定不得带窗口动作，两个窗口条目不得抢同一键）、`TestTheTableStillDispatchesThePageKeys`；清空注册后页面切换测试转红。`docs/shortcuts.md` 更新了来源说明。
- [x] **设置页的快捷键说明**（14:43 完成）。设置页在「Git 行为」与「关于」之间新增「快捷键」卡片，按「全局 / 历史 / 工作区 / 分支 / 工作树」分组（无键位的 Stash 不显示空标题），每行「动作 —— 键名（等宽）」，动作名随语言切换；只读，不做自定义。卡片直接遍历 `shortcutTable()`，所以新键位只要进了表就会出现在卡片里，反之会被源码扫描测试拦下。测试：`TestTheSettingsCardListsEveryShortcut`（每条动作与键名都在页面上、有内容的组都有标题；去掉卡片调用后转红）。`make check` 全过。
- [x] **设计复查落地：对比度、字号预设、错误占位、工作区行距**（17:08 完成）。① 新增 `tertiaryTextInk(tk, ground)`（`selection.go:25`，复用 `capsuleInk`：浅底压深、深底向白提亮、已达标原样返回）：侧栏分组计数、更多提示与逐行快捷键在 `FillTertiary` / 选中底色上补到 ≥4.5:1（Latte 的 Subtext0 在 Base 上 4.37:1、选中底色上 3.65:1），`view.go:369,448,550` 使用；补测试时发现 Frappe 深色在选中底色上只有 4.28:1（设计 agent 原以为深色都达标而直接返回原色），因此改用 `capsuleInk` 的提亮分支，`TestTertiaryLabelsReadOnEverySidebarGround` 覆盖全部方案 × 浅/深 × 三种地面，`TestTertiaryInkOnlyMovesFailingColors` 保证达标的颜色不被改动；回退到 `deepInk` 后 Frappe 用例转红；② 页头/空页/提交标题的 Heading 5 与分支、配色、引用触发器的裸 `ui.Text` 改用 `scaledFont`，字号预设不再只缩放一部分（`view.go:650,740`、`commit_info.go:38`、`workspace.go:144`、`settings_page.go:201`、`history_ref.go:262`）；③ 提交补丁与工作区补丁的读取失败改用 `diffPlaceholder` 的图标+居中样式，文件栏加载/空态改用 `listEmptyState`，重试按钮原位保留（`history.go:840-858,932-962`、`workspace.go:432-442`）；④ 工作树行两行文字的行距由 `Gap(1)` 改为与提交行一致的 `Gap(4)`（`history.go:791`），工作区补丁头的目录名在 260px 截断时补完整路径 tooltip（`history_panes.go:262`）。验证：`make check` 全过（含 `go test -race ./...` 与 `go vet`）；四组合窄窗口截图已用新构建重拍。未改判据：30/34/38px 的三种头部高度保持原样，未全局替换 `TextTertiary`。
- [x] **修改最后一次提交的预填说明可能来自别的分支**（产品审查 2026-11-15 提出，17:27 完成）。位置：`workspace.go:904-919`（用 `a.commits[0].Subject` 同时决定复选框与预填）、`history_ref.go:508-548`（限定引用后整体替换 `a.commits`）、`gitrepo/history_page.go:45-47`（全引用按 topo-order，首行不保证是 HEAD）、`gitrepo/repository.go:255-265`（amend 作用于 HEAD）。现状：先看别的分支历史，再点工作树行勾「修改最后一次提交」，说明框里是该分支最新提交的标题；不改就提交会把 HEAD 的说明改成它。历史读取失败或当前引用没有提交时复选框直接消失，即使 HEAD 存在。做法：新增 `gitrepo.HeadSubject`（`log -1 --format=%s HEAD`，空仓库返回空），每次 reload 与状态一起读入 `a.headSubject`；提交框的复选框与预填都改读它，不再碰 `a.commits[0]`。测试：`TestHeadSubjectReadsHeadEvenWhenAnotherBranchIsNewer`、`TestHeadSubjectOfAnEmptyRepositoryIsEmpty`、`TestAmendPrefillsHeadNotTheBrowsedBranch`（渲染提交框点选 amend，预填 HEAD 而非当前浏览分支）、`TestAmendStaysAvailableWithoutALoadedHistory`（历史读取失败时复选框仍在）；回退到 `a.commits[0]` 后两个提交框测试转红。`make check` 全过。

- [x] **焦点在工作树行时 Space/Enter 被空处理器吞掉**（产品审查 2026-11-15 提出，17:27 完成）。位置：`history.go:796-800`（`OnShortcut(Enter/Space, func(){})`）、窗口级动作 `shortcuts.go:142-157`、文件行的正确做法 `workspace.go:345-353`；承诺见 `docs/shortcuts.md:32-33` 与「文件栏的键盘操作」条目。现状：点首行（进入提交框最自然的方式）后该行保留焦点，Space 不暂存、Enter 不打开已自动选中的文件；⇧↵ 仍可用（只注册在窗口级），于是文档并列的两个键一个生效一个不生效。做法：处理器改成状态感知——`worktreeSelected` 时执行 `toggleSelectedStage` / `openSelectedFile`，否则继续吞键以免焦点行把选择弹回。测试：`TestSpaceOnTheFocusedWorktreeRowStagesItsFile`（真实仓库、点首行后用 Space，完成后 `base.txt` 在索引里）与 `TestSpaceOnTheFocusedRowLeavesACommitSelectionAlone`（同焦点、选中提交时 Space 不启动操作）；回退成空处理器后第一个测试 10s 超时转红。Enter 走同一状态判断与 `openSelectedFile`，没有单独测试它拉起外部应用。`make check` 全过。
- [x] **侧栏「工作树」不标当前工作树**（产品审查 2026-11-15 提出，17:27 完成）。位置：`view.go:331-344`（标题只取 `filepath.Base`，`sidebarRefRow` 的 `current` 硬编码 `false`）、`view.go:404-437`（只有 `current` 才画勾与「当前」，且短名没有 tooltip）、`workspaces.go:79-93`（工作树页已按清理后的路径判断当前）。现状：打开工作树时侧栏每个工作树看起来一样；两个不同父目录下的同名工作树显示成两行相同文字，点错就切到另一份 checkout。做法：按 `filepath.Clean` 后的路径判断 `current` 并传给 `sidebarRefRow`（复用绿色勾与「当前」描述）；`sidebarRefRow` 改为返回行元素，工作树行总是带完整路径 tooltip，当前工作树的描述写成「当前 · 路径」。测试：`TestTheSidebarNamesTheCurrentWorktree`（当前行有「当前」与路径，另一行只有完整路径）；把 `current` 写死为 false 后转红。`make check` 全过。
- [x] **中文里「远程」与「远端」两套叫法**（产品审查 2026-11-15 提出，17:27 完成）。位置：`i18n.go:186`（侧栏组名「远程」）、`:189`（「远端分支」）、`:554`（`busyDeleteRemoteBranch`「正在删除远程分支…」）、`docs/git-terms.md:34`（规定用「远端」）。做法：侧栏组名、自动获取说明与删除分支的进行中文案统一为「远端」（共 4 处），新增 `TestChineseCopyCallsRemotesYuanDuan` 扫描全部中文词条；`chineseEnglishTerms` 无需改动。把 `remotes` 改回「远程」后测试转红，恢复后转绿。`make check` 全过。
- [x] **工作区文件行与提交文件行是两套字号/行高**（设计复查 2026-11-15 提出，17:37 完成）。位置：`workspace.go`（`fileRow` 固定 `Height(30)`、12/10 字号）与 `history.go`（`detailFileRowHeight()` = `scaledFont(26)`、11/9 字号）。现状：同一 300px 文件栏里两种密度，大字号下工作区行变得局促；Lobe 的 `action.StyleButton` 与 Tag 字号固定（Small 12 / 默认 13），标题栏与暂存控件不跟随字号预设。做法：新增 `file_row.go`，两处文件行共用 `fileRowShell`（高度 `scaledFont(26)`、`Padding(0,6)`、`Gap(8)`、选中/悬停底色、`RoleOption`）与 `fileRowPath`（11/9 字号、文件夹阈值与完整路径 tooltip 的入口），删掉 `detailFileRow*` 常量；工作区行的暂存图标 24/13 收到 22/12 以放进 26px 行，行的 `Description` 仍由列表设置（tooltip 写在共用外壳里会先占掉描述，测试当场发现）。测试：`TestBothFileListsDrawOneKindOfRow`（小/标准/大 三档下两行高度都等于 `scaledFont(26)`，文件名与文件夹的排版高度一致；把工作区行改回 `Height(30)` 后三档全红，并报出 30 vs 26/29.9）。`make check` 全过。截图：`scripts/native-narrow-shots.py` 增加中文浅色 + 大字号一组，并在每组里按 Home 回工作树行再截一张，共 10 张；大字号下两栏文件行等高、不拥挤。未做 Lobe 组件字号覆盖（风险高，等上游或另行评估）。

- [x] **重载会让在途的历史走查永久停住**（产品复查 2026-11-15 提出，17:56 完成）。位置：`app.go:538-548`（`loadSeq++` 后只置 `loading`）、`:706-729`（两个历史标志只在 `reset` 分支清）、`:787-792`（`reload()` 传 `reset=false`）、`history_ref.go:515-540`（走查回调只在未过期时清标志）。现状：切换引用后立刻 ⌘R、自动 fetch 完成或暂存后自动重载，在途走查的回调被判过期并直接返回，重载自己也不清标志，于是列表永远停在「正在重读历史…」、方向键被守卫挡住、页脚「加载更多」可见但点了没反应；先点「加载更多」再 ⌘R 则停在「正在加载更多…」。做法：`openRepository` 在 `loadSeq++` 后清 `historyLoadingMore` / `historyReloading`，并清掉被替换走查的 `pendingJumpSHA`（它指向的走查已不存在）。测试：`TestAReloadReplacesAHistoryWalkStillOut`（三个状态在 `reload()` 后归零）；移除这三行后测试转红。
- [x] **切页后引用选择器仍占着方向键**（产品复查 2026-11-15 提出，17:56 完成）。位置：`view.go:603-614`（`openPage` 不碰 `historyRefOpen`）、`keyboard.go:100-109`（`moveSelection` 在弹层打开时直接返回）、`history_ref.go:265-266`（`wasOpen` 决定是否清查询）。现状：打开「所有引用」后按 ⌘2/⌘3/⌘4 切页，历史页头被卸载而弹层状态没关，分支/Stash/工作树页的方向键全部失效；⌘1 回来还会带着旧搜索词重新弹出。做法：`openPage` 离开历史页时关闭弹层并清查询；换仓库的重置分支同样清。测试：`TestTheRefPickerClosesWhenAnotherPageOpens`（切到分支页后状态归零且方向键选中第一行）；移除清理后转红。
- [x] **详情隐藏或引用弹层在前时 ⌘↩ 仍会提交**（产品复查 2026-11-15 提出，17:56 完成）。位置：`keyboard.go:65-74`、`shortcuts.go:82-89`、文档 `docs/shortcuts.md`（写明「摘要页在前」才生效）。现状：`canCommitShortcut` 只看标签页与对话框，隐藏整个详情区或弹层盖住提交框时，⌘↩ 仍按已输入的消息提交，读者看不到提交的是哪条说明与哪批暂存。做法：`historyDetailsHidden` 或 `historyRefOpen` 时拒绝。测试：`TestCommitShortcutReachesTheHistoryCommitBox` 补两段（隐藏详情、弹层在前都拒绝，恢复后重新可用）；移除守卫后转红。
- [x] **引用弹层的重试在读取中仍可点**（产品复查 2026-11-15 提出，17:56 完成）。位置：`history_ref.go:319-329`；对照 `view.go:715-736` 的统一重试反馈。现状：其他失败读取在 `loading` 时显示「正在重试…」并禁用，弹层里的重试却始终可点，每点一次启动一次完整重读（`openRepository` 只挡 busy，不挡 loading），多次点击并发读取并互相作废。做法：与 `failedRead` 一致——`loading` 时标签换成「正在重试…」并禁用，`busy` 时保持「重试」但禁用。测试：`TestTheRefPickerRetryWaitsForTheReadItStarted`（读取中只有禁用的「正在重试…」，没有第二个「重试」）；把 `retrying` 写死为 false 后转红。
- [x] **设置说明与文件行缩放不符**（产品复查 2026-11-15 提出，17:56 完成）。位置：`i18n.go` 的 `appearanceHelp`（中文「列表行高与差异行距保持不变」、英文同义）。现状：上一轮统一文件行后 `fileRowHeight()` 随字号缩放，提交列表行与差异行距仍固定，说明把两者混成一句，读者会以为设置失效。做法：中文改为「提交列表行高与差异行距保持不变，文件列表行高随字号缩放」，英文点名 commit rows / file rows；测试 `TestAppearanceHelpNamesTheFixedSpacing` 增加「文件列表 / file rows」要求。
- [x] **设计复查第三批：大字号 SHA 行、英文搜索占位、灰色标签对比度**（17:56 完成）。① 大字号下摘要的 40 位 SHA 把复制按钮挤出面板：哈希改 `SingleLine().Grow(1).Basis(0).MinWidth(0)`、复制按钮 `Shrink(0)`，截图中 SHA 中略而按钮在位（`commit_info.go`）；② 英文搜索占位被 292 宽搜索框截在「ref」：缩短为「Search commits, authors, SHA, refs」（`i18n.go`）；③ 「0 个已暂存」用 `gray` 标签，Lobe 的灰度族不在配色派生范围内，填充标签的字色固定为 `#666666`，实测浅色 4.2:1、深色 2.2:1：在 `themes.go` 里按调色板中性色派生 `GrayFillTertiary/GrayFillQuaternary`，并用 `capsuleInk` 算出达标的 `GrayActive`，所有 `gray` 标签（含签名不可验证）一起修好；计数器本身改用中性标签。测试：`TestPaletteTextTokensStayReadable` 增加「灰标签字色 / 灰填充」≥4.5:1，删掉派生时四种方案全部转红（Latte 4.47、Mocha 2.42、Frappe 1.79、Macchiato 2.14）。截图：`make lobe-e2e-build` 后重拍 10 张，大字号 SHA、英文占位与中性徽章均已复核。

### 文档与规则

- [x] **中文界面保留英文的 Git 术语清单**（15:26 完成）。`docs/git-terms.md` 写清两类保留（产品/平台名、无稳定中文名的 Git 对象与旗标）与「动作一律中文」的规则，并逐项列出词、示例键、理由；清单唯一来源是 `i18n.go` 的 `chineseEnglishTerms`。新增 `TestChineseStringsOnlyKeepTheListedEnglishTerms`：扫描中文文案里的拉丁词（至少两个字母，忽略 `%s` 这类占位符），未登记即失败并报出 i18n 键。现有清单：Git、SuperGit、Lobe、MyGo、UI、Catppuccin/Frappe/Macchiato、Finder、Stash、HEAD、detached、SHA、Ours/Theirs、hooks、force-with-lease、.gitmodules。临时加入 `"index"` 后测试转红。
- [x] **签名状态的边界说明**（15:26 完成）。`docs/signature-status.md` 记录读取命令（`%G?`/`%GS`、必须带 `--no-show-signature`）、状态字母与文案/配色表，以及四条边界：无法校验可能显示成未签名（2026-11-15 实测：未配 `allowedSignersFile` 时 stdout 给 `N`、stderr 报错、`git verify-commit` 退出码 1）、B/X/Y/R 只有单元测试、不显示 `%GK` 指纹、签名者是用户数据。改动检查单要求新增状态字母时同步 `TestSignatureBadgesCoverEveryStatus`。
- [x] **整行选择与文本选择两套模型的边界**（15:26 完成）。`docs/selection-model.md` 写清两套模型：行号按钮按行选择（工作区只收增删行，提交补丁含上下文行；单击/Shift/⌘/拖动/右键/方向键/Esc/⌘C 各自规则），代码文本拖选是独立段落；⌘C 在有行选择时归行复制，否则留给系统复制文字；以及切文件、切提交、补丁重读时的清空时机。补 `TestAContextAnchorDragSelectsOnlyChangedLines`：从上下文锚点拖到增行只选中区间内两个改动行，悬停在上下文行不再扩展；去掉 `dragToLine` 的类型守卫后转红。
- [x] **版本号自动注入**（16:24 完成）。新增 `tools/appversion`：短版本取最近标签（去掉前缀 `v`，只接受 1–3 段数字），构建号取 `git rev-list --count HEAD`；标签不可用（如 `v1.2.3-rc1`、`nightly`）、提交数不是数字、不在 Git 仓库或 git 不可用时，沿用 `packaging/Info.plist` 的原值。工具只重写复制到 bundle 的那份 plist，源码文件保持不动。`make app` 改为调用该工具；`TestVersionFieldsFollowTheNearestTag`（9 种输入）、`TestTheRewriteKeepsTheRestOfThePlist`、`TestGitVersionReadsTheCheckout` 覆盖规则。验证：本目录（无 `.git`）连续两次 `make app` 均为 `0.1.0 (1)` 且 `plutil -lint` 通过；临时仓库带 `v1.2.3` 标签时为 `1.2.3 (2)`，无标签时为 `0.1.0 (1)`。把标签前缀处理换成直接取值后测试转红。README 已记录规则。

### 测试与工具

- [x] **文档同步测试改为共用常量**（16:38 完成）。`docs/graph-colors.md` 新增第 7 节「机器校验的数值」表（capsule-tint/border/height/corner、宽 150/320、node-dot-radius/merge-half/ring/halo、lane-ink-min），正文措辞不再被测试读取。测试改为按键名解析该表（`graphColorDocNumbers` + `checkDocNumber`），期望值仍由代码常量与 `graphNodeRing` 现算；`TestEveryDocNumberRowIsChecked` 防止表里出现无人读取的键。验收：把正文「半径 4」改成「半径四」后两个测试仍通过；只把表里的 4 改成 5 后 `TestGraphColorDocMatchesTheGraphGeometry` 失败并报出代码值；恢复后转绿。`make check` 全过。
- [x] **菜单文案三份的成组测试**（15:42 完成）。`i18n_action_copy_test.go` 新增 `actionCopyGroups`：每个动作一组，登记菜单项、对话框标题、确认按钮、`busy*` 进行中文案与 `*Failed` 失败提示，并写明该组的中文动词与英文词干；`TestEveryCopyOfAnActionUsesOneVerb` 要求每条中文包含该动词，`busy*/Failed` 的英文以该词干开头的词出现。菜单与标题的「…」关系仍由既有测试保证，通用按钮（保存/删除/创建）不分组。
- [x] **动作名与失败文案的对应**（15:42 完成）。同一张 `actionCopyGroups` 表把 `busyPush` 与 `pushFailed` 等成对登记；覆盖检查要求每个 `busy*`（`busyNow` 除外）和每个 `*Failed` 都必须出现在某个动作组，没有动作副本的失败提示登记在 `actionCopyExempt` 并写原因（读取失败、目录选择、剪贴板、冲突解决）。顺带修正两处真实漂移：`busyRevert`「正在还原…」→「正在回退…」、`busyDiscard`「正在丢弃改动…」→「正在放弃改动…」，`discardMessage` 的「丢弃」也改为「放弃」；回退 `busyRevert` 后测试转红。规则写入 `docs/git-terms.md`。
- [x] **引用选择器与提交选择弹窗的重复代码**（16:10 完成）。新增 `picker.go` 收拢两者共用的部分：`stepPickerCursor`（环绕移动，可用 `selectable` 跳过不能选的行）、`pickerCursorAt`（Enter 取高亮或第一条可选行）、`placePicker` / `repairPicker`（首帧与查询变化时放置高亮、刷新缩表后拉回并只滚动一次）、`pickerSearchBar`（同一搜索框 + Up/Down + Enter 接管）、`pickerRowBase`（30px 选项行外壳）、`pickerRows`（320px 上限滚动列表）与 `pickerReveal`。两个弹层保留各自不同的行内容、分组、错误行与「当前分支不可选」规则，一步都走共享实现；既有 helper 名与测试调用保持不变（`stepRefCursor`、`refChoiceAt`、`stepCheckoutCursor`、`checkoutBranchAt` 改为委托）。`go test -race` 与 `make check` 全过；把 `stepPickerCursor` 的环绕去掉后 `TestRefCursorWrapsAndFollowsTheQuery` 转红。
- [ ] **子模块忽略配置的一致性测试**。等「子模块 ignore」的 A/B 选定后，补一条两种配置下 Status 与 Diff 一致的测试（第 218 轮测试缺口的最后一项）。
- [x] **补「重载失败」与「引用选择器错误文案」测试**。前者见上面的修复项；后者断言 `tt.Find(historyErr)`，并在无错误时确认 Tab→Down 仍能移动高亮。（已完成 2026-11-15 12:13：新增重载横幅/首次打开区分/标签分组/引用弹层方向键/重试忙碌态/工作树表单恢复六组测试）
- [x] **工作树 Enter 的创建路径测试**（15:56 完成）。`worktree_form_test.go` 建真实仓库（`git init` + 一次提交）与 `worktreeApp` 夹具，把 `performDoneAs` 的收尾投递接到测试通道（新增 `app.runOnMain` 测试缝，`hopToMain` 默认仍走 `mygo.RunOnMain`），由测试自己执行收尾以避免与 worker 竞态。`TestEnterNeedsACompleteWorktreeForm`：分支为空时 Enter 不关表单、不启动；`TestEnterOnACompleteWorktreeFormCreatesTheWorktree`：完整表单 Enter 后 `busy` 且 `busyKey=busyCreateWorktree`，收尾后目录存在、`feature` 分支出现。把 Enter 的提交条件改为恒假后转红。
- [x] **工作树创建失败的真实失败路径**（15:56 完成）。`TestAFailedWorktreeCreationReopensTheFormWithItsValues`：目标目录预先存在，git 拒绝后表单重新打开，路径、分支、起点三项原样保留，应用退出 busy。把恢复回调里的 `form.open = true` 改为 false 后转红。
- [x] **窄窗口四组合截图复查**（17:08 完成，含设计修复后的重拍）。新增 `scripts/native-narrow-shots.py`：自建夹具（8 个提交、3 个本地分支、1 个远端分支、1 个标签、工作区有一个未提交改动，提交里有长中英文标题和真实文件改动），按 zh/en × light/dark 预置 `settings.json`，用 `SUPERGIT_WINDOW_SIZE=980x640` 启动应用各截一张到 `build/narrow-shots/`，同时存下快照文本。四张图（1960×1280，即 980×640@2x）复查：历史列表、文件栏、摘要/补丁栏、侧栏计数均无相撞；长标题中略只出现一个省略号，`origin/…` 胶囊被 24% 上限折成 `+1`；深色下胶囊墨色与选中行可读；侧栏计数完整（放大核对「3」未被裁）。注意：该页面属于 Lobe 预览应用（`make lobe-e2e-build` 产出的二进制），`make e2e-build` 的 `supergit-e2e` 是旧的 desktop 界面，用它截不出这一版布局（本轮先踩了这个坑，截图一度是旧三栏界面且窗口 1440 宽）。截图前需等「正在加载差异…/Loading diff…」消失，否则文件栏会停在加载态。首轮截图后设计 agent 又改了对比度、行距与占位样式，最终四张图是 `make lobe-e2e-build` 重新构建后重拍的。17:37 起脚本再加一组中文浅色 + 大字号，并在每组里按 Home 回到工作树行再截一张（共 10 张），用来对照两栏文件行；补丁行不是无障碍标签，等它出现要改为等文件名行 + 加载文案消失。

### 验证缺口（环境解锁后）

- [ ] **设置窗口的交互复测**。待补：⌘W 关闭后重开、面板内切换主题时主窗口同步、重置设置确认框落在设置窗口、最小化后 ⌘, 恢复（代码已加 `Restore()`，无头不可验证）。另外复现一次「锁屏 + ⌘,」建窗失败与主窗口持续「still draws」的现象，判断是 macOS 遮挡还是 `invalidate` 失效。
- [ ] **锁屏时的 e2e 限制记录**。`CGSSessionScreenIsLocked=true` 时 ⌘ 类按键仍能切页，但 `click`/`pointer`、方向键和焦点断言都不生效。跑 UI 复测前先用 `ioreg -n Root -d1` 确认锁屏状态；锁屏期间只跑单元测试与构建。
- [ ] **长分支列表的 idle 重绘复测**。现状：三次全新启动（含滚轮滚动后）`wait` 都在 0.0s 内 settle；唯一一次「still draws」出现在按 ⌘, 之后，设置窗口没有出现在 `windows` 列表里。期望：解锁后重跑「打开设置窗口 → 关闭 → 滚动 300+ 行分支页」，`wait` 仍能 settle。

---

## 三、观察与讨论

### 布局与渲染

- **说明文字限宽**：`settingsHelp` 统一限宽 520px、最多 3 行；新增说明先按这个长度写，若必须更长，改为可展开的「更多」而不是直接截断。
- **设置卡片顺序回访**：已按「外观 → 提交列表 → 历史线图 → 提交详情 → Git 行为 → 关于」分组，实际使用一段时间后再决定是否把「Git 行为」提前。
- **慢远端下自动 fetch 的提示时机**：打开仓库时执行一次自动 fetch，但打开很慢的远端时标题栏提示的出现时间取决于仓库加载完成，尚未在慢网络下目视。
- **「加载」与「获取」两组动词**：「加载」用于差异/仓库/历史，「获取」用于 fetch；观察读者是否混淆「加载仓库」与「获取远端」。
- **设置页路径选区范围**：选区只覆盖「路径：xxx」这一行，向上拖进关于卡片正文不会连成一段；观察是否有复制整段的需求。
- **固定高度不随字号缩放**：行高、胶囊高度（20px）、图标容器等仍是固定值，1.15 倍下目视正常；若以后加「特大」预设，这些高度要一起缩放。
- **复制按钮与字号**：复制按钮宽度参与 SHA 行布局；1.15 字号 + 980 宽下 SHA 行不换行、图标完整，字号再放大需复查。
- **详情胶囊的字符预算**：详情面板引用胶囊用 40 字符预算，980 宽下是否仍会出现两个省略号未复查；宽度上限已随字号缩放。
- **差异里最后一个文件的滚动定位**：受滚动上限限制，内容不足一屏时文件头只能尽量靠上，到不了顶部；与常见 Git GUI 一致，观察是否有用户觉得「点了没反应」。
- **行号栏的固定宽度**：行号栏 92px、每列 34px、`diffWidth` 的 +110 都是固定值，不随字号缩放；5 位行号以内够用，观察超过 99999 行的文件。
- **「差异已隐藏」文案宽度与二次说明**：中文 10 号字宽 5 个字，英文更短；若文案变长，块头右缘可能与左侧「N 个文件变更」相撞。提示只解释了「为什么没有差异」，没有解释「点击文件后发生了什么」；若读者仍以为点文件没反应，在文件列表空态补一句恢复路径。
- **页内进度**：进度只在标题栏，页内按钮只有禁用态；观察长操作（克隆、大仓库拉取）是否需要在内容区放进度条。
- **文件行文件夹阈值**：260 是旧分栏宽度下定的；文件栏现在 180–300 自适应，阈值是否过早出现或一直不出现，需要与面板宽度联动复查。
- **A/M 同现两块的显示**：已在「待你选择」里；若保持现状，观察新读者是否会误以为重复。
- **Stash 行悬停提示**：Stash 行仍用消息作为悬停提示；行只有一行文本、300px 宽下经常中略，因此保留。若行加宽或改为两行，同步删掉提示。
- **引用胶囊的悬停提示**：行提示已删，胶囊提示保留，会盖住下一行约 20px；观察是否需要改为更短的提示，或只在胶囊被中略时出现。
- **提交选择弹窗不带作者/日期**：顶部只给短 SHA 与标题，作者/日期在右侧详情里，这是有意避免重复展示；若读者要核对「这条提交是谁的」，再考虑补一行。
- **单独成行的「时间」已被移除**：作者与时间合行后删掉了 `when` 的中英文键和 `metaRow`；以后若想换行显示时区或单独成行，需要把键和行一起加回。

### 历史与提交

- **标题被中略时的提示**：第 186 轮删掉了整行提示，标题被中略时只能选中后看详情。期望只在确实被截断时提示全文；难点是 MyGo 没有现成的文本宽度 API，需先确认 `display.Text` 的绘制尺寸能否读回，否则无法判断。等上游能力，不阻塞其他工作。
- **提交菜单项的「…」视觉标记**：带「…」的菜单项（打开弹窗/确认）与立即执行的菜单项靠命名区分，视觉上没有区别；Sublime Merge 同样如此。观察是否需要给危险/弹窗项加图标或颜色。
- **签名状态边界**已列在「待开发」，这里只观察：真实使用中是否遇到 `N` 与配置错误混淆的情况。
- **文件历史里的提交若不在已加载窗口**：第 182 轮已让详情先在当前页列表里查、再退回已加载窗口；观察文件历史超过 200 条时的翻页体验。
- **分支选择的第一行回退**：搜索隐藏选中项后移到第一行，逻辑只覆盖当前页重绘路径；若以后分支列表改为分页或懒加载，第一行可能不在已加载范围内。
- **分支检出失败文案**：工作区不干净时检出失败，提示沿用 `git switch` 原文；没有在脏工作区下实测。观察是否需要补「先提交或暂存」的行动建议。
- **详情按钮的 Tab 长度**：分支详情 footer 的按钮默认可聚焦，位于列表之后，从列表按 Tab 会进入按钮组；焦点在按钮上时方向键仍移动列表选中。是否改成单一 focus group 与「Tab 停靠点模型」一起决定。
- **标签提示的位置文案**：空状态提示写的是「在提交上右键创建标签」；若分支页以后也能直接建标签，这句话要改回「在历史页」并同步测试。
- **空状态文案统一**：空仓库里分支页也有「创建第一次提交后…」提示，与历史页的指路句语义相近但位置不同；观察是否统一措辞。
- **键盘守卫的覆盖范围**：目前只覆盖历史列表；若分支/Stash 列表也出现「后台重载但旧行仍在」的状态，需要同样的守卫。
- **详情滚动的复位规则**：只在「选中变化」时复位；同一提交刷新（amend、重新加载详情）保留阅读位置，这是有意的。若读者觉得意外再改。
- **菜单文案的三份对应**已列在「待开发」；这里观察实际文案改动是否仍漏改其中一份。

### 工作区与差异

- **标题栏在窄窗口下的拥挤**（第 266 轮新增）：分支选择器、ahead/behind、操作标签、Fetch/Pull/Push、进度与刷新现在共用标题栏；窗口拉窄时先观察是否出现挤压或截断，再决定把远端动作收进一个菜单。
- **打开仓库时默认选中谁**（待确认）：现在仍自动选中第一条提交，工作树行只在列表顶部可见；若希望一打开就看到实时改动，可改为默认选中工作树行。请选择：A 保持第一条提交；B 默认工作树行（提交框与改动立刻可见）。
- **文件栏里的路径与暂存按钮争宽度**：工作区文件行右侧有 +/− 按钮，300 宽下 `internal/lobedesk/xxx.go` 这类路径的文件夹名常被中略；观察是否把文件夹列在窄栏里隐藏，只留文件名（悬停看全路径）。
- **错误状态下不画工作区行**：历史读取失败、重走中、文件历史加载中不显示工作区行，避免与错误提示冲突，这是有意的；观察读取失败时若读者想提交，是否会被迫先清掉错误或切回工作区页。
- **差异已隐藏时点击文件的反馈**：观察读者是否仍以为「点文件没反应」；若是，补空态说明（与上面的文案条合并处理）。
- **双击复位的判定窗口**：只验证了自动化 `click --clicks 2`；真实触控板与鼠标的双击间隔阈值不同。有人反馈「双击只拖动了一点点」时再复查。
- **行尾暂存图标与 Enter**：若保持现状，观察误触频率；若改成不抢 Enter，需要动 MyGo 的按钮结构。
- **本地瞬时命令的通用 busy 文案**：创建/重命名/删除本地分支、创建标签、解决冲突仍是「正在处理…」；都是本地瞬时命令，正常仓库不会超过 200ms，慢文件系统或超大仓库上发现卡顿再补常量。
- **操作计时器的空重绘**：每次操作都会排一个 200ms 的重绘定时器，若结束后立刻开始下一次，前一个定时器会触发一次空重绘（无害）；操作频繁时改成可取消的定时器。
- **延迟与快速失败**：延迟统一作用于所有操作（含 fetch），快速失败的 fetch 不闪进度但仍弹失败提示；观察是否需要在失败提示里带「重试」。
- **取消的实测覆盖**：只实测了带慢钩子的提交（以及代码路径上的 fetch），真实慢网络下的 fetch 取消未实测。
- **滚动条只在悬停/滚动时出现**：框架行为；触控板用户不用指针悬停时仍可能以为到底，观察反馈后再考虑常显选项。
- **窗口尺寸只能在启动时设定**：`SUPERGIT_WINDOW_SIZE` 只决定启动尺寸，运行中改不了；若频繁需要验证中间宽度，给 e2e 加 `window resize` 命令。

### 线图

- **惯性滚动下的线图接缝**：自动化滚轮是一次性位移，看不到连续滚动时行边界的采样；若有人反馈滚动时线闪断，再查行绘制是否按整数像素对齐。
- **节点「大」倍率的边缘间距**：节点 1.3 倍时首尾车道距列边缘约 1.3px；以后加更大倍率，需要同步核对 `graphWidth`（自然宽度为 16 + `graphLaneGap`×车道数，当前间距 20）与实际 `FitWidth` 车道中心距。
- **节点下限与选中环**：节点下限按「线宽 + 1」写死，选中环公式只在数学上守住缝隙；粗线+小节点已截图确认，触控板高倍率屏下未看。加「特小」节点或更大线宽时重新确认。
- **胶囊边框阈值**：1.25:1 是经验值，低于它时胶囊边框向墨色混合；若选中行上觉得深色边框偏重，可调低阈值或改为只提高底色比例。
- **车道间距的历史取舍**：第 249 / 250 轮曾比较 40 / 32；当前 `graphLaneGap=20`，窄列表经 `FitWidth` 继续压缩，最低 6 DIP。五次曲线让两端贴近竖直；后续调整以现有布局和回归为准，不再把 32 当作当前常量。

### 状态与操作

- **复制名称与历史入口**已在「待你选择」；这里观察右键菜单与详情图标的实际使用频率。
- **错误 toast 的英文首行**：若暂时保持，观察非英文读者是否能理解；至少把右侧详情 footer 让出来（见待选择）。
- **快速操作闪进度**：200ms 延迟已统一，观察是否有短暂操作仍闪一下转圈。
- **清除文件筛选时详情闪烁**：观察清除文件筛选、切换引用后详情是否出现旧内容一帧。
- **签名徽章的性能**：若做列表徽章，观察 200 条历史的首次加载时间。

### 工程与测试环境

- **锁屏时的 e2e 限制**：已列在「验证缺口」；这里只保留环境判断方法（`ioreg -n Root -d1`）。
- **`native-check` 对桌面焦点敏感**：完整跑三轮时第 1 轮 `--history`、第 2 轮 `--files` 曾因焦点断言失败，单独运行均通过；再次出现时记录是否有其它 App/自动化实例在前台，必要时加串行锁或前台等待。
- **e2e 滚轮与滚动条**：e2e 滚轮一次只移动约一两行，要滚到底更适合拖滚动条；写滚动验证时优先拖滚动条，若做常显滚动条补一条可拖动测试。
- **应用自发退出**：e2e 期间退出过两次（打开右键菜单后、提交完成后各一次），无崩溃报告、日志为空；再出现时启动 `GOTRACEBACK=all` 并用 `log stream` 抓原因。
- **980×640 下页面串页**：一次滚动详情后截图显示设置页，随后 ⌘2 恢复，复现步骤不明，怀疑是自动化 ref/坐标与真实布局的时序差；再出现时记录坐标与页面。
- **大字号窗口尺寸漂移**：1.15 字号 + 980×640 曾有一次在打开设置页后变成 1210×795，未复现；再出现时记录操作顺序与页面。
- **文档测试只按标准档计算**：`TestGraphColorDocMatchesTheGraphGeometry` 只按「标准」档的线宽/节点计算；改默认档时文档要说明是哪一档。
- **配色方案名测试**：`TestEveryPaletteHasItsName` 保证方案有名字；若再出现「表格里的 label 交给 `a.t(...)`」的写法，需要同样单独守一遍。
- **空状态提示测试只认关键词**：`TestEmptyStateHintsPointAtTheRightPlace` 只认「历史」「右上角」「above」，不保证语义；改文案时仍要读一遍句子。
- **省略号字面量测试的白名单**：白名单是 `i18n.go` 与 `model.go`（`elideMiddle`）；把 `elideMiddle` 挪到别的文件时白名单要跟着改，否则误报。
- **文档同步测试的数字仍靠子串匹配**：`半径 4`、`14×14` 等按子串匹配，文档改措辞会误报；已列在「待开发」的共用常量改造。
- **红绿测试的可读性**：实现细节型重绘测试缺说明；切换按钮测试只断言「开始切换」，未断言目标分支（需要等异步 git 操作完成或暴露待切目标），暂缓。
- **通知测试绕过归属判断**：`settingsNoticeOwner` 仍只能通过真实保存路径覆盖；要稳定覆盖需加 hook 或 e2e。
- **设置窗口 Restore/Focus 无自动化检查**：需要原生自动化或抽出窗口接口断言调用顺序，与「验证缺口」同一件事。
- **本地分支行 Space 的按钮路径没有用例**：按钮需要 Tab 到达，焦点 API 不支持按名聚焦；目前靠条件注册保证，留作观察项。
- **通知与对话框的跨窗口边界**：已修的 owner/队列见 DONE 第 235、237 轮；未覆盖的销毁分支与同窗口多对话框仍无法无头模拟。
- **`historyPageStale` 的指针比较**：`repo` 参数用指针比较；若以后支持同一路径的仓库对象重建，需要用路径或代次比较。
- **批量暂存按路径分批**：每 256 条路径一批，没有按参数字节数；单条极长路径（十几 KB）理论上仍可能撑爆命令行，目前没有这种路径。若出现超长路径，改成按字节切批。
- **状态刷新是全量读取**：3000 个文件暂存后刷新约 0.2 秒，可以接受；将来十万级改动需要增量刷新。
- **`diffWidth` 的 0.62 是刻意的余量**：比 Menlo 实际约 0.60 略宽，少算会把行尾裁在滚不到的地方；字号很大时最右会多出约 150px 空白，暂时保持估算。
- **「移除工作树」不走 `perform` 族**：用 `beginOperation` + 手动 `queue`，没有统一的重试/失败处理；若以后给操作加重试，要把它也接进去。
- **图标素材在仓库外**：原始素材默认在 `~/Downloads/GitApp_Icons`，换机器或重装后需要重新拿到；成品已入库，构建与运行不受影响。若要在别的机器改图，把素材放到仓库外但记录来源与校验值。
- **快捷键文档的来源会过期**：`docs/shortcuts.md` 的 Sublime Merge 键位来自本机安装包；升级后需要重新核对。

---

## 本轮整理说明

- 删除了所有已完成条目与按轮次的历史小节（内容见 DONE.md）。
- 合并了重复条目：设计参考与「讨论」清单、「待你选择」与按页待办、子模块两条、菜单长度两条、Tab 停靠点三条、通知与提示若干条。
- 删除了因第 251–252 轮布局重写而失效的条目：「信息块首帧跳动」「只有提交详情按内容收缩」「三块详情的固定高度/拖拽」「重置到提交对话框没有取消」（第 233 轮已补取消按钮并默认聚焦）。
- 原「待你选择」中已由后续轮次完成的项（如标签行选中语义、Stash 行 Enter、远端行可聚焦、重试按钮、对话框焦点）已从清单移除。

## 用户提供参考
- [ ] https://pbs.twimg.com/media/HXKNC1cW8AA-CUR?format=jpg&name=4096x4096 参考这个风格，比如按钮，颜色，tab 样式等
