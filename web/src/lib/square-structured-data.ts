export const SQUARE_CANONICAL_URL = 'https://square.degov.ai/';
export const DEGOV_PUBLISHER_ID = 'https://degov.ai/#organization';
export const SQUARE_WEBSITE_ID = `${SQUARE_CANONICAL_URL}#website`;
export const SQUARE_COLLECTION_ID = `${SQUARE_CANONICAL_URL}#dao-directory`;

export function buildSquareDirectoryStructuredData() {
  return [
    {
      '@context': 'https://schema.org',
      '@id': SQUARE_WEBSITE_ID,
      '@type': 'WebSite',
      name: 'DeGov Square',
      url: SQUARE_CANONICAL_URL,
      publisher: {
        '@id': DEGOV_PUBLISHER_ID
      }
    },
    {
      '@context': 'https://schema.org',
      '@id': SQUARE_COLLECTION_ID,
      '@type': 'CollectionPage',
      name: 'DeGov Square DAO Directory',
      description:
        'Discover public DAO governance sites, proposal activity, supported networks, and DeGov-managed governance communities.',
      url: SQUARE_CANONICAL_URL,
      isPartOf: {
        '@id': SQUARE_WEBSITE_ID
      },
      publisher: {
        '@id': DEGOV_PUBLISHER_ID
      }
    }
  ];
}

export function serializeJsonLd(value: unknown) {
  return JSON.stringify(value).replace(/</g, '\\u003c');
}
