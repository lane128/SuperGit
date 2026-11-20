#!/usr/bin/env python3
"""Verify parent/submodule/worktree navigation with owned disposable fixtures."""
import argparse
import sys
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
root = pathlib.Path(tempfile.mkdtemp(prefix='root-navigation-', dir=artifacts_parent))
print('ARTIFACTS', root, flush=True)
(root / 'script.py').write_bytes(pathlib.Path(__file__).read_bytes())
parent, source, linked, home = [root / p for p in ('parent-root', 'child-source', 'linked-root', 'home')]
child = parent / 'module-root'
env = {k:v for k,v in os.environ.items() if not k.startswith('GIT_')}
env.update(HOME=str(home), GIT_CONFIG_GLOBAL='/dev/null', GIT_CONFIG_NOSYSTEM='1', GIT_TERMINAL_PROMPT='0', GIT_ALLOW_PROTOCOL='file', GIT_OPTIONAL_LOCKS='0')
bootstrap = True
results = []
app = None
failed = False

def git(repo, *args):
    assert repo.resolve().is_relative_to(root)
    if not bootstrap:
        assert args[0] in ('rev-parse','symbolic-ref','show-ref','ls-files','status','show','worktree'), args
        if args[0] == 'worktree': assert args[1:] == ('list','--porcelain')
    p = subprocess.run(['git','-C',str(repo),*args], env=env, capture_output=True, timeout=30)
    with (root / 'git.log').open('ab') as f: f.write(repr((str(repo),args)).encode()+b'\n'+p.stdout+p.stderr+b'\n')
    assert p.returncode == 0, (args,p.stderr)
    return p.stdout

def setup(repo, label):
    subprocess.run(['git','init','-b','main',str(repo)],env=env,check=True,capture_output=True)
    git(repo,'config','user.name','Navigation Fixture')
    git(repo,'config','user.email','navigation@example.invalid')
    git(repo,'config','commit.gpgsign','false')
    (repo / (label+'.txt')).write_text(label+' baseline\n')
    git(repo,'add','--all')
    git(repo,'commit','-m','Unique '+label+' history')

def fingerprint():
    data={}
    for name,repo in [('parent',parent),('child',child),('linked',linked)]:
        files={str(p.relative_to(repo)):p.read_bytes().hex() for p in repo.rglob('*') if p.is_file() and '.git' not in p.relative_to(repo).parts}
        data[name]={'root':str(repo),'head':git(repo,'rev-parse','HEAD').decode().strip(),'refs':git(repo,'show-ref').decode(),
                    'index':git(repo,'ls-files','--stage','-z').hex(),'status':git(repo,'status','--porcelain=v1','-z').hex(),
                    'symbolic':git(repo,'symbolic-ref','HEAD').decode(),'files':files}
    return data

def note(text):
    results.append(text)
    (root/'results.txt').write_text('\n'.join(results)+'\n')
    print(text,flush=True)

try:
    setup(source,'CHILD')
    setup(parent,'PARENT')
    git(parent,'submodule','add','--name','module-root',str(source),'module-root')
    git(parent,'commit','-am','Unique PARENT module registered')
    git(child,'config','user.name','Navigation Fixture')
    git(child,'config','user.email','navigation@example.invalid')
    git(parent,'worktree','add','-b','linked-branch',str(linked))
    (linked/'LINKED.txt').write_text('LINKED committed content\n')
    git(linked,'add','LINKED.txt')
    git(linked,'commit','-m','Unique LINKED history')
    for repo,label in [(parent,'PARENT'),(child,'CHILD'),(linked,'LINKED')]:
        (repo/(label+'.txt')).write_text(label+' dirty navigation sentinel\n')
    bootstrap=False
    before=fingerprint()
    (root/'before.git.json').write_text(json.dumps(before,indent=2)+'\n')
    settings=home/'Library/Application Support/supergit/settings.json'
    settings.parent.mkdir(parents=True)
    settings.write_text(json.dumps({'language':'en','theme':'light','autoFetch':False}))
    binary,=(project/'build').glob('supergit-lobe*e2e')
    (root/'binary.json').write_text(json.dumps({'sha256':hashlib.sha256(binary.read_bytes()).hexdigest(),'mtime':binary.stat().st_mtime},indent=2)+'\n')
    socketdir=pathlib.Path(tempfile.mkdtemp(prefix='sgroots-',dir='/tmp'))
    socketdir.chmod(0o700)
    socket=socketdir/'app.sock'
    env.update(MYGO_AUTOMATION=str(socket),SUPERGIT_WINDOW_SIZE='1320x800')
    log=(root/'app.log').open('w')
    app=subprocess.Popen([str(binary),str(parent)],env=env,cwd=project,stdout=log,stderr=subprocess.STDOUT)
    session='root-navigation-'+str(app.pid)
    (root/'pid.txt').write_text(str(app.pid))
    end=time.monotonic()+20
    while not socket.exists():
        assert app.poll() is None and time.monotonic()<end,'No native automation'
        time.sleep(.05)
    def auto(*args):
        p=subprocess.run([str(project/'build/mygo-automation'),*args,'--session',session],env=env,cwd=project,capture_output=True,text=True,timeout=40)
        with (root/'automation.log').open('a') as f:f.write(repr(args)+'\n'+p.stdout+p.stderr+'\n')
        assert p.returncode==0,(args,p.stdout,p.stderr)
        return p.stdout
    def snap():return auto('snapshot','--full')
    def entries(text=None):
        rows=[]
        for line in (snap() if text is None else text).splitlines():
            m=re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)',line)
            if m:rows.append((m[1],m[2],json.loads(m[3]) if m[3] else '',m[4]))
        return rows
    def press(role,label):
        matches=[e for e in entries() if e[1]==role and e[2]==label and 'disabled' not in e[3]]
        assert len(matches)==1,(role,label,matches)
        auto('press',matches[0][0],'--settle')
    def wait(text):auto('wait','--text',text,'--timeout','25s','--settle')
    def ready():
        end=time.monotonic()+25
        while not any(e[1]=="menuButton" and e[2]=="Sync" and 'disabled' not in e[3] for e in entries()):
            assert time.monotonic()<end,'Repository did not finish loading'
            time.sleep(.05)
    def evidence(name, text=None):
        text=snap() if text is None else text
        (root/(name+'.snapshot.txt')).write_text(text)
        auto('screenshot',str(root/(name+'.png')))
        current=fingerprint()
        (root/(name+'.git.json')).write_text(json.dumps(current,indent=2)+'\n')
        assert current==before,'Navigation changed Git/data fingerprint'
        return text
    def group(label):
        item=next(e for e in entries() if e[1]=='disclosure' and e[2]==label)
        if 'expanded' not in item[3]:auto('press',item[0],'--settle')
    def verify(repo,label,name):
        ready()
        end=time.monotonic()+25
        while not any(e[1]=='button' and e[2]==repo.name and str(repo) in e[3] and 'bounds=46,' in e[3] for e in entries()):
            assert time.monotonic()<end,'Visible repository root did not switch'
            time.sleep(.05)
        ready()
        press('button','History')
        wait('Unique '+label)
        auto('key','cmd+0','--settle')
        sha=git(repo,'rev-parse','HEAD').decode().strip()
        wait(sha)
        wait('Unique '+label)
        history=evidence(name+'-history')
        assert any(e[1]=='button' and e[2]==repo.name and str(repo) in e[3] and 'bounds=46,' in e[3] for e in entries(history)),'Wrong visible root'
        press('tab','Changes')
        press('listItem',label+'.txt')
        wait('+'+label+' dirty navigation sentinel')
        # A watcher refresh can begin after the earlier repository-ready check.
        # Observe the real action state instead of treating --settle as completion.
        end=time.monotonic()+25
        while True:
            changes=snap()
            rows=entries(changes)
            if (any(e[1]=="menuButton" and e[2]=="Sync" and 'disabled' not in e[3] for e in rows)
                and any(e[1]=='button' and e[2]=='Stage file' and 'disabled' not in e[3] for e in rows)):
                break
            assert time.monotonic()<end,'Current file action did not become available'
            time.sleep(.05)
        changes=evidence(name+'-changes', changes)
        assert any(e[1]=='button' and e[2]=='Stage file' and 'disabled' not in e[3] for e in entries(changes)),'Current file action target missing'
        for other in ('PARENT','CHILD','LINKED'):
            if other!=label:assert '+'+other+' dirty navigation sentinel' not in changes,'Other root diff leaked'
        note('PASS '+name+' root='+str(repo)+' HEAD='+sha+' History/Files/diff/actions match; all three fingerprints unchanged')
    auto('open','--socket',str(socket))
    wait('Commit summary (required)')
    def title(repo):
        candidates=[e for e in entries() if e[1]=='button' and e[2]==repo.name and 'bounds=46,' in e[3]]
        assert len(candidates)==1,('Missing repository title',repo,candidates)
        return candidates[0][0]
    def draft(repo, value):
        field=next(e for e in entries() if e[1]=='textField' and e[2]=='Commit summary (required)')
        auto('fill',field[0],value,'--settle')
    def assert_reset():
        fields=[e for e in entries() if e[1]=='textField' and e[2] in ('Commit summary (required)','Description (optional)')]
        assert len(fields)==2 and all('value=' not in e[3] for e in fields),'Root switch retained another root draft'
    def return_parent(current,name):
        auto('press',title(current),'--settle')
        wait(str(parent))
        evidence(name+'-recent-menu')
        press('menuItem',str(parent))
        verify(parent,'PARENT',name)
        assert_reset()
    verify(parent,'PARENT','00-parent')
    for cycle in (1,2):
        prefix='cycle-'+str(cycle)
        draft(parent,'PARENT navigation draft '+str(cycle))
        group('Submodules')
        press('button','module-root')
        verify(child,'CHILD',prefix+'-child')
        assert_reset()
        draft(child,'CHILD navigation draft '+str(cycle))
        return_parent(child,prefix+'-child-return-parent')
        draft(parent,'PARENT linked draft '+str(cycle))
        group('Worktrees')
        if cycle == 1:
            pre=evidence(prefix+'-linked-before-semantic')
            rows=[e for e in entries(pre) if e[1]=='button' and e[2]=='linked-root' and str(linked) in e[3]]
            assert len(rows)==1,('Linked row target ambiguous',rows)
            auto('press',rows[0][0],'--settle')
            end=time.monotonic()+5
            while time.monotonic()<end and not any(e[1]=='button' and e[2]==linked.name and 'bounds=46,' in e[3] for e in entries()):
                time.sleep(.05)
            post=evidence(prefix+'-linked-after-semantic')
            switched=any(e[1]=='button' and e[2]==linked.name and 'bounds=46,' in e[3] for e in entries(post))
            if not switched:
                rows=[e for e in entries(post) if e[1]=='button' and e[2]=='linked-root' and str(linked) in e[3]]
                assert len(rows)==1,('Linked mouse target ambiguous',rows)
                bounds=re.search(r'bounds=([\d.]+),([\d.]+),([\d.]+)x([\d.]+)',rows[0][3])
                x,y,w,h=map(float,bounds.groups())
                note('OBSERVATION semantic linked press had no root change; Sync/dialog/tooltip surfaces='+repr([e for e in entries(post) if e[1] in ('dialog','alertDialog','tooltip') or e[2]=='Sync']))
                auto('click',str(x+w/2),str(y+h/2),'--settle')
                evidence(prefix+'-linked-after-coordinate')
                note('OBSERVATION clicked exact linked row center '+str((x+w/2,y+h/2)))
        else:
            rows=[e for e in entries() if e[1]=='button' and e[2]=='linked-root' and str(linked) in e[3]]
            assert len(rows)==1
            bounds=re.search(r'bounds=([\d.]+),([\d.]+),([\d.]+)x([\d.]+)',rows[0][3])
            x,y,w,h=map(float,bounds.groups())
            auto('click',str(x+w/2),str(y+h/2),'--settle')
        verify(linked,'LINKED',prefix+'-linked')
        assert_reset()
        draft(linked,'LINKED navigation draft '+str(cycle))
        return_parent(linked,prefix+'-linked-return-parent')
        note('PASS '+prefix+' complete parent→child→parent→linked→parent via sidebar/title-path menus; draft reset observed on every root switch')
except BaseException:
    failed = True
    note('FAIL '+traceback.format_exc())
    if app is not None and app.poll() is None:
        try:evidence('failure')
        except Exception:pass
finally:
    if app is not None:
        app.terminate()
        try:app.wait(timeout=10)
        except subprocess.TimeoutExpired:
            app.kill();app.wait(timeout=5)
        log.close()
        note('PROCESS_STOPPED='+str(app.returncode))

sys.exit(1 if failed else 0)
