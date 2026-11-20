# 快速输入异常与测试清理问题复盘

## 快速按键的根因与修复

MyGo 的窗口快捷键回调在下一次 view build 时执行，而文本事件会先投递给上一帧仍有焦点的编辑器。因此，CmdP 与后续文字在刷新前连续到达时，正文会收到文字，命令查询却为空。这不是查询匹配器的问题。

确定性复现直接向同一个真实 SDK 引擎连续投递事件，不在每个按键后调用 Tester 的自动 settle。原版本在“打开、输入 crbr、Enter、Backspace、Escape、输入 X”的无中间显示帧序列后，得到 `bodycrbrX`，查询为空。两种快捷键都存在该问题。

兼容补丁为可能改变输入焦点的离散操作建立输入边界。在下一次键盘、文本、组合输入、编辑命令或指针按下到达前，先完成待处理的 view build，再按新焦点投递事件。已完成的 build 清除边界；纯文本连续输入不会逐字强制绘制。已绘制菜单保留原有整批按键事务，防止 Enter 关闭菜单后的 Down 被误当成重新打开菜单。

原固定 SDK 版本不变：`v0.8.4-0.20261108082913-9233663736ca`。项目以 `go.mod replace` 使用 `third_party/mygo` 兼容副本；没有修改共享模块缓存。保留范围相对原模块只有 `ui/input.go`、`ui/runtime.go` 两项运行时差异，以及新增 `ui/input_barrier_test.go`。小补丁见 `mygo-input-order.patch`，原许可证保留。独立审查逐文件与缓存原模块比对，确认声明保留范围无遗漏。

`SUPERGIT_UPSTREAM.json` 记录 1969 个文件的指纹，`make check` 会校验文件集合与内容，并运行 SDK UI 包及应用全部普通、race、vet 检查。回归同时验证完整无帧间隔往返和普通文本仍批量绘制。

## 两项 Undo 测试的根因与修复

同一个测试二进制在正常环境通过，在受限沙箱失败。补充失败诊断后发现：`/usr/bin/git` 是 Xcode 启动代理，可能往测试指定的 TMPDIR 写入 `xcrun_db`，并输出 `DVTFilePathFSEvents` / `DARWIN_USER_CACHE_DIR` 警告。前者被误认为业务 scratch 泄漏；后者混入测试 helper 的 CombinedOutput，被误认为工作区 status 非空。原文件字节和实际 Git 状态没有改变。

`useDirectGitForTempAssertions` 在改变 TMPDIR 前通过 xcrun 找到真正 Developer Git，仅当 PATH 原本指向 `/usr/bin/git` 时在这两个测试中使用它；自定义 Git 工具链保持原样。PATH 由 `t.Setenv` 恢复。不放宽目录条目数量、清理、文件内容或工作区状态断言，保留具体错误和首次 status 诊断。修复后的同一测试二进制在沙箱内三轮通过。

## 完整检查发现的加载 fixture 生命周期缺口

第一次完整重跑通过了上述两项测试，但 `TestOpeningRepositoryRunsStatusOnce` 的临时 shim 目录清理与后台详情读取竞争。helper 原先使用 `context.Background()`，仅等 snapshot.loading 结束；这个回调会继续启动 files、diff 和独立签名读取。

修复只涉及测试 fixture：拥有可取消上下文，退出时取消并释放历史遍历；停止后不向遗弃的主线程队列阻塞；返回前等待详情和独立签名读取。该 fixture 是合法未签名提交，真实签名回调的状态 N 是进程退出后的完成证据，空队列不是完成证据。

增加慢签名回归。在旧 helper 下可观察到 loading 已结束但 files/diff 仍读取；临时 overlay 去掉签名等待时，回归仍会以签名状态为空失败。最终两项加载测试十轮 race 通过（10.939s）。读取失败不会被当成已完成，会使该 fixture 明确失败。没有为业务代码增加测试钩子。

## 复验结果

- SDK 原行为 RED：`build/followup/sdk-input-red.log`；完整 SDK UI GREEN：`build/followup/sdk-ui-full.log`。
- Undo 沙箱诊断与三轮 GREEN：本次审查 worktree 的 `build/followup/undo-diagnostics.log`、`undo-first-status.log`、`undo-green-sandbox.log`。
- 加载 helper 两次 RED 证据及十轮 race GREEN：`load-lifetime-red.log`、`load-signature-red.log`、`load-lifetime-green.log`。
- 真实窗口 Computer Use：原异常序列一次、随后十轮交替 CmdP/CmdShiftP 往返，加五轮快速粘贴 Checkout 往返，共十六轮，正文精确为 `body` + 11 个 X + 5 个 Z，Summary 精确保留，无多字或丢字。顶部按钮、方向键跳过禁用项、无结果 Enter、背景 Cmd4 隔离、中文与 emoji 粘贴均复验。
- 通过面板在 owned fixture 实际创建并切换 `acceptance/rapid-input`，再用 Checkout 回 main。HEAD 和文件不变，Index 与工作区干净；refs 仅新增预期验收分支。原生包 SHA256：`3ac2c330368561951fec129a85537fd802f3525b78edbece79013778df4334a6`。
- Windows/Linux amd64 CGO-disabled 编译通过；不是这两个系统的原生运行验收。
- 原有显示恢复候选检查入口已适配固定兼容副本并实际通过，保留其原 RED/GREEN 验证。
- 最终 `make check` 退出码 0，完整日志 `build/followup/full-check-accepted.log`。普通 SDK 整包及应用全部普通测试通过；应用全部 race 通过（最近非缓存 lobedesk 713.963s、gitrepo 59.628s）；SDK race 通过（最终 162.075s）；SDK 与应用 vet 通过。最后汇总复用同一源码指纹对应的成功缓存，2388 项源码指纹再次核对一致。
- SDK 的 `TestDiffFrameAllocs` / `TestPathContains` 含精确分配次数断言，在本机 race 插桩下会失败；未修改的上游 SDK 同样复现（帧 3/4 次，Contains 3 次），并非输入补丁引入。最终入口在普通 SDK 整包保留这两项严格检查，在 SDK race 阶段仅排除这两个测试；应用全仓 race 不排除任何测试。其余 SDK race 已全部通过。基线证据 `sdk-upstream-race-frame-alloc.log` / `sdk-upstream-race-alloc.log`。
- 快照校验通过，并用临时小型 fixture 反向验证篡改文件或新增文件均被拒绝。使用真实 Developer Git 的临时 PATH 仅影响验证进程，不更改用户设置。
- 独立只读审查共三轮：第二轮指出独立签名读取等待缺口，补回归修复后第三轮确认覆盖；无未处理重要问题。

实现仍在 `/Users/adam/SuperGit`。无 commit、安装或发布，没有操作用户真实仓库的数据。
