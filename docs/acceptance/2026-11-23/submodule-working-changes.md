# 子模块工作区改动 — 原生验收

**原生验收通过。** 仅在子模块内部编辑文件时，父仓库的 Files 和改动总数不会多出一条子模块路径，子模块侧栏显示自己的文件数。实际 gitlink 更新仍可暂存。

## 构建与环境

- macOS 27.2.1 / Apple Silicon，Go 1.27.3。基准 `0fedcfe` 加本次未提交改动。
- `make app e2e-build` 生成 `build/SuperGit.app` 0.1.0 (38)，`codesign --verify --deep --strict` 通过。独立启动标准 bundle，确认打开可见原生窗口；用户配置和网络访问与验收隔离。
- 同一工作树的 e2e bundle 由 `build/mygo-automation` 驱动，Git 操作结果由独立 `git` 检查。验证了浅色 1320×840、深色 980×680（115% 字体，侧栏最窄宽度）。

## 验收结果

| 场景 | 结果 |
| --- | --- |
| 父文件修改，子模块 HEAD 未动、内部一处修改和一个新文件 | 父仓库显示 1 项改动，只列父文件；侧栏分支旁显示 `2`。Stage All 只暂存父文件，不改变子模块 Index。 |
| 父仓库 porcelain 不变时再改一个子模块文件 | 徽标 `2 → 3`；进入子模块看到 3 项。减少一项后切回父仓库显示 `2`；清理后徽标消失。 |
| 子模块 HEAD 移动且存在本地文件改动 | 父仓库 Modified Files 出现一条 gitlink。暂存后只出现在 Staged Files。侧栏仍显示 detached ref 和 `1`，子模块本地内容不变。 |
| 三层子模块与嵌套 `ignore=all` | 最内层文件改动只显示在它自己的模块行。嵌套 gitlink 移动后，即使最外层 porcelain 不变，最外层的中间模块徽标也会更新。 |
| 窄侧栏及辅助功能 | 徽标可见，长分支或短 hash 在宽度不足时可截断；辅助功能描述保留完整 ref 和文件数。 |

## 自动化验证

- `go test -cover ./...` 在首轮审查修复后通过，`internal/lobedesk` 85.5%，`internal/gitrepo` 80.1%。最终源码通过 `go vet ./...` 和 `internal/lobedesk` 的定向竞态测试，覆盖子模块计数、轮询、侧栏布局及丢弃前校验。
- 上一轮 `make check` 通过 MyGo 快照检查和全量普通测试，但其竞态阶段编译的是最后几处侧栏布局修正前的源码，`TestSubmoduleCountLeavesRoomForThePathTail` 因路径和分支标签宽度不足失败，日志未报告数据竞态。最终源码下该用例和大字体布局用例的定向竞态测试通过；原流程未执行到的 MyGo 竞态和 Go vet 步骤已单独通过。最终源码的 `make check` 重跑通过（退出码 0），包括 MyGo 快照检查、全量普通测试、SuperGit 竞态测试、MyGo 竞态测试（按 Makefile 排除两个精确分配计数用例）和 `go vet`。
- 回归测试覆盖父仓库状态不变、多版本同一路径计数、未跟踪文件变化保持父仓库选择、重复子模块行、子模块读取错误恢复、单个探测超时、父仓库状态读取失败和丢弃前校验。
- `scripts/native-lobe-submodule-smoke.py` 使用本次 e2e 二进制通过，验证初始化、按父仓库 Index 还原子模块 HEAD、拒绝覆盖子模块未提交的文件，以及打开子模块时保留各自仓库的数据。证据在 `build/lobe-submodule-x09mc6kh/`。
- 使用本地测试仓库。未修改、读取或安装用户的业务仓库和已安装应用；未使用远端网络。

## 证据

截图和辅助功能树位于本地构建目录 `build/submodule-working-yy_1059o/`（Git 忽略）：`light-initial.png`、`dark-initial.png`、`light-staged.png`、`dark-staged.png` 以及对应 `*.snapshot.txt`。驱动为临时文件 `/tmp/supergit-submodule-working-acceptance.py`。手动复现步骤见 [设计文档](../../features/submodule-working-changes.md#原生验收)。
