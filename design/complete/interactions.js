(()=>{
 const A=window.GitDesignApp,M=A.M,D=window.GitDesignData,$=A.$;
 let progressToken=0,commands=[],commandIndex=0;
 const value=id=>$(id)?.value?.trim()||'',checked=id=>!!$(id)?.checked;
 const snapshots=()=>Array.from($('overlay-root').querySelectorAll('input[id],select[id],textarea[id]')).map(el=>({id:el.id,value:el.value,checked:el.checked}));
 function run(fn){const fields=snapshots();try{fn()}catch(e){const s=A.state;if(s.modal&&s.modal.type!=='progress'){s.modal.error=e.message;A.render();for(const f of fields){const el=$(f.id);if(el){el.value=f.value;if('checked'in el)el.checked=f.checked}}}else A.toast(e.message)}}
 function progress(title,description,steps,complete){const token=++progressToken;A.modal('progress',{title,description,steps,step:0});let step=0;function tick(){if(token!==progressToken||A.state.modal?.type!=='progress')return;step++;if(step<=steps.length){A.state.modal.step=step;A.renderModal();setTimeout(tick,440)}else{A.state.modal=null;run(()=>{complete();A.render()})}}setTimeout(tick,300)}
 function close(){if(A.state.modal?.type==='progress'){progressToken++;M.log('取消模拟后台任务','尚未执行的结果未应用。')}A.closeModal()}
 function selectScope(id){if(A.state.operation)throw Error('先完成或中止当前 Git 操作，再切换工作区。');M.scope(id);A.navigate('workspace')}
 function canLeave(){if(A.state.operation)throw Error('先完成或中止当前 Git 操作。')}
 function openCommands(){A.modal('command');commandsForQuery();$('command-query')?.focus()}
 function commandsForQuery(){const q=$('command-query')?.value.toLowerCase()||'';const s=A.state;const all=[...D.routes.map(([route,icon,name])=>({name:'前往'+name,icon,action:'nav',args:{route}})),{name:'打开最近仓库',icon:'repo',action:'repositories'},{name:'克隆仓库',icon:'pull',action:'clone'},{name:'创建分支',icon:'branch',action:'new-branch'},{name:'新建 Worktree',icon:'tree',action:'new-worktree'},{name:'保存当前改动为 Stash',icon:'stash',action:'new-stash'},{name:'编辑 Rebase 计划',icon:'merge',action:'rebase'},{name:'偏好设置',icon:'settings',action:'settings'},{name:'查看操作记录',icon:'terminal',action:'log'},{name:'页面地图与验收清单',icon:'repo',action:'guide'}];if(s.operation)all.unshift({name:'处理当前冲突',icon:'alert',action:'conflicts'});commands=all.filter(c=>c.name.toLowerCase().includes(q));commandIndex=0;renderCommands()}
 function renderCommands(){const root=$('command-list');if(!root)return;root.innerHTML=commands.map((c,i)=>`<button class="command-item ${i===commandIndex?'selected':''}" data-action="command-run" data-index="${i}">${A.icon(c.icon)}${A.esc(c.name)}<small>↵</small></button>`).join('')||'<div class="file-empty">没有匹配的操作</div>'}
 function scene(id){canLeave();const s=A.state;if(id==='conflicts'){M.prepareConflict();A.navigate('conflicts');return}if(id==='rebase'){M.cleanScene();s.rebasePlan=structuredClone(D.initial().rebasePlan);s.rebaseTarget='main';A.navigate('rebase');return}if(id==='empty'){M.cleanScene();A.navigate('workspace');$('scene').value='empty';return}if(id==='error'){s.networkError={title:'无法连接远端 origin',command:'git fetch origin',detail:'ssh: Could not resolve hostname example.invalid\nfatal: Could not read from remote repository.'};A.navigate('error');return}if(id==='push-check'){s.scope='main';const m=s.modules.find(x=>x.id==='core');m.pinned=m.head;m.index=m.head;m.published='a43bc18';A.navigate('workspace');A.modal('push');$('scene').value='push-check';return}A.navigate(id)}
 function ancestor(a,b){const map=new Map(M.rawHistory().map(c=>[c.sha,c])),todo=[b],seen=new Set();while(todo.length){const x=todo.pop();if(x===a)return true;if(seen.has(x))continue;seen.add(x);todo.push(...(map.get(x)?.parents||[]))}return false}
 function historyAction(type,sha){const s=A.state;if(M.files().length)throw Error('工作区有改动。先提交或 Stash，再执行 '+type+'。');if(!M.current().branch)throw Error('先从 detached HEAD 创建分支。');const h=M.rawHistory().find(h=>h.sha===sha);if(!h)throw Error('提交不在当前历史中。');M.save(type==='revert'?'Revert 提交':'Cherry-pick 提交');const newSha=M.nextSha();let patches=structuredClone(h.files);if(type==='revert')patches=patches.map(f=>({...f,rows:f.rows.map(r=>({...r,t:r.t==='add'?'del':r.t==='del'?'add':'ctx',old:r.next,next:r.old}))}));const entry={sha:newSha,parents:[M.head()],title:type==='revert'?'Revert "'+h.title+'"':h.title,body:type==='revert'?'反向撤销 '+h.sha:'Cherry-picked from '+h.sha,author:'示例开发者',time:'刚刚',files:patches};if(s.scope==='core'){s.coreCommits.unshift(entry);s.modules.find(m=>m.id==='core').head=newSha}else if(!M.isChild()){s.commits.unshift(entry);s.branches.find(b=>b.name===M.current().branch).sha=newSha}else throw Error('此子模块样例没有足够历史执行该操作。');M.current().ahead++;s.historySelected=newSha;s.historyFile=0;s.modal=null;M.log('git '+type+(h.parents.length>1?' -m '+value('mainline'):'')+' '+sha);A.render();A.toast('已模拟创建新提交 '+newSha)}
 function action(name,d={}){
  const s=A.state;
  if(['workspace','history','branches','worktrees','submodules','stash','repositories','conflicts','rebase','guide'].includes(name)){if(name==='rebase'&&M.isChild())throw Error('当前 rebase 样例使用主仓库。先返回父仓库。');A.navigate(name);return}
  switch(name){
   case 'nav':A.navigate(d.route);break;
   case 'reset':progressToken++;M.reset();A.navigate('workspace');A.toast('已恢复初始示例数据');break;
   case 'command':openCommands();break;
   case 'command-run':{const c=commands[Number(d.index)];if(c){s.modal=null;action(c.action,c.args||{})}break}
   case 'close-modal':close();break;
   case 'log':A.toggleLog();break;
   case 'refresh':M.log('git status --porcelain=v2 -z');A.render();A.toast('已重新读取当前模拟状态');break;
   case 'scope':selectScope(d.id);break;
   case 'select-file':s.selectedFile=d.id;s.group=d.group;s.selectedLines=[];A.render();break;
   case 'select-line':s.selectedLines=s.selectedLines.includes(d.id)?s.selectedLines.filter(x=>x!==d.id):[...s.selectedLines,d.id];A.render();break;
   case 'stage-file':M.stage(d.id,s.group!=='staged');A.render();break;
   case 'stage-all':M.stageAll(d.stage==='true');A.render();break;
   case 'stage-lines':{const n=M.stageLines();A.render();A.toast('已模拟调整 '+n+' 行的暂存状态');break}
   case 'diff-mode':s.diffMode=d.mode;A.render();break;
   case 'history-mode':s.historyMode=d.mode;A.render();break;
   case 'select-commit':if(!M.visibleHistory().some(h=>h.sha===d.sha))throw Error('该提交位于当前历史边界之外。');s.historySelected=d.sha;s.historyFile=0;A.render();break;
   case 'history-file':case 'stash-file':s.historyFile=Number(d.index);A.render();break;
   case 'copy':navigator.clipboard.writeText(d.text).then(()=>A.toast('已复制 '+d.text),()=>A.toast('请手动复制：'+d.text));break;
   case 'new-branch':A.modal('new-branch',{sha:d.sha||M.head()});break;
   case 'confirm-new-branch':{const name=value('new-branch-name'),upstream=s.modal.upstream;M.createBranch(name,value('new-branch-base'),checked('branch-checkout'));if(upstream&&!M.isChild())A.state.branches.find(b=>b.name===name).upstream=upstream;A.state.modal=null;A.render();A.toast(upstream?'已模拟创建并设置跟踪分支':'已模拟创建分支');break}
   case 'branch-tab':s.branchTab=d.tab;A.render();break;
   case 'checkout':canLeave();if(d.name===M.current().branch){A.navigate('workspace');return}if(M.files().length)A.modal('checkout',{name:d.name});else{M.checkout(d.name);A.render();A.toast('已切换到 '+d.name)}break;
   case 'confirm-checkout':{const name=s.modal.name;M.checkout(name,document.querySelector('input[name="checkout-policy"]:checked')?.value==='stash');A.state.modal=null;A.render();A.toast('已切换到 '+name);break}
   case 'branch-menu':A.modal('branch-menu',{name:d.name});break;
   case 'rename-branch':A.modal('rename-branch',{name:d.name});break;
   case 'confirm-rename':M.renameBranch(s.modal.name,value('rename-value'));A.state.modal=null;A.render();A.toast('已模拟重命名分支');break;
   case 'delete-branch':if(s.worktrees.some(w=>s.scopes[w.id].branch===d.name))throw Error('该分支在 worktree 中使用，不能删除。');A.modal('delete-branch',{name:d.name});break;
   case 'confirm-delete-branch':{const b=s.branches.find(b=>b.name===s.modal.name);if(!ancestor(b.sha,M.head()))throw Error('该分支包含未合入当前分支的提交。默认删除已阻止；先检查历史并合入。');M.deleteBranch(s.modal.name);A.state.modal=null;A.render();A.toast('已模拟删除本地分支');break}
   case 'track-branch':{const r=s.remoteBranches.find(r=>r.name===d.name);A.modal('new-branch',{name:d.name.replace(/^origin\//,''),sha:r.sha,upstream:d.name});break}
   case 'new-worktree':canLeave();if(M.isChild())throw Error('此工作目录管理页使用主仓库，请先返回父仓库。');A.modal('new-worktree');break;
   case 'confirm-worktree':{const id=M.addWorktree(value('worktree-branch'),value('worktree-path'),value('worktree-base'));A.state.modal=null;selectScope(id);A.toast('已模拟创建并打开独立工作区');break}
   case 'remove-worktree':A.modal('remove-worktree',{id:d.id});break;
   case 'confirm-remove-worktree':M.removeWorktree(s.modal.id);A.state.modal=null;A.render();A.toast('已模拟移除工作目录；分支仍然保留');break;
   case 'new-stash':canLeave();A.modal('new-stash');break;
   case 'confirm-stash':M.stashSave(value('stash-name'),checked('stash-untracked'));A.state.modal=null;A.render();A.toast('已模拟保存改动，子模块内部状态保持独立');break;
   case 'select-stash':s.selectedStash=Number(d.id);s.historyFile=0;A.render();break;
   case 'apply-stash':case 'pop-stash':canLeave();M.applyStash(Number(d.id),name==='pop-stash');A.render();A.toast(name==='pop-stash'?'已模拟恢复改动并移除记录':'已模拟恢复改动，Stash 仍然保留');break;
   case 'drop-stash':A.modal('drop-stash',{id:Number(d.id)});break;
   case 'confirm-drop-stash':M.dropStash(s.modal.id);A.state.modal=null;A.render();A.toast('已模拟删除 Stash');break;
   case 'select-module':s.selectedModule=d.id;A.navigate('submodules');break;
   case 'enter-module':{const m=s.modules.find(m=>m.id===d.id);if(!m.initialized){A.modal('init-module',{id:d.id});return}selectScope(d.id);break}
   case 'back-parent':{const m=s.modules.find(m=>m.id===s.scope);selectScope(m?.parent||'main');s.selectedModule=m?.id||'core';break}
   case 'module-stage':{const m=s.modules.find(m=>m.id===d.id);const old=s.scope;if(old!==m.parent)s.scope=m.parent;try{M.stage('module:'+d.id,m.index===m.pinned)}finally{A.state.scope=old}A.render();break}
   case 'init-module':A.modal('init-module',{id:d.id});break;
   case 'init-all':A.modal('init-all');break;
   case 'confirm-init':{const ids=s.modal.type==='init-all'?s.modules.filter(m=>!m.initialized).map(m=>m.id):[s.modal.id],recursive=checked('init-recursive');progress('初始化子模块','获取所需提交，并检出父仓库记录的版本。',['检查引用与地址','获取子模块内容','检出记录版本'],()=>{ids.forEach(id=>M.initializeModule(id,recursive));A.toast('已模拟初始化子模块')});break}
   case 'module-update':A.modal('module-update',{id:d.id,target:d.target});break;
   case 'confirm-module-update':{const {id,target}=s.modal,method=value('module-method');M.updateModule(id,target,method);A.state.modal=null;A.render();A.toast('已模拟更新子模块；父仓库指针尚未提交');break}
   case 'remote-edit':A.modal('remote-edit',{name:d.name});break;
   case 'confirm-remote':{const url=value('remote-url');if(!/^(https?:\/\/|ssh:\/\/|[^\s@]+@[^\s:]+:|\/)/.test(url))throw Error('填写有效的 HTTPS、SSH 或本地仓库地址。');M.save('修改远端地址');s.remotes.find(r=>r.name===s.modal.name).url=url;M.log('git remote set-url '+s.modal.name+' '+url);s.modal=null;A.render();A.toast('已模拟更新远端地址');break}
   case 'remote-config':s.branchTab='remotes';A.navigate('branches');break;
   case 'fetch':case 'retry-fetch':progress('Fetch 远端引用','在后台获取引用，不改动工作区文件。',['连接 origin','获取远端引用','刷新分支状态'],()=>{M.log('git fetch origin');A.state.networkError=null;if(A.state.route==='error')A.state.route='branches';A.toast('Fetch 模拟完成，工作区改动保持不变')});break;
   case 'pull':A.modal('pull');break;
   case 'confirm-pull':{if(s.operation)throw Error('先完成当前 Git 操作。');if(M.files().length)throw Error('工作区有改动。先提交或 Stash，子模块内部改动也需要处理。');const method=value('pull-method');progress('Pull 远端更新','按选择的策略处理当前分支。',['获取远端引用','检查提交祖先关系','更新本地状态'],()=>{M.log('git pull '+(method==='ff'?'--ff-only':method==='rebase'?'--rebase':'--no-rebase'));A.toast('Pull 模拟完成；此场景没有新的远端提交')});break}
   case 'push':if(!M.current().branch)throw Error('detached HEAD 不能直接推送，先创建分支。');A.modal('push');break;
   case 'confirm-push':{const include=checked('push-modules');if(M.publicationChecks().length&&!include)throw Error('先发布被引用的子模块提交，父仓库 Push 已阻止。');progress('Push 提交','先处理依赖的子模块，再推送父仓库。',['检查远端与引用','发布子模块依赖','推送当前分支','刷新跟踪状态'],()=>{M.push(include);A.toast('已模拟完成推送与引用检查')});break}
   case 'settings':A.modal('settings');break;
   case 'settings-tab':s.settings.tab=d.tab;A.render();break;
   case 'setting-theme':s.settings.theme=d.value;A.render();break;
   case 'setting-density':s.settings.density=d.value;A.render();break;
   case 'setting-accent':s.settings.accent=d.value;A.render();break;
   case 'rebase-to':s.rebaseTarget=d.name;A.navigate('rebase');break;
   case 'rebase-move':{const i=Number(d.index),j=i+Number(d.dir);if(j>=0&&j<s.rebasePlan.length)[s.rebasePlan[i],s.rebasePlan[j]]=[s.rebasePlan[j],s.rebasePlan[i]];A.render();break}
   case 'start-rebase':M.validateRebase();A.modal('start-rebase');break;
   case 'confirm-rebase':M.startRebase();A.render();A.toast(A.state.operation?'Rebase 模拟暂停：请处理冲突':'Rebase 模拟完成，历史已更新');break;
   case 'load-conflict':canLeave();M.prepareConflict();A.navigate('conflicts');break;
   case 'resolve-choice':s.conflict.result=d.choice==='both'?'fn refresh(&mut self, event: Event) {\n    self.cache.invalidate(&event.path);\n    self.schedule_refresh(event.path);\n}':s.conflict[d.choice];s.conflict.resolved=false;A.render();break;
   case 'show-base':A.modal('show-base');break;
   case 'mark-resolved':M.resolve();A.render();A.toast('已模拟暂存解决结果，可以继续操作');break;
   case 'continue-operation':M.continueOperation();A.render();A.toast('已模拟完成操作，历史与分支引用已更新');break;
   case 'abort':A.modal('abort');break;
   case 'confirm-abort':M.abort();A.render();A.toast('已模拟中止，并恢复操作前状态');break;
   case 'cherry-pick':case 'revert':canLeave();A.modal(name,{sha:d.sha});break;
   case 'confirm-commit-action':historyAction(s.modal.type,s.modal.sha);break;
   case 'merge-branch':A.modal('merge',{name:d.name});break;
   case 'confirm-merge':{canLeave();if(M.files().length)throw Error('先提交或 Stash 当前改动。');const source=s.branches.find(b=>b.name===s.modal.name);if(ancestor(source.sha,M.head())){s.modal=null;A.render();A.toast('当前分支已经包含该分支的提交，无需合并');break}M.save('合并分支');const sha=M.nextSha(),oldHead=M.head();s.commits.unshift({sha,parents:[oldHead,source.sha],title:"Merge branch '"+source.name+"'",body:'模拟合并所选分支。',author:'示例开发者',time:'刚刚',files:[D.patches.refresh]});s.branches.find(b=>b.name===M.current().branch).sha=sha;M.current().ahead++;M.log('git merge '+source.name);s.modal=null;s.route='history';s.historySelected=sha;s.historyFile=0;A.render();A.toast('已模拟创建双父合并提交');break}
   case 'open-repo':A.modal('open-repo');break;
   case 'confirm-open-repo':{const path=value('repo-path');if(!path)throw Error('选择一个目录。');A.state.modal=null;selectScope(path.includes('sample-review')?'review':'main');A.toast('已载入目录对应的示例仓库');break}
   case 'clone':A.modal('clone');break;
   case 'confirm-clone':{const url=value('clone-url'),path=value('clone-path'),recursive=checked('clone-recursive');if(!/^(https?:\/\/\S+|ssh:\/\/\S+|[^\s@]+@[^\s:]+:\S+)$/.test(url))throw Error('填写有效的 HTTPS 或 SSH 仓库地址。');if(!path||path==='~/'||path==='/')throw Error('选择一个独立的目标目录。');progress('克隆仓库','目标 '+path,['检查地址与目标目录','获取对象与引用','检出工作目录',...(recursive?['初始化子模块']:[])],()=>{A.state.cloneComplete=true;M.log('git clone '+(recursive?'--recurse-submodules ':'')+url+' '+path);A.state.route='repositories';A.toast('已模拟克隆。最近仓库列表中已加入演示入口')});break}
   case 'cancel-progress':close();A.toast('已取消模拟任务');break;
   case 'undo':{if(s.operation)throw Error('当前操作进行中，请先中止。');const label=M.undo();A.render();A.toast('已恢复“'+label+'”之前的演示状态');break}
   default:A.toast('此操作未定义：'+name);
  }
 }
 document.addEventListener('click',e=>{const el=e.target.closest('[data-action]');if(!el||el.disabled)return;run(()=>action(el.dataset.action,el.dataset))});
 document.addEventListener('submit',e=>{if(e.target.id==='commit-form'){e.preventDefault();run(()=>{const sha=M.commit();A.render();A.toast('已模拟提交 '+sha+'；仅包含暂存内容')})}});
 document.addEventListener('input',e=>{const el=e.target,s=A.state,b=el.dataset.bind;if(el.id==='command-query'){commandsForQuery();return}if(b==='commit-title'){M.current().title=el.value;const btn=$('commit-button');if(btn)btn.disabled=!el.value.trim()||!M.groupFiles('staged').length||!M.current().branch||!!s.operation;return}if(b==='commit-body'){M.current().body=el.value;return}if(b==='conflict-result'){s.conflict.result=el.value;s.conflict.resolved=false;$('conflict-status').textContent='结果已修改，需重新标记解决。';const btn=$('root').querySelector('[data-action="continue-operation"]');if(btn)btn.disabled=true;return}if(b==='filter'){s.filter=el.value;A.render()}if(b==='branch-filter'){s.branchFilter=el.value;A.render()}if(b==='repo-filter'){s.repoFilter=el.value;A.render()}if(b==='setting-font'){s.settings.font=Number(el.value);A.applyAppearance();$('font-label').textContent=el.value+' px'}if(b==='git-path')s.settings.gitPath=el.value;if(b==='rebase-title')s.rebasePlan[Number(el.dataset.index)].title=el.value});
 document.addEventListener('change',e=>{const el=e.target,s=A.state,b=el.dataset.bind;if(el.id==='scene'){run(()=>scene(el.value));return}if(b==='rebase-action'){s.rebasePlan[Number(el.dataset.index)].action=el.value;A.render()}if(b==='rebase-target'){s.rebaseTarget=el.value;A.render()}if(b==='rebase-conflict')s.rebaseConflict=el.checked;if(b==='git-engine'){s.settings.git=el.value;A.render()}if(b==='pull-default')s.settings.pull=el.value});
 document.addEventListener('keydown',e=>{
  const s=A.state,editing=['INPUT','TEXTAREA','SELECT'].includes(document.activeElement?.tagName);
  if((e.metaKey||e.ctrlKey)&&e.key.toLowerCase()==='k'){e.preventDefault();openCommands();return}
  if(e.key==='Escape'&&s.modal){e.preventDefault();close();return}
  if(s.modal?.type==='command'&&['ArrowDown','ArrowUp','Enter'].includes(e.key)){e.preventDefault();if(e.key==='Enter'){const c=commands[commandIndex];if(c)run(()=>{s.modal=null;action(c.action,c.args||{})})}else if(commands.length){commandIndex=(commandIndex+(e.key==='ArrowDown'?1:-1)+commands.length)%commands.length;renderCommands();$('command-list').children[commandIndex]?.scrollIntoView({block:'nearest'})}return}
  if(e.key==='Tab'&&s.modal){const dialog=$('overlay-root').querySelector('[role="dialog"]'),els=dialog?Array.from(dialog.querySelectorAll('button:not(:disabled),input:not(:disabled),select:not(:disabled),textarea:not(:disabled)')).filter(x=>x.offsetParent!==null):[];if(els.length){const first=els[0],last=els.at(-1);if(e.shiftKey&&document.activeElement===first){e.preventDefault();last.focus()}else if(!e.shiftKey&&document.activeElement===last){e.preventDefault();first.focus()}}}
  if(s.modal)return;
  if((e.metaKey||e.ctrlKey)&&e.key==='Enter'&&s.route==='workspace'){e.preventDefault();$('commit-form')?.requestSubmit()}
  if(!editing&&!e.metaKey&&!e.ctrlKey&&e.key.toLowerCase()==='s'&&s.route==='workspace'){e.preventDefault();run(()=>action('stage-lines'))}
  if(!editing&&(e.key==='Enter'||e.key===' ')&&document.activeElement?.matches('tr[data-action]')){e.preventDefault();document.activeElement.click()}
 });
 window.addEventListener('hashchange',()=>{const route=location.hash.slice(1);if(['workspace','history','branches','worktrees','submodules','stash','conflicts','rebase','repositories','error','guide'].includes(route)&&route!==A.state.route)A.navigate(route)});
 A.start();
})();
