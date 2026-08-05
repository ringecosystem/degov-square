import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

import {
  buildSquareDaoClickEventParams,
  getChannelGroupFromReferrer,
  getTargetHostClass,
  SQUARE_DAO_CLICK_EVENT_NAME
} from '../src/lib/analytics.ts';

const webRoot = join(fileURLToPath(new URL('..', import.meta.url)));
const read = (path) => readFileSync(join(webRoot, path), 'utf8');

assert.equal(SQUARE_DAO_CLICK_EVENT_NAME, 'degov_square_dao_click');
assert.deepEqual(
  Object.keys(
    buildSquareDaoClickEventParams({
      daoCode: 'lisk-dao',
      targetUrl: 'https://lisk.degov.ai/',
      referrer: 'https://chatgpt.com/share/example',
      currentHost: 'square.degov.ai'
    })
  ).sort(),
  ['channel_group', 'dao_slug_or_public_id', 'source_surface', 'target_host_class']
);
assert.deepEqual(
  buildSquareDaoClickEventParams({
    daoCode: 'lisk-dao',
    targetUrl: 'https://lisk.degov.ai/',
    referrer: 'https://chatgpt.com/share/example',
    currentHost: 'square.degov.ai'
  }),
  {
    source_surface: 'square',
    dao_slug_or_public_id: 'lisk-dao',
    target_host_class: 'degov-dao-host',
    channel_group: 'ai-search-assistant-referral'
  }
);

assert.equal(getTargetHostClass('https://lisk.degov.ai/'), 'degov-dao-host');
assert.equal(getTargetHostClass('https://collectives.kusama.network/'), 'external-dao-host');
assert.equal(getTargetHostClass('not a url'), 'unknown');
assert.equal(getChannelGroupFromReferrer('', 'square.degov.ai'), 'direct-unknown');
assert.equal(
  getChannelGroupFromReferrer('https://www.square.degov.ai/', 'square.degov.ai'),
  'direct-unknown'
);
assert.equal(
  getChannelGroupFromReferrer('https://www.google.com/search?q=degov', 'square.degov.ai'),
  'organic-search'
);
assert.equal(
  getChannelGroupFromReferrer('https://x.com/ai_degov', 'square.degov.ai'),
  'social-organic'
);
assert.equal(
  getChannelGroupFromReferrer('https://docs.degov.ai/integration', 'square.degov.ai'),
  'documentation-referral'
);
assert.equal(
  getChannelGroupFromReferrer('https://lisk.degov.ai/proposals', 'square.degov.ai'),
  'cross-product-degov-referral'
);
assert.equal(
  getChannelGroupFromReferrer('https://sex.com/post', 'square.degov.ai'),
  'other-external-referral'
);
assert.equal(
  getChannelGroupFromReferrer('https://evilbing.com/post', 'square.degov.ai'),
  'other-external-referral'
);

const analyticsSource = read('src/lib/analytics.ts');
const homeSource = read('src/app/_components/home-client.tsx');
const itemSource = read('src/app/_components/daoItem.tsx');
const combinedSource = `${analyticsSource}\n${homeSource}\n${itemSource}`;

for (const field of [
  'wallet_address',
  'user_profile',
  'notification_destination',
  'oauth',
  'private_profile',
  'window.location.search',
  'window.location.href'
]) {
  assert.ok(!combinedSource.includes(field), `analytics source must not include ${field}`);
}

assert.ok(
  homeSource.includes('onClick={() => trackSquareDaoClick(value.code, value.website)}'),
  'desktop DAO directory links must track public DAO discovery clicks'
);
assert.ok(
  itemSource.includes('onClick={() => trackSquareDaoClick(dao.code, website)}'),
  'mobile DAO directory links must track public DAO discovery clicks'
);
assert.ok(
  analyticsSource.includes('`${SQUARE_DAO_CLICK_EVENT_NAME}:${params.dao_slug_or_public_id}`'),
  'dedupe key must use session plus public DAO id only'
);

console.log('Verified Square DAO discovery analytics contract.');
