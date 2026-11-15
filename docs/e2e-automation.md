# e2e 自动化验证笔记

范围：`cmd/supergit-lobe`（`-tags e2e` 构建）与 `mygo-automation` 的配合。这里是验证时踩过的操作细节，不是产品文档；实现以代码与 `mygo-automation skills` 的输出为准。

## 启动

```sh
go build -tags e2e -o /tmp/supergit-e2e ./cmd/supergit-lobe
HOME=/tmp/sg-e2e-home MYGO_AUTOMATION=/tmp/sg-e2e.sock \
  SUPERGIT_WINDOW_SIZE=1000x700 /tmp/supergit-e2e /tmp/sg-e2e-repo
# 另一个终端
mygo-automation open --socket /tmp/sg-e2e.sock
```

- 每个会话用独立的 `HOME`、socket 与 fixture 仓库，避免读到本机偏好。
- 需要点击（指针）时，macOS 只把指针事件发给激活窗口。后台用 `&` 直接拉起的进程可能全部点击失效而键盘仍可用；把可执行文件包进最小 `.app` 用 `open` 启动，或确保它是当前前台窗口。同 bundle id 的两个实例会互相抢激活，多仓库验证时给每个实例不同的 `CFBundleIdentifier`，或一次只跑一个。
- 结束时删除临时仓库、`HOME`、socket 与二进制；预览仓库（如 `~/Repos/lazygit`）必须保持未修改。

## 定位元素：先读 bounds，不靠目测

- `snapshot --json` 给出每个节点的 `bounds`（DIP）。悬停、点击都用它算坐标；历史记录里多次出现「目测偏了一截」导致的假失败。
- ref 会随重绘失效：每次界面变化后重新 `snapshot`，不要复用上一帧的 ref。
- `click <sel>` 优先；`press`/`focus`/`fill`/`act` 按元素操作。确实要按坐标时用 `click <x> <y>`、`pointer move <x> <y>`。
- 截图是逻辑尺寸的 2 倍（Retina）。从截图量出来的像素坐标先除以 2 再换算成窗口 DIP。

## 右键菜单

`mygo-automation skills` 明确建议不要用 `click --button right` 打开上下文菜单（菜单会留在截图里，坐标还要换算，ref 也会随重绘变化）。优先：

1. 用菜单键打开：`key Shift+F10`（或 `press <ref>` 前的菜单键写法，以 skills 输出为准）。
2. 再 `snapshot` 找到菜单项 ref，`press <ref>`。

只能右键时，按下面的流程，并把每一步写进验证记录：

```sh
# 1. 右键打开菜单（坐标用 snapshot 的 bounds）
mygo-automation click 420 260 --button right
# 2. 截图；菜单与行都在图里
mygo-automation screenshot /tmp/sg-menu.png
# 3. 在截图里量出菜单项像素坐标，除以 2 得到 DIP，
#    再用 click <x> <y>；量不到就重新截图，不要猜
mygo-automation click 512 301
```

注意：菜单打开期间再 `snapshot` 可能拿不到菜单项（系统/绘制菜单的覆盖层不一定进辅助功能树）。项目里的菜单动作优先用 headless 测试（`Tester.Find` 找绘制出来的菜单项）覆盖，e2e 只验证打开/关闭与关键项。

## 滚动

- `wheel` 一次只移动约一两行，滚到内容底部很慢；验证滚动相关行为时优先拖滚动条（按下、移动、松开三步），并断言滚动后的内容与选中态。
- 需要程序化滚动时用 `wheel x y dx dy`，注意指针位置要落在目标滚动区内，否则滚的是页面外层。

## 等待与判定

- 用 `wait <sel>`、`wait --text <s>` 等界面内容出现，而不是固定 `sleep`；它们只保证界面出现，不保证后台 Git 完成。
- `--settle` 等「窗口把自己请求的帧画完」，默认 10s；它不等应用自己设的计时器或后台任务。带 `--settle` 的动作会打印与上次快照的差异，`(no changes)` 也要读，别只看退出码。
- 后台任务失败时界面会出 toast/错误块；先找它，再判断操作是否成功。
- 验证写操作后回到 shell 用 Git 核对（`git status`、`git diff --cached`、`git config --get` 等），不只信截图。
