'use client';
import { RainbowKitProvider, RainbowKitAuthenticationProvider } from '@rainbow-me/rainbowkit';
import { QueryClientProvider } from '@tanstack/react-query';
import * as React from 'react';
import { WagmiProvider } from 'wagmi';
import { base } from 'wagmi/chains';

import { APP_NAME } from '@/config/base';
import { config, queryClient } from '@/config/wagmi';
import { useAuthStatus } from '@/hooks/useAuthStatus';
import { useRainbowKitTheme } from '@/hooks/useRainbowKitTheme';
import { authenticationAdapter } from '@/lib/auth/siwe-adapter';
import { useMiniApp } from '@/provider/miniapp';
import { getChainById, getDefaultChain, getDefaultChainId } from '@/utils/chains';
import '@rainbow-me/rainbowkit/styles.css';

export function DAppProvider({ children }: React.PropsWithChildren<unknown>) {
  const rainbowKitTheme = useRainbowKitTheme();
  const authStatus = useAuthStatus();
  const { isMiniApp } = useMiniApp();

  const baseChain = React.useMemo(() => getChainById(base.id) ?? getDefaultChain(), []);
  const initialChain = isMiniApp ? baseChain : getDefaultChain();
  const initialChainId = isMiniApp ? base.id : getDefaultChainId();
  const rainbowKitKey = isMiniApp ? 'miniapp' : 'browser';

  return (
    <WagmiProvider config={config}>
      <QueryClientProvider client={queryClient}>
        <RainbowKitAuthenticationProvider adapter={authenticationAdapter} status={authStatus}>
          <RainbowKitProvider
            key={rainbowKitKey}
            theme={rainbowKitTheme}
            locale="en-US"
            appInfo={{ appName: APP_NAME }}
            initialChain={initialChain}
            id={initialChainId ? String(initialChainId) : undefined}
          >
            {children}
          </RainbowKitProvider>
        </RainbowKitAuthenticationProvider>
      </QueryClientProvider>
    </WagmiProvider>
  );
}
