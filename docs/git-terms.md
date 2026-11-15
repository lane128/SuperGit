# 中文界面保留英文的 Git 术语

这份文档说明中文界面里哪些词故意保留英文、为什么，以及新增一个英文词要走哪几步。改动 i18n 文案前先读这里；判定的自动检查是 `TestChineseStringsOnlyKeepTheListedEnglishTerms`。

## 规则

默认全中文。只有两类词保留英文：

1. **产品与平台名**：Git、SuperGit、Lobe、MyGo、Catppuccin 及其变体、Finder、UI。翻译它们会让读者认不出自己在用什么。
2. **没有稳定中文名的 Git 对象或旗标**：Stash、HEAD（含 `detached HEAD`）、SHA、Ours/Theirs、hooks、force-with-lease、`.gitmodules`。这些词在 Git 文档、命令行输出和主流 GUI 里都以英文出现；硬译会造成「界面一个词、终端另一个词」。

Git 的**动作**一律用中文：提交、分支、合并、变基、拣选（cherry-pick）、回退（revert）、获取（fetch）、推送（push）、清理（prune）、重置（reset）、检出（checkout）、暂存／取消暂存、暂存区。命令名本身（`git rebase` 之类）只在解释行为时出现，不当作按钮或菜单文案。

## 当前清单

清单的唯一来源是 `internal/lobedesk/i18n.go` 的 `chineseEnglishTerms`；下表解释每一项，改动时两边一起改。

| 词 | 类别 | 出现位置（示例键） | 为什么保留 |
| --- | --- | --- | --- |
| Git | 产品 | `openRepo`、`gitBehavior` | 产品名，不译 |
| SuperGit | 产品 | `openRepoHint` | 应用名 |
| Lobe / MyGo / UI | 产品 / 平台 | `paletteDefault`、`settingsAboutText`、`previewNote` | 组件库与界面栈的名字 |
| Catppuccin / Frappe / Macchiato | 配色方案名 | `paletteCatppuccin*` | 方案原名；译名无法对上社区文档 |
| Finder | 平台 | `openInFinder`、`revealInFinder` | macOS 菜单里的固定叫法 |
| Stash | Git 对象 | `stashes`、`saveStashTitle`、`stashFileAction` | 既是对象名也是功能名；「贮藏」没有通行度 |
| HEAD / detached HEAD | Git 引用 | `detached`、`checkoutCommitMessage`、`undoCommitMessage` | 与 `git` 输出一致；「分离头指针」反而更难认 |
| SHA | Git 标识 | `copySHA`、`searchHistory` | 提交哈希的通用叫法 |
| Ours / Theirs | 冲突侧 | `useOurs`、`useTheirs` | 与 `git checkout --ours/--theirs` 对应 |
| hooks | Git 机制 | `commitHelp` | 与仓库里的 `hooks/` 目录对应 |
| force-with-lease | 推送旗标 | `forcePushMessage` | 具体旗标名，译名会隐藏真实行为 |
| .gitmodules | 文件名 | `noSubmodulesHint` | 仓库里的真实文件 |
| Git | 命令与概念 | 多处 | 见第一类 |

`git` 小写、`index`、`origin`、`upstream` 等**不在**清单里：界面用「暂存区」「远端」「上游」表达，远端名（`origin`）作为用户数据原样显示，不参与文案扫描。

## 同一个动作只用一个动词

同一个动作在界面里有好几份文案：菜单项、对话框标题、确认按钮、标题栏的进行中文案、失败提示。它们各自是一个 i18n 键，改一份容易漏另一份——`busyRevert` 曾写「正在还原…」而失败提示写「回退失败」，`busyDiscard` 曾写「正在丢弃改动…」而菜单写「放弃改动…」，两种说法指同一件事。

这些成组的键登记在 `internal/lobedesk/i18n_action_copy_test.go` 的 `actionCopyGroups`：每组写明中文动词（每条中文都必须包含）与英文词干（`busy*` 与 `*Failed` 的英文必须以该词干开头的某个词出现）。新增一个进行中文案或失败提示时，把它加进对应动作组；没登记的会被测试的覆盖检查拦下。读取失败、系统目录选择、剪贴板这类没有动作副本的失败提示登记在 `actionCopyExempt`，并写明原因。

菜单项与对话框标题的关系另由 `TestActionLabelsCarryTheirEllipsisInI18n` 固定（菜单 = 标题 + 「…」），通用确认按钮（保存、删除、创建）被多个对话框共用，不进入动作组。

## 新增或删除一个英文词

1. 在 `internal/lobedesk/i18n.go` 的 `chineseEnglishTerms` 里加／删这个词。
2. 更新本文件上表，写清类别、示例键和理由。
3. 跑 `go test ./internal/lobedesk/ -run TestChineseStringsOnlyKeepTheListedEnglishTerms`。未登记的拉丁词会让测试列出所在的 i18n 键；登记后通过。

测试只扫描 `localeChinese` 的文案。Git 自身的错误输出（`git` 子进程原文）会原样进入 toast 和错误行，那是用户数据，不做翻译也不受这份清单约束。

## 已知边界

- 单字母占位符（`%s`、`%d`）不算英文词；测试按「至少两个字母」识别拉丁词。
- 英文文案（`localeEnglish`）不扫描：它本来就该是英文。
- 远端分支名、文件路径、用户提交说明只显示、不翻译，也不在扫描范围内。
- 如果将来决定全中文，先在这张表上定译名，再改 `chineseEnglishTerms` 和文案；不要只改一处。
