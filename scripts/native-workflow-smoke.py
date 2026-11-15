#!/usr/bin/env python3
"""Drive a running e2e MyGo build against a task-created local fixture only."""
import argparse
import json
import pathlib
import re
import subprocess

parser = argparse.ArgumentParser()
parser.add_argument("root", type=pathlib.Path)
parser.add_argument("--session", default="product")
args = parser.parse_args()
root = args.root.resolve()
if not root.name.startswith("supergit-workflows.") or not (root / "remote.git").is_dir():
    raise SystemExit("requires a create-workflow-demo.sh fixture")
repo = root / "workspace"
automation = pathlib.Path("build/mygo-automation").resolve()


def auto(*items):
    result = subprocess.run([str(automation), *items, "--session", args.session], capture_output=True, text=True)
    if result.returncode:
        raise RuntimeError(f"automation {items}: {result.stdout}\n{result.stderr}")
    return result.stdout


def controls():
    text = auto("snapshot", "-i")
    entries = []
    for line in text.splitlines():
        match = re.match(r'\s*(@e\d+) \[(\w+)\] ("(?:[^"\\]|\\.)*")(.*)', line)
        if match:
            entries.append((match[1], match[2], json.loads(match[3]), match[4]))
    return entries


def ref(label, role="button", nth=0, prefix=False):
    found = [e for e in controls() if e[1] == role and (e[2].startswith(label) if prefix else e[2] == label) and "disabled" not in e[3]]
    if len(found) <= nth:
        raise RuntimeError(f"missing enabled {role} {label!r}: {controls()}")
    return found[nth][0]


def press(label, **kwargs):
    auto("press", ref(label, **kwargs), "--settle")


def fill(label, value):
    auto("fill", ref(label, role="textField"), value, "--settle")


def route(key):
    auto("press", "key=" + key, "--settle")


def wait(text):
    auto("wait", "--text", text, "--timeout", "20s", "--settle")


def idle():
    wait("就绪")


def execute():
    press("确认执行")
    idle()


def git(*items, directory=repo):
    return subprocess.check_output(["git", "-C", str(directory), *items], text=True)


def screenshot(name):
    auto("screenshot", str(pathlib.Path("build", name).resolve()))


remote = git("remote", "get-url", "origin").strip()
if pathlib.Path(remote).resolve() != root / "remote.git":
    raise SystemExit("fixture remote must remain local")
idle()
route("stash")
if not git("stash", "list").strip():
    press("保存 Stash…")
    fill("Stash 说明", "Native workflow draft")
    press("检查操作")
    execute()
assert not git("status", "--porcelain").strip(), "stash did not clean worktree"
press("stash@{0}", prefix=True)
wait("draft kept by Stash")
screenshot("product-stash-preview.png")
print("PASS native stash save and diff", flush=True)

route("branches")
press("创建分支…")
fill("分支名称", "native/test")
press("检查操作")
execute()
assert git("branch", "--show-current").strip() == "native/test"
screenshot("product-branches-preview.png")
print("PASS native create and switch branch", flush=True)

route("remotes")
press("Push…")
fill("远端目标分支", "native/published")
press("检查操作")
wait("refs/heads/native/published")
screenshot("product-push-confirmation.png")
execute()
assert git("rev-parse", "native/test").strip() == git("rev-parse", "refs/heads/native/published", directory=root / "remote.git").strip()
press("Fetch")
execute()
press("Pull（仅快进）")
execute()
screenshot("product-remotes-preview.png")
print("PASS native explicit push, fetch and ff-only pull", flush=True)

route("branches")
press("Merge", nth=0)
execute()
wait("进行中：merge")
route("conflicts")
press("conflict.txt")
wait("最终结果")
screenshot("product-conflict-preview.png")
fill("冲突最终结果", "main and feature combined\n")
press("保存并标记解决…")
execute()
wait("没有未解决的冲突")
press("继续操作")
execute()
assert len(git("show", "-s", "--format=%P", "HEAD").split()) == 2
assert not (repo / ".git" / "MERGE_HEAD").exists()
assert (repo / "conflict.txt").read_text() == "main and feature combined\n"
print("PASS native merge conflict edit, resolve and continue", flush=True)

route("worktrees")
press("新建 Worktree…")
fill("新目录路径", str(root / "review"))
fill("新分支名称", "native/review")
press("检查操作")
execute()
assert (root / "review" / ".git").is_file()
press("移除")
execute()
assert not (root / "review").exists()
print("PASS native worktree create and safe removal", flush=True)

route("stash")
press("应用")
execute()
assert (repo / "draft.txt").read_text() == "draft kept by Stash\n"
assert git("stash", "list").strip()
print("PASS native stash apply preserves entry", flush=True)

route("branches")
press("切换", nth=0)
execute()
wait("工作区或暂存区有改动")
assert git("branch", "--show-current").strip() == "native/test"
assert (repo / "draft.txt").read_text() == "draft kept by Stash\n"
print("PASS native dirty checkout refusal preserves files", flush=True)

hook = repo / ".git" / "hooks" / "pre-push"
assert not hook.exists(), "test must not overwrite an existing hook"
hook.write_text("#!/bin/sh\nsleep 30\n")
hook.chmod(0o755)
try:
    route("remotes")
    press("Push…")
    fill("远端目标分支", "native/cancelled")
    press("检查操作")
    press("确认执行")
    wait("取消任务")
    press("取消任务")
    idle()
    assert not git("for-each-ref", "--format=%(refname)", "refs/heads/native/cancelled", directory=root / "remote.git").strip()
finally:
    hook.unlink()
print("PASS native network cancellation releases write guard", flush=True)

route("history")
fill("搜索已加载提交", "Main changes")
wait("Main changes")
screenshot("product-history-search.png")
fill("搜索已加载提交", "")
route("repositories")
press("克隆仓库…")
fill("仓库地址", str(root / "remote.git"))
fill("新目录路径", str(root / "clone"))
press("检查操作")
execute()
wait("已打开")
assert (root / "clone" / ".git").is_dir()
print("PASS native clone local bare remote", flush=True)

route("repositories")
press("初始化仓库…")
fill("新目录路径", str(root / "initialized"))
press("检查操作")
execute()
wait("已打开")
assert (root / "initialized" / ".git").is_dir()
print("PASS native init new repository", flush=True)
screenshot("product-initialized-preview.png")
