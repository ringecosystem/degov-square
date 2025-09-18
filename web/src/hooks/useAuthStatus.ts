'use client';

import { useMemo } from 'react';

import { useAuthStore } from '@/stores/auth';

/**
 * Hook that provides the authentication status for RainbowKit
 * Returns 'loading' | 'unauthenticated' | 'authenticated'
 */
export const useAuthStatus = () => {
  const { token } = useAuthStore();

  // Determine authentication status
  const status = useMemo(() => {
    // If we haven't loaded the initial token yet, show loading
    if (token === undefined) {
      return 'loading' as const;
    }

    // If we have a valid token, user is authenticated
    if (token) {
      return 'authenticated' as const;
    }

    // Otherwise, user is unauthenticated
    return 'unauthenticated' as const;
  }, [token]);

  return status;
};