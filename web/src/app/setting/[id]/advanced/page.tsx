'use client';

import { useState } from 'react';
import { z } from 'zod';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { isAddress } from 'viem';

import { Button } from '@/components/ui/button';
import { LoadedButton } from '@/components/ui/loaded-button';
import { Input } from '@/components/ui/input';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@/components/ui/form';

// Schema for validation
const advancedSettingsSchema = z.object({
  adminAddress: z
    .string()
    .min(1, { message: 'Admin address is required' })
    .refine(
      (value) => isAddress(value),
      'Invalid admin address format'
    ) as z.ZodType<`0x${string}`>,
  walletConnectProjectId: z.string().min(1, { message: 'WalletConnect project ID is required' })
});

type AdvancedSettingsFormValues = z.infer<typeof advancedSettingsSchema>;

export default function AdvancedSettingPage() {
  const [isLoading, setIsLoading] = useState(false);

  // Mock initial data - would come from API in a real app
  const defaultValues = {
    adminAddress: '0x3E8436e87Abb49efe1A958EE73fbB7A12B419aAB' as `0x${string}`,
    walletConnectProjectId: ''
  };

  const form = useForm<AdvancedSettingsFormValues>({
    resolver: zodResolver(advancedSettingsSchema),
    defaultValues,
    mode: 'onChange'
  });

  const handleSubmit = (values: AdvancedSettingsFormValues) => {
    setIsLoading(true);

    // Simulate API call
    setTimeout(() => {
      console.log('Saved settings:', values);
      setIsLoading(false);
    }, 1000);
  };

  return (
    <div className="bg-card h-[calc(100vh-300px)] rounded-[14px] p-[20px]">
      <Form {...form}>
        <form onSubmit={form.handleSubmit(handleSubmit)} className="flex flex-col gap-[20px]">
          <div className="flex flex-col gap-[5px]">
            <FormLabel className="text-foreground text-[14px]">Change Admin</FormLabel>
            <FormField
              control={form.control}
              name="adminAddress"
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <Input className="border-border h-[39px]" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <div className="flex flex-col gap-[5px]">
            <FormLabel className="text-foreground text-[14px]">walletConnectProjectId</FormLabel>
            <FormField
              control={form.control}
              name="walletConnectProjectId"
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <Input
                      className="border-border h-[39px]"
                      placeholder="Wallet Connect API Key"
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <div className="flex justify-center gap-[20px]">
            <Button
              type="button"
              variant="outline"
              className="bg-card h-[37px] w-[155px] rounded-full border"
              onClick={() => form.reset(defaultValues)}
            >
              Cancel
            </Button>
            <LoadedButton
              type="submit"
              variant="default"
              className="h-[37px] w-[155px] rounded-full"
              isLoading={isLoading}
              disabled={!form.formState.isDirty || !form.formState.isValid}
            >
              Save Changes
            </LoadedButton>
          </div>
        </form>
      </Form>
    </div>
  );
}
