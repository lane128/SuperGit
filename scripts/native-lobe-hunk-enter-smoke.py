#!/usr/bin/env python3
"""Native focused hunk-heading Enter acceptance; run only after the new build is ready."""
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
parser.add_argument('--project', type=pathlib.Path, default=pathlib.Path(__file__).resolve().parents[1])
parser.add_argument('--artifacts-parent', type=pathlib.Path)
args = parser.parse_args()
project = args.project.resolve()
artifacts_parent = (args.artifacts_parent or project / 'build').resolve()
artifacts_parent.mkdir(parents=True, exist_ok=True)
root = pathlib.Path(tempfile.mkdtemp(prefix='hunk-enter-', dir=artifacts_parent))
print('ARTIFACTS', root, flush=True)
(root / 'script.py').write_bytes(pathlib.Path(__file__).read_bytes())
binary, = (project / 'build').glob('supergit-lobe*e2e')
automation = project / 'build/mygo-automation'
(root / 'binary.json').write_text(json.dumps({'sha256': hashlib.sha256(binary.read_bytes()).hexdigest(), 'mtime': binary.stat().st_mtime}, indent=2)+'\n')
repo, home = root/'workspace', root/'home'
file, protected = repo/'hunks.txt', repo/'protected.bin'
base_lines = [f'context line {i:02d}\n'.encode() for i in range(1,41)]
base_bytes = b''.join(base_lines)
partial_lines = base_lines.copy()
partial_lines[3] = b'TARGET replacement line 04\n'
partial_bytes = b''.join(partial_lines)
working_lines = partial_lines.copy()
working_lines[29] = b'OTHER replacement line 30\n'
working_bytes = b''.join(working_lines)
protected_index = b'protected\x00baseline\xff\n'
protected_working = b'protected\x00external dirty sentinel\xfe\n'
env = {k:v for k,v in os.environ.items() if not k.startswith('GIT_')}
env.update(HOME=str(home), GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM='1', GIT_TERMINAL_PROMPT='0', GIT_OPTIONAL_LOCKS='0')
bootstrap = True
app = None
results = []
step = 'bootstrap'
session = 'hunk-enter-'+str(os.getpid())

def git(*items):
    if not bootstrap:
        assert items[0] in ('rev-parse','show','diff','ls-files','status','for-each-ref','symbolic-ref'), 'CLI write bypassed native acceptance'
    p = subprocess.run(['git','-C',str(repo),*items], env=env, capture_output=True, timeout=30)
    with (root/'git.log').open('ab') as f:f.write(repr(items).encode()+b'\n'+p.stdout+p.stderr+b'\n')
    assert p.returncode == 0, (items,p.stderr)
    return p.stdout

def same(actual, expected, label):
    assert actual == expected, label+': '+repr(actual)+' != '+repr(expected)

def note(text):
    results.append(text)
    (root/'results.txt').write_text('\n'.join(results)+'\n')
    print(text,flush=True)

def state():
    return {'head':git('rev-parse','HEAD').decode().strip(),'refs':git('for-each-ref','--format=%(refname) %(objectname)').decode(),
            'index':git('ls-files','--stage','-z').hex(),'status':git('status','--porcelain=v1','-z').hex(),
            'working':file.read_bytes().hex(),'protected_working':protected.read_bytes().hex(),
            'index_blob':git('show',':hunks.txt').hex(),'protected_index':git('show',':protected.bin').hex()}

def auto(*items):
    p=subprocess.run([str(automation),*items,'--session',session],env=env,cwd=project,capture_output=True,text=True,timeout=40)
    with (root/'automation.log').open('a') as f:f.write(repr(items)+'\n'+p.stdout+p.stderr+'\n')
    assert p.returncode == 0,(items,p.stdout,p.stderr)
    return p.stdout

def snapshot():return auto('snapshot','--full')

def entries(text=None):
    rows=[]
    for line in (snapshot() if text is None else text).splitlines():
        m=re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)',line)
        if m:rows.append((m[1],m[2],json.loads(m[3]) if m[3] else '',m[4]))
    return rows

def control(role,label,description=None):
    end=time.monotonic()+25
    while True:
        rows=[e for e in entries() if e[1]==role and e[2]==label and 'disabled' not in e[3] and
              (description is None or 'description='+json.dumps(description,ensure_ascii=False) in e[3])]
        if len(rows)==1:return rows[0]
        assert len(rows)<2,('Ambiguous control',rows)
        assert app.poll() is None and time.monotonic()<end,('Missing control',role,label)
        time.sleep(.05)

def ready():control('menuButton','Sync')

def press(role,label,description=None):auto('press',control(role,label,description)[0],'--settle')

def wait(text):auto('wait','--text',text,'--timeout','25s','--settle')

def evidence(name):
    text=snapshot()
    (root/(name+'.snapshot.txt')).write_text(text)
    auto('screenshot',str(root/(name+'.png')))
    (root/(name+'.git.json')).write_text(json.dumps(state(),indent=2)+'\n')
    return text

def invariants(expected_index):
    same(git('rev-parse','HEAD'),initial_head,'Hunk action moved HEAD')
    same(git('for-each-ref','--format=%(refname) %(objectname)'),initial_refs,'Hunk action moved refs')
    same(git('show',':hunks.txt'),expected_index,'Wrong Index hunk bytes')
    same(file.read_bytes(),working_bytes,'Hunk action changed worktree bytes')
    same(git('show',':protected.bin'),protected_index,'Hunk action changed unrelated Index blob')
    same(protected.read_bytes(),protected_working,'Hunk action changed unrelated worktree bytes')

def focused_enter(staged, expected):
    press('listItem','hunks.txt','Staged Files' if staged else 'Modified Files')
    wait('TARGET replacement line 04')
    ready()
    # Move away from patch actions, then focus the hunk TITLE, never its hover button.
    auto('pointer','move','10','10','--settle')
    header=control('button',target_header)
    assert header[2].startswith('@@ '), 'Target must be the hunk title'
    auto('act',header[0],'scrollIntoView','--settle')
    auto('focus',control('button',target_header)[0],'--settle')
    selected=evidence(step+'-heading-focused')
    assert any(e[1]=='button' and e[2]==target_header and 'focused' in e[3] for e in entries(selected)), 'Hunk title did not receive focus'
    auto('key','Enter','--settle')
    end=time.monotonic()+25
    while git('show',':hunks.txt')!=expected:
        assert app.poll() is None and time.monotonic()<end,'Focused hunk Enter did not produce exact expected Index'
        time.sleep(.05)
    ready()
    invariants(expected)
    evidence(step)

failed=False
try:
    subprocess.run(['git','init','-b','main',str(repo)],env=env,check=True,capture_output=True)
    for key,value in [('user.name','Hunk Fixture'),('user.email','hunk@example.invalid'),('commit.gpgsign','false')]:git('config',key,value)
    file.write_bytes(base_bytes)
    protected.write_bytes(protected_index)
    git('add','--all')
    git('commit','--no-gpg-sign','-m','Hunk fixture base')
    initial_head=git('rev-parse','HEAD')
    initial_refs=git('for-each-ref','--format=%(refname) %(objectname)')
    initial_index=git('ls-files','--stage','-z')
    bootstrap=False
    file.write_bytes(working_bytes)
    protected.write_bytes(protected_working)
    diff=git('diff','--no-ext-diff','--unified=3','--','hunks.txt')
    (root/'fixture.diff').write_bytes(diff)
    headers=[line.decode() for line in diff.splitlines() if line.startswith(b'@@ ')]
    assert len(headers)==2,('Fixture requires two separate hunks',headers)
    target_header=headers[0]
    assert b'-context line 04\n+TARGET replacement line 04' in diff and b'-context line 30\n+OTHER replacement line 30' in diff,'Both hunks must be del/add replacements'
    settings=home/'Library/Application Support/supergit/settings.json'
    settings.parent.mkdir(parents=True)
    settings.write_text(json.dumps({'language':'en','theme':'light','autoFetch':False}))
    socketdir=pathlib.Path(tempfile.mkdtemp(prefix='sghunk-',dir='/tmp'))
    socketdir.chmod(0o700)
    socket=socketdir/'app.sock'
    env.update(MYGO_AUTOMATION=str(socket),SUPERGIT_WINDOW_SIZE='1320x800')
    log=(root/'app.log').open('w')
    app=subprocess.Popen([str(binary),str(repo)],env=env,cwd=project,stdout=log,stderr=subprocess.STDOUT)
    (root/'pid.txt').write_text(str(app.pid))
    end=time.monotonic()+20
    while not socket.exists():
        assert app.poll() is None and time.monotonic()<end,'Native automation unavailable'
        time.sleep(.05)
    auto('open','--socket',str(socket))
    wait('Commit summary (required)')
    ready()
    press('tab','Changes')
    step='01-target-hunk-enter-stage'
    invariants(base_bytes)
    focused_enter(False,partial_bytes)
    note('PASS focused unstaged hunk TITLE+Enter stages target replacement only; other hunk/worktree/unrelated file preserved')
    step='02-staged-hunk-enter-unstage'
    focused_enter(True,base_bytes)
    same(git('ls-files','--stage','-z'),initial_index,'Hunk unstage did not restore exact initial Index entries')
    note('PASS focused staged hunk TITLE+Enter restores exact initial Index; both replacements remain in worktree')
    step='03-restage-target-hunk-enter'
    focused_enter(False,partial_bytes)
    note('PASS target hunk can be staged again using focused heading Enter')
    step='04-ui-partial-commit'
    subject='Focused hunk Enter commit'
    body='Commit only target replacement\n\nKeep other replacement and unrelated binary dirty'
    auto('fill',control('textField','Commit summary (required)')[0],subject,'--settle')
    auto('fill',control('textField','Description (optional)')[0],body,'--settle')
    before_index=git('ls-files','--stage','-z')
    press('button','Commit')
    end=time.monotonic()+25
    while git('rev-parse','HEAD')==initial_head:
        assert app.poll() is None and time.monotonic()<end,'UI Commit did not advance HEAD'
        time.sleep(.05)
    ready()
    same(git('show','-s','--format=%P','HEAD').strip(),initial_head.strip(),'Wrong commit parent')
    same(git('show','HEAD:hunks.txt'),partial_bytes,'UI commit included other hunk or lost target')
    same(git('show','HEAD:protected.bin'),protected_index,'Commit included unrelated dirty binary')
    same(git('show','-s','--format=%B','HEAD').rstrip(b'\n'),(subject+'\n\n'+body).encode(),'Full commit message mismatch')
    same(git('show','-s','--format=%an <%ae>','HEAD').strip(),b'Hunk Fixture <hunk@example.invalid>','Commit identity mismatch')
    same(git('ls-files','--stage','-z'),before_index,'Commit changed Index entries')
    same(file.read_bytes(),working_bytes,'Commit changed file worktree bytes')
    same(protected.read_bytes(),protected_working,'Commit changed unrelated working bytes')
    same(git('symbolic-ref','HEAD').strip(),b'refs/heads/main','Commit changed active branch')
    same(git('status','--porcelain=v1','-z'),b' M hunks.txt\x00 M protected.bin\x00','Expected exact post-commit status mismatch')
    same(git('diff','--cached','--name-only'),b'','Commit left staged changes')
    same(git('diff','--name-only').decode().splitlines(),['hunks.txt','protected.bin'],'Expected unstaged paths mismatch')
    evidence(step)
    note('PASS UI commit tree contains target replacement only; expected parent/full message/author and exact Index/worktree/unrelated bytes preserved')
except BaseException:
    failed=True
    note('FAIL '+step+'\n'+traceback.format_exc())
    if app is not None and app.poll() is None:
        try:evidence('failure-'+step)
        except Exception as capture_error:(root/'capture-error.txt').write_text(repr(capture_error))
finally:
    if app is not None:
        app.terminate()
        try:app.wait(timeout=10)
        except subprocess.TimeoutExpired:
            app.kill();app.wait(timeout=5)
        log.close()
        note('PROCESS_STOPPED='+str(app.returncode))
if failed:raise SystemExit(1)
