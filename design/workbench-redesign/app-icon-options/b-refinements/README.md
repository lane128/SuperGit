# SuperGit 提交主线图标

[原 B 与四款细化稿](SuperGit-B-Refinements.png) 展示浅色、深色与 48、32、24 px 的 Dock 缩放效果。预览使用项目的图标圆角和阴影。

| 方案 | 变化 |
| --- | --- |
| B1 · 平衡 | 放大顶端节点、缩小中间节点，让橙色支线以一条连续弧线分出 |
| B2 · HEAD | 用大空心环表示 HEAD，普通提交与支线末端用实心点，主线加粗以便小尺寸识别 |
| B3 · 合流 | 支线由下方分出，经一个侧边提交回到主线顶端 |
| B4 · 微浮雕 | 沿用 B1 的比例，通过同色渐变、局部高光和轻阴影增加层次 |

B1 · 平衡为 SuperGit 当前使用的图标。

每套各有浅色和深色 1024 × 1024 PNG 源图，使用 sRGB、8-bit 和不透明正方形画布。在项目根目录运行以下命令可重新生成 B1 的 Dock 与应用包资源。

```sh
make icons ICONS_DIR="$PWD/design/workbench-redesign/app-icon-options/b-refinements/B1-Refined"
make icns
```
