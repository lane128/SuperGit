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
