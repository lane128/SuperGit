#!/usr/bin/env python3
"""Exercise loaded history, virtual scrolling and SHA selection on a large fixture."""
import argparse
import json
import pathlib
import re
import subprocess
import time

parser = argparse.ArgumentParser()
parser.add_argument("root", type=pathlib.Path)
parser.add_argument("--session", required=True)
args = parser.parse_args()
root = args.root.resolve()
if not root.name.startswith("supergit-history.") or not (root / "fixture.json").is_file():
    raise SystemExit("requires a fresh create-history-demo.py fixture")
repo = root / "workspace"
automation = pathlib.Path("build/mygo-automation").resolve()
measurements = {}


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


def ref(label, role="button"):
    return next(e[0] for e in controls() if e[1] == role and e[2] == label and "disabled" not in e[3])


def press(label):
    auto("press", ref(label), "--settle")


def fill(value):
    auto("fill", ref("搜索已加载提交", "textField"), value, "--settle")


def wait(text):
    auto("wait", "--text", text, "--timeout", "30s", "--settle")


def screenshot(name):
    auto("pointer", "move", "20", "40", "--settle")
    auto("screenshot", str(pathlib.Path("build", name).resolve()))


assert int(git("rev-list", "--all", "--count")) == 12024
assert len(git("ls-files", "-z").split("\0")) - 1 == 10001
wait("就绪")
auto("press", "key=history", "--settle")
wait("最近 300 条提交")
fill("Perf commit 11050")
assert not any(e[1] == "button" and e[2] == "Perf commit 11050" for e in controls()), "search escaped the loaded history"
fill("")
started = time.monotonic()
for count in (600, 900, 1200, 1500, 1800, 2100, 2400):
    press("继续加载 300 条")
    wait(f"最近 {count} 条提交")
    wait("就绪")
measurements["load_300_to_2400_seconds"] = time.monotonic() - started
assert any(e[2] == "继续加载 300 条" and "disabled" not in e[3] for e in controls()), "history continuation was capped"
print("PASS native history pages from 300 through 2,400 on 12,024-commit/10,001-file repository", flush=True)

before = {e[2] for e in controls() if e[1] == "button" and e[2].startswith("Perf commit")}
started = time.monotonic()
for _ in range(4):
    auto("wheel", "450", "600", "0", "1200", "--precise", "--settle")
after = {e[2] for e in controls() if e[1] == "button" and e[2].startswith("Perf commit")}
assert before and after and before != after, "virtual list did not scroll"
measurements["four_scroll_commands_seconds"] = time.monotonic() - started
screenshot("history-large-scrolled.png")
print("PASS native virtual history scroll changes visible rows", flush=True)

# A commit beyond the previous 2,000-row cutoff must resolve to its exact SHA/Diff.
deep_sha = git("log", "--all", "--format=%H", "--grep=^Perf commit 10000$", "-1").strip()
fill("Perf commit 10000")
press("Perf commit 10000")
wait(deep_sha)
wait("M   files/00000.txt")
press("M   files/00000.txt")
wait("Version 10000")
screenshot("history-paged-deep-selection.png")
assert any(e[2] == "继续加载 300 条" and "disabled" not in e[3] for e in controls())
print("PASS native history beyond 2,000 rows selects exact SHA and read-only Diff", flush=True)

expected = git("log", "--all", "--format=%H", "--grep=^Perf commit 11050$", "-1").strip()
started = time.monotonic()
fill("  PERF commit 11050  ")
press("Perf commit 11050")
wait(expected)
wait("M   files/00000.txt")
press("M   files/00000.txt")
wait("Version 11050")
measurements["filtered_select_and_diff_seconds"] = time.monotonic() - started
screenshot("history-filtered-selection.png")
print("PASS native filtered row selects the exact original SHA and file Diff", flush=True)

# Shift every original index with an external commit, leaving the search active.
git("-c", "user.name=Preview Developer", "-c", "user.email=preview@example.invalid", "commit", "--allow-empty", "-m", "History snapshot refresh marker")
press("刷新 ⌘R")
wait("就绪")
press("Perf commit 11050")
wait(expected)
wait("Version 11050")
fill("")
wait("History snapshot refresh marker")
assert any(e[2] == "继续加载 300 条" and "disabled" not in e[3] for e in controls())
wait("最近 2400 条提交")
screenshot("history-large-refreshed.png")
print("PASS native snapshot invalidates cached history and retains correct filtered selection", flush=True)
# Native list navigation must select original SHA identities, not row numbers.
def key(value):
    auto("key", value, "--settle")


head_before_keys = git("rev-parse", "HEAD")
count_before_keys = int(git("rev-list", "--all", "--count"))
index_before_keys = git("ls-files", "--stage")
loaded = git("log", "--topo-order", "--all", "HEAD", "-n", "2400", "--format=%H").splitlines()
key("CmdOrCtrl+2")
key("Home")
wait(loaded[0])
key("Down")
wait(loaded[1])
key("Up")
wait(loaded[0])
key("End")
wait(loaded[-1])
wait(git("show", loaded[-1] + ":files/00000.txt").strip())
screenshot("history-keyboard-end.png")
key("Up")
wait(loaded[-2])
key("Home")
wait(loaded[0])
print("PASS native history arrows and Home/End select exact SHAs and scroll virtual rows", flush=True)

key("CmdOrCtrl+F")
fill("Perf commit 1105")
wait("Perf commit 11059")
key("Down")  # The search field owns this arrow; selection must not move.
assert loaded[0] in auto("snapshot"), "search editing moved commit selection"
assert any(e[1] == "textField" and e[2] == "搜索已加载提交" and "focused" in e[3] for e in controls())
key("Escape")
assert any(e[1] == "list" and 'key=history-commits' in e[3] and "focused" in e[3] for e in controls()), "Escape did not focus results"
filtered = git("log", "--topo-order", "--all", "HEAD", "--format=%H", "--grep=^Perf commit 1105").splitlines()
key("Down")
wait(filtered[0])
key("Down")
wait(filtered[1])
key("Enter")
wait("Version 11058")
screenshot("history-keyboard-filtered.png")
assert any(e[1] == "textField" and 'value="Perf commit 1105"' in e[3] for e in controls()), "Escape cleared the query"
print("PASS native search keeps editing keys; Escape focuses filtered results and Enter reads exact Diff", flush=True)

git("-c", "user.name=Preview Developer", "-c", "user.email=preview@example.invalid", "commit", "--allow-empty", "-m", "Keyboard history insertion")
key("CmdOrCtrl+R")
wait("就绪")
assert filtered[1] in auto("snapshot"), "refresh changed selected SHA"
key("Down")
wait(filtered[2])
wait("Version 11057")
print("PASS native history selection and focus survive refresh with inserted original indices", flush=True)

key("CmdOrCtrl+F")
fill("no matching history row")
key("Enter")
assert any(e[1] == "list" and 'key=history-commits' in e[3] and "focused" in e[3] for e in controls()), "search Enter did not focus results"
for value in ("Down", "Home", "End", "Enter"):
    key(value)
assert filtered[2] in auto("snapshot"), "empty results selected unrelated commit"
assert git("rev-parse", "HEAD") != head_before_keys  # Only the explicit fixture insertion above.
assert int(git("rev-list", "--all", "--count")) == count_before_keys + 1, "browsing created an extra commit"
assert git("log", "-1", "--format=%s").strip() == "Keyboard history insertion"
assert git("ls-files", "--stage") == index_before_keys
key("CmdOrCtrl+F")
fill("")
key("Enter")
key("Home")
wait("Keyboard history insertion")
wait("从文件列表选择一项")
screenshot("history-keyboard-refreshed.png")
print("PASS native empty-result keys are inert and search Enter returns to read-only list", flush=True)

fill("Perf commit 10000")
press("Perf commit 10000")
wait(deep_sha)
wait("M   files/00000.txt")
press("M   files/00000.txt")
wait("Version 10000")
press("继续加载 300 条")
wait("最近 2700 条提交")
assert deep_sha in auto("snapshot"), "appending history changed selected SHA"
wait("Version 10000")
fill("")
key("CmdOrCtrl+2")
key("End")
paged_last = git("log", "--topo-order", "--all", "HEAD", "-n", "2700", "--format=%H").splitlines()[-1]
wait(paged_last)
wait(git("show", paged_last + ":files/00000.txt").strip())
assert git("ls-files", "--stage") == index_before_keys
screenshot("history-paged-after-refresh.png")
print("PASS native pagination after ref refresh preserves selection and reaches the exact next page", flush=True)
pathlib.Path("build/history-native-timings.json").write_text(json.dumps(measurements, indent=2) + "\n")
print("Automation command timings (include CLI, waits and Git; not frame times):", measurements, flush=True)
