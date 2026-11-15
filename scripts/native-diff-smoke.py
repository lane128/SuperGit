#!/usr/bin/env python3
"""Verify native line actions against real Index bytes in a disposable fixture."""
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
    result = subprocess.run([str(automation), *items, "--session", args.session], capture_output=True, text=True)
    if result.returncode:
        raise RuntimeError(f"automation {items}: {result.stdout}\n{result.stderr}")
    return result.stdout


def controls():
    entries = []
    for line in auto("snapshot", "-i").splitlines():
        match = re.match(r'\s*(@e\d+) \[(\w+)\] ("(?:[^"\\]|\\.)*")(.*)', line)
        if match:
            entries.append((match[1], match[2], json.loads(match[3]), match[4]))
    return entries


def press(label, role="button", nth=0):
    found = [e for e in controls() if e[1] == role and e[2] == label and "disabled" not in e[3]]
    if len(found) <= nth:
        raise RuntimeError(f"missing enabled {role} {label!r}: {controls()}")
    auto("press", found[nth][0], "--settle")


def wait(text):
    auto("wait", "--text", text, "--timeout", "20s", "--settle")


def screenshot(name):
    auto("screenshot", str(pathlib.Path("build", name).resolve()))


if pathlib.Path(git("remote", "get-url", "origin").strip()).resolve() != root / "remote.git":
    raise SystemExit("fixture remote must remain local")
path = repo / "lines.txt"
assert not path.exists(), "must not overwrite an existing fixture file"
base = "one\ntwo\nthree\nfour\n"
working = "ONE\nTWO\nthree\nfour\nfive\n"
path.write_text(base)
link = repo / "target-link"
assert not link.exists() and not link.is_symlink(), "must not overwrite a fixture symlink"
link.symlink_to("old-target")
git("add", "--", "lines.txt", "target-link")
git("commit", "-m", "Line selection fixture")
path.write_text(working)
link.unlink()
link.symlink_to("new-target")
wait("就绪")
press("刷新 ⌘R")
wait("就绪")
press("lines.txt")
wait("+TWO")
press("选择旧行 1", role="checkBox")
press("选择新行 1", role="checkBox")
wait("已选 2 行")
press("左右视图")
wait("已选 2 行")
screenshot("diff-split-selection.png")
press("暂存选中行")
wait("就绪")
wait("已选 0 行")
assert git("show", ":lines.txt") == "ONE\ntwo\nthree\nfour\n", "partial stage changed unselected lines"
assert path.read_text() == working, "stage changed working file"
print("PASS native unified selection survives split and stages only one replacement", flush=True)

# Stage a second pair from the remaining worktree Diff. Unstage only the first.
wait("+TWO")
press("选择旧行 2", role="checkBox")
press("选择新行 2", role="checkBox")
wait("已选 2 行")
press("暂存选中行")
wait("就绪")
assert git("show", ":lines.txt") == "ONE\nTWO\nthree\nfour\n"
press("lines.txt", nth=1)
wait("HEAD → Index")
wait("+ONE")
press("选择旧行 1", role="checkBox")
press("选择新行 1", role="checkBox")
wait("已选 2 行")
press("统一视图")
wait("已选 2 行")
screenshot("diff-unified-unstage.png")
press("取消暂存选中行")
wait("就绪")
wait("已选 0 行")
assert git("show", ":lines.txt") == "one\nTWO\nthree\nfour\n", "unstage changed adjacent staged content"
assert path.read_text() == working, "unstage changed working file"
print("PASS native line unstage preserves adjacent staged changes and working file", flush=True)

press("选择旧行 2", role="checkBox")
wait("已选 1 行")
press("刷新 ⌘R")
wait("就绪")
wait("已选 0 行")
press("选择旧行 2", role="checkBox")
wait("已选 1 行")
press("lines.txt", nth=0)
wait("Index → 工作区")
wait("已选 0 行")
press("选择新行 1", role="checkBox")
wait("已选 1 行")
press("draft.txt")
wait("draft kept by Stash")
press("lines.txt", nth=0)
wait("+ONE")
wait("已选 0 行")
print("PASS native refresh, file and group switch clear line selection", flush=True)

# External edits invalidate the exact Diff captured by the UI.
press("选择新行 1", role="checkBox")
wait("已选 1 行")
path.write_text(working + "external\n")
# Polling may clear the selection before the click; otherwise ApplyLines rejects
# the stale Diff. Both paths must leave Index and working content untouched.
buttons = [e for e in controls() if e[1] == "button" and e[2] == "暂存选中行" and "disabled" not in e[3]]
if buttons:
    auto("press", buttons[0][0], "--settle")
wait("+external")
wait("已选 0 行")
wait("就绪")
assert git("show", ":lines.txt") == "one\nTWO\nthree\nfour\n"
assert path.read_text() == working + "external\n"
print("PASS native stale Diff refuses Index write and refreshes actual state", flush=True)

# A history Diff is read-only even with a workspace line selected.
wait("+external")
press("选择新行 1", role="checkBox")
wait("已选 1 行")
auto("press", "key=history", "--settle")
wait("Line selection fixture")
press("Line selection fixture")
wait("A   lines.txt")
press("A   lines.txt")
wait("提交差异 · 只读")
wait("+one")
press("左右视图")
screenshot("diff-history-readonly.png")
assert not [e for e in controls() if e[1] == "checkBox" and e[2].startswith("选择")], "history exposes workspace selection"
assert not [e for e in controls() if e[2] in ("暂存选中行", "取消暂存选中行", "暂存片段", "取消暂存片段")], "history exposes write actions"
print("PASS native history split Diff remains read-only", flush=True)

auto("press", "key=workspace", "--settle")
press("target-link")
wait("+new-target")
wait("此差异仅支持文件级操作。")
assert not [e for e in controls() if e[1] == "checkBox" and e[2].startswith("选择")], "symlink exposes line controls"
assert not [e for e in controls() if e[2] in ("暂存选中行", "取消暂存选中行", "暂存片段", "取消暂存片段")], "symlink exposes partial write actions"
assert git("show", ":target-link") == "old-target"
assert link.readlink() == pathlib.Path("new-target")
print("PASS native existing symlink is limited to file-level staging", flush=True)
