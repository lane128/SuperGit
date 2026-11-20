#!/usr/bin/env python3
"""Exercise the standard macOS bundle from make app in disposable fixtures."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile

project = Path(__file__).resolve().parents[1]
build = project / "build"
build.mkdir(exist_ok=True)
trial = Path(tempfile.mkdtemp(prefix="native-supergit-", dir=build))
print(f"Native SuperGit logs: {trial}", flush=True)
app = trial / "SuperGit.app"
binary = app / "Contents/MacOS/supergit"
subprocess.run(
    ["make", "app", f"APP={app}", f"APP_BINARY={trial / 'supergit-e2e'}"],
    cwd=project, check=True, env={**os.environ, "GOFLAGS": "-tags=e2e"},
)
subprocess.run(["codesign", "--verify", "--deep", "--strict", str(app)], check=True)
identity = subprocess.check_output([
    "plutil", "-extract", "CFBundleIdentifier", "raw", "-o", "-", str(app / "Contents/Info.plist"),
], text=True).strip()
if identity != "dev.supergit.preview":
    raise RuntimeError(f"Unexpected standard bundle identity: {identity}")
subprocess.run([
    "go", "build", "-o", str(trial / "mygo-automation"),
    "github.com/egoist/mygo/plugins/automation/cmd/mygo-automation",
], cwd=project, check=True)
subprocess.run([
    sys.executable, "-I", str(project / "scripts/native-repository-tabs-smoke.py"),
    str(app), str(trial / "mygo-automation"),
], cwd=project, check=True)
env = {**os.environ, "SUPERGIT_NATIVE_BUILD_DIR": str(trial), "SUPERGIT_NATIVE_BINARY": str(binary)}
for flow in ("toolbar-sync", "daily-workflow", "stash", "conflict-side", "submodule"):
    command = [sys.executable, "-I", str(project / "scripts" / f"native-lobe-{flow}-smoke.py")]
    if flow == "daily-workflow":
        command.extend(["--artifacts-parent", str(trial)])
    subprocess.run(command, cwd=project, check=True, env=env)
