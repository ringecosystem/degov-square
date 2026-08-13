import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { createServer } from 'node:http';
import { once } from 'node:events';

const fixtureDao = {
  id: 'fixture-dao',
  name: 'Fixture Governance DAO',
  code: 'fixture',
  chainId: 1,
  chainName: 'Fixture Chain',
  seq: 1,
  timeSyncd: '2026-01-01T00:00:00Z',
  ctime: '2026-01-01T00:00:00Z',
  utime: '2026-01-01T00:00:00Z',
  state: 'ACTIVE',
  tags: [],
  chips: [],
  metricsCountMembers: 10,
  metricsCountProposals: 4,
  metricsCountVote: 20,
  metricsSumPower: '100',
  endpoint: 'https://fixture.degov.ai/',
  logo: 'https://fixture.degov.ai/logo.png',
  chainLogo: 'https://fixture.degov.ai/chain.png',
  liked: false,
  lastProposal: null
};

let fixtureFails = false;
const fixtureServer = createServer((_request, response) => {
  response.setHeader('content-type', 'application/json');
  if (fixtureFails) {
    response.statusCode = 503;
    response.end(JSON.stringify({ errors: [{ message: 'fixture unavailable' }] }));
    return;
  }

  response.end(JSON.stringify({ data: { daos: [fixtureDao] } }));
});

fixtureServer.listen(0, '127.0.0.1');
await once(fixtureServer, 'listening');
const fixtureAddress = fixtureServer.address();
assert.ok(fixtureAddress && typeof fixtureAddress === 'object');

const portProbe = createServer();
portProbe.listen(0, '127.0.0.1');
await once(portProbe, 'listening');
const appAddress = portProbe.address();
assert.ok(appAddress && typeof appAddress === 'object');
const appPort = appAddress.port;
await new Promise((resolve, reject) =>
  portProbe.close((error) => (error ? reject(error) : resolve()))
);

const app = spawn('pnpm', ['exec', 'next', 'dev', '--turbopack', '-p', String(appPort)], {
  cwd: new URL('..', import.meta.url),
  env: {
    ...process.env,
    NEXT_PUBLIC_GRAPHQL_ENDPOINT: `http://127.0.0.1:${fixtureAddress.port}/graphql`
  },
  stdio: ['ignore', 'pipe', 'pipe']
});

let appLogs = '';
app.stdout.on('data', (chunk) => {
  appLogs += chunk;
});
app.stderr.on('data', (chunk) => {
  appLogs += chunk;
});

async function fetchRenderedHome() {
  let lastError;
  for (let attempt = 0; attempt < 60; attempt += 1) {
    try {
      const response = await fetch(`http://127.0.0.1:${appPort}/`);
      if (response.ok) return response.text();
      lastError = new Error(`HTTP ${response.status}`);
    } catch (error) {
      lastError = error;
    }
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  throw new Error(`Square test server did not become ready: ${lastError}\n${appLogs}`);
}

try {
  const successHtml = await fetchRenderedHome();
  assert.match(successHtml, /Fixture Governance DAO/);
  assert.match(successHtml, /href="https:\/\/fixture\.degov\.ai\/"/);
  assert.match(successHtml, /All DAOs/);
  assert.doesNotMatch(
    successHtml,
    /Representative public DAOs|Last server read:|ItemList|Square indexes public DAO governance sites|DeGov public DAO registry|Directory count:/
  );

  fixtureFails = true;
  const failureHtml = await fetchRenderedHome();
  assert.match(failureHtml, /All DAOs/);
  assert.doesNotMatch(failureHtml, /Fixture Governance DAO|href="https:\/\/fixture\.degov\.ai\/"/);
  assert.doesNotMatch(
    failureHtml,
    /temporarily unavailable in server HTML|DAO directory data is temporarily unavailable|Representative DAO links are temporarily unavailable|Directory count:/
  );

  console.log('Verified rendered Square content provenance contract.');
} finally {
  app.kill('SIGTERM');
  await Promise.race([once(app, 'exit'), new Promise((resolve) => setTimeout(resolve, 5_000))]);
  await new Promise((resolve, reject) =>
    fixtureServer.close((error) => (error ? reject(error) : resolve()))
  );
}
