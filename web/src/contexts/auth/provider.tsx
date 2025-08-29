'use client';

import { useState, useEffect } from 'react';

import { AuthContext } from './context';

import type { ReactNode } from 'react';

const AUTH_TOKEN_KEY = 'degov_auth_token';

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider = ({ children }: AuthProviderProps) => {
  const [token, setTokenState] = useState<string | null>(null);
  const [isInitialized, setIsInitialized] = useState(false);

  // Initialize token from localStorage on mount
  useEffect(() => {
    if (typeof window !== 'undefined') {
      const storedToken = localStorage.getItem(AUTH_TOKEN_KEY);
      setTokenState(storedToken);
      setIsInitialized(true);

      // Listen for auth token changes (from SIWE adapter)
      const handleTokenChange = (e: Event) => {
        const customEvent = e as CustomEvent<{ token: string | null }>;
        setTokenState(customEvent.detail.token);
      };
      
      window.addEventListener('auth-token-change', handleTokenChange);
      return () => window.removeEventListener('auth-token-change', handleTokenChange);
    }
  }, []);

  const setToken = (newToken: string | null) => {
    setTokenState(newToken);
    
    if (typeof window !== 'undefined') {
      if (newToken) {
        localStorage.setItem(AUTH_TOKEN_KEY, newToken);
      } else {
        localStorage.removeItem(AUTH_TOKEN_KEY);
      }
    }
  };

  const isAuthenticated = Boolean(token);

  // Don't render children until token is initialized from localStorage
  if (!isInitialized) {
    return null;
  }

  return (
    <AuthContext.Provider value={{ token, setToken, isAuthenticated }}>
      {children}
    </AuthContext.Provider>
  );
};
