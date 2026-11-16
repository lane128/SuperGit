#!/usr/bin/env python3
"""Drag all three Lobe dividers and verify their widths after restarting.

Only a new disposable repository and isolated settings are written.
Run after make lobe-e2e-build, serially with other native UI scripts.
"""
import json
import os
import pathlib
import re
import subprocess
import tempfile
import time

project = pathlib.Path(__file__).resolve().parents[1]
build = project / "build"
candidates = list(build.glob("supergit-lobe*e2e"))
automation = build / "mygo-automation"
if len(candidates) != 1 or not automation.is_file():
    raise SystemExit("run make lobe-e2e-build first")
artifacts = build / "workbench-layout-smoke"
artifacts.mkdir(parents=True, exist_ok=True)

with tempfile.TemporaryDirectory(prefix="supergit-layout.") as directory:
    root = pathlib.Path(directory)
    repo, home, socket = root / "workspace", root / "home", root / "app.sock"
    settings = home / "Library/Application Support/supergit/settings.json"
    settings.parent.mkdir(parents=True)
    settings.write_text(json.dumps({"language": "en", "theme": "light"}))
    env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
    env.update(HOME=str(home), GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0", MYGO_AUTOMATION=str(socket), SUPERGIT_WINDOW_SIZE="1320x640")

    def git(*args):
        return subprocess.check_output(["git", "-C", str(repo), *args], env=env, text=True)

    subprocess.run(["git", "init", "-b", "main", str(repo)], env=env, check=True, capture_output=True)
    git("config", "user.name", "Layout Fixture")
    git("config", "user.email", "layout@example.invalid")
    (repo / "notes.txt").write_text("layout content\n")
    git("add", "--", "notes.txt")
    git("commit", "-m", "Layout fixture commit")
    before = (git("rev-parse", "HEAD"), git("ls-files", "--stage"), (repo / "notes.txt").read_bytes())
    session = "layout-" + str(os.getpid())
    app = None

    def auto(*args):
        result = subprocess.run([str(automation), *args, "--session", session], cwd=project, text=True, capture_output=True, timeout=40)
        if result.returncode:
            raise RuntimeError(f"{args}: {result.stdout}{result.stderr}")
        return result.stdout

    def geometry():
        entries = []
        for line in auto("snapshot", "--full").splitlines():
            match = re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)', line)
            if not match:
                continue
            bounds = re.search(r"bounds=([-0-9.]+),([-0-9.]+),([-0-9.]+)x([-0-9.]+)", match[4])
            if bounds:
                entries.append((match[2], tuple(float(v) for v in bounds.groups())))
        history = next(r for role, r in entries if role == "group" and r[0] > 150 and r[1] < 150 and r[2] >= 260 and r[3] > 300)
        tab_at = next(i for i, (role, _) in enumerate(entries) if role == "tabList")
        details = next(r for role, r in reversed(entries[:tab_at]) if role == "scroll")
        return (history[0], history[2], details[0])

    def stop():
        if app is not None:
            subprocess.run([str(automation), "close", "--session", session], cwd=project, capture_output=True)
            app.terminate()
            try:
                app.wait(timeout=10)
            except subprocess.TimeoutExpired:
                app.kill()
                app.wait()

    with (artifacts / "app.log").open("w") as log:
        try:
            def launch():
                if socket.exists():
                    socket.unlink()  # socket belongs to this disposable test
                process = subprocess.Popen([str(candidates[0]), str(repo)], cwd=project, env=env, stdout=log, stderr=subprocess.STDOUT)
                deadline = time.monotonic() + 20
                while not socket.exists():
                    if process.poll() is not None or time.monotonic() > deadline:
                        raise RuntimeError("layout fixture app did not open automation socket")
                    time.sleep(0.05)
                auto("open", "--socket", str(socket))
                auto("wait", "--text", "Layout fixture commit", "--timeout", "20s", "--settle")
                return process

            app = launch()
            original = geometry()
            for index, distance, key in ((0, 16, "sidebarSplit"), (1, -24, "historySplit"), (2, -20, "filesSplit")):
                current = geometry()
                x = (current[0] - 3.5, current[0] + current[1] + 3.5, current[2] - 3.5)[index]
                for kind, px in (("move", x), ("down", x), ("move", x + distance), ("up", x + distance)):
                    auto("pointer", kind, str(px), "300", "--settle")
                updated = geometry()
                assert abs(updated[index] - current[index]) > 5, f"{key} pointer drag did not change pane geometry: {current} -> {updated}"
                saved = json.loads(settings.read_text())
                assert saved.get(key, 0) > 0, f"{key} was not persisted after pointer release"
            preferred = geometry()
            auto("screenshot", str(artifacts / "three-dividers-dragged.png"))
            stop()
            app = None
            app = launch()
            restored = geometry()
            assert all(abs(a - b) <= 2 for a, b in zip(preferred, restored)), f"restart lost preferred pane geometry: {preferred} -> {restored}"
            assert (git("rev-parse", "HEAD"), git("ls-files", "--stage"), (repo / "notes.txt").read_bytes()) == before, "layout interactions changed fixture Git state"
            auto("screenshot", str(artifacts / "three-dividers-restored.png"))
            (artifacts / "result.json").write_text(json.dumps({"original": original, "dragged": preferred, "restored": restored}, indent=2))
            print("PASS native three-divider pointer drag, persisted settings and restart recovery", flush=True)
            # The historical TODO also requires double-clicking Files to reset.
            auto("click", str(restored[2] - 3.5), "300", "--clicks", "2", "--settle")
            reset = geometry()
            assert json.loads(settings.read_text()).get("filesSplit", 0) == 0, "Files double-click did not clear preferred width"
            assert all(abs(reset[i] - restored[i]) <= 2 for i in (0, 1)), "Files reset moved other dividers"
            original_files_width = original[2] - original[0] - original[1]
            reset_files_width = reset[2] - reset[0] - reset[1]
            assert abs(reset_files_width - original_files_width) <= 2, f"Files reset did not restore default width: {reset_files_width} vs {original_files_width}"
            stop()
            app = None
            app = launch()
            reset_restored = geometry()
            assert all(abs(a - b) <= 2 for a, b in zip(reset, reset_restored)), "Restart lost Files default reset"
            assert (git("rev-parse", "HEAD"), git("ls-files", "--stage"), (repo / "notes.txt").read_bytes()) == before, "Files reset changed fixture Git state"
            auto("screenshot", str(artifacts / "files-divider-reset.png"))
            (artifacts / "result.json").write_text(json.dumps({"original": original, "dragged": preferred, "restored": restored, "reset": reset, "reset_restored": reset_restored}, indent=2))
            print("PASS native Files divider double-click restores default width and persists across restart", flush=True)
        except Exception:
            try:
                (artifacts / "failure.snapshot.txt").write_text(auto("snapshot", "--full"))
                auto("screenshot", str(artifacts / "failure.png"))
            except Exception:
                pass
            raise
        finally:
            stop()
