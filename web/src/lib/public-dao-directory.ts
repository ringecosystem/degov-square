import { createPublicClient } from '@/lib/graphql/client';
import { QUERY_DAOS } from '@/lib/graphql/queries';
import type { Dao } from '@/lib/graphql/types';

import { transformDaoData } from './dao-directory';

import type { DaoInfo } from '@/utils/config';

export type PublicDaoDirectoryResult = {
  daos: DaoInfo[];
  failed: boolean;
};

type PublicDaosResponse = {
  daos: Dao[];
};

export async function getPublicDaoDirectory(): Promise<PublicDaoDirectoryResult> {
  try {
    const client = createPublicClient();
    const data = await client.request<PublicDaosResponse>(QUERY_DAOS);

    return {
      daos: (data.daos ?? [])
        .filter((dao) => !dao.tags?.includes('demo'))
        .map((dao) => transformDaoData(dao)),
      failed: false
    };
  } catch (error) {
    console.error('Failed to load public DAO directory for initial HTML:', error);
    return {
      daos: [],
      failed: true
    };
  }
}
