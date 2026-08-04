import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const webRoot = join(fileURLToPath(new URL('..', import.meta.url)));
const pageSource = readFileSync(join(webRoot, 'src/app/page.tsx'), 'utf8');

const assertContains = (source, expected, label) => {
  assert.ok(source.includes(expected), `${label} must include ${expected}`);
};

assertContains(pageSource, "const canonicalUrl = 'https://square.degov.ai/';", 'Square canonical');
assertContains(pageSource, '<h1', 'Square homepage');
assertContains(pageSource, 'DeGov Square DAO Directory', 'Square H1 text');
assertContains(pageSource, 'Square indexes public DAO governance sites', 'Directory description');
assertContains(pageSource, 'DeGov public DAO registry', 'Directory provenance label');
assertContains(
  pageSource,
  "const directorySourceUrl = 'https://github.com/ringecosystem/degov-registry';",
  'Directory provenance URL'
);
assertContains(pageSource, 'Last server read:', 'Directory freshness text');
assertContains(pageSource, 'function isHttpUrl(value: string)', 'Representative link URL guard');
assertContains(
  pageSource,
  "url.protocol === 'http:' || url.protocol === 'https:'",
  'HTTP URL guard'
);
assertContains(
  pageSource,
  '.filter((dao) => isHttpUrl(dao.website))',
  'Representative DAO selection'
);
assertContains(pageSource, 'aria-label="Representative public DAOs"', 'Representative DAO links');
assertContains(
  pageSource,
  'Representative DAO links are temporarily unavailable in server HTML',
  'Directory unavailable fallback'
);
assertContains(pageSource, 'initialLoadFailed={initialDirectory.failed}', 'Client retry state');

console.log('Verified Square content provenance contract.');
