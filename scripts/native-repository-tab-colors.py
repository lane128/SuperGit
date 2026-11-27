#!/usr/bin/env python3
"""Exercise approved repository cards through the native e2e app.

Run make app and make e2e-build first. Uses only disposable repositories and
profiles. The source of the e2e executable matches the standard app bundle.
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
BUILD = PROJECT / "build"


def run_case(out, mode, width, scale):
    case = out / f"{mode}-{width}-{scale}"
    case.mkdir()
    profile = case / "profile"
    config = profile / "supergit"
    config.mkdir(parents=True)
    (config / "settings.json").write_text(json.dumps({
        "language": "en", "theme": mode, "palette": "supergit", "fontScale": scale,
        "autoFetch": False, "autoCheckUpdates": False, "autoDownloadUpdates": False,
    }))
    env = {k: v for k, v in os.environ.items() if not k.startswith(("GIT_", "MYGO_", "SUPERGIT_"))}
    env.update(SUPERGIT_TEST_CONFIG_DIR=str(profile), SUPERGIT_WINDOW_SIZE=f"{width}x{640 if width == 900 else 840}",
               GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0",
               GIT_AUTHOR_NAME="Fixture", GIT_COMMITTER_NAME="Fixture",
               GIT_AUTHOR_EMAIL="fixture@example.invalid", GIT_COMMITTER_EMAIL="fixture@example.invalid")
    roots = []
    presets = ["default", "green", "rose", "blue", "amber", "violet", "gray"]
    for name in ["api", "web-console", "设计系统-repository-with-a-long-name", "release-tools", "docs", "service", "infra"]:
        root = case / name
        subprocess.run(["git", "init", "-b", "main", str(root)], env=env, check=True, capture_output=True)
        (root / "sample.txt").write_text("base\n")
        subprocess.run(["git", "-C", str(root), "add", "."], env=env, check=True)
        subprocess.run(["git", "-C", str(root), "commit", "--no-gpg-sign", "-m", "Fixture"], env=env, check=True, capture_output=True)
        roots.append(str(root.resolve()))
    (Path(roots[1]) / "sample.txt").write_text("dirty fixture\n")
    colors = config / "repository-colors"
    colors.mkdir()

    def preference(root):
        return colors / (hashlib.sha256(root.encode()).hexdigest() + ".json")

    for root, preset in zip(roots, presets):
        preference(root).write_text(json.dumps(preset))
    session_file = config / "repository-session.json"
    session_file.write_text(json.dumps({"roots": roots[:3], "active": roots[0]}))

    def fingerprint():
        return {str(p): hashlib.sha256(p.read_bytes()).hexdigest()
                for root in roots for p in Path(root).rglob("*") if p.is_file()}

    before = fingerprint()
    with tempfile.TemporaryDirectory(prefix="sg-colors-", dir="/tmp") as scratch:
        socket = Path(scratch) / "app.sock"
        env.update(MYGO_AUTOMATION=str(socket), MYGO_AUTOMATION_DIR=scratch)
        session = Path(scratch).name
        with (case / "actions.log").open("w") as trace:
            def auto(*args):
                result = subprocess.run([str(BUILD / "mygo-automation"), *map(str, args), "--session", session],
                                        capture_output=True, text=True, timeout=25)
                trace.write(repr(args) + "\n" + result.stdout + result.stderr + "\n")
                trace.flush()
                result.check_returncode()
                return result.stdout

            def tree():
                value = auto("snapshot", "--full")
                (case / "latest.txt").write_text(value)
                return value

            def one(role, name):
                matches = [line.strip() for line in tree().splitlines()
                           if re.match(r'\s*@e\d+ \[' + re.escape(role) + r'\] ' + re.escape(json.dumps(name, ensure_ascii=False)) + r'(?= |$)', line)]
                assert len(matches) == 1, (role, name, matches)
                return matches[0]

            def click(role, name, *options):
                node = one(role, name)
                assert " disabled" not in node and " offscreen" not in node, node
                auto("click", node.split()[0], *options, "--settle")

            def ready():
                deadline = time.monotonic() + 20
                while time.monotonic() < deadline:
                    if socket.exists():
                        try:
                            auto("open", "--socket", socket)
                            current = tree()
                            new_tab = next((line for line in current.splitlines() if '[button] "New Tab"' in line), "")
                            if re.search(r'\[tab\] "api".*checked', current) and new_tab and ' disabled' not in new_tab:
                                return
                        except (AssertionError, subprocess.CalledProcessError):
                            pass
                    time.sleep(.1)
                raise AssertionError("fixture app did not become ready")

            for launch in range(2):
                with (case / f"app-{launch}.log").open("w") as log:
                    proc = subprocess.Popen([str(BUILD / "supergit-e2e")], env=env, cwd=PROJECT, stdout=log, stderr=subprocess.STDOUT)
                    try:
                        ready()
                        if launch == 0:
                            first = one("tab", "api")
                            assert "checked" in first, first
                            auto("screenshot", case / "cards.png")
                            click("tab", "web-console", "--button", "right")
                            auto("screenshot", case / "palette.png")
                            assert len(re.findall(r'\[menuItemRadio\]', tree())) == 7
                            click("menuItemRadio", "Amber")
                            assert "checked" in one("tab", "api"), "inactive color selected another tab"
                            assert json.loads(preference(roots[1]).read_text()) == "amber"
                            # Keyboard opens on the inactive target without switching the workbench.
                            auto("focus", one("tab", "web-console").split()[0], "--settle")
                            auto("key", "Shift+F10", "--settle")
                            auto("key", "Right", "Enter", "--settle")
                            assert json.loads(preference(roots[1]).read_text()) == "rose"
                            auto("key", "Shift+F10", "--settle")
                            auto("key", "Escape", "--settle")
                            assert "menuItemRadio" not in tree()
                            auto("screenshot", case / "recolored.png")
                        else:
                            assert json.loads(preference(roots[1]).read_text()) == "rose"
                            click("tab", "web-console", "--button", "right")
                            assert "checked" in one("menuItemRadio", "Rose"), "restart lost assignment"
                            click("menuItemRadio", "Default")
                            assert json.loads(preference(roots[1]).read_text()) == "default"
                            auto("screenshot", case / "same-background.png")
                            click("tab", "api", "--button", "right")
                            click("menuItemRadio", "Gray")
                            auto("screenshot", case / "gray-selected.png")
                            click("button", "Scroll tabs right")
                            assert "checked" in one("tab", "api"), "scrolling changed active checkout"
                            auto("screenshot", case / "overflow.png")
                    finally:
                        proc.terminate()
                        proc.wait(timeout=15)
                        auto("close")
                if launch == 0:
                    session_file.write_text(json.dumps({"roots": roots, "active": roots[0]}))
    assert fingerprint() == before, "UI actions modified repository data"
    return {"theme": mode, "width": width, "scale": scale, "result": "PASS"}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()
    out = args.out.resolve()
    out.mkdir(parents=True, exist_ok=False)
    results = []
    for mode in ("light", "dark"):
        for width, scale in ((900, 1), (1320, 1), (900, 1.15), (1320, 1.15)):
            result = run_case(out, mode, width, scale)
            results.append(result)
            (out / "results.json").write_text(json.dumps(results, indent=2))
            print(json.dumps(result), flush=True)
