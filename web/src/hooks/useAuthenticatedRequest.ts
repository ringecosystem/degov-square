'use client';

import { useCallback } from 'react';

import { useAuthStore } from '@/stores/auth';
import { getToken } from '@/stores/auth';

import { useSiweAuth } from './useSiweAuth';

export const useAuthenticatedRequest = () => {
  const { token: contextToken } = useAuthStore();
  const { authenticate } = useSiweAuth();

  const executeWithAuth = useCallback(
    async <T>(
      requestFn: (token: string) => Promise<T>,
      options?: { skipAutoAuth?: boolean }
    ): Promise<T> => {
      let token = getToken() || contextToken;

      if (!token && !options?.skipAutoAuth) {
        const res = await authenticate();
        if (res?.success) {
          token = getToken() || contextToken;
        }
      }

      if (!token) {
        throw new Error('Authentication required. Please connect your wallet and sign in.');
      }

      try {
        return await requestFn(token);
      } catch (error) {
        if (
          error instanceof Error &&
          (error.message.includes('401') || error.message.includes('Unauthorized'))
        ) {
          useAuthStore.getState().setToken(null);

          if (!options?.skipAutoAuth) {
            const res = await authenticate();
            if (res?.success) {
              const newToken = getToken() || contextToken;
              if (newToken) {
                return await requestFn(newToken);
              }
            }
          }

          throw new Error('Session expired. Please sign in again.');
        }

        throw error;
      }
    },
    [contextToken, authenticate]
  );

  return { executeWithAuth };
};
