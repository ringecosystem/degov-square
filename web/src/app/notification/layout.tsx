'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { cn } from '@/lib/utils';
import Image from 'next/image';

const NAVS = [
  { label: 'Subscription', path: '/notification/subscription' },
  { label: 'Subscribed DAOs', path: '/notification/subscribed-daos' },
  { label: 'Subscribed Proposals', path: '/notification/subscribed-proposals' }
];

export default function NotificationLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  return (
    <div className="container py-6">
      <Link href="/" className="mb-6 flex items-center gap-2">
        <Image
          src="/back.svg"
          alt="back"
          width={32}
          height={32}
          className="size-[32px] flex-shrink-0"
        />
        <h1 className="text-2xl font-bold">Notifications Settings</h1>
      </Link>

      <div className="flex min-h-[80vh] w-full gap-6">
        <aside className="w-[270px] flex-shrink-0">
          <div className="flex flex-col gap-2">
            {NAVS.map((nav) => (
              <Link
                key={nav.path}
                href={nav.path}
                className={cn(
                  'rounded-[10px] px-4 py-3 text-left text-base transition-colors',
                  pathname === nav.path ||
                    (pathname === '/notification' && nav.path === '/notification/subscription')
                    ? 'bg-card'
                    : 'hover:bg-background/80 text-muted-foreground'
                )}
              >
                {nav.label}
              </Link>
            ))}
          </div>
        </aside>

        <main className="flex-1">{children}</main>
      </div>
    </div>
  );
}
