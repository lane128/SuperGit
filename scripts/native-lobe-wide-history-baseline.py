#!/usr/bin/env python3
"""Exercise a wide DAG and interleaved input using only a disposable fixture."""
import argparse
import hashlib
import io
import json
import os
import pathlib
import re
import runpy
import subprocess
import tempfile
import time

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--input-pressure", action="store_true", help="Send consecutive wheel events and edit search while history is still loading")
options = parser.parse_args()

project = pathlib.Path(__file__).resolve().parents[1]
NativeMemorySampler = runpy.run_path(str(project / "scripts/native-memory-sampler.py"))["NativeMemorySampler"]
build = project / "build"
automation = build / "mygo-automation"
candidates = list(build.glob("supergit-lobe*e2e"))
if not automation.is_file() or len(candidates) != 1:
    raise SystemExit("Run make lobe-e2e-build first")
artifacts = pathlib.Path(tempfile.mkdtemp(prefix="lobe-wide-history-", dir=build))
git_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
git_env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")
print(f"Artifacts: {artifacts}", flush=True)
results = {"parallel_branches": 32, "commits_per_branch": 1000, "expected_commits": 32002, "input_pressure": options.input_pressure, "runs": []}

with tempfile.TemporaryDirectory(prefix="supergit-wide-history.") as directory:
    root = pathlib.Path(directory)
    repo = root / "workspace"
    subprocess.run(["git", "init", "--template=", "-b", "main", str(repo)], env=git_env, check=True, capture_output=True)

    def git(*args, **kwargs):
        return subprocess.check_output(["git", "-C", str(repo), *args], env=git_env, **kwargs)

    stream = io.BytesIO()

    def line(text):
        stream.write(text.encode() + b"\n")

    def data(text):
        raw = text.encode()
        line(f"data {len(raw)}")
        stream.write(raw + b"\n")

    def commit(ref, mark, subject, parent=None, merges=()):
        line(f"commit refs/heads/{ref}")
        line(f"mark :{mark}")
        line(f"committer Fixture <fixture@example.invalid> {1800000000 + mark} +0000")
        data(subject)
        if parent:
            line(f"from :{parent}")
        for merge in merges:
            line(f"merge :{merge}")

    commit("main", 1, "Wide root")
    line("M 100644 inline fixture.txt")
    data("wide DAG fixture\n")
    line("")
    tips = []
    mark = 1
    for branch in range(32):
        parent = 1
        for revision in range(1000):
            mark += 1
            commit(f"parallel-{branch:02d}", mark, f"Wide branch {branch:02d} revision {revision:04d}", parent)
            line("")
            parent = mark
        tips.append(parent)
    commit("main", mark + 1, "Wide merge tip", 1, tips)
    line("")
    line("done")
    git("fast-import", "--quiet", "--done", input=stream.getvalue())
    git("reset", "--hard", "HEAD")
    assert int(git("rev-list", "--all", "--count")) == 32002
    assert len(git("show", "-s", "--format=%P", "HEAD").split()) == 33
    assert len(git("branch", "--format=%(refname)").splitlines()) == 33
    fixture_head = git("rev-parse", "HEAD").decode().strip()

    def fingerprint():
        return {"head": git("rev-parse", "HEAD").decode().strip(),
                "refs": hashlib.sha256(git("for-each-ref", "--format=%(refname) %(objectname)")).hexdigest(),
                "index": hashlib.sha256(git("ls-files", "--stage", "-z")).hexdigest(),
                "status": git("status", "--porcelain", "-z").decode(),
                "bytes": hashlib.sha256((repo / "fixture.txt").read_bytes()).hexdigest()}

    before = fingerprint()
    assert not before["status"]
    for number in range(1, 4):
        home = root / f"home-{number}"
        settings = home / "Library/Application Support/supergit"
        settings.mkdir(parents=True)
        (settings / "settings.json").write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False}))
        socket = root / f"app-{number}.sock"
        session = f"supergit-wide-history-{number}"
        env = {**git_env, "HOME": str(home), "MYGO_AUTOMATION": str(socket), "SUPERGIT_WINDOW_SIZE": "1320x640"}
        with (artifacts / f"app-{number}.log").open("w") as log, (artifacts / f"automation-{number}.log").open("w") as trace:
            started = time.monotonic()
            app = subprocess.Popen([str(candidates[0]), str(repo)], cwd=project, env=env, stdout=log, stderr=subprocess.STDOUT)
            sampler = NativeMemorySampler(app.pid)

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

            def press(role, name):
                item = next(e for e in entries() if e[1] == role and e[2] == name and "disabled" not in e[3])
                auto("press", item[0], "--settle")

            def wait(text):
                auto("wait", "--text", text, "--timeout", "80s", "--settle")

            try:
                deadline = time.monotonic() + 30
                while not socket.exists():
                    if app.poll() is not None or time.monotonic() > deadline:
                        raise AssertionError("Native app did not open automation socket")
                    time.sleep(0.05)
                auto("open", "--socket", str(socket))
                auto("wait", "--text", "Commit message", "--timeout", "30s", "--settle")
                auto("key", "cmd+1", "--settle")
                wait("Wide merge tip")
                sampler.phase("history_traversal_and_input")
                measurement = {"run": number, "launch_to_first_history_seconds": time.monotonic() - started}
                initial = entries()
                assert any(e[2] == "Loading complete history…" for e in initial), "Traversal finished before interleaved input; sample is not accepted"
                tip = next(e for e in initial if e[1] == "listItem" and e[2] == "Wide merge tip")
                auto("focus", tip[0], "--settle")
                tick = time.monotonic()
                auto("key", "end", "--settle")
                active = entries()
                measurement["end_during_read_roundtrip_seconds"] = time.monotonic() - tick
                assert any(e[1] == "listItem" and e[2].startswith("Wide branch") and "checked" in e[3] and "offscreen" not in e[3] for e in active), "End failed to select a loaded row during traversal"
                assert any(e[2] == "Loading complete history…" for e in active), "Traversal ended during probe; cannot claim interleaving"
                if options.input_pressure:
                    sampler.phase("wheel_and_search_during_read")
                    visible = [e for e in active if e[1] == "listItem" and e[2].startswith("Wide ") and "offscreen" not in e[3]]
                    anchor = visible[len(visible) // 2]
                    bounds = re.search(r"bounds=([\d.-]+),([\d.-]+),([\d.-]+)x([\d.-]+)", anchor[3])
                    assert bounds, "Visible history row has no bounds"
                    x, y, width, height = map(float, bounds.groups())
                    tick = time.monotonic()
                    for _ in range(12):
                        auto("wheel", str(x + width / 2), str(y + height / 2), "0", "-60", "--precise", "--settle")
                    wheeled = entries()
                    measurement["twelve_wheels_roundtrip_seconds"] = time.monotonic() - tick
                    same_row = next((e for e in wheeled if e[1] == "listItem" and e[2] == anchor[2]), None)
                    moved = same_row is None
                    if same_row is not None:
                        after_bounds = re.search(r"bounds=([\d.-]+),([\d.-]+),([\d.-]+)x([\d.-]+)", same_row[3])
                        moved = after_bounds is not None and abs(float(after_bounds[2]) - y) > 10
                    assert moved, "Consecutive wheel events did not move history rows"
                    assert any(e[2] == "Loading complete history…" for e in wheeled), "Traversal ended before search editing"
                    search = next(e for e in wheeled if e[1] == "textField" and "Search commits" in e[3])
                    auto("focus", search[0], "--settle")
                    query = "Wide branch"
                    tick = time.monotonic()
                    for character in query:
                        auto("type", character, "--settle")
                    edited = entries()
                    measurement["search_typing_roundtrip_seconds"] = time.monotonic() - tick
                    field = next(e for e in edited if e[1] == "textField" and "Search commits" in e[3])
                    assert f'value={json.dumps(query)}' in field[3], "Search lost typed characters"
                    assert any(e[2] == "Loading complete history…" for e in edited), "Traversal ended during typing; sample is not accepted"
                    assert any(e[1] == "listItem" and e[2].startswith(query) and "offscreen" not in e[3] for e in edited), "Typed search did not expose matching rows"
                    (artifacts / f"search-{number}.snapshot.txt").write_text(auto("snapshot", "--full"))
                    auto("screenshot", str(artifacts / f"search-{number}.png"))
                    auto("fill", field[0], "", "--settle")
                    cleared = entries()
                    cleared_search = next(e for e in cleared if e[1] == "textField" and "Search commits" in e[3])
                    assert "value=" not in cleared_search[3], "Search clear did not take effect"
                    assert any(e[2] == "Loading complete history…" for e in cleared), "Traversal ended before input probe completed"
                    measurement["wheel_and_search_while_reading"] = True
                (artifacts / f"reading-{number}.snapshot.txt").write_text(auto("snapshot", "--full"))
                auto("screenshot", str(artifacts / f"reading-{number}.png"))
                tick = time.monotonic()
                press("button", "Cancel")
                wait("History reading is paused. Loaded commits remain available.")
                measurement["cancel_to_paused_seconds"] = time.monotonic() - tick
                tick = time.monotonic()
                press("button", "Retry")
                wait("32002 commits")
                measurement["retry_to_complete_history_seconds"] = time.monotonic() - tick
                final = entries()
                rows = [e for e in final if e[1] == "listItem" and e[2].startswith("Wide ")]
                assert 0 < len(rows) < 100, "Wide history constructed all rows"
                measurement["complete_history_row_controls"] = len(rows)
                sampler.phase("root_navigation")
                # Return keyboard ownership from the edited search to history.
                if options.input_pressure:
                    row = next(e for e in rows if "offscreen" not in e[3])
                    auto("focus", row[0], "--settle")
                # Clear any line selection and navigate the history to its true root.
                auto("key", "escape", "--settle")
                auto("key", "end", "--settle")
                wait("Wide root")
                assert any(e[1] == "listItem" and e[2] == "Wide root" and "checked" in e[3] for e in entries())
                auto("screenshot", str(artifacts / f"root-{number}.png"))
                assert fingerprint() == before and before["head"] == fixture_head
                sampler.close()
                measurement.update(sampler.result())
                assert measurement["rss_samples"] > 0 and measurement["app_peak_sampled_rss_kib"] > 0, "No application RSS observed"
                results["runs"].append(measurement)
                (artifacts / "results.json").write_text(json.dumps(results, indent=2) + "\n")
                print("PASS " + json.dumps(measurement), flush=True)
            except BaseException:
                (artifacts / f"failure-{number}.memory.json").write_text(json.dumps(sampler.result(), indent=2) + "\n")
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
    assert fingerprint() == before
    results["state_preserved"] = True
    (artifacts / "results.json").write_text(json.dumps(results, indent=2) + "\n")
