'use client';
import Link from 'next/link';
import { cn } from '@/lib/utils';
import Image from 'next/image';
import { useSelectedLayoutSegment, useParams, usePathname } from 'next/navigation';

const NAVS = [
  { label: 'Basic', value: 'basic' },
  { label: 'Treasury', value: 'treasury' },
  { label: 'Safes', value: 'safes' },
  { label: 'Advanced', value: 'advanced' }
];

function NavLink({ nav }: { nav: (typeof NAVS)[0] }) {
  const id = useSelectedLayoutSegment();
  const pathname = usePathname();
  const isActive = `/setting/${id}/${nav.value}` === pathname;
  const href = `/setting/${id}/${nav.value}`;

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

export default function SettingLayout({ children }: { children: React.ReactNode }) {
  const params = useParams();
  const { id } = params || {};

  return (
    <div className="container space-y-[20px]">
      <Link href="/" className="flex items-center gap-2">
        <Image
          src="/back.svg"
          alt="back"
          width={32}
          height={32}
          className="size-[32px] flex-shrink-0"
        />
        <h1 className="text-[18px] font-semibold">DAO Settings</h1>
      </Link>

      <div className="flex w-full gap-[30px]">
        <aside className="w-[300px] flex-shrink-0">
          <div className="flex flex-col gap-[10px]">
            {NAVS.map((nav) => (
              <NavLink key={nav.path} nav={nav} id={id as string} />
            ))}
          </div>
        </aside>

        <main className="flex-1">{children}</main>
      </div>
    </div>
  );
}
