import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

import {
  buildSquareDirectoryStructuredData,
  DEGOV_PUBLISHER_ID,
  serializeJsonLd,
  SQUARE_CANONICAL_URL,
  SQUARE_COLLECTION_ID,
  SQUARE_ITEM_LIST_ID,
  SQUARE_WEBSITE_ID
} from '../src/lib/square-structured-data.ts';

const webRoot = join(fileURLToPath(new URL('..', import.meta.url)));

const read = (path) => readFileSync(join(webRoot, path), 'utf8');

const visibleDaos = [
  { name: 'Lisk DAO', websiteUrl: 'https://lisk.degov.ai/' },
  { name: 'Kusama Fellowship', websiteUrl: 'https://collectives.kusama.network/' }
];

const hiddenDao = { name: 'Hidden DAO', websiteUrl: 'https://hidden.example/' };
const structuredData = buildSquareDirectoryStructuredData(visibleDaos);
assert.ok(structuredData, 'visible directory items must produce structured data');

const serialized = serializeJsonLd(structuredData);
assert.ok(!serialized.includes('<'), 'serialized JSON-LD must escape literal < characters');
const parsed = JSON.parse(serialized);
const website = parsed.find((item) => item['@type'] === 'WebSite');
const collectionPage = parsed.find((item) => item['@type'] === 'CollectionPage');
assert.ok(website, 'Square directory structured data must include WebSite');
assert.ok(collectionPage, 'Square directory structured data must include CollectionPage');

assert.equal(website['@id'], SQUARE_WEBSITE_ID, 'WebSite @id');
assert.equal(website.url, SQUARE_CANONICAL_URL, 'WebSite URL');
assert.equal(website.publisher['@id'], DEGOV_PUBLISHER_ID, 'WebSite publisher identity');

assert.equal(collectionPage['@id'], SQUARE_COLLECTION_ID, 'CollectionPage @id');
assert.equal(collectionPage.url, SQUARE_CANONICAL_URL, 'CollectionPage URL');
assert.equal(collectionPage.isPartOf['@id'], SQUARE_WEBSITE_ID, 'CollectionPage website link');
assert.equal(
  collectionPage.publisher['@id'],
  DEGOV_PUBLISHER_ID,
  'CollectionPage publisher identity'
);
assert.equal(collectionPage.mainEntity['@id'], SQUARE_ITEM_LIST_ID, 'ItemList @id');
assert.equal(collectionPage.mainEntity['@type'], 'ItemList', 'mainEntity type');
assert.equal(collectionPage.mainEntity.numberOfItems, visibleDaos.length, 'visible item count');

const listItems = collectionPage.mainEntity.itemListElement;
assert.deepEqual(
  listItems.map((item) => item.name),
  visibleDaos.map((dao) => dao.name),
  'JSON-LD item names must match visible DAO labels'
);
assert.deepEqual(
  listItems.map((item) => item.url),
  visibleDaos.map((dao) => dao.websiteUrl),
  'JSON-LD item URLs must match visible DAO links'
);
assert.deepEqual(
  listItems.map((item) => item.position),
  [1, 2],
  'JSON-LD item positions must follow visible order'
);
assert.ok(
  !JSON.stringify(parsed).includes(hiddenDao.name),
  'JSON-LD must not include hidden or client-only DAO items'
);
assert.equal(
  buildSquareDirectoryStructuredData([]),
  null,
  'directory structured data must be omitted without visible items'
);
assert.ok(
  serializeJsonLd(
    buildSquareDirectoryStructuredData([
      { name: 'Bad </script><script>alert(1)</script>', websiteUrl: 'https://example.com/' }
    ])
  ).includes('\\u003c/script>'),
  'JSON-LD serialization must harden script-closing DAO names'
);

const pageSource = read('src/app/page.tsx');
assert.ok(
  pageSource.includes(
    'const directoryStructuredData = buildSquareDirectoryStructuredData(representativeDaos);'
  ),
  'page must derive JSON-LD from the same representative DAO items shown in initial HTML'
);
assert.ok(
  pageSource.includes(
    'dangerouslySetInnerHTML={{ __html: serializeJsonLd(directoryStructuredData) }}'
  ),
  'page must inject escaped JSON-LD rather than raw JSON.stringify output'
);
assert.ok(
  pageSource.includes('id="square-directory-structured-data"'),
  'page must emit an identifiable Square directory JSON-LD block'
);
assert.ok(
  pageSource.includes('href={dao.websiteUrl}'),
  'visible representative DAO links must use the same normalized websiteUrl field'
);

for (const privateRoute of [
  'src/app/add/layout.tsx',
  'src/app/oauth/layout.tsx',
  'src/app/notification/layout.tsx',
  'src/app/setting/layout.tsx',
  'src/app/not-found.tsx'
]) {
  const source = read(privateRoute);
  assert.ok(
    !source.includes('application/ld+json') &&
      !source.includes('square-directory-structured-data') &&
      !source.includes('CollectionPage') &&
      !source.includes('ItemList'),
    `${privateRoute} must not emit convincing public Square directory structured data`
  );
}

console.log('Verified Square directory structured-data contract.');
