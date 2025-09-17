'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import Image from 'next/image';
import Link from 'next/link';
import { useState, useReducer, useCallback, Suspense } from 'react';
import { useForm } from 'react-hook-form';
import { toast } from 'react-toastify';
import { z } from 'zod';

import { useIsMobileAndSubSection } from '@/app/notification/_hooks/isMobileAndSubSection';
import { Countdown } from '@/components/countdown';
import { ErrorIcon } from '@/components/icons/error-icon';
import { Form, FormControl, FormField, FormItem, FormLabel } from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { LoadedButton } from '@/components/ui/loaded-button';
import { Skeleton } from '@/components/ui/skeleton';
import {
  useResendOTP,
  useVerifyNotificationChannel,
  useNotificationChannels
} from '@/hooks/useNotification';
import { extractErrorMessage } from '@/utils/graphql-error-handler';

// Schema for validation
const subscriptionSchema = z.object({
  email: z.string().email({ message: 'Please enter a valid email address' }),
  verificationCode: z.string().min(1, { message: 'Please enter verification code' })
});

type SubscriptionFormValues = z.infer<typeof subscriptionSchema>;

interface FormState {
  email: string;
  verificationCode: string;
}

type FormAction =
  | { type: 'SET_EMAIL'; payload: string }
  | { type: 'SET_VERIFICATION_CODE'; payload: string }
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
          verificationCode: ''
        })
      };
    case 'SET_VERIFICATION_CODE':
      return { ...state, verificationCode: action.payload };
    case 'RESET_VERIFICATION':
      return {
        ...state,
        verificationCode: ''
      };
    default:
      return state;
  }
};

function SubscriptionPageContent() {
  const isMobileAndSubSection = useIsMobileAndSubSection();

  // API hooks
  const resendOTPMutation = useResendOTP();
  const verifyEmailMutation = useVerifyNotificationChannel();
  const { data: notificationChannels, isLoading: channelsLoading } = useNotificationChannels();

  // State management
  const [formState, dispatch] = useReducer(formReducer, {
    email: '',
    verificationCode: ''
  });

  const [countdown, setCountdown] = useState<CountdownState>({
    active: false,
    duration: 60,
    key: 0
  });

  const [isChangingEmail, setIsChangingEmail] = useState(false);
  const [emailError, setEmailError] = useState<string>('');
  const [verificationError, setVerificationError] = useState<string>('');

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

  const hasVerifiedEmail = notificationChannels?.isEmailBound && !isChangingEmail;

  // Loading states
  const sendingLoading = resendOTPMutation.isPending;
  const verifyLoading = verifyEmailMutation.isPending;

  const handleSendCode = useCallback(async () => {
    setEmailError('');

    if (!formState.email) {
      setEmailError('Please enter your email address');
      return;
    }

    if (!isEmailValid) {
      setEmailError('Please enter a valid email address');
      return;
    }

    if (sendingLoading) return;

    // Check if the new email is the same as the current verified email when changing email
    if (
      isChangingEmail &&
      notificationChannels?.emailAddress &&
      formState.email === notificationChannels.emailAddress
    ) {
      setEmailError("Your new email shouldn't be the same as the old one.");
      return;
    }

    resendOTPMutation.mutate(
      { type: 'EMAIL' as const, value: formState.email },
      {
        onSuccess: (data) => {
          if (data.code === 0) {
            const rate = data.rateLimit || 60;
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
          const errorMessage =
            extractErrorMessage(error) || error.message || 'Failed to send verification code';
          toast.error(errorMessage);
        }
      }
    );
  }, [
    formState.email,
    isEmailValid,
    sendingLoading,
    resendOTPMutation,
    isChangingEmail,
    notificationChannels?.emailAddress
  ]);

  const handleVerify = useCallback(async () => {
    setVerificationError('');

    if (!formState.verificationCode.trim()) {
      setVerificationError('Please enter verification code');
      return;
    }

    if (verifyLoading) return;

    verifyEmailMutation.mutate(
      { type: 'EMAIL' as const, value: formState.email, otpCode: formState.verificationCode },
      {
        onSuccess: (data) => {
          if (data.code === 0) {
            toast.success('Email verified successfully!');
            // Reset form state
            dispatch({ type: 'RESET_VERIFICATION' });
            form.reset();
            setCountdown({ active: false, duration: 60, key: 0 });
            setIsChangingEmail(false);
            setVerificationError('');
          } else {
            setVerificationError('Invalid verification code. Please try again.');
          }
        },
        onError: (error: any) => {
          const errorMessage = extractErrorMessage(error) || error.message || 'Verification failed';
          toast.error(errorMessage);
        }
      }
    );
  }, [formState.verificationCode, formState.email, verifyEmailMutation, verifyLoading, form]);

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

  const handleChangeEmail = useCallback(() => {
    setIsChangingEmail(true);
    // Pre-fill with current verified email
    const currentEmail = notificationChannels?.emailAddress || '';
    dispatch({ type: 'SET_EMAIL', payload: currentEmail });
    dispatch({ type: 'RESET_VERIFICATION' });
    form.setValue('email', currentEmail);
    form.setValue('verificationCode', '');
    setCountdown({ active: false, duration: 60, key: 0 });
    setEmailError('');
    setVerificationError('');
  }, [form, notificationChannels?.emailAddress]);

  return (
    <>
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
      {channelsLoading ? (
        // Show loading state
        <div className="space-y-[20px] rounded-lg">
          <div className="space-y-[8px]">
            <label className="text-[14px] text-white">Your Email</label>
            <div className="flex items-center gap-[10px]">
              <Skeleton className="h-[39px] max-w-[335px] flex-1 rounded-[100px]" />
              <Skeleton className="h-[39px] min-w-[120px] rounded-[100px]" />
            </div>
          </div>
        </div>
      ) : hasVerifiedEmail ? (
        // Show verified email with Change button
        <div className="space-y-[20px] rounded-lg">
          <div className="flex flex-col gap-[5px]">
            <label className="text-[14px] text-white">Your Email</label>
            <div className="flex items-center gap-[10px]">
              <Input
                className="h-[39px] max-w-[335px] flex-1 rounded-[100px] border-gray-600 bg-gray-700 text-white placeholder:text-gray-400"
                value={notificationChannels?.emailAddress || ''}
                readOnly
              />
              <LoadedButton
                type="button"
                onClick={handleChangeEmail}
                variant="default"
                className="bg-foreground min-w-[120px] rounded-[100px] p-[10px] text-black hover:opacity-80"
              >
                Change
              </LoadedButton>
            </div>
          </div>
        </div>
      ) : (
        // Show send/verify flow
        <Form {...form}>
          <form onSubmit={form.handleSubmit(handleVerify)} className="space-y-[20px]">
            <div className="space-y-[20px] rounded-lg">
              {/* Email Field */}
              <FormField
                control={form.control}
                name="email"
                render={({ field }) => (
                  <FormItem className="space-y-[5px]">
                    <FormLabel className="mb-0 text-[14px] text-white">Your Email</FormLabel>
                    <FormControl>
                      <div className="space-y-[5px]">
                        <div className="flex items-center gap-[10px]">
                          <Input
                            className={`h-[39px] max-w-[335px] flex-1 rounded-[100px] border-gray-600 bg-gray-700 text-white placeholder:text-gray-400 ${
                              emailError ? 'border-red-500' : ''
                            }`}
                            placeholder="yourname@example.com"
                            value={formState.email}
                            onChange={(e) => {
                              dispatch({ type: 'SET_EMAIL', payload: e.target.value });
                              field.onChange(e.target.value);
                              setEmailError('');
                            }}
                          />
                          <LoadedButton
                            type="button"
                            onClick={handleSendCode}
                            variant="default"
                            className="bg-foreground min-w-[120px] rounded-[100px] p-[10px] text-black hover:opacity-80"
                            isLoading={sendingLoading}
                            disabled={sendingLoading || countdown.active}
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
                        {emailError && (
                          <div className="flex items-center gap-[5px] text-[12px]">
                            <ErrorIcon className="h-4 w-4 flex-shrink-0 text-[#FF3C3F]" />
                            <span>{emailError}</span>
                          </div>
                        )}
                      </div>
                    </FormControl>
                  </FormItem>
                )}
              />

              {/* Verification Code Field */}
              <FormField
                control={form.control}
                name="verificationCode"
                render={({ field }) => (
                  <FormItem className="space-y-[5px]">
                    <FormLabel className="mb-0 text-[14px] text-white">Verification Code</FormLabel>
                    <FormControl>
                      <div className="space-y-[5px]">
                        <div className="flex items-center gap-[10px]">
                          <Input
                            className={`h-[39px] max-w-[335px] flex-1 rounded-[100px] border-gray-600 bg-gray-700 text-white placeholder:text-gray-400 ${
                              verificationError ? 'border-red-500' : ''
                            }`}
                            placeholder="e.g., 123456"
                            disabled={!formState.email}
                            value={formState.verificationCode}
                            onChange={(e) => {
                              dispatch({ type: 'SET_VERIFICATION_CODE', payload: e.target.value });
                              field.onChange(e.target.value);
                              setVerificationError('');
                            }}
                          />
                          <LoadedButton
                            type="submit"
                            variant="default"
                            className="bg-foreground text-background min-w-[120px] rounded-[100px] p-[10px] hover:opacity-80"
                            isLoading={verifyLoading}
                            disabled={verifyLoading}
                          >
                            Verify
                          </LoadedButton>
                        </div>
                        {verificationError && (
                          <div className="flex items-center gap-[5px] text-[12px]">
                            <ErrorIcon className="h-4 w-4 flex-shrink-0 text-[#FF3C3F]" />
                            <span>{verificationError}</span>
                          </div>
                        )}
                      </div>
                    </FormControl>
                  </FormItem>
                )}
              />
            </div>
          </form>
        </Form>
      )}
    </>
  );
}

export default function SubscriptionPage() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <SubscriptionPageContent />
    </Suspense>
  );
}
