# 历史 Graph 与引用标签样式验收

状态：实现、自动化验证与两轮代码审查完成；用户已实机确认，已合并到 main 并推送。

## 构建与范围

- 分支：`worktree-history-visual-styles`，基于 `ef793b2`。
- 显示版本：SuperGit 0.1.0 (55)，本地未提交功能构建。
- 设计：`docs/features/history-visual-styles.md`，用户已批准。
- 实现：三种 Graph（rounded / chamfer / compact）× 两种引用标签（segmented / stripe），默认 rounded + segmented。
- 不替换安装版；实机测试使用 `build/SuperGit.app` 和独立配置／测试仓库。

## 已完成验证

- 新设置测试先失败：保存结果缺少两个样式字段，设置 UI 缺少选项；实现后通过。
- HEAD / Stash 标签测试先失败：缺少 HEAD 标记和 `stash@{12}`；实现后通过。
- 选中节点连线回归先失败：采样像素等于背景色，证明外围光圈遮住连线；移除外围不透明填充后通过。
- Graph 子任务先以旧路径验证 R1 地标失败，三种路径实现后通过端点、镜像、缩放、15/20px 轨道、28/44px 行高、裁剪拼接和长边点数上界测试。
- 标签文本的真实布局测量与渲染边界检查通过，含长路径、中文、无分隔符长名、不同字号。
- 全部产品主题的普通／hover／selected 背景文字数值对比度检查通过（主文本、前缀、图标／HEAD 标记 ≥ 4.5:1）。
- 专项回归暴露了新增 HEAD 占用宽度和英文设置分段控件过宽；已调整预算与短文案，相应回归通过。
- `go vet ./...` 通过。
- `GOFLAGS=-tags=e2e make app` 成功，bundle 代码签名验证通过。

## 引用标签后续修正（实机反馈）

用户在实机检查后提出两点：

1. 引用标签文字偏下、未垂直居中。两次像素测量确认：`ui.RichText` 的行框比普通 `ui.Text` 高（15.31 对 12.96pt），在 20pt 胶囊中墨迹中心低约 2pt。改为与旧胶囊一致的普通 `ui.Text` 组合（前缀弱化 + 名称），新增像素回归测试，用相同高度的普通文本行作为基准比较墨迹上下边界。
2. 分支名被限制最大宽度。原实现有三处人工上限：单胶囊 150pt、metadata 行再乘 24% 列宽、标签最多 20 字符。现改为只受该行引用预算（已扣除作者和时间）约束；标签宽度随名称增长，只有预算不足时才按宽度中略。首帧尚未测到列宽时暂用 150pt 布局，下一帧改用真实预算。新增回归：206pt 的 `feature/history/reference-layout` 在 500pt 预算中完整显示；窄行拥挤场景由 `+4` 改为显示 HEAD 名称加 `+3`（受预算约束，标签不再被 150pt 上限强制折叠）。
3. 同步更新 `docs/graph-colors.md` 的胶囊宽度与中略说明，删除因此不再使用的 `rowCapsuleWidthForList`、`historyRefPillLabel` 与 `refBadgeRunes`；详情面板的 40 字符预算与渲染不变。`go vet ./...` 通过。

新增／更新的回归测试均在最终代码上通过：`TestHistoryRefLabelsSitOnThePillMidline`（与相同高度普通文本行的墨迹行比较）、`TestHistoryRefLabelUsesTheAvailableWidth`（206pt 名称在 500pt 预算内完整显示）、`TestHistoryRefMeasurementsMatchRenderedText`（前缀与末段分别查找）、窄行 `+3` 预期。实机脚本的焦点激活改为带重试的循环，避免并发负载下首次激活落在设置窗口出现之前。修正后完整 `go test ./internal/lobedesk` 通过（236.561s）。

## 完整验证命令

工作目录为隔离 worktree；Go 构建使用任务独立缓存，避免共享缓存文件消失造成干扰：

```sh
GOCACHE=/tmp/supergit-history-styles-go-cache go test -cover ./...
GOCACHE=/tmp/supergit-history-styles-go-cache go vet ./...
GOCACHE=/tmp/supergit-history-styles-go-cache make e2e-build
GOCACHE=/tmp/supergit-history-styles-go-cache GOFLAGS=-tags=e2e make app
python3 scripts/native-history-visual-styles.py --output /tmp/supergit-history-style-evidence
```

最终完整 coverage 日志：`/tmp/supergit-history-styles-coverage-final.log`。`go test -cover ./...` 退出码 0，全部包通过；`internal/lobedesk` 耗时 295.837s、覆盖率 85.9%。首次完整运行暴露的窄行 HEAD 旧预期已按新的完整 HEAD 标记预算修正，最终完整运行通过。最终 `go vet ./...` 与 `git diff --check` 均通过。

普通构建也已验证：`make app APP=build/SuperGit-ReleaseCheck.app APP_BINARY=build/supergit-release-check` 成功；与 e2e bundle 分开保存，避免覆盖隔离测试构建。本次未涉及跨包架构／底层构建，不额外运行 `make check`，也未重复运行 `make test`。

## 独立代码审查

已完成两轮；范围为 diff、新增文件与直接相关代码，审查者未重跑测试。

- 首轮发现 Stash 标签预算未计入日期／间距：补充先失败的回归测试，按实际 metadata 宽度测量，必要时移到标题行；修复后通过。
- 首轮发现原生脚本可能误启动不含配置隔离的普通构建：启动前检查 `go version -m` 的 `e2e` 标签。已实测拒绝普通二进制，错误为 `Refusing to launch a non-e2e binary`。
- 第二轮聚焦上述修复，确认两个必须修复项已解决，未发现新的必须修复项。
- 实机脚本补充原生焦点诊断：`window` 只选择自动化目标；设置窗口有键盘焦点时才排出通知队列。macOS 会拒绝脚本启动的后台应用自我激活（AppKit 调用返回成功但窗口不会成为 key window），因此保存失败场景在获得焦点时断言 `Could not save settings` 可见，否则记录 `save_warning_visual_verified: false` 并改为断言：本次运行已应用选择、被阻断路径未被写入、原设置文件字节不变、恢复可写后重新保存成功且重启恢复。提示的持续显示与手动关闭由 `TestSettingsSaveFailureRemainsVisibleUntilDismissed`（main 与 settings 两个所有者）覆盖。

## 实机结果与限制

最终运行命令：

```sh
python3 scripts/native-history-visual-styles.py --output /tmp/history-style-evidence-v8
```

退出码 0；证据根目录 `/tmp/history-style-evidence-v8`。

- `results.json`：六种组合分别在 light-wide（约 900px）、light-standard（约 600px）、dark-narrow（约 360px）验证，共 18 次选择和持久化检查；全部通过。每个场景另有保存失败条目，记录 `save_warning_visual_verified: false`（原因见下）。
- 每个场景保存 `<graph>-<refs>.png`、对应 snapshot 和 `settings-<graph>-<refs>.png`。已查看浅色标准列表 R1+A2（`origin/feature/history/reference-layout` 现完整显示，HEAD 胶囊文字居中）、深色窄列表 R4+B1（前缀按预算省略、保留末段与 `+1`），以及设置窗口截图；目标标签与路径未见明显越界或断裂。其他组合截图已生成，未逐张人工复核。
- `restart/restored.snapshot.txt`：重启后 Short curve + Compact stripe 恢复，检查通过。
- 三个场景的 `save-failure.snapshot.txt`：均确认切换已在本轮生效、被阻断路径未被写入、原设置文件字节不变；恢复可写后重新保存 Short curve，最终重启通过。由于 macOS 拒绝脚本应用自我激活，通知未在原生快照中绘制，故未做像素级断言。
- `windows-before-activation.txt` / `windows-after-activation.txt` 与 `activation.txt`：原生焦点诊断和测试进程激活记录。
- Git fixture 的 HEAD、show-ref 和 porcelain status 前后完全一致。
- `reference.html`、`reference.sha256`：本地第四轮预览副本及摘要。截图与预览保持本地，不纳入仓库。

早期脚本运行失败及处理：隐藏侧栏时不存在 Settings 按钮，改用应用已有的 `cmd+,` 快捷键；保存失败通知在脚本环境中始终不出现。逐步排查确认自动化 `window` 只选择目标、通知仅在窗口获得 OS 键盘焦点时排出，而 macOS 会拒绝脚本启动的后台应用激活（此前一次通过是应用恰好已在前台）。最终把通知断言改为有焦点时执行，同时始终验证失败路径的会话内生效、磁盘不变与恢复；通知持续显示由门面测试覆盖。未通过延长等待或跳过验证掩盖结果。

补充：修正后完整 `go test ./internal/lobedesk` 在 236.561s 通过；之后仅把宽度回归测试参数化到 1.0／1.15 字号倍率，该测试与居中测试定向通过，`go vet ./internal/lobedesk` 通过。

未覆盖／待用户检查：

- 未在本轮原生脚本自动操作引用右键菜单／`+N` 菜单／tooltip、跨仓库切换、Graph 隐藏后切换标签、恢复默认、长列表滚动和平移。相关既有测试随完整套件通过；reset 及中英文窄设置／字体缩放另有新增单元布局测试，但不等同于原生实机覆盖。
- 深色截图中少数非历史标签的工具栏图标显示异常；未证实是否为既有图标渲染或截图问题，不归为本次已修复项，也未扩大修改范围。
- 本次为已批准预览的局部渲染与设置接入，未重新执行整页设计探索／多技能 UI 流水线。

用户已在实际 App 中确认视觉效果与设置组合，本次变更已合并到 main 并推送。
