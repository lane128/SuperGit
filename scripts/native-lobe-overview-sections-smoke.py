#!/usr/bin/env python3
"""Check ready worktree overview section clipping with 1,000 owned dirty files."""
import hashlib
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
artifacts = pathlib.Path(tempfile.mkdtemp(prefix="lobe-overview-sections-", dir=build))
with tempfile.TemporaryDirectory(prefix="supergit-overview-sections.") as directory:
    root = pathlib.Path(directory)
    repo, home = root / "repo", root / "home"
    settings = home / "Library/Application Support/supergit/settings.json"
    settings.parent.mkdir(parents=True)
    settings.write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False}))
    env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
    env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")

    def git(*args):
        return subprocess.check_output(["git", "-C", str(repo), *args], env=env)

    subprocess.run(["git", "init", "-b", "main", str(repo)], env=env, check=True, capture_output=True)
    git("config", "user.name", "Overview Fixture")
    git("config", "user.email", "overview@example.invalid")
    paths = [f"files/{i:04d}.txt" for i in range(1000)]
    (repo / "files").mkdir()
    for path in paths:
        (repo / path).write_text("base\n")
    git("add", "--all")
    git("commit", "-m", "Overview fixture base")
    for path in paths:
        (repo / path).write_text("changed " + path + "\n")

    def fingerprint():
        return (git("rev-parse", "HEAD"), git("show-ref"), (repo / ".git/index").read_bytes(),
                git("status", "--porcelain=v1", "-z"),
                hashlib.sha256(b"".join((repo / p).read_bytes() for p in paths)).hexdigest())

    before = fingerprint()
    socket = root / "app.sock"
    session = "overview-sections-" + str(os.getpid())
    env.update(HOME=str(home), MYGO_AUTOMATION=str(socket), SUPERGIT_WINDOW_SIZE="1320x640")
    with (artifacts / "app.log").open("w") as log, (artifacts / "automation.log").open("w") as trace:
        app = subprocess.Popen([str(binaries[0]), str(repo)], cwd=project, env=env, stdout=log, stderr=subprocess.STDOUT)

        def auto(*args):
            result = subprocess.run([str(automation), *args, "--session", session], cwd=project, capture_output=True, text=True, timeout=65)
            trace.write(repr(args) + "\n" + result.stdout + result.stderr + "\n")
            trace.flush()
            if result.returncode:
                raise RuntimeError((args, result.stdout, result.stderr))
            return result.stdout

        def snapshot():
            return auto("snapshot", "--full")

        def ref(role, label):
            lines = [line for line in snapshot().splitlines() if re.search(r'\[' + role + r'\] ' + re.escape(json.dumps(label)) + r'(?: |$)', line)]
            assert len(lines) == 1, (role, label, lines)
            return re.search(r"@e\d+", lines[0])[0]

        def headers(text):
            return [line for line in text.splitlines() if '[button] "files/' in line and ' · Unstaged"' in line]

        try:
            deadline = time.monotonic() + 20
            while not socket.exists():
                if app.poll() is not None or time.monotonic() > deadline:
                    raise RuntimeError("native automation unavailable")
                time.sleep(0.05)
            auto("open", "--socket", str(socket))
            auto("wait", "--text", "Commit summary (required)", "--timeout", "30s", "--settle")
            auto("key", "cmd+1", "--settle")
            auto("wait", "--text", "1000 changes", "--timeout", "30s", "--settle")
            auto("press", ref("listItem", "1000 changes"), "--settle")
            auto("wait", "--text", "+changed files/0000.txt", "--timeout", "60s", "--settle")
            # Require actual patches in the rendered region. A clipped tree
            # does not establish whether every offscreen read has finished.
            deadline = time.monotonic() + 90
            while True:
                first = snapshot()
                if '"Loading diff…"' not in first and 0 < len(headers(first)) < 100:
                    break
                if time.monotonic() > deadline:
                    raise AssertionError("Ready overview did not clip distant section headers")
                time.sleep(0.1)
            assert 'files/0900.txt · Unstaged"' not in first
            (artifacts / "first.snapshot.txt").write_text(first)
            auto("screenshot", str(artifacts / "first.png"))
            deadline = time.monotonic() + 30
            while True:
                auto("wheel", "1150", "520", "0", "500000", "--precise", "--settle")
                last = snapshot()
                if any('files/0999.txt · Unstaged"' in line and "offscreen" not in line for line in headers(last)) and "+changed files/0999.txt" in last:
                    break
                if time.monotonic() > deadline:
                    raise AssertionError("Last ready patch was not revealed by scrolling")
            assert 0 < len(headers(last)) < 100
            (artifacts / "last.snapshot.txt").write_text(last)
            auto("screenshot", str(artifacts / "last.png"))
            auto("wheel", "1150", "520", "0", "-500000", "--precise", "--settle")
            auto("wait", "--text", "+changed files/0000.txt", "--timeout", "30s", "--settle")
            assert fingerprint() == before, "Overview navigation changed fixture Git state or bytes"
            result = {"files": 1000, "first_header_controls": len(headers(first)), "last_header_controls": len(headers(last)), "last_patch_reached": True, "returned_to_first_patch": True, "git_state_preserved": True}
            (artifacts / "results.json").write_text(json.dumps(result, indent=2) + "\n")
            print("PASS " + json.dumps(result), flush=True)
        except BaseException:
            (artifacts / "failure.snapshot.txt").write_text(snapshot())
            auto("screenshot", str(artifacts / "failure.png"))
            print(artifacts, flush=True)
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
print(artifacts)
