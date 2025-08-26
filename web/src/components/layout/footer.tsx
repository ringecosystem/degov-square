import Link from 'next/link';

import { Logo } from '@/components/ui/Logo';

export function Footer() {
  const year = new Date().getFullYear();
  return (
    <>
      <div className="container md:hidden">
        <div className="border-border border-t"></div>
      </div>
      <footer className="container py-[40px]">
        <div className="flex flex-col gap-[40px] md:flex-row md:justify-between md:gap-0">
          {/* Logo and Description */}
          <div className="flex flex-col gap-[20px]">
            <a href="https://degov.ai" target="_blank" rel="noopener noreferrer">
              <Logo width={146.5} height={30} className="text-muted-foreground shrink-0" />
            </a>
            <p className="text-muted-foreground text-[14px] font-normal md:h-[80px]">
              DeGov.AI is an open-source tool for DAOs built based on the
              <br className="hidden md:block" />
              OpenZeppelin governor contracts.
            </p>
            <p className="text-muted-foreground hidden text-[14px] font-normal md:block">
              ©{year} RingDAO
            </p>
          </div>

          <div className="flex flex-col gap-[40px] md:flex-row md:items-start md:justify-end md:gap-[120px]">
            {/* Resources Column */}
            <div className="flex flex-col gap-[20px]">
              <h3 className="text-muted-foreground text-[16px] leading-[1.2] font-semibold">
                Resources
              </h3>
              <div className="flex flex-col gap-[20px]">
                <Link
                  href=" https://docs.degov.ai/integration/deploy"
                  className="text-muted-foreground block text-[14px] leading-[1.2] font-normal transition-colors hover:opacity-80"
                >
                  Deploy By Yourself
                </Link>
                <Link
                  href="https://docs.degov.ai/faqs"
                  className="text-muted-foreground block text-[14px] leading-[1.2] font-normal transition-colors hover:opacity-80"
                >
                  FAQs
                </Link>
                <Link
                  href="https://github.com/ringecosystem/degov/blob/main/LICENSE.md"
                  className="text-muted-foreground block text-[14px] leading-[1.2] font-normal transition-colors hover:opacity-80"
                >
                  License
                </Link>
                <Link
                  href="https://docs.degov.ai"
                  className="text-muted-foreground block text-[14px] leading-[1.2] font-normal transition-colors hover:opacity-80"
                >
                  Docs
                </Link>
              </div>
            </div>

            {/* Community Column */}
            <div className="flex flex-col gap-[20px]">
              <h3 className="text-muted-foreground text-[16px] leading-[1.2] font-semibold">
                Community
              </h3>
              <div className="flex flex-col gap-[20px]">
                <Link
                  href="https://x.com/ai_degov"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-muted-foreground block text-[14px] leading-[1.2] font-normal transition-colors hover:opacity-80"
                >
                  X
                </Link>
                <Link
                  href="https://t.me/RingDAO_Hub"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-muted-foreground block text-[14px] leading-[1.2] font-normal transition-colors hover:opacity-80"
                >
                  Telegram
                </Link>
                <Link
                  href="https://github.com/ringecosystem/degov "
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-muted-foreground block text-[14px] leading-[1.2] font-normal transition-colors hover:opacity-80"
                >
                  GitHub
                </Link>
              </div>
            </div>
          </div>

          {/* Copyright at bottom on mobile */}
          <p className="text-muted-foreground text-[14px] font-normal md:hidden">
            ©{year} RingDAO
          </p>
        </div>
      </footer>
    </>
  );
}
