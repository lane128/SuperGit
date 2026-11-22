#!/usr/bin/env python3
"""Native New Repository to zero-parent first UI commit in owned fixtures only."""
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

script_root=pathlib.Path(__file__).resolve().parents[1]
parser=argparse.ArgumentParser(description=__doc__)
parser.add_argument('--project',type=pathlib.Path,default=script_root)
parser.add_argument('--artifacts-parent',type=pathlib.Path)
args=parser.parse_args()
project=args.project.resolve()
root=pathlib.Path(tempfile.mkdtemp(prefix='new-repository-native-',dir=(args.artifacts_parent or project/'build').resolve()))
print('ARTIFACTS',root,flush=True)
(root/'script.py').write_bytes(pathlib.Path(__file__).read_bytes())
binary,=(project/'build').glob('supergit-lobe*e2e')
automation=project/'build/mygo-automation'
(root/'binary.json').write_text(json.dumps({'sha256':hashlib.sha256(binary.read_bytes()).hexdigest(),'mtime':binary.stat().st_mtime},indent=2)+'\n')
active,home,existing=[root/x for x in ('old-workspace','home','existing destination')]
destination=root/'新建 space repository'
cancelled=root/'cancelled destination'
field='Destination directory (absolute path)'
text_name='文件 space.txt'
text_bytes=b'First owned native repository commit\n'
binary_bytes=b'first binary\x00\xff\n'
env={k:v for k,v in os.environ.items() if not k.startswith('GIT_')}
env.update(HOME=str(home),GIT_CONFIG_GLOBAL=str(home/'.gitconfig'),GIT_CONFIG_NOSYSTEM='1',GIT_TERMINAL_PROMPT='0',GIT_OPTIONAL_LOCKS='0',GIT_ALLOW_PROTOCOL='file')
bootstrap=True
app=None
log=None
results=[]
step='bootstrap'

def git(repo,*items,check=True):
    assert repo.resolve().is_relative_to(root),'Git target escaped owned fixtures'
    if not bootstrap:
        assert items[0] in ('rev-parse','symbolic-ref','for-each-ref','ls-files','status','show','ls-tree','rev-list','diff','remote'), 'CLI writes cannot replace UI acceptance'
        if items[0]=='remote':assert len(items)==1,'Post-bootstrap remote write'
    p=subprocess.run(['git','-C',str(repo),*items],env=env,capture_output=True,timeout=30)
    with (root/'git.log').open('ab') as f:f.write(repr((str(repo),items)).encode()+b'\n'+p.stdout+p.stderr+b'\n')
    if not check:return p
    assert p.returncode==0,(repo,items,p.stderr)
    return p.stdout

def files(repo):return {str(p.relative_to(repo)):p.read_bytes().hex() for p in repo.rglob('*') if p.is_file() and '.git' not in p.relative_to(repo).parts}

def state(repo):
    head=git(repo,'rev-parse','--verify','HEAD',check=False)
    return {'root':str(repo),'head':head.stdout.decode().strip() if head.returncode==0 else None,
            'symbolic':git(repo,'symbolic-ref','HEAD').decode(),'refs':git(repo,'for-each-ref','--format=%(refname) %(objectname)').decode(),
            'index':git(repo,'ls-files','--stage','-z').hex(),'status':git(repo,'status','--porcelain=v1','-z').hex(),'files':files(repo)}

def originals():return {'active':state(active),'existing':files(existing)}

def note(text):
    results.append(text);(root/'results.txt').write_text('\n'.join(results)+'\n');print(text,flush=True)

def same(a,b,label):assert a==b,label+': '+repr(a)+' != '+repr(b)

def auto(*items):
    started=time.monotonic()
    p=subprocess.run([str(automation),*items,'--session',session],env=env,cwd=root,capture_output=True,text=True,timeout=40)
    finished=time.monotonic()
    with (root/'automation.log').open('a') as f:f.write(repr(items)+'\n'+p.stdout+p.stderr+'\n')
    with (root/'automation-timing.jsonl').open('a') as f:f.write(json.dumps({'command':items,'started':started,'finished':finished,'returncode':p.returncode})+'\n')
    assert p.returncode==0,(items,p.stdout,p.stderr)
    return p.stdout

def snapshot():return auto('snapshot','--full')

def entries(text=None):
    found=[]
    for line in (snapshot() if text is None else text).splitlines():
        m=re.match(r'\s*(@e\d+) \[(\w+)\](?: ("(?:[^"\\]|\\.)*"))?(.*)',line)
        if m:found.append((m[1],m[2],json.loads(m[3]) if m[3] else '',m[4]))
    return found

def control(role,label,description=None,enabled=True):
    end=time.monotonic()+25
    while True:
        found=[e for e in entries() if e[1]==role and e[2]==label and (not enabled or 'disabled' not in e[3]) and
               (description is None or 'description='+json.dumps(description,ensure_ascii=False) in e[3])]
        if len(found)==1:return found[0]
        assert len(found)<2,('Ambiguous control',found)
        assert app.poll() is None and time.monotonic()<end,('Missing control',role,label,description)
        time.sleep(.05)

def press(role,label):auto('press',control(role,label)[0],'--settle')
def fill(label,value):auto('fill',control('textField',label)[0],value,'--settle')
def wait(text):auto('wait','--text',text,'--timeout','25s','--settle')
def ready():control('menuButton','Sync')

def evidence(name):
    text=snapshot();(root/(name+'.snapshot.txt')).write_text(text);auto('screenshot',str(root/(name+'.png')))
    current=originals();same(current,before,'Original repository/sentinel changed')
    data={'originals':current,'destination':state(destination) if (destination/'.git').is_dir() else None,'cancelled_exists':cancelled.exists()}
    (root/(name+'.git.json')).write_text(json.dumps(data,indent=2)+'\n')
    return text

def root_ready(repo):
    end=time.monotonic()+25
    while not any(e[1]=='button' and e[2]==repo.name and 'bounds=46,' in e[3] and 'description='+json.dumps(str(repo),ensure_ascii=False)+' ' in e[3] for e in entries()):
        assert time.monotonic()<end,'Visible root did not become exact target'
        time.sleep(.05)
    ready()

def open_new(repo=None):
    if repo is None:press('button','New Repository')
    else:
        title=next(e for e in entries() if e[1]=='button' and e[2]==repo.name and 'bounds=46,' in e[3])
        auto('press',title[0],'--settle');press('menuItem','New Repository')
    wait('New repository');control('textField',field)

def no_form():assert not any(e[1]=='textField' and e[2]==field for e in entries()),'New repository form remains open'

def start(repo=None):
    global app,log,session
    sockdir=pathlib.Path(tempfile.mkdtemp(prefix='sgnew-',dir='/tmp'));sockdir.chmod(0o700);socket=sockdir/'app.sock'
    launch_env={**env,'MYGO_AUTOMATION':str(socket),'SUPERGIT_WINDOW_SIZE':'1100x720'}
    log=(root/('app-active.log' if repo else 'app-empty.log')).open('w')
    app=subprocess.Popen([str(binary)]+([str(repo)] if repo else []),env=launch_env,cwd=root,stdout=log,stderr=subprocess.STDOUT)
    session='new-repository-native-'+str(app.pid)
    (root/('pid-active.txt' if repo else 'pid-empty.txt')).write_text(str(app.pid))
    end=time.monotonic()+20
    while not socket.exists():
        assert app.poll() is None and time.monotonic()<end,'Native socket unavailable'
        time.sleep(.05)
    auto('open','--socket',str(socket))

def stop():
    global app,log
    if app is None:return
    app.terminate()
    try:app.wait(timeout=10)
    except subprocess.TimeoutExpired:app.kill();app.wait(timeout=5)
    log.close();note('PROCESS_STOPPED='+str(app.returncode));app=None

def unborn():
    assert git(destination,'rev-parse','--verify','HEAD',check=False).returncode!=0,'Unexpected commit exists'
    same(git(destination,'symbolic-ref','HEAD').strip(),b'refs/heads/main','Wrong unborn branch')
    same(git(destination,'for-each-ref'),b'','Unborn repo acquired refs')

def activate_owned_window():
    # Accessibility focus does not activate an inactive native macOS window.
    # Activate only this script's child process through the normal AppKit API.
    assert app is not None and app.poll() is None
    script='ObjC.import("AppKit"); $.NSRunningApplication.runningApplicationWithProcessIdentifier('+str(app.pid)+').activateWithOptions(3)'
    result=subprocess.run(['/usr/bin/osascript','-l','JavaScript','-e',script],capture_output=True,text=True,timeout=10)
    (root/(step+'-activation.txt')).write_text(result.stdout+result.stderr)
    assert result.returncode==0,('Owned window activation failed',result.stderr)
    end=time.monotonic()+5
    while True:
        windows=auto('windows')
        if any('[content]' in line and ' focused' in line for line in windows.splitlines()):return
        assert app.poll() is None and time.monotonic()<end,'Owned native window did not obtain focus'
        time.sleep(.05)

def row_enter(path,group,expected):
    # Capture slow screenshots and Git fingerprints before establishing focus.
    evidence(step+'-before-focus')
    activate_owned_window()
    ready()
    row=control('listItem',path,group)
    (root/(step+'-windows-before-focus.txt')).write_text(auto('windows'))
    focused_at=time.monotonic()
    auto('focus',row[0],'--settle')
    text=snapshot()
    (root/(step+'-row-focused.snapshot.txt')).write_text(text)
    (root/(step+'-windows-after-focus.txt')).write_text(auto('windows'))
    assert any(e[1]=='listItem' and e[2]==path and 'focused' in e[3] and 'description='+json.dumps(group) in e[3] for e in entries(text)),'File row not focused'
    assert any(e[1]=="menuButton" and e[2]=="Sync" and 'disabled' not in e[3] for e in entries(text)),'Repository loading before file-row Enter'
    key_at=time.monotonic()
    auto('key','Enter','--settle')
    after=snapshot()
    (root/(step+'-after-enter.snapshot.txt')).write_text(after)
    (root/(step+'-enter-timing.json')).write_text(json.dumps({'focus_started':focused_at,'key_started':key_at,'key_snapshot_finished':time.monotonic()},indent=2)+'\n')
    end=time.monotonic()+25
    while git(destination,'ls-files','-z')!=expected:
        assert app.poll() is None and time.monotonic()<end,'UI file row Enter did not produce expected Index path list'
        time.sleep(.05)
    ready();unborn()
    same((destination/text_name).read_bytes(),text_bytes,'UI action changed text working bytes')
    same((destination/'binary.bin').read_bytes(),binary_bytes,'UI action changed binary working bytes')
    evidence(step)

failed=False
try:
    home.mkdir()
    (home/'.gitconfig').write_text('[user]\n name = New Repository Fixture\n email = new-repository@example.invalid\n useConfigOnly = true\n[commit]\n gpgsign = false\n')
    subprocess.run(['git','init','-b','main',str(active)],env=env,check=True,capture_output=True)
    oldfile=active/'old.txt';oldfile.write_bytes(b'old baseline\n');git(active,'add','--all');git(active,'commit','-m','Old repository base')
    oldfile.write_bytes(b'old staged sentinel\n');git(active,'add','old.txt');oldfile.write_bytes(b'old working sentinel\n')
    existing.mkdir();(existing/'sentinel.bin').write_bytes(b'do not change\x00\xff')
    settings=home/'Library/Application Support/supergit/settings.json';settings.parent.mkdir(parents=True)
    settings.write_text(json.dumps({'language':'en','theme':'light','autoFetch':False}))
    bootstrap=False
    before=originals();(root/'before.git.json').write_text(json.dumps(before,indent=2)+'\n')
    step='01-empty-entry-cancel'
    start();wait('New Repository');open_new()
    assert 'disabled' in control('button','Create',enabled=False)[3],'Blank form allowed Create'
    fill(field,str(cancelled));press('button','Cancel');no_form();assert not cancelled.exists();evidence(step)
    note('PASS empty-state New repository entry/Cancel leaves destination absent and original fingerprints unchanged')
    stop()
    step='02-title-entry-cancel'
    start(active);wait('Commit message');root_ready(active);press('tab','Changes')
    fill('Commit message', 'Keep old new-repository draft' + '\n\n' + 'Retain old description')
    open_new(active);fill(field,str(cancelled));press('button','Cancel');no_form()
    text=evidence(step);assert not cancelled.exists()
    assert 'value="Keep old new-repository draft"' in text and 'value="Retain old description"' in text,'Cancel changed old draft'
    note('PASS title-menu form Cancel keeps dirty old root and full draft intact')
    step='03-relative-path-refusal'
    open_new(active);fill(field,'relative-new-repository');press('button','Create')
    wait('Enter an absolute path for the destination directory.');evidence(step)
    assert not (root/'relative-new-repository').exists(),'Relative path unexpectedly created'
    note('PASS relative destination rejected with editable form and no directory creation')
    step='04-existing-destination-refusal'
    fill(field,str(existing));press('button','Create');wait('The destination exists or cannot be accessed.')
    evidence(step);control('textField',field)
    note('PASS occupied sentinel destination refused; all original data preserved and field editable for retry')
    step='05-ui-create-unborn'
    fill(field,str(destination));press('button','Create');root_ready(destination);no_form()
    assert (destination/'.git').is_dir(),'UI Create did not initialize destination'
    unborn();same(git(destination,'ls-files','--stage','-z'),b'','New Index not empty')
    same(git(destination,'status','--porcelain=v1','-z'),b'','New worktree not clean');same(git(destination,'remote'),b'','New repo has remote')
    press('tab','Changes');text=evidence(step+'-changes')
    assert all('value=' not in e[3] for e in entries(text) if e[1]=='textField' and e[2] == 'Commit message'),'Create kept old draft'
    assert not any(e[1]=='checkBox' and e[2]=='Amend last commit' for e in entries(text)),'Unborn repository offers Amend'
    press('button','History');wait('No commits yet.');evidence(step+'-history')
    note('PASS UI Create opens precise Unicode/space root: unborn main/no refs/empty Index/clean/no remotes/No commits yet, old draft reset')
    (destination/text_name).write_bytes(text_bytes);(destination/'binary.bin').write_bytes(binary_bytes)
    press('tab','Changes');auto("key", "cmd+r", "--settle");ready();control('listItem',text_name,'Untracked Files');control('listItem','binary.bin','Untracked Files')
    step='06-unborn-ui-stage-text'
    row_enter(text_name,'Untracked Files',text_name.encode()+b'\x00')
    same(git(destination,'show',':'+text_name),text_bytes,'Staged text blob mismatch')
    note('PASS file-row Enter stages exact text before first commit')
    step='07-unborn-ui-unstage-text'
    row_enter(text_name,'Staged Files',b'');same(git(destination,'ls-files','--stage','-z'),b'','Unstage retained Index entries')
    note('PASS staged file-row Enter restores empty unborn Index and keeps both working files')
    step='08-unborn-ui-restage-text'
    row_enter(text_name,'Untracked Files',text_name.encode()+b'\x00')
    step='09-unborn-ui-stage-binary'
    expected_paths=b'binary.bin\x00'+text_name.encode()+b'\x00'
    row_enter('binary.bin','Untracked Files',expected_paths)
    same(git(destination,'show',':binary.bin'),binary_bytes,'Staged binary blob mismatch')
    note('PASS UI restages text and stages binary with exact blobs; repository remains unborn')
    step='10-first-ui-commit'
    subject='First native New Repository commit';body='Created through native UI\n\nText and binary fixture'
    fill('Commit message', subject + '\n\n' + body)
    index=git(destination,'ls-files','--stage','-z');press('button','Commit')
    end=time.monotonic()+25
    while git(destination,'rev-parse','--verify','HEAD',check=False).returncode:
        assert app.poll() is None and time.monotonic()<end,'UI Commit did not create initial HEAD'
        time.sleep(.05)
    ready();head=git(destination,'rev-parse','HEAD')
    same(git(destination,'show','-s','--format=%P','HEAD').strip(),b'','First commit has parents')
    same(git(destination,'rev-list','--count','HEAD').strip(),b'1','Unexpected extra commits')
    same(git(destination,'for-each-ref','--format=%(refname) %(objectname)'),b'refs/heads/main '+head,'Unexpected refs after first commit')
    same(git(destination,'show','HEAD:'+text_name),text_bytes,'Committed text blob mismatch');same(git(destination,'show','HEAD:binary.bin'),binary_bytes,'Committed binary blob mismatch')
    same(git(destination,'show','-s','--format=%B','HEAD').rstrip(b'\n'),(subject+'\n\n'+body).encode(),'Full message mismatch')
    identity=b'New Repository Fixture <new-repository@example.invalid>'
    same(git(destination,'show','-s','--format=%an <%ae>%n%cn <%ce>','HEAD').splitlines(),[identity,identity],'Author/committer mismatch')
    same(git(destination,'ls-files','--stage','-z'),index,'First commit changed Index entries');same(git(destination,'status','--porcelain=v1','-z'),b'','First commit did not leave clean worktree')
    same((destination/text_name).read_bytes(),text_bytes,'Commit changed text worktree');same((destination/'binary.bin').read_bytes(),binary_bytes,'Commit changed binary worktree')
    evidence(step+'-clean');press('button','History');wait(subject);auto('key','cmd+0','--settle');wait(head.decode().strip())
    press('listItem',text_name);wait('+First owned native repository commit');evidence(step+'-history-files')
    note('PASS first UI commit has zero parents/exact single main ref/full message/author+committer/text+binary tree/Index and clean worktree; History SHA/Files/diff correct')
    step='11-return-original'
    title=next(e for e in entries() if e[1]=='button' and e[2]==destination.name and 'bounds=46,' in e[3])
    auto('press',title[0],'--settle');press('menuItem',str(active));root_ready(active);evidence(step)
    note('PASS recent repository return opens original root with unchanged HEAD/refs/Index/staged+working bytes and sentinel')
except BaseException:
    failed=True;note('FAIL '+step+'\n'+traceback.format_exc())
    if app is not None and app.poll() is None:
        try:evidence('failure-'+step)
        except Exception as error:(root/'capture-error.txt').write_text(repr(error))
finally:stop()
if failed:raise SystemExit(1)
