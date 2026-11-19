#!/usr/bin/env python3
"""Exercise one Lobe daily workflow using newly owned local fixtures only.

Pass --project to the checkout containing build/supergit-lobe*e2e and
build/mygo-automation. Artifacts and fixtures are retained. After bootstrap,
workspace staging and commits must be performed by the native UI; only the
owned peer may use Git add/commit. No network remotes are permitted.
"""
import argparse
import hashlib
import json
import os
import pathlib
import re
import subprocess
import tempfile
import time
import traceback

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--project', type=pathlib.Path)
parser.add_argument('--artifacts-parent', type=pathlib.Path)
args = parser.parse_args()
script_dir = pathlib.Path(__file__).resolve().parent
project = (args.project or (script_dir.parent if script_dir.name == 'scripts' else script_dir)).resolve()
parent = (args.artifacts_parent or project / 'build').resolve()
root = pathlib.Path(tempfile.mkdtemp(prefix='lobe-daily-', dir=parent))
print('ARTIFACTS', root, flush=True)
repo, remote, peer, config = (root / name for name in ('workspace', 'remote.git', 'peer', 'config'))
build = pathlib.Path(os.environ.get('SUPERGIT_NATIVE_BUILD_DIR', project / 'build'))
binary = pathlib.Path(os.environ.get('SUPERGIT_NATIVE_BINARY', build / 'supergit-lobe-e2e'))
automation = build / 'mygo-automation'
(root / 'binary.txt').write_text(str(binary) + '\nsha256=' + hashlib.sha256(binary.read_bytes()).hexdigest() + '\nmtime=' + str(binary.stat().st_mtime) + '\n')
(root / 'script.py').write_bytes(pathlib.Path(__file__).read_bytes())
env = {k: v for k, v in os.environ.items() if not k.startswith('GIT_')}
env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM='1', GIT_TERMINAL_PROMPT='0', GIT_ALLOW_PROTOCOL='file')
bootstrap = True
results = []
step = 'bootstrap'
app = None
session = 'lobe-daily-' + str(os.getpid())
base_bytes = b'one\ntwo\nthree\nfour\n'
alpha_bytes = b'one\nALPHA\ntwo\nthree\nfour\n'
working_bytes = b'one\nALPHA\ntwo\nBETA\nthree\nfour\n'
partial_bytes = b'one\ntwo\nBETA\nthree\nfour\n'
file = repo / 'daily.txt'
protected = repo / 'protected.txt'
summary = 'Daily partial commit'
body = 'First paragraph\n\nSecond paragraph'


def git(*items, directory=repo):
    directory = directory.resolve()
    assert directory.is_relative_to(root), 'Git target escaped owned fixtures'
    if directory == repo and not bootstrap:
        assert items[0] not in ('add', 'commit', 'restore', 'reset', 'apply', 'update-index'), 'Workspace write bypassed UI'
    result = subprocess.run(['git', '-C', str(directory), *items], env=env, capture_output=True, timeout=30)
    with (root / 'git.log').open('ab') as trace:
        trace.write((repr((str(directory), items)) + '\n').encode() + result.stdout + result.stderr + b'\n')
    if result.returncode:
        raise RuntimeError('Git command failed: ' + repr(items) + ' ' + result.stderr.decode(errors='replace'))
    return result.stdout


def same(actual, expected, label):
    assert actual == expected, label + ': actual=' + repr(actual) + ' expected=' + repr(expected)


def note(text):
    results.append('PASS ' + text)
    (root / 'results.txt').write_text('\n'.join(results) + '\n')
    print(results[-1], flush=True)


def state(directory=repo):
    files = {p: (directory / p).read_bytes().hex() for p in ('daily.txt', 'protected.txt', 'peer.txt') if (directory / p).exists()}
    return {'head': git('rev-parse', 'HEAD', directory=directory).decode().strip(),
            'symbolic': git('symbolic-ref', 'HEAD', directory=directory).decode().strip(),
            'index': git('ls-files', '--stage', directory=directory).decode(),
            'status': git('status', '--porcelain=v1', '-z', directory=directory).hex(),
            'refs': git('for-each-ref', '--format=%(refname) %(objectname)', directory=directory).decode(),
            'files': files}


def save_state(name):
    current = state()
    (root / (name + '.git.json')).write_text(json.dumps(current, indent=2) + '\n')
    return current


def local_remote_only():
    same(git('remote').decode().split(), ['origin'], 'Unexpected remotes')
    # Check effective fetch and push URLs, including local url rewriting.
    for items in (('remote', 'get-url', '--all', 'origin'), ('remote', 'get-url', '--push', '--all', 'origin')):
        urls = git(*items).decode().splitlines()
        same(len(urls), 1, 'Ambiguous remote URL')
        assert '://' not in urls[0] and pathlib.Path(urls[0]).is_absolute(), 'Remote must be a local path'
        same(pathlib.Path(urls[0]).resolve(), remote.resolve(), 'Remote escaped owned bare fixture')


def auto(*items):
    result = subprocess.run([str(automation), *items, '--session', session], env=env, cwd=project, capture_output=True, text=True, timeout=40)
    with (root / 'automation.log').open('a') as trace:
        trace.write(repr(items) + '\n' + result.stdout + result.stderr + '\n')
    if result.returncode:
        raise RuntimeError('Automation failed: ' + repr(items) + ' ' + result.stdout + result.stderr)
    return result.stdout


def snapshot(name=None):
    text = auto('snapshot', '--full')
    if name:
        (root / (name + '.snapshot.txt')).write_text(text)
    return text


def entries(text=None):
    found = []
    for line in (snapshot() if text is None else text).splitlines():
        match = re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)', line)
        if match:
            found.append((match[1], match[2], json.loads(match[3]) if match[3] else '', match[4]))
    return found


def control(role, label, description=None, enabled=True, timeout=20):
    deadline = time.monotonic() + timeout
    while True:
        controls = entries()
        matches = [e for e in controls if e[1] == role and e[2] == label and
                   (not enabled or 'disabled' not in e[3]) and
                   (description is None or 'description=' + json.dumps(description, ensure_ascii=False) in e[3])]
        if len(matches) == 1:
            return matches[0]
        if len(matches) > 1:
            raise RuntimeError('Ambiguous native control ' + repr((role, label, description, matches)))
        if app.poll() is not None or time.monotonic() > deadline:
            raise RuntimeError('Missing native control ' + repr((role, label, description, controls)))
        time.sleep(.05)


def press(role, label, description=None):
    auto('press', control(role, label, description)[0], '--settle')


def wait(text):
    auto('wait', '--text', text, '--timeout', '25s', '--settle')


def ready():
    control('button', 'Refresh')


def evidence(name):
    snapshot(name)
    auto('screenshot', str(root / (name + '.png')))
    save_state(name)


def wait_git(items, expected, directory=repo):
    deadline = time.monotonic() + 25
    while git(*items, directory=directory) != expected:
        if app.poll() is not None or time.monotonic() > deadline:
            raise AssertionError('Expected Git state not reached: ' + repr(items))
        time.sleep(.05)
    ready()


def select_version(staged):
    press('listItem', 'daily.txt', 'Staged Files' if staged else 'Modified Files')


def select_and_apply(line_label, staged, expected):
    select_version(staged)
    # Line-number controls are actual selectable row handles, not code paragraphs.
    press('button', line_label)
    action = 'Unstage selection' if staged else 'Stage selection'
    control('button', action)
    evidence(step + '-selected')
    press('button', action)
    wait_git(('show', ':daily.txt'), expected)
    same(file.read_bytes(), working_bytes, 'Line action changed working bytes')
    same(git('rev-parse', 'HEAD'), initial_head, 'Line action changed HEAD')
    same(git('show', ':protected.txt'), protected_bytes, 'Line action touched unrelated Index blob')
    same(protected.read_bytes(), protected_bytes, 'Line action touched unrelated file')
    evidence(step)


def fill_message(subject, description):
    auto('fill', control('textField', 'Commit summary (required)')[0], subject, '--settle')
    auto('fill', control('textField', 'Description (optional)')[0], description, '--settle')


def ui_commit(subject, description, expected_tree, parent):
    fill_message(subject, description)
    before_index = git('ls-files', '--stage')
    before_files = (file.read_bytes(), protected.read_bytes())
    press('button', 'Commit')
    deadline = time.monotonic() + 25
    while git('rev-parse', 'HEAD') == parent:
        if app.poll() is not None or time.monotonic() > deadline:
            raise AssertionError('UI Commit did not advance HEAD')
        time.sleep(.05)
    ready()
    new_head = git('rev-parse', 'HEAD')
    same(git('show', '-s', '--format=%P', 'HEAD').strip(), parent.strip(), 'Commit parent mismatch')
    same(git('show', 'HEAD:daily.txt'), expected_tree, 'Commit included unselected worktree content')
    same(git('show', 'HEAD:protected.txt'), protected_bytes, 'Commit changed protected file')
    same(git('ls-files', '--stage'), before_index, 'Commit changed Index entries')
    same((file.read_bytes(), protected.read_bytes()), before_files, 'Commit changed working bytes')
    same(git('show', '-s', '--format=%an <%ae>', 'HEAD').strip(), b'Daily Fixture <daily@example.invalid>', 'Commit author mismatch')
    message = subject + ('\n\n' + description if description else '')
    same(git('show', '-s', '--format=%B', 'HEAD').rstrip(b'\n'), message.encode(), 'Full commit message mismatch')
    evidence(step)
    return new_head


try:
    subprocess.run(['git', 'init', '-b', 'main', str(repo)], env=env, check=True, capture_output=True)
    subprocess.run(['git', 'init', '--bare', '-b', 'main', str(remote)], env=env, check=True, capture_output=True)
    for key, value in (('user.name', 'Daily Fixture'), ('user.email', 'daily@example.invalid'), ('commit.gpgsign', 'false'), ('tag.gpgsign', 'false')):
        git('config', key, value)
    file.write_bytes(base_bytes)
    protected_bytes = b'protected baseline\n'
    protected.write_bytes(protected_bytes)
    git('add', '--', 'daily.txt', 'protected.txt')
    git('commit', '--no-gpg-sign', '-m', 'Daily fixture base')
    git('remote', 'add', 'origin', str(remote))
    local_remote_only()
    initial_head = git('rev-parse', 'HEAD')
    initial_index = git('ls-files', '--stage')
    bootstrap = False
    settings = config / 'supergit/settings.json'
    settings.parent.mkdir(parents=True)
    settings.write_text(json.dumps({'language': 'en', 'theme': 'light', 'autoFetch': False}))
    # macOS Unix socket paths are short; this private directory is task-owned.
    socket_dir = pathlib.Path(tempfile.mkdtemp(prefix='sgdaily-', dir='/tmp'))
    socket_dir.chmod(0o700)
    socket = socket_dir / 'app.sock'
    env.update(SUPERGIT_TEST_CONFIG_DIR=str(config), MYGO_AUTOMATION=str(socket), SUPERGIT_WINDOW_SIZE='1320x800')
    app_log = (root / 'app.log').open('w')
    app = subprocess.Popen([str(binary), str(repo)], env=env, cwd=project, stdout=app_log, stderr=subprocess.STDOUT)
    (root / 'pid.txt').write_text(str(app.pid))
    deadline = time.monotonic() + 20
    while not socket.exists():
        if app.poll() is not None or time.monotonic() > deadline:
            raise RuntimeError('Native automation did not start')
        time.sleep(.05)
    auto('open', '--socket', str(socket))
    wait('Commit summary (required)')
    ready()
    step = '01-open-edit-refresh'
    press('tab', 'Changes')
    fill_message(summary, body)
    before = save_state('opened')
    file.write_bytes(working_bytes)
    wait('+ALPHA')
    wait('+BETA')
    current = entries()
    assert any(e[1] == 'textField' and e[2] == 'Description (optional)' and 'focused' in e[3] for e in current), 'External edit stole editor focus'
    press('button', 'Refresh')
    wait('+ALPHA')
    ready()
    same(git('rev-parse', 'HEAD'), initial_head, 'Refresh changed HEAD')
    same(git('ls-files', '--stage'), initial_index, 'Refresh changed Index')
    same(file.read_bytes(), working_bytes, 'Refresh changed worktree')
    text = snapshot()
    assert 'value=' + json.dumps(summary) in text and 'value=' + json.dumps(body) in text, 'Refresh lost draft'
    evidence(step)
    note('open by fixture path, external save/watch and Refresh update Diff without Git writes or draft/focus loss')

    step = '02-stage-alpha'
    select_and_apply('2 Added line', False, alpha_bytes)
    note('UI partial stage ALPHA only; exact Index blob and unselected BETA/working bytes verified')
    step = '03-stage-beta'
    select_and_apply('4 Added line', False, working_bytes)
    note('UI partial stage BETA; exact Index now contains both additions')
    step = '04-unstage-alpha'
    select_and_apply('2 Added line', True, partial_bytes)
    note('UI partial unstage ALPHA preserves staged BETA, protected entry, HEAD and worktree')

    step = '05-partial-ui-commit'
    first_head = ui_commit(summary, body, partial_bytes, initial_head)
    same(file.read_bytes(), working_bytes, 'Partial commit changed remaining worktree change')
    assert git('diff', '--', 'daily.txt').strip(), 'Partial commit lost residual ALPHA'
    same(git('diff', '--cached'), b'', 'Partial commit left staged changes')
    note('UI commit tree contains BETA only, exact full message/author/parent; residual ALPHA remains unstaged')

    step = '06-remainder-ui-commit'
    select_version(False)
    auto('focus', control('listItem', 'daily.txt', 'Modified Files')[0], '--settle')
    auto('key', 'Enter', '--settle')
    wait_git(('show', ':daily.txt'), working_bytes)
    second_head = ui_commit('Daily remaining change', '', working_bytes, first_head)
    same(git('status', '--porcelain=v1'), b'', 'Remainder UI commit did not clean workspace')
    note('UI stages remainder and creates second commit; both commit trees checked and workspace clean for FF Pull')

    step = '07-push-cancel'
    local_remote_only()
    before_push = state()
    press('button', 'Push')
    wait('Push main at commit ' + second_head.decode().strip()[:8] + ' to origin/main.')
    press('button', 'Cancel')
    same(git('for-each-ref', '--format=%(refname)', 'refs/heads', directory=remote), b'', 'Cancelled Push wrote remote')
    same(git('for-each-ref', '--format=%(upstream)', 'refs/heads/main').strip(), b'', 'Cancelled Push set upstream')
    same(state(), before_push, 'Cancelled Push changed local repository')
    evidence(step)
    note('UI Push Cancel preserves empty local remote, upstream and local Git state')

    step = '08-push-confirm'
    press('button', 'Push')
    wait('Push main at commit ' + second_head.decode().strip()[:8] + ' to origin/main.')
    press('button', 'Push')
    # A first Push creates this ref asynchronously; absence must remain pollable.
    wait_git(('for-each-ref', '--format=%(objectname)', 'refs/heads/main'), second_head, directory=remote)
    wait('Pushed')
    ready()
    same(git('config', 'branch.main.remote').strip(), b'origin', 'Push upstream remote mismatch')
    same(git('config', 'branch.main.merge').strip(), b'refs/heads/main', 'Push upstream branch mismatch')
    after_push = state()
    for key in ('head', 'symbolic', 'index', 'status', 'files'):
        same(after_push[key], before_push[key], 'Push changed local ' + key)
    evidence(step)
    note('UI Push publishes exactly second UI commit to owned bare main and sets upstream; local checkout unchanged')

    step = '09-peer-advance-fetch'
    subprocess.run(['git', 'clone', str(remote), str(peer)], env=env, check=True, capture_output=True)
    git('config', 'user.name', 'Peer Fixture', directory=peer)
    git('config', 'user.email', 'peer@example.invalid', directory=peer)
    git('config', 'commit.gpgsign', 'false', directory=peer)
    (peer / 'peer.txt').write_bytes(b'peer fast-forward payload\n')
    git('add', '--', 'peer.txt', directory=peer)
    git('commit', '--no-gpg-sign', '-m', 'Daily peer advance', directory=peer)
    same(pathlib.Path(git('remote', 'get-url', '--push', 'origin', directory=peer).decode().strip()).resolve(), remote.resolve(), 'Peer remote escaped fixture')
    git('push', 'origin', 'main', directory=peer)
    peer_head = git('rev-parse', 'HEAD', directory=peer)
    before_fetch = state()
    local_remote_only()
    press('button', 'Fetch')
    wait('Fetched from remotes')
    wait_git(('rev-parse', 'refs/remotes/origin/main'), peer_head)
    after_fetch = state()
    for key in ('head', 'symbolic', 'index', 'status', 'files'):
        same(after_fetch[key], before_fetch[key], 'Fetch changed local ' + key)
    same(git('rev-list', '--left-right', '--count', 'HEAD...@{upstream}').strip(), b'0\t1', 'Fetch ahead/behind mismatch')
    evidence(step)
    note('UI Fetch updates remote-tracking to peer commit; HEAD/Index/worktree unchanged and behind=1')

    step = '10-ff-pull'
    local_remote_only()
    press('button', 'Pull (fast-forward only)')
    wait('Fast-forwarded to upstream')
    wait_git(('rev-parse', 'HEAD'), peer_head)
    same(git('show', '-s', '--format=%P', 'HEAD').strip(), second_head.strip(), 'Pull did not FF onto expected parent')
    same(git('rev-parse', 'refs/heads/main', directory=remote), peer_head, 'Remote main mismatch')
    same(git('rev-parse', 'refs/remotes/origin/main'), peer_head, 'Remote tracking mismatch')
    same(file.read_bytes(), working_bytes, 'Pull changed committed daily content')
    same(protected.read_bytes(), protected_bytes, 'Pull changed protected file')
    same((repo / 'peer.txt').read_bytes(), b'peer fast-forward payload\n', 'Pull missed peer file')
    same(git('status', '--porcelain=v1'), b'', 'Pull left dirty state')
    assert not (repo / '.git/MERGE_HEAD').exists(), 'FF Pull started merge'
    evidence(step)
    note('UI FF-only Pull reaches peer SHA; local/bare/tracking agree, correct files, clean state and no merge')
    note('complete daily workflow passed; only bootstrap and owned peer used Git add/commit')
except BaseException as error:
    results.append('FAIL ' + step + ': ' + repr(error))
    (root / 'failure.txt').write_text(traceback.format_exc())
    if app is not None and app.poll() is None:
        try:
            evidence('failure-' + step)
        except Exception as capture_error:
            (root / 'capture-error.txt').write_text(repr(capture_error))
    print(results[-1], flush=True)
finally:
    if app is not None:
        app.terminate()
        try:
            app.wait(timeout=10)
        except subprocess.TimeoutExpired:
            app.kill()
            app.wait(timeout=5)
        results.append('PROCESS_STOPPED=' + str(app.returncode))
        app_log.close()
    (root / 'results.txt').write_text('\n'.join(results) + '\n')
    print('RESULTS', root / 'results.txt', flush=True)
if any(row.startswith('FAIL ') for row in results):
    raise SystemExit(1)
