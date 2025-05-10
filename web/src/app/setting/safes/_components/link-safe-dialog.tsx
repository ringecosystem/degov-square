'use client';

import { useState } from 'react';
import { z } from 'zod';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { isAddress } from 'viem';
import { X } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { LoadedButton } from '@/components/ui/loaded-button';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { InputSelect } from '@/components/ui/input-select';
import { Input } from '@/components/ui/input';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@/components/ui/form';
import { Separator } from '@/components/ui/separator';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select';

// Safe type definition
export type Safe = {
  id: number;
  name: string;
  network: string;
  networkLogo: string;
  safeAddress: string;
  safeLink: string;
};

// Network options
const networkOptions = [
  { label: 'Ethereum', value: 'ethereum', logo: '/example/network1.svg' },
  { label: 'Polygon', value: 'polygon', logo: '/example/network2.svg' },
  { label: 'Optimism', value: 'optimism', logo: '/example/network3.svg' },
  { label: 'Arbitrum', value: 'arbitrum', logo: '/example/network4.svg' }
];

// Safe schema for validation
const safeSchema = z.object({
  safeName: z.string().min(1, { message: 'Safe name is required' }),
  safeAddress: z
    .string()
    .min(1, { message: 'Safe address is required' })
    .refine((value) => isAddress(value), 'Invalid safe address format') as z.ZodType<`0x${string}`>,
  network: z.string({
    required_error: 'Network is required'
  })
});

type SafeFormValues = z.infer<typeof safeSchema>;

export interface LinkSafeDialogProps {
  open: boolean;
  onOpenChange: (value: boolean) => void;
  onAddSafe: (safe: Safe) => void;
  isLoading?: boolean;
}

export function LinkSafeDialog({
  open,
  onOpenChange,
  onAddSafe,
  isLoading = false
}: LinkSafeDialogProps) {
  const form = useForm<SafeFormValues>({
    resolver: zodResolver(safeSchema),
    defaultValues: {
      safeName: '',
      safeAddress: '' as `0x${string}`,
      network: 'ethereum'
    },
    mode: 'onChange'
  });

  const handleAddSafe = (values: SafeFormValues) => {
    const selectedNetworkInfo = networkOptions.find((network) => network.value === values.network);

    // Create a new safe object with the form values
    const newSafe: Safe = {
      id: Math.floor(Math.random() * 1000), // Generate a random ID for demo purposes
      name: values.safeName,
      network: selectedNetworkInfo?.label || 'Ethereum',
      networkLogo: selectedNetworkInfo?.logo || '/example/network1.svg',
      safeAddress: values.safeAddress,
      safeLink: `https://safe.gnosis.io/app/eth:${values.safeAddress}/balances`
    };

    // Add the new safe
    onAddSafe(newSafe);

    // Close dialog and reset form
    onOpenChange(false);
    form.reset();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="border-border/20 bg-card w-[660px] rounded-[26px] p-[20px]">
        <DialogHeader className="flex w-full flex-row items-center justify-between">
          <DialogTitle className="text-[18px] font-normal">Link Safe</DialogTitle>
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={() => onOpenChange(false)}
          >
            <X className="h-4 w-4" />
          </Button>
        </DialogHeader>

        <Separator className="my-6" />
        <Form {...form}>
          <form onSubmit={form.handleSubmit(handleAddSafe)} className="flex flex-col gap-6">
            <div>
              <FormLabel className="text-foreground/70 mb-2 block">Safe Name</FormLabel>
              <FormField
                control={form.control}
                name="safeName"
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <InputSelect
                        options={networkOptions}
                        {...field}
                        placeholder="Enter your safe name"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div>
              <FormLabel className="text-foreground/70 mb-2 block">Safe Address</FormLabel>
              <FormField
                control={form.control}
                name="safeAddress"
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <Input
                        placeholder="0xC9EA55E644F496D6CaAEDcBAD91dE7481Dcd7517"
                        className="border-border/20 h-12 rounded-lg"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <Separator className="my-6" />

            <div className="grid grid-cols-2 gap-[20px]">
              <Button
                className="border-border/20 bg-card h-12 rounded-full border"
                variant="outline"
                onClick={() => onOpenChange(false)}
                type="button"
              >
                Cancel
              </Button>
              <LoadedButton
                className="h-12 rounded-full"
                variant="default"
                isLoading={isLoading}
                type="submit"
                disabled={!form.formState.isValid}
              >
                Add to Safes
              </LoadedButton>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
