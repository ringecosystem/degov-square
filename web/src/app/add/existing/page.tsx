'use client';

import { Separator } from '@/components/ui/separator';
import { AddDaoProvider, useAddDao } from '@/contexts/add-dao';

import { Review } from './_components/review';
import { Step1Form } from './_components/step1-form';
import { Step2Form } from './_components/step2-form';

function AddExistingContent() {
  const {
    currentStep,
    step1Data,
    step2Data,
    handleStep1Submit,
    handleStep2Submit,
    handleBackToStep1,
    handleBackToStep2,
    handleReviewSubmit
  } = useAddDao();

  return (
    <div className="md:bg-card container mx-auto flex flex-col gap-[15px] md:w-[800px] md:gap-[20px] md:rounded-[14px] md:p-[20px]">
      <header>
        <h2 className="text-[26px] font-semibold">Add existing DAO</h2>
      </header>
      <Separator className="my-0" />

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
  );
}

export default function AddExisting() {
  return (
    <AddDaoProvider>
      <AddExistingContent />
    </AddDaoProvider>
  );
}
