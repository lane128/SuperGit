#!/usr/bin/env python3
"""Verify native two-window confirmation ownership in an isolated fixture.

Automation targets windows but does not activate OS keyboard focus. This probe
therefore verifies rendered ownership and conflicting requests, not OS focus.
"""
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
candidates = list(build.glob("supergit-lobe*e2e"))
if not automation.is_file() or len(candidates) != 1:
    raise SystemExit("run make lobe-e2e-build first")
artifacts = pathlib.Path(tempfile.mkdtemp(prefix="lobe-window-dialog-", dir=build))

with tempfile.TemporaryDirectory(prefix="supergit-window-dialog.") as directory:
    root = pathlib.Path(directory)
    repo = root / "repo"
    home = root / "home"
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
    path = repo / "fixture.txt"
    path.write_text("base\n")
    git("add", "--", "fixture.txt")
    git("commit", "-m", "Window ownership fixture")
    path.write_text("dirty working bytes\n")

    def fingerprint():
        return (git("rev-parse", "HEAD"), git("show-ref"), git("status", "--porcelain=v1", "-z"),
                hashlib.sha256((repo / ".git/index").read_bytes()).hexdigest(), path.read_bytes())

    before = fingerprint()
    socket = root / "app.sock"
    session = "window-dialog-" + str(os.getpid())
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

        def snapshot(window):
            auto("window", window)
            return auto("snapshot", "--full")

        def press(window, label, role="button"):
            text = snapshot(window)
            matches = [line for line in text.splitlines()
                       if re.search(r'\[' + re.escape(role) + r'\] ' + re.escape(json.dumps(label, ensure_ascii=False)) + r'(?: |$)', line)
                       and "disabled" not in line]
            assert len(matches) == 1, (label, matches)
            ref = re.search(r"@e\d+", matches[0])[0]
            if "offscreen" in matches[0]:
                auto("act", ref, "scrollIntoView", "--settle")
            auto("press", ref, "--settle")

        def wait(window, text):
            auto("window", window)
            auto("wait", "--text", text, "--timeout", "20s", "--settle")

        def cancel_present(text):
            return '[button] "Cancel"' in text

        try:
            deadline = time.monotonic() + 20
            while not socket.exists():
                if app.poll() is not None or time.monotonic() > deadline:
                    raise RuntimeError("native automation socket unavailable")
                time.sleep(0.05)
            auto("open", "--socket", str(socket))
            auto("wait", "--text", "Commit summary (required)", "--timeout", "30s", "--settle")
            auto("key", "cmd+1", "--settle")
            wait("w1", "1 change")
            press("w1", "1 change", "listItem")
            wait("w1", "dirty working bytes")
            press("w1", "Settings")
            windows = auto("windows")
            panel = next(re.match(r"w\d+", line)[0] for line in windows.splitlines()
                         if "SuperGit · Settings" in line)
            (artifacts / "windows.txt").write_text(windows)

            press(panel, "About", "tab")

            # First confirmation wins across windows; later requests must not
            # replace its action, message, or owner, even with explicit targeting.
            press(panel, "Reset all settings…")
            wait(panel, "repositories and the working tree are not touched")
            assert cancel_present(snapshot(panel)) and not cancel_present(snapshot("w1"))
            press("w1", "Discard all")
            assert cancel_present(snapshot(panel)) and not cancel_present(snapshot("w1"))
            assert "Keep the Index and every untracked file" not in snapshot(panel)
            auto("window", panel)
            auto("screenshot", str(artifacts / "settings-confirmation.png"))
            press(panel, "Cancel")
            assert not cancel_present(snapshot(panel))
            assert fingerprint() == before, "settings cancellation/conflicting discard changed Git state"

            press("w1", "Discard all")
            wait("w1", "Keep the Index and every untracked file")
            assert cancel_present(snapshot("w1")) and not cancel_present(snapshot(panel))
            press(panel, "Reset all settings…")
            assert cancel_present(snapshot("w1")) and not cancel_present(snapshot(panel))
            assert "Keep the Index and every untracked file" in snapshot("w1")
            auto("window", "w1")
            auto("screenshot", str(artifacts / "main-confirmation.png"))
            press("w1", "Cancel")
            assert not cancel_present(snapshot("w1"))
            assert fingerprint() == before, "discard cancellation/conflicting reset changed Git state"
            auto("window", "w1")
            auto("key", "cmd+2", "--settle")
            press("w1", "New branch…")
            wait("w1", "New branch")
            fields = [line for line in snapshot("w1").splitlines() if "[textField]" in line]
            assert len(fields) == 1, fields
            auto("fill", re.search(r"@e\d+", fields[0])[0], "retained-window-draft", "--settle")
            press(panel, "Reset all settings…")
            wait(panel, "repositories and the working tree are not touched")
            main_text = snapshot("w1")
            assert 'value="retained-window-draft"' in main_text and cancel_present(main_text)
            assert cancel_present(snapshot(panel)), "settings confirmation blocked by main input"
            (artifacts / "concurrent-main.snapshot.txt").write_text(main_text)
            auto("window", "w1")
            auto("screenshot", str(artifacts / "concurrent-main.png"))
            auto("window", panel)
            auto("screenshot", str(artifacts / "concurrent-settings.png"))
            press(panel, "Cancel")
            assert 'value="retained-window-draft"' in snapshot("w1"), "settings cancellation lost main draft"
            assert not cancel_present(snapshot(panel))
            press("w1", "Cancel")
            assert not cancel_present(snapshot("w1"))
            assert fingerprint() == before, "concurrent modal cancellation changed Git state"
            result = {"state_preserved": True, "settings_then_main": True,
                      "main_then_settings": True, "concurrent_prompt_draft_preserved": True,
                      "os_focus_switch_verified": False}
            (artifacts / "results.json").write_text(json.dumps(result, indent=2) + "\n")
            print("PASS native two-window confirmation ownership, both conflict orders and cancellation; OS focus excluded")
            print(artifacts)
        except BaseException:
            for window in ("w1", locals().get("panel")):
                if window:
                    try:
                        (artifacts / f"failure-{window}.snapshot.txt").write_text(snapshot(window))
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
