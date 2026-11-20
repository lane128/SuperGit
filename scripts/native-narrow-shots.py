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
if os.environ.get("SUPERGIT_SHOT_EXTERNAL_OPEN") == "1":
    paths[-1] = "very-long-directory-name/another-long-directory/文件 name;$(touch nope).txt"
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
    if palette := os.environ.get("SUPERGIT_SHOT_PALETTE"):
        settings["palette"] = palette
    if font_scale:
        settings["fontScale"] = font_scale
    for preference, variable, allowed in (
        ("historyLine", "SUPERGIT_SHOT_GRAPH_LINE", {"thin", "standard", "thick"}),
        ("historyNode", "SUPERGIT_SHOT_GRAPH_NODE", {"small", "standard", "large"}),
    ):
        if value := os.environ.get(variable):
            if value not in allowed:
                raise SystemExit(f"Invalid {variable}: {value}")
            settings[preference] = value
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
    open_sentinel = root / (name + ".open-args")
    if os.environ.get("SUPERGIT_SHOT_EXTERNAL_OPEN") == "1":
        launcher_dir = root / (name + "-launcher")
        launcher_dir.mkdir()
        launcher = launcher_dir / "open"
        launcher.write_text("#!/bin/sh\nprintf '%s\\0' \"$@\" > \"$SUPERGIT_OPEN_SENTINEL\"\n")
        launcher.chmod(0o700)
        env["PATH"] = str(launcher_dir) + os.pathsep + env.get("PATH", "")
        env["SUPERGIT_OPEN_SENTINEL"] = str(open_sentinel)
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

        startup_summary = "提交信息" if language == "zh" else "Commit message"
        auto("wait", "--text", startup_summary, "--timeout", "20s", "--settle")
        startup_tab = next(e for e in controls() if e[1] == "tab" and e[2] == ("改动" if language == "zh" else "Changes"))
        assert "checked" in startup_tab[3], "launch did not select Changes"
        assert any(e[1] == "textField" and e[2] == startup_summary for e in controls()), "launch did not show working-tree commit editor"
        partial_state = "部分暂存" if language == "zh" else "Partially staged"
        for action_name in (("暂存文件", "取消暂存") if language == "zh" else ("Stage file", "Unstage file")):
            assert any(e[1] == "button" and e[2] == action_name and partial_state in e[3] for e in controls()), "partial file state lost its version-specific action: " + action_name
        capture("-startup-changes")
        if os.environ.get("SUPERGIT_SHOT_BRANCH_ENTRY") == "1":
            press("button", "管理分支" if language == "zh" else "Manage branches")
            auto("wait", "--text", "创建分支…" if language == "zh" else "New branch…", "--timeout", "20s", "--settle")
            assert any(e[1] == "button" and e[2] == ("创建分支…" if language == "zh" else "New branch…") for e in controls()), "mouse entry did not open branch management"
            capture("-branch-management")
        auto("key", "cmd+1", "--settle")
        auto("wait", "--text", ready_text, "--timeout", "30s", "--settle")
        press("listItem", ready_text)
        auto("wait", "--text", "Refs: narrow-tabs-fixture", "--timeout", "20s", "--settle")
        entries = controls()
        files_title = "文件" if language == "zh" else "Files"
        title_at = next(i for i, e in enumerate(entries) if e[1] == "text" and e[2] == files_title)
        file_scroll = next(e for e in entries[title_at + 1:] if e[1] == "scroll")
        details_label = "文件详情" if language == "zh" else "File details"
        tab_at = next(i for i, e in enumerate(entries) if e[1] == "tabList" and e[2] == details_label)
        def detail_tabs():
            current = controls()
            start = next(i for i, e in enumerate(current) if e[1] == "tabList" and e[2] == details_label)
            return [e for e in current[start + 1:] if e[1] == "tab"]
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
        if os.environ.get("SUPERGIT_SHOT_GRAPH_IDENTITIES") == "1":
            for subject, suffix in (
                ("Detail tabs empty fixture", "-graph-selected-head"),
                ("Merge hotfix into main", "-graph-selected-merge"),
            ):
                row = next(e for e in controls() if e[1] == "listItem" and e[2] == subject)
                assert "offscreen" not in row[3], "graph identity row is outside the visible fixture: " + subject
                press("listItem", subject)
                selected = next(e for e in controls() if e[1] == "listItem" and e[2] == subject)
                assert "checked" in selected[3], "graph identity selection was not applied: " + subject
                capture(suffix)
            press("listItem", ready_text)
            auto("wait", "--text", "Refs: narrow-tabs-fixture", "--timeout", "20s", "--settle")
        for path in paths:
            press("listItem", path)
        tabs = detail_tabs()
        assert len(tabs) == len(paths) + 1, "lost file tabs"
        assert "checked" in tabs[-1][3] and "offscreen" not in tabs[-1][3], "active tab was not revealed"
        close_label = ("关闭 " if language == "zh" else "Close ") + paths[-1]
        close_button = next(e for e in controls() if e[1] == "button" and e[2] == close_label)
        assert "offscreen" not in close_button[3], "active tab close control was not revealed"
        capture("-tabs")
        if os.environ.get("SUPERGIT_SHOT_EXTERNAL_OPEN") == "1":
            selected_file = next(e for e in controls() if e[1] == "listItem" and e[2] == paths[-1])
            auto("focus", selected_file[0], "--settle")
            auto("key", "shift+f10", "--settle")
            press("menuItem", "在 Finder 中显示" if language == "zh" else "Reveal in Finder")
            opened_path = pathlib.Path(git("rev-parse", "--show-toplevel").removesuffix("\n")) / paths[-1]
            expected_open = b"-R\0" + str(opened_path).encode() + b"\0"
            deadline = time.monotonic() + 10
            while not open_sentinel.exists() or open_sentinel.read_bytes() != expected_open:
                if app.poll() is not None or time.monotonic() >= deadline:
                    actual = open_sentinel.read_bytes() if open_sentinel.exists() else b""
                    raise AssertionError(f"external reveal arguments {actual!r}; expected {expected_open!r}")
                time.sleep(0.05)
            (shots / (name + "-external-open.json")).write_text(json.dumps({"path": str(opened_path), "relative_path": paths[-1], "action": "reveal", "argument_count": 2, "literal_path_verified": True}, ensure_ascii=False) + "\n")
        press("listItem", "Detail tabs empty fixture")
        assert len(detail_tabs()) == 1
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
        summary_label = "提交信息" if language == "zh" else "Commit message"
        summary = next(e for e in entries if e[1] == "textField" and e[2] == summary_label)
        auto("fill", summary[0], "Acceptance draft with staged changes", "--settle")
        deadline = time.monotonic() + 20
        while not any(e[1] == "button" and e[2] == commit_label and "disabled" not in e[3] for e in controls()):
            if time.monotonic() > deadline:
                raise AssertionError("staged fixture with summary never enables commit")
            time.sleep(0.05)
        assert "narrow@example.invalid" in auto("snapshot", "--full"), "effective author email missing"
        capture("-commit-enabled")
        # Changes keeps the editor and action beside either file version.
        press("tab", "改动" if language == "zh" else "Changes")
        capture("-changes-overview")
        for label, expected in (("release-notes.md · " + ("未暂存" if language == "zh" else "Unstaged"), "+edited in the working tree"),
                                ("release-notes.md · " + ("已暂存" if language == "zh" else "Staged"), "+staged version")):
            path_rows = [e for e in controls() if e[1] == "listItem" and e[2] == "docs/release-notes.md"]
            auto("press", path_rows[0 if "working" in expected else 1][0], "--settle")
            auto("wait", "--text", expected, "--timeout", "20s", "--settle")
            comparison = ("暂存区 → 工作树" if language == "zh" else "Index → Working tree") if "working" in expected else ("HEAD → 暂存区" if language == "zh" else "HEAD → Index")
            auto("wait", "--text", comparison, "--timeout", "20s", "--settle")
            assert any(e[1] == "text" and e[2] == "docs/release-notes.md" for e in controls()), "Diff header lost original path"
            current = controls()
            action = next(e for e in current if e[1] == "button" and e[2] == commit_label)
            editor = next(e for e in current if e[1] == "textField" and e[2] == summary_label)
            active = next(e for e in detail_tabs() if "checked" in e[3])
            assert active[2] == label, f"Changes lost version identity: {active}"
            x, y, w, h = bounds(action)
            ex, ey, ew, eh = bounds(editor)
            assert h > 0 and y + h <= 640 and "offscreen" not in action[3], "Changes commit action is unreachable"
            detail_at = next(i for i, e in enumerate(current) if e[1] == "tabList" and e[2] == details_label)
            detail_viewport = next(e for e in reversed(current[:detail_at]) if e[1] == "scroll")
            dx, dy, dw, dh = bounds(detail_viewport)
            assert ex >= dx - 1 and ex + ew <= dx + dw + 1 and ey >= dy + dh - 1, "commit editor is outside the detail column"
            assert ey + eh <= y + 1, "commit button overlaps the editor"
            assert "Acceptance draft with staged changes" in auto("snapshot", "--full"), "Changes lost draft"
            version = "unstaged" if "working" in expected else "staged"
            capture("-changes-" + version)
            first_addition = next(i for i, e in enumerate(current) if e[1] == "insertion")
            gutter = next(e for e in reversed(current[:first_addition]) if e[1] == "button")
            auto("press", gutter[0], "--settle")
            selected_text = "已选 1 行" if language == "zh" else "1 selected ·"
            auto("wait", "--text", selected_text, "--timeout", "20s", "--settle")
            action_label = ("暂存选区" if language == "zh" else "Stage selection") if version == "unstaged" else ("取消暂存选区" if language == "zh" else "Unstage selection")
            selection_action = next(e for e in controls() if e[1] == "button" and e[2] == action_label)
            sx, sy, sw, sh = bounds(selection_action)
            assert sx >= 0 and sx + sw <= width and sy + sh <= 640 and "offscreen" not in selection_action[3], "selection action is clipped"
            capture("-changes-selection-" + version)
            press("button", "清除差异行选择" if language == "zh" else "Clear the Line Selection")
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
