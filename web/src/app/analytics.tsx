'use client';

import Script from 'next/script';
import { usePathname } from 'next/navigation';

import { APP_URL, GOOGLE_ANALYTICS_TAG } from '@/config/base';

const excludedPrefixes = ['/add', '/oauth', '/notification', '/setting'];

export function Analytics() {
  const pathname = usePathname() || '/';
  const excluded = excludedPrefixes.some((prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`));

  if (!GOOGLE_ANALYTICS_TAG || excluded) {
    return null;
  }

  const pageLocation = new URL(pathname, APP_URL).toString();
  const analyticsTagLiteral = JSON.stringify(GOOGLE_ANALYTICS_TAG);
  const pathnameLiteral = JSON.stringify(pathname);
  const pageLocationLiteral = JSON.stringify(pageLocation);

  return (
    <>
      <Script
        src={`https://www.googletagmanager.com/gtag/js?id=${GOOGLE_ANALYTICS_TAG}`}
        strategy="afterInteractive"
      />
      <Script id="google-analytics" strategy="afterInteractive">
        {`
          window.dataLayer = window.dataLayer || [];
          function gtag(){dataLayer.push(arguments);}
          gtag('js', new Date());
          gtag('config', ${analyticsTagLiteral}, {
            page_path: ${pathnameLiteral},
            page_location: ${pageLocationLiteral}
          });
        `}
      </Script>
    </>
  );
}
