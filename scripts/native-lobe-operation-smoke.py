#!/usr/bin/env python3
"""Drive Lobe operation cards after creating real conflicts in a temp fixture."""
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
artifacts = build / "lobe-operation-smoke"
artifacts.mkdir(parents=True, exist_ok=True)
session = "supergit-lobe-operation"


def run():
    with tempfile.TemporaryDirectory(prefix="supergit-lobe-operation.") as directory:
        root = pathlib.Path(directory)
        repo, home = root / "workspace", root / "home"
        settings = home / "Library/Application Support/supergit"
        settings.mkdir(parents=True)
        (settings / "settings.json").write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False}))
        git_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
        git_env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")

        def git(*args, check=True):
            result = subprocess.run(["git", "-C", str(repo), *args], env=git_env, capture_output=True, text=True)
            if check and result.returncode:
                raise AssertionError(f"git {args}: {result.stdout}{result.stderr}")
            return result

        subprocess.run(["git", "init", "-b", "main", str(repo)], env=git_env, check=True, capture_output=True)
        git("config", "user.name", "Fixture")
        git("config", "user.email", "fixture@example.invalid")
        path = repo / "文件 space.txt"
        path.write_text("base\n")
        git("add", "--all")
        git("commit", "--no-gpg-sign", "-m", "Operation fixture base")
        git("checkout", "-b", "side")
        path.write_text("side\n")
        git("commit", "--no-gpg-sign", "-am", "Operation side change", "-m", "Complete source body\n\nRefs: operation-fixture")
        side = git("rev-parse", "HEAD").stdout.strip()
        source_message = git("show", "-s", "--format=%B", side).stdout
        git("checkout", "main")
        path.write_text("main\n")
        git("commit", "--no-gpg-sign", "-am", "Operation main change", "-m", "Complete main body\n\nRefs: rebase-fixture")
        original_head = git("rev-parse", "HEAD").stdout
        original_index = git("ls-files", "--stage").stdout

        def start_conflict():
            result = git("cherry-pick", side, check=False)
            assert result.returncode != 0 and (repo / ".git/CHERRY_PICK_HEAD").exists()
            assert git("ls-files", "-u").stdout.strip(), "Fixture did not create unresolved conflict"

        start_conflict()
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
                entries = []
                for line in auto("snapshot", "--full").splitlines():
                    match = re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)', line)
                    if match:
                        entries.append((match[1], match[2], json.loads(match[3]) if match[3] else "", match[4]))
                return entries

            def press(label):
                matches = [e for e in controls() if e[1] == "button" and e[2] == label and "disabled" not in e[3]]
                if not matches:
                    raise AssertionError(f"Missing enabled button {label}: {controls()}")
                auto("press", matches[-1][0], "--settle")

            def wait(text):
                auto("wait", "--text", text, "--timeout", "25s", "--settle")

            def ready():
                deadline = time.monotonic() + 25
                while not any(e[1] == "button" and e[2] == "Refresh" and "disabled" not in e[3] for e in controls()):
                    if time.monotonic() > deadline:
                        raise AssertionError("Operation/refresh did not finish")
                    time.sleep(0.05)

            def fingerprint():
                return git("rev-parse", "HEAD").stdout, git("ls-files", "--stage").stdout, path.read_bytes()

            def no_operation():
                assert not (repo / ".git/CHERRY_PICK_HEAD").exists()
                assert not git("ls-files", "-u").stdout.strip()
                assert not git("status", "--porcelain").stdout.strip()

            try:
                deadline = time.monotonic() + 20
                while not socket.exists():
                    if app.poll() is not None:
                        raise RuntimeError("Native app exited before automation connected")
                    if time.monotonic() > deadline:
                        raise TimeoutError("Native automation socket unavailable")
                    time.sleep(0.05)
                auto("open", "--socket", str(socket))
                wait("Operation main change")
                press("In progress: cherry-pick")
                wait("In progress: cherry-pick")
                assert any(e[1] == "button" and e[2] == "Continue" and "disabled" in e[3] for e in controls()), "Continue enabled with unresolved conflict"
                before_cancel = fingerprint()
                press("Abort")
                wait("Abort the operation?")
                press("Cancel")
                assert fingerprint() == before_cancel and (repo / ".git/CHERRY_PICK_HEAD").exists(), "Abort cancel changed state"
                press("Abort")
                wait("Abort the operation?")
                press("Abort")
                wait("Aborted")
                ready()
                no_operation()
                assert fingerprint() == (original_head, original_index, b"main\n")
                print("PASS native conflict Continue disabled; Abort cancel preserves conflict and confirmed Abort restores original state", flush=True)

                start_conflict()
                press("Refresh")
                ready()
                wait("In progress: cherry-pick")
                before_skip = fingerprint()
                press("Skip")
                press("Cancel")
                assert fingerprint() == before_skip and (repo / ".git/CHERRY_PICK_HEAD").exists()
                press("Skip")
                press("Skip")
                ready()
                no_operation()
                assert fingerprint() == (original_head, original_index, b"main\n")
                print("PASS native Skip cancellation and confirmation preserve original branch while dropping interrupted cherry-pick", flush=True)

                start_conflict()
                path.write_text("resolved main and side\n")
                git("add", "--", path.name)
                press("Refresh")
                ready()
                wait("In progress: cherry-pick")
                before_continue = fingerprint()
                press("Continue")
                wait("Continued")
                ready()
                no_operation()
                assert git("rev-parse", "HEAD^").stdout == original_head
                assert git("show", "-s", "--format=%B", "HEAD").stdout == source_message, "Continue lost source message body"
                assert fingerprint()[1:] == before_continue[1:], "Continue changed resolved Index or working bytes"
                assert git("show", "HEAD:" + path.name).stdout == "resolved main and side\n"
                (artifacts / "final.snapshot.txt").write_text(auto("snapshot", "--full"))
                auto("screenshot", str(artifacts / "final.png"))
                print("PASS native Continue commits externally resolved Index, retains full source message and ends cherry-pick", flush=True)

                def rebase_conflict():
                    git("reset", "--hard", original_head.strip())
                    result = git("rebase", side, check=False)
                    assert result.returncode != 0 and (repo / ".git/rebase-merge").is_dir(), "Fixture did not start rebase"
                    assert git("ls-files", "-u").stdout.strip(), "Rebase did not create conflict stages"
                    press("Refresh")
                    ready()
                    wait("In progress: rebase")
                    press("In progress: rebase")
                    assert any(e[1] == "button" and e[2] == "Continue" and "disabled" in e[3] for e in controls()), "Rebase Continue enabled with unresolved files"

                def rebase_finished(expected_head=None):
                    no_operation()
                    assert not (repo / ".git/rebase-merge").exists() and not (repo / ".git/rebase-apply").exists()
                    assert git("symbolic-ref", "--short", "HEAD").stdout.strip() == "main"
                    if expected_head is not None:
                        assert git("rev-parse", "HEAD").stdout == expected_head

                rebase_conflict()
                before_abort = fingerprint()
                press("Abort")
                wait("Abort the operation?")
                press("Cancel")
                assert fingerprint() == before_abort and (repo / ".git/rebase-merge").is_dir()
                press("Abort")
                press("Abort")
                ready()
                rebase_finished(original_head)
                assert fingerprint()[1:] == (original_index, b"main\n")
                print("PASS native Rebase unresolved Continue disabled; Abort cancellation preserves conflict and confirmation restores original branch", flush=True)

                rebase_conflict()
                before_skip = fingerprint()
                press("Skip")
                press("Cancel")
                assert fingerprint() == before_skip and (repo / ".git/rebase-merge").is_dir()
                press("Skip")
                press("Skip")
                ready()
                rebase_finished(side + "\n")
                assert path.read_text() == "side\n"
                print("PASS native Rebase Skip cancel/confirm drops only interrupted replay and restores branch identity", flush=True)

                rebase_conflict()
                path.write_text("resolved rebase main and side\n")
                git("add", "--", path.name)
                press("Refresh")
                ready()
                before_rebase_continue = fingerprint()
                press("Continue")
                ready()
                rebase_finished()
                assert git("rev-parse", "HEAD^").stdout.strip() == side
                assert git("show", "-s", "--format=%B", "HEAD").stdout == git("show", "-s", "--format=%B", original_head.strip()).stdout
                assert fingerprint()[1:] == before_rebase_continue[1:], "Rebase Continue changed resolved Index or bytes"
                (artifacts / "final.snapshot.txt").write_text(auto("snapshot", "--full"))
                auto("screenshot", str(artifacts / "final.png"))
                print("PASS native Rebase Continue retains complete replay message and resolved bytes on expected parent", flush=True)

                def merge_conflict():
                    git("reset", "--hard", original_head.strip())
                    result = git("merge", "--no-edit", side, check=False)
                    assert result.returncode != 0 and (repo / ".git/MERGE_HEAD").is_file()
                    assert git("ls-files", "-u").stdout.strip()
                    press("Refresh")
                    ready()
                    wait("In progress: merge")
                    press("In progress: merge")
                    assert not any(e[1] == "button" and e[2] in ("Continue", "Skip") for e in controls()), "Merge exposed replay-only controls"

                def fill_field(label, value):
                    entries = [e for e in controls() if e[1] == "textField" and e[2] == label]
                    assert len(entries) == 1, f"Missing commit field {label}: {entries}"
                    auto("fill", entries[0][0], value, "--settle")

                merge_conflict()
                fill_field("Commit summary (required)", "Native merge resolution")
                assert any(e[1] == "button" and e[2] == "Commit" and "disabled" in e[3] for e in controls()), "Unresolved merge allowed Commit"
                before_merge_abort = fingerprint()
                press("Abort")
                press("Cancel")
                assert fingerprint() == before_merge_abort and (repo / ".git/MERGE_HEAD").is_file()
                press("Abort")
                press("Abort")
                ready()
                no_operation()
                assert not (repo / ".git/MERGE_HEAD").exists()
                assert fingerprint() == (original_head, original_index, b"main\n")
                print("PASS native Merge disables unresolved Commit; Abort cancel/confirm preserves then restores original state", flush=True)

                merge_conflict()
                path.write_text("resolved merge main and side\n")
                git("add", "--", path.name)
                press("Refresh")
                ready()
                fill_field("Commit summary (required)", "Native merge resolution")
                fill_field("Description (optional)", "Complete merge resolution body\n\nRefs: native-merge-fixture")
                resolved_merge_index = git("ls-files", "--stage").stdout
                press("Commit")
                wait("Committed")
                ready()
                no_operation()
                assert not (repo / ".git/MERGE_HEAD").exists()
                assert git("symbolic-ref", "--short", "HEAD").stdout.strip() == "main"
                parents = git("rev-list", "--parents", "-n", "1", "HEAD").stdout.split()
                assert parents[1:] == [original_head.strip(), side], f"Wrong merge parents: {parents}"
                assert git("show", "-s", "--format=%B", "HEAD").stdout.strip() == "Native merge resolution\n\nComplete merge resolution body\n\nRefs: native-merge-fixture"
                assert git("ls-files", "--stage").stdout == resolved_merge_index and path.read_text() == "resolved merge main and side\n"
                (artifacts / "final.snapshot.txt").write_text(auto("snapshot", "--full"))
                auto("screenshot", str(artifacts / "final.png"))
                print("PASS native Merge Commit retains both parents, complete entered message and externally resolved Index/bytes", flush=True)

                def revert_conflict():
                    git("reset", "--hard", original_head.strip())
                    result = git("revert", "--no-edit", side, check=False)
                    assert result.returncode != 0 and (repo / ".git/REVERT_HEAD").is_file()
                    assert git("ls-files", "-u").stdout.strip()
                    press("Refresh")
                    ready()
                    wait("In progress: revert")
                    press("In progress: revert")
                    assert any(e[1] == "button" and e[2] == "Continue" and "disabled" in e[3] for e in controls())

                for action in ("Abort", "Skip"):
                    revert_conflict()
                    before_control = fingerprint()
                    press(action)
                    press("Cancel")
                    assert fingerprint() == before_control and (repo / ".git/REVERT_HEAD").is_file()
                    press(action)
                    press(action)
                    ready()
                    no_operation()
                    assert not (repo / ".git/REVERT_HEAD").exists() and not (repo / ".git/sequencer").exists()
                    assert fingerprint() == (original_head, original_index, b"main\n")
                    print(f"PASS native Revert {action} cancel preserves conflict; confirm restores original HEAD/Index/bytes", flush=True)

                revert_conflict()
                path.write_text("resolved revert main and side\n")
                git("add", "--", path.name)
                press("Refresh")
                ready()
                before_revert_continue = fingerprint()
                press("Continue")
                ready()
                no_operation()
                assert not (repo / ".git/REVERT_HEAD").exists() and not (repo / ".git/sequencer").exists()
                assert git("symbolic-ref", "--short", "HEAD").stdout.strip() == "main"
                assert git("rev-parse", "HEAD^").stdout == original_head
                expected_revert_message = f'Revert "Operation side change"\n\nThis reverts commit {side}.'
                assert git("show", "-s", "--format=%B", "HEAD").stdout.strip() == expected_revert_message
                assert fingerprint()[1:] == before_revert_continue[1:], "Revert Continue changed resolved Index or bytes"
                (artifacts / "final.snapshot.txt").write_text(auto("snapshot", "--full"))
                auto("screenshot", str(artifacts / "final.png"))
                print("PASS native Revert Continue keeps original parent, generated full message and externally resolved Index/bytes", flush=True)
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
