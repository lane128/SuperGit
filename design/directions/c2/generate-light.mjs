import fs from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { svgDocument } from '../primitives.mjs';
import { createScreens } from './screens-light.mjs';

const root = fileURLToPath(new URL('./', import.meta.url));
const frames = [];
for (const locale of ['zh', 'en']) {
  const { workspace, history } = createScreens(locale);
  frames.push([`C2L-workspace-${locale}.svg`, svgDocument(workspace(), 1040, 700, `SuperGit C2 light workspace (${locale})`)]);
  frames.push([`C2L-history-${locale}.svg`, svgDocument(history(), 1040, 700, `SuperGit C2 light history (${locale})`)]);
}

for (const [file, body] of frames) await fs.writeFile(`${root}${file}`, body + '\n');
console.log(`PASS wrote ${frames.length} C2 light frames`);
