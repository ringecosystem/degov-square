import { NextResponse } from 'next/server';

import type { NextRequest } from 'next/server';

export function middleware(request: NextRequest) {
  if (request.nextUrl.pathname !== '/oauth/authorize' || !request.nextUrl.search) {
    return NextResponse.next();
  }

  const redirectUrl = request.nextUrl.clone();
  redirectUrl.hash = request.nextUrl.search.slice(1);
  redirectUrl.search = '';

  return NextResponse.redirect(redirectUrl);
}

export const config = {
  matcher: ['/oauth/authorize']
};
