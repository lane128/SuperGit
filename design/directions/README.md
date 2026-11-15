# 三套 UI 方向稿

用同一份示例数据（`atlas` 仓库、`feature/partial-stage` 分支、12 条提交、混合暂存状态）画了三套方向，每套包含「工作区」和「历史」两页。目的是先比较信息层级、密度、字形和 Git 状态表达，再选一套进入 MyGo 实现。

先看 `overview.png`（三套工作区 + 历史的总览），再看各自 1:1 的 PNG。

## A. 原生 macOS 工艺

文件：`A-workspace.png`、`A-history.png`

- 真实窗口层级：38px 标题栏 + 彩色交通灯 + 44px 工具栏，不做网页式顶栏。
- SF 字体，28px 行，名称优先；路径在名称后吸收截断，按完整路径段收缩成 `…/desktop/`。
- 分支切换、ahead/behind、Fetch / Pull / Push 常驻，符合桌面 Git 客户端的肌肉记忆。
- 历史使用圆角线图；提交详情固定在工作区下方，引用列独立。
- 适合：想要「像 macOS 原生应用」、第一眼没有学习成本。

## B. 技术编辑风

文件：`B-workspace.png`、`B-history.png`

- Avenir Next + New York 衬线标题 + Menlo，顶部全局模式栏 + 左侧 56px 图标轨道。
- 账本式文件行：左侧色脊表达状态，名称、路径、类型、状态、行内操作各占固定列。
- 正交线图，引用紧跟在提交说明后；右侧检查器承载完整提交详情和操作。
- 适合：想和 Sublime Merge / GitHub 拉开差异，同时保留高信息密度。

## C. 密集仪表

文件：`C-workspace.png`、`C-history.png`

- 深色，JetBrains Mono + SF，32px 顶栏 + 44px 轨道 + 250 / 466 / 280 三栏。
- 24px 行，文件、Diff、提交三栏并置；45° 线图 + 底部轨道图例。
- 右侧同时承载提交、Git 状态、同步、危险操作；底部状态条常驻 HEAD、ahead/behind、Index 计数。
- 适合：长时间盯屏、以键盘和密度为先的高级用户。

## C2. C 的收敛迭代

以 C 为基础，去掉页面标题栏、左侧状态表和右侧重复的 Git/同步/危险操作面板；文件详情归左列，提交归右列，Diff 历史和设置归标签页。部分暂存、命令面板、危险确认、子模块更新和撤销另有细节页。

文件：`c2/C2-workspace.png`、`c2/C2-history.png`、`c2/C2-details.png`；说明见 `c2/README.md`。

日光模式与中英文：`c2/C2L-workspace-zh.png`、`c2/C2L-workspace-en.png`、`c2/C2L-history-zh.png`、`c2/C2L-history-en.png`。

## 三个方向共同遵守的部分

- 复用 `design/compact/lucide-icons.json` 里的 Lucide 0.468.0 图标，不新引入图标集。
- 文件名优先，路径、类型、状态随后；状态不只靠颜色，另有字母或文字。
- 子模块与普通仓库使用同一套行、同一套 Diff、同一套操作，不单独做「模块页」。
- 行内操作保持克制：暂存 / 取消 / 丢弃只在当前上下文出现。

## 复现

```bash
cd design/directions
node generate.mjs   # 写出 6 个 SVG + overview.svg
node render.mjs     # 写出 6 个 PNG + overview.png + check.json
```

`render.mjs` 使用本机 Chrome 和 `/private/tmp/ui-browser.yhx7Lf` 下的 Playwright；该路径不存在时只能生成 SVG。

当前检查结果（`check.json`）：

```
Rendered 6 frames + overview.
Overflow 0; collisions 0; missing labels 0; truncated 0; errors 0.
```

检查覆盖：SVG 解析错误、文本越界、文本与文本碰撞、期望标签可见性、`<title>` 截断标记。碰撞检测只比较文本节点，图标与矩形之间的压盖靠 1:1 PNG 人工复核；本轮已修掉 A 的侧栏标签压选中行、A 详情按钮压省略号、B 的「已暂存」贴行、B 的 SHA 压省略号、C 的重复计数。

## 边界

- 没有修改 `design/compact/` 下的既有设计稿。
- 没有接入 MyGo / Go 后端，也没有改真实应用；所有画面是静态 SVG。
- 没有提交、推送或发布。
- 示例数据是固定的，不代表真实仓库的极端情况（超长分支名、上千文件、冲突、子模块嵌套）。
- 这仍是方向稿，不是可直接发布的成品 UI。

## 建议的下一步

先选一个方向；选定后只实现「工作区 + 历史」两屏到 MyGo，用真实仓库和真实子模块走一遍暂存、取消、提交、切分支，再决定是否扩展设置、Stash、冲突和 Worktree 页。
