// Direction A: native macOS craft.
import {
  A as p, W, H, text, centerText, paragraph, rect, line, path, circle, icon,
  pill, button, iconButton, input, checkbox, graph, commits, diffRows, content, estimate, fit, fitPath,
} from './primitives.mjs';

const sans = 'a';

function trafficLights() {
  return circle(20, 19, 6, '#ff5f57', '#e0443e', 0.5)
    + circle(40, 19, 6, '#febc2e', '#d89e20', 0.5)
    + circle(60, 19, 6, '#28c840', '#1f9e34', 0.5);
}

function toolbarButton(x, y, label, iconName) {
  return `<g data-component="button" data-label="${label}">`
    + icon(x + 6, y + 6, iconName, p.secondary, 14, 1.6)
    + text(x + 24, y + 13, label, { size: 11, color: p.ink, weight: 500, maxWidth: 40, family: sans })
    + '</g>';
}

function sidebar(active) {
  const items = [
    ['工作区', 'changes'], ['历史', 'history'], ['子模块', 'box'], ['分支与标签', 'branch'],
    ['远端与同步', 'remote'], ['Stash', 'stash'], ['冲突处理', 'conflict'], ['Worktrees', 'worktree'],
  ];
  let s = rect(0, 82, 196, H - 82, p.shell) + line(195, 82, 195, H, p.line);
  s += text(16, 96, '仓库', { size: 11, color: p.secondary, weight: 600, maxWidth: 160, family: sans });
  for (let i = 0; i < 4; i++) {
    const [label, ico] = items[i], y = 116 + i * 28;
    if (ico === active) s += rect(8, y - 13, 180, 26, p.selected, 'none', 5) + rect(8, y - 8, 2, 16, p.accent, 'none', 1);
    s += icon(18, y - 7, ico, ico === active ? p.accent : p.secondary, 14, 1.7);
    s += text(40, y, label, { size: 13, color: ico === active ? p.ink : p.ink, weight: ico === active ? 600 : 400, maxWidth: 140, family: sans });
  }
  s += line(16, 224, 180, 224, p.line);
  s += text(16, 240, '同步与工具', { size: 11, color: p.secondary, weight: 600, maxWidth: 160, family: sans });
  for (let i = 0; i < 4; i++) {
    const [label, ico] = items[i + 4], y = 256 + i * 28;
    if (ico === active) s += rect(8, y - 13, 180, 26, p.selected, 'none', 5) + rect(8, y - 8, 2, 16, p.accent, 'none', 1);
    s += icon(18, y - 7, ico, ico === active ? p.accent : p.secondary, 14, 1.7);
    s += text(40, y, label, { size: 13, color: p.ink, weight: ico === active ? 600 : 400, maxWidth: 140, family: sans });
  }
  s += text(16, 388, '本地分支', { size: 11, color: p.secondary, weight: 600, maxWidth: 160, family: sans });
  ['feature/partial-stage', 'main', 'fix/refresh'].forEach((branch, i) => {
    const y = 406 + i * 24;
    s += icon(18, y - 6, 'branch', i === 0 ? p.accent : p.secondary, 12, 1.6);
    s += text(36, y, branch, { size: 11, color: i === 0 ? p.ink : p.secondary, weight: i === 0 ? 500 : 400, maxWidth: 144, family: sans });
  });
  s += line(16, 496, 180, 496, p.line);
  s += text(16, 514, '工作区', { size: 11, color: p.secondary, weight: 600, maxWidth: 160, family: sans });
  s += icon(18, 536, 'worktree', p.accent, 14, 1.7) + text(40, 542, '主工作区', { size: 13, color: p.ink, weight: 500, maxWidth: 130, family: sans });
  s += icon(18, 566, 'worktree', p.secondary, 14, 1.7) + text(40, 572, 'core', { size: 13, color: p.secondary, maxWidth: 130, family: sans });
  s += line(16, 620, 180, 620, p.line);
  s += icon(18, 640, 'settings', p.secondary, 14, 1.7) + text(40, 647, '设置', { size: 13, color: p.ink, maxWidth: 120, family: sans });
  return s;
}

function chrome(active) {
  let s = rect(0, 0, W, H, p.bg);
  s += rect(0, 0, W, 38, p.shell) + line(0, 38, W, 38, p.line);
  s += trafficLights();
  s += centerText(W / 2, 19, 'atlas', 13, p.ink, 600, 280, sans);
  s += icon(W / 2 + 30, 13, 'down', p.muted, 12, 1.6);
  s += iconButton(1000, 10, 18, 'more', p, { color: p.secondary });
  s += rect(0, 38, W, 44, p.surface) + line(0, 81, W, 81, p.line);
  s += iconButton(10, 48, 24, 'panel', p, { color: p.secondary });
  s += iconButton(36, 48, 24, 'chevronLeft', p, { color: p.secondary });
  s += iconButton(58, 48, 24, 'chevronRight', p, { color: p.muted });
  s += line(84, 48, 84, 72, p.line);
  s += rect(348, 46, 220, 28, p.raised, p.line, 6);
  s += icon(358, 53, 'branch', p.accent, 14, 1.7);
  s += text(378, 60, content.branch, { size: 13, color: p.ink, weight: 600, maxWidth: 156, family: sans });
  s += icon(548, 54, 'down', p.secondary, 12, 1.6);
  s += text(580, 60, '↑2', { size: 11, color: p.good, weight: 600, maxWidth: 24, family: sans });
  s += text(604, 60, '↓0', { size: 11, color: p.muted, weight: 500, maxWidth: 24, family: sans });
  s += toolbarButton(800, 47, 'Fetch', 'remote');
  s += toolbarButton(866, 47, 'Pull', 'pull');
  s += toolbarButton(926, 47, 'Push', 'push');
  s += iconButton(1004, 48, 24, 'more', p, { color: p.secondary });
  s += sidebar(active);
  s += statusBar();
  return s;
}

function statusBar() {
  let s = rect(0, 672, W, 28, p.shell) + line(0, 672, W, 672, p.line);
  s += icon(12, 679, 'branch', p.secondary, 13, 1.6);
  s += text(30, 686, content.repo, { size: 11, color: p.ink, weight: 600, maxWidth: 40, family: sans });
  s += text(64, 686, content.branch, { size: 11, color: p.secondary, maxWidth: 140, family: sans });
  s += text(212, 686, '↑2  ↓0', { size: 11, color: p.good, weight: 600, maxWidth: 64, family: sans });
  s += centerText(520, 686, content.upstream, 11, p.muted, 400, 260, sans);
  s += text(1024, 686, 'Index 1 · Worktree 3', { size: 11, color: p.secondary, maxWidth: 150, anchor: 'end', family: sans });
  return s;
}

function sectionHeader(x, y, w, label, count, action) {
  return text(x, y, label, { size: 11, color: p.secondary, weight: 600, maxWidth: 120, family: sans })
    + pill(x + 54, y - 9, String(count), p, { fill: p.raised, stroke: 'none', color: p.secondary, height: 18, size: 11, radius: 9, maxWidth: 24 })
    + text(x + w, y, action, { size: 11, color: p.accent, weight: 500, maxWidth: 90, anchor: 'end', family: sans });
}

function fileRowA(x, y, w, file, { selected = false, action = '', staged = false } = {}) {
  const actionX = x + w - 22, statusX = actionX - 20, typeX = statusX - 32, textX = x + 46;
  const nameMax = Math.min(150, typeX - textX - 8);
  const shownName = fit(file.name, nameMax, 13, sans);
  const nameW = estimate(shownName, 13, sans);
  const pathMax = Math.max(0, typeX - (textX + nameW + 8) - 4);
  const shownPath = pathMax > 24 ? fitPath(file.path.endsWith('/') ? file.path : file.path + '/', pathMax, 11, sans) : '';
  const statusColor = file.status === 'M' ? p.warning : file.status === 'A' ? p.good : file.status === 'D' ? p.danger : p.accent;
  let s = `<g data-component="file-row" data-label="${file.name}">`;
  if (selected) s += rect(x, y, w, 28, p.selected, 'none', 4) + rect(x, y + 4, 2, 20, p.accent, 'none', 1);
  s += circle(x + 12, y + 14, 3, statusColor);
  s += icon(x + 26, y + 7, file.type === 'sub' ? 'box' : 'file', p.secondary, 14, 1.6);
  s += text(textX, y + 14, shownName, { size: 13, color: p.ink, weight: selected ? 600 : 500, maxWidth: nameMax, family: sans });
  if (shownPath) s += text(textX + nameW + 8, y + 14, shownPath, { size: 11, color: p.muted, maxWidth: pathMax, family: sans });
  s += text(typeX, y + 14, file.type, { size: 11, color: p.muted, maxWidth: 26, family: sans });
  s += centerText(statusX + 9, y + 14, file.status, 11, statusColor, 600, 16, sans);
  if (action) s += rect(actionX, y + 5, 18, 18, p.surface, p.strong, 4) + icon(actionX + 2, y + 7, action === 'stage' ? 'plus' : 'minus', action === 'stage' ? p.accent : p.secondary, 14, 1.8);
  return s + '</g>';
}

function commitPanel() {
  const x = 208, w = 280;
  let s = line(196, 320, 496, 320, p.line);
  s += text(x, 336, `提交到 ${content.branch}`, { size: 11, color: p.secondary, weight: 500, maxWidth: w, family: sans });
  s += input(x, 344, w, 28, content.subject, p, { size: 12, family: sans });
  s += input(x, 384, w, 48, '补充说明（可选）', p, { placeholder: true, size: 12, family: sans });
  s += text(x, 448, '1 个已暂存文件', { size: 11, color: p.secondary, maxWidth: 140, family: sans });
  s += button(x, 456, w, 28, '提交暂存内容', p, { primary: true, size: 12, family: sans, radius: 5 });
  s += text(x, 504, '提交选项', { size: 11, color: p.secondary, weight: 600, maxWidth: 120, family: sans });
  s += checkbox(x, 518, '运行 Git hooks', true, p, { size: 11 });
  s += checkbox(x, 540, '签名提交', false, p, { size: 11 });
  s += text(x, 568, '未暂存的 3 个文件会保留在工作区。', { size: 11, color: p.muted, maxWidth: w, family: sans });
  s += line(196, 592, 496, 592, p.line);
  s += text(x, 608, '上次提交', { size: 11, color: p.secondary, weight: 600, maxWidth: 80, family: sans });
  s += text(x, 626, 'b71f9c2', { size: 11, color: p.ink, weight: 500, maxWidth: 60, family: sans, mono: true });
  s += text(x + 66, 626, 'feat: preserve partial staging', { size: 11, color: p.secondary, maxWidth: 210, family: sans });
  return s;
}

function diffA() {
  const x = 496, y = 128, w = 544;
  let s = rect(x, y, w, 36, p.raised) + line(x, y + 36, x + w, y + 36, p.line);
  s += icon(x + 12, y + 10, 'file', p.secondary, 14, 1.6);
  s += text(x + 34, y + 18, content.diffPath, { size: 12, color: p.ink, weight: 500, maxWidth: 270, family: sans, mono: true });
  s += pill(x + w - 246, y + 8, 'Index → 工作区', p, { fill: p.raised, stroke: p.line, color: p.secondary, height: 20, size: 11, radius: 10, maxWidth: 96 });
  s += rect(x + w - 140, y + 6, 124, 24, p.raised, p.line, 5);
  s += centerText(x + w - 108, y + 18, '统一', 11, p.ink, 600, 50, sans);
  s += centerText(x + w - 60, y + 18, '左右', 11, p.secondary, 500, 50, sans);
  s += line(x, y + 36, x + w, y + 36, p.line);
  s += rect(x, y + 36, w, 28, p.raised);
  s += text(x + 12, y + 50, content.hunk, { size: 11, color: p.secondary, maxWidth: 360, family: sans, mono: true });
  s += button(x + w - 120, y + 38, 108, 24, '暂存片段', p, { size: 11, family: sans, radius: 4 });
  s += line(x, y + 64, x + w, y + 64, p.line);
  const top = y + 64, lh = 21, gutter = 76;
  s += rect(x, top, gutter, diffRows.length * lh, p.raised);
  diffRows.forEach((r, i) => {
    const yy = top + i * lh;
    if (r.kind !== 'ctx') s += rect(x, yy, w, lh, r.kind === 'add' ? p.add : p.del);
    s += text(x + 10, yy + 11, r.oldNo, { size: 11, color: p.muted, maxWidth: 24, family: sans, mono: true });
    s += text(x + 36, yy + 11, r.newNo, { size: 11, color: p.muted, maxWidth: 24, family: sans, mono: true });
    s += centerText(x + 64, yy + 11, r.kind === 'add' ? '+' : r.kind === 'del' ? '−' : '', 12, r.kind === 'add' ? p.good : r.kind === 'del' ? p.danger : p.muted, 500, 14, sans);
    s += text(x + gutter + 12, yy + 11, r.code, { size: 11, color: p.ink, maxWidth: w - gutter - 20, family: sans, mono: true });
  });
  s += line(x + gutter, top, x + gutter, top + diffRows.length * lh, p.line);
  const bottom = top + diffRows.length * lh;
  s += text(x + 12, bottom + 12, '8 行 · +4 −1', { size: 11, color: p.secondary, maxWidth: 200, family: sans });
  s += text(x + w - 12, bottom + 12, '下一个改动', { size: 11, color: p.accent, weight: 500, maxWidth: 90, anchor: 'end', family: sans });
  s += line(x + 12, bottom + 28, x + w - 12, bottom + 28, p.line);
  s += text(x + 12, bottom + 46, '此文件的最近提交', { size: 11, color: p.secondary, weight: 600, maxWidth: 200, family: sans });
  content.fileHistory.forEach((r, i) => {
    const yy = bottom + 68 + i * 22;
    s += text(x + 12, yy, r.sha, { size: 11, color: p.ink, weight: 500, maxWidth: 56, family: sans, mono: true });
    s += text(x + 74, yy, r.when, { size: 11, color: p.muted, maxWidth: 60, family: sans });
    s += text(x + 140, yy, r.subject, { size: 11, color: p.secondary, maxWidth: 360, family: sans });
  });
  s += line(x + 12, bottom + 116, x + w - 12, bottom + 116, p.line);
  s += checkbox(x + 12, bottom + 132, '忽略空白', false, p, { size: 11 });
  s += text(x + 120, bottom + 138, '上下文 3 行', { size: 11, color: p.secondary, maxWidth: 90, family: sans });
  s += icon(x + w - 22, bottom + 131, 'down', p.secondary, 12, 1.6);
  return s;
}

export function workspaceA() {
  let s = chrome('changes');
  s += rect(196, 82, 844, 40, p.surface);
  s += text(220, 106, '工作区', { size: 20, color: p.ink, weight: 600, maxWidth: 200, family: sans });
  s += text(1024, 106, '3 个未暂存 · 1 个已暂存', { size: 11, color: p.secondary, maxWidth: 200, anchor: 'end', family: sans });
  s += line(196, 128, 1040, 128, p.line);
  s += rect(196, 128, 844, 544, p.surface);
  s += line(496, 128, 496, 672, p.line);
  s += sectionHeader(212, 156, 272, '未暂存', 3, '全部暂存');
  content.unstaged.forEach((file, i) => { s += fileRowA(208, 168 + i * 28, 280, file, { selected: i === 0, action: i === 0 ? 'stage' : '' }); });
  s += sectionHeader(212, 264, 272, '已暂存', 1, '全部取消');
  s += fileRowA(208, 276, 280, content.staged[0], { action: 'unstage' });
  s += commitPanel();
  s += diffA();
  return s;
}

function historyTableA() {
  let s = rect(196, 128, 844, 26, p.raised);
  const cols = [['线图', 204], ['提交说明', 354], ['引用', 578], ['作者', 776], ['时间', 866], ['SHA', 944]];
  cols.forEach(([label, x]) => { s += text(x, 141, label, { size: 11, color: p.secondary, weight: 600, maxWidth: 80, family: sans }); });
  s += line(196, 154, 1040, 154, p.line);
  const top = 154;
  s += graph(196, top, { row: 28, lane: 18, style: 'rounded', p, selected: 0, colors: [p.main, p.fix, p.topic, p.topic2], width: 150, edge: 1.5, node: 3.5, mergeNode: 4.6 });
  commits.forEach((c, i) => {
    const y = top + i * 28 + 14;
    if (i === 0) s += rect(196, y - 14, 844, 28, p.selected, 'none', 0);
    s += text(354, y, c.subject, { size: 13, color: p.ink, weight: i === 0 ? 600 : 400, maxWidth: 212, family: sans });
    c.refs.forEach((ref, j) => {
      const refX = 570 + (j === 0 ? 0 : 46);
      if (ref === 'HEAD') s += pill(refX, y - 10, ref, p, { fill: p.accent, stroke: p.accent, color: p.onAccent, height: 20, size: 11, radius: 10, maxWidth: 40 });
      else s += pill(refX, y - 10, ref, p, { fill: p.surface, stroke: p.strong, color: p.secondary, height: 20, size: 11, radius: 10, maxWidth: 150 });
    });
    s += text(776, y, 'dev', { size: 11, color: p.secondary, maxWidth: 60, family: sans });
    s += text(866, y, i < 2 ? '今天 10:42' : '昨天', { size: 11, color: p.muted, maxWidth: 68, family: sans });
    s += text(944, y, c.sha, { size: 11, color: p.secondary, maxWidth: 80, family: sans, mono: true });
    if (i === 0) s += icon(1016, y - 7, 'more', p.secondary, 14, 1.6);
    s += line(196, y + 14, 1040, y + 14, i === 0 ? p.strong : p.line);
  });
  return s;
}

function detailA() {
  let s = rect(196, 500, 844, 172, p.surface) + line(196, 500, 1040, 500, p.line);
  s += text(220, 524, 'feat: preserve partial staging', { size: 15, color: p.ink, weight: 600, maxWidth: 620, family: sans });
  s += text(220, 546, 'b71f9c2 · dev · 今天 10:42', { size: 11, color: p.secondary, maxWidth: 420, family: sans, mono: true });
  s += paragraph(220, 568, '保留部分暂存后的工作区版本。', 620, { size: 11, color: p.secondary, lineHeight: 16, maxLines: 1, family: sans });
  s += text(220, 598, '2 个变更文件', { size: 11, color: p.secondary, weight: 600, maxWidth: 120, family: sans });
  s += icon(220, 612, 'file', p.secondary, 12, 1.6) + text(238, 618, 'internal/gitrepo/stage.go', { size: 11, color: p.ink, maxWidth: 280, family: sans });
  s += text(520, 618, 'M', { size: 11, color: p.warning, weight: 600, maxWidth: 14, family: sans });
  s += icon(220, 634, 'file', p.secondary, 12, 1.6) + text(238, 640, 'internal/desktop/workspace.go', { size: 11, color: p.ink, maxWidth: 280, family: sans });
  s += text(520, 640, 'M', { size: 11, color: p.warning, weight: 600, maxWidth: 14, family: sans });
  s += text(1024, 598, '+18  −4', { size: 11, color: p.secondary, maxWidth: 100, anchor: 'end', family: sans, mono: true });
  s += button(860, 510, 140, 26, '完整提交详情', p, { size: 11, family: sans, radius: 5 });
  s += iconButton(1016, 511, 24, 'more', p, { color: p.secondary });
  const actions = ['复制 SHA', '新建分支…', '打标签…', 'Cherry-pick…', 'Revert…', '在终端中打开'];
  let ax = 220;
  actions.forEach((label) => {
    s += text(ax, 662, label, { size: 11, color: label === 'Revert…' ? p.danger : p.accent, weight: 500, maxWidth: 100, family: sans });
    ax += estimate(label, 11, sans) + 18;
  });
  return s;
}

export function historyA() {
  let s = chrome('history');
  s += rect(196, 82, 844, 46, p.surface);
  s += text(220, 106, '历史', { size: 20, color: p.ink, weight: 600, maxWidth: 160, family: sans });
  s += rect(780, 92, 220, 28, p.surface, p.strong, 6);
  s += icon(790, 99, 'search', p.muted, 14, 1.6);
  s += text(810, 106, '搜索提交、SHA 或引用', { size: 11, color: p.muted, maxWidth: 180, family: sans });
  s += iconButton(1008, 94, 24, 'filter', p, { color: p.secondary });
  s += line(196, 128, 1040, 128, p.line);
  s += historyTableA();
  s += detailA();
  return s;
}

export const meta = {
  id: 'A',
  name: '原生 macOS 工艺',
  line: '真实窗口层级、SF 字体、名称优先文件行、圆角线图',
  notes: [
    '28px 行；名称固定列，路径在后吸收截断。',
    '顶栏常驻 ahead / behind 与分支切换。',
    '文件行和 Diff 都提供全部暂存 / 取消 / 丢弃入口。',
  ],
};
