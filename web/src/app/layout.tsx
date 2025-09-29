import { Geist, Geist_Mono } from 'next/font/google';
import './globals.css';
import { ToastContainer } from 'react-toastify';

import { Footer } from '@/components/layout/footer';
import { Header } from '@/components/layout/header';
import { TooltipProvider } from '@/components/ui/tooltip';
import { APP_NAME, APP_DESCRIPTION } from '@/config/base';
import { ConfirmProvider } from '@/provider/confirm';
import { DAppProvider } from '@/provider/dapp';
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
          <QueryProvider>
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
          </QueryProvider>
        </NextThemeProvider>
      </body>
    </html>
  );
}
