import { NotificationLayoutClient } from './notification-layout-client';

import type { Metadata } from 'next';

export const metadata: Metadata = {
  robots: {
    index: false,
    follow: false
  }
};

export default function NotificationLayout({ children }: { children: React.ReactNode }) {
  return <NotificationLayoutClient>{children}</NotificationLayoutClient>;
}
