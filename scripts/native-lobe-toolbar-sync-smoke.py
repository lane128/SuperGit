#!/usr/bin/env python3
"""Accept the macOS toolbar and Sync menu against an owned bare remote."""
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import time

project = Path(__file__).resolve().parents[1]
build = Path(os.environ.get("SUPERGIT_NATIVE_BUILD_DIR", project / "build"))
binary = Path(os.environ.get("SUPERGIT_NATIVE_BINARY", build / "supergit-lobe-e2e"))
driver = build / "mygo-automation"
if not binary.is_file() or not driver.is_file():
    raise SystemExit("Run make lobe-e2e-build first")

artifacts = Path(tempfile.mkdtemp(prefix="toolbar-sync-", dir=build)).resolve()
branch = "feature/überlange-virtual-live-preview-performance-navigation-ios-643"
tracking = f"refs/remotes/origin/{branch}"
destination = f"refs/heads/{branch}"
repo, bare, peer = [artifacts / name for name in ("checkout", "origin.git", "peer")]
env = {key: value for key, value in os.environ.items() if not key.startswith(("GIT_", "MYGO_", "SUPERGIT_"))}
env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_ALLOW_PROTOCOL="file",
           GIT_TERMINAL_PROMPT="0", GIT_OPTIONAL_LOCKS="0",
           GIT_AUTHOR_NAME="Fixture", GIT_AUTHOR_EMAIL="fixture@example.invalid",
           GIT_COMMITTER_NAME="Fixture", GIT_COMMITTER_EMAIL="fixture@example.invalid")
window_size = os.environ.get("SUPERGIT_WINDOW_SIZE", "900x640")
window_width = int(window_size.split("x")[0])
size_label = "narrow" if window_width == 900 else "wide"
profile = artifacts / "profile"
settings = profile / "supergit" / "settings.json"
settings.parent.mkdir(parents=True)
settings.write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False, "fontScale": 1.15}))


def git(*args, directory=repo):
    return subprocess.check_output(["git", "-C", str(directory), *args], env=env, text=True, timeout=15).strip()


def control(snapshot, role, label):
    pattern = re.compile(r'\s*@e\d+ \[' + role + r'\] ' +
                         re.escape(json.dumps(label, ensure_ascii=False)) + r'(?= |$)')
    return [line for line in snapshot.splitlines() if pattern.match(line)]


def bounds(row):
    match = re.search(r'\bbounds=([-\d.]+),([-\d.]+),([\d.]+)x([\d.]+)', row)
    assert match, row
    return tuple(map(float, match.groups()))


print(f"Toolbar and Sync evidence: {artifacts}", flush=True)
git("init", "--bare", "-b", branch, str(bare), directory=artifacts)
git("init", "-b", branch, str(repo), directory=artifacts)
(repo / "work.txt").write_text("base\n")
git("add", "work.txt")
git("commit", "--no-gpg-sign", "-m", "Fixture base")
git("remote", "add", "origin", str(bare))
git("push", "--set-upstream", "origin", branch)
base = git("rev-parse", "HEAD")
tree = git("rev-parse", "HEAD^{tree}")
head = base
for index in range(102):
    head = git("commit-tree", tree, "-p", head, "-m", f"Local commit {index + 1}")
git("update-ref", destination, head, base)
git("branch", "main", head)
git("branch", "trial-5", head)
assert git("rev-list", "--left-right", "--count", "HEAD...@{upstream}") == "102\t0"
assert Path(git("remote", "get-url", "--push", "origin")).resolve() == bare

# Keep both destinations local and independent so the option scope is observable.
other = artifacts / "upstream.git"
git("clone", "--bare", str(bare), str(other), directory=artifacts)
git("remote", "add", "upstream", str(other))
for name in ("origin", "upstream"):
    git("config", "remote." + name + ".tagOpt", "--no-tags")

with tempfile.TemporaryDirectory(prefix="sg-sync-", dir="/tmp") as directory:
    scratch = Path(directory).resolve()
    socket = scratch / "app.sock"
    app_env = {**env, "MYGO_AUTOMATION": str(socket), "MYGO_AUTOMATION_DIR": str(scratch / "driver"),
               "SUPERGIT_TEST_CONFIG_DIR": str(profile), "SUPERGIT_WINDOW_SIZE": window_size}
    with (artifacts / "app.log").open("w") as log, (artifacts / "actions.log").open("w") as trace:
        app = subprocess.Popen([str(binary), str(repo)], env=app_env, stdout=log, stderr=subprocess.STDOUT)

        def until(check, description):
            deadline = time.monotonic() + 22
            while app.poll() is None and time.monotonic() < deadline:
                result = check()
                if result:
                    return result
                time.sleep(.05)
            raise AssertionError(description)

        def auto(*args):
            result = subprocess.run([str(driver), *map(str, args), "--session", scratch.name],
                                    env=app_env, capture_output=True, text=True, timeout=25)
            print(args, result.stdout, result.stderr, sep="\n", file=trace, flush=True)
            result.check_returncode()
            return result.stdout

        def snapshot():
            return auto("snapshot", "--full")

        def press(role, label):
            def enabled():
                matches = [row for row in control(snapshot(), role, label)
                           if " disabled" not in row and " offscreen" not in row]
                return matches[0] if len(matches) == 1 else None

            x, y, w, h = bounds(until(enabled, f"No enabled {role} {label}"))
            auto("click", x + w / 2, y + h / 2, "--settle")

        def sync(action):
            press("menuButton", "Sync")
            press("menuItem", action)

        def branch_button(status, name=branch):
            caption = f"{name} ({status})" if status else name
            description = "description=" + json.dumps(caption, ensure_ascii=False)

            def current():
                return next((item for item in control(snapshot(), "button", "Branches") if description in item), None)

            return until(current, f"Missing branch caption {caption!r}")

        try:
            until(socket.exists, "No native automation socket")
            auto("open", "--socket", socket)
            button = branch_button("102 ahead · 0 behind")
            x, y, w, h = bounds(button)
            layout = snapshot()
            commands, = control(layout, "button", "Command palette")
            left, top, icon_width, icon_height = bounds(commands)
            assert abs(x + w - left - 360) <= 1 and abs(left + 180 - window_width / 2) <= 1, (commands, button)
            assert abs(left + icon_width - x) <= 1 and (top, icon_height) == (y, h), (commands, button)
            assert 38 <= y < y + h < 80, button
            for label in ("Fetch", "Pull (fast-forward only)", "Push", "Refresh"):
                assert not control(layout, "button", label), label
            (artifacts / f"{size_label}-toolbar.txt").write_text(layout)
            auto("screenshot", artifacts / f"{size_label}-long-branch.png")

            press("button", "Command palette")
            catalog = snapshot()
            expected = (project / "internal/lobedesk/testdata/sublime-merge-2125-palette.txt").read_text().splitlines()
            missing = [label for label in [*expected, "About"] if len(control(catalog, "listItem", label)) != 1]
            assert len(expected) == 75 and not missing, missing
            (artifacts / "command-catalog.txt").write_text(catalog)
            auto("screenshot", artifacts / "command-catalog.png")
            auto("key", "Escape", "--settle")
            press("button", "Command palette")
            auto("type", "crbr", "--settle")
            assert control(snapshot(), "listItem", "Create Branch…")
            auto("key", "Escape", "--settle")
            press("button", "Branches")
            assert control(snapshot(), "textField", "Filter branches")
            auto("key", "Backspace", "--settle")
            assert control(snapshot(), "textField", "Search commands…")
            auto("key", "Escape", "--settle")

            # Switch the same window between short and long refs through Checkout.
            for name in ("main", branch):
                press("button", "Branches")
                auto("type", name, "--settle")
                press("listItem", name + " · Local branches")
                until(lambda: git("branch", "--show-current") == name, f"Checkout did not switch to {name}")
                state = "102 ahead · 0 behind" if name == branch else "No upstream"
                current = branch_button(state, name)
                assert bounds(current) == (x, y, w, h), current
                if name == "main":
                    auto("screenshot", artifacts / f"{size_label}-short-branch.png")

            press("menuButton", "Sync")
            opened = snapshot()
            assert control(opened, "menuItem", "102 ahead") and control(opened, "menuItem", "0 behind")
            assert "origin/" + branch in opened
            auto("screenshot", artifacts / "sync-menu.png")
            press("menuItem", "102 ahead")
            press("menuItem", "Next page")
            sha = git("rev-parse", "HEAD~6")
            press("menuItem", sha[:8] + " · Local commit 96")
            until(lambda: any("checked" in row for row in control(snapshot(), "listItem", "Local commit 96")),
                  "Outgoing commit did not open in History")
            press("menuButton", "Sync")
            press("menuItem", "Push branch…")
            press("button", "Cancel")
            assert git("rev-parse", destination, directory=bare) == base
            sync("Push branch…")
            press("button", "Push")
            until(lambda: git("rev-parse", destination, directory=bare) == head, "Push did not publish exact HEAD")
            branch_button("")
            assert git("rev-parse", "HEAD") == head and not git("status", "--porcelain")

            git("clone", "--branch", branch, str(bare), str(peer), directory=artifacts)
            (peer / "peer.txt").write_text("remote fast-forward\n")
            git("add", "peer.txt", directory=peer)
            git("commit", "--no-gpg-sign", "-m", "Peer update", directory=peer)
            new_head = git("rev-parse", "HEAD", directory=peer)
            git("push", "origin", branch, directory=peer)
            index = git("ls-files", "--stage")

            sync("Fetch")
            until(lambda: git("rev-parse", tracking) == new_head, "Fetch did not advance remote-tracking ref")
            branch_button("0 ahead · 1 behind")
            sync("1 behind")
            press("menuItem", new_head[:8] + " · Peer update")
            until(lambda: any("checked" in row for row in control(snapshot(), "listItem", "Peer update")),
                  "Incoming commit did not open in History")
            assert git("rev-parse", "HEAD") == head and git("ls-files", "--stage") == index
            assert (repo / "work.txt").read_text() == "base\n" and not (repo / "peer.txt").exists()

            sync("Pull (fast-forward only)")
            until(lambda: git("rev-parse", "HEAD") == new_head, "Pull did not fast-forward")
            branch_button("")
            assert git("rev-parse", "HEAD^") == head and not git("status", "--porcelain")
            assert (repo / "peer.txt").read_text() == "remote fast-forward\n"
            def choose_command(query, field):
                press("button", "Command palette")
                auto("type", query, "--settle")
                auto("key", "Enter", "--settle")
                until(lambda: control(snapshot(), "textField", field), f"{query}: {field}")

            options = ("fetch", "fetch --prune", "fetch --tags", "fetch --all",
                       "fetch --all --prune", "fetch --all --tags")
            choose_command("repo.fetch", "Fetch Options")
            assert all(control(snapshot(), "listItem", entry) for entry in options)
            auto("screenshot", artifacts / "fetch-options.png")
            press("listItem", "fetch --tags")
            assert control(snapshot(), "textField", "Remote name")
            auto("key", "Backspace", "--settle")
            assert control(snapshot(), "textField", "Fetch Options")
            auto("key", "Backspace", "--settle")
            assert control(snapshot(), "textField", "Search commands…")
            auto("key", "Escape", "--settle")

            def has_ref(ref):
                return subprocess.run(["git", "-C", str(repo), "show-ref", "--verify", "--quiet", ref], env=env).returncode == 0

            for index, option in enumerate(options):
                # Tags point at unreachable commits to rule out Git's automatic tag following.
                touched = ("origin", "upstream") if "--all" in option else ("upstream",)
                for name, server in (("origin", bare), ("upstream", other)):
                    new_ref = f"refs/heads/trial-{index}"
                    git("update-ref", new_ref, base, directory=server)
                    git("update-ref", f"refs/remotes/{name}/obsolete-{index}", base)
                    orphan = git("commit-tree", tree, "-m", f"{name} tag {index}", directory=server)
                    git("tag", f"{name}-tag-{index}", orphan, directory=server)
                choose_command("repo.fetch", "Fetch Options")
                press("listItem", option)
                if "--all" not in option:
                    press("listItem", "upstream")
                for name in touched:
                    until(lambda: has_ref(f"refs/remotes/{name}/trial-{index}"), f"{option} missed {name}")
                for name in ("origin", "upstream"):
                    assert has_ref(f"refs/remotes/{name}/trial-{index}") == (name in touched), (option, name)
                    assert has_ref(f"refs/remotes/{name}/obsolete-{index}") != (name in touched and "--prune" in option), (option, name)
                    assert has_ref(f"refs/tags/{name}-tag-{index}") == (name in touched and "--tags" in option), (option, name)
                assert git("rev-parse", "HEAD") == new_head and not git("status", "--porcelain")

            choose_command("Merge Branch", "Branch name")
            auto("type", "trial-5", "--settle")
            assert control(snapshot(), "text", f"Merge refs/heads/trial-5 into {branch}")
            auto("key", "Down", "--settle")
            assert control(snapshot(), "text", f"Merge refs/remotes/origin/trial-5 into {branch}")
            auto("key", "Down", "--settle")
            assert control(snapshot(), "text", f"Merge refs/remotes/upstream/trial-5 into {branch}")
            auto("screenshot", artifacts / "merge-preview.png")
            auto("key", "Enter", "--settle")
            press("button", "Cancel")
            assert git("rev-parse", "HEAD") == new_head

            # Offer a unique remote commit and confirm the exact SHA through the UI.
            merge_tip = git("commit-tree", git("rev-parse", "HEAD^{tree}", directory=peer), "-p", new_head, "-m", "Palette merge", directory=peer)
            git("update-ref", "refs/heads/merge-target", merge_tip, directory=peer)
            git("push", "origin", "merge-target", directory=peer)
            choose_command("repo.fetch", "Fetch Options")
            press("listItem", "fetch")
            press("listItem", "origin")
            until(lambda: has_ref("refs/remotes/origin/merge-target"), "remote target not fetched")
            choose_command("Merge Branch", "Branch name")
            auto("type", "origin/merge-target", "--settle")
            press("listItem", "origin/merge-target")
            press("button", "Merge into current branch")
            until(lambda: git("rev-parse", "HEAD") == merge_tip, "Merge did not use the previewed commit")
            assert git("branch", "--show-current") == branch and not git("status", "--porcelain")
            choose_command("repo.push", "Push options")
            assert all(control(snapshot(), "listItem", opt) for opt in ("push", "push --set-upstream", "push --force-with-lease"))
            auto("screenshot", artifacts / "push-options.png")
            auto("key", "Escape", "--settle")
            choose_command("remote.add", "Remote name")
            auto("type", "fixture-mirror", "--settle")
            auto("key", "Enter", "--settle")
            assert control(snapshot(), "textField", "Git address")
            auto("key", "Backspace", "--settle")
            previous, = control(snapshot(), "textField", "Remote name")
            assert "fixture-mirror" in previous, previous
            auto("key", "Enter", "--settle")
            auto("type", str(other), "--settle")
            auto("key", "Enter", "--settle")
            until(lambda: "fixture-mirror" in git("remote").splitlines(), "Add Remote did not store the name")
            assert git("remote", "get-url", "fixture-mirror") == str(other)
            choose_command("view.selectTheme", "Select Theme")
            auto("screenshot", artifacts / "select-theme.png")
            auto("key", "Escape", "--settle")
            choose_command("commit.viewTree", "Search this directory")
            press("listItem", "work.txt")
            until(lambda: control(snapshot(), "button", "Back"), "View Tree did not open the file")
            auto("screenshot", artifacts / "tree-file.png")
            press("button", "Back")
            assert control(snapshot(), "listItem", "work.txt")
            auto("key", "Escape", "--settle")

            def run_command(action):
                press("button", "Command palette")
                auto("type", action, "--settle")
                auto("key", "Enter", "--settle")

            (repo / "work.txt").write_text("tracked edit\n")
            (repo / "untracked.txt").write_text("untracked edit\n")
            run_command("repo.reload")
            choose_command("repo.push", "Push options")
            auto("key", "Escape", "--settle")
            run_command("workspace.stageAllModified")
            until(lambda: git("diff", "--cached", "--name-only") == "work.txt", "Stage All Modified staged incorrect files")
            run_command("workspace.stageAllFiles")
            until(lambda: set(git("diff", "--cached", "--name-only").splitlines()) == {"untracked.txt", "work.txt"}, "Stage All missed untracked file")
            run_command("workspace.unstageAllFiles")
            until(lambda: not git("diff", "--cached", "--name-only"), "Unstage All left staged files")
            assert (repo / "work.txt").read_text() == "tracked edit\n"
            choose_command("stash.save", "Stash options")
            press("listItem", "stash push --include-untracked")
            assert control(snapshot(), "textField", "Message (optional)")
            auto("key", "Enter", "--settle")
            until(lambda: git("stash", "list") and not git("status", "--porcelain"), "Stash with an empty message did not save both files")
            if " disabled" not in control(catalog, "listItem", "Git Flow: Initialize")[0]:
                choose_command("init_gitflow", "Production branch")
                press("listItem", "main")
                press("listItem", "trial-5")
                press("button", "Git Flow: Initialize")
                until(lambda: "gitflow.branch.develop=trial-5" in git("config", "--local", "--list"), "Git Flow did not use the selected development branch")
                assert git("config", "gitflow.branch.master") == "main"
                choose_command("edit_gitflow_config", "Choose a setting")
                assert control(snapshot(), "listItem", "Feature branch prefix")
                auto("screenshot", artifacts / "git-flow-configure.png")
                auto("key", "Escape", "--settle")
                print("PASS initialized real Git Flow from the palette, verified both branches, inspected Configure.", flush=True)
            choose_command("lfs_edit_pattern", "Path or pattern")
            press("listItem", "Add pattern…")
            auto("type", "*.nativeasset", "--settle")
            auto("key", "Enter", "--settle")
            until(lambda: git("check-attr", "filter", "--", "file.nativeasset").endswith(": lfs"), "LFS edit did not apply its attribute")
            print("PASS LFS tracking rule via palette and git check-attr.", flush=True)
            print("PASS tree → file → Back, tracked/all staging, unstage and optional-message stash with Git state checks.", flush=True)
            print("PASS all 75 reference command labels and About in the native picker, Push options, Add Remote with Back and Git config readback, theme menu.", flush=True)
            print(f"Evidence: {artifacts}", flush=True)
            print("PASS six fetch variants across two local remotes, scoped tags and prune, Merge preview, cancel and confirmed SHA.", flush=True)
            print(f"PASS 360 pt centered at {window_width} pt and 115% font, Commands and Checkout, short/long refs, Fetch, Pull and confirmed Push.", flush=True)
        finally:
            if app.poll() is None:
                app.terminate()
                app.wait(timeout=10)
