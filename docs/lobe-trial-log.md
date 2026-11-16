# Lobe 试用记录

此记录区分源码原生试用、此前安装的测试包和默认旧 desktop 包。只记录实际执行的场景，不用后端测试替代 UI 验收。

## 2026-11-16：子模块内部改动拒绝的当前语言说明

- 英文界面实际 dirty 子模块更新拒绝原为中文正文；真实 checked / 普通更新与中英文回归 RED→GREEN，以可识别错误接入当前语言说明，保留后端原文本和更新保护。
- 全量 Go / race / vet、两种构建通过（`build/lobe-submodule-dirty-full-check.log`），原生子模块五组、完整详情标签十组通过（`build/lobe-submodule-dirty-native.log` / `build/lobe-submodule-dirty-detail.log`）。完整结果 `build/lobe-submodule-2vnzuocy/results.json` 已读，英文恢复说明完整显示及切 root 后无旧错误的截图已查看；父子 HEAD / refs / Index 条目 / 状态、配置与脏 Unicode 字节保护保持。
- 本节关闭上一节观察到的该条中文正文问题；不扩大为全部后端错误本地化验收，R04 其余尾项保留。默认依赖和安装包不变；未提交或发布。

## 2026-11-16：子模块跨仓库错误归属复核

- 原生五组全部通过，最新日志 `build/lobe-submodule-root-followup.log`、结果 `build/lobe-submodule-psyuv4uc/results.json`；先捕获父仓库真实 dirty 更新拒绝，再检查 child Worktrees / History 全辅助角色无父错误标题 / 正文。三张前后截图已查看，父子 HEAD / refs / Index 条目 / 状态、.gitmodules 与 dirty Unicode 字节保持。
- 通知归属及子模块原目标测试通过（1.866s）。首次菜单遇自动重读禁用，证据 `build/lobe-submodule-z8lm00lf/` 保留；探针增加就绪后重取菜单目标，产品源码未变。脚本每次新建产物目录 / 会话，不覆盖早期记录。这是已完成错误后的切换，不证明运行中竞态或 OS 焦点；英文界面拒绝正文仍为后端中文，留待 R04 语言映射。未提交、安装或发布。

## 2026-11-16：设置页原生最小初始尺寸

- `native-lobe-settings-layout-smoke.py --minimum-window` 以隔离 Go overlay 将设置窗口初始尺寸设为已有 560×480 下限，构建独立 e2e 二进制。中英文 × 浅深色 × 标准 / 1.15 字号共八组原生检查通过：实际滚动视口宽度、交互控件横向边界、底部 Reset / Cancel 坐标可达，Git 与偏好文件精确不变。
- 日志 `build/lobe-settings-minimum-native.log`、完整结果 `build/lobe-settings-layout-4qt1xdmo/results.json` 已核对；英文深色大字号外观、中文浅色大字号底部与确认截图已查看。只验收以最小尺寸打开原生窗口，不证明实际拖动缩窗、窗口尺寸记忆或全部焦点 / 悬停状态。产品源码、普通构建、默认依赖和已安装包不变，R09 未关闭；未提交或发布。

## 2026-11-16：四种补丁视图的 Unicode 行尾

- 普通固定依赖，独立仓库 / HOME / 应用进程；混合中文、全角、ASCII，1.15 字号，4,000 行 / 8,000 增删行。工作区与历史各三轮通过四种视图的横向位移、行尾边界及 End / Home，完整 Git 指纹保持。结果 `build/lobe-long-patch-zg00bb_l/results.json`、`build/lobe-long-patch-0iowt7s3/results.json`，日志 `build/lobe-unicode-projections-current-target.log` 已完整核对；四视图第三轮截图均完整显示 `wide-tail-3999`，无省略号。
- 工作区探针前三次失败的直接诊断证据指向观察器重读，第三次需临时 overlay 才捕获瞬态；普通构建已恢复。探针分离行尾测量 / 当前端点恢复，并修正重读后使用旧 gutter 标签的后续竞态。只验证读取后的导航，不声明重读保留旧选择与滚动位置。本轮没有产品修复、帧延迟或 RSS 改善结论；极长宽度 / 任意 Unicode 精度仍有限制。默认依赖、安装包未更新，R16 未关闭；未提交或发布。

## 2026-11-16：总览变高占位节裁剪复验

- 普通固定依赖的新源码构建，全量 Go / race / vet、普通 / e2e 构建通过（`build/lobe-placeholder-full-check.log`）。1,000 脏文件总览原生验证首屏 3 个 / 末尾 6 个节标题、末尾实际补丁及返回、完整 Git 指纹保持，结果 `build/lobe-overview-sections-lv8fimhd/results.json`，末尾截图已查看。
- 完整详情标签十组最终通过，日志 `build/lobe-placeholder-detail-ready.log`，含 hooks 失败保留草稿、Index-only 提交、Amend 作者及深历史；代表动作截图已查看。首次历史文件点击后意外切换历史 / 页面仍未定位，证据 `build/lobe-placeholder-detail-failure-v1ixczhw/`。第二次快捷键在身份读取 / Commit 禁用时发送，探针增加就绪等待并保存发送前控件后通过；该就绪问题与首次页面切换分开记录。
- 加载 / 错误 / 空补丁裁剪由 24 组实际 UI 布局回归验证，含普通节高 / 滚动范围对照、实时列宽切换和加载优先级，不能称为原生三种占位状态全覆盖。冲突说明和未知列宽首帧仍普通布局；无新帧耗时 / RSS 数据。默认 MyGo、已安装包未更新，R16 未关闭；未提交或发布。

## 2026-11-16：候选构建的 40 行添加选区

- `check-mygo-display-link-recovery.py --native-long-range --range-line-kind add` 无诊断 overlay 三轮通过。精确 Index 在对应原行之后插入所选末 40 条 working 行；Staged 的 40 条添加行新行号从 3962 到 4040（跳过上下文），取消暂存后外部清理前完整指纹恢复。
- 门槛 `build/mygo-frame-recovery-w27bmngy/`、结果 `build/lobe-long-patch-gv7wvl0w/results.json` 已核对，代表暂存 / 取消暂存截图已查看。精确 RED / 完整 Darwin GREEN 保持，普通 e2e 自动恢复。只验收隔离候选；默认依赖、安装包未更新，普通多行停帧项仍开放，未提交或发布。

## 2026-11-16：长补丁单行丢弃与撤销

- 普通固定依赖，独立 `--discard-undo` / `--discard-undo --line-kind add` 各三轮通过。脚本拥有仓库 / HOME / 应用，确认前完整指纹不变；丢弃后精确 working 字节正确，Index / HEAD / refs / 状态保留；通过既有 Undo 恢复完整原始指纹，不做外部还原替代。
- 添加 / 删除结果分别 `build/lobe-long-patch-fymi2qa2/results.json`、`build/lobe-long-patch-e_6_gg3_/results.json`，完整日志与代表丢弃 / 撤销截图已核对。删除行首次测试把恢复位置错误预期为文件末尾，源码和快照确认现有替换配对顺序后修正预期，未改生产逻辑。仅单行，未验收多行丢弃或真实用户删除流程；未提交、安装或发布。

## 2026-11-16：长补丁单添加行往返

- 普通固定依赖构建，`native-lobe-long-patch-baseline.py --stage-unstage --line-kind add` 三轮通过。选择工作区末条添加行暂存，Index 精确为原 after 4000 行加 working 末行，工作区字节 / HEAD / refs 不变；精确 Staged 标签取消暂存后、外部清理前恢复完整指纹。
- 日志 `build/lobe-added-line-target-followup.log`、结果 `build/lobe-long-patch-_4zuou9o/results.json` 已核对，第三轮截图包含 Line staged / Line unstaged，最终 Staged Files 0。首次新探针 End 落入历史列表，重新建立 diff 行选区后通过，未执行错误写入。仅单添加行范围，未验收 40 行添加选区或丢弃，未改生产代码、默认依赖、安装或发布。

## 2026-11-16：40 行跨视口往返的候选对照

- 普通固定依赖下，长补丁 40 行暂存通过，但 Staged 选区在 37–38 行附近停止推进，末尾按键未进入选区处理，CLI 报告窗口未稳定；该普通构建多行验收仍开放。
- 隔离 MyGo 恢复候选＋诊断 overlay 三轮通过后，持久入口 `check-mygo-display-link-recovery.py --native-long-range` 的无诊断 overlay 构建也实际三轮通过。选满 40 行、端点可见、精确菜单、暂存 Index 只去掉原版本前 40 行、工作区 / HEAD / refs 保持，取消暂存后外部清理前完整指纹恢复。结果 `build/lobe-long-patch-u19cb976/results.json` 与门槛 `build/mygo-frame-recovery-l_4ol70i/` 已核对，第三轮截图显示 Selected lines unstaged / Staged Files 0。
- 普通 e2e 构建已自动恢复。探针取消暂存点击不先 settle，以便观察短暂成功提示；普通构建单行往返按此新顺序再次三轮通过，结果 `build/lobe-long-patch-hrdan6p8/results.json`。候选未接入默认依赖、未更新安装包、未提交或发布，详见 [恢复候选记录](diagnostics/mygo-display-link-recovery.md)。

## 2026-11-16：长补丁导航后单行暂存

- 普通固定依赖的 Lobe 构建，`native-lobe-long-patch-baseline.py --stage-line` 三轮通过。8000 条增删行先做总览与文件标签 End/Home，再显式聚焦首条删除行并经菜单 Stage line；Index 精确去掉原版本首行，全部 working 字节、HEAD / refs 保持，只有 long.txt 为 MM。
- 完整日志 `build/lobe-long-patch-partial-stage-followup.log` 与结果 `build/lobe-long-patch-l50k34dn/results.json` 已读，第三轮截图显示首行已从未暂存版本移除、Staged Files 1 与 Line staged。最终完整 fixture 指纹恢复。首次 probe 菜单等待失败因没有显式键盘 focus，未执行暂存，证据保留。
- 后续 `--stage-unstage` 三轮从精确 Staged 标签取消暂存该行，任何外部清理前完整指纹恢复；日志 `build/lobe-long-patch-stage-unstage.log` 与结果 `build/lobe-long-patch-8k4qu0s7/results.json` 已读，第三轮最终截图显示 Staged Files 0 / Line unstaged。仅覆盖单删除行暂存 / 取消暂存往返；没有长补丁多选、添加行或丢弃结论。没有生产源码修改、提交、安装或发布。另完成的 MyGo display link 恢复候选仅在临时依赖副本验证，尚未接入此普通构建，详见 [独立候选记录](diagnostics/mygo-display-link-recovery.md)。

## 2026-11-16：宽行与横向尾标记

- `native-lobe-long-patch-baseline.py --wide-lines` 历史三轮通过；工作区模式首次 Home 被自动刷新重载补丁打断，日志 `build/lobe-long-patch-oho73f9g/automation-1.log` 留存明确 Loading diff / 子树替换。探针等待当前末行并重新选中 gutter 后三轮通过。
- 完整日志 `build/lobe-wide-line-check.log` / `build/lobe-wide-working-line-check.log` 已读；结果 `build/lobe-long-patch-6mlxlj06/` / `build/lobe-long-patch-dxo_k0x9/` 均保留完整指纹，工作区每轮横向行尾可见断言通过，成功 wide-tail-1.png 已查看。
- 历史首屏 49 个增删控件、工作区总览 20 / 标签 49；应用采样最大 RSS 中位 212,064 / 222,976 KiB。不是 Unicode / 极长宽度、所有投影横向或部分写入验收；未改生产代码、提交、安装或发布。

## 2026-11-16：九处分散 hunk 的原始行号与导航

- 长补丁脚本 `--multi-hunk` 及 `--multi-hunk --worktree` 顺序各三轮通过；真实 Git 校验 9 hunk / 18 增删行，原始 1 / 4,000 端点与上下文保持，完整指纹不变。
- 完整日志 `build/lobe-multi-hunk-check.log`、结果 `build/lobe-long-patch-ocyw_q5y/` / `build/lobe-long-patch-exby5sz_/` 已核对；工作区文件标签末行截图已查看。首屏增删控件历史 12、工作区总览 6 / 标签 12，应用采样最大 RSS 中位分别 195,536 / 198,960 KiB。
- 仅只读分散 hunk 导航，不代表密集变化、部分写入或冲突编辑。未改生产代码、提交、安装或发布。

## 2026-11-16：加载期重复滚轮与搜索编辑

- `native-lobe-wide-history-baseline.py --input-pressure` 三轮通过：读取期间 12 次 precise wheel，原可见行实际位移；逐字输入与清空搜索框，精确值 / 匹配行保持，每个交互检查点仍读取。随后 Cancel / Retry / 完整 32,002 条历史与根导航通过，仓库指纹保持。
- 完整日志 `build/lobe-loading-input-pressure-check.log`、结果 `build/lobe-wide-history-rtp1atxz/results.json` 已读，search-1.png 已查看。12 次逐次 settle 轮事件中位 6.164s、逐字输入中位 263ms，均含自动化；不是帧时间或真实硬件高频事件流。
- 只测搜索编辑，未涵盖提交草稿 / 输入法。无新增生产源码变更、提交、安装或发布。

## 2026-11-16：工作区长补丁与单文件标签

- `native-lobe-long-patch-baseline.py --worktree` 自建唯一文件 4,000 行全替换，保持未暂存状态。初次探针发现标签构建 8,000 行，已裁剪；回归 RED→GREEN 并保留按下 gutter 的跨视口拖选 / 释放。
- 最终 `bg2iq8w8g` 全链通过，完整日志 `build/lobe-working-patch-virtual-final.log` 已读：Go / race / vet、两种构建、工作区与历史各三轮、完整详情标签和空白。工作区结果 `build/lobe-long-patch-_hsqqjex/`、历史 `build/lobe-long-patch-up3z661w/`，工作区完整脏状态指纹保持；file-tab-last-1.png 已查看。
- 总览首屏 20 个、文件标签 49 个增删控件，两处 End / Home 显露通过。应用采样最大 RSS 中位 207,312 KiB，非精确峰值；未覆盖宽单行 / 多 hunk / 长补丁部分写入或系统文本跨屏拖选。未提交、安装或发布；先前批量无响应不据此关闭。

## 2026-11-16：长补丁与宽 DAG RSS 补采

- 新共用采样工具只观察自建应用及后代进程，两套脚本顺序各三轮通过。完整日志 `build/lobe-patch-dag-memory-check.log` 已读；结果 `build/lobe-long-patch-due3jb5e/`、`build/lobe-wide-history-e49lgn7x/` 均保持完整仓库指纹，代表末行 / 根提交截图已查看。
- 长补丁首屏 49 个增删控件、End / Home 显露保持，应用采样最大 RSS 中位 201,408 KiB，每轮 17 个样本；宽 DAG 完整历史 27 行，读取期间 End、取消 / 重试保持，应用采样最大 RSS 中位 715,024 KiB，每轮 121 个样本。
- 宽 DAG 高值出现在根导航，未做分配归因。子进程可能漏采，共享页可重复计数，不是精确峰值或帧时间；无新增生产变更、提交、安装或发布。

## 2026-11-16：工作区 Files 裁剪与一次未定位无响应

- 分组列表裁剪后，全量 Go / race / vet 和两种构建通过。首个原生门槛 `biq9aelww` 在批量恢复后等待干净超时 60s，随后快照 i/o 超时；完整日志 `build/lobe-worktree-virtual-final-check.log`、产物 `build/lobe-performance-fx39t26c/`，未取得 Go 栈，根因未明。
- 脚本增加对自建进程的超时 Go 栈留证和失败测量 JSON；生产代码不变，重跑 `bbjk2dc6v` 三轮通过，结果 `build/lobe-performance-1b_nsfq9/results.json` 为 `state_preserved=true`，batch-1.png 已查看。每轮 Files 38 行，批量应用采样最大 RSS 中位 399,856 KiB；包含其他工作，不能作为行独占内存或固定改善比例。
- 随后完整 `native-detail-tabs-smoke.py` 通过，日志 `build/lobe-worktree-virtual-detail-check.log` 已读，涵盖工作区 / 历史 Tab、折叠、精确暂存、提交 / hook、Index 保护、2,405 条历史及 2,403 条文件历史；空白检查通过。无响应仍未解决，继续保留诊断证据。未提交、安装或发布。

## 2026-11-16：批量父文件与子模块变更

- `native-lobe-performance-baseline.py --with-submodule --batch-files 1000` 在自建夹具中改写 1,000 个父文件＋一个子文件；三轮精确路径、两层身份与恢复后完整指纹通过。
- 完整日志 `build/lobe-batch-performance-check.log` 已读，结果 `build/lobe-performance-__72hzpm/results.json` 为 `state_preserved=true`，已查看 batch-1.png；截图中刷新后的历史仍在读取。
- 保存至状态中位 6.106s（2.406–6.111），恢复 2.492s（2.371–5.424），快照往返约 139ms。Files 全部构建 1,001 行，批量阶段应用采样最大 RSS 中位 494,800 KiB；不是帧时间、精确峰值或行控件独占内存。列表裁剪与更多负载仍待处理，R16 不关闭。未提交、安装或发布。

## 2026-11-16：大仓库与已初始化子模块

- `native-lobe-performance-baseline.py --with-submodule` 联合父 10,003 tracked 条目 / 12,025 提交与本地子仓库 1,000 文件；全部数据和初始化路径来自脚本自建 fixture。
- 三轮父文件 / 子文件保存与恢复均通过，父 Files 正确显示唯一子模块改动，恢复后两层完整指纹保持。完整日志 `build/lobe-combined-performance-check.log`，结果 `build/lobe-performance-wzgwdr25/results.json`；已查看 submodule-dirty-1.png。
- 子保存到父改动状态中位 5.894s，恢复到父干净 2.828s，整段应用采样最大 RSS 中位 263,232 KiB。轮询 / 历史读取 / 自动化均计入，采样不是精确峰值；未覆盖多个 / 嵌套 / 未初始化子模块或批量改动，不据此关闭 R16。未提交、安装或发布。

## 2026-11-16：宽 DAG 与加载期输入

- `scripts/native-lobe-wide-history-baseline.py` 自建 32 并行分支、32,002 提交 / 33 分支、一个 tracked 文件的隔离仓库，三个独立进程 / HOME。分支提交不改文件，不能代表改动密集或子模块联合负载。
- `build/lobe-wide-history-performance-check.log` 与 `build/lobe-wide-history-47puw5bg/results.json` 三轮通过；End 操作前后仍在读取，已加载分支行被选择并进入视口，随后 Cancel / Retry 与最终根提交显露通过。Cancel 中位 21ms，Retry 全历史约 12.179s，完整历史仅 27 个行控件，最终指纹保持。
- 已查看读取中和根提交截图，宽车道汇合可见。未采 RSS 或帧时间，不覆盖连续滚轮 / 编辑 / 写入等完整加载期压力；R16 继续开放。未提交、安装或发布。

## 2026-11-16：长单文件历史补丁

- `scripts/native-lobe-long-patch-baseline.py` 在临时仓库中生成单文件两版本各 4,000 行、约 8,000 条增删行；三个独立进程 / HOME 的首屏均只有 49 个行文本控件。
- `build/lobe-long-patch-performance-check.log` 与 `build/lobe-long-patch-6kpa47ix/results.json` 三轮通过；选中到首行可访问中位 80ms，End / Home 导航约 0.699 / 0.697s，两端行号视口断言及原仓库指纹通过。已查看末行截图，临时仓库已清理。
- 本轮没有生产改动，未采长补丁 RSS，不代表帧时间、宽单行 / 多 hunk / 工作区长补丁或万提交联合读取；R16 继续开放。未提交、安装或发布。

## 2026-11-16：单文件外部保存的自动刷新

- `build/lobe-incremental-performance-check.log` 三轮通过；完整结果 `build/lobe-performance-bac0tbuk/results.json` 的最终指纹保持。只修改脚本自建 fixture 的一个 tracked 文本文件，未手动刷新；唯一 dirty 路径、HEAD / Index / refs 保留，恢复原字节后自动显示干净。
- 保存到状态更新中位 3.256s，工作树选择至补丁文本可访问中位 34ms，恢复到干净中位 2.843s（范围 2.777–5.274）。已查看增量截图：正文在提交表单下方，不据可访问文本称作屏内可见或帧延迟；轮询 / 历史重读和自动化往返均计入耗时。
- 只有脚本与文档更新，沿用上轮已验证生产构建。未覆盖大批量增量、子模块联合样本或加载期混合输入，R16 保持未完成。未提交、安装或发布。

## 2026-11-16：整批审查修复后的原生复测

- 四项审查问题均经当前源码回归 RED→GREEN。完整 Go / race / vet 和两种构建通过；三轮原生性能验证未变刷新保留最早提交的完整 SHA、万文件仍只构建 39 行 / 14 节、仓库指纹不变。结果 `build/lobe-performance-6cbo2hw8/results.json`，已查看最早提交截图。
- 原生新增断言曾复现渲染阶段将部分历史当成搜索结果，补充逐帧回归后修复。随后 detail-tabs 后段仍按旧行为等待最新提交出现，快照实际保留原选择；改等完整提交数并断言旧选择后，完整重跑通过。证据 `build/lobe-review-final-check.log` 与 `build/lobe-review-detail-final.log`；后者含 2,405 条主历史 / 2,403 条文件历史与最早行显露。
- 万文件 End 中位 0.690s，保留原选择的未变刷新 ready / 完整历史中位 0.297 / 2.638s，整段应用采样最大 RSS 中位 250,752 KiB；不是帧耗时或精确内存峰值。旧刷新基线丢失选择的限制已补记，不据此计算加速比。
- 四项修复已通过当前门槛；此前偶发跳变没有完整同源证据，仍保留原记录。没有新增跨平台、系统菜单或双窗口验收。未提交、安装或发布。

## 2026-11-16：万文件历史提交渲染修复复测

- 分阶段采样将增长缩小到万文件提交选中，两个独立回归复现全部文件行 / 节标题构建，视口裁剪修复后 RED→GREEN，完整高度、End 末行 / 原始位置和跨节光标显露通过。
- 新增 End 测试首轮未取得焦点，诊断显示测试直接置于 Column 导致滚动区高度为零；按真实布局包入 600 高 Row 后通过，未添加生产焦点绕过。
- `build/lobe-large-commit-full.log` 全部输出已读：Go 测试 / race / vet、两种构建、3 轮原生性能、完整 detail-tabs、16 组 / 96 张矩阵及空白检查通过。结果 `build/lobe-performance-4s51w4jc/results.json`，state_preserved=true，每轮万文件仅构建 39 个 Files 行 / 14 个节标题。
- 已查看 oldest-1.png、980 英文深色大字号标签与中文浅色大字号可提交截图。万文件选中阶段应用 RSS 中位 240,960 KiB，整段 254,416 KiB；End 中位 0.694s。首轮启动 1.385s，另两轮约 0.455 / 0.458s，未排除或解释该值。更完整比较和采样限制见性能基线文档。
- 未改工作区分组列表，R16 保留联合样本等缺口；后续独立整批审查及四项修复见上节。未提交、安装或发布。

## 2026-11-16：当前 Lobe 大仓库端到端初测

- `build/lobe-native-performance-check.log` 三轮通过，原始 `results.json` 与截图在 `build/lobe-performance-s6o948w5/`。每轮独立应用进程与 HOME，万文件 / 万提交 fixture，HEAD / Index / refs / status / 代表字节保护通过，临时 fixture 已清理。
- 首批历史中位 0.453s、Cancel 0.016s，Retry 完整历史 2.312s；未变刷新 ready / 完整历史 1.724 / 3.503s。End 至含 10,000 文件的初始提交约 2.418s；已查看截图，Files 10,000 与正确选中可见。
- 应用采样最大 RSS 中位 1,915,712 KiB，应用与后代当次 RSS 合计最大值中位 1,930,672 KiB。每轮 75–76 次、每约 100ms 加命令开销采样，可能漏短命子进程、重复共享页，不能称为精确峰值或泄漏证据。详细方法 / 范围见性能基线文档；正在补分阶段采样以定位增长，不代替 R16 完整验收。

## 2026-11-16：双远端与不同名 upstream

- `build/lobe-multi-remote-check.log` 完整输出已读，完整同步脚本 6 组通过。新增本地裸 backup 与原有 origin，main upstream 配置为 backup/publish/nested，Push 确认目标正确；取消双方不变，确认只推进 backup 的指定目标，origin 全 refs 不变、backup/main 不存在。
- 两远端分别新增 fixture 引用，原生 Fetch 同时读取；核对 main HEAD / branch / Index / status、暂存二进制与未暂存 Unicode 字节均保留。已查看 multi-remote-push.png 和 final.png，显示两个远端的正确目标及获取成功。
- 只扩展隔离脚本，沿用 box4lkusy 全量门槛后的二进制；无真实网络、凭据、远端选择器、任意远端名规则或系统菜单 Force / prune 验收。

## 2026-11-16：忽略文件保护的当前语言

- 原生截图中的中文原因已修复：后端类型化两种 ignored 拒绝，保留原路径与旧文本；Lobe 根据当前语言输出恢复说明，未改变保护条件。
- 真实 RemoveWorktree / cherry-pick 碰撞回归 RED→GREEN，核对中英文说明及二进制字节 / HEAD / Index / 工作树身份。首个碰撞用例误选 SwitchBranch，读取其 --no-overwrite-ignore 路径后改用实际 protectIgnored 调用者 cherry-pick；未修改分支切换行为。
- `build/lobe-ignored-feedback-full.log` 完整输出已读：全部 Go 测试 / race / vet、两种构建、原生 Worktree 8 组和完整 detail-tabs 通过。已查看更新后的 ignored-refusal.png，英文说明完整可见。之前条目的中文缺口是当时状态，现已补齐；R04 其他边界仍未关闭。

## 2026-11-16：Worktree 锁定与忽略文件原生保护

- 完整 Worktree 脚本 8 组通过，日志 `build/lobe-worktree-protection-check.log`。新增已加载锁定时移除菜单禁用、确认后外部加锁时拒绝旧目标并给出英文恢复说明、普通 status 干净时 ignored 二进制文件仍阻止移除。
- 核对准确二进制字节、子工作树 HEAD / Index 与主工作树原状态；只删除本场景新建 ignored fixture 文件后，原有干净移除 / 保留分支 Delete 继续通过。已查看 locked-menu.png、ignored-refusal.png。
- 无生产代码变化。ignored 拒绝原因在英文界面仍显示中文，R04 保留该修复项；系统目录面板与其他平台未验收。较早条目中的锁定 / ignored 未验收描述仅代表当时覆盖范围。

## 2026-11-16：四类冲突操作卡

- 扩展现有 operation 脚本：cherry-pick / Rebase / Merge / Revert 共 11 组原生场景完整通过，退出码 0。Rebase / Revert 核对 Continue 禁用、Abort / Skip 取消与确认、原分支恢复、解决后 Continue 保留完整消息 / 预期父提交 / Index 和字节。
- Merge 核对卡片不提供 replay 专用控制、未解决时非空标题仍不能 Commit、Abort 取消 / 确认；外部解决后从提交框输入完整标题 / 正文，产生预期双父提交，暂存与工作字节保留。已查看 Merge 和 Revert 最终截图。
- 所有操作均在临时仓库与独立 HOME 中，冲突发起、文件编辑及暂存由 fixture 显式执行。该记录只验收操作卡与 Merge 完成提交，不代表应用内冲突编辑、系统菜单发起或 D4 标记策略通过。生产代码沿用上次全量门槛，安装包未更新。

## 2026-11-16：Stash 实际三方冲突

- 扩展隔离脚本，分别经 UI Apply / Pop 产生真实三方冲突；保留 HEAD 与选中 stash，Index 有 1 / 2 / 3 三阶段且文件包含双方字节与冲突标记。错误详情包含当前语言恢复说明和原始 Git stderr；显式外部解决并暂存后核对准确 stage0 字节，HEAD / stash 不变。
- 第一轮误在紧凑提示阶段等待完整说明超时；第二轮要求 stderr 包含 CONFLICT，实际为 Index was not unstashed。查看完整失败快照、收集器与既有回归，并独立用临时 Git fixture 分离 stdout / stderr，确认 CONFLICT 位于 stdout；据此修正断言，第三轮完整脚本全部通过，退出码 0。已查看三方 Pop 详情及最终截图。
- 当前错误详情保留 stderr，不保证 stdout；三阶段及字节断言仍严格保留。本轮只扩展原生验证，沿用最新通过全量门槛的二进制。外部编辑 / 暂存不算应用内冲突编辑验收，也没有选择 D4 残留标记策略。

## 2026-11-16：后端性能基线

- 本机大仓库与小文件 / 子模块 benchmark 均通过，三组采样数据及环境 / 限制见 [性能基线](performance-baseline-2026-11-16.md)。大仓库 10,001 文件、12,024 提交、25 分支，运行后规模 / HEAD 与生成元数据一致且保持干净。
- 原始结果保留在 `build/lobe-backend-baseline-20261116-bv8kjjs0z.log`，仅清理本次生成的临时 fixture。该后端基线不代表 Lobe 打开、滚动、取消或 UI 帧耗时；没有新增峰值 RSS 证据，R16 未整体完成。

## 2026-11-16：脏工作区拒绝的当前语言

- Worktree 最终截图中的中文后端说明已溯源到共享 clean 保护，改为可识别错误；保留 legacy 原始文本，Lobe 显示当前语言的提交 / Stash 恢复说明。
- 真实 dirty checkout 回归 RED→GREEN；全部 Go 测试 / race / vet、两种构建、Worktree 与完整 detail-tabs 原生回归通过，退出码 0。原生脚本在脏目录拒绝时等待完整英文说明，最终截图已查看，提示全文英文；拒绝仍保留文件与分支。
- 这项窄范围文案修复不代表全部后端错误已本地化。此前 Worktree 混合语言记录是修复前证据，当前结果以本节为准。

## 2026-11-16：Worktree 移除后的保留分支动作

- 扩展当前隔离 Worktree 脚本，在干净移除后核对分支仍存在，等待 main 工作树就绪后坐标点击现有 toast Delete。等待 Branch deleted，核对仅原分支消失、当前仍为 main，主 HEAD / Index / Unicode 字节不变。
- 完整 Worktree 脚本通过，退出码 0，最终截图已查看：列表仅 main，删除成功提示可见。脚本没有新增生产行为；沿用刚通过全量检查 / 构建 / 原生门槛的二进制。系统目录面板回填、锁定 / ignored UI 保护仍未验收。
- 最终截图还保留先前脏工作树拒绝的中英文混合原始反馈；该文案边界仍归 R04，不将本次成功删除视为修复它的证据。

## 2026-11-16：长分支与最小窗口

- 引用脚本扩展 35 个长 Unicode 分支和两个本地远端；980×640 下验证无结果 Enter、鼠标仅选择、聚焦 Enter checkout、远端搜索隔离及 HEAD / Index 保护。首次等待文案拼错，按真实文本修正后交互通过；截图暴露长当前分支使顶栏操作越界、标题与搜索重叠。
- 新增中英文 × 标准 / 1.15 字号布局测试，复现后限制分支显示宽度、复用中间省略并保留完整 tooltip / 辅助描述；页副标题按剩余空间收缩。另覆盖分支列表尚未加载的同类越界。
- 完整门槛 `make check`、两种构建、980×640 引用脚本与完整 detail-tabs 原生冒烟全部通过，退出码 0。原生快照新增边界与重叠检查，可拒绝修复前快照；已查看最终英文浅色截图，顶栏操作、搜索和 footer 均可见。随后 1320 宽标准浅色、980 宽 1.15 字号深色引用脚本均通过；大字号截图另发现说明栏固定高度不足。窄面板布局回归 RED→GREEN 后改为按文本展开，原生断言能拒绝旧溢出快照；重新通过完整 Go / race / vet、两种构建、大字号深色引用及完整 detail-tabs，最终截图已查看，三行说明位于分隔线上方。该结果不等同 R08 全部菜单或 R09 全矩阵验收；未更新安装包。

## 2026-11-16：本地检测与工作台续作

- 平台：macOS 27.2.1，arm64；Go + MyGo / Lobe 原生 GPU UI。
- 构建对象：`cmd/supergit-lobe`，普通和 e2e 构建；分支 `feat/detail-tabs`，基于 `d1e6527` 的未提交工作区。该 SHA 本身不包含本轮修改，不能单独作为可复现版本号；对应代码及执行账本均留在本地，未生成源码提交或发布版本。
- 验证环境：脚本创建临时仓库、本地子模块源及独立 HOME / 偏好；没有使用真实工作仓库测试写入，没有真实远端推送。
- 最新全量门槛：`make check`（全部 Go 测试、race、vet）、`make lobe-build`、`make lobe-e2e-build` 均通过。

| 场景 | 实际结果 | 证据 |
| --- | --- | --- |
| 工作区 / 历史 Files 的 Tab 离开和 Shift+Tab 返回 | 通过；方向键保留文件导航 | `scripts/native-detail-tabs-smoke.py`；`build/detail-tabs-smoke/automation.log` |
| 文件标签、同名 / 双版本、关闭 / 草稿、SHA / 仓库切换、设置显隐 | 完整原生冒烟通过 | 同上；日志、最终快照和截图位于该目录 |
| hook 拒绝、完整消息 Commit / Amend、文件 / 批量操作、取消与确认 | 通过；只写入 fixture 内明确目标 | 同上 |
| 主历史 2,405 条、文件历史 2,403 条完整遍历及最早记录定位 | 通过 | 同上 |
| 自动 Fetch 关闭时的再次保存、未选中 Unicode 文件保存 | 不按 Refresh 自动更新；草稿与输入焦点保留 | `scripts/native-local-watch-smoke.py`；`build/local-watch-smoke/` |
| 外部 Stage / Commit / 切分支、子模块 HEAD 前进 | 不按 Refresh 自动更新；观察过程不改写父仓库 HEAD / Index | 同上；已查看最终截图 |
| 可见区读取优先级、跨视图缓存、预算与代次隔离 | Go 行为回归通过，包含实际 UI 引擎滚动；不是帧耗时测量 | `overview_read_queue_test.go`、`worktree_diff_cache_test.go` |
| 活动标签缩窄、16 组 × 6 张显示矩阵、三处分栏拖动及重启恢复 | 最新长分支 / 说明栏修复后重跑 16 组 × 6 张截图通过，抽查窄窗大字号 / 中文提交表单 / 宽屏长说明；三处分栏拖动与重启恢复沿用前段证据 | `scripts/native-narrow-shots.py`、`scripts/native-workbench-layout-smoke.py`；`build/narrow-shots/` |

## 2026-11-16：候选主线核对与 Amend

- 新增 [能力差距表](lobe-capability-gaps.md)，明确旧 desktop 原生脚本不能作为 Lobe 验收；同步、Stash、操作卡、Worktree / Submodule 仍需当前源码的专用闭环证据。
- 真实 reload + UI checkbox 测试复现 Amend 只预填标题、遗漏正文；已改为读取固定 HEAD SHA 的完整消息。当前完整 `native-detail-tabs-smoke.py` 新增断言验证摘要和两段正文的预填，随后完整 Amend 回写及作者 / Index 保护均通过。
- 本次 `make check`（全部测试 / race / vet）、普通 / e2e 构建及完整标签式原生冒烟均通过；本次未重跑 local-watch 与显示矩阵。仍为未提交源码，安装包未更新。

## 2026-11-16：Lobe 本地远端同步闭环

- 新增并运行 `python3 -I scripts/native-lobe-sync-smoke.py`，使用当前 Lobe e2e 二进制、全新临时仓库 / 本地裸远端 / peer、独立 HOME 和关闭自动 Fetch 的配置。
- 通过：无 upstream 的 Pull 禁用；Push 取消不写远端 / upstream；首次 Push 写入正确 SHA 并设置 upstream；外部推进后 Fetch 不改 HEAD / Index / 工作字节；Pull 快进正确更新 Unicode / 空格路径；真实分叉后 Pull 报错且不隐式合并、不改本地内容。
- 第一版脚本查询尚未创建的 ref，提前报错退出；改为允许空结果的 ref 轮询后完整通过。仅新增脚本与文档，本轮没有生产代码变化，沿用上一轮通过全量门槛的二进制；没有重新声称本轮执行全量 Go 检查。
- 已查看 `build/lobe-sync-smoke/final.png`：分叉状态及 Pull failed 显示准确；长 Git 原文 toast 占据大块补丁区域，归入 R04 处理。完整日志与快照同目录保留。没有真实网络或用户远端写入。

## 2026-11-16：Lobe Stash 闭环

- 新增 `scripts/native-lobe-stash-smoke.py`，当前 Lobe e2e 二进制、新建临时仓库与独立 HOME、关闭自动 Fetch。完整运行通过保存（含未跟踪）、预览、Apply、Drop 取消 / 确认、Pop 成功及重叠拒绝。
- 每步核对真实 HEAD / Index / 文件内容与 stash ref；Apply / Pop 恢复不同的暂存与工作区版本，Drop 不改工作文件。最终失败场景保留 stash、HEAD、Index 和原有 tracked 字节；Git 会先恢复未跟踪文件，再因 tracked 重叠报错，纯 Git 临时 fixture 已独立复现。不能把失败表述为所有文件均未改变。
- 前两次脚本定位失败（页快捷键及菜单控件角色）；第三次发现失败原子性假设不成立。按实际快照 / 源码和纯 Git 证据修正后，第四次完整运行通过；没有修改生产代码。日志与截图在 `build/lobe-stash-smoke/`，已查看最终截图。
- R04 新证据：英文错误标题配中文后端正文，且未说明可能已恢复未跟踪文件。该反馈缺口仍未修复。实际三方冲突写入 / 解决、跨平台及用户真实仓库验收不在本次结论内。

## 2026-11-16：Lobe Worktree 闭环

- 新增 `scripts/native-lobe-worktree-smoke.py`，当前 Lobe e2e 构建、临时仓库及独立 HOME / 偏好，关闭自动 Fetch。
- 原生通过表单创建含空格路径及新分支、菜单切换工作树及返回、移除取消、dirty 目录移除拒绝、干净移除后保留分支。逐步核对主仓库 HEAD / Index / Unicode 文件不变，移除成功后 checkout 目录和工作树登记消失、分支 SHA 保留。
- 首轮创建通过，但在异步切换尚未结束时过早断言失败；快照显示 Refresh 禁用且旧根仍在。修正为等待目标行 Current 与页面恢复可用后，完整重跑通过。没有生产代码修改；没有重新宣称执行全量 Go 检查。
- 已查看 `build/lobe-worktree-smoke/final.png`：仅主工作树保留、`native/review` 分支保留及成功提示可见；未点击删除分支。日志 / 快照同目录。原生系统目录面板回填、锁定 / ignored 目录保护和其他平台不在本次结论内。

## 2026-11-16：Lobe Submodule 闭环

- 新增并运行 `scripts/native-lobe-submodule-smoke.py`；临时父仓库、本地两提交子模块源、独立 HOME / 偏好，关闭自动 Fetch。Unicode / 空格路径与独立 module name 均保留；仅 fixture 进程允许 file 协议，没有联网。
- 原生通过：Update 初始化已有配置模块、child HEAD 偏移后恢复父 Index 指定 SHA、dirty 更新拒绝保护 child HEAD / Index / 文件及 parent HEAD / Index、打开 child 并在 History 显示独立子提交列表且保留 dirty 字节。
- 首轮在打开子仓库后未切 History，等待不可见提交超时；快照证明根已切换且仍在 Worktrees 页。显式导航后完整重跑通过；仅脚本 / 文档变化，没有新全量 Go 检查或安装。
- 已查看 `build/lobe-submodule-smoke/final.png`，child 根、detached HEAD、两条子提交与改动可见。旧错误 toast 仍跨 root 保留；首轮快照另显示子仓库 Worktrees 页路径为 `.git/modules/...`，需后续调查正确性，不能用历史通过掩盖该页缺口。日志和快照同目录。

## 2026-11-16：Lobe cherry-pick 操作卡

- 新增并运行 `scripts/native-lobe-operation-smoke.py`，临时仓库、Unicode / 空格路径、独立 HOME，关闭自动 Fetch。操作发起及冲突解决由 fixture 的 Git / 文件写入完成，UI 验收范围为进行中操作卡。
- 原生通过真实冲突的 Continue 禁用、Abort 取消保留冲突及确认恢复原 HEAD / Index / 字节、Skip 取消 / 确认、外部解决并暂存后的 Continue。继续产生的提交父节点正确，完整源消息含多段正文 / footer 保留，实际提交树与已解决内容一致，Index / 工作字节不被替换且操作标记清除。
- 首次脚本用后缀匹配工作区行，也匹配了提交标题，未执行操作即失败；读取快照后改用顶栏精确进行中入口，完整重跑通过。没有生产代码变化，没有新全量 Go 检查。已查看 `build/lobe-operation-smoke/final.png`，最终干净状态与 Continued 提示可见；日志 / 快照同目录。
- merge / rebase / revert 的当前源码原生闭环、UI 发起整合、冲突编辑及 D4 残留标记策略不在本次验收内。

## 2026-11-16：子模块 Worktrees 路径修复

- 根因：Git 的 `worktree list --porcelain -z` 对 absorbed submodule 主记录返回 `.git/modules/...`，原解析直接用作 checkout 身份。`Worktrees` 现通过已有 `Open` / Git `--show-toplevel` 校正第一条非 bare 主记录，保留 linked 路径、bare 记录及暂时不可用目录的原始记录，不改工作文件。
- 新增 `TestSubmoduleWorktreesUseCheckoutPathsFromBothContexts`，真实子模块含独立 module name、Unicode / 空格 checkout 和 linked child。从两处读取均先复现错误路径，再通过修复；不把当前 linked 根错误替换为主 checkout。
- 本次 `make check`（全部 Go 测试 / race / vet）、普通 / e2e 构建及子模块、工作树、完整 detail-tabs 三套原生脚本全部通过。子模块脚本新增断言确认实际 child checkout、Current 与不暴露元数据路径；工作树移除保护与全部工作台提交 / 标签 / 历史回归仍通过。
- 该修复每次 Worktrees 读取增加一次主目录解析 Git 命令；没有按全部工作树逐条启动进程。旧 toast 跨 root 的问题仍未修复。先前对应路径缺口由本节后续证据关闭；没有安装或提交源码。

## 2026-11-16：通知的仓库归属

- 三项实际 UI 回归先复现已显示 toast 切 root 后残留、旧队列进入新 root、上一帧按钮跨 root 执行。主窗口现在记录通知 root，在根变化时通过 UI 引擎关闭已显示 toast，过滤旧队列，并在操作回调检查当前根；同根刷新保留错误，设置窗口使用独立上下文。
- `make check`（全部 Go 测试 / race / vet）、普通 / e2e 构建、子模块与完整 detail-tabs 原生脚本全部通过，退出码 0。子模块脚本新增断言确认父仓库更新失败提示不进入 child；已查看本次 `build/lobe-submodule-smoke/final.png`，旧提示已退出。
- 先前试用记录中的跨 root 提示缺口由本节关闭。未声称完成 R04：长 Git 错误布局、语言与部分恢复说明、确认归属及历史选择跳变仍需处理。没有提交、安装或发布。

## 2026-11-16：确认框归属与覆盖保护

- 实际 UI 回归复现后来确认替换前一对象、切 root 后旧确认仍显示与旧回调仍执行；修复后先开的确认保留，后来请求显示当前语言重试提示，旧 root 确认与回调失效，同根刷新保留。
- 通用确认框按捕获的窗口绘制，main / Settings 分别用独立 UI Tester 验证不会交叉显示。设置重置确认不绑定仓库根；关闭设置窗口清理不可处理的遗留确认。没有把这些测试说成原生双窗口焦点切换验收。
- `make check`（全部 Go 测试 / race / vet）、普通 / e2e 构建、原生操作卡与完整 detail-tabs 脚本全部通过，退出码 0。Abort / Skip 取消与确认、Continue、工作台原有提交 / Amend / hooks / 文件操作与长历史回归均通过。
- 通用确认框之外的 prompt / reset / checkout / blame / worktree 自定义弹层仍随焦点；对象状态变化保护仍需按具体动作审计。R04 的长错误布局与 Stash 部分恢复说明仍未完成；没有提交、安装或发布。

## 2026-11-16：Stash 失败说明与 Git 原文

- 后端 overlap / CONFLICT / 状态检测冲突错误原先丢弃 Git 原文；新增错误链保留原始 cause，同时保持旧 desktop 的 Error 短句兼容。三类原文保留回归 RED→GREEN；其他错误保持原样。
- 新版按当前语言提示处理重叠改动或冲突，并提醒即使失败也可能已恢复部分文件，包括未跟踪文件。真实 Apply / Pop × 中英文回归验证提示与 Git 原文、保留 stash SHA 和原有 tracked 字节；原有失败后 reload 保留。
- 全部 Go 测试 / race / vet、普通 / e2e 构建、原生 Stash 与完整 detail-tabs 全部通过，退出码 0。原生新增英文说明 / 部分恢复 / 原始 Git 输出 / 无旧中文提示断言。已查看 `build/lobe-stash-smoke/final.png`，上述内容可见；长 toast 仍遮挡补丁，布局缺口尚未关闭。
- 新测试初次因异步测试 dispatcher 与 MyGo 同步 RunOnMain 契约不一致而误判 Cancelled；读 SDK 确认后只修正该测试的同步等待，再复现真实反馈缺失并修复。没有修改生产取消逻辑。未提交、安装或发布。

## 2026-11-16：长错误正文与滚动详情

- UI 回归先复现 100 行 Unicode 正文全部展开在 toast，修复为紧凑当前语言提示和“查看详情”；完整原文留在原窗口滚动详情中，可复制并用 Esc 关闭，切 root 后退出。已有动作的错误保持原操作按钮，尚未纳入压缩。
- 设置最小 560×480 窗口下标题曾横向溢出；边界回归 RED→GREEN，改为 600×420 首选尺寸加 90% 宽高上限。主窗口与设置窗口隔离、原文恢复 / 精确复制 / root 退出和 Esc 回归通过；尚未验证字号 1.15 和长无断点文本。
- 全部 Go 测试 / race / vet 和普通 / e2e 构建通过。原生首次因新增 find 调用未定义中断；提取原有 press 查找逻辑后，原生 Stash 和完整 detail-tabs 重跑全部通过，退出码 0。新增真实 status 高度 <180 DIP、打开完整详情保留英文说明和 Git 原文、Esc 关闭断言。
- 已查看 `build/lobe-stash-smoke/compact-error.png` 与 `error-details.png`，长输出不再自动遮住大片补丁。分支 footer 的实际可点击性、长标题、自带动作错误、关闭窗口生命周期及其他 R04 项尚未验收，不将 R04 整体标完成。未提交、安装或发布。

## 2026-11-16：长标题与窗口关闭边界

- 长错误标题未走正文压缩路径，仍撑大提示；新回归复现后，改用当前语言的短标题，将原始标题与正文完整保留到详情，精确复制回归通过。
- 关闭设置窗口后详情状态原先仍保留；新回归复现后，在窗口关闭时清理，重开不显示旧内容。窗口隔离、原文 / 复制、root 与 Esc 既有回归保持通过。
- 新 UI 回归覆盖 980×640、1.15 字号及 512 段连续 Unicode token：文本宽度受约束，Copy / Close 控件在窗口内，完整复制正确。这是 UI Tester 证据，未新增原生大字号测试。
- 全部 Go 测试 / race / vet、普通 / e2e 构建、原生 Stash 与完整 detail-tabs 再次全部通过，退出码 0。分支 footer 的实际点击、已有动作的长错误及其他弹层 / 对象状态仍未完成 R04 验收。未提交、安装或发布。

## 2026-11-16：分支 footer 实际点击

- 紧凑 toast 仍拦截 Copy name 点击；980 / 1320 × 普通 / 1.15 字号四组 UI 回归均先失败，按实际 footer 高度预留提示底部空间后通过。主窗口每帧重置测量值，设置窗口不使用分支 footer 间距。
- 全部 Go 测试 / race / vet、普通 / e2e 构建通过。原生首轮进入 Branches 但未选中行，详情为空而等待 footer 超时；读取失败快照后补上选择本地 main。同步与完整 detail-tabs 重跑全部通过，退出码 0。
- 原生新增：Pull failed 仍在时比较提示下边界与 Copy 按钮上边界，截图后用坐标点击而非语义 press，等待 Copied，再核对 HEAD / Index 不变。已查看 `build/lobe-sync-smoke/branch-footer.png`：提示位于 footer 上方，按钮区可见。
- 该证据关闭原始 toast 遮挡 footer 子项，未代表 R04 所有弹层 / 对象状态保护通过。未提交、安装或发布。

## 2026-11-16：通用输入弹层归属

- 核对当前 askPrompt 调用均由主窗口仓库操作发起。回归先复现后来输入覆盖当前目标 / 草稿、旧 root 输入残留与旧回调执行；修复为主窗口固定归属、root 检查和保留先开输入的重试提示。同根刷新保留草稿，多步标签式输入可提交到下一步。
- 相关 UI 回归与现有键盘 / modal 检查通过；全部 Go 测试 / race / vet、普通 / e2e 构建、原生 Stash 与完整 detail-tabs 全部通过，退出码 0。原生保存输入路径仍正常，未新增原生双窗口焦点切换验收。
- Reset / checkout / worktree / blame 的独立弹层、同根对象状态及原历史选择跳变仍需单独核对；R04 未整体完成。未提交、安装或发布。

## 2026-11-16：Reset 表单归属与选定目标

- 新回归先复现后来 Reset 请求替换选定提交、切 root 后旧表单残留、旧表单向新仓库启动操作。修复后捕获 root，保留当前目标并提示先处理，主窗口固定绘制，在打开 / 绘制 / 写入前丢弃过期表单。
- 同根刷新保留选定 SHA；实际 UI Hard Reset 模式点击仍触发危险二次确认，切 root 后该回调不能启动。既有忙碌 / 无仓库 / 取消 / modal 检查通过，测试没有操作源码仓库。
- 全部 Go 测试 / race / vet、普通 / e2e 构建、原生操作卡与完整 detail-tabs 全部通过，退出码 0。原生操作卡继续覆盖 Abort / Skip / Continue，尚未新增原生 Reset 实际执行场景。
- 同根 HEAD / 分支基线变化保护及 checkout / worktree / blame 独立弹层仍需核对；R04 未整体完成。未提交、安装或发布。

## 2026-11-16：提交分支选择器归属

- 回归先复现后来请求覆盖选定提交 / 搜索、旧 root 选择器残留和旧 Enter 向新仓库启动切分支；修复为主窗口固定归属、捕获 root 并丢弃过期状态，保留当前请求和输入，提示后来请求重试。
- 同根刷新保留搜索 / 光标，窗口隔离及既有搜索、键盘、列表缩短定位和无障碍回归通过。全部 Go 测试 / race / vet、普通 / e2e 构建、完整 detail-tabs 原生脚本通过，退出码 0。
- 原生回归覆盖既有工作台流程，未新增选择器实际切分支或双窗口焦点验收。同根分支 tip 变化及 Worktree / Blame 归属仍需审计，R04 保持未完成。未提交、安装或发布。

## 2026-11-16：Worktree 表单归属与失败恢复

- 回归先复现后来请求覆盖草稿、旧表单跨 root 显示 / 提交、失败后恢复到另一仓库或覆盖新草稿；已绑定原 root，固定主窗口绘制，后来请求提示重试，按表单代次保护异步恢复。旧目录选择结果不会写入新表单，当前结果仍保留已输入目录名。
- 新增仓库 / 草稿 / 恢复、窗口隔离及目录返回检查 RED→GREEN；原有真实创建、Git 失败重试、忙碌保留输入和键盘回归通过。全部 Go 测试 / race / vet、普通 / e2e 构建、原生 Worktree 与完整 detail-tabs 全部通过，退出码 0。
- 原生仍验证实际表单创建、切换及取消 / dirty 拒绝 / 干净移除保护；系统目录选择和双窗口原生焦点未新增验收。同根对象状态与 Blame 归属仍待核对，R04 保持未完成。未提交、安装或发布。

## 2026-11-16：Blame 弹层归属与异步结果

- 临时真实 Git fixture 回归先复现后来请求替换文件 / 版本 / 选项、旧仓库或已关闭弹层接收读取结果、旧行向新仓库导航；新增 root / open / 读取代次检查，固定主窗口绘制，保留当前对象并提示后来请求重试。
- 上一帧行回调还能影响重新打开的 Blame，另经 RED→GREEN 修复为捕获 root / 代次。同根刷新保留读取和选项，既有行菜单、键盘和无障碍检查通过。
- 全部 Go 测试 / race / vet、普通 / e2e 构建及完整 detail-tabs 原生回归通过，退出码 0；原生未新增 Blame 专项或双窗口焦点验收。工作区与固定 revision 语义保留，同根对象基线和其他 R04 项仍待核对。未提交、安装或发布。

## 2026-11-16：普通刷新读取反馈

- 已有失败后的旧数据提示、实际 Retry 启动新读取、单一 Retry 与 Retrying 禁用回归通过；普通刷新此前仅禁用 Refresh，顶栏进度只覆盖 Fetch / 写入。新中英文 UI 回归复现读取无文字后，补为超过 200ms 显示旋转图标和已有本地化加载文字，完成后退出。
- 新回归覆盖延迟、操作进度优先、读取期间 Refresh 禁用及不提供无效取消按钮。全部 Go 测试 / race / vet、普通 / e2e 构建、完整 detail-tabs 原生回归通过，退出码 0；未新增刻意延迟读取的原生验收。
- 同根对象状态、跨类型弹层与其他 R04 验收仍待核对；未提交、安装或发布。

## 2026-11-16：跨类型弹层与失败恢复

- 回归复现输入 / 确认 / 错误详情等不同类型弹层重叠，以及旧 root 的其他类型状态残留；按窗口检查当前弹层并清理过期状态，后来交互请求提示重试。七类 × 六种后来类型的 42 组检查通过，主窗口和设置窗口保持独立。
- 真实 Git 失败另复现 Worktree 自动恢复盖住当前输入；改为保留草稿，完成当前弹层后可重新打开。toast 动作会关闭 toast，直接拒绝错误详情会丢失查看请求；新增回归复现后，保留完整原文并延后到当前弹层结束时显示。
- 全部 Go 测试 / race / vet、普通 / e2e 构建、原生 Stash / Worktree / 操作卡 / 完整 detail-tabs 均通过，退出码 0。原生保留既有正常操作流程验收，未新增原生并发弹层或双窗口场景；同根对象变化等 R04 项仍待核对。未提交、安装或发布。

## 2026-11-16：Reset 同仓库 HEAD / 分支基线

- UI 回归复现同 root 的 HEAD / 分支变化后模式仍可用，以及 Hard 二次确认丢失原分支基线；已捕获打开时基线并在模式 / 二次确认执行时复核，执行前还通过新 checked 后端入口读取实际 Git 状态。
- 真实 Git 外部提交、同 SHA 的另一分支、detached × soft / mixed / hard 九组拒绝场景先失败后通过，拒绝后 HEAD / Index / 工作字节保留，正常三种模式通过。实际 UI Hard 模式及确认回调之间外部切分支、UI 不刷新回归，通过临时恢复旧入口验证能检出损坏，再恢复 checked 入口通过并核对本地化提示。
- 全量首轮在 TestChineseStringsOnlyKeepTheListedEnglishTerms 失败，race / vet / 构建 / 原生未运行；把中文提示中的 Reset 改为“重置”后，全部 Go 测试 / race / vet、普通 / e2e 构建、原生操作卡与完整 detail-tabs 重跑通过，退出码 0。
- 当前是 UI Tester + 真 Git 基线证据，未新增原生 Reset 实际执行；应用内锁不约束外部 Git 进程，HEAD / 分支复核不构成跨进程原子保证，也不是工作字节快照。其他对象仍需审计。未提交、安装或发布。

## 2026-11-16：本地分支删除的对象身份

- 普通与强制删除确认此前仅捕获名称，临时真实 Git 回归复现确认期间 tip 移动 / 同名不同 tip 重建后仍删除新对象；菜单 / footer 改为传入不可变分支对象，确认显示名称＋原 SHA，执行前复核实际 refs/heads 目标。
- changed-target 拒绝及应用确认回归 RED→GREEN，缺失引用保留原始 Git 错误并显示中英文重试说明；正常 merged 删除、unmerged 普通拒绝 / 强制成功、当前分支和 linked 工作树分支保护通过。
- 全部 Go 测试 / race / vet、普通 / e2e 构建、原生同步与完整 detail-tabs 均通过，退出码 0。原生同步仍覆盖 Push / Fetch / Pull 和可点击 footer，没有新增实际分支删除场景。
- 同名同 SHA 重建视为同一目标；应用内锁不构成与外部 Git 的原子互斥。其他对象仍待核对，R04 未整体完成。未提交、安装或发布。

## 2026-11-16：标签删除的原对象与符号引用

- 真实 Git 测试复现旧确认删除后来移动的轻量标签、同一提交上重建的附注标签，以及符号标签误删其目标引用。新版确认捕获原始 ObjectSHA，后端使用 Git 旧 OID 条件删除和 --no-deref；旧 desktop 的符号标签删除也只移除选中名称。
- 替换 / 缺失拒绝、轻量 / 附注正常删除、两种入口符号引用保护及应用真实确认回归通过。失败保留原始 Git 输出，并按当前语言说明刷新重选。
- 全部 Go 测试 / race / vet、两种构建、原生同步及完整 detail-tabs 通过，退出码 0。原生脚本没有新增标签删除执行场景；同名同 OID 重建不可区分，R04 仍未整体完成。未提交、安装或发布。

## 2026-11-16：提交关联分支切换的目标

- 真实 UI Enter 复现选择器展示原提交、外部移动分支后却切到新目标。选择器和提交菜单现在携带选定 SHA，后端在应用内写锁下检查实际 refs/heads 目标；缺失 / 变化时保留 HEAD / Index / 工作字节，并提供当前语言的恢复说明和 Git 原文。
- 后端 moved / recreated 拒绝及 UI Enter 回归 RED→GREEN；正常目标成功、未跟踪文件 / Index 改动拒绝、linked worktree 拒绝通过。全部 Go 测试 / race / vet、两种构建、完整原生 detail-tabs 通过，退出码 0。
- 普通分支切换保留跟随分支的语义；检查与执行之间不保证外部 Git 进程原子互斥。没有新增原生选择器切分支场景。未提交、安装或发布。

## 2026-11-16：Abort / Skip 原操作身份

- 真实冲突仓库复现旧确认会中止 / 跳过后来开始的同类型 cherry-pick，即使 HEAD 未变化。新版确认捕获 Kind / HEAD / Token，后端执行前复核；刷新操作卡不会替换已打开确认的目标。
- 两类动作的 Git / UI 回归 RED→GREEN，拒绝后新操作、HEAD、Index 和冲突字节保留，原身份正常执行通过。中英文恢复说明通过。
- 全部 Go 测试 / race / vet、两种构建、原生操作卡和完整 detail-tabs 通过，退出码 0。原生操作卡再次验证冲突 Continue 禁用、Abort / Skip 取消与确认及正常 Continue；没有新增原生替换操作场景。检查不保证跨 Git 进程原子互斥。未提交、安装或发布。

## 2026-11-16：Continue 的操作身份

- 真实冲突仓库复现原操作卡会继续后来开始、已在外部解决并暂存的新 cherry-pick。新版 Continue 捕获显示的 Kind / HEAD / Token，执行前复核完整身份；拒绝时新操作、HEAD、Index 和工作字节保留。
- Git / UI 回归 RED→GREEN，原身份解决冲突并暂存后正常 Continue 会创建提交；全部 Go 测试 / race / vet、两种构建、原生操作卡与完整 detail-tabs 通过，退出码 0。
- 保留原交互，无额外确认步骤；原生正常 Continue 验证通过，尚无新增原生替换操作场景。不保证与外部 Git 进程原子互斥。未提交、安装或发布。

## 2026-11-16：分支重命名的原目标

- 原输入仅捕获名称，真实 Git / UI 回归复现分支移动或不同 tip 重建后仍重命名新目标。菜单 / footer 现在捕获完整 Branch，执行前复核原 SHA；缺失时保留 Git 错误并给当前语言恢复说明。
- stale-target 回归 RED→GREEN；正常重命名保留 Git 对当前 HEAD 和跟踪配置的迁移，已有名称拒绝且不覆盖。全部 Go 测试 / race / vet、两种构建、完整 detail-tabs 通过，退出码 0。
- 尚无新增原生重命名执行场景；同名同 SHA 重建不可区分，不保证与外部 Git 原子互斥。未提交、安装或发布。

## 2026-11-16：整合操作的双方目标

- 实际 footer 回归复现 Merge / Rebase 会跟随后来移动的目标分支、或合入后来切换的当前分支。确认现在固定选定提交 SHA，并显示原短 SHA，四类整合执行前核对确认时 HEAD / 分支。
- 真实 UI 四组两侧变化、Git 四类操作 × HEAD 提交 / 同 SHA 切分支 / detached 十二组拒绝及原目标四组正常执行通过。目标分支移动时只整合原提交；当前侧变化拒绝，保留 refs / HEAD / Index。刷新 UI 状态不重定向已开确认。
- 全部 Go 测试 / race / vet、两种构建、原生操作卡及完整 detail-tabs 通过，退出码 0。尚无新增原生 Merge / Rebase / Revert 发起场景，不保证与外部 Git 进程原子互斥。未提交、安装或发布。

## 2026-11-16：Undo 原提交与进行中操作

- 真实右键提交菜单的 Undo 回归复现旧确认撤销后来提交；后端又复现冲突已解决并暂存、但未结束的 cherry-pick 可被软重置。新版确认显示并捕获原 SHA / 分支，执行前复核 HEAD / 分支并要求空闲。
- UI / Git 回归 RED→GREEN：HEAD 前进、同 SHA 切分支与 detached 均拒绝；拒绝保留 refs / Index / 工作字节，正常撤销保留已有 Index 和工作文件，根提交拒绝。
- 全部 Go 测试 / race / vet、两种构建、原生操作卡与完整 detail-tabs 通过，退出码 0。尚无新增原生 Undo 执行场景，不保证外部 Git 进程原子互斥。未提交、安装或发布。

## 2026-11-16：Worktree 移除的原路径与对象

- 原确认虽然保存 Worktree，执行时却只传路径；真实 Git / UI 回归复现旧确认删除同路径改挂另一分支的新工作树。新版执行前核对原路径、HEAD、分支及锁定状态，拒绝保留登记 / refs / 替代目录字节。
- HEAD 前进、切分支、同路径替换和缺失拒绝通过；正常移除保留分支，脏 / 暂存 / 锁定 / 忽略文件 / 当前工作树保护通过。全部 Go 测试 / race / vet、两种构建、原生 Worktree 与完整 detail-tabs 通过，退出码 0。
- 原生再次验证创建、切换、移除取消、dirty 拒绝和正常干净移除；尚无新增原生同路径替换场景。同路径 / 分支 / HEAD / 状态重建不可区分，不保证与外部进程原子互斥。未提交、安装或发布。

## 2026-11-16：保留分支成功动作

- 原成功提示按名称删除分支，且 Repository 指针变化会使动作失效。新版捕获移除时名称＋HEAD，核对 root 并使用 checked 普通删除；同 root 刷新可用，切 root 失效。
- 真实辅助动作与实际移除后生成的 onAction 回归通过；完成真实刷新后原分支正常删除，移动后的 tip 保留。暂时恢复旧删除调用可复现实际成功按钮误删，恢复 checked 后通过。测试通过 busy 状态等待实际操作完成。
- 全部 Go 测试 / race / vet、两种构建、原生 Worktree 与完整 detail-tabs 通过，退出码 0。原生尚未点击成功按钮删除分支；同名同 SHA 重建与外部进程原子边界沿用。未提交、安装或发布。

## 2026-11-16：独立窗口通知队列

- 成功打开仓库原先清空两个通知队列，真实切换 / 重新打开回归复现独立设置失败提示和同 root 有效结果丢失。现在按已捕获 root 丢弃旧仓库结果，保留独立通知及顺序。
- 两组真实读取与现有通知 / 设置回归通过；全部 Go 测试 / race / vet、两种构建、原生子模块和完整 detail-tabs 通过，退出码 0。原生继续验证父仓库通知不进入 child，未新增原生双窗口焦点场景。
- 审计全部生产 onAction 赋值，目前只有两种成功提示带动作；没有生产带动作错误输入。当前长错误详情入口有效，未来此类输入需另作验证。未提交、安装或发布。

## 2026-11-16：标签推送原对象与远端

- 旧确认实际复现附注对象替换后推送新对象、远端地址变化后写入新远端；后端另复现轻量标签 tip 变化和独立 push URL 重定向。新版捕获 Tag.ObjectSHA 与 Remote.Fingerprint，执行前核对原始引用及有效 fetch / push 地址。
- 源 refspec 使用原对象 OID，保留附注对象，远端已有标签仍拒绝覆盖。真实 Git 与 UI 确认回归 RED→GREEN；缺失标签 / 远端、正常轻量 / 附注推送、中英文恢复说明与原始 Git 输出检查通过。全部使用临时本地 bare 远端。
- 全量 Go 测试 / race / vet、两种构建、原生同步与完整 detail-tabs 通过，退出码 0。原生仍覆盖既有 Push / Fetch / Pull 及错误下 footer 可点击性，没有新增原生标签推送执行场景。应用内写锁不能排除外部 Git 配置在检查后改变。未提交、安装或发布。

## 2026-11-16：远端分支删除租约

- 旧回调只捕获名称，真实 Git / UI 确认回归复现删除后来 tip 或重定向后的另一远端分支。新版捕获所选 SHA 与原 Remote.Fingerprint，使用显式 `--force-with-lease=refs/heads/<branch>:<originalSHA>` 删除租约。
- 后端覆盖 stale / refreshed tracking ref、fetch URL / push URL 重定向与远端目标消失；UI 在打开确认后刷新 remoteBranches，旧确认仍拒绝新 tip。正常删除清理原远端及跟踪引用并保留 HEAD；确认显示原 SHA，中英文错误说明保留 Git 原文。全部使用临时本地 bare 远端。
- 全量 Go 测试 / race / vet、两种构建、原生同步与完整 detail-tabs 通过，退出码 0。原生尚未执行远端分支删除；同 SHA 重建不可区分，远端配置复核仍有外部进程竞态边界。未提交、安装或发布。

## 2026-11-16：普通推送原提交

- 当前与非当前分支确认旧回调可推送后来提交或重定向后的远端，真实 Git / UI 回归 RED→GREEN。新版捕获源名称 / SHA、原远端与目标，使用不可变 OID refspec，保留非强制与既有子模块发布检查；原生确认断言包含 fixture 原 SHA。
- 正常非当前分支首次推送保留 HEAD / checkout 并设置上游。自定义 fetch 映射回归复现了假定 origin/* 跟踪路径的问题，改为写入 Git remote/merge 配置对后通过；配置分两次写入，失败可能留下部分配置。config.lock 场景验证远端已经推送完成时明确提示上游配置失败，保留 Git 原文。pre-push hook 中移动源引用仍发送原 OID。
- 首轮全量检查发现新增部分完成文案未登记动作组，补齐后全部 Go 测试 / race / vet 和两种构建通过。原生同步因等待旧文案失败，失败快照确认新原 SHA 文案正确；更新脚本后原生同步与完整 detail-tabs 重跑通过，退出码 0。
- 原生覆盖取消、首次普通推送、上游建立及既有 Fetch / Pull / footer 行为，尚无原生 stale / 非当前 / 自定义映射专项场景。强制推送仍走旧接口，待后续固定源、远端和租约；外部远端配置检查竞态边界保留。未提交、安装或发布。

## 2026-11-16：强制推送固定租约

- 旧隐式 force-with-lease 在确认后 fetch 时跟随更新的跟踪引用；真实 Git / UI 回归复现旧确认覆盖后来远端 tip。源变化与地址重定向同样复现。新版固定源 OID、原远端和显式预期远端 SHA，确认后刷新不改变租约。
- 原始租约可正常覆盖，不存在基线仅能创建不存在的目标，不能覆盖已有分支；读取远端引用失败时不弹出确认。真实回调、中英文恢复说明与 Git 原文保留检查通过。全部使用临时本地 bare 远端。
- 全部 Go 测试 / race / vet、两种构建、原生同步与完整 detail-tabs 通过，退出码 0。原生仍覆盖普通 Push / Fetch / Pull，尚无原生强制推送执行场景。同 SHA 重建不可区分，远端配置复核仍有外部进程竞态边界。未提交、安装或发布。

## 2026-11-16：子模块更新原目标

- 旧菜单只传路径，真实 Git / UI 菜单回归复现 Index 变化后更新到后来目标，另覆盖当前检出变化与未初始化 Index 变化。新版比较原名称、Index / Recorded、Current / Branch 与初始化身份；变化或目标消失时拒绝旧动作。
- 原已初始化 / 未初始化目标更新成功并保留父 Index；既有 dirty / pending operation 保护通过。首轮中文术语检查因 Index 未登记失败，译为“暂存区指针”后，全部 Go 测试 / race / vet、两种构建、原生子模块、Worktree 与完整 detail-tabs 通过，退出码 0。
- 原生继续验证 Unicode / 空格路径初始化、恢复到父 Index、dirty 拒绝和子仓库独立导航；没有新增原生 stale-Index 场景。子模块 URL 配置未捕获指纹，外部进程仍可能在复核后改变 Index；没有跨进程原子保证。Worktree 表单起点保留可编辑命名引用在提交时解析的行为。未提交、安装或发布。

## 2026-11-16：原生引用详情动作

- `python3 -I scripts/native-lobe-reference-smoke.py` 在 macOS 独立 HOME / Git 配置、临时工作仓库及本地 bare 远端中执行。通过坐标点击详情 footer，验证非当前分支推送取消 / 发布 / 设置上游（主分支未切换）、重命名保留 tip / 跟踪配置、本地分支删除取消 / 确认且远端保留、附注标签推送保留原对象、本地标签删除取消 / 确认且远端保留、远端分支删除取消 / 显式租约确认并清理跟踪引用。
- 每段核对主 HEAD、符号分支、Index、Unicode 文件字节与干净状态。首轮推送通过后因本地 topic 与 origin/topic 同名而选行歧义，修正本地 / 远端身份条件后完整重跑通过，退出码 0。日志、最终快照与截图位于 `build/lobe-reference-smoke/`，最终截图已查看。
- 本轮仅增加脚本和文档；生产源码保持前轮已通过全量 Go / race / vet、两种构建及三项原生回归的状态。系统右键菜单无法由当前 MyGo automation 选择，Force / prune 仍不算原生验收通过；本脚本亦不证明 stale 对象竞态、真实网络或跨平台。未提交、安装或发布。

## 2026-11-16：原生 Merge / Rebase 发起

- 引用脚本扩展临时分叉场景：main 与 integration-topic 各有独立文件，footer Merge 取消保持原 HEAD，确认产生正确的两个父提交并保留双方字节。再造 main / rebase-onto 分叉，footer Rebase 取消保留源 HEAD，确认后的父提交为原 onto SHA，完整 source 消息及双方文件保留，main / 工作区保持正常。
- `python3 -I scripts/native-lobe-reference-smoke.py` 完整通过，退出码 0，包含此前六组引用动作；最终截图已查看。仅脚本与文档变化，生产源码无修改，沿用前轮完整 Go / 构建门槛证据。无冲突发起不替代冲突编辑、Revert 发起、真实远端或跨平台验收。未提交、安装或发布。

## 2026-11-16：设置重置确认的双窗口归属

- 新原生脚本在独立 HOME / 临时脏仓库分别请求设置重置与主窗口 Discard all。首次因未选择工作区行、第二次因省略号匹配失败而未进入核心检查；修正探针后确认框实际出现在主窗口，双窗口失败快照与日志保留在 `build/lobe-window-dialog-lpsnblin/`。
- 原因是设置页重置入口使用基于键盘焦点的 `settingsNoticeOwner`。现改用明确的设置窗口 owner。单元回归初次调用 IsFocused 等待不存在的原生事件循环超时，不能计为 RED；调整夹具后有效复现错窗，最小修复后通过。
- 确认相关测试、全量 Go / race / vet、两种构建、新双窗口与完整 detail-tabs 原生回归通过，退出码 0。完整日志 `build/lobe-settings-owner-final.log`、结果 `build/lobe-window-dialog-1_f5lhbq/results.json` 已读；设置与主窗口确认截图已查看。两种请求顺序保留原确认及 owner，取消后 HEAD / refs / 原始 Index / 状态 / 工作字节不变。
- 追加并发输入检查：主窗口 New branch 填入 `retained-window-draft`，设置重置确认同时显示且各在原窗口；取消设置后草稿精确保留，再取消主窗口输入，完整仓库指纹不变。首轮因 `wait --text` 不匹配 Branch name placeholder 超时，快照证明命名框已正确打开；改为等待可见标题后通过。日志 `build/lobe-window-concurrent-check-2.log`、结果 `build/lobe-window-dialog-zlhce9k1/results.json` 已读，两窗口并发截图已查看。
- 命令目标切换不等于系统窗口激活；当前工具的元素 focus / 坐标事件也不调用原生窗口 Focus。本轮不覆盖真实 OS 焦点切换、实际恢复设置或成功 / 失败通知路由，R04 保持开放。未提交、安装或发布。

## 2026-11-16：原生 Reset 三种模式与 Undo

- 新脚本 `native-lobe-reset-smoke.py` 每模式使用独立临时仓库 / HOME / 原生进程，跟踪文件含 Unicode / 空格路径，保留不同暂存与工作版本及未跟踪二进制文件。历史行 focus + Shift+F10 成功打开应用内菜单，可选择 Reset 与 Undo；系统右键菜单限制不能泛化为这些应用内入口不可自动化。
- 四组一次通过，退出码 0。各模式先取消并核对完整原 HEAD / refs / 原始 Index / 状态 / 字节。Soft 与 Undo 执行后退回原父提交、保留 Index / 工作字节；Mixed 改写目标 Index 而保留工作字节；Hard 二次确认取消不写，确认后恢复目标 Index / 工作字节。main 分支身份和未跟踪二进制文件均保留。
- 完整日志 `build/lobe-reset-native-check.log`、四组结果 `build/lobe-reset-84j5c3xn/results.json` 已读；Hard 二次确认、Undo 完成截图已查看。追加 `--stale` 四组通过：确认打开后用独立 Git commit-tree / update-ref 推进临时仓库 HEAD，Hard 在二次确认后推进；三种 Reset 和 Undo 均拒绝旧基线，完整外部 HEAD / refs / 原始 Index / 状态 / 工作字节与未跟踪文件不变。日志 `build/lobe-reset-stale-native-check.log`、结果 `build/lobe-reset-4wr2c2i1/results.json` 已读，Hard / Undo 拒绝截图已查看且恢复说明可见。仅脚本 / 文档变化，生产验证沿用设置 owner 修复后的完整门槛；同 SHA 切分支、进行中操作替换、跨进程原子互斥、真实用户仓库和跨平台不在本轮结论内。未提交、安装或发布。

## 2026-11-16：Force/prune 与读取中确认保留

- 新 `native-lobe-force-prune-smoke.py` 使用独立临时脏仓库、HOME 与本地裸远端。顶栏 Push / Fetch 聚焦后 Shift+F10 打开应用绘制的菜单，成功验证 Force 的源提交 / 显式租约、取消不写远端、分叉替换到原源 SHA；远端推进后再 fetch，旧确认仍被拒绝，远端与本地脏状态不变。prune 只清除 origin/ghost，保留本地 ghost 与完整 HEAD / Index / 字节 / upstream，远端引用不写入。
- 原生前两次失败均发生在等待完整错误说明：未进入已有的 View details 流程。修正后核对本地化恢复说明和 Git stale info；不能把原生超时归因于确认丢失。源代码调查另由独立单元测试有效复现 loading / busy 时确认被消耗，因此保留这项修复：仓库主确认按钮在读取 / 写入期间禁用，目标 / 回调保留，完成后可执行一次，取消仍可用；无仓库 root 的独立设置确认继续可用。
- 相关回归 RED→GREEN，`build/lobe-force-prune-final.log` 中全量 Go / race / vet 和两种构建通过；其后原生超时导致剩余脚本未执行。修正探针后的 `build/lobe-force-prune-followup.log` 29 行已全读，Force/prune、双窗口并发草稿、旧 HEAD 四组 Reset / Undo 与完整工作台全部通过。结果 `build/lobe-force-prune-6tlagcn8/results.json`、`build/lobe-window-dialog-j6_pga2e/results.json`、`build/lobe-reset-5s5jfluk/results.json` 已核对；租约拒绝详情与 prune 完成截图已查看。
- 不证明真实网络认证、任意远端配置、真实系统焦点、与外部 Git 原子互斥或跨平台。未提交、安装或发布。

## 2026-11-16：Cherry-pick / Revert 原生发起

- 新 `native-lobe-integrate-smoke.py` 创建两个独立临时仓库 / HOME / 原生进程，fixture 含共同 base、带完整正文与 Unicode 路径文件的 topic 提交及 main 独立变更。历史行聚焦后 Shift+F10 选择应用内 Cherry-pick / Revert，先取消并核对 refs / 原始 Index / 状态 / 字节，再确认实际执行。
- 两组一次通过，退出码 0。新提交父 SHA 为原 main，topic 引用不变；Cherry-pick 完整保留源说明与两文件精确字节，Revert 正文包含原 SHA 且 tree 恢复为基础版本。工作区 / Index 干净，无 CHERRY_PICK_HEAD / REVERT_HEAD / MERGE_HEAD / sequencer 残留。
- 完整日志 `build/lobe-integrate-native-check.log`、结果 `build/lobe-integrate-dwh605ha/results.json` 已读，两张最终截图已查看。仅脚本 / 文档变化，沿用前轮全量生产检查；不证明应用内冲突编辑、操作替换竞态或跨平台。未提交、安装或发布。

## 2026-11-16：最新工作台显示矩阵

- 确认保护修复后的当前 Lobe 二进制完整重跑 `python3 -I scripts/native-narrow-shots.py`，退出码 0：16 组 × 6 场景，96 张截图和对应快照。覆盖三种宽度、中英、浅深色和四组 1.15 字号；列宽、活动标签 / 关闭入口、固定分组、提交可用状态、作者邮箱与 fixture Git 指纹均通过。
- 完整日志 `build/lobe-latest-visual-matrix.log` 已读。图像保存在 `build/narrow-shots-82hkcttz/`，抽查 980 英文深色大字号可提交表单、980 中文浅色大字号长文件标签、1600 英文浅色大字号完整正文；关键控件可见。脚本每轮使用独立 `build/narrow-shots-*`，避免覆盖既有证据。
- 该结果仅覆盖工作台，不覆盖全部页面、悬停 / 焦点或全部菜单操作。核对 `native-lobe-reference-smoke.py` 的 35 长分支用例仅验证无结果 Enter、鼠标选行、聚焦 Enter checkout 和两个远端搜索隔离；R08 / R09 保持未完成。生产代码未变，沿用前轮全量检查；未提交、安装或发布。

## 2026-11-16：历史菜单窄窗可达性

- 新 `native-lobe-menu-access-smoke.py` 创建独立仓库、HOME、35 个长 Unicode 分支及两个本地裸远端，980×640 下核对含完整消息复制与 HEAD Undo 的 18 项菜单：全部 bounds 在窗口内、Down 逐项焦点可达；已查看菜单截图。
- 分支 picker 与底部 Reset 通过当前快照 DIP 坐标实际点击打开，Reset 另由方向键聚焦后 Enter 打开。picker 使用 Escape，Reset 使用 Cancel 关闭；完整 HEAD / refs / Index 条目 / status / 文件字节保持。首轮激活探针误找 picker Cancel 而失败，快照及源码确认实际无此按钮；改用既有 Escape 方式后完整通过。
- 日志 `build/lobe-menu-access-activation-followup.log`、结果 `build/lobe-menu-access-7kuy4v8m/results.json` 已读。后续加入 2001 年不同作者提交，四种过滤菜单坐标点击均写入正确可见查询、保留当前匹配并排除旧提交；搜索 Enter 保持 checkout 与完整 fixture Git 状态。首轮因解析器要求带标签而未定位只有 placeholder 的搜索框，按快照修正探针后通过；证据 `build/lobe-menu-access-1zx921o_/results.json`，过滤截图已查看。
- 最新完整脚本另通过真实读取失败恢复：无匹配结果与读取错误分开；仅移开自建 fixture `.git` 后 Refresh 显示错误和 stale 说明、保留旧历史，finally 恢复 `.git`，坐标 Retry 清除错误并恢复历史读取，HEAD / refs / Index 条目 / 状态 / 字节不变。完整日志 `build/lobe-menu-access-recovery.log`、结果 `build/lobe-menu-access-sfbuiw3v/results.json` 已读，错误截图已查看。仅测试仓库受影响，未触及源码 checkout 的 Git 元数据。
- 最新扩展完整通过全部 11 个表单 / 确认入口坐标打开与取消：picker、detached checkout、Merge / Rebase、普通 / 附注标签、新分支、Cherry-pick / Revert、Undo / Reset；每次取消都核对 Git 状态。加四种过滤共 15 项鼠标激活。`build/lobe-menu-access-dialogs.log` 和 `build/lobe-menu-access-vmz51dcg/results.json` 已读，其余已有断言仍全通过。
- 三项复制命令未实际点击，避免覆盖当前系统剪贴板；全部 18 项键盘焦点覆盖不等于三项复制已执行。R08 及跨平台验收继续开放；仅脚本 / 文档变化，沿用前轮全量生产检查，未提交、安装或发布。

## 2026-11-16：Merge 冲突应用内取边

- 新 `native-lobe-conflict-side-smoke.py` 使用两套独立仓库 / HOME / 原生进程。fixture 创建同一 Unicode / 空格路径的真实 Merge 冲突；通过文件行 Shift+F10 → Use ours / Use theirs，核对工作字节和 stage-0 内容精确等于原 Index 第 2 / 3 阶段，不再有 unmerged 条目，引用、MERGE_HEAD 与无关二进制保持。
- 未解决时 Commit 禁用；取边后提交表单启用，写入完整消息，核对正确双父提交、topic 保留、干净 Index / 工作区及 Merge 标记清理。两组通过，完整日志 `build/lobe-conflict-side-followup.log`、结果 `build/lobe-conflict-side-i1wtl9ae/results.json` 已读，Theirs 完成截图已查看。
- 首轮探针把 disabled 直接拼在按钮标签后，实际有 keyshortcuts 属性间隔；快照证明产品正确禁用，修正解析后通过。仅脚本 / 文档变化，沿用前轮全量生产检查。本证据覆盖 Merge 整文件取边，不覆盖三方手工编辑、其他操作取边、删除型冲突或跨平台，R11 继续开放。未提交、安装或发布。

## 2026-11-16：工作区总览就绪节裁剪

- 1,000 个文件节的展开 / 折叠回归复现全部屏外标题构建；`detail_overview.go` 改为对高度确定的就绪补丁和折叠标题连续段复用 `virtualPatchRows`。加载 / 错误 / 空补丁仍正常布局，以保留可变文字高度及可见读取优先级。另复现并修复远处选中行因外层裁剪无法显露的问题，保留匹配 root / generation / path 的目标节。
- 上述回归 RED→GREEN；全量 Go 38.156s、race 157.177s、vet、两种构建及完整详情标签十组原生回归通过，完整日志 `build/lobe-overview-sections-check.log` 已读。
- 新 `native-lobe-overview-sections-smoke.py` 使用独立千文件脏仓库 / HOME。首次末尾断言失败时快照显示文件仍在 Loading diff，探针等待标题而非补丁就绪；修正为等待队列加载结束并核对末尾标题在屏内及实际补丁后通过。就绪首屏 3 个节标题、末尾 6 个；最后补丁与返回首个补丁可达，HEAD / refs / 原始 Index / status / 全部工作字节保持。日志 `build/lobe-overview-sections-native-followup.log`、结果 `build/lobe-overview-sections-ny31jbxn/results.json` 已读，末尾截图已查看。
- 只证明就绪节构建范围与导航保护，不是帧耗时或 RSS 改善测量；变高占位节和既有偶发批量恢复无响应仍未解决，R16 不关闭。未提交、安装或发布。

## 2026-11-16：Rebase 冲突取边含义

- 有效 RED 复现 Rebase 菜单仍显示 current branch / incoming；现文件列表和 Diff 标题菜单共用上下文文案，中文为「变基基底 / 重放提交」，英文为 rebased base / replayed commit，成功提示同样区分；后端 Git 命令未变。
- 文案回归 GREEN；完整 Go 37.975s、race 159.564s、vet、普通 / e2e 构建及完整详情标签十组原生回归通过，`build/lobe-rebase-side-copy-green.log` 与 `build/lobe-rebase-side-check.log` 已读。
- `native-lobe-conflict-side-smoke.py --rebase` 两组验证真实 stage 2 / 3 身份、精确工作 / 暂存字节、上下文成功提示，随后 UI Abort 恢复原 main / refs / 干净状态。结果 `build/lobe-conflict-side-yr7qi2k_/results.json` 已读，Theirs 解决后截图已查看；同轮 Merge Ours / Theirs 完整提交回归结果 `build/lobe-conflict-side-0_8t0nv6/results.json` 已读并通过。
- 不覆盖 Rebase 取边后 Continue、三方手工编辑、删除 / 二进制提示或 D4 标记策略。R01 / R11 保持开放；未提交、安装或发布。

## 2026-11-16：冲突只读预览与外部处理提示

- 总览与文件标签增加中英文只读冲突说明、外部处理文本 / 二进制后使用原文件菜单「标记为已解决」的路径；UD / DU / DD 说明取不存在一侧会删除并暂存路径。含提示的节依赖换行高度，保持实际布局；未更改后端取边 / 暂存命令或残留标记策略。
- 16 组渲染及高度回归有效 RED→GREEN，千节裁剪回归通过。全量 Go 38.365s、race 157.414s、vet、两种构建及完整详情标签十组原生通过，完整日志 `build/lobe-conflict-preview-check.log` 已读。
- 现有真实文本 Merge / Rebase 两种取边新增未解决时提示等待，全部通过；结果 `build/lobe-conflict-side-18fu2lww/results.json` / `build/lobe-conflict-side-m7flbpur/results.json` 已读。Rebase Theirs 解决后截图已查看；该截图不展示未解决提示，不据此声称提示的原生视觉验收。真实删除 / 二进制取边样本后续共 8 组通过：Merge / Rebase 各两种取边核对精确阶段字节或暂存删除，Merge 完整提交、Rebase Abort 恢复原 main（包括原有删除）、refs 和无关二进制保护。完整日志 `build/lobe-conflict-shapes-check.log` 及四套结果 `build/lobe-conflict-side-kd56jv15/`、`build/lobe-conflict-side-q4mgk0gb/`、`build/lobe-conflict-side-7wlhcnt9/`、`build/lobe-conflict-side-xncjf9y6/` 的 results.json 均已读。原删除冲突截图中的提示在屏外；后续补验 Merge 删除型两种取边 / Rebase 二进制两种取边，滚动后提示屏内可读，结果 `build/lobe-conflict-side-klszt012/results.json` / `build/lobe-conflict-side-38zl3_qw/results.json` 已读，两类 unresolved 截图已查看。第一轮只显露首条提示使第二条仍在屏下，改为显露最后一条后通过；后一轮 Rebase 成功提示已出现但刷新未完成、Abort 仍禁用，补等待实际可用后恢复流程通过。日志 `build/lobe-conflict-hints-visible-followup.log` 与 `build/lobe-conflict-hints-ready.log` 已读。这两次失败来自探针的显露 / 就绪假设，没有新增生产修改。
- R01 / R11 仍开放；未提交、安装或发布。

## 2026-11-16：旧 Abort / Skip 确认与操作替换

- 新 `native-lobe-operation-stale-smoke.py` 使用两套独立仓库 / HOME、980×640 窗口。打开原 cherry-pick 的 Abort / Skip 确认后，由 fixture 外部替换为同类型、同 HEAD、不同目标提交的真实冲突；旧确认均拒绝并显示操作已变化的恢复说明。
- 操作标记 / MERGE_MSG、HEAD / 分支 / refs、逻辑 Index / 状态、Unicode 冲突字节与无关二进制均保持替换后的精确值。完整日志 `build/lobe-operation-stale-followup.log`、结果 `build/lobe-operation-stale-gwtngzfl/results.json` 已读，Skip 拒绝截图已查看。
- 首轮 Abort 已通过；Skip 探针随后尝试点击模态框后方的 Refresh，实际辅助树只暴露模态确认，故失败属于探针路径错误。改为直接确认路径后两组通过，生产代码未变。确认期间刷新仍只有既有 Go 回归，不据此声称原生覆盖；旧 Continue 卡片、其他操作类型和跨进程原子互斥未验收。
- R04 继续开放；未提交、安装或发布。

## 2026-11-16：设置页布局与重置可达

- 设置页实际最小尺寸 560×480，独立于主窗口 980×640。新增 `TestSettingsChoicesFitMinimumWindowAndResetCanBeReached`：中英文 × 浅深色 × 标准 / 1.15 字号共 8 组验证关键选项横向边界、完整滚动范围、底部重置坐标激活及确认所属设置窗口，全部通过；日志 `build/lobe-settings-layout-measured.log` 已读。新增测试后的全量 `go test ./...` 通过，lobedesk 37.882s，完整日志 `build/lobe-settings-layout-suite.log` 已读；`git diff --check` 无错误。
- 初轮几何通过但缺少设置窗口生命周期使确认被清理；加入模拟窗口后，完整 settingsRoot 会调用原生 IsFocused，测试等待系统事件循环。停止该任务并用 15s 超时栈定位后，按既有确认测试方式直接渲染实际设置页和确认视图，回归通过。属于测试夹具问题，生产代码未变。
- 新 `native-lobe-settings-layout-smoke.py` 在独立仓库 / HOME 中通过默认 820×720 设置窗口八组矩阵：所有交互控件横向边界、滚到底部后坐标打开重置确认及取消、Git 状态与偏好文件精确不变。完整日志 `build/lobe-settings-native-layout.log`、结果 `build/lobe-settings-layout-40ndm5ll/results.json` 已读。共 24 张截图，查看英文深色大字号外观、中文浅色大字号底部和确认代表图。
- 当前原生自动化无窗口尺寸修改入口，因此不声称原生最小尺寸验收；所有悬停 / 焦点及真实设置保存 / 重置通知路由不在本轮范围。R09 保持开放；未提交、安装或发布。

## 2026-11-16：Unicode 宽行与完整尾标记

- 混合中文 / 全角 / ASCII 的宽行暴露 rune 数低估及两个普通字宽高估；实际字形回归约束滚动宽度两端，最终按宽字符 1 em / 普通 0.62 em 估算。随后截图显示省略号，实际交互行布局回归再复现代码区少约 14–16 DIP，修正 gutter / padding / hover 总预留为 138 DIP，专项 RED→GREEN。
- 最终完整日志 `build/lobe-unicode-row-final-check.log` 已读：全量 Go 41.012s、race 165.384s、vet、两种构建、完整详情标签十组原生通过；详情探针仓库确认显式定位主窗口，不再随设置窗口 OS 焦点查找。
- 独立仓库 / HOME 下混合 Unicode、1.15 字号、4,000 行全替换的三轮原生通过，8,000 增删行、纵横导航及完整指纹保持。结果 `build/lobe-long-patch-i6oljx2b/results.json` 已读，wide-tail-1.png 已查看，`wide-tail-3999` 完整可读。计时 / RSS 与失败调查记录见性能基线；不声称任意 Unicode 字形精度、超过 20,000 DIP 的极长行或其他投影横向行尾已验收。未提交、安装或发布。

## 2026-11-16：Rebase 取边后 Continue

- `native-lobe-conflict-side-smoke.py --rebase-continue` 在独立仓库 / HOME 中通过文本 / 二进制 / 删除 × 两侧共六组原生取边后 Continue。Ours 保留变基基底并省略空重放，Theirs 生成以 topic 为唯一父的重放提交，原完整消息 / 作者 / 邮箱 / 作者日期保留；两种结果均精确核对 Index / 工作字节或删除、main / topic、无关二进制、干净状态和无残留操作。
- 完整日志 `build/lobe-rebase-side-continue.log`、`build/lobe-rebase-side-continue-shapes.log` 及三套结果 JSON 已读：`build/lobe-conflict-side-o40jntnx/`、`build/lobe-conflict-side-oziemhcb/`、`build/lobe-conflict-side-_kqq593z/`；文本 Theirs 最终截图已查看，操作卡消失、main 新提交正确、工作区干净及 Continued 提示。
- 只有隔离脚本和记录变化，沿用最新完整验证通过的生产构建；不代表三方手工编辑、其他操作取边或 D4 标记策略已决定。未提交、安装或发布。

## 2026-11-16：原生旧 Continue 操作卡

- 扩展 `native-lobe-operation-stale-smoke.py`，原已解决 cherry-pick 冲突的 Continue 可用后保存控件引用；外部替换成同类型、同 HEAD 的不同目标并解决，直接点击原控件，精确恢复提示拒绝。
- 原生 Abort / Skip / Continue 三组完整通过，替换操作 / MERGE_MSG、HEAD / 分支 / refs、Index / 状态、工作字节和无关二进制完整保持。完整日志 `build/lobe-operation-stale-continue.log`、结果 `build/lobe-operation-stale-9oyofyxg/results.json` 已读，Continue 拒绝截图已查看。
- 只修改隔离探针和记录，沿用最新完整门槛生产构建；不代表其他操作类型、模态确认期间刷新或外部进程原子互斥。未提交、安装或发布。

## 2026-11-16：设置保存 / 重置结果与窗口归属

- 实际保存失败原生调查引出系统焦点决定通知 owner 的问题，已改为设置页写入与语言修改 / 重置显式指定设置窗口；主窗口分栏等偏好写入仍归主窗口。中英文实际外观卡片保存失败、重置成功 / 失败的队列归属回归有效 RED→GREEN。
- 全量 Go 38.131s、race 161.385s、vet、两种构建通过，完整日志 `build/lobe-settings-notice-owner-check.log` 已读。原生中英文各一套独立 HOME / 脏仓库，通过强制写入失败保留旧配置与当次主题、恢复保存、实际重置默认值并保留语言、主窗口不显示设置结果、完整 Git 指纹不变；详情标签原生十组通过，日志 `build/lobe-settings-persistence-followup.log`、结果 `build/lobe-settings-layout-terpjqit/results.json` 已读，中文重置后截图已查看。
- 首次及修复归属后等待提示均曾超时；进一步确认设置窗口无 OS 键盘焦点，既有规则不消费提示队列。保留该规则，原生两种提示可见字段均为 false；队列归属与真实持久化通过，不等于 OS 焦点切换后提示可见验收。未提交、安装或发布。

## 问题与复现记录

- 缓存阶段一次完整原生运行：选择 `Tab fixture changes` 后，初始快照显示正确 SHA，等待完整说明期间却变成 `Tab fixture base`，导致 20 秒超时。应用日志没有 panic、重复 key 或 fatal。相同二进制重跑、随后调度及本地检测的完整回归均通过；原因仍未确定，不能宣称修复。若再次出现，保留按键 / press 顺序、前后快照、当前原生焦点和应用日志，归入 R04 调查。
- 本地文件观察使用 Git 状态和当前改动文件的元数据；刻意保留相同大小、mtime、模式的字节改写不保证识别。没有逐次读取或哈希全部大文件内容。
- 曾有版本因背景检测错误文案未登记到动作文案覆盖表而使 `make check` 失败；已按现有机制登记背景读取原因，最终全量门槛通过。

## 安装与下一次日常试用

此前单独安装的 `SuperGit Lobe.app` 是更早的测试包，没有自动更新本轮工作台、缓存或本地检测修改。`make app` 当前仍打包旧 desktop；最新 Lobe 可通过 `make lobe-run REPO=/path/to/repository` 运行，构建入口见 [打包说明](packaging.md)。本轮没有覆盖安装或发布。

尚无用户真实日常开发会话、Windows / Linux 原生验证、开发者签名 / 公证及真实远端关键流程的新增验收记录。后续日常试用记录应附上构建日期 / 包版本与源码状态、平台、用户选择的场景、实际结果及问题复现步骤；只有执行后才追加为通过。候选主线能力差距仍按 R11 逐项核对。
