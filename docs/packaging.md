# 打包与图标

本文记录 `.app` 的图标素材规则、重打包后的图标缓存验证步骤，以及发布前的检查清单。相关文件和命令都以仓库当前内容为准，改构建流程时同步更新本文。

## 图标从哪里来

应用有两张成品图标：

| 文件 | 用途 |
|---|---|
| `internal/appicon/icons/app-light.png` | 浅色外观的 Dock 图标（编译时嵌入） |
| `internal/appicon/icons/app-dark.png` | 深色外观的 Dock 图标（编译时嵌入） |
| `packaging/SuperGit.icns` | `.app` 在 Finder / 启动台里的包图标 |

运行中的 Dock 图标由共享的 `internal/appicon` 状态机切换：打包版经 `internal/desktop/dockicon.go`，Lobe 预览版经 `internal/lobedesk/dockicon.go`。两边都只在窗口外观变化时调用系统。`.icns` 只含浅色，因为 macOS 的包图标不能随外观变化，一个 bundle 只有一份。

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
- **日夜切换**：打开应用后切换明暗，Dock 图标应在下一次重绘时换成另一张；没有变化说明 `internal/appicon` 的嵌入或调用被改坏了。打包版跟随应用内的明暗开关（侧栏按钮或 `preferences.json`），系统外观变化不会改变它；只有 Lobe 预览版在「跟随系统」主题下才随 macOS 外观切换。

## 发布前检查清单

按顺序执行，全部通过才算可发布：

1. **格式与静态检查**
   ```sh
   gofmt -l internal cmd tools
   go vet ./...
   ```
   `gofmt` 必须无输出。
2. **测试**（`make check` 包含后两项，多包并行时比逐条跑快）
   ```sh
   go test ./...
   go test -race ./...
   ```
3. **构建与签名**
   ```sh
   make app
   codesign --verify --verbose=2 build/SuperGit.app
   ```
   当前是 ad-hoc 签名（`codesign --force --sign -`），分发前需要开发者签名与公证，这一步尚未接入。
4. **图标**：按上一节验证成品尺寸、`.icns` 档位与日夜 Dock 切换。
5. **e2e 冒烟**：`make native-check`，覆盖打开/历史/暂存/提交/标签/工作树/键盘导航等脚本。
6. **版本号**：手动改 `packaging/Info.plist` 的 `CFBundleShortVersionString` 与 `CFBundleVersion`；两个字段目前没有自动递增。
7. **已知边界复核**：读一遍 `README.md` 的「当前边界与后续」，确认这次改动没有让其中任何一条过期（例如签名状态、预览版能力表）。
