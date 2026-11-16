#!/usr/bin/env python3
"""Verify the display-link candidate in an isolated pinned MyGo copy.

This does not change the project's dependency or module cache. --native also
runs application checks and batch/detail-tabs probes. --native-long-range runs
40-line stage/unstage probes. Both restore the ordinary e2e build afterward.
"""
import argparse
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import tempfile


PINNED = "v0.8.4-0.20261108082913-9233663736ca"
MODULE = "github.com/egoist/mygo"
REGRESSION = "TestRequestedFrameRecoversWhenRunningDisplayLinkStopsTicking"
project = pathlib.Path(__file__).resolve().parent.parent
patch = project / "docs/diagnostics/mygo-display-link-recovery.patch"
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--native", action="store_true", help="Also run application Go/race/vet and native batch/detail-tabs probes")
parser.add_argument("--native-long-range", action="store_true", help="Also verify three native 40-line stage/unstage roundtrips using the candidate")
parser.add_argument("--range-line-kind", choices=("del", "add"), default="del", help="Changed-line kind for --native-long-range (default: del)")
options = parser.parse_args()
if options.range_line_kind != "del" and not options.native_long_range:
    parser.error("--range-line-kind add requires --native-long-range")
build = project / "build"
build.mkdir(exist_ok=True)
artifacts = pathlib.Path(tempfile.mkdtemp(prefix="mygo-frame-recovery-", dir=build))
print("Artifacts:", artifacts, flush=True)
# Prevent inherited Git settings from changing patch application semantics.
environment = {key: value for key, value in os.environ.items() if not key.startswith("GIT_")}
environment.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")
summary = {"module": MODULE, "version": PINNED, "patch_sha256": hashlib.sha256(patch.read_bytes()).hexdigest(), "default_dependency_changed": False}


def run(name, command, cwd=project, expected=0):
    print("RUN", name, flush=True)
    result = subprocess.run(command, cwd=cwd, env=environment, capture_output=True, text=True)
    (artifacts / (name + ".log")).write_text(result.stdout + result.stderr)
    if expected is not None and result.returncode != expected:
        raise RuntimeError(f"{name} exited {result.returncode}; see {artifacts / (name + '.log')}")
    return result


def apply(name, module_root, include):
    command = ["git", "apply", "--verbose", "--include=" + include, str(patch)]
    run(name, command, cwd=module_root)


try:
    info = json.loads(run("module", ["go", "list", "-m", "-json", MODULE]).stdout)
    if info.get("Version") != PINNED or "Replace" in info:
        raise RuntimeError("This candidate requires the original pinned module, without a replacement")
    with tempfile.TemporaryDirectory(prefix="supergit-mygo-frame-") as temporary:
        temporary = pathlib.Path(temporary)
        module_root = temporary / "mygo"
        shutil.copytree(info["Dir"], module_root)
        darwin = module_root / "internal/darwin"
        darwin.chmod(darwin.stat().st_mode | 0o200)
        surface = darwin / "surface.go"
        surface.chmod(surface.stat().st_mode | 0o200)
        modfile = artifacts / "candidate.mod"
        shutil.copyfile(project / "go.mod", modfile)
        shutil.copyfile(project / "go.sum", modfile.with_suffix(".sum"))
        run("replace", ["go", "mod", "edit", "-modfile", str(modfile), "-replace", MODULE + "=" + str(module_root)])
        apply("add-regression", module_root, "internal/darwin/stalled_display_link_test.go")
        if not (darwin / "stalled_display_link_test.go").is_file():
            raise RuntimeError("Patch did not add the regression")
        red = run("red", ["go", "test", "-modfile", str(modfile), MODULE + "/internal/darwin", "-run", "^" + REGRESSION + "$", "-count=1"], expected=None)
        if red.returncode == 0 or "frames [], want one unpaced recovery frame" not in red.stdout + red.stderr:
            raise RuntimeError("Original dependency did not fail with the expected missing recovery frame")
        summary["red_verified"] = True
        apply("apply-recovery", module_root, "internal/darwin/surface.go")
        run("green", ["go", "test", "-modfile", str(modfile), MODULE + "/internal/darwin", "-count=1"])
        summary["darwin_green"] = True
        if options.native:
            for name, command in (
                ("application-go", ["go", "test", "-modfile", str(modfile), "./..."]),
                ("application-race", ["go", "test", "-race", "-modfile", str(modfile), "./..."]),
                ("application-vet", ["go", "vet", "-modfile", str(modfile), "./..."]),
            ):
                run(name, command)
        if options.native or options.native_long_range:
            run("ordinary-build", ["make", "lobe-e2e-build"])
            binaries = list(build.glob("supergit-lobe*e2e"))
            if len(binaries) != 1:
                raise RuntimeError("Expected one ordinary Lobe e2e binary")
            try:
                run("candidate-build", ["go", "build", "-modfile", str(modfile), "-tags", "e2e", "-o", str(binaries[0]), "./cmd/supergit-lobe"])
                if options.native:
                    run("native-batch", ["python3", "-I", "scripts/native-lobe-performance-baseline.py", "--with-submodule", "--batch-files", "1000"])
                    run("native-detail-tabs", ["python3", "-I", "scripts/native-detail-tabs-smoke.py"])
                    summary["native_passed"] = True
                if options.native_long_range:
                    run("native-long-range", ["python3", "-I", "scripts/native-lobe-long-patch-baseline.py", "--stage-unstage", "--selected-lines", "40", "--line-kind", options.range_line_kind])
                    summary["native_range_line_kind"] = options.range_line_kind
                    summary["native_long_range_passed"] = True
            finally:
                run("restore-ordinary-build", ["make", "lobe-e2e-build"])
                summary["ordinary_build_restored"] = True
        summary["passed"] = True
finally:
    (artifacts / "results.json").write_text(json.dumps(summary, indent=2) + "\n")
print("PASS", json.dumps(summary), flush=True)
