#!/usr/bin/env python3
"""Verify pending operation confirmations cannot target replacement conflicts."""
import json
import os
import pathlib
import re
import subprocess
import tempfile
import time

project = pathlib.Path(__file__).resolve().parents[1]
build = project / "build"
automation = build / "mygo-automation"
binaries = list(build.glob("supergit-lobe*e2e"))
if len(binaries) != 1 or not automation.is_file():
    raise SystemExit("run make lobe-e2e-build first")
artifacts = pathlib.Path(tempfile.mkdtemp(prefix="lobe-operation-stale-", dir=build))
results = []
for action in ("Abort", "Skip", "Continue"):
    captures = artifacts / action.lower()
    captures.mkdir()
    with tempfile.TemporaryDirectory(prefix="supergit-operation-stale.") as directory:
        root = pathlib.Path(directory)
        repo, home = root / "repo", root / "home"
        settings = home / "Library/Application Support/supergit/settings.json"
        settings.parent.mkdir(parents=True)
        settings.write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False}))
        git_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
        git_env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")

        def git(*args, check=True):
            result = subprocess.run(["git", "-C", str(repo), *args], env=git_env, capture_output=True)
            if check and result.returncode:
                raise AssertionError((args, result.stdout, result.stderr))
            return result.stdout

        subprocess.run(["git", "init", "-b", "main", str(repo)], env=git_env, check=True, capture_output=True)
        git("config", "user.name", "Operation Fixture")
        git("config", "user.email", "operation@example.invalid")
        path = repo / "冲突 space.txt"
        untouched = repo / "untouched.bin"
        path.write_bytes(b"base\n")
        untouched.write_bytes(b"\x00unchanged\xff\n")
        git("add", "--", path.name, untouched.name)
        git("commit", "--no-gpg-sign", "-m", "Operation fixture base")
        git("checkout", "-b", "topic")
        path.write_bytes(b"first side\n")
        git("commit", "--no-gpg-sign", "-am", "First operation target")
        first = git("rev-parse", "HEAD").strip().decode()
        path.write_bytes(b"replacement side\n")
        git("commit", "--no-gpg-sign", "-am", "Replacement operation target")
        replacement = git("rev-parse", "HEAD").strip().decode()
        git("checkout", "main")
        path.write_bytes(b"current side\n")
        git("commit", "--no-gpg-sign", "-am", "Current operation branch")
        original_head = git("rev-parse", "HEAD")

        def start(target):
            git("cherry-pick", target, check=False)
            assert git("ls-files", "-u"), "fixture needs an actual conflict"
            assert (repo / ".git/CHERRY_PICK_HEAD").read_text().strip() == target
            assert git("rev-parse", "HEAD") == original_head, "replacement must retain HEAD"

        def fingerprint():
            return {
                "head": git("rev-parse", "HEAD"),
                "branch": git("symbolic-ref", "HEAD"),
                "refs": git("show-ref"),
                "index": git("ls-files", "--stage", "-z"),
                "status": git("status", "--porcelain=v1", "-z"),
                "working": path.read_bytes(),
                "untouched": untouched.read_bytes(),
                "operation": (repo / ".git/CHERRY_PICK_HEAD").read_bytes(),
                "message": (repo / ".git/MERGE_MSG").read_bytes(),
            }

        start(first)
        if action == "Continue":
            path.write_bytes(b"first resolved\n")
            git("add", "--", path.name)
        socket = root / "app.sock"
        session = "operation-stale-" + action.lower() + "-" + str(os.getpid())
        env = {**git_env, "HOME": str(home), "MYGO_AUTOMATION": str(socket), "SUPERGIT_WINDOW_SIZE": "980x640"}
        with (captures / "app.log").open("w") as log, (captures / "automation.log").open("w") as trace:
            app = subprocess.Popen([str(binaries[0]), str(repo)], cwd=project, env=env, stdout=log, stderr=subprocess.STDOUT)

            def auto(*args):
                result = subprocess.run([str(automation), *args, "--session", session], cwd=project, capture_output=True, text=True, timeout=40)
                trace.write(repr(args) + "\n" + result.stdout + result.stderr + "\n")
                trace.flush()
                if result.returncode:
                    raise RuntimeError((args, result.stdout, result.stderr))
                return result.stdout

            def snapshot():
                return auto("snapshot", "--full")

            def press(label):
                lines = [line for line in snapshot().splitlines() if '[button] ' + json.dumps(label) + ' ' in line and "disabled" not in line]
                assert len(lines) == 1, (label, lines)
                auto("press", re.search(r"@e\d+", lines[0])[0], "--settle")

            def wait(text):
                auto("wait", "--text", text, "--timeout", "25s", "--settle")

            def ready():
                deadline = time.monotonic() + 25
                while not any('[button] "Refresh"' in line and "disabled" not in line for line in snapshot().splitlines()):
                    if time.monotonic() > deadline:
                        raise AssertionError("repository did not finish reading")
                    time.sleep(0.05)

            try:
                deadline = time.monotonic() + 20
                while not socket.exists():
                    if app.poll() is not None or time.monotonic() > deadline:
                        raise RuntimeError("native automation unavailable")
                    time.sleep(0.05)
                auto("open", "--socket", str(socket))
                wait("Current operation branch")
                ready()
                press("In progress: cherry-pick")
                if action == "Continue":
                    original_card = snapshot()
                    lines = [line for line in original_card.splitlines() if '[button] "Continue"' in line and "disabled" not in line]
                    assert len(lines) == 1, "original resolved operation must enable Continue"
                    original_continue = re.search(r"@e\d+", lines[0])[0]
                    (captures / "original-card.snapshot.txt").write_text(original_card)
                else:
                    press(action)
                    wait("Abort the operation?" if action == "Abort" else "Skip this commit?")
                    (captures / "original-confirmation.snapshot.txt").write_text(snapshot())
                git("cherry-pick", "--abort")
                start(replacement)
                if action == "Continue":
                    path.write_bytes(b"replacement resolved\n")
                    git("add", "--", path.name)
                expected = fingerprint()
                if action == "Continue":
                    # Use the original rendered control without a fresh snapshot
                    # that could turn this into a refreshed-card acceptance test.
                    auto("press", original_continue, "--settle")
                else:
                    press(action)
                wait("The pending operation changed. Refresh, check the operation card, and confirm again.")
                ready()
                assert fingerprint() == expected, "old confirmation changed replacement operation or repository contents"
                assert expected["operation"].strip().decode() == replacement and first != replacement
                (captures / "refused.snapshot.txt").write_text(snapshot())
                auto("screenshot", str(captures / "refused.png"))
                results.append({"action": action, "refresh_before_confirmation": False, "same_kind_same_head_replacement": True, "old_control_refused": True, "pending_confirmation": action != "Continue", "replacement_operation_and_git_state_preserved": True})
                print("PASS native stale " + action + ": same-kind/same-HEAD replacement refused; operation, refs, Index and bytes preserved", flush=True)
            except BaseException:
                (captures / "failure.snapshot.txt").write_text(snapshot())
                auto("screenshot", str(captures / "failure.png"))
                print(captures, flush=True)
                raise
            finally:
                try:
                    auto("close")
                finally:
                    app.terminate()
                    try:
                        app.wait(timeout=10)
                    except subprocess.TimeoutExpired:
                        app.kill()
                        app.wait()
(artifacts / "results.json").write_text(json.dumps(results, indent=2) + "\n")
print(artifacts)
