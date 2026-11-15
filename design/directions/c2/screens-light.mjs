// C2 light theme, localized. Same three-column frame as the dark C2 probe.
import {
  W, H, text, centerText, paragraph, rect, line, circle, icon,
  pill, button, iconButton, input, checkbox, graph, commits, content,
  estimate, fit, fitPath, svgDocument,
} from '../primitives.mjs';
import { LIGHT, STRINGS } from './light.mjs';
import { c2DiffRows } from './data.mjs';

const sans = 'c';

export function createScreens(locale) {
  const t = STRINGS[locale];
  if (!t) throw new Error(`Unknown locale: ${locale}`);
  const p = LIGHT;
  const lang = locale;

  function keycap(x, y, label, w = 30) {
    return rect(x, y, w, 14, p.raised, p.line, 2)
      + centerText(x + w / 2, y + 7, label, 9, p.secondary, 500, w - 4, sans);
  }

  /* -------------------------------------------------------------- chrome */

  function topbar() {
    let s = rect(0, 0, W, 32, p.shell) + line(0, 32, W, 32, p.line);
    s += icon(12, 9, 'box', p.accent, 14, 1.7);
    s += text(32, 16, 'atlas', { size: 12, color: p.ink, weight: 600, maxWidth: 46, family: sans });
    s += line(82, 8, 82, 24, p.line);
    s += rect(92, 6, 172, 20, p.surface, p.line, 3);
    s += icon(100, 10, 'branch', p.accent, 12, 1.7);
    s += text(116, 16, content.branch, { size: 11, color: p.ink, weight: 600, maxWidth: 126, family: sans });
    s += icon(250, 11, 'down', p.muted, 10, 1.6);
    s += text(274, 16, '↑2', { size: 11, color: p.good, weight: 600, maxWidth: 18, family: sans });
    s += text(294, 16, '↓0', { size: 11, color: p.muted, maxWidth: 18, family: sans });
    s += rect(600, 6, 332, 20, p.surface, p.line, 3);
    s += icon(608, 10, 'search', p.muted, 12, 1.6);
    s += text(626, 16, t.searchCommand, { size: 11, color: p.muted, maxWidth: 230, family: sans });
    s += keycap(894, 9, '⌘K', 30);
    s += rect(940, 6, 68, 20, p.raised, p.line, 3);
    s += icon(946, 10, 'remote', p.secondary, 12, 1.6);
    s += text(962, 16, t.sync, { size: 11, color: p.ink, weight: 500, maxWidth: lang === 'en' ? 26 : 24, family: sans });
    s += icon(992, 11, 'down', p.secondary, 10, 1.6);
    s += iconButton(1012, 4, 24, 'more', p, { color: p.secondary });
    return s;
  }

  function rail(active) {
    const items = [['changes', 'changes'], ['history', 'history'], ['box', 'box'], ['branch', 'branch'], ['more', 'more']];
    let s = rect(0, 32, 44, H - 32, p.shell) + line(43, 32, 43, H, p.line);
    items.forEach(([label, ico], i) => {
      const y = 48 + i * 38, on = label === active;
      if (on) s += rect(0, y - 6, 44, 32, p.selected, 'none', 0) + rect(0, y - 6, 3, 32, p.accent, 'none', 1);
      s += icon(14, y, ico, on ? p.accent : p.secondary, 16, 1.7);
    });
    s += line(12, 228, 32, 228, p.line);
    s += icon(14, 646, 'settings', p.secondary, 16, 1.7);
    return s;
  }

  function statusBar() {
    let s = rect(0, 676, W, 24, p.shell) + line(0, 676, W, 676, p.line);
    s += text(12, 688, 'HEAD', { size: 11, color: p.muted, weight: 500, maxWidth: 34, family: sans });
    s += text(48, 688, 'b71f9c2', { size: 11, color: p.ink, weight: 500, maxWidth: 56, family: sans, mono: true });
    s += text(112, 688, 'Index', { size: 11, color: p.secondary, maxWidth: 36, family: sans });
    s += text(150, 688, '1', { size: 11, color: p.ink, maxWidth: 12, family: sans });
    s += text(172, 688, 'Worktree', { size: 11, color: p.secondary, maxWidth: 54, family: sans });
    s += text(230, 688, '3', { size: 11, color: p.warning, maxWidth: 12, family: sans });
    s += centerText(520, 688, '↑2 ↓0 · origin/feature/partial-stage', 11, p.secondary, 400, 280, sans);
    s += circle(852, 688, 3, p.good);
    s += text(862, 688, t.synced, { size: 11, color: p.muted, maxWidth: 100, family: sans });
    s += text(1024, 688, 'UTF-8 · LF', { size: 11, color: p.secondary, maxWidth: 70, anchor: 'end', family: sans });
    return s;
  }

  function shell(active) {
    let s = rect(0, 0, W, H, p.bg);
    s += topbar();
    s += rail(active);
    s += statusBar();
    return s;
  }

  /* ----------------------------------------------------------- workspace */

  function section(x, y, label, count, action) {
    let s = text(x, y, label, { size: 11, color: p.secondary, weight: 600, maxWidth: 100, family: sans });
    s += text(x + estimate(label, 11, sans) + 6, y, String(count), { size: 11, color: p.accent, weight: 600, maxWidth: 22, family: sans });
    if (action) s += text(292, y, action, { size: 11, color: p.accent, weight: 500, maxWidth: 96, anchor: 'end', family: sans });
    return s;
  }

  function fileRow(x, y, w, file, { selected = false, hover = false, action = '', conflict = false } = {}) {
    const actionX = x + w - 18, statusX = conflict ? actionX - 46 : actionX - 22, typeX = statusX - 26, textX = x + 28;
    const nameMax = Math.min(116, typeX - textX - 8);
    const shownName = fit(file.name, nameMax, 12, sans);
    const nameW = estimate(shownName, 12, sans);
    const rawPath = file.path.endsWith('/') ? file.path : (file.path.includes('/') ? file.path + '/' : file.path);
    const pathMax = Math.max(0, typeX - (textX + nameW + 9) - 2);
    const shownPath = pathMax > 20 ? fitPath(rawPath, pathMax, 11, sans) : '';
    const statusColor = conflict ? p.danger : file.status === 'M' ? p.warning : file.status === 'A' ? p.good : file.status === 'D' ? p.danger : p.accent;
    let s = `<g data-component="file-row" data-label="${file.name}">`;
    if (selected) s += rect(x, y, w, 24, p.selected, 'none', 2) + rect(x - 4, y, 2, 24, p.accent, 'none', 1);
    else if (hover) s += rect(x, y, w, 24, p.raised, 'none', 2);
    s += rect(x, y + 3, 2, 18, statusColor, 'none', 1);
    s += icon(x + 10, y + 5, file.type === 'sub' ? 'box' : 'file', p.secondary, 14, 1.6);
    s += text(textX, y + 12, shownName, { size: 12, color: p.ink, weight: selected ? 600 : 500, maxWidth: nameMax, family: sans });
    if (shownPath) s += text(textX + nameW + 9, y + 12, shownPath, { size: 11, color: p.muted, maxWidth: pathMax, family: sans });
    s += text(typeX, y + 12, file.type, { size: 11, color: p.muted, maxWidth: 22, family: sans });
    s += centerText(statusX + 7, y + 12, conflict ? '!' : file.status, 11, statusColor, 600, 14, sans);
    if (conflict) s += text(actionX, y + 12, t.resolve, { size: 11, color: p.danger, weight: 500, maxWidth: 34, anchor: 'end', family: sans });
    else if (action) s += icon(actionX, y + 4, action === 'stage' ? 'plus' : 'minus', action === 'stage' ? p.accent : p.secondary, 14, 1.8);
    return s + '</g>';
  }

  function fileColumn() {
    const title = t.workspace, titleW = estimate(title, 13, sans);
    let s = rect(44, 32, 260, H - 56, p.surface) + line(303, 32, 303, 676, p.line);
    s += text(56, 46, title, { size: 13, color: p.ink, weight: 600, maxWidth: 82, family: sans });
    s += text(56 + titleW + 9, 46, t.changes(4), { size: 11, color: p.secondary, maxWidth: 90, family: sans });
    s += iconButton(274, 34, 24, 'more', p, { color: p.secondary });
    s += section(56, 72, t.unstaged, 3, t.stageAll);
    content.unstaged.forEach((file, i) => {
      s += fileRow(48, 84 + i * 24, 248, file, { selected: i === 0, action: i === 0 ? 'stage' : '' });
    });
    s += section(56, 168, t.staged, 1, t.unstageAll);
    s += fileRow(48, 180, 248, { ...content.staged[0], path: t.projectRoot }, { action: 'unstage' });
    s += line(48, 216, 296, 216, p.line);
    s += text(56, 232, 'stage.go', { size: 13, color: p.ink, weight: 600, maxWidth: 220, family: sans });
    s += text(56, 250, 'internal/gitrepo/stage.go', { size: 11, color: p.muted, maxWidth: 238, family: sans });
    s += text(56, 272, `Go · ${t.modified} · ${t.hunk(1)} · +4 −1`, { size: 11, color: p.secondary, maxWidth: 238, family: sans });
    const stageW = lang === 'en' ? 76 : 70;
    const discardX = 56 + stageW + 12;
    s += button(56, 288, stageW, 22, t.stageFile, p, { primary: true, size: 11, family: sans, radius: 3 });
    s += text(discardX, 299, t.discard, { size: 11, color: p.danger, weight: 500, maxWidth: 54, family: sans });
    s += text(discardX + estimate(t.discard, 11, sans) + 14, 299, t.open, { size: 11, color: p.secondary, maxWidth: 40, family: sans });
    s += text(56, 324, t.keyboardHint, { size: 11, color: p.muted, maxWidth: 238, family: sans });
    return s;
  }

  function diffColumn() {
    const x = 304, y = 32, w = 470;
    let s = rect(x, y, w, H - 56, p.surface) + line(x + w, y, x + w, 676, p.line);
    s += rect(x, y, w, 30, p.raised) + line(x, y + 30, x + w, y + 30, p.line);
    s += icon(x + 12, y + 8, 'file', p.secondary, 13, 1.6);
    s += text(x + 30, y + 15, content.diffPath, { size: 11, color: p.ink, weight: 500, maxWidth: 250, family: sans, mono: true });
    const segW = lang === 'en' ? 116 : 92;
    const segX = x + 310;
    s += rect(segX, y + 4, segW, 20, p.surface, p.line, 3);
    s += centerText(segX + segW / 4, y + 14, t.unified, 11, p.ink, 600, segW / 2 - 8, sans);
    s += centerText(segX + segW * 3 / 4, y + 14, t.split, 11, p.secondary, 400, segW / 2 - 8, sans);
    s += iconButton(x + 440, y + 4, 24, 'more', p, { color: p.secondary });
    s += rect(x, y + 30, w, 26, p.surface);
    let tabX = x + 12;
    s += text(tabX, y + 43, t.changesTab, { size: 11, color: p.accent, weight: 600, maxWidth: 80, family: sans });
    s += rect(tabX, y + 53, estimate(t.changesTab, 11, sans), 2, p.accent, 'none', 1);
    tabX += estimate(t.changesTab, 11, sans) + 20;
    s += text(tabX, y + 43, t.historyTab(2), { size: 11, color: p.secondary, maxWidth: 70, family: sans });
    tabX += estimate(t.historyTab(2), 11, sans) + 22;
    s += text(tabX, y + 43, t.infoTab, { size: 11, color: p.secondary, maxWidth: 40, family: sans });
    s += line(x, y + 56, x + w, y + 56, p.line);
    s += rect(x, y + 56, w, 24, p.raised);
    s += text(x + 12, y + 68, '@@ −40,22 +40,25 @@', { size: 11, color: p.secondary, maxWidth: 260, family: sans, mono: true });
    s += text(x + 392, y + 68, '1 / 1', { size: 11, color: p.muted, maxWidth: 40, anchor: 'end', family: sans });
    s += line(x, y + 80, x + w, y + 80, p.line);
    const top = y + 80, lh = 19, gutter = 58;
    s += rect(x, top, gutter, c2DiffRows.length * lh, p.raised);
    c2DiffRows.forEach((r, i) => {
      const yy = top + i * lh;
      if (r.kind !== 'ctx') s += rect(x, yy, w, lh, r.kind === 'add' ? p.add : p.del);
      if (i >= 15 && i <= 18) {
        s += rect(x, yy, w, lh, p.selectOverlay);
        s += rect(x, yy, 2, lh, p.accent, 'none', 0);
      }
      s += text(x + 6, yy + 10, r.oldNo, { size: 11, color: p.muted, maxWidth: 20, family: sans, mono: true });
      s += text(x + 28, yy + 10, r.newNo, { size: 11, color: p.muted, maxWidth: 20, family: sans, mono: true });
      s += centerText(x + 50, yy + 10, r.kind === 'add' ? '+' : r.kind === 'del' ? '−' : '', 11, r.kind === 'add' ? p.good : r.kind === 'del' ? p.danger : p.muted, 500, 12, sans);
      s += text(x + gutter + 8, yy + 10, r.code, { size: 11, color: p.ink, maxWidth: w - gutter - 14, family: sans, mono: true });
    });
    s += line(x + gutter, top, x + gutter, top + c2DiffRows.length * lh, p.line);
    const bottom = top + c2DiffRows.length * lh;
    s += rect(x, bottom, w, 26, p.raised) + line(x, bottom + 26, x + w, bottom + 26, p.line);
    const stageLines = t.stageLines(4);
    const stageW = Math.max(64, estimate(stageLines, 11, sans) + 22);
    const stageX = x + w - 12 - stageW;
    const clearX = stageX - 16;
    s += text(x + 12, bottom + 13, t.linesSelected(4), { size: 11, color: p.accent, weight: 600, maxWidth: 90, family: sans });
    s += text(x + 104, bottom + 13, t.selectHint, { size: 11, color: p.muted, maxWidth: Math.max(70, clearX - x - 114), family: sans });
    s += text(clearX, bottom + 13, t.clear, { size: 11, color: p.secondary, maxWidth: 44, anchor: 'end', family: sans });
    s += button(stageX, bottom + 4, stageW, 22, stageLines, p, { primary: true, size: 11, family: sans, radius: 3 });
    const foot = bottom + 26;
    s += line(x, foot, x + w, foot, p.line);
    s += rect(x + 12, foot + 11, 80, 6, p.raised, 'none', 2);
    s += rect(x + 12, foot + 11, 62, 6, p.good, 'none', 2);
    s += rect(x + 76, foot + 11, 16, 6, p.danger, 'none', 2);
    s += text(x + 100, foot + 14, t.footerSummary(4, 1, 1), { size: 11, color: p.secondary, maxWidth: 154, family: sans });
    s += checkbox(x + 246, foot + 8, t.ignoreWhitespace, false, p, { size: 11 });
    s += text(x + w - 30, foot + 14, t.context(3), { size: 11, color: p.secondary, maxWidth: 80, anchor: 'end', family: sans });
    s += icon(x + w - 20, foot + 8, 'down', p.secondary, 12, 1.6);
    return s;
  }

  function commitColumn() {
    const x = 774, w = 266;
    let s = rect(x, 32, w, H - 56, p.surface) + line(x, 32, x, 676, p.line);
    s += text(x + 16, 46, t.commit, { size: 13, color: p.ink, weight: 600, maxWidth: 60, family: sans });
    s += text(x + w - 16, 46, t.stagedCount(1), { size: 11, color: p.secondary, maxWidth: 92, anchor: 'end', family: sans });
    s += text(x + 16, 68, t.commitTo(content.branch), { size: 11, color: p.muted, maxWidth: w - 32, family: sans });
    s += input(x + 12, 78, w - 26, 28, content.subject, p, { size: 12, family: sans, radius: 3 });
    s += input(x + 12, 114, w - 26, 44, t.description, p, { placeholder: true, size: 11, family: sans, radius: 3 });
    s += text(x + 16, 176, 'README.md · +18 −4', { size: 11, color: p.secondary, maxWidth: 200, family: sans });
    s += button(x + 12, 184, w - 26, 28, t.commitFile(1), p, { primary: true, size: 12, family: sans, radius: 3 });
    s += rect(x + w - 46, 190, 24, 16, p.keycapOverlay, 'none', 3);
    s += centerText(x + w - 34, 198, '⌘↩', 10, p.onAccent, 500, 22, sans);
    s += text(x + 16, 230, t.commitOptions, { size: 11, color: p.secondary, maxWidth: 210, family: sans });
    s += icon(x + w - 20, 224, 'down', p.muted, 12, 1.6);
    s += line(x + 12, 248, x + w - 12, 248, p.line);
    s += text(x + 16, 268, t.lastCommit, { size: 11, color: p.muted, weight: 600, maxWidth: 70, family: sans });
    s += text(x + 16, 286, 'b71f9c2 · feat: preserve partial staging', { size: 11, color: p.secondary, maxWidth: 218, family: sans });
    s += icon(x + w - 20, 280, 'history', p.muted, 12, 1.6);
    return s;
  }

  function workspace() {
    let s = shell('changes');
    s += fileColumn();
    s += diffColumn();
    s += commitColumn();
    return s;
  }

  /* ------------------------------------------------------------- history */

  function filterRow() {
    const titleW = estimate(t.history, 15, sans);
    const subX = 56 + titleW + 10;
    let s = rect(44, 32, 696, 38, p.surface) + line(44, 70, 740, 70, p.line);
    s += text(56, 50, t.history, { size: 15, color: p.ink, weight: 600, maxWidth: 70, family: sans });
    s += text(subX, 50, t.commitsSummary(12, 2), { size: 11, color: p.secondary, maxWidth: 430 - subX, family: sans });
    s += rect(440, 40, 64, 20, p.raised, p.line, 3);
    s += text(448, 50, t.all, { size: 11, color: p.ink, weight: 500, maxWidth: 32, family: sans });
    s += icon(486, 44, 'down', p.secondary, 12, 1.6);
    s += rect(516, 40, 208, 20, p.surface, p.strong, 3);
    s += icon(524, 44, 'search', p.muted, 12, 1.6);
    s += text(542, 50, t.searchHistory, { size: 11, color: p.muted, maxWidth: 170, family: sans });
    return s;
  }

  function historyTable() {
    const x = 44, w = 696, top = 92;
    let s = rect(x, 70, w, 22, p.raised);
    const cols = [[t.graph, 52], [t.subjectRefs, 172], [t.author, 472], [t.when, 562], ['SHA', 632]];
    cols.forEach(([label, colX]) => { s += text(colX, 81, label, { size: 11, color: p.secondary, weight: 600, maxWidth: 110, family: sans }); });
    s += icon(714, 75, 'more', p.secondary, 12, 1.6);
    s += line(x, 92, x + w, 92, p.line);
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
        s += pill(rx, y - 8, ref, p, { fill: p.raised, stroke: p.line, color: p.secondary, height: 16, size: 11, radius: 3, maxWidth: rw, family: sans });
        rx += rw + 4;
      });
      s += text(472, y, 'dev', { size: 11, color: p.secondary, maxWidth: 50, family: sans });
      s += text(562, y, i < 2 ? '10:42' : t.yesterday, { size: 11, color: p.muted, maxWidth: 56, family: sans });
      s += text(632, y, c.sha, { size: 11, color: p.secondary, maxWidth: 58, family: sans, mono: true });
      if (i === 0) s += icon(716, y - 6, 'more', p.secondary, 12, 1.6);
      s += line(x, y + 12, x + w, y + 12, i === 0 ? p.strong : p.line);
    });
    s += line(44, 388, 740, 388, p.line);
    s += text(52, 404, t.lanes, { size: 11, color: p.secondary, weight: 600, maxWidth: 40, family: sans });
    [['main', p.main], ['fix', p.fix], ['feature', p.topic], ['topic', p.topic2]].forEach(([label, color], i) => {
      const lx = 100 + i * 56;
      s += circle(lx, 404, 3, color);
      s += text(lx + 8, 404, label, { size: 11, color: p.secondary, maxWidth: 46, family: sans });
    });
    s += centerText(392, 404, t.loadEarlier, 11, p.accent, 500, 150, sans);
    s += text(724, 404, '12 / 128', { size: 11, color: p.muted, maxWidth: 70, anchor: 'end', family: sans });
    return s;
  }

  function historyDetail() {
    const x = 740, w = 300;
    const description = lang === 'zh' ? content.description : t.descriptionText;
    let s = rect(x, 32, w, H - 56, p.surface) + line(x, 32, x, 676, p.line);
    s += text(756, 50, t.commitDetails, { size: 12, color: p.ink, weight: 600, maxWidth: 130, family: sans });
    s += iconButton(1008, 36, 24, 'close', p, { color: p.secondary });
    s += paragraph(756, 78, 'feat: preserve partial staging', 268, { size: 15, color: p.ink, lineHeight: 19, maxLines: 2, family: sans });
    s += paragraph(756, 104, description, 268, { size: 11, color: p.secondary, lineHeight: 15, maxLines: 2, family: sans });
    s += text(756, 128, `b71f9c2 · dev · ${t.today} 10:42`, { size: 11, color: p.secondary, maxWidth: 250, family: sans, mono: true });
    s += pill(756, 144, 'HEAD', p, { fill: p.accent, stroke: p.accent, color: p.onAccent, height: 18, size: 11, radius: 9, maxWidth: 42, family: sans });
    s += pill(804, 144, content.branch, p, { fill: p.raised, stroke: p.line, color: p.secondary, height: 18, size: 11, radius: 9, maxWidth: 180, family: sans });
    s += line(756, 172, 1020, 172, p.line);
    s += text(756, 188, `${t.filesChanged(2)} · +18 −4`, { size: 11, color: p.secondary, maxWidth: 240, family: sans });
    s += icon(756, 200, 'file', p.secondary, 12, 1.6) + text(774, 206, 'internal/gitrepo/stage.go', { size: 11, color: p.ink, maxWidth: 200, family: sans });
    s += text(1020, 206, 'M', { size: 11, color: p.warning, weight: 600, maxWidth: 14, anchor: 'end', family: sans });
    s += icon(756, 222, 'file', p.secondary, 12, 1.6) + text(774, 228, 'internal/desktop/workspace.go', { size: 11, color: p.ink, maxWidth: 200, family: sans });
    s += text(1020, 228, 'M', { size: 11, color: p.warning, weight: 600, maxWidth: 14, anchor: 'end', family: sans });
    s += text(756, 260, t.parent, { size: 11, color: p.secondary, weight: 600, maxWidth: 60, family: sans });
    s += text(756, 278, 'a43bc18 · Preserve commit message draft', { size: 11, color: p.ink, maxWidth: 250, family: sans });
    s += text(756, 304, t.author, { size: 11, color: p.secondary, weight: 600, maxWidth: 40, family: sans });
    s += text(1020, 304, `dev · ${t.today} 10:42 · ${t.unsigned}`, { size: 11, color: p.secondary, maxWidth: 180, anchor: 'end', family: sans });
    s += line(756, 322, 1020, 322, p.line);
    s += text(756, 342, t.copySha, { size: 11, color: p.accent, weight: 500, maxWidth: 70, family: sans });
    s += text(824, 342, t.openTerminal, { size: 11, color: p.accent, weight: 500, maxWidth: 100, family: sans });
    s += text(1004, 342, t.more, { size: 11, color: p.secondary, weight: 500, maxWidth: 40, anchor: 'end', family: sans });
    s += icon(1012, 336, 'down', p.muted, 10, 1.6);
    s += text(756, 370, t.onCurrentBranch, { size: 11, color: p.muted, maxWidth: 240, family: sans });
    return s;
  }

  function history() {
    let s = shell('history');
    s += filterRow();
    s += historyTable();
    s += historyDetail();
    return s;
  }

  return { workspace, history };
}

export function lightMeta(locale) {
  return {
    id: `C2L-${locale}`,
    name: '密集仪表 · 日光版',
    line: 'C2 三栏、24px 行和交互不变；日光配色与中英文界面',
    notes: [
      '日光模式用白色内容面、冷灰外壳和更深的蓝色强调，状态色保持同一语义。',
      '英文标签更长的地方改用自适应宽度：标签页、分段控件、选择动作条和页脚。',
      '英文页脚省略 hunk 数，避免与「忽略空白」和上下文控件挤在一起。',
    ],
  };
}
