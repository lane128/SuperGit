#!/usr/bin/env python3
"""Create local lightweight tags through the real UI in a fresh fixture only."""
import argparse
import json
import pathlib
import re
import subprocess

parser = argparse.ArgumentParser()
parser.add_argument("root", type=pathlib.Path)
parser.add_argument("--session", required=True)
args = parser.parse_args()
root = args.root.resolve()
if not root.name.startswith("supergit-workflows.") or not (root / "remote.git").is_dir():
    raise SystemExit("requires a fresh create-workflow-demo.sh fixture")
repo = root / "workspace"
automation = pathlib.Path("build/mygo-automation").resolve()


def git(*items):
    return subprocess.check_output(["git", "-C", str(repo), *items], text=True)


def auto(*items):
    result = subprocess.run([str(automation), *items, "--session", args.session], capture_output=True, text=True, timeout=35)
    if result.returncode:
        raise RuntimeError(f"automation {items}: {result.stdout}\n{result.stderr}")
    return result.stdout


def controls():
    entries = []
    for line in auto("snapshot", "--full").splitlines():
        match = re.match(r'\s*(@e\d+) \[(\w+)\] ("(?:[^"\\]|\\.)*")(.*)', line)
        if match:
            entries.append((match[1], match[2], json.loads(match[3]), match[4]))
    return entries


def ref(label, role="button"):
    found = [e for e in controls() if e[1] == role and e[2] == label and "disabled" not in e[3]]
    if not found:
        raise RuntimeError(f"missing enabled {role} {label!r}: {controls()}")
    return found[0][0]


def press(label):
    auto("press", ref(label), "--settle")


def fill(label, value):
    auto("fill", ref(label, "textField"), value, "--settle")


def key(value):
    auto("key", value, "--settle")


def wait(text):
    auto("wait", "--text", text, "--timeout", "25s", "--settle")


def screenshot(name):
    auto("pointer", "move", "20", "40", "--settle")
    auto("screenshot", str(pathlib.Path("build", name).resolve()))


def assert_unchanged(expected_head):
    assert git("rev-parse", "HEAD") == expected_head, "tag workflow moved HEAD"
    assert git("ls-files", "--stage") == index, "tag workflow changed Index"
    assert git("status", "--porcelain") == status, "tag workflow changed working status"
    assert {p: (repo / p).read_bytes() for p in working} == working, "tag workflow changed working bytes"
    assert git("ls-remote", "--tags", "origin") == remote_tags, "local workflow published a tag"
    assert git("symbolic-ref", "refs/tags/dangling-browse").strip() == "refs/heads/missing-tag-target", "browsing altered dangling symbolic tag"


def open_form(name):
    press("创建轻量标签…")
    wait("标签名称")
    fields = [e[2] for e in controls() if e[1] == "textField"]
    assert fields == ["标签名称"], f"tag SHA must be read-only: {fields}"
    wait("目标提交 SHA：" + selected_sha)
    fill("标签名称", name)


assert pathlib.Path(git("remote", "get-url", "origin").strip()).resolve() == root / "remote.git"
wait("就绪")
key("CmdOrCtrl+" + str(4))
wait("当前仓库没有可解析的本地标签。")
# Git silently omits dangling symbolic refs. The UI states that boundary,
# including when this is the only local tag; browsing must preserve the ref.
git("symbolic-ref", "refs/tags/dangling-browse", "refs/heads/missing-tag-target")
key("CmdOrCtrl+R")
wait("就绪")
wait("当前仓库没有可解析的本地标签。")
wait("悬空符号引用不会列出")
screenshot("tag-browse-empty.png")
selected_sha = git("rev-parse", "HEAD").strip()
selected_subject = git("log", "-1", "--format=%s").strip()
git("commit", "--allow-empty", "-m", "Tag fixture later HEAD")
head = git("rev-parse", "HEAD")
# Preserve both Index and worktree versions, plus the existing untracked draft.
(repo / "tag-dirty.txt").write_text("INDEX VERSION\n")
git("add", "--", "tag-dirty.txt")
(repo / "tag-dirty.txt").write_text("WORKTREE VERSION\n")
index = git("ls-files", "--stage")
status = git("status", "--porcelain")
working = {line.split("\t", 1)[1]: None for line in index.splitlines()}
working.update({p: None for p in git("ls-files", "--others", "--exclude-standard").splitlines()})
working = {p: (repo / p).read_bytes() for p in working}
remote_tags = git("ls-remote", "--tags", "origin")
tags_before = git("show-ref", "--tags") if git("tag", "--list").strip() else ""
key("CmdOrCtrl+R")
wait("就绪")
key("CmdOrCtrl+2")
press(selected_subject)
wait(selected_sha)
open_form("cancelled-form")
key("Escape")
assert git("tag", "--list", "cancelled-form") == ""
open_form("cancelled-confirmation")
press("检查操作")
wait("确认创建轻量标签")
wait("本地轻量标签：refs/tags/cancelled-confirmation")
wait("目标提交 SHA：" + selected_sha)
wait("仓库：" + str(repo))
assert git("tag", "--list", "cancelled-confirmation") == ""
press("返回编辑")
wait("标签名称")
assert any(e[1] == "textField" and e[2] == "标签名称" and 'value="cancelled-confirmation"' in e[3] for e in controls())
press("检查操作")
press("取消")
assert git("tag", "--list", "cancelled-confirmation") == ""
assert (git("show-ref", "--tags") if git("tag", "--list").strip() else "") == tags_before
assert_unchanged(head)
print("PASS native tag form has name-only editing and form/confirmation cancellation never writes", flush=True)

name = "release/候选"
open_form(name)
press("检查操作")
wait("不覆盖已有标签")
screenshot("tag-create-confirmation.png")
press("确认执行")
wait("就绪")
wait("tag: " + name)
assert git("rev-parse", "refs/tags/" + name).strip() == selected_sha, "tag followed HEAD instead of selected commit"
assert git("cat-file", "-t", "refs/tags/" + name).strip() == "commit", "tag is not lightweight"
assert_unchanged(head)
screenshot("tag-created-history.png")
# Constrain the detail pane to 400 DIP: actions must wrap inside the pane,
# and the new button must remain usable through the metadata scroll area.
auto("pointer", "down", "707", "300", "--settle")
auto("pointer", "move", "1040", "300", "--settle")
auto("pointer", "up", "1040", "300", "--settle")
auto("wheel", "1200", "200", "0", "180", "--precise", "--settle")
entry = next(e for e in controls() if e[1] == "button" and e[2] == "创建轻量标签…")
bounds = re.search(r"bounds=([\d.]+),([\d.]+),([\d.]+)x([\d.]+)", entry[3])
assert bounds and float(bounds[1]) >= 1040 and float(bounds[1]) + float(bounds[3]) <= 1440, entry
screenshot("tag-narrow-history.png")
open_form("cancelled-narrow-pane")
press("取消")
assert git("tag", "--list", "cancelled-narrow-pane") == ""
auto("pointer", "down", "1040", "300", "--settle")
auto("pointer", "move", "707", "300", "--settle")
auto("pointer", "up", "707", "300", "--settle")
assert_unchanged(head)
print("PASS native tag creation targets the older selected SHA, refreshes decoration and keeps actions usable in a narrow pane without changing dirty state or remote", flush=True)

# Select a different commit before trying the same name, so an overwrite would
# visibly change its OID rather than silently repeating the same target.
press("Tag fixture later HEAD")
selected_sha = head.strip()
wait(selected_sha)
open_form(name)
press("检查操作")
press("确认执行")
wait("reference already exists")
wait("就绪")
assert git("rev-parse", "refs/tags/" + name).strip() != selected_sha
assert git("rev-parse", "refs/tags/" + name).strip() == git("rev-parse", head.strip() + "^").strip()
assert_unchanged(head)
screenshot("tag-existing-ref-error.png")
print("PASS native duplicate tag at a different target fails without overwriting the original ref", flush=True)

open_form("stale-confirmation")
press("检查操作")
git("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "--only", "-m", "External tag confirmation advance")
advanced_head = git("rev-parse", "HEAD")
press("确认执行")
wait("当前分支或上游已变化，请刷新并重新确认")
assert git("tag", "--list", "stale-confirmation") == ""
assert_unchanged(advanced_head)
print("PASS native external HEAD advance refuses stale tag confirmation without writing a ref", flush=True)

# Import an independent 360-commit chain: its oldest tagged commit must remain
# inspectable without appending a fake row or loading the entire graph.
records = []
for n in range(360):
    message = "Archived tag target\n\nPINNED TARGET BODY\n" if n == 0 else f"Tag browser commit {n:03d}\n"
    content = "OLDEST TAG CONTENT\n" if n == 0 else f"TAG HISTORY {n:03d}\n"
    record = f"commit refs/heads/tag-browser-history\nmark :{n + 1}\ncommitter Test <test@example.invalid> {1700000000 + n} +0000\ndata {len(message.encode())}\n{message}"
    if n:
        record += f"from :{n}\n"
    record += f"M 100644 inline archived.txt\ndata {len(content.encode())}\n{content}\n"
    records.append(record)
subprocess.run(["git", "-C", str(repo), "fast-import", "--quiet"], input="".join(records).encode(), capture_output=True, check=True)
old_sha = git("rev-parse", "refs/heads/tag-browser-history~359").strip()
new_sha = git("rev-parse", "refs/heads/tag-browser-history").strip()
archive = "archive/旧版本"
git("tag", "-a", archive, "-m", "Annotation is not the commit subject", old_sha)
archive_object = git("rev-parse", "refs/tags/" + archive).strip()
git("tag", "-a", "nested-release", "-m", "Nested annotation", "refs/tags/" + archive)
git("tag", "artifacts/blob", "HEAD:conflict.txt")
git("symbolic-ref", "refs/tags/release-symbolic", "refs/heads/main")
expected_refs = git("for-each-ref", "--format=%(refname) %(objectname) %(symref)")
key("CmdOrCtrl+" + str(4))
key("CmdOrCtrl+R")
wait("就绪")
wait("搜索本地标签")
fill("搜索本地标签", "候选")
wait("匹配 1 / 5 个标签")
fill("搜索本地标签", "no-match-for-tags")
wait("没有匹配的标签。")
screenshot("tag-browse-no-match.png")
fill("搜索本地标签", "artifacts/blob")
wait("目标不是提交对象，无法显示提交详情。")
assert any(e[1] == "button" and e[2] == "查看标签提交：artifacts/blob" and "disabled" in e[3] for e in controls())
fill("搜索本地标签", "release-symbolic")
wait("符号引用")
wait("引用目标：refs/heads/main")
fill("搜索本地标签", archive)
wait("附注标签")
wait("标签对象：" + archive_object)
wait("目标对象：" + old_sha)
screenshot("tag-browse-annotated.png")
# Branch insertion precedes the tag section. Refresh must preserve the search
# field's focus/value instead of changing its ordinal control identity.
git("branch", "tag-browse/extra")
expected_refs = git("for-each-ref", "--format=%(refname) %(objectname) %(symref)")
key("CmdOrCtrl+R")
wait("就绪")
assert any(e[1] == "textField" and e[2] == "搜索本地标签" and "focused" in e[3].split() and ('value="' + archive + '"') in e[3] for e in controls()), "branch insertion lost tag search focus/value"
assert git("for-each-ref", "--format=%(refname) %(objectname) %(symref)") == expected_refs
assert_unchanged(advanced_head)
print("PASS native tag inventory distinguishes empty/search/no-match and annotated/symbolic/noncommit targets while preserving search focus", flush=True)

press("查看标签提交：" + archive)
wait("此提交尚未载入历史列表；详情按固定 SHA 显示。")
wait("标签来源：" + archive)
wait(old_sha)
wait("+OLDEST TAG CONTENT")
assert not any(e[1] == "listItem" and ("key=" + old_sha) in e[3] for e in controls()), "tag browsing fabricated a graph row"
assert any(e[1] == "textField" and e[2] == "搜索已加载提交" and "value=" not in e[3] for e in controls()), "tag navigation retained an unrelated history search"
screenshot("tag-browse-unloaded-commit.png")
key("CmdOrCtrl+R")
wait("就绪")
wait("+OLDEST TAG CONTENT")
wait("标签来源：" + archive)
key("CmdOrCtrl+" + str(4))
fill("搜索本地标签", "nested-release")
press("查看标签提交：nested-release")
wait(old_sha)
wait("标签来源：nested-release")
wait("+OLDEST TAG CONTENT")
press("Tag browser commit 359")
wait(new_sha)
assert not any(e[1] == "text" and e[2].startswith("标签来源：") for e in controls()), "explicit history selection retained old tag metadata pin"
assert git("for-each-ref", "--format=%(refname) %(objectname) %(symref)") == expected_refs
assert_unchanged(advanced_head)
print("PASS native old and nested tag targets open exact unloaded commit details without changing graph rows; refresh preserves pin and explicit history selection replaces it", flush=True)

# This task-created ref is intentionally made unavailable, then repaired by
# fixture setup. The UI itself only retries reads and never repairs refs.
key("CmdOrCtrl+" + str(4))
fill("搜索本地标签", "")
broken_ref = repo / ".git/refs/tags/broken-browse"
assert not broken_ref.exists()
broken_bytes = "f" * 40 + "\n"
broken_ref.write_text(broken_bytes)
key("CmdOrCtrl+R")
wait("就绪")
wait("标签读取失败：")
assert not any(e[1] == "text" and e[2] == "当前仓库没有可解析的本地标签。" for e in controls()), "tag read failure became empty success"
press("重试读取标签")
wait("就绪")
wait("标签读取失败：")
assert broken_ref.read_text() == broken_bytes, "read-only retry altered broken ref"
assert_unchanged(advanced_head)
screenshot("tag-browse-read-error.png")
broken_ref.write_text(advanced_head)
expected_refs = git("for-each-ref", "--format=%(refname) %(objectname) %(symref)")
press("重试读取标签")
wait("就绪")
wait("匹配 6 / 6 个标签")
assert not any(e[1] == "text" and e[2].startswith("标签读取失败：") for e in controls()), "old tag error survived recovery"
assert git("for-each-ref", "--format=%(refname) %(objectname) %(symref)") == expected_refs
assert_unchanged(advanced_head)
screenshot("tag-browse-recovered.png")
print("PASS native tag read failure stays distinct from empty and read-only retry recovers after explicit fixture repair", flush=True)
