#!/usr/bin/env python3
"""Exercise the delivery CLI against a local Git fixture and temporary state."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile


SOURCE = Path(__file__).resolve().parents[1]
TAG = "v9.9.9"


def isolated_environment(root):
    env = {
        name: value for name, value in os.environ.items()
        if not name.upper().startswith(("GIT_", "GO", "CGO_"))
    }
    env.update(
        LC_ALL="C",
        TMPDIR=str(root),
        GIT_ALLOW_PROTOCOL="file",
        GIT_CONFIG_NOSYSTEM="1",
        GIT_CONFIG_GLOBAL=os.devnull,
        GIT_TERMINAL_PROMPT="0",
        GIT_TEMPLATE_DIR=str(root / "git-template"),
        CGO_ENABLED="0",
        GOCACHE=str(root / "go-cache"),
        GOMODCACHE=str(root / "go-mod-cache"),
        GOENV="off",
        GOFLAGS="-mod=readonly -buildvcs=false",
        GOPROXY="off",
        GOSUMDB="off",
        GOTELEMETRY="off",
        GOTOOLCHAIN="local",
        GOVCS="*:off",
        GOWORK="off",
    )
    return env


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def state_snapshot(state):
    return {
        path.relative_to(state): (path.read_bytes(), path.stat().st_mtime_ns)
        for path in state.rglob("*") if path.is_file()
    }


def dry_run(root):
    remote = root / "remote.git"
    work = root / "work"
    keys = root / "keys"
    state = root / "state"
    release_output = root / "release-output"
    for directory in (root / "git-template", keys, state, release_output):
        directory.mkdir(mode=0o700)
    env = isolated_environment(root)

    def run(label, *args, cwd=root, expected=0):
        try:
            result = subprocess.run(
                args, cwd=cwd, env=env, capture_output=True, text=True,
                check=False, timeout=120,
            )
        except subprocess.TimeoutExpired as exc:
            raise RuntimeError(f"{label} timed out") from exc
        if result.returncode != expected:
            details = "\n".join(filter(None, (result.stdout.strip(), result.stderr.strip())))
            raise RuntimeError(f"{label} exited {result.returncode}, expected {expected}\n{details}")
        return result

    def git(*args):
        return run(f"git {args[0]}", "git", "-C", work, *args).stdout.strip()

    def delivery(command, *args, expected=0):
        return run(
            f"delivery {command}", "go", "run", "./tools/delivery", command, *args,
            cwd=SOURCE, expected=expected,
        )

    def task(command, *args, expected=0):
        return delivery(command, *args, "-state", state, expected=expected)

    run("initialize bare remote", "git", "init", "--bare", "--initial-branch=main", remote)
    run("initialize work repo", "git", "init", "--initial-branch=main", work)
    fixture = work / "fixture.txt"
    fixture.write_text("delivery fixture\n", encoding="utf-8")
    git("add", "fixture.txt")
    git("config", "user.name", "Delivery Dry Run")
    git("config", "user.email", "dry-run@example.invalid")
    git("commit", "-m", "Create delivery fixture")
    git("remote", "add", "origin", remote)
    git("tag", TAG)
    git("push", "origin", "main", TAG)

    private_key = keys / "update.key"
    public_key = keys / "update.pub"
    run("generate signing keys", "go", "run", "./tools/releasepack", "-genkey",
        "-priv", private_key, "-pub", public_key, cwd=SOURCE)
    private_key.chmod(0o600)
    preflight = (
        "-repo", work, "-tag", TAG, "-pubkey", public_key.read_text(encoding="ascii").strip(),
        "-privkey", private_key,
    )
    print(delivery("preflight", *preflight).stdout.strip())
    fixture.write_text("uncommitted fixture change\n", encoding="utf-8")
    dirty = delivery("preflight", *preflight, expected=1)
    require("FAIL clean:" in dirty.stdout, "dirty preflight did not report a failed clean check")
    print("Dirty worktree rejected by preflight.")
    git("restore", "fixture.txt")

    for task_id in ("task-a", "task-b"):
        task("start", "-id", task_id, "-request", "Exercise delivery phase gates",
             "-repo", work, "-by", "dry-run", "-version-policy", "exact", "-version", "9.9.9",
             "-channel", "darwin-arm64", "-acceptance", "Exercise local preflight")
    task("queue", "-id", "task-a", "-acquire")
    blocked = task("queue", "-id", "task-b", "-acquire", expected=1)
    require(blocked.stdout == "" and 'held by task "task-a"' in blocked.stderr,
            "task B was not rejected because task A held the window")
    print("Task A holds the window and task B is blocked.")

    candidate = git("rev-parse", "HEAD")
    task("advance", "-id", "task-a", "-to", "analysis")  # start records the intake check
    for check, phase in (
        ("analysis", "design"),
        ("design", "design_review"),
        ("design_review", "implementation"),
        ("tests", "code_review"),
        ("code_review", "acceptance"),
        ("acceptance", "pull_request"),
        ("remote_checks", "merge"),
        ("merge", "release_preflight"),
        ("release_preflight", "tag"),
        ("package", "publish"),
        ("publish", "verify"),
        ("public_verify", "complete"),
    ):
        if check == "code_review":
            fixture.write_text("delivery fixture with review fix\n", encoding="utf-8")
            git("add", "fixture.txt")
            git("commit", "-m", "Apply fixture review fix")
            candidate = git("rev-parse", "HEAD")
        task("check", "-id", "task-a", "-name", check, "-candidate", candidate,
             "-passed", "-evidence", f"Dry-run fixture for {check}")
        if check == "code_review":
            rejected = task("advance", "-id", "task-a", "-to", phase,
                            "-candidate", candidate, expected=1)
            require(rejected.stdout == "" and '"tests"' in rejected.stderr,
                    "revised candidate was not rejected for missing review-phase tests")
            task("check", "-id", "task-a", "-name", "tests", "-candidate", candidate,
                 "-passed", "-evidence", "Dry-run fixture for tests rerun after review fix")
            print("Code review rejected the revised candidate until its tests were recorded.")
        result = task("advance", "-id", "task-a", "-to", phase, "-candidate", candidate)
        require(result.stdout.strip() == phase, f"task A did not report phase {phase}")
    task("queue", "-id", "task-b", "-acquire")
    print("Task A completed and task B acquired the window.")

    snapshot = state_snapshot(state)
    read_status = ("status", "-id", "task-a", "-json")
    first = task(*read_status).stdout
    require(json.loads(first)["phase"] == "complete", "task A did not persist completion")
    require(task(*read_status).stdout == first and state_snapshot(state) == snapshot,
            "repeated status changed the task or state files")
    repeated = task("advance", "-id", "task-a", "-to", "complete", expected=1)
    require(repeated.stdout == "" and "cannot advance" in repeated.stderr,
            "repeated advance did not report an invalid transition")
    require(task(*read_status).stdout == first and state_snapshot(state) == snapshot,
            "repeated advance changed the task or state files")
    require(git("tag", "--list").splitlines() == [TAG], "work repo tag set changed")
    require(not any(release_output.iterdir()), "dry-run release output directory is not empty")
    print("Repeated status and advance left state unchanged. The fixture still has only v9.9.9.")


def main():
    try:
        with tempfile.TemporaryDirectory(prefix="supergit-delivery-") as directory:
            dry_run(Path(directory))
    except (OSError, RuntimeError, ValueError, KeyError) as exc:
        print(f"FAIL delivery dry-run: {exc}", file=sys.stderr)
        return 1
    print("PASS delivery dry-run")
    return 0


if __name__ == "__main__":
    sys.exit(main())
