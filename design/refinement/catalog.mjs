import {tracks as priorTracks,settings,handoffs} from '../roadmap/catalog.mjs';
export {settings,handoffs};
export const tracks=structuredClone(priorTracks);
const workspace=tracks.find(t=>t.id==='B');
workspace.frames.find(f=>f.id==='B2').title='选择四行，等待暂存';
workspace.frames.find(f=>f.id==='B2').notes=['选择只建立范围，不修改 Index；“暂存 4 行”才执行写入。','成功后再更新已暂存计数；外部变化时拒绝旧补丁并保留草稿。'];
workspace.edges[0]=['B1','B2','选择片段或行'];
workspace.edges[1]=['B2','B3','暂存成功后提交'];
export const extras=[
 {id:'U1',title:'列表悬停与键盘焦点',kind:'file-focus',stage:'R2',capability:'交互细节提案',notes:['文字、图标与状态列各用固定锚点；悬停操作不挤动文件名。','键盘焦点和选中分别表达；Tab 进入操作，Enter 打开，Esc 退出菜单。'],states:['默认','悬停','焦点']},
 {id:'U2',title:'读取失败，保留已读内容',kind:'read-error',stage:'R1',capability:'已有错误保护能力的新呈现',notes:['局部失败贴近当前 Diff；保留文件列表与提交草稿。','重试针对原仓库与文件；错误摘要与原始输出分层。'],states:['读取中','读取失败','重试']},
 {id:'U3',title:'危险确认与具体对象',kind:'danger-dialog',stage:'R1',capability:'已有删除动作的确认设计',notes:['明确删除的是 Stash 条目，保留分支；不可逆动作使用独立危险色。','取消与删除分开；确认不会默认聚焦危险操作。'],states:['待确认','取消','执行结果']},
 {id:'U4',title:'上下文菜单与禁用原因',kind:'context-menu',stage:'R2',capability:'已有动作的交互细节提案',notes:['右键菜单复用页面中的 Git 动作，不引入新操作。','被禁止的动作显示原因；Esc 关闭后焦点返回原对象。'],states:['菜单展开','禁用说明','键盘退出']},
 {id:'U5',title:'提交执行中与结果归属',kind:'commit-busy',stage:'R1',capability:'已有写入串行化的新呈现',notes:['执行中固定目标仓库，防止重复提交；标题与 Index 内容已经确认。','没有安全取消能力时不显示虚假的取消写入按钮；等真实结果。'],states:['运行 hooks','等待 Git 退出','成功或失败']},
 {id:'U6',title:'空结果与可恢复搜索',kind:'search-empty',stage:'R1',capability:'已有搜索的新呈现',notes:['明确搜索范围为已载入历史；空结果与读取失败分开。','清除搜索后恢复原列表与选择，支持继续加载。'],states:['空结果','清除搜索','恢复原选择']}
];
