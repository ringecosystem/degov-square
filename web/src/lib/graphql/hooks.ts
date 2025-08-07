import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

import { useAuth } from '@/contexts/auth';

import { createAuthorizedClient, createPublicClient } from './client';
import { QUERY_NONCE, LOGIN_MUTATION, MODIFY_LIKE_DAO_MUTATION, QUERY_DAOS } from './queries';

import type {
  NonceResponse,
  LoginResponse,
  LoginVariables,
  ModifyLikeDaoVariables,
  DaosResponse,
  NonceVariables
} from './types';

export const QUERY_KEYS = {
  nonce: (length: number) => ['nonce', length] as const,
  daos: () => ['daos'] as const
} as const;

export const useQueryNonce = (length: number = 10) => {
  return useQuery({
    queryKey: QUERY_KEYS.nonce(length),
    queryFn: async () => {
      const client = createAuthorizedClient();
      const variables: NonceVariables = { length };
      const data = await client.request<{ nonce: string }>(QUERY_NONCE, { input: variables });
      return data.nonce;
    },
    staleTime: 0,
    gcTime: 0
  });
};

export const useLogin = () => {
  const { setToken } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (variables: LoginVariables) => {
      const client = createAuthorizedClient();
      const data = await client.request<LoginResponse>(LOGIN_MUTATION, variables);
      return data.login.token;
    },
    onSuccess: (token) => {
      setToken(token);
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.daos() });
    },
    onError: (error) => {
      console.error('Login failed:', error);
    }
  });
};

export const useQueryDaos = () => {
  const { token } = useAuth();

  return useQuery({
    queryKey: [QUERY_KEYS.daos(), token],
    queryFn: async () => {
      const client = createAuthorizedClient(token);
      const data = await client.request<DaosResponse>(QUERY_DAOS);
      return data;
    },
    staleTime: 1000 * 60 * 5
  });
};

export const useQueryDaosPublic = () => {
  return useQuery({
    queryKey: ['daos-public'],
    queryFn: async () => {
      const client = createPublicClient();
      const data = await client.request<DaosResponse>(QUERY_DAOS);
      return data;
    },
    staleTime: 1000 * 60 * 5
  });
};

export const useModifyLikeDao = () => {
  const { token } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (variables: ModifyLikeDaoVariables) => {
      if (!token) {
        throw new Error('Authentication required');
      }

      const client = createAuthorizedClient(token);
      const data = await client.request<{ modifyLikeDao: boolean }>(
        MODIFY_LIKE_DAO_MUTATION,
        variables
      );
      return data.modifyLikeDao;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.daos() });
    },
    onError: (error) => {
      console.error('Failed to modify DAO like status:', error);
    }
  });
};

export const useLikeDao = () => {
  const modifyLikeMutation = useModifyLikeDao();

  return {
    likeDao: (daoCode: string) => modifyLikeMutation.mutate({ input: { daoCode, action: 'LIKE' } }),
    unlikeDao: (daoCode: string) =>
      modifyLikeMutation.mutate({ input: { daoCode, action: 'UNLIKE' } }),
    ...modifyLikeMutation
  };
};

export const useLogout = () => {
  const { setToken } = useAuth();
  const queryClient = useQueryClient();

  return () => {
    setToken(null);
    queryClient.clear();
  };
};
