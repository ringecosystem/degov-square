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
    <div className="bg-card rounded-[14px]">
      <div className="bg-card rounded-2xl p-8">
        <h2 className="mb-8 text-xl font-medium">Advanced Settings</h2>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(handleSubmit)} className="flex flex-col gap-8">
            <div>
              <FormLabel className="text-foreground/70 mb-2 block">Change Admin</FormLabel>
              <FormField
                control={form.control}
                name="adminAddress"
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <Input className="border-border/20 h-12 rounded-lg" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div>
              <FormLabel className="text-foreground/70 mb-2 block">
                walletConnectProjectId
              </FormLabel>
              <FormField
                control={form.control}
                name="walletConnectProjectId"
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <Input
                        className="border-border/20 h-12 rounded-lg"
                        placeholder="Wallet Connect API Key"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className="mt-4 flex justify-center gap-4">
              <Button
                type="button"
                variant="outline"
                className="border-border/20 bg-card h-12 min-w-[140px] rounded-full border"
                onClick={() => form.reset(defaultValues)}
              >
                Cancel
              </Button>
              <LoadedButton
                type="submit"
                variant="default"
                className="h-12 min-w-[140px] rounded-full"
                isLoading={isLoading}
                disabled={!form.formState.isDirty || !form.formState.isValid}
              >
                Save Changes
              </LoadedButton>
            </div>
          </form>
        </Form>
      </div>
    </div>
  );
}
