import Image from 'next/image';
import Link from 'next/link';

import { ConnectButton } from '@/components/connect-button';

export function Header() {
  return (
    <header className="border-border border-b py-[20px]">
      <div className="container flex items-center justify-between">
        <Link href="/">
          <Image src="/logo.svg" alt="DeGov.AI" width={127} height={26} />
        </Link>
        <ConnectButton />
      </div>
    </header>
  );
}
