import { useMemo } from 'react';

import { useQueryDaosPublic, useQueryDaos } from '@/lib/graphql';
import { useAuth } from '@/contexts/auth';
import type { Dao } from '@/lib/graphql/types';
import type { DaoInfo } from '@/utils/config';

// Transform GraphQL DAO data to match existing DaoInfo interface
function transformDaoData(dao: Dao, index: number, likedDaos: Array<{code: string}> = []): DaoInfo {
  // Fallback images in case logo URLs are not available
  const fallbackDaoIcons = ['/example/dao1.svg', '/example/dao2.svg', '/example/dao3.svg'];
  const fallbackNetworkIcon = '/example/network1.svg';

  // Check if this DAO is liked by the user
  const isLiked = likedDaos.some(likedDao => likedDao.code === dao.code);

  return {
    id: dao.id,
    name: dao.name,
    code: dao.code,
    daoIcon: dao.logo || fallbackDaoIcons[index % fallbackDaoIcons.length], // Use real logo or fallback
    network: dao.chainName || `Chain ${dao.chainId}`,
    networkIcon: dao.chainLogo || fallbackNetworkIcon, // Use real chain logo or fallback
    proposals: dao.metricsCountProposals,
    favorite: isLiked, // Set based on user's liked status
    settable: true,
    website: dao.endpoint || '',
    indexer: '', // Not available in GraphQL data
    chainId: dao.chainId.toString(),
    chips: dao.chips
  };
}

export function useGraphqlDaoData() {
  const { isAuthenticated } = useAuth();
  
  // Always use public query as fallback
  const publicQuery = useQueryDaosPublic();
  // Only enable auth query when authenticated
  const authQuery = useQueryDaos();
  
  // Use auth data if available and user is authenticated, otherwise use public data
  const graphqlData = (isAuthenticated && authQuery.data) ? authQuery.data : publicQuery.data;
  const isLoading = isAuthenticated ? authQuery.isLoading : publicQuery.isLoading;
  const error = isAuthenticated ? authQuery.error : publicQuery.error;

  const daoData = useMemo(() => {
    if (!graphqlData?.daos) return [];

    // Get liked DAOs from authenticated query
    const likedDaos = isAuthenticated ? (graphqlData.likedDaos || []) : [];

    // Show all DAOs without demo filtering
    return graphqlData.daos
      ?.filter((dao) => !dao.tags?.includes('demo'))
      .map((dao, index) => transformDaoData(dao, index, likedDaos));
  }, [graphqlData, isAuthenticated]);

  const refreshData = () => {
    // React Query will handle refresh automatically
    // Could use queryClient.invalidateQueries if needed
  };

  return {
    daoData,
    isLoading,
    error: error?.message || null,
    refreshData,
    // Additional data from GraphQL (only available when authenticated)
    likedDaos: isAuthenticated ? (graphqlData?.likedDaos || []) : [],
    subscribedDaos: isAuthenticated ? (graphqlData?.subscribedDaos || []) : [],
    // Helper function to get like status for a specific DAO
    isLiked: (daoCode: string) => {
      if (!isAuthenticated || !graphqlData?.likedDaos) return false;
      return graphqlData.likedDaos.some(dao => dao.code === daoCode);
    }
  };
}
