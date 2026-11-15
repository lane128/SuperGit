# SuperGit · 紧凑工作台设计

这一轮把界面收敛到“先看清 Git 状态和文件状态”的紧凑工作台：28px 行高、固定列、居中控件、三套按钮外观、三套历史走线，子模块直接复用仓库界面。所有内容都是静态设计稿，**尚未应用到 MyGo 原生客户端**。

## 先看什么

| 文件 | 内容 |
| --- | --- |
| `review.pdf` | 26 页评审版：S1–S5 规范、关键界面、子模块、设置、6 个交互状态 |
| `roadmap.pdf` | 52 页完整版：40 个工作流画板 + 5 页规范 + 8 个分区 |
| `S1.png` … `S5.png` | 五页设计规范，逐项说明密度、按钮、Diff、线图、子模块 |
| `frames/` | 46 张界面状态 PNG（A1–H5、U1–U6） |
| `ux-states.png` | U1–U6 交互状态总览 |
| `compare-A1.png` / `compare-B1.png` / `compare-C1.png` / `compare-D1.png` | 与上一轮精修稿的并排对照 |
| `roadmap-preview.png` | 40 个画板与状态转场的总览 |

## 八点反馈对应的改动

1. **提示没有居中**：提示条改为图标组与文字组共用同一个中心线（`primitives.mjs` 的 `notice()`）。单行 40px、双行 64px，两行文字作为一个块居中；渲染时自动测量，本次 0 偏差。
2. **按钮文字居中、样式要多**：标签或“图标 + 标签”作为一个内容组在按钮内居中（`data-button-content` 对 `data-center` 实测，0 偏差）。提供 A 紧凑默认、B 舒适桌面、C 低强调三套外观，全部映射 `ui.Button` / `ui.PrimaryButton`，不新增控件语义。
3. **文件行太大**：文件列表改为 28px 单行，六列固定：图标、路径（左侧省略）、名称、类型 30px、状态 16px、操作 28×28 预留槽。选中只改底色，列不移动；完整路径放在 Diff 头部。
4. **基于业界常用设计**：参考 Sublime Merge 的选择驱动详情、文件/片段/行暂存，GitHub Primer 的按钮与树形列表约定，再落到 MyGo 现有组件，不再自造一套通用控件。
5. **线图重做**：同一组提交给出 A 小圆角、B 45° 转轨、C 正交三种走线；28px 行、18px 轨道、1.5px 边，合并节点空心、八爪鱼合并按 slot 错开、多引用各自成 pill、过滤边界用短虚线，选中保留行底色。规则检查禁止 C 曲线。
6. **子模块就是仓库**：子模块列表、详情、Diff、历史使用与普通仓库相同的页签和组件；只额外显示父 HEAD / Index / 当前检出三指针，并明确“暂存父指针不会提交子模块内部改动”。打开子模块 = 新增仓库页签。
7. **图标用现成资源**：32 个图标取自 `lucide-static@0.468.0` 原始几何，统一由 `ui.Icon` 承载；保留 `LUCIDE-LICENSE`（ISC，Feather 部分 MIT）。不再手绘通用图标。
8. **简洁、易读**：浅色中性工作台；Git 状态使用一致的字母 + 颜色 + 分组语言；界面只保留定位、状态、动作三类信息。深色 token 已定义并通过对比度检查，本轮导出以浅色界面为主。

## 共用规则

- 四级文字：20/28 页面标题、15/24 分组标题、13/20 正文与操作、11/16 辅助信息；代码 13px。
- 间距基准 4px；控件高 28px；焦点环 2px 在元素外绘制。
- 颜色 token 同时定义浅色与深色；22 组前景/背景对比度 ≥ 4.5。
- Diff：统一/左右、两侧 gutter、行号 11px、代码 13px、行高 22px、`ContextLines=3`。

## MyGo 组件映射

| 界面元素 | 组件 |
| --- | --- |
| 主按钮 / 次按钮 / 链接 | `ui.PrimaryButton` / `ui.Button` / `ui.Link` |
| 顶栏 Fetch、Pull、Push | `ui.Toolbar` |
| 统一 / 左右切换 | `ui.Segmented` |
| 历史搜索 | `ui.SearchField` |
| 文件列表、分支列表 | `ui.Table`（固定列、选择、双击） |
| 分组列表、侧栏 | `ui.List` / `ui.Sidebar` |
| 仓库页签 | `ui.Tabs` |
| 路径导航 | `ui.Breadcrumbs` |
| 弹窗 / 危险确认 | `ui.Dialog` / `ui.AlertDialog` |
| 右键菜单 | `ui.ContextMenu` |
| 悬停说明 | `ui.Tooltip` |
| 完成提示 | `ui.Toast` |
| 计数 | `ui.Badge` |
| 等待 | `ui.Spinner` |
| 提示条 | `ui.Row`（图标 + 文字 + 可选操作） |
| Diff | `editor.DiffView`（`DiffState`、`DiffSplit`/`DiffUnified`） |
| 图标 | `ui.Icon` + Lucide |

历史线图是唯一自绘部分：MyGo 没有时间线组件，节点与边由 Git 父关系数据驱动，不复制第三方图形。

## 验证

```sh
node design/compact/generate.mjs
node design/compact/render.mjs
swift build/inspect-compact.swift
```

最近一次结果：

- 生成 40 个工作流画板、5 页规范、8 个分区、40 条转场、7 类设置。
- 渲染 46 个界面状态、5 页规范、8 个分区、4 组对照；PDF 52 / 26 页。
- 文字越界 0、截断控件 0、碰撞 0、按钮居中偏差 0、提示对齐偏差 0、28px 行高偏差 0。
- 字号仅 11/13/15/20；所有 `data-component` 都有 MyGo 映射；所有图标来源为 `lucide-static@0.468.0`；线图边 1.5px 且无 C 曲线。
- PDFKit：52 / 26 页、每页可选文字、中文与英文工作区均存在，7 页原生渲染抽查通过。

`render.mjs` 使用隔离目录中的 Playwright 与本机 Chrome；该依赖路径随临时目录变化，重新生成前需确认可用。

## 边界

- 这是静态设计提案，不改 Go/MyGo 客户端源码，不执行真实 Git 写操作。
- 未 commit、未 push、未发布；功能范围沿用上一轮。
- 颜色、字号、行高与控件外观仍需在 MyGo 实际渲染中复核，尤其是字体度量、焦点环和深色主题。
