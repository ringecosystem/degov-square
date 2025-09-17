'use client';

import { useSearchParams, useRouter, usePathname } from 'next/navigation';
import { useEffect } from 'react';

import { useDisconnectWallet } from '@/hooks/useDisconnectWallet';
import { useAuthStore } from '@/stores/auth';

export const useUrlAuthSync = () => {
  const searchParams = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  const { setLocalAuth, clearAuth, localAddress, token, isLocalMode } = useAuthStore();
  const { disconnectWallet } = useDisconnectWallet();

  useEffect(() => {
    const urlToken = searchParams.get('token');
    const urlAddress = searchParams.get('address');

    if (urlToken && urlAddress) {
      const handleUrlAuth = async () => {
        await disconnectWallet(urlAddress as `0x${string}`);
        // Set local auth mode with URL params
        setLocalAuth(urlAddress, urlToken);

        // Redirect to clean URL after successful auth
        router.replace(pathname);
      };

      handleUrlAuth();
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
