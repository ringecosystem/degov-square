import { HomeClient } from './_components/home-client';
import { getPublicDaoDirectory } from '@/lib/public-dao-directory';

import type { Metadata } from 'next';

const title = 'DeGov Square DAO Directory';
const description =
  'Discover public DAO governance sites, proposal activity, supported networks, and DeGov-managed governance communities.';
const canonicalUrl = 'https://square.degov.ai/';
const shareImageUrl =
  'https://raw.githubusercontent.com/ringecosystem/degov-registry/refs/heads/main/assets/common/degov-1024.png';

export const metadata: Metadata = {
  title,
  description,
  alternates: {
    canonical: canonicalUrl
  },
  openGraph: {
    type: 'website',
    siteName: 'DeGov Square',
    title,
    description,
    url: canonicalUrl,
    images: [
      {
        url: shareImageUrl,
        width: 1024,
        height: 1024,
        alt: 'DeGov Square'
      }
    ]
  },
  twitter: {
    card: 'summary',
    title,
    description,
    images: [shareImageUrl]
  }
};

export const dynamic = 'force-dynamic';

export default async function Home() {
  const initialDirectory = await getPublicDaoDirectory();

  return (
    <HomeClient
      initialDaoData={initialDirectory.daos}
      initialLoadFailed={initialDirectory.failed}
    />
  );
}
