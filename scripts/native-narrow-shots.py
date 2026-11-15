#!/usr/bin/env python3
"""Capture the history page at the minimum window size in five looks.

The narrow layout was reviewed once, in English with a light palette. A reader
running Chinese in a dark palette at 980x640 sees the same columns with longer
labels and darker capsules, so this launches the app four times (zh/en x
light/dark) at the minimum size and keeps one screenshot each under build/.
A fifth run repeats the Chinese light look at the large font preset, where the
two file lists used to disagree about how tall a row is.

Each run keeps two screenshots: the commit's files and patch, then the working
tree's, since the two lists are the same pane and have to read the same.

The fixture is built here instead of reusing a demo: the check needs a few
branches, a remote, a tag and both Chinese and English subjects long enough to
truncate, so one look can show every piece the narrow layout has to fit.
"""
import json
import os
import pathlib
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
shots = build / "narrow-shots"

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
git("remote", "add", "origin", str(remote))
git("push", "-u", "origin", "main", "feature/narrow-window-ref-capsules")
(workspace / "docs/release-notes.md").write_text("release notes\nautumn campaign\nedited in the working tree\n")

# The first commit subject is fixture text, so the wait works in both locales.
ready_text = "Initial commit"

combinations = [
    ("zh-light", "zh", "light", None),
    ("zh-dark", "zh", "dark", None),
    ("en-light", "en", "light", None),
    ("en-dark", "en", "dark", None),
    # The large preset is where the working tree's fixed row and the commit's
    # scaled one were furthest apart.
    ("zh-light-large", "zh", "light", 1.15),
]

shots.mkdir(parents=True, exist_ok=True)
failures = []
for name, language, theme, font_scale in combinations:
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
        "SUPERGIT_WINDOW_SIZE": "980x640",
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
        # The History page is the narrow layout under review; the app may
        # restore another page, so ask for it instead of assuming.
        auto("key", "cmd+2", "--settle")
        auto("wait", "--text", ready_text, "--timeout", "30s", "--settle")
        # The files column and the patch load in a goroutine; settle returns
        # while they are still reading, so wait for the loading copy to leave
        # before the screenshot or the columns look empty for no reason.
        loading = "正在加载差异…" if language == "zh" else "Loading diff…"
        auto("wait", "--text", loading, "--gone", "--timeout", "20s", "--settle")
        shot = auto("screenshot", str(shots / (name + ".png"))).strip()
        print(shot, flush=True)
        (shots / (name + ".snapshot.txt")).write_text(auto("snapshot", "-i"))
        # The working tree's files sit under the same pane as the commit's;
        # Home walks the list back to the working tree's row. The patch rows
        # are drawn, not labeled, so wait for the changed file's row and then
        # for the patch's loading copy to leave.
        auto("key", "home", "--settle")
        auto("wait", "--text", "docs/release-notes.md", "--timeout", "20s", "--settle")
        auto("wait", "--text", loading, "--gone", "--timeout", "20s", "--settle")
        shot = auto("screenshot", str(shots / (name + "-worktree.png"))).strip()
        print(shot, flush=True)
        (shots / (name + "-worktree.snapshot.txt")).write_text(auto("snapshot", "-i"))
        print(name, "captured", flush=True)
    except Exception as error:  # keep the other three looks running
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
