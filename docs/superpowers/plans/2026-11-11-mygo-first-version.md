# SuperGit MyGo First Version Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Build and launch a real macOS Git client with MyGo native UI, safe staging, commit, history graph, and submodule awareness.

**Architecture:** `internal/gitrepo` owns Git subprocesses and repository models. `internal/desktop` owns application state, background jobs, and native views. `cmd/supergit` starts MyGo and accepts an optional repository path. Source design prototypes remain intact.

**Tech Stack:** Go 1.27.3, MyGo v0.8.3 native UI, system Git.

**Spec:** `docs/superpowers/specs/2026-11-11-mygo-first-version-design.md`

## Global Constraints

- Go 1.27.3、MyGo v0.8.3 原生 GPU UI；不使用 WebView 包装设计稿。
- 使用 os/exec 参数数组，不调用 shell，不拼接命令字符串执行。
- 文件参数使用 `--` 分隔；支持空格、中文、换行、重命名和前导短横线。
- 提交仅包含 Index，保留 Git hooks 和失败输出，禁止隐式 add --all。
- 不自动 pull、push、reset、discard 或初始化子模块，不删除用户内容。
- 保留旧设计稿、不 commit、不发布。

## Review Focus

- 初次提交前取消暂存不能删除工作区文件；在 Task 1 验证。
- 重命名和带换行的路径必须保持完整；在 Task 1 使用真实仓库验证。
- git hook 失败时保留暂存内容并报告失败；在 Task 1 验证。
- 未初始化子模块目录不能误识别为父仓库；在 Task 1 验证。
- 切换仓库后的慢任务不能覆盖当前仓库状态；在 Task 2 验证。

---

### Task 1: Real Git boundary

**Files:** Create `go.mod`, `internal/gitrepo/{types,repository,status,diff,history,submodule}.go`; tests `internal/gitrepo/*_test.go`.

**Interfaces:** `Open(ctx context.Context, path string) (*Repository,error)`; `(*Repository).Status(ctx) (Status,error)`; `Diff(ctx,path string,staged bool) (Diff,error)`; `Stage(ctx,path) error`; `Unstage(ctx,path) error`; `ApplyHunk(ctx,patch string,reverse bool) error`; `Commit(ctx,message string) error`; `History(ctx,limit int) ([]Commit,error)`; `CommitFiles(ctx,sha string) ([]ChangedFile,error)`; `Show(ctx,sha,path string) (Diff,error)`; `Submodules(ctx) ([]Submodule,error)`; `Worktrees(ctx) ([]Worktree,error)`; `Branches(ctx) ([]Branch,error)`.

- [x] Write integration tests using temporary Git repos for odd filenames, partial Index/worktree state, unborn unstage, selected-file commit, failed hooks, hunk staging/reverse, history parents, worktrees and submodule HEAD/Index/current separation.
- [x] Run `go test ./internal/gitrepo`; Expected: missing implementation failures.
- [x] Implement models and safe command execution, bounded output, NUL parsers, context cancellation and serialized writes.
- [x] Run `go test ./internal/gitrepo`; Expected: all tests pass.

### Task 2: Native application and asynchronous state

**Files:** Create `cmd/supergit/main.go`, `internal/desktop/{state,app,workspace,history,repositories,theme}.go`, `internal/desktop/state_test.go`.

**Consumes:** Repository and typed models from Task 1.
**Produces:** `desktop.Run(initialPath string) error`; native MyGo window with folder picker, navigation, workspace, commit form, history graph, submodules, branches and worktree views.

- [x] Test background result generation matching and write busy guard before implementation. Expected: fails for missing state behavior.
- [x] Implement pure state transitions and worker completion messages; consume them only on main-thread frames.
- [x] Use MyGo native widgets, scroll/virtual lists and drawing primitives. Keep all Git calls in goroutines. Use coherent split-pane layout and semantic diff colors.
- [x] Connect file and hunk staging, commit, history selection, submodule enter/back, theme and code font adjustments; show failures and actual operation output.
- [x] Run `go test ./...`; Expected: all tests pass. Run `go build ./cmd/supergit`; Expected: native executable builds.

### Task 3: Actual use and delivery

**Files:** Create `README.md`, `Makefile`, `.gitignore`, `scripts/create-demo.sh`; output executable under `build/`.

**Consumes:** Native app from Task 2.
**Produces:** Reproducible build/run commands and an actual opened native window on a task-owned demo repo.

- [x] Run `go test -race ./...` and `go vet ./...`; Expected: pass.
- [x] Create demo repository with two diff hunks, staged/unstaged files, a real merge, worktrees and submodule pointer change. Only write under a new task-owned temporary directory.
- [x] Build `build/supergit`, run on the demo and confirm native launch. Capture a screenshot if macOS permissions allow, without reading unrelated windows.
- [x] Review Git writes, filenames, subprocess/thread boundaries, child repository navigation, stale asynchronous results and missing feature claims. Fix important defects and rerun relevant checks.
- [x] Report implemented capabilities, exact verification, runnable binary and first-version limitations.

No commit steps: the user has not authorized commits. Worktree scripts require a Git checkout; this workspace is not one, so keep the execution record in the adjacent progress file and work in place.
