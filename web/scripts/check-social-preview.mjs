import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const webRoot = join(fileURLToPath(new URL('..', import.meta.url)));
const pageSource = readFileSync(join(webRoot, 'src/app/page.tsx'), 'utf8');

const expectedImageUrl = 'https://degov.ai/images/degov-social-card.png';
const expectedImageWidth = 1200;
const expectedImageHeight = 630;
const expectedImageType = 'image/png';

const assertContains = (source, expected, label) => {
  assert.ok(source.includes(expected), `${label} must include ${expected}`);
};

assertContains(pageSource, "const canonicalUrl = 'https://square.degov.ai/';", 'Square canonical');
assertContains(pageSource, `const shareImageUrl = '${expectedImageUrl}';`, 'Square share image');
assertContains(pageSource, 'width: 1200', 'Open Graph image width');
assertContains(pageSource, 'height: 630', 'Open Graph image height');
assertContains(pageSource, "type: 'image/png'", 'Open Graph image type');
assertContains(pageSource, 'alt: shareImageAlt', 'Open Graph image alt');
assertContains(pageSource, "card: 'summary_large_image'", 'Twitter large card');
assertContains(pageSource, 'url: shareImageUrl', 'Twitter image object URL');
assertContains(pageSource, 'url: canonicalUrl', 'Open Graph URL');
assertContains(pageSource, 'canonical: canonicalUrl', 'canonical metadata');
assert.ok(
  !pageSource.includes('raw.githubusercontent.com'),
  'social image must not use registry raw square icon fallback'
);
assert.ok(!pageSource.includes("card: 'summary'"), 'Twitter card must not use small summary card');

const response = await fetch(expectedImageUrl, {
  headers: {
    'user-agent': 'degov-square-social-preview-check'
  }
});
assert.equal(response.status, 200, 'social image must return HTTP 200');
assert.ok(
  response.headers.get('content-type')?.startsWith(expectedImageType),
  'social image must return image/png'
);
assert.ok(response.headers.get('cache-control'), 'social image must declare cache behavior');

const body = Buffer.from(await response.arrayBuffer());
assert.ok(
  body.subarray(0, 8).equals(Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a])),
  'social image must be a PNG'
);
assert.equal(body.readUInt32BE(16), expectedImageWidth, 'social image width');
assert.equal(body.readUInt32BE(20), expectedImageHeight, 'social image height');

console.log('Verified Square social preview metadata.');
