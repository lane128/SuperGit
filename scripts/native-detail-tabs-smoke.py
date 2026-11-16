#!/usr/bin/env python3
"""Exercise Lobe detail tabs against a disposable repo and independent settings.

Requires make lobe-e2e-build and an unlocked macOS graphical session. Only the
new fixture is modified. Logs/screenshots remain in build/detail-tabs-smoke.
"""
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
artifacts = build / "detail-tabs-smoke"
artifacts.mkdir(parents=True, exist_ok=True)


def run():
    with tempfile.TemporaryDirectory(prefix="supergit-detail-tabs.") as directory:
        root = pathlib.Path(directory)
        repo = root / "workspace"
        home = root / "home"
        settings = home / "Library/Application Support/supergit"
        settings.mkdir(parents=True)
        (settings / "settings.json").write_text(json.dumps({"language": "en", "theme": "light"}))
        git_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
        git_env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")

        def git(*args):
            return subprocess.check_output(["git", "-C", str(repo), *args], env=git_env, text=True)

        subprocess.run(["git", "init", "-b", "main", str(repo)], env=git_env, check=True, capture_output=True)
        git("config", "user.name", "Fixture")
        git("config", "user.email", "fixture@example.invalid")
        paths = ["src/a.go", "other/a.go", *[f"files/file-{i:02d}.txt" for i in range(12)]]
        for path in paths:
            file = repo / path
            file.parent.mkdir(parents=True, exist_ok=True)
            file.write_text("base version\n")
        git("add", "--all")
        git("commit", "-m", "Tab fixture base")
        for path in paths:
            (repo / path).write_text("historical version\n")
        git("commit", "-am", "Tab fixture changes", "-m", "Complete message body\n\nRefs: detail-tabs-fixture")
        git("commit", "--allow-empty", "-m", "Tab fixture empty")
        other_repo = root / "other-worktree"
        git("worktree", "add", "--detach", str(other_repo), "HEAD")
        (repo / paths[0]).write_text("index version\n")
        git("add", "--", paths[0])
        (repo / paths[0]).write_text("working version\n")
        (repo / paths[1]).write_text("other working version\n")

        def fingerprint():
            return (git("rev-parse", "HEAD"), git("ls-files", "--stage"), {p: (repo / p).read_bytes() for p in paths})

        before = fingerprint()
        socket = root / "app.sock"
        session = "detail-tabs-" + str(os.getpid())
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
                result = []
                for line in auto("snapshot", "--full").splitlines():
                    match = re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)', line)
                    if match:
                        result.append((match[1], match[2], json.loads(match[3]) if match[3] else "", match[4]))
                return result

            def press(role, label, occurrence=None):
                deadline = time.monotonic() + 20
                while True:
                    matches = [e for e in controls() if e[1] == role and e[2] == label and "disabled" not in e[3]]
                    if matches:
                        break
                    if time.monotonic() > deadline:
                        raise AssertionError(f"missing enabled {role} {label!r}")
                    time.sleep(0.05)
                if occurrence is None and len(matches) != 1:
                    raise AssertionError(f"ambiguous {role} {label!r}: {matches}")
                auto("press", matches[occurrence or 0][0], "--settle")

            def wait(text):
                auto("wait", "--text", text, "--timeout", "20s", "--settle")

            def tabs():
                return [e for e in controls() if e[1] == "tab"]

            def check_file_group_tab(row):
                auto("focus", row[0], "--settle")
                auto("key", "Tab", "--settle")
                assert any(e[1] == "tab" and "focused" in e[3] for e in controls()), "Tab did not leave Files for the detail tab strip"
                auto("key", "shift+Tab", "--settle")
                assert any(e[0] == row[0] and "focused" in e[3] for e in controls()), "Shift+Tab did not return to the remembered file row"

            try:
                deadline = time.monotonic() + 20
                while not socket.exists():
                    if app.poll() is not None:
                        raise RuntimeError("native app exited; see app.log")
                    if time.monotonic() > deadline:
                        raise TimeoutError("native automation socket unavailable; see app.log")
                    time.sleep(0.05)
                auto("open", "--socket", str(socket))
                auto("key", "cmd+1", "--settle")
                wait("Tab fixture changes")
                press("listItem", "2 changes")
                wait("+working version")
                assert any(e[2] == "Commit" for e in tabs())
                group_names = {e[2] for e in controls() if e[1] == "button"}
                assert {"Modified Files", "Untracked Files", "Staged Files"}.issubset(group_names), "fixed file groups missing, including the empty untracked group"
                auto("screenshot", str(artifacts / "worktree-groups-graph.png"))
                summary_field = next(e for e in controls() if e[1] == "textField" and e[2] == "Commit summary (required)")
                body_field = next(e for e in controls() if e[1] == "textField" and e[2] == "Description (optional)")
                auto("fill", summary_field[0], "Draft survives tabs", "--settle")
                auto("fill", body_field[0], "Details retained", "--settle")
                assert "Author: " in auto("snapshot", "--full"), "effective commit author is missing"
                auto("screenshot", str(artifacts / "commit-form-identity.png"))
                press("listItem", paths[0], occurrence=0)
                wait("+working version")
                press("listItem", paths[0], occurrence=1)
                wait("+index version")
                assert {"a.go · Staged", "a.go · Unstaged"}.issubset({e[2] for e in tabs()}), "stage versions were merged"
                staged_row = [e for e in controls() if e[1] == "listItem" and e[2] == paths[0]][1]
                check_file_group_tab(staged_row)
                press("button", "Modified Files")
                press("tab", "a.go · Unstaged")
                press("button", "Refresh")
                wait("+working version")
                auto("wait", "--text", "Loading diff…", "--gone", "--timeout", "20s", "--settle")
                assert any(e[2] == "a.go · Unstaged" and "checked" in e[3] for e in tabs()), "folded-group refresh redirected active tab"
                stage_actions = [e for e in controls() if e[1] == "button" and e[2] == "Stage file"]
                assert len(stage_actions) == 1, "folded active unstaged tab lost its own stage header"
                assert fingerprint() == before, "folded-group refresh changed repository bytes"
                press("button", "Modified Files")
                press("listItem", paths[1])
                wait("+other working version")
                assert len([e for e in tabs() if e[2] == "a.go · Unstaged"]) == 2, "same basenames were merged"
                press("listItem", paths[1])
                assert len(tabs()) == 4, "reopening duplicated a tab"
                press("button", "Close " + paths[0], occurrence=0)
                assert any(e[2] == "a.go · Unstaged" and "checked" in e[3] for e in tabs()), "closing a background tab changed active target"
                press("button", "Close " + paths[1])
                press("button", "Close " + paths[0])
                assert len(tabs()) == 1 and "checked" in tabs()[0][3]
                assert "Draft survives tabs" in auto("snapshot", "--full"), "tab close lost draft"
                press("listItem", "Tab fixture changes")
                wait("Complete message body")
                wait("+historical version")
                history_file = next(e for e in controls() if e[1] == "listItem" and e[2] == paths[2])
                auto("focus", history_file[0], "--settle")
                auto("key", "Down", "--settle")
                assert any(e[2] == "file-01.txt" and "checked" in e[3] for e in tabs()), "historical file arrow escaped the file list"
                assert any(e[1] == "listItem" and e[2] == "Tab fixture changes" and "checked" in e[3] for e in controls()), "file navigation changed selected SHA"
                focused_file = next(e for e in controls() if e[1] == "listItem" and e[2] == paths[3])
                check_file_group_tab(focused_file)
                print("PASS native working and historical Files single Tab exit and remembered Shift+Tab entry", flush=True)
                for path in paths:
                    press("listItem", path)
                assert len(tabs()) == len(paths) + 1, "not all file tabs were opened"
                press("tab", "Summary")
                auto("screenshot", str(artifacts / "history-many-tabs.png"))
                press("listItem", "Tab fixture empty")
                assert len(tabs()) == 1, "new SHA retained old file tabs"
                for subject in ("Tab fixture changes", "Tab fixture empty", "Tab fixture changes"):
                    press("listItem", subject)
                wait("Complete message body")
                assert fingerprint() == before, "browsing changed HEAD, Index or working bytes"
                print("PASS native tab deduplication, close/draft, historical summary, many tabs, SHA replacement and read-only browsing", flush=True)

                press("listItem", paths[0])
                press("button", "Settings")
                windows = auto("windows")
                settings_window = next(re.match(r"(w\d+)", line)[1] for line in windows.splitlines() if "SuperGit · Settings" in line)
                auto("window", settings_window)
                press("checkBox", "Patches")
                auto("window", "w1")
                assert len(tabs()) == 1 and tabs()[0][2] == "Summary", "hiding patches did not return to summary"
                assert not any(e[1] in ("insertion", "deletion") for e in controls()), "hidden patch still has selectable lines"
                auto("window", settings_window)
                press("checkBox", "Patches")
                press("checkBox", "File list")
                auto("window", "w1")
                assert not any(e[1] == "listItem" and e[2] in paths for e in controls()), "file column setting did not hide files"
                auto("window", settings_window)
                press("checkBox", "File list")
                auto("window", "w1")
                press("listItem", paths[0])
                assert len(tabs()) == 2, "re-enabled patch cannot open a file"
                press("button", "Hide commit details")
                assert not tabs(), "hidden details retained accessible tabs"
                assert not any(e[1] in ("insertion", "deletion") for e in controls()), "hidden details retained patch rows"
                press("button", "Show commit details")
                assert len(tabs()) == 2, "restoring details lost valid file tabs"
                assert fingerprint() == before, "settings interaction changed repository data"
                print("PASS native legacy detail settings hide/restore patches and file column without repository writes", flush=True)

                press("listItem", "2 changes")
                press("listItem", paths[0], occurrence=0)
                wait("+working version")
                # The file view header follows the two file-list stage buttons.
                press("button", "Stage file", occurrence=-1)
                deadline = time.monotonic() + 15
                while git("show", ":" + paths[0]) != "working version\n":
                    if time.monotonic() > deadline:
                        raise AssertionError("stage did not update the intended Index entry")
                    time.sleep(0.05)
                auto("wait", "--text", "All changes", "--timeout", "20s", "--settle")
                assert not any(e[2] == "a.go · Unstaged" for e in tabs()), "refresh silently retargeted vanished file version"
                after_stage = fingerprint()
                assert after_stage[0] == before[0] and after_stage[2] == before[2], "stage changed HEAD or working bytes"
                for path in paths[1:]:
                    assert git("show", ":" + path) == "historical version\n", "stage touched an unrelated Index entry"
                press("button", "Refresh")
                wait("+working version")
                assert fingerprint() == after_stage, "refresh wrote repository data"
                assert "Draft survives tabs" in auto("snapshot", "--full"), "refresh lost draft"
                print("PASS native exact-file stage and refresh preserve HEAD, working bytes, unrelated Index entries and draft", flush=True)

                # A tracked patch exceeds the existing 4 MiB Git output cap.
                oversized = paths[2]
                (repo / oversized).write_text("x" * (5 << 20) + "\n")
                press("button", "Refresh")
                wait("Git 输出超过 4 MiB")
                wait("+other working version")
                (repo / oversized).write_text("recovered patch\n")
                press("button", "Retry")
                wait("+recovered patch")
                assert git("rev-parse", "HEAD") == before[0] and git("ls-files", "--stage") == after_stage[1]
                (artifacts / "final.snapshot.txt").write_text(auto("snapshot", "--full"))
                print("PASS native local patch failure and same-target retry retain other patches without repository writes", flush=True)

                press("listItem", paths[1])
                assert len(tabs()) > 1
                before_switch = fingerprint()
                other_index = subprocess.check_output(["git", "-C", str(other_repo), "ls-files", "--stage"], env=git_env, text=True)

                def switch_worktree(name):
                    group = next(e for e in controls() if e[1] == "disclosure" and e[2] == "Worktrees")
                    if "expanded" not in group[3]:
                        auto("press", group[0], "--settle")
                    press("button", name)
                    row = next(e for e in controls() if e[1] == "listItem" and e[2] == name)
                    auto("focus", row[0], "--settle")
                    auto("key", "Enter", "--settle")
                    deadline = time.monotonic() + 20
                    while not any(e[1] == "text" and e[2] == name and "description=" in e[3] for e in controls()):
                        if time.monotonic() > deadline:
                            raise AssertionError("repository title did not switch to " + name)
                        time.sleep(0.05)
                    press("button", "History")
                    wait("Tab fixture empty")

                switch_worktree("other-worktree")
                assert len(tabs()) == 1, "new root retained file tabs from prior checkout"
                press("listItem", "Tab fixture changes")
                wait("+historical version")
                press("listItem", paths[0])
                assert len(tabs()) == 2
                switch_worktree("workspace")
                assert len(tabs()) == 1, "returning root revived discarded file tabs"
                assert fingerprint() == before_switch, "repository navigation wrote original checkout data"
                assert subprocess.check_output(["git", "-C", str(other_repo), "ls-files", "--stage"], env=git_env, text=True) == other_index
                assert all((other_repo / path).read_text() == "historical version\n" for path in paths)
                print("PASS native repository-root switching isolates file tabs and leaves both checkouts unchanged", flush=True)

                # All writes below remain in the disposable fixture. Verify the
                # actual UI submit path, hook failure, full message and Amend.
                press("listItem", "3 changes")

                def fill_commit(summary, body):
                    fields = controls()
                    summary_field = next(e for e in fields if e[1] == "textField" and e[2] == "Commit summary (required)")
                    body_field = next(e for e in fields if e[1] == "textField" and e[2] == "Description (optional)")
                    auto("fill", summary_field[0], summary, "--settle")
                    auto("fill", body_field[0], body, "--settle")

                fill_commit("Native form commit", "First paragraph\n\nSecond paragraph")
                hook = repo / ".git/hooks/commit-msg"
                hook.write_text('#!/bin/sh\n: > "$(dirname "$0")/commit-msg-ran"\nexit 1\n')
                hook.chmod(0o755)
                before_commit = fingerprint()
                # Settling layout does not wait for repository/identity reads.
                # Send the shortcut only when the same form's Commit is ready.
                deadline = time.monotonic() + 20
                while True:
                    ready_controls = controls()
                    if any(e[1] == "button" and e[2] == "Commit" and "disabled" not in e[3] for e in ready_controls):
                        break
                    if time.monotonic() > deadline:
                        raise AssertionError("Commit form did not become ready before Cmd+Enter")
                    time.sleep(0.05)
                (artifacts / "submit-ready.controls.json").write_text(json.dumps(ready_controls, indent=2) + "\n")
                (artifacts / "submit-key.response.txt").write_text(auto("key", "cmd+Enter", "--settle"))
                deadline = time.monotonic() + 20
                while not hook.with_name("commit-msg-ran").exists():
                    if time.monotonic() > deadline:
                        raise AssertionError("Cmd+Enter did not reach the fixture's commit-msg hook")
                    time.sleep(0.05)
                wait("Author: Fixture <fixture@example.invalid>")
                assert fingerprint() == before_commit, "failed hook changed HEAD, Index or working bytes"
                snapshot = auto("snapshot", "--full")
                assert "Native form commit" in snapshot and "Second paragraph" in snapshot, "failed hook lost draft"
                hook.write_text("#!/bin/sh\nexit 0\n")
                deadline = time.monotonic() + 20
                while not any(e[1] == "button" and e[2] == "Commit" and "disabled" not in e[3] for e in controls()):
                    if time.monotonic() > deadline:
                        raise AssertionError("failed hook did not restore the enabled Commit button")
                    time.sleep(0.05)
                press("button", "Commit")
                deadline = time.monotonic() + 20
                while git("rev-parse", "HEAD") == before_commit[0]:
                    if time.monotonic() > deadline:
                        raise AssertionError("Commit did not write the fixture's staged tree")
                    time.sleep(0.05)
                wait("Author: Fixture <fixture@example.invalid>")
                actual = git("show", "-s", "--format=%an <%ae>%n%B", "HEAD")
                assert actual.rstrip("\n") == "Fixture <fixture@example.invalid>\nNative form commit\n\nFirst paragraph\n\nSecond paragraph", "UI commit changed author or complete message"
                committed = fingerprint()
                assert committed[0] != before_commit[0] and committed[1:] == before_commit[1:], "commit changed Index or working bytes"
                git("config", "user.name", "Current Fixture Committer")
                git("config", "user.email", "committer@example.invalid")
                press("button", "Refresh")
                wait("Author: Current Fixture Committer <committer@example.invalid>")
                press("checkBox", "Amend last commit")
                wait("Author: Fixture <fixture@example.invalid>")
                wait("Committer: Current Fixture Committer <committer@example.invalid>")
                amend_fields = controls()
                assert any(e[1] == "textField" and e[2] == "Commit summary (required)" and 'value="Native form commit"' in e[3] for e in amend_fields), "Amend did not prefill HEAD summary"
                assert any(e[1] == "textField" and e[2] == "Description (optional)" and "First paragraph" in e[3] and "Second paragraph" in e[3] for e in amend_fields), "Amend lost HEAD body while prefilling"
                fill_commit("Native amended title", "Amended body")
                press("button", "Amend commit")
                deadline = time.monotonic() + 20
                while git("rev-parse", "HEAD") == committed[0]:
                    if time.monotonic() > deadline:
                        raise AssertionError("Amend did not rewrite the fixture's HEAD")
                    time.sleep(0.05)
                actual = git("show", "-s", "--format=%an <%ae>%n%cn <%ce>%n%B", "HEAD")
                assert actual.rstrip("\n") == "Fixture <fixture@example.invalid>\nCurrent Fixture Committer <committer@example.invalid>\nNative amended title\n\nAmended body", "Amend replaced preserved author or message"
                amended = fingerprint()
                assert amended[0] != committed[0] and amended[1:] == committed[1:], "Amend changed Index or working bytes"
                print("PASS native commit form complete message, hook failure/draft retention, Index-only commit and preserved Amend author", flush=True)

                def wait_enabled_button(label):
                    deadline = time.monotonic() + 20
                    while not any(e[1] == "button" and e[2] == label and "disabled" not in e[3] for e in controls()):
                        if time.monotonic() > deadline:
                            raise AssertionError("button did not become available: " + label)
                        time.sleep(0.05)

                wait_enabled_button("Stage all")
                untracked_name = "new notes.txt"
                untracked_bytes = b"keep this untracked file\n"
                (repo / untracked_name).write_bytes(untracked_bytes)
                press("button", "Refresh")
                wait("Untracked Files")
                wait_enabled_button("Stage all")
                batch_before = fingerprint()
                press("button", "Stage all", occurrence=-1)
                deadline = time.monotonic() + 20
                while "??" in git("status", "--porcelain") or git("diff", "--name-only"):
                    if time.monotonic() > deadline:
                        raise AssertionError("Stage All did not include tracked and untracked files")
                    time.sleep(0.05)
                assert git("show", ":" + untracked_name).encode() == untracked_bytes
                assert fingerprint()[0] == batch_before[0] and fingerprint()[2] == batch_before[2], "Stage All changed HEAD or working bytes"
                wait("All changes")
                wait_enabled_button("Unstage all")
                press("button", "Unstage all", occurrence=-1)
                deadline = time.monotonic() + 20
                while git("diff", "--cached", "--name-only"):
                    if time.monotonic() > deadline:
                        raise AssertionError("Unstage All left Index changes")
                    time.sleep(0.05)
                assert fingerprint() == batch_before, "Stage All / Unstage All failed round trip"
                # Preserve a real staged version while discarding later working
                # edits. Untracked deletion must remain an independent action.
                wait_enabled_button("Stage all")
                git("add", "--", paths[1])
                (repo / paths[1]).write_text("new unstaged on staged version\n")
                press("button", "Refresh")
                wait("new unstaged on staged version")
                discard_before = fingerprint()
                # Repository confirmations belong to their initiating main
                # window even while the settings window has native focus.
                def answer_confirmation(message, answer):
                    auto("window", "w1")
                    wait(message)
                    press("button", answer, occurrence=-1)
                    auto("window", "w1")

                press("button", "Discard all")
                answer_confirmation("Keep the Index and every untracked file", "Cancel")
                assert fingerprint() == discard_before, "cancelled Discard All changed repository"
                press("button", "Discard all")
                answer_confirmation("Keep the Index and every untracked file", "Discard all")
                deadline = time.monotonic() + 20
                while git("diff", "--name-only"):
                    if time.monotonic() > deadline:
                        raise AssertionError("Discard All did not restore tracked working changes")
                    time.sleep(0.05)
                discarded = fingerprint()
                assert discarded[:2] == discard_before[:2], "Discard All changed HEAD or Index"
                assert (repo / untracked_name).read_bytes() == untracked_bytes, "Discard All deleted untracked data"
                assert (repo / paths[1]).read_text() == git("show", ":" + paths[1]), "Discard All lost staged version"
                press("listItem", untracked_name)
                press("button", "Delete file…", occurrence=-1)
                answer_confirmation("Delete untracked file", "Cancel")
                assert (repo / untracked_name).read_bytes() == untracked_bytes, "cancelled delete changed file"
                press("button", "Delete file…", occurrence=-1)
                answer_confirmation("Delete untracked file", "Delete")
                deadline = time.monotonic() + 20
                while (repo / untracked_name).exists():
                    if time.monotonic() > deadline:
                        raise AssertionError("confirmed file-header delete did not remove target")
                    time.sleep(0.05)
                assert fingerprint() == discarded, "untracked deletion changed unrelated tracked state"
                deadline = time.monotonic() + 20
                while any(e[1] == "listItem" and e[2] == untracked_name for e in controls()):
                    if time.monotonic() > deadline:
                        raise AssertionError("deleted untracked file remained in the refreshed Files list")
                    time.sleep(0.05)
                auto("screenshot", str(artifacts / "file-actions-final.png"))
                print("PASS native Stage All / Unstage All scopes, Discard All cancellation and Index preservation, explicit untracked file deletion", flush=True)

                history_before = fingerprint()
                imported = []
                for i in range(2401):
                    message = f"Automatic native commit {i:04d}"
                    imported.append(f"commit refs/heads/native-bulk-history\ncommitter Fixture <fixture@example.invalid> {1700000000+i} +0000\ndata {len(message)}\n{message}\n\n")
                subprocess.run(["git", "-C", str(repo), "fast-import", "--quiet"], input="".join(imported) + "done\n", env=git_env, text=True, check=True, capture_output=True)
                for i in range(24):
                    git("branch", f"native-ref-{i:02d}", "HEAD")
                count = int(git("rev-list", "--all", "--count"))
                press("button", "Refresh")
                wait(f"{count} commits")
                current = controls()
                assert not any(e[1] == "button" and e[2] == "Load more commits" for e in current), "history still requires a manual page button"
                assert any(e[1] == "button" and e[2] == "native-ref-23" for e in current), "expanded sidebar truncates actual references"
                visible = [e for e in current if e[1] == "listItem" and e[2].startswith("Automatic native commit")]
                assert len(visible) < 100, "native history built all offscreen commit controls"
                press("listItem", "Native amended title")
                selected = next(e for e in controls() if e[1] == "listItem" and e[2] == "Native amended title")
                auto("focus", selected[0], "--settle")
                auto("key", "end", "--settle")
                wait("Automatic native commit 0000")
                assert any(e[1] == "listItem" and e[2] == "Automatic native commit 0000" and "checked" in e[3] for e in controls()), "End did not select and reveal the oldest commit"
                assert fingerprint() == history_before, "automatic history browsing changed HEAD, Index or working bytes"
                auto("screenshot", str(artifacts / "complete-history-end.png"))
                print(f"PASS native automatic traversal of {count} commits, clipped rows, oldest-row keyboard reveal and complete sidebar references", flush=True)

                # Extend only the disposable fixture's current branch with real
                # file revisions, then enter its history through the UI menu.
                parent = git("rev-parse", "HEAD").strip()
                branch = git("symbolic-ref", "HEAD").strip()
                imported = []
                for i in range(2401):
                    message = f"Native file revision {i:04d}"
                    content = f"file history revision {i:04d}\n"
                    start = f"from {parent}\n" if i == 0 else ""
                    imported.append(f"commit {branch}\ncommitter Fixture <fixture@example.invalid> {1800000000+i} +0000\ndata {len(message)}\n{message}\n{start}M 100644 inline {paths[1]}\ndata {len(content)}\n{content}\n")
                subprocess.run(["git", "-C", str(repo), "fast-import", "--quiet"], input="".join(imported) + "done\n", env=git_env, text=True, check=True, capture_output=True)
                file_count = len(git("log", "--follow", "--format=%H", "HEAD", "--", paths[1]).splitlines())
                file_before = fingerprint()
                refreshed_count = int(git("rev-list", "--all", "--count"))
                press("button", "Refresh")
                wait(f"{refreshed_count} commits")
                wait("Automatic native commit 0000")
                changes = next(e for e in controls() if e[1] == "listItem" and re.fullmatch(r"\d+ changes?", e[2]))
                auto("press", changes[0], "--settle")
                press("listItem", paths[1], occurrence=0)
                row = next(e for e in controls() if e[1] == "listItem" and e[2] == paths[1])
                auto("focus", row[0], "--settle")
                auto("key", "Shift+F10", "--settle")
                press("menuItem", "File history…")
                wait(f"{file_count} commits")
                row = next(e for e in controls() if e[1] == "listItem" and e[2] == "Native file revision 2400")
                auto("focus", row[0], "--settle")
                auto("key", "end", "--settle")
                wait("Tab fixture base")
                assert any(e[1] == "listItem" and e[2] == "Tab fixture base" and "checked" in e[3] for e in controls()), "file history did not reveal its oldest commit"
                assert fingerprint() == file_before, "file history browsing changed repository state"
                auto("screenshot", str(artifacts / "complete-file-history-end.png"))
                print(f"PASS native automatic file history of {file_count} commits and oldest-row reveal", flush=True)
            except Exception:
                try:
                    (artifacts / "failure.snapshot.txt").write_text(auto("snapshot", "--full"))
                    auto("screenshot", str(artifacts / "failure.png"))
                    for window in auto("windows").splitlines():
                        match = re.match(r"(w\d+)", window)
                        if match:
                            auto("window", match[1])
                            (artifacts / ("failure-" + match[1] + ".snapshot.txt")).write_text(auto("snapshot", "--full"))
                    auto("window", "w1")
                except Exception:
                    pass
                raise
            finally:
                subprocess.run([str(automation), "close", "--session", session], cwd=project, capture_output=True)
                app.terminate()
                try:
                    app.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    app.kill()
                    app.wait()


if __name__ == "__main__":
    run()
