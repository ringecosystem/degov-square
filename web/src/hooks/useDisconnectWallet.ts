import { useCallback } from 'react';
import { useDisconnect } from 'wagmi';

import { useAuthStore } from '@/stores/auth';

export const useDisconnectWallet = () => {
  const { disconnect } = useDisconnect();
  const { clearAuth } = useAuthStore();

  const disconnectWallet = useCallback(
    async (address: `0x${string}`) => {
      // Handle URL auth disconnection - clear everything

      // Handle wallet disconnection - also clear any existing auth states
      try {
        // Clear any tokens first
        clearAuth();

        // Try to revoke wallet permissions
        if (typeof window !== 'undefined' && window?.ethereum?.request) {
          try {
            await window.ethereum.request({
              method: 'wallet_revokePermissions',
              params: [{ eth_accounts: address }]
            });
            console.log('Wallet permissions revoked successfully');
          } catch (error) {
            console.error('Error revoking wallet permissions:', error);
          }
        }

        // Disconnect wallet
        disconnect();
      } catch (error) {
        console.error('Error during wallet disconnection:', error);
      }
    },
    [disconnect, clearAuth]
  );
  return { disconnectWallet };
};
