import { NotFoundIcon } from '@/components/icons/not-found-icon';

import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Page Not Found | DeGov Square',
  robots: {
    index: false,
    follow: false
  }
};

export default function NotFound() {
  return (
    <div className="mx-auto flex min-h-[55dvh] w-full max-w-[640px] flex-col items-center justify-center gap-[24px] px-[20px] text-center">
      <NotFoundIcon className="text-muted-foreground h-16 w-auto" />
      <div className="flex flex-col gap-[8px]">
        <h1 className="text-foreground text-[24px] leading-[1.25] font-semibold">Page Not Found</h1>
        <p className="text-muted-foreground text-[15px] leading-[1.6]">
          The page you are looking for does not exist or has been moved.
        </p>
      </div>
    </div>
  );
}
