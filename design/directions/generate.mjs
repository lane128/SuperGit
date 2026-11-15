import fs from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import * as A from './a.mjs';
import * as B from './b.mjs';
import * as C from './c.mjs';
import { W, H, rect, line, text, paragraph, svgDocument } from './primitives.mjs';

const root = fileURLToPath(new URL('./', import.meta.url));
const dirs = [
  { key: 'A', meta: A.meta, workspace: A.workspaceA(), history: A.historyA() },
  { key: 'B', meta: B.meta, workspace: B.workspaceB(), history: B.historyB() },
  { key: 'C', meta: C.meta, workspace: C.workspaceC(), history: C.historyC() },
];

for (const d of dirs) {
  await fs.writeFile(`${root}${d.key}-workspace.svg`, svgDocument(d.workspace, W, H, `${d.key} ${d.meta.name} · 工作区`));
  await fs.writeFile(`${root}${d.key}-history.svg`, svgDocument(d.history, W, H, `${d.key} ${d.meta.name} · 历史`));
}

const OW = 1688, OH = 1120;
const ink = '#1d1d1f', secondary = '#66645e', rule = '#c9c6bd', ground = '#ecebe6';
let o = rect(0, 0, OW, OH, ground);
o += text(48, 52, 'SuperGit 方向对比', { size: 30, color: ink, weight: 600, maxWidth: 800, family: 'serif' });
o += text(48, 84, '同一组仓库数据，三种产品语言。先看结构与节奏，1:1 细节见各方向 PNG。', { size: 14, color: secondary, maxWidth: 1200, family: 'b' });
o += line(48, 108, OW - 48, 108, rule);
o += text(48, 128, '工作区', { size: 11, color: secondary, weight: 600, maxWidth: 100, family: 'b', tracking: 0.3 });
o += text(48, 610, '历史', { size: 11, color: secondary, weight: 600, maxWidth: 100, family: 'b', tracking: 0.3 });

dirs.forEach((d, i) => {
  const x = 48 + i * 560;
  o += `<g transform="translate(${x} 144) scale(0.5)">${d.workspace}</g>`;
  o += rect(x, 144, 520, 350, 'none', rule, 4);
  o += text(x, 520, `${d.key} · ${d.meta.name}`, { size: 15, color: ink, weight: 600, maxWidth: 520, family: 'serif' });
  o += text(x, 542, d.meta.line, { size: 11, color: secondary, maxWidth: 520, family: 'b' });
  d.meta.notes.forEach((note, j) => {
    o += text(x, 566 + j * 18, `· ${note}`, { size: 11, color: secondary, maxWidth: 520, family: 'b' });
  });
  o += `<g transform="translate(${x} 690) scale(0.5)">${d.history}</g>`;
  o += rect(x, 690, 520, 350, 'none', rule, 4);
  o += text(x, 1064, `${d.key} · ${d.meta.name} · 历史`, { size: 13, color: ink, weight: 600, maxWidth: 520, family: 'serif' });
});
o += text(48, 1100, '所有数据为静态样例；未接入 MyGo 原生客户端，未执行 Git 写操作。', { size: 11, color: secondary, maxWidth: 1500, family: 'b' });

await fs.writeFile(`${root}overview.svg`, svgDocument(o, OW, OH, 'SuperGit 方向对比'));
console.log('PASS wrote 6 direction frames and overview.svg');
