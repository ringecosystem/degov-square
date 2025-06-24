import { useMemo } from 'react';
import { useReadContract, useReadContracts } from 'wagmi';

import { type Step1FormValues } from '@/app/add/existing/_components/step1-form';
import { type Step2FormValues } from '@/app/add/existing/_components/step2-form';
import { abi as governorAbi } from '@/config/abi/governor';
import { abi as timeLockAbi } from '@/config/abi/timeLock';
import { abi as tokenAbi } from '@/config/abi/token';

import type { Address } from 'viem';

interface StaticGovernanceParams {
  proposalThreshold: bigint;
  votingDelay: bigint;
  votingPeriod: bigint;
  timeLockDelay: bigint;
}

interface GovernanceParams extends StaticGovernanceParams {
  quorum: bigint;
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

export function useStaticGovernanceParams(
  step1Data: Step1FormValues | null,
  step2Data: Step2FormValues | null,
  enabled: boolean = false
) {
  const governorAddress = step2Data?.governorAddress as Address;
  const timeLockAddress = step2Data?.timeLockAddress as Address;
  const chainId = step1Data?.chainId ? parseInt(step1Data.chainId) : undefined;
  const isEnabled =
    enabled && Boolean(governorAddress) && Boolean(timeLockAddress) && Boolean(chainId);

  const { data, isLoading, error, isFetching } = useReadContracts({
    contracts: [
      {
        address: governorAddress as `0x${string}`,
        abi: governorAbi,
        functionName: 'proposalThreshold' as const,
        chainId
      },
      {
        address: governorAddress as `0x${string}`,
        abi: governorAbi,
        functionName: 'votingDelay' as const,
        chainId
      },
      {
        address: governorAddress as `0x${string}`,
        abi: governorAbi,
        functionName: 'votingPeriod' as const,
        chainId
      },
      {
        address: timeLockAddress as `0x${string}`,
        abi: timeLockAbi,
        functionName: 'getMinDelay' as const,
        chainId
      }
    ],
    query: {
      retry: false,
      staleTime: 24 * 60 * 60 * 1000, // 1 day
      enabled: isEnabled
    }
  });

  const isComplete = useMemo(() => {
    if (!data) return false;
    return data.every((item) => item.status === 'success');
  }, [data]);

  const formattedData: StaticGovernanceParams | null = useMemo(() => {
    if (!isComplete || !data) return null;

    return {
      proposalThreshold: data[0].result as bigint,
      votingDelay: data[1].result as bigint,
      votingPeriod: data[2].result as bigint,
      timeLockDelay: data[3].result as bigint
    };
  }, [data, isComplete]);

  return {
    data: formattedData,
    isLoading,
    isFetching,
    error: error as Error | null,
    isComplete
  };
}

export function useQuorum(
  step1Data: Step1FormValues | null,
  step2Data: Step2FormValues | null,
  enabled: boolean = false
) {
  const governorAddress = step2Data?.governorAddress as Address;
  const chainId = step1Data?.chainId ? parseInt(step1Data.chainId) : undefined;
  const isEnabled = enabled && Boolean(governorAddress) && Boolean(chainId);

  const {
    data: clockData,
    isLoading: isClockLoading,
    isFetching: isClockFetching,
    refetch: refetchClock
  } = useReadContract({
    address: governorAddress as `0x${string}`,
    abi: governorAbi,
    functionName: 'clock' as const,
    chainId,
    query: {
      enabled: isEnabled,
      staleTime: 0
    }
  });

  const clockEnabled = isEnabled && Boolean(clockData);

  const {
    data: quorumData,
    isLoading: isQuorumLoading,
    error: quorumError,
    isFetching: isQuorumFetching
  } = useReadContract({
    address: governorAddress as `0x${string}`,
    abi: governorAbi,
    functionName: 'quorum' as const,
    args: clockData ? [BigInt(clockData)] : undefined,
    chainId,
    query: {
      enabled: clockEnabled
    }
  });

  return {
    quorum: quorumData as bigint | undefined,
    clockData: clockData as bigint | undefined,
    isLoading: isClockLoading || isQuorumLoading,
    isFetching: isClockFetching || isQuorumFetching,
    error: quorumError as Error | null,
    refetchClock,
    isComplete: Boolean(quorumData)
  };
}

export function useGovernanceParams(
  step1Data: Step1FormValues | null,
  step2Data: Step2FormValues | null,
  enabled: boolean = false
) {
  const {
    data: staticParams,
    isLoading: isStaticLoading,
    isFetching: isStaticFetching,
    error: staticError,
    isComplete: isStaticComplete
  } = useStaticGovernanceParams(step1Data, step2Data, enabled);

  const {
    quorum,
    isLoading: isQuorumLoading,
    isFetching: isQuorumFetching,
    error: quorumError,
    refetchClock,
    isComplete: isQuorumComplete
  } = useQuorum(step1Data, step2Data, enabled);

  const isComplete = isStaticComplete && isQuorumComplete;

  const formattedData: GovernanceParams | null = useMemo(() => {
    if (!isComplete || !staticParams || quorum === undefined) {
      return null;
    }

    return {
      proposalThreshold: staticParams.proposalThreshold,
      votingDelay: staticParams.votingDelay,
      votingPeriod: staticParams.votingPeriod,
      timeLockDelay: staticParams.timeLockDelay,
      quorum
    };
  }, [staticParams, quorum, isComplete]);

  return {
    data: formattedData,
    isLoading: isStaticLoading || isQuorumLoading,
    isFetching: isStaticFetching || isQuorumFetching,
    isStaticLoading,
    isStaticFetching,
    isQuorumLoading,
    isQuorumFetching,
    error: staticError || quorumError,
    refetchClock,
    isComplete
  };
}

export function useAddDaoReview(
  step1Data: Step1FormValues | null,
  step2Data: Step2FormValues | null,
  enabled: boolean = false
) {
  const { data: governanceParams, isLoading: isGovernanceLoading } = useGovernanceParams(
    step1Data,
    step2Data,
    enabled
  );

  const governorAddress = step2Data?.governorAddress as Address;
  const tokenAddress = step2Data?.tokenAddress as Address;
  const tokenType = step2Data?.tokenType;
  const chainId = step1Data?.chainId ? parseInt(step1Data.chainId) : undefined;

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
