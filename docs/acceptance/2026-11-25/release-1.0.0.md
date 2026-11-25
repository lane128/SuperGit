# SuperGit 1.0.0 发布记录

状态：已发布。Tag `v1.0.0` 指向 `113f25b`，GitHub Release 已标记为 latest：
https://github.com/lane128/SuperGit/releases/tag/v1.0.0

## 产物

| 资产 | 说明 |
| --- | --- |
| `SuperGit-1.0.0-darwin-arm64.zip` | macOS Apple Silicon 应用包，15,368,314 字节 |
| `update.json` | Ed25519 签名的更新清单，指向 `releases/download/v1.0.0` |
| `SHA256SUMS.txt` | 压缩包校验值 |

版本 1.0.0（build 57）。构建时通过 `UPDATE_PUBKEY` 内嵌了更新公钥。

## 验证

- 干净的 `113f25b` 工作树执行 `make check`：普通测试、`-race`（`internal/lobedesk` 882.6s）、MyGo SDK 与 `go vet` 全部通过，退出码 0，无 FAIL。
- `make native-check` 与 `make desktop-native-check` 通过；其中旧桌面冒烟先修正了一条过时预期（`113f25b`，仅测试脚本）：不冲突的未跟踪文件允许随分支切换，改为验证真实覆盖冲突。
- `gofmt -l` 无输出，`git diff --check` 通过；图标成品 1024×1024，`.icns` 含 10 个档位。
- 公网下载复核：`shasum -a 256 -c SHA256SUMS.txt` 通过；解压后 `CFBundleShortVersionString` 1.0.0、`CFBundleVersion` 57；`codesign --verify --deep --strict` 通过。
- `https://github.com/lane128/SuperGit/releases/latest/download/update.json` 可解析，版本 1.0.0；`releasepack` 打包时已自校验签名。
- 本地安装版已替换为同一构建并启动。

## 密钥

发布签名私钥位于 `~/.config/supergit-release/update.key`（600），公钥同目录。私钥不在仓库内，后续版本必须沿用同一密钥签名。

## 已知边界

- ad-hoc 签名，没有 Apple Developer ID 签名与公证；下载者可能被 Gatekeeper 拦截。Ed25519 校验更新包，不等同公证。
- 仅提供 macOS Apple Silicon 构建；Intel、Windows、Linux 未发布。
- 旧本地构建未内嵌更新公钥，需要手动安装本版本一次才能验签后续更新。

## 回滚

撤销 Release 或删除远端 tag 属于破坏性操作，需单独确认后执行；代码回滚可 revert `113f25b` 与 `e118e2c`。
