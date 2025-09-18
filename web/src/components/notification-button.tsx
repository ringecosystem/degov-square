'use client';

import { useConnectModal } from '@rainbow-me/rainbowkit';
import { useRouter } from 'next/navigation';

import { useAccount } from '@/hooks/useAccount';
import { useSiweAuth } from '@/hooks/useSiweAuth';
import { useAuthStore } from '@/stores/auth';

import { NotificationIcon } from './icons/notification-icon';
import { Button } from './ui/button';

export function NotificationButton() {
  const router = useRouter();
  const { token, localAddress } = useAuthStore();
  const { address } = useAccount();
  const { openConnectModal } = useConnectModal();
  const { authenticate } = useSiweAuth();

  const handleClick = async () => {
    if (!address && !localAddress) {
      openConnectModal?.();
      return;
    }

    if ((address || localAddress) && !token) {
      const result = await authenticate();
      if (!result.success) {
        return;
      }
    }

    router.push('/notification/subscription');
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
