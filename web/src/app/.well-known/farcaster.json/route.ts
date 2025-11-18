import {
  APP_NAME,
  APP_DESCRIPTION,
  APP_URL,
  APP_ICON_URL,
  APP_SPLASH_IMAGE_URL,
  APP_SPLASH_BACKGROUND_COLOR
} from '@/config/base';

function withValidProperties(properties: Record<string, unknown>) {
  return Object.fromEntries(
    Object.entries(properties).filter(([_, value]) => {
      // Always include boolean values (like noindex)
      if (typeof value === 'boolean') return true;
      // Filter out empty arrays
      if (Array.isArray(value)) return value.length > 0;
      // Filter out empty strings, null, undefined
      return !!value;
    })
  );
}

export async function GET() {
  const isProduction = process.env.NODE_ENV === 'production';

  const miniappConfig: Record<string, unknown> = {
    version: '1',
    name: APP_NAME,
    homeUrl: APP_URL,
    iconUrl: APP_ICON_URL,

    splashImageUrl: APP_SPLASH_IMAGE_URL,
    splashBackgroundColor: APP_SPLASH_BACKGROUND_COLOR,

    primaryCategory: 'utility',
    tags: ['infra', 'dao', 'governance', 'ai'],

    subtitle: 'Onchain governance market',
    description: APP_DESCRIPTION
  };

  if (!isProduction) {
    miniappConfig.noindex = true;
  }

  const farcasterConfig = {
    accountAssociation: {
      header: process.env.MINIAPP_ACCOUNT_ASSOCIATION_HEADER || '',
      payload: process.env.MINIAPP_ACCOUNT_ASSOCIATION_PAYLOAD || '',
      signature: process.env.MINIAPP_ACCOUNT_ASSOCIATION_SIGNATURE || ''
    },
    baseBuilder: {
      ownerAddress: process.env.NEXT_PUBLIC_OWNER_ADDRESS || ''
    },
    miniapp: withValidProperties(miniappConfig)
  };

  return Response.json(farcasterConfig);
}
