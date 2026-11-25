# SuperGit

[English](README.md) | 简体中文

使用 Go、[MyGo](https://github.com/egoist/mygo) 与 [Lobe MyGo](https://mygo.lobehub.com) 构建的原生 GPU 加速 Git 桌面客户端 —— 无 WebView，直接调用系统 Git。

> 当前在 **macOS（Apple Silicon）** 上构建与验证，支持中英文界面。

## 下载安装

最新版本为 [SuperGit 1.0.0](https://github.com/lane128/SuperGit/releases/latest)，适用于 macOS（Apple Silicon）。应用目前采用 ad-hoc 签名、未公证，下载后可能被 macOS 拦截；详见[打包与发布签名](docs/packaging.md)。

## 核心功能

- **多仓库工作台**：单窗口多仓库标签页，支持拖拽排序、未提交改动圆点提示、重新打开已关闭标签页（`⌘⇧T`）、重启自动恢复与本地变更自动检测。
- **提交历史图与 Diff**：真实父 SHA 提交 DAG、分支/引用筛选与搜索；统一/左右分栏 Diff，支持文件、片段（Hunk）与单行级暂存、同路径已暂存/未暂存独立标签查看与图片变更预览。
- **完整 Git 工作流**：Commit / Amend、分支、标签、远端、Stash、Worktree、Submodule、文件 Blame、Merge、Rebase、Cherry-pick、Revert 与 Reset。
- **内置冲突编辑器**：三方文本冲突对比与结果编辑（Base / Ours / Theirs / 结果），标记已解决前自动拦截残留冲突标记。
- **远端同步与安全**：Fetch、快进 Pull、Push / Force-with-lease、ahead/behind 计数与实时传输进度条。沿用系统 Git 凭据助手与 SSH Agent，不在应用内保存令牌。
- **键盘优先与可定制**：命令面板（`⌘P`）、快速检出（`⌘B`）、快速切换仓库（`⌘⇧O`）、浅色/深色主题、独立界面/代码字体设置与应用内签名更新。

## 快速开始

**环境要求：** macOS（Apple Silicon）· Go 1.27+ · Xcode Command Line Tools（CGO 与 macOS SDK）· 系统 `git`

```sh
# 构建并打开 macOS 应用包
make app
open build/SuperGit.app

# 或直接打开指定仓库
./build/supergit /path/to/repository
```

开发与验证：

```sh
make test   # 运行 Go 测试
make check  # 运行测试、竞态检测与 go vet
make demo   # 创建临时演示仓库并启动
```

## 常用快捷键

| 快捷键 | 功能 |
| --- | --- |
| `⌘P` | 命令面板 |
| `⌘B` | 快速检出分支 |
| `⌘O` / `⌘⇧O` | 打开 / 切换仓库 |
| `⌘R` | 刷新当前仓库 |
| `⌘↵` | 提交已暂存改动 |
| `⌘,` | 打开设置 |

更多文档：[完整快捷键](docs/shortcuts.md) · [打包与发布签名](docs/packaging.md)

## 许可证

代码采用 [GNU General Public License v3.0 (GPL-3.0)](LICENSE) 协议开源。
内置字体遵循 SIL Open Font License 1.1 协议，详见 [internal/lobedesk/fonts/LICENSES.txt](internal/lobedesk/fonts/LICENSES.txt)。
