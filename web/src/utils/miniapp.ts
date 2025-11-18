import sdk from '@farcaster/miniapp-sdk';

type MiniAppContext = Awaited<typeof sdk.context>;

let cachedIsMiniApp: boolean | null = null;
let cachedContext: MiniAppContext | null = null;

function hasWindow(): boolean {
  return typeof window !== 'undefined';
}

/**
 * Detect if running in Base Mini App environment (with cache)
 */
export async function detectMiniApp(): Promise<boolean> {
  if (!hasWindow()) return false;

  if (cachedIsMiniApp !== null) {
    return cachedIsMiniApp;
  }

  try {
    const status = await sdk.isInMiniApp();
    cachedIsMiniApp = status;
    return status;
  } catch (error) {
    console.error('[MiniApp] Environment detection failed:', error);
    cachedIsMiniApp = false;
    return false;
  }
}

/**
 * Notify Base Mini App container that the app is ready
 */
export async function notifyMiniAppReady(): Promise<boolean> {
  const isMiniApp = await detectMiniApp();
  if (!isMiniApp) return false;

  try {
    await sdk.actions.ready();
    return true;
  } catch (error) {
    console.error('[MiniApp] Ready signal failed:', error);
    return false;
  }
}

/**
 * Manually update cache state (for Hook or Provider use)
 */
export function setMiniAppCache(params: { isMiniApp?: boolean; context?: MiniAppContext | null }) {
  if (typeof params.isMiniApp === 'boolean') {
    cachedIsMiniApp = params.isMiniApp;
  }
  if (params.context !== undefined) {
    cachedContext = params.context ?? null;
  }
}

/**
 * Read cached Mini App environment status
 */
export function getCachedMiniAppStatus(): boolean {
  return cachedIsMiniApp === true;
}

/**
 * Read cached Mini App context
 */
export function getCachedMiniAppContext<T = MiniAppContext>(): T | null {
  return (cachedContext as T | null) ?? null;
}
