#!/usr/bin/env python3
"""Native merge-only resolution acceptance, using two retained owned fixtures."""
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

parser = argparse.ArgumentParser()
parser.add_argument('--project', type=pathlib.Path)
parser.add_argument('--artifacts-parent', type=pathlib.Path)
args = parser.parse_args()
script_dir = pathlib.Path(__file__).resolve().parent
project = (args.project or script_dir.parent).resolve()
base = (args.artifacts_parent or project / 'build').resolve()
root = pathlib.Path(tempfile.mkdtemp(prefix='merge-resolve-', dir=base))
print('ARTIFACTS', root, flush=True)
binary, = (project / 'build').glob('supergit-lobe*e2e')
automation = project / 'build/mygo-automation'
(root / 'binary.txt').write_text(str(binary) + '\nsha256=' + hashlib.sha256(binary.read_bytes()).hexdigest() + '\nmtime=' + str(binary.stat().st_mtime) + '\n')
(root / 'script.py').write_bytes(pathlib.Path(__file__).read_bytes())
allresults = []
blocked = False
for scenario in ('external-resolution', 'markers-observation'):
    if blocked:
        break
    out = root / scenario
    out.mkdir()
    repo, home = out / 'repo', out / 'home'
    file = repo / 'conflict.txt'
    protected = repo / 'protected.bin'
    protected_bytes = b'untouched\x00\xff\n'
    resolved = b'explicit combined resolution: main and topic\n'
    env = {k: v for k, v in os.environ.items() if not k.startswith('GIT_')}
    env.update(HOME=str(home), GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM='1', GIT_TERMINAL_PROMPT='0', GIT_ALLOW_PROTOCOL='file')
    bootstrap = True
    results = []
    app = None
    step = 'bootstrap'

    def git(*items, check=True):
        if not bootstrap:
            assert items[0] not in ('add', 'commit', 'reset', 'restore', 'apply', 'update-index', 'merge'), 'Bypassed UI acceptance step'
        proc = subprocess.run(['git', '-C', str(repo), *items], env=env, capture_output=True, timeout=30)
        with (out / 'git.log').open('ab') as trace:
            trace.write(repr(items).encode() + b'\n' + proc.stdout + proc.stderr + b'\n')
        if check and proc.returncode:
            raise RuntimeError(repr(items) + ': ' + proc.stderr.decode(errors='replace'))
        return proc.stdout if check else proc

    def note(text):
        results.append(text)
        (out / 'results.txt').write_text('\n'.join(results) + '\n')
        print(scenario, text, flush=True)

    def same(actual, expected, label):
        assert actual == expected, label + ': ' + repr(actual) + ' != ' + repr(expected)

    def state(name=None):
        data = {'head': git('rev-parse', 'HEAD').decode().strip(), 'refs': git('show-ref').decode(),
                'symbolic': git('symbolic-ref', 'HEAD').decode().strip(), 'index': git('ls-files', '--stage').decode(),
                'unmerged': git('ls-files', '-u').decode(), 'status': git('status', '--porcelain=v1', '-z').hex(),
                'working': file.read_bytes().hex(), 'protected': protected.read_bytes().hex(),
                'merge_head': (repo / '.git/MERGE_HEAD').read_text() if (repo / '.git/MERGE_HEAD').exists() else ''}
        if name:
            (out / (name + '.git.json')).write_text(json.dumps(data, indent=2) + '\n')
        return data

    def auto(*items):
        proc = subprocess.run([str(automation), *items, '--session', session], env=env, cwd=project, capture_output=True, text=True, timeout=40)
        with (out / 'automation.log').open('a') as trace:
            trace.write(repr(items) + '\n' + proc.stdout + proc.stderr + '\n')
        if proc.returncode:
            raise RuntimeError(repr(items) + ': ' + proc.stdout + proc.stderr)
        return proc.stdout

    def snapshot(name=None):
        text = auto('snapshot', '--full')
        if name:
            (out / (name + '.snapshot.txt')).write_text(text)
        return text

    def entries(text=None):
        rows = []
        for line in (snapshot() if text is None else text).splitlines():
            match = re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)', line)
            if match:
                rows.append((match[1], match[2], json.loads(match[3]) if match[3] else '', match[4]))
        return rows

    def control(role, label, enabled=True):
        until = time.monotonic() + 25
        while True:
            rows = [e for e in entries() if e[1] == role and e[2] == label and (not enabled or 'disabled' not in e[3])]
            if len(rows) == 1:
                return rows[0]
            if len(rows) > 1:
                raise RuntimeError('Ambiguous control: ' + repr((role, label, rows)))
            if app.poll() is not None or time.monotonic() > until:
                raise RuntimeError('Missing control: ' + repr((role, label, entries())))
            time.sleep(.05)

    def press(role, label):
        auto('press', control(role, label)[0], '--settle')

    def wait(text):
        auto('wait', '--text', text, '--timeout', '25s', '--settle')

    def ready():
        control('button', 'Refresh')

    def evidence(name):
        text = snapshot(name)
        auto('screenshot', str(out / (name + '.png')))
        state(name)
        return text

    def mark_resolved():
        row = control('listItem', 'conflict.txt')
        auto('focus', row[0], '--settle')
        auto('key', 'Shift+F10', '--settle')
        wait('Mark resolved')
        evidence(step + '-menu')
        press('menuItem', 'Mark resolved')
        ready()

    def wait_resolved_bytes(expected):
        until = time.monotonic() + 25
        while True:
            unmerged = git('ls-files', '-u')
            if not unmerged and git('show', ':conflict.txt') == expected:
                break
            if app.poll() is not None or time.monotonic() > until:
                raise AssertionError('UI resolution did not produce expected stage-0 blob')
            time.sleep(.05)
        ready()
        same(file.read_bytes(), expected, 'Resolution changed working bytes')
        same(git('rev-parse', 'HEAD'), original_head, 'Resolution moved HEAD')
        same(git('show-ref'), original_refs, 'Resolution moved refs')
        same(protected.read_bytes(), protected_bytes, 'Resolution changed unrelated binary')
        same((repo / '.git/MERGE_HEAD').read_text().strip(), incoming.decode().strip(), 'Resolution lost merge operation')
        stages = git('ls-files', '--stage', '--', 'conflict.txt').decode().splitlines()
        assert len(stages) == 1 and ' 0\tconflict.txt' in stages[0], 'Expected only stage 0'

    try:
        subprocess.run(['git', 'init', '-b', 'main', str(repo)], env=env, check=True, capture_output=True)
        git('config', 'user.name', 'Merge Fixture')
        git('config', 'user.email', 'merge@example.invalid')
        git('config', 'commit.gpgsign', 'false')
        file.write_bytes(b'base\n')
        protected.write_bytes(protected_bytes)
        git('add', '--all')
        git('commit', '--no-gpg-sign', '-m', 'Merge fixture base')
        git('checkout', '-b', 'topic')
        file.write_bytes(b'topic version\n')
        git('commit', '--no-gpg-sign', '-am', 'Merge incoming')
        incoming = git('rev-parse', 'HEAD')
        git('checkout', 'main')
        file.write_bytes(b'main version\n')
        git('commit', '--no-gpg-sign', '-am', 'Merge current')
        original_head = git('rev-parse', 'HEAD')
        original_refs = git('show-ref')
        merge = git('merge', '--no-edit', 'topic', check=False)
        assert merge.returncode != 0 and (repo / '.git/MERGE_HEAD').exists() and git('ls-files', '-u'), 'No fixture merge conflict'
        marker_bytes = file.read_bytes()
        assert b'<<<<<<<' in marker_bytes and b'=======' in marker_bytes and b'>>>>>>>' in marker_bytes
        bootstrap = False
        settings = home / 'Library/Application Support/supergit/settings.json'
        settings.parent.mkdir(parents=True)
        settings.write_text(json.dumps({'language': 'en', 'theme': 'light', 'autoFetch': False}))
        socket_dir = pathlib.Path(tempfile.mkdtemp(prefix='sgmerge-', dir='/tmp'))
        socket_dir.chmod(0o700)
        socket = socket_dir / 'app.sock'
        env.update(MYGO_AUTOMATION=str(socket), SUPERGIT_WINDOW_SIZE='1320x800')
        log = (out / 'app.log').open('w')
        app = subprocess.Popen([str(binary), str(repo)], env=env, cwd=project, stdout=log, stderr=subprocess.STDOUT)
        (out / 'pid.txt').write_text(str(app.pid))
        session = 'merge-resolve-' + str(app.pid)
        until = time.monotonic() + 20
        while not socket.exists():
            if app.poll() is not None or time.monotonic() > until:
                raise RuntimeError('Native process unavailable')
            time.sleep(.05)
        auto('open', '--socket', str(socket))
        wait('Commit summary (required)')
        ready()
        press('tab', 'Changes')
        press('listItem', 'conflict.txt')
        wait('Conflict preview only.')
        step = '01-conflict-visible'
        text = evidence(step)
        assert 'In progress: merge' in text and any(e[1] == 'button' and e[2] == 'Commit' and 'disabled' in e[3] for e in entries(text)), 'Unresolved merge not visible or Commit allowed'
        assert not any(e[1] == 'button' and e[2] == 'Continue' for e in entries(text)), 'Unexpected merge Continue'
        state('before-mark')
        note('PASS conflict preview and merge operation visible; Commit disabled until file resolution')
        if scenario == 'external-resolution':
            step = '02-external-edit-mark-resolved'
            file.write_bytes(resolved)
            press('button', 'Refresh')
            ready()
            # Preview remains a conflict until its current bytes are staged by UI.
            assert git('ls-files', '-u'), 'External edit unexpectedly staged itself'
            mark_resolved()
            wait('Current file staged and marked resolved')
            wait_resolved_bytes(resolved)
            evidence(step)
            note('PASS external explicit resolution → UI Mark resolved yields exact stage0/working bytes; HEAD/refs/merge state/unrelated file preserved')
            step = '03-ui-merge-commit'
            subject = 'Native external merge resolution'
            body = 'Combined main and incoming text\n\nRefs: owned-merge-fixture'
            auto('fill', control('textField', 'Commit summary (required)')[0], subject, '--settle')
            auto('fill', control('textField', 'Description (optional)')[0], body, '--settle')
            resolved_index = git('ls-files', '--stage')
            press('button', 'Commit')
            wait('Committed')
            until = time.monotonic() + 25
            while (repo / '.git/MERGE_HEAD').exists() or git('rev-parse', 'HEAD') == original_head:
                if time.monotonic() > until:
                    raise AssertionError('UI Commit did not finish merge')
                time.sleep(.05)
            ready()
            same(git('show', '-s', '--format=%P', 'HEAD').split(), [original_head.strip(), incoming.strip()], 'Merge parent order mismatch')
            same(git('show', '-s', '--format=%B', 'HEAD').rstrip(b'\n'), (subject + '\n\n' + body).encode(), 'Full merge message mismatch')
            same(git('show', 'HEAD:conflict.txt'), resolved, 'Merge tree resolution mismatch')
            same(git('ls-files', '--stage'), resolved_index, 'Commit changed resolved Index')
            same(file.read_bytes(), resolved, 'Commit changed working bytes')
            same(git('ls-files', '-u'), b'', 'Merge kept conflict stages')
            same(git('status', '--porcelain=v1', '-z'), b'', 'Merge worktree not clean')
            same(git('symbolic-ref', '--short', 'HEAD').strip(), b'main', 'Merge changed branch identity')
            same(git('rev-parse', 'topic'), incoming, 'Merge moved incoming branch')
            same(protected.read_bytes(), protected_bytes, 'Merge changed protected file')
            evidence(step)
            note('PASS UI Commit finishes merge with two expected parents/full message/exact resolution; clean state and MERGE_HEAD removed')
        else:
            step = '02-markers-mark-observation'
            before_mark = state()
            mark_resolved()
            text = evidence(step)
            after_mark = state()
            dialog = any(e[1] in ('dialog', 'alertDialog') for e in entries(text))
            if dialog:
                note('OBSERVATION Mark resolved opened dialog with markers; exact UI/Git state captured; recovery requires handling displayed prompt')
                raise RuntimeError('Unexpected Mark resolved dialog; stop for evidence review')
            if not git('ls-files', '-u'):
                same(git('show', ':conflict.txt'), marker_bytes, 'Marker observation staged unexpected content')
                same(file.read_bytes(), marker_bytes, 'Marker action changed working bytes')
                note('OBSERVATION markers accepted into stage0 unchanged; no confirmation dialog; visible status=' + repr([e[2] for e in entries(text) if e[1] == 'status']))
                accepted = True
            else:
                same(after_mark['index'], before_mark['index'], 'Rejected marker action changed Index')
                same(file.read_bytes(), marker_bytes, 'Rejected marker action changed worktree')
                note('OBSERVATION markers remain unmerged; visible feedback=' + repr([e[2] for e in entries(text) if e[1] == 'status']))
                accepted = False
            same(git('rev-parse', 'HEAD'), original_head, 'Marker observation committed')
            same(git('show-ref'), original_refs, 'Marker observation moved refs')
            assert (repo / '.git/MERGE_HEAD').exists(), 'Marker action ended merge'
            step = '03-markers-recovery-before-commit'
            file.write_bytes(resolved)
            press('button', 'Refresh')
            ready()
            if accepted:
                # Correct the worktree then stage the Modified version through UI.
                rows = [e for e in entries() if e[1] == 'listItem' and e[2] == 'conflict.txt' and 'description="Modified Files"' in e[3]]
                assert len(rows) == 1, 'Recovery Modified version missing'
                auto('focus', rows[0][0], '--settle')
                auto('key', 'Enter', '--settle')
            else:
                mark_resolved()
            wait_resolved_bytes(resolved)
            evidence(step)
            note('PASS markers scenario recoverable before commit: external correction plus UI staging yields exact resolved stage0; HEAD/refs unchanged, merge still pending')
    except BaseException as error:
        blocked = True
        note('FAIL ' + step + ': ' + repr(error))
        (out / 'failure.txt').write_text(traceback.format_exc())
        if app is not None and app.poll() is None:
            try:
                evidence('failure-' + step)
            except Exception as capture_error:
                (out / 'capture-error.txt').write_text(repr(capture_error))
    finally:
        if app is not None:
            app.terminate()
            try:
                app.wait(timeout=10)
            except subprocess.TimeoutExpired:
                app.kill()
                app.wait(timeout=5)
            log.close()
            results.append('PROCESS_STOPPED=' + str(app.returncode))
        (out / 'results.txt').write_text('\n'.join(results) + '\n')
        allresults.extend([scenario + ' ' + row for row in results])
        (root / 'results.txt').write_text('\n'.join(allresults) + '\n')
if blocked:
    raise SystemExit(1)
