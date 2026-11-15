import fs from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { chromium } from '/private/tmp/ui-browser.yhx7Lf/node_modules/playwright/index.mjs';

const root = fileURLToPath(new URL('./', import.meta.url));
const frames = [
  {
    id: 'C2L-workspace-zh', file: 'C2L-workspace-zh.svg', height: 700,
    labels: ['工作区', '4 个改动', '未暂存', '已暂存', '全部暂存', '全部取消', 'stage.go', 'workspace.go', 'core', 'README.md', '提交', '提交到 feature/partial-stage', '补充说明（可选）', '提交 1 个文件', '提交选项 · hooks 将运行', '上次提交', '已选 4 行', '暂存 4 行', '忽略空白', '上下文 3', '同步'],
  },
  {
    id: 'C2L-workspace-en', file: 'C2L-workspace-en.svg', height: 700,
    labels: ['Workspace', '4 changes', 'Unstaged', 'Staged', 'Stage all', 'Unstage all', 'stage.go', 'workspace.go', 'core', 'README.md', 'Commit', '1 staged', 'Commit to feature/partial-stage', 'Description (optional)', 'Commit 1 file', 'Commit options · hooks will run', 'Last commit', '4 lines selected', 'Stage 4 lines', 'Ignore whitespace', 'Context 3', 'Sync', 'Search or run a command'],
  },
  {
    id: 'C2L-history-zh', file: 'C2L-history-zh.svg', height: 700,
    labels: ['历史', '12 个提交 · HEAD 领先 2', '全部', '搜索提交、SHA 或引用', '线图', '提交说明与引用', '作者', '时间', '轨道', '加载更早的提交 →', '提交详情', '变更 2 个文件 · +18 −4', '父提交', '复制 SHA', '在终端中打开', '更多', '此提交已在当前分支。'],
  },
  {
    id: 'C2L-history-en', file: 'C2L-history-en.svg', height: 700,
    labels: ['History', '12 commits · HEAD ahead 2', 'All', 'Search commits, SHA, or refs', 'Graph', 'Subject and refs', 'Author', 'When', 'Lanes', 'Load earlier commits →', 'Commit details', '2 files changed · +18 −4', 'Parent', 'Copy SHA', 'Open in terminal', 'More', 'This commit is on the current branch.'],
  },
];
const report = { frames: [], overflows: [], collisions: [], missing: [], truncated: [], errors: [] };

const browser = await chromium.launch({ executablePath: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', headless: true });
try {
  const page = await browser.newPage({ viewport: { width: 1040, height: 700 }, deviceScaleFactor: 2 });
  page.on('pageerror', (e) => report.errors.push(e.message));
  for (const f of frames) {
    await page.setViewportSize({ width: 1040, height: f.height });
    await page.goto(`file://${root}${f.file}`);
    await page.evaluate(() => document.fonts.ready);
    const parser = await page.locator('parsererror').count();
    if (parser) report.errors.push(`${f.id}: parser error`);
    const result = await page.evaluate(() => {
      const svg = document.querySelector('svg');
      const W = Number(svg.getAttribute('width')), H = Number(svg.getAttribute('height'));
      const texts = [...svg.querySelectorAll('text')].map((e) => {
        const b = e.getBBox();
        return { label: e.firstChild?.textContent || '', x: b.x, y: b.y, w: b.width, h: b.height, size: Number(e.getAttribute('font-size')) };
      });
      const overflows = texts.filter((t) => t.x < -0.5 || t.y < -0.5 || t.x + t.w > W + 0.5 || t.y + t.h > H + 0.5);
      const collisions = [];
      for (let i = 0; i < texts.length; i++) for (let j = i + 1; j < texts.length; j++) {
        const a = texts[i], b = texts[j];
        const w = Math.min(a.x + a.w, b.x + b.w) - Math.max(a.x, b.x);
        const h = Math.min(a.y + a.h, b.y + b.h) - Math.max(a.y, b.y);
        if (w > 4 && h > 5 && a.label.trim() && b.label.trim()) collisions.push([a.label, b.label, Math.round(w), Math.round(h)]);
      }
      return { texts, overflows, collisions };
    });
    const labels = new Set(result.texts.map((t) => t.label));
    const missing = f.labels.filter((l) => !labels.has(l));
    const truncated = await page.evaluate(() => [...document.querySelectorAll('text')].filter((e) => e.querySelector('title')).map((e) => e.firstChild?.textContent || ''));
    report.frames.push({ id: f.id, textCount: result.texts.length });
    report.overflows.push(...result.overflows.map((t) => ({ id: f.id, ...t })));
    report.collisions.push(...result.collisions.map((c) => ({ id: f.id, pair: c })));
    report.missing.push(...missing.map((m) => ({ id: f.id, label: m })));
    report.truncated.push(...truncated.map((t) => ({ id: f.id, label: t })));
    await page.screenshot({ path: `${root}${f.id}.png` });
  }
  await fs.writeFile(`${root}check-light.json`, JSON.stringify(report, null, 2) + '\n');
  console.log(`Rendered ${frames.length} C2 light frames.`);
  console.log(`Overflow ${report.overflows.length}; collisions ${report.collisions.length}; missing labels ${report.missing.length}; truncated ${report.truncated.length}; errors ${report.errors.length}.`);
  if (report.overflows.length || report.collisions.length || report.missing.length || report.truncated.length || report.errors.length) {
    console.log(JSON.stringify({ overflows: report.overflows, collisions: report.collisions.slice(0, 60), missing: report.missing, truncated: report.truncated, errors: report.errors }, null, 2));
    process.exitCode = 1;
  }
} finally {
  await browser.close();
}
