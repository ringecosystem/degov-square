import { Suspense } from 'react';

import { OAuthAuthorizeClient } from './oauth-authorize-client';

export default function OAuthAuthorizePage() {
  return (
    <Suspense
      fallback={
        <div className="mx-auto flex w-full max-w-[520px] flex-col gap-[16px] px-[20px]">
          <div className="bg-card border-border rounded-[8px] border p-[24px]">
            <p className="text-muted-foreground text-[14px]">Loading authorization request...</p>
          </div>
        </div>
      }
    >
      <OAuthAuthorizeClient />
    </Suspense>
  );
}
