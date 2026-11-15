(() => {
  'use strict';
  const icons = {
    branch:'M6 3v12a3 3 0 0 0 3 3h2M6 7h5a4 4 0 0 0 4-4M4 3a2 2 0 1 0 4 0a2 2 0 1 0-4 0M13 18a2 2 0 1 0 4 0a2 2 0 1 0-4 0M13 3a2 2 0 1 0 4 0a2 2 0 1 0-4 0',
    history:'M4 7a8 8 0 1 1-1 7M3 3v5h5M11 6v5l3 2',
    changes:'M5 3h8l4 4v12H5zM13 3v5h4M8 12h6M11 9v6',
    box:'M3 6l8-4l8 4v10l-8 4l-8-4zM3 6l8 4l8-4M11 10v10M7 4l8 4',
    file:'M5 2h8l4 4v14H5zM13 2v5h4',
    folder:'M2 6V4h6l2 2h9v12H2z',
    chevron:'M8 5l6 6l-6 6',
    down:'M6 8l5 5l5-5',
    search:'M15 15l5 5M17 9a7 7 0 1 1-14 0a7 7 0 1 1 14 0',
    refresh:'M18 6a8 8 0 1 0 1 8M18 2v5h-5',
    fetch:'M11 2v11M7 9l4 4l4-4M3 15v4h16v-4',
    pull:'M11 2v16M6 13l5 5l5-5',
    push:'M11 20V4M6 9l5-5l5 5',
    plus:'M11 4v14M4 11h14',
    minus:'M4 11h14',
    dots:'M5 11h.01M11 11h.01M17 11h.01',
    stash:'M3 6h16v4H3zM5 10v9h12v-9M9 14h4M5 3h12',
    settings:'M8 3h6l1 3l3 1v6l-3 1l-1 3H8l-1-3l-3-1V7l3-1zM14 10a3 3 0 1 1-6 0a3 3 0 1 1 6 0',
    worktree:'M11 3v6M4 17v-6h14v6M8 3h6v4H8zM1 17h6v4H1zM15 17h6v4h-6z',
    check:'M4 11l4 4l10-10',
    tag:'M3 3h8l9 9l-8 8l-9-9zM7 7h.01',
    remote:'M6 16a4 4 0 1 1 0-8a6 6 0 0 1 11-1a4 4 0 1 1 0 9M11 10v9M8 16l3 3l3-3',
    commit:'M3 11h4M15 11h4M15 11a4 4 0 1 1-8 0a4 4 0 1 1 8 0',
    code:'M7 5l-5 6l5 6M15 5l5 6l-5 6M13 3l-4 16',
    copy:'M8 7h11v13H8zM4 15H2V2h11v2',
    arrow:'M3 11h15M13 6l5 5l-5 5',
    expand:'M8 3H3v5M14 3h5v5M3 14v5h5M19 14v5h-5',
    lock:'M5 9h12v11H5zM7 9V6a4 4 0 0 1 8 0v3',
    table:'M2 3h18v16H2zM2 8h18M8 3v16',
  };
  const icon = name => `<svg class="icon" viewBox="0 0 22 22" aria-hidden="true"><path d="${icons[name] || icons.file}" fill="none"></path></svg>`;
  const escape = s => String(s).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
  const commits = [
    {sha:'b71f9c2',p:['6d20a8e'],lane:0,title:'保留部分暂存后的工作区版本',ref:'feature/partial-stage',time:'11:42',author:'dev',body:'按选中片段生成补丁，只更新暂存区。\n保留同一文件中尚未选择的改动。',kind:'stage'},
    {sha:'6d20a8e',p:['c54a916'],lane:0,title:'显示子模块的三个版本指针',time:'11:16',author:'dev',body:'分别展示父提交、暂存区与子模块当前版本。\n指针变化和子模块内部改动独立显示。',kind:'module'},
    {sha:'c54a916',p:['a8b1d03','8b2f19c'],lane:0,title:"Merge branch 'fix/refresh'",time:'10:58',author:'maintainer',body:'合入路径级刷新。\n此处查看与第一父提交的差异。',kind:'refresh'},
    {sha:'8b2f19c',p:['09ce45b'],lane:1,title:'外部修改后刷新当前文件',ref:'fix/refresh',time:'10:36',author:'dev',body:'仅使发生变化的路径失效。\n保留其他文件的选择和阅读位置。',kind:'refresh'},
    {sha:'a8b1d03',p:['09ce45b'],lane:0,title:'切换文件时保留阅读位置',ref:'main',time:'10:21',author:'maintainer',body:'按文件记录 Diff 阅读位置。\n同一路径的暂存区与工作区分别处理。',kind:'workspace'},
    {sha:'09ce45b',p:['e92f3a7','a43bc18'],lane:0,title:"Merge branch 'feature/submodules'",time:'昨天',author:'maintainer',body:'将子模块状态接入主工作区。\n不隐式更新子模块当前检出版本。',kind:'module'},
    {sha:'a43bc18',p:['f2d783c'],lane:2,title:'区分 gitlink 与子模块当前版本',ref:'feature/submodules',time:'昨天',author:'dev',body:'父仓库版本、暂存区版本与当前检出各自保留。',kind:'module'},
    {sha:'e92f3a7',p:['f1a623b'],lane:0,title:'按路径缓存 Index 条目',time:'昨天',author:'dev',body:'文件变化只使对应缓存条目失效。',kind:'refresh'},
    {sha:'f1a623b',p:['f2d783c'],lane:0,title:'支持嵌套仓库路径',time:'昨天',author:'dev',body:'保留原始路径，使用 literal pathspec。',kind:'workspace'},
    {sha:'f2d783c',p:['d42a8b1'],lane:0,title:'建立工作区导航',ref:'v0.3.0',time:'周一',author:'maintainer',body:'工作区、历史与子模块共享仓库入口。',kind:'workspace'},
    {sha:'d42a8b1',p:['0ab2c3d'],lane:0,title:'添加只读历史 Diff',time:'周一',author:'dev',body:'提交选择固定到不可变的对象 SHA。',kind:'stage'},
    {sha:'0ab2c3d',p:[],lane:0,title:'初始化项目',time:'周一',author:'maintainer',body:'创建示例项目基础结构。',kind:'workspace'},
  ];
  const stageRows = [
    ['ctx','func (r *Repository) StageLines(',48,48],['ctx','    ctx context.Context, patch Patch,',49,49],['ctx',') error {',50,50],['ctx','    r.writeMu.Lock()',51,51],['ctx','    defer r.writeMu.Unlock()',52,52],['ctx','',53,53],['del','    return r.stageFile(ctx, patch.Path)',54,null],['add','    if err := r.verifyPatch(ctx, patch); err != nil {',null,54],['add','        return err',null,55],['add','    }',null,56],['add','    return r.applyToIndex(ctx, patch)',null,57],['ctx','}',55,58],
  ];
  const refreshRows = [
    ['ctx','func (w *Watcher) Changed(path string) {',23,23],['del','    w.cache.Clear()',24,null],['del','    w.refreshAll()',25,null],['add','    w.cache.Invalidate(path)',null,24],['add','    w.scheduleRefresh(path)',null,25],['ctx','}',26,26],['ctx','',27,27],['ctx','func (w *Watcher) Close() {',28,28],['ctx','    w.cancel()',29,29],['ctx','}',30,30],
  ];
  const workspaceRows = [
    ['ctx','func (a *app) selectFile(path string) {',18,18],['del','    a.scrollOffset = 0',19,null],['add','    a.saveScroll(a.selectedPath)',null,19],['ctx','    a.selectedPath = path',20,20],['add','    a.restoreScroll(path)',null,21],['ctx','    a.loadDiff()',21,22],['ctx','}',22,23],
  ];
  const testRows = [
    ['add','func TestStageLinesPreservesWorktree(t *testing.T) {',null,12],['add','    repo := newFixture(t)',null,13],['add','    repo.Write("notes.txt", "first\\nsecond\\n")',null,14],['add','    repo.StageLines("notes.txt", 1, 1)',null,15],['add','',null,16],['add','    require.Equal(t, "first\\n", repo.Index("notes.txt"))',null,17],['add','    require.Equal(t, "first\\nsecond\\n", repo.Read("notes.txt"))',null,18],['add','}',null,19],
  ];
  const moduleRows = [['del','Subproject commit a43bc18a43bc18a43bc18a43bc18a43bc18a43bc18',1,null],['add','Subproject commit e92f3a7e92f3a7e92f3a7e92f3a7e92f3a7e92f3a7',null,1]];
  const files = {
    stage:[{path:'internal/gitrepo/stage.go',status:'M',add:4,del:1,rows:stageRows},{path:'internal/desktop/workspace.go',status:'M',add:2,del:1,rows:workspaceRows},{path:'internal/gitrepo/stage_test.go',status:'A',add:8,del:0,rows:testRows}],
    refresh:[{path:'internal/gitrepo/watch.go',status:'M',add:2,del:2,rows:refreshRows},{path:'internal/desktop/workspace.go',status:'M',add:2,del:1,rows:workspaceRows}],
    workspace:[{path:'internal/desktop/workspace.go',status:'M',add:2,del:1,rows:workspaceRows}],
    module:[{path:'modules/core',status:'M',add:1,del:1,rows:moduleRows},{path:'.gitmodules',status:'M',add:2,del:1,rows:[['ctx','[submodule "core"]',1,1],['ctx','    path = modules/core',2,2],['del','    url = ../core.git',3,null],['add','    url = ../atlas-core.git',null,3]]}],
  };
  const workingFiles = [...files.stage.map(f=>({...f,staged:false})),{path:'README.md',status:'M',add:2,del:1,staged:true,rows:[['ctx','# Atlas',1,1],['ctx','',2,2],['del','Stage complete files.',3,null],['add','Stage files, hunks, or selected lines.',null,3],['add','Keep unrelated changes in the working tree.',null,4]]},{...files.module[0],staged:false}];
  const modules = [
    {id:'core',name:'core',path:'modules/core',branch:'feature/index-cache',recorded:'a43bc18',index:'a43bc18',current:'e92f3a7',state:'指针变化',body:'子模块当前版本比父仓库记录前进 2 个提交，另有 1 个文件尚未提交。',changes:[['internal/cache.go','M','内部未提交'],['子模块提交','↑ 2','尚未记录到父仓库']]},
    {id:'theme',name:'theme',path:'modules/theme',branch:'detached HEAD',recorded:'52b8a1d',index:'52b8a1d',current:'52b8a1d',state:'版本一致',body:'当前版本与父仓库、暂存区记录一致。子模块工作区干净。',changes:[]},
    {id:'tokenizer',name:'tokenizer',path:'vendor/tokenizer',branch:'未初始化',recorded:'13af0c1',index:'13af0c1',current:null,state:'未初始化',body:'父仓库已记录这个子模块。初始化后将检出暂存区所记录的提交。',changes:[]},
  ];
  const directions = {
    alloy:{title:'铂金，把工具做得安静。',summary:'灰白与低饱和青绿，像一件打磨过的桌面工具。控件退到边缘，提交和代码成为主体。',layout:'引用留在左侧，历史与详情在上，Diff 横向铺开。密度接近专业 Git 工具，最适合先落地为日常工作台。'},
    night:{title:'夜航，让历史与代码并肩。',summary:'蓝灰色层次构成安静的暗室。用分区明度组织内容，而非用发光描边制造科技感。',layout:'历史贯穿左中部，右侧整列留给代码。提交摘要压在历史下方，适合长时间审阅和连续比较。'},
    loom:{title:'织线，让分支关系成为主角。',summary:'浅紫灰的外壳、柔和的几何边界。把 Git 的线、结点和分叉作为识别语言，保留代码工具的克制。',layout:'窄导航在左，引用地图在右。上方展开历史线图，下方是不对称的提交摘要与改动审阅台。'},
  };
  const state = {direction:'alloy',scene:'history',selected:commits[0].sha,file:0,query:'',split:false,table:false,module:'core',working:0,commitTitle:'feat: stage only selected changes',commitBody:''};
  const root = document.getElementById('prototype');
  const tool = (action,label,ico,cls='',title='') => `<button type="button" class="tool ${cls}" data-action="${action}" title="${escape(title || label)}">${ico?icon(ico):''}${label?`<span>${label}</span>`:''}</button>`;
  const commit = () => commits.find(c=>c.sha===state.selected) || commits[0];
  const activeFiles = () => state.scene==='workspace' ? workingFiles : files[commit().kind];
  const activeFile = () => state.scene==='workspace' ? workingFiles[state.working] : activeFiles()[Math.min(state.file,activeFiles().length-1)];
  const basename = p=>p.split('/').at(-1);
  const dirname = p=>p.includes('/') ? p.slice(0,p.lastIndexOf('/')) : '项目根目录';
  const fullSha = s=>(s+'e841752c2c98d21ac8d0c481784261bb7f8').slice(0,40);
  const pageLinks = () => `<div class="page-links">${[['workspace','changes','工作区',workingFiles.filter(f=>!f.staged).length],['history','history','历史',''],['modules','box','子模块',3]].map(([id,ico,label,count])=>`<button type="button" class="page-link ${state.scene===id?'active':''}" data-scene="${id}" aria-current="${state.scene===id?'page':'false'}" title="${label}">${icon(ico)}<span>${label}</span>${count!==''?`<span class="count">${count}</span>`:''}</button>`).join('')}</div>`;
  function refs(){ return `<div class="sidebar-section"><div class="sidebar-title">${icon('down')}本地分支<button type="button" data-action="new-branch" title="创建分支">${icon('plus')}</button></div>${[['feature/partial-stage','b71f9c2','',true],['main','a8b1d03','',false],['fix/refresh','8b2f19c','fix',false],['feature/submodules','a43bc18','core',false]].map(([name,sha,color,current])=>`<button type="button" class="ref-item ${current?'active':''}" data-ref="${sha}" title="定位 ${name} 的提交"><i class="branch-dot ${color}"></i><span>${name}</span>${current?'<small>↑2</small>':''}</button>`).join('')}</div><div class="sidebar-section"><div class="sidebar-title">${icon('down')}远端</div><button class="ref-item" data-ref="a8b1d03" title="定位 origin/main">${icon('remote')}<span>origin/main</span></button><button class="ref-item" data-ref="c54a916" title="定位 origin/feature">${icon('remote')}<span>origin/feature</span></button></div><div class="sidebar-section"><div class="sidebar-title">${icon('down')}标签</div><button class="ref-item" data-ref="f2d783c" title="查看标签 v0.3.0">${icon('tag')}<span>v0.3.0</span></button></div>`; }
  function navigation(){return `<aside class="navigation" aria-label="仓库导航">${pageLinks()}${refs()}<div class="sidebar-section"><div class="sidebar-title">${icon('down')}子模块</div><button class="ref-item" data-module="core">${icon('box')}<span>core</span><small>↑2</small></button><button class="ref-item" data-module="theme">${icon('box')}<span>theme</span></button></div><div class="sidebar-footer"><span class="sidebar-label">${icon('worktree')} 主工作区</span><button type="button" data-action="appearance" title="当前设计方向">${icon('settings')}</button></div></aside>`;}
  function titlebar(){return `<header class="titlebar"><div class="traffic" aria-hidden="true"><i></i><i></i><i></i></div><span class="app-wordmark">${icon('branch')}SuperGit</span><button class="repo-selector" data-action="repository">${icon('folder')}<span>atlas</span>${icon('down')}<span class="repo-subtitle">主工作区</span></button><span class="repo-branch">${icon('branch')}feature/partial-stage</span><div class="titlebar-spacer"></div><div class="toolbar-group">${tool('search','', 'search','icon-only optional','搜索提交 ⌘F')}<span class="toolbar-sep optional"></span>${tool('fetch','Fetch','fetch')}${tool('pull','Pull','pull','optional')}${tool('push','Push','push','outlined')}<span class="toolbar-sep optional"></span>${tool('refresh','','refresh','icon-only','刷新示例状态')}</div></header>`;}
  function history(){return `<section class="history-pane" aria-label="历史提交"><div class="section-toolbar"><h2>历史</h2><span class="subtle">所有引用</span><div class="toolbar-group"><label class="searchbox">${icon('search')}<input id="commit-search" aria-label="搜索提交" placeholder="搜索提交、SHA 或引用" value="${escape(state.query)}"><kbd>⌘ F</kbd></label>${tool('graph-mode','',state.table?'history':'table','icon-only',state.table?'显示线图':'显示父提交关系表')}</div></div><div class="history-summary"><span class="graph-key"><i></i>主线</span><span class="graph-key"><i></i>fix/refresh</span><span class="graph-key"><i></i>子模块</span><span>12 条示例提交</span></div><div id="history-rows" class="${state.table?'graph-table-wrap':'graph-scroll'}"></div></section>`;}
  function renderRows(){
    const holder=document.getElementById('history-rows'); if(!holder) return;
    const q=state.query.trim().toLowerCase();
    const shown=commits.filter(c=>(c.title+' '+c.sha+' '+(c.ref||'')).toLowerCase().includes(q));
    if(!shown.length){holder.innerHTML='<p class="graph-empty">没有匹配的提交。试试 SHA 或分支名称。</p>';return;}
    if(state.table){holder.innerHTML=`<table class="graph-table"><thead><tr><th>提交</th><th>父提交</th><th>说明</th></tr></thead><tbody>${shown.map(c=>`<tr><td><code>${c.sha}</code></td><td><code>${c.p.join(', ')||'根提交'}</code></td><td><button data-commit="${c.sha}">${escape(c.title)}</button></td></tr>`).join('')}</tbody></table>`;return;}
    const row=parseFloat(getComputedStyle(root).getPropertyValue('--row'));
    const x=lane=>14+lane*24;
    const y=index=>row*index+row/2;
    const bySha=new Map(shown.map((c,i)=>[c.sha,{...c,i}]));
    const edges=[];
    for(let i=0;i<shown.length;i++){
      const c=shown[i];
      for(const parent of c.p){
        const p=bySha.get(parent); const lane=c.lane===0 && p && p.lane!==0 ? p.lane:c.lane;
        const color=['var(--graph-main)','var(--graph-fix)','var(--graph-core)'][lane];
        if(!p){edges.push(`<path class="dag-path" d="M${x(c.lane)} ${y(i)}v${row*.38}" stroke="${color}" stroke-dasharray="2 3"></path>`);continue;}
        const xx=x(c.lane),px=x(p.lane),yy=y(i),py=y(p.i);
        const path=xx===px?`M${xx} ${yy}V${py}`:`M${xx} ${yy}C${xx} ${yy+row*.55} ${px} ${py-row*.55} ${px} ${py}`;
        edges.push(`<path class="dag-path" d="${path}" stroke="${color}"></path>`);
      }
    }
    const nodes=shown.map((c,i)=>{const color=['var(--graph-main)','var(--graph-fix)','var(--graph-core)'][c.lane];return `<g><circle class="dag-node" cx="${x(c.lane)}" cy="${y(i)}" r="${c.p.length>1?5.7:4.8}" style="fill:${color}"></circle>${c.p.length>1?`<circle cx="${x(c.lane)}" cy="${y(i)}" r="2.4" fill="var(--surface)"></circle>`:''}</g>`;}).join('');
    holder.innerHTML=`<div class="graph-body"><svg class="dag-svg" viewBox="0 0 81 ${shown.length*row}" style="height:${shown.length*row}px" role="img" aria-label="按真实父提交关系连接的示例 DAG；双圈表示合并提交。父关系也可切换表格查看。">${edges.join('')}${nodes}</svg>${shown.map(c=>`<button type="button" class="commit-row ${c.sha===state.selected?'selected':''}" data-commit="${c.sha}" aria-pressed="${c.sha===state.selected}" title="${escape(c.title)}\n${c.sha}\n父提交：${c.p.join(', ')||'无'}"><span class="commit-main">${c.ref?`<span class="ref-badge ${c.sha==='b71f9c2'?'current':''}">${icon(c.ref.startsWith('v')?'tag':'branch')}${escape(c.ref)}</span>`:''}<span class="commit-subject">${escape(c.title)}</span></span><code class="commit-hash">${c.sha}</code><span class="commit-time">${c.time}</span></button>`).join('')}</div>`;
  }
  function inspector(){const c=commit();return `<aside class="inspector" aria-label="提交详情"><div class="inspector-kicker">${icon(c.p.length>1?'branch':'commit')}<span>${c.sha}</span>${tool('commit-menu','','dots','icon-only','提交操作')}</div><h3>${escape(c.title)}</h3><p>${escape(c.body).replace(/\n/g,'<br>')}</p><div class="author-line"><span class="avatar">${c.author==='dev'?'d':'m'}</span><span>${c.author}</span><time>11 月 11 日 ${c.time}</time></div><div class="inspector-meta"><span>提交</span><code>${fullSha(c.sha)}</code><span>父级</span><code>${c.p.join(' / ')||'根提交'}</code><span>变更</span><span>${files[c.kind].length} 个文件</span></div><div class="inspector-actions"><button class="text-action" data-action="new-branch">${icon('branch')}创建分支</button><button class="text-action" data-action="copy-sha">${icon('copy')}复制 SHA</button></div></aside>`;}
  function syntax(text){return escape(text).replace(/\b(func|return|defer|if|nil|error|type|struct)\b/g,'<span class="syntax-key">$1</span>');}
  function diffLine(r,side){if(side==='old' && r[0]==='add' || side==='new' && r[0]==='del')return ''; const n=side==='new'?r[3]:r[2];return `<div class="diff-line ${r[0]}" ${state.scene==='workspace' && r[0]!=='ctx'?'data-select-line="true" tabindex="0" role="button" aria-label="选择这一行改动"':''}><span class="line-no">${n??r[2]??''}</span>${!side?`<span class="line-no">${r[3]??''}</span>`:''}<span class="sign">${r[0]==='add'?'+':r[0]==='del'?'−':' '}</span><code>${syntax(r[1])||' '}</code></div>`;}
  function diffContent(){const f=activeFile();return `${state.split?`<div class="diff-split">${['old','new'].map(side=>`<div class="split-side"><div class="split-title">${side==='old'?'修改前':'修改后'}</div>${f.rows.map(r=>diffLine(r,side)).join('')}</div>`).join('')}</div>`:`<div class="hunk-head"><span>@@ −${f.rows.find(r=>r[2])?.[2]||0},${f.rows.filter(r=>r[0]!=='add').length} +${f.rows.find(r=>r[3])?.[3]||0},${f.rows.filter(r=>r[0]!=='del').length} @@</span>${state.scene==='workspace'?tool('stage-hunk',f.staged?'取消暂存片段':'暂存片段',f.staged?'minus':'plus'):''}</div>${f.rows.map(r=>diffLine(r)).join('')}`}`;}
  function review(){const fs=activeFiles(),f=activeFile();const add=fs.reduce((a,f)=>a+f.add,0),del=fs.reduce((a,f)=>a+f.del,0);return `<section class="review-pane" aria-label="文件差异"><div class="review-head"><h2>${state.scene==='workspace'?(f.staged?'暂存区':'工作区'):'提交改动'}</h2><span class="change-total"><span class="positive">+${add}</span><span class="negative">−${del}</span></span><div class="toolbar-group"><div class="segmented" aria-label="Diff 视图"><button type="button" data-diff="unified" class="${!state.split?'active':''}" aria-pressed="${!state.split}">统一</button><button type="button" data-diff="split" class="${state.split?'active':''}" aria-pressed="${state.split}">左右</button></div>${tool('copy-path','','copy','icon-only','复制文件路径')}</div></div><div class="review-content"><div class="file-list" aria-label="变更文件"><div class="file-group-title"><span>${fs.length} 个变更文件</span><span>与第一父提交比较</span></div>${fs.map((file,i)=>`<button type="button" class="file-row ${i===state.file?'active':''}" data-file="${i}" title="${escape(file.path)}">${icon(file.path.startsWith('modules')?'box':'file')}<span class="file-name">${escape(basename(file.path))}<small class="file-dir">${escape(dirname(file.path))}</small></span><span class="file-status">${file.status}</span></button>`).join('')}</div><div class="code-panel"><div class="code-path">${icon('file')}<span class="path-label">${escape(f.path)}</span>${state.scene==='workspace'?tool('stage-file',f.staged?'取消暂存文件':'暂存文件',f.staged?'minus':'plus'): '<span style="margin-left:auto;font-family:var(--body)">只读</span>'}</div><div class="code-scroll ${state.split?'split-scroll':''}">${diffContent()}</div></div></div></section>`;}
  function working(){return `<section class="history-pane" aria-label="工作区文件"><div class="workspace-heading"><h2>工作区</h2>${tool('stash','','stash','icon-only','保存到 Stash')}</div><div class="working-files">${[false,true].map(staged=>{const group=workingFiles.filter(f=>f.staged===staged);return `<div class="working-group">${icon('down')}${staged?'已暂存':'未暂存'}<span>${group.length}</span></div>${group.map(f=>{const idx=workingFiles.indexOf(f);return `<div class="working-file ${idx===state.working?'active':''}" data-working="${idx}" tabindex="0" role="button" aria-label="查看 ${escape(f.path)}">${icon(f.path.startsWith('modules')?'box':'file')}<div class="working-name">${escape(basename(f.path))}<span>${escape(dirname(f.path))}</span></div><span class="file-status">${f.status}</span><button type="button" class="file-stage-button" data-stage="${idx}" title="${staged?'取消暂存':'暂存'} ${escape(f.path)}">${icon(staged?'minus':'plus')}</button></div>`;}).join('')}${group.length?'':'<p class="empty-staged">选择文件或片段后暂存。</p>'}`;}).join('')}</div></section>`;}
  function compose(){const count=workingFiles.filter(f=>f.staged).length;return `<aside class="inspector"><div class="compose"><div class="compose-label">${icon('commit')}提交到 <strong>feature/partial-stage</strong></div><input id="commit-title" aria-label="提交标题" placeholder="提交标题" value="${escape(state.commitTitle)}"><textarea id="commit-body" aria-label="提交说明" placeholder="补充说明（可选）">${escape(state.commitBody)}</textarea><div class="compose-footer"><span>${count} 个已暂存文件</span><button class="tool accent" data-action="commit" ${count && state.commitTitle.trim()?'':'disabled'}>提交暂存内容 <small>⌘ ↵</small></button></div></div></aside>`;}
  function moduleList(){return `<section class="history-pane"><div class="workspace-heading"><h2>子模块</h2><span class="subtle">3</span></div><div class="module-list">${modules.map(m=>`<button type="button" class="module-item ${state.module===m.id?'active':''}" data-module="${m.id}">${icon('box')}<span><strong>${m.name}</strong><small>${m.path}</small><span class="module-stamp"><i></i>${m.state}${m.id==='core'?' / 内部修改':''}</span></span></button>`).join('')}</div></section>`;}
  function moduleDetail(){const m=modules.find(m=>m.id===state.module);return `<section class="review-pane"><div class="module-detail"><div class="module-detail-top"><div><h2>${m.name}</h2><p class="module-path">${m.path}</p></div><span class="module-state">${m.state}</span></div><p class="module-explanation">${m.body}</p><div class="pointer-table" aria-label="子模块三个版本指针">${[['父提交记录','commit',m.recorded,'父仓库 HEAD 中的 gitlink'],['暂存区记录','stash',m.index,'下一次父仓库提交会记录的版本'],['子模块当前版本','box',m.current,m.branch]].map(([label,ico,sha,desc])=>`<div class="pointer-row"><span>${icon(ico)}${label}</span><code>${sha||'尚未检出'}</code><small>${desc}</small></div>`).join('')}</div>${m.id==='core'?'<div class="pointer-delta"><strong>当前版本前进 2 个提交。</strong><br>暂存指针后，父仓库会记录 e92f3a7；子模块内部未提交的内容仍保留在子模块里。</div>':''}<div class="module-actions">${tool('module-open',m.current?'进入子模块':'初始化子模块',m.current?'arrow':'fetch','accent')}${m.id==='core'?tool('module-stage','暂存指针更新','plus','outlined'):''}${m.current?tool('module-checkout','检出暂存区版本','history','outlined'):''}</div><h3 class="module-change-title">${m.changes.length?'需要关注':'工作区状态'}</h3>${m.changes.length?m.changes.map(([path,status,description])=>`<div class="module-change"><code>${path}</code><span>${status}</span><span>${description}</span></div>`).join(''):'<p class="module-explanation">'+(m.current?'没有内部改动，指针记录一致。':'初始化前不进行子模块内部状态检查。')+'</p>'}</div></section>`;}
  function referenceRail(){return `<aside class="reference-rail" aria-label="引用地图">${refs()}<div class="sidebar-section"><div class="sidebar-title">${icon('box')}子模块</div><button class="ref-item" data-module="core"><span>modules/core</span><small>↑2</small></button><button class="ref-item" data-module="theme"><span>modules/theme</span></button></div><div class="rail-context"><strong>${icon('worktree')} 主工作区</strong>${workingFiles.filter(f=>!f.staged).length} 个文件未暂存<br>core 指针前进 2 个提交</div></aside>`;}
  function statusbar(){return `<footer class="statusbar"><span class="status-dot"></span><span>示例仓库</span><span class="optional">${state.scene==='history'?'历史只读':state.scene==='workspace'?workingFiles.filter(f=>!f.staged).length+' 个文件未暂存':'子模块状态独立检查'}</span><div class="status-spacer"></div><span class="optional">${icon('branch')} feature/partial-stage</span><span>${state.scene==='history'?'<kbd>↑ ↓</kbd> 选择提交':state.scene==='workspace'?'<kbd>⌘ ↵</kbd> 提交暂存内容':'HEAD / Index / 当前版本'}</span></footer>`;}
  function render(){
    root.className='desktop '+state.direction;root.dataset.scene=state.scene;
    root.innerHTML=titlebar()+`<div class="workbench">${navigation()}${state.scene==='history'?history()+inspector()+review():state.scene==='workspace'?working()+compose()+review():moduleList()+moduleDetail()}${referenceRail()}</div>`+statusbar();
    renderRows();
    const d=directions[state.direction];document.getElementById('direction-title').textContent=d.title;document.getElementById('direction-summary').textContent=d.summary;document.getElementById('direction-layout').textContent=d.layout;
    document.querySelectorAll('[data-direction]').forEach(b=>b.setAttribute('aria-pressed',String(b.dataset.direction===state.direction)));
    document.documentElement.style.setProperty('--focus',{alloy:'#276f68',night:'#456fa5',loom:'#74558d'}[state.direction]);
  }
  let toastTimer;
  function toast(text){const el=document.getElementById('toast');el.textContent=text;el.hidden=false;clearTimeout(toastTimer);toastTimer=setTimeout(()=>{el.hidden=true;},3200);}
  let dialogResult=null;
  function dialog(title,text,info,result,button='预览结果'){
    document.getElementById('dialog-title').textContent=title;document.getElementById('dialog-copy').textContent=text;document.getElementById('dialog-info').textContent=info;document.getElementById('dialog-confirm').textContent=button;document.getElementById('dialog-mark').innerHTML=icon('branch');dialogResult=result;document.getElementById('operation-dialog').showModal();
  }
  document.getElementById('operation-dialog').addEventListener('close',e=>{if(e.target.returnValue==='confirm' && dialogResult)dialogResult();dialogResult=null;});
  async function copy(text){try{await navigator.clipboard.writeText(text);toast('已复制到剪贴板');}catch{dialog('复制内容','可以选择并复制下面的文字。',text,null,'关闭');}}
  function toggleStage(i){workingFiles[i].staged=!workingFiles[i].staged;render();toast('演示：'+(workingFiles[i].staged?'已暂存 ':'已取消暂存 ')+workingFiles[i].path);}
  const actions={
    'graph-mode':()=>{state.table=!state.table;render();},
    'copy-sha':()=>copy(fullSha(commit().sha)),
    'copy-path':()=>copy(activeFile().path),
    search:()=>{if(state.scene!=='history'){state.scene='history';render();}document.getElementById('commit-search').focus();},
    refresh:()=>toast('示例状态已刷新；未执行 Git 命令'),
    appearance:()=>toast('当前为「'+({alloy:'铂金',night:'夜航',loom:'织线'}[state.direction])+'」设计方向，可在页面顶部切换'),
    repository:()=>dialog('atlas','示例仓库，用于比较三种界面方向。所有仓库操作仅在页面内演示。','主工作区\nfeature/partial-stage\n4 个未暂存文件，1 个已暂存文件',null,'返回工作台'),
    fetch:()=>dialog('获取远端更新','演示获取远端引用。此操作不切换分支，也不改动工作区。','远端 origin\n目标 refs/remotes/origin/*',()=>toast('演示：远端引用已更新')),
    pull:()=>dialog('拉取当前分支','仅允许快进；出现分叉时停止。当前示例已有本地改动，拉取前需要先处理。','origin/feature/partial-stage\n→ feature/partial-stage\n工作区有未提交改动',()=>toast('演示：工作区有改动，拉取已停止'),'查看检查结果'),
    push:()=>dialog('推送当前分支','确认具体源分支与远端目标。先检查子模块提交是否可达。','feature/partial-stage → origin\nrefs/heads/feature/partial-stage\n待推送 2 个提交',()=>toast('演示：子模块 core 有未发布提交，推送已停止'),'检查推送'),
    'new-branch':()=>dialog('从此提交创建分支','创建动作与当前选择绑定，不跟随之后的提交选择改变。','当前提交 '+commit().sha+'\n建议名称 review/partial-stage',()=>toast('演示：已创建 review/partial-stage')),
    'commit-menu':()=>dialog('提交操作','从这一提交创建分支，或将更改应用到当前分支。写入动作在实际执行前检查工作区。',commit().title+'\n'+fullSha(commit().sha),null,'返回'),
    stash:()=>dialog('保存到 Stash','保存未提交的工作区和暂存区内容。此设计稿仅预览确认流程。','atlas\nfeature/partial-stage\n保留原有部分暂存关系',()=>toast('演示：已保存 Stash')),
    'stage-file':()=>toggleStage(state.working),
    'stage-hunk':()=>{const f=activeFile();if(!f.staged){f.staged=true;render();toast('演示：已暂存所示片段，其他文件改动保留');}else{f.staged=false;render();toast('演示：已取消暂存所示片段');}},
    commit:()=>dialog('提交暂存内容','仅提交已暂存的文件内容。未暂存文件与子模块内部改动继续保留。',state.commitTitle+'\n\n'+workingFiles.filter(f=>f.staged).map(f=>f.path).join('\n'),()=>toast('演示：提交完成。原型保留示例改动，方便继续比较。'),'演示提交'),
    'module-open':()=>{const m=modules.find(m=>m.id===state.module);dialog(m.current?'进入子模块':'初始化子模块',m.current?'子模块拥有独立的历史、暂存区与提交草稿。入口保留返回父仓库的路径。':'检出父仓库暂存区所记录的版本，不跟随远端最新提交。',m.path+'\n'+(m.current?m.branch:'目标提交 '+m.index),()=>toast('演示：'+(m.current?'打开 '+m.path:'初始化到 '+m.index)));},
    'module-stage':()=>{const m=modules.find(m=>m.id===state.module);dialog('暂存子模块指针','仅将父仓库暂存区中的 gitlink 更新到当前子模块提交。子模块内部文件继续保留。',m.path+'\n'+m.index+' → '+m.current,()=>{m.index=m.current;render();toast('演示：已暂存子模块指针更新');});},
    'module-checkout':()=>{const m=modules.find(m=>m.id===state.module);dialog('检出暂存区版本','检查子模块内部改动。存在未提交内容时停止，避免覆盖。',m.path+'\n目标 '+m.index+'\n'+(m.id==='core'?'存在内部改动':'工作区干净'),()=>toast(m.id==='core'?'演示：存在内部改动，检出已停止':'演示：当前版本与暂存区记录一致'),'查看检查结果');},
  };
  document.addEventListener('click',e=>{
    const direction=e.target.closest('[data-direction]');if(direction){state.direction=direction.dataset.direction;location.hash=state.direction;render();return;}
    const stage=e.target.closest('[data-stage]');if(stage){toggleStage(Number(stage.dataset.stage));return;}
    const scene=e.target.closest('[data-scene]');if(scene && scene.tagName==='BUTTON'){state.scene=scene.dataset.scene;render();return;}
    const module=e.target.closest('[data-module]');if(module){state.module=module.dataset.module;state.scene='modules';render();return;}
    const ref=e.target.closest('[data-ref]');if(ref){state.selected=ref.dataset.ref;state.scene='history';state.file=0;state.query='';render();document.querySelector('[data-commit="'+state.selected+'"]')?.scrollIntoView({block:'nearest'});return;}
    const c=e.target.closest('[data-commit]');if(c){state.selected=c.dataset.commit;state.file=0;const scroll=document.getElementById('history-rows')?.scrollTop||0;render();document.getElementById('history-rows').scrollTop=scroll;return;}
    const f=e.target.closest('[data-file]');if(f){state.file=Number(f.dataset.file);render();return;}
    const w=e.target.closest('[data-working]');if(w){state.working=Number(w.dataset.working);render();return;}
    const diff=e.target.closest('[data-diff]');if(diff){state.split=diff.dataset.diff==='split';render();return;}
    const line=e.target.closest('[data-select-line]');if(line){line.classList.toggle('selected-line');return;}
    const action=e.target.closest('[data-action]');if(action && !action.disabled) actions[action.dataset.action]?.();
  });
  document.addEventListener('input',e=>{
    if(e.target.id==='commit-search'){state.query=e.target.value;renderRows();}
    if(e.target.id==='commit-title'){state.commitTitle=e.target.value;const button=document.querySelector('[data-action="commit"]');button.disabled=!state.commitTitle.trim()||!workingFiles.some(f=>f.staged);}
    if(e.target.id==='commit-body')state.commitBody=e.target.value;
  });
  document.addEventListener('keydown',e=>{
    if(document.getElementById('operation-dialog').open)return;
    if((e.metaKey||e.ctrlKey) && e.key.toLowerCase()==='f'){e.preventDefault();actions.search();return;}
    if((e.metaKey||e.ctrlKey) && e.key==='Enter' && state.scene==='workspace'){e.preventDefault();if(state.commitTitle.trim()&&workingFiles.some(f=>f.staged))actions.commit();return;}
    if(e.target.matches('input,textarea'))return;
    const clickable=e.target.closest('[data-working],[data-select-line]');if(clickable && (e.key==='Enter'||e.key===' ')){e.preventDefault();clickable.click();return;}
    if(state.scene==='history' && (e.key==='ArrowDown'||e.key==='ArrowUp')){const rows=[...document.querySelectorAll('[data-commit]')];const i=rows.findIndex(b=>b.dataset.commit===state.selected);const next=rows[Math.max(0,Math.min(rows.length-1,i+(e.key==='ArrowDown'?1:-1)))];if(next){e.preventDefault();next.click();document.querySelector('[data-commit="'+state.selected+'"]')?.focus({preventScroll:true});document.querySelector('[data-commit="'+state.selected+'"]')?.scrollIntoView({block:'nearest'});}}
  });
  function route(){const d=location.hash.slice(1);if(directions[d])state.direction=d;render();}
  window.addEventListener('hashchange',()=>{if(location.hash.slice(1)!==state.direction)route();});
  route();
})();
