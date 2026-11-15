import fs from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { chromium } from '/private/tmp/ui-browser.yhx7Lf/node_modules/playwright/index.mjs';

const root = fileURLToPath(new URL('./', import.meta.url));
const frames = [
  {
    id: 'C2-workspace', file: 'C2-workspace.svg', height: 700,
    labels: ['工作区', '未暂存', '已暂存', '全部暂存', '全部取消', 'stage.go', 'workspace.go', 'core', 'README.md', '提交 1 个文件', '已选 4 行', '暂存 4 行', '忽略空白', '上次提交', '⌘K', '同步'],
  },
  {
    id: 'C2-history', file: 'C2-history.svg', height: 700,
    labels: ['历史', '全部', 'HEAD', 'feat: preserve partial staging', '变更 2 个文件 · +18 −4', '复制 SHA', '更多', '加载更早的提交 →', '12 / 128'],
  },
  {
    id: 'C2-details', file: 'C2-details.svg', height: 900,
    labels: ['文件行状态', '默认', '悬停', '选中', '冲突', '子模块', '部分暂存', '已选 4 行', '提交校验', 'hooks 失败', '破坏性操作', '丢弃 4 个改动？', '命令面板', '暂存全部改动', '子模块与可撤销反馈', '已丢弃 4 个改动', '撤销'],
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
  await fs.writeFile(`${root}check.json`, JSON.stringify(report, null, 2) + '\n');
  console.log(`Rendered ${frames.length} C2 frames.`);
  console.log(`Overflow ${report.overflows.length}; collisions ${report.collisions.length}; missing labels ${report.missing.length}; truncated ${report.truncated.length}; errors ${report.errors.length}.`);
  if (report.overflows.length || report.collisions.length || report.missing.length || report.truncated.length || report.errors.length) {
    console.log(JSON.stringify({ overflows: report.overflows, collisions: report.collisions.slice(0, 40), missing: report.missing, truncated: report.truncated, errors: report.errors }, null, 2));
    process.exitCode = 1;
  }
} finally {
  await browser.close();
}
