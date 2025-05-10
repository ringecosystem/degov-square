import { useMemo } from 'react';
import { useReadContracts } from 'wagmi';

import { type Step1FormValues } from '@/app/add/existing/_components/step1-form';
import { type Step2FormValues } from '@/app/add/existing/_components/step2-form';
import { abi as governorAbi } from '@/config/abi/governor';
import { abi as timeLockAbi } from '@/config/abi/timeLock';
import { abi as tokenAbi } from '@/config/abi/token';

import type { Address } from 'viem';

interface GovernanceParams {
  proposalThreshold: bigint;
  quorum: bigint;
  votingDelay: bigint;
  votingPeriod: bigint;
  timeLockDelay: bigint;
}

interface TokenMetadata {
  symbol: string;
  name: string;
  decimals: number;
}

export interface ReviewData {
  basicInfo: Step1FormValues;
  contractInfo: Step2FormValues;
  governanceParams: GovernanceParams | null;
  tokenMetadata: TokenMetadata | null;
}

export function useAddDaoReview(
  step1Data: Step1FormValues | null,
  step2Data: Step2FormValues | null,
  enabled: boolean = false
) {
  const governorAddress = step2Data?.governorAddress as Address;
  const timeLockAddress = step2Data?.timeLockAddress as Address;
  const tokenAddress = step2Data?.tokenAddress as Address;
  const tokenType = step2Data?.tokenType;
  const chainId = step1Data?.chainId ? parseInt(step1Data.chainId) : undefined;

  // get governance params
  const { data: governanceData, isLoading: isGovernanceLoading } = useReadContracts({
    contracts: [
      {
        address: governorAddress,
        abi: governorAbi,
        functionName: 'proposalThreshold',
        chainId
      },
      {
        address: governorAddress,
        abi: governorAbi,
        functionName: 'quorum',
        args: [BigInt(0)], // use 0 as clock value, simplify processing
        chainId
      },
      {
        address: governorAddress,
        abi: governorAbi,
        functionName: 'votingDelay',
        chainId
      },
      {
        address: governorAddress,
        abi: governorAbi,
        functionName: 'votingPeriod',
        chainId
      },
      {
        address: timeLockAddress,
        abi: timeLockAbi,
        functionName: 'getMinDelay',
        chainId
      }
    ],
    query: {
      enabled: enabled && Boolean(governorAddress) && Boolean(timeLockAddress) && Boolean(chainId)
    }
  });

  // get token metadata
  const { data: tokenData, isLoading: isTokenLoading } = useReadContracts({
    contracts:
      tokenType === 'ERC20'
        ? [
            {
              address: tokenAddress,
              abi: tokenAbi,
              functionName: 'symbol',
              chainId
            },
            {
              address: tokenAddress,
              abi: tokenAbi,
              functionName: 'name',
              chainId
            },
            {
              address: tokenAddress,
              abi: tokenAbi,
              functionName: 'decimals',
              chainId
            }
          ]
        : [
            {
              address: tokenAddress,
              abi: tokenAbi,
              functionName: 'symbol',
              chainId
            },
            {
              address: tokenAddress,
              abi: tokenAbi,
              functionName: 'name',
              chainId
            }
          ],
    query: {
      enabled: enabled && Boolean(tokenAddress) && Boolean(chainId)
    }
  });

  // format governance params
  const governanceParams = useMemo((): GovernanceParams | null => {
    if (!governanceData || governanceData.some((item) => item.status !== 'success')) {
      return null;
    }
    return {
      proposalThreshold: governanceData[0].result as bigint,
      quorum: governanceData[1].result as bigint,
      votingDelay: governanceData[2].result as bigint,
      votingPeriod: governanceData[3].result as bigint,
      timeLockDelay: governanceData[4].result as bigint
    };
  }, [governanceData]);

  // format token metadata
  const tokenMetadata = useMemo((): TokenMetadata | null => {
    if (!tokenData || tokenData.some((item) => item.status !== 'success')) {
      return null;
    }
    return {
      symbol: tokenData[0].result as string,
      name: tokenData[1].result as string,
      decimals: tokenType === 'ERC20' && tokenData[2]?.result ? Number(tokenData[2].result) : 0
    };
  }, [tokenData, tokenType]);

  const reviewData: ReviewData = useMemo(
    () => ({
      basicInfo: step1Data || ({} as Step1FormValues),
      contractInfo: step2Data || ({} as Step2FormValues),
      governanceParams,
      tokenMetadata
    }),
    [step1Data, step2Data, governanceParams, tokenMetadata]
  );

  return {
    data: reviewData,
    isLoading: isGovernanceLoading || isTokenLoading
  };
}
