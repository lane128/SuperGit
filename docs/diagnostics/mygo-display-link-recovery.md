# MyGo display link 恢复候选

状态：2026-11-16，候选依赖补丁，尚未接入应用默认构建。不能据此关闭 R16 或宣称发布版本已经修复。

对应版本：`github.com/egoist/mygo v0.8.4-0.20261108082913-9233663736ca`。

## 证据

批量改写并恢复 1000 个父仓库文件和 1 个子模块文件后，原生测试等待 `No changes` 超过 60s。后续自动化窗口、树和截图请求仍成功，Git 指纹已恢复，界面仍显示旧的 1001 项变更。

临时诊断构建 `build/lobe-performance-qp7scfz2/app-2.log` 显示模型已恢复 0 项变更、历史已加载 12025 条、加载标志清除，观察器继续运行，但界面帧号一直为 428。

进一步诊断 `build/lobe-performance-oywhgfdi/app-1.log` 显示 UI host 没有在构建帧或等待 GPU，窗口可见且已请求帧。Darwin surface 的 `linkRunning=true`、`due=true`、`deferred=false`、`linkPaused=false`，但距上一帧约 40s。底层 display link 未继续交付帧；尚未解释 AppKit 为什么停止回调。

`surface.RequestFrame` 先判断 `linkRunning`，因而即使上一帧已过去很久，也不会执行已有的 `drawSoon` 恢复路径。该路径中存在可以独立复现的恢复缺口。

## 候选

同目录 `mygo-display-link-recovery.patch` 仅将已有的“距上一帧超过两个刷新周期且窗口可见或尚未绘制首帧”分支移到 `linkRunning` 判断之前。没有增加轮询、定时器、Git 重试或用户设置；隐藏窗口继续使用原有可见性判断。

回归 `TestRequestedFrameRecoversWhenRunningDisplayLinkStopsTicking` 模拟 display link 仍标记为运行但已停帧，随后请求一帧并排空主线程队列，应收到一个非 paced 恢复帧。测试使用现有 `frameHandler`，不依赖真实 display link 或睡眠。原实现 RED（frames=[]）；候选与已有 `TestDrawSoon` / `TestDeferredFrames` GREEN，0.420s。原生框架须先 `load()`，否则候选触达刷新率查询会产生测试初始化错误。

补丁语法与针对原版本的隔离 `git apply --check` 均通过。完整 Darwin 包随后通过，1.155s。第一轮应用全量检查因诊断 `.go` 文件误放在 build 根目录而额外发现混合 package 失败；这些本次生成的文件已改为 `.go.txt`。该失败不是应用回归；首次失败后普通 e2e 二进制已自动恢复。

后续 `build/lobe-display-link-candidate-followup.log` 的完整检查通过：应用 Go、race（lobedesk 156.612s）、vet、候选 e2e 构建、三轮批量保存恢复，以及全部 10 组详情标签原生烟测。`build/lobe-performance-rgvqnd6v/results.json` 为 `state_preserved=true`，三轮恢复到干净状态分别约 2.645 / 5.628 / 2.629s，Files 均为 38 个行控件。三份最终快照均显示 No changes；此时历史仍在本次恢复后的重读中，不据此声称每轮退出前完整历史已再次加载。已查看第三轮脏状态截图。验证结束后普通固定依赖的 e2e 二进制已恢复。

新增可重复入口 `scripts/check-mygo-display-link-recovery.py`，会在系统临时目录复制原固定依赖，先只加回归并要求原实现出现精确 RED，再应用候选并运行完整 Darwin 包。入口自身已实际执行通过，证据 `build/mygo-frame-recovery-lx8jxlz7/`：原实现 RED、候选 GREEN 1.688s，项目默认依赖未改变。

## 40 行选区的追加对照

长补丁探针在普通固定依赖下完成 40 行暂存后，Staged 选区端点仍不可见。直接记录选区状态的诊断构建 `build/lobe-long-patch-743lndfm/` 显示：选区只增长到 38 行（含补丁头的光标索引为 42），最后两次按键没有进入选区处理；CLI 从这一阶段开始报告窗口 10s 内未稳定。无原生窗口的实际 diffBody / 快捷键测试则能选满 40 行并显示端点，完整应用 Go 测试通过。因此尚不能将此失败归因于应用的选区滚动算法。

仅改用隔离的帧恢复候选、保持相同应用诊断 overlay 和探针流程后，`build/lobe-long-patch-7rjsvruv/` 记录 Staged 选区完整达到 40 行、端点可见，并打开 `Unstage 40 selected lines` 菜单。该轮仍未通过：点击后等待短暂成功提示超时，失败快照显示 Staged 分组已空、原未暂存补丁已恢复；但未在清理前完成精确 Git 指纹断言，不能据此验收往返写入。点击命令的 `--settle` 已先等待 10s，可能错过提示的显示期。探针改为点击后立即等待提示，随后仍要求完整 Git 指纹恢复；后续失败也会在夹具清理前保存 `failure-N.git.json`，保留实际状态与原始指纹。

修正提示观察顺序后，候选＋应用状态诊断 overlay 的三轮对照通过，证据 `build/lobe-long-patch-ja37ajwx/results.json` 与 `build/lobe-native-range-candidate-notice.log`：每轮 Staged 选区均达到 40 行，暂存后的 Index 精确等于原版本去掉前 40 行，工作区字节 / HEAD / refs 保持；取消暂存后、外部夹具清理前即恢复完整原始指纹，最终 `state_preserved=true`。第三轮截图已查看，包含 `Selected lines unstaged` 与 Staged Files 0。这些诊断构建的 RSS 不作为默认性能基线。

这一对照支持候选改善了本次原生输入推进，但未复测底层 display link 状态，亦未证明所有未稳定警告的原因。诊断结束后已恢复普通 e2e 构建，默认依赖没有变化。新增 `--native-long-range` 可独立运行这组三轮验证。该持久入口现已实际通过：`build/mygo-frame-recovery-l_4ol70i/` 记录原实现精确 RED（0.441s）、候选完整 Darwin 包 GREEN（1.470s）、无应用诊断 overlay 的候选构建及原生三轮精确往返全部通过，`ordinary_build_restored=true`，补丁 SHA256 与此前一致。原生结果 `build/lobe-long-patch-u19cb976/results.json` 完整核对，第三轮最终截图显示成功提示与 Staged Files 0。候选验证范围已确认；默认应用仍未接入，普通依赖多行原生验收保持未完成。

追加的 40 行添加选区也已通过无诊断 overlay 的持久入口：`--native-long-range --range-line-kind add`，门槛记录 `build/mygo-frame-recovery-w27bmngy/`，精确原实现 RED 0.455s / 候选完整 Darwin GREEN 1.454s；三轮结果 `build/lobe-long-patch-gv7wvl0w/results.json` 完整核对。暂存的 Index 在各原始行之后精确插入所选末 40 条 working 行，保持替换配对顺序；从 Staged 新行号 3962 到 4040 选择 40 条添加行、跳过上下文，取消暂存后、外部清理前完整指纹恢复。代表暂存 / 取消暂存截图已查看，普通 e2e 自动恢复，默认依赖未变。此结果不补足默认构建的多行验收，也不是默认性能基线。

## 验证与接入边界

临时依赖副本与独立 `build/refresh-diagnostic.mod` 用于候选测试，不改模块缓存或项目 `go.mod` / `go.sum`。候选验证包含应用 Go / race / vet、三轮同规模原生批量保存恢复和详情标签烟测，结束后无论成功失败均恢复普通依赖的 e2e 构建。候选构建的 RSS 不并入发布性能基线。

复现命令（macOS）：

```sh
python3 -I scripts/check-mygo-display-link-recovery.py
# 加应用全量检查和原生探针，结束后恢复普通 e2e 构建：
python3 -I scripts/check-mygo-display-link-recovery.py --native
# 独立验证 40 行跨视口暂存 / 取消暂存；也可与 --native 合用：
python3 -I scripts/check-mygo-display-link-recovery.py --native-long-range
# 同样验证 40 条添加行（普通依赖仍未接入候选）：
python3 -I scripts/check-mygo-display-link-recovery.py --native-long-range --range-line-kind add
```

`--native` 的组成步骤已通过上述候选门槛，新入口本身目前只运行过默认 RED→GREEN 模式，不将二者混称为整条新命令已执行。

本补丁可用于依赖维护者审查或后续受控接入；默认应用仍使用原固定版本。候选原生验证通过也只能支持恢复路径有效，不能证明 display link 停止回调的触发原因已消除，亦不能证明所有停帧场景已覆盖。完整验证结果见任务 ledger 与 `docs/performance-baseline-2026-11-16.md`。
