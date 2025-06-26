import { useQuery } from '@tanstack/react-query';
import yaml from 'js-yaml';

export interface DaoConfig {
  name: string;
  code: string;
  logo: string;
  xprofile?: string;
  chain?: {
    id: string;
    name: string;
  };
  links: {
    website: string;
    config: string;
    indexer: string;
  };
}

interface ConfigData {
  daos: DaoConfig[];
}

export function useDaoConfig() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['dao-config'],
    queryFn: async (): Promise<ConfigData> => {
      const response = await fetch('/config.yml');
      const yamlText = await response.text();
      const config = yaml.load(yamlText) as ConfigData;
      return config;
    }
  });

  return {
    daoConfigs: data?.daos || [],
    isLoading,
    error
  };
}
