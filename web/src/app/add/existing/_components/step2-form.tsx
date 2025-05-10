'use client';

import { z } from 'zod';
import { Separator } from '@/components/ui/separator';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@/components/ui/form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { isAddress } from 'viem';
import { InfoIcon } from 'lucide-react';
import { InputSelect } from '@/components/ui/input-select';
import { tokenStandardOptions } from '@/config/dao';
export const step2Schema = z.object({
  governorAddress: z
    .string()
    .min(1, 'Governor address is required')
    .refine(
      (value) => isAddress(value),
      'Invalid Ethereum address format'
    ) as z.ZodType<`0x${string}`>,
  tokenAddress: z
    .string()
    .min(1, 'Token address is required')
    .refine(
      (value) => isAddress(value),
      'Invalid Ethereum address format'
    ) as z.ZodType<`0x${string}`>,
  tokenType: z.string().min(1, 'Token type is required'),
  timeLockAddress: z
    .string()
    .min(1, 'TimeLock address is required')
    .refine(
      (value) => isAddress(value),
      'Invalid Ethereum address format'
    ) as z.ZodType<`0x${string}`>
});

export type Step2FormValues = z.infer<typeof step2Schema>;

interface Step2FormProps {
  onSubmit: (values: Step2FormValues) => void;
  onBack: () => void;
  defaultValues?: Partial<Step2FormValues>;
}

export function Step2Form({ onSubmit, onBack, defaultValues }: Step2FormProps) {
  const form = useForm<Step2FormValues>({
    resolver: zodResolver(step2Schema),
    defaultValues: {
      governorAddress: '' as `0x${string}`,
      tokenAddress: '' as `0x${string}`,
      tokenType: 'ERC20',
      timeLockAddress: '' as `0x${string}`,
      ...defaultValues
    }
  });

  return (
    <>
      <Separator className="my-0" />
      <h3 className="text-base font-medium">Provide the contracts information for the DAO</h3>

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-6">
          <FormField
            control={form.control}
            name="governorAddress"
            render={({ field }) => (
              <FormItem>
                <div className="flex items-center gap-2">
                  <FormLabel>Governor Address</FormLabel>
                  <InfoIcon className="text-muted-foreground h-4 w-4" />
                </div>
                <FormControl>
                  <Input placeholder="please enter the governor address" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="tokenAddress"
            render={({ field }) => (
              <FormItem>
                <div className="flex items-center gap-2">
                  <FormLabel>Token Address</FormLabel>
                  <InfoIcon className="text-muted-foreground h-4 w-4" />
                </div>
                <FormControl>
                  <InputSelect
                    placeholder="please enter the token address"
                    selectPlaceholder="Token Type"
                    options={tokenStandardOptions}
                    selectValue={form.watch('tokenType')}
                    onSelectChange={(value) => form.setValue('tokenType', value)}
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="timeLockAddress"
            render={({ field }) => (
              <FormItem>
                <div className="flex items-center gap-2">
                  <FormLabel>TimeLock Address</FormLabel>
                  <InfoIcon className="text-muted-foreground h-4 w-4" />
                </div>
                <FormControl>
                  <Input placeholder="please enter the time lock address" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className="flex justify-between pt-4">
            <Button variant="outline" type="button" className="rounded-full px-8" onClick={onBack}>
              Back
            </Button>
            <Button type="submit" className="rounded-full px-8">
              Next
            </Button>
          </div>
        </form>
      </Form>
    </>
  );
}
