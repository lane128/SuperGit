# e2e 自动化验证笔记

`make native-check` 以 `make app` 构建独立的标准 macOS 测试包，运行日常 Git 流程、Stash、冲突和子模块检查，产物放在 `build/native-supergit-*`。`make desktop-native-check` 验证旧桌面回退入口。以下记录适用于标准版与 Lobe 预览版的 `e2e` 构建。

## 启动

```sh
go build -tags e2e -o /tmp/supergit-e2e ./cmd/supergit-lobe
HOME=/tmp/sg-e2e-home MYGO_AUTOMATION=/tmp/sg-e2e.sock \
  SUPERGIT_WINDOW_SIZE=1000x700 /tmp/supergit-e2e /tmp/sg-e2e-repo
# 另一个终端
mygo-automation open --socket /tmp/sg-e2e.sock
```

- 每个会话用独立的 `HOME`、socket 与 fixture 仓库，避免读到本机偏好。
- 需要点击（指针）时，macOS 只把指针事件发给激活窗口。后台用 `&` 直接拉起的进程可能全部点击失效而键盘仍可用；把可执行文件包进最小 `.app` 用 `open` 启动，或确保它是当前前台窗口。同 bundle id 的两个实例会互相抢激活，多仓库验证时给每个实例不同的 `CFBundleIdentifier`，或一次只跑一个。
- 结束时删除临时仓库、`HOME`、socket 与二进制；预览仓库（如 `~/Repos/lazygit`）必须保持未修改。

## 同一仓库的日常操作链

```sh
make lobe-e2e-build
python3 -I scripts/native-lobe-daily-workflow-smoke.py
```

脚本新建独立 workspace、本地 bare remote、peer 和 HOME，关闭自动 Fetch；只允许本地文件协议并校验有效 fetch/push URL。初始化完成后，workspace 的暂存、取消暂存、提交和同步均走原生 UI，脚本拒绝通过 Git add/commit/reset 等绕过这些步骤。每次运行保留独立 `build/lobe-daily-*` 目录中的仓库、脚本快照、二进制哈希、逐步截图、无障碍快照及 Git 状态，便于定位失败；测试结束停止应用进程。

2026-11-17 首轮 macOS 英文浅色验收通过：外部保存／刷新保草稿与焦点；同一文件的 ALPHA、BETA 两处新增行分别暂存，再仅取消 ALPHA；第一次 UI 提交树仅含 BETA，消息／作者／父提交正确；第二次 UI 提交剩余 ALPHA 后工作区干净；Push 取消保持远端／upstream，再确认 Push、Fetch peer 提交和仅快进 Pull。每阶段核对实际 Index blob、HEAD、工作文件和无关文件，最终本地／bare／remote-tracking SHA 一致。首轮证据在测试 worktree `lobe-daily-wuuiwjmo/`；整合脚本并修正首次 Push 等待后，完整原生链重跑也通过，最终证据位于 `build/lobe-daily-mfzrvrxh/`（11 PASS、进程停止 exit 0），日志 `build/alignment-daily-workflow-final.log`，已读取结果并查看最终干净工作区截图。运行二进制 SHA256 `0ecdd6877d678a68a8cd52940aec647e2018b121fb21e62e079411b7ca26520d`。本轮覆盖纯新增行，未替代删除／新增配对、冲突、真实系统目录选择器或 Windows／Linux 原生验收。

## 合并冲突的完成与恢复

```sh
make lobe-e2e-build
python3 -I scripts/native-lobe-merge-resolve-smoke.py
```

脚本创建两个独立的 merge 冲突仓库及 HOME，初始化后被验收的暂存／提交步骤均走原生 UI，不接触用户仓库。第一组外部写入明确解决内容，Mark resolved 后核对精确 stage0、工作字节、HEAD／refs／MERGE_HEAD 和无关二进制保持，再通过 Commit 完成合并并核对双父顺序、完整消息、提交树及 clean 状态。merge 的实际完成入口是 Commit，其他进行中操作才使用 Continue。

第二组只观察残留标记时的当前行为并验证提交前恢复，不提交含标记的文件。Mark resolved 目前会接受当前工作文件并暂存，不扫描标记；提示现明确为“当前文件已暂存并标记为已解决”，不表示已经验证文本。安装的 Sublime 资源尚不能证明其残留标记处理策略，因此本项不作完全对齐结论，也不新增拦截政策。

2026-11-17 整合及文案修复后，两组 macOS 英文浅色原生验收通过，证据 `build/merge-resolve-fnrjop42/`，日志 `build/alignment-conflict-guidance-native.log`；结果已读取，冲突引导与标记暂存后的代表截图已查看，两个进程均停止 exit 0。文件行／差异标题菜单的真实双文件冲突回归另核对仅目标路径被解决，其他冲突保留、无自动合并完成且 Continue 仍拒绝剩余冲突。原生流程未覆盖删除冲突、其他语言／主题／最小窗口或目标平台，现有冲突两侧脚本和后端保存接口测试不能替代这些边界。

## 创建空仓库到首次提交

```sh
make lobe-e2e-build
python3 -I scripts/native-lobe-new-repository-smoke.py
```

脚本只使用独立旧脏仓库、目标目录、哨兵文件与 HOME，预设合成提交身份。空状态和仓库标题菜单分别进入创建表单；取消不创建目录并保留原仓库及完整草稿，相对路径或已有目标拒绝后可编辑重试。通过 UI 创建含中文／空格的新目录后，核对 unborn `main`、空 Index、无 HEAD／refs／remote、clean 状态及正常空历史，不自动生成提交。外部向 owned 新目录写入文本与二进制后，实际文件行回车完成暂存、撤回到空 Index、重新暂存，并通过 UI 创建首次提交；核对零父提交、仅一个提交、完整消息、作者／提交者、准确树与字节、Index、clean 状态及 History／Files／Diff，再从会话菜单返回原脏仓库。

被测目标目录创建、暂存和提交不能通过 Git 命令行代替；初始化只构造旧仓库，之后 Git 命令受只读白名单限制并禁用可选锁。每阶段比较旧 root 的 HEAD／refs／Index／工作字节及哨兵，保留独立 `build/new-repository-native-*` 中的脚本快照、二进制哈希、截图、无障碍快照和 Git 状态。可指定 `--project`、`--artifacts-parent`，失败退出码 1，退出时停止两个 owned 应用进程。

2026-11-17 测试角色首跑 `new-repository-native-nj7rbqgw/` 十项通过，两进程停止 exit 0；协调者读取完整脚本／结果并查看空历史及首次提交的代表截图。整合脚本后的 `build/new-repository-native-dbuzghmi/` 再次通过。修正文案后最终重跑 `build/new-repository-native-to3xr9e0/` 在撤回暂存回车处失败，Index 未达预期；现场已保留、进程停止，正诊断焦点与异步刷新，不能将该最终构建记为完整原生通过。同构建的独立 unborn 局部补测各用一次 Enter 成功暂存／撤回，但未覆盖切根与 watcher 时序。完整脚本将耗时取证移到聚焦前，改用紧邻 Enter 的轻量焦点及就绪快照；重跑 `build/new-repository-native-y10dfl0y/` 在 step 06 新鲜焦点断言失败，尚未发送该步 Enter，两进程停止 exit 0。新增窗口诊断运行 `build/new-repository-native-qn3nmkwd/` 确认聚焦前后原生窗口均无键盘焦点。脚本现以正常 AppKit API 激活自己创建的 child PID，并等待实际窗口焦点；运行 `build/new-repository-native-d7ww7ug3/` 中 API 返回 true，但窗口仍未获得可验证的焦点，故在发送 Enter 前失败并停止进程。当前完整原生验收受此环境焦点问题阻断，停止盲目重跑；最初撤回失败的加载／分发原因仍未完全确定，不将局部成功替代完整链失败。最终完整检查 `bjh6r48cx` 已通过普通测试、race 和 vet，退出码 0，日志 `build/alignment-new-repository-neutral-copy-check.log`；这不替代上述原生失败的诊断。相关确定性回归另覆盖空／相对／已有目录、文件和 dangling symlink、Git 实际失败后修正重试、创建后打开失败说明、busy／queued-success Cancel。以上不代表传输中取消、缺失身份恢复、中文布局、所有窗口主题或 Windows／Linux 原生运行；首次提交 native 使用可用的合成身份。

2026-11-18 独立窗口生命周期诊断：`build/.native-focus/main.go` 包装当前 `lobedesk.Run` 并使用公开 SDK API，不修改生产源码、SDK 或系统权限；在主线程记录窗口 ID、destroyed／visible／minimized、App active／hidden、原生 focused 和 Focus／Blur 事件。每进程仅调用一次目标 `Window.Focus()`，调用前要求窗口存活、可见且未最小化。`build/native-window-focus-k9w5yidq/` 在请求前已经 focused=true，约六秒观察持续为 true；因此不能证明 Focus 恢复了失焦。第二个有区分意义的探针 `build/native-window-focus-kpjh7j1y/` 使用合成身份，经实际 UI 从旧仓库新建中文／空格目录，切到空 History 再回 Changes，各检查点均 focused；启动约 0.276 s 时尚未 active/focused，约 0.298 s 收到正常 Focus 事件，其后未观察到 Blur。两次均未发送文件操作按键，旧仓库指纹／工作字节不变，进程停止 exit 0，最终截图已查看。该结果表明上述有限切根流程本次未复现失焦，不证明原完整链已恢复，也不证明最初 Enter 失败原因。保留全部旧失败；未向生产加入抢焦点逻辑，原完整 New Repository／快捷键原生验收仍开放。

2026-11-18 提交信息快捷键独立原生验收：`build/commit-focus-native-w5axwdho/` 使用当前生产源码 e2e 构建、独立 unborn 仓库／HOME 与合成身份。先确认实际窗口 focused，再从 Stash 发一次 Cmd+9；不点击标题或用 focus 命令替代跳转，随后真实键入 ` marker`，Summary 正确接收、Body 原文保持，Tab 可离开标题，Git HEAD 符号引用／refs／Index／status 和工作字节不变。已读取结果并查看最终截图；进程停止 exit 0。此项覆盖 macOS 英文浅色标准字号 1100×720 的一次真实按键链，不覆盖 Windows／Linux 原生、全部布局或旧 New Repository 完整链，也不证明参考软件的精确焦点行为。

2026-11-18 快速检出原生尝试：当前普通 e2e 二进制 `build/supergit-lobe-quick-checkout-native`、隔离 HOME／合成身份和自建 main/topic 仓库；驱动 `build/.native-focus/quick-checkout.py`，现场 `build/quick-checkout-native-cufnn0ut/`。五秒窗口焦点前提检查未见 focused，在发送任何 CmdB 或填写草稿前停止 exit 0，日志 `build/alignment-quick-checkout-native.log`；不得计作原生成功或快捷键缺陷，未绕过权限或盲目重跑。真实控件测试覆盖 Summary／Body／History 搜索取消回原输入、延时单 Esc、空／禁用结果、完整本地范围和旧状态回调；owned Git 夹具覆盖成功精确 branch／SHA，以及外部 dirty／tip 移动／其他 worktree 占用后的拒绝保留。`build/quick-checkout-captures/` 八组中英文／明暗／窄宽窗截图及边界断言通过，设计角色审图无操作阻断。Windows/Linux 为编译验证，完整检查见 quick-checkout 计划。

2026-11-18 远端快速检出原生尝试：`build/supergit-lobe-remote-checkout-native` 与 `build/.native-focus/remote-quick-checkout.py`，隔离 HOME／合成身份及仅本地路径的自建远端引用。现场 `build/remote-checkout-native-aanrrd_f/` 在五秒自然窗口焦点前提失败，尚未发送 CmdB 或填写草稿，进程停止 exit 0；完整日志 `build/alignment-remote-checkout-native.log` 已读取。不得计为原生成功，未绕过权限或盲目重跑。实际控件及 owned Git 测试覆盖远端命名创建、明确跟踪目标、本地 tip 与 upstream 保持、过期引用／名称／状态拒绝、返回查询与高亮、Tab 和最终原输入恢复。`build/remote-checkout-captures/` 24 组中英文／明暗／窄宽窗／三个步骤渲染和边界断言通过，设计角色检查全部截图无操作阻碍；Windows/Linux 编译通过。完整检查结果见 remote-quick-checkout 计划。

2026-11-18 命令面板→快速检出交接：当前普通 e2e 二进制 `build/supergit-lobe-palette-checkout-native`、驱动 `build/.native-focus/palette-checkout.py`，自建本地仓库／隔离 HOME／合成身份。现场 `build/palette-checkout-native-n1ht4y3x/` 在五秒自然窗口焦点前提失败，尚未填写草稿或发送任何 palette 按键，进程停止 exit 0；完整 `build/alignment-palette-checkout-native.log` 已读。没有原生交接成功证据，未绕过权限或盲目重跑。实际控件使用生产共用的互斥绘制入口，覆盖双 alias、Enter／点击交接、原 Body 继续输入、Esc／背景取消、十项唯一目录与执行守卫、真实 root A→B／epoch／generation；owned Git 验证成功精确目标和 dirty／tip／worktree 拒绝。16 组全目录／检出筛选的中英文明暗窄宽窗截图及边界通过，目录 `build/palette-checkout-captures/`；最终完整检查见 palette-checkout-handoff 计划。下述原九项面板的成功不替代本次交接验收。

2026-11-18 命令面板原生验收：`build/command-palette-native-25zo22ih/` 使用最终普通 e2e 二进制、owned 脏仓库、隔离 HOME 与合成身份，无远端访问。驱动 `build/.native-focus/command-palette.py` 核对 CmdP／CmdShiftP、真实查询输入、底层 Cmd4 隔离、一次 Esc 返回原正文并继续输入、Enter 执行“跳到提交信息”后真实标题获焦；完整草稿、HEAD／refs／Index／status／工作字节均保留，进程停止 exit 0，面板与执行后截图已查看。首次脚本将 ActiveDescendant 辅助功能焦点误作输入焦点；SDK 明确把活动选项报告为焦点，修正为实际查询更新断言。第二次发现重复行 tooltip 先吃 Esc，添加窗口已聚焦／Tab／延时回归并观察 RED 后，只移除命令面板重复名称 tooltip；第三次原生链通过。`build/palette-captures/` 另有中英文、明暗、360×360／900×640 的确定性渲染及控件横向边界、面板高度验证；滚动内容行可在 viewport 内裁切。这些不代表 Windows/Linux 原生或 Sublime GUI 实测，也不替代之前 F7／新仓库焦点失败。

2026-11-18 F7 Pull 原生尝试：`build/pull-shortcut-native-8a86jz1d/` 使用当前普通 e2e 构建、自建 bare／source／peer、独立 HOME 与合成身份，设置只允许 file 传输。驱动 `build/.native-focus/pull-shortcut.py` 在五秒有界窗口焦点前提检查中始终未看到 focused，故在填写草稿或发送 F7 之前失败；owned 进程停止 exit 0，日志 `build/alignment-pull-shortcut-native.log`。没有真实用户远端操作，也没有生产抢焦点修复；此结果只记录原生验收被焦点前提阻断，不证明 F7 成功或失败。Summary／Body／History 搜索的快捷键与本地快进／dirty／分叉拒绝由真实控件和 Git fixture 回归覆盖，详情见 pull-shortcut 计划。

2026-11-18 提交跳转记录后退／前进原生尝试：普通 e2e 二进制 `build/supergit-lobe-history-navigation-native`、驱动 `build/.native-focus/history-navigation.py`，自建两提交、部分暂存／工作文件脏状态、独立 HOME／合成身份。现场 `build/history-navigation-native-bje6hqjh/` 在五秒自然窗口焦点前提失败，尚未填写草稿或发送导航键，进程停止 exit 0；完整日志 `build/alignment-history-navigation-native.log` 已读。没有原生成功证据，未绕过权限或盲目重跑。确定性回归覆盖 A→P→B 浏览轨迹、Back 后新分支截断、SHA＋query 恢复与同帧缓存失效、真实父按钮／Summary／Body／历史搜索输入保护、同 SHA HEAD 同步查询、旧 callback／root／epoch／reload／file／worktree／隐藏 anchor、pending 编辑后清空查询取消及命令面板阻止迟到选择。真实 Git 分叉 merge 验证第二父和子选择的完整往返；420 提交夹具覆盖分页成功后才记账及选择／query／暂停／modal 取消，保持 HEAD／符号 HEAD／refs／Index／status／工作字节与草稿／amend。相关最终回归通过 3.375 s，日志 `build/alignment-history-navigation-final-related.log`；独立测试审查两项问题均已复现 RED 并修复至 GREEN。设置最小 560×480、大字号 1.15 的中英文明暗四图在 `build/history-navigation-captures/`，边界断言通过，已查看英文浅色与中文深色；Windows/Linux 编译通过，最终全量检查见 history-navigation-stack 计划。所有记录规则为 SuperGit 有界适配，不宣称参考 GUI 完整一致。

2026-11-18 子提交导航原生尝试：当前普通 e2e 二进制 `build/supergit-lobe-child-navigation-native`、驱动 `build/.native-focus/child-navigation.py`，自建两提交脏仓库／独立 HOME／合成身份。现场 `build/child-navigation-native-ejqi6uh5/` 在五秒自然窗口焦点前提失败，尚未填写草稿或发送 Alt+Up，进程停止 exit 0；完整日志 `build/alignment-child-navigation-native.log` 已读。没有原生成功证据，未绕过权限或盲目重跑。确定性回归覆盖完整前缀扫描、搜索隐藏子提交、merge 第二父边、真实多 tips 的单提交分页及固定 tips 后新建引用不扩范围、暂停与失败尾页、单／多／零候选、文件历史恢复反馈、真实编辑器保护、箭头／搜索／Enter／Esc／背景取消与原输入继续键入、保留回调的后台快捷键隔离及旧 root／epoch／reload／source／generation／edge 拒绝。owned Git 导航保持 refs／HEAD／Index／文件和草稿，八组中英文明暗窄宽窗截图与边界断言通过（`build/child-navigation-captures/`），已查看窄窗英文浅色和中文深色代表图。Windows/Linux 仅编译验证；完整检查见 child-navigation 计划。参考 GUI 多候选规则仍未观察。

2026-11-18 第二父提交原生验收：`build/second-parent-native-5yodbll3/` 使用当前普通 e2e 构建、自建真实 main/topic merge、部分暂存文件、独立 HOME 和合成身份。先聚焦详情 Summary，Alt+Shift+Down 不改变 merge 选择；再聚焦 merge 行发同一键，核对第二父完整 SHA 及已加载详情／topic.txt 补丁。Enter／Space 保持新选择，再按同键（单父提交）不动作；Cmd9 后草稿保持，完整 HEAD／符号引用／refs／Index／status／working 字节不变。结果与截图已读／查看，进程停止 exit 0。驱动 `build/.native-focus/second-parent-navigation.py`，日志 `build/alignment-second-parent-native.log`。原生覆盖 macOS 英文浅色 1100×720，三父、分页、其余输入焦点和守卫以真实控件／确定性回归补充，不代表参考 GUI 或 Windows/Linux 原生验收。

2026-11-18 第一父提交原生验收：`build/parent-navigation-native-5y_3s5pv/` 使用当前普通 e2e 构建、隔离 HOME／合成身份以及自建两提交、部分暂存夹具。聚焦历史子提交行后发送 Alt+Down，核对父提交选中、完整 SHA 对应详情与 owned.txt 文件加载完成；根提交再次按键保持选择，Cmd9 后原草稿保持。HEAD／符号引用／refs／Index／status／工作字节不变，进程停止 exit 0，最终截图已查看。日志 `build/alignment-parent-navigation-native-ready.log`，驱动 `build/.native-focus/parent-navigation.py`。初次 sx3i38f1 截图仍在读取详情，后续增加实际详情就绪断言后通过。键盘焦点沿用现有行为留在发起行；选择／显露／详情移动到父提交。本次不证明参考 GUI 焦点语义，或原生 merge／分页／错误路径；这些由确定性回归覆盖。完整检查结果见 parent-navigation 计划。

2026-11-18 文件标签快捷键原生验收：`build/file-tabs-native-sdnhbzw5/` 使用当前普通 e2e 二进制、隔离 HOME／unborn 仓库及合成身份；驱动为 `build/.native-focus/file-tabs.py`。先从实际 Open in file tab 按钮打开文件，分别在真实 Summary／Body 聚焦后发送 Ctrl+PageUp/Down、Cmd+PageUp/Down、Cmd+Shift+[/]，每次核对活动标签确实获得焦点；单文件与永久 Commit 首尾循环，Tab 可离开，草稿及 HEAD 符号引用／refs／Index／status／工作字节保持。已读完整结果并检查最终截图，进程停止 exit 0。自然原生窗口焦点是已检查前提，没有激活／Window.Focus 补救。历史搜索框只在真实控件回归中覆盖，此次原生运行不涵盖历史 SHA、溢出标签栏、跨平台或参考 GUI。首次驱动误用 option 角色查找 opener，在发快捷键前停止；根据快照改为实际 Open in file tab 按钮后通过。

2026-11-18 有限 Enter 链路日志：普通模块缓存不支持 overlay，故使用 SDK 临时副本、独立 modfile 与 overlay 构建；生产源码、go.mod／go.sum、模块缓存和普通二进制均不修改。首个日志运行 `build/new-repository-enter-trace-jrcqfg8w/` 在 step 06 因诊断 observer 读取 frame-local `row.Focused()` 而 panic（exit 2），排除出产品验收；移除此读取后，`build/new-repository-enter-trace-7k3m74oh/` 沿完整前序到 step 07，通过一次 Enter 暂存和一次 Enter 撤回，两进程停止 exit 0。SDK 当刻 windowFocused=true，精确路径的行 observer／shortcut 均进入，guard 允许、target current，Git 操作成功后主线程回调触发刷新。两步之间另有一次 watcher 接受重读（loadSeq 4→5），撤回 Enter 时它已进入读取完成回调约 120 ms，loading=false；因此该次 watcher 没有阻挡撤回。目标仍 unborn main、无 refs、Index 为空，工作字节／旧仓库／哨兵不变；结果、完整应用日志、最终 Git 状态和截图已读取／查看。测试角色只读独立核对了此顺序。探针只观察自然焦点，不调用激活或 Window.Focus，在 step 07 截断；此成功不解释原 `to3xr9e0` 失败，也不替代完整链。watcher 日志未记录 stamp 差异，不能确定其变化来源。

2026-11-18 普通构建完整对照：`build/new-repository-baseline-6jzkt62s/` 使用当前无 overlay／无 SDK 日志的 e2e 二进制（SHA256 `fe264fe93b37a58978f5a066fcf4e90a91758719da11771eaac36676453c4b53`），沿原脚本全部阶段通过，共十项 PASS、两进程停止 exit 0。临时驱动只固定该二进制路径，并将 AppKit 激活替换为限时检查自然 native focus；原有新鲜行焦点／Refresh 就绪断言、一次 Enter、Git 状态和工作字节检查全部保留。覆盖 Cancel／相对及已有路径拒绝、中文空格目录创建、空历史、文本暂存／撤回／重新暂存、二进制暂存、首次 UI 提交、History SHA／Files／Diff 及返回原脏仓库；初始提交零父节点、仅 main ref、完整消息／合成身份、精确文本与二进制树、clean 工作区均核对。已读完整日志 `build/alignment-new-repository-baseline-full.log`、提交后与返回原仓库的 Git 证据，已查看提交成功截图。本构建此配置的完整链一次通过；保留旧失败，未改生产焦点行为，也未定位或宣称修复间歇失焦／原 Enter 失败。此结果不代表无焦点时可执行、canonical 脚本的激活路径已恢复或所有语言／主题／平台验收。

## 克隆仓库的失败恢复与切换

```sh
make lobe-e2e-build
python3 -I scripts/native-lobe-clone-smoke.py
```

脚本创建独立脏仓库、含子模块的 source、本地 bare remote、已有目标哨兵文件和 HOME，只允许本地文件协议。空状态与仓库标题菜单分别进入两字段表单，验证取消不创建目录、不改原仓库或完整草稿；已有目标拒绝；不存在的本地来源触发真实 Git 失败并保留输入和完整错误；通过 UI 修改来源与新目标后重试。成功自动打开含中文／空格的目标路径，核对 HEAD、提交树、Index、refs、origin fetch/push 地址、upstream、文本／二进制字节、clean 状态、草稿 reset 及子模块不自动初始化，再从会话菜单返回原脏仓库。

初始化后 Git 命令只读，不能用命令行 clone 代替 UI；每步比较原脏仓库、source、子模块源、bare 和已有目标的指纹。默认保留独立 `build/clone-native-*` 中的脚本快照、二进制哈希、截图、无障碍快照和 Git 状态，失败退出码 1，结束停止两个应用进程；`--project`、`--artifacts-parent` 可指定路径。失败或取消可能留下目标目录，应用不自动删除或覆盖。

2026-11-17 测试角色首跑 `clone-native-93yucvbp/` 的六项原生流程通过，两个进程均停止 exit 0；协调者已读取完整脚本、结果、二进制哈希并查看失败恢复及成功 History／Files 截图。该轮使用 macOS 英文浅色 1100×720，二进制 SHA256 `8a8e8014c48525246b550977995e72d2ebf9429425254e619cc36aae6d23057f`。整合重跑 `build/clone-native-_ewekcu1/` 通过，但截图检查另外发现切回原仓库时短暂显示上一仓库提交正文。补充仅接纳新 snapshot、暂扣详情回调的回归后，在提交 SHA 变化时清空旧正文和读取错误。修复后的最终整合原生链 `build/clone-native-l432e25j/` 再次六项通过，并新增切回后无旧正文断言；日志 `build/alignment-clone-detail-reset-native.log`，两个进程停止 exit 0，结果／哈希及返回原仓库的加载中截图已读取。最终二进制 SHA256 `9342e1d20f8f4b9852eac5d48d867dc871b171966df2ba4aa69cb41a57f581df`。运行中 Escape／Tab／Cancel、晚到成功的取消保护、空源成功及后续打开失败说明另由 UI Tester／真实本地 Git 回归覆盖；原生脚本不代表这些时序边界、全部语言／主题、长错误滚动、远端认证或 Windows／Linux 运行验收。后端沿用每条 Git 命令两分钟超时。

## 聚焦片段标题的键盘部分提交

```sh
make lobe-e2e-build
python3 -I scripts/native-lobe-hunk-enter-smoke.py
```

脚本创建同文件两个分离的删除／新增替换片段及无关脏二进制，明确聚焦原始 `@@` 标题而非悬浮按钮，通过回车键暂存目标、在已暂存侧撤回、重新暂存，再通过 UI 提交。初始化后 Git 只读白名单禁止用命令行绕过暂存／提交；每阶段核对精确 Index blob、HEAD／refs、工作区和无关文件，最终核对提交树仅含目标片段、父节点、完整消息、作者及剩余未暂存状态。补丁刷新后每次重新聚焦目标标题，不要求旧焦点沿用到新补丁。

2026-11-17 最终整合 macOS 英文浅色原生链通过，证据 `build/hunk-enter-g7qjf4oe/`、日志 `build/alignment-hunk-enter-native-final-copy.log`，4 PASS、进程停止 exit 0；结果、二进制身份和最终保留另一片段的截图已读取。二进制 SHA256 `ac118626293fe2bbb853157c154a5f3c9814f6a6bee4fc4a3816788e7a4faa63`。自动回归另覆盖输入框、历史视图、弹窗、选区、加载／读取错误及过期目标；本脚本不代表 Summary overview、所有冲突类型、中文界面或 Windows／Linux 原生验收。

## 父仓库、子模块与 Worktree 连续切换

```sh
make lobe-e2e-build
python3 -I scripts/native-lobe-root-navigation-smoke.py
```

脚本创建独立父仓库、已初始化子模块、linked worktree 与 HOME，两轮沿侧栏进入目标，再用仓库标题菜单中的完整路径返回父仓库。每个目标核对 History 的完整 HEAD、Files、diff 和文件操作入口；每次切换核对草稿按现有 reset 规则清空。初始化后仅允许只读 Git 命令，每步比较三个 checkout 的 HEAD、refs、Index 条目、状态和工作文件字节，不能用重启或命令行打开代替菜单返回。

最近列表只在当前进程保留最多八个成功打开的 canonical checkout root，包含当前仓库；菜单排除当前 root，通常最多七个可切换目标。列表不恢复旧草稿，重启后清空。完整路径作为无障碍名称，并在鼠标或键盘高亮候选时于菜单底部直接换行显示；菜单项不依赖 tooltip。一次 Escape 应关闭菜单并恢复焦点。脚本保留独立 `build/root-navigation-*` 中的证据、脚本快照与二进制哈希，异常退出码为 1；可通过 `--project`、`--artifacts-parent` 指定路径。此流程不等于持久化 Open Recent、系统目录选择器或跨平台原生运行验收。

2026-11-17 最终整合原生链通过，证据 `build/root-navigation-ru2r8bcn/`、日志 `build/alignment-root-navigation-ready-final.log`，两轮全部完成，进程停止 exit 0；结果和最终父仓库截图已读取。脚本在断言文件操作前等待实际 Refresh／Stage file 可用，避免自动刷新期间的瞬时禁用误报；只读 Git 取证设置 `GIT_OPTIONAL_LOCKS=0`。独立 980×640 菜单专项 `root-navigation-3ogrfgts/` 另验九根访问后的容量／淘汰、同 basename 的完整路径、鼠标／键盘准确目标、End 末项可达及一次 Escape 稳定回焦，十一根 Git／文件指纹保持；设计角色已查看首末项路径和关闭后截图。以上均使用二进制 SHA256 `e24fc1c111de82f29feb1c543b888a219e557411e6a69cd932eec278c3babee0`，菜单专项扩展脚本仍保留在测试 worktree，未并入此整合入口。

## 标签式详情的隔离验证

```sh
make lobe-e2e-build
python3 -I scripts/native-detail-tabs-smoke.py
```

脚本自行创建并清理临时仓库与独立偏好，日志和截图保留在 `build/detail-tabs-smoke/`。当前覆盖同名不同路径、暂存 / 未暂存双版本、重复打开、关闭活动 / 后台 / 最后标签、草稿保留、历史完整说明、多文件标签、空提交与快速 SHA 替换；浏览前后比较 HEAD、Index 条目及工作文件字节。2026-11-15 在英文浅色 1320×640 图形会话中通过。后续已补充并通过指定文件暂存、刷新后版本标签协调、草稿保留、单文件超限失败与同目标重试；写入后核对 HEAD、工作区及无关 Index 条目。同一脚本现已通过历史文件方向键保持 SHA、设置窗口隐藏 / 恢复补丁与文件列、整体隐藏 / 恢复详情，以及两个隔离 checkout 之间切仓库的标签清理。最终回归还覆盖折叠未暂存分组后刷新，确认活动标签、实际补丁和 Stage file 操作目标一致。2026-11-16 已补齐文件列表整体单一 Tab 停靠点；脚本分别验证工作区 / 历史 Files 的 Tab 离开与 Shift+Tab 恢复原行，原生控件测试另覆盖左右键可达分组 / 操作及 Space 绑定聚焦版本。上述为 macOS 实际验证，未覆盖其他操作系统或所有读屏行为。

`python3 -I scripts/native-narrow-shots.py` 最新续作重跑已生成 16 组、96 张截图：三种宽度 × 中英文 × 浅深色，另加四组 1.15 大字号；每组摘要 / 多标签 / 空提交 / 工作区 / 禁用提交 / 可提交表单。最新 `native-workbench-layout-smoke.py` 另验证三处分栏指针拖动和重启恢复，并在 Files 双击后核对偏好清除、默认宽度与其他分栏保留，第二次重启仍保持；Git 指纹不变，结果见 `build/workbench-layout-smoke/result.json`。全部组的快照几何核对通过（文件列含分隔线 ≥180 DIP，详情 ≥300 DIP），脚本检查活动标签和关闭入口显露；检查了窄窗大字号和宽窗长正文的代表截图。最新确认保护修复后再次完整重跑通过，日志 `build/lobe-latest-visual-matrix.log`，96 张图像及对应快照在 `build/narrow-shots-82hkcttz/`。脚本现每次创建独立的 `build/narrow-shots-*` 目录，保留以前的证据；工作台矩阵不代表全部页面或菜单交互验收。

原生适配器把 `ui.RoleOption` 输出为 `listItem`，标签选中态输出为 `checked`；无标签文本框可能仅输出 `placeholder=`。解析快照时不能要求每个控件都有引号包围的名称。

## 经典三栏 Changes 首轮验证（2026-11-17）

Changes 新入口将 Files／提交编辑区与 Diff 并排。顶部页面页签与详情页签分别以 `Workbench pages`／`File details` 标记原生 TabList（中文对应“工作台页面”／“文件详情”）；脚本必须按详情 TabList 限定标签计数，不能把页面页签计入文件数。测量 Diff 边界应读取外层横向 Scroll 视口，内部 TabList 会随横向偏移而移动。

本轮 `SUPERGIT_SHOT_PALETTE=supergit SUPERGIT_SHOT_CASES=980-zh-light,980-en-dark-large python3 -I scripts/native-narrow-shots.py` 通过，18 张截图和快照在 `build/narrow-shots-sskgkn9q/`。每组保留原六种场景，新增 Changes 总览／未暂存／已暂存三种；检查草稿、双版本、固定提交按钮及编辑区相对 Diff 视口的位置，并保持 Git 指纹不变。已查看英文深色大字号已暂存截图；完整 16 组新布局矩阵尚未执行。

更新后的 `native-detail-tabs-smoke.py` 从 Changes 执行指定文件暂存、实际提交、失败 hook 后草稿保留、Amend 作者保留以及批量 Stage／Unstage／Discard 确认；原先的版本、窗口设置、切仓库、2405 提交与 2403 文件历史回归全部通过。`native-workbench-layout-smoke.py` 重跑通过历史布局三处分栏拖动／重启恢复及 Files 双击复位；Changes 自身的分栏拖动及不同宽度恢复仍需专项验收。`build/detail-tabs-smoke/changes-staged.png` 捕获到刷新中的禁用状态，不作为提交按钮已就绪的证据；实际写入检查均等待按钮就绪并核对 Git 字节。

后续比较基准增量又通过两组窄窗检查，18 张截图位于 `build/narrow-shots-7b0h48t3/`，新增断言完整原始路径及 Index → Working tree／HEAD → Index（中文为“暂存区 → 工作树”／“HEAD → 暂存区”）。已检查英文深色大字号未暂存截图。`native-workbench-layout-smoke.py` 现另通过 Changes 指针拖动、草稿保留、偏好写入和重启恢复，结果在 `build/workbench-layout-smoke/changes-result.json`；已检查 `changes-divider-restored.png`。Go 回归另外验证 Changes 宽度在最小空间夹紧后恢复首选宽度，不覆盖偏好。

选区条增量已通过完整 `make check`（tests／race／vet）。`python3 -I scripts/native-lobe-long-patch-baseline.py --selection-bar --selected-lines 40 --font-scale 1.15` 的三轮验证均确认精确 Index、工作文件字节不变，以及取消暂存后完整指纹恢复；结果在 `build/lobe-long-patch-yw4wcvzg/`，已检查暂存／取消暂存代表截图。两组窄窗新增选区动作可见性与清除检查，共 22 张截图在 `build/narrow-shots-yfembwc0/`。

后续顶栏合并将 Changes／History／Stash 嵌入标题栏，仓库名及原路径右键菜单移到侧栏。首轮两组原生窄窗 22 张截图通过，证据 `build/narrow-shots-3dc5lko8/`；已查看英文深色大字号已暂存选区截图。设计检查建议补相邻控件不重叠断言，新断言发现长分支按钮越过可收缩容器、与 ahead／behind 计数重叠；修复后专项及完整 Go 测试通过。完整原生矩阵现已通过 16 组、176 张截图，覆盖 980／1320／1600、中英文、浅深色及四组 1.15 字号，证据 `build/narrow-shots-f552yhkq/`；已查看宽窗英文浅色大字号代表截图。该矩阵早于随后头像背景从半透明 FillSecondary 改为不透明 BgElevated 的对比度修复，最终源码重跑已通过两组窄窗 22 张截图（`build/narrow-shots-jl70iaa8/`）、历史／Changes 分栏拖动与重启恢复，以及 detail-tabs 全部十组（包括实际暂存、提交、失败 hook 草稿保留和 Amend 作者保留）；已查看英文深色大字号截图，头像文字对比度恢复。该源码的完整 `make check` 已通过 tests／race／vet，`git diff --check` 同时通过。继续调查默认页面迁移时，发现 Changes 的仓库重置会错误地选中首条历史提交；真实仓库回归先失败，重置逻辑改为按当前页面选择工作区后通过，同时确认 History 原有初始选中行为保留。该上下文修复随后通过完整 `make check`。

默认页面现已迁移到 Changes。新增原生启动断言先在旧默认页复现提交编辑区不可见（`build/narrow-shots-hd78gn6a/`），改动后通过中英文两组窄窗，共 24 张截图位于 `build/narrow-shots-glq8ti3w/`。每组先确认 Changes 选中并显示工作区提交编辑器，保留启动截图，然后 Cmd+1 进入历史检查；已查看英文深色大字号启动截图。历史相关脚本统一先等 Changes 编辑区加载，再显式进入 History；设置和窄窗矩阵按语言使用对应标签。launch-to-history 性能探针因此包含显式页面切换耗时，不能与以前默认 History 的数据当作同一口径比较。

默认页迁移后，detail-tabs 全十组、历史／Changes 分栏拖动重启，以及 operation 脚本中的 cherry-pick／rebase／merge／revert 取消、继续、跳过与精确 Git 状态检查均通过。默认页迁移的完整 tests／race／vet、32 个原生脚本的语法检查和 diff 检查均通过；设置八组矩阵及自动刷新（外部文件保存、Stage／Commit／切分支与子模块 HEAD）通过。随后 Stash 探针在再次保存后立即 Pop 时失败：快照中刷新和 Pop 都仍禁用，证明“Stash saved”提示并不代表后续刷新完成；已将脚本 press 改为限时等待目标操作实际启用，保留生产守卫，语法与 diff 检查通过。修正等待后，Stash 保存／Apply／Pop／Drop、重叠拒绝和真实三方冲突全部通过；三轮 40 行选区暂存／取消暂存精确字节校验通过（`build/lobe-long-patch-z5xx9m4l/`）。完整默认启动矩阵通过 16 组／192 张截图（`build/narrow-shots-j1c6ojfi/`）。

下一增量在文件状态标签之外，用空框／横杠框／勾框表示未暂存／部分暂存／已暂存；部分暂存的两个版本都显示整个路径的混合状态，但操作和说明仍按该行版本提供 Stage file／Unstage file。原始路径、刷新代次、子模块及冲突守卫继续沿用。八种状态／版本组合的实际控件测试先失败后通过，另保留原文件状态标签与窄列边界检查。SDK ActionIcon 的 Title 创建独立 Tooltip 关系，状态说明通过该入口提供，Label 保留短操作名称。该增量已通过两组窄窗状态断言／24 张截图（`build/narrow-shots-kz4gxn0c/`），已查看英文深色大字号启动图，部分暂存两版均显示横杠框；detail-tabs 全十组实际文件操作回归通过。该增量的完整 tests／race／vet 及差异格式检查通过。历史图形和跨平台执行仍待验收。

历史节点增量已通过专项 DAG／配色／几何回归，包含真实 HEAD、重复父 SHA、仅加载一个子提交的筛选结果与全部节点／线宽组合的选择环间距。原生宽 DAG 的三轮输入压力检查通过（32,002 提交、32 个并行分支、33 父提交的 HEAD），覆盖读取期间滚动／搜索、取消／重试和根提交导航，证据 `build/lobe-wide-history-08kbtb1n/`；已查看首轮根提交截图。两组默认节点预设的窄窗检查通过，共 24 张截图在 `build/narrow-shots-uo9evshj/`，已查看中浅／英深大字号 History 摘要图，确认 HEAD 空环、合并中心点与分叉菱形并存。小节点＋粗线原生补查另通过两组／24 张截图（`build/narrow-shots-u7pcwj7z/`），已查看英文深色大字号摘要图：空心 HEAD 与合并中心点仍可区分，当前普通选中节点的外环保留间隙；随后启用 `SUPERGIT_SHOT_GRAPH_IDENTITIES=1`，新增选中 HEAD／合并节点的原生断言与截图，两组／28 张截图通过（`build/narrow-shots-h_bux1_x/`）；已查看中文浅色选中 HEAD 和英文深色大字号选中合并，身份环与选择环之间有可见间隙。这些截图的详情区仍在加载，只用于节点几何验收。该增量完整 `make check` 已通过 tests／race／vet，差异格式检查通过（`build/lobe-graph-check.log`）。窄窗脚本可用 `SUPERGIT_SHOT_GRAPH_LINE=thin|standard|thick`、`SUPERGIT_SHOT_GRAPH_NODE=small|standard|large` 指定隔离偏好；无效值会拒绝运行。

整体验收反馈修复：侧栏新增“管理分支”／“Manage branches”入口，复用原 Branches 页与 Cmd+2；设置保存失败保留 warning，直到用户关闭，避免当前会话已生效被误认为持久化成功。两项均由实际渲染回归先复现再修复：中英入口点击后保持原历史引用；主／设置两窗口真实写入失败在模拟超过 30 秒后仍可见，其他普通／撤销提示的计时保持原样。原生分支入口两组／26 张截图通过（`build/narrow-shots-_uejh95g/`），已查看英文深色大字号管理页。中英文设置持久化探针通过旧配置保留、当前主题保留、恢复保存、确认重置与 Git 指纹检查（`build/lobe-settings-layout-v7vgp2ya/`）；结果的 save-warning／reset-success visual_verified 均为 false，原因仍为设置窗口未获 OS 焦点，因此不据此宣称原生失败提示视觉或 30 秒驻留验收完成。30 秒驻留目前由实际 ToastHost 渲染回归覆盖。本修复的完整 tests／race／vet 与差异格式检查已通过（`build/lobe-acceptance-fixes-check.log`）。

Diff 选行标记增量：单文件和工作区概览的增删选行在原有 92 DIP 行号栏右侧留白绘制勾形，保留行号、代码位置与原按钮命中区域。实际渲染回归覆盖选中增删行、未选上下文行及按钮 Checked 状态。首次原生矩阵虽通过 16 组／192 张截图，但视觉检查发现 SDK Painter.Line 仅绘制水平／垂直线；补充斜线像素断言先复现，再用 StrokePolyline 修复。最终源码两组／24 张截图通过（`build/narrow-shots-dpa2a7o6/`），已查看中文浅色未暂存和英文深色大字号已暂存选行图。三轮 40 条删除行通过选区栏暂存／取消暂存，核对准确 Index、工作字节保留及完整指纹恢复（`build/lobe-long-patch-1o6zvkm0/results.json`）。完整 tests／race／vet 和差异格式检查通过（`build/lobe-diff-marker-diagonal-check.log`）；早期源码的全量检查已中止，不计作最终验证。实际屏幕阅读器播报仍未验收。

设置帮助文案增量：560×480 最小窗口、1.15 倍字号下，英文外观说明需要四行，旧共享帮助组件的三行上限会截断结尾。真实渲染高度对照先复现，再取消行数上限，复用现有设置滚动区；中英完整内容、最小窗口选项／导航／重置可达性回归通过。原生最小窗口八组语言／主题／字号矩阵通过，包含边界、坐标 Reset／取消、Git 与保存偏好保持（`build/lobe-settings-layout-ylf8dwt0/`）；已查看英文深色大字号及中文浅色大字号外观图，完整说明与卡片留白可见。完整 tests／race／vet 及差异格式检查通过（`build/lobe-settings-help-check.log`）。

SuperGit 侧栏配色增量：按 UI-Spec 使用浅／深模式独立深蓝背景、选中面和浅色正文／次要文字，颜色副本仅供侧栏渲染；主内容、菜单和其他配色保留窗口 token。实际渲染背景／选中面断言先复现旧样式，再修复通过；另检查主内容不受影响、其他配色保持、正文／次要文字／快捷键／当前分支勾在侧栏各状态背景上的对比度不低于 4.5:1，并运行既有导航回归。原生中文浅色与英文深色大字号两组／26 张截图通过，覆盖分支管理入口与 Changes／History 浏览（`build/narrow-shots-xrrbtua7/`）；已查看浅色启动和深色 History 摘要图。完整 tests／race／vet 与差异格式检查通过（`build/lobe-sidebar-theme-check.log`）。

Changes 操作按钮增量：选区暂存／取消暂存改用共享绿色主操作，概览批量暂存／取消暂存与文件分组取消暂存迁移到共享普通按钮，概览批量放弃使用危险角色；处理函数、目标身份、禁用说明和确认流程保持原有逻辑。既有选区／批量范围／概览动作／工作区／按钮回归通过。原生浅色中文、深色英文大字号两组／24 张截图通过（`build/narrow-shots-sqjdvnw6/`），已查看概览批量操作和选区按钮。detail-tabs 全十组通过，包含准确文件暂存、草稿、hook 失败、Index-only 提交／Amend、批量范围与放弃取消（`build/lobe-changes-buttons-native.log`）。完整 tests／race／vet 及差异格式检查通过（`build/lobe-changes-buttons-check.log`）。设置分段控件后续已完成：共享 29pt 缩放高度与蓝色选中态，保留 toggleButton 勾选语义、点击当前值不清空／不保存和 Tab＋空格切换一次；新渲染回归先复现旧 26pt／白底选中样式，再通过。设置、持久化、禁用说明和最小窗口回归通过。原生八组最小窗口矩阵通过（`build/lobe-settings-layout-3gbexj5u/`），已查看英文深色和中文浅色大字号图；中英文失败／恢复／重置数据验证通过（`build/lobe-settings-layout-6q2_htxt/results.json`），warning／reset visual flags 仍因 OS 焦点为 false。完整 tests／race／vet 与差异格式检查通过（`build/lobe-settings-segments-check.log`）。Toast 样式后续增量已实现：复用引擎队列／时钟／归属，使用 10 DIP 圆角、16 DIP 与原有底部动作区避让、共享实底关闭／操作按钮；可撤销提示从 ToastParts.Left 读取剩余时间。产品与设计审查发现键盘进入后层 Undo 时不可见，实际渲染回归先复现，再修复为悬停或焦点进入均展开；聚焦超过 10 秒仍可用空格撤销。首轮原生探针因逐帧倒计时使 settle 等到提示过期而失败，FramePending 回归复现后改用 100ms 定时刷新，不改计时来源。悬停暂停、离开后恢复并过期、持久错误不显示倒计时与窗口／仓库归属回归通过。560×480／1.15 字号下，中英／浅深色三条多行 warning 展开几何检查通过；该有限夹具不代表任意长正文均已覆盖。最终源码原生长补丁单行丢弃／Undo 三轮均恢复完整指纹且 Index 保持（`build/lobe-long-patch-_xpjjw3w/results.json`），detail-tabs 十组通过（`build/lobe-toast-final-native.log`）；已查看实际 Undo／进度条截图。Stash 六组及重叠失败／冲突流程通过（`build/lobe-toast-error-native.log`），已查看新的紧凑错误卡与完整错误详情截图。最终完整 tests／race／vet 与差异格式检查通过（`build/lobe-toast-final-check.log`，已读取；lobedesk tests 48.701s、race 312.532s）。运行期间补充的最小窗口几何回归另经四组定向验证通过。Toast 样式与撤销倒计时增量验收完成；设置窗口 OS 焦点与屏幕阅读器实际播报仍未验收。

文件与 Diff 行密度增量：工作区／历史文件列表共享 29pt 缩放行高和 12pt 文件名，单文件／概览／历史补丁共享 23pt 行高，虚拟占位继续消费同一尺寸。实际渲染回归先复现旧 26pt／11pt 文件行和 21pt Diff 行，再通过新尺寸、相邻行位置与大字号中英文字完整性检查；相关选行／概览／大列表／虚拟化回归通过。原生窄窗两组／24 张截图通过（`build/narrow-shots-41s5samx/`），已查看中文浅色未暂存与英文深色大字号已暂存选行图。1.15 字号下三轮 40 行选区栏暂存／取消暂存均核对准确 Index、工作字节保持与完整指纹恢复（`build/lobe-long-patch-b53lxnv0/results.json`，已读取）。detail-tabs 十组通过，包含 2405 提交及 2403 文件历史导航（`build/lobe-file-diff-density-native.log`）。完整 tests／race／vet 与差异格式检查通过（`build/lobe-file-diff-density-check.log`，已读取；lobedesk tests 62.086s、race 312.463s），文件与 Diff 密度增量验收完成。历史 34pt 单行画稿与当前作者／时间双行内容的重排独立验收。

历史行密度增量：保留作者／时间偏好，双行标准 44pt、大字号 50.6pt、行间隔随字号缩放到 2pt；二者均隐藏时采用缩放后的 34pt 单行，工作区伪行同步省略通用的“提交改动”副标题。小字号引用胶囊仍固定 20pt，实际渲染测得 39.6pt 行高底部留白不足，因此双行保底 41.6pt。中英／三字号／四种显隐组合共 24 组渲染检查及历史／线图／导航回归通过。原生粗线／大节点两组／28 张截图通过（`build/narrow-shots-w7aly0u_/`），已查看中文浅色选中 merge 和英文深色大字号选中 HEAD，行内容与图谱中心对齐；这些身份截图中详情仍在加载，不据此宣称 Diff 加载完成。32002 提交、32 条并行分支的大历史三轮通过读取中滚轮／搜索、取消／重试、根导航与夹具状态保持，完成后只构建 31 个行控件（`build/lobe-wide-history-reaygx9t/results.json`，已读取）。detail-tabs 十组及其大历史末行揭示通过（`build/lobe-history-density-native.log`）。完整 tests／race／vet 与差异格式检查通过（`build/lobe-history-density-check.log`，已读取；lobedesk tests 63.982s、race 302.342s）。保留元数据的历史行密度增量验收完成；Windows／Linux 原生、实际辅助功能播报和设置窗口 OS 焦点视觉限制仍未关闭。

整合后三栏布局验收：SuperGit 配色完整十六组矩阵通过，包含 980／1320／1600 宽度、中英、浅深主题及四组 1.15 大字号，共 208 张截图与 208 份快照（`build/lobe-classic-final-matrix.log`、`build/narrow-shots-_332jisa`）。每组检查 Changes 默认入口、双版本文件身份、草稿保持、比较基准、固定提交／选区动作边界及完整 Git 指纹。已查看六张代表图：窄窗中文浅色大字号选区、英文深色大字号已暂存及溢出历史标签、宽窗英文浅色大字号启动、中文分支管理和宽窗深色长摘要。活动标签与关闭、提交和选区操作可见；非活动前置标签在横向视口边缘裁剪。不声称逐张人工检查了全部图片。原生三分栏拖动／重启、Files 双击复位／重启、Changes 拖动／草稿／持久化三组通过（`build/lobe-classic-final-splitters.log`），changes-result.json 为 514→578→578，恢复截图已查看。布局／夹紧／状态专项通过（`build/lobe-classic-layout-focused.log`，0.956s）。本轮仅验收与文档更新，产品源码沿用上轮完整 tests／race／vet 通过版本；macOS 证据不替代 Windows／Linux 原生或 OS 焦点验收。

主界面控件增量：标题栏进行中操作入口、刷新失败横幅及列表错误重试迁移到共享普通按钮，空页面选择仓库使用主操作按钮；保留警告文字、导航、加载／忙碌禁用与横幅独占重试逻辑。既有读取／重试／操作／标题栏／空页面／按钮回归通过（`build/lobe-main-controls-focused-final.log`，6.449s）。原生菜单／读取恢复通过（`build/lobe-main-controls-native.log`），已查看 `build/lobe-menu-access-pc6w_arl/read-failure.png`，980×640 英文浅色下 stale 说明与 Retry 完整可见。操作首轮在外部 Git 变更后、刷新仍禁用时立即定位而失败；已保存失败证据，并让探针等待最多 25 秒后只激活一次，未改变生产加载守卫。重跑全部十一组 cherry-pick／rebase／merge／revert 取消／确认／继续通过，核对准确 HEAD／Index／工作字节及完整提交说明（`build/lobe-main-controls-operation-native-retry.log`，已读取）。已查看最终干净 Changes 与完成 Toast 图；该图不表示进行中标题栏的视觉矩阵已完成。完整 tests／race／vet 与差异格式检查通过（`build/lobe-main-controls-check.log`，已读取；lobedesk tests 60.836s、race 302.428s）。系统目录选择器实际交互和完整视觉矩阵仍待验收。

历史引用筛选／Blame 控件增量：引用触发器、重试、Blame 关闭／重试迁移到共享样式；忽略空白提供明确 toggleButton／Checked 状态。真实仓库鼠标／空格回归先复现旧控件缺少状态，再通过并保留原始文件、版本和内容；相关专项回归通过（`build/lobe-picker-blame-controls-focused.log`，1.936s）。原生首轮因解析器只接受有名称控件而漏掉 placeholder-only 搜索框；轨迹确认界面存在，修正解析器后重跑通过（`build/lobe-picker-blame-controls-native-retry.log`，已读取）。`build/lobe-menu-access-i0qv36lu/results.json` 确认引用搜索／Enter、Blame 鼠标／空格、菜单遍历／取消、过滤／读取恢复及 Git 状态保持。已查看两张引用弹层和 blame-selected.png：英文／浅色／标准字号 980×640 下引用弹层与开关清晰，但 Blame 内容面板底部超出窗口，此项视觉验收未通过，待修复。控件迁移源码的完整 tests／race／vet 与差异格式检查已通过（`build/lobe-picker-blame-controls-check.log`，已读取；lobedesk tests 59.805s、race 302.904s）。随后四组中英／标准与大字号边界回归复现 Blame 源码视口底部达 643pt；原因是 SDK Modal 内边距叠加固定 620pt 子面板。已移除重复内边距，并将子面板限制到窗口尺寸减 40pt，保留默认关闭与阴影。新边界及相关 Blame／引用／按钮回归通过（`build/lobe-blame-layout-focused.log`，2.149s）。最终源码原生边界、引用／Blame／菜单／过滤／恢复及 Git 状态检查通过（`build/lobe-blame-layout-native.log`、`build/lobe-menu-access-y4ftbn6l/results.json`，已读取）。已查看新的 blame-selected.png，英文／浅色／标准字号 980×640 下底部圆角和窗口留白完整，选中与聚焦开关、关闭、源码均可见；该截图不代表完整语言／主题／目标平台矩阵。最终源码完整 tests／race／vet 和差异格式检查通过（`build/lobe-blame-layout-check.log`，已读取；lobedesk tests 60.443s、race 302.194s）。引用／Blame 控件迁移及已复现的最小窗口溢出修复在上述覆盖范围内验收完成，完整视觉矩阵仍开放。

操作与错误恢复控件增量：继续／跳过、工作区补丁／概览／提交说明重试及 Mark resolved 迁移到共享普通按钮，Abort 使用危险角色。原专项测试发现概览错误区虚拟高度仍按旧 24pt Retry 估算，在中英／标准与大字号／220 与 500 宽度下产生滚动范围差异；改为与按钮渲染共用 workbenchButtonBaseHeight 后，千节普通布局与虚拟布局对照及操作／冲突／概览回归通过（`build/lobe-operation-controls-focused-final.log`，5.977s）。原生 cherry-pick／rebase／merge／revert 流程、三组同类同 HEAD 旧操作拒绝和 detail-tabs 十组均通过（`build/lobe-operation-controls-native.log`，已读取）；包含局部补丁失败／重试、2405 提交及 2403 文件历史导航。已查看 `build/lobe-operation-stale-5868n445/continue/refused.png`，操作按钮完整可见、Abort 危险色与替换操作拒绝提示清晰。changes-staged 图在加载期间捕获，不代表最终启用态。完整 tests／race／vet 与差异格式检查通过（`build/lobe-operation-controls-check.log`，已读取；lobedesk tests 61.020s、race 300.826s）。完整语言／主题／最小宽度视觉验收仍开放。

输入弹窗／Reset 控件增量：通用输入弹窗取消／提交迁移到共享普通／主操作按钮，Reset soft／mixed 使用普通角色、hard 使用危险角色，Cancel 保留默认焦点与 Hard Reset 二次确认。相关回归通过（`build/lobe-dialog-controls-focused.log`，7.277s）。原生 soft／mixed／hard／undo 取消及精确状态验证、对应四组 stale 拒绝全部通过（`build/lobe-reset-f_w6893o/results.json`、`build/lobe-reset-rwu33lbt/results.json`，已读取）；双窗口确认归属／冲突请求／取消通过（`build/lobe-window-dialog-x9pqjpcn`），不包含 OS 焦点验证。已查看 hard-confirmation.png 和 main-confirmation.png，危险操作文字完整、Hard Reset 取消按钮有清晰焦点环。迁移后的输入弹窗另经 Stash 保存及六组操作流程通过（`build/lobe-dialog-controls-prompt-native.log`）。完整 tests／race／vet 与差异格式检查通过（`build/lobe-dialog-controls-check.log`，已读取；lobedesk tests 60.837s、race 300.798s）。原生视觉范围为英文／浅色／标准字号／1320×640，完整视觉矩阵仍开放。

历史恢复控件增量：历史／文件历史取消与重试、提交文件列表／补丁重试共六处迁移到共享 29pt 缩放实底按钮，保留加载禁用条件及原处理函数。既有历史与共享按钮回归通过（`build/lobe-history-controls-focused.log`，2.698s）。原生 32002 提交／32 分支三轮通过加载中输入、取消、重试、根导航与夹具状态保持，完成后仅构建 31 个历史行控件（`build/lobe-wide-history-oe_g_tt0/results.json`，已读取）；已查看 reading-1.png，滚动区外 Cancel 完整可见。原生视觉范围为英文／浅色／标准字号／1320×640。完整 tests／race／vet 与差异格式检查通过（`build/lobe-history-controls-check.log`，已读取；lobedesk tests 63.536s、race 299.775s）。本控件增量在上述范围内完成验收。

当前 Toast 组件覆盖核对：提示归属／寿命、长错误详情、设置失败提示的既有回归已从旧 SDK ToastHost 迁移到 app.toastHost，补充两窗口保存失败后超过 30 秒仍可见、点击当前 Close 后消失的验证。定向回归通过（`build/lobe-notice-host-coverage-focused.log`）。同步修正中英外观帮助中的旧说明：文件／提交行随字号调整，Diff 行距固定。最小设置窗口八组矩阵通过（`build/lobe-settings-layout-kvy53dh3/`），已查看英文深色和中文浅色大字号完整帮助文案。完整 tests／race／vet 与差异格式检查通过（`build/lobe-notice-host-coverage-check.log`，已读取；lobedesk tests 60.424s、race 299.427s）。当前组件回归覆盖和帮助文案同步已完成。焦点调查的原生失败／恢复／重置数据检查通过，但 warning／reset visual flags 均仍为 false（`build/lobe-settings-layout-ck9hsdul/results.json`）；通过现有设置入口重新聚焦的尝试也未获得 OS 焦点（`build/lobe-settings-layout-tbomzk0l/results.json`），未把这些结果计为视觉验收，未放松生产焦点或消息归属守卫。

跨平台构建探测：在 macOS arm64 主机上，以 `GOOS=linux|windows GOARCH=amd64 CGO_ENABLED=0` 分别成功构建 Lobe GUI 和 Git 后端测试二进制。`file` 确认 GUI 为 Linux x86-64 ELF／Windows PE32+；Linux GUI 仍有 `/lib64/ld-linux-x86-64.so.2` 动态加载器依赖，不是独立静态分发物。本机未运行这些目标产物，未运行目标平台测试。源码核对发现 `menu.go` 的外部打开仍直接执行 macOS `open`，因此可编译不代表跨平台日常操作已适配，Windows／Linux 原生验收继续开放。

跨平台外部打开适配增量：新增 `external_open.go` 命令构造，macOS 保留 `open`／Finder／Terminal，Windows 使用 Explorer 与 ShellExecuteW 启动独立 cmd，Linux 使用 `xdg-open` 和已安装的 GNOME／Konsole／Xfce／系统终端／xterm。文件定位在 Linux 打开所在目录，不声称选择文件；相应中英文文案按平台映射。Unicode、空格和 shell 元字符路径作为独立参数传递，终端目录通过参数或 cwd 设置，不拼进 shell 源码。命令回归从旧 macOS-only 实现 RED→GREEN，现有选中文件 Enter 实际子进程回归通过。产品小范围只读审查另指出启动失败 stderr 丢失，真实失败脚本先复现仅有退出码；现保留最多 16 KiB 诊断并通过回归。Linux 终端宿主探测测试在 Windows 明确跳过 POSIX fixture，不把该跳过算作 Windows 终端运行验证。macOS 原生两组窄窗／26 张截图与 detail-tabs 全十组通过（`build/narrow-shots-6w513u1r/`、`build/lobe-platform-open-native.log`），已查看英文深色大字号 Changes 启动图；这些原生流程验证工作台未回退，不等于外部应用启动已在所有平台验收。完整 tests／race／vet、Linux／Windows amd64 GUI 交叉编译、Windows lobedesk 测试二进制编译及差异格式检查均通过（`build/lobe-platform-open-check.log`）；目标二进制格式已确认。额外 `SUPERGIT_SHOT_EXTERNAL_OPEN=1` 原生探针仅在隔离子进程 PATH 内放置临时 open 启动器，接收 Unicode／空格／shell 元字符文件名，不实际启动外部应用。首轮两组未收到请求（`build/narrow-shots-61bn64ns/`）：失败树确认焦点在历史文件，而 `canOpenSelectedFile` 的 Enter 仅对工作区启用，属于探针使用了错误入口。第二轮尝试历史文件 Shift+F10 →“用默认应用打开”时菜单未提供该项（`build/narrow-shots-051oolfw/`）；完整失败树确认仅有复制／Finder 定位／文件历史／Blame。据实际菜单改为“在 Finder 中显示”后，第三轮确实收到了 `-R` 和含完整文件名的参数（`build/narrow-shots-jhi3sxel/`），但预期的 `/var/...` 与 Git 返回的规范仓库根 `/private/var/...` 不同而失败。已核对 `gitrepo.Open` 使用 `rev-parse --show-toplevel`，据同一 Git 根构造预期并继续逐字节比较相对文件名后，中英文两组／24 张截图完整通过（`build/narrow-shots-52haodyd/`、`build/lobe-platform-open-canonical-reveal-native.log`）。两份 `*-external-open.json` 已读取，确认实际界面 Reveal 动作收到 `-R` 和完整原始文件名两个参数；生产守卫和菜单不变。默认应用打开由已通过的实际子进程回归覆盖。Windows ShellExecute／Explorer 参数解析、控制台 cwd 与 Linux 终端实际行为仍待目标平台验收。

## 本地自动检测的隔离验证

```sh
make lobe-e2e-build
python3 -I scripts/native-local-watch-smoke.py
```

2026-11-16 在 macOS 原生会话通过。脚本使用新建父仓库、本地子模块源、独立 HOME 与偏好，明确关闭自动 Fetch，全程不按 Refresh。覆盖已修改文件再次保存、未选中 Unicode 文件保存、草稿与输入焦点保留、外部暂存 / 提交 / 切分支和子模块 HEAD 前进；比较 HEAD、Index 与工作字节，确认检测本身没有执行写入。日志、最终快照和截图保存在 `build/local-watch-smoke/`。运行后再执行完整 `native-detail-tabs-smoke.py`，验证自动检测不会破坏既有标签、重试、提交与批量确认流程。

调度的取消、旧仓库 / 代次拒绝、前台 / 后台间隔、错误退避及去重另有 Go 回归；原生脚本不以等待某个固定秒数代替内容断言。当前改动文件内容变化通过元数据观察，保持大小、mtime 和模式的字节改写不保证识别。

## Lobe 同步的隔离验证

```sh
make lobe-e2e-build
python3 -I scripts/native-lobe-sync-smoke.py
```

2026-11-16 在 macOS 原生会话通过。新建工作仓库、本地裸远端和 peer checkout，关闭自动 Fetch，使用独立 HOME / 偏好。验证无 upstream 时 Pull 禁用、Push 取消不写远端和 upstream、首次 Push 设置 upstream、外部推进后 Fetch 只更新跟踪引用、Pull 快进并更新 Unicode / 空格路径；另造真实分叉，验证仅快进 Pull 报错且 HEAD / Index / 工作文件不变、没有启动 merge。日志、最终快照和截图位于 `build/lobe-sync-smoke/`。后续完整脚本新增并通过两个本地裸远端：main 配置 upstream 为 backup/publish/nested，Push 确认显示固定目标，取消两个远端均不变，发布只更新 backup 的指定分支、不创建 backup/main；Fetch 读取双方独立新引用，保留暂存二进制与未暂存 Unicode 文件、HEAD / Index / branch / status。已查看 multi-remote-push.png 和最终双远端引用截图。没有真实网络、凭据、远端选择器、任意远端名规则或 force push 验收。

等待异步 Push 首次创建远端 ref 时用 `for-each-ref` 空结果轮询；不能用缺失 ref 会直接报错的 `rev-parse` 作为轮询条件。第一版脚本因这项等待问题退出，修正后完整运行通过；无生产代码修改。失败截图显示长 Git 原文 toast 占据补丁区域，作为 R04 现有反馈问题的证据保留。

分支 footer 修复后，新增当前原生验收：分叉 Pull 失败后进入 Branches，明确选择本地 main 再等待详情；保持错误提示可见，比较提示下边界与 Copy branch name 上边界，用坐标点击并等待 Copied，核对 HEAD / Index 不变。全部 Go 测试 / race / vet、两种构建通过；首轮因缺少选中行步骤超时，补足后本同步脚本及完整 detail-tabs 重跑通过。`build/lobe-sync-smoke/branch-footer.png` 已查看，提示位于按钮区上方。

## Lobe 引用详情动作的隔离验证

2026-11-17 分支／Worktree 控件增量：新建分支、行内切换、详情操作区、Worktree 新建及表单按钮迁移到共享 29pt 缩放样式；危险操作用危险角色，创建用主操作角色，保留禁用说明、就绪判断、确认与仓库身份校验。相关回归通过（`build/lobe-reference-worktree-controls-focused.log`，7.965s）。原生引用流程与 35 长分支／双远端搜索／聚焦 Enter 全部通过（`build/lobe-reference-controls-ready-native.log`）；Worktree 创建、切换、锁定／新锁定拒绝、取消／脏目录／ignored 二进制保护、干净移除及原分支删除均通过（`build/lobe-worktree-controls-native.log`）。前三次引用探针失败分别暴露成功提示早于刷新就绪、文本等待匹配输入值、列表已出现但操作仍禁用；快照／轨迹保留于 `build/lobe-reference-controls-failed-*`。探针现限时等待真实目标行与启用状态，不重发 Enter，不放松生产守卫。已查看引用最终图和 Worktree ignored 拒绝图；原生视觉范围为英文／浅色／标准字号／1320×640，表单焦点及完整视觉矩阵仍开放。完整 tests／race／vet 与差异格式检查通过（`build/lobe-reference-worktree-controls-check.log`，已读取；lobedesk tests 59.812s、race 304.900s）；最终探针修改另通过 AST 语法与差异格式检查。

```sh
make lobe-e2e-build
python3 -I scripts/native-lobe-reference-smoke.py
```

2026-11-16 在 macOS 原生会话通过。新建临时工作仓库、本地 bare 远端、独立 HOME / Git 配置；通过坐标点击现有详情 footer，覆盖非当前分支 Push 取消 / 发布 / upstream（不 checkout）、分支重命名保留 tip 和跟踪配置、本地分支删除取消 / 确认且远端保留、附注标签推送保留原对象、本地标签删除取消 / 确认且远端保留、远端分支删除取消 / 显式租约确认并清理跟踪引用。每段核对主 HEAD / Index / Unicode 文件字节与干净状态。日志、最终快照和截图在 `build/lobe-reference-smoke/`，最终截图已查看。

引用脚本还覆盖 35 个长 Unicode 分支、双远端、无结果 Enter 不作用于旧行、鼠标仅选择及聚焦 Enter checkout。`SUPERGIT_REFERENCE_WINDOW_SIZE=980x640` 可验证最小窗口；`SUPERGIT_REFERENCE_FONT_SCALE=1.15`、`SUPERGIT_REFERENCE_THEME=dark` 仅调整隔离 fixture 偏好。脚本在最终快照检查顶栏操作边界与顺序、页面标题 / 搜索 / 新分支入口不重叠。修复前快照能使断言失败；980×640 英文浅色、1320 标准浅色及980 大字号深色均已完整通过。大字号截图另发现说明栏高度不足，修复后原生断言核对换行说明不越过引用列表起点；全量 / 构建 / 引用及工作台再次通过，最终大字号截图已查看。这些组合不等同全矩阵验收。

首轮在成功推送后因本地 topic 和 origin/topic 共享标签而选行不明确；按行的本地 / 远端身份区分后完整重跑通过。强制推送与 prune 仅从系统右键菜单进入，当前 MyGo automation 文档明确不支持选择该菜单，因此这两项仍保留专项原生验收；不通过新增产品按钮规避工具限制。stale 对象拒绝已有真实 Git / Go UI 回归，本脚本未新增相应原生竞态场景。

同日扩展并通过无冲突 Merge / Rebase 的 UI 发起：创建双方分叉后从详情 footer 取消 / 确认 Merge，核对两个父提交和两组文件；另建分叉 source / onto 后取消 / 确认 Rebase，核对新父提交、完整源说明与双方文件，保持 main 和干净状态。完整引用脚本重跑通过，最终截图已查看。冲突编辑及 Revert 发起不包含在这两个无冲突场景内。

## Lobe Stash 的隔离验证

2026-11-17 控件样式增量已验证：Save／Apply／Pop／Retry 使用共享 29pt 缩放实底按钮，Drop 使用危险角色；保存触发器保留 menuButton 语义和展开选中态。既有 Stash／共享按钮回归通过，原生保存、Apply、Pop、Drop 取消／确认、重叠拒绝及两组三方冲突全部通过（`build/lobe-stash-controls-native.log`，已读取），已查看 `build/lobe-stash-smoke/compact-error.png` 中的操作按钮与错误恢复入口。完整 tests／race／vet 与差异格式检查通过（`build/lobe-stash-controls-check.log`，已读取；lobedesk tests 59.341s、race 299.191s）。本次原生范围为英文／浅色／标准字号／1320×640，不扩充为其他语言、主题或最小宽度的视觉验收。

```sh
make lobe-e2e-build
python3 -I scripts/native-lobe-stash-smoke.py
```

2026-11-16 在 macOS 原生会话通过。脚本创建新仓库与独立 HOME，关闭自动 Fetch，使用含 Unicode / 空格的 tracked 路径、不同的 Index / 工作区内容及未跟踪路径。验证含未跟踪保存、预览、Apply 恢复精确 Index / 工作字节且保留条目、Drop 取消与确认只影响 stash、Pop 成功恢复并删除条目、重叠 Pop 失败保留 stash 与原有 tracked 内容。日志、最终快照与截图在 `build/lobe-stash-smoke/`。

当前快捷键表中 Stash 是 `cmd+3`，Worktrees 是 `cmd+4`；保存下拉控件的原生角色是 `menuButton`。脚本前两次定位失败已按实际源码与快照修正。首次进入最后的失败场景时，发现「失败保证全部文件不变」的测试假设错误；独立纯 Git fixture 证实 `stash apply --index` 即使因 tracked 重叠退出 1，也可能恢复未跟踪文件。最终脚本精确核对 HEAD / Index / 原有 tracked 字节不变，stash 保留，恢复的未跟踪字节与 stash 一致；不自动删除这些文件或伪造原子回滚。该重叠拒绝场景本身不代表三方冲突。后续脚本另加入 Apply / Pop 真正三方冲突：从未暂存改动保存 stash，再提交同文件不同字节，经 UI 应用后核对冲突标记、双方文本、Index 三阶段、HEAD / stash 保留；打开详情核对恢复说明和 Git stderr。显式外部编辑并 git add 后核对准确 stage0 字节及 HEAD / stash 不变。这不代表应用内冲突编辑验收。首次完整文案等待放在紧凑提示阶段导致超时，第二次误以为 stderr 含 CONFLICT；查看快照 / 收集器源码并用纯 Git 分流 stdout / stderr 后，按实际错误契约修正，完整脚本全部通过。当前错误详情保留 stderr，本机 CONFLICT 在 stdout，stderr 是 Index was not unstashed。

Stash 反馈修复后，2026-11-16 重新执行全部 Go 测试 / race / vet、两种构建、本脚本及完整 detail-tabs，全部通过。新增原生断言确认英文行动说明包含未跟踪文件可能已恢复的提醒、完整 Git 原文仍在且没有旧中文后端提示。已查看最终截图；长 toast 仍遮挡补丁，尚未完成 R04 布局验收。

长正文详情修复后，再次通过全部 Go 测试 / race / vet 和两种构建；脚本首次因新增 `find` 未定义而中断，复用既有元素查找逻辑修正后，本脚本及完整 detail-tabs 重跑通过。新增真实错误 status 高度 <180 DIP、点击 View details 检查完整说明 / Git 原文、Esc 关闭断言。`compact-error.png` 与 `error-details.png` 分别保留紧凑提示和完整详情证据，均已查看。此处不代表分支 footer 或所有 R04 场景通过。

## Lobe Worktree 的隔离验证

```sh
make lobe-e2e-build
python3 -I scripts/native-lobe-worktree-smoke.py
```

2026-11-16 在 macOS 原生会话通过。临时仓库、独立 HOME / 偏好、关闭自动 Fetch；使用实际创建表单填写含空格的目录、新分支与起点。覆盖创建、菜单切换工作树及返回、移除确认取消、dirty 目录拒绝、恢复本次 fixture 自己改动的字节后干净移除。核对主仓库 HEAD / Index / Unicode 文件不变，成功移除目录后先核对分支保留，再等待主工作树就绪，坐标点击 toast 的 Delete；等待 Branch deleted 后核对仅原分支消失、当前仍为 main，主 HEAD / Index / Unicode 文件不变。日志、最终快照和截图在 `build/lobe-worktree-smoke/`。

脚本在 dirty 目录移除拒绝时还等待完整英文提交 / Stash 恢复说明，防止英文标题配中文保护说明回归。共享 clean 错误类型及 Lobe 当前语言映射修复后，全部 Go / race / vet、两种构建、此脚本与完整工作台原生冒烟通过；最终截图已查看。

切换工作树是异步仓库读取：目标分支名在旧列表中也可能已存在，不能仅等分支文字。脚本等待目标工作树行的 `Current` 描述和创建按钮恢复可用，再检查实际根目录切换。首轮因过早断言失败，按加载中的快照修正等待后完整通过；没有修改生产代码。后续扩展完整脚本 8 组通过：已加载锁定时 Remove 菜单项禁用；确认打开后外部新加锁时拒绝且显示目标变化说明；普通 status 干净但有 ignored 二进制文件时拒绝移除，保留精确文件字节 / 子工作树 HEAD / Index / 主工作树状态。已查看 locked-menu.png 与 ignored-refusal.png。ignored 拒绝原因后续已完成当前语言映射，脚本核对完整英文恢复说明；全量 Go / race / vet、两种构建及原生 Worktree / 完整工作台回归通过，更新后的 ignored-refusal.png 已查看。原有后端中文文本保持兼容；系统目录面板回填仍需人工。

## Lobe Submodule 的隔离验证

```sh
make lobe-e2e-build
python3 -I scripts/native-lobe-submodule-smoke.py
```

2026-11-16 在 macOS 原生会话通过。临时父仓库、本地两提交子模块源、Unicode / 空格子目录、独立 HOME / 偏好；环境 `GIT_ALLOW_PROTOCOL=file` 仅限 fixture 与其 App 进程。先在 fixture 中 deinit 自建模块，再原生 Update 初始化；将 child HEAD 移到旧提交后原生 Update 恢复父 Index 目标；dirty child 更新拒绝且 child HEAD / Index / 字节与 parent HEAD / Index 保持；打开子仓库并切到 History 检查独立提交列表。早期日志 / 最终快照与截图在 `build/lobe-submodule-smoke/`。当前脚本每次新建 `build/lobe-submodule-*/` 目录与独立会话，保留旧证据；最新 `build/lobe-submodule-psyuv4uc/results.json` 五组通过，日志 `build/lobe-submodule-root-followup.log`。新增保存切换前真实拒绝的快照 / 截图，在 child Worktrees 和 History 检查所有辅助角色中无父错误标题 / 正文，并比较父子 HEAD / refs / Index 条目 / 状态及 .gitmodules / 脏文件字节；三张前后截图已查看。首次复验遇到自动重读令菜单禁用，`build/lobe-submodule-z8lm00lf/` 留证；菜单探针在读取完成后重取可用目标，未点击禁用动作。归属回归测试通过（`build/lobe-submodule-root-ownership.log`）。这是已完成错误后的 root 切换，不覆盖运行中完成竞态或 OS 焦点。

打开子仓库保留当前页，因此需显式进入 History 后再等待子提交；首轮误等 Worktrees 页不可见的提交而超时，快照已确认 root 正确切换，修正脚本后完整通过。失败快照另暴露子仓库 Worktrees 页误用 `.git/modules/...` 路径。后续已修复主记录路径解析，并加入实际 child checkout / Current / 非元数据目录的原生断言；全量测试 / race / vet、两种构建、子模块 / 工作树 / 完整工作台原生回归均通过。真实仓库回归另覆盖从子模块主目录与 linked checkout 两处读取，防止误把当前 linked 根用作主目录。旧错误 toast 跨 root 残留后续已修复；脚本新增父仓库更新失败提示不进入 child History 的断言，全部 Go 测试 / race / vet、两种构建、子模块与完整工作台原生回归通过。当前最终截图亦已核对旧提示退出。后续 dirty 拒绝原文为中文的语言缺口已修复，后端使用可识别错误保留原 Error 文本，Lobe 显示当前语言恢复说明；真实 checked / 普通更新 × 中英文回归 RED→GREEN。全量 Go / race / vet、两种构建、子模块五组和完整详情标签十组通过，日志 `build/lobe-submodule-dirty-full-check.log` / `build/lobe-submodule-dirty-native.log` / `build/lobe-submodule-dirty-detail.log`。最新结果 `build/lobe-submodule-2vnzuocy/results.json` 增加 `dirty_guidance_in_current_english=true`，完整结果与英文错误 / child History 截图已核对；仍检查跨 root 不残留新旧正文及父子状态不变。ignore 策略、递归发布、真实网络与跨平台不在本次结论内。

## Lobe 进行中操作卡的隔离验证

```sh
make lobe-e2e-build
python3 -I scripts/native-lobe-operation-smoke.py
```

2026-11-16 在 macOS 原生会话通过。新建临时仓库与独立 HOME，关闭自动 Fetch；fixture 通过 Git 制造真实 cherry-pick 冲突，顶栏进行中入口打开操作卡。验证冲突时 Continue 禁用，Abort 取消保留冲突、确认恢复原 HEAD / Index / 字节；Skip 取消与确认；fixture 外部解决冲突并暂存后，原生 Continue 保留完整源提交消息、提交准确 Index、结束 CHERRY_PICK_HEAD 并保持工作文件。日志 / 最终快照与截图在 `build/lobe-operation-smoke/`。

首次脚本按 `change` 后缀定位工作区行，与提交标题重名而失败；依据实际快照改用顶栏精确进行中按钮后完整通过，没有生产代码变化。后续脚本新增 Rebase / Merge / Revert 的独立冲突场景，完整重跑共 11 组通过。Rebase 验证未解决时 Continue 禁用，Abort / Skip 取消与确认、恢复 main 身份，解决后 Continue 保留完整 replay 消息及预期父提交。Merge 不暴露 Continue / Skip，填写标题后未解决仍不能 Commit，Abort 取消 / 确认正确；外部解决后经提交框输入完整说明并提交，核对双父提交与准确 Index / 字节。Revert 验证 Continue 禁用、Abort / Skip 取消与恢复，以及解决后 Continue 的父提交 / 完整生成消息 / Index / 字节。已查看 Merge 和 Revert 最终截图。冲突发起与解决仍由 fixture 显式执行；该脚本不验收系统菜单发起、应用内冲突编辑或残留标记策略。

通用确认框窗口 / root 归属修复后，2026-11-16 重新执行全部 Go 测试 / race / vet、两种构建及本操作卡和完整 detail-tabs 原生脚本，全部通过。跨窗口渲染与后来确认不覆盖原对象另由 UI Tester 回归覆盖；本原生脚本没有新增双窗口焦点切换场景。

## Lobe 设置页的布局矩阵

2026-11-17 分类布局接入后，设置使用外观／历史／Git／快捷键／关于五类导航，右侧独立滚动并保留各分类位置。`native-lobe-settings-layout-smoke.py` 现按原生 `tab` 角色逐类点击，检查各页控件边界，再从关于页打开恢复默认确认。`native-detail-tabs-smoke.py` 从历史分类修改详情显示，双窗口确认脚本从关于分类打开重置。旧的“所有控件同页”假设不再适用。

可用 `SUPERGIT_SETTINGS_PALETTE=supergit` 指定隔离 fixture 的新配色，不修改用户设置；工作台截图脚本对应 `SUPERGIT_SHOT_PALETTE=supergit`。分类版原生最小窗口八组通过，证据 `build/lobe-settings-layout-e08nqnop`；中英文保存失败／恢复／实际重置通过，证据 `build/lobe-settings-layout-z9p2a68c`；双窗口确认归属与取消通过，证据 `build/lobe-window-dialog-a22vbfmu`。已查看英文深色大字号外观／Git 页面，换行分段不越界。写失败／重置通知的 OS 聚焦限制仍按脚本结果记录，不把“未泄漏到主窗口”等同“设置窗口提示已可见”。这些证据只覆盖 macOS 原生会话。

以下记录保留分类迁移前的历史验收范围：

`python3 -I scripts/native-lobe-settings-layout-smoke.py` 使用中英文 × 浅深色 × 标准 / 1.15 字号共 8 个独立仓库 / HOME，打开默认 820×720 设置窗口，核对所有已构建交互控件的横向边界，滚到底部后坐标打开重置确认并坐标取消；Git 指纹和 settings.json 精确不变。每组保存外观 / 底部 / 确认三张截图。结果 `build/lobe-settings-layout-40ndm5ll/results.json`，日志 `build/lobe-settings-native-layout.log`，代表大字号截图已查看。

最小 560×480 的八组几何 / 滚动 / 重置确认由 `TestSettingsChoicesFitMinimumWindowAndResetCanBeReached` 验证；该测试直接渲染实际设置页与确认视图，不调用需要原生事件循环的窗口焦点查询。当前自动化没有调整原生窗口尺寸的入口。后续 `python3 -I scripts/native-lobe-settings-layout-smoke.py --minimum-window` 在新产物目录保存 `.go.txt` 与 overlay 映射，仅将设置窗口初始宽高替换为已有最小常量，使用 `go build -tags e2e -overlay` 构建独立二进制，产品源码 / 普通二进制 / 默认依赖不变。八组原生检查通过，实际滚动视口宽度为 560 DIP，交互控件横向边界、底部 Reset / Cancel 坐标可达、Git / settings 精确不变；日志 `build/lobe-settings-minimum-native.log`、结果 `build/lobe-settings-layout-4qt1xdmo/results.json` 已读，英文深色大字号外观、中文浅色大字号底部与确认截图已查看。同一探针的默认 820×720 路径随后八组全部通过，使用未加 overlay 的普通二进制，日志 `build/lobe-settings-default-followup.log`、结果 `build/lobe-settings-layout-9gdfne3b/results.json` 已读，英文深色大字号代表截图已查看。最小窗口结果覆盖原生最小初始尺寸，不证明实际拖动缩窗或窗口尺寸记忆。设置自动保存，没有独立 Save 按钮；默认模式不实际重置偏好，也不覆盖全部悬停 / 焦点。

`python3 -I scripts/native-lobe-settings-layout-smoke.py --persistence` 另运行中英文两套独立 HOME：将自身 fixture 配置暂存为备份、创建同名目录以阻止原子写入，Dark 修改仍当次生效而旧文件保持；finally 恢复测试路径后 Light 修改持久化成功，实际确认 Reset 核对默认主题 / 图谱 / 自动获取 / 详情 / 字号及语言保留，仓库指纹保持。记录主窗口无保存警告 / 重置成功提示，截图与结果保留于 `build/lobe-settings-layout-terpjqit/`；完整日志 `build/lobe-settings-persistence-followup.log` 已读，两语言及完整详情标签回归通过。相关生产修复全量 Go / race / vet / 构建通过，见 `build/lobe-settings-notice-owner-check.log`。

设置页写入 / 语言修改 / 重置显式指定通知 owner，主窗口偏好写入默认 main，队列归属回归 RED→GREEN。设置窗口只有获得系统键盘焦点才消费提示队列；自动化窗口目标不会激活 OS 焦点，故本轮两种语言 `save_warning_visual_verified=false`、`reset_success_visual_verified=false`。探针仅在检测到设置窗口真实 focused 时等待提示显示；不能将未显示当作显示通过，也不为探针修改通知展示规则。

## Lobe 旧操作确认的隔离验证

`python3 -I scripts/native-lobe-operation-stale-smoke.py` 在两套独立仓库 / HOME 中打开 Abort / Skip 确认，再由 fixture 外部中止原 cherry-pick、用另一个提交制造同类型同 HEAD 的替换冲突。旧确认必须给出目标变化恢复说明，且操作标记 / MERGE_MSG / HEAD / 分支 / refs / 逻辑 Index / 状态 / Unicode 字节 / 无关二进制完整保持。980×640 两组通过，日志 `build/lobe-operation-stale-followup.log`，结果 `build/lobe-operation-stale-gwtngzfl/results.json`，Skip 截图已查看。

首次探针尝试在模态确认打开后点击底层 Refresh，实际辅助树只暴露模态框；移除该不可操作步骤后通过。后续扩展第三组旧 Continue 卡片：原冲突先解决并暂存，使原卡片可继续；保存原控件引用，外部替换为同类型 / 同 HEAD、不同目标并解决后，直接点击原控件，避免获取新快照将测试变为新卡片执行。必须显示精确操作变化恢复提示，完整替换指纹保持；如果本地检测先刷新导致操作成功，该运行不能算旧卡片保护通过。完整三组原生重跑通过，日志 `build/lobe-operation-stale-continue.log`、结果 `build/lobe-operation-stale-9oyofyxg/results.json` 已读，Continue 拒绝截图已查看。原生结论不包含确认期间显式刷新、其他操作类型或外部 Git 原子互斥。正常操作路径仍由原 operation 脚本覆盖。

## Lobe 冲突取边的隔离验证

`python3 -I scripts/native-lobe-conflict-side-smoke.py` 创建两套独立 Merge 冲突仓库 / HOME，通过已有文件菜单 Use ours / Use theirs 核对对应原 Index 第 2 / 3 阶段的精确工作字节与暂存内容。取边保留引用 / MERGE_HEAD / 无关二进制；随后 UI Commit 核对完整消息、双父提交、topic 保留及干净状态。最新完整日志 `build/lobe-conflict-side-followup.log`、结果 `build/lobe-conflict-side-i1wtl9ae/results.json`。首次失败是 disabled 解析假设错误，修正探针后两组通过。后续新增 `--rebase`：`python3 -I scripts/native-lobe-conflict-side-smoke.py --rebase` 在两套真实 Rebase 冲突中核对 Ours 的 stage 2 为变基基底、Theirs 的 stage 3 为重放提交，精确暂存字节与上下文成功提示，然后通过 UI Abort 恢复原 main / refs / 干净状态。最新全量门槛日志 `build/lobe-rebase-side-check.log`，Rebase 结果 `build/lobe-conflict-side-yr7qi2k_/results.json`，同轮 Merge 回归结果 `build/lobe-conflict-side-0_8t0nv6/results.json` 均已核对。该早期样本未覆盖的其他冲突形状及 Continue 见下方后续证据，三方手工编辑和其他操作取边仍未验收。

冲突预览提示接通后，默认文本 Merge / Rebase 原生场景均新增未解决时的真实说明等待，全量 Go / race / vet、两种构建及完整工作台同时通过；日志 `build/lobe-conflict-preview-check.log`，结果 `build/lobe-conflict-side-18fu2lww/results.json` / `build/lobe-conflict-side-m7flbpur/results.json`。新增 `--shape binary` / `--shape deletion` 用于独立验证二进制字节及不存在一侧的暂存删除；扩展场景 Merge / Rebase × binary / deletion × Ours / Theirs 共 8 组已通过，完整日志 `build/lobe-conflict-shapes-check.log`，结果目录依次为 `build/lobe-conflict-side-kd56jv15/`、`build/lobe-conflict-side-q4mgk0gb/`、`build/lobe-conflict-side-7wlhcnt9/`、`build/lobe-conflict-side-xncjf9y6/`。删除不存在一侧核对工作路径和 stage-0 条目均不存在；Rebase Abort 恢复原 main 删除状态。脚本另显露实际提示并要求不处于 offscreen；删除型显露最后一条提示，确保外部处理说明与删除后果同时可见。取边后的 Rebase Abort 等待刷新完成后的可用按钮，不能只等待成功提示。可见性跟进结果 `build/lobe-conflict-side-klszt012/results.json` / `build/lobe-conflict-side-38zl3_qw/results.json` 通过，代表截图已查看。该批只覆盖整文件取边与 Merge Commit / Rebase Abort，不代表外部编辑器调用或三方编辑。

后续 `--rebase-continue` 隐含 Rebase，改为操作卡 Continue 完成。可叠加 `--shape binary` / `--shape deletion`；默认文本。三种形状 × Ours / Theirs 共六组通过：精确阶段字节或删除；Ours 空重放后 main 等于 topic；Theirs 新提交唯一父为 topic，完整原消息、作者 / 邮箱 / 作者日期保留；topic、无关二进制、干净状态和无残留操作均核对。日志 `build/lobe-rebase-side-continue.log` / `build/lobe-rebase-side-continue-shapes.log`，结果目录 `build/lobe-conflict-side-o40jntnx/`、`build/lobe-conflict-side-oziemhcb/`、`build/lobe-conflict-side-_kqq593z/` 已核对，文本 Theirs 最终截图已查看。仅探针变化，沿用已通过完整检查的生产构建。

## Lobe 工作区总览节裁剪

`python3 -I scripts/native-lobe-overview-sections-smoke.py` 在新建千文件脏仓库与独立 HOME 中等待当前渲染区域显示实际补丁，再核对首屏 / 末尾节标题少于 100 个、末尾标题在视口中且对应补丁文本存在、返回首个补丁及完整 Git 指纹不变。裁剪后的辅助树不能证明全部屏外读取已完成。当前首屏 3 个、末尾 6 个；结果 `build/lobe-overview-sections-ny31jbxn/results.json`，日志 `build/lobe-overview-sections-native-followup.log`。只等待标题存在不足以证明补丁就绪；本脚本不测帧时间 / RSS，也不强制制造加载 / 错误 / 空补丁占位状态。后续该三种状态已接入实际宽度 / 字号测量与裁剪：`TestLargeWorktreeOverviewClipsPlaceholderSections` 以中英文 × 标准 / 大字体 × 三状态 × 两列宽共 24 组核对普通布局节高、完整滚动范围、远处滚动 / 返回和实时列宽切换；`TestOverviewViewportReprioritizesPendingReads` 保留首个可见文件读取优先级断言。日志 `build/lobe-placeholder-matrix-followup.log`、全量 Go / race / vet / 构建 `build/lobe-placeholder-full-check.log` 均通过。新构建原生结果 `build/lobe-overview-sections-lv8fimhd/results.json` 仍为首屏 3 / 末尾 6，末尾截图已查看；完整详情标签十组最终通过 `build/lobe-placeholder-detail-ready.log`。详情脚本在提交快捷键前等待实际启用的 Commit 并存 `submit-ready.controls.json`：第二次失败日志证实此前快捷键发送时身份仍读取、Commit 禁用，settle 不保证 Git 读取完成。首次屏内文件点击后意外切换历史 / 页面的原因仍未定位，证据保留于 `build/lobe-placeholder-detail-failure-v1ixczhw/`；不能用后续通过关闭该问题。冲突说明和未知宽度的首帧保持普通布局。

## Lobe 大仓库端到端采样

```sh
make lobe-e2e-build
python3 -I scripts/native-lobe-performance-baseline.py
```

脚本只接收自行生成的 fixture，不接受用户仓库路径。万文件 / 万提交仓库，三次独立 HOME / 应用进程，记录首批 / 全历史、Cancel / Retry、选中说明补丁、End 到初始万文件提交、未变刷新，以及每约 100ms 的应用 / 后代 RSS。检查 HEAD / Index / refs / clean status / 代表文件字节，退出时清理该次生成目录，结果保留在新建 `build/lobe-performance-*/`。

2026-11-16 初测三轮通过，详细范围、硬件和限制见 [性能基线](performance-baseline-2026-11-16.md)。耗时含自动化往返，RSS 是采样最大值、可能重复共享页且漏短命 Git 进程；不表示帧耗时、精确峰值、泄漏或预算已达标。分阶段采样及回归确认历史 Files 行 / 屏外补丁节全部构建；视口裁剪后全量与原生门槛通过，脚本增加万文件仅构建 <100 个 Files 行 / 节标题的断言并保存 oldest 快照，三轮实际为 39 行 / 14 节。顺序采样改善及首轮启动较慢值均记录在性能文档；未覆盖工作区大文件分组及所有联合样本，R16 不关闭。

后续审查修复后，脚本另要求未变刷新保留最早提交完整 SHA；三轮通过。新增单 tracked 文件外部保存 / 恢复，期间不点击 Refresh，验证自动显示唯一改动、补丁文本可访问和恢复干净，同时核对 HEAD / Index / refs 与完整最终指纹。三轮通过，完整日志 `build/lobe-incremental-performance-check.log`；文本可访问不代表正文已在屏内，轮询和自动化耗时计入结果。

联合子模块模式：

```sh
python3 -I scripts/native-lobe-performance-baseline.py --with-submodule
```

在自建父仓库中加入已初始化的本地 1,000 文件子模块，实际父规模为 10,003 tracked 条目 / 12,025 提交。额外观察子文件保存后父 Files 出现唯一子模块改动、恢复后干净，并核对父 / 子完整指纹。2026-11-16 三轮通过，证据 `build/lobe-combined-performance-check.log` / `build/lobe-performance-wzgwdr25/`；只代表单个本地已初始化子模块，不替代所有联合负载。

批量模式：

```sh
python3 -I scripts/native-lobe-performance-baseline.py --with-submodule --batch-files 1000
```

同时改写 1,000 个父 tracked 文件及一个子文件，等待 1,001 条改动，记录工作区 Files 控件数量、快照往返和批量阶段 RSS。核对精确改动路径、父 / 子 HEAD / Index / refs，恢复每个原文件字节后等待干净并核对完整指纹。2026-11-16 三轮通过，结果 `build/lobe-performance-__72hzpm/`，完整日志 `build/lobe-batch-performance-check.log`。实际全部构建 1,001 行；RSS 含历史重读、快照与恢复，不能当作行控件独占内存或帧耗时。

工作区 Files 裁剪后，批量脚本要求该视口仅构建 <100 行；三轮实际 38 行，结果 `build/lobe-performance-1b_nsfq9/`。完整详情标签脚本也通过。首次原生批量恢复曾发生一次 60s 无响应，失败记录保留，重跑未复现；脚本现对 `TimeoutExpired` 保存测量 JSON 并向其自建应用发 SIGQUIT 留 Go 栈后清理进程。该诊断只针对测试拥有的应用，不操作用户应用或仓库；没有把重跑通过视为已定位修复。

### MyGo display link 恢复候选的隔离验证

```sh
python3 -I scripts/check-mygo-display-link-recovery.py
# 可选：应用 Go / race / vet、三轮批量恢复与详情标签，结束后恢复普通 e2e：
python3 -I scripts/check-mygo-display-link-recovery.py --native
```

默认命令只接受当前固定的原版 MyGo，在系统临时目录复制依赖，先加入回归并要求精确 RED，再应用本地候选补丁并通过完整 Darwin 包；项目依赖配置和模块缓存不变。2026-11-16 默认入口已运行通过，证据 `build/mygo-frame-recovery-lx8jxlz7/`。候选应用全量检查、三轮批量恢复及 10 组详情标签组成门槛另已通过 `build/lobe-display-link-candidate-followup.log`；新入口的 `--native` 整条命令尚未运行。此候选未接入默认依赖，不代表默认构建已经修复停帧，详见 [范围和证据](diagnostics/mygo-display-link-recovery.md)。

### 长单文件历史补丁

```sh
python3 -I scripts/native-lobe-long-patch-baseline.py
```

自建临时仓库与两个 fixture 提交，4,000 行全替换，三个独立应用进程 / HOME。检查约 8,000 条增删行的首屏控件数、End / Home 两端文本与行号在视口内、原 HEAD / Index / refs / 字节不变；不接受用户仓库路径。2026-11-16 三轮通过，首屏 49 行，截图与日志保留在新建 `build/lobe-long-patch-*/`。耗时含自动化往返，未采 RSS；不等于工作区长补丁或所有多 hunk 场景已验收。

工作区长补丁模式：

```sh
python3 -I scripts/native-lobe-long-patch-baseline.py --worktree
```

在自建夹具中保留唯一 tracked 文件 4,000 行全替换的未暂存状态，分别浏览 Commit 总览和精确 Unstaged 文件标签；断言 <200 个首屏增删控件、End / Home 端点在视口内、完整 HEAD / Index / refs / status / 字节指纹保持。2026-11-16 修复文件标签全行构建后，工作区与默认历史模式各三轮、完整 Go / 构建 / 详情标签通过，证据 `build/lobe-working-patch-virtual-final.log` / `build/lobe-long-patch-_hsqqjex/`；实际总览 20 个、文件标签 49 个首屏控件。只读浏览，不代表长补丁部分写入或系统文本跨屏拖选验收。

分散 hunk 模式可单独或叠加工作区模式：

```sh
python3 -I scripts/native-lobe-long-patch-baseline.py --multi-hunk
python3 -I scripts/native-lobe-long-patch-baseline.py --multi-hunk --worktree
```

实际核对 4,000 行中九处分散替换、9 个 hunk / 18 增删行，沿用原始 1 / 4,000 行两端导航与指纹断言。2026-11-16 各三轮通过，日志 `build/lobe-multi-hunk-check.log`；只证明分散 hunk 的只读导航，不替代密集 hunk 或部分写入验收。

宽行模式：

```sh
python3 -I scripts/native-lobe-long-patch-baseline.py --wide-lines
python3 -I scripts/native-lobe-long-patch-baseline.py --wide-lines --worktree
```

每行增加宽 ASCII 片段及唯一尾标记。初版仅在工作区文件标签验证横向行尾，历史及工作区各三轮日志 `build/lobe-wide-line-check.log` / `build/lobe-wide-working-line-check.log` 保留。当前共享探针分别覆盖历史 Summary / 历史文件标签，以及工作区总览 / 工作区文件标签：以真实横向 wheel 核对文字位移 >100 DIP、尾端右边界进入 1320 DIP 窗口的补丁视口，保存每种视图截图，再滚回并检查 Home。下方混合 Unicode 样本已完成四视图复验；不把早期 ASCII 结果扩充为当时未测的范围。

Unicode 宽行及字号模式：

```sh
python3 -I scripts/native-lobe-long-patch-baseline.py --worktree --unicode-lines --font-scale 1.15
python3 -I scripts/native-lobe-long-patch-baseline.py --unicode-lines --font-scale 1.15
```

`--unicode-lines` 隐含宽行模式，混合中文、全角和 ASCII；`--font-scale` 支持 1 / 1.15，默认 1。行尾测量至多四次尝试位移；若本地检测重载补丁并重置读位，独立端点恢复流程在 30 秒期限内等待 Refresh 可用 / 无 Loading diff，再从当前树获取并激活 gutter、显式 focus、End，要求末行 gutter 在屏内；返回横向起点后同样恢复当前端点再测 Home。前三次工作区扩展失败经辅助树 / 临时 overlay 确认观察器重读，随后一次失败为使用重读前旧 gutter 标签，均保留诊断证据。此流程验证读取后的导航，不验证重读保留旧选择或读位。辅助树的原文标签与文本 bounds 不足以证明完整尾标记可读，须查看截图是否存在省略号；2026-11-16 首组三轮坐标与指纹通过，但截图发现尾标记省略；新增实际交互行文本分配回归，复现并修正 110 DIP 预留不足（实际需要 138 DIP）。最终三轮全部通过，结果 `build/lobe-long-patch-i6oljx2b/results.json`、完整门槛日志 `build/lobe-unicode-row-final-check.log` 已读，wide-tail-1.png 中完整尾标记无省略号；全量 Go / race / vet、两种构建和完整详情标签原生均通过。后续普通构建四视图验收：工作区 / 历史各三轮全部通过，日志 `build/lobe-unicode-projections-current-target.log`，完整结果 `build/lobe-long-patch-zg00bb_l/results.json` / `build/lobe-long-patch-0iowt7s3/results.json` 均 `state_preserved=true`。已查看第三轮 `wide-worktree_overview-tail-3.png`、`wide-file_tab-tail-3.png`、`wide-history_summary-tail-3.png`、`wide-history_file_tab-tail-3.png`，完整尾标记无省略号。探针扩展期间无产品源码或默认依赖变更。现有 20,000 DIP 宽度上限及任意 Unicode 字形精度不在该样本验收范围。

### 长补丁导航后的单行暂存

```sh
python3 -I scripts/native-lobe-long-patch-baseline.py --stage-line
```

`--stage-line` 隐含 `--worktree`。在自建 8000 条增删行夹具完成总览和文件标签 End/Home 后，重新获取首条删除行 gutter、显式 focus，再经 Shift+F10 的 Stage line 执行。断言 Index 精确为原 after 版本去掉首行、working 字节 / HEAD / refs 不变，只有 long.txt 为 MM。每轮拥有的应用退出后，恢复该自建夹具 Index，以便下一轮和最终完整指纹检查；不接收用户仓库路径。

2026-11-16 三轮通过，结果 `build/lobe-long-patch-l50k34dn/results.json`，日志 `build/lobe-long-patch-partial-stage-followup.log`，第三轮暂存截图已查看。首次直接 press 后发菜单键没有焦点，探针超时，未执行写入；修正为显式 focus 后通过。后续 `python3 -I scripts/native-lobe-long-patch-baseline.py --stage-unstage`（隐含 stage-line / worktree）三轮通过精确 Staged 标签的 Unstage line，UI 操作后、任何外部夹具清理前即核对完整原始指纹恢复；日志 `build/lobe-long-patch-stage-unstage.log`、结果 `build/lobe-long-patch-8k4qu0s7/results.json` 已读，第三轮最终截图已查看。覆盖长补丁首条删除行的暂存 / 取消暂存往返，不代表多选 / 添加行 / 丢弃或任意密集 hunk 已验收。

### 长补丁单添加行往返

```sh
python3 -I scripts/native-lobe-long-patch-baseline.py --stage-unstage --line-kind add
```

`--line-kind add` 隐含暂存 / 工作区模式；先在实际 diff 中重新选择首行，再用 End 到最后一条添加行，避免无选区时 End 作用于历史列表。全替换夹具中未选择的删除行保留为上下文，因此精确 Index 应为原 after 版本 4000 行加所选 working 末行；不是只剩所选添加行。精确 Staged 标签的新行号为 4001，取消暂存后、任何外部清理前核对完整原始指纹。该选项拒绝分散 multi-hunk 夹具，默认删除行行为保留。

2026-11-16 普通固定依赖三轮通过，日志 `build/lobe-added-line-target-followup.log`、结果 `build/lobe-long-patch-_4zuou9o/results.json` 已核对，暂存与取消暂存截图已查看。首次探针在未重新选择当前 diff 行时 End 落到历史末项，导航阶段失败、未写入；修正探针后通过。只验收单添加行，不扩大为 40 行添加选区或丢弃。

### 长补丁单行丢弃与原生撤销

```sh
python3 -I scripts/native-lobe-long-patch-baseline.py --discard-undo
python3 -I scripts/native-lobe-long-patch-baseline.py --discard-undo --line-kind add
```

此模式仅操作脚本生成的全替换夹具，不接受用户仓库路径，拒绝与暂存、多行或分散 hunk 模式组合。完成 8000 条增删行导航后，经行菜单进入既有丢弃确认，确认前核对完整指纹不变；确认后核对精确 working 字节及 HEAD / Index / refs / 状态不变，再点击原生成功提示中的 Undo，要求完整原始指纹恢复，不使用外部还原代替撤销。

普通固定依赖下，单添加行 / 单删除行各三轮通过；日志 `build/lobe-discard-added-line.log`、`build/lobe-discard-deleted-line-pairing.log`，结果 `build/lobe-long-patch-fymi2qa2/results.json`、`build/lobe-long-patch-e_6_gg3_/results.json` 全部核对，代表丢弃与撤销截图已查看。丢弃添加行移除 working 末行；丢弃首条删除行按现有替换配对顺序，在 working 首行后恢复原 after 首行。首次删除行探针错误预期恢复在末尾，源码与快照证明预期错误，修正后通过，不作为产品 RED→GREEN。没有多行丢弃或用户真实删除操作验收。

### 长补丁 40 行跨视口选区（候选验证）

```sh
# 当前普通 e2e 构建；默认依赖下仍有未解决的原生输入推进失败：
python3 -I scripts/native-lobe-long-patch-baseline.py --stage-unstage --selected-lines 40
# 在独立 MyGo 副本应用恢复候选，验证后恢复普通 e2e：
python3 -I scripts/check-mygo-display-link-recovery.py --native-long-range
```

该模式通过 39 次 Shift+Down 选中前 40 条删除行，要求端点显露及菜单精确显示 40 行操作；暂存后验证 Index 等于原版本移除前 40 行，working 字节 / HEAD / refs 保持，取消暂存后、外部清理前必须恢复完整 Git 指纹。为观察短暂成功提示，Unstage 菜单点击不先等待可能耗尽提示显示期的 settle；随后等待原有提示并核对指纹。失败时也保存清理前的 `failure-N.git.json`。不覆盖添加行或丢弃。

2026-11-16 普通固定依赖的状态诊断显示 Staged 选区只到 38 行，末尾按键未进入选区处理，CLI 连续报告窗口未稳定；不能计作通过。隔离帧恢复候选＋应用诊断 overlay 的三轮结果 `build/lobe-long-patch-ja37ajwx/results.json` 全部精确断言通过，第三轮截图显示 Selected lines unstaged 与 Staged Files 0。持久入口 `--native-long-range` 的无诊断 overlay 版本后续也已实际三轮通过：`build/mygo-frame-recovery-l_4ol70i/` 保存精确 RED / 完整 Darwin GREEN / 构建恢复记录，原生结果 `build/lobe-long-patch-u19cb976/results.json` 全部精确断言通过，第三轮成功截图已查看。候选通过不等于默认依赖已修复，详见 [恢复候选证据](diagnostics/mygo-display-link-recovery.md)。

40 行添加选区使用同一持久入口的 `--range-line-kind add`，要求同时指定 `--native-long-range`：

```sh
python3 -I scripts/check-mygo-display-link-recovery.py --native-long-range --range-line-kind add
```

该命令已实际通过，门槛 `build/mygo-frame-recovery-w27bmngy/`、三轮原生结果 `build/lobe-long-patch-gv7wvl0w/results.json` 已核对，普通 e2e 自动恢复。工作区从末条添加行向上选 40 行，精确 Index 按替换配对顺序在对应原行之后插入所选 working 行；在 Staged 中从新行号 3962 到 4040 选 40 条添加行（中间上下文不计入选区），取消暂存后外部清理前完整原始指纹恢复。代表暂存 / 取消暂存截图已查看。它与前述删除行一样，仅验收隔离候选，不代表普通默认依赖已通过。

### 宽 DAG 与加载期输入

```sh
python3 -I scripts/native-lobe-wide-history-baseline.py
```

自建隔离 fixture：32 并行分支、32,002 提交、33 父汇合节点。三个独立应用进程 / HOME；要求 End 前后仍在读取，选中已加载行并显露，再取消 / 重试至完整历史、End 到根提交，核对原仓库指纹。2026-11-16 三轮通过，完整历史只构建 27 个行控件；结果与截图保留于 `build/lobe-wide-history-*/`。分支提交不改文件，未采 RSS，也不代表所有加载期交互已验收。

两套长补丁 / 宽 DAG 脚本现通过显式受信路径加载 `native-memory-sampler.py`，采集应用及后代 RSS、阶段样本数与错误。2026-11-16 各三轮顺序通过，完整日志 `build/lobe-patch-dag-memory-check.log`；结果分别 `build/lobe-long-patch-due3jb5e/`、`build/lobe-wide-history-e49lgn7x/`。每轮 17 / 121 个样本，应用采样最大 RSS 中位 201,408 / 715,024 KiB；缺失短阶段样本不记作零，不当作精确峰值、堆归因或帧时间。

加载期重复交互模式：

```sh
python3 -I scripts/native-lobe-wide-history-baseline.py --input-pressure
```

每轮在历史读取时发送 12 次 precise wheel 并核对实际行位移，再逐字输入 / 清空搜索，核对精确值和可见匹配行；要求所有交互检查点仍显示读取中，随后走 Cancel / Retry / 完整历史 / 根导航与指纹保护。2026-11-16 三轮通过，证据 `build/lobe-loading-input-pressure-check.log` / `build/lobe-wide-history-rtp1atxz/`。轮事件逐次 settle，不代表真实硬件高频事件或帧率；搜索编辑不替代所有输入控件验收。

### 大历史完成后的浏览性能

```sh
make lobe-e2e-build
python3 -I scripts/native-lobe-large-history-performance.py
```

此入口使用隔离仓库和 HOME：64 条交错分支各 500 个提交、周期性交叉合并、512 个额外引用，共 32,002 个提交和 577 个分支引用。核对完整历史、可见行控件数量、End 到根提交、Home、搜索输入及清空后的导航，以及 HEAD／refs／Index／status 指纹不变；采集应用与后代 RSS，保留历史／搜索截图、辅助功能失败快照和 `results.json`。当前合成样本门槛为应用采样峰值 RSS 小于 384 MiB，单次 End／Home／搜索自动化往返小于 3 秒。该门槛仅约束此夹具，不是任意仓库的内存上限或帧率承诺。

2026-11-17 最终样本通过，证据 `build/lobe-large-performance-9v4794fl/`：完整历史 1.802 秒，End 0.691 秒，搜索 0.682 秒，仅构建 12 个可见历史行控件，应用采样峰值 RSS 252,896 KiB（约 247 MiB）。运行时同时有项目检查，启动时间包含资源竞争；耗时包含自动化往返。已查看完整历史和搜索截图，标题独占第一行，作者／引用在第二行，多 lane 图列不会挤走标题。

原 `native-lobe-wide-history-baseline.py --input-pressure` 保留其“交互时仍在加载”的断言；此次 32,002 提交在首次输入前已读完，该入口因此拒绝样本，不算加载期输入验收通过。新入口专门验证完成加载后的浏览。取消／重试／旧上下文拒绝另由 Go 集成测试验证；并未复现用户另一台电脑上的真实仓库和 1.8 GB RSS。

## 定位元素：先读 bounds，不靠目测

- `snapshot --json` 给出每个节点的 `bounds`（DIP）。悬停、点击都用它算坐标；历史记录里多次出现「目测偏了一截」导致的假失败。
- ref 会随重绘失效：每次界面变化后重新 `snapshot`，不要复用上一帧的 ref。
- `click <sel>` 优先；`press`/`focus`/`fill`/`act` 按元素操作。确实要按坐标时用 `click <x> <y>`、`pointer move <x> <y>`。
- 截图是逻辑尺寸的 2 倍（Retina）。从截图量出来的像素坐标先除以 2 再换算成窗口 DIP。

## 右键菜单

区分应用绘制的菜单与系统上下文菜单。应用绘制的菜单可沿用已经通过的路径：聚焦目标行，`key Shift+F10` 打开，再从新的 `snapshot` 找到 `menuItem` ref 并 `press`；每次重绘后重新读取 ref。

当前安装的 `mygo-automation skills` 明确说明系统上下文菜单不能选择，并要求不要用 `click --button right` 打开。不能把截图后猜坐标点击作为受支持的替代路径。Force / prune 实际使用应用绘制的菜单，已由 `native-lobe-force-prune-smoke.py` 通过 focus + Shift+F10 路径原生验证，不能归入系统菜单限制。应用内菜单可在打开后从当前快照取得 DIP bounds 做坐标激活；这与系统菜单不同，也不能仅凭 semantic press 宣称鼠标可达。不要为绕过驱动限制新增产品入口。驱动升级后先重新阅读该版本的 guide，再决定是否扩展自动化。

`python3 -I scripts/native-lobe-menu-access-smoke.py` 在独立 35 长分支 / 双本地远端仓库、980×640 窗口下核对历史菜单全部 18 项 bounds 和 Down 焦点遍历，并实际坐标打开分支 picker / 底部 Reset、用 Enter 打开 Reset，关闭后核对 Git 状态未变。结果保存在每次独立的 `build/lobe-menu-access-*`；最新完整结果 `build/lobe-menu-access-sfbuiw3v/results.json`，日志 `build/lobe-menu-access-recovery.log`。后续覆盖四种坐标过滤、正确查询与匹配 / 排除记录、搜索 Enter 保持 checkout，以及无结果与读取失败区分：仅移开自建 fixture `.git`，Refresh 显示旧内容提示，finally 恢复后坐标 Retry 成功重读。picker 用 Escape 关闭，Reset 用 Cancel。最新 `build/lobe-menu-access-vmz51dcg/results.json`、`build/lobe-menu-access-dialogs.log` 进一步通过全部 11 个表单 / 确认入口的坐标打开与取消，加四种过滤共 15 项鼠标激活。该脚本不执行确认后的 Git 写操作；三项复制命令只验证 bounds / 键盘焦点，未实际点击以免覆盖当前系统剪贴板。

父提交导航补充验收：`native-lobe-menu-access-smoke.py` 先将父 SHA 按钮 `scrollIntoView`，确认可见后按实际 bounds 坐标点击，核对原父提交选中、返回源提交及完整 Git fingerprint。该鼠标路径曾稳定暴露后续 Blame 首次 Escape 被背景摘要引用 Tooltip 消耗的问题；`TestBlameEscapeWithHoveredBackgroundRef` 已复现并验证修复，摘要提示在主窗口弹窗期间暂停，保留无障碍说明。`build/lobe-blame-background-tooltip-native.log` 和 `build/lobe-menu-access-i9jx512g/results.json` 已通过该完整流程，修复后的 Blame 截图已检查。旧 `build/lobe-menu-access-jiqbt71o/blame-dismiss-diagnostics.json` 中 `open_after_close_click=false` 不能作为 Close 点击证据：移开指针后第二次 Escape 已关闭，条件分支没有执行 Close。最终探针要求第一次 Escape 成功，失败即保存截图并报错，不移开指针重试。最终严格版本再次通过（`build/lobe-blame-background-tooltip-native-final.log`），`build/lobe-menu-access-ysc9hq_4/results.json` 明确记录 `visible_parent_coordinate_navigation=true` 与 `blame_first_escape=true`。该源码的完整 `make check` 也通过：常规测试、race、vet 和 diff 格式检查，日志为 `build/lobe-blame-background-tooltip-check.log`；lobedesk 测试分别用时 60.537s / 302.550s。

## 滚动

- `wheel` 一次只移动约一两行，滚到内容底部很慢；验证滚动相关行为时优先拖滚动条（按下、移动、松开三步），并断言滚动后的内容与选中态。
- 需要程序化滚动时用 `wheel x y dx dy`，注意指针位置要落在目标滚动区内，否则滚的是页面外层。

## 等待与判定

- 用 `wait <sel>`、`wait --text <s>` 等界面内容出现，而不是固定 `sleep`；它们只保证界面出现，不保证后台 Git 完成。
- `--settle` 等「窗口把自己请求的帧画完」，默认 10s；它不等应用自己设的计时器或后台任务。带 `--settle` 的动作会打印与上次快照的差异，`(no changes)` 也要读，别只看退出码。
- 后台任务失败时界面会出 toast/错误块；先找它，再判断操作是否成功。
- 验证写操作后回到 shell 用 Git 核对（`git status`、`git diff --cached`、`git config --get` 等），不只信截图。

## 命令面板创建 Stash（2026-11-18）

`TestPaletteStash*` 使用 production 的 prompt→commandPicker 渲染顺序，验证36组取消（Summary／Body／历史搜索 × 两个面板快捷键 × Enter／点击 × Escape／背景／Cancel），以键入 X 替换原选中内容证明原选区恢复；一个 prompt、输入归属、空值／纯空白不提交、Tab 后不抢焦点均有实际断言。初次真实测试复现通用 Modal 丢原 opener 以及400px控件窄窗越界；只有这次 palette Stash handoff 改用共享 overlay key 的自适应 DialogBase，其他通用 prompt 保留原渲染路径。

新入口 tracked-only；owned Go探针确认原 SaveStash(false) 在仅未跟踪时返回nil却无新条目，新增入口据此禁用且显示原因，不改原包含未跟踪或单路径流程。两个真实接受场景覆盖 Enter／Save点击；新 Stash 的 index/work blob OID与原Index和工作文件哈希精确相等，旧 Stash SHA、HEAD／本地refs／untracked／ignored／提交草稿和amend保留。取消原始Index及内容不变，重复／保留回调不新增第二条；root／epoch／离开返回及打开表单后busy/loading/conflict/operation/only-untracked均拒绝写入。

产品复核指出会话变化静默拒绝的问题，先确定性RED再修复：同仓库epoch变化或不可执行状态明确提示，异仓库旧回调不污染新仓库。最终focused 2.074秒、相关回归17.414秒、定向race 11.048秒及vet通过，Windows／Linux编译通过。设计实际查看 `build/palette-stash-captures/` 的8张中英浅深360×360入口／表单截图，无视觉阻塞项。完整检查 b1dnknvnu 已通过（普通测试117.133秒、race 399.629秒、vet无错误），已读取完整日志 `build/alignment-palette-stash-full-check.log` 并确认455个冻结源码指纹一致、差异格式检查通过。最终结果见 `docs/superpowers/plans/2026-11-18-palette-stash.md`；没有新增原生尝试，也没有操作真实项目Stash。

## 命令面板创建并切换分支

2026-11-18 新增 `branch.create`，明确从当前 HEAD 创建并切换，名称表单显示捕获起点的短 SHA；执行传入完整 SHA。保留既有后端 clean／名称保护，新增入口复核根目录、加载 epoch、UI HEAD／分支和就绪状态，回调只消费一次。参考资源证明 create_branch 目录／定向入口存在，但未证明其默认起点和 checkout 行为；这里沿用 SuperGit 既有分支页行为。

UI Tester 验证36组 Summary／Body／历史搜索 × 两个面板别名 × Enter／点击 × Escape／背景／Cancel 的交接取消、实际选中文本恢复和草稿／amend保留；另覆盖空白名、禁用原因、过期状态、Tab离开输入及360×360中英浅深色布局。临时自有仓库验证 Enter／按钮创建、detached HEAD、invalid／duplicate名称、外部变脏拒绝，以及表单打开后外部推进 main 时新分支仍从捕获 SHA 创建。完整 refs 快照（含额外保留 tag）证明原分支精确推进和全部旧引用不变；原始Index／工作文件保持，重复回调不增建引用。异仓库旧回调静默，同仓库拒绝仅告警一次。

初轮分支／Stash回归5.153秒；相关普通18.478秒、定向race 19.632秒及vet通过；审查后补强断言与截图3.850秒通过。日志为 `build/alignment-palette-create-branch-{first,related,review-check}.log`。Windows／Linux amd64编译通过。设计已看8张新截图；浅色就绪按钮初图停在100ms颜色动画起点，测试时钟推进150ms后呈现正常绿底白字，未修改生产样式。焦点／点击行为依据测试断言，不依据截图。首次完整检查biz9d85a6在既有中文创建动词一致性测试失败，race／vet未执行；将新入口“新建”统一为“创建”后，该规则和分支测试3.770秒通过。设计独立重看四张完成动画的表单图并撤回对比度问题。最终文案的Windows／Linux编译通过；修正后完整检查bmv4njj44已通过（普通lobedesk测试122.132秒、race 407.226秒、vet无错误），完整日志 `build/alignment-palette-create-branch-final-check.log` 已读取。最终458个源码输入指纹一致，无新增／移除输入，差异格式检查通过。实现与自动化验收已完成，原生运行与参考默认语义仍未验证；最终结果见 `docs/superpowers/plans/2026-11-18-palette-create-branch.md`。没有新增原生尝试、真实项目分支操作、commit或发布。

## 命令面板选择历史引用范围

2026-11-18 新增单一命令 `history.chooseRef`，目录现有15项，无新增专用键位。复用原页头引用搜索／列表／错误Retry内容；新面板阶段沿用同一DialogBase会话和overlay key，恢复原opener选区，原页头Popover行为保留。文案明确仅影响历史列表、不切分支；文件历史期间禁用并说明。Sublime toggle_ref_filter与此范围选择器的运行时等价性尚未确认。

缺少入口RED后，24组Summary／Body／历史搜索 × 双面板别名 × Enter／点击 × Esc／背景取消通过，验证真实选中文字替换及scope/search/draft/amend保留、首个Enter不穿透。真实自有仓库中Enter／点击确认读取feature旧提交的单提交历史，保留main HEAD、全部refs（含既有Stash）、原始Index、已暂存／工作／未跟踪状态及字节。另覆盖无匹配、空缓存引用、读取错误Tab可到Retry、候选删除和root／epoch／文件历史变化后的关闭及存活输入选区恢复。空缓存不等于完整unborn原生验收；内部旧闭包未被直接调用，删除覆盖基于当前UI事件。

产品未发现明确行为问题，测试建议已补强；设计实际查看4张中英浅深360×360错误态图，未发现布局阻塞，主查看正常／错误三张代表图。保留既有Git错误原文，本地化简短摘要记录为后续小项。图片位于 `build/palette-history-ref-captures/`，截图不证明native焦点。

补充回归先复现共享modal登记缺失，修复后header／picker／catalog检查6.108秒通过，包含保留后台回调阻断、实际页签快捷键阻断及取消后守卫释放。此前相关普通16.689秒、定向race 33.264秒和vet通过；最终Windows／Linux amd64编译通过。最终完整检查bi667xs46已通过（普通lobedesk测试121.691秒、race 413.632秒、vet无错误）。完整日志 `build/alignment-palette-history-ref-full-check.log` 已读取，460个冻结源码指纹一致，无新增／移除输入，最终差异格式检查通过；见 `docs/superpowers/plans/2026-11-18-palette-history-ref.md`。未进行原生权限重试、真实项目写操作、commit或发布。

## 文件历史的固定起点恢复验证

2026-11-18 仅补测试：删除后的重命名路径（含Unicode和类似pathspec前缀）按每页两条读取，必须与固定tip的完整五条 `git log --follow` SHA顺序一致；首屏后在新HEAD重建同名文件，原历史仍保留改名前祖先并排除无关提交。UI暂停／失败后修改目标文件并产生新HEAD，再实际点击Retry，核对原tips、每个原SHA、排除新HEAD及最终没有更多页。空页但HasMore为真立即失败，避免分页不前进。

初始后端夹具曾因未使用字面路径、revision带换行失败，已修正并纠正早期通过声明；未发现生产缺陷。修正后端定向0.634秒；最终相关后端／UI 1.586／2.599秒，定向race 1.693／1.936秒，vet通过，完整日志已读。完整检查普通lobedesk 136.440秒、race 443.822秒通过，但vet因后续临时键盘测试位于build/、被识别为独立包而失败（undefined app），完整日志已读；这不是通过的完整验收。临时测试移入正确源码包后，重新完整检查退出0：普通lobedesk 134.007秒、race 441.496秒、vet通过；完整日志和验证JSON已读，471个输入无变动／新增／移除／缺失，八个包一致，最终差异格式通过。文件历史恢复与首批创建键盘验证完成自动化验收。见 `docs/superpowers/plans/2026-11-18-file-history-recovery-verification.md`。测试均使用自有临时仓库，不代表原生或Sublime运行等价。

## 重命名文件行的路径方向

2026-11-18 稳定帧实图确认宽栏重命名行把旧路径箭头放在新文件名右侧、且替代当前目录；同名跨目录移动目标不清楚。新增渲染回归确认四个宽态RED、窄态对照通过。共享 `fileRowPath` 已改为“旧路径 → 当前文件名”，当前目录保留右侧；箭头独立不可压缩，旧路径省略后仍显示。窄于260仍优先文件名，普通行布局／行高保持。

最终候选相关测试2.795秒、race 2.559秒、vet及Windows／Linux amd64编译通过。120子例覆盖commit／workspace、根目录／长路径、两主题及五种窗宽；现有阈值单元测试覆盖精确259／260，窗口用例依实际行宽分支，不将两者混同。根目录同名标签使用行label和绘制name计数，未宣称独立字形bounds。设计实际查看最终三张代表图，产品核对360图，箭头截断问题已解决。正式renderer与候选逐字节绑定，已移入源码；474输入最终完整门槛通过，退出0：普通lobedesk 134.353秒、race 443.171秒、vet无错误；完整日志和验证JSON已读，源码无变化／新增／移除／缺失，八个包一致，最终格式检查通过。实际源码六张图与已审查候选PNG逐一hash一致，视觉修正及自动化验收完成。见 `docs/superpowers/plans/2026-11-18-rename-row-direction.md`。编译和截图不等于原生或Sublime运行等价。

## 新建仓库的实际键盘输入验证

2026-11-18 新增 `internal/lobedesk/new_repository_keyboard_test.go`：空状态／已有仓库入口真实点击，打开即输入Unicode和空格路径，Escape／Cancel后重开，核对无创建及原root／draft保持；占用路径Enter报错保留原文件，再无需显式refocus地全选替换新路径、Enter创建，核对unborn main、空Index、切根及成功后草稿清除。真实空白／相对路径校验也有覆盖。

360×360大字号长错误，中英浅深实际通过 `a.applyTheme` 并断言主题，核对按钮启用、正尺寸、视口边界及实际Cancel关闭。早期仅设置themeMode的测试不足以证明主题，已按独立测试审查修正。最终overlay相关2.448秒、race 3.317秒、包vet通过；正式移入源码包后相关2.159秒及全量vet通过，八个Go包扫描正常。471输入完整make check已通过并核对日志／源码指纹／八个包集合／差异格式（普通134.007秒，race 441.496秒，vet通过）。随后将已审查的异步Git失败键盘重试和真实rename解析→菜单→路径范围两项测试移入源码，473输入完整检查已通过（普通134.767秒、race 444.587秒、vet通过），完整日志／指纹／八包集合／格式已核对。两项测试自动化验收完成。后续重命名行视觉修正也已通过474输入完整门槛，见独立计划。不证明原生窗口焦点、错误全文滚动可达性或Sublime运行等价。见 `docs/superpowers/plans/2026-11-18-new-repository-keyboard-verification.md`。

## 历史提交差异的比较基准说明

2026-11-18 在历史文件差异视口上方增加次级中英文说明：普通／merge提交显示“第一父提交短SHA → 当前提交短SHA”，根提交显示“根提交，无父提交”。未知元数据、隐藏差异及工作区路径不显示该历史说明；不增加比较选择器，不改变Git命令或导航。说明位于虚拟滚动内容之外，保留行高／滚动偏移语义。

缺少说明先RED后GREEN；自有临时Git仓库覆盖root／ordinary／不同tree的双父merge／empty提交的真实ShowAll内容与文案一致，核对第一、第二父顺序并保留refs／HEAD／原始Index／暂存与工作差异／未跟踪文件及draft／amend。300px大字号中英浅深十二张空diff截图位于 `build/commit-diff-baseline-captures/`，设计实际查看四张代表图无阻断；相关未知／隐藏断言及非空窄窗有覆盖。实际selectCommit快速切换并晚交付旧读取回调，核对新说明／加载／补丁一致且草稿不变。

最终相关普通3.363秒、定向baseline race 2.241秒通过；此前相关race 12.561秒、vet及Windows／Linux编译通过。完整检查bos2ak80l已通过，退出码0：普通lobedesk测试132.707秒、race 434.768秒、vet无错误。完整日志 `build/alignment-commit-diff-baseline-full-check.log` 已读取，470个冻结源码输入全部匹配，无新增／移除／缺失输入，结果见 `build/alignment-commit-diff-baseline-source-verification.json`，最终差异格式检查通过。实现与自动化验收完成，见 `docs/superpowers/plans/2026-11-18-commit-diff-baseline.md`。未宣称原生运行或Sublime标签等价；没有真实项目Git写操作、commit、安装或发布。

## 历史搜索未完成时的零匹配反馈

2026-11-18 核对SHA输入调用链后，仅调整既有零匹配空态：当前历史源未读完时，中英文提示“已读取的历史中暂无匹配，历史尚未读完。”／“No matches in loaded history. History is incomplete.”，完成源保留原文案。普通历史依据historyMore，文件历史使用自身more／loading／paused／error状态，不借用普通历史完成标志。没有新增SHA解析、Enter行为、命令入口、范围或加载策略。

实际渲染20例覆盖两语言×普通及文件历史的加载、暂停、错误和完成状态，先观察16个未完成例RED，再修复通过；同时断言原选择／查询／文件范围／草稿／amend及工作区行保留。相关历史／父子导航／中文规则检查10.414秒通过（`build/alignment-history-search-feedback-green.log`）。三个角色已完成独立复核，补充12例中英文状态优先级回归及非活动历史源状态隔离后，最终相关检查10.449秒通过；此前定向race 21.747秒、vet及Windows／Linux编译通过。完整检查b9kqe2k7d已通过，退出码0：普通lobedesk测试133.366秒、race 435.676秒、vet无错误。完整日志 `build/alignment-history-search-feedback-full-check.log` 已读取，468个冻结源码输入全部匹配，无新增／移除／缺失输入，结果见 `build/alignment-history-search-feedback-source-verification.json`；最终差异格式检查通过。实现与自动化验收完成，见 `docs/superpowers/plans/2026-11-18-history-search-feedback.md`；本段不代表原生运行或Sublime输入语义已验证。

## 命令面板克隆仓库

2026-11-18 增加无新快捷键的 `repo.clone`，目录17项，无仓库也可打开既有Clone表单。缺少入口RED后，直接旧Modal交接的实际取消选区恢复失败；palette来源改用同一DialogBase会话，普通菜单／空态Clone继续使用原Modal，共用表单校验及操作逻辑。24组有／无仓库 × 双别名 × Enter／点击 × Esc／背景／Cancel实际验证地址输入焦点、单一弹窗、不误提交、重复别名不重置、取消后X替换原Body选区并保留草稿／amend；另覆盖Cmd+Enter隔离和旧root／epoch／generation及busy／modal拒绝。

自有临时本地仓库覆盖成功克隆并核对base.txt内容、已存在目标哨兵不变，以及成功回调排队后取消仍保留旧仓库和错误反馈。360×360中英浅深四图在 `build/palette-clone-captures/`，设计实际读取并复核按钮颜色动画完成后的四图，无对比度阻断；焦点证据来自UI测试。补测先复现错误区导致按钮越界，palette专用错误滚动区缩小后通过；普通Clone错误区高度保留。

最终相关普通13.620秒、定向race 23.450秒及vet通过，Windows／Linux amd64编译通过。完整检查bzistppsr已通过，退出码0：普通lobedesk测试133.887秒、race 436.683秒、vet无错误。完整日志 `build/alignment-palette-clone-full-check.log` 已读取；467个冻结源码输入全部匹配，无新增／移除／缺失输入，结果见 `build/alignment-palette-clone-source-verification.json`，最终差异格式检查通过。实现与自动化验收完成，见 `docs/superpowers/plans/2026-11-18-palette-clone.md`。未验证真实传输中取消、各平台原生运行或Sublime面板等价；未操作真实远端／项目Git数据、commit、安装或发布。

## 命令面板创建本地轻量标签

2026-11-18 增加 `tag.create`：“在所选提交创建轻量标签…”，通用目录16项，无新增快捷键。要求history页可见选中提交行；工作区行、其他页和过滤后不可见的选择禁用，不回退HEAD。详情隐藏不影响可见行，文件历史中有效提交允许。表单显示本地轻量标签及捕获短SHA，调用既有promptCreateTag传完整SHA；没有附注、推送或目标选择器。

缺少命令RED后，36组Summary／Body／历史搜索 × 双别名 × Enter／点击 × Esc／背景／Cancel及空白名测试通过。真实自有仓库中Enter／点击在旧提交创建轻量标签（对象类型commit），改选另一个有效提交不会替换捕获目标；Unicode／分层名、非法名／重名错误有覆盖。精确比较原有refs（含Stash）、HEAD、原始Index、暂存／工作／未跟踪状态及字节，保留draft／amend。8张360×360中英浅深入口／表单图位于 `build/palette-create-tag-captures/`；设计实际读取4张表单图，主看两张表单及英文入口，未发现布局阻塞。Tab和取消恢复依据UI Tester，不依据截图。

测试审查后新增取消旧回调回归，先在临时仓库复现取消后仍可创建标签，再给共享palette prompt增加取消失效钩子。按同类检查另复现分支／Stash三种取消共6例并接入相同钩子；原菜单和成功操作语义保留。最终相关普通11.026秒、定向race 34.775秒及vet通过（`build/alignment-palette-prompt-cancel-green.log`），最终Windows／Linux amd64编译通过。新UI测试未另做hook拒绝专项；作者修复后未要求角色独立重跑。

完整检查bn2mvkmf2已通过，退出码0：普通lobedesk测试126.121秒、race 423.627秒、vet无错误。完整日志 `build/alignment-palette-create-tag-full-check.log` 已读取；464个冻结源码输入指纹全部匹配，无新增／移除／缺失输入，结果见 `build/alignment-palette-create-tag-source-verification.json`，最终差异格式检查通过。实现与自动化验收已完成，见 `docs/superpowers/plans/2026-11-18-palette-create-tag.md`。Sublime默认目标／标签类型和各平台原生运行未验证。未执行真实项目标签、推送、commit、安装或发布。

## 命令面板 Fetch（2026-11-18）

`TestPaletteFetch*` 覆盖 Summary／Body／历史搜索 × Enter／点击六组合。只使用两个受控配置的本地 file remote；比较获取前后 HEAD／本地分支／symbolic HEAD／原始 Index／stage 列表／状态／配置／暂存和工作、未跟踪字节及远端 refs，另验双方 tracking refs 和 FETCH_HEAD 更新、已有 stale tracking ref 保留。Escape／背景取消不产生 Git 写入，重复和保留回调不重启；真正 disabled 行显示无仓库／无远端／忙碌／加载／弹窗等原因，过期 root／epoch／generation 拒绝执行。原输入选区通过完成后输入 X 替换证明，草稿和 amend 保留。普通获取未显式请求 prune，实际 Git 配置仍适用。

失败反馈测试发现共用 performDoneAs 时序问题：worker 的 defer cancel 先于 UI continuation，普通错误会被误判为已取消。`TestOperationCompletionKeepsResultBeforeContextCleanup` 等待 worker context 清理后才运行回调，确定性 RED 复现，再提前捕获取消判定修复；失败／主动取消／成功三类结果均通过。相关测试 18.926 秒、定向 race 10.686 秒及 vet 通过，Windows／Linux编译通过。四张 360×360 中英浅深截图在 `build/palette-fetch-captures/`，设计角色实际查看，无本次范围内阻塞项。

F6 没有绑定 Fetch；参考仅证明命令声明，全远端直接执行为现有产品适配。没有新增原生尝试，仍不能把 UI Tester 输入证据当作 macOS 原生接受。完整检查已通过（普通 lobedesk 117.324 秒、race 390.915 秒、vet 干净），完整日志已读，453 个冻结源码输入匹配；证据见 `docs/superpowers/plans/2026-11-18-palette-fetch.md`。

## F8 普通 Push 入口（2026-11-18）

`TestPushShortcut*` 使用实际 Summary／Body／历史搜索按键验证 F8 打开现有普通确认，来源SHA与remote/目标可见，重复按键不叠窗；Escape／背景／Cancel九组合通过替换选中文本证明恢复。guard覆盖无仓库、detached/unborn、无remote、busy/loading/history reload、chooser/ref及主窗口各类modal，保留回调仍须重验。修正过期reset/child夹具后，生产代码无需额外修复。

两个owned本地文件传输裸远端场景分别有／无upstream，确认前及取消后比较HEAD／symbolic HEAD／refs／原始Index／状态／配置／暂存与未暂存、未跟踪字节，确认后检查精确目标OID与既有tracking策略。已有PushChecked测试复用源／URL／pushURL／缺源／非快进拒绝保护。实现先在独立快照worktree验收，整合后相关lobedesk测试通过5.760秒，checked-push层2.867秒，vet干净；Windows／Linux编译通过。

不发送到真实远端，不执行项目Push或force。没有新增原生尝试；已知自然窗口焦点前提不能由UI Tester通过推断为原生成功。参考F8声明包含定向palette步骤，当前复用确认只代表动作入口适配。完整检查状态与证据见 `docs/superpowers/plans/2026-11-18-push-shortcut.md`。

## 命令面板交接仓库选择器（2026-11-18）

通用目录新增唯一 `repo.quickSwitch`（11项，别名不重复），与检出面板复用共享 overlay key 和互斥渲染。`TestPaletteRepositorySwitch*` 验证24组合（Summary／Body／历史搜索 × 两个面板快捷键 × Enter／点击 × Escape／背景），以直接替换原选中文本证明恢复；单弹窗、搜索输入归属、Tab 后逻辑文本焦点消失且仓库选项获焦、动态禁用、无仓库禁用／无候选空态、过期回调及两个自有仓库的实际确认成功／失败均有断言。专用切换测试也改经生产共享渲染路径，回归直接入口。最终相关测试通过 23.698秒。

新增命令的四张360×360中英／明暗截图位于 `build/palette-repository-switch-captures/`，设计角色实际查看并接受；原选择器的路径与空态视觉沿用已通过矩阵。Windows／Linux编译通过。本增量没有重试未变化的macOS自然焦点前提；不宣称原生交接通过。完整检查状态见 `docs/superpowers/plans/2026-11-18-palette-repository-switch-handoff.md`。

## 快速切换仓库验收（2026-11-18）

会话 MRU 选择器使用平台专用快捷键，复用现有打开流程。`TestQuickRepositorySwitch*` 覆盖真实标题／正文／历史搜索的原选区取消恢复、主快捷键与当前平台别名、背景取消、空结果、失效 root／loadSeq／generation、保留的后台回调，以及两个带暂存／未暂存／未跟踪内容的临时仓库 A→B→A。缺目录、移除 `.git`、损坏 Index 导致 Open 成功但 Status 失败、过期失败回调均验证当前仓库／草稿／MRU 的边界；仓库往返比较 HEAD／symbolic HEAD／refs／Index／状态／夹具文件字节。

24 组渲染截图位于 `build/repository-switch-captures/`，覆盖 360×360／900×640、中英、明暗和同名长路径／两种空状态。设计角色实际检查全部 12 张窄版及 4 张宽版路径图，没有需修改项。Windows／Linux 仅证明编译通过，尚非原生运行验收。

macOS e2e 构建通过。首个原生夹具 `build/repository-switch-native-bey_q8kf/` 因干净仓库的 Changes 不显示文件而等待失败；修正为已修改文件后，`build/repository-switch-native-iubt74eu/` 在五秒自然窗口焦点检查处停止，未发送输入，应用进程退出 0。没有绕过系统权限或继续盲发键，不能据此声称原生切换／取消成功。最新完整日志：`build/alignment-repository-switch-native.log`。完整检查结果见 `docs/superpowers/plans/2026-11-18-quick-repository-switch.md`。

## Lobe 工作台验收脚本

2026-11-17 最终集成验收：`build/lobe-integrated-final-matrix.log` 的 16 组 SuperGit 组合全部通过，208 张截图位于 `build/narrow-shots-_xcp22dz`；已抽查窄窗口中英大字号的选区/标签、宽屏启动 Changes 和长摘要。随后详情流程首次在第七组 Cmd+Enter 等待 hook 时失败，按键响应显示仓库重读，草稿与暂存文件保留；完整证据已复制到 `build/lobe-integrated-final-detail-tabs-failed-1`。未修改脚本复跑全部十组通过（`build/lobe-integrated-final-detail-tabs-reproduce.log`），含 hook 拒绝保留草稿、Index-only 提交、Amend 作者保留及 2405/2403 条历史遍历。保留首次时序失败，不能把复跑通过解释为已修复其触发条件；提交按钮就绪快照与后续快捷键投递并非原子操作。

先运行 `make lobe-e2e-build`；以下脚本在 macOS 图形会话中串行运行，使用临时仓库和独立 HOME，不更新已安装应用。

- `python3 scripts/native-detail-tabs-smoke.py`：标签 / 草稿 / 偏好、真实 hook 拒绝与 Commit / Amend、文件和批量操作范围、取消确认及 Index 保护；再验证 2,405 条主历史和 2,403 条文件历史自动完整加载、End 揭示最早行，以及浏览前后 HEAD / Index / 工作文件不变。文件历史入口使用文件行聚焦后 `Shift+F10`，选择辅助功能树中的 `File history…` 菜单项。截图与日志保存在 `build/detail-tabs-smoke/`。
- `python3 scripts/native-narrow-shots.py`：980 / 1320 / 1600 宽 × 中文 / 英文 × 浅色 / 深色，加四组 1.15 字号组合；每组捕获摘要、多标签、空提交、工作区、禁用 / 启用提交。只复查某组时可设置 `SUPERGIT_SHOT_CASES=980-zh-light-large`，单组通过不代表完整矩阵通过。失败保存截图和辅助功能快照。
- `python3 scripts/native-workbench-layout-smoke.py`：真实指针拖动三处分界，核对偏好写入，重启后核对尺寸恢复和 Git 状态不变，结果在 `build/workbench-layout-smoke/`。
- `python3 -I scripts/native-lobe-window-dialog-smoke.py`：临时脏仓库与独立 HOME，分别先开设置重置确认 / 主窗口 Discard 确认，再从另一窗口请求确认；验证先开的确认保留、只在 owner 窗口显示、两边取消均保留 HEAD / refs / 原始 Index / 状态 / 工作字节。另验证主窗口 New branch 草稿与设置重置确认同时存在，取消设置确认保留精确草稿，再取消命名不写引用。每次证据位于独立 `build/lobe-window-dialog-*/`。不执行重置或丢弃，不证明真实系统窗口焦点切换。等待命名框使用可见标题，`wait --text` 不匹配字段 placeholder。

- `python3 -I scripts/native-lobe-reset-smoke.py`：四个独立临时脏仓库 / HOME，历史行 `Shift+F10` 选择应用绘制的菜单，验证 Soft / Mixed / Hard Reset 与 Undo 的取消、Hard 二次确认及实际 HEAD / Index / 工作字节和未跟踪二进制保护。加 `--stale` 时改验确认打开后外部 HEAD 推进的拒绝，Hard 在二次确认后推进；核对本地化恢复说明和完整外部新状态不变。证据位于 `build/lobe-reset-*/`；不操作源仓库，未覆盖同 SHA 切分支、进行中操作替换或跨进程原子互斥。

- `python3 -I scripts/native-lobe-force-prune-smoke.py`：独立临时脏仓库 / HOME 与本地裸远端，顶栏 focus + Shift+F10 打开应用内 Push / Fetch 菜单；验证 Force 的源 / 租约文案、取消、分叉替换、远端推进并 fetch 后旧租约拒绝、详情内当前语言说明与 Git stale info；prune 只删过期跟踪引用，保留同名本地分支、HEAD / 原始 Index / 暂存与工作字节 / upstream。长错误必须先打开 View details，不能直接等待被折叠的正文。证据位于 `build/lobe-force-prune-*/`。

- `python3 -I scripts/native-lobe-integrate-smoke.py`：两个独立临时仓库 / HOME，历史行 focus + Shift+F10 → Cherry-pick / Revert，验证取消不写入、所选 SHA、新提交父提交、完整说明 / 原 SHA、准确 Unicode 路径文件字节、topic 引用保留和无残留进行中操作。证据位于 `build/lobe-integrate-*/`；只覆盖无冲突发起，不是应用内冲突编辑。

自动化 `window w1` 只改变命令目标，不改变原生键盘焦点；当前确认框固定到发起动作的 owner 窗口，切换命令目标不会迁移它。`focus` 与坐标输入也只把事件送给目标内容，不调用系统窗口激活。旧脚本的 `window auto` 只适用于实际焦点恰为确认 owner 的场景；双窗口验收应分别读取两个窗口，并向 owner 窗口的新鲜按钮 ref 回答，不能把命令目标切换当作真实系统焦点切换。

## 统一 History 窗口（2026-11-19）

默认入口不再是 Changes：仓库打开后直接停在 History，选中当前分支最新提交（有提交时），工作树是列表首行、Stash 是其后的一行，页面只剩 History（⌘2 分支页、⌘4 工作树页仍保留）。启动探针不应再等 `Commit summary (required)`，也不存在 `Changes` 页面标签；可用 `All refs` 或具体提交标题作为历史就绪门，再按需选中工作树行（如 `1 change` / `2 changes`）。

`python3 -I scripts/native-lobe-unified-history.py` 是本次入口的独立原生检查：临时仓库含一个比 HEAD 更晚的旁支提交和一个 Stash，断言没有 Changes / History / Stash 页签、行顺序为工作树 → Stash → HEAD → 更新旁支、点击工作树行出现提交编辑区、点击 Stash 行出现 Apply / Pop / Drop，且浏览不改动 Git 数据；同时用 PATH 垫片记录打开期间只有一次 `git status --porcelain`。结果和截图保存在 `build/lobe-unified-history-*/`。多数既有 `native-lobe-*` 脚本仍按旧默认等待 Changes 编辑区或按 `Changes` 标签，运行前需按上述门更新；本轮未逐一重跑这些脚本。
