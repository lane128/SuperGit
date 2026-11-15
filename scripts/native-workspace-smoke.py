#!/usr/bin/env python3
"""Exercise grouped file selection using real keys in a fresh local fixture."""
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
    return next(e[0] for e in controls() if e[1] == role and e[2] == label and "disabled" not in e[3])


def key(value):
    auto("key", value, "--settle")


def wait(text):
    auto("wait", "--text", text, "--timeout", "25s", "--settle")


def screenshot(name):
    auto("pointer", "move", "20", "40", "--settle")
    auto("screenshot", str(pathlib.Path("build", name).resolve()))


def workspace_focused():
    # Native single selection may put accessibility focus on the chosen row.
    # Either the list itself or its descendant owns file navigation.
    inside = False
    for line in auto("snapshot", "-i").splitlines():
        if '[list] "工作区文件"' in line and "key=workspace-files" in line:
            inside = True
        elif inside and not line.startswith("  "):
            break
        if inside and "focused" in line.split():
            return True
    return False


def on_workspace():
    return any(e[1] == "treeItem" and "key=workspace" in e[3].split() and ({"checked", "selected"} & set(e[3].split())) for e in controls())


assert pathlib.Path(git("remote", "get-url", "origin").strip()).resolve() == root / "remote.git"
wait("就绪")
paths = ["a-dual.txt", *(f"m-{n:03d}.txt" for n in range(60)), "z-index.txt"]
for path in paths:
    assert not (repo / path).exists()
    (repo / path).write_text("BASE " + path + "\n")
git("add", "--all")
git("commit", "-m", "Workspace keyboard fixture")
(repo / "a-dual.txt").write_text("DUAL INDEX\n")
(repo / "z-index.txt").write_text("LAST INDEX\n")
git("add", "--", "a-dual.txt", "z-index.txt")
(repo / "a-dual.txt").write_text("DUAL WORKTREE\n")
for path in paths[1:-1]:
    (repo / path).write_text("WORKTREE " + path + "\n")
head = git("rev-parse", "HEAD")
index = git("ls-files", "--stage")
working = {path: (repo / path).read_bytes() for path in paths}
key("CmdOrCtrl+R")
wait("就绪")
key("CmdOrCtrl+1")
assert workspace_focused(), "Cmd+1 did not focus workspace list"
key("Home")
wait("+DUAL WORKTREE")
wait("Index → 工作区")
key("Down")
wait("+WORKTREE m-000.txt")
key("End")
wait("+LAST INDEX")
wait("HEAD → Index")
assert any(e[2] == "z-index.txt" and e[1] == "button" for e in controls()), "End did not scroll the selected row into view"
screenshot("workspace-keyboard-end.png")
key("Up")
wait("+DUAL INDEX")
key("Up")
wait("+WORKTREE m-059.txt")
wait("Index → 工作区")
key("Down")
wait("+DUAL INDEX")
key("Enter")
wait("+DUAL INDEX")
assert git("rev-parse", "HEAD") == head and git("ls-files", "--stage") == index
assert all((repo / path).read_bytes() == value for path, value in working.items())
print("PASS native workspace arrows skip headers, cross groups and show exact dual-path Diff; Home/End scroll", flush=True)

# Commit editor owns ordinary keys, leaving the selected Index Diff untouched.
auto("focus", ref("提交说明", "textField"), "--settle")
auto("type", "Workspace keyboard draft", "--settle")
key("Enter")
auto("type", "body retained", "--settle")
for value in ("Up", "Down", "Home", "End"):
    key(value)
assert any(e[1] == "textField" and e[2] == "提交说明" and "focused" in e[3] for e in controls())
assert "Workspace keyboard draft\\nbody retained" in auto("snapshot")
assert '"+DUAL INDEX"' in auto("snapshot"), "editor arrows moved file selection"
assert git("rev-parse", "HEAD") == head and git("ls-files", "--stage") == index
key("CmdOrCtrl+1")
assert workspace_focused()
print("PASS native workspace draft editing keeps focus and leaves file selection, Index and HEAD intact", flush=True)

# Add an earlier status row while the Index instance of the dual-path file is selected.
(repo / "0-insert.txt").write_text("INSERTED FILE\n")
key("CmdOrCtrl+R")
wait("就绪")
wait("+DUAL INDEX")
assert workspace_focused(), "snapshot refresh dropped workspace focus"
key("Down")
wait("+LAST INDEX")
key("Up")
wait("+DUAL INDEX")
assert git("ls-files", "--stage") == index
print("PASS native workspace refresh preserves path and staged-group identity after insertion", flush=True)

# Remove only the Index instance: reconciliation must follow the same path in worktree.
git("restore", "--staged", "--", "a-dual.txt")
index_after_move = git("ls-files", "--stage")
key("CmdOrCtrl+R")
wait("就绪")
wait("+DUAL WORKTREE")
wait("Index → 工作区")
assert workspace_focused()
key("Down")
wait("+WORKTREE m-000.txt")
key("Up")
wait("+DUAL WORKTREE")
git("restore", "--", "a-dual.txt")
key("CmdOrCtrl+R")
wait("就绪")
# Git emits tracked changes before untracked entries; fallback is m-000.
wait("+WORKTREE m-000.txt")
assert workspace_focused()
assert any(e[1] == "listItem" and e[2] == "m-000.txt" and ({"checked", "selected"} & set(e[3].split())) for e in controls()), "same-row fallback lost native selection"
key("Down")
wait("+WORKTREE m-001.txt")
assert git("ls-files", "--stage") == index_after_move
assert "Workspace keyboard draft\\nbody retained" in auto("snapshot")
screenshot("workspace-keyboard-reconciled.png")
print("PASS native workspace cursor follows group movement and vanished-file fallback with draft preserved", flush=True)

# Make the fixture clean, then create its known merge conflict outside the app.
git("add", "--all")
git("commit", "-m", "Prepare keyboard conflict fixture")
merge = subprocess.run(["git", "-C", str(repo), "merge", "--no-edit", "feature"], capture_output=True, text=True)
assert merge.returncode != 0 and "CONFLICT" in merge.stdout
key("CmdOrCtrl+R")
wait("就绪")
key("CmdOrCtrl+1")
key("Home")
wait("diff --cc conflict.txt")
assert on_workspace(), "arrow browsing unexpectedly entered conflict editor"
conflict_head = git("rev-parse", "HEAD")
conflict_index = git("ls-files", "--stage")
conflict_bytes = (repo / "conflict.txt").read_bytes()
key("Enter")
wait("最终结果")
assert any(e[2] == "冲突最终结果" for e in controls()), "Enter did not activate conflict editor"
assert git("rev-parse", "HEAD") == conflict_head and git("ls-files", "--stage") == conflict_index
assert (repo / "conflict.txt").read_bytes() == conflict_bytes
screenshot("workspace-keyboard-conflict.png")
print("PASS native workspace conflict browsing is read-only; Enter explicitly opens editor without writes", flush=True)

# Return to a clean fixture and verify all navigation is inert when only headers remain.
git("merge", "--abort")
key("CmdOrCtrl+1")
key("CmdOrCtrl+R")
wait("就绪")
wait("0 个文件有改动")
wait("从文件列表选择一项")
empty_head = git("rev-parse", "HEAD")
empty_index = git("ls-files", "--stage")
for value in ("Down", "Home", "End", "Enter", "Up"):
    key(value)
assert on_workspace() and workspace_focused()
assert git("rev-parse", "HEAD") == empty_head and git("ls-files", "--stage") == empty_index
assert "Workspace keyboard draft\\nbody retained" in auto("snapshot")
screenshot("workspace-keyboard-empty.png")
print("PASS native empty workspace ignores navigation and activation while retaining draft", flush=True)

# Start a real no-change merge: only the operation bar changes, while the
# empty file list/model selection stays the same. Its identity must survive.
candidate = git("commit-tree", git("rev-parse", "HEAD^{tree}").strip(), "-p", empty_head.strip(), "-m", "Empty merge focus fixture").strip()
git("merge", "--no-edit", "--no-commit", "--no-ff", candidate)
key("CmdOrCtrl+R")
wait("就绪")
wait("进行中：merge")
screenshot("workspace-focus-integration-bar.png")
assert workspace_focused(), "adding operation bar changed workspace list identity and lost keyboard focus"
assert git("ls-files", "--stage") == empty_index
assert "Workspace keyboard draft\\nbody retained" in auto("snapshot")
git("merge", "--abort")
key("CmdOrCtrl+R")
wait("就绪")
assert workspace_focused(), "removing operation bar lost workspace list focus"
assert git("rev-parse", "HEAD") == empty_head and git("ls-files", "--stage") == empty_index
# The same identity guarantee must preserve editor focus, without moving it
# into the file list merely because the operation changes.
auto("focus", ref("提交说明", "textField"), "--settle")
for operation in ("start", "abort"):
    if operation == "start":
        git("merge", "--no-edit", "--no-commit", "--no-ff", candidate)
    else:
        git("merge", "--abort")
    key("CmdOrCtrl+R")
    wait("就绪")
    if operation == "start":
        wait("进行中：merge")
    assert any(e[1] == "textField" and e[2] == "提交说明" and "focused" in e[3].split() for e in controls()), "operation bar change lost editor focus"
    assert "Workspace keyboard draft\\nbody retained" in auto("snapshot")
    assert git("rev-parse", "HEAD") == empty_head and git("ls-files", "--stage") == empty_index
key("CmdOrCtrl+1")
assert workspace_focused()
print("PASS native operation bar appearance and removal preserve workspace list/editor focus and draft", flush=True)

# A target disappearing while the commit editor owns focus must not steal it.
(repo / "a-dual.txt").write_text("TEMPORARY FOCUS CHANGE\n")
key("CmdOrCtrl+R")
wait("就绪")
key("CmdOrCtrl+1")
key("Home")
wait("+TEMPORARY FOCUS CHANGE")
auto("focus", ref("提交说明", "textField"), "--settle")
git("restore", "--", "a-dual.txt")
key("CmdOrCtrl+R")
wait("就绪")
wait("0 个文件有改动")
assert any(e[1] == "textField" and e[2] == "提交说明" and "focused" in e[3].split() for e in controls()), "fallback stole focus from commit editor"
assert git("rev-parse", "HEAD") == empty_head and git("ls-files", "--stage") == empty_index
assert "Workspace keyboard draft\\nbody retained" in auto("snapshot")
print("PASS native file disappearance preserves commit-editor focus instead of stealing it", flush=True)


def assert_no_preview_error():
    assert not any('[text] "工作区 Diff 读取失败：' in line for line in auto("snapshot", "--full").splitlines()), "old workspace preview error remained visible"


def assert_preview_error():
    assert any('[text] "工作区 Diff 读取失败：' in line for line in auto("snapshot", "--full").splitlines()), "failure was only shown in footer"
    assert not any(e[1] == "button" and (e[2] == "暂存选中行" or e[2] == "取消暂存选中行") for e in controls()), "failed preview exposed line actions"


# Oversized tracked text exercises the real bounded Git read, without changing
# the Index during UI browsing/retry. The fixture itself supplies recovery edits.
large_text = ("x" * 79 + "\n") * 70000
(repo / "a-dual.txt").write_text(large_text)
key("CmdOrCtrl+R")
wait("就绪")
key("CmdOrCtrl+1")
key("Home")
wait("工作区 Diff 读取失败：")
wait("Index → 工作区")
assert_preview_error()
screenshot("workspace-diff-read-error.png")
failed_index = git("ls-files", "--stage")
failed_bytes = (repo / "a-dual.txt").read_bytes()
auto("press", ref("重试读取工作区 Diff"), "--settle")
wait("工作区 Diff 读取失败：")
assert git("rev-parse", "HEAD") == empty_head and git("ls-files", "--stage") == failed_index
assert (repo / "a-dual.txt").read_bytes() == failed_bytes
(repo / "a-dual.txt").write_text("RECOVERED WORKTREE DIFF\n")
key("CmdOrCtrl+1")
key("Enter")
wait("+RECOVERED WORKTREE DIFF")
wait("Index → 工作区")
assert_no_preview_error()
assert git("rev-parse", "HEAD") == empty_head and git("ls-files", "--stage") == failed_index
assert (repo / "a-dual.txt").read_text() == "RECOVERED WORKTREE DIFF\n"
print("PASS native workspace Diff failure is explicit and read-only retry recovers the current worktree path", flush=True)

# The same raw path now has an oversized Index version and a small worktree
# version: errors and retries must follow the selected group exactly.
(repo / "a-dual.txt").write_text(large_text)
git("add", "--", "a-dual.txt")
(repo / "a-dual.txt").write_text(large_text + "WORKTREE VERSION AFTER INDEX ERROR\n")
key("CmdOrCtrl+R")
wait("就绪")
key("CmdOrCtrl+1")
key("End")
wait("工作区 Diff 读取失败：")
wait("HEAD → Index")
assert_preview_error()
failed_index = git("ls-files", "--stage")
failed_bytes = (repo / "a-dual.txt").read_bytes()
auto("press", ref("重试读取工作区 Diff"), "--settle")
wait("工作区 Diff 读取失败：")
wait("HEAD → Index")
screenshot("workspace-index-diff-read-error.png")
assert git("rev-parse", "HEAD") == empty_head and git("ls-files", "--stage") == failed_index
assert (repo / "a-dual.txt").read_bytes() == failed_bytes
key("CmdOrCtrl+1")
key("Home")
wait("+WORKTREE VERSION AFTER INDEX ERROR")
wait("Index → 工作区")
assert_no_preview_error()
key("End")
wait("工作区 Diff 读取失败：")
(repo / "a-dual.txt").write_text("RECOVERED INDEX DIFF\n")
git("add", "--", "a-dual.txt")
(repo / "a-dual.txt").write_text("WORKTREE VERSION AFTER INDEX ERROR\n")
recovered_index = git("ls-files", "--stage")
key("CmdOrCtrl+1")
key("End")
key("Enter")
wait("+RECOVERED INDEX DIFF")
wait("HEAD → Index")
assert_no_preview_error()
assert git("rev-parse", "HEAD") == empty_head and git("ls-files", "--stage") == recovered_index
assert (repo / "a-dual.txt").read_text() == "WORKTREE VERSION AFTER INDEX ERROR\n"
print("PASS native same-path Index failure retries exact group and switching to worktree clears old error", flush=True)

# A second edit to an already-dirty worktree can fail in the probe itself.
# It must clear stale lines, then recover automatically without stealing focus.
key("Home")
wait("+WORKTREE VERSION AFTER INDEX ERROR")
auto("focus", ref("提交说明", "textField"), "--settle")
(repo / "a-dual.txt").write_text(large_text)
wait("工作区 Diff 读取失败：")
assert_preview_error()
(repo / "a-dual.txt").write_text("AUTOMATIC DIFF RECOVERY\n")
wait("+AUTOMATIC DIFF RECOVERY")
assert_no_preview_error()
assert any(e[1] == "textField" and e[2] == "提交说明" and "focused" in e[3].split() for e in controls()), "automatic failure/recovery stole editor focus"
assert "Workspace keyboard draft\\nbody retained" in auto("snapshot")
assert git("rev-parse", "HEAD") == empty_head and git("ls-files", "--stage") == recovered_index
assert (repo / "a-dual.txt").read_text() == "AUTOMATIC DIFF RECOVERY\n"
screenshot("workspace-diff-auto-recovered.png")
print("PASS native repeated dirty edits show automatic Diff failure then recover without writes or focus loss", flush=True)
