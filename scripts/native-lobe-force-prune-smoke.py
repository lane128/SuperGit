#!/usr/bin/env python3
"""Verify Force/prune app-rendered menus using an owned local bare remote."""
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
artifacts = pathlib.Path(tempfile.mkdtemp(prefix="lobe-force-prune-", dir=build))
with tempfile.TemporaryDirectory(prefix="supergit-force-prune.") as directory:
    root = pathlib.Path(directory)
    repo, remote, home = (root / name for name in ("repo", "remote.git", "home"))
    settings = home / "Library/Application Support/supergit/settings.json"
    settings.parent.mkdir(parents=True)
    settings.write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False}))
    env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
    env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")

    def git(*args, at=repo):
        return subprocess.check_output(["git", "-C", str(at), *args], env=env)

    for target, flags in ((repo, []), (remote, ["--bare"])):
        subprocess.run(["git", "init", *flags, "-b", "main", str(target)], env=env, check=True, capture_output=True)
        git("config", "user.name", "Fixture", at=target)
        git("config", "user.email", "fixture@example.invalid", at=target)
    path = repo / "文件 space.txt"
    path.write_text("base\n")
    git("add", "--", path.name)
    git("commit", "-m", "Force fixture base")
    base = git("rev-parse", "HEAD").strip().decode()
    git("remote", "add", "origin", str(remote))
    git("push", "-u", "origin", "main")
    git("branch", "ghost", base)
    git("update-ref", "refs/heads/ghost", base, at=remote)
    remote_tree = git("rev-parse", "HEAD^{tree}", at=remote).strip().decode()
    remote_tip = git("commit-tree", remote_tree, "-p", base, "-m", "Remote divergent fixture", at=remote).strip().decode()
    git("update-ref", "refs/heads/main", remote_tip, base, at=remote)
    path.write_text("local committed version\n")
    git("commit", "-am", "Force fixture source")
    source = git("rev-parse", "HEAD").strip().decode()
    git("fetch", "origin")
    path.write_text("local staged version\n")
    git("add", "--", path.name)
    path.write_text("local working version\n")
    untracked = repo / "untracked.bin"
    untracked.write_bytes(b"keep\x00\xff")

    def local_state():
        return (git("rev-parse", "HEAD"), git("symbolic-ref", "HEAD"),
                git("for-each-ref", "--format=%(refname) %(objectname)", "refs/heads"),
                git("ls-files", "--stage"), git("status", "--porcelain=v1", "-z"),
                (repo / ".git/index").read_bytes(), path.read_bytes(), untracked.read_bytes(),
                git("config", "--get-regexp", r"^branch\.main\."))

    before = local_state()
    initial_remote_refs = git("show-ref", at=remote)
    socket = root / "app.sock"
    session = "force-prune-" + str(os.getpid())
    env.update(HOME=str(home), MYGO_AUTOMATION=str(socket), SUPERGIT_WINDOW_SIZE="1320x640")
    with (artifacts / "app.log").open("w") as log, (artifacts / "automation.log").open("w") as trace:
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

        def ref(role, label):
            matches = [line for line in snapshot().splitlines()
                       if re.search(r'\[' + re.escape(role) + r'\] ' + re.escape(json.dumps(label, ensure_ascii=False)) + r'(?: |$)', line)
                       and "disabled" not in line]
            assert len(matches) == 1, (role, label, matches)
            return re.search(r"@e\d+", matches[0])[0]

        def press(role, label):
            deadline = time.monotonic() + 25
            while True:
                text = snapshot()
                matches = [line for line in text.splitlines()
                           if re.search(r'\[' + re.escape(role) + r'\] ' + re.escape(json.dumps(label, ensure_ascii=False)) + r'(?: |$)', line)]
                assert len(matches) == 1, (role, label, matches)
                if "disabled" not in matches[0]:
                    auto("press", re.search(r"@e\d+", matches[0])[0], "--settle")
                    return
                if time.monotonic() > deadline:
                    raise AssertionError(f"{label} remained disabled")
                time.sleep(0.05)

        def wait(text):
            auto("wait", "--text", text, "--timeout", "25s", "--settle")

        def menu(button, item):
            press("menuButton", "Sync")
            (artifacts / (button.lower() + "-menu.snapshot.txt")).write_text(snapshot())
            press("menuItem", item)

        def force_confirmation(expected):
            menu("Push", "Force push…")
            wait("Replace origin/main")
            text = snapshot()
            assert f"expected: {expected[:8]}" in text and source[:8] in text, "confirmation has incorrect source or lease"

        try:
            deadline = time.monotonic() + 20
            while not socket.exists():
                if app.poll() is not None or time.monotonic() > deadline:
                    raise RuntimeError("native automation unavailable")
                time.sleep(0.05)
            auto("open", "--socket", str(socket))
            auto("key", "cmd+1", "--settle")
            wait("Force fixture source")
            auto("wait", "--text", "Loading", "--gone", "--timeout", "20s", "--settle")
            force_confirmation(remote_tip)
            auto("screenshot", str(artifacts / "force-confirmation.png"))
            press("button", "Cancel")
            assert git("show-ref", at=remote) == initial_remote_refs
            assert local_state() == before
            force_confirmation(remote_tip)
            press("button", "Force push")
            wait("Force pushed")
            assert git("rev-parse", "refs/heads/main", at=remote).strip().decode() == source
            assert git("rev-parse", "refs/heads/ghost", at=remote).strip().decode() == base
            assert local_state() == before
            print("PASS native Force push menu, exact lease/source, cancel and divergent replacement preserve local dirty state", flush=True)

            force_confirmation(source)
            external_tip = git("commit-tree", remote_tree, "-p", source, "-m", "External remote fixture advance", at=remote).strip().decode()
            git("update-ref", "refs/heads/main", external_tip, source, at=remote)
            # Updating tracking refs after consent must not broaden its lease.
            git("fetch", "origin")
            remote_before_rejection = git("show-ref", at=remote)
            press("button", "Force push")
            wait("Force push failed")
            press("button", "View details")
            wait("The remote branch changed or force push was refused.")
            assert "stale info" in snapshot(), "details omitted the original Git lease rejection"
            assert git("show-ref", at=remote) == remote_before_rejection
            assert local_state() == before
            (artifacts / "lease-rejected.snapshot.txt").write_text(snapshot())
            auto("screenshot", str(artifacts / "lease-rejected.png"))
            print("PASS native old Force lease rejects remote advance even after fetch; remote and local dirty state preserved", flush=True)
            auto("key", "Escape", "--settle")

            assert git("rev-parse", "refs/remotes/origin/ghost").strip().decode() == base
            git("update-ref", "-d", "refs/heads/ghost", at=remote)
            expected_remote_refs = git("show-ref", at=remote)
            menu("Fetch", "Fetch and prune deleted remote branches")
            wait("Fetched and pruned remotes")
            assert not git("for-each-ref", "--format=%(refname)", "refs/remotes/origin/ghost").strip()
            assert git("rev-parse", "refs/remotes/origin/main").strip().decode() == external_tip
            assert git("show-ref", at=remote) == expected_remote_refs
            assert local_state() == before, "prune changed local heads/Index/bytes/upstream"
            (artifacts / "final.snapshot.txt").write_text(snapshot())
            auto("screenshot", str(artifacts / "final.png"))
            (artifacts / "results.json").write_text(json.dumps({"force_cancel": True, "force_replace": True,
                "stale_lease_after_fetch_rejected": True, "prune_only_tracking_ref": True,
                "local_dirty_state_preserved": True}, indent=2) + "\n")
            print("PASS native Fetch prune removes stale tracking ref, retains same-named local branch and preserves local dirty state", flush=True)
            print(artifacts)
        except BaseException:
            try:
                (artifacts / "failure.snapshot.txt").write_text(snapshot())
            except Exception:
                pass
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
                    app.wait(timeout=10)
