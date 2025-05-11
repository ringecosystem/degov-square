'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

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
import { useIsMobileAndSubSection } from '@/app/notification/_hooks/isMobileAndSubSection';
import Link from 'next/link';
import Image from 'next/image';

// Schema for validation
const emailSchema = z.object({
  email: z.string().email({ message: 'Please enter a valid email address' })
});

type EmailFormValues = z.infer<typeof emailSchema>;

export default function SubscriptionPage() {
  const [isLoading, setIsLoading] = useState(false);
  const [isVerified, setIsVerified] = useState(false);
  const isMobileAndSubSection = useIsMobileAndSubSection();

  const form = useForm<EmailFormValues>({
    resolver: zodResolver(emailSchema),
    defaultValues: {
      email: ''
    },
    mode: 'onChange'
  });

  const handleVerify = (values: EmailFormValues) => {
    setIsLoading(true);

    // Simulate API call
    setTimeout(() => {
      console.log('Verifying email:', values.email);
      setIsLoading(false);
      setIsVerified(true);
    }, 1000);
  };

  return (
    <div className="md:bg-card h-[calc(100vh-300px)] space-y-[15px] md:space-y-[20px] md:rounded-[14px] md:p-[20px]">
      {isMobileAndSubSection && (
        <Link href={`/notification`} className="flex items-center gap-[5px] md:gap-[10px]">
          <Image
            src="/back.svg"
            alt="back"
            width={32}
            height={32}
            className="size-[32px] flex-shrink-0"
          />
          <h1 className="text-[18px] font-semibold">Subscription</h1>
        </Link>
      )}
      <Form {...form}>
        <form onSubmit={form.handleSubmit(handleVerify)}>
          <FormField
            control={form.control}
            name="email"
            render={({ field }) => (
              <FormItem className="flex-1 gap-[5px]">
                <FormLabel className="text-[14px]">Email</FormLabel>
                <FormControl>
                  <div className="flex items-center gap-[10px]">
                    <Input
                      className="border-border/20 h-[39px] rounded-lg"
                      placeholder="Email@example.com"
                      {...field}
                    />
                    <LoadedButton
                      type="submit"
                      variant="default"
                      className="fixed bottom-[20px] h-[37px] w-[calc(100%-40px)] rounded-full p-[10px] md:static md:w-[155px]"
                      isLoading={isLoading}
                      disabled={!form.formState.isValid || isLoading}
                    >
                      Verify
                    </LoadedButton>
                  </div>
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </form>
      </Form>

      <p className="text-[12px] md:text-[14px]">
        Please set up your email to receive the notification from the DAOs you are interested in or
        the proposals you are interested in. This will help you to keep track of the latest updates
        and news from the DAOs and proposals you care about.
      </p>
    </div>
  );
}
