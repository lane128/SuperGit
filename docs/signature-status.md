# 提交签名状态：读什么、显示什么、哪里会看不清

这份文档记录提交签名徽章的数据来源、每个状态的含义和几处 Git 自身的边界。改签名相关代码或文案前先读这里；文案与配色的单元测试在 `internal/lobedesk/signature_test.go`，读取与解析在 `internal/gitrepo/signature.go`。

## 读取方式

对选中的提交执行一条只读查询：

```
git show --no-patch --no-show-signature --encoding=UTF-8 \
  --format=format:%G?%x00%GS%x00 <sha> --
```

- `%G?` 是 Git 的验签结论，一个字母；`%GS` 是签名者名字（GPG 的 uid 或 SSH key 的注释／id）。
- 两个字段用 NUL 分隔，`parseSignature` 要求恰好两段且以 NUL 结尾，否则当作无效记录报错。
- `--no-show-signature` 必须带上：全局配置 `log.showSignature=true` 会把验签文本混进 stdout，破坏上面的格式。
- 查询只读，不写 Index、HEAD 或工作区；失败只影响这一条提交的徽章，不影响提交详情其余内容。

## 显示规则

签名徽章只出现在提交详情的元数据区，历史列表行不显示（避免逐行验签拖慢长列表）。`Signature.Signed()` 对空状态和 `N` 都返回 false：没有签名就不画徽章，而不是画一个「未签名」徽章。

| `%G?` | 界面文案（中文） | 配色 | 含义 |
| --- | --- | --- | --- |
| `G` | 已验证签名 | success | 签名有效且密钥可信 |
| `U` | 签名有效（信任未知） | info | 签名有效，但密钥不在信任模型内 |
| `B` | 签名无效 | error | 内容与签名不符 |
| `X` | 签名已过期 | error | 签名本身过期 |
| `Y` | 签名密钥已过期 | error | 密钥过期 |
| `R` | 签名密钥已吊销 | error | 密钥被吊销 |
| `E` | 无法验证签名 | gray | Git 无法执行校验（缺 verifier、格式不支持等） |
| `N` | 不显示 | — | 没有签名 |

## 已知边界

1. **无法校验与未签名可能混在一起**（2026-11-15 实测）。SSH 签名但未配置 `gpg.ssh.allowedSignersFile` 时，`git show --format=%G?` 在 stderr 打印 `error: gpg.ssh.allowedSignersFile needs to be configured and exist for ssh signature verification`，同时 stdout 给出 `N`，`git verify-commit` 退出码为 1。应用只读 stdout 的字母，因此显示为「未签名」，与真的没签名无法区分。要区分只能另跑 `git verify-commit` 读退出码和 stderr，这会为每条提交多一次进程，暂不做。
2. **四种失败状态只有单元测试**。B / X / Y / R 的文案与配色由 `TestSignatureBadgesCoverEveryStatus` 固定；e2e 目前只实际看到过 `G` 与 `N`。构造过期、吊销或无效签名需要专门的密钥与时间环境，未纳入 fixture。
3. **没有显示密钥指纹**。`%GS` 给的是签名者名字或 key id，不是指纹。`%GK` 可以给出 key 的指纹，若要显示，应加在徽章的 tooltip 而不是主文案，避免把元数据行撑宽。
4. **签名者名字是用户数据**。`%GS` 可能为空（Git 无法命名密钥）或很长；徽章只显示状态，签名者文本按普通次要信息显示，不做翻译。
5. **Shell / 全局配置的影响**：`gpg.program`、`gpg.ssh.program`、`user.signingkey` 和信任库都由 Git 自己读；应用不缓存跨提交的结论，切换提交就重新查询。

## 改动检查单

- 新增一个 Git 可能返回的状态字母：在 `signatureLabel` / `signatureTagColor` 补文案与颜色，并把字母加进 `TestSignatureBadgesCoverEveryStatus`。
- 改文案：同时更新中英文两份 `i18n.go`，并同步上表。
- 改查询命令：保留 `--no-show-signature` 与 NUL 分隔格式，跑 `go test ./internal/gitrepo/ ./internal/lobedesk/`。
