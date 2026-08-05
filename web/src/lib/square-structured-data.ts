export const SQUARE_CANONICAL_URL = 'https://square.degov.ai/';
export const DEGOV_PUBLISHER_ID = 'https://degov.ai/#organization';
export const SQUARE_WEBSITE_ID = `${SQUARE_CANONICAL_URL}#website`;
export const SQUARE_COLLECTION_ID = `${SQUARE_CANONICAL_URL}#dao-directory`;
export const SQUARE_ITEM_LIST_ID = `${SQUARE_CANONICAL_URL}#dao-directory-items`;

export type SquareDirectoryStructuredDataItem = {
  name: string;
  websiteUrl: string;
};

export function buildSquareDirectoryStructuredData(items: SquareDirectoryStructuredDataItem[]) {
  if (items.length === 0) return null;

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
      },
      mainEntity: {
        '@id': SQUARE_ITEM_LIST_ID,
        '@type': 'ItemList',
        name: 'Representative public DAOs',
        numberOfItems: items.length,
        itemListElement: items.map((item, index) => ({
          '@type': 'ListItem',
          position: index + 1,
          name: item.name,
          url: item.websiteUrl
        }))
      }
    }
  ];
}

export function serializeJsonLd(value: unknown) {
  return JSON.stringify(value).replace(/</g, '\\u003c');
}
