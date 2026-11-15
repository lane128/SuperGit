# 紧凑 Git 工作台 · 本轮设计依据

目标是快速理解仓库、Index、工作区和文件状态。沿用用户已认可的功能范围，降低视觉负担。所有选项均为静态评审方案，不增加产品设置。

## 参考与取舍

- Sublime Merge Getting Started：https://www.sublimemerge.com/docs/getting_started 。实际查看官方 overview.png 和 stage_lines.png。借鉴位置/历史/文件的稳定分区、选择驱动详情、文件/片段/行操作的作用范围。文档没有说明双击子模块行为，不将自拟行为归因于 Sublime Merge。
- GitHub Primer Button：https://primer.style/product/components/button/ 。借鉴 primary/default/invisible/danger 层级，以及标签与前置图标整体居中的规则。
- GitHub Primer TreeView：https://primer.style/product/components/tree-view/ 。借鉴文件图标、尾部状态、截断与完整名称语义。28px 行高是本稿提案，不声称是这些产品的精确规格。
- Lucide：https://lucide.dev/icons ，https://lucide.dev/license 。使用固定版本 lucide-static 0.468.0 的原始 SVG 几何，保留许可文件；不再自行绘制通用图标。

## 共用规则

基础色：画布 #f7f8fa，内容 #ffffff，侧栏 #f1f3f5，文字 #24292f，边界 #d0d7de，操作 #245fbe。状态颜色配合文字 M/A/D/R/?/U 和明确说明。界面 Helvetica Neue / PingFang SC；代码 Menlo。文字层级仍为 20/15/13/11px；主要文件名13px、辅助路径11px。

布局为中性浅色工作台，仓库身份和 Git 状态构成主层级。移除大面积品牌色导航和括号装饰。文件项28px单行，名称、路径、类型、状态列固定；当前完整路径位于 Diff 头部。选中只改变底色，悬停不移动列。

```text
仓库页签 / 当前仓库
位置与引用 | 文件名   所在路径   类型 状态 | 当前路径 / Index → 工作区
分支       | 28px 单行文件列表           | Diff
子模块仓库 | 已暂存 / 未暂存分组          | 行号 / 增删 / 内容
           | 提交说明                    | 片段与行级操作
```

子模块使用相同的仓库图标、28px列表、工作区、历史和提交区域。父仓库记录的 gitlink 与子模块内部文件明确分开；关联信息放入普通属性区，不另造卡片风格。

提示按实际行数建立一个文字组，再把文字组与图标垂直居中于容器。按钮标签居中，有图标时图标+标签整体居中。新增设计校验读取真实 SVG bbox，而不是只依靠固定 baseline。

## 可比较方案

按钮三套：A 平面边框（本稿默认）、B 轻立体桌面、C 低强调。每套展示同一组主/次/无边框/危险按钮；仅供设计选型。

历史轨道三套：A 小圆角、B 45°转轨、C 正交。三套使用同一组真实父关系样例；检查连续主线、分叉、合并、八爪鱼合并、同一提交多个引用、分页边界、过滤后的缺失父节点，以及选中时线和节点的可读性。提交行28px；线宽1.5px，普通节点半径3.5px、合并节点空心环，节点和引用不互相遮挡。

本轮身份主要来自文件状态表达、信息密度、稳定作用范围和操作一致性。审查时先检查这些规则是否落实，再评价外观。
