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
        <h1 className="text-[18px] font-semibold">Notifications Settings</h1>
      </Link>

      <div className="flex w-full gap-[30px]">
        <aside className="w-[300px] flex-shrink-0">
          <div className="flex flex-col gap-[10px]">
            {NAVS.map((nav) => (
              <Link
                key={nav.path}
                href={nav.path}
                className={cn(
                  'border-border hover:border-foreground bg-card text-foreground rounded-[14px] border px-[20px] py-[15px] text-left text-[14px] transition-colors',
                  pathname === nav.path ||
                    (pathname === '/notification' && nav.path === '/notification/subscription')
                    ? 'border-foreground'
                    : ''
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
