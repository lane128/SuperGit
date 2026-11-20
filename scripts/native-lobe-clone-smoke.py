#!/usr/bin/env python3
"""Native Clone user flow against new local-only fixtures; retain all evidence."""
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

parser=argparse.ArgumentParser(description=__doc__)
parser.add_argument('--project',type=pathlib.Path,default=pathlib.Path(__file__).resolve().parents[1])
parser.add_argument('--artifacts-parent',type=pathlib.Path)
args=parser.parse_args()
project=args.project.resolve()
root=pathlib.Path(tempfile.mkdtemp(prefix='clone-native-',dir=args.artifacts_parent or project/'build'))
print('ARTIFACTS',root,flush=True)
(root/'script.py').write_bytes(pathlib.Path(__file__).read_bytes())
binary,=(project/'build').glob('supergit-lobe*e2e')
automation=project/'build/mygo-automation'
(root/'binary.json').write_text(json.dumps({'sha256':hashlib.sha256(binary.read_bytes()).hexdigest(),'mtime':binary.stat().st_mtime},indent=2)+'\n')
active,source,subsource,remote,home=[root/x for x in ('old-workspace','source','subsource','remote.git','home')]
destination=root/'克隆 space checkout'
existing=root/'existing destination'
failed_destination=root/'failed attempt'
cancelled_destination=root/'cancelled destination'
source_field='Source URL or local path'
destination_field='Destination directory (absolute path)'
env={k:v for k,v in os.environ.items() if not k.startswith('GIT_')}
env.update(HOME=str(home),GIT_CONFIG_GLOBAL=os.devnull,GIT_CONFIG_NOSYSTEM='1',GIT_TERMINAL_PROMPT='0',GIT_OPTIONAL_LOCKS='0',GIT_ALLOW_PROTOCOL='file')
bootstrap=True
app=None
log=None
results=[]
step='bootstrap'
readme='文件 space.txt'
source_bytes=b'Owned local clone content\n'
source_binary=b'clone binary\x00\xff\n'
old_initial=b'old baseline\n'
old_index=b'old staged sentinel\n'
old_working=b'old working sentinel\n'

def git(repo,*items):
    assert repo.resolve().is_relative_to(root),'Git target escaped fixtures'
    if not bootstrap:
        assert items[0] in ('rev-parse','show-ref','show','ls-files','status','symbolic-ref','for-each-ref','remote','config','ls-tree'), 'CLI write bypassed UI'
        if items[0]=='config':assert items[1:2]==('--get',),'Post-bootstrap config write'
        if items[0]=='remote':assert items[1:2]==('get-url',),'Post-bootstrap remote write'
    p=subprocess.run(['git','-C',str(repo),*items],env=env,capture_output=True,timeout=30)
    with (root/'git.log').open('ab') as f:f.write(repr((str(repo),items)).encode()+b'\n'+p.stdout+p.stderr+b'\n')
    assert p.returncode==0,(repo,items,p.stderr)
    return p.stdout

def setup(repo,name):
    subprocess.run(['git','init','-b','main',str(repo)],env=env,check=True,capture_output=True)
    for k,v in [('user.name','Clone Fixture'),('user.email','clone@example.invalid'),('commit.gpgsign','false')]:git(repo,'config',k,v)
    (repo/'base.txt').write_text(name+' baseline\n')
    git(repo,'add','--all');git(repo,'commit','-m',name+' base')

def files(repo):
    return {str(p.relative_to(repo)):p.read_bytes().hex() for p in repo.rglob('*') if p.is_file() and '.git' not in p.relative_to(repo).parts}

def state(repo,bare=False):
    data={'head':git(repo,'rev-parse','HEAD').decode().strip(),'refs':git(repo,'show-ref').decode(),'symbolic':git(repo,'symbolic-ref','HEAD').decode(),'files':files(repo)}
    if not bare:data.update(index=git(repo,'ls-files','--stage','-z').hex(),status=git(repo,'status','--porcelain=v1','-z').hex())
    return data

def originals():return {'active':state(active),'source':state(source),'subsource':state(subsource),'bare':state(remote,True),'existing':files(existing)}

def note(text):
    results.append(text);(root/'results.txt').write_text('\n'.join(results)+'\n');print(text,flush=True)

def auto(*items):
    p=subprocess.run([str(automation),*items,'--session',session],env=env,cwd=root,capture_output=True,text=True,timeout=40)
    with (root/'automation.log').open('a') as f:f.write(repr(items)+'\n'+p.stdout+p.stderr+'\n')
    assert p.returncode==0,(items,p.stdout,p.stderr)
    return p.stdout

def snap():return auto('snapshot','--full')

def entries(text=None):
    rows=[]
    for line in (snap() if text is None else text).splitlines():
        m=re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)',line)
        if m:rows.append((m[1],m[2],json.loads(m[3]) if m[3] else '',m[4]))
    return rows

def control(role,label,enabled=True):
    end=time.monotonic()+25
    while True:
        rows=[e for e in entries() if e[1]==role and e[2]==label and (not enabled or 'disabled' not in e[3])]
        if len(rows)==1:return rows[0]
        assert len(rows)<2,('Ambiguous control',role,label,rows)
        assert app.poll() is None and time.monotonic()<end,('Missing control',role,label)
        time.sleep(.05)

def press(role,label):auto('press',control(role,label)[0],'--settle')
def fill(label,value):auto('fill',control('textField',label)[0],value,'--settle')
def wait(text):auto('wait','--text',text,'--timeout','25s','--settle')
def ready():control('menuButton','Sync')

def evidence(name):
    text=snap();(root/(name+'.snapshot.txt')).write_text(text);auto('screenshot',str(root/(name+'.png')))
    current=originals();(root/(name+'.git.json')).write_text(json.dumps(current,indent=2)+'\n')
    assert current==before,'Clone flow changed source/original repository or existing destination'
    (root/(name+'.destinations.json')).write_text(json.dumps({str(p):{'exists':p.exists(),'files':files(p) if p.is_dir() else {}} for p in (destination,failed_destination,cancelled_destination)},indent=2)+'\n')
    return text

def root_ready(repo):
    end=time.monotonic()+25
    while not any(e[1]=='button' and e[2]==repo.name and 'bounds=46,' in e[3] and 'description='+json.dumps(str(repo),ensure_ascii=False)+' ' in e[3] for e in entries()):
        assert time.monotonic()<end,'Exact repository title root did not open'
        time.sleep(.05)
    ready()

def no_form():assert not any(e[1]=='textField' and e[2]==source_field for e in entries()),'Clone form remains open'

def open_clone(repo=None):
    if repo is None:press('button','Clone repository…')
    else:
        row=next(e for e in entries() if e[1]=='button' and e[2]==repo.name and 'bounds=46,' in e[3])
        auto('press',row[0],'--settle');press('menuItem','Clone repository…')
    wait('Clone repository');control('textField',source_field)

def stop():
    global app,log
    if app is None:return
    app.terminate()
    try:app.wait(timeout=10)
    except subprocess.TimeoutExpired:app.kill();app.wait(timeout=5)
    log.close();note('PROCESS_STOPPED='+str(app.returncode));app=None

def start(repo=None):
    global app,log,session
    socketdir=pathlib.Path(tempfile.mkdtemp(prefix='sgclone-',dir='/tmp'));socketdir.chmod(0o700);socket=socketdir/'app.sock'
    launch_env={**env,'MYGO_AUTOMATION':str(socket),'SUPERGIT_WINDOW_SIZE':'1100x720'}
    log=(root/('app-active.log' if repo else 'app-empty.log')).open('w')
    app=subprocess.Popen([str(binary)]+([str(repo)] if repo else []),env=launch_env,cwd=root,stdout=log,stderr=subprocess.STDOUT)
    session='clone-native-'+str(app.pid)
    (root/('pid-active.txt' if repo else 'pid-empty.txt')).write_text(str(app.pid))
    end=time.monotonic()+20
    while not socket.exists():
        assert app.poll() is None and time.monotonic()<end,'Native socket unavailable'
        time.sleep(.05)
    auto('open','--socket',str(socket))

failed=False
try:
    setup(active,'Old workspace');setup(source,'Clone source');setup(subsource,'Clone submodule')
    (active/'base.txt').write_bytes(old_index);git(active,'add','base.txt');(active/'base.txt').write_bytes(old_working)
    (source/readme).write_bytes(source_bytes);(source/'binary.bin').write_bytes(source_binary)
    git(source,'submodule','add',str(subsource),'child module')
    git(source,'add','--all');git(source,'commit','-m','Unique Clone source tip','-m','Full clone fixture body')
    git(source,'branch','feature/fixture');git(source,'tag','clone-fixture-tag')
    subprocess.run(['git','init','--bare','-b','main',str(remote)],env=env,check=True,capture_output=True)
    git(source,'remote','add','origin',str(remote));git(source,'push','origin','--all');git(source,'push','origin','--tags')
    existing.mkdir();(existing/'sentinel.bin').write_bytes(b'existing destination\x00keep\xff')
    settings=home/'Library/Application Support/supergit/settings.json';settings.parent.mkdir(parents=True)
    settings.write_text(json.dumps({'language':'en','theme':'light','autoFetch':False}))
    bootstrap=False
    before=originals();(root/'before.git.json').write_text(json.dumps(before,indent=2)+'\n')
    expected_head=git(source,'rev-parse','HEAD');expected_index=git(source,'ls-files','--stage','-z')
    step='01-empty-state-cancel'
    start();wait('Clone repository…');open_clone()
    assert 'disabled' in control('button','Clone',enabled=False)[3],'Blank Clone form allowed submit'
    fill(source_field,str(remote));fill(destination_field,str(cancelled_destination));evidence(step+'-filled')
    press('button','Cancel');no_form();assert not cancelled_destination.exists();evidence(step)
    note('PASS empty-state Clone entry/form/Cancel; no destination or original Git/data changes')
    stop()
    step='02-active-form-cancel'
    start(active);wait('Commit message');root_ready(active);press('tab','Changes')
    fill('Commit message', 'Keep old Clone draft' + '\n\n' + 'Keep old Clone description')
    open_clone(active);fill(source_field,str(remote));fill(destination_field,str(cancelled_destination));press('button','Cancel');no_form()
    text=evidence(step)
    assert 'value="Keep old Clone draft"' in text and 'value="Keep old Clone description"' in text,'Cancel changed original draft'
    assert not cancelled_destination.exists();root_ready(active)
    note('PASS title-menu Clone cancel preserves original root/staged+working bytes and full draft')
    step='03-existing-destination'
    open_clone(active);fill(source_field,str(remote));fill(destination_field,str(existing));press('button','Clone')
    wait('The destination exists or cannot be accessed.')
    evidence(step);control('textField',source_field);control('textField',destination_field)
    note('PASS existing destination refused in form; sentinel and all original fingerprints preserved, fields editable')
    step='04-missing-source-failure'
    missing=root/'does-not-exist.git'
    fill(source_field,str(missing));fill(destination_field,str(failed_destination));press('button','Clone')
    # The enabled Clone button identifies completion, not just the first rendered frame.
    wait("does not exist")
    control('button','Clone');control('textField',source_field);control('textField',destination_field)
    text=evidence(step)
    assert str(missing) in text and 'git clone:' in text,'Missing source error lost original Git detail'
    assert 'value='+json.dumps(str(missing),ensure_ascii=False) in text,'Source field lost failure input'
    assert 'value='+json.dumps(str(failed_destination),ensure_ascii=False) in text,'Destination field lost failure input'
    note('PASS real local-source Git failure remains editable with original detail; failed destination exists='+str(failed_destination.exists()))
    step='05-edit-and-retry-success'
    fill(source_field,str(remote));fill(destination_field,str(destination));press('button','Clone')
    root_ready(destination);no_form()
    assert (destination/'.git').is_dir(),'UI did not create clone repository'
    assert git(destination,'rev-parse','HEAD')==expected_head,'Wrong cloned HEAD'
    assert git(destination,'ls-files','--stage','-z')==expected_index,'Clone Index differs from source tree'
    assert git(destination,'status','--porcelain=v1','-z')==b'','Cloned worktree not clean'
    assert git(destination,'ls-tree','-r','HEAD')==git(source,'ls-tree','-r','HEAD'),'Cloned tree differs'
    assert (destination/readme).read_bytes()==source_bytes and (destination/'binary.bin').read_bytes()==source_binary,'Cloned file bytes differ'
    for items in [('remote','get-url','--all','origin'),('remote','get-url','--push','--all','origin')]:
        assert git(destination,*items).decode().splitlines()==[str(remote)],'Clone remote escaped owned local source'
    assert git(destination,'symbolic-ref','HEAD').strip()==b'refs/heads/main','Wrong checkout branch'
    assert git(destination,'config','--get','branch.main.remote').strip()==b'origin'
    assert git(destination,'config','--get','branch.main.merge').strip()==b'refs/heads/main'
    assert git(destination,'rev-parse','refs/remotes/origin/main')==expected_head
    assert git(destination,'rev-parse','refs/remotes/origin/feature/fixture')==expected_head
    assert git(destination,'rev-parse','refs/tags/clone-fixture-tag')==expected_head
    assert not (destination/'.git/modules').exists() and not (destination/'child module/.git').exists(),'Clone unexpectedly initialized submodule'
    press('tab','Changes');text=evidence(step+'-clean-reset')
    assert all('value=' not in e[3] for e in entries(text) if e[1]=='textField' and e[2] == 'Commit message'),'Successful Clone kept old draft'
    press('button','History');wait('Unique Clone source tip');auto('key','cmd+0','--settle');wait(expected_head.decode().strip());wait('Full clone fixture body')
    press('listItem',readme);wait('+Owned local clone content');evidence(step+'-history-files')
    (root/'clone-final.git.json').write_text(json.dumps(state(destination),indent=2)+'\n')
    note('PASS UI retry clones exact local source tree/Index/refs/upstream and bytes, automatically opens correct root/HEAD/History/Files; clean state, draft reset, submodule not initialized, all originals preserved')
    step='06-return-old-root'
    title=next(e for e in entries() if e[1]=='button' and e[2]==destination.name and 'bounds=46,' in e[3])
    auto('press',title[0],'--settle');press('menuItem',str(active));root_ready(active);text=evidence(step)
    assert 'Full clone fixture body' not in text,'Previous repository message leaked into returned repository'
    note('PASS cloned repository enters session menu; return opens exact original root with original dirty Index/worktree intact and no stale clone message')
except BaseException:
    failed=True;note('FAIL '+step+'\n'+traceback.format_exc())
    if app is not None and app.poll() is None:
        try:evidence('failure-'+step)
        except Exception as error:(root/'capture-error.txt').write_text(repr(error))
finally:stop()
if failed:raise SystemExit(1)
