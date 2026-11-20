# SuperGit UI 开发设计规范

[设计交接入口](README.md) · [布局对比 PDF](SuperGit-Design-Variants.pdf) · [组件合订画稿](SuperGit-Design-Reference.pdf)

<a id="scope"></a>

## 实施范围

以 Go/MyGo 的 `cmd/supergit-lobe` 为首个接入与验证入口。UI 复用 `internal/lobedesk` 的状态、命令与守卫，Git 读写走 `internal/gitrepo`。应用打包入口的切换另见项目 README 的迁移条件。本文的尺寸是 100% 界面字号下的逻辑点，画稿样例使用演示数据。

| 画稿意图 | 当前 Lobe | 编码接入 |
| --- | --- | --- |
| 统一无描边填充按钮 | 多处使用 SDK 的 Default、Primary、Text | 先共用一个按钮样式入口，覆盖各类控件和状态，再迁移页面 |
| 顶栏切换改动、历史、Stash，左侧显示仓库 | 工作树是历史列表首行，侧栏包含当前仓库引用，详情含文件分栏和已有标签 | 首轮复用工作树和提交选择。完成入口、双版本 Diff 与分栏回归后，再落地完整三栏视图和最近仓库列表 |
| 文件按目录组织，部分暂存行显示混合态 | 已修改、未跟踪和已暂存分别呈现 | 先实现显示层分组与单行样式。目录分组如启用，在版本组内展开，继续以原始路径和暂存版本识别文件 |
| 弧形分支、快速预览、线性区间折叠 | 已有真实父提交 DAG、虚拟列表、引用颜色与详细面板 | 先更新绘制和节点。快速预览与线性折叠独立接入，验证筛选、分页与键盘导航 |
| 设置独立窗口和分类导航 | 独立窗口中按卡片纵向滚动 | 改布局并复用已有偏好读写。设置内容、选项与禁用逻辑由现有模型提供 |
| 需处理的错误 Toast 持续显示 | 普通 Toast 默认 5 秒，撤销为 8 秒，长错误可打开详情 | 延续 5／8 秒，给带详情或恢复动作且需处理的错误启用持久显示 |

<a id="foundation"></a>

## 基础规格

### 主题与颜色

新增设计主题以 `SuperGit` 为用户可见配色名称。浅、深两套颜色由同一配色定义提供，主题模式使用「跟随系统／浅色／深色」。在 [themes.go](../../internal/lobedesk/themes.go) 与 [view.go](../../internal/lobedesk/view.go) 的现有主题入口解析，在两个窗口共用。已有保存的配色键保持可解析。下面的语义名称供实现对照，实际 Go 命名与已有主题结构一致即可。

| 角色 | 浅色 | 深色 |
| --- | --- | --- |
| 窗口底色 | `#F3F5F8` | `#13161B` |
| 主要内容面 | `#FFFFFF` | `#191C21` |
| 分组面 | `#F9FAFC` | `#1B1E24` |
| 次级填充面 | `#EAEFF6` | `#232831` |
| 分隔线 | `#DCE3ED` | `#303641` |
| 正文／次要文字 | `#1B2535`／`#4D5B70` | `#ECF0F7`／`#A8B4C7` |
| 行号／弱文字 | `#5B6A80` | `#93A1B8` |
| 仓库侧栏／侧栏选中 | 窗口底色／选中面 | `#111D34`／`#22385F` |
| 侧栏正文／次要文字／分隔线 | 沿用主题正文、次要文字和分隔线，选中项用强调文字 | `#EAF0FB`／`#A9B8D3`／`#2A3A56` |
| 选中面／强调文字／焦点环 | `#E8F0FF`／`#2456BD`／`#2A66DC` | `#213653`／`#8CB4FF`／`#79A5FF` |
| 新增、成功前景／底色 | `#16603A`／`#EAF6EE` | `#8AD5AA`／`#1A3428` |
| 删除、错误前景／底色 | `#9A3346`／`#FAECEF` | `#F2A1AE`／`#39252D` |
| 警告前景／底色 | `#7D521A`／`#FFF2DC` | `#EEC783`／`#3C3321` |
| Toast 内容面 | `#FFFFFF` | `#262D38` |

文本使用实际前景和背景验色。正文和可操作按钮文字以 4.5:1 为目标。状态同时使用文字、符号、形状或勾选，像 `+`、`−`、`M`、`?` 这样的信号始终可辨识。分支配色遵循 [历史线图](#history) 的规则。

### 字体与尺寸

跟随 macOS 系统字体，中文使用苹方回退，Diff、行号、路径和 SHA 使用等宽字体。复用 [fontscale.go](../../internal/lobedesk/fontscale.go) 的字号缩放入口，按现有布局行为测量中英文和 1.15 大字号。

| 项目 | 标准规格 |
| --- | --- |
| 正文、按钮、单行文件 | 12 pt |
| Diff 等宽正文 | 11.5 pt |
| 操作图标／图文间隔 | 14 pt／6 pt |
| 按钮／独立图标按钮 | 29 pt 高／29 × 29 pt |
| 按钮内间距／圆角 | 横向按文案或图标测量，圆角 6 pt |
| 文件行／历史行／Diff 行 | 29／34／23 pt，文件行按字号与实际内容测量 |
| 标准线图主线／键盘焦点环 | 1.7／2 pt |
| 弹窗 | 12 pt 圆角、20 pt 内容边距 |
| Toast | 10 pt 圆角，离窗口边缘和最近固定底部操作区各 16 pt |

保留主窗口现有 980 × 640 最小尺寸和设置窗口 560 × 480 最小尺寸；设置窗口默认 820 × 720。长标题可换行或省略，完整身份通过提示和辅助功能可读。活动详情标签、关闭工具与主要操作在最小尺寸仍可达。

<a id="buttons"></a>

## 按钮与控件

合订第 6–7 页是按钮形态和状态的参照。无描边填充包括文字、图标加文字、纯图标、下拉、拆分与分段。统一字体、高度、圆角和交互反馈。普通为中性灰，主要为绿色，危险为红色。蓝底表示可选控件已选中或菜单已展开。

| 角色 | 浅色默认／悬停／按下 | 深色默认／悬停／按下 | 前景 |
| --- | --- | --- | --- |
| 普通 | `#E8EDF5`／`#DCE5F2`／`#D2DDEE` | `#2C333F`／`#384252`／`#424F62` | 主题正文 |
| 主要 | `#176840`／`#1B7547`／`#125736` | `#196C43`／`#1B7448`／`#145C39` | `#FFFFFF` |
| 危险 | `#A53A47`／`#AF3B4A`／`#8D2D38` | `#A44451`／`#AC4655`／`#923642` | `#FFFFFF` |
| 仓库侧栏内普通操作 | 导航行沿用侧栏底色，悬停使用主题弱填充 | `#1D2C48`／`#263858`／`#304569` | 侧栏正文 |

| 状态 | 行为与样式 |
| --- | --- |
| 默认、悬停、按下 | 使用对应底色，按钮布局与文案位置稳定 |
| 键盘聚焦 | 独立 2 pt 外环，可以与选中或其他可用状态同时出现 |
| 禁用 | 浅色底 `#EFF2F6`、字 `#627084`；深色底 `#252A33`、字 `#8996AA`。保持键鼠禁用与原因文本一致 |
| 加载 | 原宽高内显示旋转指示与进行中文案。仍可使用已有的取消入口，阻止重复触发当前操作 |
| 展开／选中 | 复用选中面与强调文字。菜单返回焦点到触发控件；分段只有一个当前项 |

纯图标按钮提供可访问名称和 Tooltip。拆分按钮的主区域执行当前动作，箭头只打开同一对象的可用菜单。菜单、按钮、右键与快捷键共用已有命令及禁用原因。

当前 SDK 的 `action.StyleButton` 和 `action.Button` 预设高度是 24／32／40 pt，`action.Default` 带边框。实现时用应用内共用样式将按钮和菜单触发器对齐到本表，复用 MyGo 原生按钮、菜单、下拉与分段语义。将某个页面改用 `action.Filled` 只能覆盖其填充颜色，仍需核对尺寸、焦点及无障碍名称。

<a id="workbench"></a>

## 工作台与 Diff

合订第 2、4 页。仓库栏负责当前仓库及切换入口，顶栏展示分支、页签、同步和进行中状态。改动视图中文件列表与提交表单同列，Diff 占据剩余宽度。历史或分支相关操作沿用已有入口的可达性，包含 Worktrees 和 Submodules。

评审布局时，查看对比稿第 2–4 页。A 保留固定仓库栏，B 给并排 Diff 留出更多宽度，C 把提交范围和提交表单集中到右侧。第 5 页用于核对窄窗中长分支名、同步计数、选区操作条与撤销 Toast 同屏时的可达性。

- 「改动」「历史」可以首先映射到现有 `selectWorktree` 和提交选择；版本标签、独立滚动位置、提交草稿和可隐藏详情继续使用现有状态。
- 改动列表把文件状态与暂存状态分别表达。用空、横杠、勾表示未暂存、部分暂存和已暂存。同一路径的两版可独立查看；目录名称只参与显示，操作仍携带 `fileActionTarget` 的 root、刷新代次、原始 entry 和 staged 版本。
- Diff 头部显示原始路径、比较基准和文件操作。未暂存比较 Index → 工作树，已暂存比较 HEAD → Index。历史补丁保留只读，支持已有浏览操作。
- 选行使用蓝色背景和标记，增删符号及行号保留。选区操作条展示数量和范围，按所处版本提供暂存或取消暂存。开始写入前，选区和文件操作通过现有刷新代次及内容校验。
- 提交按钮显示当前已暂存文件数。标题、正文、作者信息、冲突与忙状态继续使用现有提交守卫，`⌘↵` 与点击走同一命令。工作树中其余改动仍按当前 Git 语义处理。
- 新布局可参考画稿三栏比例。最小窗口优先显示活动标签、关闭入口与底部操作；过长列表和 Diff 使用已有虚拟化和独立滚动。分栏宽度继续按有效空间夹紧，恢复或拖动后核对持久化。

<a id="history"></a>

## 历史线图

合订第 3–4 页。保持现有父 SHA DAG 和历史读取边界，线图与提交行共用定位，弧线连接实际分支。工作区伪节点用虚线，HEAD 用空环，合并用环加内点，分叉用菱形，普通提交用小圆点。缺失、过滤或未加载的父提交继续显示真实边界。

分支轨道与引用标签使用同一语义色，标准样例分别采用蓝色、橙色（浅 `#8C4718`／深 `#F5B171`）和粉色（浅 `#9A3F64`／深 `#F2A2BC`）。继续复用现有 lane 颜色映射、过滤稳定性和扩展轨道。线宽与节点大小由当前偏好决定，1.7 pt 是标准画稿值。

点击提交后在详情区显示提交信息与文件。快速预览为独立的只读后续，展示已加载的作者、SHA、时间和文件统计，悬停或键盘打开均保持当前提交选择。区间折叠为另一项后续，仅针对已加载、无引用、无合并或分叉节点的单父线性区间，折叠计数取实际行数。扩展筛选、分页或键盘导航前先验证折叠与模型行的映射。

<a id="settings"></a>

## 设置窗口

合订第 8–10 页。左栏依次为外观、历史、Git、快捷键和关于，右栏按内容分组，标签与辅助说明靠左，控件靠右。右栏独立滚动，随窗口宽度缩窄；窄窗口的分段控件可换行排布以保持命中目标。

| 分类 | 已有数据与交互 |
| --- | --- |
| 外观 | 模式、配色、语言、字号；切换主题时保持设置页和焦点 |
| 历史 | 作者／时间、提交线图、线宽、节点、详情区域；提交信息与 Diff 至少保留一个 |
| Git | 自动 Fetch、获取间隔、清理失效跟踪分支；在相关控件旁展示启用条件 |
| 快捷键 | 按现有 `shortcutTable` 生成只读清单，按分组展示 |
| 关于 | 应用信息与实际设置文件路径，提供恢复默认设置入口 |

每次改动调用现有持久化路径，以保存结果显示成功或失败。成功状态在保存完成后出现；失败留在设置窗口。恢复默认沿用已有确认和偏好模型，恢复受影响设置并保留当前语言。读写配置以代码解析的路径为准。

<a id="dialogs"></a>

## 弹窗

合订第 11–12 页。普通确认、危险确认与输入表单共用窗口和按钮底形。标题写动作，内容显示当前对象及影响，变更范围内的路径、分支、远端、Stash 或提交标识使用可辨认的文本。底部右对齐放取消与确认。

| 类型 | 初始焦点与执行 |
| --- | --- |
| 普通确认 | 保留已有操作目标与守卫。Push 明示本地分支、目标远端和源提交 |
| 危险确认 | 默认焦点在取消，确认按钮用危险填充色。Stash 同时显示仓库、序号和目标 SHA |
| 表单 | 首个可编辑字段获得焦点。校验失败显示字段原因并禁用提交。创建标签针对当前确认的提交 |
| 恢复设置 | 说明偏好及分栏影响，在设置窗口完成确认和反馈 |

`Esc` 关闭可取消的空闲对话框，关闭后焦点回到原触发处。弹窗期间以既有快捷键守卫和窗口所有权为准。确认回调执行前复用当次操作的目标校验，区分取消、执行中、成功、失败和目标变化。

<a id="feedback"></a>

## Toast、内联提示与进度

合订第 13–16 页。Toast 锚定在发起窗口的右下方，避开该窗口底部固定操作区；同时遵守仓库归属。标题、图标、详情与至多一个操作在紧凑卡片内，关闭按钮使用同一填充风格。正文按实际宽度换行或转入详情。

| 情形 | 表现与期限 |
| --- | --- |
| 信息、成功、非阻塞警告 | 普通 Toast 5 秒 |
| 可撤销的行改动 | 8 秒，倒计时跟随同一计时源；撤销动作绑定对应操作与仓库 |
| 需处理的错误 | 到关闭或打开详情为止，SDK `ToastProps.Timeout` 使用负数 |
| 页面阻塞或字段错误 | 在受影响的页面、分组或字段旁显示原因和恢复入口，状态变化后更新 |
| Tooltip | 提示图标动作、完整路径或禁用原因；禁用原因也通过辅助功能说明暴露 |
| 正在执行 | 保留现有 200 ms 的进行中显示延迟，采用旋转指示和具体操作名；只有真实进度可用时展示百分比 |
| 长错误详情 | 独立可滚动内容，保留原始错误与已有上下文，支持复制详情与关闭 |

复用 `flushNotice` 的队列、仓库校验与发起窗口模型、`toastHost` 的底部预留，以及 `noticeDetailsDialog` 的原始内容。设置窗口关闭前尚未展示的消息沿用现有接管行为；切换仓库后清理旧仓库提示及操作回调。SDK Toast 在指针悬停时暂停计时，撤销进度需取同一个计时状态。

<a id="rollout"></a>

## 实现顺序与代码位置

| 顺序 | 实施内容 | 现有 Go 入口 | 交付检查 |
| --- | --- | --- | --- |
| 1 | 注册浅深配色、统一按钮状态与尺寸 | [themes.go](../../internal/lobedesk/themes.go)、[view.go](../../internal/lobedesk/view.go)、[workspace.go](../../internal/lobedesk/workspace.go) | 普通／主要／危险、全部状态、图标与文字对齐、主窗与设置窗主题一致 |
| 2 | 落地设置分类与弹窗壳 | [settings_window.go](../../internal/lobedesk/settings_window.go)、[settings_page.go](../../internal/lobedesk/settings_page.go)、[settings.go](../../internal/lobedesk/settings.go)、[menu.go](../../internal/lobedesk/menu.go)、[dialog_scope.go](../../internal/lobedesk/dialog_scope.go) | 最小窗可浏览及重置、双窗口焦点、禁用原因、设置保存成功与失败、取消及过期目标 |
| 3 | 统一 Toast、局部提示、进度与错误详情 | [notice_layout.go](../../internal/lobedesk/notice_layout.go)、[app.go](../../internal/lobedesk/app.go)、[error_details.go](../../internal/lobedesk/error_details.go)、[error_feedback.go](../../internal/lobedesk/error_feedback.go) | 5／8 秒与持久错误、窗口及仓库归属、底部动作可点击、原错误可复制 |
| 4 | 工作台入口、分组与 Diff 选区 | [view.go](../../internal/lobedesk/view.go)、[workbench_layout.go](../../internal/lobedesk/workbench_layout.go)、[history_panes.go](../../internal/lobedesk/history_panes.go)、[workspace.go](../../internal/lobedesk/workspace.go)、[file_actions.go](../../internal/lobedesk/file_actions.go)、[lines.go](../../internal/lobedesk/lines.go)、[patch_virtual.go](../../internal/lobedesk/patch_virtual.go) | 文件两版、选择和滚动、标题与草稿、可调整分栏、真实 Index 与工作树结果 |
| 5 | 更新线图与历史呈现 | [history.go](../../internal/lobedesk/history.go)、[graph.go](../../internal/lobedesk/graph.go)、[history_panes.go](../../internal/lobedesk/history_panes.go) | merge、detached HEAD、筛选后的轨道颜色、父节点边界、大历史虚拟化 |

最近仓库列表、完整三栏重排、快速预览与历史折叠分别建立小范围改动，基于上表已有功能逐项验收。对照现有可用动作见 [命令映射](../../docs/workbench-command-map.md)。

<a id="verification"></a>

## 验收

先跑相关 Go 测试，再构建与实际验证本次触及的页面。原生验证使用临时仓库与隔离偏好，步骤见 [原生自动化笔记](../../docs/e2e-automation.md)。

```sh
go test ./internal/lobedesk
make check
make lobe-e2e-build
```

| 改动 | 可直接运行的现有原生脚本 | 验收重点 |
| --- | --- | --- |
| 工作台布局与样式 | `python3 -I scripts/native-narrow-shots.py`；`python3 -I scripts/native-workbench-layout-smoke.py` | 浅深主题、中英文、980／1320／1600 × 640、1.15 字号、长路径与标签露出 |
| 工作区、Diff、历史和标签 | `python3 -I scripts/native-detail-tabs-smoke.py`；`python3 -I scripts/native-local-watch-smoke.py` | 双版本 Diff、快捷键、刷新期间草稿、真实 Git 指纹和视口虚拟化 |
| 设置及多窗弹窗 | `python3 -I scripts/native-lobe-settings-layout-smoke.py --minimum-window`；`python3 -I scripts/native-lobe-settings-layout-smoke.py --persistence`；`python3 -I scripts/native-lobe-window-dialog-smoke.py` | 独立窗口、560 × 480、写失败／恢复、焦点与取消 |
| 同步、失败详情与底部可达性 | `python3 -I scripts/native-lobe-sync-smoke.py`；`python3 -I scripts/native-lobe-stash-smoke.py` | 确认目标、长错误详情、已有 footer 操作与真实写入范围 |

Go 回归可定位到 [workbench_layout_test.go](../../internal/lobedesk/workbench_layout_test.go)、[graph_test.go](../../internal/lobedesk/graph_test.go)、[settings_focus_test.go](../../internal/lobedesk/settings_focus_test.go)、[modal_focus_test.go](../../internal/lobedesk/modal_focus_test.go)、[toast_owner_test.go](../../internal/lobedesk/toast_owner_test.go) 和 [error_details_test.go](../../internal/lobedesk/error_details_test.go)。样式验收覆盖鼠标悬停与按下、键盘焦点以及选中态。保留与本次改动相关的浅深色截图、最小窗口快照及 Git 行为断言。
