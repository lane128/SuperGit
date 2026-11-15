#!/usr/bin/env python3
"""Create an isolated local fixture: 10,000 tracked files and 12,024 commits."""
import io
import json
import os
import pathlib
import subprocess
import tempfile

root = pathlib.Path(tempfile.mkdtemp(prefix="supergit-history."))
repo = root / "workspace"
repo.mkdir()
env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")


def git(*args, **kwargs):
    return subprocess.run(["git", "-C", str(repo), *args], env=env, check=True, capture_output=True, **kwargs)


git("init", "--template=", "-b", "main")
stream = io.BytesIO()


def line(text):
    stream.write(text.encode() + b"\n")


def data(text):
    raw = text.encode()
    line("data " + str(len(raw)))
    stream.write(raw + b"\n")


def commit(ref, mark, subject, parent=None, merge=None):
    line("commit refs/heads/" + ref)
    line("mark :" + str(mark))
    line("committer Preview Developer <preview@example.invalid> " + str(1800000000 + mark) + " +0000")
    data(subject)
    if parent:
        line("from :" + str(parent))
    if merge:
        line("merge :" + str(merge))


line("blob")
line("mark :50000")
data("Tracked fixture contents\n")
for number in range(1, 12001):
    feature = None
    if number % 500 == 0:
        feature = 20000 + number
        commit("feature-" + str(number), feature, f"Perf feature {number:05d}", number - 1)
        line("M 100644 inline feature.txt")
        data(f"Feature at {number}\n")
        line("")
    commit("main", number, f"Perf commit {number:05d}", number - 1 if number > 1 else None, feature)
    if number == 1:
        for index in range(10000):
            line(f"M 100644 :50000 files/{index:05d}.txt")
    line("M 100644 inline files/00000.txt")
    data(f"Version {number}\n")
    if feature:
        line("M 100644 inline feature.txt")
        data(f"Feature at {number}\n")
    line("")
line("done")
git("fast-import", "--quiet", "--done", input=stream.getvalue())
git("reset", "--hard", "HEAD")
counts = {
    "tracked_files": len(git("ls-files", "-z").stdout.split(b"\0")) - 1,
    "commits": int(git("rev-list", "--all", "--count").stdout),
    "head": git("rev-parse", "HEAD").stdout.decode().strip(),
}
(root / "fixture.json").write_text(json.dumps(counts, indent=2) + "\n")
print(root)
