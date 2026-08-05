'use client';

export const SQUARE_DAO_CLICK_EVENT_NAME = 'degov_square_dao_click';

export type ChannelGroup =
  | 'organic-search'
  | 'social-organic'
  | 'documentation-referral'
  | 'cross-product-degov-referral'
  | 'ai-search-assistant-referral'
  | 'other-external-referral'
  | 'direct-unknown';

export type TargetHostClass = 'degov-dao-host' | 'external-dao-host' | 'unknown';

export type SquareDaoClickEventParams = {
  source_surface: 'square';
  dao_slug_or_public_id: string;
  target_host_class: TargetHostClass;
  channel_group: ChannelGroup;
};

declare global {
  interface Window {
    gtag?: (command: 'event', eventName: string, params: SquareDaoClickEventParams) => void;
  }
}

const SEARCH_HOST_SUBSTRINGS = ['google.', 'yahoo.'];
const SEARCH_HOST_DOMAINS = ['bing.com', 'duckduckgo.com', 'baidu.com', 'yandex.com'];
const SOCIAL_HOSTS = [
  'x.com',
  'twitter.com',
  'linkedin.com',
  'facebook.com',
  'discord.com',
  'telegram.org',
  't.me'
];
const AI_REFERRER_HOSTS = [
  'chatgpt.com',
  'perplexity.ai',
  'copilot.microsoft.com',
  'gemini.google.com'
];

function normalizePublicId(value: string | null | undefined): string {
  return String(value ?? '')
    .trim()
    .slice(0, 160);
}

function normalizeHost(hostname: string | null | undefined): string {
  return String(hostname ?? '')
    .toLowerCase()
    .replace(/^www\./, '');
}

function hostMatchesDomain(hostname: string, candidate: string): boolean {
  return hostname === candidate || hostname.endsWith(`.${candidate}`);
}

function hostMatchesDomains(hostname: string, candidates: string[]): boolean {
  return candidates.some((candidate) => hostMatchesDomain(hostname, candidate));
}

export function getChannelGroupFromReferrer(
  referrer: string | null | undefined,
  currentHost: string | null | undefined
): ChannelGroup {
  if (!referrer) return 'direct-unknown';

  let referrerUrl: URL;
  try {
    referrerUrl = new URL(referrer);
  } catch {
    return 'direct-unknown';
  }

  const referrerHost = normalizeHost(referrerUrl.hostname);
  const current = normalizeHost(currentHost);

  if (current && referrerHost === current) {
    return 'direct-unknown';
  }

  if (hostMatchesDomains(referrerHost, AI_REFERRER_HOSTS)) {
    return 'ai-search-assistant-referral';
  }

  if (
    SEARCH_HOST_SUBSTRINGS.some((candidate) => referrerHost.includes(candidate)) ||
    hostMatchesDomains(referrerHost, SEARCH_HOST_DOMAINS)
  ) {
    return 'organic-search';
  }

  if (hostMatchesDomains(referrerHost, SOCIAL_HOSTS)) {
    return 'social-organic';
  }

  if (referrerHost === 'docs.degov.ai') {
    return 'documentation-referral';
  }

  if (referrerHost.endsWith('.degov.ai') || referrerHost === 'degov.ai') {
    return 'cross-product-degov-referral';
  }

  return 'other-external-referral';
}

export function getTargetHostClass(targetUrl: string | null | undefined): TargetHostClass {
  if (!targetUrl) return 'unknown';

  let url: URL;
  try {
    url = new URL(targetUrl);
  } catch {
    return 'unknown';
  }

  const host = normalizeHost(url.hostname);
  return host.endsWith('.degov.ai') || host === 'degov.ai' ? 'degov-dao-host' : 'external-dao-host';
}

export function buildSquareDaoClickEventParams({
  daoCode,
  targetUrl,
  referrer,
  currentHost
}: {
  daoCode: string | null | undefined;
  targetUrl: string | null | undefined;
  referrer: string | null | undefined;
  currentHost: string | null | undefined;
}): SquareDaoClickEventParams {
  return {
    source_surface: 'square',
    dao_slug_or_public_id: normalizePublicId(daoCode),
    target_host_class: getTargetHostClass(targetUrl),
    channel_group: getChannelGroupFromReferrer(referrer, currentHost)
  };
}

export function sendSquareAnalyticsEvent(
  eventName: string,
  params: SquareDaoClickEventParams
): boolean {
  if (typeof window === 'undefined' || typeof window.gtag !== 'function') {
    return false;
  }

  window.gtag('event', eventName, params);
  return true;
}

export function trackSquareDaoClick(daoCode: string, targetUrl: string): void {
  const params = buildSquareDaoClickEventParams({
    daoCode,
    targetUrl,
    referrer: document.referrer,
    currentHost: window.location.hostname
  });
  const dedupeKey = `${SQUARE_DAO_CLICK_EVENT_NAME}:${params.dao_slug_or_public_id}`;

  try {
    if (window.sessionStorage.getItem(dedupeKey)) return;
    if (sendSquareAnalyticsEvent(SQUARE_DAO_CLICK_EVENT_NAME, params)) {
      window.sessionStorage.setItem(dedupeKey, '1');
    }
  } catch {
    sendSquareAnalyticsEvent(SQUARE_DAO_CLICK_EVENT_NAME, params);
  }
}
