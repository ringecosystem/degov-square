import { Geist, Geist_Mono } from 'next/font/google';
import Script from 'next/script';
import './globals.css';
import { ToastContainer } from 'react-toastify';

import { Footer } from '@/components/layout/footer';
import { Header } from '@/components/layout/header';
import { TooltipProvider } from '@/components/ui/tooltip';
import {
  APP_NAME,
  APP_DESCRIPTION,
  APP_URL,
  APP_ICON_URL,
  APP_SPLASH_IMAGE_URL,
  APP_SPLASH_BACKGROUND_COLOR,
  GOOGLE_ANALYTICS_TAG
} from '@/config/base';
import { ConfirmProvider } from '@/provider/confirm';
import { DAppProvider } from '@/provider/dapp';
import { MiniAppProvider } from '@/provider/miniapp';
import { QueryProvider } from '@/provider/query';
import { NextThemeProvider } from '@/provider/theme';

import type { Metadata } from 'next';

const geistSans = Geist({
  variable: '--font-geist-sans',
  subsets: ['latin']
});

const geistMono = Geist_Mono({
  variable: '--font-geist-mono',
  subsets: ['latin']
});

export const metadata: Metadata = {
  title: APP_NAME,
  description: APP_DESCRIPTION,
  other: {
    'fc:miniapp': JSON.stringify({
      version: 'next',
      imageUrl: APP_ICON_URL,
      button: {
        title: `Launch ${APP_NAME}`,
        action: {
          type: 'launch_miniapp',
          name: APP_NAME,
          url: APP_URL,
          splashImageUrl: APP_SPLASH_IMAGE_URL,
          splashBackgroundColor: APP_SPLASH_BACKGROUND_COLOR
        }
      }
    })
  }
};

export default function RootLayout({
  children
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={`${geistSans.variable} ${geistMono.variable} antialiased`}>
        <NextThemeProvider>
          <QueryProvider>
            <MiniAppProvider>
              <TooltipProvider>
                <DAppProvider>
                  <ConfirmProvider>
                    <div className="bg-background flex min-h-dvh flex-col font-sans antialiased">
                      <Header />
                      <main className="flex-1 py-[20px] md:py-[30px]">{children}</main>
                      <Footer />
                    </div>
                    <ToastContainer
                      pauseOnFocusLoss={false}
                      theme="dark"
                      className="w-auto text-[14px] md:w-[380px]"
                    />
                  </ConfirmProvider>
                </DAppProvider>
              </TooltipProvider>
            </MiniAppProvider>
          </QueryProvider>
        </NextThemeProvider>

        {GOOGLE_ANALYTICS_TAG && (
          <Script
            src={`https://www.googletagmanager.com/gtag/js?id=${GOOGLE_ANALYTICS_TAG}`}
            strategy="afterInteractive"
          />
        )}
        {GOOGLE_ANALYTICS_TAG && (
          <Script id="google-analytics" strategy="afterInteractive">
            {`
              window.dataLayer = window.dataLayer || [];
              function gtag(){dataLayer.push(arguments);}
              gtag('js', new Date());
              gtag('config', '${GOOGLE_ANALYTICS_TAG}');
            `}
          </Script>
        )}
      </body>
    </html>
  );
}
