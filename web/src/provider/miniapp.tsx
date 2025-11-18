'use client';

import sdk from '@farcaster/miniapp-sdk';
import { createContext, useContext, useEffect, useMemo, useState, useCallback } from 'react';

import {
  detectMiniApp,
  getCachedMiniAppContext,
  getCachedMiniAppStatus,
  notifyMiniAppReady,
  setMiniAppCache
} from '@/utils/miniapp';

interface MiniAppState {
  isMiniApp: boolean;
  isLoading: boolean;
  context: Awaited<typeof sdk.context> | null;
  error: unknown;
  isReady: boolean;
  markReady: () => Promise<void>;
}

const MiniAppContext = createContext<MiniAppState>({
  isMiniApp: false,
  isLoading: true,
  context: null,
  error: null,
  isReady: false,
  markReady: async () => {}
});

export function MiniAppProvider({ children }: React.PropsWithChildren) {
  const [isMiniApp, setIsMiniApp] = useState<boolean>(getCachedMiniAppStatus());
  const [context, setContext] = useState<Awaited<typeof sdk.context> | null>(
    getCachedMiniAppContext()
  );
  const [error, setError] = useState<unknown>(null);
  const [isLoading, setIsLoading] = useState<boolean>(!getCachedMiniAppStatus());
  const [isReady, setIsReady] = useState<boolean>(false);

  useEffect(() => {
    let active = true;

    const bootstrap = async () => {
      setIsLoading(true);
      try {
        const status = await detectMiniApp();
        if (!active) return;

        setIsMiniApp(status);
        if (!status) {
          setContext(null);
          setIsReady(false);
          setError(null);
          setMiniAppCache({ isMiniApp: false, context: null });
          setIsLoading(false);
          return;
        }

        const ctx = await sdk.context;
        if (!active) return;
        setContext(ctx);
        setIsReady(false);
        setError(null);
        setMiniAppCache({ isMiniApp: true, context: ctx });
      } catch (err) {
        if (!active) return;
        console.error('[MiniApp] Initialization failed:', err);
        setError(err);
        setMiniAppCache({ isMiniApp: false, context: null });
      } finally {
        if (active) {
          setIsLoading(false);
        }
      }
    };

    bootstrap();

    return () => {
      active = false;
    };
  }, []);

  const markReady = useCallback(async () => {
    try {
      const sent = await notifyMiniAppReady();
      if (sent) {
        setIsReady(true);
      }
    } catch (err) {
      console.error('[MiniApp] Ready call failed:', err);
      setError(err);
    }
  }, []);

  const value = useMemo<MiniAppState>(
    () => ({
      isMiniApp,
      context,
      isLoading,
      error,
      isReady,
      markReady
    }),
    [context, error, isLoading, isMiniApp, isReady, markReady]
  );

  return <MiniAppContext.Provider value={value}>{children}</MiniAppContext.Provider>;
}

export function useMiniApp() {
  return useContext(MiniAppContext);
}
