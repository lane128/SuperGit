#!/usr/bin/env python3
"""Measure current Lobe on a fresh large fixture; never accept a user repo path.

Automation round trips are included. RSS is sampled, not an exact peak or a
unique-memory measurement; shared pages can appear in multiple processes.
"""
import argparse
import hashlib
import json
import os
import pathlib
import re
import shutil
import signal
import subprocess
import tempfile
import threading
import time

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--with-submodule", action="store_true", help="Add an initialized local 1,000-file submodule to the generated fixture")
parser.add_argument("--batch-files", type=int, choices=(0, 1000), default=0, help="Also save/restore 1,000 generated tracked files together")
options = parser.parse_args()

project = pathlib.Path(__file__).resolve().parent.parent
build = project / "build"
automation = build / "mygo-automation"
candidates = list(build.glob("supergit-lobe*e2e"))
if not automation.is_file() or len(candidates) != 1:
    raise SystemExit("Run make lobe-e2e-build first")
artifacts = pathlib.Path(tempfile.mkdtemp(prefix="lobe-performance-", dir=build))
git_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
git_env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")
fixture = None


def run():
    global fixture
    generated = subprocess.check_output(["python3", "-I", str(project / "scripts/create-history-demo.py")], cwd=project, env=git_env, text=True).strip()
    fixture = pathlib.Path(generated)
    if not fixture.name.startswith("supergit-history.") or not (fixture / "fixture.json").is_file():
        fixture = None
        raise AssertionError("Generator did not return a fixture it owns")
    repo = fixture / "workspace"
    metadata = json.loads((fixture / "fixture.json").read_text())

    def git(*args):
        return subprocess.check_output(["git", "-C", str(repo), *args], env=git_env)

    assert metadata["tracked_files"] == len(git("ls-files", "-z").split(b"\0")) - 1 == 10001
    assert metadata["commits"] == int(git("rev-list", "--all", "--count")) == 12024
    probe_sha = metadata["head"]
    child = None
    if options.with_submodule:
        source = fixture / "module-source"
        subprocess.run(["git", "init", "--template=", "-b", "main", str(source)], env=git_env, check=True, capture_output=True)
        for index in range(1000):
            (source / f"child-{index:04d}.txt").write_text(f"child fixture {index}\n")
        subprocess.run(["git", "-C", str(source), "add", "--", "."], env=git_env, check=True, capture_output=True)
        subprocess.run(["git", "-C", str(source), "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "core.hooksPath=/dev/null", "commit", "--no-gpg-sign", "-m", "Child fixture"], env=git_env, check=True, capture_output=True)
        git("-c", "protocol.file.allow=always", "submodule", "add", "--", str(source), "vendor/module")
        git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "core.hooksPath=/dev/null", "commit", "--no-gpg-sign", "-m", "Add local module fixture")
        child = repo / "vendor/module"
        metadata.update(tracked_files=10003, commits=12025, head=git("rev-parse", "HEAD").decode().strip(), initialized_submodules=1, child_tracked_files=1000)
        assert len(git("ls-files", "-z").split(b"\0")) - 1 == metadata["tracked_files"]
        assert int(git("rev-list", "--all", "--count")) == metadata["commits"]

    def child_git(*args):
        return subprocess.check_output(["git", "-C", str(child), *args], env=git_env)

    def fingerprint():
        observation = {
            "head": git("rev-parse", "HEAD").decode().strip(),
            "index_sha256": hashlib.sha256(git("ls-files", "--stage", "-z")).hexdigest(),
            "status": git("status", "--porcelain", "-z").decode(),
            "file_sha256": hashlib.sha256((repo / "files/00000.txt").read_bytes()).hexdigest(),
            "refs_sha256": hashlib.sha256(git("for-each-ref", "--format=%(refname) %(objectname)")).hexdigest(),
        }
        if child is not None:
            observation["child_head"] = child_git("rev-parse", "HEAD").decode().strip()
            observation["child_index_sha256"] = hashlib.sha256(child_git("ls-files", "--stage", "-z")).hexdigest()
            observation["child_refs_sha256"] = hashlib.sha256(child_git("for-each-ref", "--format=%(refname) %(objectname)")).hexdigest()
            observation["child_status"] = child_git("status", "--porcelain", "-z").decode()
            observation["child_file_sha256"] = hashlib.sha256((child / "child-0000.txt").read_bytes()).hexdigest()
        return observation

    before = fingerprint()
    oldest_sha = git("rev-list", "--max-parents=0", "HEAD").decode().strip()
    assert before["head"] == metadata["head"] and not before["status"]
    results = {"fixture": metadata, "branches": len(git("branch", "--format=%(refname)").splitlines()), "batch_files": options.batch_files, "runs": []}
    (artifacts / "fixture.json").write_text(json.dumps(metadata, indent=2) + "\n")
    for number in range(1, 4):
        home = fixture / f"home-{number}"
        settings = home / "Library/Application Support/supergit"
        settings.mkdir(parents=True)
        (settings / "settings.json").write_text(json.dumps({"language": "en", "theme": "light", "autoFetch": False}))
        socket = fixture / f"app-{number}.sock"
        session = f"supergit-lobe-performance-{number}"
        env = {**git_env, "HOME": str(home), "MYGO_AUTOMATION": str(socket), "SUPERGIT_WINDOW_SIZE": "1320x640"}
        measurements = {"run": number, "app_peak_sampled_rss_kib": 0, "app_and_descendants_peak_sampled_rss_kib": 0, "rss_samples": 0}
        stop = threading.Event()
        phase = {"name": "launch"}
        measurements["rss_by_phase"] = {}
        started = time.monotonic()
        with (artifacts / f"app-{number}.log").open("w") as log, (artifacts / f"automation-{number}.log").open("w") as trace:
            app = subprocess.Popen([str(candidates[0]), str(repo)], cwd=project, env=env, stdout=log, stderr=subprocess.STDOUT)

            def sample_memory():
                while not stop.is_set():
                    try:
                        output = subprocess.check_output(["ps", "-axo", "pid=,ppid=,rss="], text=True, timeout=3)
                        rows = [tuple(map(int, line.split())) for line in output.splitlines() if line.strip()]
                        descendants = {app.pid}
                        while True:
                            expanded = descendants | {pid for pid, parent, rss in rows if parent in descendants}
                            if expanded == descendants:
                                break
                            descendants = expanded
                        app_rss = sum(rss for pid, parent, rss in rows if pid == app.pid)
                        total = sum(rss for pid, parent, rss in rows if pid in descendants)
                        measurements["app_peak_sampled_rss_kib"] = max(measurements["app_peak_sampled_rss_kib"], app_rss)
                        measurements["app_and_descendants_peak_sampled_rss_kib"] = max(measurements["app_and_descendants_peak_sampled_rss_kib"], total)
                        measurements["rss_samples"] += 1
                        at = measurements["rss_by_phase"].setdefault(phase["name"], {"app_peak_rss_kib": 0, "total_peak_rss_kib": 0, "samples": 0})
                        at["app_peak_rss_kib"] = max(at["app_peak_rss_kib"], app_rss)
                        at["total_peak_rss_kib"] = max(at["total_peak_rss_kib"], total)
                        at["samples"] += 1
                    except (subprocess.SubprocessError, ValueError) as err:
                        measurements.setdefault("rss_sampling_errors", []).append(str(err))
                    stop.wait(0.1)

            sampler = threading.Thread(target=sample_memory, daemon=True)
            sampler.start()

            def auto(*args):
                command = [str(automation), *args, "--session", session]
                trace.write("START " + repr(args) + "\n")
                trace.flush()
                client = subprocess.Popen(command, cwd=project, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
                try:
                    stdout, stderr = client.communicate(timeout=60)
                except subprocess.TimeoutExpired:
                    # Capture both sides before terminating either owned process:
                    # the application Go stack alone cannot expose a client stall
                    # or the native frames beneath AppKit's event loop.
                    diagnostics = {"command": list(args), "phase": phase["name"], "app_pid": app.pid, "client_pid": client.pid}
                    try:
                        sampled = subprocess.run(["/usr/bin/sample", str(app.pid), "2", "10", "-file", str(artifacts / f"failure-{number}.native-sample.txt")], capture_output=True, text=True, timeout=10)
                        diagnostics["native_sample_returncode"] = sampled.returncode
                        diagnostics["native_sample_output"] = sampled.stdout + sampled.stderr
                    except (OSError, subprocess.SubprocessError) as error:
                        diagnostics["native_sample_error"] = str(error)
                    try:
                        client.send_signal(signal.SIGQUIT)
                        stdout, stderr = client.communicate(timeout=10)
                    except (OSError, subprocess.SubprocessError) as error:
                        diagnostics["client_stack_error"] = str(error)
                        client.kill()
                        stdout, stderr = client.communicate(timeout=5)
                    diagnostics["client_returncode"] = client.returncode
                    (artifacts / f"failure-{number}.client-stack.txt").write_text(stdout + stderr)
                    for diagnostic_name, diagnostic_args in (("windows", ["windows"]), ("tree", ["snapshot", "--full"]), ("screenshot", ["screenshot", str(artifacts / f"failure-{number}.live.png")])):
                        try:
                            response = subprocess.run([str(automation), *diagnostic_args, "--session", session], cwd=project, capture_output=True, text=True, timeout=8)
                            diagnostics[diagnostic_name + "_returncode"] = response.returncode
                            (artifacts / f"failure-{number}.live-{diagnostic_name}.txt").write_text(response.stdout + response.stderr)
                        except (OSError, subprocess.SubprocessError) as error:
                            diagnostics[diagnostic_name + "_error"] = str(error)
                    try:
                        diagnostics["fixture_state_matches_before"] = fingerprint() == before
                    except (OSError, subprocess.SubprocessError) as error:
                        diagnostics["fixture_state_error"] = str(error)
                    (artifacts / f"failure-{number}.diagnostics.json").write_text(json.dumps(diagnostics, indent=2) + "\n")
                    raise
                trace.write(repr(args) + "\n" + stdout + stderr + "\n")
                trace.flush()
                if client.returncode:
                    raise RuntimeError(f"{args}: {stdout}{stderr}")
                return stdout

            def entries():
                found = []
                for line in auto("snapshot", "--full").splitlines():
                    match = re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)', line)
                    if match:
                        found.append((match[1], match[2], json.loads(match[3]) if match[3] else "", match[4]))
                return found

            def press(label, role="button"):
                matches = [e for e in entries() if e[1] == role and e[2] == label and "disabled" not in e[3]]
                assert matches, f"Missing enabled {role} {label}"
                auto("press", matches[-1][0], "--settle")

            def wait(text):
                auto("wait", "--text", text, "--timeout", "60s", "--settle")

            def ready():
                deadline = time.monotonic() + 60
                while not any(e[1] == "menuButton" and e[2] == "Sync" and "disabled" not in e[3] for e in entries()):
                    if time.monotonic() > deadline:
                        raise AssertionError("Refresh did not finish")
                    time.sleep(0.05)

            try:
                deadline = time.monotonic() + 30
                while not socket.exists():
                    if app.poll() is not None or time.monotonic() > deadline:
                        raise AssertionError("Native app failed to open automation socket")
                    time.sleep(0.05)
                auto("open", "--socket", str(socket))
                auto("wait", "--text", "Commit message", "--timeout", "30s", "--settle")
                auto("key", "cmd+1", "--settle")
                wait("Perf commit 12000")
                ready()
                measurements["launch_to_first_history_seconds"] = time.monotonic() - started
                phase["name"] = "history_traversal"
                current = entries()
                cancel = [e for e in current if e[1] == "button" and e[2] == "Cancel" and "disabled" not in e[3]]
                if cancel:
                    tick = time.monotonic()
                    auto("press", cancel[-1][0], "--settle")
                    wait("History reading is paused. Loaded commits remain available.")
                    measurements["history_cancel_to_paused_seconds"] = time.monotonic() - tick
                    tick = time.monotonic()
                    press("Retry")
                    wait(f"{metadata['commits']} commits")
                    measurements["resume_to_complete_history_seconds"] = time.monotonic() - tick
                else:
                    measurements["history_cancel_not_observed"] = True
                    wait(f"{metadata['commits']} commits")
                measurements["launch_to_complete_history_including_probe_seconds"] = time.monotonic() - started
                tick = time.monotonic()
                snapshot = auto("snapshot", "--full")
                measurements["complete_history_snapshot_roundtrip_seconds"] = time.monotonic() - tick
                visible = [e for e in entries() if e[1] == "listItem" and e[2].startswith("Perf commit")]
                assert 0 < len(visible) < 100
                measurements["visible_commit_rows"] = len(visible)
                phase["name"] = "head_selection"
                tick = time.monotonic()
                press("Perf commit 12000", "listItem")
                wait(probe_sha)
                wait("Version 12000")
                measurements["head_selection_to_summary_patch_seconds"] = time.monotonic() - tick
                row = next(e for e in entries() if e[1] == "listItem" and e[2] == "Perf commit 12000")
                auto("focus", row[0], "--settle")
                phase["name"] = "oldest_10000_files_selection"
                tick = time.monotonic()
                auto("key", "end", "--settle")
                wait("Perf commit 00001")
                assert any(e[1] == "listItem" and e[2] == "Perf commit 00001" and "checked" in e[3] for e in entries())
                measurements["end_key_to_oldest_selection_seconds"] = time.monotonic() - tick
                oldest_entries = entries()
                file_rows = [e for e in oldest_entries if e[1] == "listItem" and e[2].startswith("files/")]
                section_headers = [e for e in oldest_entries if e[1] == "button" and e[2].startswith("files/")]
                measurements["oldest_file_row_controls"] = len(file_rows)
                measurements["oldest_section_header_controls"] = len(section_headers)
                assert 0 < len(file_rows) < 100, "Oldest commit built all file rows"
                assert 0 < len(section_headers) < 100, "Oldest commit built all patch section headers"
                (artifacts / f"oldest-{number}.snapshot.txt").write_text(auto("snapshot", "--full"))
                auto("screenshot", str(artifacts / f"oldest-{number}.png"))
                phase["name"] = "refresh_after_oldest"
                tick = time.monotonic()
                auto("key", "cmd+r", "--settle")
                ready()
                measurements["unchanged_refresh_to_ready_seconds"] = time.monotonic() - tick
                wait(f"{metadata['commits']} commits")
                measurements["unchanged_refresh_to_full_history_seconds"] = time.monotonic() - tick
                wait(oldest_sha)
                measurements["unchanged_refresh_retains_oldest_sha"] = True
                phase["name"] = "incremental_external_save"
                tracked = repo / "files/00000.txt"
                original = tracked.read_bytes()
                tick = time.monotonic()
                tracked.write_bytes(b"native incremental refresh probe\n")
                wait("1 change")
                measurements["external_save_to_dirty_status_seconds"] = time.monotonic() - tick
                changed = fingerprint()
                for identity in ("head", "index_sha256", "refs_sha256"):
                    assert changed[identity] == before[identity], f"Incremental refresh changed {identity}"
                assert git("status", "--porcelain").decode().splitlines() == [" M files/00000.txt"]
                tick = time.monotonic()
                press("1 change", "listItem")
                wait("native incremental refresh probe")
                measurements["dirty_row_selection_to_patch_seconds"] = time.monotonic() - tick
                auto("screenshot", str(artifacts / f"incremental-{number}.png"))
                tick = time.monotonic()
                tracked.write_bytes(original)
                wait("No changes")
                measurements["external_restore_to_clean_status_seconds"] = time.monotonic() - tick
                if child is not None:
                    assert fingerprint() == before, "Parent save/restore changed the child repository"
                    phase["name"] = "combined_submodule_external_save"
                    child_file = child / "child-0000.txt"
                    child_original = child_file.read_bytes()
                    tick = time.monotonic()
                    child_file.write_bytes(b"native child external save probe\n")
                    wait("1 change")
                    measurements["child_external_save_to_parent_dirty_status_seconds"] = time.monotonic() - tick
                    assert any(e[1] == "listItem" and e[2] == "vendor/module" for e in entries()), "Parent Files did not expose dirty submodule"
                    current = fingerprint()
                    for identity in ("head", "index_sha256", "refs_sha256", "file_sha256", "child_head", "child_index_sha256", "child_refs_sha256"):
                        assert current[identity] == before[identity], f"Child observation changed {identity}"
                    assert child_git("status", "--porcelain").decode().splitlines() == [" M child-0000.txt"]
                    parent_status = git("status", "--porcelain", "-z").split(b"\0")
                    assert len(parent_status) == 2 and parent_status[0][3:] == b"vendor/module"
                    auto("screenshot", str(artifacts / f"submodule-dirty-{number}.png"))
                    tick = time.monotonic()
                    child_file.write_bytes(child_original)
                    wait("No changes")
                    measurements["child_external_restore_to_parent_clean_status_seconds"] = time.monotonic() - tick
                if options.batch_files:
                    assert fingerprint() == before, "Prior probes changed the batch baseline"
                    phase["name"] = "batch_external_save"
                    originals = {repo / f"files/{index:05d}.txt": (repo / f"files/{index:05d}.txt").read_bytes() for index in range(options.batch_files)}
                    tick = time.monotonic()
                    for index, path in enumerate(originals):
                        path.write_bytes(f"native batch external save {index}\n".encode())
                    if child is not None:
                        child_file.write_bytes(b"native batch child external save\n")
                    expected_paths = {f"files/{index:05d}.txt" for index in range(options.batch_files)}
                    if child is not None:
                        expected_paths.add("vendor/module")
                    wait(f"{len(expected_paths)} changes")
                    measurements["batch_save_to_dirty_status_seconds"] = time.monotonic() - tick
                    tick = time.monotonic()
                    current_entries = entries()
                    measurements["batch_status_snapshot_roundtrip_seconds"] = time.monotonic() - tick
                    measurements["batch_file_row_controls"] = len([e for e in current_entries if e[1] == "listItem" and e[2] in expected_paths])
                    assert 0 < measurements["batch_file_row_controls"] < 100, "Batch Files built distant offscreen rows"
                    actual_paths = {record[3:].decode() for record in git("status", "--porcelain", "-z").split(b"\0") if record}
                    assert actual_paths == expected_paths, "Batch changed unexpected parent paths"
                    current = fingerprint()
                    identities = ["head", "index_sha256", "refs_sha256"]
                    if child is not None:
                        identities += ["child_head", "child_index_sha256", "child_refs_sha256"]
                        assert child_git("status", "--porcelain").decode().splitlines() == [" M child-0000.txt"]
                    for identity in identities:
                        assert current[identity] == before[identity], f"Batch observation changed {identity}"
                    (artifacts / f"batch-{number}.snapshot.txt").write_text(auto("snapshot", "--full"))
                    auto("screenshot", str(artifacts / f"batch-{number}.png"))
                    tick = time.monotonic()
                    for path, content in originals.items():
                        path.write_bytes(content)
                    if child is not None:
                        child_file.write_bytes(child_original)
                    wait("No changes")
                    measurements["batch_restore_to_clean_status_seconds"] = time.monotonic() - tick
                    assert all(path.read_bytes() == content for path, content in originals.items()), "Batch restore lost original bytes"
                assert fingerprint() == before, "Performance browsing changed fixture state"
                assert measurements["rss_samples"] > 0
                (artifacts / f"final-{number}.snapshot.txt").write_text(auto("snapshot", "--full"))
                results["runs"].append(measurements)
                (artifacts / "results.json").write_text(json.dumps(results, indent=2) + "\n")
                print("PASS " + json.dumps(measurements), flush=True)
            except BaseException as error:
                (artifacts / f"failure-{number}.measurements.json").write_text(json.dumps(measurements, indent=2) + "\n")
                if isinstance(error, subprocess.TimeoutExpired) and app.poll() is None:
                    # This process is owned by this probe. Preserve Go stacks if
                    # native automation cannot get an answer from its UI thread.
                    app.send_signal(signal.SIGQUIT)
                    app.wait(timeout=10)
                    raise
                try:
                    (artifacts / f"failure-{number}.snapshot.txt").write_text(auto("snapshot", "--full"))
                    auto("screenshot", str(artifacts / f"failure-{number}.png"))
                except Exception:
                    pass
                raise
            finally:
                stop.set()
                sampler.join(timeout=4)
                subprocess.run([str(automation), "close", "--session", session], cwd=project, capture_output=True)
                app.terminate()
                try:
                    app.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    app.kill()
                    app.wait(timeout=5)
    assert fingerprint() == before
    results["state_preserved"] = True
    (artifacts / "results.json").write_text(json.dumps(results, indent=2) + "\n")


print(f"Artifacts: {artifacts}", flush=True)
try:
    run()
finally:
    if fixture is not None:
        shutil.rmtree(fixture)  # Only the generator-created directory owned above.
