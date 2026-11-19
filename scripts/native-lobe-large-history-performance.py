#!/usr/bin/env python3
"""Measure completed large-history browsing, including many refs and long edges.

Complements wide-history-baseline.py, whose interleaved-loading probe deliberately
rejects runs that finish loading before input starts. Uses disposable Git data.
"""
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

project = pathlib.Path(__file__).resolve().parents[1]
build = project / "build"
automation = build / "mygo-automation"
candidates = list(build.glob("supergit-lobe*e2e"))
if not automation.is_file() or len(candidates) != 1:
    raise SystemExit("Run make lobe-e2e-build first")
Sampler = runpy.run_path(str(project / "scripts/native-memory-sampler.py"))["NativeMemorySampler"]
artifacts = pathlib.Path(tempfile.mkdtemp(prefix="lobe-large-performance-", dir=build))
print(f"Artifacts: {artifacts}", flush=True)
git_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
git_env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")
results = {"branches": 64, "commits_per_branch": 500, "extra_refs": 512, "expected_commits": 32002}

with tempfile.TemporaryDirectory(prefix="supergit-large-performance.") as directory:
    root = pathlib.Path(directory)
    repo = root / "workspace"
    subprocess.run(["git", "init", "--template=", "-b", "main", str(repo)], env=git_env, check=True, capture_output=True)

    def git(*args, **kwargs):
        return subprocess.check_output(["git", "-C", str(repo), *args], env=git_env, **kwargs)

    stream = io.BytesIO()

    def line(text):
        stream.write(text.encode() + b"\n")

    def commit(branch, mark, subject, parents=()):
        line(f"commit refs/heads/{branch}")
        line(f"mark :{mark}")
        line(f"committer Fixture <fixture@example.invalid> {1800000000 + mark} +0000")
        encoded = subject.encode()
        line(f"data {len(encoded)}")
        stream.write(encoded + b"\n")
        for index, parent in enumerate(parents):
            line(f"{'from' if index == 0 else 'merge'} :{parent}")
        line("")

    commit("main", 1, "Performance root")
    tips = [1] * 64
    mark = 1
    # Interleave object timestamps; periodically connect neighboring branches.
    for revision in range(500):
        for branch in range(64):
            mark += 1
            parents = [tips[branch]]
            if revision and revision % 37 == 0:
                parents.append(tips[(branch + 1) % 64])
            commit(f"parallel-{branch:02d}", mark, f"Performance branch {branch:02d} revision {revision:04d}", parents)
            tips[branch] = mark
    commit("main", mark + 1, "Performance merge tip", [1, *tips])
    line("done")
    git("fast-import", "--quiet", "--done", input=stream.getvalue())
    git("reset", "--hard", "HEAD")
    # A large inventory exercises sidebar costs without adding fake commits.
    ref_input = "".join(f"create refs/heads/inventory-{i:04d} refs/heads/parallel-{i % 64:02d}\n" for i in range(512))
    git("update-ref", "--stdin", input=ref_input.encode())
    assert int(git("rev-list", "--all", "--count")) == 32002

    def fingerprint():
        return {
            "head": git("rev-parse", "HEAD").decode().strip(),
            "refs": hashlib.sha256(git("for-each-ref", "--format=%(refname) %(objectname)")).hexdigest(),
            "index": hashlib.sha256(git("ls-files", "--stage", "-z")).hexdigest(),
            "status": git("status", "--porcelain", "-z").decode(),
        }

    before = fingerprint()
    home = root / "home"
    settings = home / "Library/Application Support/supergit"
    settings.mkdir(parents=True)
    (settings / "settings.json").write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False}))
    socket = root / "app.sock"
    session = "supergit-large-performance"
    env = {**git_env, "HOME": str(home), "MYGO_AUTOMATION": str(socket), "SUPERGIT_WINDOW_SIZE": "1320x640"}
    with (artifacts / "app.log").open("w") as log, (artifacts / "automation.log").open("w") as trace:
        started = time.monotonic()
        app = subprocess.Popen([str(candidates[0]), str(repo)], cwd=project, env=env, stdout=log, stderr=subprocess.STDOUT)
        sampler = Sampler(app.pid)

        def auto(*args):
            tick = time.monotonic()
            completed = subprocess.run([str(automation), *args, "--session", session], cwd=project, capture_output=True, text=True, timeout=90)
            trace.write(repr(args) + "\n" + completed.stdout + completed.stderr + "\n")
            trace.flush()
            if completed.returncode:
                raise RuntimeError(f"{args}: {completed.stdout}{completed.stderr}")
            return completed.stdout, time.monotonic() - tick

        def entries():
            found = []
            for text in auto("snapshot", "--full")[0].splitlines():
                match = re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)', text)
                if match:
                    found.append((match[1], match[2], json.loads(match[3]) if match[3] else "", match[4]))
            return found

        try:
            deadline = time.monotonic() + 30
            while not socket.exists():
                if app.poll() is not None or time.monotonic() > deadline:
                    raise AssertionError("Native app did not open automation socket")
                time.sleep(0.05)
            auto("open", "--socket", str(socket))
            auto("wait", "--text", "Commit summary (required)", "--timeout", "30s", "--settle")
            auto("key", "cmd+1", "--settle")
            auto("wait", "--text", "32002 commits", "--timeout", "80s", "--settle")
            results["launch_to_complete_history_seconds"] = time.monotonic() - started
            sampler.phase("completed_history_browsing")
            auto("screenshot",str(artifacts / "history.png"))
            current = entries()
            visible = [e for e in current if e[1] == "listItem" and e[2].startswith("Performance ") and "offscreen" not in e[3]]
            assert 1 <= len(visible) <= 100, "History did not retain viewport virtualization"
            results["visible_history_rows"] = len(visible)
            tip = next(e for e in visible if e[2] == "Performance merge tip")
            auto("focus", tip[0], "--settle")
            _, elapsed = auto("key", "end", "--settle")
            results["end_roundtrip_seconds"] = elapsed
            assert elapsed < 3, "End blocked on completed history"
            assert any(e[1] == "listItem" and e[2] == "Performance root" and "checked" in e[3] and "offscreen" not in e[3] for e in entries()), "End did not reveal the oldest commit"
            _, elapsed = auto("key", "home", "--settle")
            results["home_roundtrip_seconds"] = elapsed
            assert elapsed < 3, "Home blocked on completed history"
            search = next(e for e in entries() if e[1] == "textField" and "Search commits" in e[3])
            auto("focus", search[0], "--settle")
            _, elapsed = auto("type", "Performance branch 03", "--settle")
            results["search_roundtrip_seconds"] = elapsed
            assert elapsed < 3, "Search blocked on completed history"
            edited = entries()
            field = next(e for e in edited if e[1] == "textField" and "Search commits" in e[3])
            assert 'value="Performance branch 03"' in field[3], "Search lost input"
            assert any(e[1] == "listItem" and e[2].startswith("Performance branch 03") for e in edited), "Search did not find loaded older history"
            auto("screenshot",str(artifacts / "search.png"))
            auto("key", "cmd+a", "--settle")
            auto("key", "backspace", "--settle")
            auto("wait", "--text", "32002 commits", "--timeout", "10s", "--settle")
            cleared = entries()
            cleared_search = next(e for e in cleared if e[1] == "textField" and "Search commits" in e[3])
            assert "value=" not in cleared_search[3], "Search clear did not restore the full history"
            worktree = next(e for e in cleared if e[1] == "listItem" and e[2] == "No changes")
            auto("focus", worktree[0], "--settle")
            auto("key", "home", "--settle")
            auto("key", "down", "--settle")
            auto("wait", "--text", "Performance merge tip", "--timeout", "10s", "--settle")
            results.update(sampler.result())
            assert results["app_peak_sampled_rss_kib"] < 384 * 1024, "Application RSS exceeded the synthetic workload budget"
            assert fingerprint() == before, "Browsing modified Git data"
            results["git_unchanged"] = True
            results["passed"] = True
        except Exception:
            results.update(sampler.result())
            results["passed"] = False
            try:
                auto("screenshot",str(artifacts / "failure.png"))
                (artifacts / "failure.snapshot.txt").write_text(auto("snapshot", "--full")[0])
            except Exception:
                pass
            raise
        finally:
            sampler.close()
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
