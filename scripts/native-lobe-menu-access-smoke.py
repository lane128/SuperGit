#!/usr/bin/env python3
"""Check history-menu bounds and keyboard traversal in an owned 35-branch repo."""
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
if len(candidates) != 1 or not automation.is_file():
    raise SystemExit("run make lobe-e2e-build first")
artifacts = pathlib.Path(tempfile.mkdtemp(prefix="lobe-menu-access-", dir=build))
with tempfile.TemporaryDirectory(prefix="supergit-menu-access.") as directory:
    root = pathlib.Path(directory)
    repo, home = root / "repo", root / "home"
    settings = home / "Library/Application Support/supergit/settings.json"
    settings.parent.mkdir(parents=True)
    settings.write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False}))
    env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
    env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")

    def git(*args):
        return subprocess.check_output(["git", "-C", str(repo), *args], env=env)

    subprocess.run(["git", "init", "-b", "main", str(repo)], env=env, check=True, capture_output=True)
    git("config", "user.name", "Earlier Author")
    git("config", "user.email", "earlier@example.invalid")
    (repo / "fixture.txt").write_text("earlier\n")
    git("add", "--", "fixture.txt")
    earlier_env = {**env, "GIT_AUTHOR_DATE": "2001-01-01T12:00:00+0000", "GIT_COMMITTER_DATE": "2001-01-01T12:00:00+0000"}
    subprocess.run(["git", "-C", str(repo), "commit", "-m", "Earlier filter exclusion"], env=earlier_env, check=True, capture_output=True)
    git("config", "user.name", "Menu Fixture")
    git("config", "user.email", "menu@example.invalid")
    (repo / "fixture.txt").write_text("base\n")
    git("commit", "-am", "Menu access fixture", "-m", "Complete menu source body")
    selected_day = git("show", "-s", "--format=%ad", "--date=short", "HEAD").decode().strip()
    for i in range(35):
        git("branch", f"stress/{i:02d}-长分支-" + "description-" * 8)
    for remote in ("origin", "backup"):
        bare = root / (remote + ".git")
        subprocess.run(["git", "init", "--bare", str(bare)], env=env, check=True, capture_output=True)
        git("remote", "add", remote, str(bare))
        git("push", remote, "main:topic")
        git("fetch", remote)

    def state():
        return (git("show-ref"), git("symbolic-ref", "HEAD"), git("ls-files", "--stage"),
                git("status", "--porcelain=v1", "-z"), (repo / "fixture.txt").read_bytes())

    before = state()
    socket = root / "app.sock"
    session = "menu-access-" + str(os.getpid())
    env.update(HOME=str(home), MYGO_AUTOMATION=str(socket), SUPERGIT_WINDOW_SIZE="980x640")
    with (artifacts / "app.log").open("w") as log, (artifacts / "automation.log").open("w") as trace:
        app = subprocess.Popen([str(candidates[0]), str(repo)], cwd=project, env=env, stdout=log, stderr=subprocess.STDOUT)

        def auto(*args):
            result = subprocess.run([str(automation), *args, "--session", session], cwd=project,
                                    capture_output=True, text=True, timeout=40)
            trace.write(repr(args) + "\n" + result.stdout + result.stderr + "\n")
            trace.flush()
            if result.returncode:
                raise RuntimeError((args, result.stdout, result.stderr))
            return result.stdout

        def snapshot():
            return auto("snapshot", "--full")

        def entries(text, role):
            result = []
            for line in text.splitlines():
                match = re.match(r'\s*(@e\d+) \[' + role + r'\](?: ("(?:[^"\\]|\\.)*"))?(.*)', line)
                if match:
                    result.append((match[1], json.loads(match[2]) if match[2] else "", match[3]))
            return result

        def history_row():
            rows = [e for e in entries(snapshot(), "listItem") if e[1] == "Menu access fixture"]
            assert len(rows) == 1, rows
            return rows[0][0]

        def opened():
            auto("focus", history_row(), "--settle")
            auto("key", "Shift+F10", "--settle")
            text = snapshot()
            assert entries(text, "menuItem"), text
            return text

        try:
            deadline = time.monotonic() + 20
            while not socket.exists():
                if app.poll() is not None or time.monotonic() > deadline:
                    raise RuntimeError("native automation unavailable")
                time.sleep(0.05)
            auto("open", "--socket", str(socket))
            auto("wait", "--text", "Commit message", "--timeout", "30s", "--settle")
            auto("key", "cmd+1", "--settle")
            auto("wait", "--text", "Menu access fixture", "--timeout", "20s", "--settle")
            auto("press", history_row(), "--settle")
            auto("wait", "--text", "Complete menu source body", "--timeout", "20s", "--settle")
            auto("wait", "--text", "Loading", "--gone", "--timeout", "20s", "--settle")
            text = opened()
            (artifacts / "menu.snapshot.txt").write_text(text)
            auto("screenshot", str(artifacts / "menu.png"))
            items = entries(text, "menuItem")
            outside = []
            for _, label, details in items:
                match = re.search(r"bounds=([-0-9.]+),([-0-9.]+),([0-9.]+)x([0-9.]+)", details)
                assert match, (label, details)
                x, y, w, h = map(float, match.groups())
                if "offscreen" in details or x < 0 or y < 0 or x + w > 980.1 or y + h > 640.1:
                    outside.append({"label": label, "bounds": [x, y, w, h]})
            assert not outside, f"Menu items outside 980x640: {outside}"
            enabled = {e[1] for e in items if "disabled" not in e[2]}
            reached = set()
            for _ in range(len(enabled) + 1):
                current = entries(snapshot(), "menuItem")
                reached.update(e[1] for e in current if "focused" in e[2])
                auto("key", "Down", "--settle")
            assert reached == enabled, ("keyboard traversal missed entries", sorted(enabled - reached))
            auto("key", "Escape", "--settle")

            def coordinate_click(role, prefix):
                matches = [e for e in entries(snapshot(), role) if e[1].startswith(prefix)]
                assert len(matches) == 1, matches
                bounds = re.search(r"bounds=([-0-9.]+),([-0-9.]+),([0-9.]+)x([0-9.]+)", matches[0][2])
                assert bounds
                x, y, w, h = map(float, bounds.groups())
                assert 0 <= x < x + w <= 980.1 and 0 <= y < y + h <= 640.1
                auto("click", str(x + w / 2), str(y + h / 2), "--settle")

            coordinate_dialogs = (("Check out a branch…", "Check out a branch at this commit"),
                                  ("Check out this commit (detached)…", "Check out commit"),
                                  ("Merge origin/topic into main…", "Merge into current branch"),
                                  ("Rebase main onto origin/topic…", "Rebase current onto this branch"),
                                  ("Create tag…", "Create tag"),
                                  ("Create annotated tag…", "Create annotated tag"),
                                  ("Create branch here…", "New branch"),
                                  ("Cherry-pick commit…", "Cherry-pick commit"),
                                  ("Revert commit…", "Revert commit"),
                                  ("Undo last commit…", "Undo last commit"),
                                  ("Reset main to This Commit…", "Reset current branch"))
            for prefix, title in coordinate_dialogs:
                opened()
                coordinate_click("menuItem", prefix)
                auto("wait", "--text", title, "--timeout", "20s", "--settle")
                if title == "Check out a branch at this commit":
                    auto("key", "Escape", "--settle")
                else:
                    coordinate_click("button", "Cancel")
                assert state() == before, "Coordinate activation/cancel mutated fixture"
            opened()
            for _ in range(len(enabled) + 1):
                focused = [e for e in entries(snapshot(), "menuItem") if "focused" in e[2]]
                if any(e[1].startswith("Reset main to This Commit…") for e in focused):
                    break
                auto("key", "Down", "--settle")
            else:
                raise AssertionError("Reset never gained keyboard focus")
            auto("key", "Enter", "--settle")
            auto("wait", "--text", "Reset current branch", "--timeout", "20s", "--settle")
            coordinate_click("button", "Cancel")
            assert state() == before, "Menu browsing mutated fixture"

            def search_field():
                lines = [line for line in snapshot().splitlines() if "[textField]" in line and 'placeholder="Search commits, authors, SHA, refs"' in line]
                assert len(lines) == 1, lines
                return (re.search(r"@e\d+", lines[0])[0], "", lines[0])

            filters = (("Filter to this author", "Menu Fixture"), ("Filter to this day", selected_day),
                       ("Filter to the last 7 days", None), ("Filter to this month", None))
            filter_queries = {}
            for label, expected_query in filters:
                auto("fill", search_field()[0], "", "--settle")
                auto("wait", "--text", "Earlier filter exclusion", "--timeout", "20s", "--settle")
                opened()
                coordinate_click("menuItem", label)
                rows = entries(snapshot(), "listItem")
                assert any(e[1] == "Menu access fixture" for e in rows), (label, rows)
                assert not any(e[1] == "Earlier filter exclusion" for e in rows), (label, rows)
                field = search_field()
                value = re.search(r'value=("(?:[^"\\\\]|\\\\.)*")', field[2])
                assert value, (label, field)
                query = json.loads(value[1])
                assert query and (expected_query is None or query == expected_query), (label, query)
                filter_queries[label] = query
                assert state() == before, "History filtering changed checkout or Git state"
                auto("focus", field[0], "--settle")
                auto("key", "Enter", "--settle")
                assert state() == before, "History search Enter changed checkout or Git state"
            (artifacts / "filtered.snapshot.txt").write_text(snapshot())
            auto("screenshot", str(artifacts / "filtered.png"))
            auto("fill", search_field()[0], "no-such-history-result", "--settle")
            auto("wait", "--text", "No commits match this search.", "--timeout", "20s", "--settle")
            assert "Could not read the repository" not in snapshot()
            auto("key", "Enter", "--settle")
            assert state() == before, "Empty search Enter changed checkout"
            auto("fill", search_field()[0], "", "--settle")
            git_dir, parked = repo / ".git", root / "parked-git"
            git_dir.rename(parked)
            try:
                auto("key", "cmd+r", "--settle")
                auto("wait", "--text", "Could not read the repository", "--timeout", "20s", "--settle")
                failure = snapshot()
                assert "Showing the last successful read." in failure
                assert "No commits match this search." not in failure
                assert any(e[1] == "Menu access fixture" for e in entries(failure, "listItem"))
                (artifacts / "read-failure.snapshot.txt").write_text(failure)
                auto("screenshot", str(artifacts / "read-failure.png"))
            finally:
                parked.rename(git_dir)
            coordinate_click("button", "Retry")
            auto("wait", "--text", "Could not read the repository", "--gone", "--timeout", "20s", "--settle")
            auto("wait", "--text", "Earlier filter exclusion", "--timeout", "20s", "--settle")
            assert state() == before, "Read failure/retry mutated fixture"
            (artifacts / "recovered.snapshot.txt").write_text(snapshot())

            def wait_control(role, label, ready=False):
                deadline = time.monotonic() + 25
                while True:
                    text = snapshot()
                    matches = [e for e in entries(text, role) if e[1] == label and "disabled" not in e[2]]
                    is_ready = any(e[1] == "Sync" and "disabled" not in e[2] for e in entries(text, "menuButton"))
                    if matches and (not ready or is_ready):
                        return matches[-1]
                    if app.poll() is not None or time.monotonic() > deadline:
                        raise AssertionError(("Missing ready control", role, label, text))
                    time.sleep(0.05)

            parent_sha = git("rev-parse", "HEAD^").decode().strip()
            parent_control = wait_control("button", parent_sha[:8], ready=True)
            auto("act", parent_control[0], "scrollIntoView", "--settle")
            parent_control = wait_control("button", parent_sha[:8], ready=True)
            assert "offscreen" not in parent_control[2], parent_control
            auto("screenshot", str(artifacts / "parent-navigation.png"))
            coordinate_click("button", parent_sha[:8])
            deadline = time.monotonic() + 25
            while not any(e[1] == "Earlier filter exclusion" and "checked" in e[2]
                          for e in entries(snapshot(), "listItem")):
                if app.poll() is not None or time.monotonic() > deadline:
                    raise AssertionError("Parent navigation did not select the original parent SHA")
                time.sleep(0.05)
            assert state() == before, "Parent navigation changed checkout or Git state"
            auto("press", wait_control("listItem", "Menu access fixture", ready=True)[0], "--settle")
            auto("wait", "--text", "Complete menu source body", "--timeout", "25s", "--settle")
            print("PASS native parent-SHA navigation; no Git mutation", flush=True)

            for current_ref, query, next_ref in (("All refs", "main", "main"), ("main", "All refs", "All refs")):
                auto("press", wait_control("button", current_ref, ready=True)[0], "--settle")
                fields = [e for e in entries(snapshot(), "textField") if 'placeholder="Search branches or tags"' in e[2]]
                assert len(fields) == 1, fields
                auto("fill", fields[0][0], query, "--settle")
                wait_control("listItem", next_ref)
                auto("screenshot", str(artifacts / ("ref-picker-main.png" if query == "main" else "ref-picker-all.png")))
                auto("key", "Enter", "--settle")
                wait_control("button", next_ref, ready=True)
                assert state() == before, "Ref picker changed checkout or Git state"

            auto("press", history_row(), "--settle")
            file_row = wait_control("listItem", "fixture.txt", ready=True)
            auto("focus", file_row[0], "--settle")
            auto("key", "Shift+F10", "--settle")
            blame_item = wait_control("menuItem", "Blame…")
            auto("press", blame_item[0], "--settle")
            toggle = wait_control("toggleButton", "Ignore whitespace")
            assert "checked" not in toggle[2], toggle
            auto("press", toggle[0], "--settle")
            toggle = wait_control("toggleButton", "Ignore whitespace")
            assert "checked" in toggle[2], toggle
            auto("wait", "--text", "Loading diff…", "--gone", "--timeout", "25s", "--settle")
            blame_state = snapshot()
            viewports = entries(blame_state, "scroll")
            assert viewports, blame_state
            for _, label, details in viewports:
                bounds = re.search(r"bounds=([-0-9.]+),([-0-9.]+),([0-9.]+)x([0-9.]+)", details)
                assert bounds, (label, details)
                x, y, w, h = map(float, bounds.groups())
                assert x >= 20 and y >= 20 and x + w <= 960.1 and y + h <= 620.1, (label, details)
            auto("screenshot", str(artifacts / "blame-selected.png"))
            auto("key", "Space", "--settle")
            toggle = wait_control("toggleButton", "Ignore whitespace")
            assert "checked" not in toggle[2], toggle
            auto("key", "Escape", "--settle")
            if entries(snapshot(), "toggleButton"):
                auto("screenshot", str(artifacts / "blame-escape-failure.png"))
                raise AssertionError("Blame did not close on first Escape")
            assert state() == before, "Blame whitespace toggle or close changed Git state"
            print("PASS native ref picker search/Enter and Blame pointer/Space toggle; no Git mutation", flush=True)
            (artifacts / "results.json").write_text(json.dumps({"visible_parent_coordinate_navigation": True, "blame_first_escape": True, "ref_picker_search_enter": True, "blame_toggle_pointer_space": True,"all_bounds_visible": True, "keyboard_entries": sorted(reached), "coordinate_picker_and_reset": True, "coordinate_dialogs": [label for label, _ in coordinate_dialogs], "keyboard_reset_activation": True, "coordinate_filters": filter_queries, "filter_enter_preserved_checkout": True, "empty_search_distinct_from_read_failure": True, "coordinate_retry_recovered": True, "git_state_preserved": True}, indent=2) + "\n")
            print("PASS menu bounds and Down traversal; no Git mutation", flush=True)
        except BaseException:
            (artifacts / "failure.snapshot.txt").write_text(snapshot())
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
                    app.wait()
print(artifacts)
