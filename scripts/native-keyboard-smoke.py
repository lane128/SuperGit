#!/usr/bin/env python3
"""Drive real key events and Diff target changes in a fresh local fixture."""
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
    for line in auto("snapshot", "-i").splitlines():
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


def wait(text):
    auto("wait", "--text", text, "--timeout", "25s", "--settle")


def key(value):
    auto("key", value, "--settle")


def selected(page):
    return any(e[1] == "treeItem" and "key=" + page in e[3].split() and ({"checked", "selected"} & set(e[3].split())) for e in controls())


def screenshot(name):
    auto("screenshot", str(pathlib.Path("build", name).resolve()))


assert pathlib.Path(git("remote", "get-url", "origin").strip()).resolve() == root / "remote.git"
wait("就绪")
base = {}
for letter in ("A", "B"):
    path = repo / (letter.lower() + "-scroll.txt")
    assert not path.exists()
    base[letter] = "".join(f"{letter} base {n:04d}\n" for n in range(1, 601))
    path.write_text(base[letter])
commit_file = repo / "commit-target.txt"
assert not commit_file.exists()
commit_file.write_text("before keyboard commit\n")
git("add", "--", "a-scroll.txt", "b-scroll.txt", "commit-target.txt")
git("commit", "-m", "Keyboard scroll fixture")
for letter in ("A", "B"):
    (repo / (letter.lower() + "-scroll.txt")).write_text("".join(f"{letter} changed {n:04d}\n" for n in range(1, 601)))
commit_file.write_text("after keyboard commit\n")
git("add", "--", "commit-target.txt")
key("CmdOrCtrl+R")
wait("就绪")
auto("focus", ref("提交说明", "textField"), "--settle")
auto("type", "Keyboard draft", "--settle")
key("Enter")
auto("type", "details kept", "--settle")
head = git("rev-parse", "HEAD")
index = git("ls-files", "--stage")
assert "Keyboard draft\\ndetails kept" in auto("snapshot"), "plain Enter did not edit the draft"

for number, page in enumerate(("workspace", "history", "modules", "branches", "remotes", "stash", "conflicts", "repositories", "worktrees"), 1):
    key("CmdOrCtrl+" + str(number))
    assert selected(page), f"page shortcut {number} did not select {page}"
key("CmdOrCtrl+2")
key("CmdOrCtrl+F")
auto("type", "Main changes", "--settle")
assert any(e[1] == "textField" and e[2] == "搜索已加载提交" and 'value="Main changes"' in e[3] and "focused" in e[3] for e in controls()), "history search did not take keyboard focus"
assert "keyshortcuts=" in auto("snapshot", "--full"), "shortcut metadata missing"
fill("搜索已加载提交", "")
key("CmdOrCtrl+Enter")
wait("就绪")
assert git("rev-parse", "HEAD") == head and git("ls-files", "--stage") == index, "history shortcut committed hidden Index"
key("CmdOrCtrl+1")
assert "Keyboard draft\\ndetails kept" in auto("snapshot"), "page navigation lost commit draft"
print("PASS native page shortcuts, search focus and ordinary draft editing; history cannot commit", flush=True)

key("CmdOrCtrl+4")
press("创建分支…")
fill("分支名称", "keyboard/not-created")
for value in ("CmdOrCtrl+1", "CmdOrCtrl+F", "CmdOrCtrl+R", "CmdOrCtrl+Enter"):
    key(value)
assert any(e[2] == "分支名称" and 'value="keyboard/not-created"' in e[3] for e in controls()), "shortcut escaped branch form"
assert git("rev-parse", "HEAD") == head and git("ls-files", "--stage") == index
press("检查操作")
for value in ("CmdOrCtrl+1", "CmdOrCtrl+R", "CmdOrCtrl+Enter"):
    key(value)
assert any(e[2] == "确认执行" for e in controls()), "shortcut dismissed confirmation"
assert git("rev-parse", "HEAD") == head and git("ls-files", "--stage") == index, "confirmation shortcut wrote Index/HEAD"
screenshot("keyboard-protected-confirmation.png")
key("Escape")
assert selected("branches")
assert not git("branch", "--list", "keyboard/not-created").strip(), "Escape executed branch creation"
press("创建分支…")
key("Escape")
assert selected("branches"), "Escape did not cancel form"
print("PASS native action form and confirmation block shortcuts; Escape cancels without Git writes", flush=True)

key("CmdOrCtrl+1")
press("a-scroll.txt")
wait("-A base 0001")
auto("wheel", "1100", "600", "0", "1500", "--precise", "--settle")
scrolled = auto("snapshot")
assert '"-A base 0001"' not in scrolled, "workspace Diff did not scroll"
markers = re.findall(r'"(-A base \d{4})"', scrolled)
assert markers, scrolled
marker = markers[len(markers) // 2]
git("branch", "keyboard/preserve-scroll")
key("CmdOrCtrl+4")
wait("keyboard/preserve-scroll")
key("CmdOrCtrl+1")
wait(marker)
assert '"-A base 0001"' not in auto("snapshot"), "unchanged target lost scroll on automatic refresh"

key("CmdOrCtrl+2")
press("Keyboard scroll fixture")
wait("A   a-scroll.txt")
press("A   a-scroll.txt")
wait("+A base 0001")
auto("wheel", "1100", "650", "0", "1000", "--precise", "--settle")
assert '"+A base 0001"' not in auto("snapshot"), "history Diff did not scroll"
press("A   b-scroll.txt")
wait("+B base 0001")
key("CmdOrCtrl+1")
wait(marker)
assert '"-A base 0001"' not in auto("snapshot"), "history file selection reset workspace scroll"
print("PASS native unchanged target retains scroll; historical files reset only their own scroll", flush=True)

(repo / "a-scroll.txt").write_text(base["A"])
wait("-B base 0001")
assert '"-A base ' not in auto("snapshot"), "fallback kept previous file preview"
screenshot("keyboard-fallback-diff-top.png")
print("PASS native automatic file fallback resets Diff preview and starts at the top", flush=True)

# The explicit workspace shortcut commits only its Index, leaving both dirty files.
key("CmdOrCtrl+Enter")
wait("就绪")
assert git("rev-parse", "HEAD") != head, "explicit workspace shortcut did not commit"
assert git("log", "-1", "--format=%B").strip() == "Keyboard draft\ndetails kept"
assert git("diff", "--cached", "--name-only").strip() == ""
assert (repo / "b-scroll.txt").read_text().startswith("B changed 0001\n")
assert git("show", "HEAD:b-scroll.txt") == base["B"], "workspace commit included unstaged contents"
print("PASS native workspace shortcut explicitly commits Index only", flush=True)

merge = subprocess.run(["git", "-C", str(repo), "merge", "--no-edit", "feature"], capture_output=True, text=True)
assert merge.returncode != 0 and "CONFLICT" in merge.stdout
wait("进行中：merge")
key("CmdOrCtrl+7")
press("conflict.txt")
wait("最终结果")
fill("冲突最终结果", "unsaved keyboard conflict draft\n")
conflicted_bytes = (repo / "conflict.txt").read_bytes()
conflict_head = git("rev-parse", "HEAD")
key("CmdOrCtrl+Enter")
key("Escape")
assert "unsaved keyboard conflict draft" in auto("snapshot")
assert (repo / "conflict.txt").read_bytes() == conflicted_bytes and git("rev-parse", "HEAD") == conflict_head
screenshot("keyboard-preserved-conflict.png")
print("PASS native conflict editing ignores commit/cancel shortcuts and preserves unsaved text", flush=True)
