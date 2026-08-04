import { HomeClient } from './_components/home-client';
import { getPublicDaoDirectory } from '@/lib/public-dao-directory';

import type { Metadata } from 'next';

const title = 'DeGov Square DAO Directory';
const description =
  'Discover public DAO governance sites, proposal activity, supported networks, and DeGov-managed governance communities.';
const canonicalUrl = 'https://square.degov.ai/';
const shareImageUrl = 'https://degov.ai/images/degov-social-card.png';
const shareImageAlt = 'DeGov Square DAO Directory — discover public DAO governance.';

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
        width: 1200,
        height: 630,
        alt: shareImageAlt,
        type: 'image/png'
      }
    ]
  },
  twitter: {
    card: 'summary_large_image',
    title,
    description,
    images: [
      {
        url: shareImageUrl,
        alt: shareImageAlt
      }
    ]
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
