import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

import { useAuth } from '@/contexts/auth';

import { createAuthorizedClient, createPublicClient } from './client';
import { QUERY_NONCE, LOGIN_MUTATION, MODIFY_LIKE_DAO_MUTATION, QUERY_DAOS } from './queries';

import type {
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
      const client = createPublicClient();
      const variables: NonceVariables = { input: { length } };
      const data = await client.request<{ nonce: string }>(QUERY_NONCE, variables);
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
      const client = createPublicClient();
      const data = await client.request<LoginResponse>(LOGIN_MUTATION, variables);
      return data.login.token;
    },
    onSuccess: (token) => {
      setToken(token);
      queryClient.invalidateQueries({ 
        predicate: (query) => query.queryKey[0] === 'daos'
      });
    },
    onError: (error) => {
      console.error('Login failed:', error);
    }
  });
};

export const useQueryDaos = () => {
  const { token } = useAuth();

  return useQuery({
    queryKey: [...QUERY_KEYS.daos(), token],
    queryFn: async () => {
      if (!token) throw new Error('No authentication token available');
      const client = createAuthorizedClient(token);
      const data = await client.request<DaosResponse>(QUERY_DAOS);
      return data;
    },
    enabled: !!token, // Only run query when token is available
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
  const { token, setToken } = useAuth();
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
    onMutate: async (variables) => {
      // Cancel any outgoing refetches (so they don't overwrite our optimistic update)
      await queryClient.cancelQueries({ queryKey: QUERY_KEYS.daos() });

      // Snapshot the previous value
      const previousData = queryClient.getQueryData([...QUERY_KEYS.daos(), token]);

      // Optimistically update to the new value
      queryClient.setQueryData([...QUERY_KEYS.daos(), token], (old: DaosResponse | undefined) => {
        if (!old) return old;
        
        const updatedLikedDaos = variables.input.action === 'LIKE'
          ? [...(old.likedDaos || []), { code: variables.input.daoCode }]
          : (old.likedDaos || []).filter((dao) => dao.code !== variables.input.daoCode);

        return {
          ...old,
          likedDaos: updatedLikedDaos
        };
      });

      return { previousData };
    },
    onError: (error, _variables, context) => {
      // If the mutation fails, use the context returned from onMutate to roll back
      if (context?.previousData) {
        queryClient.setQueryData([...QUERY_KEYS.daos(), token], context.previousData);
      }
      
      if (error instanceof Error && error.message.includes('401')) {
        setToken(null);
      }
      console.error('Failed to modify DAO like status:', error);
    },
    onSettled: () => {
      // Always refetch after error or success
      queryClient.invalidateQueries({ 
        predicate: (query) => query.queryKey[0] === 'daos'
      });
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
