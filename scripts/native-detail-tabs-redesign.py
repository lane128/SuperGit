#!/usr/bin/env python3
"""Verify detail-tab rendering/dragging in isolated native app fixtures.

Run make app and make e2e-build first. Screenshots use the same source with
its e2e automation plugin; the standard bundle launch is verified separately.
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


def run(palette, mode, scale, out, window_width):
    case = out / f"{palette}-{mode}-{scale}"
    case.mkdir()
    home = case / "home"
    config = home / "Library/Application Support/supergit"
    config.mkdir(parents=True)
    (config / "settings.json").write_text(json.dumps({
        "language": "en", "theme": mode, "palette": palette, "fontScale": scale, "historySplit": .25,
        "autoFetch": False, "autoCheckUpdates": False, "autoDownloadUpdates": False,
    }))
    env = {k: v for k, v in os.environ.items() if not k.startswith(("GIT_", "MYGO_", "SUPERGIT_"))}
    env.update(HOME=str(home), GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1",
               GIT_TERMINAL_PROMPT="0", SUPERGIT_WINDOW_SIZE=f"{window_width}x840")
    repo = case / "repo"

    def git(*args):
        return subprocess.check_output(["git", "-C", str(repo), *args], env=env, text=True)

    subprocess.run(["git", "init", "-b", "main", str(repo)], env=env, check=True, capture_output=True)
    git("config", "user.name", "Fixture")
    git("config", "user.email", "fixture@example.invalid")
    dual = "internal/lobedesk/commit_details_select_test.go"
    paths = [dual, "deleted.go", *[f"files/configuration-verification-long-file-{i:02d}.go" for i in range(12)],
             "tests/café-校验-👩‍💻-configuration-verification-文件.go"]
    for path in paths:
        f = repo / path
        f.parent.mkdir(parents=True, exist_ok=True)
        f.write_text("original content\n")
    git("add", "--all")
    git("commit", "--no-gpg-sign", "-m", "Detail tab fixture")
    for path in paths:
        (repo / path).write_text("working content\n")
    git("add", "--", dual)
    (repo / dual).write_text("working content\nunstaged addition\n")
    (repo / "deleted.go").unlink()
    (repo / "added.go").write_text("added content\n")
    git("add", "--", "added.go")

    def fingerprint():
        return {"head": git("rev-parse", "HEAD"), "index": git("ls-files", "--stage"),
                "status": git("status", "--porcelain"),
                "files": {str(p.relative_to(repo)): hashlib.sha256(p.read_bytes()).hexdigest()
                          for p in repo.rglob("*") if p.is_file() and ".git" not in p.relative_to(repo).parts}}

    before = fingerprint()
    # Keep the Unix socket path under macOS's 104-byte limit.
    with tempfile.TemporaryDirectory(prefix="sg-tabs-") as socketdir:
        socket = Path(socketdir) / "app.sock"
        env["MYGO_AUTOMATION"] = str(socket)
        session = "tabs-" + Path(socketdir).name
        with (case / "app.log").open("w") as log, (case / "actions.log").open("w") as trace:
            app = subprocess.Popen([str(BUILD / "supergit-e2e"), str(repo)], cwd=PROJECT, env=env, stdout=log, stderr=subprocess.STDOUT)

            def auto(*args):
                command = [str(BUILD / "mygo-automation"), *map(str, args), "--session", session]
                result = subprocess.run(command, cwd=PROJECT, capture_output=True, text=True, timeout=30)
                trace.write(repr(args) + "\n" + result.stdout + result.stderr + "\n")
                trace.flush()
                result.check_returncode()
                return result.stdout

            def controls():
                tree = auto("snapshot", "--full")
                (case / "latest.txt").write_text(tree)
                rows = []
                for line in tree.splitlines():
                    match = re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)', line)
                    if not match:
                        continue
                    ref, role, name, rest = match.groups()
                    bounds = re.search(r'\bbounds=([-\d.]+),([-\d.]+),([\d.]+)x([\d.]+)', rest)
                    rows.append({"ref": ref, "role": role, "name": json.loads(name) if name else "", "rest": rest,
                                 "rect": tuple(map(float, bounds.groups())) if bounds else None})
                return rows

            def press(role, name, occurrence=0):
                matches = [r for r in controls() if r["role"] == role and r["name"] == name]
                assert matches, (role, name)
                auto("press", matches[occurrence]["ref"], "--settle")

            def details():
                rows = controls()
                start = next(i for i, r in enumerate(rows) if r["role"] == "tabList" and r["name"] == "File details")
                viewport = next(r for r in reversed(rows[:start]) if r["role"] == "scroll")
                return viewport, [r for r in rows[start + 1:] if r["role"] == "tab"]

            def selected():
                viewport, tabs = details()
                tab = next(r for r in tabs if "checked" in r["rest"] or "selected" in r["rest"])
                return viewport, tab

            def assert_visible():
                vp, tab = selected()
                x, y, w, h = tab["rect"]
                vx, vy, vw, vh = vp["rect"]
                assert x >= vx - .6 and x + w + 24 <= vx + vw + .6, (vp, tab)
                assert abs(h - 36 * scale) < .6, tab
                assert "description=" in tab["rest"], tab
                return {"viewport": vp["rect"], "tab": tab["rect"], "name": tab["name"]}

            try:
                deadline = time.monotonic() + 20
                while not socket.exists():
                    assert app.poll() is None, "native app exited"
                    if time.monotonic() > deadline:
                        raise TimeoutError("automation socket unavailable")
                    time.sleep(.05)
                auto("open", "--socket", socket)
                auto("wait", "--text", "Detail tab fixture", "--timeout", "20s", "--settle")
                changes = next(r for r in controls() if r["role"] == "listItem" and re.fullmatch(r"\d+ changes", r["name"]))
                auto("press", changes["ref"], "--settle")
                for path in ["added.go", "deleted.go", dual]:
                    press("listItem", path)
                    auto("screenshot", case / (Path(path).stem + ".png"))
                press("listItem", dual, 1)
                auto("screenshot", case / "statuses.png")
                for path in paths[2:]:
                    press("listItem", path)
                assert_visible()
                auto("screenshot", case / "overflow.png")
                # Make the actual details viewport narrow by dragging its splitter.
                vp, _ = details()
                rows = controls()
                start = next(i for i, r in enumerate(rows) if r["ref"] == vp["ref"])
                files = next(r for r in reversed(rows[:start]) if r["role"] == "scroll")
                # Split handles are pointer-only; derive the history handle
                # from the Files pane's left edge and the existing 8-DIP gap.
                fx, fy, fw, fh = files["rect"]
                auto("pointer", "down", fx - 4, fy + fh / 2)
                auto("pointer", "move", 1150, fy + fh / 2)
                auto("pointer", "up", 1150, fy + fh / 2, "--settle")
                press("tab", "commit_details_select_test.go · Unstaged")
                geometry = assert_visible()
                if window_width == 900:
                    assert abs(geometry["viewport"][2] - 230) < .6, geometry
                auto("wait", "--text", "+unstaged addition", "--timeout", "20s", "--settle")
                auto("screenshot", case / "narrow-unstaged.png")
                # The whole six-DIP rail is a pointer target, including its top edge.
                vp, tab = selected()
                vx, vy, vw, vh = vp["rect"]
                old = tab["name"]
                auto("pointer", "down", vx + 2, vy + vh + .5)
                auto("pointer", "move", vx + 1, vy + vh + .5)
                auto("pointer", "up", vx + 1, vy + vh + .5, "--settle")
                _, tabs = details()
                assert selected()[1]["name"] == old, "rail drag changed selection"
                assert tabs[0]["rect"][0] >= vx - .5, "rail did not reveal first tab"
                # Native wheel still moves this same container.
                auto("wheel", vx + vw / 2, vy + 12, 400, 0, "--precise", "--settle")
                assert details()[1][0]["rect"][0] < vx - 1, "horizontal wheel did not scroll"
                press("tab", "commit_details_select_test.go · Staged")
                assert_visible()
                # Close the visible staged version using pointer coordinates.
                _, tab = selected()
                x, y, w, h = tab["rect"]
                auto("click", x + w + 12, y + h / 2, "--settle")
                assert_visible()
                focused = [r for r in controls() if "focused" in r["rest"]]
                assert any(r["role"] == "tab" for r in focused), "close did not restore tab focus"
                auto("key", "Tab", "--settle")
                assert not any(r["role"] == "tab" and "focused" in r["rest"] for r in controls()), "Tab did not leave group"
                auto("key", "shift+Tab", "--settle")
                assert any(r["role"] == "tab" and "focused" in r["rest"] for r in controls()), "Shift+Tab did not return"
                auto("screenshot", case / "closed-successor.png")
                assert fingerprint() == before, "viewing/closing tabs changed the repository"
                (case / "result.json").write_text(json.dumps({"passed": True, **geometry}, indent=2))
                print(f"PASS {case.name}", flush=True)
                return {"case": case.name, "passed": True, **geometry}
            finally:
                app.terminate()
                try:
                    app.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    app.kill()
                    app.wait(timeout=10)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--single", action="store_true")
    parser.add_argument("--window-width", type=int, choices=(900, 1320), default=1320)
    args = parser.parse_args()
    out = Path(tempfile.mkdtemp(prefix="detail-tabs-native-", dir=BUILD))
    print(out, flush=True)
    cases = [("supergit", "light", 1)] if args.single else [(p, m, s) for p in ("supergit", "catppuccin-macchiato") for m in ("light", "dark") for s in (1, 1.15)]
    results = [run(*case, out, args.window_width) for case in cases]
    (out / "results.json").write_text(json.dumps(results, indent=2))
