# 单需求自动发布工作流实施计划（第一阶段：交付基础）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立可持久恢复的需求交付台账、发布预检和发布包核验入口，使「一次授权、通过后自动发布一个版本」具备可靠的确定性骨架。

**Architecture:** 新增 `internal/delivery` 保存任务阶段、授权、检查结果与发布窗口；新增 `tools/delivery` 提供 `start/check/advance/status/block/resume/authorize/revoke/queue/preflight/verify-release` 命令。Makefile 的测试范围改为显式源码包，发布路径改为「预检 → 构建 → 打包 → 独立核验」；`tools/releasepack` 校验 App 内置公钥与签名私钥一致。后续计划再接入分析、设计、编码、审查、原生验收的 Agent 执行器和 GitHub/Release 适配器。

**Tech Stack:** Go 1.27.3、标准库 `os/exec`、`crypto/ed25519`、现有 `internal/updater`、Makefile、`gh`（后续计划使用）。

**Spec:** `docs/superpowers/specs/2026-11-28-per-request-release-design.md`

## Global Constraints

- 第一阶段只实现确定性基础，不自动发布真实版本；所有发布相关测试使用临时仓库、临时密钥和临时产物。
- Go 包范围固定为 `./cmd/... ./internal/... ./tools/...`；只有 MyGo SDK 检查可以使用模块路径。
- 不新增第三方依赖；沿用 `go.mod` 现有依赖。
- 私有密钥只允许出现在测试临时目录或维护者本机路径；不得打印、提交或写入任务记录。
- 面向用户的 commit message 使用英文；每个任务结束运行 `gofmt`、`go vet` 和该任务的定向测试。
- 发布相关测试在非 macOS 上跳过需要 `ditto`/`codesign` 的用例；其余逻辑保持跨平台可测。
- 本计划不创建 tag、不发布 Release、不修改已发布资产。

## Review Focus

1. 任务在「创建 tag 后、发布前」崩溃并恢复时，不能生成第二个 tag 或第二个 Release；同名同 SHA 的 tag 与同内容资产应复用，内容不同则停止并保留现场。
2. 第一条需求已合并但尚未完成发布时，第二条需求不能合入 `main`，否则一个版本会包含两条需求。
3. `release-pack` 在 App 内置更新公钥为空、缺失或与签名私钥不匹配时必须失败，不能产出可发布归档。
4. 工作树脏、分支不是 `main`、本地 `main` 落后或领先远端、tag 不指向 HEAD 时，发布预检必须失败。
5. 代码在审查或验收后发生变化时，旧检查结果不能继续放行；候选版本不一致必须阻塞推进。

---

### Task 1: 统一源码包范围

**Files:**
- Modify: `Makefile:1-20`（变量区）
- Modify: `Makefile:75-89`（`test`、`check`）

**Interfaces:**
- Produces: `SOURCE_PKGS ?= ./cmd/... ./internal/... ./tools/...`，后续任务和文档引用该变量。

- [ ] **Step 1: 添加变量并替换测试目标**

在 `GO ?= go` 后加入：

```make
SOURCE_PKGS ?= ./cmd/... ./internal/... ./tools/...
```

将 `test` 改为 `$(GO) test $(SOURCE_PKGS)`；将 `check` 中两处 `./...` 和 `go vet ./...` 改为 `$(SOURCE_PKGS)`，保留 `github.com/egoist/mygo/ui` 的显式测试。

- [ ] **Step 2: 验证展开结果**

Run: `make -n test && make -n check`
Expected: 输出中普通测试、race、vet 均使用 `./cmd/... ./internal/... ./tools/...`；不再出现裸 `./...`；MyGo SDK 命令保持原样。

- [ ] **Step 3: 运行一个快速包确认命令可用**

Run: `go test ./internal/appicon`
Expected: PASS。

- [ ] **Step 4: Commit**

```bash
git add Makefile
git commit -m "chore(build): scope Go checks to source packages"
```

---

### Task 2: 发布预检逻辑

**Files:**
- Create: `internal/delivery/preflight.go`
- Test: `internal/delivery/preflight_test.go`

**Interfaces:**
- Consumes: `supergit/internal/updater.ParsePublicKey`、`updater.ParseVersion`。
- Produces:

```go
type PreflightOptions struct {
    RepoDir     string // default "."
    Branch      string // default "main"
    Remote      string // default "origin"
    Tag         string
    PublicKey   string
    PrivKeyPath string
}

type PreflightCheck struct {
    Name   string
    Passed bool
    Detail string
}

type PreflightReport struct {
    Checks []PreflightCheck
}

func (r PreflightReport) OK() bool
func Preflight(ctx context.Context, opts PreflightOptions) (PreflightReport, error)
```

检查名固定为 `branch`、`clean`、`synced`、`tag`、`public_key`、`private_key`。

- [ ] **Step 1: 写失败测试**

```go
func TestPreflightPassesCleanSyncedTaggedCheckout(t *testing.T)
func TestPreflightRejectsDirtyWorktree(t *testing.T)
func TestPreflightRejectsStaleRemote(t *testing.T)
func TestPreflightRejectsWrongBranch(t *testing.T)
func TestPreflightRejectsTagNotOnHead(t *testing.T)
func TestPreflightRejectsMismatchedPrivateKey(t *testing.T)
func TestPreflightRejectsWorldReadablePrivateKey(t *testing.T)
```

测试夹具：`t.TempDir()` 内 `git init`、设置 `user.name/user.email`、提交一次、创建 `v1.2.3` 标签、创建 bare remote 并 push。断言使用 `report.OK()` 与按名称查找的检查项。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/delivery -run 'TestPreflight' -v`
Expected: 编译失败，`Preflight` 未定义。

- [ ] **Step 3: 实现预检**

用 `exec.CommandContext("git", "-C", repo, ...)` 执行：
- `branch --show-current`
- `status --porcelain`
- `fetch --quiet <remote> <branch>` 后比较 `rev-parse HEAD` 与 `rev-parse <remote>/<branch>`
- `rev-parse refs/tags/<tag>^{}` 与 HEAD
- `updater.ParseVersion(strings.TrimPrefix(tag, "v"))`
- 公钥解析
- 私钥：`os.Stat` 模式必须为 `0600`，读取 base64、解码长度等于 `ed25519.PrivateKeySize`，派生公钥与 `PublicKey` 相等

所有失败写入对应检查项的 `Detail`，不返回 error；只有无法执行 Git 等 I/O 错误返回 error。`RepoDir`、`Branch`、`Remote` 为空时套用默认值。

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/delivery -run 'TestPreflight' -v`
Expected: 全部 PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/delivery/preflight.go internal/delivery/preflight_test.go
git commit -m "feat(delivery): add release preflight checks"
```

---

### Task 3: releasepack 校验内置公钥

**Files:**
- Modify: `tools/releasepack/main.go:114-125`
- Modify: `tools/releasepack/main_test.go`
- Modify: `Makefile:34-42`（`app` 目标）

**Interfaces:**
- Consumes: `loadPrivateKey` 派生的 `pub`、App plist 中的 `SuperGitUpdatePublicKey`。
- Produces: `packageRelease` 在内置公钥缺失或不匹配时返回错误；`make app UPDATE_PUBKEY=...` 把公钥写入 bundle plist。

- [ ] **Step 1: 写失败测试**

```go
func TestPackageReleaseRequiresMatchingEmbeddedPublicKey(t *testing.T)
```

用现有 `generateKeypair` 生成密钥，分别构造 plist：缺少 `SuperGitUpdatePublicKey`、写错公钥、写对公钥。断言前两者 `packageRelease` 返回错误，第三者成功。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./tools/releasepack -run TestPackageReleaseRequiresMatchingEmbeddedPublicKey -v`
Expected: 缺少/错误公钥的用例意外成功，测试失败。

- [ ] **Step 3: 实现校验并写入 plist**

在 `packageRelease` 读取 plist 后解析 `SuperGitUpdatePublicKey`，与 `base64.StdEncoding.EncodeToString(pub)` 精确比较；缺失、空值或不一致返回 `fmt.Errorf("embedded update public key ...")`。

在 `Makefile` 的 `app` 目标中，`plutil -insert CFBundleDocumentTypes ...` 之后加入：

```make
	$(if $(UPDATE_PUBKEY),plutil -insert SuperGitUpdatePublicKey -string "$(UPDATE_PUBKEY)" "$(APP)/Contents/Info.plist",)
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./tools/releasepack -run 'TestPackageRelease' -v`
Expected: PASS（非 macOS 的 `ditto` 用例按现有规则跳过）。

- [ ] **Step 5: Commit**

```bash
git add tools/releasepack/main.go tools/releasepack/main_test.go Makefile
git commit -m "fix(release): require the embedded update key to match the signing key"
```

---

### Task 4: 预检命令与 Makefile 发布门

**Files:**
- Create: `tools/delivery/main.go`
- Test: `tools/delivery/main_test.go`
- Modify: `Makefile:44-56`（`release-pack`）
- Modify: `docs/packaging.md` 的发布命令示例（本任务先改命令，Task 8 统一校对）

**Interfaces:**
- Consumes: `delivery.Preflight`。
- Produces:

```go
func run(args []string, stdout, stderr io.Writer) int
```

`preflight` 子命令参数：`-repo`、`-branch`、`-remote`、`-tag`、`-pubkey`、`-privkey`、`-json`。

- [ ] **Step 1: 写失败测试**

```go
func TestRunPreflightReportsChecksAndExitCode(t *testing.T)
func TestRunPreflightRequiresTagAndKeys(t *testing.T)
```

断言：全通过退出 0 且输出 `PASS branch`；任一失败退出 1 且输出 `FAIL <name>`；缺少 `-tag` 退出 2；`-json` 输出可解析的 `PreflightReport`。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./tools/delivery -run TestRunPreflight -v`
Expected: 编译失败，`run` 未定义。

- [ ] **Step 3: 实现命令与 Makefile 门**

CLI 使用 `flag.NewFlagSet`，把 `main` 变成 `os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))`。失败检查逐行输出 `FAIL name: detail`，全部通过输出 `PASS name: detail`。

Makefile 增加：

```make
RELEASE_TAG ?=

release-preflight:
	@test -n "$(RELEASE_TAG)" || (echo "RELEASE_TAG is required" >&2; exit 2)
	@test -n "$(UPDATE_PUBKEY)" || (echo "UPDATE_PUBKEY is required" >&2; exit 2)
	@test -n "$(UPDATE_PRIVKEY)" || (echo "UPDATE_PRIVKEY is required" >&2; exit 2)
	$(GO) run ./tools/delivery preflight -tag "$(RELEASE_TAG)" -pubkey "$(UPDATE_PUBKEY)" -privkey "$(UPDATE_PRIVKEY)"

release-pack: release-preflight app
.NOTPARALLEL: release-pack
```

- [ ] **Step 4: 验证命令与顺序**

Run: `go test ./tools/delivery -run TestRunPreflight -v`
Expected: PASS。

Run: `make -n release-pack RELEASE_TAG=v9.9.9 UPDATE_PUBKEY=x UPDATE_PRIVKEY=/tmp/k`
Expected: `preflight` 命令出现在 `go build` 之前。

- [ ] **Step 5: Commit**

```bash
git add tools/delivery/main.go tools/delivery/main_test.go Makefile docs/packaging.md
git commit -m "feat(release): gate packaging on release preflight"
```

---

### Task 5: 发布包核验逻辑与命令

**Files:**
- Create: `internal/delivery/release.go`
- Test: `internal/delivery/release_test.go`
- Modify: `tools/delivery/main.go`
- Modify: `tools/delivery/main_test.go`
- Modify: `Makefile`

**Interfaces:**
- Consumes: `updater.Manifest`、`updater.ParsePublicKey`、`updater.VerifyArchiveDigest`、`updater.InspectZipSafety`、`updater.ExtractAndVerifyApp`、`updater.VerifyAppBundleMetadata`、`updater.ExpectedBundleID`。
- Produces: `VerifyRelease`；CLI `verify-release` 参数 `-out`、`-archive`、`-manifest`、`-pubkey`、`-platform`；Makefile `verify-release` 目标读取 `$(RELEASE_OUT)` 和 `$(UPDATE_PUBKEY)`。

```go
type VerifyOptions struct {
    OutDir          string
    ArchivePath     string // empty: derive from manifest asset
    ManifestPath    string // empty: OutDir/update.json
    PublicKey       string
    Platform        string // default "darwin-arm64"
    VerifySignature func(context.Context, string) error
}

type ReleaseVerification struct {
    Version     string
    Build       int
    ArchivePath string
    SHA256      string
    Size        int64
    BundlePath  string
    Checks      []PreflightCheck
}

func VerifyRelease(ctx context.Context, opts VerifyOptions) (ReleaseVerification, error)
```

检查名固定为 `manifest`、`size`、`sha256`、`ed25519`、`zip_safety`、`extract`、`codesign`。

- [ ] **Step 1: 写失败测试**

```go
func TestVerifyReleaseAcceptsMatchingSignedArchive(t *testing.T)
func TestVerifyReleaseRejectsTamperedArchive(t *testing.T)
func TestVerifyReleaseRejectsBadSignature(t *testing.T)
func TestVerifyReleaseRejectsWrongBundleID(t *testing.T)
func TestVerifyReleaseRejectsUnsafeZip(t *testing.T)
func TestVerifyReleaseRejectsSizeMismatch(t *testing.T)
```

夹具：构造最小 `.app`（Info.plist 含 `CFBundleIdentifier=cc.supermix.supergit`、版本、可执行文件），用 `archive/zip` 打包；计算 SHA-256、用 ed25519 签名；写 `update.json`。测试注入 `VerifySignature: func(context.Context, string) error { return nil }`，避免依赖 `codesign`。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/delivery -run TestVerifyRelease -v`
Expected: 编译失败，`VerifyRelease` 未定义。

- [ ] **Step 3: 实现核验**

读取 `update.json`（上限 1 MiB），按 `Platform` 选择 asset；若 `ArchivePath` 为空，用 `path.Base(asset.URL)` 拼到 `OutDir`。依次核对文件大小、SHA-256、`VerifyArchiveDigest`、`InspectZipSafety`、`ExtractAndVerifyApp`（传入注入的签名校验）、`VerifyAppBundleMetadata`。所有检查写入 `Checks`；任一失败返回 `error`，已完成的检查保留在返回值中。

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/delivery -run TestVerifyRelease -v`
Expected: PASS。

- [ ] **Step 5: 接入 CLI 与 Makefile**

在 `tools/delivery/main.go` 增加 `verify-release` 子命令，把标志映射到 `VerifyOptions`，使用默认 `codesign` 校验（`VerifySignature=nil`）。成功输出版本、构建号、归档路径、SHA-256 和 bundle 路径；失败逐项输出 `FAIL <name>` 并退出 1。

在 `Makefile` 增加：

```make
verify-release:
	@test -n "$(UPDATE_PUBKEY)" || (echo "UPDATE_PUBKEY is required" >&2; exit 2)
	$(GO) run ./tools/delivery verify-release -out "$(RELEASE_OUT)" -pubkey "$(UPDATE_PUBKEY)"
```

在 `tools/delivery/main_test.go` 增加 `TestRunVerifyReleaseReportsAndExits`，断言成功退出 0、篡改归档退出 1、缺少 `-pubkey` 退出 2。

- [ ] **Step 6: 运行命令测试**

Run: `go test ./internal/delivery ./tools/delivery -run 'TestVerifyRelease|TestRunVerifyRelease' -v`
Expected: PASS。

- [ ] **Step 7: Commit**

```bash
git add internal/delivery/release.go internal/delivery/release_test.go tools/delivery/main.go tools/delivery/main_test.go Makefile
git commit -m "feat(delivery): verify signed release archives end to end"
```

---

### Task 6: 任务台账与阶段门

**Files:**
- Create: `internal/delivery/task.go`
- Create: `internal/delivery/store.go`
- Test: `internal/delivery/task_test.go`
- Test: `internal/delivery/store_test.go`

**Interfaces:**
- Produces:

```go
type Phase string
const (
    PhaseIntake Phase = "intake"
    PhaseAnalysis Phase = "analysis"
    PhaseDesign Phase = "design"
    PhaseDesignReview Phase = "design_review"
    PhaseImplementation Phase = "implementation"
    PhaseCodeReview Phase = "code_review"
    PhaseAcceptance Phase = "acceptance"
    PhasePullRequest Phase = "pull_request"
    PhaseMerge Phase = "merge"
    PhaseReleasePreflight Phase = "release_preflight"
    PhaseTag Phase = "tag"
    PhasePublish Phase = "publish"
    PhaseVerify Phase = "verify"
    PhaseComplete Phase = "complete"
)

type Status string
const (
    StatusActive Status = "active"
    StatusBlocked Status = "blocked"
    StatusCancelled Status = "cancelled"
)

type Authorization struct {
    AuthorizedBy   string
    Mode           string // "auto"
    Autonomy       string // "per_request"
    Release        bool
    VersionPolicy  string // "patch" | "minor" | "exact"
    Version        string
    Channel        string
    AcceptanceItems []string
    PolicyVersion  string
    GrantedAt      time.Time
    RevokedAt      *time.Time
    Revisions      []AuthorizationRevision
}

type AuthorizationRevision struct {
    By            string
    Reason        string
    Release       bool
    VersionPolicy string
    Version       string
    At            time.Time
}

type Event struct {
    At     time.Time
    Phase  Phase
    Action string
    Detail string
}

type Check struct {
    Name       string
    Candidate  string
    Passed     bool
    Evidence   string
    RecordedAt time.Time
}

type Task struct {
    ID                  string
    Request             string
    Repository          string
    BaseBranch          string
    Tier                string
    Phase               Phase
    Status              Status
    Authorization       Authorization
    Candidate           string
    Checks              map[string]Check
    PR                  int
    Version             string
    Tag                 string
    BlockedReason       string
    ReviewRounds        int
    CreatedAt           time.Time
    UpdatedAt           time.Time
    History             []Event
}

func NewTask(id string, req StartRequest, now time.Time) (Task, error)
func (t *Task) RecordCheck(c Check) error
func (t *Task) SetCandidate(candidate string, now time.Time) error
func (t *Task) SetTier(tier string, now time.Time) error
func (t *Task) SetPR(number int, now time.Time) error
func (t *Task) Advance(to Phase, now time.Time) error
func (t *Task) Block(reason string, now time.Time) error
func (t *Task) Resume(now time.Time) error
func (t *Task) Revoke(by string, now time.Time) error
func (t *Task) AuthorizeRevision(rev AuthorizationRevision, now time.Time) error
```

`StartRequest` 字段：`Request, Repository, BaseBranch, AuthorizedBy, VersionPolicy, Version, Channel string; AcceptanceItems []string; Release bool`。

`Store`：

```go
type Store struct { Root string }
func NewStore(root string) *Store
func (s *Store) Create(t Task) error
func (s *Store) Load(id string) (Task, error)
func (s *Store) Save(t Task) error
func (s *Store) Update(id string, fn func(*Task) error) (Task, error)
func (s *Store) List() ([]Task, error)
```

- [ ] **Step 1: 写失败测试**

```go
func TestAdvanceRejectsOutOfOrderPhase(t *testing.T)
func TestAdvanceRequiresPassedCheckForCurrentCandidate(t *testing.T)
func TestAdvanceRejectsCheckFromOlderCandidate(t *testing.T)
func TestSkipDesignRecordsReason(t *testing.T)
func TestRevokedAuthorizationBlocksMerge(t *testing.T)
func TestAuthorizeRevisionClearsRevocation(t *testing.T)
func TestStoreRoundTripAndList(t *testing.T)
func TestStoreRejectsDuplicateTaskAndBadID(t *testing.T)
```

阶段顺序：`intake→analysis→design→design_review→implementation→code_review→acceptance→pull_request→merge→release_preflight→tag→publish→verify→complete`。仅允许从 `analysis` 直接跳到 `implementation`（设计跳过）。离开当前阶段所需的检查名：`intake`、`analysis`、`design`、`design_review`、`tests`、`code_review`、`acceptance`、`remote_checks`、`merge`、`release_preflight`、`package`、`publish`、`public_verify`。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/delivery -run 'TestAdvance|TestSkip|TestRevoked|TestAuthorizeRevision|TestStore' -v`
Expected: 编译失败。

- [ ] **Step 3: 实现台账**

- `NewTask` 要求 `AuthorizedBy`、`Request` 非空，`Release` 为 true，`VersionPolicy` 合法；写入 intake 阶段的 `intake` 检查。
- `Advance` 校验下一阶段合法；离开当前阶段的检查必须 `Passed`，且 `Check.Candidate == T.Candidate`（intake/analysis 阶段允许空候选）。
- 跳过设计：`analysis→implementation` 允许，但必须存在 `design` 检查且 `Evidence` 以 `skipped:` 开头。
- 进入 `merge` 要求授权未撤销且 `Release` 为 true；`merge` 及之后撤销授权立即把状态置为 `blocked`。
- `AuthorizeRevision` 追加修订，并用修订中的 `Release`、`VersionPolicy`、`Version` 覆盖授权字段；清空 `RevokedAt`、状态恢复 `active`。
- 每次变更追加 `Event` 并更新 `UpdatedAt`。
- `Store` 使用 `<root>/tasks/<id>.json`，目录 0700，临时文件同目录写入后 `os.Rename`；ID 只允许 `[a-z0-9-]`。

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/delivery -run 'TestAdvance|TestSkip|TestRevoked|TestAuthorizeRevision|TestStore' -v`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/delivery/task.go internal/delivery/store.go internal/delivery/task_test.go internal/delivery/store_test.go
git commit -m "feat(delivery): persist task phases and enforce authorization gates"
```

---

### Task 7: 发布窗口与任务命令

**Files:**
- Create: `internal/delivery/queue.go`
- Test: `internal/delivery/queue_test.go`
- Modify: `tools/delivery/main.go`
- Test: `tools/delivery/main_flow_test.go`
- Create: `scripts/check-delivery-dry-run.py`

**Interfaces:**
- Produces:

```go
type ReleaseWindow struct {
    TaskID string
    Since  time.Time
}

func (s *Store) AcquireReleaseWindow(id string, now time.Time) error
func (s *Store) ReleaseReleaseWindow(id string) error
func (s *Store) CurrentReleaseWindow() (ReleaseWindow, error)
```

CLI 子命令：`start`、`check`、`advance`、`status`、`block`、`resume`、`authorize`、`revoke`、`queue`。`status` 支持 `-json`。

- [ ] **Step 1: 写失败测试**

```go
func TestReleaseWindowBlocksSecondTask(t *testing.T)
func TestReleaseWindowReleasedOnComplete(t *testing.T)
func TestRunTaskFlowPersistsAcrossInvocations(t *testing.T)
func TestRunStatusJSONIncludesPhaseAndAuthorization(t *testing.T)
func TestRunRevokeThenAuthorizeRevision(t *testing.T)
```

`TestReleaseWindowBlocksSecondTask`：任务 A 获取窗口；任务 B 获取失败；A 完成后 B 成功。`TestRunTaskFlowPersistsAcrossInvocations`：每次 `run` 使用新的 `run` 调用，模拟进程重启，状态仍可继续。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/delivery -run TestReleaseWindow -v && go test ./tools/delivery -run 'TestRunTaskFlow|TestRunStatusJSON|TestRunRevoke' -v`
Expected: 编译失败。

- [ ] **Step 3: 实现队列与命令**

- 队列文件 `<root>/queue.json`；持有者不匹配时 `Acquire` 返回错误；`Advance` 进入 `merge` 前必须当前任务持有窗口；`verify→complete` 自动释放窗口。
- `start` 参数：`-id -request -repo -by -version-policy -version -channel -acceptance -release`；`-release` 默认 true，本阶段不接受 `false`，否则退出 2。
- `check` 参数：`-id -name -candidate -passed -evidence`。
- `advance` 参数：`-id -to -candidate`（候选变化时自动 `SetCandidate` 并要求重新记录检查）。
- `queue` 参数：`-acquire -release -id`。
- `status` 无 `-id` 时列出全部任务摘要。
- 所有命令错误退出 1，参数错误退出 2。

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/delivery ./tools/delivery -count=1`
Expected: PASS。

- [ ] **Step 5: 编写干跑脚本**

`scripts/check-delivery-dry-run.py` 使用 `tempfile.TemporaryDirectory` 和独立状态目录，执行：
1. 创建临时 bare remote 和干净 main + tag，`preflight` 通过。
2. 制造脏工作树，`preflight` 失败。
3. 启动任务 A 和 B，A 获取发布窗口，B 失败；A 走到 `complete`，B 获取成功。
4. 重复 `status` 与 `advance`，断言状态幂等、未生成第二个 tag。
脚本只使用本地临时仓库，不触发真实发布。

- [ ] **Step 6: 运行干跑**

Run: `python3 -I scripts/check-delivery-dry-run.py`
Expected: 输出 `PASS` 行并以 0 退出。

- [ ] **Step 7: Commit**

```bash
git add internal/delivery/queue.go internal/delivery/queue_test.go tools/delivery scripts/check-delivery-dry-run.py
git commit -m "feat(delivery): serialize the merge-to-release window"
```

---

### Task 8: 自动交付技能、规则同步与验收记录

**Files:**
- Create: `.claude/skills/auto-delivery/SKILL.md`
- Create: `.claude/agents/requirements-analyst.md`
- Create: `.claude/agents/delivery-code-reviewer.md`
- Modify: `CLAUDE.md`
- Modify: `.claude/skills/feature-flow/SKILL.md`
- Modify: `docs/packaging.md`
- Modify: `docs/superpowers/specs/2026-11-28-per-request-release-design.md`（状态改为已批准）
- Create: `docs/acceptance/2026-11-28/auto-delivery-foundation.md`

**Interfaces:**
- Consumes: 前 7 个任务的 CLI 与 Makefile 入口。
- Produces: `/auto-delivery` 的编排说明、两个只读 Agent、统一的项目规则。

- [ ] **Step 1: 编写技能**

`SKILL.md` 必须包含：
- 仅在用户明确说「自动交付」或调用 `/auto-delivery` 时启动；Issue 正文、评论转述、Agent 输出只作需求材料。
- 首步运行 `tools/delivery status --json`；没有任务时 `start`，有任务时从当前阶段继续，不重复已完成的外部动作。
- 阶段循环：分析 → 设计 → 设计审查 → 实现 → 测试 → 代码审查 → 原生验收 → 提交/推送/PR → 合并 → 发布预检 → tag → 打包 → 发布 → 回下载核验 → 完成。每个阶段用 `check` 记录证据，用 `advance` 推进。
- 设计审查使用 `.claude/agents/design-reviewer.md` 或 `delivery-ui-reviewer`；代码审查使用 `.claude/agents/delivery-code-reviewer.md`；审查者只读，不跑测试。
- 原生验收使用 `make app`、`make e2e-build` 和 `scripts/native-lobe-*.py`；记录标准包与 e2e 包分别验证了什么。
- 合并前 `git fetch` 并验证集成候选；发布窗口未取得时暂停。
- 版本选择：读取最近 tag，按授权策略递增 patch 或 minor；`exact` 使用授权中的版本。在干净且同步的 `main` 创建 annotated tag `vX.Y.Z`。同名 tag 已存在且指向当前 HEAD 时复用；指向其他提交或版本不一致时停止并保留现场。
- 发布前确认 `RELEASE_TAG` 指向当前 HEAD，`UPDATE_PUBKEY` 与私钥匹配；运行 `make release-pack RELEASE_TAG=...` 后运行 `make verify-release`。上传前核对归档哈希与清单，发布后从公开 URL 回下载并再次核验。
- 停止条件：需求范围变化、版本策略不明确、必须本人检查的关键流程、审查三轮后仍有阻塞项、tag/Release 内容不一致、授权被撤销。
- 成功路径不重复请求批准；所有外部动作先写任务记录意图，再执行并回读真实状态。

- [ ] **Step 2: 编写 Agent**

`requirements-analyst.md`：只读分析需求、现有行为、验收场景和 T1/T2/T3，输出缺口与建议，不改文件。
`delivery-code-reviewer.md`：只读审查候选补丁、相关文件、测试计划；按 bug、回归、安全、架构风险排序；不跑测试、不改文件。

- [ ] **Step 3: 同步规则**

- `CLAUDE.md`：新增「Automated delivery」小节，说明一次授权覆盖合并与发布、状态由 `tools/delivery` 保存、普通模式仍按原确认规则、源码包范围为 `SOURCE_PKGS`。
- `feature-flow/SKILL.md`：T3 可转交 `/auto-delivery`；测试命令引用 `SOURCE_PKGS`。
- `docs/packaging.md`：发布顺序改为「干净同步 main → 创建 tag → `make release-pack RELEASE_TAG=...` → `verify-release` → 发布」；删除过时的 README 章节引用。
- 设计文档状态改为 `Status: Approved — 2026-11-28`。

- [ ] **Step 4: 静态检查**

Run: `rg -n './\\.\\.\\.' Makefile .claude/skills/feature-flow/SKILL.md CLAUDE.md`
Expected: 仅 MyGo SDK 显式模块路径或说明性文字，不出现普通测试使用 `./...`。

Run: `rg -n 'auto-delivery|delivery status|release-preflight|verify-release' CLAUDE.md .claude/skills docs/packaging.md`
Expected: 四个入口均有引用。

- [ ] **Step 5: 记录验收**

`docs/acceptance/2026-11-28/auto-delivery-foundation.md` 记录：源码提交、运行的命令与结果、干跑输出、未覆盖项（真实 Agent 编排、真实 GitHub PR、真实发布）。明确本阶段未创建 tag 或 Release。

- [ ] **Step 6: 运行完整定向检查**

Run: `gofmt -l internal/delivery tools/delivery tools/releasepack && go vet ./internal/delivery ./tools/delivery ./tools/releasepack && go test ./internal/delivery ./tools/delivery ./tools/releasepack -count=1`
Expected: `gofmt` 无输出；vet 与测试 PASS。

- [ ] **Step 7: Commit**

```bash
git add .claude CLAUDE.md docs/superpowers/specs/2026-11-28-per-request-release-design.md docs/packaging.md docs/acceptance/2026-11-28/auto-delivery-foundation.md
git commit -m "docs(delivery): add the auto-delivery workflow and release rules"
```

---

## 后续计划（不在本计划范围）

- Agent 执行器：把分析、设计、实现、审查、原生验收串成可恢复的循环；支持中断后从 `tools/delivery status` 继续。
- GitHub 适配器：创建 PR、等待必需检查、合并、回读合并提交；处理 main 推进和冲突。
- 发布适配器：在受保护环境创建 tag、调用固定的打包/验包程序、上传资产、回下载核验；复用同一份已验归档。
- 真实试用：由维护者明确授权一条小需求，跑完整链路到公开版本，记录验收证据。

## Self-Review

- 规格覆盖：授权记录、阶段门、发布窗口、预检、验包、失败恢复、执行边界均有对应任务；Agent 编排和真实发布明确列为后续计划。
- 类型一致性：`PreflightCheck` 同时用于预检与验包报告；`AuthorizationRevision` 在 Task 6 定义，Task 7 的 CLI 复用；`Store` 方法在 Task 6 定义，Task 7 只增加队列方法。
- Review Focus 的五个失败模式分别由 Task 7（窗口/恢复）、Task 2（脏/落后/分支/tag）、Task 2+3（公钥）、Task 6（候选变化）覆盖。
- 本计划不包含真实发布动作。
