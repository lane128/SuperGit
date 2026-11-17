#!/usr/bin/env python3
"""Resolve isolated text, binary or deletion conflicts through Lobe side actions."""
import argparse
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
parser = argparse.ArgumentParser()
parser.add_argument("--rebase", action="store_true", help="Verify rebase side identities, then Abort back to the original branch")
parser.add_argument("--shape", choices=("text", "binary", "deletion"), default="text", help="Conflict contents; deletion removes the original main side")
parser.add_argument("--rebase-continue", action="store_true", help="Verify rebase side identities, then finish with the operation card Continue action")
options = parser.parse_args()
options.rebase = options.rebase or options.rebase_continue
kind = "rebase" if options.rebase else "merge"
artifacts = pathlib.Path(tempfile.mkdtemp(prefix="lobe-conflict-side-", dir=build))
results = []
for side in ("ours", "theirs"):
    captures = artifacts / side
    captures.mkdir()
    with tempfile.TemporaryDirectory(prefix="supergit-conflict-side.") as directory:
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
        git("config", "user.name", "Conflict Fixture")
        git("config", "user.email", "conflict@example.invalid")
        path = repo / "冲突 space.txt"
        untouched = repo / "untouched.bin"
        binary = options.shape == "binary"
        base_bytes = b"base\x00\xff\n" if binary else b"base\n"
        incoming_bytes = b"incoming\x00\xfe\n" if binary else b"incoming version\n"
        current_bytes = b"current\x00\xfd\n" if binary else b"current version\n"
        path.write_bytes(base_bytes)
        untouched.write_bytes(b"\x00unchanged\xff\n")
        git("add", "--", path.name, untouched.name)
        git("commit", "-m", "Conflict fixture base")
        git("checkout", "-b", "topic")
        path.write_bytes(incoming_bytes)
        git("commit", "-am", "Conflict fixture incoming")
        incoming = git("rev-parse", "HEAD").strip().decode()
        git("checkout", "main")
        if options.shape == "deletion":
            git("rm", "--", path.name)
        else:
            path.write_bytes(current_bytes)
        git("commit", "-am", "Conflict fixture current")
        original = git("rev-parse", "HEAD").strip().decode()
        original_message = git("show", "-s", "--format=%B", "HEAD")
        original_author = git("show", "-s", "--format=%an%n%ae%n%aI", "HEAD")
        operation_args = ["rebase", "topic"] if options.rebase else ["merge", "--no-edit", "topic"]
        merged = subprocess.run(["git", "-C", str(repo), *operation_args], env=env, capture_output=True)
        assert merged.returncode and git("ls-files", "-u"), "fixture must create an unmerged Index"
        stage = b"2" if side == "ours" else b"3"
        stages = git("ls-files", "-u", "--", path.name)
        side_exists = any(record.split(b"\t", 1)[0].split()[-1] == stage for record in stages.splitlines())
        expected = git("show", ":" + stage.decode() + ":" + path.name) if side_exists else None

        def assert_resolved_side():
            if expected is None:
                assert not path.exists() and not git("ls-files", "--stage", "--", path.name), "Missing side did not stage deletion"
            else:
                assert path.read_bytes() == expected and git("show", ":" + path.name) == expected

        original_refs = git("show-ref")
        socket = root / "app.sock"
        session = "conflict-side-" + side + "-" + str(os.getpid())
        env.update(HOME=str(home), MYGO_AUTOMATION=str(socket), SUPERGIT_WINDOW_SIZE="1320x640")
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

            def ref(role, label):
                lines = [line for line in snapshot().splitlines() if re.search(r'\[' + role + r'\] ' + re.escape(json.dumps(label, ensure_ascii=False)) + r'(?: |$)', line) and "disabled" not in line]
                assert len(lines) == 1, (role, label, lines)
                return re.search(r"@e\d+", lines[0])[0]

            def press(role, label):
                auto("press", ref(role, label), "--settle")

            def wait(text):
                auto("wait", "--text", text, "--timeout", "25s", "--settle")

            try:
                deadline = time.monotonic() + 20
                while not socket.exists():
                    if app.poll() is not None or time.monotonic() > deadline:
                        raise RuntimeError("native automation unavailable")
                    time.sleep(0.05)
                auto("open", "--socket", str(socket))
                auto("wait", "--text", "Commit summary (required)", "--timeout", "30s", "--settle")
                auto("key", "cmd+1", "--settle")
                wait("In progress: " + kind)
                press("button", "In progress: " + kind)
                wait(path.name)
                wait("Conflict preview only. Resolve text or binary files with an external tool, then use Mark resolved in the file menu.")
                if options.shape == "deletion":
                    wait("This conflict includes a deletion. Choosing a side with no file deletes and stages this path.")
                guidance = "Conflict preview only. Resolve text or binary files with an external tool, then use Mark resolved in the file menu."
                reveal_text = "This conflict includes a deletion. Choosing a side with no file deletes and stages this path." if options.shape == "deletion" else guidance
                auto("act", ref("text", reveal_text), "scrollIntoView", "--settle")
                visible_snapshot = snapshot()
                assert any(json.dumps(guidance) in line and "offscreen" not in line for line in visible_snapshot.splitlines()), "Conflict guidance cannot be revealed"
                if options.shape == "deletion":
                    assert any('"This conflict includes a deletion.' in line and "offscreen" not in line for line in visible_snapshot.splitlines()), "Deletion consequence is not visible beside guidance"
                (captures / "unresolved.snapshot.txt").write_text(visible_snapshot)
                auto("screenshot", str(captures / "unresolved.png"))
                auto("fill", ref("textField", "Commit summary (required)"), "Native " + side + " merge resolution", "--settle")
                assert any('[button] "Commit"' in line and "disabled" in line for line in snapshot().splitlines()), "Unresolved merge enabled Commit"
                auto("focus", ref("listItem", path.name), "--settle")
                auto("key", "Shift+F10", "--settle")
                label = "Use ours (current branch)" if side == "ours" else "Use theirs (incoming)"
                success = "Took our version" if side == "ours" else "Took their version"
                if options.rebase:
                    label = "Use ours (rebased base)" if side == "ours" else "Use theirs (replayed commit)"
                    success = "Took the rebased base version" if side == "ours" else "Took the replayed commit version"
                    assert expected == (incoming_bytes if side == "ours" else (None if options.shape == "deletion" else current_bytes)), "Rebase stage meanings changed"
                press("menuItem", label)
                wait(success)
                assert not git("ls-files", "-u"), "Side selection left conflict stages"
                assert_resolved_side()
                assert git("show-ref") == original_refs, "Resolving a file moved refs"
                if options.rebase:
                    assert git("rev-parse", "HEAD").strip().decode() == incoming
                    assert (repo / ".git/rebase-merge").is_dir() or (repo / ".git/rebase-apply").is_dir()
                else:
                    assert (repo / ".git/MERGE_HEAD").read_text().strip() == incoming
                assert untouched.read_bytes() == b"\x00unchanged\xff\n"
                if options.rebase:
                    (captures / "resolved.snapshot.txt").write_text(snapshot())
                    auto("screenshot", str(captures / "resolved.png"))
                    if options.rebase_continue:
                        deadline = time.monotonic() + 25
                        while not any('[button] "Continue"' in line and "disabled" not in line for line in snapshot().splitlines()):
                            if time.monotonic() > deadline:
                                raise AssertionError("Rebase refresh did not restore Continue readiness")
                            time.sleep(0.05)
                        press("button", "Continue")
                        wait("Continued")
                        assert git("symbolic-ref", "--short", "HEAD").strip() == b"main"
                        assert git("rev-parse", "topic").strip().decode() == incoming
                        if side == "ours":
                            assert git("rev-parse", "HEAD").strip().decode() == incoming, "Empty replay should retain rebased base"
                        else:
                            assert git("rev-list", "--parents", "-n", "1", "HEAD").decode().split()[1:] == [incoming]
                            assert git("show", "-s", "--format=%B", "HEAD") == original_message
                            assert git("show", "-s", "--format=%an%n%ae%n%aI", "HEAD") == original_author
                        assert_resolved_side()
                        assert untouched.read_bytes() == b"\x00unchanged\xff\n"
                        assert not git("status", "--porcelain=v1", "-z") and not git("ls-files", "-u")
                        assert not (repo / ".git/rebase-merge").exists() and not (repo / ".git/rebase-apply").exists()
                        (captures / "final.snapshot.txt").write_text(snapshot())
                        auto("screenshot", str(captures / "final.png"))
                        results.append({"kind": kind, "shape": options.shape, "side": side, "ui_resolution_staged_exact_side": True, "ui_continue_completed": True, "empty_replay_dropped": side == "ours", "replayed_metadata_preserved": side == "theirs", "topic_and_untouched_binary_preserved": True})
                        print("PASS native rebase UI " + side + ": exact side staged, Continue completed, expected ancestry and contents", flush=True)
                        continue
                    deadline = time.monotonic() + 25
                    while not any('[button] "Abort"' in line and "disabled" not in line for line in snapshot().splitlines()):
                        if time.monotonic() > deadline:
                            raise AssertionError("Rebase refresh did not restore Abort readiness")
                        time.sleep(0.05)
                    press("button", "Abort")
                    press("button", "Abort")
                    wait("Aborted")
                    assert git("symbolic-ref", "--short", "HEAD").strip() == b"main"
                    assert git("rev-parse", "HEAD").strip().decode() == original
                    assert git("show-ref") == original_refs
                    if options.shape == "deletion":
                        assert not path.exists(), "Abort restored a file originally deleted on main"
                    else:
                        assert path.read_bytes() == current_bytes
                    assert untouched.read_bytes() == b"\x00unchanged\xff\n"
                    assert not git("status", "--porcelain=v1", "-z") and not git("ls-files", "-u")
                    assert not (repo / ".git/rebase-merge").exists() and not (repo / ".git/rebase-apply").exists()
                    results.append({"kind": kind, "shape": options.shape, "side": side, "rebase_stage_identity": True, "localized_success": success, "ui_resolution_staged_exact_side": True, "abort_restored_original_branch": True})
                    print("PASS native rebase UI " + side + ": actual stage identity, exact staged bytes, localized success, Abort restoration", flush=True)
                    continue
                auto("fill", ref("textField", "Description (optional)"), "Complete UI side resolution\n\nRefs: native-conflict-side", "--settle")
                deadline = time.monotonic() + 25
                while True:
                    if any('[button] "Commit"' in line and "disabled" not in line for line in snapshot().splitlines()):
                        break
                    if time.monotonic() > deadline:
                        raise AssertionError("Resolved merge did not enable Commit")
                    time.sleep(0.05)
                press("button", "Commit")
                wait("Committed")
                assert git("rev-list", "--parents", "-n", "1", "HEAD").decode().split()[1:] == [original, incoming]
                assert git("show", "-s", "--format=%B", "HEAD").decode().strip() == "Native " + side + " merge resolution\n\nComplete UI side resolution\n\nRefs: native-conflict-side"
                assert git("symbolic-ref", "--short", "HEAD").strip() == b"main"
                assert git("rev-parse", "topic").strip().decode() == incoming
                assert not git("status", "--porcelain=v1", "-z") and not git("ls-files", "-u")
                assert_resolved_side()
                assert untouched.read_bytes() == b"\x00unchanged\xff\n"
                assert not (repo / ".git/MERGE_HEAD").exists()
                (captures / "final.snapshot.txt").write_text(snapshot())
                auto("screenshot", str(captures / "final.png"))
                results.append({"kind": kind, "shape": options.shape, "side": side, "ui_resolution_staged_exact_side": True, "refs_preserved_until_commit": True, "merge_parents_and_message": True, "untouched_binary_preserved": True})
                print("PASS native merge UI " + side + ": exact side staged, refs protected, full Commit and two parents", flush=True)
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
