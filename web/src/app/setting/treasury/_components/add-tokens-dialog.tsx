'use client';

import { useState } from 'react';
import { z } from 'zod';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { isAddress } from 'viem';

import { Button } from '@/components/ui/button';
import { LoadedButton } from '@/components/ui/loaded-button';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { InputSelect } from '@/components/ui/input-select';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@/components/ui/form';
import { Separator } from '@/components/ui/separator';
import { TokenStandard, tokenStandardOptions } from '@/config/dao';

// Token type definition
export type Token = {
  id: number;
  address: string;
  name: string;
  symbol: string;
  owner: string;
  tokenLogo: string;
  type: 'ERC20' | 'ERC721';
  chainId: number;
};

// Token schema for validation
const tokenSchema = z.object({
  contractAddress: z
    .string()
    .min(1, { message: 'Contract address is required' })
    .refine(
      (value) => isAddress(value),
      'Invalid token contract address format'
    ) as z.ZodType<`0x${string}`>,
  tokenType: z.enum([TokenStandard.ERC20, TokenStandard.ERC721], {
    required_error: 'Token type is required'
  })
});

type TokenFormValues = z.infer<typeof tokenSchema>;

export interface AddTokensDialogProps {
  open: boolean;
  onOpenChange: (value: boolean) => void;
  onAddToken: (token: Token) => void;
  isLoading?: boolean;
}

export function AddTokensDialog({
  open,
  onOpenChange,
  onAddToken,
  isLoading = false
}: AddTokensDialogProps) {
  const [searchResults, setSearchResults] = useState<Token[]>([]);
  const [selectedTokens, setSelectedTokens] = useState<number[]>([]);

  // Mock token search function
  const mockTokenSearch = (address: `0x${string}`, type: string): Token[] => {
    return [
      {
        id: 1,
        address: address,
        name: 'Governance RING',
        symbol: 'gRING',
        owner: '0x3B9F644BC66573f36DaD6920974766628ET5FE890',
        tokenLogo: '/example/token1.svg',
        type: type === 'ERC-20' ? 'ERC20' : 'ERC721',
        chainId: 1
      },
      {
        id: 2,
        address: address,
        name: 'USD Coin',
        symbol: 'USDC',
        owner: '0x3B9F644BC66573f36DaD6920974766628ET5FE890',
        tokenLogo: '/example/token2.svg',
        type: type === 'ERC-20' ? 'ERC20' : 'ERC721',
        chainId: 1
      }
    ];
  };

  const form = useForm<TokenFormValues>({
    resolver: zodResolver(tokenSchema),
    defaultValues: {
      contractAddress: '' as `0x${string}`,
      tokenType: TokenStandard.ERC20
    }
  });

  const onSearch = (values: TokenFormValues) => {
    const results = mockTokenSearch(values.contractAddress, values.tokenType);
    setSearchResults(results);
  };

  const toggleToken = (tokenId: number) => {
    setSelectedTokens((prev) =>
      prev.includes(tokenId) ? prev.filter((id) => id !== tokenId) : [...prev, tokenId]
    );
  };

  const handleAddTokens = () => {
    searchResults
      .filter((token) => selectedTokens.includes(token.id))
      .forEach((token) => onAddToken(token));

    // Close dialog and reset state
    onOpenChange(false);
    setSelectedTokens([]);
    setSearchResults([]);
    form.reset();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="border-border/20 bg-card w-[660px] rounded-[26px] p-[20px]">
        <DialogHeader className="flex w-full flex-row items-center justify-between">
          <DialogTitle className="text-[18px] font-normal">Add Tokens</DialogTitle>
        </DialogHeader>

        <Separator className="my-0" />
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSearch)} className="flex flex-col gap-6">
            <FormField
              control={form.control}
              name="contractAddress"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Token Contract Address</FormLabel>
                  <FormControl>
                    <InputSelect
                      placeholder="Enter token contract address"
                      selectPlaceholder="Select Standard"
                      options={tokenStandardOptions}
                      selectValue={form.watch('tokenType')}
                      onSelectChange={(value) => form.setValue('tokenType', value as TokenStandard)}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </form>
        </Form>

        <div className="bg-background flex flex-col gap-[20px] rounded-[14px] p-[20px]">
          <div className="flex items-center gap-[20px]">
            <span className="text-muted-foreground w-[100px]">Token Name</span>
            <span>Governance RING</span>
          </div>
          <div className="flex items-center gap-[20px]">
            <span className="text-muted-foreground w-[100px]">Token Symbol</span>
            <span>gRING</span>
          </div>
          <div className="flex items-center gap-[20px]">
            <span className="text-muted-foreground w-[100px]">Token Owner</span>
            <span>0x3B9F644BC66573f36DaD692097476628ET5FE890</span>
          </div>
        </div>

        <Separator className="my-4" />

        <div className="grid grid-cols-2 gap-[20px]">
          <Button
            className="border-border/20 bg-card rounded-full border"
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <LoadedButton
            className="rounded-full"
            variant="default"
            isLoading={isLoading}
            onClick={handleAddTokens}
            disabled={selectedTokens.length === 0}
          >
            Add to Treasury
          </LoadedButton>
        </div>
      </DialogContent>
    </Dialog>
  );
}
