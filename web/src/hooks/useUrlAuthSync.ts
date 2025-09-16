'use client';

import { useSearchParams, useRouter, usePathname } from 'next/navigation';
import { useEffect } from 'react';
import { useAuthStore } from '@/stores/auth';
import { useDisconnectWallet } from '@/hooks/useDisconnectWallet';

export const useUrlAuthSync = () => {
  const searchParams = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  const { setLocalAuth, clearAuth, localAddress, token, isLocalMode } = useAuthStore();
  const { disconnectWallet } = useDisconnectWallet();

  useEffect(() => {
    const urlToken = searchParams.get('token');
    const urlAddress = searchParams.get('address');

    // Debug logging for development
    if (process.env.NODE_ENV === 'development') {
      console.log('URL Auth Sync - Token:', urlToken ? 'Present' : 'Missing');
      console.log('URL Auth Sync - Address:', urlAddress ? urlAddress : 'Missing');
    }

    if (urlToken && urlAddress) {
      if (process.env.NODE_ENV === 'development') {
        console.log('Setting local auth with URL parameters');
      }
      disconnectWallet(urlAddress as `0x${string}`);
      // Set local auth mode with URL params
      setLocalAuth(urlAddress, urlToken);

      // Redirect to clean URL after successful auth
      router.replace(pathname);
    }
  }, [searchParams, setLocalAuth, disconnectWallet, router, pathname]);

  return {
    localAddress,
    token,
    isAuthenticated: !!token,
    isLocalMode: isLocalMode(),
    clearAuth
  };
};
