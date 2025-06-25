import Image from 'next/image';
import Link from 'next/link';
export function Footer() {
  const year = new Date().getFullYear();
  return (
    <footer className="container hidden items-center justify-between py-[10px] md:flex">
      <p className="text-muted-foreground text-[14px] font-medium">&copy; {year} RingDAO</p>
      <div className="flex items-center gap-[10px]">
        <Link
          href="https://github.com/ringecosystem/degov/blob/main/LICENSE.md"
          className="text-muted-foreground hover:text-foreground text-[14px] font-medium transition-colors"
          target="_blank"
          rel="noopener noreferrer"
        >
          Licenses
        </Link>
        <Link
          href="https://github.com/ringecosystem/degov/discussions"
          className="text-muted-foreground hover:text-foreground text-[14px] font-medium transition-colors"
          target="_blank"
          rel="noopener noreferrer"
        >
          Help
        </Link>
        <Link
          href="https://x.com/ai_degov"
          className="bg-muted flex size-[24px] flex-shrink-0 items-center justify-center rounded-full transition-opacity hover:opacity-80"
          target="_blank"
          rel="noopener noreferrer"
        >
          <Image src="/social/x.svg" alt="Twitter" width={12} height={12} />
        </Link>
        <Link
          href="https://t.me/DeGov_AI"
          className="bg-muted flex size-[24px] flex-shrink-0 items-center justify-center rounded-full transition-opacity hover:opacity-80"
          target="_blank"
          rel="noopener noreferrer"
        >
          <Image src="/social/telegram.svg" alt="Telegram" width={12} height={10} />
        </Link>
        <Link
          href="https://github.com/ringecosystem/degov"
          className="bg-muted flex size-[24px] flex-shrink-0 items-center justify-center rounded-full transition-opacity hover:opacity-80"
          target="_blank"
          rel="noopener noreferrer"
        >
          <Image src="/social/github.svg" alt="GitHub" width={10.714} height={12.857} />
        </Link>
      </div>
    </footer>
  );
}
