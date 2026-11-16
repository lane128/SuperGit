# e2e 自动化验证笔记

范围：`cmd/supergit-lobe`（`-tags e2e` 构建）与 `mygo-automation` 的配合。这里是验证时踩过的操作细节，不是产品文档；实现以代码与 `mygo-automation skills` 的输出为准。

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

## 标签式详情的隔离验证

```sh
make lobe-e2e-build
python3 -I scripts/native-detail-tabs-smoke.py
```

脚本自行创建并清理临时仓库与独立偏好，日志和截图保留在 `build/detail-tabs-smoke/`。当前覆盖同名不同路径、暂存 / 未暂存双版本、重复打开、关闭活动 / 后台 / 最后标签、草稿保留、历史完整说明、多文件标签、空提交与快速 SHA 替换；浏览前后比较 HEAD、Index 条目及工作文件字节。2026-11-15 在英文浅色 1320×640 图形会话中通过。后续已补充并通过指定文件暂存、刷新后版本标签协调、草稿保留、单文件超限失败与同目标重试；写入后核对 HEAD、工作区及无关 Index 条目。同一脚本现已通过历史文件方向键保持 SHA、设置窗口隐藏 / 恢复补丁与文件列、整体隐藏 / 恢复详情，以及两个隔离 checkout 之间切仓库的标签清理。最终回归还覆盖折叠未暂存分组后刷新，确认活动标签、实际补丁和 Stage file 操作目标一致。2026-11-16 已补齐文件列表整体单一 Tab 停靠点；脚本分别验证工作区 / 历史 Files 的 Tab 离开与 Shift+Tab 恢复原行，原生控件测试另覆盖左右键可达分组 / 操作及 Space 绑定聚焦版本。上述为 macOS 实际验证，未覆盖其他操作系统或所有读屏行为。

`python3 -I scripts/native-narrow-shots.py` 最新续作重跑已生成 16 组、96 张截图：三种宽度 × 中英文 × 浅深色，另加四组 1.15 大字号；每组摘要 / 多标签 / 空提交 / 工作区 / 禁用提交 / 可提交表单。最新 `native-workbench-layout-smoke.py` 另验证三处分栏指针拖动和重启恢复，并在 Files 双击后核对偏好清除、默认宽度与其他分栏保留，第二次重启仍保持；Git 指纹不变，结果见 `build/workbench-layout-smoke/result.json`。全部组的快照几何核对通过（文件列含分隔线 ≥180 DIP，详情 ≥300 DIP），脚本检查活动标签和关闭入口显露；检查了窄窗大字号和宽窗长正文的代表截图。最新确认保护修复后再次完整重跑通过，日志 `build/lobe-latest-visual-matrix.log`，96 张图像及对应快照在 `build/narrow-shots-82hkcttz/`。脚本现每次创建独立的 `build/narrow-shots-*` 目录，保留以前的证据；工作台矩阵不代表全部页面或菜单交互验收。

原生适配器把 `ui.RoleOption` 输出为 `listItem`，标签选中态输出为 `checked`；无标签文本框可能仅输出 `placeholder=`。解析快照时不能要求每个控件都有引号包围的名称。

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

```sh
make lobe-e2e-build
python3 -I scripts/native-lobe-reference-smoke.py
```

2026-11-16 在 macOS 原生会话通过。新建临时工作仓库、本地 bare 远端、独立 HOME / Git 配置；通过坐标点击现有详情 footer，覆盖非当前分支 Push 取消 / 发布 / upstream（不 checkout）、分支重命名保留 tip 和跟踪配置、本地分支删除取消 / 确认且远端保留、附注标签推送保留原对象、本地标签删除取消 / 确认且远端保留、远端分支删除取消 / 显式租约确认并清理跟踪引用。每段核对主 HEAD / Index / Unicode 文件字节与干净状态。日志、最终快照和截图在 `build/lobe-reference-smoke/`，最终截图已查看。

引用脚本还覆盖 35 个长 Unicode 分支、双远端、无结果 Enter 不作用于旧行、鼠标仅选择及聚焦 Enter checkout。`SUPERGIT_REFERENCE_WINDOW_SIZE=980x640` 可验证最小窗口；`SUPERGIT_REFERENCE_FONT_SCALE=1.15`、`SUPERGIT_REFERENCE_THEME=dark` 仅调整隔离 fixture 偏好。脚本在最终快照检查顶栏操作边界与顺序、页面标题 / 搜索 / 新分支入口不重叠。修复前快照能使断言失败；980×640 英文浅色、1320 标准浅色及980 大字号深色均已完整通过。大字号截图另发现说明栏高度不足，修复后原生断言核对换行说明不越过引用列表起点；全量 / 构建 / 引用及工作台再次通过，最终大字号截图已查看。这些组合不等同全矩阵验收。

首轮在成功推送后因本地 topic 和 origin/topic 共享标签而选行不明确；按行的本地 / 远端身份区分后完整重跑通过。强制推送与 prune 仅从系统右键菜单进入，当前 MyGo automation 文档明确不支持选择该菜单，因此这两项仍保留专项原生验收；不通过新增产品按钮规避工具限制。stale 对象拒绝已有真实 Git / Go UI 回归，本脚本未新增相应原生竞态场景。

同日扩展并通过无冲突 Merge / Rebase 的 UI 发起：创建双方分叉后从详情 footer 取消 / 确认 Merge，核对两个父提交和两组文件；另建分叉 source / onto 后取消 / 确认 Rebase，核对新父提交、完整源说明与双方文件，保持 main 和干净状态。完整引用脚本重跑通过，最终截图已查看。冲突编辑及 Revert 发起不包含在这两个无冲突场景内。

## Lobe Stash 的隔离验证

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

## 定位元素：先读 bounds，不靠目测

- `snapshot --json` 给出每个节点的 `bounds`（DIP）。悬停、点击都用它算坐标；历史记录里多次出现「目测偏了一截」导致的假失败。
- ref 会随重绘失效：每次界面变化后重新 `snapshot`，不要复用上一帧的 ref。
- `click <sel>` 优先；`press`/`focus`/`fill`/`act` 按元素操作。确实要按坐标时用 `click <x> <y>`、`pointer move <x> <y>`。
- 截图是逻辑尺寸的 2 倍（Retina）。从截图量出来的像素坐标先除以 2 再换算成窗口 DIP。

## 右键菜单

区分应用绘制的菜单与系统上下文菜单。应用绘制的菜单可沿用已经通过的路径：聚焦目标行，`key Shift+F10` 打开，再从新的 `snapshot` 找到 `menuItem` ref 并 `press`；每次重绘后重新读取 ref。

当前安装的 `mygo-automation skills` 明确说明系统上下文菜单不能选择，并要求不要用 `click --button right` 打开。不能把截图后猜坐标点击作为受支持的替代路径。Force / prune 实际使用应用绘制的菜单，已由 `native-lobe-force-prune-smoke.py` 通过 focus + Shift+F10 路径原生验证，不能归入系统菜单限制。应用内菜单可在打开后从当前快照取得 DIP bounds 做坐标激活；这与系统菜单不同，也不能仅凭 semantic press 宣称鼠标可达。不要为绕过驱动限制新增产品入口。驱动升级后先重新阅读该版本的 guide，再决定是否扩展自动化。

`python3 -I scripts/native-lobe-menu-access-smoke.py` 在独立 35 长分支 / 双本地远端仓库、980×640 窗口下核对历史菜单全部 18 项 bounds 和 Down 焦点遍历，并实际坐标打开分支 picker / 底部 Reset、用 Enter 打开 Reset，关闭后核对 Git 状态未变。结果保存在每次独立的 `build/lobe-menu-access-*`；最新完整结果 `build/lobe-menu-access-sfbuiw3v/results.json`，日志 `build/lobe-menu-access-recovery.log`。后续覆盖四种坐标过滤、正确查询与匹配 / 排除记录、搜索 Enter 保持 checkout，以及无结果与读取失败区分：仅移开自建 fixture `.git`，Refresh 显示旧内容提示，finally 恢复后坐标 Retry 成功重读。picker 用 Escape 关闭，Reset 用 Cancel。最新 `build/lobe-menu-access-vmz51dcg/results.json`、`build/lobe-menu-access-dialogs.log` 进一步通过全部 11 个表单 / 确认入口的坐标打开与取消，加四种过滤共 15 项鼠标激活。该脚本不执行确认后的 Git 写操作；三项复制命令只验证 bounds / 键盘焦点，未实际点击以免覆盖当前系统剪贴板。

## 滚动

- `wheel` 一次只移动约一两行，滚到内容底部很慢；验证滚动相关行为时优先拖滚动条（按下、移动、松开三步），并断言滚动后的内容与选中态。
- 需要程序化滚动时用 `wheel x y dx dy`，注意指针位置要落在目标滚动区内，否则滚的是页面外层。

## 等待与判定

- 用 `wait <sel>`、`wait --text <s>` 等界面内容出现，而不是固定 `sleep`；它们只保证界面出现，不保证后台 Git 完成。
- `--settle` 等「窗口把自己请求的帧画完」，默认 10s；它不等应用自己设的计时器或后台任务。带 `--settle` 的动作会打印与上次快照的差异，`(no changes)` 也要读，别只看退出码。
- 后台任务失败时界面会出 toast/错误块；先找它，再判断操作是否成功。
- 验证写操作后回到 shell 用 Git 核对（`git status`、`git diff --cached`、`git config --get` 等），不只信截图。

## Lobe 工作台验收脚本

先运行 `make lobe-e2e-build`；以下脚本在 macOS 图形会话中串行运行，使用临时仓库和独立 HOME，不更新已安装应用。

- `python3 scripts/native-detail-tabs-smoke.py`：标签 / 草稿 / 偏好、真实 hook 拒绝与 Commit / Amend、文件和批量操作范围、取消确认及 Index 保护；再验证 2,405 条主历史和 2,403 条文件历史自动完整加载、End 揭示最早行，以及浏览前后 HEAD / Index / 工作文件不变。文件历史入口使用文件行聚焦后 `Shift+F10`，选择辅助功能树中的 `File history…` 菜单项。截图与日志保存在 `build/detail-tabs-smoke/`。
- `python3 scripts/native-narrow-shots.py`：980 / 1320 / 1600 宽 × 中文 / 英文 × 浅色 / 深色，加四组 1.15 字号组合；每组捕获摘要、多标签、空提交、工作区、禁用 / 启用提交。只复查某组时可设置 `SUPERGIT_SHOT_CASES=980-zh-light-large`，单组通过不代表完整矩阵通过。失败保存截图和辅助功能快照。
- `python3 scripts/native-workbench-layout-smoke.py`：真实指针拖动三处分界，核对偏好写入，重启后核对尺寸恢复和 Git 状态不变，结果在 `build/workbench-layout-smoke/`。
- `python3 -I scripts/native-lobe-window-dialog-smoke.py`：临时脏仓库与独立 HOME，分别先开设置重置确认 / 主窗口 Discard 确认，再从另一窗口请求确认；验证先开的确认保留、只在 owner 窗口显示、两边取消均保留 HEAD / refs / 原始 Index / 状态 / 工作字节。另验证主窗口 New branch 草稿与设置重置确认同时存在，取消设置确认保留精确草稿，再取消命名不写引用。每次证据位于独立 `build/lobe-window-dialog-*/`。不执行重置或丢弃，不证明真实系统窗口焦点切换。等待命名框使用可见标题，`wait --text` 不匹配字段 placeholder。

- `python3 -I scripts/native-lobe-reset-smoke.py`：四个独立临时脏仓库 / HOME，历史行 `Shift+F10` 选择应用绘制的菜单，验证 Soft / Mixed / Hard Reset 与 Undo 的取消、Hard 二次确认及实际 HEAD / Index / 工作字节和未跟踪二进制保护。加 `--stale` 时改验确认打开后外部 HEAD 推进的拒绝，Hard 在二次确认后推进；核对本地化恢复说明和完整外部新状态不变。证据位于 `build/lobe-reset-*/`；不操作源仓库，未覆盖同 SHA 切分支、进行中操作替换或跨进程原子互斥。

- `python3 -I scripts/native-lobe-force-prune-smoke.py`：独立临时脏仓库 / HOME 与本地裸远端，顶栏 focus + Shift+F10 打开应用内 Push / Fetch 菜单；验证 Force 的源 / 租约文案、取消、分叉替换、远端推进并 fetch 后旧租约拒绝、详情内当前语言说明与 Git stale info；prune 只删过期跟踪引用，保留同名本地分支、HEAD / 原始 Index / 暂存与工作字节 / upstream。长错误必须先打开 View details，不能直接等待被折叠的正文。证据位于 `build/lobe-force-prune-*/`。

- `python3 -I scripts/native-lobe-integrate-smoke.py`：两个独立临时仓库 / HOME，历史行 focus + Shift+F10 → Cherry-pick / Revert，验证取消不写入、所选 SHA、新提交父提交、完整说明 / 原 SHA、准确 Unicode 路径文件字节、topic 引用保留和无残留进行中操作。证据位于 `build/lobe-integrate-*/`；只覆盖无冲突发起，不是应用内冲突编辑。

自动化 `window w1` 只改变命令目标，不改变原生键盘焦点；当前确认框固定到发起动作的 owner 窗口，切换命令目标不会迁移它。`focus` 与坐标输入也只把事件送给目标内容，不调用系统窗口激活。旧脚本的 `window auto` 只适用于实际焦点恰为确认 owner 的场景；双窗口验收应分别读取两个窗口，并向 owner 窗口的新鲜按钮 ref 回答，不能把命令目标切换当作真实系统焦点切换。
