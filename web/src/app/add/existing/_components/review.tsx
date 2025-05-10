'use client';

import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { Step1FormValues } from './step1-form';
import { Step2FormValues } from './step2-form';
import { useAddDaoReview } from '@/hooks/useAddDaoReview';
import { getChains } from '@/utils/chains';
import { useConfirm } from '@/contexts/confirm-context';
import { useCallback } from 'react';
interface ReviewProps {
  step1Data: Step1FormValues;
  step2Data: Step2FormValues;
  onSubmit: () => void;
  onBack: () => void;
}

export function Review({ step1Data, step2Data, onSubmit, onBack }: ReviewProps) {
  const { data, isLoading } = useAddDaoReview(step1Data, step2Data, true);
  const { confirm } = useConfirm();
  // 查找链名称
  const network =
    getChains().find((chain) => chain.id.toString() === step1Data.chainId)?.name ||
    step1Data.chainId;

  // 获取显示数据 - 基础信息
  const basicInfo = {
    Name: step1Data.name,
    Network: network,
    'DAO Url': `https://${step1Data.daoUrl}${step1Data.domain}`
  };

  // 获取显示数据 - 治理合约信息
  const governorInfo = {
    'Governor Address': step2Data.governorAddress,
    'Proposal threshold': isLoading
      ? 'Loading...'
      : data.governanceParams?.proposalThreshold.toString() || 'N/A',
    Quorum: isLoading ? 'Loading...' : data.governanceParams?.quorum.toString() || 'N/A',
    'Proposal delay': isLoading
      ? 'Loading...'
      : `${data.governanceParams?.votingDelay.toString() || 'N/A'} day`,
    'Voting period': isLoading
      ? 'Loading...'
      : `${data.governanceParams?.votingPeriod.toString() || 'N/A'} days`
  };

  // 获取显示数据 - 代币信息
  const tokenInfo = {
    'Token name': isLoading ? 'Loading...' : data.tokenMetadata?.name || 'N/A',
    'Token Address': step2Data.tokenAddress,
    'Token type': step2Data.tokenType,
    'token symbol': isLoading ? 'Loading...' : data.tokenMetadata?.symbol || 'N/A',
    'token decimal': isLoading ? 'Loading...' : data.tokenMetadata?.decimals.toString() || 'N/A'
  };

  // 获取显示数据 - 时间锁信息
  const timeLockInfo = {
    'TimeLock Address': step2Data.timeLockAddress,
    'TimeLock delay': isLoading
      ? 'Loading...'
      : `${data.governanceParams?.timeLockDelay.toString() || 'N/A'} day`
  };

  const handleSubmit = useCallback(() => {
    confirm({
      title: 'Congratulations !',
      description:
        'We have received your DAO’s information, will review it and get back to you soon.',
      confirmText: 'Ok',
      variant: 'default',
      onConfirm: onSubmit
    });
  }, [onSubmit]);

  return (
    <>
      <Separator className="my-0" />
      <h3 className="text-base font-medium">
        Review all the information of the DAO before proceeding to build the DAO.
      </h3>

      <div className="mt-4 flex flex-col gap-6">
        {/* 基础信息 */}
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

        {/* 治理合约信息 */}
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

        {/* 代币信息 */}
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

        {/* 时间锁信息 */}
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

        {/* 操作按钮 */}
        <div className="flex justify-between pt-4">
          <Button variant="outline" type="button" className="rounded-full px-8" onClick={onBack}>
            Back
          </Button>
          <Button type="button" className="rounded-full px-8" onClick={handleSubmit}>
            Submit
          </Button>
        </div>
      </div>
    </>
  );
}
