window.GitDesignData=(()=>{
 const row=(t,text,old=null,next=null)=>({t,text,old,next});
 const patch=(path,rows)=>({path,rows});
 const staging=patch('src/git/staging.rs',[
  row('ctx','use crate::git::{GitError, Repository};',1,1),row('ctx','',2,2),row('ctx','impl Repository {',3,3),row('ctx','    /// Stage only the selected changes.',4,4),
  row('del','    pub async fn stage_file(&self, path: &Path) {',5),row('add','    pub async fn stage_lines(',null,5),row('add','        &self,',null,6),row('add','        path: &Path,',null,7),row('add','        selection: LineSelection,',null,8),row('add','    ) -> Result<(), GitError> {',null,9),
  row('del','        self.run(&["add", path]).await;',6),row('add','        let patch = self.build_patch(path, selection)?;',null,10),row('add','        self.apply_to_index(patch).await?;',null,11),row('add','        self.refresh_status().await;',null,12),row('add','        Ok(())',null,13),row('ctx','    }',7,14),row('ctx','}',8,15)]);
 const refresh=patch('src/git/status.rs',[row('ctx','impl Repository {',1,1),row('ctx','    fn on_external_change(&mut self, event: Event) {',2,2),row('del','        self.refresh_all();',3),row('add','        self.cache.invalidate(&event.path);',null,3),row('add','        self.schedule_refresh(event.path);',null,4),row('ctx','    }',4,5),row('ctx','}',5,6)]);
 const ui=patch('src/ui/workspace.rs',[row('ctx','impl Workspace {',1,1),row('ctx','    fn on_selection_changed(&mut self) {',2,2),row('del','        self.refresh_all();',3),row('add','        self.refresh_selected_file();',null,3),row('add','        self.notify();',null,4),row('ctx','    }',4,5),row('ctx','}',5,6)]);
 const readme=patch('README.md',[row('ctx','# Sample Workspace',1,1),row('ctx','',2,2),row('del','Stage whole files from the workspace.',3),row('add','Stage files, hunks, or selected lines.',null,3),row('add','Review exactly what goes into each commit.',null,4)]);
 const index=patch('src/index.rs',[row('ctx','impl IndexCache {',1,1),row('ctx','    pub fn invalidate(&mut self, path: &Path) {',2,2),row('del','        self.entries.clear();',3),row('add','        self.entries.remove(path);',null,3),row('add','        self.generation += 1;',null,4),row('ctx','    }',4,5),row('ctx','}',5,6)]);
 const parser=patch('src/parser.rs',[row('ctx','fn parse_reference(input: &str) -> Result<Reference> {',1,1),row('del','    parse(input)',2),row('add','    parse(input.trim())',null,2),row('ctx','}',3,3)]);
 const tests=patch('tests/partial_stage.rs',[row('add','#[test]',null,1),row('add','fn stages_only_selected_lines() {',null,2),row('add','    let repo = TestRepository::new();',null,3),row('add','    repo.modify("notes.txt", "first\\nsecond");',null,4),row('add','    repo.stage_lines("notes.txt", 1..2);',null,5),row('add','    assert_eq!(repo.staged_lines(), 1);',null,6),row('add','}',null,7)]);
 const modulePatch=patch('modules/core',[row('del','Subproject commit a43bc18',1),row('add','Subproject commit e92f3a7',null,1)]);
 function file(p,staged=false,status='M'){return {...structuredClone(p),id:p.path,status,rows:p.rows.map((r,i)=>({...r,id:p.path+'#'+i,staged:r.t==='ctx'?false:staged,done:false}))}}
 const commits=[
  {sha:'b71f9c2',parents:['6d20a8e'],title:'perf: refresh only affected repository paths',body:'局部刷新受影响路径，保留文件选择和滚动位置。',author:'示例开发者',time:'12 分钟前',files:[refresh,ui]},
  {sha:'6d20a8e',parents:['c54a916'],title:'feat: show submodule pointer changes',body:'区分主仓库记录、暂存区记录与子模块当前 HEAD。',author:'示例开发者',time:'38 分钟前',files:[modulePatch,readme]},
  {sha:'c54a916',parents:['a8b1d03','8b2f19c'],title:"Merge branch 'fix/refresh'",body:'将外部修改刷新修复合入当前开发分支。\n第一父提交来自当前分支；第二父提交来自 fix/refresh。',author:'示例开发者',time:'1 小时前',files:[refresh]},
  {sha:'8b2f19c',parents:['09ce45b'],title:'fix: refresh after external changes',body:'通过文件监听刷新外部 Git 操作后的状态。',author:'示例开发者',time:'1 小时前',files:[refresh]},
  {sha:'a8b1d03',parents:['09ce45b'],title:'fix: preserve scroll position between files',body:'每个文件保留独立的 diff 阅读位置。',author:'示例开发者',time:'2 小时前',files:[ui]},
  {sha:'09ce45b',parents:['f2d783c'],title:'feat: add workspace navigation',body:'增加工作区、历史和 worktree 导航。',author:'示例开发者',time:'昨天',files:[ui,readme]},
  {sha:'f2d783c',parents:[],title:'chore: initialize application shell',body:'创建基础桌面窗口。',author:'示例开发者',time:'昨天',files:[readme]}
 ];
 const coreCommits=[{sha:'e92f3a7',parents:['f1a623b'],title:'perf: cache index entries by path',body:'按路径缓存索引，避免无关文件扫描。',author:'示例开发者',time:'25 分钟前',files:[index]},{sha:'f1a623b',parents:['a43bc18'],title:'fix: handle nested repository paths',body:'支持嵌套仓库路径。',author:'示例开发者',time:'昨天',files:[parser]},{sha:'a43bc18',parents:[],title:'feat: introduce repository core',body:'子模块示例历史的起点。',author:'示例开发者',time:'前天',files:[readme]}];
 function initial(){return {
  route:'workspace',scope:'main',selectedFile:'src/git/staging.rs',group:'unstaged',selectedLines:[],filter:'',diffMode:'unified',historySelected:'b71f9c2',historyFile:0,historyMode:'graph',branchFilter:'',branchTab:'local',selectedModule:'core',moduleTab:'overview',repoFilter:'',empty:false,focusSelection:null,
  settings:{theme:'system',density:'comfortable',accent:'teal',font:12,git:'bundled',gitPath:'/usr/bin/git',pull:'ff',tab:'appearance'},
  scopes:{main:{name:'主工作区',path:'~/Projects/sample-workspace',branch:'feature/quick-stage',ahead:2,files:[file(staging),file(ui),file(tests,false,'A'),file(readme,true)],title:'feat: support partial staging',body:''},review:{name:'review-parser',path:'~/Projects/sample-review',branch:'review/parser',ahead:0,files:[file(parser)],title:'',body:''},hotfix:{name:'fix-refresh',path:'~/Projects/sample-fix',branch:'fix/refresh',ahead:1,files:[],title:'',body:''},core:{name:'modules/core',path:'~/Projects/sample-workspace/modules/core',branch:'feature/index-cache',ahead:2,files:[file(index)],title:'perf: invalidate only affected index entries',body:''},theme:{name:'modules/theme',path:'~/Projects/sample-workspace/modules/theme',branch:null,ahead:0,files:[],title:'',body:''},tokenizer:{name:'vendor/tokenizer',path:'~/Projects/sample-workspace/modules/core/vendor/tokenizer',branch:null,ahead:0,files:[],title:'',body:''}},
  branches:[{name:'feature/quick-stage',sha:'b71f9c2',upstream:'origin/feature/quick-stage'},{name:'main',sha:'a8b1d03',upstream:'origin/main'},{name:'fix/refresh',sha:'8b2f19c',upstream:'origin/fix/refresh'},{name:'review/parser',sha:'a8b1d03',upstream:null}],
  remoteBranches:[{name:'origin/main',sha:'a8b1d03'},{name:'origin/feature/quick-stage',sha:'c54a916'},{name:'origin/fix/refresh',sha:'09ce45b'}],remotes:[{name:'origin',url:'git@example.invalid:team/sample-workspace.git'}],
  worktrees:[{id:'main',locked:true},{id:'review',locked:false},{id:'hotfix',locked:false}],
  modules:[{id:'core',path:'modules/core',parent:'main',initialized:true,pinned:'a43bc18',index:'a43bc18',head:'e92f3a7',published:'a43bc18'},{id:'tokenizer',path:'vendor/tokenizer',parent:'core',initialized:false,pinned:'13af0c1',index:'13af0c1',head:'13af0c1',published:'13af0c1'},{id:'theme',path:'modules/theme',parent:'main',initialized:false,pinned:'52b8a1d',index:'52b8a1d',head:'52b8a1d',published:'52b8a1d'}],
  commits:structuredClone(commits),coreCommits:structuredClone(coreCommits),extraCommits:[],
  stashes:[{id:1,name:'WIP: keyboard navigation',branch:'feature/quick-stage',scope:'main',time:'昨天',files:[file(ui)],includeUntracked:false}],selectedStash:1,
  operation:null,rebasePlan:[{sha:'6d20a8e',action:'pick',title:'feat: show submodule pointer changes'},{sha:'b71f9c2',action:'pick',title:'perf: refresh only affected repository paths'}],rebaseTarget:'main',rebaseConflict:true,
  conflict:{file:'src/git/status.rs',base:'fn refresh(&mut self, event: Event) {\n    self.cache.clear();\n    self.notify();\n}',ours:'fn refresh(&mut self, event: Event) {\n    self.cache.invalidate(&event.path);\n    self.notify();\n}',theirs:'fn refresh(&mut self, event: Event) {\n    self.schedule_refresh(event.path);\n}',result:'<<<<<<< 当前分支\n    self.cache.invalidate(&event.path);\n    self.notify();\n=======\n    self.schedule_refresh(event.path);\n>>>>>>> 合入分支',resolved:false},
  modal:null,networkError:null,logs:[{command:'git status --porcelain=v2 -z',result:'演示仓库已载入；未执行真实 Git 命令。'}],serial:0,lastAction:null,cloneComplete:false
 }}
 const routes=[['workspace','changes','工作区'],['history','history','历史线图'],['branches','branch','分支与远端'],['worktrees','tree','Worktrees'],['submodules','box','Submodules'],['stash','stash','Stash']];
 const scenes=[['workspace','日常改动与提交'],['history','历史线图与提交 Diff'],['submodules','子模块与引用更新'],['worktrees','多个 Worktree'],['branches','分支与远端'],['stash','Stash 检查与恢复'],['conflicts','合并冲突'],['rebase','交互式 Rebase'],['repositories','打开与克隆仓库'],['push-check','Submodule 发布检查'],['empty','干净工作区'],['error','网络错误与重试'],['guide','页面地图与验收清单']];
 const guide=[
  {title:'仓库入口',pages:'仓库列表 → 打开目录 / 克隆 → 工作区',accept:'可以搜索仓库、选择示例目录、校验克隆地址、查看克隆进度并打开新仓库。',states:'最近仓库、无匹配结果、克隆中、地址错误。'},
  {title:'查看改动与提交',pages:'工作区 → 文件 / Hunk / 行 → 暂存 → 提交 → 历史',accept:'增删行可以分别选择；暂存后两组列表同步；提交只包含暂存内容；新提交进入历史。',states:'未暂存、部分暂存、已暂存、无标题、干净工作区。'},
  {title:'历史与比较',pages:'线图 → 提交详情 → 文件 → Diff → 父提交',accept:'父引用生成连线；支持分叉、双父合并、HEAD、分支标签、统一与左右 Diff；可切换关系表。',states:'普通提交、合并提交、历史边界、子模块独立历史。'},
  {title:'分支与远端',pages:'分支列表 → 创建 / 检出 / 重命名 / 删除；远端 → URL',accept:'显示跟踪关系与 worktree 占用；有改动切换时提供保留或先 Stash；被 worktree 占用的分支不能删除。',states:'当前分支、无 upstream、其他 worktree 占用、待确认删除。'},
  {title:'Worktree',pages:'工作目录列表 → 新建 → 打开 → 移除',accept:'新 worktree 使用独立路径和分支；分别保留改动、提交草稿和阅读位置；有改动不能直接移除。',states:'主工作区、干净、有改动、路径冲突、分支已占用。'},
  {title:'Submodule',pages:'子模块树 → 引用对比 → 进入工作区 → 子模块提交 → 主仓库暂存指针',accept:'分别显示 HEAD / Index / 当前检出；内部改动独立；支持未初始化与嵌套子模块；区分记录版本和上游版本。',states:'未初始化、内部脏、指针变化、detached HEAD、未发布提交。'},
  {title:'Stash',pages:'保存 → 检查文件 Diff → 应用 / 应用后移除 / 删除',accept:'可包含未跟踪文件；应用前检查与当前改动的路径重叠；发现重叠先要求处理。',states:'空列表、独立暂存内容、应用阻塞、删除确认。'},
  {title:'冲突',pages:'操作横幅 → 三方对照 → 编辑结果 → 标记解决 → 继续 / 中止',accept:'支持采用当前、采用合入和手动编辑；结果仍含冲突标记时阻止继续；合并与 rebase 有不同的后续动作。',states:'未解决、已编辑、已标记、等待继续、已中止。'},
  {title:'Rebase',pages:'计划 → 调序 / pick / reword / squash / fixup / drop → 预检 → 执行',accept:'首条不能 squash/fixup；目标与范围清晰；可模拟中途冲突；完成后历史引用更新，保留操作恢复入口。',states:'工作区不干净、非法计划、待确认重写、冲突暂停、完成。'},
  {title:'同步与恢复',pages:'Fetch / Pull / Push → 影响预览 → 后台进度 → 成功或重试',accept:'Pull 默认 fast-forward only；Push 检查子模块提交可达性；不会将强制推送混入普通 Push；日志可展开。',states:'网络错误、子模块未发布、执行中、已完成、恢复上次本地修改。'},
  {title:'偏好与效率',pages:'外观 / Git 与同步 / 快捷键',accept:'主题、密度、字号和强调色即时生效；内置 / 系统 Git、Pull 策略可预览；⌘K 搜索操作。',states:'浅色、深色、跟随系统、紧凑、舒适。'}
 ];
 return {initial,routes,scenes,guide,row,patch,file,patches:{staging,refresh,ui,readme,index,parser,tests},coreCommits};
})();
