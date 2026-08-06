import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const webRoot = join(fileURLToPath(new URL('..', import.meta.url)));
const pageSource = readFileSync(join(webRoot, 'src/app/page.tsx'), 'utf8');

const assertContains = (source, expected, label) => {
  assert.ok(source.includes(expected), `${label} must include ${expected}`);
};

assertContains(pageSource, 'const canonicalUrl = SQUARE_CANONICAL_URL;', 'Square canonical');
assertContains(pageSource, '<h1', 'Square homepage');
assertContains(pageSource, 'DeGov Square DAO Directory', 'Square H1 text');
assertContains(pageSource, 'Square indexes public DAO governance sites', 'Directory description');
assertContains(pageSource, 'DeGov public DAO registry', 'Directory provenance label');
assertContains(
  pageSource,
  "const directorySourceUrl = 'https://github.com/ringecosystem/degov-registry';",
  'Directory provenance URL'
);
assertContains(pageSource, 'initialLoadFailed={initialDirectory.failed}', 'Client retry state');
assertContains(pageSource, 'initialDaoData={initialDirectory.daos}', 'Initial directory data');

assert.ok(
  !pageSource.includes('Last server read:') && !pageSource.includes('new Date()'),
  'page must not present request/render time as directory freshness'
);
assert.ok(
  !pageSource.includes('representativeDaos') &&
    !pageSource.includes('Representative public DAOs') &&
    !pageSource.includes('.slice(0, 6)'),
  'page must not emit a duplicate arbitrary representative DAO navigation'
);

const homeClientSource = readFileSync(join(webRoot, 'src/app/_components/home-client.tsx'), 'utf8');
const daoListSource = readFileSync(join(webRoot, 'src/app/_components/daoList.tsx'), 'utf8');
const daoItemSource = readFileSync(join(webRoot, 'src/app/_components/daoItem.tsx'), 'utf8');

assertContains(
  homeClientSource,
  'const displayDaoData = daoData.length > 0 ? daoData : initialDaoData;',
  'HomeClient initial DAO authority'
);
assertContains(
  homeClientSource,
  'dataSource={filteredAndSortedData}',
  'Desktop initial directory table'
);
assertContains(
  homeClientSource,
  'daoInfo={filteredAndSortedData}',
  'Mobile initial directory list'
);
assertContains(homeClientSource, 'href={value?.website}', 'Desktop DAO links');
assertContains(daoListSource, '<DaoItem {...v} key={v.id}', 'Mobile DAO item render');
assertContains(daoItemSource, 'href={website}', 'Mobile DAO links');

console.log('Verified Square content provenance contract.');
