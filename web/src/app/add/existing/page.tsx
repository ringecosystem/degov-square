'use client';

import { useRouter } from 'next/navigation';
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

import { InputAddon } from '@/components/ui/input-addon';
import { InputSelect } from '@/components/ui/input-select';
import { getChains } from '@/utils/chains';
import { isAddress } from 'viem';

// Define form schema with validation
const formSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  chainId: z.string().min(1, 'Chain is required'),
  description: z.string().min(1, 'Description is required'),
  owner: z
    .string()
    .min(1, 'Owner address is required')
    .refine(
      (value) => isAddress(value),
      'Invalid Ethereum address format'
    ) as z.ZodType<`0x${string}`>,
  email: z.string().email('Invalid email format').optional().or(z.literal('')),
  telegram: z.string().optional().or(z.literal('')),
  daoUrl: z.string().min(1, 'DAO URL is required'),
  domain: z.string()
});

const domainSuffix = '.degov.ai';

type FormValues = z.infer<typeof formSchema>;

const chainOptions = getChains().map((chain) => ({
  label: chain.name,
  value: chain.id.toString()
}));

export default function AddExisting() {
  const router = useRouter();

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: '',
      chainId: '1',
      description: '',
      owner: '' as `0x${string}`,
      email: '',
      telegram: '',
      daoUrl: '',
      domain: domainSuffix
    }
  });

  function onSubmit(values: FormValues) {
    // Form submission logic would go here
    console.log(values);
    // router.push("/success-page");
  }

  return (
    <div className="container flex flex-col gap-[20px] py-6">
      <div className="bg-card mx-auto flex w-[800px] flex-col gap-[20px] rounded-[14px] p-[20px]">
        <header>
          <h2 className="text-[24px] font-bold">Add existing DAO</h2>
        </header>
        <Separator className="my-0" />
        <h3 className="text-base font-medium">
          Provide the most basic information for displaying the DAO
        </h3>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-6">
            <div className="flex gap-4">
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem className="flex-1">
                    <FormLabel>Name</FormLabel>
                    <FormControl>
                      <InputSelect
                        placeholder="Enter your DAO name"
                        selectPlaceholder="Select chain"
                        options={chainOptions}
                        selectValue={form.watch('chainId')}
                        onSelectChange={(value) => form.setValue('chainId', value)}
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Description</FormLabel>
                  <FormControl>
                    <textarea
                      placeholder="Write a description for your DAO. This will be displayed on the DAO dashboard"
                      className="file:text-foreground placeholder:text-muted-foreground selection:bg-primary selection:text-primary-foreground dark:bg-input/30 border-input flex min-h-[120px] w-full min-w-0 resize-none rounded-md border bg-transparent px-3 py-2 text-base shadow-xs transition-[color,box-shadow] outline-none"
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="owner"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>DAO Owner</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder="Please enter the DAO owner address" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Email</FormLabel>
                  <FormControl>
                    <Input placeholder="We can connect you for details" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="telegram"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Telegram</FormLabel>
                  <FormControl>
                    <Input placeholder="We can connect you for details" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="daoUrl"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>DAO Url</FormLabel>
                  <FormControl>
                    <InputAddon suffix={domainSuffix} placeholder="DAO-name" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className="flex justify-between pt-4">
              <Button
                variant="outline"
                type="button"
                className="rounded-full px-8"
                onClick={() => router.push('')}
              >
                Back
              </Button>
              <Button type="submit" className="rounded-full px-8">
                Next
              </Button>
            </div>
          </form>
        </Form>
      </div>
    </div>
  );
}
