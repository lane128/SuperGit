#!/usr/bin/env python3
"""Measure long-patch navigation in repositories owned by this script."""
import argparse
import hashlib
import json
import os
import pathlib
import re
import runpy
import subprocess
import tempfile
import time

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--worktree", action="store_true", help="Browse an unstaged 8,000-line patch in both Commit overview and file tab")
parser.add_argument("--multi-hunk", action="store_true", help="Change nine separated lines across a 4,000-line file instead of replacing every line")
parser.add_argument("--wide-lines", action="store_true", help="Add wide ASCII text and probe horizontal tails in overview and file tabs")
parser.add_argument("--unicode-lines", action="store_true", help="Use wide mixed CJK and fullwidth text for horizontal tail reachability")
parser.add_argument("--font-scale", type=float, choices=(1, 1.15), default=1, help="Isolated fixture text scale")
parser.add_argument("--stage-line", action="store_true", help="After virtualized worktree navigation, stage the first deletion and verify exact Index bytes")
parser.add_argument("--stage-unstage", action="store_true", help="Also unstage the same selection through its exact Staged file tab and verify the original Index")
parser.add_argument("--selected-lines", type=int, choices=(1, 40), default=1, help="Number of consecutive changed lines to stage; 40 crosses the virtualized viewport")
parser.add_argument("--line-kind", choices=("del", "add"), default="del", help="Stage first deletions (default) or last additions in the full-replacement fixture")
parser.add_argument("--discard-undo", action="store_true", help="Discard one changed line through confirmation, then undo and verify exact fixture restoration")
parser.add_argument("--selection-bar", action="store_true", help="Use the visible Changes selection bar for stage/unstage instead of context menus")
options = parser.parse_args()
if options.selection_bar:
    options.stage_unstage = True
if options.line_kind == "add" and options.multi_hunk:
    parser.error("Addition staging requires the full-replacement fixture")
if options.discard_undo:
    if options.stage_line or options.stage_unstage or options.selected_lines != 1 or options.multi_hunk:
        parser.error("Discard/undo is a separate single-line full-replacement probe")
    options.worktree = True
elif options.stage_unstage or options.selected_lines > 1 or options.line_kind == "add":
    options.stage_line = True
if options.selected_lines > 1 and options.multi_hunk:
    parser.error("Multi-line selection requires consecutive changes, not the separated multi-hunk fixture")
if options.stage_line:
    options.worktree = True
if options.unicode_lines:
    options.wide_lines = True

project = pathlib.Path(__file__).resolve().parents[1]
NativeMemorySampler = runpy.run_path(str(project / "scripts/native-memory-sampler.py"))["NativeMemorySampler"]
build = project / "build"
automation = build / "mygo-automation"
candidates = list(build.glob("supergit-lobe*e2e"))
if not automation.is_file() or len(candidates) != 1:
    raise SystemExit("Run make lobe-e2e-build first")
artifacts = pathlib.Path(tempfile.mkdtemp(prefix="lobe-long-patch-", dir=build))
env_git = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
env_git.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")
changed_positions = {0, *range(499, 4000, 500)} if options.multi_hunk else set(range(4000))
results = {"lines_per_version": 4000, "expected_changed_lines": 2 * len(changed_positions), "worktree": options.worktree, "multi_hunk": options.multi_hunk, "wide_lines": options.wide_lines, "unicode_lines": options.unicode_lines, "font_scale": options.font_scale, "stage_line": options.stage_line, "stage_unstage": options.stage_unstage, "selected_lines": options.selected_lines, "line_kind": options.line_kind, "discard_undo": options.discard_undo, "selection_bar": options.selection_bar, "runs": []}


def fixture_content(version):
    rows = []
    for index in range(4000):
        text = f"{version if index in changed_positions else 'stable'} long patch line {index:04d}"
        if options.wide_lines:
            segment = " 中文宽行ＡＢＣ" if options.unicode_lines else " wide-segment"
            text += segment * 16 + f" wide-tail-{index:04d}"
        rows.append(text + "\n")
    return "".join(rows)
print(f"Artifacts: {artifacts}", flush=True)

with tempfile.TemporaryDirectory(prefix="supergit-long-patch.") as directory:
    root = pathlib.Path(directory)
    repo = root / "workspace"

    def git(*args):
        return subprocess.check_output(["git", "-C", str(repo), *args], env=env_git)

    subprocess.run(["git", "init", "-b", "main", str(repo)], env=env_git, check=True, capture_output=True)
    git("config", "user.name", "Fixture")
    git("config", "user.email", "fixture@example.invalid")
    target = repo / "long.txt"
    for version in ("before", "after"):
        target.write_text(fixture_content(version))
        git("add", "--", "long.txt")
        git("-c", "core.hooksPath=/dev/null", "commit", "--no-gpg-sign", "-m", f"Long patch {version}")

    def fingerprint():
        return {"head": git("rev-parse", "HEAD").decode().strip(),
                "index": hashlib.sha256(git("ls-files", "--stage", "-z")).hexdigest(),
                "refs": hashlib.sha256(git("for-each-ref", "--format=%(refname) %(objectname)")).hexdigest(),
                "status": git("status", "--porcelain", "-z").decode(),
                "bytes": hashlib.sha256(target.read_bytes()).hexdigest()}

    if options.worktree:
        target.write_text(fixture_content("working"))
    comparison = () if options.worktree else ("HEAD^", "HEAD")
    added, deleted, path = git("diff", "--numstat", *comparison, "--", "long.txt").decode().strip().split("\t")
    assert path == "long.txt" and int(added) == int(deleted) == len(changed_positions)
    patch = git("diff", "--no-ext-diff", "--unified=3", *comparison, "--", "long.txt").decode()
    results["hunks"] = sum(line.startswith("@@ ") for line in patch.splitlines())
    assert results["hunks"] == (9 if options.multi_hunk else 1)
    deleted_prefix = "after" if options.worktree else "before"
    added_prefix = "working" if options.worktree else "after"
    before = fingerprint()
    assert before["status"] == (" M long.txt\0" if options.worktree else "")
    for number in range(1, 4):
        home = root / f"home-{number}"
        settings = home / "Library/Application Support/supergit"
        settings.mkdir(parents=True)
        (settings / "settings.json").write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False, "fontScale": options.font_scale}))
        socket = root / f"app-{number}.sock"
        session = f"supergit-long-patch-{number}"
        env = {**env_git, "HOME": str(home), "MYGO_AUTOMATION": str(socket), "SUPERGIT_WINDOW_SIZE": "1320x640"}
        with (artifacts / f"app-{number}.log").open("w") as log, (artifacts / f"automation-{number}.log").open("w") as trace:
            app = subprocess.Popen([str(candidates[0]), str(repo)], cwd=project, env=env, stdout=log, stderr=subprocess.STDOUT)
            sampler = NativeMemorySampler(app.pid)

            def auto(*args):
                completed = subprocess.run([str(automation), *args, "--session", session], cwd=project, capture_output=True, text=True, timeout=40)
                trace.write(repr(args) + "\n" + completed.stdout + completed.stderr + "\n")
                trace.flush()
                if completed.returncode:
                    raise RuntimeError(f"{args}: {completed.stdout}{completed.stderr}")
                return completed.stdout

            def entries():
                found = []
                for line in auto("snapshot", "--full").splitlines():
                    match = re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)', line)
                    if match:
                        found.append((match[1], match[2], json.loads(match[3]) if match[3] else "", match[4]))
                return found

            def press(role, label, settle=True):
                item = next(e for e in entries() if e[1] == role and e[2] == label and "disabled" not in e[3])
                auto("press", item[0], *(("--settle",) if settle else ()))

            def wait(text):
                auto("wait", "--text", text, "--timeout", "30s", "--settle")

            def select_patch_line(label):
                # A conservative startup reload can leave old text accessible
                # while navigation is disabled and its selection is discarded.
                # This probe measures navigation after reads, not selection
                # preservation across an automatic reload.
                deadline = time.monotonic() + 30
                while True:
                    current = entries()
                    ready = any(e[1] == "button" and e[2] == "Refresh" and "disabled" not in e[3] for e in current)
                    reading_patch = any(e[1] == "text" and e[2] == "Loading diff…" for e in current)
                    targets = [e for e in current if e[1] == "button" and "disabled" not in e[3] and (e[2] == label if label is not None else re.fullmatch(r"\d+ (?:Added|Deleted) line", e[2]))]
                    if ready and not reading_patch and targets:
                        target = targets[0]
                        auto("press", target[0], "--settle")
                        # Accessible Press does not guarantee keyboard focus.
                        current = next((e for e in entries() if e[1] == "button" and e[2] == target[2]), None)
                        if current is not None:
                            auto("focus", current[0], "--settle")
                            after = entries()
                            if any(e[1] == "button" and e[2] == "Refresh" and "disabled" not in e[3] for e in after) and not any(e[1] == "text" and e[2] == "Loading diff…" for e in after):
                                return
                    if time.monotonic() > deadline:
                        raise AssertionError(f"Patch navigation did not become ready for {label}")
                    time.sleep(0.05)

            def restore_patch_endpoint():
                # The initial watch probe can reload unchanged dirty content
                # after a tail screenshot. Re-establish the endpoint from the
                # current patch, rather than waiting forever for the old view.
                deadline = time.monotonic() + 30
                while True:
                    current = entries()
                    gutters = [e for e in current if e[1] == "button" and re.fullmatch(r"\d+ (?:Added|Deleted) line", e[2])]
                    if gutters:
                        select_patch_line(None)
                        auto("key", "end", "--settle")
                        if any(e[1] == "button" and e[2] == "4000 Added line" and "offscreen" not in e[3] for e in entries()):
                            return
                    if time.monotonic() > deadline:
                        raise AssertionError("Current patch endpoint could not be restored after horizontal scrolling")
                    time.sleep(0.05)

            def probe_wide_tail(measurement, projection):
                tail = next(e for e in entries() if e[1] == "insertion" and e[2].endswith("wide-tail-3999"))
                bounds = re.search(r"bounds=([\d.-]+),([\d.-]+),([\d.-]+)x([\d.-]+)", tail[3])
                assert bounds, "Wide line has no text bounds"
                x, y, text_width, text_height = map(float, bounds.groups())
                assert text_width > 500, "Wide fixture did not produce wide text"
                tick = time.monotonic()
                auto("wheel", str(x + 10), str(y + 2), "20000", "0", "--precise", "--settle")
                # Local-watch refresh can replace a patch and reset its scroll.
                for retry in range(4):
                    moved = next((e for e in entries() if e[1] == "insertion" and e[2].endswith("wide-tail-3999")), None)
                    if moved is None:
                        restore_patch_endpoint()
                        auto("wheel", "1150", "520", "20000", "0", "--precise", "--settle")
                        continue
                    moved_bounds = re.search(r"bounds=([\d.-]+),([\d.-]+),([\d.-]+)x([\d.-]+)", moved[3])
                    assert moved_bounds, "Scrolled wide line has no bounds"
                    left, top, width, height = map(float, moved_bounds.groups())
                    if left < x - 100:
                        break
                    auto("wheel", "1150", str(top + 2), "20000", "0", "--precise", "--settle")
                else:
                    raise AssertionError(f"{projection}: wide patch did not stabilize after refresh")
                # The owned window is 1320 DIP wide; require the text's right
                # edge inside the detail viewport, rather than text existence.
                assert left < x - 100 and x - 92 < left + width <= 1320, f"{projection}: horizontal scroll did not expose the wide line tail"
                measurement[f"wide_{projection}_scroll_to_tail_seconds"] = time.monotonic() - tick
                measurement[f"wide_{projection}_tail_visible"] = True
                auto("screenshot", str(artifacts / f"wide-{projection}-tail-{number}.png"))
                auto("wheel", "1150", str(top + 2), "-20000", "0", "--precise", "--settle")
                restore_patch_endpoint()

            try:
                deadline = time.monotonic() + 30
                while not socket.exists():
                    if app.poll() is not None or time.monotonic() > deadline:
                        raise AssertionError("Native app did not open its automation socket")
                    time.sleep(0.05)
                auto("open", "--socket", str(socket))
                auto("wait", "--text", "Commit summary (required)", "--timeout", "30s", "--settle")
                auto("key", "cmd+1", "--settle")
                wait("Long patch after")
                # Start from the small parent, then measure selecting the 8,000-line patch.
                press("listItem", "Long patch before")
                wait("before long patch line 0000")
                sampler.phase("patch_selection")
                tick = time.monotonic()
                press("listItem", "1 change" if options.worktree else "Long patch after")
                wait(f"{deleted_prefix} long patch line 0000")
                wait("1 Deleted line")
                measurement = {"run": number, "selection_to_first_patch_seconds": time.monotonic() - tick}
                current = entries()
                changed = [e for e in current if e[1] in ("insertion", "deletion")]
                assert 0 < len(changed) < 200, "Long patch constructed all changed rows"
                measurement["first_patch_changed_controls"] = len(changed)
                (artifacts / f"first-{number}.snapshot.txt").write_text(auto("snapshot", "--full"))
                select_patch_line("1 Deleted line")
                sampler.phase("patch_navigation")
                tick = time.monotonic()
                auto("key", "end", "--settle")
                wait(f"{added_prefix} long patch line 3999")
                measurement["end_to_last_line_seconds"] = time.monotonic() - tick
                last = entries()
                assert any(e[1] == "button" and e[2] == "4000 Added line" and "offscreen" not in e[3] for e in last), "Last gutter not in viewport"
                assert len([e for e in last if e[1] in ("insertion", "deletion")]) < 200
                auto("screenshot", str(artifacts / f"last-{number}.png"))
                if options.wide_lines:
                    probe_wide_tail(measurement, "worktree_overview" if options.worktree else "history_summary")
                tick = time.monotonic()
                auto("key", "home", "--settle")
                wait(f"{deleted_prefix} long patch line 0000")
                measurement["home_to_first_line_seconds"] = time.monotonic() - tick
                assert any(e[1] == "button" and e[2] == "1 Deleted line" and "offscreen" not in e[3] for e in entries()), "First gutter not in viewport"
                if options.worktree:
                    sampler.phase("file_tab_navigation")
                    tick = time.monotonic()
                    press("listItem", "long.txt")
                    wait("long.txt · Unstaged")
                    wait(f"{deleted_prefix} long patch line 0000")
                    measurement["file_tab_selection_to_patch_seconds"] = time.monotonic() - tick
                    current = entries()
                    assert any(e[1] == "tab" and e[2] == "long.txt · Unstaged" and "checked" in e[3] for e in current), "Working file tab not active"
                    controls = [e for e in current if e[1] in ("insertion", "deletion")]
                    assert 0 < len(controls) < 200, "Working file tab constructed all changed rows"
                    measurement["file_tab_first_patch_changed_controls"] = len(controls)
                    select_patch_line("1 Deleted line")
                    tick = time.monotonic()
                    auto("key", "end", "--settle")
                    wait(f"{added_prefix} long patch line 3999")
                    measurement["file_tab_end_to_last_line_seconds"] = time.monotonic() - tick
                    assert any(e[1] == "button" and e[2] == "4000 Added line" and "offscreen" not in e[3] for e in entries()), "Working file tab last gutter not in viewport"
                    auto("screenshot", str(artifacts / f"file-tab-last-{number}.png"))
                    if options.wide_lines:
                        probe_wide_tail(measurement, "file_tab")
                    auto("key", "home", "--settle")
                    wait(f"{deleted_prefix} long patch line 0000")
                    assert any(e[1] == "button" and e[2] == "1 Deleted line" and "offscreen" not in e[3] for e in entries()), "Working file tab first gutter not in viewport"
                elif options.wide_lines:
                    press("listItem", "long.txt")
                    wait(f"{deleted_prefix} long patch line 0000")
                    assert any(e[1] == "tab" and e[2] == "long.txt" and "checked" in e[3] for e in entries()), "Historical file tab not active"
                    select_patch_line("1 Deleted line")
                    auto("key", "end", "--settle")
                    wait(f"{added_prefix} long patch line 3999")
                    assert any(e[1] == "button" and e[2] == "4000 Added line" and "offscreen" not in e[3] for e in entries()), "Historical file tab last gutter not in viewport"
                    probe_wide_tail(measurement, "history_file_tab")
                    auto("key", "home", "--settle")
                    wait(f"{deleted_prefix} long patch line 0000")
                    assert any(e[1] == "button" and e[2] == "1 Deleted line" and "offscreen" not in e[3] for e in entries()), "Historical file tab first gutter not in viewport"
                assert fingerprint() == before, "Browsing changed fixture state"
                if options.discard_undo:
                    adding = options.line_kind == "add"
                    press("button", "1 Deleted line")
                    if adding:
                        auto("key", "end", "--settle")
                        wait("working long patch line 3999")
                    handle_label = "4000 Added line" if adding else "1 Deleted line"
                    press("button", handle_label)
                    handle = next(e for e in entries() if e[1] == "button" and e[2] == handle_label)
                    auto("focus", handle[0], "--settle")
                    auto("key", "Shift+F10", "--settle")
                    wait("Discard changes to this line…")
                    press("menuItem", "Discard changes to this line…")
                    wait("Discard line changes")
                    assert fingerprint() == before, "Discard changed fixture before confirmation"
                    # The modal confirmation follows the underlying file actions
                    # in the tree and is the last enabled Discard button.
                    confirm = [e for e in entries() if e[1] == "button" and e[2] == "Discard" and "disabled" not in e[3]][-1]
                    auto("press", confirm[0])
                    wait("Selected line changes discarded")
                    working = fixture_content("working").encode()
                    if adding:
                        expected_working = b"".join(working.splitlines(keepends=True)[:-1])
                    else:
                        # Replacement pairs retain their positions: the reverse
                        # patch keeps working row 1, then restores original row 1.
                        working_rows = working.splitlines(keepends=True)
                        restored = fixture_content("after").encode().splitlines(keepends=True)[0]
                        expected_working = working_rows[0] + restored + b"".join(working_rows[1:])
                    assert target.read_bytes() == expected_working, "Discard did not change only the selected line"
                    discarded = fingerprint()
                    for protected in ("head", "index", "refs", "status"):
                        assert discarded[protected] == before[protected], f"Discard changed protected {protected}"
                    measurement["partial_discard_exact_working_bytes"] = True
                    measurement["partial_discard_index_preserved"] = True
                    (artifacts / f"partial-discard-{number}.snapshot.txt").write_text(auto("snapshot", "--full"))
                    auto("screenshot", str(artifacts / f"partial-discard-{number}.png"))
                    press("button", "Undo", settle=False)
                    wait("Discard undone")
                    assert fingerprint() == before, "Native undo did not restore exact original fixture state"
                    measurement["partial_discard_undo_exact_fingerprint"] = True
                    (artifacts / f"partial-discard-undo-{number}.snapshot.txt").write_text(auto("snapshot", "--full"))
                    auto("screenshot", str(artifacts / f"partial-discard-undo-{number}.png"))
                if options.stage_line:
                    if options.selection_bar:
                        press("tab", "Changes")
                    # Select a newly acquired gutter after End/Home and any
                    # conservative watcher reload; never reuse an old row ref.
                    adding = options.line_kind == "add"
                    if adding:
                        # Home may be followed by a watcher reload that clears
                        # selection; establish the diff target before using End.
                        press("button", "1 Deleted line")
                        auto("key", "end", "--settle")
                        wait("working long patch line 3999")
                    stage_gutter = "4000 Added line" if adding else "1 Deleted line"
                    stage_endpoint = f"{4001 - options.selected_lines} Added line" if adding else f"{options.selected_lines} Deleted line"
                    press("button", stage_gutter)
                    gutter = next(e for e in entries() if e[1] == "button" and e[2] == stage_gutter)
                    auto("focus", gutter[0], "--settle")
                    for _ in range(options.selected_lines - 1):
                        auto("key", "Shift+Up" if adding else "Shift+Down", "--settle")
                    if options.selected_lines > 1:
                        endpoint = next(e for e in entries() if e[1] == "button" and e[2] == stage_endpoint)
                        assert "offscreen" not in endpoint[3], "Range selection did not reveal its offscreen endpoint"
                        auto("focus", endpoint[0], "--settle")
                    if options.selection_bar:
                        wait(f"{options.selected_lines} selected ·")
                        auto("screenshot", str(artifacts / f"selection-bar-stage-{number}.png"))
                        press("button", "Stage selection")
                    else:
                        auto("key", "Shift+F10", "--settle")
                        stage_label = "Stage line" if options.selected_lines == 1 else f"Stage {options.selected_lines} selected lines"
                        wait(stage_label)
                        press("menuItem", stage_label)
                    wait("Line staged" if options.selected_lines == 1 else "Selected lines staged")
                    original_index = fixture_content("after").encode()
                    if adding:
                        # Replacement pairs retain their positions. Each selected
                        # addition follows its corresponding original context row.
                        original_rows = original_index.splitlines(keepends=True)
                        working_rows = fixture_content("working").encode().splitlines(keepends=True)
                        expected_index = b"".join(
                            row + (working_rows[index] if index >= 4000 - options.selected_lines else b"")
                            for index, row in enumerate(original_rows)
                        )
                    else:
                        expected_index = b"".join(original_index.splitlines(keepends=True)[options.selected_lines:])
                    assert git("show", ":long.txt") == expected_index, "Partial stage did not match the exact selected lines"
                    assert target.read_bytes() == fixture_content("working").encode(), "Partial stage changed working bytes"
                    current_state = fingerprint()
                    assert current_state["head"] == before["head"] and current_state["refs"] == before["refs"], "Partial stage moved HEAD or refs"
                    assert current_state["status"] == "MM long.txt\0", "Partial stage changed unexpected paths"
                    measurement["partial_stage_exact_index"] = True
                    measurement["partial_stage_working_bytes_preserved"] = True
                    (artifacts / f"partial-stage-{number}.snapshot.txt").write_text(auto("snapshot", "--full"))
                    auto("screenshot", str(artifacts / f"partial-stage-{number}.png"))
                    if options.stage_unstage:
                        file_rows = [e for e in entries() if e[1] == "listItem" and e[2] == "long.txt"]
                        assert len(file_rows) == 2, "Expected separate unstaged and staged file rows"
                        auto("press", file_rows[-1][0], "--settle")
                        wait("long.txt · Staged")
                        assert any(e[1] == "tab" and e[2] == "long.txt · Staged" and "checked" in e[3] for e in entries()), "Exact staged file tab not active"
                        staged_patch = git("diff", "--cached", "--no-ext-diff", "--unified=3", "--", "long.txt").decode()
                        staged_header = next(line for line in staged_patch.splitlines() if line.startswith("@@ "))
                        wait(staged_header)
                        unstage_gutter = f"{4002 - options.selected_lines} Added line" if adding else "1 Deleted line"
                        press("button", unstage_gutter)
                        gutter = next(e for e in entries() if e[1] == "button" and e[2] == unstage_gutter)
                        auto("focus", gutter[0], "--settle")
                        for _ in range(options.selected_lines - 1):
                            auto("key", "Shift+Down", "--settle")
                        if options.selected_lines > 1:
                            unstage_endpoint = f"{4000 + options.selected_lines} Added line" if adding else f"{options.selected_lines} Deleted line"
                            endpoint = next(e for e in entries() if e[1] == "button" and e[2] == unstage_endpoint)
                            if "offscreen" in endpoint[3]:
                                auto("key", "Shift+F10", "--settle")
                                menu_labels = [e[2] for e in entries() if e[1] == "menuItem"]
                                raise AssertionError(f"Staged range endpoint not revealed; menu selection: {menu_labels}")
                            auto("focus", endpoint[0], "--settle")
                        if options.selection_bar:
                            wait(f"{options.selected_lines} selected ·")
                            auto("screenshot", str(artifacts / f"selection-bar-unstage-{number}.png"))
                            press("button", "Unstage selection", settle=False)
                        else:
                            auto("key", "Shift+F10", "--settle")
                            unstage_label = "Unstage line" if options.selected_lines == 1 else f"Unstage {options.selected_lines} selected lines"
                            wait(unstage_label)
                            # Observe the notice before scroll/focus settling.
                            press("menuItem", unstage_label, settle=False)
                        wait("Line unstaged" if options.selected_lines == 1 else "Selected lines unstaged")
                        assert fingerprint() == before, "Native partial unstage did not restore exact original fixture state"
                        measurement["partial_unstage_exact_fingerprint"] = True
                        (artifacts / f"partial-unstage-{number}.snapshot.txt").write_text(auto("snapshot", "--full"))
                        auto("screenshot", str(artifacts / f"partial-unstage-{number}.png"))
                sampler.close()
                measurement.update(sampler.result())
                assert measurement["rss_samples"] > 0 and measurement["app_peak_sampled_rss_kib"] > 0, "No application RSS observed"
                results["runs"].append(measurement)
                (artifacts / "results.json").write_text(json.dumps(results, indent=2) + "\n")
                print("PASS " + json.dumps(measurement), flush=True)
            except BaseException:
                (artifacts / f"failure-{number}.memory.json").write_text(json.dumps(sampler.result(), indent=2) + "\n")
                (artifacts / f"failure-{number}.git.json").write_text(json.dumps({"before": before, "actual": fingerprint()}, indent=2) + "\n")
                try:
                    (artifacts / f"failure-{number}.snapshot.txt").write_text(auto("snapshot", "--full"))
                    auto("screenshot", str(artifacts / f"failure-{number}.png"))
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
                sampler.close()
                if options.stage_line:
                    # Only this script's generated fixture, after its app exits.
                    # Reset the test mutation so the next fresh process sees
                    # exactly the same unstaged patch and final fingerprint.
                    git("read-tree", "HEAD")
    assert fingerprint() == before
    results["state_preserved"] = True
    (artifacts / "results.json").write_text(json.dumps(results, indent=2) + "\n")
