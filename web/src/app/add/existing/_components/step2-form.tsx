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
import { CircleHelp } from 'lucide-react';
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
    <div className="flex flex-col gap-[20px]">
      <h3 className="text-[18px] font-semibold">Provide the contracts information for the DAO</h3>

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-[20px]">
          <FormField
            control={form.control}
            name="governorAddress"
            render={({ field }) => (
              <FormItem>
                <div className="flex items-center gap-2">
                  <FormLabel>Governor Address</FormLabel>
                  <CircleHelp className="text-muted-foreground h-4 w-4" />
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
                  <CircleHelp className="text-muted-foreground h-4 w-4" />
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
                  <CircleHelp className="text-muted-foreground h-4 w-4" />
                </div>
                <FormControl>
                  <Input placeholder="please enter the time lock address" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <Separator className="my-0" />

          <div className="flex justify-between">
            <Button
              variant="outline"
              type="button"
              className="w-[140px] rounded-full p-[10px]"
              onClick={onBack}
            >
              Back
            </Button>
            <Button type="submit" className="w-[140px] rounded-full p-[10px]">
              Next
            </Button>
          </div>
        </form>
      </Form>
    </div>
  );
}
