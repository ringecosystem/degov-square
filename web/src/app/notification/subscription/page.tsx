'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import Image from 'next/image';
import Link from 'next/link';
import { useState, useReducer, useCallback } from 'react';
import { useForm } from 'react-hook-form';
import { toast } from 'react-toastify';
import { z } from 'zod';

import { useIsMobileAndSubSection } from '@/app/notification/_hooks/isMobileAndSubSection';
import { Countdown } from '@/components/countdown';
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
import {
  useBindNotificationChannel,
  useResendOTP,
  useVerifyNotificationChannel
} from '@/hooks/useNotification';

// Schema for validation
const subscriptionSchema = z.object({
  email: z.string().email({ message: 'Please enter a valid email address' }),
  verificationCode: z.string().min(1, { message: 'Please enter verification code' })
});

type SubscriptionFormValues = z.infer<typeof subscriptionSchema>;

interface FormState {
  email: string;
  verificationCode: string;
  channelId: string | null;
}

type FormAction =
  | { type: 'SET_EMAIL'; payload: string }
  | { type: 'SET_VERIFICATION_CODE'; payload: string }
  | { type: 'SET_CHANNEL_ID'; payload: string }
  | { type: 'RESET_VERIFICATION' };

interface CountdownState {
  active: boolean;
  duration: number;
  key: number;
}

const formReducer = (state: FormState, action: FormAction): FormState => {
  switch (action.type) {
    case 'SET_EMAIL':
      return {
        ...state,
        email: action.payload,
        ...(action.payload !== state.email && {
          channelId: null,
          verificationCode: ''
        })
      };
    case 'SET_VERIFICATION_CODE':
      return { ...state, verificationCode: action.payload };
    case 'SET_CHANNEL_ID':
      return { ...state, channelId: action.payload };
    case 'RESET_VERIFICATION':
      return {
        ...state,
        channelId: null,
        verificationCode: ''
      };
    default:
      return state;
  }
};

export default function SubscriptionPage() {
  const isMobileAndSubSection = useIsMobileAndSubSection();

  // API hooks
  const bindEmailMutation = useBindNotificationChannel();
  const resendOTPMutation = useResendOTP();
  const verifyEmailMutation = useVerifyNotificationChannel();

  // State management
  const [formState, dispatch] = useReducer(formReducer, {
    email: '',
    verificationCode: '',
    channelId: null
  });

  const [countdown, setCountdown] = useState<CountdownState>({
    active: false,
    duration: 60,
    key: 0
  });

  const form = useForm<SubscriptionFormValues>({
    resolver: zodResolver(subscriptionSchema),
    defaultValues: {
      email: '',
      verificationCode: ''
    },
    mode: 'onChange'
  });

  // Email validation
  const emailSchema = z.string().email();
  const isEmailValid = emailSchema.safeParse(formState.email).success;

  // Loading states
  const sendingLoading = bindEmailMutation.isPending || resendOTPMutation.isPending;
  const verifyLoading = verifyEmailMutation.isPending;

  const handleSendCode = useCallback(async () => {
    if (!formState.email || !isEmailValid || sendingLoading) return;

    const mutation = formState.channelId ? resendOTPMutation : bindEmailMutation;
    const mutationParams = { type: 'EMAIL' as const, value: formState.email };

    mutation.mutate(mutationParams, {
      onSuccess: (data) => {
        if (data.code === 0) {
          const rate = data.rateLimit || 60;
          if (!formState.channelId) {
            dispatch({ type: 'SET_CHANNEL_ID', payload: data.id });
          }
          // Start countdown
          setCountdown({
            active: true,
            duration: rate,
            key: Math.random()
          });
          toast.success('Verification code sent successfully!');
        } else {
          toast.error(data.message || 'Failed to send verification code');
        }
      },
      onError: (error: any) => {
        const graphqlError = error.response?.errors?.[0]?.message;
        const errorMessage = graphqlError || error.message || 'Failed to send verification code';
        toast.error(errorMessage);
      }
    });
  }, [
    formState.email,
    formState.channelId,
    isEmailValid,
    sendingLoading,
    resendOTPMutation,
    bindEmailMutation
  ]);

  const handleVerify = useCallback(
    async (values: SubscriptionFormValues) => {
      if (!formState.verificationCode || !formState.channelId || verifyLoading) return;

      verifyEmailMutation.mutate(
        { id: formState.channelId, otpCode: formState.verificationCode },
        {
          onSuccess: (data) => {
            if (data.code === 0) {
              toast.success('Email verified successfully!');
              // Reset form state
              dispatch({ type: 'RESET_VERIFICATION' });
              form.reset();
              setCountdown({ active: false, duration: 60, key: 0 });
            } else {
              toast.error(data.message || 'Verification failed');
            }
          },
          onError: (error: Error) => {
            toast.error(error.message || 'Verification failed');
          }
        }
      );
    },
    [formState.verificationCode, formState.channelId, verifyEmailMutation, verifyLoading, form]
  );

  const handleCountdownEnd = useCallback(() => {
    setCountdown((prev) => ({
      ...prev,
      active: false
    }));
  }, []);

  const handleCountdownTick = useCallback((remaining: number) => {
    setCountdown((prev) => ({
      ...prev,
      duration: remaining
    }));
  }, []);

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
        <form onSubmit={form.handleSubmit(handleVerify)} className="space-y-[20px]">
          <div className="space-y-[20px] rounded-lg">
            {/* Email Field */}
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem className="space-y-[8px]">
                  <FormLabel className="text-[14px] text-white">Your Email</FormLabel>
                  <FormControl>
                    <div className="flex items-center gap-[10px]">
                      <Input
                        className="h-[39px] max-w-[335px] flex-1 rounded-[100px] border-gray-600 bg-gray-700 text-white placeholder:text-gray-400"
                        placeholder="yourname@example.com"
                        value={formState.email}
                        onChange={(e) => {
                          dispatch({ type: 'SET_EMAIL', payload: e.target.value });
                          field.onChange(e.target.value);
                        }}
                      />
                      <LoadedButton
                        type="button"
                        onClick={handleSendCode}
                        variant="default"
                        className="bg-foreground min-w-[120px] rounded-[100px] p-[10px] text-black hover:opacity-80"
                        isLoading={sendingLoading}
                        disabled={!formState.email || !isEmailValid || countdown.active}
                      >
                        {countdown.active ? (
                          <Countdown
                            key={countdown.key}
                            start={countdown.duration}
                            autoStart
                            onEnd={handleCountdownEnd}
                            onTick={handleCountdownTick}
                          />
                        ) : (
                          'Send Code'
                        )}
                      </LoadedButton>
                    </div>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Verification Code Field */}
            <FormField
              control={form.control}
              name="verificationCode"
              render={({ field }) => (
                <FormItem className="space-y-[8px]">
                  <FormLabel className="text-[14px] text-white">Verification Code</FormLabel>
                  <FormControl>
                    <div className="flex items-center gap-[10px]">
                      <Input
                        className="h-[39px] max-w-[335px] flex-1 rounded-[100px] border-gray-600 bg-gray-700 text-white placeholder:text-gray-400"
                        placeholder="e.g., 123456"
                        disabled={!formState.channelId}
                        value={formState.verificationCode}
                        onChange={(e) => {
                          dispatch({ type: 'SET_VERIFICATION_CODE', payload: e.target.value });
                          field.onChange(e.target.value);
                        }}
                      />
                      <LoadedButton
                        type="submit"
                        variant="default"
                        className="bg-foreground text-background min-w-[120px] rounded-[100px] p-[10px] hover:opacity-80"
                        isLoading={verifyLoading}
                        disabled={
                          !formState.channelId || !formState.verificationCode || verifyLoading
                        }
                      >
                        Verify
                      </LoadedButton>
                    </div>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </form>
      </Form>
    </div>
  );
}
