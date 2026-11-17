# SuperGit App 图标方案

[浅色、深色与 Dock 尺寸对比](SuperGit-Icon-Comparison.png) 包含现版与三套候选在 48 px、32 px 下的效果。预览使用项目现有的 macOS 圆角遮罩和阴影。

[B 提交主线的四款细化](b-refinements/README.md) 包含原 B 对照、浅深色源图和 48、32、24 px 缩放预览。

| 方案 | 特点 |
| --- | --- |
| A · 分叉 | 延续现版的三节点轮廓，用蓝色分叉和橙色末端减少小尺寸下的视觉干扰 |
| B · 提交主线 | 以蓝色主线和橙色支线呈现提交历史，呼应工作台线图，推荐作为默认候选 |
| C · S 形分支 | 把三个节点融入 SuperGit 首字母，强调品牌识别 |

每套目录分别包含 `Light.iconset/icon_512x512@2x.png` 和 `Dark.iconset/icon_512x512@2x.png`。源图是 1024 × 1024、sRGB、8-bit 的不透明正方形 PNG。圆角、透明边距与阴影在打包时生成。

选定后，在仓库根目录将命令中的 `B-Commit` 换成所选方案，再生成 Dock 资源及应用包图标。

```sh
make icons ICONS_DIR="$PWD/design/workbench-redesign/app-icon-options/B-Commit"
make icns
```
