// Direction C: dense instrument panel.
import {
  C as p, W, H, text, centerText, paragraph, rect, line, path, circle, icon,
  pill, button, iconButton, input, checkbox, graph, commits, diffRows, content, estimate, fit, fitPath,
} from './primitives.mjs';

const sans = 'c';

function topbar() {
  let s = rect(0, 0, W, 32, p.shell) + line(0, 32, W, 32, p.line);
  s += icon(12, 9, 'box', p.accent, 14, 1.7);
  s += text(32, 16, 'atlas', { size: 12, color: p.ink, weight: 600, maxWidth: 46, family: sans });
  s += line(82, 8, 82, 24, p.line);
  s += icon(92, 10, 'branch', p.secondary, 12, 1.7);
  s += text(108, 16, content.branch, { size: 11, color: p.secondary, maxWidth: 130, family: sans });
  s += text(246, 16, '↑2', { size: 11, color: p.good, weight: 600, maxWidth: 20, family: sans });
  s += text(268, 16, '↓0', { size: 11, color: p.muted, maxWidth: 20, family: sans });
  s += rect(700, 6, 218, 20, p.surface, p.line, 3);
  s += icon(707, 10, 'search', p.muted, 12, 1.6);
  s += text(724, 16, '搜索或运行命令', { size: 11, color: p.muted, maxWidth: 120, family: sans });
  s += rect(922, 9, 30, 14, p.raised, p.line, 2);
  s += centerText(937, 16, '⌘K', 10, p.secondary, 500, 26, sans);
  s += iconButton(980, 4, 24, 'remote', p, { color: p.secondary });
  s += iconButton(1008, 4, 24, 'more', p, { color: p.secondary });
  return s;
}

function pageHeader(title, subtitle, rightText) {
  let s = rect(44, 32, W - 44, 32, p.surface) + line(0, 64, W, 64, p.line);
  s += text(60, 48, title, { size: 15, color: p.ink, weight: 600, maxWidth: 110, family: sans });
  s += text(120, 48, subtitle, { size: 11, color: p.secondary, maxWidth: 360, family: sans });
  if (rightText) s += text(1024, 48, rightText, { size: 11, color: p.secondary, maxWidth: 240, anchor: 'end', family: sans });
  return s;
}

function rail(active) {
  const items = [['changes', 'changes'], ['history', 'history'], ['box', 'box'], ['branch', 'branch'], ['remote', 'remote'], ['stash', 'stash'], ['conflict', 'conflict']];
  let s = rect(0, 32, 44, H - 32, p.shell) + line(43, 32, 43, H, p.line);
  items.forEach(([label, ico], i) => {
    const y = 44 + i * 38, on = label === active;
    if (on) s += rect(0, y - 6, 44, 32, p.selected, 'none', 0) + rect(0, y - 6, 2, 32, p.accent, 'none', 1);
    s += icon(14, y, ico, on ? p.accent : p.secondary, 16, 1.7);
  });
  s += icon(14, 646, 'settings', p.secondary, 16, 1.7);
  return s;
}

function statusBar() {
  let s = rect(0, 676, W, 24, p.shell) + line(0, 676, W, 676, p.line);
  s += text(12, 688, 'HEAD', { size: 11, color: p.muted, weight: 500, maxWidth: 34, family: sans });
  s += text(48, 688, 'b71f9c2', { size: 11, color: p.ink, weight: 500, maxWidth: 56, family: sans, mono: true });
  s += icon(112, 682, 'branch', p.secondary, 12, 1.6);
  s += text(128, 688, content.branch, { size: 11, color: p.secondary, maxWidth: 120, family: sans });
  s += text(256, 688, '↑2  ↓0', { size: 11, color: p.good, weight: 600, maxWidth: 60, family: sans });
  s += centerText(520, 688, 'Index 1 · Worktree 3 · 0 conflicts', 11, p.secondary, 400, 280, sans);
  s += circle(950, 688, 3, p.good);
  s += text(1024, 688, 'UTF-8 · LF', { size: 11, color: p.secondary, maxWidth: 70, anchor: 'end', family: sans });
  return s;
}

function chrome(active, title, subtitle, rightText) {
  let s = rect(0, 0, W, H, p.bg);
  s += topbar();
  s += pageHeader(title, subtitle, rightText);
  s += rail(active);
  s += statusBar();
  return s;
}

function sectionHeaderC(x, y, w, label, count, actions = []) {
  let s = text(x, y, label, { size: 11, color: p.secondary, weight: 600, maxWidth: 90, family: sans });
  s += text(x + estimate(label, 11, sans) + 8, y, String(count), { size: 11, color: p.accent, weight: 600, maxWidth: 22, family: sans });
  actions.forEach(([ico, color, ax]) => { s += icon(ax, y - 6, ico, color, 12, 1.7); });
  return s;
}

function fileRowC(x, y, w, file, { selected = false, action = '', staged = false } = {}) {
  const actionX = x + w - 18, statusX = actionX - 22, typeX = statusX - 28, textX = x + 28;
  const nameMax = Math.min(110, typeX - textX - 6);
  const shownName = fit(file.name, nameMax, 12, sans);
  const nameW = estimate(shownName, 12, sans);
  const pathMax = Math.max(0, typeX - (textX + nameW + 8) - 2);
  const shownPath = pathMax > 20 ? fitPath(file.path.endsWith('/') ? file.path : file.path + '/', pathMax, 11, sans) : '';
  const statusColor = file.status === 'M' ? p.warning : file.status === 'A' ? p.good : file.status === 'D' ? p.danger : p.accent;
  let s = `<g data-component="file-row" data-label="${file.name}">`;
  if (selected) s += rect(x, y, w, 24, p.selected, 'none', 2) + rect(x - 4, y, 2, 24, p.accent, 'none', 1);
  s += rect(x, y + 3, 2, 18, statusColor, 'none', 1);
  s += icon(x + 10, y + 5, file.type === 'sub' ? 'box' : 'file', p.secondary, 14, 1.6);
  s += text(textX, y + 12, shownName, { size: 12, color: p.ink, weight: selected ? 600 : 500, maxWidth: nameMax, family: sans });
  if (shownPath) s += text(textX + nameW + 8, y + 12, shownPath, { size: 11, color: p.muted, maxWidth: pathMax, family: sans });
  s += text(typeX, y + 12, file.type, { size: 11, color: p.muted, maxWidth: 22, family: sans });
  s += centerText(statusX + 7, y + 12, file.status, 11, statusColor, 600, 14, sans);
  if (action) s += icon(actionX, y + 4, action === 'stage' ? 'plus' : 'minus', action === 'stage' ? p.accent : p.secondary, 14, 1.8);
  return s + '</g>';
}

function fileListC() {
  let s = rect(44, 64, 250, H - 64, p.surface) + line(293, 64, 293, H, p.line);
  s += sectionHeaderC(56, 78, 238, '未暂存', 3, [['plus', p.accent, 242], ['trash', p.secondary, 264]]);
  content.unstaged.forEach((file, i) => { s += fileRowC(48, 88 + i * 24, 242, file, { selected: i === 0, action: i === 0 ? 'stage' : '' }); });
  s += sectionHeaderC(56, 168, 238, '已暂存', 1, [['minus', p.secondary, 264]]);
  s += fileRowC(48, 178, 242, content.staged[0], { action: 'unstage' });
  s += line(48, 210, 290, 210, p.line);
  s += text(56, 226, '工作区状态', { size: 11, color: p.secondary, weight: 600, maxWidth: 120, family: sans });
  const rows = [
    ['Index', '1 文件', p.ink],
    ['Worktree', '3 文件', p.warning],
    ['冲突', '0', p.good],
    ['HEAD', 'b71f9c2', p.secondary],
    ['上游', 'origin/feature/…', p.secondary],
  ];
  rows.forEach(([k, v, c], i) => {
    const yy = 248 + i * 22;
    s += text(56, yy, k, { size: 11, color: p.secondary, maxWidth: 60, family: sans });
    s += text(290, yy, v, { size: 11, color: c, maxWidth: 110, anchor: 'end', family: sans });
    s += line(48, yy + 11, 290, yy + 11, p.line);
  });
  s += text(56, 380, '最近操作', { size: 11, color: p.secondary, weight: 600, maxWidth: 120, family: sans });
  const ops = [
    ['10:42', '提交 b71f9c2', p.good],
    ['10:41', 'Fetch origin', p.secondary],
    ['10:38', '暂存 stage.go', p.secondary],
    ['10:36', '读取 core 失败', p.danger],
  ];
  ops.forEach(([t, v, c], i) => {
    const yy = 400 + i * 22;
    s += text(56, yy, t, { size: 11, color: p.muted, maxWidth: 40, family: sans });
    s += text(100, yy, v, { size: 11, color: c, maxWidth: 190, family: sans });
  });
  s += line(48, 604, 290, 604, p.line);
  s += text(56, 624, '丢弃全部改动…', { size: 11, color: p.danger, weight: 500, maxWidth: 180, family: sans });
  return s;
}

function diffC() {
  const x = 294, y = 64, w = 466;
  let s = rect(x, y, w, H - y, p.surface) + line(x + w, y, x + w, H, p.line);
  s += rect(x, y, w, 28, p.raised) + line(x, y + 28, x + w, y + 28, p.line);
  s += icon(x + 10, y + 7, 'file', p.secondary, 13, 1.6);
  s += text(x + 30, y + 14, content.diffPath, { size: 11, color: p.ink, weight: 500, maxWidth: 250, family: sans, mono: true });
  s += rect(x + w - 142, y + 4, 100, 20, p.raised, p.line, 3);
  s += rect(x + w - 140, y + 6, 48, 16, p.surface, p.line, 2);
  s += centerText(x + w - 116, y + 14, '统一', 11, p.ink, 600, 42, sans);
  s += centerText(x + w - 70, y + 14, '左右', 11, p.secondary, 400, 42, sans);
  s += iconButton(x + w - 32, y + 2, 24, 'more', p, { color: p.secondary });
  s += rect(x, y + 28, w, 24, p.raised);
  s += text(x + 12, y + 40, content.hunk, { size: 11, color: p.secondary, maxWidth: 280, family: sans, mono: true });
  s += iconButton(x + w - 68, y + 30, 20, 'plus', p, { color: p.accent });
  s += iconButton(x + w - 40, y + 30, 20, 'more', p, { color: p.secondary });
  s += line(x, y + 52, x + w, y + 52, p.line);
  const top = y + 52, lh = 19, gutter = 58;
  s += rect(x, top, gutter, diffRows.length * lh, p.raised);
  diffRows.forEach((r, i) => {
    const yy = top + i * lh;
    if (r.kind !== 'ctx') s += rect(x, yy, w, lh, r.kind === 'add' ? p.add : p.del);
    if (i >= 7 && i <= 10) s += rect(x, yy, 2, lh, p.accent, 'none', 0);
    s += text(x + 6, yy + 10, r.oldNo, { size: 11, color: p.muted, maxWidth: 20, family: sans, mono: true });
    s += text(x + 28, yy + 10, r.newNo, { size: 11, color: p.muted, maxWidth: 20, family: sans, mono: true });
    s += centerText(x + 50, yy + 10, r.kind === 'add' ? '+' : r.kind === 'del' ? '−' : '', 11, r.kind === 'add' ? p.good : r.kind === 'del' ? p.danger : p.muted, 500, 12, sans);
    s += text(x + gutter + 8, yy + 10, r.code, { size: 11, color: p.ink, maxWidth: w - gutter - 14, family: sans, mono: true });
  });
  s += line(x + gutter, top, x + gutter, top + diffRows.length * lh, p.line);
  const bottom = top + diffRows.length * lh;
  s += rect(x, bottom, w, 28, p.raised) + line(x, bottom + 28, x + w, bottom + 28, p.line);
  s += text(x + 12, bottom + 14, '已选 4 行', { size: 11, color: p.accent, weight: 600, maxWidth: 80, family: sans });
  s += text(x + 90, bottom + 14, '写入 Index，工作区不变', { size: 11, color: p.secondary, maxWidth: 220, family: sans });
  s += button(x + w - 82, bottom + 4, 70, 20, '暂存 4 行', p, { primary: true, size: 11, family: sans, radius: 3 });
  s += text(x + 12, bottom + 48, '文件历史', { size: 11, color: p.secondary, weight: 600, maxWidth: 90, family: sans });
  content.fileHistory.forEach((r, i) => {
    const yy = bottom + 68 + i * 22;
    s += text(x + 12, yy, r.sha, { size: 11, color: p.ink, weight: 500, maxWidth: 52, family: sans, mono: true });
    s += text(x + 70, yy, r.when, { size: 11, color: p.muted, maxWidth: 58, family: sans });
    s += text(x + 132, yy, r.subject, { size: 11, color: p.secondary, maxWidth: 300, family: sans });
  });
  s += line(x + 12, bottom + 116, x + w - 12, bottom + 116, p.line);
  s += text(x + 12, bottom + 134, '+4 −1 · 2 hunks · 48 行上下文', { size: 11, color: p.secondary, maxWidth: 300, family: sans });
  s += checkbox(x + 12, bottom + 154, '忽略空白', false, p, { size: 11 });
  s += text(x + 100, bottom + 160, '上下文 3 行', { size: 11, color: p.secondary, maxWidth: 100, family: sans });
  s += icon(x + w - 20, bottom + 153, 'down', p.secondary, 12, 1.6);
  s += line(x + 12, bottom + 180, x + w - 12, bottom + 180, p.line);
  s += text(x + 12, bottom + 198, '改动概览', { size: 11, color: p.secondary, weight: 600, maxWidth: 90, family: sans });
  s += rect(x + 12, bottom + 212, 380, 6, p.raised, 'none', 2);
  s += rect(x + 12, bottom + 212, 304, 6, p.good, 'none', 2);
  s += rect(x + 320, bottom + 212, 72, 6, p.danger, 'none', 2);
  s += text(x + 12, bottom + 232, '4 行新增 · 1 行删除 · 2 个 hunk', { size: 11, color: p.secondary, maxWidth: 300, family: sans });
  s += text(x + 12, bottom + 256, '下一个 hunk', { size: 11, color: p.secondary, weight: 600, maxWidth: 90, family: sans });
  s += text(x + 110, bottom + 256, '@@ −96,6 +99,7 @@', { size: 11, color: p.ink, maxWidth: 190, family: sans, mono: true });
  s += text(x + w - 12, bottom + 256, '跳转 →', { size: 11, color: p.accent, weight: 500, maxWidth: 60, anchor: 'end', family: sans });
  s += text(x + 12, bottom + 280, '提交前检查：hooks 将运行，未暂存改动保留。', { size: 11, color: p.muted, maxWidth: 380, family: sans });
  const mapTop = top + 2, mapH = diffRows.length * lh - 4;
  s += rect(x + w - 8, mapTop, 3, mapH, p.raised, 'none', 1);
  diffRows.forEach((r, i) => {
    if (r.kind === 'ctx') return;
    s += rect(x + w - 8, mapTop + i * lh, 3, lh - 1, r.kind === 'add' ? p.good : p.danger, 'none', 1, 'opacity=".9"');
  });
  return s;
}

function inspectorC() {
  const x = 760, w = 280;
  let s = rect(x, 64, w, H - 64, p.surface) + line(x, 64, x, H, p.line);
  s += text(776, 82, '提交', { size: 12, color: p.ink, weight: 600, maxWidth: 60, family: sans });
  s += pill(1002, 72, '1', p, { fill: p.selected, stroke: 'none', color: p.accent, height: 18, size: 11, radius: 9, maxWidth: 24 });
  s += text(776, 102, '提交说明', { size: 11, color: p.secondary, maxWidth: 90, family: sans });
  s += input(772, 110, 256, 24, content.subject, p, { size: 11, family: sans, radius: 3 });
  s += text(776, 146, '补充说明', { size: 11, color: p.secondary, maxWidth: 90, family: sans });
  s += input(772, 154, 256, 40, '可选，提交正文', p, { placeholder: true, size: 11, family: sans, radius: 3 });
  s += button(772, 204, 256, 26, '提交 1 个文件', p, { primary: true, size: 12, family: sans, radius: 3 });
  s += line(772, 248, 1028, 248, p.line);
  s += text(776, 264, 'Git 状态', { size: 11, color: p.secondary, weight: 600, maxWidth: 90, family: sans });
  const rows = [
    ['Index', '1 文件', p.ink],
    ['Worktree', '3 文件', p.warning],
    ['冲突', '0', p.good],
    ['领先 / 落后', '↑2  ↓0', p.good],
    ['上游', 'origin/feature/…', p.secondary],
  ];
  rows.forEach(([k, v, c], i) => {
    const yy = 286 + i * 22;
    s += text(776, yy, k, { size: 11, color: p.secondary, maxWidth: 80, family: sans });
    s += text(1028, yy, v, { size: 11, color: c, maxWidth: 130, anchor: 'end', family: sans });
  });
  s += line(772, 392, 1028, 392, p.line);
  s += text(776, 408, '同步', { size: 11, color: p.secondary, weight: 600, maxWidth: 60, family: sans });
  s += button(772, 418, 80, 24, 'Fetch', p, { size: 11, family: sans, radius: 3, iconName: 'remote' });
  s += button(860, 418, 80, 24, 'Pull', p, { size: 11, family: sans, radius: 3, iconName: 'pull' });
  s += button(948, 418, 80, 24, 'Push', p, { size: 11, family: sans, radius: 3, iconName: 'push' });
  s += text(776, 462, '提交选项', { size: 11, color: p.secondary, weight: 600, maxWidth: 90, family: sans });
  s += checkbox(772, 476, '运行 hooks', true, p, { size: 11 });
  s += checkbox(772, 498, '签名提交', false, p, { size: 11 });
  s += line(772, 520, 1028, 520, p.line);
  s += text(776, 536, '危险操作', { size: 11, color: p.danger, weight: 600, maxWidth: 90, family: sans });
  s += text(776, 556, '丢弃全部改动…', { size: 11, color: p.danger, weight: 500, maxWidth: 180, family: sans });
  s += text(776, 578, '重置到 HEAD…', { size: 11, color: p.danger, maxWidth: 180, family: sans });
  s += text(776, 600, '清理未跟踪文件…', { size: 11, color: p.secondary, maxWidth: 180, family: sans });
  s += line(772, 622, 1028, 622, p.line);
  s += text(776, 638, '最近提交', { size: 11, color: p.secondary, weight: 600, maxWidth: 90, family: sans });
  s += text(776, 658, 'b71f9c2 · feat: preserve…', { size: 11, color: p.muted, maxWidth: 250, family: sans });
  return s;
}

export function workspaceC() {
  let s = chrome('changes', '工作区', 'atlas / 主工作区', '3 未暂存 · 1 已暂存');
  s += fileListC();
  s += diffC();
  s += inspectorC();
  return s;
}

function historyTableC() {
  const x = 44, w = 696, top = 110;
  let s = rect(x, 64, w, 24, p.surface);
  const chips = ['全部', '当前分支', '作者 dev', '近 7 天'];
  let cx = 52;
  chips.forEach((label, i) => {
    const cw = estimate(label, 11, sans) + 16;
    s += rect(cx, 68, cw, 16, i === 0 ? p.selected : 'none', i === 0 ? p.accent : 'none', 2);
    s += text(cx + 8, 76, label, { size: 11, color: i === 0 ? p.accent : p.secondary, weight: i === 0 ? 600 : 400, maxWidth: cw - 12, family: sans });
    cx += cw + 6;
  });
  s += text(732, 76, '12 / 128', { size: 11, color: p.muted, maxWidth: 70, anchor: 'end', family: sans });
  s += line(44, 88, 740, 88, p.line);
  s += rect(44, 88, w, 22, p.raised);
  const cols = [['线图', 52], ['提交说明与引用', 172], ['作者', 472], ['时间', 562], ['SHA', 622]];
  cols.forEach(([label, colX]) => { s += text(colX, 99, label, { size: 11, color: p.secondary, weight: 600, maxWidth: 100, family: sans }); });
  s += icon(714, 93, 'more', p.secondary, 12, 1.6);
  s += line(44, 110, 740, 110, p.line);
  s += graph(x, top, { row: 24, lane: 14, style: 'diagonal', p, selected: 0, colors: [p.main, p.fix, p.topic, p.topic2], width: 120, edge: 1.2, node: 3, mergeNode: 4 });
  commits.forEach((c, i) => {
    const y = top + i * 24 + 12;
    if (i === 0) s += rect(x, y - 12, w, 24, p.selected, 'none', 0) + rect(x, y - 10, 2, 20, p.accent, 'none', 1);
    if (i === 0) s += text(150, y, 'HEAD', { size: 11, color: p.accent, weight: 600, maxWidth: 40, anchor: 'end', family: sans });
    const refs = c.refs.filter((r) => r !== 'HEAD' && r !== content.branch);
    const refW = refs.reduce((n, r) => n + Math.min(112, estimate(r, 11, sans) + 14) + 4, 0);
    const subjectMax = Math.max(90, 288 - refW - 8);
    s += text(172, y, c.subject, { size: 12, color: p.ink, weight: i === 0 ? 600 : 400, maxWidth: subjectMax, family: sans });
    let rx = 172 + estimate(fit(c.subject, subjectMax, 12, sans), 12, sans) + 6;
    refs.forEach((ref) => {
      const rw = Math.min(112, estimate(ref, 11, sans) + 14);
      s += pill(rx, y - 8, ref, p, { fill: p.raised, stroke: p.line, color: p.secondary, height: 16, size: 11, radius: 3, maxWidth: rw });
      rx += rw + 4;
    });
    s += text(472, y, 'dev', { size: 11, color: p.secondary, maxWidth: 50, family: sans });
    s += text(562, y, i < 2 ? '10:42' : '昨天', { size: 11, color: p.muted, maxWidth: 50, family: sans });
    s += text(622, y, c.sha, { size: 11, color: p.secondary, maxWidth: 62, family: sans, mono: true });
    if (i === 0) s += icon(716, y - 6, 'more', p.secondary, 12, 1.6);
    s += line(x, y + 12, x + w, y + 12, i === 0 ? p.strong : p.line);
  });
  const legendY = 420;
  s += text(52, legendY, '轨道', { size: 11, color: p.secondary, weight: 600, maxWidth: 40, family: sans });
  [['main', p.main], ['fix', p.fix], ['feature', p.topic], ['topic', p.topic2]].forEach(([label, color], i) => {
    const lx = 100 + i * 86;
    s += circle(lx, legendY, 3, color);
    s += text(lx + 8, legendY, label, { size: 11, color: p.secondary, maxWidth: 70, family: sans });
  });
  s += line(44, 440, 740, 440, p.line, 1, 'stroke-dasharray="2 3"');
  s += text(392, 460, '加载更早的提交', { size: 11, color: p.accent, weight: 500, maxWidth: 120, anchor: 'middle', family: sans });
  return s;
}

function inspectorHistoryC() {
  const x = 740, w = 300;
  let s = rect(x, 64, w, H - 64, p.surface) + line(x, 64, x, H, p.line);
  s += text(756, 82, '提交详情', { size: 12, color: p.ink, weight: 600, maxWidth: 90, family: sans });
  s += iconButton(1008, 68, 24, 'close', p, { color: p.secondary });
  s += paragraph(756, 108, 'feat: preserve partial staging', 268, { size: 15, color: p.ink, lineHeight: 20, maxLines: 2, family: sans });
  s += text(756, 146, 'b71f9c2 · dev · 今天 10:42', { size: 11, color: p.secondary, maxWidth: 250, family: sans, mono: true });
  s += pill(756, 162, 'HEAD', p, { fill: p.accent, stroke: p.accent, color: p.onAccent, height: 18, size: 11, radius: 9, maxWidth: 42 });
  s += pill(804, 162, content.branch, p, { fill: p.raised, stroke: p.line, color: p.secondary, height: 18, size: 11, radius: 9, maxWidth: 180 });
  s += line(756, 192, 1020, 192, p.line);
  s += text(756, 208, '变更 2 文件 · +18 −4', { size: 11, color: p.secondary, maxWidth: 230, family: sans });
  s += icon(756, 220, 'file', p.secondary, 12, 1.6) + text(774, 226, 'internal/gitrepo/stage.go', { size: 11, color: p.ink, maxWidth: 190, family: sans });
  s += text(1020, 226, 'M', { size: 11, color: p.warning, weight: 600, maxWidth: 14, anchor: 'end', family: sans });
  s += icon(756, 242, 'file', p.secondary, 12, 1.6) + text(774, 248, 'internal/desktop/workspace.go', { size: 11, color: p.ink, maxWidth: 190, family: sans });
  s += text(1020, 248, 'M', { size: 11, color: p.warning, weight: 600, maxWidth: 14, anchor: 'end', family: sans });
  s += text(756, 282, '父提交', { size: 11, color: p.secondary, weight: 600, maxWidth: 60, family: sans });
  s += text(756, 302, 'a43bc18 · Preserve commit message draft', { size: 11, color: p.ink, maxWidth: 260, family: sans });
  s += line(756, 324, 1020, 324, p.line);
  s += text(756, 340, '操作', { size: 11, color: p.secondary, weight: 600, maxWidth: 50, family: sans });
  s += button(756, 352, 124, 24, '新建分支', p, { size: 11, family: sans, radius: 3 });
  s += button(890, 352, 124, 24, '打标签', p, { size: 11, family: sans, radius: 3 });
  s += button(756, 384, 124, 24, 'Cherry-pick', p, { size: 11, family: sans, radius: 3 });
  s += button(890, 384, 124, 24, 'Revert', p, { size: 11, family: sans, radius: 3, color: p.danger });
  s += text(756, 428, '复制 SHA', { size: 11, color: p.accent, weight: 500, maxWidth: 70, family: sans });
  s += text(880, 428, '在终端中打开', { size: 11, color: p.accent, weight: 500, maxWidth: 110, family: sans });
  s += line(756, 452, 1020, 452, p.line);
  s += text(756, 468, '提交信息', { size: 11, color: p.secondary, weight: 600, maxWidth: 90, family: sans });
  s += paragraph(756, 488, content.description, 264, { size: 11, color: p.secondary, lineHeight: 16, maxLines: 2, family: sans });
  s += text(756, 540, '签名', { size: 11, color: p.secondary, weight: 600, maxWidth: 40, family: sans });
  s += text(1020, 540, '未签名', { size: 11, color: p.muted, maxWidth: 70, anchor: 'end', family: sans });
  s += line(756, 564, 1020, 564, p.line);
  s += text(756, 584, '作者', { size: 11, color: p.secondary, weight: 600, maxWidth: 40, family: sans });
  s += text(1020, 584, 'dev · 今天 10:42', { size: 11, color: p.secondary, maxWidth: 150, anchor: 'end', family: sans });
  s += text(756, 608, '加载父提交', { size: 11, color: p.accent, weight: 500, maxWidth: 100, family: sans });
  s += text(756, 646, '此提交已在当前分支。', { size: 11, color: p.muted, maxWidth: 240, family: sans });
  return s;
}

export function historyC() {
  let s = chrome('history', '历史', '12 个提交 · HEAD 领先 2', '');
  s += historyTableC();
  s += inspectorHistoryC();
  return s;
}

export const meta = {
  id: 'C',
  name: '密集仪表',
  line: '24px 行、深色、三栏、45° 线图、状态条与操作面板',
  notes: [
    '文件、Diff、提交三栏并置，提交与同步都在右侧完成。',
    '24px 行；名称优先，状态色脊 + 字母，不只靠颜色。',
    '45° 线图；底部状态条常驻 HEAD、ahead/behind 与 Index 计数。',
  ],
};
