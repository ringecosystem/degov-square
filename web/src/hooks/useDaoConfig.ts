import { useQuery } from '@tanstack/react-query';
import yaml from 'js-yaml';

export interface RegistryDao {
  code: string;
  config: string;
  name: string;
  logo: string;
  siteUrl: string;
  offChainDiscussionUrl: string;
  aiAgent: {
    endpoint: string;
  };
  description: string;
  links: {
    coingecko?: string;
    website?: string;
    twitter?: string;
    discord?: string;
    telegram?: string;
    github?: string;
    [key: string]: any;
  };
  wallet: {
    walletConnectProjectId: string;
    [key: string]: any;
  };
  chain: {
    id: number;
    name: string;
    logo: string;
    rpcs: string[];
    explorers: string[];
    nativeToken: {
      symbol: string;
      priceId: string;
      decimals: number;
    };
    [key: string]: any;
  };
  indexer: {
    endpoint: string;
    startBlock?: number;
    rpc?: string;
    gateway?: string;
    [key: string]: any;
  };
  contracts: {
    governor: string;
    governorToken: {
      address: string;
      standard: string;
      [key: string]: any;
    };
    timeLock: string;
    [key: string]: any;
  };
  safes?: Array<{
    name: string;
    chainId: number;
    link: string;
    [key: string]: any;
  }>;
  timeLockAssets?: Array<{
    name: string;
    contract: string;
    standard: string;
    priceId: string;
    [key: string]: any;
  }>;
  apps?: Array<any>;
  tags?: string[];
  [key: string]: any; // 兼容未来扩展
}

export type RegistryPreDao = {
  code: string;
  config?: string;
  tags?: string[];
};

export type RegistryConfig = Record<string, RegistryPreDao[]>;

async function fetchRegistryConfig(): Promise<RegistryConfig> {
  const res = await fetch('https://raw.githubusercontent.com/ringecosystem/degov-registry/main/config.yml');
  const text = await res.text();
  return yaml.load(text) as RegistryConfig;
}

async function fetchDaoDetail(configUrl: string): Promise<RegistryDao> {
  const res = await fetch(configUrl);
  const text = await res.text();
  return yaml.load(text) as RegistryDao;
}

export function useDaoConfig(): {
  daoConfigs: RegistryDao[];
  isLoading: boolean;
  error: Error | null;
} {
  const { data, isLoading, error } = useQuery({
    queryKey: ['dao-config'],
    queryFn: async () => {
      const registryConfig = await fetchRegistryConfig();
      const allDaos: RegistryDao[] = [];

      for (const chain in registryConfig) {
        const daos = (registryConfig[chain] || []).filter(
          dao => dao.config && !(dao.tags && dao.tags.includes('demo'))
        );

        const details = await Promise.all(
          daos.map(async (dao) => {
            try {
              const detail = await fetchDaoDetail(dao.config!);
              return {
                ...dao,
                ...detail,
                chain: detail.chain || { name: chain }, // 保证有 chain 字段
              };
            } catch (e) {
              return null;
            }
          })
        );

        allDaos.push(...(details.filter(Boolean) as RegistryDao[]));
      }

      return allDaos;
    }
  });

  return {
    daoConfigs: data || [],
    isLoading,
    error
  };
}