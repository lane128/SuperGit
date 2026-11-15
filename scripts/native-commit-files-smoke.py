#!/usr/bin/env python3
"""Navigate immutable commit files with native keys in an isolated fixture."""
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


def press(label):
    ref = next(e[0] for e in controls() if e[1] == "button" and e[2] == label and "disabled" not in e[3])
    auto("press", ref, "--settle")


def key(value):
    auto("key", value, "--settle")


def wait(text):
    auto("wait", "--text", text, "--timeout", "25s", "--settle")


def list_focused(name):
    inside = False
    for line in auto("snapshot", "-i").splitlines():
        if "[list]" in line and "key=" + name in line.split():
            inside = True
        elif inside and not line.startswith("  "):
            break
        if inside and "focused" in line.split():
            return True
    return False


def screenshot(name):
    auto("pointer", "move", "20", "40", "--settle")
    auto("screenshot", str(pathlib.Path("build", name).resolve()))


assert pathlib.Path(git("remote", "get-url", "origin").strip()).resolve() == root / "remote.git"
wait("就绪")
paths = [f"history-files/{n:03d}.txt" for n in range(64)]
(repo / "history-files").mkdir()
for i, path in enumerate(paths):
    (repo / path).write_text(f"COMMIT BASE {i:03d}\n")
git("add", "--all")
git("commit", "-m", "Commit files base fixture")
base_sha = git("rev-parse", "HEAD").strip()
for i, path in enumerate(paths):
    (repo / path).write_text(f"COMMIT CONTENT {i:03d}\n")
message_body = "MESSAGE BODY START：完整说明\n\n" + "\n".join(f"  detail line {i:02d}: preserve indentation" for i in range(24)) + "\n\nRefs: fixture-message-42\nSigned-off-by: Test Developer <test@example.invalid>"
git("commit", "-am", "Commit files keyboard fixture", "-m", message_body)
large_sha = git("rev-parse", "HEAD").strip()
git("commit", "--allow-empty", "-m", "Commit files empty fixture")
empty_sha = git("rev-parse", "HEAD").strip()
# Keep real workspace and Index changes to prove historical browsing cannot write them.
(repo / paths[0]).write_text("INDEX DRAFT\n")
git("add", "--", paths[0])
(repo / paths[1]).write_text("WORKSPACE DRAFT\n")
head = git("rev-parse", "HEAD")
index = git("ls-files", "--stage")
working = {p: (repo / p).read_bytes() for p in paths}
key("CmdOrCtrl+R")
wait("就绪")
key("CmdOrCtrl+2")
assert any(e[1] == "button" and e[2] == "已到历史末尾" and "disabled" in e[3] for e in controls()), "small history did not report exhaustion"
assert not any(e[2] == "继续加载 300 条" for e in controls()), "exhausted history offered another page"
press("Commit files keyboard fixture")
wait(large_sha)
wait("+COMMIT CONTENT 000")
wait("MESSAGE BODY START：完整说明")
assert any(e[1] == "text" and "  detail line 23" in e[2] and "Refs: fixture-message-42" in e[2] and "Signed-off-by:" in e[2] for e in controls()), "full message body/trailers were lost"
auto("wheel", "1100", "170", "0", "1500", "--precise", "--settle")
screenshot("commit-message-full-body.png")
assert any(e[1] == "button" and e[2] == "M   " + paths[0] and "offscreen" not in e[3] for e in controls()), "long message displaced the file viewport"
print("PASS native full commit message preserves Unicode, indentation and trailers in bounded metadata scroll", flush=True)
key("CmdOrCtrl+Right")
assert list_focused("history-files"), "history-to-files focus transfer failed"
key("Home")
wait("+COMMIT CONTENT 000")
key("Down")
wait("+COMMIT CONTENT 001")
key("End")
wait("+COMMIT CONTENT 063")
assert any(e[1] == "button" and e[2] == "M   history-files/063.txt" and "offscreen" not in e[3] for e in controls()), "End did not reveal final file"
screenshot("commit-files-keyboard-end.png")
key("Up")
wait("+COMMIT CONTENT 062")
key("Enter")
wait("+COMMIT CONTENT 062")
print("PASS native commit-file arrows and Home/End select exact Diff and scroll virtual rows", flush=True)

key("CmdOrCtrl+Left")
assert list_focused("history-commits"), "files-to-history focus transfer failed"
key("CmdOrCtrl+Right")
assert list_focused("history-files")
wait("+COMMIT CONTENT 062")
for value in ("Home", "End", "Home", "Down", "Down", "End", "Up"):
    key(value)
wait("+COMMIT CONTENT 062")
key("CmdOrCtrl+Enter")
assert git("rev-parse", "HEAD") == head and git("ls-files", "--stage") == index
assert all((repo / p).read_bytes() == value for p, value in working.items())
print("PASS native commit-file focus roundtrip and rapid reads preserve selected Diff and repository data", flush=True)

# A repository snapshot does not replace this immutable commit-file list.
git("branch", "keyboard/commit-files-refresh")
key("CmdOrCtrl+R")
wait("就绪")
assert list_focused("history-files"), "snapshot refresh stole commit-file focus"
wait("MESSAGE BODY START：完整说明")
wait("+COMMIT CONTENT 062")
key("Up")
wait("+COMMIT CONTENT 061")
screenshot("commit-files-keyboard-refreshed.png")
print("PASS native commit-file path and focus survive repository snapshot refresh", flush=True)

# Replace a full list with an empty one, then another list occupying row zero.
key("CmdOrCtrl+Left")
press("Commit files empty fixture")
wait(empty_sha)
wait("从文件列表选择一项")
key("CmdOrCtrl+Right")
assert list_focused("history-files")
for value in ("Down", "Home", "End", "Enter", "Up"):
    key(value)
assert empty_sha in auto("snapshot")
assert not any(e[1] == "button" and e[2].startswith("M   history-files/") for e in controls())
screenshot("commit-files-keyboard-empty.png")
key("CmdOrCtrl+Left")
press("Commit files base fixture")
# The shared fixture's draft sorts before the numbered historical files.
wait("+draft kept by Stash")
key("CmdOrCtrl+Right")
key("Down")
wait("+COMMIT BASE 000")
key("Down")
wait("+COMMIT BASE 001")
assert list_focused("history-files")
assert git("rev-parse", "HEAD") == head and git("ls-files", "--stage") == index
assert all((repo / p).read_bytes() == value for p, value in working.items())
print("PASS native commit replacement resets file identity; empty-list keys are inert and replacement cursor works", flush=True)

# Rapid commit replacement cannot attach an old body to the final empty commit.
key("CmdOrCtrl+Left")
for subject in ("Commit files keyboard fixture", "Commit files empty fixture", "Commit files keyboard fixture", "Commit files empty fixture"):
    press(subject)
wait(empty_sha)
wait("从文件列表选择一项")
assert not any(e[1] == "text" and "MESSAGE BODY START" in e[2] for e in controls()), "old commit body survived selection replacement"
press("Commit files keyboard fixture")
wait(large_sha)
wait("MESSAGE BODY START：完整说明")
wait("+COMMIT CONTENT 000")
assert git("rev-parse", "HEAD") == head and git("ls-files", "--stage") == index
print("PASS native rapid commit switching and file/refresh reads retain only the selected immutable message", flush=True)

# Publish a local fixture ref without changing HEAD, Index or working files.
# The oversized message must fail independently of the valid file Diff.
message_file = root / "oversized-message.txt"
message_file.write_text("Commit message oversized fixture\n\n" + "x" * (1 << 20))
oversized_sha = git("commit-tree", git("rev-parse", "HEAD^{tree}").strip(), "-p", base_sha, "-F", str(message_file)).strip()
git("update-ref", "refs/heads/message/oversized", oversized_sha)
key("CmdOrCtrl+R")
wait("就绪")
press("Commit message oversized fixture")
wait(oversized_sha)
wait("提交说明超过 1 MiB 显示上限")
wait("+COMMIT CONTENT 000")
auto("wheel", "1100", "170", "0", "1200", "--precise", "--settle")
screenshot("commit-message-read-error.png")
press("重试读取提交说明")
wait("提交说明超过 1 MiB 显示上限")
assert oversized_sha in auto("snapshot"), "retry changed message target"
press("Commit files keyboard fixture")
wait(large_sha)
wait("MESSAGE BODY START：完整说明")
wait("+COMMIT CONTENT 000")
assert not any(e[1] == "text" and e[2].startswith("提交说明读取失败：") for e in controls()), "old message error leaked to another commit"
assert git("rev-parse", "HEAD") == head and git("ls-files", "--stage") == index
assert all((repo / p).read_bytes() == value for p, value in working.items())
print("PASS native oversized message failure and current-SHA retry leave file Diff and repository data usable", flush=True)


# Missing objects model a genuinely unavailable historical preview. Restore only
# newly created fixture objects; HEAD, Index and working files remain untouched.
def git_input(data, *items):
    return subprocess.check_output(["git", "-C", str(repo), *items], input=data).decode().strip()


def fixture_commit(tree, subject):
    raw = (f"tree {tree}\nauthor Preview Developer <preview@example.invalid> 1794390000 +0000\n"
           f"committer Preview Developer <preview@example.invalid> 1794390000 +0000\n\n{subject}\n").encode()
    return git_input(raw, "hash-object", "-t", "commit", "-w", "--stdin")


recovered_content = b"RECOVERED HISTORICAL FILE LIST\n"
blob = git_input(recovered_content, "hash-object", "-w", "--stdin")
raw_tree = b"100644 recovered.txt\x00" + bytes.fromhex(blob)
missing_tree = git_input(raw_tree, "hash-object", "-t", "tree", "--stdin")
files_error_sha = fixture_commit(missing_tree, "Commit files unavailable fixture")
git("update-ref", "refs/heads/preview/missing-tree", files_error_sha)
key("CmdOrCtrl+R")
wait("就绪")
press("Commit files unavailable fixture")
wait(files_error_sha)
wait("提交文件读取失败：")
assert any(e[1] == "text" and e[2].startswith("提交文件读取失败：") for e in controls())
assert not any(e[1] == "text" and e[2] == "此提交没有变更文件。" for e in controls()), "failed file list reported empty success"
screenshot("commit-files-read-error.png")
press("重试读取提交文件")
wait("提交文件读取失败：")
assert git_input(raw_tree, "hash-object", "-t", "tree", "-w", "--stdin") == missing_tree
press("重试读取提交文件")
wait("+RECOVERED HISTORICAL FILE LIST")
assert files_error_sha in auto("snapshot", "--full"), "file retry changed selected SHA"
assert not any(e[1] == "text" and e[2].startswith("提交文件读取失败：") for e in controls()), "file recovery kept old error"
print("PASS native unavailable commit file list reports failure and exact-SHA retry recovers restored tree", flush=True)

missing_content = b"RECOVERED HISTORICAL DIFF\n"
missing_blob = git_input(missing_content, "hash-object", "--stdin")
raw_tree = b"100644 missing.txt\x00" + bytes.fromhex(missing_blob) + b"100644 other.txt\x00" + bytes.fromhex(blob)
tree = git_input(raw_tree, "hash-object", "-t", "tree", "-w", "--stdin")
diff_error_sha = fixture_commit(tree, "Commit Diff unavailable fixture")
git("update-ref", "refs/heads/preview/missing-blob", diff_error_sha)
key("CmdOrCtrl+R")
wait("就绪")
press("Commit Diff unavailable fixture")
wait(diff_error_sha)
wait("提交 Diff 读取失败：")
assert any(e[1] == "text" and e[2].startswith("提交 Diff 读取失败：") for e in controls())
assert not any(e[1] == "text" and e[2].startswith("没有文本差异。") for e in controls()), "failed Diff reported empty success"
screenshot("commit-diff-read-error.png")
press("重试读取提交 Diff")
wait("提交 Diff 读取失败：")
press("A   other.txt")
wait("+RECOVERED HISTORICAL FILE LIST")
assert not any(e[1] == "text" and e[2].startswith("提交 Diff 读取失败：") for e in controls()), "file switch kept old error"
press("A   missing.txt")
wait("提交 Diff 读取失败：")
assert git_input(missing_content, "hash-object", "-w", "--stdin") == missing_blob
press("重试读取提交 Diff")
wait("+RECOVERED HISTORICAL DIFF")
assert diff_error_sha in auto("snapshot", "--full"), "Diff retry changed selected SHA"
screenshot("commit-preview-recovered.png")
press("Commit files keyboard fixture")
wait("+COMMIT CONTENT 000")
wait("MESSAGE BODY START：完整说明")
assert not any(e[1] == "text" and (e[2].startswith("提交文件读取失败：") or e[2].startswith("提交 Diff 读取失败：")) for e in controls()), "old preview error leaked to another commit"
assert git("rev-parse", "HEAD") == head and git("ls-files", "--stage") == index
assert all((repo / p).read_bytes() == value for p, value in working.items())
print("PASS native unavailable historical Diff retries exact path, clears on switch and recovers without writes", flush=True)
