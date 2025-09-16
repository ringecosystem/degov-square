'use client';

import { createAuthenticationAdapter } from '@rainbow-me/rainbowkit';
import { createSiweMessage } from 'viem/siwe';

import { createPublicClient } from '@/lib/graphql/client';
import { QUERY_NONCE, LOGIN_MUTATION } from '@/lib/graphql/queries';
import type { NonceVariables, LoginVariables } from '@/lib/graphql/types';
import { useAuthStore } from '@/stores/auth';
import { extractAddressFromSiweMessage } from '@/utils/siwe';


/**
 * Custom authentication adapter for RainbowKit that integrates with our existing backend
 * This adapter handles the SIWE authentication flow using our GraphQL API
 */
export const authenticationAdapter = createAuthenticationAdapter({
  getNonce: async () => {
    const client = createPublicClient();
    const variables: NonceVariables = { input: { length: 32 } };
    const data = await client.request<{ nonce: string }>(QUERY_NONCE, variables);
    return data.nonce;
  },

  createMessage: ({ nonce, address, chainId }) => {
    return createSiweMessage({
      domain: window.location.host,
      address,
      statement: 'Sign in with Ethereum to DeGov.AI',
      uri: window.location.origin,
      version: '1',
      chainId,
      nonce
    });
  },

  verify: async ({ message, signature }) => {
    const client = createPublicClient();
    const variables: LoginVariables = {
      input: {
        message,
        signature
      }
    };

    const data = await client.request<{ login: { token: string } }>(LOGIN_MUTATION, variables);

    // Store the token with address (ensure token-address binding)
    if (data.login?.token) {
      const address = extractAddressFromSiweMessage(message);
      if (!address) {
        console.error('Failed to extract address from SIWE message');
        return false;
      }
      useAuthStore.getState().setToken(data.login.token);
      useAuthStore.getState().setAddress(address);
      return true;
    }

    return false;
  },

  signOut: async () => {
    useAuthStore.getState().clearAuth();
  }
});
