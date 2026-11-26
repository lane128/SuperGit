# Homebrew Cask 安装验收

2026-11-26，Apple Silicon，macOS 27，Homebrew 7.0.16。候选 tap 为 `lane128/tap`，安装包为 GitHub Release v1.0.0。本地测试使用主机上的独立 Homebrew 前缀、Caskroom、应用目录、HOME、信任配置和下载缓存，未使用虚拟机。测试副本位于 `/private/tmp/supergit-brew.Cww7ZfjA/apps/SuperGit.app`。

| 检查 | 结果 |
| --- | --- |
| Ruby 语法、`brew style --cask`、`git diff --check`、`go vet ./...` | 通过。样式检查 1 个文件，0 个问题。 |
| 全量 Go 测试 | `go test -cover ./...` 退出码 0。 |
| 元数据和信任 | `brew trust --cask lane128/tap/supergit` 成功。`brew info --json=v2 --cask` 返回 macOS >=13、arm64、auto_updates、版本 1.0.0 和 `SuperGit.app`。 |
| 下载 | `brew fetch --cask` 成功。独立计算下载 ZIP 的 SHA256 为 `2e832999b6a3ef3f3280ae05012adc62678e8a0ef99789bbaf10b1177a7c544e`，与发布清单一致。 |
| 安装和卸载 | 安装到临时 appdir，Info.plist 版本 1.0.0，`codesign --verify --deep --strict` 成功。卸载后 app 和 Caskroom 安装记录均已清除。 |
| 同名应用 | 在另一临时 appdir 预放应用，默认安装报告 `already an App`。逐文件 SHA256 一致，无残留 Caskroom 记录。 |
| 本地源码冒烟 | `make app` 成功，Launch Services 启动本地 bundle，进程参数指向隔离 Git 仓库。 |
| 正式包首次启动 | 重新安装后，`open` 返回 -128，`spctl --assess --type execute` 返回 `rejected`。应用带有 Homebrew quarantine。用户在系统界面批准后，再次通过 Launch Services 按安装路径启动成功，确认该可执行文件的进程参数含 `test-repo`，窗口标题为 `SuperGit · test-repo`。 |
| `brew audit --cask` | 审计前置检查失败，独立安装的 Command Line Tools 的最新 SDK 为 26.2，Homebrew 要求与 macOS 27 兼容；完整 Xcode 为 27.2。 |
| 公开 tap 首次安装 | 在第二个独立 Homebrew 前缀中从空缓存、空信任记录开始，先信任单个 Cask，再执行 `brew install --cask lane128/tap/supergit`。Homebrew 自动克隆公开仓库 `https://github.com/lane128/homebrew-tap`（`ed5c43a`），下载与上表相同摘要的 v1.0.0。为保留现有应用，验收将 appdir 指向临时目录。签名校验通过，用户批准首次启动后核对窗口标题为 `SuperGit · test-repo`。卸载后，测试应用和 Caskroom 记录均已清除。 |
