'use client';

import { ShieldCheck } from 'lucide-react';
import { useSearchParams } from 'next/navigation';
import { useCallback, useEffect, useMemo, useState } from 'react';

import { ConnectButton } from '@/components/connect-button';
import { Button } from '@/components/ui/button';
import { useSiweAuth } from '@/hooks/useSiweAuth';
import {
  getOAuthAuthorizeParams,
  OAuthApiError,
  stytchAuthorizeStart,
  stytchAuthorizeSubmit,
  type StytchOAuthAuthorizeStartResponse,
  type StytchOAuthScopeResult
} from '@/lib/api/oauth';
import { getToken, useAuthStore } from '@/stores/auth';

export const OAuthAuthorizeClient = () => {
  const searchParams = useSearchParams();
  const queryString = searchParams.toString();
  const authorizeParams = useMemo(
    () => getOAuthAuthorizeParams(new URLSearchParams(queryString)),
    [queryString]
  );

  const { token } = useAuthStore();
  const { authenticate, isAuthenticating, canAuthenticate } = useSiweAuth();
  const [startResponse, setStartResponse] = useState<StytchOAuthAuthorizeStartResponse | null>(
    null
  );
  const [startedKey, setStartedKey] = useState('');
  const [isStarting, setIsStarting] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [needsSignIn, setNeedsSignIn] = useState(false);

  const requestKey = `${token ?? ''}:${queryString}`;
  const hasRequiredParams = Boolean(authorizeParams.client_id && authorizeParams.redirect_uri);

  const startAuthorization = useCallback(
    async (authToken?: string | null) => {
      const activeToken = authToken || getToken() || token;
      if (!activeToken || !hasRequiredParams) {
        return;
      }

      setIsStarting(true);
      setError('');
      setNeedsSignIn(false);
      setStartedKey(`${activeToken}:${queryString}`);
      try {
        const response = await stytchAuthorizeStart(authorizeParams, activeToken);
        setStartResponse(response);
      } catch (err) {
        if (err instanceof OAuthApiError && err.status === 401) {
          useAuthStore.getState().clearAuth();
          setStartResponse(null);
          setStartedKey('');
          setNeedsSignIn(true);
          setError('Session expired. Please sign in again.');
          return;
        }
        setError(err instanceof Error ? err.message : 'Failed to load authorization request');
      } finally {
        setIsStarting(false);
      }
    },
    [authorizeParams, hasRequiredParams, queryString, token]
  );

  useEffect(() => {
    if (!token || !hasRequiredParams || startedKey === requestKey) {
      return;
    }
    void startAuthorization(token);
  }, [hasRequiredParams, requestKey, startAuthorization, startedKey, token]);

  const signIn = useCallback(async () => {
    setError('');
    setNeedsSignIn(false);
    const result = await authenticate();
    if (!result.success) {
      setNeedsSignIn(true);
      setError(result.error || 'Sign in failed');
      return;
    }
    await startAuthorization(result.token || getToken());
  }, [authenticate, startAuthorization]);

  const submitConsent = useCallback(
    async (consentGranted: boolean) => {
      const activeToken = getToken() || token;
      if (!activeToken) {
        setError('Please sign in first');
        return;
      }

      setIsSubmitting(true);
      setError('');
      setNeedsSignIn(false);
      try {
        const response = await stytchAuthorizeSubmit(authorizeParams, activeToken, consentGranted);
        if (!response.redirect_uri) {
          setError('Authorization response did not include a redirect URL');
          return;
        }
        window.location.assign(response.redirect_uri);
      } catch (err) {
        if (err instanceof OAuthApiError && err.status === 401) {
          useAuthStore.getState().clearAuth();
          setStartResponse(null);
          setStartedKey('');
          setNeedsSignIn(true);
          setError('Session expired. Please sign in again.');
          return;
        }
        setError(err instanceof Error ? err.message : 'Authorization failed');
      } finally {
        setIsSubmitting(false);
      }
    },
    [authorizeParams, token]
  );

  const clientName =
    startResponse?.client.client_name ||
    startResponse?.client.client_id ||
    authorizeParams.client_id ||
    'Unknown app';
  const scopes = startResponse?.scope_results?.length
    ? startResponse.scope_results
    : authorizeParams.scope
      ? authorizeParams.scope.split(/\s+/).map((scope) => ({
          scope,
          description: '',
          is_grantable: true
        }))
      : [];

  return (
    <div className="mx-auto flex w-full max-w-[520px] flex-col gap-[16px] px-[20px]">
      <div className="bg-card border-border flex flex-col gap-[20px] rounded-[8px] border p-[24px] shadow-sm">
        <div className="flex items-start gap-[12px]">
          <div className="bg-secondary text-secondary-foreground flex size-[40px] flex-shrink-0 items-center justify-center rounded-[8px]">
            <ShieldCheck size={20} />
          </div>
          <div className="min-w-0">
            <h1 className="text-[20px] leading-[1.3] font-semibold">Authorize app</h1>
            <p className="text-muted-foreground mt-[4px] text-[14px] leading-[1.5]">
              {startResponse ? `${clientName} wants access to DeGov Square.` : clientName}
            </p>
          </div>
        </div>

        {!hasRequiredParams && (
          <Notice message="Missing OAuth request details. Please return to the app and try again." />
        )}

        {hasRequiredParams && !token && (
          <div className="flex flex-col gap-[14px]">
            <p className="text-muted-foreground text-[14px] leading-[1.5]">
              Connect your wallet and sign in to continue.
            </p>
            <div className="flex flex-wrap items-center gap-[10px]">
              <ConnectButton />
              <Button
                onClick={signIn}
                disabled={!canAuthenticate || isAuthenticating}
                className="rounded-[100px]"
              >
                {isAuthenticating ? 'Signing in...' : needsSignIn ? 'Sign in again' : 'Sign in'}
              </Button>
            </div>
          </div>
        )}

        {hasRequiredParams && token && (
          <div className="flex flex-col gap-[18px]">
            {isStarting && (
              <p className="text-muted-foreground text-[14px]">Loading authorization request...</p>
            )}

            {startResponse && (
              <>
                <div className="flex flex-col gap-[8px]">
                  <p className="text-[14px] font-medium">Requested access</p>
                  <ScopeList scopes={scopes} />
                </div>

                <div className="flex flex-col-reverse gap-[10px] sm:flex-row sm:justify-end">
                  <Button
                    variant="outline"
                    onClick={() => submitConsent(false)}
                    disabled={isSubmitting}
                    className="rounded-[100px]"
                  >
                    Deny
                  </Button>
                  <Button
                    onClick={() => submitConsent(true)}
                    disabled={isSubmitting}
                    className="rounded-[100px]"
                  >
                    {isSubmitting ? 'Submitting...' : 'Allow'}
                  </Button>
                </div>
              </>
            )}

            {!isStarting && !startResponse && (
              <Button
                onClick={() => startAuthorization(token)}
                disabled={isStarting}
                className="self-start rounded-[100px]"
              >
                Load request
              </Button>
            )}
          </div>
        )}

        {error && <Notice message={error} />}
      </div>
    </div>
  );
};

const ScopeList = ({ scopes }: { scopes: StytchOAuthScopeResult[] }) => {
  if (!scopes.length) {
    return <p className="text-muted-foreground text-[14px]">No scopes requested.</p>;
  }

  return (
    <div className="border-border divide-border overflow-hidden rounded-[8px] border">
      {scopes.map((scope) => (
        <div key={scope.scope} className="flex flex-col gap-[3px] px-[12px] py-[10px]">
          <span className="text-[14px] font-medium">{scope.scope}</span>
          {scope.description && (
            <span className="text-muted-foreground text-[13px] leading-[1.4]">
              {scope.description}
            </span>
          )}
        </div>
      ))}
    </div>
  );
};

const Notice = ({ message }: { message: string }) => (
  <div className="border-destructive/30 bg-destructive/10 text-destructive rounded-[8px] border px-[12px] py-[10px] text-[14px] leading-[1.5]">
    {message}
  </div>
);
