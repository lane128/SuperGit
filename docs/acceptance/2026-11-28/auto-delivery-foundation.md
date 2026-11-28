# 自动交付基础验收

日期：2026-11-28。源码提交：`7f76c09`。技能、规则与本记录随其后的文档提交入库。

## 检查命令与结果

| 命令 | 结果 |
| --- | --- |
| `rg -n './\.\.\.' Makefile .claude/skills/feature-flow/SKILL.md CLAUDE.md` | 通过。匹配项为 Makefile 的 `SOURCE_PKGS` 定义和两处对应的源码包列表。 |
| `rg -n 'auto-delivery\|delivery status\|release-preflight\|verify-release' CLAUDE.md .claude/skills docs/packaging.md` | 通过。四个入口均有引用。 |
| `gofmt -l internal/delivery tools/delivery tools/releasepack && go vet ./internal/delivery ./tools/delivery ./tools/releasepack && go test ./internal/delivery ./tools/delivery ./tools/releasepack -count=1` | 通过。`gofmt` 和 `go vet` 无输出，三个包测试通过。 |
| `python3 -I scripts/check-delivery-dry-run.py` | 通过。退出码为 `0`。 |

定向测试输出：

```text
ok  	supergit/internal/delivery	70.980s
ok  	supergit/tools/delivery	1.849s
ok  	supergit/tools/releasepack	0.404s
```

## 干跑输出

```text
PASS branch
PASS clean
PASS synced
PASS tag
PASS public_key
PASS private_key
Dirty worktree rejected by preflight.
Task A holds the window and task B is blocked.
Task A completed and task B acquired the window.
Repeated status and advance left state unchanged. The fixture still has only v9.9.9.
PASS delivery dry-run
```

## 覆盖范围

- 静态检查核对了源码包范围和交付入口。定向测试覆盖交付状态、CLI 与发布工具。
- 干跑使用临时 Git 仓库、仅本地的裸远端、隔离账本和临时签名密钥，验证发布预检、阶段推进、发布窗口互斥及任务完成后释放窗口。重复读取状态和无效的重复推进均未改变账本。临时仓库预置的测试 tag `v9.9.9` 保持不变，发布产物目录为空。
- 本阶段未在 SuperGit 仓库创建 tag 或 Release，未运行真实 Agent 编排、创建真实 GitHub PR 或进行真实发布。原生应用启动、正式打包及公开资产回下载均未执行。

## 第 8 项审查修复复验

修复基线为 `e6875ee`。技能和打包说明更新了受保护的发布交接、合并前归属核对、版本选择恢复、远端 tag 核对和验包后的版本与 URL 校验。两名只读 Agent 的配置增加了与设计审查一致的颜色。Go 阶段门、CLI 流程测试、队列测试及干跑脚本均覆盖 `code_review` 阶段复验。

### 回归测试

RED 在修复前运行 `go test ./internal/delivery -run '^TestCodeReviewRequiresCurrentTestsRecordedDuringReview$' -count=1 -v`，退出码 `1`。四个场景均错误地通过了向 `acceptance` 的推进：

```text
advance to acceptance with missing tests = <nil>, want tests gate error
advance to acceptance with implementation phase tests = <nil>, want tests gate error
advance to acceptance with previous candidate tests = <nil>, want tests gate error
advance to acceptance with failed rerun tests = <nil>, want tests gate error
```

GREEN 在修复后运行 `go test ./internal/delivery ./tools/delivery -run '^(TestCodeReviewRequiresCurrentTestsRecordedDuringReview|TestAdvanceRequiresPassedCheckForCurrentCandidate|TestRunTaskFlowPersistsAcrossInvocations)$' -count=1 -v`，退出码 `0`。四个无效场景均被拒绝，随后记录有效测试可推进。CLI 流程测试确认失败的推进未改写账本。

```text
--- PASS: TestAdvanceRequiresPassedCheckForCurrentCandidate (0.00s)
--- PASS: TestCodeReviewRequiresCurrentTestsRecordedDuringReview (0.00s)
    --- PASS: TestCodeReviewRequiresCurrentTestsRecordedDuringReview/missing (0.00s)
    --- PASS: TestCodeReviewRequiresCurrentTestsRecordedDuringReview/implementation_phase (0.00s)
    --- PASS: TestCodeReviewRequiresCurrentTestsRecordedDuringReview/previous_candidate (0.00s)
    --- PASS: TestCodeReviewRequiresCurrentTestsRecordedDuringReview/failed_rerun (0.00s)
--- PASS: TestRunTaskFlowPersistsAcrossInvocations (0.02s)
```

### 重跑检查

上表两个 `rg` 命令均退出 `0`。第一个在 `Makefile:2`、`CLAUDE.md:36` 和 `.claude/skills/feature-flow/SKILL.md:53` 匹配到 `SOURCE_PKGS` 范围。第二个在项目规则、两个技能及打包说明中覆盖 `auto-delivery`、`delivery status`、`release-preflight` 和 `verify-release` 四项引用。

重跑上表的完整定向检查，`gofmt` 和 `go vet` 无输出，退出码 `0`。

```text
ok  	supergit/internal/delivery	11.243s
ok  	supergit/tools/delivery	1.718s
ok  	supergit/tools/releasepack	0.361s
```

重跑 `python3 -I scripts/check-delivery-dry-run.py`，退出码 `0`。

```text
PASS branch
PASS clean
PASS synced
PASS tag
PASS public_key
PASS private_key
Dirty worktree rejected by preflight.
Task A holds the window and task B is blocked.
Code review rejected the revised candidate until its tests were recorded.
Task A completed and task B acquired the window.
Repeated status and advance left state unchanged. The fixture still has only v9.9.9.
PASS delivery dry-run
```

发布交接、归属和资产校验规则经过文档静态核对。受保护的执行器、真实 Agent 编排、网络 PR、合并及发布未在本阶段实跑。SuperGit 仓库未创建 tag 或 Release。

## 第 2 轮审查修复复验

修复基线为 `8c3414c`。技能要求每次进入 `tag` 及恢复构建或打包前重新 fetch，验证 `HEAD`、`origin/main` 和 `FETCH_HEAD` 都等于 `release_selection` 中的提交。发现推进或不匹配时记录预期和观察到的提交并暂停。打包说明的 tag 命令显式指定已记录的提交，并由同一检查控制执行。

重跑上表两个 `rg` 命令，均退出 `0`。源码范围仍使用 `SOURCE_PKGS`，交付和验包入口均有引用。`git diff --check` 通过。

从打包说明提取实际 shell 片段，在替代 Git 命令的测试环境中核对提交比较和 tag 参数。该检查没有创建任何 tag。

```text
PASS synchronized checkout passes and supplies the recorded commit to the tag command
PASS resumed checkout blocks when only FETCH_HEAD reports advanced main
PASS newer synchronized checkout blocks because the recorded commit differs
```

重跑上表的完整定向检查，`gofmt` 和 `go vet` 无输出，退出码 `0`。

```text
ok  	supergit/internal/delivery	11.001s
ok  	supergit/tools/delivery	3.455s
ok  	supergit/tools/releasepack	0.441s
```

重跑 `python3 -I scripts/check-delivery-dry-run.py`，退出码 `0`。

```text
PASS branch
PASS clean
PASS synced
PASS tag
PASS public_key
PASS private_key
Dirty worktree rejected by preflight.
Task A holds the window and task B is blocked.
Code review rejected the revised candidate until its tests were recorded.
Task A completed and task B acquired the window.
Repeated status and advance left state unchanged. The fixture still has only v9.9.9.
PASS delivery dry-run
```

本轮只修改文档。未执行真实 Agent 编排、打包或发布，未对 SuperGit 仓库创建 tag、推送、合并或创建 Release。

## 最终审查遗留问题复验

修复基线：`125e37f`；验收对象为其上的未提交补丁。沿用已批准的 T3 交付设计。本轮不改界面，未做 UI 原型、主题或尺寸验收。

| 原遗留项 | 回归验证 |
| --- | --- |
| 同时间戳陈旧写 | `Store.Save` 拒绝覆盖同一时刻已持久化的撤销记录。 |
| 暂停来源混淆 | 人工录入与撤销相同的原因文本，重新授权仍保留人工暂停。旧账本按说明显式恢复。 |
| plist 空白被忽略 | 带填充空白的版本、构建号不参与规范化版本比较；可执行文件按字面名称查找。 |
| 合并前窗口无法恢复 | 撤销后的 `pull_request` 持有者经维护者记录释放依据后让出窗口，原任务保持暂停和撤销状态。 |
| 标签阶段版本修订 | 进入 `tag` 后拒绝不同版本或策略，原选择保留；此前改版使旧门禁失效，延迟写不能复用旧版本计划。演练使用不同的评审候选 SHA 和合并 SHA。 |
| 打包中断留下单独 ZIP | 相同输入补齐已验证的原清单；内容或说明变化、恢复目录含额外文件均拒绝并保留原文件。完整产物无需重新安装。 |
| 缺目录模式的 ZIP 条目 | 接受空目录标记并补足可用权限；拒绝携带数据的目录标记。 |

最终四个受影响包的 `go test -race ... -count=1`、全源码和 SDK 的 `go vet github.com/egoist/mygo/ui ./cmd/... ./internal/... ./tools/...`、`git diff --check` 及 `python3 -I scripts/check-delivery-dry-run.py` 均通过。独立审查和两次范围复审在验收前完成；最后两条关于原子读取授权和旧账本恢复的发现，通过状态接口回归与恢复文档补齐。

原生验收使用隔离目录、临时测试密钥和本地版本 `99.0.0`：`make app` 构建真实 ad-hoc 签名应用；`releasepack` 在模拟中断后恢复字节完全相同的 ZIP / `update.json`；完整产物再次运行不改动。`delivery verify-release` 通过 manifest、size、SHA-256、Ed25519、ZIP 安全、解压和 macOS codesign 七项检查。验包解出的应用对隔离 Git 仓库启动，关闭自动更新；仅检查启动，无窗口交互断言。

未使用生产密钥或发布执行器，未提交、推送、合并、创建真实标签、创建 GitHub Release 或做公开回下载。已存在且无恢复记录的单独 ZIP 保留现场，不会自动覆盖。
