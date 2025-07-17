import { useQuery } from '@tanstack/react-query';

import type { DaoInfo } from '@/utils/config';

import { useDaoConfig } from './useDaoConfig';

async function getProposalsCount(indexerUrl: string): Promise<number> {
  try {
    const response = await fetch(indexerUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        query: `
          query MyQuery {
            dataMetrics {
              proposalsCount
            }
          }
        `
      })
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    const data = await response.json();

    if (data.errors) {
      console.error('GraphQL errors:', data.errors);
      return 0;
    }

    const proposalsCount = data.data?.dataMetrics?.[0]?.proposalsCount || 0;
    return proposalsCount;
  } catch (error) {
    console.error(`Error fetching proposals count from ${indexerUrl}:`, error);
    return 0;
  }
}

export function useDaoData() {
  const { daoConfigs, isLoading: isLoadingConfigs, error: configError } = useDaoConfig();

  const {
    data: daoData = [],
    isLoading: isLoadingProposals,
    error: proposalsError
  } = useQuery({
    queryKey: ['dao-data', daoConfigs],
    queryFn: async (): Promise<DaoInfo[]> => {
      const daoInfoPromises = daoConfigs.map(async (dao, index) => {
        const proposalsCount = await getProposalsCount(dao.indexer.endpoint);

        const chainId = dao.chain?.id?.toString();

        return {
          id: index.toString(),
          name: dao.name,
          code: dao.code,
          daoIcon: dao.logo,
          network: dao.chain?.name || 'Unknown Network',
          networkIcon: dao.chain?.logo,
          proposals: proposalsCount,
          favorite: false,
          settable: true,
          website: dao.siteUrl || '',
          indexer: dao.indexer?.endpoint || '',
          chainId: chainId
        };
      });

      return Promise.all(daoInfoPromises);
    },
    enabled: !!daoConfigs.length
  });

  const isLoading = isLoadingConfigs || isLoadingProposals;
  const error = configError || proposalsError;

  const refreshData = async () => {
    // React Query will handle refresh automatically when queryKey changes
    // We can also use queryClient.invalidateQueries if needed
  };

  return {
    daoData: daoData || [],
    isLoading,
    error: error?.message || null,
    refreshData
  };
}
