# 工作台下一步改动计划

> **执行方式**：后续实现使用 `superpowers:executing-plans`，按任务逐项完成与验证。本文只整理需求和实施计划，不代表已开始修改应用。

**Goal:** 根据本次测试包的 8 点反馈，改善布局可调性、历史浏览、提交表单和文件操作的可发现性。

**Architecture:** 延续 Lobe 四列工作台与现有标签上下文模型；复用 `splitDrag`、历史分页后端、Git 操作入口与快捷键分发。新增界面入口必须沿用已有写入守卫、身份校验与确认流程。

**Tech Stack:** Go、MyGo、Lobe MyGo、`internal/lobedesk`、`internal/gitrepo`。

**Spec:** 本文件「一、需求与设计」是本轮需求依据；用户本次文字与参考截图优先。原有约束见仓库根目录 `TODO.md`，已有标签行为见 `docs/selection-model.md`。本文的「建议」与「实施默认」是规划建议，不能回写成用户已明确选择的产品规则。

**状态：已确认范围实现与验收完成（2026-11-16）；N05 新菜单 / 状态区域及安装待办保留。** 后续用户授权继续完成全部 TODO，原缩窄后活动标签屏外的轻微问题已修复，自动显露活动标签和关闭按钮；1600→320 DIP 回归 RED→GREEN、全量测试 / race / vet、两种构建、新一轮完整 16 组显示矩阵与真实分栏重启验证通过。续作记录位于 `.superpowers/sdd/2026-11-16-todo-completion/progress.md`。下方 Task 7 保留前批验收时间线。 用户通过 `/loop 继续完成计划未完成的内容` 授权实施。Task 1 / Task 3 / Task 4 已完成代码和自动化验证；工作区截图复查、提交表单及真实 Commit / Amend 原生冒烟通过，完整 16 组显示矩阵通过；Task 5 已完成文件头、批量操作及真实取消 / 丢弃 / 删除验证；Task 2 主历史 / 引用遍历、完整侧栏及文件历史自动分页已完成，主历史 2,405 条、文件历史 2,403 条原生验证通过；Task 6 能力盘点已写入 `docs/workbench-command-map.md`，新增菜单 / 状态区仍待范围明确；Task 7 原生验收及独立审查完成，两项重要发现修复后再次通过全量与原生回归，一项缩窄后活动标签可留在屏外的轻微问题保留。未 commit、未更新已安装应用。执行记录位于 `.superpowers/sdd/2026-11-15-workbench-next-improvements/progress.md`。

## 一、需求与设计

### 反馈对照

截图按内容命名，避免聊天中的 Image 编号与附件顺序不一致。

| ID | 用户反馈 / 参考截图内容 | 下一步目标 |
|---|---|---|
| N01 | History、Files 调整宽度；下面不需要加载更多，直接全部展示 | 调整各列分界；历史不再要求手动翻页，列表不截断；侧栏展开后也不以「更多」跳走代替完整内容 |
| N02 | 未提交改动在 History 中使用虚线，参考工作区节点与 HEAD 之间的虚线 | 工作区是独立的未提交状态，使用虚线连接真实 HEAD，不伪装成已提交节点 |
| N03 | 标题 / 说明两段 placeholder 是否误导成两个输入区 | 让标题与正文的字段边界明确；建议改成单行标题 + 多行正文 |
| N04 | 顶部项目名字与 icon 浪费空间，参考侧栏头像 / 仓库名截图 | 移除定位栏顶部重复的仓库头像和名称，History 上移 |
| N05 | 软件状态栏上面的功能需要进入规划 | 明确盘点顶部操作、系统菜单栏与状态反馈；位置解释和新增范围见 N05 专节 |
| N06 | Files 固定三类，参考三条折叠分组标题 | 按 Modified Files → Untracked Files → Staged Files 固定顺序显示，0 项分组也保留 |
| N07 | Commit 按钮不像按钮；显示当前提交 Author、Email | 保留明确按钮外形与可用状态，显示本次实际作者身份 |
| N08 | Diff 每个文件右侧显示可操作按钮，参考 Discard / Stage / Unstage | 文件头右侧提供整文件操作；总览提供对应批量操作；历史保持只读 |

### N01：可调列宽与完整列表（P1）

**现状依据**：`view.go:sidebar` 使用固定 184 DIP；`history.go` 已有提交列表与详情之间的拖动分界；`history_panes.go:historyFilesWidth` 将 Files 自动限制在 180–300 DIP，不能独立拖动。历史页通过固定页脚的 `loadMoreHistoryRow` 手动加载；定位栏引用组还有 6 条上限和「更多」入口。

**目标与边界**：

- 保留四列：定位栏｜History 提交列表｜Files｜标签式 Diff。三处分界均可拖动；已有 History 分界继续可用，补齐定位栏与 Files 的分界。
- 拖动结束保存布局，重启恢复；双击分界恢复对应默认宽度，不重置主题、语言或仓库偏好。
- 实施默认：定位栏默认 184 DIP、Files 默认 300 DIP；History 保持现有初始比例。建议最小宽度为定位栏 160、History 260、Files 180、Diff 300 DIP；实现时按实际内容区域统一预算，含所有分界、边框与内边距，避免分别钳制后总宽度溢出。
- 窄窗先限制拖动范围；窗口变窄时只钳制当前有效布局，不能把用户宽窗偏好永久覆盖。隐藏 Files / 详情后，其余列正常填充，恢复时回到合法宽度。
- 从产品上取消「加载更多提交」步骤：打开当前历史范围后自动串行读完全部页，不要求滚到底或点击才能继续。可先显示已读结果和加载进度，不能一次无界阻塞主线程。
- 后端允许继续分页读取；「完整展示」指最终包含当前引用 / 筛选范围内全部记录，不是取消安全输出限制，也不是一次创建全部历史行控件。到达遍历末尾才显示加载完成。
- 切仓库、引用、文件历史或刷新时取消旧遍历，校验 root / 遍历代次；失败保留明确的加载失败状态与重试，不把部分结果标成完整列表。用户主动取消要显示「已停止，历史尚未读完」。
- 定位栏分支 / 远端 / 标签等分组展开后展示全部已读取条目，保留折叠和滚动；不再用前 6 条 + 「更多」跳到另一页替代内容。超过现有后端读取上限的对象必须同步补齐读取，不能只去掉 UI 上限。
- Files 展示当前目标的全部文件，允许滚动和虚拟化，不增加手动加载入口。

**验收**：980 / 1320 / 1600 宽，标准 / 1.15 字号下拖动不遮挡；Files ≥180、Diff ≥300；重启、双击复位和显隐可恢复。2,400 条以上提交自动读取到末尾，前后数量与同一遍历范围的 Git 查询一致，无「加载更多」按钮；不改变筛选 / 引用范围。30 个以上分支展开后可到达最后一项；快速切换无旧列表回流。长历史滚动不为所有不可见行创建控件，并记录实际加载耗时和节点规模。

### N02：工作区虚线图谱（P1）

**现状依据**：`history.go:historyWorktreeRow` 在图谱位置只画空白占位；工作区没有连线。提交图谱由 `graph.go` 管理。

- 工作区有未提交改动时，图谱区域显示与历史节点有形状区别的工作区节点，以及通向实际 HEAD 的虚线；颜色延续对应车道，虚线本身也能表达未提交状态。
- 已提交记录及它们之间的正常连线保持实线，不因工作区有改动而把整条分支画成虚线。
- 干净工作区不画待提交虚线；仅有进行中操作但没有改动时，保留操作提示，不虚构改动。
- 空仓库没有 HEAD，只显示独立工作区标记；Detached HEAD 仍按真实 HEAD SHA 对齐。
- 工作区行固定、历史可滚动：图形必须使用真实行位置和车道。HEAD 因筛选 / 滚动不在可见范围时不连到第一条可见提交；建议使用短虚线延续标记及「基于 HEAD」提示，不跨越空白伪造父子关系。
- 关闭图谱设置时同步隐藏这些图形，文字入口仍可用。

**验收**：干净 / 有改动 / 仅未跟踪 / 空仓库 / Detached HEAD / HEAD 非首条或不可见 / 筛选 / 图谱关闭场景；浅深色下虚线可辨。浏览图谱不修改仓库。

### N03：提交输入边界（P1）

**现状依据**：`workspace.go:commitForm` 使用一个 `TextArea`，占位为「提交标题（必填）\n\n详细说明（可选）」；两段文字没有两个真实字段的边界。

**建议采用**：单行「提交标题（必填）」输入框 + 多行「详细说明（可选）」输入框，字段名可见，placeholder 只作示例 / 提示。另一种成本较低的做法是保留单一多行框、只显示一段占位并在外部解释首行为标题；本计划建议前者，使用户所见与输入结构一致。

- 标题字段不接收 Enter 换行；正文支持多行。标题粘贴含多行内容时必须有明确分配规则，建议首行进标题、后续完整内容进正文，不能静默丢字。
- 提交时按 Git message 的标题、空行、正文组合，不注入 placeholder。打开旧草稿 / Amend 时正确拆分；未编辑的既有完整 message 不因渲染或切标签而被改写空行。
- 标题 / 正文共用一个草稿生命周期；切换 / 关闭文件标签、刷新与隐藏详情均不丢内容。
- ⌘↩ 沿用统一的提交条件；正文 Enter、方向键、撤销 / 重做与复制由当前字段处理。

**验收**：空值、只有空白、仅标题、多段正文、含多行粘贴、旧草稿、Amend 开关；中英文与窄窗下字段边界明确。提交后的完整 message 与用户输入一致。

### N04：去掉重复仓库头像 / 名称（P1）

**现状依据**：截图对应 `view.go:sidebarHeader`，该行同时显示头像和仓库名；窗口标题区域另有仓库名与路径提示。

- 移除定位栏顶部这行头像 + 项目名及其占位，History 成为第一项；不是仅把文字隐藏而保留空白高度。
- 保留仓库识别和切换的一个明确入口：现有标题区域仓库名 / 路径提示与仓库菜单仍可用，同名不同路径可辨。
- 本反馈不扩展为删除应用图标、Dock 图标、窗口系统标题或所有位置的仓库名。

**验收**：定位栏首屏增加可用空间；History 可正常选择 / 聚焦；打开仓库、切换仓库与设置入口不丢失；未打开仓库时也无悬空留白。

### N05：顶部功能、系统菜单与状态反馈规划（P1，部分范围待确认）

**解释边界**：用户说「软件状态栏上面的功能」，本次没有对应完整区域截图，不能直接认定为要求新增底部状态栏或系统托盘。此处同时列出最可能相关的区域，后续实现前确认所指范围；已有功能盘点可以先做。

**已确认的现有顶栏功能**（`view.go:titleBar`）：仓库名称 / 仓库菜单、分支切换、ahead / behind、进行中操作、Fetch / Pull / Push、后台任务进度与取消、刷新。这些都应在下一版规划与操作清单中有落点，不能随着 N04 清理重复信息一起被删掉。

**系统菜单栏建议清单（规划候选，不视为新增功能已获授权）**：

| 菜单 | 建议包含的既有能力 | 一致性要求 |
|---|---|---|
| 应用 | 关于、设置、退出；macOS 标准隐藏项 | 设置聚焦独立设置窗口，退出遵循系统行为 |
| 文件 | 打开仓库、最近仓库、Clone / Init、关闭窗口 | 与当前打开 / 最近仓库入口共用实现；尚无动作的项目不伪装可用 |
| 编辑 | 撤销 / 重做、剪切 / 复制 / 粘贴、全选、查找 | 作用于当前焦点和所属窗口，不能在正文中触发后台 Git 操作 |
| 视图 | History、相关已有页面、显示 / 隐藏详情、恢复默认布局 | 复用现有动作和快捷键；N01 的布局复位范围必须明确 |
| 仓库 | 刷新、Fetch / Pull / Push、分支切换、在文件管理器中显示 | 统一可用条件；远端写入保持明确触发和既有确认规则 |
| 窗口 / 帮助 | 最小化、切换窗口、快捷键说明 / 已有帮助入口 | 优先系统标准行为，不新增无内容的入口 |

**状态反馈建议**：先盘点顶栏、页脚与通知已有状态，形成「状态 → 展示位置 → 可用动作 → 错误恢复」表，至少覆盖无仓库、读取中、读完 / 未读完、Git 忙碌、可取消任务、进行中合并 / 变基、错误与重试。优先复用顶栏任务进度和局部错误，不重复堆叠相同信息。是否需要新增固定底部状态栏，留作此条的待确认项。

**技术前置**：先检查当前 MyGo 版本的原生应用菜单接口与各平台差异。当前找到的 `overlay.MenuItem` 是应用内右键 / 下拉菜单，不能据此宣称系统菜单栏已实现。菜单、工具栏与快捷键共用可用条件与动作，不分别复制写操作逻辑。

**验收**：输出功能盘点与映射表；确认范围内的菜单在无仓库、历史只读、忙碌、弹层、设置窗口聚焦时状态正确，快捷键不重复执行。未接通的能力明确列为后续项，不显示可点击空壳。

### N06：Files 固定三类（P1）

**现状依据**：`workspace.go:workingFileSections` 仅绘制非空分组，顺序为未暂存 → 已暂存 → 未跟踪；干净时整个列表替换为空状态。

固定显示顺序与双语名称：

| 顺序 | 中文 | 英文 | 内容 |
|---|---|---|---|
| 1 | 已修改文件 | Modified Files | 已跟踪但尚未暂存的变化，包含修改 / 删除 / 类型变化等，不仅限状态字母 M |
| 2 | 未跟踪文件 | Untracked Files | 尚未被 Git 跟踪的文件 |
| 3 | 已暂存文件 | Staged Files | 当前 Index 中等待提交的变化 |

- 三个标题始终出现，计数为 0 也不消失；干净时三个 0 与简短干净提示可共存。折叠后标题和计数仍在，折叠状态沿用仓库级偏好。
- 同一已跟踪路径同时有 Index 和工作区变化时，在第 1、3 组各出现一次，分别指向准确版本；计数按各组条目，不把三个数相加误称为唯一文件总数。
- 暂存新文件后从未跟踪迁到已暂存，保留原始路径及版本身份。文件列表和总览分组 / 顺序、键盘导航必须一致。
- 未解决冲突在已修改组内用明确冲突状态标识，不新建第四组，也不能提供普通 Stage / Discard 绕过现有冲突流程；若已有子模块边界阻止操作，继续显示原因。
- 此固定三类仅用于工作区。历史提交 Files 继续是该提交的变更文件列表，不套用工作区暂存分类。

**验收**：三组全空、每组单独非空、同路径双版本、新文件暂存 / 取消暂存、折叠、冲突与子模块；刷新前后顺序固定，活动文件标签不被迁组静默重定向。

### N07：Commit 按钮与作者身份（P1）

**现状依据**：`workspace.go:commitForm` 在不可提交时将按钮 Variant 从 Primary 改成 Text，视觉上退化成一行文字；表单目前只显示提交目标分支和暂存数量，没有本次作者身份。

- Commit / Amend 始终保持明确的按钮容器、尺寸和点击边界；可用时使用主题主色填充，禁用时使用仍可辨识的按钮底色 / 边框与禁用文字，不能变成无容器的普通文本。
- 对零暂存、空标题、读取中、忙碌、身份缺失与冲突等状态显示具体原因；视觉强调不改变真实 `canCommit` 条件，也不自动暂存任何内容。
- 提交框显示 `Author: 名称 <email>`，可选择 / 复制；窄窗允许合理折行，不能把邮箱完全裁掉。未完成读取与读取失败分别表达。
- **身份来源**：读取当前仓库执行环境中 Git 的有效作者身份，考虑仓库 / worktree 配置、includeIf 与环境变量；不能只读全局 `user.name/user.email`，不能拿当前选中历史提交的作者当作下一次提交者。建议通过参数数组调用 `git var GIT_AUTHOR_IDENT` 并解析姓名 / 邮箱；实际写入继续由 Git 决定。
- Amend 默认保留原提交 Author，与当前 Committer 可能不同：显示将保留的作者，必要时辅助说明当前提交者；不得暗中增加 `--reset-author`。普通提交和 Amend 的身份读取分别验证。
- 身份缺失时展示可操作的配置提示，后端继续保留 Git 错误；不自动填写猜测值、不写入用户配置、不增加身份编辑器。
- 读取在后台执行，按 repo root / 代次校验；切仓库和 Amend 状态变化时更新，不逐帧启动 Git。hooks 可能改变最终提交，成功后以实际新提交元信息为准。

**验收**：可用 / 禁用 / Hover / Focus / 执行中、普通 / Amend、局部配置覆盖全局、includeIf / 环境覆盖、身份缺失、同名不同仓库；浅深色下按钮外形明确。使用隔离仓库核对实际 commit Author / Email；失败保留草稿，hooks 仍执行。

### N08：Diff 文件头直接操作（P0 正确性 / P1 展示）

**现状依据**：`detail_overview.go:worktreeOverviewFile` 的头部仅有折叠、路径 / 分组、状态图标和打开标签入口；`history_panes.go:worktreeDiffHeader` 已有部分单文件动作，可抽取共用规则，不能另写一套不一致的目标选择。

| 位置 / 文件状态 | 文件头右侧动作 | 结果与范围 |
|---|---|---|
| 工作区已跟踪、未暂存 | 丢弃 / Discard；暂存 / Stage | 丢弃仅撤销当前文件尚未暂存的变化，保留其 Index；暂存只更新该文件 Index |
| 工作区未跟踪 | 删除文件 / Delete；暂存 / Stage | 删除必须用明确删除文案和确认；暂存只加入该文件 |
| 已暂存 | 取消暂存 / Unstage | 保留工作区内容；有同路径未暂存版本时不能丢失 |
| 历史提交 | 只读辅助动作，如复制路径 / 打开已有入口 | 不显示可写的 Stage / Unstage / Discard |
| 未解决冲突 / 不支持的子模块等 | 现有解决入口或不可操作原因 | 不通过新按钮绕过既有保护 |

- 总览中每个文件块右侧直接展示适用按钮，折叠状态下仍能到达；单文件标签头部使用一致规则。点击操作不能顺带触发折叠或更换活动标签。
- 总览头部提供参考图中的批量动作：Stage All、Discard All；已暂存组提供 Unstage All。按状态 / 分组说明准确范围，空目标禁用。
- 实施默认：Stage All 包含当前工作区可暂存的已修改和未跟踪文件；Unstage All 仅处理当前 Index。冲突或受保护条目存在时不得静默跳过后宣称全部成功，需先提示阻塞原因。
- Discard All 默认仅覆盖已跟踪文件的未暂存变化，不包含删除未跟踪文件或清空 Index；按钮旁或确认框必须写清范围。未跟踪文件删除使用单独、明确的确认流程，不能混入泛称「丢弃全部」。
- 操作目标捕获 root + 原始路径 + 暂存版本；丢弃确认打开后若 root / 状态版本改变，禁止对新上下文执行旧确认。不得使用当前后台游标代替点击文件的身份。
- 文件名、按钮与打开标签入口分配固定优先级；长路径截断并提供完整路径提示，按钮不被挤出可见区。折叠与操作各有可访问名称及键盘焦点。
- 继续复用参数化 Git 调用、literal pathspec、写前校验、忙碌 / 加载 / 冲突守卫和危险操作确认。行级 / 片段动作仍保留，与本次整文件动作明确区分。
- 操作成功后刷新三类计数、总览与准确版本标签；失败局部报错，不清空草稿，不误关仍存在的标签。仓库外部变化后的整文件覆盖风险必须经现有保护或执行前重新确认处理。

**验收**：展开 / 折叠 / 窄窗 / 长路径 / 同名不同目录 / 同路径双版本 / 历史只读 / 冲突 / 空组；逐项核对 HEAD、Index、工作区、未跟踪文件。丢弃 / 删除测试仅用临时 fixture，确认取消完全无写入。全部按钮与快捷键 / 右键行为一致。

## 二、全局约束

- 保留 Go + MyGo 原生 UI，不切 WebView，不更换默认打包入口。
- 980×640 最小窗口继续可用；三种宽度、中英、浅深色、1.15 字号纳入验收。
- 原始路径 + 暂存版本、仓库 root + 不可变 SHA 的身份规则继续有效；异步结果不能串仓库 / 目标。
- 提交只处理 Index，保留 hooks；禁止隐藏 stage-all，禁止绕过冲突 / 子模块保护。
- Git 使用参数数组和 literal pathspec；特殊路径、重命名与部分暂存必须保留。
- 所有写入回归用新建隔离仓库与独立偏好；不对用户实际工作区执行测试性暂存、提交、丢弃或删除。
- 丢弃、删除等危险动作需明确范围并确认；不新增自动破坏或自动远端写入。
- 不把截图中的个人姓名、邮箱写入测试或文档；样例使用虚构身份。
- 既有草稿、独立滚动、标签生命周期、旧设置兼容与快捷键归属不回退。
- 先写有意义的失败回归，再实现；重要写入改动做独立审查，修复后运行完整检查。commit、安装新包和发布另需明确授权。

## 三、实现任务与依赖

以下是待执行任务；测试名为拟新增名称，本文不宣称这些测试已存在。每个任务均按「失败回归 → 实现 → 专项与全量验证」执行。

### Task 1：可调四列布局（N01 的布局部分，关联 N04）

**文件**：修改 `internal/lobedesk/view.go`、`history.go`、`history_panes.go`、`splitter.go`、`settings.go`；新增 `workbench_layout_test.go`，扩展 `splitter_test.go` / `settings_test.go`。

**接口**：沿用 `splitDrag` 与 `splitHandle`；新增定位栏 / Files 分界状态与可缺省的持久化字段，保留现有 `HistorySplit` 含义。布局统一计算各列宽度，不能让新字段改变旧 JSON 的读取结果。

- [x] 先新增 `TestWorkbenchDividersRespectMinimumWidths`、`TestWorkbenchWidthsSurviveRestartAndNarrowing`、`TestWorkbenchDividerResetKeepsOtherPreferences`；验证新布局下三处分界可拖、显隐可恢复与总宽度不溢出，旧布局不支持的新交互先失败。
- [x] 实现分界、持久化、窄窗钳制与局部双击复位；移除侧栏重复头像 / 名称占位，保留顶栏仓库菜单。
- [x] 运行 `go test ./internal/lobedesk -run 'TestWorkbench|TestSplit|TestSettings|TestThePatchKeepsItsWidth' -count=1`，再运行 `make check`。另有原生控件指针拖动 / 显隐恢复测试通过，`make lobe-build` 通过；三种窗口宽度的实际图形拖动留在 Task 7。

**Expected**：专项及全量通过；980 下所有列可用，旧偏好加载无迁移损失。

### Task 2：完整历史与侧栏列表（N01 的完整展示部分）

**文件**：修改 `internal/lobedesk/history.go`、`app.go`、`view.go`、`history_ref.go`；按实际读取上限调整 `internal/gitrepo/history.go` 及引用查询入口；新增 `history_auto_load_test.go`，扩展 `sidebar_test.go`。

**接口**：复用 `loadMoreHistory` 的分页 / 代次语义，改为事件驱动自动推进；需要历史行虚拟化时保留图谱完整模型和全局行索引。不能在绘制中直接启动下一页读取。

- [x] 新增 `TestHistoryLoadsToTraversalEndWithoutMoreButton`、`TestAutomaticHistoryReadRejectsOldContext`、`TestAutomaticHistoryFailureIsNotReportedAsComplete`、`TestSidebarShowsEveryExpandedReference`；用多页 / 多引用 fixture 证明旧限制存在。
- [x] 自动读取到遍历末尾，取消旧任务，处理进度 / 错误 / 取消；去掉手动页脚与引用截断；为大量历史行加入可见区域节点裁剪，确保图谱与键盘揭示正确。
- [x] 运行 `go test ./internal/lobedesk ./internal/gitrepo -run 'TestHistory|TestAutomaticHistory|TestSidebar|TestAReload|TestAFailedRef' -count=1`；对 2,400 条以上历史记录耗时 / 数量 / 可见控件规模，再运行 `make check`。

- [x] 边界复核补项：按文件历史改用 `FileHistoryPage`，固定 HEAD / 原始路径，自动续读、取消 / 重试和旧回调隔离；2,405 条跨重命名后端及 UI 回归、2,403 条文件历史原生计数与最早行揭示通过。Git `--follow --skip` 会丢失重命名祖先，实际采用流式跳过旧记录，仅保留当前页；内存有界，遍历成本随页深度增长。刷新入口也已移除旧 200 条读取。

**Expected**：无需点击读取完整范围，主线程可交互，无串页、重复提交或静默截断。

### Task 3：工作区虚线与固定三类 Files（N02、N06）

**文件**：修改 `internal/lobedesk/history.go`、`graph.go`、`workspace.go`、`worktree_overview.go`、`i18n.go`；新增 `worktree_graph_test.go` / `fixed_file_groups_test.go`，扩展 `graph_test.go` 与文件焦点回归。

**接口**：虚线工作区连接以真实 HEAD SHA 为依据，不注入伪造历史 Commit；三类沿用既有稳定 group key，更新显示顺序与文案。`overviewKeys` 与文件列表使用同一顺序来源。

- [x] 新增 `TestWorkingChangesUseDashedHeadConnection`、`TestHiddenHeadDoesNotConnectToFirstVisibleCommit`、`TestWorkspaceAlwaysShowsThreeFileGroups`、`TestOverviewAndFilesShareGroupOrder`；先观察空组消失 / 缺少虚线等预期失败。
- [x] 实现虚线 / 端点边界；固定三组和计数，保留冲突标识、折叠偏好、双版本身份与键盘导航。
- [x] 运行 `go test ./internal/lobedesk -run 'TestWorkingChanges|TestHiddenHead|TestWorkspaceAlways|TestOverviewAndFiles|TestGraph|TestRefreshPreservesActive|TestDetailTabsKeepKeyboard' -count=1`，再运行 `make check`。普通 / e2e 构建及原生冒烟通过；截图发现的固定行间距导致虚线回折，新增 `TestWorktreeEdgeDoesNotDoubleBackAcrossPinnedGap` RED→GREEN，完整检查及原生复测通过。截图：`build/detail-tabs-smoke/worktree-groups-graph.png`。

**Expected**：未提交与已提交状态可辨；空组始终保留，历史不出现工作区分类，刷新与折叠不重定向目标。

### Task 4：清晰提交表单、按钮与有效身份（N03、N07）

**文件**：修改 `internal/lobedesk/workspace.go`、`app.go`、`keyboard.go`、`settings_page.go`（若快捷键说明受影响）、`i18n.go`；按需要新增 `commit_form.go`、`internal/gitrepo/identity.go` 及 `identity_test.go`；新增 `commit_form_identity_test.go` / 扩展 `commit_form_amend_test.go`。

**接口**：新增 `Repository.AuthorIdentity(ctx)` 或等价只读接口，返回姓名 / 邮箱与错误；Amend 读取 HEAD 原作者。UI 状态需捕获 root / 请求代次。草稿保留一个权威完整 message，字段映射不能制造两个相互覆盖的来源。

- [x] 新增 `TestCommitFieldsHaveExplicitBoundaries`、`TestCommitPastePreservesMessageBody`、`TestDisabledCommitKeepsButtonShape`、`TestCommitIdentityFollowsEffectiveRepositoryAuthor`、`TestAmendDisplaysPreservedAuthor`；覆盖异步换仓库与旧草稿。
- [x] 实现字段拆分 / 组合、稳定按钮样式、身份读取 / 显示与缺失提示；保留 canCommit、hooks、Index-only 和快捷键守卫。
- [x] 运行 `go test ./internal/lobedesk ./internal/gitrepo -run 'TestCommit|TestAmend|TestDraft|TestAuthorIdentity|TestEveryCopy|TestChineseStrings' -count=1`；用隔离仓库确认真实作者、完整 message、Amend 与 hooks，再运行 `make check`。

**Expected**：表单边界不误导，禁用时仍像按钮，预览作者与实际 Git 语义一致，草稿与消息正文无损。

### Task 5：Diff 文件与批量操作（N08，依赖 Task 3）

**文件**：修改 `internal/lobedesk/detail_overview.go`、`history_panes.go`、`menu.go`、`workspace.go`；必要时新增 `file_actions.go` 共用动作准备与守卫；新增 `diff_file_actions_test.go`，扩展 `overview_lines_test.go`、`detail_refresh_target_test.go`。

**接口**：整文件动作接收不可变目标（root、原始路径、暂存状态及确认所需状态版本），共用现有 Stage / Unstage / Discard 后端。按钮、右键与快捷键调用相同准备逻辑；不借用 `selectedPath` 充当点击目标。

- [x] 新增 `TestOverviewFileActionsKeepExactTarget`、`TestCollapsedFileKeepsItsActions`、`TestHistoricalDiffHasNoWriteActions`、`TestDiscardConfirmationRejectsChangedContext`、`TestBatchActionsDeclareTheirScope`；真实 fixture 断言错误目标不会被修改。
- [x] 实现文件头按钮与明确范围的批量按钮，复用危险确认与状态守卫；刷新后三组、草稿和版本标签协调正确。
- [x] 运行 `go test ./internal/lobedesk ./internal/gitrepo -run 'TestOverviewFile|TestCollapsedFile|TestHistoricalDiff|TestDiscardConfirmation|TestBatchActions|TestRefreshPreservesActive|TestOverviewLineAction|TestStage|TestUnstage|TestDiscard' -count=1`，再运行 `make check`。

**Expected**：操作仅改变授权目标；取消确认无写入；历史无写入口；双版本、冲突和特殊路径不被新入口绕过。

### Task 6：顶部 / 菜单 / 状态能力盘点（N05）

**文件**：规划阶段新增 `docs/workbench-command-map.md`；接通经确认的范围时修改 `internal/lobedesk/view.go`、`shortcuts.go`、`keyboard.go` 和实际应用初始化入口，按需要新增 `application_menu.go` / `application_menu_test.go`。

- [x] 盘点 MyGo 原生菜单能力、现有顶栏 / 快捷键 / 状态反馈，写出功能、位置、动作 ID、启用条件、平台支持与空缺的映射表。见 `docs/workbench-command-map.md`；后续菜单 / 新状态区保持待确认，不视为已实施。
- [ ] 对「状态栏上面的功能」确认具体指向；仅在范围明确后接通候选系统菜单或新增状态区域。盘点完成不等于候选菜单已实现。
- [ ] 对确认范围先新增 `TestApplicationMenuSharesActionAvailability`、`TestMenuEditingUsesFocusedWindow`、`TestBusyStateHasConsistentCancelAndRetry` 等行为回归，再接通入口。
- [ ] 运行相关专项与 `make check`，在实际 macOS 菜单和主窗口 / 设置窗口之间验证焦点、快捷键和禁用状态；其他平台明确记录支持程度。

**Expected**：功能清单完整、范围明确；实现部分无空壳菜单、重复快捷键或后台误操作。

### Task 7：原生验收与交付（依赖已确认并实现的 Task 1–6）

**文件**：扩展 `scripts/native-detail-tabs-smoke.py`、`scripts/native-narrow-shots.py`、`docs/e2e-automation.md`、`docs/selection-model.md`、`docs/shortcuts.md`；更新 TODO / DONE 的实际状态。

- [x] 增加真实拖动 / 重启布局、自动完整历史、固定空组、虚线节点、输入 / 作者身份、按钮视觉与准确文件操作的原生流程；所有写入只发生在新 fixture。主历史 2,405 条、文件历史 2,403 条真实计数 / 最早行揭示通过。
- [x] 运行 `make check`、`make lobe-build`、`make lobe-e2e-build`；随后串行执行原生交互与截图脚本。2026-11-16 最终源代码全量测试 / race / vet、两构建及原生冒烟通过。
- [x] 复测 980 / 1320 / 1600 × 640，中英文、浅深色和关键 1.15 字号。额外截图记录 Commit 可用 / 禁用、三组计数 0、双版本文件头操作与长路径。2026-11-16 完整 16 组 × 6 张矩阵一次运行全部通过，实际三处分界拖动 / 偏好 / 重启恢复通过；早先单组偶发丢标签仍保留历史记录，不将本次通过说成已证明其成因。
- [x] 做一次独立代码审查，重点审查文件写入和历史遍历生命周期；完整未提交快照在独立 worktree 审查，无 Critical，发现两项 Important 和一项 Minor。两项重要问题已完成 RED→GREEN 修复：恢复合法 gitlink 整文件暂存 / 取消暂存并保留子模块丢弃保护；主 / 文件历史刷新失败后保留未完成状态与局部重试。修复后的最终全量测试 / race / vet、普通 / e2e 构建及全部原生冒烟均通过，结果见执行记录。完整显示矩阵和分栏重启验证在这两项守卫 / 失败状态修复前通过；修复未改布局代码，未重复矩阵。轻微项保留：缩窄详情区后活动标签可能在屏外，可横向滚动或重新激活恢复；因此不宣称所有缩窄场景均自动显露活动标签。审查报告及复现保存在忽略目录的 `final-review-20261116` 工作树，不因未 commit 而清理。
- [ ] 用户明确授权安装后才更新 `/Applications/SuperGit Lobe.app`；先备份可恢复旧包，再核对签名和实际构建。此前的安装授权不自动延伸到未来版本。

**Expected**：证据支持本轮已实现部分，未确认的 N05 或未完成任务仍显式保留，不能用测试通过替代需求完成。

## 四、Review Focus

1. 同路径同时有 staged / unstaged、特殊字符和重命名时，文件按钮与批量操作是否只写正确目标；确认框在切仓库后能否误执行。
2. 全历史自动加载期间切 root / 引用 / 筛选、刷新或取消后，旧页是否回流，部分结果是否被误标为完整；大仓库是否创建所有隐藏行控件。
3. 三处分界、持久化和隐藏列叠加后，总宽度是否超预算；窄窗是否永久破坏宽窗偏好，恢复后活动标签是否仍可到达。
4. Amend 的 Author 与当前 Committer 不同时是否显示错误身份；拆分字段、粘贴和快捷键是否改写消息或把输入交给后台操作。
5. HEAD 不在首行 / 可见范围时虚线是否连错提交；固定三组是否影响冲突保护、双版本身份与键盘焦点；系统菜单是否绕过相同守卫。

## 五、建议执行顺序与完成口径

建议顺序：**Task 1 → Task 3 → Task 4 → Task 5 → Task 2 → Task 6 → Task 7**。先改善当前截图中直接可见的操作体验，再处理完整历史的规模风险；N05 的盘点可提前，新增区域 / 菜单需在范围明确后实施。

规划阶段只完成「整理改动点」；用户随后授权循环实现，以上勾选状态记录实际完成进度，未勾选项继续执行。原计划 T3 的动态读取优先级 / 跨视图缓存、R03 文件列表单一 Tab 停靠点继续保留，不因本计划产生而自动完成。可调分栏是本次用户明确提出的新要求，执行时需把 TODO 中原先待选的分界需求关联到 N01；其余旧产品决策不重新默认批准。
