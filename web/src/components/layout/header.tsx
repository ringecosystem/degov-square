'use client';

import Image from 'next/image';
import Link from 'next/link';

import { ConnectButton } from '@/components/connect-button';

import { NotificationButton } from '../notification-button';
export function Header() {
  return (
    <header className="border-border bg-background/80 supports-[backdrop-filter:blur(0px)]:bg-background/60 sticky top-0 z-50 border-b py-[10px] backdrop-blur md:py-[20px]">
      <div className="mx-auto flex items-center justify-between px-[10px] md:container">
        <Link href="/">
          <Image
            src="/logo.svg"
            alt="DeGov.AI"
            width={117.2}
            height={24}
            className="block md:hidden"
          />
          <Image
            src="/logo.svg"
            alt="DeGov.AI"
            width={127}
            height={26}
            className="hidden md:block"
          />
        </Link>
        <div className="flex items-center gap-[5px] md:gap-[10px]">
          <ConnectButton />
          <NotificationButton />
        </div>
      </div>
    </header>
  );
}
