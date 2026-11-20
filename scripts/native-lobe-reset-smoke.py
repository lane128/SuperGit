#!/usr/bin/env python3
"""Exercise existing Reset/Undo UI with owned repositories and independent HOME."""
import argparse
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
artifacts = pathlib.Path(tempfile.mkdtemp(prefix="lobe-reset-", dir=build))
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--stale", action="store_true", help="Advance owned HEAD after opening each operation, then require rejection")
options = parser.parse_args()
results = []


def run_case(mode):
    case_artifacts = artifacts / mode
    case_artifacts.mkdir()
    with tempfile.TemporaryDirectory(prefix="supergit-reset.") as directory:
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
        path = repo / "文件 space.txt"
        path.write_text("base version\n")
        git("add", "--", path.name)
        git("commit", "-m", "Reset fixture base")
        base = git("rev-parse", "HEAD")
        path.write_text("tip version\n")
        git("commit", "-am", "Reset fixture tip")
        path.write_text("staged version\n")
        git("add", "--", path.name)
        path.write_text("working version\n")
        untracked = repo / "untracked.bin"
        untracked.write_bytes(b"owned untracked\x00\xff")
        before = (git("rev-parse", "HEAD"), git("ls-files", "--stage"), path.read_bytes(),
                  git("status", "--porcelain=v1", "-z"))
        before_index_bytes = (repo / ".git/index").read_bytes()
        before_refs = git("show-ref")
        socket = root / "app.sock"
        session = "reset-" + mode + "-" + str(os.getpid())
        env.update(HOME=str(home), MYGO_AUTOMATION=str(socket), SUPERGIT_WINDOW_SIZE="1320x640")
        with (case_artifacts / "app.log").open("w") as log, (case_artifacts / "automation.log").open("w") as trace:
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
                auto("press", ref(role, label), "--settle")

            def wait(text):
                auto("wait", "--text", text, "--timeout", "20s", "--settle")

            def open_action():
                label = "Reset fixture tip" if mode == "undo" else "Reset fixture base"
                row = ref("listItem", label)
                auto("focus", row, "--settle")
                auto("key", "Shift+F10", "--settle")
                (case_artifacts / "menu.snapshot.txt").write_text(snapshot())
                press("menuItem", "Undo last commit…" if mode == "undo" else "Reset main to This Commit…")
                wait("Undo last commit" if mode == "undo" else "Reset current branch")

            def unchanged():
                assert (git("rev-parse", "HEAD"), git("ls-files", "--stage"), path.read_bytes(),
                        git("status", "--porcelain=v1", "-z")) == before
                assert (repo / ".git/index").read_bytes() == before_index_bytes
                assert git("show-ref") == before_refs
                assert untracked.read_bytes() == b"owned untracked\x00\xff"

            try:
                deadline = time.monotonic() + 20
                while not socket.exists():
                    if app.poll() is not None or time.monotonic() > deadline:
                        raise RuntimeError("native automation unavailable")
                    time.sleep(0.05)
                auto("open", "--socket", str(socket))
                auto("wait", "--text", "Commit message", "--timeout", "30s", "--settle")
                auto("key", "cmd+1", "--settle")
                wait("Reset fixture tip")
                auto("wait", "--text", "Loading", "--gone", "--timeout", "20s", "--settle")
                open_action()
                press("button", "Cancel")
                unchanged()
                open_action()
                labels = {"soft": "Soft (keep staged)", "mixed": "Mixed (unstage)", "hard": "Hard (discard changes)"}
                if options.stale:
                    # Capture Hard's original baseline through its second
                    # confirmation before an external process advances HEAD.
                    if mode == "hard":
                        press("button", labels[mode])
                        wait("Hard reset")
                    tree = git("rev-parse", "HEAD^{tree}").strip().decode()
                    new_head = git("commit-tree", tree, "-p", before[0].strip().decode(),
                                   "-m", "External fixture advance").strip().decode()
                    git("update-ref", "refs/heads/main", new_head, before[0].strip().decode())
                    external_state = (git("rev-parse", "HEAD"), git("show-ref"), git("ls-files", "--stage"),
                                      git("status", "--porcelain=v1", "-z"), path.read_bytes(),
                                      (repo / ".git/index").read_bytes())
                    press("button", "Undo" if mode == "undo" else "Hard reset" if mode == "hard" else labels[mode])
                    wait("The current HEAD, branch, or pending operation changed." if mode == "undo"
                         else "The current HEAD or branch changed.")
                    assert (git("rev-parse", "HEAD"), git("show-ref"), git("ls-files", "--stage"),
                            git("status", "--porcelain=v1", "-z"), path.read_bytes(),
                            (repo / ".git/index").read_bytes()) == external_state, "stale operation changed external state"
                    assert untracked.read_bytes() == b"owned untracked\x00\xff"
                    assert git("symbolic-ref", "--short", "HEAD").strip() == b"main"
                    (case_artifacts / "stale.snapshot.txt").write_text(snapshot())
                    auto("screenshot", str(case_artifacts / "stale.png"))
                    results.append({"mode": mode, "external_head_preserved": True,
                                    "stale_operation_rejected": True, "index_worktree_untracked_preserved": True})
                    print(f"PASS native stale {mode}: rejected old baseline and preserved external HEAD/refs/Index/bytes", flush=True)
                    return
                if mode == "undo":
                    press("button", "Undo")
                    wait("Last commit undone")
                else:
                    press("button", labels[mode])
                    if mode == "hard":
                        wait("Hard reset")
                        unchanged()
                        press("button", "Cancel")
                        unchanged()
                        open_action()
                        press("button", labels[mode])
                        wait("Hard reset")
                        auto("screenshot", str(case_artifacts / "hard-confirmation.png"))
                        press("button", "Hard reset")
                    wait("Current branch reset")
                assert git("rev-parse", "HEAD") == base
                assert git("symbolic-ref", "--short", "HEAD").strip() == b"main"
                if mode in ("soft", "undo"):
                    assert git("ls-files", "--stage") == before[1]
                    assert path.read_bytes() == before[2]
                else:
                    assert git("show", ":" + path.name) == b"base version\n"
                    assert path.read_bytes() == (b"base version\n" if mode == "hard" else before[2])
                assert untracked.read_bytes() == b"owned untracked\x00\xff"
                assert not (repo / ".git/MERGE_HEAD").exists()
                (case_artifacts / "final.snapshot.txt").write_text(snapshot())
                auto("screenshot", str(case_artifacts / "final.png"))
                results.append({"mode": mode, "cancel_preserved_state": True,
                                "expected_head_index_worktree": True, "untracked_preserved": True})
                print(f"PASS native {mode}: cancellation, expected HEAD/Index/working bytes and untracked preservation", flush=True)
            except BaseException:
                try:
                    (case_artifacts / "failure.snapshot.txt").write_text(snapshot())
                except Exception:
                    pass
                print(case_artifacts, flush=True)
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


for mode in ("soft", "mixed", "hard", "undo"):
    run_case(mode)
(artifacts / "results.json").write_text(json.dumps(results, indent=2) + "\n")
print(artifacts)
