import { getApiBase } from './base';

export interface OAuthAuthorizeParams {
  client_id: string;
  redirect_uri: string;
  response_type: string;
  scope: string;
  state: string;
  nonce: string;
  code_challenge: string;
  code_challenge_method: string;
}

export interface StytchOAuthClientInfo {
  client_id: string;
  client_name: string;
  client_description?: string;
  client_type?: string;
  logo_url?: string;
}

export interface StytchOAuthScopeResult {
  scope: string;
  description?: string;
  is_grantable: boolean;
}

export interface StytchOAuthAuthorizeStartResponse {
  client: StytchOAuthClientInfo;
  consent_required: boolean;
  scope_results: StytchOAuthScopeResult[];
}

export interface StytchOAuthAuthorizeSubmitResponse {
  redirect_uri: string;
}

export const getOAuthAuthorizeParams = (searchParams: URLSearchParams): OAuthAuthorizeParams => {
  return {
    client_id: searchParams.get('client_id') ?? '',
    redirect_uri: searchParams.get('redirect_uri') ?? '',
    response_type: searchParams.get('response_type') || 'code',
    scope: searchParams.get('scope') ?? '',
    state: searchParams.get('state') ?? '',
    nonce: searchParams.get('nonce') ?? '',
    code_challenge: searchParams.get('code_challenge') ?? '',
    code_challenge_method: searchParams.get('code_challenge_method') ?? ''
  };
};

export const stytchAuthorizeStart = async (
  params: OAuthAuthorizeParams,
  token: string
): Promise<StytchOAuthAuthorizeStartResponse> => {
  return postStytchOAuth('/api/oauth/stytch/authorize/start', params, token);
};

export const stytchAuthorizeSubmit = async (
  params: OAuthAuthorizeParams,
  token: string,
  consentGranted: boolean
): Promise<StytchOAuthAuthorizeSubmitResponse> => {
  return postStytchOAuth(
    '/api/oauth/stytch/authorize/submit',
    { ...params, consent_granted: consentGranted },
    token
  );
};

const postStytchOAuth = async <T>(path: string, body: unknown, token: string): Promise<T> => {
  const response = await fetch(`${getApiBase()}${path}`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(body)
  });

  const data = await response.json().catch(() => null);
  if (!response.ok) {
    const message =
      typeof data?.error === 'string' ? data.error : `Request failed: ${response.status}`;
    throw new Error(message);
  }

  return data as T;
};
