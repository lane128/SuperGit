#!/usr/bin/env python3
"""Launch, drive and close a native build using isolated local repositories."""
import argparse
import os
import pathlib
import subprocess
import sys
import time

parser = argparse.ArgumentParser()
checks = parser.add_mutually_exclusive_group()
checks.add_argument("--diff", action="store_true", help="exercise line selection and split Diff")
checks.add_argument("--watch", action="store_true", help="exercise automatic change detection")
checks.add_argument("--history", action="store_true", help="exercise a 12,024-commit history fixture")
checks.add_argument("--keyboard", action="store_true", help="exercise keyboard routing and Diff scroll state")
checks.add_argument("--files", action="store_true", help="exercise grouped workspace file keyboard selection")
checks.add_argument("--commit-files", action="store_true", help="exercise immutable commit-file keyboard selection")
checks.add_argument("--stash", action="store_true", help="exercise Stash preview failure, identity and retry")
checks.add_argument("--tags", action="store_true", help="exercise local lightweight tag creation and confirmation")
args = parser.parse_args()
mode = next((name for name in ("tags", "stash", "commit_files", "files", "keyboard", "history", "watch", "diff") if getattr(args, name)), "workflows")
project = pathlib.Path(__file__).resolve().parents[1]
build = project / "build"
fixture_command = [sys.executable, "-I", str(project / "scripts/create-history-demo.py")] if mode == "history" else ["sh", str(project / "scripts/create-workflow-demo.sh")]
root = pathlib.Path(subprocess.check_output(fixture_command, text=True, cwd=project).strip())
root_file = "workflow-demo-root.txt" if mode == "workflows" else mode + "-demo-root.txt"
(build / root_file).write_text(str(root) + "\n")
fixture_home = root / "test-home"
fixture_home.mkdir(mode=0o700)
socket = root / "automation.sock"
session = "supergit-workflows-" + str(os.getpid())
automation = build / "mygo-automation"
env = {**os.environ, "MYGO_AUTOMATION": str(socket), "HOME": str(fixture_home), "GIT_CONFIG_GLOBAL": os.devnull, "GIT_CONFIG_NOSYSTEM": "1"}
with (build / (mode + "-native.log")).open("w") as app_log:
    app = subprocess.Popen([str(build / "supergit-e2e"), str(root / "workspace")], cwd=project, env=env, stdout=app_log, stderr=subprocess.STDOUT)
    try:
        deadline = time.monotonic() + 15
        while not socket.exists():
            if app.poll() is not None:
                raise RuntimeError("native app exited before its automation socket opened")
            if time.monotonic() >= deadline:
                raise TimeoutError("native app did not open its automation socket")
            time.sleep(0.05)
        subprocess.run([str(automation), "open", "--session", session, "--socket", str(socket)], check=True, cwd=project)
        smoke = {"tags": "native-tags-smoke.py", "stash": "native-stash-smoke.py", "diff": "native-diff-smoke.py", "watch": "native-watch-smoke.py", "workflows": "native-workflow-smoke.py", "history": "native-history-smoke.py", "keyboard": "native-keyboard-smoke.py", "files": "native-workspace-smoke.py", "commit_files": "native-commit-files-smoke.py"}[mode]
        subprocess.run([sys.executable, "-I", str(project / "scripts" / smoke), str(root), "--session", session], check=True, cwd=project)
        print("Native workflows passed. Fixture retained at", root, flush=True)
    except Exception:
        # Capture the actual native state before cleanup, rather than losing
        # the failing frame when a background Git read finishes unexpectedly.
        result = subprocess.run([str(automation), "snapshot", "--session", session], cwd=project, capture_output=True, text=True)
        (build / (mode + "-failure-snapshot.txt")).write_text(result.stdout + result.stderr)
        subprocess.run([str(automation), "screenshot", str(build / (mode + "-failure.png")), "--session", session], cwd=project, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        raise
    finally:
        subprocess.run([str(automation), "close", "--session", session], cwd=project, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        app.terminate()
        try:
            app.wait(timeout=10)
        except subprocess.TimeoutExpired:
            app.kill()
            app.wait()
