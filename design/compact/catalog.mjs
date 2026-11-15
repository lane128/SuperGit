import {tracks as priorTracks,settings,handoffs} from '../roadmap/catalog.mjs';
export {settings,handoffs};
export const tracks=structuredClone(priorTracks);
const workspace=tracks.find(t=>t.id==='B');
workspace.frames.find(f=>f.id==='B2').title='选择行，再显式暂存';
workspace.frames.find(f=>f.id==='B2').notes=['选择只建立范围；点击“暂存 4 行”才写入 Index。','文件列表28px单行；名称、路径、类型、状态列固定。'];
workspace.edges[0]=['B1','B2','选择行'];
workspace.edges[1]=['B2','B3','暂存后提交'];
const modules=tracks.find(t=>t.id==='D');
modules.frames.find(f=>f.id==='D2').notes=['子模块与普通仓库共用页签、文件列表、Diff 与历史。','打开后是 core 自己的仓库页面，不另造一套界面。'];
export const extras=[
 {id:'U1',title:'文件行悬停与键盘焦点',kind:'file-focus',stage:'R2',capability:'ui.List / ui.Tooltip 的交互提案',notes:['28px 文件行；操作按钮占据固定槽位，不推动名称。','焦点环 2px；工具提示说明动作只作用于当前文件。'],states:['默认','悬停','键盘焦点']},
 {id:'U2',title:'Diff 读取失败，保留已读内容',kind:'read-error',stage:'R1',capability:'已有错误保护的新呈现',notes:['局部错误贴近当前 Diff；文件列表和提交草稿保留。','提示中的图标与两行文字作为一组垂直居中；操作独立靠右。'],states:['读取中','读取失败','重试']},
 {id:'U3',title:'危险动作使用 AlertDialog',kind:'danger-dialog',stage:'R1',capability:'ui.AlertDialog',notes:['红色只表达危险语义，不新造红色按钮组件。','确认与取消都使用普通按钮；具体对象和不可撤销范围写在正文。'],states:['待确认','取消','执行结果']},
 {id:'U4',title:'子模块右键菜单',kind:'context-menu',stage:'R2',capability:'ui.ContextMenu / ui.Menu',notes:['菜单复用页面已有动作；禁用项说明原因。','Esc 关闭并把焦点还给原对象。'],states:['展开','禁用说明','退出']},
 {id:'U5',title:'提交执行中',kind:'commit-busy',stage:'R1',capability:'ui.Dialog + ui.Spinner',notes:['目标与 Index 内容已固定，防止重复提交。','MyGo 按钮没有内建 loading；等待状态用 Spinner 与禁用按钮表达。'],states:['运行 hooks','等待 Git','成功或失败']},
 {id:'U6',title:'搜索无结果',kind:'search-empty',stage:'R1',capability:'ui.SearchField 的空状态',notes:['说明搜索范围，提供清除条件与继续加载。','空结果与读取失败分开，不共用错误样式。'],states:['空结果','清除搜索','恢复选择']}
];
