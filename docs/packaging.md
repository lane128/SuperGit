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
7. **已知边界复核**。核对本页的签名与公证状态、支持平台和标准包与预览包的能力差异。结合 [Lobe 试用记录](lobe-trial-log.md) 检查本次改动是否需要更新这些说明。

## 自动更新签名与发布顺序

应用内检查更新默认读取 `https://github.com/lane128/SuperGit/releases/latest/download/update.json`，并通过内置的 Ed25519 公钥校验安装包 SHA-256 摘要签名，再结合 `codesign --verify --deep --strict` 完成 macOS 包校验与原地原子替换。

首次配置时，在受保护的执行器中使用固定版本的 `releasepack -genkey -priv <private-key-path> -pub <public-key-path>` 生成签名密钥对。私钥文件权限为 `0600`。自动交付的候选代码仅获得 `UPDATE_PUBKEY`，由受保护的执行器使用固定版本的工具打包、签名、验包和上传。`UPDATE_PRIVKEY` 和发布凭据仅保留在执行器中，禁止进入候选 checkout、其 Makefile 环境或验收与任务记录。执行器不可用时暂停并报告。

1. **确认 `main` 仍指向已记录的发布提交**。从发布记录读取 `RELEASE_TAG`（`vX.Y.Z`）和 `RELEASE_COMMIT`（完整提交 SHA）。在发布 checkout 重新获取 `origin/main`。若 `main` 已推进或 checkout 不再匹配，暂停并记录实际与预期提交，交由维护者决定。保留原先的发布选择及任何已有 tag。
   ```sh
   check_release_checkout() {
     test -n "${RELEASE_COMMIT:-}" &&
       git fetch --no-tags -- origin refs/heads/main &&
       test "$(git branch --show-current)" = main &&
       release_status=$(git status --porcelain) &&
       test -z "$release_status" &&
       test "$(git rev-parse --verify HEAD)" = "$RELEASE_COMMIT" &&
       test "$(git rev-parse --verify refs/remotes/origin/main)" = "$RELEASE_COMMIT" &&
       test "$(git rev-parse --verify FETCH_HEAD)" = "$RELEASE_COMMIT"
   }
   ```
2. **创建 annotated tag**。选定 tag 尚不存在时，在检查全部通过后显式指定已记录的发布提交。恢复任务时先重新运行检查，再核对同名 tag 是否指向记录中的提交且符合授权版本。遇到 tag 或 Release 不一致时暂停并保留现场。
   ```sh
   test -n "${RELEASE_TAG:-}" && check_release_checkout &&
     git tag -a "$RELEASE_TAG" "$RELEASE_COMMIT" -m "SuperGit $RELEASE_TAG"
   ```
3. **构建并交接受保护的打包步骤**。恢复构建或打包前，重新运行 `check_release_checkout` 并确认通过。在选定提交上运行 `make app RELEASE_VERSION="${RELEASE_TAG#v}" UPDATE_PUBKEY="$PUBKEY"`并检查产物。将构建产物、校验和、公钥、tag、提交及预期下载 URL 交给受保护的执行器。执行器验证交接记录，使用固定版本的 `releasepack -tag "$RELEASE_TAG"` 打包已交接的 `.app`，并核对应用内置公钥与签名私钥。执行器只读取产物，禁止运行候选二进制、脚本和 Makefile。仓库的组合目标 `make release-pack RELEASE_TAG=...` 会运行 `release-preflight` 并构建 `app`，仅适用于维护者已确认可信的普通发布 checkout。
   如果安装 ZIP 与 `update.json` 的过程中断，使用同一应用、签名密钥、打包参数和 `RELEASE_OUT` 重试。`releasepack` 从输出目录的 `.release-pack-pending` 恢复之前验证过的 ZIP 和清单。输入或已有文件发生冲突时，工具会报错并保留现场。旧工具留下的单独 ZIP 缺少已签名清单和恢复记录，需要先核对并保留该文件，再到独立输出目录重新打包。
4. **验证待发布资产**。执行器从固定版本的发布工具运行 `make verify-release RELEASE_OUT="$RELEASE_OUT" UPDATE_PUBKEY="$PUBKEY"`。默认产物为 `build/release/SuperGit-<version>-darwin-arm64.zip` 和 `build/release/update.json`。验包通过后，从已验证的解包应用读取 `CFBundleShortVersionString`，与 `update.json` 的 `version` 一起比对去除 `v` 的选定 tag。核对归档文件名及 `platforms.darwin-arm64.url` 与该 tag 的预期发布 URL 相符，包括 `SuperGit-<version>-darwin-arm64.zip`。任一不符便暂停。记录归档 SHA-256 与清单。
5. **发布并回下载核验**。上传前从远端读取 tag，确认 annotated tag 解析后的提交与记录中的提交相同。执行器将已验证的归档和 `update.json` 上传到该 tag 对应的 GitHub Release。从公开 URL 下载两份资产，核对归档哈希和清单是否与本地记录一致，再用固定版本的工具运行 `make verify-release RELEASE_OUT=/path/to/downloads UPDATE_PUBKEY="$PUBKEY"`。

本地端到端验收可临时指向 HTTP 目录。仅显式指定回环地址时允许本地 HTTP。

```sh
SUPERGIT_UPDATE_URL="http://127.0.0.1:8765/update.json" \
SUPERGIT_UPDATE_PUBKEY="$PUBKEY" \
open -n /path/to/test/SuperGit.app
```
