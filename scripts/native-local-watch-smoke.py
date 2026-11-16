#!/usr/bin/env python3
"""Native local-change detection in disposable repositories; never presses Refresh."""
import json
import os
import pathlib
import re
import subprocess
import tempfile
import time

project = pathlib.Path(__file__).resolve().parent.parent
build = project / "build"
automation = build / "mygo-automation"
candidates = list(build.glob("supergit-lobe*e2e"))
if not automation.is_file() or len(candidates) != 1:
    raise SystemExit("Run make lobe-e2e-build first")
artifacts = build / "local-watch-smoke"
artifacts.mkdir(parents=True, exist_ok=True)
session = "supergit-local-watch"


def run():
    with tempfile.TemporaryDirectory(prefix="supergit-local-watch.") as directory:
        root = pathlib.Path(directory)
        repo, source, home = root / "workspace", root / "child-source", root / "home"
        settings = home / "Library/Application Support/supergit"
        settings.mkdir(parents=True)
        (settings / "settings.json").write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False}))
        git_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
        git_env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")

        def git(*args, directory=repo):
            return subprocess.check_output(["git", "-C", str(directory), *args], env=git_env, text=True)

        def init(directory):
            subprocess.run(["git", "init", "-b", "main", str(directory)], env=git_env, check=True, capture_output=True)
            git("config", "user.name", "Fixture", directory=directory)
            git("config", "user.email", "fixture@example.invalid", directory=directory)

        init(source)
        (source / "child.txt").write_text("child base\n")
        git("add", "--all", directory=source)
        git("commit", "--no-gpg-sign", "-m", "Child base", directory=source)
        init(repo)
        for path in ("a.txt", "文件 space.txt"):
            (repo / path).write_text("base\n")
        git("-c", "protocol.file.allow=always", "submodule", "add", str(source), "child")
        git("add", "--all")
        git("commit", "--no-gpg-sign", "-m", "Watch baseline")
        (repo / "a.txt").write_text("dirty before opening\n")
        socket = root / "app.sock"
        env = {**git_env, "HOME": str(home), "MYGO_AUTOMATION": str(socket), "SUPERGIT_WINDOW_SIZE": "1320x640"}
        with (artifacts / "app.log").open("w") as log, (artifacts / "automation.log").open("w") as trace:
            app = subprocess.Popen([str(candidates[0]), str(repo)], cwd=project, env=env, stdout=log, stderr=subprocess.STDOUT)

            def auto(*args):
                result = subprocess.run([str(automation), *args, "--session", session], cwd=project, capture_output=True, text=True, timeout=40)
                trace.write(repr(args) + "\n" + result.stdout + result.stderr + "\n")
                trace.flush()
                if result.returncode:
                    raise RuntimeError(f"{args}: {result.stdout}{result.stderr}")
                return result.stdout

            def controls():
                result = []
                for line in auto("snapshot", "--full").splitlines():
                    match = re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)', line)
                    if match:
                        result.append((match[1], match[2], json.loads(match[3]) if match[3] else "", match[4]))
                return result

            def wait(text):
                auto("wait", "--text", text, "--timeout", "25s", "--settle")

            def assert_draft():
                fields = [e for e in controls() if e[1] == "textField"]
                assert any(e[2] == "Commit summary (required)" and "Watch draft" in e[3] for e in fields), "automatic refresh lost summary draft"
                assert any(e[2] == "Description (optional)" and "Keep this body" in e[3] for e in fields), "automatic refresh lost body draft"

            try:
                deadline = time.monotonic() + 20
                while not socket.exists():
                    if app.poll() is not None:
                        raise RuntimeError("native app exited before automation connected")
                    if time.monotonic() > deadline:
                        raise TimeoutError("native automation socket unavailable")
                    time.sleep(0.05)
                auto("open", "--socket", str(socket))
                wait("Watch baseline")
                row = next(e for e in controls() if e[1] == "listItem" and (e[2].endswith(" change") or e[2].endswith(" changes")))
                auto("press", row[0], "--settle")
                wait("+dirty before opening")
                fields = {e[2]: e[0] for e in controls() if e[1] == "textField"}
                auto("fill", fields["Commit summary (required)"], "Watch draft", "--settle")
                auto("fill", fields["Description (optional)"], "Keep this body", "--settle")
                auto("focus", fields["Commit summary (required)"], "--settle")
                before_head, before_index = git("rev-parse", "HEAD"), git("ls-files", "--stage")
                (repo / "a.txt").write_text("saved externally again\n")
                wait("+saved externally again")
                assert_draft()
                assert any(e[1] == "textField" and e[2] == "Commit summary (required)" and "focused" in e[3] for e in controls()), "automatic refresh stole editor focus"
                (repo / "文件 space.txt").write_text("non-selected external save\n")
                wait("+non-selected external save")
                assert git("rev-parse", "HEAD") == before_head and git("ls-files", "--stage") == before_index, "local detection wrote HEAD or Index"
                print("PASS native automatic dirty-file and non-selected Unicode-file saves, draft and editor focus, autoFetch disabled", flush=True)
                git("add", "--", "a.txt")
                wait("a.txt · Staged")
                git("commit", "--no-gpg-sign", "-m", "Watch external commit")
                wait("Watch external commit")
                git("checkout", "-b", "watched-branch")
                wait("watched-branch")
                assert_draft()
                print("PASS native external Stage, Commit and branch switch without pressing Refresh", flush=True)
                child = repo / "child"
                git("config", "user.name", "Fixture", directory=child)
                git("config", "user.email", "fixture@example.invalid", directory=child)
                (child / "child.txt").write_text("child advanced externally\n")
                git("commit", "--no-gpg-sign", "-am", "Child advanced", directory=child)
                expected_head, expected_index = git("rev-parse", "HEAD"), git("ls-files", "--stage")
                wait("child · Unstaged")
                assert_draft()
                assert git("rev-parse", "HEAD") == expected_head and git("ls-files", "--stage") == expected_index, "child detection wrote parent repository"
                assert (repo / "文件 space.txt").read_text() == "non-selected external save\n"
                (artifacts / "final.snapshot.txt").write_text(auto("snapshot", "--full"))
                auto("screenshot", str(artifacts / "final.png"))
                print("PASS native submodule HEAD observation and parent HEAD/Index preservation", flush=True)
            except BaseException:
                try:
                    (artifacts / "failure.snapshot.txt").write_text(auto("snapshot", "--full"))
                    auto("screenshot", str(artifacts / "failure.png"))
                except Exception:
                    pass
                raise
            finally:
                subprocess.run([str(automation), "close", "--session", session], cwd=project, capture_output=True)
                app.terminate()
                try:
                    app.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    app.kill()
                    app.wait(timeout=5)


if __name__ == "__main__":
    run()
