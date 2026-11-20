#!/usr/bin/env python3
"""Check the macOS repository-tab journey through a disposable app bundle."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import plistlib
import re
import shutil
import signal
import subprocess
import tempfile
import time
import uuid


def run(bundle, driver, artifacts, scratch):
    app = scratch / 'SuperGit Tabs Fixture.app'
    shutil.copytree(bundle, app)
    plist = app / 'Contents/Info.plist'
    info = plistlib.loads(plist.read_bytes())
    info.update(CFBundleIdentifier=f'dev.supergit.fixture.tabs.{uuid.uuid4().hex}',
                CFBundleName=app.stem, CFBundleDisplayName=app.stem)
    plist.write_bytes(plistlib.dumps(info))
    subprocess.run(['codesign', '--force', '--sign', '-', str(app)], check=True)
    subprocess.run(['codesign', '--verify', '--deep', '--strict', str(app)], check=True)
    executable = app / 'Contents/MacOS' / info['CFBundleExecutable']
    session = scratch.name
    profile = artifacts / 'profile'
    settings = profile / 'supergit' / 'settings.json'
    settings.parent.mkdir(parents=True)
    settings.write_text(json.dumps({'language': 'en', 'theme': 'light', 'autoFetch': False}))
    env = {k: v for k, v in os.environ.items() if not k.startswith(('GIT_', 'MYGO_', 'SUPERGIT_'))}
    test_env = dict(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM='1', GIT_ALLOW_PROTOCOL='file',
                    GIT_OPTIONAL_LOCKS='0', GIT_TERMINAL_PROMPT='0',
                    GIT_AUTHOR_NAME='Tabs Fixture', GIT_COMMITTER_NAME='Tabs Fixture',
                    GIT_AUTHOR_EMAIL='fixture@example.invalid', GIT_COMMITTER_EMAIL='fixture@example.invalid',
                    SUPERGIT_TEST_CONFIG_DIR=str(profile), SUPERGIT_WINDOW_SIZE='980x640',
                    MYGO_AUTOMATION_DIR=str(scratch / 'driver'))
    env.update(test_env)

    def command(*args):
        result = subprocess.run(args, env=env, cwd=artifacts, capture_output=True, text=True, timeout=20)
        with (artifacts / 'actions.log').open('a') as log:
            print(repr(args), result.stdout, result.stderr, sep='\n', file=log)
        result.check_returncode()
        return result.stdout

    def auto(*args):
        return command(str(driver), *map(str, args), '--session', session)

    def until(check, message):
        deadline = time.monotonic() + 15
        while time.monotonic() < deadline:
            result = check()
            if result:
                return result
            time.sleep(.08)
        raise AssertionError(message)

    def open_app(index):
        socket = scratch / f'app-{index}.sock'
        options = {**test_env, 'MYGO_AUTOMATION': str(socket)}
        output = artifacts / f'app-{index}.log'
        output.touch()
        argv = ['/usr/bin/open', '-n', '-F', '-W', '-a', str(app), '--stdout', str(output), '--stderr', str(output)]
        for key, value in options.items():
            argv.extend(('--env', f'{key}={value}'))
        process = subprocess.Popen(argv, env=env, cwd=artifacts)
        launches.append(process)
        until(socket.exists, 'The fixture app did not expose its automation socket')
        auto('open', '--socket', socket)
        return process

    def control(tree, role, label):
        pattern = re.compile(r'\s*@e\d+ \[' + re.escape(role) + r'\] ' +
                             re.escape(json.dumps(label, ensure_ascii=False)) + r'(?= |$)')
        return [line.strip() for line in tree.splitlines() if pattern.match(line)]

    def snapshot():
        return auto('snapshot', '--full')

    def one(tree, role, label):
        matches = control(tree, role, label)
        assert len(matches) == 1, (role, label, matches)
        return matches[0]

    def rect(line):
        bounds = re.search(r'\bbounds=([-\d.]+),([-\d.]+),([\d.]+)x([\d.]+)', line)
        assert bounds, line
        return tuple(map(float, bounds.groups()))

    def click(role, label):
        line = one(snapshot(), role, label)
        assert ' disabled' not in line and ' offscreen' not in line, line
        x, y, width, height = rect(line)
        auto('click', x + width / 2, y + height / 2, '--settle')

    def selected(label, ready=False):
        def check():
            tree = snapshot()
            tabs = control(tree, 'tab', label)
            if len(tabs) != 1 or ' checked ' not in tabs[0]:
                return None
            if ready and not any(' disabled' not in item for item in control(tree, 'menuButton', 'Sync')):
                return None
            return tree
        return until(check, f'Tab did not activate: {label}')

    def seed(label):
        repo = artifacts / f'{label} Repository'
        repo.mkdir()
        path = repo / f'{label.lower()}-fixture.txt'
        command('git', '-C', str(repo), 'init', '-b', 'main')
        path.write_text('base\n')
        command('git', '-C', str(repo), 'add', '--', path.name)
        command('git', '-C', str(repo), 'commit', '--no-gpg-sign', '-m', f'{label} baseline')
        path.write_text(f'{label} worktree\n')
        return repo, path.name

    def fingerprint(repo):
        return {str(path.relative_to(repo)): hashlib.sha256(path.read_bytes()).hexdigest()
                for path in repo.rglob('*') if path.is_file()}

    launches = []
    alpha, alpha_file = seed('Alpha')
    beta, beta_file = seed('Beta')
    gamma, _ = seed('Gamma')
    nested_alpha = alpha / 'Nested Folder'
    nested_alpha.mkdir()
    fixtures = (alpha, beta, gamma)
    before = [fingerprint(repo) for repo in fixtures]
    try:
        process = open_app(1)
        selected('Welcome')
        command('/usr/bin/open', '-a', str(app), str(alpha))
        selected(alpha.name, ready=True)
        click('listItem', '1 change')
        field = one(snapshot(), 'textField', 'Commit message').split()[0]
        auto('fill', field, 'Alpha fixture draft\n\nAlpha fixture body', '--settle')
        loaded = snapshot()
        for label in ('Open Repository…', 'Open a Git repository'):
            assert not control(loaded, 'button', label), f'Duplicate open button in workbench: {label}'
        branch = one(loaded, 'button', 'Branches')
        x, _, width, _ = rect(branch)
        window_width = int(test_env['SUPERGIT_WINDOW_SIZE'].split('x')[0])
        left, _, _, _ = rect(one(loaded, 'button', 'Command palette'))
        assert abs((left + x + width) / 2 - window_width / 2) <= 1, branch
        auto('screenshot', artifacts / 'workbench.png')
        click('button', 'Branches')
        one(snapshot(), 'textField', 'Filter branches')
        auto('key', 'Escape', '--settle')
        click('button', 'New Tab')
        welcome = selected('Welcome')
        open_button = one(welcome, 'button', 'Open Repository…')
        assert ' disabled' not in open_button and ' offscreen' not in open_button, open_button
        assert beta.name not in welcome, 'Beta unexpectedly appeared in Recents'
        auto('screenshot', artifacts / 'welcome.png')
        command('/usr/bin/open', '-a', str(app), str(beta))
        tree = selected(beta.name, ready=True)
        one(tree, 'tab', alpha.name)
        assert not control(tree, 'tab', 'Welcome'), 'Beta did not reuse Welcome'
        click('listItem', '1 change')
        field = one(snapshot(), 'textField', 'Commit message').split()[0]
        auto('fill', field, 'Beta fixture draft\n\nBeta fixture body', '--settle')
        for repo, own, other in ((alpha, alpha_file, beta_file), (beta, beta_file, alpha_file)):
            click('tab', repo.name)
            tree = selected(repo.name, ready=True)
            assert own in tree and other not in tree, 'Changes crossed repository tabs'
            for name in ('Alpha', 'Beta'):
                for part in ('draft', 'body'):
                    assert (f'{name} fixture {part}' in tree) == repo.name.startswith(name), f'{name} {part} crossed repository tabs'
        # A folder delivered with a checkout already active adds its own tab.
        command('/usr/bin/open', '-a', str(app), str(gamma))
        occupied = selected(gamma.name, ready=True)
        for repo in fixtures:
            one(occupied, 'tab', repo.name)
        assert not control(occupied, 'tab', 'Welcome')
        # A nested folder belongs to the existing Alpha checkout.
        command('/usr/bin/open', '-a', str(app), str(nested_alpha))
        reused = selected(alpha.name, ready=True)
        for repo in fixtures:
            one(reused, 'tab', repo.name)
        assert not control(reused, 'tab', nested_alpha.name)
        click('listItem', '1 change')
        reused = snapshot()
        assert 'Alpha fixture draft' in reused and 'Beta fixture draft' not in reused
        (artifacts / 'nested-checkout-reused.txt').write_text(reused)
        click('button', f'Close {gamma}')
        until(lambda: not control(snapshot(), 'tab', gamma.name), 'Clean fixture tab did not close')
        click('tab', beta.name)
        selected(beta.name, ready=True)
        click('listItem', '1 change')
        assert 'Beta fixture draft' in snapshot()
        auto('screenshot', artifacts / 'tabs.png')
        auto('menu', app.stem, f'Quit {app.stem}')
        process.wait(timeout=15)
        open_app(2)
        restored = selected(beta.name, ready=True)
        assert restored.index(one(restored, 'tab', alpha.name)) < restored.index(one(restored, 'tab', beta.name))
        assert not control(restored, 'tab', 'Welcome')
        (artifacts / 'restored.txt').write_text(restored)
        auto('screenshot', artifacts / 'restored.png')
        for repo in (beta, alpha):
            if repo == alpha:
                click('tab', alpha.name)
            selected(repo.name, ready=True)
            click('listItem', '1 change')
            tree = snapshot()
            for name in ('Alpha', 'Beta'):
                for part in ('draft', 'body'):
                    assert (f'{name} fixture {part}' in tree) == repo.name.startswith(name), f'{name} {part} did not restore in its own tab'
        auto('menu', app.stem, f'Quit {app.stem}')
        launches[-1].wait(timeout=15)
    finally:
        # Match the unique scratch executable, never any installed application.
        listing = subprocess.check_output(['/bin/ps', '-axo', 'pid=,comm='], text=True)
        for line in listing.splitlines():
            fields = line.strip().split(None, 1)
            if len(fields) == 2 and fields[1] == str(executable):
                try:
                    os.kill(int(fields[0]), signal.SIGTERM)
                except ProcessLookupError:
                    pass
        for process in launches:
            if process.poll() is None:
                process.wait(timeout=10)
        assert before == [fingerprint(repo) for repo in fixtures], 'Browsing changed fixture bytes'


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('bundle', type=Path)
    parser.add_argument('driver', type=Path)
    args = parser.parse_args()
    bundle = args.bundle.resolve(strict=True)
    driver = args.driver.resolve(strict=True)
    artifacts = Path(tempfile.mkdtemp(prefix='repository-tabs-', dir=bundle.parent)).resolve()
    print(f'Repository tab evidence: {artifacts}', flush=True)
    with tempfile.TemporaryDirectory(prefix='sg-tabs-', dir='/tmp') as directory:
        run(bundle, driver, artifacts, Path(directory).resolve())
    print('PASS New Tab, folder delivery to Welcome and occupied workbench, canonical tab reuse, independent titles and descriptions across ordered restart, and unchanged fixtures.', flush=True)
