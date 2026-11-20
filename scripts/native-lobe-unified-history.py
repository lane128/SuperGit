#!/usr/bin/env python3
"""Check the unified History window against a disposable repository.

Asserts four aligned behaviors:
  1. one destination: no Changes / History / Stash page tabs;
  2. the working tree, stashes, and commits scroll together;
  3. a descendant on another branch precedes its checked-out parent;
  4. sidebar branches, remote branches, tags, and stashes reveal their rows
     without changing the list's ref scope or commit count.

Also records the git commands run during the session. Uses disposable Git data and an isolated HOME; the user's
settings, repositories and installed application are untouched.
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
binary = build / "supergit-e2e"
if not binary.is_file():
    binary = build / "supergit-lobe-e2e"
if not automation.is_file() or not binary.is_file():
    raise SystemExit("Run make e2e-build first")
artifacts = pathlib.Path(tempfile.mkdtemp(prefix="lobe-unified-history-", dir=build))
print(f"Artifacts: {artifacts}", flush=True)

git_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
git_env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0", LC_ALL="C")
results = {}

with tempfile.TemporaryDirectory(prefix="supergit-unified-history.") as directory:
    root = pathlib.Path(directory)
    repo = root / "workspace"
    subprocess.run(["git", "init", "--template=", "-b", "main", str(repo)], env=git_env, check=True, capture_output=True)

    def git(*args, **kwargs):
        return subprocess.check_output(["git", "-C", str(repo), *args], env=git_env, **kwargs)

    git("config", "user.name", "Fixture")
    git("config", "user.email", "fixture@example.invalid")
    (repo / "base.txt").write_text("base\n")
    git("add", "--", "base.txt")
    git("commit", "-m", "Root commit")
    root_sha = git("rev-parse", "HEAD").decode().strip()
    # Place the tag beyond the first 200-commit page and the remote midway
    # through the history. Fast-import adds the commits in one Git process.
    stream = []
    for i in range(240):
        subject = f"History row {i:03d}"
        parent = f"from {root_sha}\n" if i == 0 else ""
        stream.append(f"commit refs/heads/main\ncommitter Fixture <fixture@example.invalid> {1700000000+i} +0000\ndata {len(subject)}\n{subject}\n{parent}\n")
    git("fast-import", "--quiet", input=("".join(stream) + "done\n").encode())
    midpoint = git("rev-parse", "HEAD~120").decode().strip()
    git("update-ref", "refs/remotes/origin/checkpoint", midpoint)
    (repo / "base.txt").write_text("base\nmain tip\n")
    git("commit", "-am", "Main tip")
    head = git("rev-parse", "HEAD").decode().strip()
    # The child on future must connect down to its parent on main.
    git("switch", "-c", "future")
    future_env = {**git_env, "GIT_AUTHOR_DATE": "2030-01-01T00:00:00 +0000", "GIT_COMMITTER_DATE": "2030-01-01T00:00:00 +0000"}
    (repo / "future.txt").write_text("future\n")
    git("add", "--", "future.txt")
    subprocess.run(["git", "-C", str(repo), "commit", "-m", "Future side tip"], env=future_env, check=True, capture_output=True)
    future_sha = git("rev-parse", "HEAD").decode().strip()
    git("switch", "main")
    # A tag older than HEAD gives the sidebar something to locate without
    # changing the list's scope.
    git("tag", "-a", "root-tag", "-m", "Root release", root_sha)
    (repo / "base.txt").write_text("base\nmain tip\nstashed\n")
    git("stash", "push", "-m", "Fixture stash")
    (repo / "base.txt").write_text("base\nmain tip\ndirty\n")

    def fingerprint():
        return {
            "head": git("rev-parse", "HEAD").decode().strip(),
            "refs": hashlib.sha256(git("for-each-ref", "--format=%(refname) %(objectname)")).hexdigest(),
            "stashes": git("stash", "list").decode(),
            "index": hashlib.sha256(git("ls-files", "--stage", "-z")).hexdigest(),
            "status": git("status", "--porcelain", "-z").decode(),
        }

    before = fingerprint()
    results["head"] = head[:12]
    results["branch_tip_above_head"] = "Future side tip"

    home = root / "home"
    settings = home / "Library/Application Support/supergit"
    settings.mkdir(parents=True)
    (settings / "settings.json").write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False}))
    calls = root / "git-calls.log"
    shim = root / "shim"
    shim.mkdir()
    real_git = subprocess.check_output(["which", "git"], env=git_env).decode().strip()
    (shim / "git").write_text(f'#!/bin/sh\nprintf "%s\\n" "$*" >> "{calls}"\nexec "{real_git}" "$@"\n')
    (shim / "git").chmod(0o755)
    socket = root / "app.sock"
    session = "supergit-unified-history"
    env = {**git_env, "HOME": str(home), "MYGO_AUTOMATION": str(socket),
           "SUPERGIT_WINDOW_SIZE": "1280x800", "PATH": f"{shim}:{git_env.get('PATH', '')}"}
    with (artifacts / "app.log").open("w") as log, (artifacts / "automation.log").open("w") as trace:
        started = time.monotonic()
        app = subprocess.Popen([str(binary), str(repo)], cwd=project, env=env, stdout=log, stderr=subprocess.STDOUT)

        def auto(*args):
            completed = subprocess.run([str(automation), *args, "--session", session], cwd=project, capture_output=True, text=True, timeout=90)
            trace.write(repr(args) + "\n" + completed.stdout + completed.stderr + "\n")
            trace.flush()
            if completed.returncode:
                raise RuntimeError(f"{args}: {completed.stdout}{completed.stderr}")
            return completed.stdout

        def entries():
            found = []
            for text in auto("snapshot", "--full").splitlines():
                match = re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)', text)
                if match:
                    found.append((match[1], match[2], json.loads(match[3]) if match[3] else "", match[4]))
            return found

        def visible(entries_list, name):
            return [e for e in entries_list if e[2] == name and "offscreen" not in e[3]]

        try:
            deadline = time.monotonic() + 30
            while not socket.exists():
                if app.poll() is not None or time.monotonic() > deadline:
                    raise AssertionError("Native app did not open automation socket")
                time.sleep(0.05)
            auto("open", "--socket", str(socket))
            auto("wait", "--text", "Main tip", "--timeout", "30s", "--settle")
            auto("wait", "--text", "243 commits", "--timeout", "30s", "--settle")
            results["open_seconds"] = round(time.monotonic() - started, 3)
            auto("screenshot", str(artifacts / "history.png"))

            found = entries()
            tabs = [e for e in found if e[1] == "tab" and e[2] in ("Changes", "History", "Stash")]
            assert not tabs, f"the workbench still switches pages through tabs: {tabs}"
            results["page_tabs"] = "none"

            rows = [e for e in found if e[1] == "listItem" and "offscreen" not in e[3]]
            names = [e[2] for e in rows]
            results["list_rows"] = names
            worktree = visible(found, "1 change")
            stash = [e for e in found if e[1] == "listItem" and "Fixture stash" in e[2] and "offscreen" not in e[3]]
            head_row = visible(found, "Main tip")
            side_row = visible(found, "Future side tip")
            assert worktree, "the working tree is not a row of the log"
            assert stash, "the stash is not a row of the log"
            assert head_row, "the checked-out commit is not in the log"
            assert side_row, "the newer side branch is missing from the log"
            positions = [found.index(e) for e in (worktree[0], stash[0], side_row[0], head_row[0])]
            assert positions == sorted(positions), f"rows out of order: worktree={positions[0]} stash={positions[1]} side={positions[2]} head={positions[3]}"
            assert "checked" in head_row[0][3], "History did not open on the checked-out commit"
            results["row_order"] = "worktree, stash, child on future, HEAD"

            worktree_ref = worktree[0][0]
            auto("focus", worktree_ref, "--settle")
            auto("click", worktree_ref, "--settle")
            auto("wait", "--text", "Commit summary (required)", "--timeout", "10s", "--settle")
            auto("screenshot", str(artifacts / "worktree-row.png"))
            results["worktree_row_shows_commit_editor"] = True

            stash_ref = stash[0][0]
            auto("focus", stash_ref, "--settle")
            auto("click", stash_ref, "--settle")
            auto("wait", "--text", "Pop (apply and drop)", "--timeout", "10s", "--settle")
            auto("screenshot", str(artifacts / "stash-row.png"))
            found = entries()
            selected_stash = [e for e in found if e[2] == stash[0][2] and "offscreen" not in e[3]]
            assert selected_stash and any("checked" in e[3] or "selected" in e[3] for e in selected_stash), "the stash row was not selected"
            results["stash_row_opens_details"] = True

            # 4. Sidebar refs locate and select; they do not re-scope the list.
            def wait_for_all_commits():
                # A repository refresh paints the first 200 rows before the
                # pinned traversal delivers the remainder of this fixture.
                auto("wait", "--text", "243 commits", "--timeout", "10s", "--settle")

            def sidebar_row(name, role="button"):
                for e in entries():
                    if e[1] != role or name not in e[2] or "offscreen" in e[3]:
                        continue
                    match = re.search(r"bounds=([\d.]+),", e[3])
                    if match and float(match.group(1)) < 320:
                        return e
                return None

            def history_row(name):
                return [e for e in entries() if e[1] == "listItem" and name in e[2] and "offscreen" not in e[3]]

            def wait_selected(name):
                deadline = time.monotonic() + 5
                while time.monotonic() < deadline:
                    rows = history_row(name)
                    if rows and any("selected" in e[3] or "checked" in e[3] for e in rows):
                        return rows
                    time.sleep(0.1)
                return history_row(name)

            def all_refs_shown():
                return any(e[2] == "All refs" and "offscreen" not in e[3] for e in entries())

            def locate_commit(ref, subject, sha, screenshot):
                control = sidebar_row(ref)
                assert control, f"missing sidebar ref {ref}"
                auto("click", control[0], "--settle")
                auto("wait", "--text", sha, "--timeout", "10s", "--settle")
                rows = wait_selected(subject)
                assert any(re.search(rf'\bkey="?{sha}"?(?:\s|$)', r[3]) for r in rows if "selected" in r[3] or "checked" in r[3]), f"{ref} did not reveal {sha}"
                assert visible(entries(), sha), f"Summary did not show {sha}"
                wait_for_all_commits()
                assert all_refs_shown(), f"{ref} changed the History scope"
                auto("screenshot", str(artifacts / screenshot))
                results.setdefault("located_commits", {})[ref] = sha

            bounds = re.search(r"bounds=([\d.]+),([\d.]+),([\d.]+)x([\d.]+)", worktree[0][3])
            x, y, w, h = map(float, bounds.groups())
            auto("wheel", str(int(x+w/2)), str(int(y+h/2)), "0", "1000", "--precise", "--settle")
            assert not history_row("1 change") and not history_row("Fixture stash") and not history_row("Main tip")
            assert any("History row " in row[2] and "offscreen" not in row[3] for row in entries())
            auto("screenshot", str(artifacts / "history-scrolled.png"))
            results["changes_scrolls_with_commits"] = True

            wait_for_all_commits()
            assert all_refs_shown(), "the header does not start on All refs"
            locate_commit("future", "Future side tip", future_sha, "sidebar-branch-locate.png")
            remote_heading = sidebar_row("Remotes", role="disclosure")
            assert remote_heading, "missing Remotes heading"
            auto("click", remote_heading[0], "--settle")
            locate_commit("origin/checkpoint", "History row 119", midpoint, "sidebar-remote-locate.png")

            # Scope the list to main, then locate a commit reachable only
            # through future. Both the selection and Summary follow the click.
            header = next(e for e in visible(entries(), "All refs") if e[1] == "button")
            auto("click", header[0], "--settle")
            main_choice = next(e for e in visible(entries(), "main") if e[1] == "listItem")
            auto("click", main_choice[0], "--settle")
            auto("wait", "--text", "242 commits", "--timeout", "10s", "--settle")
            auto("wait", "--text", head, "--timeout", "10s", "--settle")
            locate_commit("future", "Future side tip", future_sha, "sidebar-cross-scope.png")
            results["located_outside_main_scope"] = True

            tags_heading = sidebar_row("Tags", role="disclosure")
            assert tags_heading, "the sidebar has no Tags group"
            auto("click", tags_heading[0], "--settle")
            locate_commit("root-tag", "Root commit", root_sha, "sidebar-tag-locate.png")
            results["sidebar_tag_located"] = True

            stashes_heading = sidebar_row("Stashes", role="disclosure")
            assert stashes_heading, "the sidebar has no Stashes group"
            auto("click", stashes_heading[0], "--settle")
            stash_sidebar = sidebar_row("Fixture stash")
            assert stash_sidebar, "the stash is not listed in the sidebar"
            auto("focus", stash_sidebar[0], "--settle")
            auto("click", stash_sidebar[0], "--settle")
            stash_rows = wait_selected("Fixture stash")
            assert stash_rows and any("selected" in e[3] or "checked" in e[3] for e in stash_rows), f"the sidebar stash click did not select its row: {stash_rows}"
            wait_for_all_commits()
            auto("screenshot", str(artifacts / "sidebar-stash-locate.png"))
            results["sidebar_stash_located"] = True

            time.sleep(0.5)
            assert fingerprint() == before, "browsing modified Git data"
            results["git_unchanged"] = True
            results["passed"] = True
        except Exception:
            results["passed"] = False
            try:
                auto("screenshot", str(artifacts / "failure.png"))
                (artifacts / "failure.snapshot.txt").write_text(auto("snapshot", "--full"))
            except Exception:
                pass
            raise
        finally:
            if calls.exists():
                log_lines = calls.read_text().splitlines()
                results["status_calls_while_open"] = len([line for line in log_lines if " status --porcelain" in line])
                (artifacts / "git-calls.txt").write_text(calls.read_text())
            (artifacts / "results.json").write_text(json.dumps(results, indent=2))
            try:
                auto("close")
            except Exception:
                pass
            app.terminate()
            try:
                app.wait(timeout=10)
            except subprocess.TimeoutExpired:
                app.kill()
                app.wait(timeout=5)
print(json.dumps(results, indent=2), flush=True)
