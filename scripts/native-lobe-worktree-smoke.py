#!/usr/bin/env python3
"""Exercise Lobe worktree creation, switching and guarded removal in a temp repo."""
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
artifacts = build / "lobe-worktree-smoke"
artifacts.mkdir(parents=True, exist_ok=True)
session = "supergit-lobe-worktree"


def run():
    with tempfile.TemporaryDirectory(prefix="supergit-lobe-worktree.") as directory:
        root = pathlib.Path(directory)
        repo, checkout, home = root / "workspace", root / "review space", root / "home"
        settings = home / "Library/Application Support/supergit"
        settings.mkdir(parents=True)
        (settings / "settings.json").write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False}))
        git_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
        git_env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")

        def git(*args, directory=repo):
            return subprocess.check_output(["git", "-C", str(directory), *args], env=git_env, text=True)

        subprocess.run(["git", "init", "-b", "main", str(repo)], env=git_env, check=True, capture_output=True)
        git("config", "user.name", "Fixture")
        git("config", "user.email", "fixture@example.invalid")
        (repo / "文件 space.txt").write_text("base\n")
        git("add", "--all")
        git("commit", "--no-gpg-sign", "-m", "Worktree fixture base")
        original_head, original_index = git("rev-parse", "HEAD"), git("ls-files", "--stage")
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

            def find(label, role="button"):
                matches = [e for e in controls() if e[1] == role and e[2] == label and "disabled" not in e[3]]
                if not matches:
                    raise AssertionError(f"Missing enabled {role} {label}: {controls()}")
                return matches[-1]

            def press(label, role="button"):
                auto("press", find(label, role)[0], "--settle")

            def fill(label, value):
                matches = [e for e in controls() if e[1] == "textField" and (e[2] == label or f'placeholder="{label}"' in e[3])]
                assert len(matches) == 1, f"Missing unique field {label}"
                auto("fill", matches[0][0], value, "--settle")

            def wait(text):
                auto("wait", "--text", text, "--timeout", "25s", "--settle")

            def row_menu(name, action):
                row = find(name, "listItem")
                auto("focus", row[0], "--settle")
                auto("key", "shift+F10", "--settle")
                press(action, "menuItem")

            def wait_current(name):
                deadline = time.monotonic() + 25
                while True:
                    entries = controls()
                    current = any(e[1] == "listItem" and e[2] == name and "Current" in e[3] for e in entries)
                    ready = any(e[1] == "button" and e[2] == "New worktree…" and "disabled" not in e[3] for e in entries)
                    if current and ready:
                        return
                    if time.monotonic() > deadline:
                        raise AssertionError(f"Worktree {name} did not become current and ready: {entries}")
                    time.sleep(0.05)

            def main_unchanged():
                assert git("rev-parse", "HEAD") == original_head
                assert git("ls-files", "--stage") == original_index
                assert (repo / "文件 space.txt").read_text() == "base\n"

            try:
                deadline = time.monotonic() + 20
                while not socket.exists():
                    if app.poll() is not None:
                        raise RuntimeError("Native app exited before automation connected")
                    if time.monotonic() > deadline:
                        raise TimeoutError("Native automation socket unavailable")
                    time.sleep(0.05)
                auto("open", "--socket", str(socket))
                auto("wait", "--text", "Commit message", "--timeout", "30s", "--settle")
                auto("key", "cmd+1", "--settle")
                wait("Worktree fixture base")
                auto("key", "cmd+4", "--settle")
                wait("New worktree…")
                press("New worktree…")
                fill("Worktree directory (absolute path)", str(checkout))
                fill("New branch name", "native/review")
                fill("Start point (branch or commit)", "main")
                press("Create")
                wait("Worktree created")
                wait("review space")
                assert (checkout / ".git").is_file()
                assert git("branch", "--show-current", directory=checkout).strip() == "native/review"
                assert git("rev-parse", "HEAD", directory=checkout) == original_head
                main_unchanged()
                print("PASS native worktree form creates new branch and space-path checkout without changing main", flush=True)

                row_menu("review space", "Switch to this worktree")
                wait_current("review space")
                row_menu("workspace", "Switch to this worktree")
                wait_current("workspace")
                main_unchanged()
                print("PASS native worktree switching and returning preserve both checkouts", flush=True)

                # A lock loaded into the UI must disable removal.
                git("worktree", "lock", "--reason", "native lock protection", str(checkout))
                auto("key", "cmd+r", "--settle")
                wait_current("workspace")
                row = find("review space", "listItem")
                auto("focus", row[0], "--settle")
                auto("key", "shift+F10", "--settle")
                remove = [e for e in controls() if e[1] == "menuItem" and e[2] == "Remove worktree…"]
                assert len(remove) == 1 and "disabled" in remove[0][3], remove
                auto("screenshot", str(artifacts / "locked-menu.png"))
                auto("key", "Escape", "--settle")
                assert (checkout / ".git").is_file()
                assert git("rev-parse", "HEAD", directory=checkout) == original_head
                main_unchanged()
                git("worktree", "unlock", str(checkout))
                auto("key", "cmd+r", "--settle")
                wait_current("workspace")
                print("PASS native locked-worktree menu disables removal and preserves checkout", flush=True)

                # A lock arriving after confirmation opens invalidates that target.
                row_menu("review space", "Remove worktree…")
                wait("Remove worktree")
                git("worktree", "lock", "--reason", "lock after confirmation", str(checkout))
                press("Remove")
                wait("Could not remove worktree")
                wait("The worktree changed or no longer exists. Refresh and confirm its path, branch, and commit again.")
                assert "locked lock after confirmation" in git("worktree", "list", "--porcelain")
                assert (checkout / ".git").is_file()
                assert git("rev-parse", "HEAD", directory=checkout) == original_head
                assert git("ls-files", "--stage", directory=checkout) == original_index
                main_unchanged()
                git("worktree", "unlock", str(checkout))
                auto("key", "cmd+r", "--settle")
                wait_current("workspace")
                print("PASS native pending removal refuses a newly locked target without changing either checkout", flush=True)

                dirty = checkout / "文件 space.txt"
                dirty.write_text("keep dirty bytes\n")
                row_menu("review space", "Remove worktree…")
                wait("Remove worktree")
                press("Cancel")
                assert dirty.read_text() == "keep dirty bytes\n" and (checkout / ".git").is_file()
                row_menu("review space", "Remove worktree…")
                wait("Remove worktree")
                press("Remove")
                wait("Could not remove worktree")
                wait("The working tree or Index has changes. Commit or stash them before retrying.")
                assert dirty.read_text() == "keep dirty bytes\n" and (checkout / ".git").is_file()
                assert git("branch", "--show-current", directory=checkout).strip() == "native/review"
                main_unchanged()
                print("PASS native remove cancellation and dirty-worktree refusal preserve files and branch", flush=True)

                # Restore only the fixture bytes we changed, then remove via UI.
                dirty.write_text("base\n")
                assert not git("status", "--porcelain", directory=checkout).strip()

                # Ignored content is invisible to ordinary status, but must survive.
                exclude = pathlib.Path(git("rev-parse", "--path-format=absolute", "--git-path", "info/exclude", directory=checkout).strip())
                with exclude.open("a") as patterns:
                    patterns.write("\n/keep-ignored.bin\n")
                ignored = checkout / "keep-ignored.bin"
                ignored_bytes = b"fixture ignored content\x00\xff\n"
                ignored.write_bytes(ignored_bytes)
                assert not git("status", "--porcelain", directory=checkout).strip()
                assert git("ls-files", "--others", "--ignored", "--exclude-standard", directory=checkout).strip() == ignored.name
                row_menu("review space", "Remove worktree…")
                wait("Remove worktree")
                press("Remove")
                wait("Could not remove worktree")
                wait('Ignored file "keep-ignored.bin" is present. To preserve its contents, this operation stopped. Move it out or back it up before retrying.')
                assert ignored.read_bytes() == ignored_bytes
                assert (checkout / ".git").is_file()
                assert git("rev-parse", "HEAD", directory=checkout) == original_head
                assert git("ls-files", "--stage", directory=checkout) == original_index
                assert dirty.read_text() == "base\n"
                main_unchanged()
                auto("screenshot", str(artifacts / "ignored-refusal.png"))
                print("PASS native ignored-file removal refusal preserves exact binary bytes and both checkout identities", flush=True)
                ignored.unlink()  # Only this scenario's newly created fixture file.
                row_menu("review space", "Remove worktree…")
                wait("Remove worktree")
                press("Remove")
                wait("Worktree removed")
                assert not checkout.exists(), "Successful removal left checkout directory"
                assert git("rev-parse", "refs/heads/native/review") == original_head, "Removal deleted retained branch"
                assert "review space" not in git("worktree", "list", "--porcelain")
                main_unchanged()
                (artifacts / "final.snapshot.txt").write_text(auto("snapshot", "--full"))
                auto("screenshot", str(artifacts / "final.png"))
                print("PASS native clean worktree removal retains branch and main HEAD/Index/bytes", flush=True)

                # Exercise the existing retained-branch toast with a real pointer.
                wait_current("workspace")
                delete = find("Delete")
                assert "offscreen" not in delete[3], f"Retained-branch action is offscreen: {delete}"
                bounds = re.search(r"bounds=([-0-9.]+),([-0-9.]+),([0-9.]+)x([0-9.]+)", delete[3])
                assert bounds, f"Retained-branch action has no bounds: {delete}"
                x, y, width, height = map(float, bounds.groups())
                auto("click", str(x + width / 2), str(y + height / 2), "--settle")
                wait("Branch deleted")
                assert not git("for-each-ref", "--format=%(refname)", "refs/heads/native/review").strip()
                assert git("symbolic-ref", "--short", "HEAD").strip() == "main"
                main_unchanged()
                (artifacts / "final.snapshot.txt").write_text(auto("snapshot", "--full"))
                auto("screenshot", str(artifacts / "final.png"))
                print("PASS native retained-branch toast Delete removes only original branch without changing main", flush=True)
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
