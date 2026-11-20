#!/usr/bin/env python3
"""Check the 900x700 window against a disposable repository.

Asserts the narrow window's three open points:
  1. the window really opens at 900x700 instead of falling back;
  2. the working tree's commit editor stays visible and inside the window;
  3. the command palette copies the selected commit's full SHA;
  4. the stash header's Apply changes the working tree (the context menu's
     Apply already worked; this checks the header button).

Uses disposable Git data and an isolated HOME; the user's settings,
repositories and installed application are untouched.
"""
import hashlib
import json
import os
import pathlib
import re
import struct
import subprocess
import tempfile
import time

project = pathlib.Path(__file__).resolve().parents[1]
build = project / "build"
automation = build / "mygo-automation"
candidates = list(build.glob("supergit-lobe*e2e"))
if not automation.is_file() or len(candidates) != 1:
    raise SystemExit("Run make lobe-e2e-build first")
artifacts = pathlib.Path(tempfile.mkdtemp(prefix="lobe-narrow-window-", dir=build))
print(f"Artifacts: {artifacts}", flush=True)

WINDOW = (900, 700)
git_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
git_env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0", LC_ALL="C")
results = {"window_requested": list(WINDOW)}


def png_size(path):
    with open(path, "rb") as f:
        head = f.read(24)
    if head[:8] != b"\x89PNG\r\n\x1a\n":
        raise AssertionError(f"{path} is not a PNG")
    return struct.unpack(">II", head[16:24])


with tempfile.TemporaryDirectory(prefix="supergit-narrow-window.") as directory:
    root = pathlib.Path(directory)
    repo = root / "workspace"
    subprocess.run(["git", "init", "--template=", "-b", "main", str(repo)], env=git_env, check=True, capture_output=True)

    def git(*args, **kwargs):
        return subprocess.check_output(["git", "-C", str(repo), *args], env=git_env, **kwargs)

    def write(name, text):
        (repo / name).write_text(text)

    git("config", "user.name", "Fixture")
    git("config", "user.email", "fixture@example.invalid")
    write("base.txt", "base\n")
    git("add", "--", "base.txt")
    git("commit", "-m", "Root commit")
    write("base.txt", "base\nmain tip\n")
    git("commit", "-am", "Main tip")
    # A real merge gives the graph a second parent and a fork to draw.
    git("switch", "-c", "feature")
    write("feature.txt", "feature\n")
    git("add", "--", "feature.txt")
    git("commit", "-m", "Feature work")
    git("switch", "main")
    git("merge", "--no-ff", "feature", "-m", "Merge feature")
    # Future is a child of HEAD and must appear above its parent.
    git("switch", "-c", "future")
    future_env = {**git_env, "GIT_AUTHOR_DATE": "2030-01-01T00:00:00 +0000", "GIT_COMMITTER_DATE": "2030-01-01T00:00:00 +0000"}
    write("future.txt", "future\n")
    git("add", "--", "future.txt")
    subprocess.run(["git", "-C", str(repo), "commit", "-m", "Future side tip"], env=future_env, check=True, capture_output=True)
    git("switch", "main")
    head = git("rev-parse", "HEAD").decode().strip()
    write("base.txt", "base\nmain tip\nstashed\n")
    git("stash", "push", "-m", "Fixture stash")
    # Dirty state on a different file, so applying the stash cannot conflict
    # with the working tree and the Apply button's effect is unambiguous.
    write("untracked.txt", "untracked\n")

    def fingerprint():
        return {
            "head": git("rev-parse", "HEAD").decode().strip(),
            "refs": hashlib.sha256(git("for-each-ref", "--format=%(refname) %(objectname)")).hexdigest(),
            "stashes": git("stash", "list").decode(),
            "index": hashlib.sha256(git("ls-files", "--stage", "-z")).hexdigest(),
            "status": git("status", "--porcelain", "-z").decode(),
            "base": (repo / "base.txt").read_text(),
            "untracked": (repo / "untracked.txt").read_text(),
        }

    before = fingerprint()
    results["head"] = head[:12]

    home = root / "home"
    settings = home / "Library/Application Support/supergit"
    settings.mkdir(parents=True)
    (settings / "settings.json").write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False}))
    socket = root / "app.sock"
    session = "supergit-narrow-window"
    env = {**git_env, "HOME": str(home), "MYGO_AUTOMATION": str(socket),
           "SUPERGIT_WINDOW_SIZE": f"{WINDOW[0]}x{WINDOW[1]}"}
    with (artifacts / "app.log").open("w") as log, (artifacts / "automation.log").open("w") as trace:
        app = subprocess.Popen([str(candidates[0]), str(repo)], cwd=project, env=env, stdout=log, stderr=subprocess.STDOUT)

        def auto(*args):
            completed = subprocess.run([str(automation), *args, "--session", session], cwd=project, capture_output=True, text=True, timeout=90)
            trace.write(repr(args) + "\n" + completed.stdout + completed.stderr + "\n")
            trace.flush()
            if completed.returncode:
                raise RuntimeError(f"{args}: {completed.stdout}{completed.stderr}")
            return completed.stdout

        def entries():
            found = []
            for text in auto("snapshot", "--full").splitlines():
                match = re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)', text)
                if match:
                    found.append((match[1], match[2], json.loads(match[3]) if match[3] else "", match[4]))
            return found

        def shown(items, name, role=None):
            return [e for e in items if e[2] == name and "offscreen" not in e[3] and (role is None or e[1] == role)]

        def bounds(extra):
            match = re.search(r"bounds=([\d.]+),([\d.]+),([\d.]+)x([\d.]+)", extra)
            return tuple(float(v) for v in match.groups()) if match else None

        try:
            deadline = time.monotonic() + 30
            while not socket.exists():
                if app.poll() is not None or time.monotonic() > deadline:
                    raise AssertionError("Native app did not open automation socket")
                time.sleep(0.05)
            auto("open", "--socket", str(socket))
            results["windows"] = auto("windows")
            auto("wait", "--text", "Merge feature", "--timeout", "30s", "--settle")
            auto("screenshot", str(artifacts / "01-history-900x700.png"))
            size = png_size(artifacts / "01-history-900x700.png")
            results["screenshot_size"] = list(size)
            if size[0] not in (WINDOW[0], WINDOW[0] * 2) or size[1] not in (WINDOW[1], WINDOW[1] * 2):
                raise AssertionError(f"window opened at {size[0]}x{size[1]} pixels, not {WINDOW[0]}x{WINDOW[1]} (or 2x)")

            found = entries()
            tabs = [e for e in found if e[1] == "tab" and e[2] in ("Changes", "History", "Stash")]
            assert not tabs, f"page tabs returned: {tabs}"
            worktree = [e for e in found if e[1] == "listItem" and "change" in e[2] and "offscreen" not in e[3]]
            stash = [e for e in found if e[1] == "listItem" and "Fixture stash" in e[2] and "offscreen" not in e[3]]
            merge = shown(found, "Merge feature", "listItem")
            future = shown(found, "Future side tip", "listItem")
            assert worktree and stash and merge and future, "narrow history is missing one of its rows"
            order = [found.index(e) for e in (worktree[0], stash[0], future[0], merge[0])]
            assert order == sorted(order), f"row order broke: {order}"
            results["row_order"] = "worktree, stash, child on future, HEAD"

            # A selected commit sits on two lanes; its tint must not cut the
            # lane that passes beside the node.
            feature = shown(found, "Feature work", "listItem")
            assert feature, "the merged feature commit is missing"
            auto("focus", feature[0][0], "--settle")
            auto("click", feature[0][0], "--settle")
            auto("screenshot", str(artifacts / "06-selected-commit-lanes.png"))

            # 2. The working tree's commit editor, inside the window.
            auto("focus", worktree[0][0], "--settle")
            auto("click", worktree[0][0], "--settle")
            auto("wait", "--text", "Commit summary (required)", "--timeout", "10s", "--settle")
            auto("screenshot", str(artifacts / "02-worktree-editor.png"))
            found = entries()
            summary = [e for e in found if e[1] == "textField" and
                       ("Commit summary (required)" in e[2] or 'placeholder="Commit summary (required)"' in e[3])]
            assert summary, "the commit summary field is missing at 900 points"
            summary_bounds = bounds(summary[0][3])
            assert summary_bounds, f"no bounds for the summary field: {summary[0]}"
            x, _, w, _ = summary_bounds
            if w < 120 or x + w > WINDOW[0] + 0.5:
                raise AssertionError(f"commit summary field is squeezed or clipped: {summary_bounds}")
            results["summary_bounds"] = list(summary_bounds)
            commit_button = shown(found, "Commit", "button")
            assert commit_button, "the Commit button is missing at 900 points"

            # 3. The stash header Apply, checked against Git rather than a toast.
            auto("focus", stash[0][0], "--settle")
            auto("click", stash[0][0], "--settle")
            auto("wait", "--text", "Pop (apply and drop)", "--timeout", "10s", "--settle")
            auto("screenshot", str(artifacts / "03-stash-details.png"))
            found = entries()
            apply_rows = shown(found, "Apply", "button")
            assert apply_rows, "the stash header has no Apply button"
            assert "disabled" not in apply_rows[0][3], "the stash header Apply button is disabled"
            auto("click", apply_rows[0][0], "--settle")
            applied = False
            deadline = time.monotonic() + 10
            while time.monotonic() < deadline:
                if "stashed" in (repo / "base.txt").read_text():
                    applied = True
                    break
                time.sleep(0.1)
            if not applied:
                found = entries()
                notices = [e[2] for e in found if e[1] in ("text", "alert", "status")]
                raise AssertionError(f"header Apply did not change the working tree; visible text: {notices}")
            assert git("stash", "list").decode().strip(), "Apply dropped the stash"
            results["stash_header_apply"] = True
            auto("screenshot", str(artifacts / "04-after-header-apply.png"))

            # 4. Copy SHA from the palette.
            found = entries()
            merge = shown(found, "Merge feature", "listItem")
            assert merge, "the merge commit row is gone"
            auto("focus", merge[0][0], "--settle")
            auto("click", merge[0][0], "--settle")
            auto("wait", "--text", head, "--timeout", "10s", "--settle")
            auto("key", "cmd+p", "--settle")
            auto("type", "copy sha", "--settle")
            auto("screenshot", str(artifacts / "05-copy-sha-palette.png"))
            found = entries()
            row = shown(found, "Copy SHA", "listItem")
            if not row:
                raise AssertionError("the palette does not list Copy SHA for the selected commit")
            assert "disabled" not in row[0][3], "the Copy SHA row is disabled with a selected commit"
            auto("key", "Enter", "--settle")
            time.sleep(0.3)
            copied = subprocess.run(["pbpaste"], capture_output=True, text=True, check=True).stdout.strip()
            if copied != head:
                raise AssertionError(f"clipboard holds {copied!r}, want {head}")
            results["palette_copy_sha"] = True
            results["palette_closed"] = not shown(entries(), "Command Palette", "dialog")

            # Browsing plus Apply leaves exactly the stash-applied change.
            after = fingerprint()
            assert after["head"] == before["head"] and after["refs"] == before["refs"], "HEAD or refs changed"
            assert after["stashes"] == before["stashes"], "stash list changed after Apply"
            assert "stashed" in after["base"], "stash content missing from base.txt"
            assert after["untracked"] == before["untracked"], "unrelated working file changed"
            results["git_unchanged_except_apply"] = True
            results["passed"] = True
        except Exception:
            results["passed"] = False
            try:
                auto("screenshot", str(artifacts / "failure.png"))
                (artifacts / "failure.snapshot.txt").write_text(auto("snapshot", "--full"))
            except Exception:
                pass
            raise
        finally:
            (artifacts / "results.json").write_text(json.dumps(results, indent=2))
            try:
                auto("close")
            except Exception:
                pass
            app.terminate()
            try:
                app.wait(timeout=10)
            except subprocess.TimeoutExpired:
                app.kill()
                app.wait(timeout=5)
print(json.dumps(results, indent=2), flush=True)
