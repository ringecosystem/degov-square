'use client';
import { RainbowKitProvider, RainbowKitAuthenticationProvider } from '@rainbow-me/rainbowkit';
import { QueryClientProvider } from '@tanstack/react-query';
import * as React from 'react';
import { WagmiProvider } from 'wagmi';

import { APP_NAME } from '@/config/base';
import { config, queryClient } from '@/config/wagmi';
import { useAuthStatus } from '@/hooks/useAuthStatus';
import { useRainbowKitTheme } from '@/hooks/useRainbowKitTheme';
import { authenticationAdapter } from '@/lib/auth/siwe-adapter';
import { getDefaultChain, getDefaultChainId } from '@/utils/chains';
import '@rainbow-me/rainbowkit/styles.css';

export function DAppProvider({ children }: React.PropsWithChildren<unknown>) {
  const rainbowKitTheme = useRainbowKitTheme();
  const authStatus = useAuthStatus();
  
  return (
    <WagmiProvider config={config}>
      <QueryClientProvider client={queryClient}>
        <RainbowKitAuthenticationProvider
          adapter={authenticationAdapter}
          status={authStatus}
        >
          <RainbowKitProvider
            theme={rainbowKitTheme}
            locale="en-US"
            appInfo={{ appName: APP_NAME }}
            initialChain={getDefaultChain()}
            id={getDefaultChainId() ? String(getDefaultChainId()) : undefined}
          >
            {children}
          </RainbowKitProvider>
        </RainbowKitAuthenticationProvider>
      </QueryClientProvider>
    </WagmiProvider>
  );
}
