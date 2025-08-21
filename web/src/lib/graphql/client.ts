import { GraphQLClient } from 'graphql-request';

const GRAPHQL_ENDPOINT = process.env.NEXT_PUBLIC_GRAPHQL_ENDPOINT ?? '';

export const graphqlClient = new GraphQLClient(GRAPHQL_ENDPOINT, {
  headers: {
    'Content-Type': 'application/json'
  }
});

export const createAuthorizedClient = (token?: string | null) => {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json'
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  return new GraphQLClient(GRAPHQL_ENDPOINT, { headers });
};

export const createPublicClient = () => {
  return new GraphQLClient(GRAPHQL_ENDPOINT, {
    headers: {
      'Content-Type': 'application/json'
    }
  });
};
