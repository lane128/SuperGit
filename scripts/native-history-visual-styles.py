#!/usr/bin/env python3
"""Verify independent history style choices using an isolated native app/config/repo.

Build: make e2e-build && GOFLAGS=-tags=e2e make app
Run: python3 scripts/native-history-visual-styles.py --output /tmp/history-style-evidence
The output directory must be new. All repository/config writes stay inside it.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import time

PROJECT = Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--output", type=Path)
parser.add_argument("--binary", type=Path, default=PROJECT / "build/SuperGit.app/Contents/MacOS/supergit")
args = parser.parse_args()
out = args.output.resolve() if args.output else Path(tempfile.mkdtemp(prefix="history-styles-"))
if args.output:
    out.mkdir(parents=True, exist_ok=False)
automation = PROJECT / "build/mygo-automation"
assert args.binary.is_file() and automation.is_file(), "Build the e2e bundle and automation first"
# Prove config isolation is compiled in BEFORE starting the executable. A
# normal `make app` can overwrite the same path with a release-style build.
metadata = subprocess.check_output(["go", "version", "-m", str(args.binary)], text=True)
tag_line = re.search(r"(?m)^\s*build\s+-tags=(.*)$", metadata)
tags = tag_line.group(1).strip().strip('"').replace(",", " ").split() if tag_line else []
if "e2e" not in tags:
    raise SystemExit("Refusing to launch a non-e2e binary; run GOFLAGS=-tags=e2e make app first")
repo, config = out / "repo", out / "config"
repo.mkdir()
(config / "supergit").mkdir(parents=True)
settings = config / "supergit/settings.json"
env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")

def git(*argv):
    return subprocess.check_output(["git", "-C", str(repo), *argv], env=env, stderr=subprocess.STDOUT).decode().strip()

git("init", "-b", "main")
git("config", "user.name", "History Fixture")
git("config", "user.email", "fixture@example.invalid")
git("config", "commit.gpgsign", "false")

def commit(file, text, title):
    (repo / file).write_text(text)
    git("add", "--", file)
    git("commit", "--no-gpg-sign", "-m", title)

commit("base.txt", "base\n", "Prepare history workspace")
git("tag", "v1.0")
git("checkout", "-b", "feature/history/reference-layout")
commit("topic.txt", "topic\n", "Refine history reference layout")
git("update-ref", "refs/remotes/origin/feature/history/reference-layout", "HEAD")
git("checkout", "main")
commit("main.txt", "main\n", "Keep history navigation responsive")
git("merge", "--no-ff", "feature/history/reference-layout", "-m", "Merge history reference layout")
git("checkout", "-b", "fix/focus")
commit("focus.txt", "focus\n", "Restore focus after closing a tab")
git("checkout", "main")
git("merge", "--no-ff", "fix/focus", "-m", "Merge focus repair")
commit("latest.txt", "latest\n", "Verify history visual styles")
for tag in ("v1.4.0", "release/2026.11-preview.3", "stable"):
    git("tag", tag)
git("update-ref", "refs/remotes/origin/main", "HEAD")
(repo / "base.txt").write_text("pending spacing\n")
git("stash", "push", "-m", "Adjust history spacing")
before = (git("rev-parse", "HEAD"), git("show-ref"), git("status", "--porcelain"))
preview = PROJECT / "design/history-style-preview.html"
if preview.exists():
    raw = preview.read_bytes()
    (out / "reference.html").write_bytes(raw)
    (out / "reference.sha256").write_text(hashlib.sha256(raw).hexdigest() + "\n")

session = "history-styles-" + str(os.getpid())
results = []

def run_case(name, theme, width, restart=False):
    capture = out / name
    capture.mkdir()
    if not restart:
        settings.write_text(json.dumps({"language": "en", "theme": theme, "autoFetch": False,
            "autoCheckUpdates": False, "sidebarHidden": True, "historySplit": width / 1500,
            "historyGraphStyle": "rounded", "historyRefStyle": "segmented"}))
    # Keep the socket short enough for macOS sockaddr_un, even in deep worktrees.
    with tempfile.TemporaryDirectory(prefix="sg-style-") as sockdir:
        sock = Path(sockdir) / "app.sock"
        launch_env = dict(env, SUPERGIT_TEST_CONFIG_DIR=str(config), MYGO_AUTOMATION=str(sock), SUPERGIT_WINDOW_SIZE="1560x920")
        with (capture / "app.log").open("w") as log, (capture / "automation.log").open("w") as trace:
            process = subprocess.Popen([str(args.binary), str(repo)], cwd=PROJECT, env=launch_env, stdout=log, stderr=subprocess.STDOUT)
            def auto(*argv):
                call = subprocess.run([str(automation), *argv, "--session", session], cwd=PROJECT, text=True, capture_output=True, timeout=40)
                trace.write(repr(argv) + "\n" + call.stdout + call.stderr + "\n")
                trace.flush()
                if call.returncode:
                    raise RuntimeError((argv, call.stdout, call.stderr))
                return call.stdout
            def snapshot():
                return auto("snapshot", "--full")
            def press(label, role):
                lines = [line for line in snapshot().splitlines() if f'[{role}] {json.dumps(label)} ' in line and "disabled" not in line]
                assert len(lines) == 1, (label, lines)
                ref = re.search(r"@e\d+", lines[0])[0]
                if "offscreen" in lines[0]:
                    auto("act", ref, "scrollIntoView", "--settle")
                auto("press", ref, "--settle")
            try:
                deadline = time.monotonic() + 30
                while not sock.exists():
                    if process.poll() is not None or time.monotonic() > deadline:
                        raise RuntimeError("Native app did not start; inspect app.log")
                    time.sleep(.05)
                auto("open", "--socket", str(sock))
                auto("window", "w1")
                auto("wait", "--text", "Verify history visual styles", "--timeout", "30s", "--settle")
                initial = snapshot()
                (capture / "initial.snapshot.txt").write_text(initial)
                auto("key", "cmd+,", "--settle")
                windows = auto("windows")
                panel = next(re.search(r"w\d+", line)[0] for line in windows.splitlines() if "SuperGit · Settings" in line)
                auto("window", panel)
                press("History", "tab")
                if restart:
                    state = snapshot()
                    assert re.search(r'\[toggleButton\] "Short curve" .*checked', state), state
                    assert re.search(r'\[toggleButton\] "Compact stripe" .*checked', state), state
                    (capture / "restored.snapshot.txt").write_text(state)
                    results.append({"case": name, "restart": "compact + stripe restored"})
                    return
                for shape, label in (("rounded", "Rounded"), ("chamfer", "Chamfer"), ("compact", "Short curve")):
                    for style, ref_label in (("segmented", "Split capsule"), ("stripe", "Compact stripe")):
                        auto("window", panel)
                        press(label, "toggleButton")
                        press(ref_label, "toggleButton")
                        saved = json.loads(settings.read_text())
                        assert saved["historyGraphStyle"] == shape and saved["historyRefStyle"] == style, saved
                        auto("screenshot", str(capture / f"settings-{shape}-{style}.png"))
                        auto("window", "w1")
                        auto("wait", "--settle")
                        state = snapshot()
                        assert '"HEAD"' in state or re.search(r'\[text\] "\+\d+"', state), "HEAD or reference overflow missing"
                        assert "stash@{0}" in state, "Stash identity missing"
                        (capture / f"{shape}-{style}.snapshot.txt").write_text(state)
                        auto("screenshot", str(capture / f"{shape}-{style}.png"))
                        results.append({"case": name, "graph": shape, "refs": style, "saved": True})
                # Automation window selection does not grant native keyboard
                # focus, and macOS refuses to activate a background app that a
                # script launched, so Settings may not drain its notice queue.
                # Try once anyway; assert the rendered notice only when it did.
                (capture / "windows-before-activation.txt").write_text(auto("windows"))
                activation = subprocess.run(["/usr/bin/osascript", "-l", "JavaScript", "-e",
                    'ObjC.import("AppKit"); $.NSRunningApplication.runningApplicationWithProcessIdentifier('
                    + str(process.pid) + ').activateWithOptions(3)'],
                    capture_output=True, text=True, timeout=10)
                assert activation.returncode == 0, activation.stderr
                (capture / "activation.txt").write_text(activation.stdout + activation.stderr)
                auto("window", "w1")
                auto("key", "cmd+,", "--settle")
                windows = auto("windows")
                panel_focused = any(panel in line and " focused" in line for line in windows.splitlines())
                (capture / "windows-after-activation.txt").write_text(windows)
                # A deterministic write error that touches only this fixture.
                auto("window", panel)
                backup = settings.with_suffix(".saved.json")
                original = settings.read_bytes()
                settings.rename(backup)
                settings.mkdir()
                try:
                    press("Rounded", "toggleButton")
                    if panel_focused:
                        auto("wait", "--text", "Could not save settings", "--timeout", "10s", "--settle")
                    state = snapshot()
                    (capture / "save-failure.snapshot.txt").write_text(state)
                    if panel_focused:
                        assert "Could not save settings" in state, state
                    # Without native focus the notice stays queued; the choice
                    # must still apply to this run and the disk must be intact.
                    assert re.search(r'\[toggleButton\] "Rounded" [^\n]*checked', state), "failed persistence did not apply the choice"
                    assert settings.is_dir(), "failed save wrote to the blocked path"
                    assert backup.read_bytes() == original, "failed save rewrote the previous preferences"
                finally:
                    settings.rmdir()
                    backup.rename(settings)
                results.append({"case": name, "save_failure": "applied in session, disk untouched", "save_warning_visual_verified": panel_focused})
                # Restore the last combination through the real settings UI.
                press("Short curve", "toggleButton")
            finally:
                try:
                    auto("close")
                finally:
                    process.terminate()
                    try:
                        process.wait(timeout=10)
                    except subprocess.TimeoutExpired:
                        process.kill()
                        process.wait()

try:
    run_case("light-wide", "light", 900)
    run_case("light-standard", "light", 600)
    run_case("dark-narrow", "dark", 360)
    run_case("restart", "dark", 360, restart=True)
    after = (git("rev-parse", "HEAD"), git("show-ref"), git("status", "--porcelain"))
    assert before == after, "Style changes mutated the fixture repository"
    (out / "results.json").write_text(json.dumps({"passed": True, "cases": results, "repository_unchanged": True}, indent=2))
    combinations = sum(1 for r in results if "graph" in r)
    warning_visual = all(r.get("save_warning_visual_verified") for r in results if "save_failure" in r)
    print(json.dumps({"passed": True, "output": str(out), "combinations": combinations, "save_warning_visual_verified": warning_visual}))
except Exception as error:
    (out / "results.json").write_text(json.dumps({"passed": False, "cases": results, "error": str(error)}, indent=2))
    raise
