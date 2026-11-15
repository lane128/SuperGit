import fs from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { svgDocument } from '../primitives.mjs';
import { workspaceC2, historyC2, detailsC2 } from './screens.mjs';

const root = fileURLToPath(new URL('./', import.meta.url));
const frames = [
  ['C2-workspace.svg', svgDocument(workspaceC2(), 1040, 700, 'SuperGit C2 workspace')],
  ['C2-history.svg', svgDocument(historyC2(), 1040, 700, 'SuperGit C2 history')],
  ['C2-details.svg', svgDocument(detailsC2(), 1040, 900, 'SuperGit C2 states and interactions')],
];

for (const [file, body] of frames) await fs.writeFile(`${root}${file}`, body + '\n');
console.log(`PASS wrote ${frames.length} C2 frames`);
