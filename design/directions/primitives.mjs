// Direction probes: shared drawing helpers, palettes, sample data.
import fs from 'node:fs';

export const W = 1040, H = 700;
const raw = JSON.parse(fs.readFileSync(new URL('../compact/lucide-icons.json', import.meta.url), 'utf8'));

export const SANS = {
  a: '-apple-system, BlinkMacSystemFont, "SF Pro Text", "Helvetica Neue", "PingFang SC", sans-serif',
  b: '"Avenir Next", "PingFang SC", "Helvetica Neue", sans-serif',
  c: '-apple-system, BlinkMacSystemFont, "SF Pro Text", "PingFang SC", sans-serif',
};
export const SERIF = {
  a: '"New York", ui-serif, Georgia, serif',
  b: '"New York", ui-serif, Georgia, serif',
  c: '"New York", ui-serif, Georgia, serif',
};
export const MONO = {
  a: '"SF Mono", Menlo, monospace',
  b: 'Menlo, "SF Mono", monospace',
  c: '"JetBrains Mono", "SF Mono", Menlo, monospace',
};

export const A = {
  bg: '#f2f2f4', shell: '#ececf0', surface: '#ffffff', raised: '#f5f5f7',
  ink: '#1d1d1f', secondary: '#6e6e73', muted: '#86868b',
  line: '#e5e5ea', strong: '#d1d1d6', accent: '#007aff', onAccent: '#ffffff',
  selected: '#e8f1ff', add: '#eaf7ef', del: '#fff0f0',
  good: '#1f7a45', danger: '#c0392b', warning: '#9a6700', warningBg: '#fff8e6',
  main: '#007aff', fix: '#e07b39', topic: '#2f9e6f', topic2: '#8a63c7',
  shadow: 'rgba(0,0,0,0.18)',
};
export const B = {
  bg: '#efede7', shell: '#e7e4dc', surface: '#fbfaf6', raised: '#f3f0e9',
  ink: '#1b1a17', secondary: '#64615a', muted: '#807c73',
  line: '#dcd7cc', strong: '#b9b2a4', accent: '#b4432f', onAccent: '#ffffff',
  selected: '#eee5d6', add: '#e7f1e8', del: '#f8e8e4',
  good: '#2f6b45', danger: '#a33a2a', warning: '#8a5a1f', warningBg: '#f7eeda',
  main: '#b4432f', fix: '#8a6a2f', topic: '#3f6b5a', topic2: '#6b5a7a',
  shadow: 'rgba(0,0,0,0.12)',
};
export const C = {
  bg: '#0d1117', shell: '#11161d', surface: '#151b23', raised: '#1b232d',
  ink: '#e6edf3', secondary: '#9aa7b4', muted: '#77838f',
  line: '#25303c', strong: '#334150', accent: '#5cc8ff', onAccent: '#06151d',
  selected: '#12354a', add: '#123528', del: '#3a1d24',
  good: '#5bd6a0', danger: '#ff7b8a', warning: '#f0b75a', warningBg: '#2b2315',
  main: '#5cc8ff', fix: '#f0b75a', topic: '#5bd6a0', topic2: '#c792ea',
  shadow: 'rgba(0,0,0,0.5)',
};

const aliases = {
  changes: 'file-code-2', file: 'file', folder: 'folder', box: 'folder-git-2',
  branch: 'git-branch', commit: 'git-commit-horizontal', history: 'history',
  plus: 'plus', minus: 'minus', check: 'check', close: 'x', down: 'chevron-down',
  arrow: 'arrow-right', back: 'arrow-left', settings: 'settings-2',
  warning: 'triangle-alert', remote: 'refresh-cw', stash: 'archive',
  conflict: 'git-merge', worktree: 'folder-git-2', search: 'search',
  more: 'ellipsis', info: 'info', error: 'circle-alert', pull: 'download',
  push: 'upload', spinner: 'refresh-cw', trash: 'x', copy: 'copy',
  external: 'external-link', terminal: 'external-link', tag: 'file',
  panel: 'layers', chevronLeft: 'arrow-left', chevronRight: 'chevron-right',
  eye: 'search', filter: 'chevron-down', branchPlus: 'git-branch',
};

export function icon(x, y, name, color, size = 16, stroke = 1.7) {
  const key = aliases[name] || name;
  const source = raw[key];
  if (!source) throw new Error(`Unknown icon: ${name}`);
  const body = source.replace(/^[\s\S]*?<svg[^>]*>/, '').replace(/<\/svg>[\s\S]*$/, '');
  const sw = (stroke * 24 / size).toFixed(2);
  return `<g data-icon="${key}" transform="translate(${x} ${y}) scale(${size / 24})" fill="none" stroke="${color}" stroke-width="${sw}" stroke-linecap="round" stroke-linejoin="round">${body}</g>`;
}

export function estimate(s, size = 13, family = 'a') {
  let n = 0;
  for (const c of String(s ?? '')) {
    const code = c.codePointAt(0);
    if (code > 0x2e80) n += 1.0;
    else if (family === 'mono') n += 0.6;
    else if (/[MW@%]/.test(c)) n += 0.86;
    else if (/[A-Z]/.test(c)) n += 0.66;
    else if (/[il.,:;!|'` ]/.test(c)) n += 0.3;
    else if (/[0-9]/.test(c)) n += 0.56;
    else n += 0.54;
  }
  const factor = family === 'b' ? 1.05 : family === 'serif' ? 0.99 : 1;
  return n * size * factor;
}

export function fit(s, width, size = 13, family = 'a') {
  s = String(s ?? '');
  if (estimate(s, size, family) <= width) return s;
  let out = '';
  for (const c of s) {
    if (estimate(out + c + '…', size, family) > width) break;
    out += c;
  }
  return out + '…';
}

export function fitStart(s, width, size = 13, family = 'a') {
  s = String(s ?? '');
  if (estimate(s, size, family) <= width) return s;
  let out = '';
  for (let i = s.length - 1; i >= 0; i--) {
    const next = s[i] + out;
    if (estimate('…' + next, size, family) > width) break;
    out = next;
  }
  return '…' + out;
}

// Keep the last whole path segments when a path must shrink, so it reads
// "…/desktop/" instead of "…al/desktop/".
export function fitPath(s, width, size = 11, family = 'a') {
  s = String(s ?? '');
  if (estimate(s, size, family) <= width) return s;
  const trailing = s.endsWith('/');
  const parts = s.replace(/\/+$/, '').split('/').filter(Boolean);
  for (let n = 1; n < parts.length; n++) {
    const candidate = `…/${parts.slice(-n).join('/')}${trailing ? '/' : ''}`;
    if (estimate(candidate, size, family) <= width) return candidate;
  }
  return fit(`…/${parts[parts.length - 1] || ''}${trailing ? '/' : ''}`, width, size, family);
}

export const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));

export const rect = (x, y, w, h, fill, stroke = 'none', r = 0, extra = '') =>
  `<rect x="${x}" y="${y}" width="${w}" height="${h}" rx="${r}" fill="${fill}" stroke="${stroke}" ${extra}/>`;
export const line = (x1, y1, x2, y2, color, width = 1, extra = '') =>
  `<line x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}" stroke="${color}" stroke-width="${width}" ${extra}/>`;
export const path = (d, color, width = 1.5, extra = '') =>
  `<path d="${d}" fill="none" stroke="${color}" stroke-width="${width}" stroke-linecap="round" stroke-linejoin="round" ${extra}/>`;
export const circle = (x, y, r, fill, stroke = 'none', sw = 1) =>
  `<circle cx="${x}" cy="${y}" r="${r}" fill="${fill}" stroke="${stroke}" stroke-width="${sw}"/>`;

export function text(x, y, s, { size = 13, color, weight = 400, maxWidth = 1000, anchor = 'start', family = 'a', mono = false, opacity = 1, tracking = 0 } = {}) {
  const fam = mono ? MONO[family] : (family === 'serif' ? SERIF[family] : SANS[family]);
  const shown = fit(s, maxWidth, size, family);
  return `<text x="${x}" y="${y}" font-size="${size}" fill="${color}" font-weight="${weight}" text-anchor="${anchor}" dominant-baseline="central" data-max-width="${maxWidth}" data-family="${family}" font-family="${esc(fam)}"${tracking ? ` letter-spacing="${tracking}"` : ''}${opacity < 1 ? ` opacity="${opacity}"` : ''}>${esc(shown)}${shown !== String(s) ? `<title>${esc(s)}</title>` : ''}</text>`;
}

export function centerText(cx, cy, s, size, color, weight = 400, maxWidth = 1000, family = 'a', mono = false) {
  return text(cx, cy, s, { size, color, weight, maxWidth, anchor: 'middle', family, mono });
}

export function paragraph(x, y, s, width, { size = 11, color, lineHeight = 16, maxLines = 3, family = 'a' } = {}) {
  let rows = [], row = '';
  for (const c of String(s)) {
    if (c === '\n' || estimate(row + c, size, family) > width) {
      rows.push(row);
      row = c === '\n' ? '' : c;
      if (rows.length >= maxLines) break;
    } else row += c;
  }
  if (row && rows.length < maxLines) rows.push(row);
  return rows.map((v, i) => text(x, y + i * lineHeight, v, { size, color, maxWidth: width, family })).join('');
}

export function pill(x, y, label, p, { fill = p.raised, stroke = p.line, color = p.secondary, height = 20, size = 11, weight = 500, radius = 10, family = 'a', maxWidth = 0 } = {}) {
  const w = maxWidth || estimate(label, size, family) + 16;
  return `<g data-pill="${esc(label)}">${rect(x, y, w, height, fill, stroke, radius)}${centerText(x + w / 2, y + height / 2, label, size, color, weight, w - 10, family)}</g>`;
}

export function button(x, y, w, h, label, p, { primary = false, danger = false, disabled = false, iconName = '', iconColor = '', radius = 5, size = 12, family = 'a', fill = '', stroke = '', color = '' } = {}) {
  const bg = fill || (disabled ? p.raised : primary ? p.accent : danger ? p.danger : p.surface);
  const fg = color || (disabled ? p.muted : primary || danger ? p.onAccent : p.ink);
  const bd = stroke || (primary || danger ? bg : p.strong);
  const contentW = w - (iconName ? 22 : 12);
  const cx = x + w / 2 + (iconName ? 8 : 0);
  let s = `<g data-component="button" data-label="${esc(label)}">`;
  s += rect(x, y, w, h, bg, bd, radius);
  if (iconName) s += icon(x + 8, y + (h - 13) / 2, iconName, iconColor || fg, 13, 1.6);
  s += centerText(cx, y + h / 2, label, size, fg, 500, contentW, family);
  return s + '</g>';
}

export function iconButton(x, y, size, name, p, { active = false, color = '', box = false } = {}) {
  const c = color || (active ? p.accent : p.secondary);
  let s = `<g data-component="icon-button" data-label="${esc(name)}">`;
  if (box) s += rect(x, y, size, size, active ? p.selected : 'none', 'none', 4);
  s += icon(x + (size - 14) / 2, y + (size - 14) / 2, name, c, 14, 1.7);
  return s + '</g>';
}

export function input(x, y, w, h, value, p, { placeholder = false, focused = false, size = 12, family = 'a', radius = 4, bg = '', color = '' } = {}) {
  const fill = bg || p.surface;
  return `<g data-component="input">${focused ? rect(x - 2, y - 2, w + 4, h + 4, 'none', p.accent, radius + 2, 'stroke-width="2"') : ''}${rect(x, y, w, h, fill, focused ? p.accent : p.strong, radius)}${text(x + 8, y + h / 2, value, { size, color: color || (placeholder ? p.muted : p.ink), maxWidth: w - 16, family })}</g>`;
}

export function checkbox(x, y, label, checked, p, { size = 12 } = {}) {
  return `<g data-component="checkbox">${rect(x, y, 12, 12, checked ? p.accent : 'none', checked ? p.accent : p.strong, 2)}${checked ? path(`M${x + 3} ${y + 6}l2 2 4-4`, p.onAccent, 1.6) : ''}${text(x + 19, y + 6, label, { size, color: p.secondary, maxWidth: 220 - 19 })}</g>`;
}

export function svgDocument(content, width = W, height = H, title = 'SuperGit direction probe') {
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}" role="img" aria-label="${esc(title)}"><title>${esc(title)}</title>${content}</svg>`;
}

// Same sample data as design/compact/graph.mjs.
export const commits = [
  { sha: 'b71f9c2', lane: 0, subject: 'feat: preserve partial staging', refs: ['HEAD', 'feature/partial-stage'], parents: [1] },
  { sha: '6d20a8e', lane: 0, subject: 'Merge branch fix/refresh', refs: [], parents: [2, 3] },
  { sha: 'c54a916', lane: 0, subject: 'Keep selected file after refresh', refs: [], parents: [4] },
  { sha: '8b2f19c', lane: 1, subject: 'Refresh only changed paths', refs: ['fix/refresh'], parents: [5] },
  { sha: 'a8b1d03', lane: 0, subject: 'Merge three feature branches', refs: ['main'], parents: [6, 7, 8] },
  { sha: '09ce45b', lane: 1, subject: 'Watch index changes', refs: [], parents: [9] },
  { sha: 'a43bc18', lane: 0, subject: 'Preserve commit message draft', refs: [], parents: [10] },
  { sha: 'e92f3a7', lane: 2, subject: 'Read submodule pointers', refs: ['feature/modules'], parents: [10] },
  { sha: 'f1a623b', lane: 3, subject: 'Improve diff navigation', refs: ['feature/diff'], parents: [11] },
  { sha: 'f2d783c', lane: 1, subject: 'Use stable file selection', refs: [], parents: [10] },
  { sha: '61e780b', lane: 0, subject: 'Separate Index and worktree', refs: [], parents: [12] },
  { sha: '73d842c', lane: 3, subject: 'Keep old and new line numbers', refs: [], parents: [12] },
];

export const diffRows = [
  { kind: 'ctx', oldNo: 48, newNo: 48, code: 'func (r *Repository) StageLines(' },
  { kind: 'ctx', oldNo: 49, newNo: 49, code: '    ctx context.Context, patch Patch,' },
  { kind: 'ctx', oldNo: 50, newNo: 50, code: ') error {' },
  { kind: 'ctx', oldNo: 51, newNo: 51, code: '    r.writeMu.Lock()' },
  { kind: 'ctx', oldNo: 52, newNo: 52, code: '    defer r.writeMu.Unlock()' },
  { kind: 'ctx', oldNo: 53, newNo: 53, code: '' },
  { kind: 'del', oldNo: 54, newNo: '', code: '    return r.stageFile(ctx, patch.Path)' },
  { kind: 'add', oldNo: '', newNo: 54, code: '    if err := r.verifyPatch(ctx, patch); err != nil {' },
  { kind: 'add', oldNo: '', newNo: 55, code: '        return err' },
  { kind: 'add', oldNo: '', newNo: 56, code: '    }' },
  { kind: 'add', oldNo: '', newNo: 57, code: '    return r.applyToIndex(ctx, patch)' },
  { kind: 'ctx', oldNo: 55, newNo: 58, code: '}' },
];

export function edgeRoute(a, b, style, slot, row) {
  if (a.x === b.x) return `M${a.x} ${a.y}V${b.y}`;
  const d = Math.sign(b.x - a.x);
  const bend = d > 0 ? a.y + row * 0.5 + slot * 2.5 : b.y - row * 0.5 - slot * 2.5;
  if (style === 'orthogonal') return `M${a.x} ${a.y}V${bend}H${b.x}V${b.y}`;
  if (style === 'diagonal') {
    const run = Math.min(Math.abs(b.x - a.x), row * 0.62), start = d > 0 ? a.y + 3 : b.y - run - 3;
    return `M${a.x} ${a.y}V${start}L${b.x} ${start + run}V${b.y}`;
  }
  const r = 3;
  return `M${a.x} ${a.y}V${bend - r}Q${a.x} ${bend} ${a.x + d * r} ${bend}H${b.x - d * r}Q${b.x} ${bend} ${b.x} ${bend + r}V${b.y}`;
}

export function graph(x, top, { row = 28, lane = 18, style = 'rounded', p, selected = 0, colors, width = 160, edge = 1.5, node = 3.5, mergeNode = 4.6 } = {}) {
  const positions = new Map(commits.map((c, i) => [i, { x: x + 14 + c.lane * lane, y: top + 14 + i * row }]));
  let s = `<g data-graph-style="${style}" data-row="${row}">`;
  for (let i = 0; i < commits.length; i++) {
    const c = commits[i], a = positions.get(i);
    c.parents.forEach((parent, slot) => {
      const b = positions.get(parent), color = colors[c.lane % colors.length];
      if (b) s += `<path data-child="${c.sha}" d="${edgeRoute(a, b, style, slot, row)}" fill="none" stroke="${color}" stroke-width="${edge}"/>`;
      else s += line(a.x, a.y + 4, a.x, a.y + 11, color, edge, 'stroke-dasharray="2 3"');
    });
  }
  for (let i = 0; i < commits.length; i++) {
    const c = commits[i], a = positions.get(i), merge = c.parents.length > 1, color = colors[c.lane % colors.length];
    s += `<g data-node="${c.sha}">${circle(a.x, a.y, merge ? mergeNode : node, merge ? p.surface : color, color, edge)}`;
    if (i === selected) s += circle(a.x, a.y, 1.2, color);
    s += '</g>';
  }
  return s + '</g>';
}

export const content = {
  repo: 'atlas',
  branch: 'feature/partial-stage',
  upstream: 'origin/feature/partial-stage',
  unstaged: [
    { name: 'stage.go', path: 'internal/gitrepo', type: 'go', status: 'M' },
    { name: 'workspace.go', path: 'internal/desktop', type: 'go', status: 'M' },
    { name: 'core', path: 'modules', type: 'sub', status: 'M' },
  ],
  staged: [{ name: 'README.md', path: '项目根目录', type: 'md', status: 'M' }],
  subject: 'feat: preserve partial staging',
  description: '保留部分暂存后的工作区版本。',
  diffPath: 'internal/gitrepo/stage.go',
  hunk: '@@ −48,8 +48,11 @@',
  ahead: 2,
  behind: 0,
  fileHistory: [
    { sha: 'b71f9c2', when: '今天 10:42', subject: 'feat: preserve partial staging' },
    { sha: 'a43bc18', when: '昨天', subject: 'Preserve commit message draft' },
  ],
};
