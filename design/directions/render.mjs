import fs from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { chromium } from '/private/tmp/ui-browser.yhx7Lf/node_modules/playwright/index.mjs';

const root = fileURLToPath(new URL('./', import.meta.url));
const frames = [
  { id: 'A-workspace', file: 'A-workspace.svg', labels: ['stage.go', 'workspace.go', 'core', 'README.md', '全部暂存', '全部取消', '提交暂存内容', '↑2', 'Index → 工作区', '暂存片段', '提交选项'] },
  { id: 'A-history', file: 'A-history.svg', labels: ['HEAD', 'feature/partial-stage', 'feat: preserve partial staging', '完整提交详情', 'Cherry-pick…', 'Revert…'] },
  { id: 'B-workspace', file: 'B-workspace.svg', labels: ['全部改动 3', '全部暂存', '全部取消', '提交 1 个文件', '相关提交'] },
  { id: 'B-history', file: 'B-history.svg', labels: ['全部提交', 'HEAD', '完整提交详情', 'Cherry-pick…', 'Revert…', '加载更早的提交 →'] },
  { id: 'C-workspace', file: 'C-workspace.svg', labels: ['未暂存', '已暂存', 'Index', 'Worktree', '暂存 4 行', '丢弃全部改动…', '提交 1 个文件'] },
  { id: 'C-history', file: 'C-history.svg', labels: ['HEAD', '加载更早的提交', 'Cherry-pick', 'Revert', '提交详情'] },
];
const report = { frames: [], overflows: [], collisions: [], missing: [], truncated: [], errors: [] };

const browser = await chromium.launch({ executablePath: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', headless: true });
try {
  const page = await browser.newPage({ viewport: { width: 1040, height: 700 }, deviceScaleFactor: 2 });
  page.on('pageerror', (e) => report.errors.push(e.message));
  for (const f of frames) {
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
  await page.setViewportSize({ width: 1688, height: 1120 });
  await page.goto(`file://${root}overview.svg`);
  await page.evaluate(() => document.fonts.ready);
  if (await page.locator('parsererror').count()) report.errors.push('overview: parser error');
  await page.screenshot({ path: `${root}overview.png` });
  await fs.writeFile(`${root}check.json`, JSON.stringify(report, null, 2) + '\n');
  console.log(`Rendered ${frames.length} frames + overview.`);
  console.log(`Overflow ${report.overflows.length}; collisions ${report.collisions.length}; missing labels ${report.missing.length}; truncated ${report.truncated.length}; errors ${report.errors.length}.`);
  if (report.overflows.length || report.collisions.length || report.missing.length || report.errors.length) {
    console.log(JSON.stringify({ overflows: report.overflows, collisions: report.collisions.slice(0, 30), missing: report.missing, errors: report.errors }, null, 2));
    process.exitCode = 1;
  }
} finally {
  await browser.close();
}
