import { HomeClient } from './_components/home-client';
import { getPublicDaoDirectory } from '@/lib/public-dao-directory';
import {
  buildSquareDirectoryStructuredData,
  serializeJsonLd,
  SQUARE_CANONICAL_URL
} from '@/lib/square-structured-data';

import type { Metadata } from 'next';

const title = 'DeGov Square DAO Directory';
const description =
  'Discover public DAO governance sites, proposal activity, supported networks, and DeGov-managed governance communities.';
const canonicalUrl = SQUARE_CANONICAL_URL;
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
  const directoryStructuredData = buildSquareDirectoryStructuredData();

  return (
    <>
      <script
        id="square-directory-structured-data"
        type="application/ld+json"
        suppressHydrationWarning
        dangerouslySetInnerHTML={{ __html: serializeJsonLd(directoryStructuredData) }}
      />
      <HomeClient initialDaoData={initialDirectory.daos} />
    </>
  );
}
