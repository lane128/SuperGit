#!/usr/bin/env python3
"""Drive external changes while the native app refreshes itself; local fixture only."""
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


def git(*items, directory=repo):
    return subprocess.check_output(["git", "-C", str(directory), *items], text=True)


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


def ref(label, role="button"):
    found = [e for e in controls() if e[1] == role and e[2] == label and "disabled" not in e[3]]
    if not found:
        raise RuntimeError(f"missing enabled {role} {label!r}: {controls()}")
    return found[0][0]


def press(label, role="button"):
    auto("press", ref(label, role), "--settle")


def fill(label, value):
    auto("fill", ref(label, "textField"), value, "--settle")


def wait(text):
    auto("wait", "--text", text, "--timeout", "25s", "--settle")


def route(key):
    auto("press", "key=" + key, "--settle")


def screenshot(name):
    auto("screenshot", str(pathlib.Path("build", name).resolve()))


if pathlib.Path(git("remote", "get-url", "origin").strip()).resolve() != root / "remote.git":
    raise SystemExit("fixture remote must remain local")
wait("就绪")
path = repo / "watched.txt"
assert not path.exists(), "must not overwrite a fixture file"
path.write_text("base\n")
big = repo / "large-diff.txt"
assert not big.exists(), "must not overwrite a fixture file"
big.write_text("base\n")
source = root / "child-source"
source.mkdir(mode=0o700)
git("init", "-b", "main", directory=source)
git("config", "user.name", "Preview Developer", directory=source)
git("config", "user.email", "preview@example.invalid", directory=source)
(source / "child.txt").write_text("child contents\n")
git("add", "--", "child.txt", directory=source)
git("commit", "-m", "Child fixture", directory=source)
git("-c", "protocol.file.allow=always", "submodule", "add", str(source), "child")
git("add", "--", "watched.txt", "large-diff.txt")
git("commit", "-m", "External watch fixture")
path.write_text("first edit\n")
wait("watched.txt")
press("watched.txt")
wait("+first edit")
fill("提交说明", "draft stays while Git changes")
path.write_text("second edit\n")
wait("+second edit")
assert "draft stays while Git changes" in auto("snapshot"), "automatic refresh overwrote commit draft"
print("PASS native external edits and repeated dirty-file edits refresh Diff without a click", flush=True)

press("选择旧行 1", role="checkBox")
wait("已选 1 行")
git("branch", "watch/reference")
route("branches")
wait("watch/reference")
route("workspace")
wait("已选 1 行")
screenshot("watch-preserved-selection.png")
print("PASS native external ref refresh preserves unchanged Diff selection", flush=True)

git("add", "--", "watched.txt")
wait("HEAD → Index")
wait("已选 0 行")
assert "draft stays while Git changes" in auto("snapshot")
git("commit", "-m", "External watched commit")
route("history")
wait("External watched commit")
git("switch", "-c", "watch/switched")
wait("watch/switched")
route("workspace")
assert "draft stays while Git changes" in auto("snapshot"), "external commit/switch overwrote draft"
print("PASS native external stage, commit and checkout update groups, history and branch", flush=True)

route("modules")
for number in (1, 2):
    git("-c", "user.name=Preview Developer", "-c", "user.email=preview@example.invalid", "commit", "--allow-empty", "-m", f"Child advance {number}", directory=repo / "child")
    head = git("rev-parse", "HEAD", directory=repo / "child").strip()
    wait(head[:8])
screenshot("watch-second-submodule-head.png")
print("PASS native repeated child HEAD movements update module pointers", flush=True)

route("workspace")
big.write_text("x" * (5 << 20) + "\n")
wait("large-diff.txt")
press("large-diff.txt")
wait("Git 输出超过 4 MiB")
git("branch", "watch/after-large-diff")
route("branches")
wait("watch/after-large-diff")
big.write_text("base\n")
print("PASS native selected oversized Diff cannot suppress external branch updates", flush=True)

merge = subprocess.run(["git", "-C", str(repo), "merge", "--no-edit", "feature"], capture_output=True, text=True)
assert merge.returncode != 0 and "CONFLICT" in merge.stdout, merge.stdout + merge.stderr
wait("进行中：merge")
route("conflicts")
press("conflict.txt")
wait("最终结果")
fill("冲突最终结果", "unsaved combined result\n")
git("branch", "watch/during-conflict")
route("branches")
wait("watch/during-conflict")
route("conflicts")
wait("unsaved combined result")
screenshot("watch-preserved-conflict.png")
# An external edit must not be overwritten using the preserved stale snapshot.
conflicted = repo / "conflict.txt"
external = conflicted.read_text() + "external conflict edit\n"
conflicted.write_text(external)
press("保存并标记解决…")
press("确认执行")
wait("冲突或文件已在外部变化")
wait("就绪")
assert conflicted.read_text() == external, "stale conflict save overwrote external contents"
assert git("ls-files", "-u", "--", "conflict.txt").strip(), "stale conflict was marked resolved"
print("PASS native automatic refresh preserves conflict draft and stale save refuses overwrite", flush=True)
