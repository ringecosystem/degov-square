'use client';

import { NotificationIcon } from './icons/notification-icon';
import { Button } from './ui/button';
import { useRouter } from 'next/navigation';
export function NotificationButton() {
  const router = useRouter();

  const handleClick = () => {
    router.push('/notification');
  };

  return (
    <Button
      className="border-border bg-background hidden h-[36px] w-[36px] items-center justify-center rounded-[10px] border lg:flex"
      variant="outline"
      onClick={handleClick}
    >
      <NotificationIcon className="h-[20px] w-[20px]" />
    </Button>
  );
}
