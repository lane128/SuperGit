#!/usr/bin/env python3
"""Verify Lobe sync against a disposable local bare remote and isolated settings."""
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
artifacts = build / "lobe-sync-smoke"
artifacts.mkdir(parents=True, exist_ok=True)
session = "supergit-lobe-sync"


def run():
    with tempfile.TemporaryDirectory(prefix="supergit-lobe-sync.") as directory:
        root = pathlib.Path(directory)
        repo, remote, peer, home = (root / name for name in ("workspace", "remote.git", "peer", "home"))
        settings = home / "Library/Application Support/supergit"
        settings.mkdir(parents=True)
        (settings / "settings.json").write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False}))
        git_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
        git_env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")

        def git(*args, directory=repo):
            return subprocess.check_output(["git", "-C", str(directory), *args], env=git_env, text=True)

        subprocess.run(["git", "init", "--bare", "-b", "main", str(remote)], env=git_env, check=True, capture_output=True)
        subprocess.run(["git", "init", "-b", "main", str(repo)], env=git_env, check=True, capture_output=True)
        git("config", "user.name", "Fixture")
        git("config", "user.email", "fixture@example.invalid")
        (repo / "文件 space.txt").write_text("base\n")
        git("add", "--all")
        git("commit", "--no-gpg-sign", "-m", "Sync fixture base")
        git("remote", "add", "origin", str(remote))
        original_head = git("rev-parse", "HEAD").strip()
        assert not git("for-each-ref", "--format=%(refname)", "refs/heads", directory=remote).strip()
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

            def wait_git(args, expected, directory=repo):
                deadline = time.monotonic() + 25
                while git(*args, directory=directory).strip() != expected:
                    if time.monotonic() > deadline:
                        raise AssertionError(f"Git state did not reach {args}: {expected}")
                    if app.poll() is not None:
                        raise AssertionError("Native app exited during sync")
                    time.sleep(0.05)

            try:
                deadline = time.monotonic() + 20
                while not socket.exists():
                    if app.poll() is not None:
                        raise RuntimeError("Native app exited before automation connected")
                    if time.monotonic() > deadline:
                        raise TimeoutError("Native automation socket unavailable")
                    time.sleep(0.05)
                auto("open", "--socket", str(socket))
                wait("Sync fixture base")
                assert any(e[2] == "Pull (fast-forward only)" and "disabled" in e[3] for e in controls()), "Pull enabled without upstream"
                press("Push")
                wait(f"Push main at commit {original_head[:8]} to origin/main.")
                press("Cancel")
                assert not git("for-each-ref", "--format=%(refname)", "refs/heads", directory=remote).strip(), "Cancelled Push wrote remote"
                assert not git("for-each-ref", "--format=%(upstream)", "refs/heads/main").strip(), "Cancelled Push set upstream"
                press("Push")
                wait(f"Push main at commit {original_head[:8]} to origin/main.")
                press("Push")
                wait_git(("for-each-ref", "--format=%(objectname)", "refs/heads/main"), original_head, directory=remote)
                wait("Pushed")
                assert git("config", "branch.main.remote").strip() == "origin"
                assert git("config", "branch.main.merge").strip() == "refs/heads/main"
                assert git("rev-parse", "HEAD").strip() == original_head
                assert not git("status", "--porcelain").strip()
                print("PASS native Push cancel, first Push local remote and upstream creation", flush=True)

                subprocess.run(["git", "clone", str(remote), str(peer)], env=git_env, check=True, capture_output=True)
                git("config", "user.name", "Peer Fixture", directory=peer)
                git("config", "user.email", "peer@example.invalid", directory=peer)
                (peer / "文件 space.txt").write_text("external fast-forward\n")
                git("commit", "--no-gpg-sign", "-am", "Sync peer advance", directory=peer)
                git("push", "origin", "main", directory=peer)
                peer_head = git("rev-parse", "HEAD", directory=peer).strip()
                index_before = git("ls-files", "--stage")
                press("Fetch")
                wait("Fetched from remotes")
                wait_git(("rev-parse", "refs/remotes/origin/main"), peer_head)
                assert git("rev-parse", "HEAD").strip() == original_head, "Fetch changed HEAD"
                assert git("ls-files", "--stage") == index_before, "Fetch changed Index"
                assert (repo / "文件 space.txt").read_text() == "base\n", "Fetch changed working file"
                press("Pull (fast-forward only)")
                wait("Fast-forwarded to upstream")
                wait_git(("rev-parse", "HEAD"), peer_head)
                assert (repo / "文件 space.txt").read_text() == "external fast-forward\n"
                assert not git("status", "--porcelain").strip()
                print("PASS native Fetch preserves HEAD/Index/worktree and Pull fast-forwards Unicode path", flush=True)

                # Divergence must fail without merging, resetting, or dropping local bytes.
                (repo / "local.txt").write_text("local only\n")
                git("add", "--", "local.txt")
                git("commit", "--no-gpg-sign", "-m", "Sync local divergence")
                (peer / "remote.txt").write_text("remote only\n")
                git("add", "--", "remote.txt", directory=peer)
                git("commit", "--no-gpg-sign", "-m", "Sync remote divergence", directory=peer)
                git("push", "origin", "main", directory=peer)
                press("Refresh")
                wait("Sync local divergence")
                local_head, local_index = git("rev-parse", "HEAD"), git("ls-files", "--stage")
                press("Pull (fast-forward only)")
                wait("Pull failed")
                assert git("rev-parse", "HEAD") == local_head, "Non-fast-forward Pull rewrote HEAD"
                assert git("ls-files", "--stage") == local_index, "Failed Pull changed Index"
                assert (repo / "local.txt").read_text() == "local only\n"
                assert not (repo / "remote.txt").exists()
                assert not (repo / ".git/MERGE_HEAD").exists(), "FF-only Pull started merge"
                assert not git("status", "--porcelain").strip()
                auto("key", "cmd+2", "--settle")
                local_rows = [e for e in controls() if e[1] == "listItem" and e[2] == "main" and "Current" in e[3]]
                assert local_rows, "Local main row missing from Branches page"
                auto("press", local_rows[0][0], "--settle")
                wait("Copy branch name")
                entries = controls()
                copies = [e for e in entries if e[1] == "button" and e[2] == "Copy branch name"]
                errors = [e for e in entries if e[1] == "status" and "Pull failed" in e[2]]
                assert copies and errors, "Footer check requires both copy control and visible Pull error"

                def bounds(entry):
                    match = re.search(r"bounds=([-0-9.]+),([-0-9.]+),([0-9.]+)x([0-9.]+)", entry[3])
                    assert match, f"Missing bounds: {entry}"
                    return tuple(float(value) for value in match.groups())

                x, y, width, height = bounds(copies[-1])
                _, error_y, _, error_height = bounds(errors[-1])
                assert error_y + error_height <= y, "Pull error overlaps branch footer"
                auto("screenshot", str(artifacts / "branch-footer.png"))
                auto("click", str(x + width / 2), str(y + height / 2), "--settle")
                wait("Copied")
                assert git("rev-parse", "HEAD") == local_head and git("ls-files", "--stage") == local_index, "Footer copy changed repository"
                print("PASS native visible Pull error leaves branch footer reachable by coordinate click", flush=True)
                (artifacts / "final.snapshot.txt").write_text(auto("snapshot", "--full"))
                auto("screenshot", str(artifacts / "final.png"))
                print("PASS native divergent Pull fails without HEAD/Index/worktree changes or implicit merge", flush=True)

                # Two local remotes: an explicit upstream must determine both
                # remote and destination, even when the local branch name differs.
                backup = root / "backup.git"
                subprocess.run(["git", "init", "--bare", "-b", "main", str(backup)], env=git_env, check=True, capture_output=True)
                git("remote", "add", "backup", str(backup))
                destination = "publish/nested"
                git("push", "backup", f"{original_head}:refs/heads/{destination}")
                git("branch", f"--set-upstream-to=backup/{destination}", "main")
                origin_refs = git("for-each-ref", "--format=%(refname) %(objectname)", directory=remote)
                backup_refs = git("for-each-ref", "--format=%(refname) %(objectname)", directory=backup)
                # Push and Fetch must also preserve staged and unstaged bytes.
                staged = repo / "staged local.txt"
                staged.write_bytes(b"keep staged bytes\x00\xff\n")
                git("add", "--", staged.name)
                (repo / "文件 space.txt").write_text("keep unstaged Unicode file\n")
                protected_index = git("ls-files", "--stage")
                protected_status = git("status", "--porcelain", "-z")

                def ready():
                    deadline = time.monotonic() + 25
                    while not any(e[1] == "button" and e[2] == "Refresh" and "disabled" not in e[3] for e in controls()):
                        if time.monotonic() > deadline:
                            raise AssertionError("Refresh did not finish")
                        time.sleep(0.05)

                def protected_local():
                    assert git("rev-parse", "HEAD") == local_head
                    assert git("symbolic-ref", "--short", "HEAD").strip() == "main"
                    assert git("ls-files", "--stage") == protected_index
                    assert git("status", "--porcelain", "-z") == protected_status
                    assert staged.read_bytes() == b"keep staged bytes\x00\xff\n"
                    assert (repo / "文件 space.txt").read_text() == "keep unstaged Unicode file\n"

                press("Refresh")
                ready()
                press("Push")
                wait(f"Push main at commit {local_head.strip()[:8]} to backup/{destination}.")
                press("Cancel")
                assert git("for-each-ref", "--format=%(refname) %(objectname)", directory=remote) == origin_refs
                assert git("for-each-ref", "--format=%(refname) %(objectname)", directory=backup) == backup_refs
                protected_local()
                press("Push")
                wait(f"Push main at commit {local_head.strip()[:8]} to backup/{destination}.")
                press("Push")
                wait_git(("for-each-ref", "--format=%(objectname)", f"refs/heads/{destination}"), local_head.strip(), directory=backup)
                wait("Pushed")
                ready()
                assert git("for-each-ref", "--format=%(refname) %(objectname)", directory=remote) == origin_refs
                assert not git("for-each-ref", "--format=%(refname)", "refs/heads/main", directory=backup).strip()
                assert git("config", "branch.main.remote").strip() == "backup"
                assert git("config", "branch.main.merge").strip() == f"refs/heads/{destination}"
                protected_local()
                auto("screenshot", str(artifacts / "multi-remote-push.png"))
                print("PASS native multi-remote Push cancel/publish follows configured upstream destination and preserves other remote plus staged/unstaged bytes", flush=True)

                # Advance independently named refs on both remotes so Fetch must
                # read both; no network service or user repository participates.
                origin_tip = git("rev-parse", "refs/heads/main", directory=remote).strip()
                git("update-ref", "refs/heads/origin-only", origin_tip, directory=remote)
                git("update-ref", "refs/heads/backup-only", local_head.strip(), directory=backup)
                press("Fetch")
                wait("Fetched from remotes")
                ready()
                wait_git(("rev-parse", "refs/remotes/origin/origin-only"), origin_tip)
                wait_git(("rev-parse", "refs/remotes/backup/backup-only"), local_head.strip())
                protected_local()
                (artifacts / "final.snapshot.txt").write_text(auto("snapshot", "--full"))
                auto("screenshot", str(artifacts / "final.png"))
                print("PASS native Fetch reads both local remotes while preserving HEAD/branch/Index and staged/unstaged file bytes", flush=True)
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
