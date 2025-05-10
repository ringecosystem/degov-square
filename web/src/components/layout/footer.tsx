import Image from 'next/image';
import Link from 'next/link';
export function Footer() {
  const year = new Date().getFullYear();
  return (
    <footer className="container flex items-center justify-between py-[10px]">
      <p className="text-muted-foreground text-[14px] font-medium">&copy; {year} RingDAO</p>
      <div className="flex items-center gap-[10px]">
        <Link
          href="/"
          className="text-muted-foreground hover:text-foreground text-[14px] font-medium transition-colors"
        >
          Licenses
        </Link>
        <Link
          href="/"
          className="text-muted-foreground hover:text-foreground text-[14px] font-medium transition-colors"
        >
          Help
        </Link>
        <Link
          href="/"
          className="bg-muted flex size-[24px] flex-shrink-0 items-center justify-center rounded-full transition-opacity hover:opacity-80"
        >
          <Image src="/social/x.svg" alt="GitHub" width={12} height={12} />
        </Link>
        <Link
          href="/"
          className="bg-muted flex size-[24px] flex-shrink-0 items-center justify-center rounded-full transition-opacity hover:opacity-80"
        >
          <Image src="/social/telegram.svg" alt="Twitter" width={12} height={10} />
        </Link>
        <Link
          href="/"
          className="bg-muted flex size-[24px] flex-shrink-0 items-center justify-center rounded-full transition-opacity hover:opacity-80"
        >
          <Image src="/social/github.svg" alt="Telegram" width={10.714} height={12.857} />
        </Link>
      </div>
    </footer>
  );
}
