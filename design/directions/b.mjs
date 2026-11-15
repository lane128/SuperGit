// Direction B: technical editorial.
import {
  B as p, W, H, text, centerText, paragraph, rect, line, path, circle, icon,
  pill, button, iconButton, input, checkbox, graph, commits, diffRows, content, estimate, fit, fitPath,
} from './primitives.mjs';

const sans = 'b';

function wordmark() {
  return rect(20, 20, 8, 8, p.accent, 'none', 1)
    + text(36, 24, 'SuperGit', { size: 16, color: p.ink, weight: 600, maxWidth: 100, family: 'serif' });
}

function tabs(items, activeIndex) {
  let s = rect(0, 48, W, 36, p.bg) + line(0, 84, W, 84, p.line);
  let x = 76;
  items.forEach((label, i) => {
    const w = estimate(label, 13, sans) + 30;
    const active = i === activeIndex;
    s += text(x + 15, 66, label, { size: 13, color: active ? p.ink : p.secondary, weight: active ? 600 : 400, maxWidth: w - 22, family: sans });
    if (active) s += rect(x + 10, 81, w - 20, 2, p.accent, 'none', 1);
    x += w;
  });
  return s;
}

function rail(active) {
  const items = [['changes', 'changes'], ['history', 'history'], ['box', 'box'], ['branch', 'branch'], ['remote', 'remote'], ['stash', 'stash']];
  let s = rect(0, 48, 56, H - 48, p.shell) + line(55, 48, 55, H, p.line);
  items.forEach(([label, ico], i) => {
    const y = 64 + i * 44, on = label === active;
    if (on) s += rect(8, y - 8, 40, 32, p.selected, 'none', 2) + rect(0, y - 8, 2, 32, p.accent, 'none', 1);
    s += icon(20, y - 1, ico, on ? p.accent : p.secondary, 18, 1.7);
  });
  s += icon(20, 640, 'settings', p.secondary, 18, 1.7);
  return s;
}

function topbar(rightText) {
  let s = rect(0, 0, W, 48, p.surface) + line(0, 48, W, 48, p.line);
  s += wordmark();
  s += line(128, 14, 128, 34, p.line);
  s += text(142, 24, 'atlas / 主工作区', { size: 11, color: p.secondary, maxWidth: 260, family: sans });
  s += rect(820, 12, 44, 24, p.bg, p.line, 3);
  s += centerText(842, 24, '⌘K', 11, p.secondary, 500, 38, sans);
  s += text(872, 24, '命令', { size: 11, color: p.muted, maxWidth: 40, family: sans });
  s += text(1024, 24, rightText, { size: 11, color: p.secondary, maxWidth: 120, anchor: 'end', family: sans });
  return s;
}

function statusBar() {
  let s = rect(0, 676, W, 24, p.surface) + line(0, 676, W, 676, p.line);
  s += text(16, 688, 'atlas / feature/partial-stage', { size: 11, color: p.ink, weight: 500, maxWidth: 300, family: sans });
  s += text(420, 688, '↑2 ↓0 · origin/feature/partial-stage', { size: 11, color: p.secondary, maxWidth: 360, family: sans });
  s += text(1024, 688, 'Index 1 · Worktree 3', { size: 11, color: p.secondary, maxWidth: 180, anchor: 'end', family: sans });
  return s;
}

function chrome(active, tabItems, activeTab, rightText) {
  let s = rect(0, 0, W, H, p.bg);
  s += topbar(rightText);
  s += tabs(tabItems, activeTab);
  s += rail(active);
  s += statusBar();
  return s;
}

function sectionLabel(x, y, w, label, count, action) {
  return text(x, y, label, { size: 11, color: p.secondary, weight: 600, maxWidth: 90, family: sans, tracking: 0.2 })
    + text(x + estimate(label, 11, sans) + 10, y, String(count), { size: 11, color: p.accent, weight: 600, maxWidth: 26, family: sans })
    + text(x + w, y, action, { size: 11, color: p.accent, weight: 500, maxWidth: 90, anchor: 'end', family: sans });
}

function fileRowB(x, y, w, file, { selected = false, action = '', staged = false } = {}) {
  const actionX = x + w - 18, statusX = actionX - 46, typeX = statusX - 32, textX = x + 34;
  const nameMax = Math.min(150, typeX - textX - 8);
  const shownName = fit(file.name, nameMax, 13, sans);
  const nameW = estimate(shownName, 13, sans);
  const pathMax = Math.max(0, typeX - (textX + nameW + 8) - 4);
  const shownPath = pathMax > 24 ? fitPath(file.path.endsWith('/') ? file.path : file.path + '/', pathMax, 11, sans) : '';
  const statusColor = file.status === 'M' ? p.warning : file.status === 'A' ? p.good : file.status === 'D' ? p.danger : p.accent;
  let s = `<g data-component="file-row" data-label="${file.name}">`;
  if (selected) s += rect(x, y, w, 28, p.selected, 'none', 2);
  s += rect(x, y + 4, 2, 20, statusColor, 'none', 1);
  s += icon(x + 12, y + 7, file.type === 'sub' ? 'box' : 'file', p.secondary, 14, 1.6);
  s += text(textX, y + 14, shownName, { size: 13, color: p.ink, weight: selected ? 600 : 500, maxWidth: nameMax, family: sans });
  if (shownPath) s += text(textX + nameW + 8, y + 14, shownPath, { size: 11, color: p.muted, maxWidth: pathMax, family: sans });
  s += text(typeX, y + 14, file.type, { size: 11, color: p.muted, maxWidth: 28, family: sans });
  s += centerText(statusX + 9, y + 14, file.status, 11, statusColor, 600, 16, sans);
  if (action) s += text(actionX, y + 14, action === 'stage' ? '暂存' : '取消', { size: 11, color: action === 'stage' ? p.accent : p.secondary, weight: 500, maxWidth: 34, anchor: 'end', family: sans });
  return s + '</g>';
}

function ledgerB() {
  const x = 68, w = 302;
  let s = sectionLabel(76, 166, 302, '未暂存', 3, '全部暂存');
  content.unstaged.forEach((file, i) => { s += fileRowB(x, 178 + i * 28, w, file, { selected: i === 0, action: i === 0 ? 'stage' : '' }); });
  s += sectionLabel(76, 270, 302, '已暂存', 1, '全部取消');
  s += fileRowB(x, 282, w, content.staged[0], { action: 'unstage' });
  s += line(56, 316, 386, 316, p.strong);
  s += line(56, 320, 386, 320, p.line);
  s += text(76, 338, '提交说明', { size: 11, color: p.secondary, weight: 600, maxWidth: 120, family: sans });
  s += input(x, 346, w, 28, content.subject, p, { size: 12, family: sans, radius: 2 });
  s += text(76, 390, '补充说明', { size: 11, color: p.secondary, weight: 600, maxWidth: 120, family: sans });
  s += input(x, 398, w, 44, '可选，提交正文', p, { placeholder: true, size: 12, family: sans, radius: 2 });
  s += text(76, 458, '1 个文件 · 将写入 Index', { size: 11, color: p.secondary, maxWidth: 220, family: sans });
  s += button(x, 466, w, 30, '提交 1 个文件', p, { primary: true, size: 12, family: sans, radius: 2 });
  s += text(76, 512, '未暂存的 3 个文件保留在工作区。', { size: 11, color: p.muted, maxWidth: w, family: sans });
  s += line(56, 534, 386, 534, p.line);
  s += text(76, 550, '上游', { size: 11, color: p.secondary, weight: 600, maxWidth: 40, family: sans });
  s += text(116, 550, 'origin/feature/partial-stage', { size: 11, color: p.ink, maxWidth: 220, family: sans });
  s += text(76, 572, '状态', { size: 11, color: p.secondary, weight: 600, maxWidth: 40, family: sans });
  s += text(116, 572, 'Index 1 · Worktree 3 · 冲突 0', { size: 11, color: p.ink, maxWidth: 220, family: sans });
  s += text(76, 594, 'HEAD', { size: 11, color: p.secondary, weight: 600, maxWidth: 40, family: sans });
  s += text(116, 594, 'b71f9c2', { size: 11, color: p.ink, maxWidth: 70, family: sans, mono: true });
  s += text(76, 626, '提交前检查', { size: 11, color: p.accent, weight: 500, maxWidth: 100, family: sans });
  return s;
}

function diffB() {
  const x = 386, y = 148, w = 654;
  let s = line(x, y, x, 676, p.line);
  s += rect(x, y, w, 40, p.surface);
  s += text(x + 20, y + 20, content.diffPath, { size: 12, color: p.ink, weight: 500, maxWidth: 360, family: sans, mono: true });
  s += text(x + w - 120, y + 20, '统一', { size: 12, color: p.ink, weight: 600, maxWidth: 40, anchor: 'end', family: sans });
  s += text(x + w - 70, y + 20, '左右', { size: 12, color: p.secondary, maxWidth: 40, anchor: 'end', family: sans });
  s += rect(x + w - 126, y + 37, 44, 2, p.accent, 'none', 1);
  s += iconButton(x + w - 28, y + 8, 24, 'more', p, { color: p.secondary });
  s += line(x, y + 40, x + w, y + 40, p.line);
  s += rect(x, y + 40, w, 30, p.raised);
  s += text(x + 20, y + 55, content.hunk, { size: 11, color: p.secondary, maxWidth: 300, family: sans, mono: true });
  s += pill(x + w - 270, y + 45, 'Index → 工作区', p, { fill: p.surface, stroke: p.line, color: p.secondary, height: 20, size: 11, radius: 2, maxWidth: 96 });
  s += text(x + w - 20, y + 55, '暂存片段', { size: 11, color: p.accent, weight: 500, maxWidth: 80, anchor: 'end', family: sans });
  s += line(x, y + 70, x + w, y + 70, p.line);
  const top = y + 70, lh = 22, gutter = 78;
  s += rect(x, top, gutter, diffRows.length * lh, p.raised);
  diffRows.forEach((r, i) => {
    const yy = top + i * lh;
    if (r.kind !== 'ctx') s += rect(x, yy, w, lh, r.kind === 'add' ? p.add : p.del);
    s += text(x + 10, yy + 11, r.oldNo, { size: 11, color: p.muted, maxWidth: 24, family: sans, mono: true });
    s += text(x + 38, yy + 11, r.newNo, { size: 11, color: p.muted, maxWidth: 24, family: sans, mono: true });
    s += centerText(x + 66, yy + 11, r.kind === 'add' ? '+' : r.kind === 'del' ? '−' : '', 12, r.kind === 'add' ? p.good : r.kind === 'del' ? p.danger : p.muted, 500, 14, sans);
    s += text(x + gutter + 14, yy + 11, r.code, { size: 12, color: p.ink, maxWidth: w - gutter - 24, family: sans, mono: true });
  });
  s += line(x + gutter, top, x + gutter, top + diffRows.length * lh, p.line);
  const bottom = top + diffRows.length * lh;
  s += text(x + 20, bottom + 16, '8 行 · +4 −1', { size: 11, color: p.secondary, maxWidth: 200, family: sans });
  s += text(x + w - 20, bottom + 16, '下一个改动 →', { size: 11, color: p.accent, weight: 500, maxWidth: 100, anchor: 'end', family: sans });
  s += line(x + 20, bottom + 32, x + w - 20, bottom + 32, p.line);
  s += text(x + 20, bottom + 54, '相关提交', { size: 11, color: p.secondary, weight: 600, maxWidth: 120, family: sans });
  content.fileHistory.forEach((r, i) => {
    const yy = bottom + 76 + i * 24;
    s += text(x + 20, yy, r.sha, { size: 11, color: p.ink, weight: 500, maxWidth: 60, family: sans, mono: true });
    s += text(x + 86, yy, r.when, { size: 11, color: p.muted, maxWidth: 70, family: sans });
    s += text(x + 158, yy, r.subject, { size: 11, color: p.secondary, maxWidth: 430, family: sans });
  });
  s += text(x + 20, bottom + 148, '本地修改，尚未进入 Index。', { size: 11, color: p.muted, maxWidth: 400, family: sans });
  return s;
}

export function workspaceB() {
  let s = chrome('changes', ['全部改动 3', '已暂存 1', '未跟踪 0'], 0, '3 未暂存 · 1 已暂存');
  s += text(76, 112, '工作区', { size: 26, color: p.ink, weight: 600, maxWidth: 260, family: 'serif' });
  s += text(76, 134, '3 个未暂存文件，1 个已暂存', { size: 11, color: p.secondary, maxWidth: 360, family: sans });
  s += text(1024, 112, content.branch, { size: 13, color: p.ink, weight: 600, maxWidth: 220, anchor: 'end', family: sans });
  s += text(1024, 134, `${content.upstream} · ↑2 ↓0`, { size: 11, color: p.secondary, maxWidth: 300, anchor: 'end', family: sans });
  s += line(56, 148, 1040, 148, p.strong);
  s += ledgerB();
  s += diffB();
  return s;
}

function historyTableB() {
  const x = 56, w = 700, top = 174;
  let s = rect(x, 148, w, 26, p.raised);
  const cols = [['线图', 64], ['提交说明与引用', 214], ['作者', 544], ['时间', 616], ['SHA', 680]];
  cols.forEach(([label, cx]) => { s += text(cx, 161, label, { size: 11, color: p.secondary, weight: 600, maxWidth: 100, family: sans, tracking: 0.2 }); });
  s += line(x, 174, x + w, 174, p.line);
  s += graph(x, top, { row: 28, lane: 18, style: 'orthogonal', p, selected: 0, colors: [p.main, p.fix, p.topic, p.topic2], width: 150, edge: 1.5, node: 3.5, mergeNode: 4.6 });
  commits.forEach((c, i) => {
    const y = top + i * 28 + 14;
    if (i === 0) s += rect(x, y - 14, w, 28, p.selected, 'none', 0) + rect(x, y - 12, 2, 24, p.accent, 'none', 1);
    const refs = c.refs.filter((r) => r !== 'HEAD' && r !== content.branch);
    const refW = refs.reduce((n, r) => n + Math.min(126, estimate(r, 11, sans) + 14) + 4, 0);
    const subjectMax = Math.max(120, 310 - refW - 8);
    s += text(214, y, c.subject, { size: 13, color: p.ink, weight: i === 0 ? 600 : 400, maxWidth: subjectMax, family: sans });
    let rx = 214 + estimate(fit(c.subject, subjectMax, 13, sans), 13, sans) + 8;
    refs.forEach((ref) => {
      const rw = Math.min(126, estimate(ref, 11, sans) + 14);
      s += pill(rx, y - 9, ref, p, { fill: p.surface, stroke: p.strong, color: p.secondary, height: 18, size: 11, radius: 2, maxWidth: rw });
      rx += rw + 4;
    });
    s += text(544, y, 'dev', { size: 11, color: p.secondary, maxWidth: 60, family: sans });
    s += text(616, y, i < 2 ? '今天 10:42' : '昨天', { size: 11, color: p.muted, maxWidth: 58, family: sans });
    s += text(680, y, c.sha, { size: 11, color: p.secondary, maxWidth: 46, family: sans, mono: true });
    if (i === 0) s += icon(732, y - 7, 'more', p.secondary, 14, 1.6);
    s += line(x, y + 14, x + w, y + 14, i === 0 ? p.strong : p.line);
  });
  s += text(x + w - 8, 526, '加载更早的提交 →', { size: 11, color: p.accent, weight: 500, maxWidth: 150, anchor: 'end', family: sans });
  return s;
}

function inspectorB() {
  const x = 756, w = 284;
  let s = rect(x, 148, w, H - 148, p.surface) + line(x, 148, x, H, p.line);
  s += text(776, 168, '选中提交', { size: 11, color: p.secondary, weight: 600, maxWidth: 120, family: sans, tracking: 0.2 });
  s += paragraph(776, 194, 'feat: preserve partial staging', 244, { size: 15, color: p.ink, lineHeight: 20, maxLines: 2, family: 'serif' });
  s += text(776, 224, 'b71f9c2 · dev · 今天 10:42', { size: 11, color: p.secondary, maxWidth: 240, family: sans, mono: true });
  s += pill(776, 240, 'HEAD', p, { fill: p.accent, stroke: p.accent, color: p.onAccent, height: 20, size: 11, radius: 2, maxWidth: 44 });
  s += pill(826, 240, content.branch, p, { fill: p.surface, stroke: p.strong, color: p.secondary, height: 20, size: 11, radius: 2, maxWidth: 180 });
  s += line(776, 272, 1020, 272, p.line);
  s += text(776, 292, '变更文件 2', { size: 11, color: p.secondary, weight: 600, maxWidth: 120, family: sans });
  s += icon(776, 306, 'file', p.secondary, 12, 1.6) + text(794, 312, 'internal/gitrepo/stage.go', { size: 11, color: p.ink, maxWidth: 180, family: sans });
  s += text(1020, 312, 'M', { size: 11, color: p.warning, weight: 600, maxWidth: 14, anchor: 'end', family: sans });
  s += icon(776, 328, 'file', p.secondary, 12, 1.6) + text(794, 334, 'internal/desktop/workspace.go', { size: 11, color: p.ink, maxWidth: 180, family: sans });
  s += text(1020, 334, 'M', { size: 11, color: p.warning, weight: 600, maxWidth: 14, anchor: 'end', family: sans });
  s += text(776, 366, '父提交', { size: 11, color: p.secondary, weight: 600, maxWidth: 60, family: sans });
  s += text(776, 386, 'a43bc18 · Preserve commit message draft', { size: 11, color: p.ink, maxWidth: 244, family: sans });
  s += button(776, 410, 244, 28, '完整提交详情', p, { size: 12, family: sans, radius: 2 });
  s += text(776, 458, '复制 SHA', { size: 11, color: p.accent, weight: 500, maxWidth: 80, family: sans });
  s += text(870, 458, '在终端中打开', { size: 11, color: p.accent, weight: 500, maxWidth: 120, family: sans });
  s += line(776, 480, 1020, 480, p.line);
  s += text(776, 500, '提交操作', { size: 11, color: p.secondary, weight: 600, maxWidth: 90, family: sans });
  ['新建分支…', '打标签…', 'Cherry-pick…', 'Revert…'].forEach((label, i) => {
    s += text(776, 522 + i * 24, label, { size: 11, color: label === 'Revert…' ? p.danger : p.ink, maxWidth: 160, family: sans });
  });
  s += line(776, 616, 1020, 616, p.line);
  s += text(776, 636, '签名', { size: 11, color: p.secondary, weight: 600, maxWidth: 50, family: sans });
  s += text(1020, 636, '未签名', { size: 11, color: p.muted, maxWidth: 80, anchor: 'end', family: sans });
  s += text(776, 658, '提交信息可由维护者后续编辑。', { size: 11, color: p.muted, maxWidth: 244, family: sans });
  return s;
}

export function historyB() {
  let s = chrome('history', ['全部提交', '当前分支', '作者 dev', '近 7 天'], 0, '12 / 128');
  s += text(76, 112, '历史', { size: 26, color: p.ink, weight: 600, maxWidth: 160, family: 'serif' });
  s += text(76, 134, '12 个提交 · 2 个合并 · 最近更新 10:42', { size: 11, color: p.secondary, maxWidth: 420, family: sans });
  s += rect(620, 96, 300, 28, p.surface, p.strong, 2);
  s += icon(630, 103, 'search', p.muted, 14, 1.6);
  s += text(650, 110, '搜索提交、SHA 或引用', { size: 11, color: p.muted, maxWidth: 250, family: sans });
  s += iconButton(966, 98, 24, 'filter', p, { color: p.secondary });
  s += line(56, 148, 1040, 148, p.strong);
  s += historyTableB();
  s += inspectorB();
  return s;
}

export const meta = {
  id: 'B',
  name: '技术编辑风',
  line: 'Avenir Next + New York、账本式文件行、正交线图、右侧提交详情',
  notes: [
    '顶部全局模式栏 + 页内筛选，不再用宽侧栏重复仓库。',
    '名称优先，状态用左侧色脊；行高仍为 28px。',
    '历史把引用放在提交说明后，右侧面板承载提交操作。',
  ],
};
