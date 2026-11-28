# 打包与图标

本文记录 `.app` 的图标素材规则、重打包后的图标缓存验证步骤，以及发布前的检查清单。相关文件和命令都以仓库当前内容为准，改构建流程时同步更新本文。

## 构建入口与当前安装版

| 目的 | 命令 | 实际入口 / 产物 |
| --- | --- | --- |
| 默认 macOS 应用包 | `make app` | `cmd/supergit` → `internal/lobedesk`，产物 `build/SuperGit.app`，包标识 `cc.supermix.supergit` |
| 旧桌面回退包 | `make desktop-app` | `cmd/supergit` + `legacydesktop`，产物 `build/SuperGit Desktop.app` |
| Finder 启动的 Lobe 预览包 | `make lobe-app` | `cmd/supergit-lobe` → `build/SuperGit Lobe Preview.app` |
| Lobe 开发二进制 | `make lobe-build` | `cmd/supergit-lobe` → `build/supergit-lobe` |
| 运行 Lobe 开发版 | `make lobe-run REPO=/path/to/repository` | 直接运行源码对应的新版界面 |
| 默认包原生验收 | `make native-check` | 以 `make app` 构建带自动化的独立测试包，覆盖日常 Git、Stash、冲突及子模块 |
| 旧桌面原生验收 | `make desktop-native-check` | 独立构建并运行旧桌面的工作流 |
| Lobe 专项原生验收 | `make lobe-e2e-build`，再运行下方 Lobe 脚本 | 预览构建与隔离 fixture |

默认工作台使用 `supergit/settings.json`。首次读取缺失选项时补入旧版主题与支持范围内的代码字号。最近仓库列表还未创建时，从 `SuperGit/preferences.json` 读取最近路径，等待用户选择后打开。旧偏好原文件保留，可由 `make desktop-app` 的回退包读取。

`make lobe-app` 用当前源码构建可从 Finder 启动的本地预览包，采用独立的应用标识和 ad-hoc 签名。运行 `open 'build/SuperGit Lobe Preview.app'` 可直接打开。先前单独安装的 `SuperGit Lobe.app` 测试包不会随源码更新。核对安装版时须同时核对打包日期、可执行入口与本次验证记录。

在 macOS 上可用 `open -a 'build/SuperGit Lobe Preview.app' '/path/to/repository'` 把文件夹交给同一个工作台。冷启动时，预览版先恢复已保存的标签，再打开收到的仓库。仓库已经打开时会切到原标签。预览包将自己声明为文件夹的备选应用。

Lobe 当前原生检查：

```sh
make check
make lobe-build
make lobe-e2e-build
python3 -I scripts/native-detail-tabs-smoke.py
python3 -I scripts/native-local-watch-smoke.py
python3 -I scripts/native-narrow-shots.py
python3 -I scripts/native-workbench-layout-smoke.py
```

各脚本顺序运行，使用临时仓库与独立偏好。验证对象、输出目录和实际试用结果见 [原生自动化说明](e2e-automation.md) 与 [Lobe 试用记录](lobe-trial-log.md)。macOS 原生通过不代表 Windows / Linux 或开发者签名、公证已经完成。

## 图标从哪里来

应用有两张成品图标：

| 文件 | 用途 |
|---|---|
| `internal/appicon/icons/app-light.png` | 浅色外观的 Dock 图标（编译时嵌入） |
| `internal/appicon/icons/app-dark.png` | 深色外观的 Dock 图标（编译时嵌入） |
| `packaging/SuperGit.icns` | `.app` 在 Finder / 启动台里的包图标 |

运行中的 Dock 图标由共享的 `internal/appicon` 状态机切换。标准包和预览包使用 `internal/lobedesk/dockicon.go`，旧桌面回退包使用 `internal/desktop/dockicon.go`。两边都只在窗口外观变化时调用系统。`.icns` 只含浅色，因为 macOS 的包图标不能随外观变化，一个 bundle 只有一份。

成品是 1024×1024 的 PNG：原始画稿是整幅方形，`tools/icongen` 会把它缩进 macOS 图标网格的 824px 圆角方块，加一圈柔和阴影。直接拿整幅方形当图标会顶满画布、硬角，和系统里其它应用不在同一网格。

## 素材入库规则

- **仓库里只放加工后的成品**：`internal/appicon/icons/app-*.png` 与 `packaging/SuperGit.icns`。原始画稿和 `*.iconset` 目录不入库，仓库里没有它们的副本，也不应该为了「留档」把整目录拷进来。
- **原始素材路径由 `ICONS_DIR` 决定**，默认 `$(HOME)/Downloads/GitApp_Icons`，里面要有 `Light.iconset/icon_512x512@2x.png` 和 `Dark.iconset/icon_512x512@2x.png`。素材放在别处时用 `make icons ICONS_DIR=/path/to/folder`。
- **换画稿的顺序固定**：先换素材，再 `make icons`（重新生成两张成品），再 `make icns`（由浅色成品生成 `.icns`），最后 `make app`。跳过 `make icons` 直接用旧成品，图标不会变。
- **`.icns` 永远从浅色成品生成**。`scripts/make-icns.sh` 只读 `app-light.png`；深色图标的落点是运行时 Dock 切换，不进包图标。
- 成品和图稿不要手工修：圆角、网格与阴影都由 `tools/icongen` 统一处理，手改会在下次 `make icons` 时被覆盖。

## 重打包后验证图标缓存

macOS 按 bundle 路径与标识缓存图标，刚替换的成品可能一时不出现。按下面的顺序检查：

```sh
# 1. 先退出正在运行的实例，否则旧进程仍显示旧图标
osascript -e 'quit app "SuperGit"'

# 2. 重建并确认签名
make app
codesign --verify --verbose=2 build/SuperGit.app

# 3. 打开 .app，分别看 Finder 与 Dock
open build/SuperGit.app
```

- **成品尺寸**：`sips -g pixelWidth -g pixelHeight internal/appicon/icons/app-light.png internal/appicon/icons/app-dark.png` 应都是 1024×1024。
- **`.icns` 档位**：`iconutil -c iconset packaging/SuperGit.icns -o /tmp/sg.iconset && ls /tmp/sg.iconset` 应列出 16/32/128/256/512 六组、共 10 个文件（含 `@2x`）。
- **Finder 缓存**：包图标没更新时，先 `touch build/SuperGit.app` 再 `killall Finder`；仍不更新时用完整路径强制重新登记（`lsregister` 不在 PATH 里）：
  ```sh
  /System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister \
    -f build/SuperGit.app
  ```
  再重开 Finder。
- **Dock 缓存**：Dock 显示的是运行中的图标，退出应用重开即可；只改 `.icns` 不影响运行时的 Dock 图标。
- **日夜切换**：打开应用后切换明暗，Dock 图标应在下一次重绘时换成另一张；没有变化说明 `internal/appicon` 的嵌入或调用被改坏了。标准包和预览包在选择「跟随系统」主题时随 macOS 外观切换。旧桌面回退包跟随应用内的明暗开关。

## 发布前检查清单

按顺序执行，全部通过才算可发布：

1. **格式检查**。运行 `gofmt -l internal cmd tools`，应无输出。
2. **测试与静态检查**。运行 `make check`，按 Makefile 的 `SOURCE_PKGS` 范围运行测试、race 检查和 `go vet`，并检查 MyGo SDK。
3. **构建与签名**
   ```sh
   make app
   codesign --verify --verbose=2 build/SuperGit.app
   ```
   当前是 ad-hoc 签名（`codesign --force --sign -`），分发前需要开发者签名与公证，这一步尚未接入。
4. **图标**：按上一节验证成品尺寸、`.icns` 档位与日夜 Dock 切换。
5. **e2e 冒烟**。运行 `make native-check` 验证标准包和 `make desktop-native-check` 验证旧桌面回退入口。系统目录选择框的选择和取消，以及重启后的系统「最近打开」菜单需在 macOS 图形会话中人工核对。
6. **版本号**：`make app` 调用 `tools/appversion`，只改写 bundle 内的 plist 副本。`CFBundleShortVersionString` 取最近标签，去掉前缀 `v` 后仅接受 1–3 段数字；`CFBundleVersion` 取 `git rev-list --count HEAD`。标签 / 提交数不可用或格式非法时，对应字段沿用 `packaging/Info.plist` 的值；源码 plist 不会被自动修改。相同 HEAD 的未提交构建可能拥有同样的包版本，试用记录还需标明源码日期和未提交状态。

## 自动更新签名与发布顺序

应用内检查更新默认读取 `https://github.com/lane128/SuperGit/releases/latest/download/update.json`，并通过内置的 Ed25519 公钥校验安装包 SHA-256 摘要签名，再结合 `codesign --verify --deep --strict` 完成 macOS 包校验与原地原子替换。

首次配置时，生成签名密钥对。私钥保存在受保护的本机路径，权限为 `0600`，切勿提交到仓库。

```sh
mkdir -p ~/.config/supergit-release
go run ./tools/releasepack -genkey -priv ~/.config/supergit-release/update.key -pub ~/.config/supergit-release/update.pub
```

1. **确认 `main` 干净且已同步**。合入已验证的改动后，在 `main` checkout 获取远端状态。以下检查必须全部通过。
   ```sh
   git fetch origin main &&
     test "$(git branch --show-current)" = main &&
     test -z "$(git status --porcelain)" &&
     test "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)"
   ```
2. **创建 annotated tag**。使用已授权的 `vX.Y.Z` 版本。以下版本仅为示例。恢复任务时，同名 tag 指向当前 `HEAD` 且符合授权版本才可复用。遇到 tag 或 Release 不一致时停止并保留现场。
   ```sh
   RELEASE_TAG=v1.2.3
   git tag -a "$RELEASE_TAG" -m "SuperGit $RELEASE_TAG"
   ```
3. **打包签名更新**。`make release-pack RELEASE_TAG=...` 先运行 `release-preflight`。预检重新获取 `origin/main`，检查分支、工作树、tag 指向和签名密钥是否匹配，再构建内置公钥的应用并生成归档与清单。
   ```sh
   PUBKEY=$(cat ~/.config/supergit-release/update.pub)
   make release-pack \
     RELEASE_TAG="$RELEASE_TAG" \
     UPDATE_PUBKEY="$PUBKEY" \
     UPDATE_PRIVKEY=~/.config/supergit-release/update.key \
     RELEASE_NOTES_ZH="本次更新内容" \
     RELEASE_NOTES_EN="Release notes"
   ```
4. **验证待发布资产**。使用与打包相同的公钥及输出目录。默认产物为 `build/release/SuperGit-<version>-darwin-arm64.zip` 和 `build/release/update.json`。验包通过后记录归档 SHA-256 与清单。
   ```sh
   make verify-release UPDATE_PUBKEY="$PUBKEY"
   ```
5. **发布并回下载核验**。将已验证的归档与 `update.json` 一起上传到该 tag 对应的 GitHub Release。从公开 URL 下载两份资产，核对归档哈希和清单是否与本地记录一致，再用相同公钥运行 `make verify-release RELEASE_OUT=/path/to/downloads UPDATE_PUBKEY="$PUBKEY"`。

本地端到端验收可临时指向 HTTP 目录。仅显式指定回环地址时允许本地 HTTP。

```sh
SUPERGIT_UPDATE_URL="http://127.0.0.1:8765/update.json" \
SUPERGIT_UPDATE_PUBKEY="$PUBKEY" \
open -n /path/to/test/SuperGit.app
```
