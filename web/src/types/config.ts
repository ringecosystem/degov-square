export interface TokenDetails {
  contract: string;
  standard: 'ERC20' | 'ERC721';
  symbol?: string;
  name?: string;
  decimals?: number;
  icon?: string;
}
