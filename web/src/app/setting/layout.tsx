import { SettingLayoutClient } from './setting-layout-client';

import type { Metadata } from 'next';

export const metadata: Metadata = {
  robots: {
    index: false,
    follow: false
  }
};

export default function SettingLayout({ children }: { children: React.ReactNode }) {
  return <SettingLayoutClient>{children}</SettingLayoutClient>;
}
