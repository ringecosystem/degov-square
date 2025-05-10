'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { Separator } from '@/components/ui/separator';
import { Step1Form, type Step1FormValues } from './_components/step1-form';
import { Step2Form, type Step2FormValues } from './_components/step2-form';

// 分步骤表单的类型
type Step = 1 | 2;

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
    // 这里可以处理整个表单的提交
    console.log('Complete Form Data:', {
      ...step1Data,
      ...values
    });

    // 提交成功后跳转到成功页面
    router.push('/add/existing/success');
  }

  // 返回上一步
  function handleBackToStep1() {
    setCurrentStep(1);
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
      </div>
    </div>
  );
}
