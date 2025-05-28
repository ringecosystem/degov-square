'use client';

import Image from 'next/image';
import Link from 'next/link';
import { usePathname } from 'next/navigation';

import { cn } from '@/lib/utils';

import { useIsMobileAndSubSection } from './_hooks/isMobileAndSubSection';

const NAVS = [
  { label: 'Subscription', value: 'subscription' },
  { label: 'Subscribed DAOs', value: 'subscribed-daos' },
  { label: 'Subscribed Proposals', value: 'subscribed-proposals' }
];

function NavLink({
  nav,
  isActive,
  href
}: {
  nav: (typeof NAVS)[0];
  isActive: boolean;
  href: string;
}) {
  return (
    <Link
      key={nav.value}
      href={href}
      className={cn(
        'border-border hover:border-foreground bg-card text-foreground rounded-[14px] border px-[20px] py-[15px] text-left text-[14px] transition-colors',
        isActive ? 'border-foreground' : ''
      )}
    >
      {nav.label}
    </Link>
  );
}

const getHref = (nav: (typeof NAVS)[0]) => {
  return `/notification/${nav.value}`;
};

export default function NotificationLayout({ children }: { children: React.ReactNode }) {
  const isMobileAndSubSection = useIsMobileAndSubSection();
  const pathname = usePathname();

  const isActive = (nav: (typeof NAVS)[0]) => {
    return `/notification/${nav.value}` === pathname;
  };

  return (
    <div className="container space-y-[20px]">
      {!isMobileAndSubSection && (
        <Link href="/" className="flex items-center gap-[5px] md:gap-[10px]">
          <Image
            src="/back.svg"
            alt="back"
            width={32}
            height={32}
            className="size-[32px] flex-shrink-0"
          />
          <h1 className="text-[18px] font-semibold">Notifications Settings</h1>
        </Link>
      )}

      <div className="flex w-full flex-col gap-[30px] md:flex-row">
        {!isMobileAndSubSection && (
          <aside className="w-full flex-shrink-0 md:w-[300px]">
            <div className="flex flex-col gap-[20px] md:gap-[10px]">
              {NAVS.map((nav) => (
                <NavLink key={nav.value} nav={nav} isActive={isActive(nav)} href={getHref(nav)} />
              ))}
            </div>
          </aside>
        )}

        <main className="flex-1">{children}</main>
      </div>
    </div>
  );
}
