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
assertContains(
  pageSource,
  'const initialDirectory = await getPublicDaoDirectory();',
  'Server DAO directory load'
);
assertContains(pageSource, 'initialDaoData={initialDirectory.daos}', 'Initial directory data');
assertContains(pageSource, 'buildSquareDirectoryStructuredData()', 'Identity-only structured data');

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
assert.ok(
  !pageSource.includes('<section') &&
    !pageSource.includes('Square indexes public DAO governance sites') &&
    !pageSource.includes('DeGov public DAO registry') &&
    !pageSource.includes('directorySourceUrl') &&
    !pageSource.includes('directoryCountText') &&
    !pageSource.includes('initialLoadFailed'),
  'page must not render visible SEO/provenance content'
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
assert.ok(
  !homeClientSource.includes('Discover public DAO governance sites') &&
    !homeClientSource.includes('DAO directory data is temporarily unavailable') &&
    !homeClientSource.includes('initialLoadFailed'),
  'HomeClient header must only show the DAO count before controls'
);

console.log('Verified Square content provenance contract.');
