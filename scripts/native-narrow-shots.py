#!/usr/bin/env python3
"""Capture Lobe detail tabs at 980/1320/1600 x 640 in zh/en and light/dark.

Four additional critical combinations use fontScale=1.15. A disposable fixture
has a long commit body, identical basenames, a long path, 13 file tabs and an
empty commit. Each look captures summary, overflowing tabs, empty history and
working changes. Logs and accessibility snapshots are kept beside screenshots.
"""
import json
import os
import pathlib
import re
import shutil
import subprocess
import sys
import tempfile
import time

project = pathlib.Path(__file__).resolve().parents[1]
build = project / "build"
automation = build / "mygo-automation"
# The History page this checks belongs to the Lobe preview app, not the older
# release build that `make e2e-build` produces.
app_binary = build / "supergit-lobe-e2e"
# Keep earlier visual evidence intact; each matrix run owns its captures.
shots = pathlib.Path(tempfile.mkdtemp(prefix="narrow-shots-", dir=build))

if not app_binary.is_file() or not automation.is_file():
    raise SystemExit("run `make lobe-e2e-build` first")

root = pathlib.Path(tempfile.mkdtemp(prefix="supergit-narrow."))
workspace = root / "workspace"
remote = root / "remote.git"
git_env = {
    key: value
    for key, value in os.environ.items()
    if not key.startswith("GIT_") or key in ("GIT_EXEC_PATH",)
}
git_env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")


def git(*args, env=None):
    merged = {**git_env, **(env or {})}
    return subprocess.run(
        ["git", "-C", str(workspace), *args],
        env=merged,
        check=True,
        capture_output=True,
        text=True,
    ).stdout


def edit(path, text, subject, when):
    """Commit a real file change so the Files column and the patch have rows."""
    target = workspace / path
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(text)
    git("add", "--", path)
    stamp = int(when)
    git(
        "commit",
        "-m",
        subject,
        env={"GIT_AUTHOR_DATE": f"{stamp} +0000", "GIT_COMMITTER_DATE": f"{stamp} +0000"},
    )


subprocess.run(["git", "init", "--bare", "-b", "main", str(remote)], env=git_env, check=True, capture_output=True)
subprocess.run(["git", "init", "-b", "main", str(workspace)], env=git_env, check=True, capture_output=True)
git("config", "user.name", "Narrow Reviewer")
git("config", "user.email", "narrow@example.invalid")
edit("docs/release-notes.md", "release notes\n", "Initial commit", 1700000000)
edit("docs/release-notes.md", "release notes\nautumn campaign\n", "Add release notes for the autumn campaign", 1700000100)
edit("src/history/ref-capsules.go", "package history\n", "修复窄窗口下引用胶囊和提交标题相撞的问题", 1700000200)
git("switch", "-c", "feature/narrow-window-ref-capsules")
edit("src/history/ref-capsules.go", "package history\n\nfunc capsule() {}\n", "Show every reference capsule in the narrow history row", 1700000300)
edit("src/history/ref-capsules.go", "package history\n\nfunc capsule() { truncate() }\n", "Truncate long reference names in the middle", 1700000400)
git("switch", "main")
edit("go.mod", "module demo\n\ngo 1.27\n", "Bump dependencies", 1700000500)
git("tag", "v1.2.3", "HEAD~2")
git("switch", "-c", "hotfix")
edit("src/diff/empty.go", "package diff\n", "Fix the crash on an empty diff", 1700000600)
git("switch", "main")
git("merge", "--no-ff", "hotfix", "-m", "Merge hotfix into main", env={"GIT_AUTHOR_DATE": "1700000700 +0000", "GIT_COMMITTER_DATE": "1700000700 +0000"})
paths = [f"same-name/{n:02d}/file.go" for n in range(12)] + ["very-long-directory-name/another-long-directory/long-file-name-for-detail-tab-review.txt"]
for path in paths:
    target = workspace / path
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text("base content\n")
git("add", "--all")
git("commit", "-m", "Detail tabs base fixture")
for path in paths:
    (workspace / path).write_text("changed content\n")
body = "Long message starts here\n\n" + "\n".join(f"detail {n:02d}: 完整说明 and preserved indentation" for n in range(24)) + "\n\nRefs: narrow-tabs-fixture"
git("commit", "-am", "Detail tabs changes fixture", "-m", body)
git("commit", "--allow-empty", "-m", "Detail tabs empty fixture")
git("remote", "add", "origin", str(remote))
git("push", "-u", "origin", "main", "feature/narrow-window-ref-capsules")
(workspace / "docs/release-notes.md").write_text("release notes\nautumn campaign\nstaged version\n")
git("add", "--", "docs/release-notes.md")
(workspace / "docs/release-notes.md").write_text("release notes\nautumn campaign\nedited in the working tree\n")


def fingerprint():
    return (git("rev-parse", "HEAD"), git("ls-files", "--stage"), git("diff"), git("diff", "--cached"))


expected_state = fingerprint()

ready_text = "Detail tabs changes fixture"
combinations = [
    (f"{width}-{language}-{theme}", language, theme, None, width)
    for width in (980, 1320, 1600)
    for language in ("zh", "en")
    for theme in ("light", "dark")
] + [
    (f"{width}-{language}-{theme}-large", language, theme, 1.15, width)
    for width, language, theme in ((980, "zh", "light"), (980, "en", "dark"), (1600, "zh", "dark"), (1600, "en", "light"))
]

selected_cases = os.environ.get("SUPERGIT_SHOT_CASES", "")
if selected_cases:
    selected_cases = set(selected_cases.split(","))
    combinations = [case for case in combinations if case[0] in selected_cases]
    if not combinations:
        raise SystemExit("SUPERGIT_SHOT_CASES matched no matrix cases")

shots.mkdir(parents=True, exist_ok=True)
failures = []
for name, language, theme, font_scale, width in combinations:
    home = root / ("home-" + name)
    settings_dir = home / "Library" / "Application Support" / "supergit"
    settings_dir.mkdir(parents=True)
    settings = {"theme": theme, "language": language}
    if font_scale:
        settings["fontScale"] = font_scale
    (settings_dir / "settings.json").write_text(json.dumps(settings) + "\n")
    socket = root / (name + ".sock")
    session = f"supergit-narrow-{name}"
    env = {
        **os.environ,
        "MYGO_AUTOMATION": str(socket),
        "SUPERGIT_WINDOW_SIZE": f"{width}x640",
        "HOME": str(home),
        "GIT_CONFIG_GLOBAL": os.devnull,
        "GIT_CONFIG_NOSYSTEM": "1",
    }
    log = (shots / (name + ".log")).open("w")
    app = subprocess.Popen(
        [str(app_binary), str(workspace)],
        cwd=project,
        env=env,
        stdout=log,
        stderr=subprocess.STDOUT,
    )
    try:
        deadline = time.monotonic() + 20
        while not socket.exists():
            if app.poll() is not None:
                raise RuntimeError("app exited before opening its automation socket")
            if time.monotonic() >= deadline:
                raise TimeoutError("app did not open its automation socket")
            time.sleep(0.05)

        def auto(*items, timeout=40):
            result = subprocess.run(
                [str(automation), *items, "--session", session],
                capture_output=True,
                text=True,
                timeout=timeout,
                cwd=project,
            )
            if result.returncode:
                raise RuntimeError(f"{items}: {result.stdout}{result.stderr}")
            return result.stdout

        auto("open", "--socket", str(socket))
        def controls():
            entries = []
            for line in auto("snapshot", "--full").splitlines():
                match = re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)', line)
                if match:
                    entries.append((match[1], match[2], json.loads(match[3]) if match[3] else "", match[4]))
            return entries

        def press(role, label):
            matches = [e for e in controls() if e[1] == role and e[2] == label]
            if len(matches) != 1:
                raise AssertionError(f"expected one {role} {label!r}, got {matches}")
            auto("press", matches[0][0], "--settle")

        def capture(suffix):
            print(auto("screenshot", str(shots / (name + suffix + ".png"))).strip(), flush=True)
            (shots / (name + suffix + ".snapshot.txt")).write_text(auto("snapshot", "--full"))

        auto("key", "cmd+1", "--settle")
        auto("wait", "--text", ready_text, "--timeout", "30s", "--settle")
        press("listItem", ready_text)
        auto("wait", "--text", "Refs: narrow-tabs-fixture", "--timeout", "20s", "--settle")
        entries = controls()
        files_title = "文件" if language == "zh" else "Files"
        title_at = next(i for i, e in enumerate(entries) if e[1] == "text" and e[2] == files_title)
        file_scroll = next(e for e in entries[title_at + 1:] if e[1] == "scroll")
        tab_at = next(i for i, e in enumerate(entries) if e[1] == "tabList")
        tab_scroll = next(e for e in reversed(entries[:tab_at]) if e[1] == "scroll")
        def bounds(entry):
            match = re.search(r"bounds=([-0-9.]+),([-0-9.]+),([-0-9.]+)x([-0-9.]+)", entry[3])
            if not match:
                raise AssertionError(f"missing bounds: {entry}")
            return tuple(float(value) for value in match.groups())
        # The file scroll viewport excludes its column's one-DIP divider.
        assert bounds(file_scroll)[2] + 1 >= 180 - 0.1, f"file column too narrow: {file_scroll}"
        assert bounds(tab_scroll)[2] >= 300 - 0.1, f"details too narrow: {tab_scroll}"
        capture("-summary")
        for path in paths:
            press("listItem", path)
        tabs = [e for e in controls() if e[1] == "tab"]
        assert len(tabs) == len(paths) + 1, "lost file tabs"
        assert "checked" in tabs[-1][3] and "offscreen" not in tabs[-1][3], "active tab was not revealed"
        close_label = ("关闭 " if language == "zh" else "Close ") + paths[-1]
        close_button = next(e for e in controls() if e[1] == "button" and e[2] == close_label)
        assert "offscreen" not in close_button[3], "active tab close control was not revealed"
        capture("-tabs")
        press("listItem", "Detail tabs empty fixture")
        assert len([e for e in controls() if e[1] == "tab"]) == 1
        capture("-empty")
        changes = next(e for e in controls() if e[1] == "listItem" and (e[2] == "1 change" or e[2] == "1 changes" or "1 个改动" in e[2] or "1 项改动" in e[2]))
        auto("press", changes[0], "--settle")
        auto("wait", "--text", "+edited in the working tree", "--timeout", "20s", "--settle")
        capture("-worktree")
        entries = controls()
        groups = ("已修改文件", "未跟踪文件", "已暂存文件") if language == "zh" else ("Modified Files", "Untracked Files", "Staged Files")
        for group in groups:
            assert any(e[1] == "button" and e[2] == group for e in entries), f"missing fixed group {group}"
        commit_label = "提交" if language == "zh" else "Commit"
        commit_button = next(e for e in entries if e[1] == "button" and e[2] == commit_label)
        assert "disabled" in commit_button[3], "empty summary permits commit"
        capture("-commit-disabled")
        summary_label = "提交标题（必填）" if language == "zh" else "Commit summary (required)"
        summary = next(e for e in entries if e[1] == "textField" and e[2] == summary_label)
        auto("fill", summary[0], "Acceptance draft with staged changes", "--settle")
        deadline = time.monotonic() + 20
        while not any(e[1] == "button" and e[2] == commit_label and "disabled" not in e[3] for e in controls()):
            if time.monotonic() > deadline:
                raise AssertionError("staged fixture with summary never enables commit")
            time.sleep(0.05)
        assert "narrow@example.invalid" in auto("snapshot", "--full"), "effective author email missing"
        capture("-commit-enabled")
        assert fingerprint() == expected_state, "screenshot browsing changed fixture HEAD, Index or worktree"
        print(name, "captured", flush=True)
    except Exception as error:  # keep the other matrix cases running
        try:
            capture("-failure")
        except Exception:
            pass
        failures.append((name, str(error)))
        print(name, "failed:", error, file=sys.stderr, flush=True)
    finally:
        subprocess.run(
            [str(automation), "close", "--session", session],
            cwd=project,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        log.close()
        app.terminate()
        try:
            app.wait(timeout=10)
        except subprocess.TimeoutExpired:
            app.kill()
            app.wait()

if failures:
    raise SystemExit("narrow shots failed: " + ", ".join(name for name, _ in failures))
print("Screenshots in", shots)
shutil.rmtree(root, ignore_errors=True)
