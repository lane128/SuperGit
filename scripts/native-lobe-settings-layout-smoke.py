#!/usr/bin/env python3
"""Capture settings layout and coordinate Reset cancellation in isolated homes."""
import argparse
import itertools
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
parser.add_argument("--persistence", action="store_true", help="Verify actual save failure, recovery and reset feedback in isolated en/zh settings windows")
parser.add_argument("--minimum-window", action="store_true", help="Build an isolated overlay that opens settings at the existing 560x480 minimum; does not resize a running window")
options = parser.parse_args()
artifacts = pathlib.Path(tempfile.mkdtemp(prefix="lobe-settings-layout-", dir=build))
binary = binaries[0]
panel_width, panel_height = (560, 480) if options.minimum_window else (820, 720)
if options.minimum_window:
    source = project / "internal/lobedesk/settings_window.go"
    original = source.read_text()
    replacement = original
    for old, new in (("settingsWindowWidth  = 820", "settingsWindowWidth  = settingsWindowMinW"),
                     ("settingsWindowHeight = 720", "settingsWindowHeight = settingsWindowMinH")):
        assert replacement.count(old) == 1, "settings window defaults changed; review minimum overlay"
        replacement = replacement.replace(old, new)
    overlay_source = artifacts / "settings-window-minimum.go.txt"
    overlay_source.write_text(replacement)
    overlay = artifacts / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {str(source): str(overlay_source)}}))
    binary = artifacts / "minimum-settings-e2e"
    with (artifacts / "build.log").open("w") as log:
        subprocess.run(["go", "build", "-tags", "e2e", "-overlay", str(overlay), "-o", str(binary), "./cmd/supergit-lobe"], cwd=project, stdout=log, stderr=subprocess.STDOUT, check=True)
results = []
cases = ((language, "light", 1) for language in ("en", "zh")) if options.persistence else itertools.product(("en", "zh"), ("light", "dark"), (1, 1.15))
for language, theme, scale in cases:
    name = f"{language}-{theme}-{scale}"
    captures = artifacts / name
    captures.mkdir()
    with tempfile.TemporaryDirectory(prefix="supergit-settings-layout.") as directory:
        root = pathlib.Path(directory)
        repo, home = root / "repo", root / "home"
        settings = home / "Library/Application Support/supergit/settings.json"
        settings.parent.mkdir(parents=True)
        fixture_settings = {"language": language, "theme": theme, "fontScale": scale, "autoFetch": False}
        if palette := os.environ.get("SUPERGIT_SETTINGS_PALETTE"):
            fixture_settings["palette"] = palette
        settings.write_text(json.dumps(fixture_settings))
        git_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
        git_env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")

        def git(*args):
            return subprocess.check_output(["git", "-C", str(repo), *args], env=git_env)

        subprocess.run(["git", "init", "-b", "main", str(repo)], env=git_env, check=True, capture_output=True)
        git("config", "user.name", "Settings Fixture")
        git("config", "user.email", "settings@example.invalid")
        path = repo / "文件 space.txt"
        path.write_bytes(b"base\n")
        git("add", "--", path.name)
        git("commit", "--no-gpg-sign", "-m", "Settings layout fixture")
        path.write_bytes(b"working\n")

        def fingerprint():
            return git("rev-parse", "HEAD"), git("show-ref"), git("ls-files", "--stage", "-z"), git("status", "--porcelain=v1", "-z"), path.read_bytes()

        before = fingerprint()
        settings_before = settings.read_bytes()
        socket = root / "app.sock"
        session = "settings-layout-" + name.replace(".", "-") + "-" + str(os.getpid())
        env = {**git_env, "HOME": str(home), "MYGO_AUTOMATION": str(socket), "SUPERGIT_WINDOW_SIZE": "980x640"}
        with (captures / "app.log").open("w") as log, (captures / "automation.log").open("w") as trace:
            app = subprocess.Popen([str(binary), str(repo)], cwd=project, env=env, stdout=log, stderr=subprocess.STDOUT)

            def auto(*args):
                result = subprocess.run([str(automation), *args, "--session", session], cwd=project, capture_output=True, text=True, timeout=40)
                trace.write(repr(args) + "\n" + result.stdout + result.stderr + "\n")
                trace.flush()
                if result.returncode:
                    raise RuntimeError((args, result.stdout, result.stderr))
                return result.stdout

            def snapshot():
                return auto("snapshot", "--full")

            def line_for(label, role="button"):
                matches = [line for line in snapshot().splitlines() if '[' + role + '] ' + json.dumps(label, ensure_ascii=False) + ' ' in line and "disabled" not in line]
                assert len(matches) == 1, (label, matches)
                return matches[0]

            def ref(line):
                return re.search(r"@e\d+", line)[0]

            def coordinate_press(label, role="button"):
                line = line_for(label, role)
                assert "offscreen" not in line, (label, line)
                match = re.search(r"bounds=([-\d.]+),([-\d.]+),([-\d.]+)x([-\d.]+)", line)
                x, y, width, height = map(float, match.groups())
                assert 0 <= x and x + width <= panel_width + 1 and 0 <= y and y + height <= panel_height + 1, line
                auto("click", str(x + width / 2), str(y + height / 2), "--settle")

            try:
                deadline = time.monotonic() + 20
                while not socket.exists():
                    if app.poll() is not None or time.monotonic() > deadline:
                        raise RuntimeError("native automation unavailable")
                    time.sleep(0.05)
                auto("open", "--socket", str(socket))
                auto("window", "w1")
                auto("wait", "--text", "Commit summary (required)" if language == "en" else "提交标题（必填）", "--timeout", "30s", "--settle")
                auto("key", "cmd+1", "--settle")
                auto("wait", "--text", "Settings layout fixture", "--timeout", "25s", "--settle")
                settings_label = "Settings" if language == "en" else "设置"
                auto("press", ref(line_for(settings_label)), "--settle")
                windows = auto("windows")
                panel_lines = [line for line in windows.splitlines() if "SuperGit · " + settings_label in line]
                assert len(panel_lines) == 1, windows
                panel = re.search(r"w\d+", panel_lines[0])[0]
                auto("window", panel)
                auto("wait", "--text", "English", "--timeout", "25s", "--settle")
                initial = snapshot()
                scroll_bounds = re.search(r"\[scroll\].*bounds=([-\d.]+),([\d.]+),([\d.]+)x([\d.]+)", initial)
                assert scroll_bounds, "settings scroll viewport missing"
                scroll_left, scroll_top, scroll_width, scroll_height = map(float, scroll_bounds.groups())
                assert abs(scroll_left - 132) < 1 and abs(scroll_left + scroll_width - panel_width) < 1, (panel_width, scroll_bounds[0])
                assert 0 < scroll_top + scroll_height <= panel_height + 1, (panel_height, scroll_bounds[0])

                def check_controls(text):
                    count = 0
                    for line in text.splitlines():
                        if not re.search(r"\[(button|radio|checkBox|toggleButton|tab)\]", line):
                            continue
                        bounds = re.search(r"bounds=([-\d.]+),([-\d.]+),([-\d.]+)x([-\d.]+)", line)
                        assert bounds, line
                        x, y, width, height = map(float, bounds.groups())
                        assert x >= 0 and x + width <= panel_width + 1, line
                        count += 1
                    return count

                interactive = check_controls(initial)
                assert interactive >= 14, "Appearance and category controls were not loaded"
                (captures / "appearance.snapshot.txt").write_text(initial)
                auto("screenshot", str(captures / "appearance.png"))
                categories = (("history", "History" if language == "en" else "历史"), ("git", "Git"), ("shortcuts", "Shortcuts" if language == "en" else "快捷键"), ("about", "About" if language == "en" else "关于"))
                for key, label in categories:
                    coordinate_press(label, "tab")
                    current = snapshot()
                    assert "checked" in line_for(label, "tab"), "selected category is not exposed"
                    interactive += check_controls(current)
                    (captures / (key + ".snapshot.txt")).write_text(current)
                    auto("screenshot", str(captures / (key + ".png")))
                assert interactive >= 40, "category settings controls were not loaded"
                reset = "Reset all settings…" if language == "en" else "恢复默认设置…"
                auto("act", ref(line_for(reset)), "scrollIntoView", "--settle")
                (captures / "bottom.snapshot.txt").write_text(snapshot())
                auto("screenshot", str(captures / "bottom.png"))
                coordinate_press(reset)
                assert "[dialog]" in snapshot(), "coordinate Reset did not open confirmation"
                auto("screenshot", str(captures / "confirmation.png"))
                coordinate_press("Cancel" if language == "en" else "取消")
                assert "[dialog]" not in snapshot(), "settings confirmation did not close"
                assert fingerprint() == before, "settings browsing or reset cancellation changed Git state"
                assert settings.read_bytes() == settings_before, "reset cancellation changed saved preferences"
                if options.persistence:
                    coordinate_press("Appearance" if language == "en" else "外观", "tab")
                    dark = "Dark" if language == "en" else "深色"
                    light = "Light" if language == "en" else "浅色"
                    failed = "Could not save settings" if language == "en" else "保存设置失败"
                    done = "Settings reset to defaults" if language == "en" else "设置已恢复默认"
                    backup = settings.with_name("fixture-settings-backup.json")
                    settings.rename(backup)
                    settings.mkdir()
                    try:
                        auto("act", ref(line_for(dark, "toggleButton")), "scrollIntoView", "--settle")
                        auto("press", ref(line_for(dark, "toggleButton")), "--settle")
                        panel_focused = any(panel in line and "focused" in line for line in auto("windows").splitlines())
                        if panel_focused:
                            auto("wait", "--text", failed, "--timeout", "25s", "--settle")
                        assert "checked" in line_for(dark, "toggleButton"), "failed persistence did not retain current-session theme"
                        assert backup.read_bytes() == settings_before and settings.is_dir(), "failed save replaced prior preferences"
                        (captures / "save-failed.snapshot.txt").write_text(snapshot())
                        auto("screenshot", str(captures / "save-failed.png"))
                        auto("window", "w1")
                        assert failed not in snapshot(), "settings save warning leaked into repository window"
                    finally:
                        settings.rmdir()
                        backup.rename(settings)
                    auto("window", panel)
                    auto("press", ref(line_for(light, "toggleButton")), "--settle")
                    saved = json.loads(settings.read_text())
                    assert saved["theme"] == "light" and saved["language"] == language
                    assert saved["autoFetch"] is False, "save recovery changed unrelated automatic fetch choice"
                    coordinate_press("About" if language == "en" else "关于", "tab")
                    auto("act", ref(line_for(reset)), "scrollIntoView", "--settle")
                    coordinate_press(reset)
                    coordinate_press("Reset all settings" if language == "en" else "恢复默认设置")
                    reset_panel_focused = any(panel in line and "focused" in line for line in auto("windows").splitlines())
                    if reset_panel_focused:
                        auto("wait", "--text", done, "--timeout", "25s", "--settle")
                    restored = json.loads(settings.read_text())
                    assert restored["theme"] == "system" and restored["language"] == language
                    assert restored["autoFetch"] is False and restored["showGraph"] is True
                    assert all(restored.get(key, True) is True for key in ("detailsInfo", "detailsChanges", "detailsDiff"))
                    assert restored.get("fontScale", 1) == 1
                    assert fingerprint() == before, "settings save/reset changed repository state"
                    (captures / "reset-done.snapshot.txt").write_text(snapshot())
                    auto("screenshot", str(captures / "reset-done.png"))
                    auto("window", "w1")
                    assert done not in snapshot(), "settings reset success leaked into repository window"
                    auto("window", panel)
                    results.append({"case": name, "save_warning_not_in_main": True, "save_warning_visual_verified": panel_focused, "old_preferences_preserved_on_failure": True, "session_theme_retained_on_failure": True, "save_recovered": True, "reset_persisted_defaults_language_kept": True, "reset_success_not_in_main": True, "reset_success_visual_verified": reset_panel_focused, "git_preserved": True})
                    print("PASS native settings persistence " + language + ": failed save preserves preferences, recovery, actual reset/defaults, no main-window notices, Git preserved", flush=True)
                    continue
                results.append({"case": name, "native_settings_size": f"{panel_width}x{panel_height}", "minimum_initial_size_overlay": options.minimum_window, "control_horizontal_bounds": True, "reset_coordinate_open_cancel": True, "git_and_settings_preserved": True})
                print("PASS native settings " + name + ": bounds, coordinate Reset/cancel, Git and saved preferences preserved", flush=True)
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
