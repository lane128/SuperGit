# 通过 Homebrew 安装 SuperGit

状态：已发布，公开 tap 安装验收通过（2026-11-26）。详见[验收记录](../acceptance/2026-11-26/homebrew-cask.md)。

日期：2026-11-26

## 背景与目标

为 Apple Silicon 用户提供稳定安装命令 `brew install --cask lane128/tap/supergit`，复用 v1.0.0 已发布的 ZIP。

## 交付范围与方案

建议新建公开 GitHub 仓库 `lane128/homebrew-tap`，由该仓库维护 `Casks/supergit.rb`。SuperGit 主仓库的 README 增加安装步骤。仅涉及分发，无界面变动。此次使用已有 Release，无需重新构建发布包或使用更新签名私钥。

Cask 固定版本、带版本的 Release URL 和 ZIP 的 SHA256，声明 Apple Silicon、macOS 13 Ventura 及以上（正式包 plist 标注 12.0，Mach-O 的 `LC_BUILD_VERSION` 实测要求 13.0），安装 `SuperGit.app`。v1.0.0 的 Release 资产摘要与 `SHA256SUMS.txt` 均为 `2e832999b6a3ef3f3280ae05012adc62678e8a0ef99789bbaf10b1177a7c544e`。发布者后续在发布新版后手动更新版本与摘要。应用内更新保留，Cask 标记 `auto_updates true`，文档说明需要时用 `brew upgrade --cask --greedy supergit`。

## 方案取舍与兼容性

独立 tap 可直接由项目维护者发布，无需等待官方 cask 收录。安装说明涵盖首次使用第三方 tap 时当前 Homebrew 要求的信任步骤；如已有同名应用，保留用户安装并说明冲突。沿用系统隔离与 Gatekeeper 行为，说明当前 Release 的 ad-hoc 签名状态。Intel 没有对应发布包。

## 测试与验收

1. 比对正式 ZIP 的散列与发布清单，读取 bundle ID、版本和 Mach-O 最低系统版本，对齐 Cask 元数据。
2. 检查 Ruby 语法，运行 Homebrew 的 Cask 样式、审计、信息读取和 fetch 校验。验证架构限制及 URL 展开结果，记录工具环境限制。
3. 独立审查变更后，在可丢弃的 Apple Silicon macOS 测试环境使用本地 tap 安装。检查签名、应用路径和版本，按安装路径启动并打开测试仓库。若 Gatekeeper 提示，可按系统设置中“隐私与安全性 → 仍要打开”的正常流程批准；仍无法启动则验收失败。卸载后确认 bundle 与安装记录均已清除。另在测试环境预放手动安装副本，确认默认安装报告冲突，原副本摘要保持一致。按 T3 流程运行项目测试及 `make app` 冒烟，记录到 `docs/acceptance/<验收日期>/`。
4. 用户验收通过后发布 tap。在无本地 tap、信任条目、Cask 记录及下载缓存的干净测试环境，按 README 首次信任并从公开远端安装，验证 ZIP 摘要、默认目录中的 v1.0.0、实际启动路径以及卸载结果。远端安装通过后再发布主仓库 README。

## 风险与回滚

上游资产重新上传会导致摘要不匹配，安装应停止并由维护者复核。安装与卸载验收使用独立测试环境，保留开发机上的应用、用户配置与 Homebrew 安装记录。回滚分发入口时 revert tap 和 README 的对应提交；用户可用 `brew uninstall --cask supergit` 卸载 brew 安装的副本。
