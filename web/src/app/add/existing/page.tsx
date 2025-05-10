'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { Step1Form, type Step1FormValues } from './_components/step1-form';
import { Step2Form, type Step2FormValues } from './_components/step2-form';
import { Review } from './_components/review';

// 分步骤表单的类型
type Step = 1 | 2 | 3;

export default function AddExisting() {
  const router = useRouter();
  const [currentStep, setCurrentStep] = useState<Step>(1);
  const [step1Data, setStep1Data] = useState<Step1FormValues | null>(null);
  const [step2Data, setStep2Data] = useState<Step2FormValues | null>(null);

  // 处理Step1表单提交
  function handleStep1Submit(values: Step1FormValues) {
    console.log('Step 1 Values:', values);
    setStep1Data(values);
    setCurrentStep(2);
  }

  // 处理Step2表单提交
  function handleStep2Submit(values: Step2FormValues) {
    console.log('Step 2 Values:', values);
    setStep2Data(values);
    setCurrentStep(3);
  }

  // 处理Review提交
  function handleReviewSubmit() {
    console.log('Complete Form Data:', {
      ...step1Data,
      ...step2Data
    });

    // 提交成功后跳转到成功页面
    router.push('/add/existing/success');
  }

  // 返回上一步
  function handleBackToStep1() {
    setCurrentStep(1);
  }

  // 返回上一步 - Step2
  function handleBackToStep2() {
    setCurrentStep(2);
  }

  return (
    <div className="container flex flex-col gap-[20px] py-6">
      <div className="bg-card mx-auto flex w-[800px] flex-col gap-[20px] rounded-[14px] p-[20px]">
        <header>
          <h2 className="text-[24px] font-bold">Add existing DAO</h2>
        </header>
        {currentStep === 1 && (
          <Step1Form onSubmit={handleStep1Submit} defaultValues={step1Data || undefined} />
        )}

        {currentStep === 2 && (
          <Step2Form
            onSubmit={handleStep2Submit}
            onBack={handleBackToStep1}
            defaultValues={step2Data || undefined}
          />
        )}

        {currentStep === 3 && step1Data && step2Data && (
          <Review
            step1Data={step1Data}
            step2Data={step2Data}
            onSubmit={handleReviewSubmit}
            onBack={handleBackToStep2}
          />
        )}
      </div>
    </div>
  );
}
