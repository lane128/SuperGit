#!/usr/bin/env python3
"""Exercise reference detail actions against disposable repositories only."""
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
artifacts = build / "lobe-reference-smoke"
artifacts.mkdir(parents=True, exist_ok=True)
session = "supergit-lobe-references"


def assert_header_layout(snapshot, window_width):
    entries = []
    for line in snapshot.splitlines():
        match = re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)', line)
        if match:
            bounds = re.search(r"bounds=([-0-9.]+),([-0-9.]+),([0-9.]+)x([0-9.]+)", match[4])
            if bounds:
                entries.append((match[2], json.loads(match[3]) if match[3] else "", match[4], tuple(map(float, bounds.groups()))))
    previous_right = 0
    for role, label in (("button", "Command palette"), ("button", "Branches"),
                        ("menuButton", "Save changes…"), ("menuButton", "Sync")):
        matches = [e for e in entries if e[0] == role and e[1] == label and e[3][1] < 76]
        assert len(matches) == 1, f"Missing title action: {label}"
        entry = matches[0]
        x, y, width, height = entry[3]
        assert "offscreen" not in entry[2] and x >= previous_right and x + width <= window_width, f"Overlapping or offscreen title action: {entry}"
        previous_right = x + width
    heading = next(e for e in entries if e[0] == "text" and e[1] == "Branches")
    search = next(e for e in entries if e[0] == "textField" and "Search branches and tags" in e[2])
    create = next(e for e in entries if e[0] == "button" and e[1] == "New branch…")
    assert heading[3][0] + heading[3][2] + 6 <= search[3][0], f"Heading overlaps search: {heading}, {search}"
    assert search[3][0] + search[3][2] <= create[3][0] and create[3][0] + create[3][2] <= window_width, f"Page controls overlap or overflow: {search}, {create}"
    help_text = next(e for e in entries if e[0] == "text" and e[1].startswith("Switching branches requires"))
    reference_scroll = next(e for e in entries if e[0] == "scroll" and heading[3][0] - 16 <= e[3][0] <= help_text[3][0] and e[3][1] > heading[3][1])
    assert help_text[3][1] >= heading[3][1] + heading[3][3] and help_text[3][1] + help_text[3][3] <= reference_scroll[3][1], f"Wrapped branch help escapes header: {help_text}, {reference_scroll}"


def run():
    with tempfile.TemporaryDirectory(prefix="supergit-lobe-references.") as directory:
        root = pathlib.Path(directory)
        repo, remote, home = (root / name for name in ("workspace", "remote.git", "home"))
        settings = home / "Library/Application Support/supergit"
        settings.mkdir(parents=True)
        (settings / "settings.json").write_text(json.dumps({
            "language": "en", "theme": os.environ.get("SUPERGIT_REFERENCE_THEME", "light"),
            "fontScale": float(os.environ.get("SUPERGIT_REFERENCE_FONT_SCALE", "1")), "autoFetch": False,
        }))
        git_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
        git_env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")

        def git(*args, directory=repo):
            return subprocess.check_output(["git", "-C", str(directory), *args], env=git_env, text=True).strip()

        subprocess.run(["git", "init", "-b", "main", str(repo)], env=git_env, check=True, capture_output=True)
        git("config", "user.name", "Fixture")
        git("config", "user.email", "fixture@example.invalid")
        (repo / "文件 space.txt").write_text("reference fixture\n")
        git("add", "--all")
        git("commit", "--no-gpg-sign", "-m", "Reference fixture base")
        original_head, original_index = git("rev-parse", "HEAD"), git("ls-files", "--stage")
        subprocess.run(["git", "clone", "--bare", str(repo), str(remote)], env=git_env, check=True, capture_output=True)
        git("branch", "topic")
        git("tag", "-a", "--no-sign", "-m", "Original annotation", "release")
        tag_object = git("rev-parse", "refs/tags/release")
        git("update-ref", "refs/heads/remote-topic", original_head, directory=remote)
        git("remote", "add", "origin", str(remote))
        git("fetch", "origin")
        socket = root / "app.sock"
        env = {**git_env, "HOME": str(home), "MYGO_AUTOMATION": str(socket), "SUPERGIT_WINDOW_SIZE": os.environ.get("SUPERGIT_REFERENCE_WINDOW_SIZE", "1320x640")}
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

            def wait(text):
                auto("wait", "--text", text, "--timeout", "25s", "--settle")

            def button(label, coordinate=False):
                # Completion notices can precede the repository reload. Wait
                # for the real action guard to clear before taking the next step.
                deadline = time.monotonic() + 25
                while True:
                    state = controls()
                    entries = [e for e in state if e[1] == "button" and e[2] == label and "disabled" not in e[3]]
                    if entries:
                        break
                    if app.poll() is not None or time.monotonic() > deadline:
                        raise AssertionError(f"Missing enabled button {label}: {state}")
                    time.sleep(0.05)
                entry = entries[-1]
                if coordinate:
                    assert "offscreen" not in entry[3], f"Offscreen footer: {entry}"
                    match = re.search(r"bounds=([-0-9.]+),([-0-9.]+),([0-9.]+)x([0-9.]+)", entry[3])
                    assert match, f"Missing footer bounds: {entry}"
                    x, y, width, height = map(float, match.groups())
                    auto("click", str(x + width / 2), str(y + height / 2), "--settle")
                else:
                    auto("press", entry[0], "--settle")

            def select(label, description=None):
                entries = [e for e in controls() if e[1] == "listItem" and e[2] == label and (description in e[3] if description is not None else 'description="origin/' not in e[3])]
                assert len(entries) == 1, f"Ambiguous reference row {label}: {entries}"
                auto("act", entries[0][0], "scrollIntoView", "--settle")
                entries = [e for e in controls() if e[1] == "listItem" and e[2] == label and (description in e[3] if description is not None else 'description="origin/' not in e[3])]
                auto("press", entries[0][0], "--settle")

            def wait_ref(ref, expected, directory=repo):
                deadline = time.monotonic() + 25
                while git("for-each-ref", "--format=%(objectname)", ref, directory=directory) != expected:
                    if app.poll() is not None or time.monotonic() > deadline:
                        raise AssertionError(f"Git reference did not reach {ref}: {expected}")
                    time.sleep(0.05)

            def unchanged_checkout():
                assert git("rev-parse", "HEAD") == original_head
                assert git("symbolic-ref", "--short", "HEAD") == "main"
                assert git("ls-files", "--stage") == original_index
                assert (repo / "文件 space.txt").read_text() == "reference fixture\n"
                assert not git("status", "--porcelain")

            try:
                deadline = time.monotonic() + 20
                while not socket.exists():
                    if app.poll() is not None or time.monotonic() > deadline:
                        raise RuntimeError("Native reference automation unavailable")
                    time.sleep(0.05)
                auto("open", "--socket", str(socket))
                auto("wait", "--text", "Commit message", "--timeout", "30s", "--settle")
                auto("key", "cmd+1", "--settle")
                wait("Reference fixture base")
                auto("key", "cmd+2", "--settle")
                select("topic")
                button("Push branch…", coordinate=True)
                wait(f"Push topic at commit {original_head[:8]} to origin/topic.")
                button("Cancel")
                assert not git("for-each-ref", "--format=%(refname)", "refs/heads/topic", directory=remote)
                button("Push branch…", coordinate=True)
                wait(f"Push topic at commit {original_head[:8]} to origin/topic.")
                button("Push")
                wait("Pushed")
                wait_ref("refs/heads/topic", original_head, directory=remote)
                assert git("config", "branch.topic.remote") == "origin"
                assert git("config", "branch.topic.merge") == "refs/heads/topic"
                unchanged_checkout()
                print("PASS native noncurrent branch Push cancel/publish/upstream without checkout", flush=True)

                select("topic")
                button("Rename…", coordinate=True)
                wait("Rename branch")
                fields = [e for e in controls() if e[1] == "textField"]
                assert len(fields) == 1, f"Unexpected rename fields: {fields}"
                auto("fill", fields[0][0], "renamed", "--settle")
                button("Rename")
                wait("Branch renamed")
                wait_ref("refs/heads/renamed", original_head)
                assert not git("for-each-ref", "--format=%(refname)", "refs/heads/topic")
                assert git("config", "branch.renamed.merge") == "refs/heads/topic"
                unchanged_checkout()
                print("PASS native branch Rename preserves original tip and tracking configuration", flush=True)

                select("renamed")
                button("Delete branch…", coordinate=True)
                wait(original_head[:8])
                button("Cancel")
                assert git("rev-parse", "refs/heads/renamed") == original_head
                button("Delete branch…", coordinate=True)
                button("Delete")
                wait("Branch deleted")
                wait_ref("refs/heads/renamed", "")
                assert git("rev-parse", "refs/heads/topic", directory=remote) == original_head
                unchanged_checkout()
                print("PASS native local branch Delete cancel/confirm leaves remote and checkout intact", flush=True)

                select("release")
                button("Push to remote…", coordinate=True)
                wait(f"Push the tag “release” at object {tag_object[:8]} to origin.")
                button("Push")
                wait("Tag pushed")
                wait_ref("refs/tags/release", tag_object, directory=remote)
                unchanged_checkout()
                print("PASS native annotated tag Push retains original tag object", flush=True)

                select("release")
                button("Delete tag…", coordinate=True)
                wait(tag_object[:8])
                button("Cancel")
                assert git("rev-parse", "refs/tags/release") == tag_object
                button("Delete tag…", coordinate=True)
                button("Delete")
                wait("Tag deleted")
                wait_ref("refs/tags/release", "")
                assert git("rev-parse", "refs/tags/release", directory=remote) == tag_object
                unchanged_checkout()
                print("PASS native tag Delete cancel/confirm leaves remote tag intact", flush=True)

                select("remote-topic", "origin/remote-topic")
                button("Delete remote branch…", coordinate=True)
                wait(original_head[:8])
                button("Cancel")
                assert git("rev-parse", "refs/heads/remote-topic", directory=remote) == original_head
                button("Delete remote branch…", coordinate=True)
                button("Delete")
                wait("Remote branch deleted")
                wait_ref("refs/heads/remote-topic", "", directory=remote)
                wait_ref("refs/remotes/origin/remote-topic", "")
                unchanged_checkout()
                print("PASS native remote branch Delete cancel/leased confirm clears tracking ref", flush=True)

                # Initiate real non-fast-forward integration through visible UI.
                git("switch", "-c", "integration-topic")
                (repo / "integration.txt").write_text("incoming change\n")
                git("add", "--", "integration.txt")
                git("commit", "--no-gpg-sign", "-m", "Native merge target")
                incoming = git("rev-parse", "HEAD")
                git("switch", "main")
                (repo / "local.txt").write_text("local change\n")
                git("add", "--", "local.txt")
                git("commit", "--no-gpg-sign", "-m", "Native merge local")
                local = git("rev-parse", "HEAD")
                auto("key", "cmd+r", "--settle")
                wait("Native merge local")
                select("integration-topic")
                button("Merge into current branch", coordinate=True)
                wait(incoming[:8])
                button("Cancel")
                assert git("rev-parse", "HEAD") == local and not (repo / "integration.txt").exists()
                button("Merge into current branch", coordinate=True)
                button("Merge into current branch")
                wait("Merged")
                parents = git("rev-list", "--parents", "-n", "1", "HEAD").split()
                assert len(parents) == 3 and parents[1:] == [local, incoming], parents
                assert (repo / "integration.txt").read_text() == "incoming change\n"
                assert (repo / "local.txt").read_text() == "local change\n"
                assert not git("status", "--porcelain")
                print("PASS native Merge initiation cancel/confirm retains both parents and file sets", flush=True)

                git("switch", "-c", "rebase-onto")
                (repo / "onto.txt").write_text("onto change\n")
                git("add", "--", "onto.txt")
                git("commit", "--no-gpg-sign", "-m", "Native rebase target")
                onto = git("rev-parse", "HEAD")
                git("switch", "main")
                (repo / "replay.txt").write_text("replay change\n")
                git("add", "--", "replay.txt")
                git("commit", "--no-gpg-sign", "-m", "Native rebase source", "-m", "Preserve the complete replay message.")
                replay, replay_message = git("rev-parse", "HEAD"), git("show", "-s", "--format=%B", "HEAD")
                auto("key", "cmd+r", "--settle")
                select("main")
                wait("Native rebase source")
                select("rebase-onto")
                button("Rebase current onto this branch", coordinate=True)
                wait(onto[:8])
                button("Cancel")
                assert git("rev-parse", "HEAD") == replay and not (repo / "onto.txt").exists()
                button("Rebase current onto this branch", coordinate=True)
                button("Rebase current onto this branch")
                wait("Rebased")
                assert git("rev-parse", "HEAD^") == onto
                assert git("show", "-s", "--format=%B", "HEAD") == replay_message
                assert (repo / "onto.txt").read_text() == "onto change\n"
                assert (repo / "replay.txt").read_text() == "replay change\n"
                assert git("symbolic-ref", "--short", "HEAD") == "main" and not git("status", "--porcelain")
                print("PASS native Rebase initiation cancel/confirm retains source message and expected parent", flush=True)

                # Stress the existing list/search at the minimum window height.
                stress_head, stress_index = git("rev-parse", "HEAD"), git("ls-files", "--stage")
                names = [f"stress/{i:02d}-long-reference-跨平台-" + "descriptive-segment-" * 4 for i in range(35)]
                for name in names:
                    git("branch", name, stress_head)
                git("remote", "add", "backup", str(remote))
                git("fetch", "backup")
                auto("key", "cmd+r", "--settle")

                def search(text):
                    fields = [e for e in controls() if e[1] == "textField" and "Search branches and tags" in e[3]]
                    assert len(fields) == 1, f"Missing branch search: {fields}"
                    auto("fill", fields[0][0], text, "--settle")

                search("stress-no-such-reference")
                wait("No branches or tags match this search.")
                auto("key", "Enter", "--settle")
                assert git("symbolic-ref", "--short", "HEAD") == "main" and git("rev-parse", "HEAD") == stress_head
                def wait_reference_row(label=None, description=None):
                    # A text wait also matches the search field's own value.
                    # Require the resulting list row and the shared busy/loading
                    # guard to clear; Enter deliberately does nothing while loading.
                    deadline = time.monotonic() + 25
                    while True:
                        state = controls()
                        rows = [e for e in state if e[1] == "listItem"
                                and (label is None or e[2] == label)
                                and (description is None or f'description={json.dumps(description)}' in e[3])]
                        ready = any(e[1] == "menuButton" and e[2] == "Sync" and "disabled" not in e[3] for e in state)
                        if len(rows) == 1 and ready:
                            return rows
                        if app.poll() is not None or time.monotonic() > deadline:
                            raise AssertionError(f"Missing unique reference row {label or description}: {state}")
                        time.sleep(0.05)

                search(names[-1])
                rows = wait_reference_row(label=names[-1])
                match = re.search(r"bounds=([-0-9.]+),([-0-9.]+),([0-9.]+)x([0-9.]+)", rows[0][3])
                assert match
                x, y, width, height = map(float, match.groups())
                auto("click", str(x + min(width / 2, 100)), str(y + height / 2), "--settle")
                assert git("symbolic-ref", "--short", "HEAD") == "main", "Selecting a long row checked it out"
                wait_reference_row(label=names[-1])
                auto("key", "Enter", "--settle")
                deadline = time.monotonic() + 25
                while git("symbolic-ref", "--short", "HEAD") != names[-1]:
                    if app.poll() is not None or time.monotonic() > deadline:
                        raise AssertionError("Enter did not check out the focused long reference")
                    time.sleep(0.05)
                assert git("rev-parse", "HEAD") == stress_head and git("ls-files", "--stage") == stress_index
                assert not git("status", "--porcelain")
                search("origin/topic")
                wait_reference_row(description="origin/topic")
                search("backup/topic")
                wait_reference_row(description="backup/topic")
                assert git("symbolic-ref", "--short", "HEAD") == names[-1], "Remote search changed checkout"
                print("PASS native 35 long branches and two remotes: search isolation, mouse selection, focused Enter checkout", flush=True)
                snapshot = auto("snapshot", "--full")
                (artifacts / "final.snapshot.txt").write_text(snapshot)
                auto("screenshot", str(artifacts / "final.png"))
                window_width = int(env["SUPERGIT_WINDOW_SIZE"].split("x")[0])
                assert_header_layout(snapshot, window_width)
                print("PASS native long-reference title actions and Branches header remain visible without overlap", flush=True)
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
