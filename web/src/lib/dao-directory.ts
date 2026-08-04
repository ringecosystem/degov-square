import type { Dao } from '@/lib/graphql/types';
import type { DaoInfo } from '@/utils/config';

export function transformDaoData(dao: Dao): DaoInfo {
  return {
    id: dao.id,
    name: dao.name,
    code: dao.code,
    daoIcon: dao.logo,
    network: dao.chainName || `Chain ${dao.chainId}`,
    networkIcon: dao.chainLogo,
    proposals: dao.metricsCountProposals,
    favorite: dao.liked,
    settable: true,
    website: dao.endpoint || '',
    indexer: '',
    chainId: dao.chainId.toString(),
    chips: dao.chips,
    lastProposal: dao.lastProposal
  };
}
