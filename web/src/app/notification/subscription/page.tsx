'use client';

import { useState } from 'react';
import { z } from 'zod';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';

import { Button } from '@/components/ui/button';
import { LoadedButton } from '@/components/ui/loaded-button';
import { Input } from '@/components/ui/input';
import { Form, FormControl, FormField, FormItem, FormMessage } from '@/components/ui/form';

// Schema for validation
const emailSchema = z.object({
  email: z.string().email({ message: 'Please enter a valid email address' })
});

type EmailFormValues = z.infer<typeof emailSchema>;

export default function SubscriptionPage() {
  const [isLoading, setIsLoading] = useState(false);
  const [isVerified, setIsVerified] = useState(false);

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
    <div className="bg-card rounded-[14px]">
      <div className="p-8">
        <h2 className="mb-6 text-lg font-medium">Email</h2>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(handleVerify)} className="flex flex-col gap-6">
            <div className="flex items-center gap-4">
              <FormField
                control={form.control}
                name="email"
                render={({ field }) => (
                  <FormItem className="flex-1">
                    <FormControl>
                      <Input
                        className="border-border/20 h-12 rounded-lg"
                        placeholder="Email@example.com"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <LoadedButton
                type="submit"
                variant="default"
                className="h-12 rounded-full px-8"
                isLoading={isLoading}
                disabled={!form.formState.isValid || isLoading}
              >
                Verify
              </LoadedButton>
            </div>
          </form>
        </Form>

        <div className="text-muted-foreground mt-6">
          <p>
            Please set up your email to receive the notification from the DAOs you are interested in
            or the proposals you are interested in.
          </p>
          <p className="mt-2">
            This will help you to keep track of the latest updates and news from the DAOs and
            proposals you care about.
          </p>
        </div>
      </div>
    </div>
  );
}
