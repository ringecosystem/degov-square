import { parseSiweMessage } from 'viem/siwe';

export function extractAddressFromSiweMessage(message: string): string | null {
  try {
    const parsed = parseSiweMessage(message);
    return parsed.address ?? null;
  } catch (error) {
    console.error('Failed to parse SIWE message:', error);
    return null;
  }
}