const trimTrailingSlash = (value: string) => value.replace(/\/+$/, '');

export const getApiBase = () => {
  const apiEndpoint = process.env.NEXT_PUBLIC_API_ENDPOINT?.trim();
  if (apiEndpoint) {
    return trimTrailingSlash(apiEndpoint);
  }

  const graphqlEndpoint = process.env.NEXT_PUBLIC_GRAPHQL_ENDPOINT?.trim();
  if (graphqlEndpoint) {
    return trimTrailingSlash(graphqlEndpoint.replace(/\/graphql\/?$/, ''));
  }

  if (typeof window !== 'undefined') {
    return window.location.origin;
  }

  return '';
};
