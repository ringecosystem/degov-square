'use client';

import { useCallback } from 'react';

import { useAuth } from '@/contexts/auth';
import { getToken } from '@/lib/auth/token-manager';

import { useSiweAuth } from './useSiweAuth';

export const useAuthenticatedRequest = () => {
  const { token: contextToken, setToken } = useAuth();
  const { authenticate } = useSiweAuth();

  const executeWithAuth = useCallback(async <T>(
    requestFn: (token: string) => Promise<T>,
    options?: { skipAutoAuth?: boolean }
  ): Promise<T> => {
    let token = getToken() || contextToken;

    // 如果没有token，先尝试自动认证
    if (!token && !options?.skipAutoAuth) {
      const authSuccess = await authenticate();
      if (authSuccess) {
        token = getToken() || contextToken;
      }
    }

    if (!token) {
      throw new Error('Authentication required. Please connect your wallet and sign in.');
    }

    try {
      return await requestFn(token);
    } catch (error) {
      // 检查是否是401错误，如果是则尝试重新认证
      if (error instanceof Error && 
          (error.message.includes('401') || error.message.includes('Unauthorized'))) {
        
        // 清除旧token
        setToken(null);
        
        if (!options?.skipAutoAuth) {
          // 尝试重新认证
          const authSuccess = await authenticate();
          if (authSuccess) {
            const newToken = getToken() || contextToken;
            if (newToken) {
              // 用新token重试请求
              return await requestFn(newToken);
            }
          }
        }
        
        throw new Error('Session expired. Please sign in again.');
      }
      
      // 其他错误直接抛出
      throw error;
    }
  }, [contextToken, authenticate, setToken]);

  return { executeWithAuth };
};