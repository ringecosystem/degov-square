'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';

import type { Step1FormValues } from '@/app/add/existing/_components/step1-form';
import type { Step2FormValues } from '@/app/add/existing/_components/step2-form';

import { AddDaoContext, type Step } from './context';

interface AddDaoProviderProps {
  children: React.ReactNode;
}

export function AddDaoProvider({ children }: AddDaoProviderProps) {
  const router = useRouter();
  const [currentStep, setCurrentStep] = useState<Step>(1);
  const [step1Data, setStep1Data] = useState<Step1FormValues | null>(null);
  const [step2Data, setStep2Data] = useState<Step2FormValues | null>(null);

  // 各步骤的加载状态
  const [isStep1Loading, setIsStep1Loading] = useState(false);
  const [isStep2Loading, setIsStep2Loading] = useState(false);
  const [isReviewLoading, setIsReviewLoading] = useState(false);

  function handleStep1Submit(values: Step1FormValues) {
    console.log('Step 1 Values:', values);
    setIsStep1Loading(true);

    // 模拟网络请求或数据处理延迟
    setTimeout(() => {
      setStep1Data(values);
      setCurrentStep(2);
      setIsStep1Loading(false);
    }, 500);
  }

  function handleStep2Submit(values: Step2FormValues) {
    console.log('Step 2 Values:', values);
    setIsStep2Loading(true);

    // 模拟网络请求或数据处理延迟
    setTimeout(() => {
      setStep2Data(values);
      setCurrentStep(3);
      setIsStep2Loading(false);
    }, 500);
  }

  function handleReviewSubmit() {
    console.log('Complete Form Data:', {
      ...step1Data,
      ...step2Data
    });

    setIsReviewLoading(true);

    // 模拟提交延迟
    setTimeout(() => {
      router.push('/add/existing/success');
      setIsReviewLoading(false);
    }, 1000);
  }

  function handleBackToStep1() {
    setCurrentStep(1);
  }

  function handleBackToStep2() {
    setCurrentStep(2);
  }

  const value = {
    currentStep,
    setCurrentStep,
    step1Data,
    setStep1Data,
    step2Data,
    setStep2Data,
    handleStep1Submit,
    handleStep2Submit,
    handleBackToStep1,
    handleBackToStep2,
    handleReviewSubmit,
    isStep1Loading,
    isStep2Loading,
    isReviewLoading,
    setIsStep1Loading,
    setIsStep2Loading,
    setIsReviewLoading,
    formData: {
      step1: step1Data,
      step2: step2Data
    }
  };

  return <AddDaoContext.Provider value={value}>{children}</AddDaoContext.Provider>;
}
