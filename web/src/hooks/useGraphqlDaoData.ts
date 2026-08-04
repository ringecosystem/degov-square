import { useMemo } from 'react';

import { useQueryDaosPublic, useQueryDaos } from '@/lib/graphql';
import { transformDaoData } from '@/lib/dao-directory';
import { useAuthStore } from '@/stores/auth';

export function useGraphqlDaoData() {
  const { isAuthenticated } = useAuthStore();

  const publicQuery = useQueryDaosPublic();
  const authQuery = useQueryDaos();

  const graphqlData = isAuthenticated() && authQuery.data ? authQuery.data : publicQuery.data;
  const isLoading = isAuthenticated() ? authQuery.isLoading : publicQuery.isLoading;
  const error = isAuthenticated() ? authQuery.error : publicQuery.error;

  const daoData = useMemo(() => {
    if (!graphqlData?.daos) return [];

    return graphqlData.daos
      ?.filter((dao) => !dao.tags?.includes('demo'))
      .map((dao) => transformDaoData(dao));
  }, [graphqlData]);

  return {
    daoData,
    isLoading,
    error: error?.message || null,
    subscribedDaos: isAuthenticated() ? graphqlData?.subscribedDaos || [] : []
  };
}
