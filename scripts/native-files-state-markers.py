#!/usr/bin/env python3
"""Verify the Files change-type badges in an isolated native app fixture.

Run `make app` and `make e2e-build` first. Screenshots use the same source
with its e2e automation plugin; the installed bundle is launched separately.
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

# The badge letter is its own image node, named "<code> <meaning>".
MARKERS = {
    "modified.go": "M Modified",
    "deleted.go": "D Deleted",
    "untracked.go": "? Untracked",
    "staged.go": "A Added",
    "conflict.go": "! Conflict",
}


def run(palette, mode, scale, out, window_width):
    case = out / f"{palette}-{mode}-{scale}-{window_width}"
    case.mkdir()
    home = case / "home"
    config = home / "Library/Application Support/supergit"
    config.mkdir(parents=True)
    (config / "settings.json").write_text(json.dumps({
        "language": "en", "theme": mode, "palette": palette, "fontScale": scale, "historySplit": .42,
        "autoFetch": False, "autoCheckUpdates": False, "autoDownloadUpdates": False,
    }))
    env = {k: v for k, v in os.environ.items() if not k.startswith(("GIT_", "MYGO_", "SUPERGIT_"))}
    env.update(HOME=str(home), GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1",
               GIT_TERMINAL_PROMPT="0", SUPERGIT_WINDOW_SIZE=f"{window_width}x840")
    repo = case / "repo"

    def git(*args, check=True):
        return subprocess.run(["git", "-C", str(repo), *args], env=env, text=True,
                              capture_output=True, check=check).stdout

    subprocess.run(["git", "init", "-b", "main", str(repo)], env=env, check=True, capture_output=True)
    git("config", "user.name", "Fixture")
    git("config", "user.email", "fixture@example.invalid")
    for path in ["modified.go", "deleted.go", "conflict.go", "dual.go"]:
        (repo / path).write_text("base content\n")
    git("add", "--all")
    git("commit", "--no-gpg-sign", "-m", "Marker fixture")
    # A/M/D/? rows, plus a conflict from a merged branch.
    git("checkout", "-b", "other")
    (repo / "conflict.go").write_text("other side\n")
    git("commit", "--no-gpg-sign", "-am", "Other side")
    git("checkout", "main")
    (repo / "conflict.go").write_text("main side\n")
    git("commit", "--no-gpg-sign", "-am", "Main side")
    git("merge", "other", check=False)
    (repo / "modified.go").write_text("working change\n")
    (repo / "deleted.go").unlink()
    (repo / "untracked.go").write_text("new file\n")
    (repo / "staged.go").write_text("staged content\n")
    git("add", "staged.go")
    git("add", "dual.go")
    (repo / "dual.go").write_text("index copy\n")
    git("add", "dual.go")
    (repo / "dual.go").write_text("worktree copy\n")

    def fingerprint():
        return {"head": git("rev-parse", "HEAD"), "index": git("ls-files", "--stage"),
                "status": git("status", "--porcelain"),
                "files": {str(p.relative_to(repo)): hashlib.sha256(p.read_bytes()).hexdigest()
                          for p in repo.rglob("*") if p.is_file() and ".git" not in p.relative_to(repo).parts}}

    before = fingerprint()
    with tempfile.TemporaryDirectory(prefix="sg-markers-") as socketdir:
        socket = Path(socketdir) / "app.sock"
        env["MYGO_AUTOMATION"] = str(socket)
        session = "markers-" + Path(socketdir).name
        with (case / "app.log").open("w") as log, (case / "actions.log").open("w") as trace:
            app = subprocess.Popen([str(BUILD / "supergit-e2e"), str(repo)], cwd=PROJECT, env=env,
                                   stdout=log, stderr=subprocess.STDOUT)

            def auto(*args):
                result = subprocess.run([str(BUILD / "mygo-automation"), *map(str, args),
                                         "--session", session], cwd=PROJECT, capture_output=True,
                                        text=True, timeout=30)
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

            def marker(code_label):
                for row in controls():
                    if row["role"] == "image" and row["name"] == code_label:
                        return row
                raise AssertionError("missing marker " + code_label)

            def row_around(rect):
                x, y, w, h = rect
                center = y + h / 2
                for row in controls():
                    if row["role"] not in ("listItem", "row") or not row["rect"]:
                        continue
                    rx, ry, rw, rh = row["rect"]
                    if ry <= center <= ry + rh and rx <= x <= rx + rw:
                        return row
                return None

            try:
                deadline = time.monotonic() + 20
                while not socket.exists():
                    assert app.poll() is None, "native app exited"
                    if time.monotonic() > deadline:
                        raise TimeoutError("automation socket unavailable")
                    time.sleep(.05)
                auto("open", "--socket", socket)
                auto("wait", "--text", "Marker fixture", "--timeout", "20s", "--settle")
                changes = next(r for r in controls() if r["role"] == "listItem"
                               and re.fullmatch(r"\d+ changes?", r["name"]))
                auto("press", changes["ref"], "--settle")
                auto("wait", "--text", "Unstaged", "--timeout", "20s", "--settle")
                auto("screenshot", case / "files.png")
                geometry = {}
                for path, label in MARKERS.items():
                    badge = marker(label)
                    x, y, w, h = badge["rect"]
                    assert abs(w - 18 * scale) < .7, (label, w)
                    assert abs(h - 18 * scale) < .7, (label, h)
                    row = row_around(badge["rect"])
                    assert row is not None, ("no row for", label)
                    rx, ry, rw, rh = row["rect"]
                    assert abs(rh - 29 * scale) < .7, ("row height", label, rh)
                    assert rx <= x and x + w <= rx + rw, ("badge leaves row", label)
                    # The badge leads the name: no text label starts left of it.
                    left_text = [r for r in controls() if r["role"] == "text" and r["rect"]
                                 and abs((r["rect"][1] + r["rect"][3] / 2) - (y + h / 2)) < 2
                                 and rx <= r["rect"][0] < x - .5]
                    assert not left_text, ("text before badge", label, left_text[:2])
                    geometry[path] = {"badge": badge["rect"], "row": row["rect"]}
                # A conflict keeps its marker and has no stage control.
                conflict_row = row_around(marker("! Conflict")["rect"])
                buttons = [r for r in controls() if r["role"] == "button" and r["rect"]
                           and conflict_row["rect"][1] <= r["rect"][1] <= conflict_row["rect"][1] + conflict_row["rect"][3]
                           and r["name"] in ("Stage file", "Unstage file")]
                assert not buttons, ("conflict keeps a stage control", buttons)
                # Staged and unstaged A/M rows keep their own action labels.
                names = {r["name"] for r in controls() if r["role"] == "button"}
                assert "Stage file" in names and "Unstage file" in names, names
                auto("screenshot", case / "files-markers.png")
                # History commit rows carry the same badge.
                commit = next(r for r in controls() if r["role"] == "listItem"
                              and "Main side" in r["name"])
                auto("press", commit["ref"], "--settle")
                auto("wait", "--text", "M Modified", "--timeout", "20s", "--settle")
                history = {r["name"] for r in controls() if r["role"] == "image"}
                assert any(n.startswith(("A ", "M ", "D ")) for n in history), history
                auto("screenshot", case / "history-markers.png")
                assert fingerprint() == before, "viewing the lists changed the repository"
                (case / "result.json").write_text(json.dumps({"passed": True, "geometry": geometry}, indent=2))
                print(f"PASS {case.name}", flush=True)
                return {"case": case.name, "passed": True}
            finally:
                app.terminate()
                try:
                    app.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    app.kill()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--window-width", type=int, default=1320)
    parser.add_argument("--cases", default="")
    args = parser.parse_args()
    out = Path(tempfile.mkdtemp(prefix="files-markers-native-", dir=BUILD))
    cases = [(p, m, s) for p in ("supergit", "catppuccin-macchiato") for m in ("light", "dark") for s in (1, 1.15)]
    if args.cases:
        wanted = set(args.cases.split(","))
        cases = [c for c in cases if f"{c[0]}-{c[1]}-{c[2]}" in wanted]
    results = [run(palette, mode, scale, out, args.window_width) for palette, mode, scale in cases]
    (out / "summary.json").write_text(json.dumps(results, indent=2))
    print("PASS", len(results), "cases:", out)


if __name__ == "__main__":
    main()
