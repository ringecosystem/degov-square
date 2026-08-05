import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const webRoot = join(fileURLToPath(new URL('..', import.meta.url)));

const read = (path) => readFileSync(join(webRoot, path), 'utf8');

const routePolicies = [
  {
    routeClass: '/add/**',
    owner: 'Square DAO onboarding',
    purpose: 'DAO submission and review workflow',
    publicness: 'transactional/write',
    expectedStatus: 'reachable browser flow; workflow auth/state is enforced in app code',
    indexPolicy: 'noindex,nofollow; excluded from robots and sitemap',
    layout: 'src/app/add/layout.tsx',
    robotsPath: '/add'
  },
  {
    routeClass: '/oauth/**',
    owner: 'Square OAuth authorization',
    purpose: 'OAuth consent and callback-adjacent authorization flow',
    publicness: 'callback/auth',
    expectedStatus: 'reachable browser flow when required OAuth params are present',
    indexPolicy: 'noindex,nofollow; excluded from robots, sitemap, analytics, and canonicals',
    layout: 'src/app/oauth/layout.tsx',
    robotsPath: '/oauth',
    sensitiveParams: ['redirect_uri', 'state', 'code', 'code_challenge']
  },
  {
    routeClass: '/notification/**',
    owner: 'Square notifications',
    purpose: 'User notification preferences and subscriptions',
    publicness: 'authenticated/user-specific',
    expectedStatus: 'anonymous users are gated by auth UI; signed-in users manage subscriptions',
    indexPolicy: 'noindex,nofollow; excluded from robots, sitemap, analytics, and canonicals',
    layout: 'src/app/notification/layout.tsx',
    robotsPath: '/notification'
  },
  {
    routeClass: '/setting/**',
    owner: 'Square DAO settings',
    purpose: 'DAO administration, safe, treasury, and advanced settings workflows',
    publicness: 'authenticated/admin/write',
    expectedStatus: 'reachable browser shell; authorization remains independent from robots',
    indexPolicy: 'noindex,nofollow; excluded from robots, sitemap, analytics, and canonicals',
    layout: 'src/app/setting/layout.tsx',
    robotsPath: '/setting'
  },
  {
    routeClass: '/privacy',
    owner: 'Square legal',
    purpose: 'Public privacy notice',
    publicness: 'public informational',
    expectedStatus: '200 public page',
    indexPolicy: 'indexable by explicit product/legal decision; not part of private noindex set',
    page: 'src/app/privacy/page.tsx'
  },
  {
    routeClass: '/terms',
    owner: 'Square legal',
    purpose: 'Public terms of service',
    publicness: 'public informational',
    expectedStatus: '200 public page',
    indexPolicy: 'indexable by explicit product/legal decision; not part of private noindex set',
    page: 'src/app/terms/page.tsx'
  },
  {
    routeClass: 'invalid routes',
    owner: 'Square routing',
    purpose: 'Unknown or invalid URLs',
    publicness: 'error/not-found',
    expectedStatus: '404 not found',
    indexPolicy: 'noindex,nofollow',
    page: 'src/app/not-found.tsx'
  }
];

const privatePolicies = routePolicies.filter((policy) => policy.layout);
const policyFields = [
  'routeClass',
  'owner',
  'purpose',
  'publicness',
  'expectedStatus',
  'indexPolicy'
];

const assertContains = (source, expected, label) => {
  assert.ok(source.includes(expected), `${label} must include ${expected}`);
};

for (const policy of routePolicies) {
  for (const field of policyFields) {
    assert.ok(policy[field], `${policy.routeClass} must define ${field}`);
  }
}

for (const policy of privatePolicies) {
  const source = read(policy.layout);
  assertContains(source, 'export const metadata', `${policy.layout} metadata export`);
  assertContains(source, 'robots', `${policy.layout} robots metadata`);
  assertContains(source, 'index: false', `${policy.layout} noindex metadata`);
  assertContains(source, 'follow: false', `${policy.layout} nofollow metadata`);
  assert.ok(
    !/(canonical|openGraph|twitter)\s*:/.test(source),
    `${policy.layout} must not promote private routes with canonical or social metadata`
  );
}

const robotsSource = read('src/app/robots.ts');
for (const policy of privatePolicies) {
  assertContains(
    robotsSource,
    `'${policy.robotsPath}'`,
    `robots.txt policy for ${policy.routeClass}`
  );
}
assert.ok(
  !/disallow\s*:\s*['"]\/['"]/.test(robotsSource),
  'robots.txt must not block the root path'
);
assert.ok(
  !/(?:disallow|allow)\s*:\s*\[[^\]]*['"]\/_next/.test(robotsSource),
  'robots.txt must not block Next.js assets'
);

const sitemapSource = read('src/app/sitemap.ts');
assertContains(sitemapSource, "url: 'https://square.degov.ai/'", 'sitemap public root');
for (const policy of privatePolicies) {
  assert.ok(
    !sitemapSource.includes(policy.robotsPath),
    `sitemap must not include ${policy.routeClass}`
  );
}

const rootPageSource = read('src/app/page.tsx');
assertContains(rootPageSource, 'canonical: canonicalUrl', 'public root canonical');
assertContains(rootPageSource, 'url: canonicalUrl', 'public root Open Graph URL');

const analyticsSource = read('src/app/analytics.tsx');
for (const policy of privatePolicies) {
  assertContains(
    analyticsSource,
    `'${policy.robotsPath}'`,
    `analytics exclusion for ${policy.routeClass}`
  );
}
assertContains(analyticsSource, 'new URL(pathname, APP_URL)', 'analytics strips query parameters');
assert.ok(
  !/useSearchParams|window\.location\.search|location\.href/.test(analyticsSource),
  'analytics must not include query strings that may contain OAuth or auth values'
);

const oauthLayoutSource = read('src/app/oauth/layout.tsx');
const oauthPageSource = read('src/app/oauth/authorize/page.tsx');
const oauthClientSource = read('src/app/oauth/authorize/oauth-authorize-client.tsx');
const middlewareSource = read('src/middleware.ts');
assert.ok(
  !/(redirect_uri|state|code|code_challenge)/.test(`${oauthLayoutSource}\n${oauthPageSource}`),
  'OAuth server metadata/page shell must not read or emit OAuth query parameters'
);
assert.ok(
  !/useSearchParams/.test(oauthClientSource),
  'OAuth route must not use useSearchParams because it can serialize query values into server HTML'
);
assertContains(oauthClientSource, 'window.location.hash.slice(1)', 'OAuth hash query parsing');
assertContains(
  middlewareSource,
  "pathname !== '/oauth/authorize'",
  'OAuth authorize middleware guard'
);
assertContains(
  middlewareSource,
  'redirectUrl.hash = request.nextUrl.search.slice(1)',
  'OAuth query-to-hash redirect'
);
assertContains(middlewareSource, "matcher: ['/oauth/authorize']", 'OAuth middleware matcher');

const notFoundSource = read('src/app/not-found.tsx');
assertContains(notFoundSource, 'export const metadata', 'global not-found metadata');
assertContains(notFoundSource, 'index: false', 'global not-found noindex metadata');
assertContains(notFoundSource, 'follow: false', 'global not-found nofollow metadata');

const legalPages = ['src/app/privacy/page.tsx', 'src/app/terms/page.tsx'];
for (const page of legalPages) {
  const source = read(page);
  assert.ok(!/robots\s*:/.test(source), `${page} must remain outside the private noindex policy`);
  assertContains(source, 'export const metadata', `${page} explicit metadata`);
}

console.log(`Verified ${routePolicies.length} Square route index policies.`);
