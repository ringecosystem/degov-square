'use client';

import { createContext, useContext } from 'react';

import type { Step1FormValues } from '@/app/add/existing/_components/step1-form';
import type { Step2FormValues } from '@/app/add/existing/_components/step2-form';

export type Step = 1 | 2 | 3;

interface AddDaoContextValue {
  currentStep: Step;
  setCurrentStep: (step: Step) => void;
  step1Data: Step1FormValues | null;
  setStep1Data: (data: Step1FormValues) => void;
  step2Data: Step2FormValues | null;
  setStep2Data: (data: Step2FormValues) => void;
  handleStep1Submit: (values: Step1FormValues) => void;
  handleStep2Submit: (values: Step2FormValues) => void;
  handleBackToStep1: () => void;
  handleBackToStep2: () => void;
  handleReviewSubmit: () => void;
  formData: {
    step1: Step1FormValues | null;
    step2: Step2FormValues | null;
  };
}

export const AddDaoContext = createContext<AddDaoContextValue | undefined>(undefined);

export function useAddDao() {
  const context = useContext(AddDaoContext);
  if (!context) {
    throw new Error('useAddDao must be used within an AddDaoProvider');
  }
  return context;
}
