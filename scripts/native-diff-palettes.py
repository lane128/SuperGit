#!/usr/bin/env python3
"""Exercise Diff colors in native windows, using isolated repositories and homes.

Run make app and make e2e-build first. The automation build uses the same source;
this script never installs the app or changes the user's preferences.
"""
import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import time

PROJECT = Path(__file__).resolve().parents[1]
BUILD = PROJECT / "build"
OPTIONS = [("delta-layered", "Delta layered"), ("git-classic", "Git classic"), ("vivid", "Vivid")]


def run(mode, width, output):
    case = output / f"{mode}-{width}"
    case.mkdir()
    home = case / "home"
    settings = home / "Library/Application Support/supergit/settings.json"
    settings.parent.mkdir(parents=True)
    settings.write_text(json.dumps({"language": "en", "theme": mode, "historySplit": .32,
                                    "autoFetch": False, "autoCheckUpdates": False,
                                    "autoDownloadUpdates": False,
                                    "fontScale": 1.15 if width < 1000 else 1,
                                    "palette": "catppuccin-macchiato" if width < 1000 else "supergit"}))
    if width < 1000:
        geometry = home / "Library/Application Support/supergit-e2e/window-state.json"
        geometry.parent.mkdir(parents=True)
        geometry.write_text(json.dumps({"settings-compact-v1": {"x": 100, "y": 100, "width": 560, "height": 680}}))
    env = {k: v for k, v in os.environ.items() if not k.startswith(("GIT_", "MYGO_", "SUPERGIT_"))}
    env.update(HOME=str(home), GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1",
               GIT_TERMINAL_PROMPT="0", SUPERGIT_WINDOW_SIZE=f"{width}x840")
    repo = case / "repo"
    subprocess.run(["git", "init", "-b", "main", str(repo)], env=env, check=True, capture_output=True)

    def git(*args):
        return subprocess.check_output(["git", "-C", str(repo), *args], env=env)

    git("config", "user.name", "Palette Fixture")
    git("config", "user.email", "palette@example.invalid")
    path = repo / "colors.txt"
    path.write_text('context\ncolor = "before"\n中文 = "旧值"\nblank = ""\n' + 'old/path/' * 25 + ' END\n')
    git("add", "colors.txt")
    git("commit", "--no-gpg-sign", "-m", "Palette base")
    path.write_text('context\ncolor = "committed"\n中文 = "中间值"\nblank = " "\n' + 'history/path/' * 25 + ' END\n')
    git("commit", "--no-gpg-sign", "-am", "Palette change")
    path.write_text('context\ncolor = "working"\n中文 = "新值"\nblank = "  "\n' + 'new/path/' * 25 + ' END\n')
    before = (git("rev-parse", "HEAD"), git("ls-files", "--stage"), git("diff"), path.read_bytes())
    with tempfile.TemporaryDirectory(prefix="sg-colors-") as socketdir:
        socket = Path(socketdir) / "app.sock"
        env["MYGO_AUTOMATION"] = str(socket)
        session = "colors-" + Path(socketdir).name
        with (case / "app.log").open("w") as log, (case / "actions.log").open("w") as trace:
            def auto(*args):
                result = subprocess.run([str(BUILD / "mygo-automation"), *map(str, args), "--session", session],
                                        cwd=PROJECT, text=True, capture_output=True, timeout=40)
                trace.write(repr(args) + "\n" + result.stdout + result.stderr + "\n")
                trace.flush()
                result.check_returncode()
                return result.stdout

            def entries():
                text = auto("snapshot", "--full")
                (case / "latest.txt").write_text(text)
                rows = []
                for line in text.splitlines():
                    match = re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)', line)
                    if match:
                        ref, role, label, rest = match.groups()
                        rows.append((ref, role, json.loads(label) if label else "", rest))
                return rows

            def press(label, role="button"):
                matches = [r for r in entries() if r[1] == role and (r[2] == label or role == "menuItem" and r[2] == label + " ✓")]
                assert len(matches) == 1, (label, matches)
                auto("press", matches[0][0], "--settle")

            def shot(name):
                (case / (name + ".txt")).write_text(auto("snapshot", "--full"))
                auto("screenshot", case / (name + ".png"))

            def launch():
                process = subprocess.Popen([str(BUILD / "supergit-e2e"), str(repo)], cwd=PROJECT,
                                           env=env, stdout=log, stderr=subprocess.STDOUT)
                deadline = time.monotonic() + 25
                while not socket.exists():
                    assert process.poll() is None, "app exited during launch"
                    if time.monotonic() >= deadline:
                        process.terminate()
                        raise TimeoutError("automation socket unavailable")
                    time.sleep(.05)
                auto("open", "--socket", socket)
                auto("window", "w1")
                auto("wait", "--text", "Palette change", "--timeout", "25s", "--settle")
                return process

            def settings_window():
                auto("window", "w1")
                press("Settings")
                windows = auto("windows")
                panel = next(line for line in windows.splitlines() if "SuperGit · Settings" in line)
                auto("window", re.search(r"w\d+", panel)[0])
                auto("wait", "--text", "Diff colors", "--timeout", "10s", "--settle")

            app = launch()
            try:
                settings_window()
                assert any(r[2] == "Diff colors" and 'description="Delta layered"' in r[3] for r in entries()), "missing default Delta"
                for key, label in OPTIONS:
                    settings_window()
                    press("Diff colors")
                    press(label, "menuItem")
                    assert json.loads(settings.read_text()).get("diffPalette", "delta-layered") == key, "palette not saved"
                    shot(key + "-settings")
                    # Settling a destroyed window reports no_window; confirm closure
                    # through the window list instead of snapshotting that window.
                    auto("menu", "File", "Close Window")
                    assert "SuperGit · Settings" not in auto("windows"), "Settings did not close"
                    auto("window", "w1")
                    changes = next(r for r in entries() if r[1] == "listItem" and re.fullmatch(r"\d+ changes?", r[2]))
                    auto("press", changes[0], "--settle")
                    auto("wait", "--text", 'color = "working"', "--timeout", "20s", "--settle")
                    shot(key + "-overview")
                    file_row = next(r for r in entries() if r[1] == "listItem" and "colors.txt" in r[2])
                    auto("press", file_row[0], "--settle")
                    gutter = next(r for r in entries() if r[1] == "button" and r[2] == "2 Added line")
                    auto("click", gutter[0], "--settle")
                    assert any(r[2] == "2 Added line" and "checked" in r[3] for r in entries()), "selection missing"
                    shot(key + "-selected")
                    # The selection bar changes row IDs/layout. Focus the visible
                    # code paragraph after that rebuild before invoking scoped Find.
                    code = next(r for r in entries() if r[1] == "insertion" and r[2] == '+color = "working"')
                    auto("click", code[0], "--settle")
                    auto("key", "cmd+f", "--settle")
                    search = next(r for r in entries() if r[1] == "textField" and r[2] == "Find in Diff")
                    auto("fill", search[0], "color", "--settle")
                    shot(key + "-search")
                    press("Close Find in Diff")
                    commit = next(r for r in entries() if r[1] == "listItem" and "Palette change" in r[2])
                    auto("press", commit[0], "--settle")
                    auto("wait", "--text", 'color = "before"', "--timeout", "20s", "--settle")
                    shot(key + "-history")
                    press("Show side-by-side diff")
                    shot(key + "-split")
                    press("Show unified diff")
                    code = next(r for r in entries() if r[1] == "deletion" and r[2] == '-color = "before"')
                    auto("click", code[0], "--settle")
                    auto("key", "cmd+f", "--settle")
                    search = next(r for r in entries() if r[1] == "textField" and r[2] == "Find in Diff")
                    auto("fill", search[0], "END", "--settle")
                    shot(key + "-long-search")
                    press("Close Find in Diff")
                    print(f"PASS {mode}/{width}/{key}", flush=True)
                auto("close")
                app.terminate()
                app.wait(timeout=10)
                if socket.exists():
                    socket.unlink()
                app = launch()
                settings_window()
                assert any(r[2] == "Diff colors" and 'description="Vivid"' in r[3] for r in entries()), "restart lost choice"
                shot("restart-vivid")
                after = (git("rev-parse", "HEAD"), git("ls-files", "--stage"), git("diff"), path.read_bytes())
                assert before == after, "palette interaction changed repository content"
                (case / "result.json").write_text(json.dumps({"passed": True, "mode": mode, "width": width,
                                                            "palettes": [k for k, _ in OPTIONS], "restart": True,
                                                            "repositoryUnchanged": True}, indent=2))
            finally:
                if app.poll() is None:
                    app.terminate()
                    app.wait(timeout=10)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--mode", choices=("light", "dark", "both"), default="both")
    parser.add_argument("--width", type=int, default=1320)
    args = parser.parse_args()
    output = Path(tempfile.mkdtemp(prefix="diff-palettes-", dir=BUILD))
    print(output, flush=True)
    for mode in (("light", "dark") if args.mode == "both" else (args.mode,)):
        run(mode, args.width, output)
