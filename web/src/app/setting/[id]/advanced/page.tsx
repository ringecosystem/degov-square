'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import Image from 'next/image';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { isAddress } from 'viem';
import { z } from 'zod';

import { useIsMobileAndSubSection } from '@/app/setting/_hooks/isMobileAndSubSection';
import { Button } from '@/components/ui/button';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { LoadedButton } from '@/components/ui/loaded-button';

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
  const { id } = useParams();
  const isMobileAndSubSection = useIsMobileAndSubSection();

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
    <div className="md:bg-card flex flex-col gap-[15px] md:h-[calc(100vh-300px)] md:gap-0 md:rounded-[14px] md:p-[20px]">
      {isMobileAndSubSection && (
        <Link href={`/setting/${id}`} className="flex items-center gap-[5px] md:gap-[10px]">
          <Image
            src="/back.svg"
            alt="back"
            width={32}
            height={32}
            className="size-[32px] flex-shrink-0"
          />
          <h1 className="text-[18px] font-semibold">Advanced</h1>
        </Link>
      )}

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

          <div className="bg-background fixed right-0 bottom-0 left-0 flex justify-center gap-[20px] p-[20px] md:static">
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
