# 历史线图的配色规则

这份文档说明历史页的线、节点和引用胶囊分别从哪里取色、为什么这样取，以及哪些对比度由测试守住。改动线图配色前先读这里，改完跑 `go test ./internal/lobedesk/`。

## 1. 车道颜色来自哪里

- 每条车道（lane）的线色来自当前配色方案自己的强调色表，而不是 Lobe 的语义 token。Catppuccin 的 Lime 与 Green、Gold 与 Yellow 在 token 里会落到同一个颜色，扩到 12 色时会重复；直接读方案色板才能让 12 条线各不相同。
- 车道顺序由 `laneAccentRoles` 固定：Peach、Teal、Red、Blue、Maroon、Lavender、Green、Mauve、Yellow、Sapphire、Pink、Sky。相邻车道是读者最容易并排比较的两条线，所以相邻槽位的色相距离有测试下限（60°，当前顺序约 115°）。
- 默认（Lobe）方案只有 10 个不重复的族：Gold、Lime 与 Orange、Yellow 太近，不单独占槽。
- 每个方案都有 light / dark 两套车道色；主题模式只决定用哪一套。方案的品牌感因此从浅色切到深色时保持同一组色相。
- 历史被筛选后车道会重排，但分支的颜色不能跟着车道号变，否则同一个分支换个筛选就换颜色。`laneColorIndex` 用分支名的 FNV-1a 哈希取槽位，颜色跟随名字；图里实际画出的颜色以 `RefLanes` 为准。

代码：`themes.go` 的 `appTheme`、`laneAccentRoles`、`lanePalette`；`graph.go` 的 `laneColorIndex`、`refLaneColor`、`commitGraph.RefColor`。

## 2. 引用胶囊穿哪条线的颜色

`refBadgeColor` 是引用胶囊的取色入口：

- **分支、远端分支**是「线」，胶囊必须穿图里真实画出的那条线的颜色：先查 `RefLanes`（这条引用实际占用的车道），查不到就退回该行所在车道的颜色（`RowLaneColor`），再退回名字哈希槽。这样点一个胶囊就能在行上找到它指的那条线；不会出现「标签说蓝色、行上却没有蓝线」。
- **标签**是「点」，不会长出自己的车道，所以沿用标签列表和引用选择器里的名字颜色（`refLaneColor`），不跟着行上的线走。
- **HEAD** 不是线，也不是标签，用主题的 Blue 单独做一个胶囊，并带勾图标。
- 胶囊同时用图标区分种类：分支是分支图标、远端是云、标签是 Tag、HEAD 是勾；颜色之外还有形状，色觉差异的读者也能分辨。

代码：`graph.go` 的 `refBadgeColor`、`refBadge`、`parseCommitRef`；`ref_capsule.go`。

## 3. 胶囊的三层颜色

一个引用胶囊由底色、边框、文字墨色组成，三者都从同一个车道色和它所在的行的底色推导：

| 层 | 公式 | 目的 |
| --- | --- | --- |
| 底色 | `color.Alpha(0.14).Over(ground)` | 让胶囊带一点这条线的色相，但不做成一整块实色 |
| 边框 | `color.Alpha(0.30).Over(ground)`；与行底色亮度不足 1.25:1 时改为向墨色混合直到达标 | 给胶囊一个边缘，圆角 10、高 20；车道色与行底色亮度接近时，边缘仍能靠明度分开 |
| 墨色 | `capsuleInk(color, 底色, 4.5)` | 标签文字与自身底色至少 4.5:1 |

`capsuleInk` 分两种表面：

- 浅色表面：向黑混，直到达到 4.5:1，保留色相（`deepInk`）。深色化后饱和度不得低于 0.35，否则标签读起来像灰字。
- 深色表面：原色如果已经达到 4.5:1 就原样使用；达不到（饱和红、饱和蓝）向白混，直到达标。提亮后饱和度不得低于 0.20，避免提成白色而丢掉线色。

中性胶囊（`+N`、远端名等没有车道可指的胶囊）没有色相可依靠，因此用完整的 `Fill`（12%）而不是更淡的 `FillSecondary`，墨色对胶囊底色同样要 4.5:1，胶囊底色对行底色至少 1.15:1。

代码：`ref_capsule.go`（`refCapsuleColors`、`capsuleInk`、`neutralCapsule`、`refPill`）。

## 4. 选中、悬停与节点

- 行的底色：普通行用页面底色，悬停行用 `FillTertiary`，选中行用 `selectionBg`（主题强调色的淡底）。胶囊、节点和线的取色都以「这一行实际的底色」为地面，而不是假定页面底色。
- 节点：普通提交是实心圆（半径 4），合并提交是方框描边（半边长 5）；选中时在节点外加 18×18 的底色晕圈和 14×14 的 2px 环，节点与环之间留出缝隙。选中的线从 2px 加到 3px（第 150 轮起线宽和节点大小可配置，选中仍比普通多 1px）。节点半径随倍率缩放，但至少比线宽多 1 DIP：否则「粗线 + 小节点」会把节点画得和线一样宽，提交看起来只剩一条线（第 162 轮）。
- 晕圈用 `graphNodeGround` 取当前行底色，这样在选中或悬停行上不会出现一个页面色的圆洞。
- 线在浅色主题上要压深到 3.2:1（`laneInk`），否则 2px 的细线在白底上太淡；深色主题保持原色。
- 边界（boundary）提交的节点下方画虚线尾巴，颜色用 `boundaryInk`，表示这里的分支边界被截断。

## 5. 测试守住的不变量

| 不变量 | 测试 |
| --- | --- |
| 每个方案 12（默认 10）个槽位、无重复、相邻色相 ≥60° | `TestLanePalettesAreWideAndOrdered` |
| 浅色胶囊墨色 ≥4.5:1 且饱和度 ≥0.35 | `TestRefLabelInkKeepsHueOnEveryLightScheme` |
| 深色胶囊墨色 ≥4.5:1、饱和度 ≥0.20，且确实有颜色被提亮 | `TestRefCapsuleInkKeepsHueOnDarkSchemes` |
| 中性胶囊在普通/悬停/选中行的填充 ≥1.15:1、文字 ≥4.5:1 | `TestNeutralCapsuleReadsOnEveryRow` |
| 所有车道色在普通/悬停/选中地面上的胶囊文字 ≥4.5:1、边框对地面 ≥1.25:1 | `TestCapsulesReadOnEveryMixedGround` |
| 分支/远端胶囊的颜色确实出现在该行的线上，标签用名字色 | `TestEveryRefPointsAtAColorOnItsRow` |
| 浅色节点/线压深后仍保留色相 | `TestGraphInkKeepsLaneLinesOnLight` |
| 文档里写的胶囊尺寸（宽 150/320、高 20、圆角 10、底色/边框比例）与代码一致 | `TestGraphColorDocMatchesTheCapsuleConstants`（读第 7 节的表） |
| 文档里写的节点半径 4、合并方框半边长 5、选中环 14×14 / 晕圈 18×18、线压深 3.2:1 与代码算出的值一致 | `TestGraphColorDocMatchesTheGraphGeometry`（读第 7 节的表） |

## 6. 已知边界

- 默认（Lobe）浅色方案里，蓝色或黄色车道的胶囊在选中行上原本与行底色的亮度差只有约 1.02–1.09:1；第 179 轮起边框在亮度差不足 1.25:1 时向墨色混合，边缘同时靠明度区分，其余方案不受影响。文字仍保持 4.5:1。
- 胶囊宽度上限：历史行 150px、详情面板 320px，均按界面字号比例放大（第 177 轮：大字号下固定宽度会把已中略的名字再截一次，一个标签出现两个省略号），长名字中略；窄窗口（980，第 171 轮）复查过：胶囊长度与窗口宽度无关，窗口变窄时先截断提交标题。省略规则本身（是否保留 `origin/` 前缀）仍是待定项。
- 中略的字符预算：历史行 20 个字符，详情面板 40 个（第 182 轮，`refBadgeRunes` / `refCapsuleDetailRunes`）。两者都是「保留头尾」的中略：详情面板原先交给布局按宽度从尾部截断，共享长前缀的两个 dependabot 引用会在详情里显示成同一个名字，改成按字符预算中略后尾部差异得以保留。
- 深色方案的胶囊墨色可能被提亮到与原线色不同，线本身保持原色；两者并排时是否脱节曾目视复查过一次（第 145 轮），新增配色方案时应重新看一遍。
- 线宽、节点大小自第 150 轮起可配置；节点放到 1.3 倍时首尾车道距列边缘约 1.3px，再大的倍率需要同时放大车道宽度。

## 7. 机器校验的数值

下面这张表是文档里被测试读取的唯一来源。正文可以改措辞，改了数字就必须同时改这里；`TestGraphColorDocMatchesTheCapsuleConstants` 与 `TestGraphColorDocMatchesTheGraphGeometry` 按键名取值，与代码算出的结果逐项比较，改代码后测试报出的期望值就是这里要填的值。

| 键 | 值 |
| --- | --- |
| capsule-tint | 0.14 |
| capsule-border | 0.30 |
| capsule-height | 20 |
| capsule-corner | 10 |
| capsule-row-width | 150 |
| capsule-detail-width | 320 |
| node-dot-radius | 4 |
| node-merge-half | 5 |
| node-ring | 14 |
| node-halo | 18 |
| lane-ink-min | 3.2:1 |
