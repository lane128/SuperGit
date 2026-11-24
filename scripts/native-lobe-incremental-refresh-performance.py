#!/usr/bin/env python3
"""Measure first usable parent and post-Fetch reconciliation in the native app."""
import argparse
import collections
import json
import os
from pathlib import Path
import re
import shutil
import statistics
import subprocess
import tempfile
import time

project = Path(__file__).resolve().parents[1]
build = project / 'build'
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--repo', type=Path, help='existing checkout, reads and Fetch only')
parser.add_argument('--samples', type=int, default=5)
args = parser.parse_args()
if args.samples < 1:
    parser.error('samples must be positive')
app_binary = build / 'supergit-lobe-e2e'
automation = build / 'mygo-automation'
if not all(p.is_file() for p in (app_binary, automation)):
    raise SystemExit('Run make lobe-e2e-build first')
evidence = Path(tempfile.mkdtemp(prefix='incremental-refresh-', dir=build))
print(f'Evidence: {evidence}', flush=True)
real_git = shutil.which('git')
git_env = {k: v for k, v in os.environ.items() if not k.startswith('GIT_')}
git_env.update(GIT_OPTIONAL_LOCKS='0', GIT_TERMINAL_PROMPT='0')
if not args.repo:
    git_env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM='1', GIT_ALLOW_PROTOCOL='file',
                   GIT_AUTHOR_NAME='Fixture', GIT_COMMITTER_NAME='Fixture',
                   GIT_AUTHOR_EMAIL='fixture@example.invalid', GIT_COMMITTER_EMAIL='fixture@example.invalid')


def git(repo, *argv, input=None):
    return subprocess.check_output([real_git, '-C', repo, *argv], env=git_env, text=True, input=input).strip()


def fixture(root):
    repo, source, remote = (root / n for n in ('workspace', 'source', 'origin.git'))
    for path in (repo, source):
        git(root, 'init', '-q', '-b', 'main', path)
    (source / 'tracked').write_text('base\n')
    git(source, 'add', '.')
    git(source, 'commit', '-qm', 'child')
    sha = git(source, 'rev-parse', 'HEAD')
    git(source, 'commit', '--allow-empty', '-qm', 'second')
    other = git(source, 'rev-parse', 'HEAD')
    modules = []
    for i in range(75):
        name = f'child-{i:02}'
        modules.append(f'[submodule "{name}"]\npath = {name}\nurl = {source}\n')
        git(repo, 'update-index', '--add', '--cacheinfo', f'160000,{sha},{name}')
        if i < 74:
            git(repo, 'clone', '-q', source, name)
            git(repo / name, 'checkout', '-q', '--detach', sha)
    (repo / '.gitmodules').write_text(''.join(modules))
    git(repo, 'add', '.gitmodules')
    git(repo, 'commit', '-qm', 'Parent fixture')
    git(root, 'clone', '-q', '--bare', repo, remote)
    git(repo, 'remote', 'add', 'origin', remote)
    git(repo, 'fetch', '-q', 'origin')
    git(repo, 'branch', '--set-upstream-to=origin/main')
    git(repo, 'remote', 'add', 'second', remote)
    git(root, 'clone', '-q', remote, 'publisher')
    for i in (1, 2):
        (repo / f'child-{i:02}' / 'tracked').write_text('changed\n')
    git(repo / 'child-01', 'add', 'tracked')
    (repo / 'child-03' / 'new').touch()
    (repo / 'parent-dirty').touch()
    git(repo, 'update-index', '--index-info', input=f'0 {"0" * len(sha)}\tchild-73\n160000 {sha} 1\tchild-73\n160000 {sha} 2\tchild-73\n160000 {other} 3\tchild-73\n')
    return repo


def until(fn, label, timeout=90):
    end = time.monotonic() + timeout
    while time.monotonic() < end:
        result = fn()
        if result:
            return result
        time.sleep(.05)
    raise AssertionError(label)


def sample(root, repo, number):
    run = root / str(number)
    run.mkdir()
    config = run / 'config' / 'supergit'
    config.mkdir(parents=True)
    (config / 'settings.json').write_text(json.dumps(dict(language='en', autoFetch=False, autoCheckUpdates=False,
                                                           theme=os.getenv('SUPERGIT_NATIVE_THEME', 'light'))))
    commands, release, blocked, fetched = (run / n for n in ('commands', 'release', 'blocked', 'fetched'))
    shim = run / 'git'
    shim.write_text('''#!/bin/sh
inspect() {
while [ "$#" -gt 0 ]; do
    if [ "$1" = '-C' ]; then checkout=$2; shift 2; break; fi
    shift
done
printf '%s\\t%s\\n' "$checkout" "$1" >> "$COMMANDS"
if [ "$1" = status ] && [ "$checkout" = "$BLOCK_CHILD" ]; then
  touch "$BLOCKED"; while [ ! -f "$RELEASE" ]; do sleep .05; done
fi
if [ "$1" = fetch ] && [ -f "$CANCEL_GATE" ]; then
  for arg do last=$arg; done
  if [ "$last" = second ]; then touch "$SECOND_STARTED"; exec sleep 60; fi
fi
verb=$1
}
inspect "$@"
if [ "$verb" = fetch ]; then
  "$REAL_GIT" "$@"; code=$?; printf '%s\\n' "$code" >> "$FETCHED"; exit "$code"
fi
exec "$REAL_GIT" "$@"
''')
    shim.chmod(0o755)
    socket = root / 'app.sock'
    socket.unlink(missing_ok=True)
    session = f'inc-{os.getpid()}-{number}'
    env = dict(git_env, SUPERGIT_TEST_CONFIG_DIR=str(config.parent), MYGO_AUTOMATION=str(socket),
               PATH=str(run) + os.pathsep + os.environ['PATH'], REAL_GIT=real_git, COMMANDS=str(commands),
               BLOCK_CHILD=str(repo / 'child-00') if not args.repo else '', RELEASE=str(release),
               BLOCKED=str(blocked), FETCHED=str(fetched), CANCEL_GATE=str(run / 'cancel-gate'),
               SECOND_STARTED=str(run / 'second-started'))
    remote_count = len(git(repo, 'remote').splitlines())
    assert remote_count > 0, 'a performance sample needs at least one remote'
    expected = f'Fetched sample {number}'
    branch = git(repo, 'branch', '--show-current')
    if not args.repo:
        git(root / 'publisher', 'commit', '--allow-empty', '-qm', expected)
        git(root / 'publisher', 'push', '-q', 'origin', 'main')
    before = git(repo, 'ls-files', '-s', '-z')
    before_head = git(repo, 'rev-parse', 'HEAD')
    configured = set()
    if (repo / '.gitmodules').exists():
        keys = git(repo, 'config', '--null', '--name-only', '--file', '.gitmodules', '--get-regexp', r'^submodule\..*\.path$')
        configured = {key[len('submodule.'):-len('.path')] for key in keys.split('\0') if key}

    def auto(*argv):
        return subprocess.check_output([automation, *argv, '--session', session], text=True, timeout=45)

    def snapshot():
        return auto('snapshot', '--full')

    def press(role, label):
        match = until(lambda: re.search(r'(@e\d+) \[' + role + r'\] "' + label + r'"(?![^\n]*disabled)', snapshot()), label)
        auto('press', match[1], '--settle')

    def records():
        return [line.split('\t', 1) for line in commands.read_text().splitlines() if '\t' in line]

    start = time.monotonic()
    with (evidence / f'app-{number}.log').open('w') as log:
        proc = subprocess.Popen([app_binary, repo], env=env, stdout=log, stderr=log)
        try:
            until(socket.exists, 'automation socket')
            auto('open', '--socket', socket)
            if not args.repo:
                until(blocked.exists, 'blocked child was never inspected')
            until(lambda: branch in snapshot() and ('Parent fixture' in snapshot() or args.repo), 'parent branch/history')
            if not args.repo:
                assert re.search(r'\[listItem\] "3 changes"', snapshot()), 'parent must include the missing gitlink, conflict and untracked file'
            auto('key', 'cmd+4', '--settle')
            auto('wait', '--text', 'Worktrees and submodules', '--timeout', '30s')
            first = time.monotonic() - start
            release.touch()
            if not args.repo:
                until(lambda: len({p for p, cmd in records() if cmd == 'status' and p != str(repo)}) >= 74, 'child traversal')
                def accurate_details():
                    rows = {name: detail for name, detail in re.findall(r'\[listItem\] "(child-\d+)"([^\n]*)', snapshot())}
                    if len(rows) != 75:
                        return False
                    return all(('Not initialized' if i == 74 else '1 file changed' if i in (1, 2, 3) else f'description="child-{i:02}"') in rows[f'child-{i:02}'] for i in range(75))
                until(accurate_details, '75 initialized, dirty and unavailable row descriptions')
            elif configured:
                def real_details_settled():
                    current = snapshot()
                    rows = dict(re.findall(r'\[listItem\] "([^"]+)"([^\n]*)', current))
                    return all(name in rows and 'Loading repository' not in rows[name] for name in configured)
                until(real_details_settled, 'real checkout detail rows')
            details = time.monotonic() - start
            if number == 1 and not args.repo:
                auto('screenshot', str(evidence / 'workspace.png'))
            auto('key', 'cmd+1', '--settle')
            press('menuButton', 'Sync')
            press('menuItem', 'Fetch')
            def all_fetches_finished():
                codes = fetched.read_text().splitlines() if fetched.exists() else []
                assert all(code == '0' for code in codes), 'Fetch failed; do not count this timing sample'
                return len(codes) == remote_count
            until(all_fetches_finished, 'not every remote finished Fetch')
            if not args.repo:
                until(lambda: expected in snapshot() and f'{number} behind' in snapshot(), 'fetched history and tracking counts')
            else:
                upstream = git(repo, 'for-each-ref', '--format=%(upstream)', f'refs/heads/{branch}') if branch else ''
                if upstream:
                    ahead, behind = git(repo, 'rev-list', '--left-right', '--count', f'HEAD...{upstream}').split()
                    caption = branch if (ahead, behind) == ('0', '0') else f'{branch} ({ahead} ahead · {behind} behind)'
                    until(lambda: f'description={json.dumps(caption, ensure_ascii=False)}' in snapshot(), 'current branch tracking counts')
            until(lambda: re.search(r'\[menuButton\] "Sync"(?![^\n]*disabled)', snapshot()), 'Sync did not settle')
            reconciliation = time.time() - fetched.stat().st_mtime
            after = records()
            cut = max(i for i, (_, cmd) in enumerate(after) if cmd == 'fetch') + 1
            # The watcher disallows overlapping polls. The third start proves the first two completed.
            until(lambda: sum(p == str(repo) and cmd == 'status' for p, cmd in records()[cut:]) >= 3, 'two complete watcher polls', 100)
            post = records()[cut:]
            boundary = [i for i, (p, cmd) in enumerate(post) if p == str(repo) and cmd == 'status'][2]
            post = post[:boundary]
            counts = collections.Counter(cmd for _, cmd in post)
            if not args.repo:
                assert counts['ls-tree'] == counts['ls-files'] == counts['submodule'] == 0, counts
                assert sum(cmd == 'status' and p != str(repo) for p, cmd in post) <= 16, counts
            if not args.repo and number == args.samples:
                git(root / 'publisher', 'commit', '--allow-empty', '-qm', 'Partial fetch retained')
                git(root / 'publisher', 'push', '-q', 'origin', 'main')
                (run / 'cancel-gate').touch()
                press('menuButton', 'Sync')
                press('menuItem', 'Fetch')
                until((run / 'second-started').exists, 'second remote must block after origin advanced')
                press('button', 'Cancel')
                until(lambda: 'Cancelled' in snapshot(), 'cancellation feedback')
                until(lambda: 'Partial fetch retained' in snapshot() and f'{number + 1} behind' in snapshot(), 'partial fetch history and tracking counts')
                press('menuButton', 'Sync')
                auto('key', 'Escape', '--settle')
            assert git(repo, 'ls-files', '-s', '-z') == before and git(repo, 'rev-parse', 'HEAD') == before_head
            return dict(first_usable=first, details=details, post_fetch=reconciliation, commands=counts)
        except Exception:
            if socket.exists():
                (evidence / f'failure-{number}.txt').write_text(snapshot())
            raise
        finally:
            if commands.exists():
                shutil.copyfile(commands, evidence / f'commands-{number}.txt')
            release.touch()
            proc.terminate()
            try:
                proc.wait(10)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait()


with tempfile.TemporaryDirectory(prefix='inc-refresh-') as directory:
    root = Path(directory).resolve()
    repo = args.repo.resolve() if args.repo else fixture(root)
    samples = []
    for number in range(1, args.samples + 1):
        samples.append(sample(root, repo, number))
        print(json.dumps(samples[-1]), flush=True)
        (evidence / 'samples.json').write_text(json.dumps(samples, indent=2))
    for field in ('first_usable', 'details', 'post_fetch'):
        values = [s[field] for s in samples]
        print(field, 'median', statistics.median(values), 'max', max(values))
