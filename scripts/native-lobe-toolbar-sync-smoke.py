#!/usr/bin/env python3
"""Accept the narrow macOS toolbar and Sync menu against an owned bare remote."""
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import time

project = Path(__file__).resolve().parents[1]
build = Path(os.environ.get("SUPERGIT_NATIVE_BUILD_DIR", project / "build"))
binary = Path(os.environ.get("SUPERGIT_NATIVE_BINARY", build / "supergit-lobe-e2e"))
driver = build / "mygo-automation"
if not binary.is_file() or not driver.is_file():
    raise SystemExit("Run make lobe-e2e-build first")

artifacts = Path(tempfile.mkdtemp(prefix="toolbar-sync-", dir=build)).resolve()
branch = "feature/überlange-virtual-live-preview-performance-navigation-ios-643"
tracking = f"refs/remotes/origin/{branch}"
destination = f"refs/heads/{branch}"
repo, bare, peer = [artifacts / name for name in ("checkout", "origin.git", "peer")]
env = {key: value for key, value in os.environ.items() if not key.startswith(("GIT_", "MYGO_", "SUPERGIT_"))}
env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_ALLOW_PROTOCOL="file",
           GIT_TERMINAL_PROMPT="0", GIT_OPTIONAL_LOCKS="0",
           GIT_AUTHOR_NAME="Fixture", GIT_AUTHOR_EMAIL="fixture@example.invalid",
           GIT_COMMITTER_NAME="Fixture", GIT_COMMITTER_EMAIL="fixture@example.invalid")
profile = artifacts / "profile"
settings = profile / "supergit" / "settings.json"
settings.parent.mkdir(parents=True)
settings.write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False, "fontScale": 1.15}))


def git(*args, directory=repo):
    return subprocess.check_output(["git", "-C", str(directory), *args], env=env, text=True, timeout=15).strip()


def control(snapshot, role, label):
    pattern = re.compile(r'\s*@e\d+ \[' + role + r'\] ' +
                         re.escape(json.dumps(label, ensure_ascii=False)) + r'(?= |$)')
    return [line for line in snapshot.splitlines() if pattern.match(line)]


def bounds(row):
    match = re.search(r'\bbounds=([-\d.]+),([-\d.]+),([\d.]+)x([\d.]+)', row)
    assert match, row
    return tuple(map(float, match.groups()))


print(f"Toolbar and Sync evidence: {artifacts}", flush=True)
git("init", "--bare", "-b", branch, str(bare), directory=artifacts)
git("init", "-b", branch, str(repo), directory=artifacts)
(repo / "work.txt").write_text("base\n")
git("add", "work.txt")
git("commit", "--no-gpg-sign", "-m", "Fixture base")
git("remote", "add", "origin", str(bare))
git("push", "--set-upstream", "origin", branch)
base = git("rev-parse", "HEAD")
tree = git("rev-parse", "HEAD^{tree}")
head = base
for index in range(102):
    head = git("commit-tree", tree, "-p", head, "-m", f"Local commit {index + 1}")
git("update-ref", destination, head, base)
assert git("rev-list", "--left-right", "--count", "HEAD...@{upstream}") == "102\t0"
assert Path(git("remote", "get-url", "--push", "origin")).resolve() == bare

with tempfile.TemporaryDirectory(prefix="sg-sync-", dir="/tmp") as directory:
    scratch = Path(directory).resolve()
    socket = scratch / "app.sock"
    app_env = {**env, "MYGO_AUTOMATION": str(socket), "MYGO_AUTOMATION_DIR": str(scratch / "driver"),
               "SUPERGIT_TEST_CONFIG_DIR": str(profile), "SUPERGIT_WINDOW_SIZE": "900x640"}
    with (artifacts / "app.log").open("w") as log, (artifacts / "actions.log").open("w") as trace:
        app = subprocess.Popen([str(binary), str(repo)], env=app_env, stdout=log, stderr=subprocess.STDOUT)

        def until(check, description):
            deadline = time.monotonic() + 22
            while app.poll() is None and time.monotonic() < deadline:
                result = check()
                if result:
                    return result
                time.sleep(.05)
            raise AssertionError(description)

        def auto(*args):
            result = subprocess.run([str(driver), *map(str, args), "--session", scratch.name],
                                    env=app_env, capture_output=True, text=True, timeout=25)
            print(args, result.stdout, result.stderr, sep="\n", file=trace, flush=True)
            result.check_returncode()
            return result.stdout

        def snapshot():
            return auto("snapshot", "--full")

        def press(role, label):
            def enabled():
                matches = [row for row in control(snapshot(), role, label)
                           if " disabled" not in row and " offscreen" not in row]
                return matches[0] if len(matches) == 1 else None

            x, y, w, h = bounds(until(enabled, f"No enabled {role} {label}"))
            auto("click", x + w / 2, y + h / 2, "--settle")

        def sync(action):
            press("menuButton", "Sync")
            press("menuItem", action)

        def branch_button(status):
            def current():
                for item in control(snapshot(), "button", "Branches"):
                    if branch in item and (f"({status})" in item if status else f"{branch}\"" in item):
                        return item
                return None

            return until(current, f"Branch button did not expose full ref and {status!r}")

        try:
            until(socket.exists, "No native automation socket")
            auto("open", "--socket", socket)
            button = branch_button("102 ahead")
            x, y, w, h = bounds(button)
            assert abs(x + w / 2 - 450) <= 1 and 0 < x < x + w < 900 and 38 <= y < y + h < 80, button
            layout = snapshot()
            for label in ("Fetch", "Pull (fast-forward only)", "Push", "Refresh"):
                assert not control(layout, "button", label), label
            (artifacts / "narrow-toolbar.txt").write_text(layout)
            auto("screenshot", artifacts / "narrow-long-branch.png")

            press("menuButton", "Sync")
            opened = snapshot()
            assert "102 ahead" in opened and "origin/" + branch in opened
            auto("screenshot", artifacts / "sync-menu.png")
            press("menuItem", "Push branch…")
            press("button", "Cancel")
            assert git("rev-parse", destination, directory=bare) == base
            sync("Push branch…")
            press("button", "Push")
            until(lambda: git("rev-parse", destination, directory=bare) == head, "Push did not publish exact HEAD")
            branch_button("")
            assert git("rev-parse", "HEAD") == head and not git("status", "--porcelain")

            git("clone", "--branch", branch, str(bare), str(peer), directory=artifacts)
            (peer / "peer.txt").write_text("remote fast-forward\n")
            git("add", "peer.txt", directory=peer)
            git("commit", "--no-gpg-sign", "-m", "Peer update", directory=peer)
            new_head = git("rev-parse", "HEAD", directory=peer)
            git("push", "origin", branch, directory=peer)
            index = git("ls-files", "--stage")

            sync("Fetch")
            until(lambda: git("rev-parse", tracking) == new_head, "Fetch did not advance remote-tracking ref")
            branch_button("1 behind")
            assert git("rev-parse", "HEAD") == head and git("ls-files", "--stage") == index
            assert (repo / "work.txt").read_text() == "base\n" and not (repo / "peer.txt").exists()

            sync("Pull (fast-forward only)")
            until(lambda: git("rev-parse", "HEAD") == new_head, "Pull did not fast-forward")
            branch_button("")
            assert git("rev-parse", "HEAD^") == head and not git("status", "--porcelain")
            assert (repo / "peer.txt").read_text() == "remote fast-forward\n"
            print("PASS 900 pt and large font; 102 ahead, confirmed Push, Fetch preserves the checkout, Pull fast-forwards.", flush=True)
        finally:
            if app.poll() is None:
                app.terminate()
                app.wait(timeout=10)
