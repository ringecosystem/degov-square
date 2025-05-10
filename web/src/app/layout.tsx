import { Geist, Geist_Mono } from 'next/font/google';

import './globals.css';
import { NextThemeProvider } from '@/provider/theme';

import type { Metadata } from 'next';
import { Header } from '@/components/layout/header';
import { Footer } from '@/components/layout/footer';
import { APP_NAME, APP_DESCRIPTION } from '@/config/base';
import { DAppProvider } from '@/provider/dapp';
import { TooltipProvider } from '@/components/ui/tooltip';
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
  description: APP_DESCRIPTION
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
          <TooltipProvider>
            <DAppProvider>
              <div className="bg-background flex min-h-dvh flex-col overflow-hidden font-sans antialiased">
                <Header />
                <main className="flex-1 py-[30px]">{children}</main>
                <Footer />
              </div>
            </DAppProvider>
          </TooltipProvider>
        </NextThemeProvider>
      </body>
    </html>
  );
}
