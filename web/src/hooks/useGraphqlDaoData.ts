import { useMemo } from 'react';

import { useQueryDaosPublic } from '@/lib/graphql';
import type { Dao } from '@/lib/graphql/types';
import type { DaoInfo } from '@/utils/config';

// Transform GraphQL DAO data to match existing DaoInfo interface
function transformDaoData(dao: Dao, index: number): DaoInfo {
  // Fallback images in case logo URLs are not available
  const fallbackDaoIcons = ['/example/dao1.svg', '/example/dao2.svg', '/example/dao3.svg'];
  const fallbackNetworkIcon = '/example/network1.svg';

  return {
    id: dao.id,
    name: dao.name,
    code: dao.code,
    daoIcon: dao.logo || fallbackDaoIcons[index % fallbackDaoIcons.length], // Use real logo or fallback
    network: dao.chainName || `Chain ${dao.chainId}`,
    networkIcon: dao.chainLogo || fallbackNetworkIcon, // Use real chain logo or fallback
    proposals: dao.metricsCountProposals,
    favorite: false, // Will be determined by liked status when auth is available
    settable: true,
    website: dao.endpoint || '',
    indexer: '', // Not available in GraphQL data
    chainId: dao.chainId.toString(),
    chips: dao.chips
  };
}

export function useGraphqlDaoData() {
  const { data: graphqlData, isLoading, error } = useQueryDaosPublic();

  const daoData = useMemo(() => {
    if (!graphqlData?.daos) return [];

    // Show all DAOs without any filtering
    return graphqlData.daos.map((dao, index) => transformDaoData(dao, index));
  }, [graphqlData]);

  const refreshData = () => {
    // React Query will handle refresh automatically
    // Could use queryClient.invalidateQueries if needed
  };

  return {
    daoData,
    isLoading,
    error: error?.message || null,
    refreshData,
    // Additional data from GraphQL
    likedDaos: graphqlData?.likedDaos || [],
    subscribedDaos: graphqlData?.subscribedDaos || []
  };
}
