#!/usr/bin/env python3
"""Verify existing history-menu Cherry-pick/Revert initiation in owned repos."""
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
candidates = list(build.glob("supergit-lobe*e2e"))
if not automation.is_file() or len(candidates) != 1:
    raise SystemExit("run make lobe-e2e-build first")
artifacts = pathlib.Path(tempfile.mkdtemp(prefix="lobe-integrate-", dir=build))
results = []


def run_case(kind):
    captures = artifacts / kind
    captures.mkdir()
    with tempfile.TemporaryDirectory(prefix="supergit-integrate.") as directory:
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
        git("config", "user.name", "Fixture")
        git("config", "user.email", "fixture@example.invalid")
        main_file = repo / "main 文件.txt"
        main_file.write_text("base\n")
        git("add", "--", main_file.name)
        git("commit", "-m", "Integration fixture base")
        base_tree = git("rev-parse", "HEAD^{tree}")
        git("checkout", "-b", "topic")
        topic_file = repo / "topic 文件.txt"
        topic_file.write_text("complete topic bytes\n")
        git("add", "--", topic_file.name)
        git("commit", "-m", "Integration fixture source", "-m", "Complete source body\n\nRefs: native-initiation")
        topic = git("rev-parse", "HEAD").strip().decode()
        source_message = git("show", "-s", "--format=%B", "HEAD").strip()
        git("checkout", "main")
        main_file.write_text("main committed version\n")
        git("commit", "-am", "Integration fixture main")
        original_head = git("rev-parse", "HEAD").strip().decode()
        target = topic if kind == "cherry-pick" else original_head
        subject = "Integration fixture source" if kind == "cherry-pick" else "Integration fixture main"
        label, action, success = (("Cherry-pick commit…", "Cherry-pick", "Cherry-picked commit") if kind == "cherry-pick"
                                  else ("Revert commit…", "Revert", "Reverted commit"))
        before = (git("show-ref"), git("ls-files", "--stage"), main_file.read_bytes(), git("status", "--porcelain=v1", "-z"))
        before_index_bytes = (repo / ".git/index").read_bytes()
        socket = root / "app.sock"
        session = "integrate-" + kind + "-" + str(os.getpid())
        env.update(HOME=str(home), MYGO_AUTOMATION=str(socket), SUPERGIT_WINDOW_SIZE="1320x640")
        with (captures / "app.log").open("w") as log, (captures / "automation.log").open("w") as trace:
            app = subprocess.Popen([str(candidates[0]), str(repo)], cwd=project, env=env, stdout=log, stderr=subprocess.STDOUT)

            def auto(*args):
                result = subprocess.run([str(automation), *args, "--session", session], cwd=project,
                                        capture_output=True, text=True, timeout=40)
                trace.write(repr(args) + "\n" + result.stdout + result.stderr + "\n")
                trace.flush()
                if result.returncode:
                    raise RuntimeError(f"{args}: {result.stdout}{result.stderr}")
                return result.stdout

            def snapshot():
                return auto("snapshot", "--full")

            def ref(role, text):
                matches = [line for line in snapshot().splitlines()
                           if re.search(r'\[' + re.escape(role) + r'\] ' + re.escape(json.dumps(text, ensure_ascii=False)) + r'(?: |$)', line)
                           and "disabled" not in line]
                assert len(matches) == 1, (role, text, matches)
                return re.search(r"@e\d+", matches[0])[0]

            def press(role, text):
                auto("press", ref(role, text), "--settle")

            def wait(text):
                auto("wait", "--text", text, "--timeout", "20s", "--settle")

            def open_confirmation():
                auto("focus", ref("listItem", subject), "--settle")
                auto("key", "Shift+F10", "--settle")
                press("menuItem", label)
                wait(target[:8])
                assert target[:8] in snapshot() and '[button] "Cancel"' in snapshot()

            try:
                deadline = time.monotonic() + 20
                while not socket.exists():
                    if app.poll() is not None or time.monotonic() > deadline:
                        raise RuntimeError("native automation unavailable")
                    time.sleep(0.05)
                auto("open", "--socket", str(socket))
                auto("wait", "--text", "Commit summary (required)", "--timeout", "30s", "--settle")
                auto("key", "cmd+1", "--settle")
                wait("Integration fixture main")
                auto("wait", "--text", "Loading", "--gone", "--timeout", "20s", "--settle")
                open_confirmation()
                press("button", "Cancel")
                assert (git("show-ref"), git("ls-files", "--stage"), main_file.read_bytes(), git("status", "--porcelain=v1", "-z")) == before
                assert (repo / ".git/index").read_bytes() == before_index_bytes
                assert not topic_file.exists()
                open_confirmation()
                auto("screenshot", str(captures / "confirmation.png"))
                press("button", action)
                wait(success)
                assert git("rev-parse", "HEAD^").strip().decode() == original_head
                assert git("symbolic-ref", "--short", "HEAD").strip() == b"main"
                assert git("rev-parse", "refs/heads/topic").strip().decode() == topic
                assert not git("status", "--porcelain=v1", "-z")
                message = git("show", "-s", "--format=%B", "HEAD")
                if kind == "cherry-pick":
                    assert message.strip() == source_message
                    assert topic_file.read_bytes() == b"complete topic bytes\n"
                    assert main_file.read_bytes() == b"main committed version\n"
                else:
                    assert target.encode() in message and b'Revert "Integration fixture main"' in message
                    assert git("rev-parse", "HEAD^{tree}") == base_tree
                    assert main_file.read_bytes() == b"base\n" and not topic_file.exists()
                assert not any((repo / ".git" / name).exists() for name in ("CHERRY_PICK_HEAD", "REVERT_HEAD", "MERGE_HEAD", "sequencer"))
                (captures / "final.snapshot.txt").write_text(snapshot())
                auto("screenshot", str(captures / "final.png"))
                results.append({"kind": kind, "cancel_preserved_state": True, "expected_commit_parent": True,
                                "complete_message_and_bytes": True, "topic_preserved": True})
                print(f"PASS native {kind} initiation: cancel, selected SHA, parent, complete message and exact bytes", flush=True)
            except BaseException:
                try:
                    (captures / "failure.snapshot.txt").write_text(snapshot())
                except Exception:
                    pass
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
                        app.wait(timeout=10)


for kind in ("cherry-pick", "revert"):
    run_case(kind)
(artifacts / "results.json").write_text(json.dumps(results, indent=2) + "\n")
print(artifacts)
