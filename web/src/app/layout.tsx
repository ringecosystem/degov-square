import { Geist, Geist_Mono } from 'next/font/google';

import './globals.css';
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
  title: 'DeGov.AI',
  description:
    'DeGov.AI is a AI-powered platform for decentralized governance, built on the Openzeppelin contracts.'
};

export default function RootLayout({
  children
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={`${geistSans.variable} ${geistMono.variable} antialiased`}>
        <NextThemeProvider>{children}</NextThemeProvider>
      </body>
    </html>
  );
}
