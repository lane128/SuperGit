#!/usr/bin/env python3
"""Read-only Stash previews against a fresh, isolated local fixture."""
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


def git_input(data, *items):
    return subprocess.check_output(["git", "-C", str(repo), *items], input=data).decode().strip()


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


def press(label):
    ref = next(e[0] for e in controls() if e[1] == "button" and e[2] == label and "disabled" not in e[3])
    auto("press", ref, "--settle")


def key(value):
    auto("key", value, "--settle")


def wait(text):
    auto("wait", "--text", text, "--timeout", "25s", "--settle")


def stash_label(sha):
    for line in git("stash", "list", "--format=%H%x09%gd%x09%gs").splitlines():
        oid, ref, message = line.split("\t", 2)
        if oid == sha:
            return ref + "  " + message
    raise AssertionError("fixture Stash not found: " + sha)


def screenshot(name):
    auto("pointer", "move", "20", "40", "--settle")
    auto("screenshot", str(pathlib.Path("build", name).resolve()))


def assert_no_error():
    assert not any(e[1] == "text" and e[2].startswith("Stash 差异读取失败：") for e in controls()), "previous Stash error survived replacement"


def assert_unchanged(expected_stashes):
    assert git("rev-parse", "HEAD") == head
    assert git("ls-files", "--stage") == index
    assert git("stash", "list", "--format=%H %gd") == expected_stashes
    assert git("status", "--porcelain") == status
    assert (repo / "stash-preview.txt").read_bytes() == working


assert pathlib.Path(git("remote", "get-url", "origin").strip()).resolve() == root / "remote.git"
wait("就绪")
key("CmdOrCtrl+6")
wait("选择 Stash 查看差异。")
(repo / "stash-preview.txt").write_text("STASH BASE\n")
git("add", "--all")
git("commit", "-m", "Prepare stash preview fixture")
head = git("rev-parse", "HEAD")
index = git("ls-files", "--stage")
working = (repo / "stash-preview.txt").read_bytes()
status = git("status", "--porcelain")
assert status == ""
tree = git("rev-parse", "HEAD^{tree}").strip()
# A well-formed two-parent Stash with unchanged trees is a valid empty preview.
index_commit = git_input(b"Empty stash index\n", "commit-tree", tree, "-p", head.strip())
empty_sha = git_input(b"Empty preview fixture\n", "commit-tree", tree, "-p", head.strip(), "-p", index_commit)
git("stash", "store", "-m", "Empty preview fixture", empty_sha)
expected_stashes = git("stash", "list", "--format=%H %gd")
key("CmdOrCtrl+R")
wait("就绪")
press(stash_label(empty_sha))
wait("此 Stash 没有可显示的差异。")
assert_no_error()
assert_unchanged(expected_stashes)
screenshot("stash-preview-empty.png")
print("PASS native Stash distinguishes no selection from successful empty preview", flush=True)

(repo / "stash-preview.txt").write_text(("x" * 79 + "\n") * 70000)
git("stash", "push", "-m", "Oversized preview fixture")
large_sha = git("rev-parse", "refs/stash").strip()
key("CmdOrCtrl+R")
wait("就绪")
press(stash_label(large_sha))
wait("Stash 差异读取失败：")
wait("Git 输出超过 4 MiB")
wait("差异 · " + large_sha[:8])
assert any(e[1] == "text" and e[2].startswith("Stash 差异读取失败：") for e in controls()), "failure appeared only in footer"
assert not any(e[2] == "此 Stash 没有可显示的差异。" for e in controls())
expected_stashes = git("stash", "list", "--format=%H %gd")
press("重试读取 Stash 差异")
wait("Stash 差异读取失败：")
wait("差异 · " + large_sha[:8])
assert_unchanged(expected_stashes)
screenshot("stash-preview-read-error.png")
print("PASS native oversized Stash shows explicit failure and read-only retry at selected SHA", flush=True)

# A new entry moves the selected Stash from slot zero to slot one. Retry must
# still fail for the large SHA, rather than displaying the small new entry.
(repo / "stash-preview.txt").write_text("SMALL STASH CONTENT\n")
git("stash", "push", "-m", "Small preview fixture")
small_sha = git("rev-parse", "refs/stash").strip()
expected_stashes = git("stash", "list", "--format=%H %gd")
press("重试读取 Stash 差异")
wait("Stash 差异读取失败：")
wait("差异 · " + large_sha[:8])
assert not any(e[1] == "text" and "+SMALL STASH CONTENT" in e[2] for e in controls()), "retry followed shifted reflog position"
key("CmdOrCtrl+R")
wait("就绪")
for sha in (small_sha, large_sha, empty_sha, large_sha, small_sha):
    press(stash_label(sha))
wait("差异 · " + small_sha[:8])
wait("+SMALL STASH CONTENT")
assert_no_error()
assert_unchanged(expected_stashes)
screenshot("stash-preview-replaced.png")
print("PASS native Stash SHA retry survives reflog shift and rapid selection rejects old errors", flush=True)

press(stash_label(large_sha))
wait("Stash 差异读取失败：")
# Only this fresh fixture's generated entry is removed, explicitly by the test.
large_ref = next(line.split()[1] for line in expected_stashes.splitlines() if line.split()[0] == large_sha)
git("stash", "drop", large_ref)
expected_stashes = git("stash", "list", "--format=%H %gd")
press("重试读取 Stash 差异")
wait("Stash 已变化或不存在，请刷新列表")
wait("差异 · " + large_sha[:8])
key("CmdOrCtrl+R")
wait("就绪")
press("重试读取 Stash 差异")
wait("Stash 已变化或不存在，请刷新列表")
screenshot("stash-preview-removed-error.png")
press(stash_label(small_sha))
wait("+SMALL STASH CONTENT")
assert_no_error()
assert_unchanged(expected_stashes)
print("PASS native removed Stash retry reports missing SHA without selecting another reflog entry or writing", flush=True)
