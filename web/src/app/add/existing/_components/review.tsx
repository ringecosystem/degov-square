'use client';

import Image from 'next/image';
import { useCallback } from 'react';

import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { useConfirm } from '@/contexts/confirm-context';
import { useAddDaoReview } from '@/hooks/useAddDaoReview';
import { getChains } from '@/utils/chains';

import type { Step1FormValues } from './step1-form';
import type { Step2FormValues } from './step2-form';
interface ReviewProps {
  step1Data: Step1FormValues;
  step2Data: Step2FormValues;
  onSubmit: () => void;
  onBack: () => void;
}

export function Review({ step1Data, step2Data, onSubmit, onBack }: ReviewProps) {
  const { data, isLoading } = useAddDaoReview(step1Data, step2Data, true);
  const { confirm } = useConfirm();
  const hasError =
    !isLoading &&
    (!data.governanceParams ||
      !data.tokenMetadata ||
      Object.values(data.governanceParams).some((v) => v === undefined || v === null));

  const network =
    getChains().find((chain) => chain.id.toString() === step1Data.chainId)?.name ||
    step1Data.chainId;

  const basicInfo = {
    Name: step1Data.name,
    Network: network,
    'DAO Url': `https://${step1Data.daoUrl}${step1Data.domain}`
  };

  const governorInfo = {
    'Governor Address': step2Data.governorAddress,
    'Proposal threshold': isLoading
      ? 'Loading...'
      : data.governanceParams?.proposalThreshold?.toString() || 'N/A',
    Quorum: isLoading ? 'Loading...' : data.governanceParams?.quorum?.toString() || 'N/A',
    'Proposal delay': isLoading
      ? 'Loading...'
      : `${data.governanceParams?.votingDelay?.toString() || 'N/A'} day`,
    'Voting period': isLoading
      ? 'Loading...'
      : `${data.governanceParams?.votingPeriod?.toString() || 'N/A'} days`
  };

  const tokenInfo = {
    'Token name': isLoading ? 'Loading...' : data.tokenMetadata?.name || 'N/A',
    'Token Address': step2Data.tokenAddress,
    'Token type': step2Data.tokenType,
    'token symbol': isLoading ? 'Loading...' : data.tokenMetadata?.symbol || 'N/A',
    'token decimal': isLoading ? 'Loading...' : data.tokenMetadata?.decimals?.toString() || 'N/A'
  };

  const timeLockInfo = {
    'TimeLock Address': step2Data.timeLockAddress,
    'TimeLock delay': isLoading
      ? 'Loading...'
      : `${data.governanceParams?.timeLockDelay?.toString() || 'N/A'} day`
  };

  const handleSubmit = useCallback(() => {
    confirm({
      title: 'Congratulations !',
      description:
        "We have received your DAO's information, will review it and get back to you soon.",
      confirmText: 'Ok',
      variant: 'default',
      onConfirm: onSubmit
    });
  }, [onSubmit, confirm]);

  return (
    <>
      <h3 className="text-[18px] font-semibold">
        Review all the information of the DAO before proceeding to build the DAO.
      </h3>

      <div className="mt-4 flex flex-col gap-6">
        {hasError ? (
          <div className="bg-background flex min-h-[388px] flex-col items-center justify-center gap-[20px] rounded-[14px]">
            <Image src="/alert.svg" alt="alert" width={60} height={60} />
            <div className="flex flex-col text-center text-[14px] font-normal">
              <span>Something unexpected happened while validating your contract information.</span>
              <span>Please check your contract address and try again.</span>
            </div>
          </div>
        ) : (
          <>
            <div className="bg-opacity-5 dark:bg-opacity-5 bg-background rounded-lg p-4">
              <h4 className="mb-4 text-lg font-bold">Basic</h4>
              <div className="space-y-2">
                {Object.entries(basicInfo).map(([key, value]) => (
                  <div key={key} className="flex">
                    <span className="w-1/3 text-gray-500">{key}</span>
                    <span className="w-2/3">{value}</span>
                  </div>
                ))}
              </div>
            </div>

            <div className="bg-opacity-5 dark:bg-opacity-5 bg-background rounded-lg p-4">
              <h4 className="mb-4 text-lg font-bold">Governor</h4>
              <div className="space-y-2">
                {Object.entries(governorInfo).map(([key, value]) => (
                  <div key={key} className="flex">
                    <span className="w-1/3 text-gray-500">{key}</span>
                    <span className="w-2/3 break-all">{value}</span>
                  </div>
                ))}
              </div>
            </div>

            <div className="bg-opacity-5 dark:bg-opacity-5 bg-background rounded-lg p-4">
              <h4 className="mb-4 text-lg font-bold">Token</h4>
              <div className="space-y-2">
                {Object.entries(tokenInfo).map(([key, value]) => (
                  <div key={key} className="flex">
                    <span className="w-1/3 text-gray-500">{key}</span>
                    <span className="w-2/3 break-all">{value}</span>
                  </div>
                ))}
              </div>
            </div>

            <div className="bg-opacity-5 dark:bg-opacity-5 bg-background rounded-lg p-4">
              <h4 className="mb-4 text-lg font-bold">TimeLock</h4>
              <div className="space-y-2">
                {Object.entries(timeLockInfo).map(([key, value]) => (
                  <div key={key} className="flex">
                    <span className="w-1/3 text-gray-500">{key}</span>
                    <span className="w-2/3 break-all">{value}</span>
                  </div>
                ))}
              </div>
            </div>
          </>
        )}

        <Separator className="my-0" />

        <div className="flex justify-between">
          <Button
            variant="outline"
            type="button"
            className="w-[140px] rounded-full p-[10px]"
            onClick={onBack}
          >
            Back
          </Button>
          <Button
            type="button"
            className="w-[140px] rounded-full p-[10px]"
            onClick={handleSubmit}
            disabled={hasError || isLoading}
          >
            Submit
          </Button>
        </div>
      </div>
    </>
  );
}
