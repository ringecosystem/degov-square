import { HomeClient } from './_components/home-client';
import { getPublicDaoDirectory } from '@/lib/public-dao-directory';

import Link from 'next/link';
import type { Metadata } from 'next';

const title = 'DeGov Square DAO Directory';
const description =
  'Discover public DAO governance sites, proposal activity, supported networks, and DeGov-managed governance communities.';
const canonicalUrl = 'https://square.degov.ai/';
const shareImageUrl = 'https://degov.ai/images/degov-social-card.png';
const shareImageAlt = 'DeGov Square DAO Directory — discover public DAO governance.';
const directorySourceUrl = 'https://github.com/ringecosystem/degov-registry';

function isHttpUrl(value: string) {
  try {
    const url = new URL(value);
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
}

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
  const representativeDaos = initialDirectory.daos
    .filter((dao) => isHttpUrl(dao.website))
    .slice(0, 6);
  const directoryReadAt = new Date().toISOString().replace(/\.\d{3}Z$/, 'Z');
  const directoryCountText = initialDirectory.failed
    ? 'temporarily unavailable in server HTML'
    : `${initialDirectory.daos.length} public DAOs`;

  return (
    <main>
      <section className="container flex flex-col gap-[12px] pt-[8px] pb-[24px] md:pb-[32px]">
        <div className="flex flex-col gap-[8px]">
          <h1 className="text-[28px] leading-[1.15] font-semibold md:text-[40px]">
            DeGov Square DAO Directory
          </h1>
          <p className="text-muted-foreground max-w-[760px] text-[15px] leading-[1.6] md:text-[16px]">
            Square indexes public DAO governance sites, supported networks, and proposal activity
            from DeGov registry-backed directory data.
          </p>
        </div>
        <div className="text-muted-foreground flex flex-col gap-[8px] text-[14px] leading-[1.6] md:flex-row md:flex-wrap md:gap-x-[24px]">
          <p>
            Directory count:{' '}
            <strong className="text-foreground font-medium">{directoryCountText}</strong>.
          </p>
          <p>
            Source:{' '}
            <Link
              href={directorySourceUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="text-foreground underline underline-offset-4"
            >
              DeGov public DAO registry
            </Link>
            .
          </p>
          <p>Last server read: {directoryReadAt}.</p>
        </div>
        {representativeDaos.length > 0 ? (
          <nav aria-label="Representative public DAOs" className="flex flex-wrap gap-[8px]">
            {representativeDaos.map((dao) => (
              <Link
                key={dao.id}
                href={dao.website}
                target="_blank"
                rel="noopener noreferrer"
                className="border-border hover:bg-accent rounded-[4px] border px-[10px] py-[6px] text-[14px] transition-colors"
              >
                {dao.name}
              </Link>
            ))}
          </nav>
        ) : (
          <p className="text-muted-foreground text-[14px] leading-[1.6]">
            Representative DAO links are temporarily unavailable in server HTML; the browser
            directory retries against the public API.
          </p>
        )}
      </section>
      <HomeClient
        initialDaoData={initialDirectory.daos}
        initialLoadFailed={initialDirectory.failed}
      />
    </main>
  );
}
