# SuperGit UI 设计交接

面向 Go/MyGo 开发。开始一个 UI 改动时，先读 [开发设计规范](UI-Spec.md) 中对应章节，再打开画稿页码核对视觉效果。规范写明产品行为、设计数值和实施范围，画稿提供布局、层级和状态的视觉参照。

## 工作台布局评审

[布局对比 PDF](SuperGit-Design-Variants.pdf) 的首页并排展示 A、B、C 三套方案，点击缩略图可跳到对应的全尺寸稿。[交互预览](SuperGit-Design-Variants.html) 使用同一组演示数据，可切换布局、主题和窗口尺寸。

| 对比稿页码 | 内容 |
| --- | --- |
| 1–4 | 三版对比、A 经典三栏、B Diff 优先、C 审阅布局 |
| 5 | 980 × 640 窄窗、同步按钮收纳、暂存操作条和撤销 Toast |
| 6–8 | 提交历史、按钮与反馈、设置与弹窗 |

A 是推荐的默认布局候选。实现时根据已选方案查画稿，由开发规范确定组件尺寸、Git 行为与验收条件。

## 组件合订画稿
[SuperGit-Design-Reference.pdf](SuperGit-Design-Reference.pdf) 合并下列画稿，含目录和章节书签。页码按合订后的 PDF 计算。

| 主题 | 合订页码 | 原始画稿 | 对应规范 |
| --- | ---: | --- | --- |
| 工作台、Git 历史、文件与行选择 | 2–5 | [UI Style Guide v2.1](SuperGit-UI-Style-Guide-v2.pdf) | [工作台](UI-Spec.md#workbench)、[历史线图](UI-Spec.md#history) |
| 浅色与深色按钮、控件形态及状态 | 6–7 | [Button Spec](SuperGit-Button-Spec.pdf) | [按钮与控件](UI-Spec.md#buttons) |
| 外观、历史与 Git、快捷键、关于 | 8–10 | [Settings & Dialogs](SuperGit-Settings-Dialog-Spec.pdf) 第 1–3 页 | [设置窗口](UI-Spec.md#settings) |
| 推送、删除 Stash、创建标签、恢复设置 | 11–12 | [Settings & Dialogs](SuperGit-Settings-Dialog-Spec.pdf) 第 4–5 页 | [弹窗](UI-Spec.md#dialogs) |
| Toast、内联提示、Tooltip、进度与错误详情 | 13–16 | [Toast & Feedback](SuperGit-Toast-Feedback-Spec.pdf) | [反馈](UI-Spec.md#feedback) |

## 开发入口

| 需要查什么 | 位置 |
| --- | --- |
| 颜色、字号、尺寸、按钮状态 | [基础规格](UI-Spec.md#foundation) |
| 画稿与现有代码的差异 | [实施范围](UI-Spec.md#scope) |
| 每部分改哪里、怎么接入 | [实现顺序](UI-Spec.md#rollout) |
| 最小窗口、深浅色、键盘与真实 Git 验收 | [验收](UI-Spec.md#verification) |
| 已有命令与禁用条件 | [工作台命令映射](../../docs/workbench-command-map.md) |
| 使用临时 Git 仓库做原生验收 | [原生自动化](../../docs/e2e-automation.md) |

先在 `cmd/supergit-lobe` 预览入口实现和验证，UI 代码在 `internal/lobedesk`，Git 操作复用 `internal/gitrepo`。落实前核对现有操作守卫、快捷键和多窗口归属。

## App 图标

[App 图标候选](app-icon-options/README.md) 提供三套浅色与深色 PNG、Dock 尺寸对比和打包方式。

## 旧稿与探索

- [第一版视觉指南](SuperGit-UI-Style-Guide.pdf) 保留了三栏工作台的早期探索。
- [第一版 HTML 原型](supergit-workbench.html) 可体验示例工作区和历史切换，使用演示数据。按钮、导航与 Git 行为按当前规范对照。
- [早期精修稿](../refinement/README.md)、[紧凑稿](../compact/README.md) 与 [方向稿](../directions/README.md) 保留历史设计依据。
