#!/usr/bin/env python3
"""Exercise workspace quick actions only in a freshly created disposable checkout."""
import argparse
import hashlib
import json
import os
import pathlib
import re
import signal
import subprocess
import tempfile
import time


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--app", type=pathlib.Path, required=True)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    parser.add_argument("--scenario", choices=["all"], default="all")
    parser.add_argument("--theme", choices=["light", "dark"], default="light")
    parser.add_argument("--size", default="1320x840")
    parser.add_argument("--language", choices=["en", "zh"], default="en")
    args = parser.parse_args()
    zh = {
        "Save changes…": "保存改动…", "Save changes including untracked files": "保存改动（含未跟踪文件）",
        "Stash all": "一键 Stash", "Stash saved": "已保存 Stash", "Stash actions": "Stash 操作",
        "Apply": "应用", "Stash applied": "已应用 Stash", "Cancel": "取消",
        "Clean workspace…": "一键清理…", "Clean workspace": "清理工作区",
        "Restore existing files / unstage": "恢复已有文件 / 撤销暂存",
        "Workspace cleaned": "已清理工作区", "New content · Keep": "新增内容 · 保留",
        "New content · Delete": "新增内容 · 将删除", "Also delete new files and folders": "同时删除新增文件和目录",
        "Cleanup incomplete": "清理未完成",
    }

    def tr(text):
        return zh.get(text, text) if args.language == "zh" else text
    project = pathlib.Path(__file__).resolve().parent.parent
    binary = args.app.resolve() / "Contents/MacOS/supergit"
    automation = project / "build/mygo-automation"
    if not binary.is_file() or not automation.is_file():
        parser.error("Build make e2e-build and GOFLAGS=-tags=e2e make app first")
    out = args.output.resolve()
    out.mkdir(parents=True, exist_ok=True)
    results = []
    with tempfile.TemporaryDirectory(prefix="supergit-workspace-actions-") as directory:
        root = pathlib.Path(directory)
        repo = root / "repo"
        config = root / "config"
        (config / "supergit").mkdir(parents=True)
        (config / "supergit/settings.json").write_text(json.dumps({"language": args.language, "theme": args.theme, "autoFetch": False, "autoCheckUpdates": False}))
        env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
        env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")

        def git(*argv):
            return subprocess.check_output(["git", "-C", str(repo), *argv], env=env, text=True)

        def put(name, content):
            path = repo / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(content)

        subprocess.run(["git", "init", "-b", "main", str(repo)], env=env, check=True, capture_output=True)
        git("config", "user.name", "Fixture")
        git("config", "user.email", "fixture@example.invalid")
        for name in ["staged.txt", "unstaged.txt", "mixed.txt", "deleted.txt", "rename-old.txt"]:
            put(name, "base " + name + "\n")
        put(".gitignore", "ignored/\n*.keep\n")
        git("add", "--all")
        git("commit", "--no-gpg-sign", "-m", "Workspace quick actions fixture")
        head = git("rev-parse", "HEAD")
        put("staged.txt", "staged changes\n")
        put("mixed.txt", "staged mixed\n")
        put("added-staged.txt", "staged new file\n")
        git("add", "--", "staged.txt", "mixed.txt", "added-staged.txt")
        put("mixed.txt", "working mixed\n")
        put("unstaged.txt", "working changes\n")
        git("rm", "--", "deleted.txt")
        git("mv", "--", "rename-old.txt", "rename-new.txt")
        put("new/文件 space.txt", "new file\n")
        put("ignored/keep.txt", "ignored secret fixture\n")
        put("new/preserve.keep", "protected alongside deletions\n")
        outside = root / "external.txt"
        outside.write_text("external must survive\n")
        (repo / "external-link").symlink_to(outside)

        def fingerprint():
            files = {}
            for folder, dirs, names in os.walk(repo, followlinks=False):
                dirs[:] = [d for d in dirs if d != ".git"]
                for name in names + [d for d in dirs if (pathlib.Path(folder) / d).is_symlink()]:
                    p = pathlib.Path(folder) / name
                    files[str(p.relative_to(repo))] = ("link:" + os.readlink(p)) if p.is_symlink() else hashlib.sha256(p.read_bytes()).hexdigest()
            return {"head": git("rev-parse", "HEAD"), "branch": git("symbolic-ref", "HEAD"), "index": git("ls-files", "--stage", "-z"), "files": files}

        original = fingerprint()
        (out / "before.json").write_text(json.dumps(original, indent=2))
        socket = root / "app.sock"
        env.update(SUPERGIT_TEST_CONFIG_DIR=str(config), MYGO_AUTOMATION=str(socket), SUPERGIT_WINDOW_SIZE=args.size)
        session = "workspace-actions-" + str(os.getpid())
        with (out / "app.log").open("w") as log, (out / "automation.log").open("w") as trace:
            launch = ["open", "-n", "-W", "--stdout", str(out / "app.log"), "--stderr", str(out / "app.log")]
            for key in ("GIT_CONFIG_GLOBAL", "GIT_CONFIG_NOSYSTEM", "GIT_TERMINAL_PROMPT", "SUPERGIT_TEST_CONFIG_DIR", "MYGO_AUTOMATION", "SUPERGIT_WINDOW_SIZE"):
                launch += ["--env", key + "=" + env[key]]
            launch += [str(args.app.resolve()), "--args", str(repo)]
            app = subprocess.Popen(launch, cwd=project, env=env, stdout=log, stderr=subprocess.STDOUT)

            def auto(*argv):
                p = subprocess.run([str(automation), *argv, "--session", session], cwd=project, text=True, capture_output=True, timeout=40)
                trace.write(repr(argv) + "\n" + p.stdout + p.stderr + "\n")
                trace.flush()
                if p.returncode:
                    raise RuntimeError(p.stdout + p.stderr)
                return p.stdout

            def controls():
                entries = []
                for line in auto("snapshot", "--full").splitlines():
                    m = re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)', line)
                    if m:
                        entries.append((m[1], m[2], json.loads(m[3]) if m[3] else "", m[4]))
                return entries

            def press(label, role="button", contains=False):
                label = tr(label)
                deadline = time.monotonic() + 25
                while True:
                    entries = controls()
                    matches = [e for e in entries if e[1] == role and (label in e[2] if contains else label == e[2]) and "disabled" not in e[3]]
                    if matches:
                        auto("press", matches[-1][0], "--settle")
                        return
                    if app.poll() is not None or time.monotonic() > deadline:
                        raise AssertionError(f"Missing enabled {role} {label}: {entries}")
                    time.sleep(0.05)

            def wait(text):
                auto("wait", "--text", tr(text), "--timeout", "25s", "--settle")

            def checkpoint(name):
                (out / (name + ".snapshot.txt")).write_text(auto("snapshot", "--full"))
                auto("screenshot", str(out / (name + ".png")))

            def passed(name):
                results.append(name)
                print("PASS " + name, flush=True)

            def preserved():
                assert git("rev-parse", "HEAD") == head
                assert git("symbolic-ref", "HEAD") == "refs/heads/main\n"
                assert (repo / "ignored/keep.txt").read_text() == "ignored secret fixture\n"
                assert (repo / "new/preserve.keep").read_text() == "protected alongside deletions\n"
                assert outside.read_text() == "external must survive\n"

            try:
                deadline = time.monotonic() + 25
                while not socket.exists():
                    if app.poll() is not None or time.monotonic() > deadline:
                        raise RuntimeError("Native app did not expose automation socket")
                    time.sleep(0.05)
                auto("open", "--socket", str(socket))
                wait("Workspace quick actions fixture")
                checkpoint("titlebar")
                press("Save changes…", "menuButton")
                wait("Save changes including untracked files")
                checkpoint("stash-menu")
                auto("key", "Escape", "--settle")
                press("Stash all")
                wait("Stash saved")
                assert not git("status", "--porcelain").strip()
                preserved()
                stash = git("rev-parse", "refs/stash").strip()
                passed("one-click Stash saves all changes and preserves ignored files and HEAD")
                press("SuperGit workspace", "listItem", contains=True)
                press("Stash actions", "menuButton")
                press("Apply", "menuItem")
                wait("Stash applied")
                assert git("rev-parse", "refs/stash").strip() == stash
                assert fingerprint() == original
                passed("existing Apply restores staged/unstaged bytes and additions exactly")

                press("Clean workspace…")
                wait("Restore existing files / unstage")
                checkpoint("clean-default")
                press("Cancel")
                assert fingerprint() == original
                passed("cancel does not change index or files")
                press("Clean workspace…")
                wait("Restore existing files / unstage")
                press("Clean workspace")
                wait("Workspace cleaned")
                assert not git("diff", "--cached").strip()
                assert not git("diff").strip()
                assert (repo / "added-staged.txt").read_text() == "staged new file\n"
                assert (repo / "rename-new.txt").read_text() == "base rename-old.txt\n"
                assert (repo / "new/文件 space.txt").read_text() == "new file\n"
                assert (repo / "external-link").is_symlink()
                preserved()
                passed("default cleanup restores HEAD and unstages while retaining additions")

                press("Clean workspace…")
                wait("New content · Keep")
                assert any(e[1] == "button" and e[2] == tr("Clean workspace") and "disabled" in e[3] for e in controls())
                press("Also delete new files and folders", "checkBox")
                wait("New content · Delete")
                checkpoint("clean-delete")
                press("Clean workspace")
                wait("Workspace cleaned")
                # Wait for state, not a possibly retained success notification.
                deadline = time.monotonic() + 25
                while git("status", "--porcelain").strip():
                    if time.monotonic() > deadline:
                        raise AssertionError("opt-in cleanup did not produce clean status")
                    time.sleep(0.05)
                assert not (repo / "added-staged.txt").exists()
                assert not (repo / "external-link").is_symlink()
                preserved()
                passed("opt-in deletion leaves clean status and preserves ignored files and symlink targets")

                put("unstaged.txt", "preview version\n")
                auto("key", "cmd+r", "--settle")
                press("Clean workspace…")
                wait("Restore existing files / unstage")
                assert not any(e[1] == "checkBox" and e[2] == tr("Also delete new files and folders") and "checked" in e[3] for e in controls())
                put("unstaged.txt", "changed after preview\n")
                before_stale = fingerprint()
                press("Clean workspace")
                wait("Cleanup incomplete")
                assert fingerprint() == before_stale
                checkpoint("stale-refused")
                passed("reopening resets deletion; changed bytes invalidate old confirmation without writes")
                (out / "after.json").write_text(json.dumps(fingerprint(), indent=2))
            except BaseException:
                try:
                    checkpoint("failure")
                except Exception:
                    pass
                raise
            finally:
                (out / "results.json").write_text(json.dumps({"theme": args.theme, "size": args.size, "language": args.language, "passed": results, "complete": len(results) == 6}, indent=2))
                subprocess.run([str(automation), "close", "--session", session], cwd=project, capture_output=True)
                # Terminate only the exact bundle instance using this script's unique fixture.
                processes = subprocess.check_output(["ps", "-axo", "pid=,args="], text=True)
                for line in processes.splitlines():
                    fields = line.strip().split(None, 1)
                    if len(fields) == 2 and fields[1] == str(binary) + " " + str(repo):
                        try:
                            os.kill(int(fields[0]), signal.SIGTERM)
                        except ProcessLookupError:
                            pass
                app.terminate()
                try:
                    app.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    app.kill()
                    app.wait(timeout=5)


if __name__ == "__main__":
    main()
