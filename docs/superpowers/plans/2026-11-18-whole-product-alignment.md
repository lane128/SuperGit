# 整体产品对齐：按完整使用流程推进

**状态：** 2026-11-18 用户明确纠正优先级后，暂停 Pull 快捷键别名支线。产品/设计/测试三角色已复用到混合改动拆分提交：普通新文本文件的部分暂存/撤回、真实 Git 的工作台两次提交旅程及提交失败恢复已实现并通过针对性验证。本文不代表整体对齐完成，不授权安装、打包、commit 或发布。本轮完整检查 `b03z7p96m` 已完成并接受：主会话已读取完整日志、源码验证 JSON 和任务输出，ordinary/race/vet 全部通过，541 个输入无变化/新增/移除，8 个包一致，检查及差异检查退出码均为0；生产输入冻结结束。前缀为 `build/alignment-new-file-split-commit`。此前放弃改动完整检查 `bmpzbp8g0` 的通过结论未用于替代本轮检查。

## 工作方式纠正

此前过度依赖 Sublime Merge 键位资源和小范围自动化可验证性来选任务，导致完整产品体验没有被持续推进。快捷键回到效率补充的位置；核心流程可靠性、在应用内完成任务的能力、工作台信息结构和真实操作体验优先。

五分钟任务已改为推进完整场景，复用现有产品、设计、测试角色及已有证据，不再默认每轮重新审查一组快捷键。新任务 `e71b6ad3` 替代 `86f029d7`，仍为当前会话内任务，最长七天；不重复启动已有完整检查。

## 证据规则

- 以 `cmd/supergit-lobe` 与当前 `internal/lobedesk` 为评审对象。旧 desktop 能力表、旧安装版及旧构建截图不能代表当前版本。
- 分别记录功能实现、实际控件测试、当前构建渲染、原生完整旅程和目标平台运行；全量测试通过不替代产品验收。
- Sublime Merge 资源声明可证明命令存在，不能证明完整布局、鼠标手势、焦点或工作流程完全一致。
- 当前原生观察有权限/系统焦点边界，不绕过，不盲发输入；受阻场景明确留待原生验收，独立临时仓库的确定性集成与离屏渲染可继续。
- README、TODO 和旧能力差距表含不同日期及两套界面的状态；不能把其中旧的“尚无入口”直接当作当前缺失。

## 初步差距与优先级

| 优先级 | 完整场景中的问题 | 当前证据 | 下一步结果 |
| --- | --- | --- | --- |
| P0/P1 | 连续选择、部分暂存、填写消息与提交的可靠性尚未收敛 | 日常原生链曾通过；`docs/e2e-automation.md:621` 仍保留首次提交输入触发重读的时序失败，复跑通过不解释原因。默认 SDK 的停帧问题也仍有未接入的隔离候选记录，两者不能未经证实归为同一原因 | 把当前源码的连续旅程绑定到构建指纹；精确定位读取/写入前提与失败，不用重跑掩盖问题 |
| P1 | 混合改动拆成两次提交的能力边界 | 本轮已开放普通新文本的按行/片段暂存、部分/全部撤回；实际工作台控件与真实 Git 已完成两次精确提交及 Index 预览。二进制/符号链接/删除/重命名/模式转换保护保留，未跟踪预览仍有 1 MiB 上限 | 本轮541输入固定源码完整检查已接受；当前软件渲染与 headless 旅程不替代原生连续操作和跨平台验收 |
| P1 | Lobe 内组合冲突编辑及完整旅程 | 当前源码已实现Base/双方比较、结果输入、快照校验保存；三尺寸实际控件→精确stage0→提交双亲旅程及针对性race通过。原文件菜单Mark resolved仍只暂存已保存工作内容 | 修正语言标签后的546输入完整检查bgydlhn8j已接受；原生/目标平台与长文件实际体验仍待验收，不能称完全对齐 |
| P1 | 分支创建→开发→合并→发布还没有当前构建统一验收 | 已有分散的分支与同步原生证据；新命令面板/快速检出的部分自然焦点前提受阻 | 用同一临时仓库串起实际 UI，逐步验证分支、Index、提交父序与本地 bare remote OID |
| P1 | Windows/Linux 与真实认证/网络恢复仍有交付证据缺口 | 已有跨平台编译及 macOS 布局/本地远端结果；不等于目标系统、认证或真实网络运行 | 单独列平台与环境验收，不用编译或本地 remote 测试关闭 |

Modified 与 Untracked 跨组选择会重建集合，是当前明确规则，产品角色认为它增加混合提交的操作步骤。它暂列为流程摩擦候选，不作为已确认的选择错误，也不直接改变 Staged/Unstaged 身份隔离。

设计角色核对了当前主工作区，三栏骨架已存在，不需要未经证实地重做布局。五个体验候选均来自源码而非当前渲染/原生验证：文件列表平铺而非文件树，跨目录定位需验证；Changes 的提交编辑区固定分配文件栏下方空间；长 Summary 正文与补丁共用滚动区；长行 Diff 依赖横向往返；同步和批量动作较依赖图标/悬停说明。这些列为完整场景中的观察点，不直接视作已确认布局缺陷或获批新增控件。下一步至少取得当前构建 Changes、历史合并节点 Summary、单文件长行 Diff 三种状态的渲染证据，再决定必要调整。

## 下一阶段交付：混合改动拆成两次提交

**演示起点：** 一个 owned 临时仓库，含已跟踪文件的多处修改、一个包含多段内容的新文本文件、未选控制文件和完整提交说明。通过当前 Lobe 界面进入工作区并辨认文件及版本。

**操作链：** 查看修改 → 选择本次内容 → 部分暂存 → 阅读暂存结果 → 填写摘要/正文 → 提交 → 在历史确认准确内容 → 对剩余改动重复一次。

**演示终点：** 两个消息与树内容准确的提交；第一次没有夹带剩余内容，第二次没有丢失剩余内容；无关文件、未选改动和分支身份保持预期；操作后选择、焦点与结果可见性可继续使用。失败必须保留草稿和有效改动，并给出可恢复路径。

第一批调查聚焦这个场景的已确认阻碍和时序失败，而不是继续 Pull 别名。新文件部分暂存对符号链接、二进制、文件模式、空文件、首次提交及特殊路径的边界需先核对；不能直接取消 `CanStageHunks` 的保护条件。涉及更多能力的调整按实际证据拆成可独立验证的实现步骤。

现有日常工作链、分支及冲突脚本可复用为扩展基础；不要求通过一个巨型测试代替各段的精确断言。后续冲突旅程接在此场景之后，不将当前缺失三方编辑悄悄算入已完成。

## 当前交付与限制

- 历史文件多选及 Backspace 放弃流程最新修复各自的固定源码完整检查均已接受。后者 ordinary Lobe 152.412 s、race Lobe 490.992 s，vet 与源码/差异检查通过；不复用这些检查认定未来功能已通过。
- 三角色已转向整体流程：产品负责真实能力缺口与步骤成本；设计负责工作台组织和阅读/操作层级；测试负责连续旅程与证据缺口。
- 未改 Git 依赖或 SDK 缓存，未安装、打包、commit 或发布；现有未提交改动保留。

## 本轮实现与旅程证据

普通新文本文件的 `new file mode 100644/100755` 现在可按行/片段暂存。`selectedPatch` 正向保留创建语义；反向撤回部分内容时改为修改已有 Index 路径，撤回所有新增行时删除该 Index 条目，让文件回到 Untracked。所有操作保留工作文件字节。`DiscardLines` 及界面新文件按行放弃的禁止条件仍保留，避免暂存能力意外扩大破坏性操作。

先以 `TestNewTextFilePartialStageAndUnstage` 在六种已有/unborn仓库和路径组合复现拒绝，再实现；真实 Git 断言部分 Index、撤回后 Index、全部撤回后路径消失、无关暂存条目及工作区不变。安全覆盖包括新增/已有 symlink、binary、empty、1 MiB 上限、可执行模式、无末尾换行限制、外部工作区/Index变化、CRLF 和 UTF-16LE 属性归一化。普通新文件实际控件的 Stage line 与 Unstage line 菜单均有测试。

独立测试审查发现两项新可达问题，主会话均先复现再修复：

- 新文件被外部文本或 symlink 替换后，旧创建片段仍会写入旧内容。`ApplyHunk` 现在在写锁内重读正确方向的 Diff，要求当前仍允许片段操作且补丁与当前 hunk 精确匹配。
- `diff.noprefix=true` 使 no-index 补丁路径不再符合 apply 的剥前缀规则。根路径会报错，嵌套路径还可能写入错误 Index 路径；no-index 现在与 tracked Diff 一样显式使用 `a/`、`b/` 前缀。回归使用 `dir/new.txt`，断言正确路径内容及错误根路径不存在。

`TestMixedChangesSplitIntoTwoCommits` 使用当前三栏工作台的实际鼠标选择、Diff 行菜单、摘要/正文输入及 Commit 按钮：新旧文件先各暂存第一部分，在 Staged 版本确认 Index 不含剩余内容，填写完整消息并提交，再提交剩余部分，最后进入 History。断言两次消息、文件内容、父序、完整最终路径、refs、未选控制文件与工作字节。起点仍是测试已打开的仓库，不包含系统选择目录；这是 headless 控件/真实 Git 证据，不是原生窗口输入验收。

提交完成逻辑还发现独立的错误分类问题：Git hook 已拒绝提交，但 worker 清理或晚到取消让 UI 回调误报为取消。`TestRejectedCommitKeepsDraftIndexAndFailure` 先 RED；`a.commit()` 现复用 `performDoneAs` 的结果快照及主线程投递。测试覆盖普通提交与 Amend：拒绝后保留真实错误、草稿、模式、HEAD 和 Index，修正临时仓库 hook 后使用原稿重试成功。此修复不被用来解释此前原生提交输入触发重读的未知时序问题。

当前旅程软件渲染图位于 `build/alignment-split-commit-shots/`：`first-index-review.png`、`first-logical-change-ready.png`、`second-logical-change-ready.png`、`completed-history.png`。设计角色实际查看后确认文件版本、Index 内容、消息与提交按钮可辨认。首次截图冻结在100ms颜色过渡首帧而误显浅色按钮，已依据 SDK 的固定测试时钟规则推进150ms后重新取图，稳定态为绿色白字；未因此修改产品配色。History hunk 标题右缘截断保留为阅读体验候选，本轮不扩大布局。

产品角色核对本场景后没有报告新的确认阻碍；测试第二轮仅审修复与影响，未发现新确定缺陷。仍未直接覆盖 `a.commit()` 执行中取消和成功后晚到取消；共享操作生命周期已有相应一般操作覆盖，原生与跨平台边界仍开放。

本轮针对性 Git 测试通过7.936s；核心两包针对性 race 通过 Git 10.558s、Lobe 14.219s。其后新增属性归一化、UI撤回与普通/Amend失败重试的针对性测试也通过。最终 `b03z7p96m` 固定源码完整检查已接受：`go test ./...` 的 Git 81.016s、Lobe 158.456s，`go test -race ./...` 的 Git 88.238s、Lobe 522.473s，`go vet ./...` 无错误；其余包通过或使用有效缓存。541个输入不变、8个包一致，源码/差异检查均为0。完整日志和验证文件均已读取；独立 overlay 冲突旅程仍单独记为针对性证据，不归入此全量结果。

## 冻结期间的下一条完整旅程：分叉、冲突与推送

在完整检查 `b03z7p96m` 运行期间，没有改动冻结源码或重复启动全量检查。独立调查写在忽略目录 `build/_keyboard/remote_conflict_journey_probe_test.go`，通过 `remote-conflict-journey-overlay.json` 以 Go overlay 运行；它不属于本轮全量检查的源码输入和结果。

`TestRemoteConflictJourneyProbe` 已用当前工作台控件与真实 owned 仓库完成：

1. 本地与本地 bare remote 各有一个新提交；通过界面 Fetch，确认 ahead=1/behind=1。
2. 从远端提交的历史菜单打开合并确认，先取消并断言 refs/Index 不变，再重新打开并接受。
3. 同文件合并冲突后，进入 Changes，确认 merge 状态、冲突存在及 Commit 禁用。
4. 明确模拟外部编辑器保存组合结果，通过界面 Refresh、Mark resolved，断言 stage0 为精确结果且 merge 仍待完成。
5. 输入完整摘要/正文，通过界面 Commit，断言两个父提交的顺序及操作结束。
6. 经界面 Push 的目的地确认推送；断言 bare remote 的 main OID 与本地合并结果相同、工作区干净、无关已跟踪文件字节保持。
7. 进入 History 并实际选择合并结果，确认完整正文及双亲信息；不只检查列表里存在该标题。

首个有效探针 ordinary 通过3.733s。最终增强版等待概览 Diff 完成加载并实际选中合并结果后，race 通过20.939s，输出 `b8brfixh1` 已读取；六张当前源码软件渲染图位于 `build/alignment-remote-conflict-shots/`。`evidence.json` 绑定探针和图片SHA256，并对照全量检查的源码快照确认541个已有输入均未变化。此记录不是全量检查完成证明。

调查中确认的操作前提与证据纠正：

- `IntegrateChecked` 共用 `clean`，包括未跟踪文件在内的任何状态条目都会阻止合并。保留当前安全规则；探针改用干净起点与已跟踪控制文件，不把其拒绝当作合并算法缺陷。用户的脏工作区→Stash→合并→恢复仍是后续场景。
- 历史行中间可能是带独立右键菜单的引用胶囊。探针明确在提交行左缘打开提交菜单，避免误将 Copy branch name/View history 菜单当作合并入口丢失。
- 概览加载与单文件加载是独立状态，截图必须等待可见概览条目完成；最终 History 也必须实际选中合并结果，不能用旧远端提交的详情图证明合并结果阅读。

这证明现有“外部编辑解决冲突”流程可以串成闭环，并未补上应用内三方组合编辑。后续能力缺口仍是当前 Lobe 内直接比较 Base/Ours/Theirs 并编辑结果；可以复用已有 `gitrepo.Conflict`/`ResolveConflict` 的版本身份、会话及外部修改检查，但不能以旧 desktop 的编辑界面当作 Lobe 已实现。保存前保留草稿、拒绝陈旧目标、明确区分保存与完成 merge，是下一实现需验证的边界。

本轮沿用现有产品角色提出的冲突闭环顺序、设计角色的版本/动作层级及软件渲染证据规则、测试角色的精确 Git/未选文件/父链要求；没有重新发起同一轮整体审核。未启动原生输入、真实网络认证或Windows/Linux运行验收，未安装、commit、push到外部远端或发布。所有测试push仅发生在owned临时bare remote。

## Lobe 应用内冲突编辑：完整检查已接受

后续实现已补上普通文本的应用内组合编辑入口：从Changes概览或冲突文件预览进入Diff详情编辑器，查看Base/当前/传入，编辑工作区初始结果，再「保存并标记解决」。详情宽度>=660时双方并排；窄区标签切换。外层三栏保留，结果占据可用剩余高度，保存后merge仍待原提交入口完成。具体范围与决策见 `2026-11-18-lobe-conflict-editor.md`。

`TestConflictEditorWorkbenchJourney`通过当前三栏实际控件在900×650、900×900、1280×900完成概览入口→来源比较→输入结果→保存stage0→输入消息→Commit，验证精确内容、双亲和干净状态。新专项测试覆盖外部变化拒绝、标记拒绝、仓库间延迟读取隔离、草稿/来源切换、Show patch、二进制退路、modify/delete空结果拒绝及实际文件菜单删除选择。owned index.lock复现写文件成功但暂存失败，验证保留草稿、准确恢复提示、重读版本后重试。

结果区高度限制和短窗口长错误遮挡Save均先由实际控件断言复现，再修复。最终针对性race通过29.860s；两轮独立测试审查的第二轮未发现新确定缺陷。设计角色实际读取四张最终软件渲染，未见阻碍使用的截断，保存/完成merge区分清楚；1280截图选Base，不作为双方并排渲染证据。图片位于 `build/alignment-conflict-editor-shots/`。

首轮固定源码完整检查`bpl45fhia`的ordinary在语言约束`TestChineseStringsOnlyKeepTheListedEnglishTerms`失败（Lobe172.296s），未继续race/vet；546输入无变化，8包一致。四条新中文来源标签带未允许的ours/theirs，现已完整译为中文，保留原术语约束；语言/编辑器针对性通过8.536s。修正后的完整检查`bgydlhn8j`已接受，前缀`build/alignment-lobe-conflict-editor-zh`；完整日志、任务输出与源码验证均已读取。ordinary Lobe174.341s、race Lobe551.859s、vet无错误，其余包通过/有效缓存；546输入无变化/新增/移除，8包一致，检查/差异检查退出0，冻结结束。此结果只覆盖该版本；后续Stash读取源码调整另行验证。原生输入、真实认证/网络、Windows/Linux运行仍未验收；没有项目commit、外部push、安装或发布。

## 冻结期间独立场景：Stash → 合并 → 恢复混合改动

忽略目录`build/_keyboard/stash_merge_journey_probe_test.go`和`stash-merge-journey-overlay.json`通过Go overlay接入额外测试，不改冻结源码。owned临时仓库含本地/传入分叉、同一notes.txt中的已暂存和未暂存内容、scratch.txt未跟踪文件及无关控制文件；开始时已有完整提交草稿。

实际工作台操作验证：Changes查看混合状态→Stash保存菜单选择包含未跟踪→输入说明后先取消，断言Index/工作字节不变→再次保存，核对Stash的Index父树、工作树和未跟踪父树→History传入提交菜单确认合并→Stash实际Apply→回到Changes。断言精确恢复`MM notes.txt`和`?? scratch.txt`、原暂存/工作区分界、无关文件、合并内容、双亲及main身份；Apply保留Stash，提交草稿保留。再次实际Apply被Git拒绝，验证refs/Index/工作字节/Stash不变且保留原始Git错误反馈。

初始ordinary通过2.643s；最终增加重复Apply失败保护后race通过15.554s。五张软件渲染与证据位于`build/alignment-stash-merge-shots/`，`evidence.json`绑定探针/图片SHA256，对照当前完整检查的快照确认546个源码输入均未变化。这是独立针对性证据，不归入bgydlhn8j全量结果。

明确缺口：`loadStashDiff`仍直接使用原生`mygo.RunOnMain`，未接入现有headless回调入口。探针诊断曾停在`stashDiffLoading=true`，其他加载/写入均结束；当前独立探针不等待这个无法接收的原生回调。第04图准确保留Loading diff，证明动作入口可用，不证明Stash补丁阅读完成，更不证明原生加载故障。不能为让测试通过而伪造已加载Diff；后续需收敛此读取生命周期并补上当前版本的补丁阅读证据。以上仅描述冻结时初版普通恢复证据；后续Pop冲突旅程及源码交付见下节。

## Stash 完整恢复旅程：补丁阅读、冲突解决与精确提交

独立overlay已扩展为Pop冲突链，最终两条独立race通过39.607s，仍绑定旧546输入且不归入bgydlhn8j全量结果。实际发现Git在失败的`stash apply --index`后将notes.txt的整个工作版本放入Index，包含原未暂存行。旅程明确进入Staged版本，通过实际Unstage line撤回那一行，再提交解决后的base.txt与原计划暂存的notes.txt内容；验证单亲为已完成的merge、完整消息、精确提交树、剩余未暂存notes和未跟踪scratch、无关文件及保留的Stash恢复副本。并未声称冲突Pop会自动恢复原暂存边界。

在bgydlhn8j接受并结束冻结后，`loadStashDiff`复用现有`hopToMain`，同时补充非nil仓库和Root归属检查。新实际控件测试先复现回调不投递，再在回调可交付后复现切仓库的旧结果覆盖详情；两者修正后通过0.892s。此项让读取可被同一旅程验证，不能拿之前headless的等待当作原生加载故障证明。

两条旅程已整合为常规源码`internal/lobedesk/stash_journey_test.go`，取消overlay初版的补丁等待豁免，必须等待真实读取并断言渲染包含原暂存/未暂存/未跟踪内容。`stash_detail_read_test.go`保留独立读取和迟到回调回归。新源码针对性race通过41.985s（普通旅程14.41s、Pop恢复25.47s），独立只读核对没有报告确定问题。

当前软件渲染位于`build/alignment-stash-complete-shots/`，主会话已读取真实Stash补丁和修正Index的截图；这些与旧`alignment-stash-merge-shots/`中的Loading图属于不同源码证据。Stash补丁是合并工作补丁，原暂存边界另由Stash第二父及恢复后Index核验，不能只靠这张补丁图判断。

完整检查`b1w8qnfx4`已接受，前缀`build/alignment-stash-complete`。主会话读取完整日志、源码验证JSON及任务输出：ordinary Lobe172.009s、race Lobe555.893s，vet无错误；其余包通过/有效缓存。548输入无变化/新增/移除、8包一致，check_exit=0、diff_check_exit=0，冻结结束。此结论覆盖Stash读取及两条常规完整旅程，不覆盖后续多文件解析修复。原生、目标平台、真实网络/认证仍开放；未commit、外部push、安装或发布。

### 冻结期间确认的后续阅读问题

主会话读取新Stash实际补丁截图，发现第二文件的`--- /dev/null`和`+++ b/scratch.txt`带增删底色及代码行号。独立`build/_keyboard/stash_header_probe_test.go`用真实Git保存多文件Stash，经`stash-header-overlay.json`复现：两行被解析为del/add，旧/新行号分别2/3，错误归属前一hunk；首个hunk.Patch还包含后一文件的`diff --git`头。日志`build/alignment-stash-header-probe.log`明确RED（0.480s）。真实代码内容以`++`开头时仍需保留add语义，不能通过 blanket `+++`/`---`过滤来修复。

当前`parseDiff`只在首个hunk前识别文件头，未处理多文件边界；StashDiff对合并补丁直接调用它。当前Stash视图禁用片段动作，此处确认的是多文件解析/阅读错误，不声称已发生错误Git写入。冻结期间不修改生产解析器；完整检查结束后应修复或为Stash采用明确的逐文件只读解析，再验证多文件头、真实以加减号开头的代码、每文件hunk归属和现有单文件部分暂存安全。此RED overlay不是常规full gate输入，不与该gate结果混记。

冻结期间已在`build/_keyboard/stash_readonly_candidate_test.go`验证窄范围候选：对现有`splitFileDiffs`逐文件结果合并只读展示，原始Text不改，各文件Hunk.Patch保持自己的文件头，展示行的Hunk按全局序号重排，整份预览CanStageHunks=false。真实Git样本含修改、未跟踪、删除、重命名、二进制、空格/中文/换行路径及代码前导加减号；候选race通过1.535s。此为临时test内实现，不是当前StashDiff产品修复，也不是完整gate结果。

`build/alignment-stash-readonly-candidate-evidence.json`绑定RED探针、GREEN候选、overlay和日志的SHA256；对照当前gate源码快照，548输入均未变化。下一步在b1w8qnfx4结论到达后，将该窄范围只读解析接到StashDiff，并在实际Stash阅读/恢复旅程中验证第二文件头不再显示代码行号和增删色；不改共享单文件暂存解析器来扩大风险。

### 紧凑窗口的同一完整场景

当前完整检查尚在运行时，以`build/_keyboard/stash_narrow_journey_test.go`替换仅测试文件的overlay，验证同一源码工作台：900×900普通保存/阅读/合并/Apply恢复通过3.275s；900×650普通恢复和Pop冲突→编辑结果→修正Index→精确提交两条旅程均通过，合计6.822s。交互控件断言包含Offscreen、禁用、非零尺寸和窗口横纵边界，未因宽度较小跳过动作或伪造补丁。

主会话读取900宽实际Stash补丁截图，长标题省略，Apply/Pop/Drop完整可见并可操作，不据此追加布局改造。第二文件头误分类仍存在，紧凑窗口通过不关闭该阅读问题。截图位于`build/alignment-stash-narrow-shots/`、`build/alignment-stash-900x650-shots/`；`build/alignment-stash-narrow-evidence.json`绑定overlay/日志/图片及548输入无变化。当前overlay文件为900×650版本，900×900日志/图片为前一尺寸运行的独立证据；均不归入常规full gate或原生验收。

## Stash 多文件阅读修复已接入，完整检查运行中

b1w8qnfx4结论接受后，将已验证候选落实为`internal/gitrepo/stash_diff.go`的`parseStashDiff`，由`StashDiff`调用。继续复用现有逐文件解析，保留原始Text、各文件自有Hunk.Patch和全局展示行Hunk索引；整份预览CanStageHunks=false。共享单文件解析和部分暂存代码未修改。

`TestStashDiffPreservesFileHeadersAndHunkOwnership`先对实际StashDiff复现错误，修复后通过0.621s；覆盖多种文件/特殊路径、真实前导加减号代码、metadata无代码行号、每片段单文件归属、读取不改变Index/refs/工作状态。原始字节比对最初因测试命令未匹配应用的core.quotepath=false而失败；已统一测试命令配置，未改变应用路径处理。

实际Stash读取→合并→普通恢复和Pop冲突→编辑→修正Index→精确提交两条常规旅程增加文件头metadata/只读断言。针对性race通过Git10.814s（含相关按行和新文件安全回归）、Lobe40.683s。主会话读取`build/alignment-stash-file-boundaries-shots/04-review-saved-patch-before-restore.png`，确认第二文件头不再带增删色和代码行号，实际代码行仍正常着色。

新固定源码完整检查`b8vnrkmy7`已完成并接受，前缀`build/alignment-stash-file-boundaries`。主会话已读取任务输出、完整日志及源码核对JSON：550输入无变化/新增/移除，8包一致，check_exit=0、diff_check_exit=0；普通测试Git82.988s/Lobe170.889s，race Git90.952s/Lobe566.054s，vet通过。该结果覆盖本轮Stash多文件解析修复，不包括ignored overlay的分支和长行调查。冻结结束。原生、目标平台和真实网络/认证验收仍开放；保留既有未提交工作，未commit、外部push、安装或发布。

## 冻结期间：完整分支生命周期独立验证

独立overlay `build/_keyboard/branch-lifecycle-overlay.json`新增临时测试文件，不修改生产输入。`TestBranchLifecycleWorkbenchProbe`使用自有临时仓库和实际1280×900工作台控件，完成侧栏创建并切换topic→编辑文件（测试输入）→实际按行暂存→输入提交说明并提交→侧栏切回main→合并→删除已合并topic。分别验证快进合并和main/topic各自提交后的双亲合并，后者精确检查父链顺序与两边提交内容。

创建、合并和删除均先取消再执行，取消后检查refs/HEAD不变；最终验证main身份、精确提交内容、干净状态、只剩main本地分支和保留标签。所有Git变更均在测试创建的临时仓库；本项目refs未修改。本轮未发现阻断该流程的新缺陷，没有因此改动产品代码。

普通初始快进旅程通过3.560s；最终两变体race通过41.350s（16.85s/22.96s），日志`build/alignment-branch-lifecycle-race.log`。保存8张软件渲染图于`build/alignment-branch-lifecycle-shots/`；主会话已读取分叉合并确认和清理后画面，确认对话框控件可见、清理后侧栏仅main且工作区干净。`build/alignment-branch-lifecycle-evidence.json`绑定本轮550个冻结输入，复核changed=[]。该overlay不属于当时运行的完整检查，也不代表原生焦点或Windows/Linux验收。

## 长行阅读调查及停止本轮自动推进

本轮独立overlay `build/_keyboard/history-long-line-overlay.json`通过真实临时仓库和History提交→long.txt→横向滚动→short.txt控件流程复现：约5000字符行实测34777.7 DIP，生产`diffWidth`上限20000，完整尾部无法容纳。初始RED日志`build/alignment-history-long-line-probe.log`，截图`build/alignment-history-long-line-shots/`。

仅在ignored候选副本移除上限后，强化实际右端与短文件可见性断言仍RED：宽度估算35954.1大于实际文本加gutter需要，滚到最右端时尾部落到视口左侧；切换short.txt后正文bounds为0×0。日志`build/alignment-history-long-line-visibility.log`准确保留两条失败。Git HEAD/Index/工作区未被阅读操作改变。后续隔离对照证实，短文件继承了长文件的横向偏移，虽然外层纵向滚动状态已归零。按详情标签隔离可显示短文件，使用实际字形宽度后可在最右端读到长行尾部。仅用控件键隔离会丢失切回长文件的阅读位置。为每个详情标签显式保留横向滚动状态后，长行尾部、短文件正文与切回位置的同一旅程通过。日志为 `build/history-diff-scroll-{control,keyed,exact,state}.log`。实验仍在 overlay 中。8,000 行工作流的软件渲染 benchmark 中，逐帧测量全部可投影的行从约 67 ms/帧升到 1.58–1.62 s/帧，不能直接集成。日志为 `build/history-diff-benchmark-{baseline,state}.log`。按文件段缓存测量值、仅对极宽行做精确测量的后续候选，在同一基准中为约 43 ms/帧，见 `build/history-diff-benchmark-fast.log`。ASCII 和中英混排测试的截图与日志见 `build/history-diff-scroll-visual-nRDW8R/`。macOS 原生窗口也完成了滚到长行尾部、切换到短文件、回到长文件原位置的旅程，Git 状态不变，见 `build/history-diff-scroll-native-RoQAlq/`。候选尚未集成到正式源码。

复用现有产品/设计/测试角色做本次窄范围核对；三方均要求补实际尾标记/短文件可见性，设计还读图确认短文件显示异常；未重新做整体审核或创建角色。本次源码未增加修改。

用户于2026-11-18明确要求本轮loop结束后停止并讨论计划。已取消周期任务`e71b6ad3`，不再自动续跑、扩展功能或启动新完整检查。阶段收口候选为可在本机完成日常Git的三栏工作台；验收覆盖修改/部分暂存/提交、分支生命周期、冲突处理、Stash及Diff阅读。最多两轮收尾（阅读阻碍；验证/构建/授权安装）的提案待讨论，不视为已经授权本地安装。Windows/Linux、真实认证/网络异常及进阶能力留作后续边界。本轮未构建安装、commit、push或发布。

## 顶部仓库标签与 History 宽度

同一窗口可打开多个 Git 根目录，每个标签持有自己的草稿、选区和读取结果。同一个 checkout 的子目录入口复用已有标签，已打开的不同 worktree 各有标签，关闭有草稿的标签需确认。返回标签时先核实 checkout 根目录与本地状态。未变化的仓库保留 History 前进/后退和分页，有变化才重读。外部更改分支或 HEAD 时取消原先的 Amend 选择并保留草稿。重读失败时仓库操作停用，保留旧内容与重试入口。切换窗口宽度会将活动标签滚动回视口。

History 详情收起后，提交列表与 Graph 按可用全宽重新分配空间，展开后恢复原分栏。980→1320→980 DIP 的软件渲染检查见 `build/repository-tabs-history-PuP1Ge/`。macOS 最终源码用真实 Git 分叉 fixture 从侧栏打开既有 worktree，切换 A/B、恢复两份草稿、收放历史详情、取消和确认关闭；两边 refs、符号 HEAD、Index、status 及 A 的文件字节保持原样。证据 `build/repository-tabs-native-final-2MxwaE/`，已查看 Changes 和展开 History 两张截图。系统文件夹选择器的第二仓库入口由 headless 注入选择结果覆盖；系统原生操作待验收。Windows/Linux amd64 已编译，运行验收待补。

首版 `build/repository-tabs-check` 的 552 个固定输入通过完整检查。随后复现并修复旧 Undo 写入后台仓库、同 SHA 外部分支切换导致 Amend 目标错误、刷新失败后 Pull 写入父仓库、History Back 丢失、End/Enter 误关标签与窄窗活动标签不可见。中间版本的全量检查发现主动创建/切换分支也会取消 Amend，已限于重新激活时检测到的外部变化，相关既有旅程复测通过。最终定向竞态测试通过 46.404s。集成 Quick Switch 前的 `make check` 全部通过，554 个输入和 8 个包在检查前后相同，证据 `build/repository-tabs-final-check-full-check.log` 和 `build/repository-tabs-final-check-source-verification.json`。

Quick Switch 十标签回归在生产源码复现了最近列表淘汰后找不到仍打开的标签。隔离候选保留最近记录顺序并补入其余打开标签，定向普通检查 8.499s、竞态检查 15.070s 通过。macOS 候选构建打开九个仓库后，在选择器搜索最早的仓库，返回原标签并恢复草稿，确认 Git 状态保持，证据 `build/repository-tabs-quick-native-up9hyY/`。候选和十标签回归已集成。选择器提示已更新为已打开或最近使用的仓库。集成后的 macOS 构建再次通过九仓库场景，证据 `build/repository-tabs-quick-integrated-native-icAkxi/`。集成后的全量普通测试、竞态测试、`go vet` 和 `git diff --check` 全部通过。555 个输入和 8 个包在检查前后相同，证据 `build/repository-tabs-quick-integrated-check-full-check.log` 和 `build/repository-tabs-quick-integrated-check-source-verification.json`。Windows/Linux amd64 集成构建见 `build/repository-tabs-quick-integrated-cross-iwGK5J/`。跨重启恢复已打开标签未实现。长行 Diff 问题与当前 Graph 宽度调整分别验收。

## 深轨选中节点与三角色验收

产品核对发现，本机 Sublime Merge Build 2125 的会话文件按顺序记录一个窗口的仓库标签和活动项。安装包更新记录确认多标签会话加载和恢复顺序。参考版实际重启是否恢复该活动项及草稿仍待实测。SuperGit 的原生测试在同名 linked worktree 的标签 `shared · North` 与 `shared · South` 间往返三轮，恢复各自 History 完整 SHA，Git 指纹一致。通过应用菜单正常退出并无参数重启，工作台为空。九标签原生测试在窗口放大后缩回 980×640，活动标签、关闭按钮与加号均可见，点击关闭后激活相邻标签。顶部加号确实弹出系统目录面板，辅助功能授权关闭，目录选择并确认未覆盖。

设计使用 Emil Design Engineering 与 Impeccable，确认窄窗中标题、引用和选中 Graph 节点需要保持对应。本轮增加了被宽度变化裁掉的节点恢复定位，手动平移移出的节点不会在缩放时被拉回。回归先失败后通过，包含大号节点和手动平移。定向竞态检查通过 26.682s。软件渲染在 `build/repository-graph-selection-resize-ogmomy/`，验证 980→1320→980 DIP 的同一提交。相同 14 分支原生场景的修复前和修复后截图分别见 `build/repository-graph-resize-native-baseline-b9DCZw/narrow-selected-lane.png` 与 `build/repository-graph-resize-native-PozXKd/narrow-selected-lane.png`。Windows/Linux amd64 交叉构建均通过。本机已安装的 SuperGit Lobe 为 0.1.1 build 8，本轮可执行构建见 `build/lobe-tabs-graph-current-7MCypm/supergit-lobe`。全量普通测试、竞态检查、`go vet` 与 `git diff --check` 全部通过，测试前后 555 个输入和 8 个包一致，证据见 `build/repository-tabs-graph-resize-check-full-check.log` 和 `build/repository-tabs-graph-resize-check-source-verification.json`。

## 正常退出与仓库标签恢复

无参数启动按上次退出时的顺序恢复 checkout，并优先读取原活动标签。其他标签在第一次选择时读取快照。恢复不触发启动 Fetch。显式启动路径单独打开指定 checkout，保留正常工作台的会话记录。找不到的路径或无法读取的活动 checkout 会跳过，活动项依次尝试右侧、左侧。关闭最后一个仓库标签会保存空会话。会话只保存标签根目录与活动项，提交草稿仅在进程内保留。

macOS 原生退出和重启分别使用四个独立仓库与四个 linked worktree。两组各通过十项断言，覆盖标签顺序、选中项、关闭标签、左右回退、显式路径隔离及 Git 状态。读取前后的截图验证活动标签稳定，加载只显示一条提示，分支名在就绪后显示。可复用探针与记录在 `build/repository-session-evidence-bwkIDv/`，最终结果分别为 `final-distinct-32te6v5f/result.json` 和 `final-linked-n7swfry_/result.json`。定向测试覆盖嵌套路径误回退、损坏索引、选择时保留另一进程新增的标签和最后一个标签关闭。Apple Silicon、Windows amd64 与 Linux amd64 均完成构建。

补充原生验收通过 `File → Close Window` 保存会话并恢复，结果在 `build/repository-session-evidence-bwkIDv/close-window-distinct-ft8l2f00/result.json`。全部 checkout 暂时无法访问时，应用进入空工作台并保留会话原文件。主动关闭全部标签后，下一次启动保持空工作台。两个场景见 `build/repository-session-evidence-bwkIDv/extra-startup-distinct-a22e9mq2/result.json`。同一构建复测 14 分支 History 从 980 DIP 放大再缩回，选中 SHA 与 Git 状态保持，截图中完整显示选中节点，证据在 `build/repository-session-graph-final-TSuQbb/`。最终普通测试、`go vet`、全量 race 测试和 `git diff --check` 通过。两次检查的 557 个源码输入与 8 个包前后相同，证据在 `build/repository-session-standard-final-source-verification.json` 与 `build/repository-session-race-final-source-verification.json`。

## Finder 预览包与标签溢出

`make lobe-app` 在 `build/SuperGit Lobe Preview.app` 生成 Apple Silicon 本地预览包。应用使用独立的 bundle identifier，plist 校验和 ad-hoc 签名验证通过。使用临时仓库和隔离偏好，经 macOS Launch Services 确认此 bundle 建立了真实窗口，记录见 `build/lobe-preview-launch-ogdXXs/result.json`。以下交互证据来自同一源码的原生自动化构建。

顶部新增仅在标签溢出时显示的「切换仓库」入口。原生用九个仓库验证窄窗显示入口、宽窗隐藏、缩回后通过入口找回最早的标签和草稿，原仓库 Git 指纹一致。十标签定向测试补充验证 Escape 返回焦点和 Enter 重新打开。记录在 `build/repository-switcher-native-OKcsQh/`。从顶部加号打开原生目录面板已验证，自动化受本机权限限制未确认第二个独立目录。独立仓库选择结果走已有测试接缝，原生双击侧栏既有 Worktree 则验证了打开第二标签并正常重启恢复。

History 原生截图在 980 和 1600 DIP 保持同一提交和 Diff，宽窗可显示完整分支名和日期。原生对比使用重启切换尺寸，独立 headless 渲染测试验证了同一窗口 980 → 1600 → 980 DIP 往返时保持选区并恢复列宽。两种证据都在 `build/repository-switcher-native-OKcsQh/history-qa/`。截图中 Diff 区块的 `@@` 标题在右缘被裁切。History 的 hunk 标题使用默认居中的 `ButtonBase`，文本定位随补丁内容宽度偏移，后续需要验证左对齐后在窄窗和横向滚动中的表现。

最终 `go test ./...`、`go test -race -timeout 15m ./...`、`go vet ./...` 和 `git diff --check` 通过。两轮测试前后 557 个源码输入与 8 个包一致。记录见 `build/repository-switcher-standard-source-verification.json` 和 `build/repository-switcher-race-source-verification.json`。Windows 和 Linux 的 amd64 交叉构建也已通过，产物位于 `build/repository-switcher-cross-O3y64G/`，运行验收待补。

## 欢迎页顶部入口与 History Diff 标题

欢迎页和关闭最后一个仓库后的空工作台复用同一标题栏 `+`。原生截图确认入口可见。测试用两个既有独立 checkout，通过注入系统选择结果，从空工作台依次点击顶部 `+` 打开两者，第二个追加并选中。原生自动化用两条已保存的根路径验证两个标签恢复，并在同一窗口放大再缩回时确认 History 保留选中提交和 Diff。目录面板内确认仍需本机人工验收。

History Diff 的 `@@` 标题采用与工作区补丁相同的起始对齐。修复前，300 DIP 测试窗中的标题文字从 x=245 开始，在右边缘裁切。修复后，300 → 420 → 300 DIP 可完整阅读，右键「复制片段」返回完整文本。独立工作流在 980 和 1600 DIP 验证摘要与单文件标签，两份 fixture 的 Git 指纹保持一致。原生截图、控件快照、结果 JSON 和独立软件渲染记录位于 `build/welcome-tabs-history-native-mj49p_hg/`。

完整普通测试及 `go vet` 在 557 个未变化的源码输入上通过，仓库标签、会话和 History 标题的定向 race 检查通过。Apple Silicon 预览包已重建并验证签名，Windows/Linux amd64 交叉构建通过。

## History 长短补丁切换

使用真实 Git fixture 在 980 和 1600 DIP 复现两个问题。长行末尾超出可滚动范围，切到短文件时正文也随旧横向偏移移出视口。History 按当前详情标签的文件段计算补丁宽度并缓存测量，横向位置由各自标签持有。最新回归打开同一提交的长文件、滚到尾部、读短文件、返回长文件。覆盖 Tab 字符和 CJK 文本，验证短文件在能放下时没有横向滚动，Git HEAD、Index 和状态保持一致。

macOS 原生通过同一长短文件旅程。截图见 `build/history-file-reading-native-VzNzVz/`，包含长行完整尾标记、短文件开头及返回原位置。Apple Silicon 预览已构建并验证签名，Windows/Linux amd64 交叉构建通过。完整 `go test ./...`、`go vet`、定向 race 和 `git diff --check` 通过，完整检查前后 558 个源码输入不变，见 `build/history-file-reading-check-source-verification.json`。8,000 行软件渲染基准在此构建约 46.8 ms/帧，记录见 `build/history-file-reading-bench-pcO4W7/result.log`。

产品核对本机 Sublime Merge 2125 的命令资源与路由，确认从现有仓库直接打开第二个仓库与先创建空白标签是两个流程。

## 新建仓库标签与 History 宽度

顶部 `+`、macOS 的 File → New Tab 和 `⌘T` 创建并选中 Welcome 标签。通过 Welcome 打开仓库会填充当前标签，打开已加载仓库的根路径或子目录会选中原有标签。两独立 checkout 回归覆盖取消、无效路径重试、草稿隔离、关闭空白页、仓库顺序和活动仓库恢复。缩窄并恢复窗口后仍可阅读原文件补丁。

History 的提交列表和 Graph 随可用空间变化。History Files 默认从 300 调至 240 DIP，窄窗继续受原有 Files 和 Diff 最小宽度约束。Changes 的默认 Files 宽度仍为 300 DIP。本机参考资源未提供精确的分栏宽度。

macOS 实窗通过指针、菜单、`⌘T`、关闭和切换标签，并在 2640 → 6016 → 2640 像素宽度验证所选提交和补丁保持。截图及结果见 `build/repository-welcome-native-coLPXM/`。系统目录选择的确认仍由回归测试注入选择结果覆盖。Windows/Linux amd64 交叉构建通过，GUI 实测待补。

## 关闭仓库标签与窄窗浏览

macOS 的 File → Close Tab 和工作台 `⌘W` 共用草稿确认。取消后保留草稿，关闭最后一个仓库后回到 Welcome。原生自动化分别验证了 File 菜单和注入按键，物理按键及 Settings 窗口的关闭路由待实测。结果见 `build/repository-close-native-QURSJP/result.json`。

History 按实际可用宽度判断提交标题是否需要悬停提示。关闭窄窗里的当前文件标签后，标签条滚动到接替选中的文件。提交行的 Graph 区域可横向浏览分支，并与顶部工作区行共用滚动位置。14 条分支线的 macOS 临时仓库验证了横向手势移动 Graph、提交标题原位、纵向手势滚动列表，以及 1960 → 6016 → 1960 像素缩放后仍显示选中的提交和补丁。结果见 `build/history-graph-input-native-DmPY0C/result.json`。

## 重新打开已关闭标签与 Graph 滚动条

「重新打开已关闭标签页」从当前窗口的关闭记录中取出最近关闭的仓库，回到原位置并读取当前 Git 状态。已确认放弃的草稿不会进入关闭记录。路径缺失或已变成另一仓库时显示读取错误，下一次恢复可处理上一条记录。关闭记录只在窗口存续期间可用。参考应用的普通打开与恢复命令共用仓库打开路径，恢复原位置是 SuperGit 的选择。

History Graph 存在横向溢出时，工作区行显示滚动条。按 Tab 聚焦后可用左右键平移，与提交行上的横向手势共用滚动位置。macOS 实窗覆盖 File 菜单恢复、注入快捷键、恢复期间重新读取 HEAD、跨仓库草稿隔离、Graph 键盘和滚轮平移、缩放以及补丁保留。验证记录位于 `build/repository-reopen-native-Had1PP/`。
