'use client';
import { useConnectModal } from '@rainbow-me/rainbowkit';
import { useAccount } from 'wagmi';

import { Button } from '../ui/button';

import { Connected } from './connected';
import { isSupportedChainById } from '@/utils/chains';
export const ConnectButton = () => {
  const { openConnectModal } = useConnectModal();
  const { chainId, address, isConnected, isConnecting, isReconnecting } = useAccount();

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

  if (!isSupportedChainById(Number(chainId))) {
    return (
      <Button variant="destructive" className="cursor-auto rounded-[100px]">
        Error Chain
      </Button>
    );
  }

  if (address) {
    return <Connected address={address} />;
  }

  return null;
};
