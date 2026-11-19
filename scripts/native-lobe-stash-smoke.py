#!/usr/bin/env python3
"""Exercise Lobe Stash in a fresh repository with isolated preferences."""
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
artifacts = build / "lobe-stash-smoke"
artifacts.mkdir(parents=True, exist_ok=True)
session = "supergit-lobe-stash"


def run():
    with tempfile.TemporaryDirectory(prefix="supergit-lobe-stash.") as directory:
        root = pathlib.Path(directory)
        repo, config = root / "workspace", root / "config"
        settings = config / "supergit"
        settings.mkdir(parents=True)
        (settings / "settings.json").write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False}))
        git_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
        git_env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")

        def git(*args):
            return subprocess.check_output(["git", "-C", str(repo), *args], env=git_env, text=True)

        subprocess.run(["git", "init", "-b", "main", str(repo)], env=git_env, check=True, capture_output=True)
        git("config", "user.name", "Fixture")
        git("config", "user.email", "fixture@example.invalid")
        tracked, extra = repo / "文件 space.txt", repo / "untracked space.txt"
        tracked.write_text("base\n")
        git("add", "--all")
        git("commit", "--no-gpg-sign", "-m", "Stash fixture base")
        original_head = git("rev-parse", "HEAD")
        tracked.write_text("staged version\n")
        git("add", "--", tracked.name)
        tracked.write_text("working version\n")
        extra.write_text("untracked version\n")
        original_index = git("ls-files", "--stage")
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

            def find(label, role="button", contains=False):
                matches = [e for e in controls() if e[1] == role and (label in e[2] if contains else e[2] == label) and "disabled" not in e[3]]
                if not matches:
                    raise AssertionError(f"Missing enabled {role} {label}: {controls()}")
                return matches[-1]

            def press(label, role="button", contains=False):
                # Success toasts can precede the repository refresh. Wait for
                # this action's guard to clear rather than treating text as ready.
                deadline = time.monotonic() + 25
                while True:
                    entries = controls()
                    matches = [e for e in entries if e[1] == role and (label in e[2] if contains else e[2] == label) and "disabled" not in e[3]]
                    if matches:
                        auto("press", matches[-1][0], "--settle")
                        return
                    if app.poll() is not None or time.monotonic() > deadline:
                        raise AssertionError(f"Missing enabled {role} {label}: {entries}")
                    time.sleep(0.05)

            def wait(text):
                auto("wait", "--text", text, "--timeout", "25s", "--settle")

            def save(message, include_untracked=False):
                press("Save changes…", "menuButton")
                press("Save changes including untracked files" if include_untracked else "Save changes", "menuItem")
                fields = [e for e in controls() if e[1] == "textField" and (e[2] == "Message (required)" or 'placeholder="Message (required)"' in e[3])]
                assert len(fields) == 1, "Stash message field missing"
                auto("fill", fields[0][0], message, "--settle")
                press("Save")
                wait("Stash saved")
                wait(message)
                press(message, "listItem", contains=True)

            def fingerprint():
                return git("rev-parse", "HEAD"), git("ls-files", "--stage"), tracked.read_bytes(), extra.read_bytes() if extra.exists() else None

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
                wait("Stash fixture base")
                auto("key", "cmd+3", "--settle")
                wait("Save changes…")
                save("Native stash complete", include_untracked=True)
                wait("+working version")
                stash_sha = git("rev-parse", "refs/stash").strip()
                assert not git("status", "--porcelain").strip(), "Stash save did not clean tracked/untracked contents"
                assert tracked.read_text() == "base\n" and not extra.exists()
                assert git("rev-parse", "HEAD") == original_head
                print("PASS native Stash saves staged/unstaged and untracked Unicode paths without moving HEAD", flush=True)

                press("Apply")
                wait("Stash applied")
                assert git("rev-parse", "refs/stash").strip() == stash_sha, "Apply dropped stash"
                assert git("ls-files", "--stage") == original_index, "Apply lost original Index version"
                assert tracked.read_text() == "working version\n" and extra.read_text() == "untracked version\n"
                assert git("rev-parse", "HEAD") == original_head
                before_drop = fingerprint()
                press("Drop…")
                wait("Drop stash")
                press("Cancel")
                assert git("rev-parse", "refs/stash").strip() == stash_sha and fingerprint() == before_drop, "Drop cancellation changed data"
                press("Drop…")
                wait("Drop stash")
                press("Drop")
                wait("Stash dropped")
                assert not git("stash", "list").strip() and fingerprint() == before_drop, "Drop changed working/Index content"
                print("PASS native Stash Apply restores Index and bytes; Drop cancellation and confirmed deletion target only stash", flush=True)

                save("Native stash pop", include_untracked=True)
                press("Pop (apply and drop)")
                wait("Stash popped")
                assert not git("stash", "list").strip()
                assert fingerprint() == before_drop, "Pop did not restore the exact staged/unstaged/untracked state"
                print("PASS native Stash Pop restores exact state and removes entry", flush=True)

                save("Native stash blocked", include_untracked=True)
                blocked_sha = git("rev-parse", "refs/stash").strip()
                tracked.write_text("overlapping local bytes\n")
                before_failed = fingerprint()
                press("Pop (apply and drop)")
                wait("Could not pop stash")
                compact = find("Could not pop stash", "status", contains=True)
                bounds = re.search(r"bounds=[^,]+,[^,]+,[^x]+x([0-9.]+)", compact[3])
                assert bounds and float(bounds[1]) < 180, "Error toast is still oversized"
                auto("screenshot", str(artifacts / "compact-error.png"))
                press("View details")
                wait("Copy details")
                feedback = "\n".join(e[2] + " " + e[3] for e in controls())
                assert "Check the working tree" in feedback and "including untracked files" in feedback, "Missing English partial-restoration guidance"
                assert "git stash:" in feedback and "would be overwritten by merge" in feedback, "Original Git output was lost"
                assert "Stash 与工作区" not in feedback, "English UI shows legacy Chinese hint"
                assert git("rev-parse", "refs/stash").strip() == blocked_sha, "Failed Pop dropped entry"
                after_failed = fingerprint()
                assert after_failed[:3] == before_failed[:3], "Overlapping Pop changed HEAD, Index or pre-existing tracked bytes"
                # Git can restore the stash's untracked files before refusing its
                # tracked merge. Failure is not an atomic rollback guarantee.
                assert before_failed[3] is None and after_failed[3] == b"untracked version\n", "Failed Pop did not preserve Git's partial untracked restoration"
                assert not git("ls-files", "-u").strip(), "Dirty refusal unexpectedly created conflicts"
                auto("screenshot", str(artifacts / "error-details.png"))
                auto("key", "Escape", "--settle")
                assert not any(e[1] == "button" and e[2] == "Copy details" for e in controls()), "Escape did not close error details"
                (artifacts / "final.snapshot.txt").write_text(auto("snapshot", "--full"))
                auto("screenshot", str(artifacts / "final.png"))
                print("PASS native failed overlapping Pop retains stash, HEAD/Index/tracked bytes; Git restores untracked content", flush=True)

                # Real three-way conflicts differ from the dirty-file refusal.
                # Reset only this script's disposable fixture between scenarios.
                for action, failure in (("Apply", "Could not apply stash"), ("Pop (apply and drop)", "Could not pop stash")):
                    git("reset", "--hard", original_head.strip())
                    if extra.exists():
                        extra.unlink()
                    tracked.write_text("stashed conflicting version\n")
                    save("Native three-way " + action)
                    conflict_stash = git("rev-parse", "refs/stash")
                    tracked.write_text("new committed version\n")
                    git("add", "--", tracked.name)
                    git("commit", "--no-gpg-sign", "-m", "Native stash conflicting target")
                    conflict_head = git("rev-parse", "HEAD")
                    press(action)
                    deadline = time.monotonic() + 25
                    while not git("ls-files", "-u").strip():
                        if app.poll() is not None or time.monotonic() > deadline:
                            raise AssertionError(f"{action} did not create a three-way conflict")
                        time.sleep(0.05)
                    wait(failure)
                    assert git("rev-parse", "refs/stash") == conflict_stash, "Conflict dropped selected stash"
                    assert git("rev-parse", "HEAD") == conflict_head, "Conflict moved HEAD"
                    stages = {line.split()[2] for line in git("ls-files", "-u").splitlines()}
                    assert stages == {"1", "2", "3"}, f"Missing three-way stages: {stages}"
                    conflict_bytes = tracked.read_text()
                    assert "<<<<<<<" in conflict_bytes and "stashed conflicting version" in conflict_bytes and "new committed version" in conflict_bytes
                    press("View details")
                    wait("Copy details")
                    wait("Check the working tree and resolve conflicts before staging files.")
                    feedback = "\n".join(e[2] + " " + e[3] for e in controls())
                    assert "do not apply the stash again without checking" in feedback and "git stash:" in feedback, "Conflict recovery guidance or Git failure missing"
                    assert "Index was not unstashed." in feedback or "CONFLICT" in feedback, "Original Git stderr diagnostic missing"
                    auto("screenshot", str(artifacts / ("three-way-" + action.split()[0].lower() + ".png")))
                    auto("key", "Escape", "--settle")
                    # External resolution is explicit; this does not validate an
                    # in-app conflict editor or a conflict-marker policy.
                    tracked.write_text("externally resolved version\n")
                    git("add", "--", tracked.name)
                    assert not git("ls-files", "-u").strip()
                    assert git("show", ":" + tracked.name) == "externally resolved version\n"
                    assert git("rev-parse", "HEAD") == conflict_head and git("rev-parse", "refs/stash") == conflict_stash
                    press("Refresh")
                    wait("Native three-way " + action)
                    print(f"PASS native {action} three-way conflict retains stash/HEAD and all stages; external resolution stages exact bytes", flush=True)
                (artifacts / "final.snapshot.txt").write_text(auto("snapshot", "--full"))
                auto("screenshot", str(artifacts / "final.png"))
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
