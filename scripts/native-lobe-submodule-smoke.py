#!/usr/bin/env python3
"""Verify Lobe submodule init/update/dirty protection using local sources only."""
import json
import os
import pathlib
import re
import subprocess
import tempfile
import time

project = pathlib.Path(__file__).resolve().parent.parent
build = pathlib.Path(os.environ.get("SUPERGIT_NATIVE_BUILD_DIR", project / "build"))
automation = build / "mygo-automation"
binary = pathlib.Path(os.environ.get("SUPERGIT_NATIVE_BINARY", build / "supergit-lobe-e2e"))
if not automation.is_file() or not binary.is_file():
    raise SystemExit("Run make lobe-e2e-build first")
artifacts = pathlib.Path(tempfile.mkdtemp(prefix="lobe-submodule-", dir=build))
session = "supergit-lobe-submodule-" + str(os.getpid())
print("Artifacts: " + str(artifacts), flush=True)
dirty_guidance = "The submodule has local changes. Commit or stash them inside the submodule before updating. Update will not discard its contents."


def run():
    with tempfile.TemporaryDirectory(prefix="supergit-lobe-submodule.") as directory:
        root = pathlib.Path(directory)
        repo, source, config = root / "workspace", root / "child-source", root / "config"
        child = repo / "子 module"
        settings = config / "supergit"
        settings.mkdir(parents=True)
        (settings / "settings.json").write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False}))
        git_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
        git_env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0", GIT_ALLOW_PROTOCOL="file")

        def git(*args, directory=repo):
            return subprocess.check_output(["git", "-C", str(directory), *args], env=git_env, text=True)

        def init(directory):
            subprocess.run(["git", "init", "-b", "main", str(directory)], env=git_env, check=True, capture_output=True)
            git("config", "user.name", "Fixture", directory=directory)
            git("config", "user.email", "fixture@example.invalid", directory=directory)

        init(source)
        (source / "文件 space.txt").write_text("child base\n")
        git("add", "--all", directory=source)
        git("commit", "--no-gpg-sign", "-m", "Child fixture base", directory=source)
        base = git("rev-parse", "HEAD", directory=source).strip()
        (source / "文件 space.txt").write_text("child target\n")
        git("commit", "--no-gpg-sign", "-am", "Child fixture target", directory=source)
        target = git("rev-parse", "HEAD", directory=source).strip()
        init(repo)
        git("submodule", "add", "--name", "fixture-child", str(source), child.name)
        git("add", "--all")
        git("commit", "--no-gpg-sign", "-m", "Parent fixture target")
        parent_head, parent_index = git("rev-parse", "HEAD"), git("ls-files", "--stage")
        git("submodule", "deinit", "--", child.name)
        assert not (child / ".git").exists()
        socket = root / "app.sock"
        env = {**git_env, "SUPERGIT_TEST_CONFIG_DIR": str(config), "MYGO_AUTOMATION": str(socket), "SUPERGIT_WINDOW_SIZE": "1320x640"}
        with (artifacts / "app.log").open("w") as log, (artifacts / "automation.log").open("w") as trace:
            app = subprocess.Popen([str(binary), str(repo)], cwd=project, env=env, stdout=log, stderr=subprocess.STDOUT)

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

            def find(label, role="button"):
                matches = [e for e in controls() if e[1] == role and e[2] == label and "disabled" not in e[3]]
                if not matches:
                    raise AssertionError(f"Missing enabled {role} {label}: {controls()}")
                return matches[-1]

            def press(label, role="button"):
                auto("press", find(label, role)[0], "--settle")

            def wait(text):
                auto("wait", "--text", text, "--timeout", "25s", "--settle")

            def menu(action):
                deadline = time.monotonic() + 25
                while True:
                    ready()
                    auto("focus", find("fixture-child", "listItem")[0], "--settle")
                    auto("key", "shift+F10", "--settle")
                    current = controls()
                    target = next((e for e in current if e[1] == "menuItem" and e[2] == action and "disabled" not in e[3]), None)
                    repository_ready = any(e[1] == "button" and e[2] == "Refresh" and "disabled" not in e[3] for e in current)
                    if target is not None and repository_ready:
                        auto("press", target[0], "--settle")
                        return
                    auto("key", "escape", "--settle")
                    if time.monotonic() > deadline:
                        raise AssertionError(f"Submodule menu did not become ready for {action}")
                    time.sleep(0.05)

            def ready():
                deadline = time.monotonic() + 25
                while True:
                    if any(e[1] == "button" and e[2] == "Refresh" and "disabled" not in e[3] for e in controls()):
                        return
                    if time.monotonic() > deadline:
                        raise AssertionError("Repository did not finish loading")
                    time.sleep(0.05)

            def unchanged():
                assert git("rev-parse", "HEAD") == parent_head, "Submodule action changed parent HEAD"
                assert git("ls-files", "--stage") == parent_index, "Submodule action changed parent Index"

            def repository_state(directory):
                return {
                    "head": git("rev-parse", "HEAD", directory=directory),
                    "refs": git("show-ref", directory=directory),
                    "index_entries": git("ls-files", "--stage", "-z", directory=directory),
                    "status": git("--no-optional-locks", "status", "--porcelain=v1", "-z", directory=directory),
                }

            def no_parent_error(entries):
                for entry in entries:
                    assert "Could not update submodule" not in entry[2], "Parent error title or toast leaked into child repository"
                    assert dirty_guidance not in entry[2], "Parent English error body leaked into child repository"
                    assert "子模块内部有改动，请先提交或 Stash；更新不会丢弃内容" not in entry[2], "Parent original error body leaked into child repository"

            try:
                deadline = time.monotonic() + 20
                while not socket.exists():
                    if app.poll() is not None:
                        raise RuntimeError("Native app exited before automation connected")
                    if time.monotonic() > deadline:
                        raise TimeoutError("Native automation socket unavailable")
                    time.sleep(0.05)
                auto("open", "--socket", str(socket))
                auto("wait", "--text", "Commit summary (required)", "--timeout", "30s", "--settle")
                auto("key", "cmd+1", "--settle")
                wait("Parent fixture target")
                auto("key", "cmd+4", "--settle")
                wait("Not initialized")
                menu("Initialize submodule")
                wait("Submodule updated")
                ready()
                assert (child / ".git").is_file()
                assert git("rev-parse", "HEAD", directory=child).strip() == target
                assert (child / "文件 space.txt").read_text() == "child target\n"
                unchanged()
                print("PASS native Update initializes configured Unicode/space submodule at parent Index target", flush=True)

                git("checkout", "--detach", base, directory=child)
                press("Refresh")
                ready()
                menu("Update submodule")
                # Wait actual child HEAD, not a previous success toast.
                deadline = time.monotonic() + 25
                while git("rev-parse", "HEAD", directory=child).strip() != target:
                    if time.monotonic() > deadline:
                        raise AssertionError("Update did not restore Index target")
                    time.sleep(0.05)
                ready()
                assert (child / "文件 space.txt").read_text() == "child target\n"
                unchanged()
                print("PASS native Update restores moved child HEAD to parent Index without staging parent", flush=True)

                (child / "文件 space.txt").write_text("keep internal dirty bytes\n")
                child_index = git("ls-files", "--stage", directory=child)
                press("Refresh")
                ready()
                menu("Update submodule")
                wait("Could not update submodule")
                ready()
                assert (child / "文件 space.txt").read_text() == "keep internal dirty bytes\n"
                assert git("rev-parse", "HEAD", directory=child).strip() == target
                assert git("ls-files", "--stage", directory=child) == child_index
                unchanged()
                print("PASS native dirty submodule update refusal preserves child bytes/Index/HEAD and parent", flush=True)

                parent_error = controls()
                assert any(e[1] == "status" and "Could not update submodule" in e[2] for e in parent_error), "Parent refusal expired before root-switch check"
                assert any(e[1] == "text" and e[2] == dirty_guidance for e in parent_error), "Dirty submodule recovery does not use current English language"
                (artifacts / "parent-error.snapshot.txt").write_text(auto("snapshot", "--full"))
                auto("screenshot", str(artifacts / "parent-error.png"))
                before_switch = {"parent": repository_state(repo), "child": repository_state(child)}
                modules_before = (repo / ".gitmodules").read_bytes()
                dirty_before = (child / "文件 space.txt").read_bytes()
                menu("Open submodule repository")
                deadline = time.monotonic() + 25
                while not any(e[1] == "tab" and e[2] == child.name and " checked" in e[3] for e in controls()):
                    if time.monotonic() > deadline:
                        raise AssertionError("Child repository tab did not become active")
                    time.sleep(.05)
                ready()
                auto("key", "cmd+4", "--settle")
                wait("Worktrees and submodules")
                no_parent_error(controls())
                (artifacts / "child-worktrees.snapshot.txt").write_text(auto("snapshot", "--full"))
                auto("screenshot", str(artifacts / "child-worktrees.png"))
                current_child = find(child.name, "listItem")
                assert "Current" in current_child[3] and str(child) in current_child[3], "Child Worktrees page does not identify its real checkout as Current"
                assert ".git/modules/" not in current_child[3], "Child Worktrees page exposes git metadata as checkout"
                print("PASS native child Worktrees page identifies real checkout and Current state", flush=True)
                # Repository switching preserves the current page; open History
                # explicitly before asserting the child's commit list.
                press("History")
                wait("Child fixture target")
                ready()
                entries = controls()
                assert any(e[1] == "tab" and e[2] == child.name and " checked" in e[3] and str(child) in e[3] for e in entries), "Open did not switch root to child"
                assert not any(e[1] == "listItem" and e[2] == "Parent fixture target" for e in entries), "Parent history leaked into child"
                no_parent_error(entries)
                unchanged()
                assert {"parent": repository_state(repo), "child": repository_state(child)} == before_switch, "Root switch changed parent or child HEAD/refs/Index entries/status"
                assert (repo / ".gitmodules").read_bytes() == modules_before, "Root switch changed submodule configuration"
                assert (child / "文件 space.txt").read_bytes() == dirty_before, "Root switch changed dirty child bytes"
                (artifacts / "results.json").write_text(json.dumps({
                    "initialized_at_index_target": True,
                    "moved_child_restored_to_index": True,
                    "dirty_update_refused": True,
                    "dirty_guidance_in_current_english": True,
                    "parent_error_present_before_switch": True,
                    "parent_error_absent_in_child_worktrees_and_history": True,
                    "child_real_checkout_current": True,
                    "parent_and_child_state_preserved_on_switch": True,
                }, indent=2) + "\n")
                (artifacts / "final.snapshot.txt").write_text(auto("snapshot", "--full"))
                auto("screenshot", str(artifacts / "final.png"))
                print("PASS native Open submodule displays child history while preserving dirty bytes and parent", flush=True)
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
