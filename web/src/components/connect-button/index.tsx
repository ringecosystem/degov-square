'use client';
import { useConnectModal } from '@rainbow-me/rainbowkit';

import { useAccount } from '@/hooks/useAccount';

import { Button } from '../ui/button';

import { Connected } from './connected';

export const ConnectButton = () => {
  const { openConnectModal } = useConnectModal();
  const { address, isConnected, isConnecting, isReconnecting, authSource } = useAccount();

  if (isConnecting || isReconnecting) {
    return null;
  }

  if (!isConnected && openConnectModal) {
    return (
      <Button onClick={openConnectModal} className="rounded-[100px]">
        Connect Wallet
      </Button>
    );
  }

  if (address) {
    return <Connected address={address} authSource={authSource} />;
  }

  return null;
};
