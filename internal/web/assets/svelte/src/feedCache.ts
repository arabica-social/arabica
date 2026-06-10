type FeedCacheEnvelope = {
  version: number;
  timestamp: number;
  did: string | null;
  app: string | null;
  url: string;
  html: string;
};

const CACHE_PREFIX = "arabica_feed_cache:";
const CACHE_VERSION = 1;
const CACHE_TTL_MS = 60 * 1000;
export const FEED_MUTATION_EVENT = "arabica:feed-mutation";

export type FeedMutationDetail = {
  source?: "comment" | "entity" | "unknown";
  action?: "create" | "delete" | "update" | "unknown";
  subjectURI?: string;
};

function currentDID() {
  return document.body?.dataset?.userDid || null;
}

function currentApp() {
  return document.body?.dataset?.app || null;
}

export function feedCacheKey(url: string) {
  const normalized = new URL(url, window.location.origin);
  normalized.hash = "";
  return `${CACHE_PREFIX}${currentApp() || "app"}:${currentDID() || "anon"}:${normalized.pathname}${normalized.search}`;
}

function isEnvelopeValid(envelope: FeedCacheEnvelope, url: string) {
  if (envelope.version !== CACHE_VERSION) return false;
  if (envelope.did !== currentDID()) return false;
  if (envelope.app !== currentApp()) return false;
  if (Date.now() - envelope.timestamp > CACHE_TTL_MS) return false;

  const expected = new URL(url, window.location.origin);
  expected.hash = "";
  return envelope.url === `${expected.pathname}${expected.search}`;
}

export function getCachedFeedHTML(url: string) {
  try {
    const raw = sessionStorage.getItem(feedCacheKey(url));
    if (!raw) return null;
    const envelope = JSON.parse(raw) as FeedCacheEnvelope;
    if (!isEnvelopeValid(envelope, url)) {
      sessionStorage.removeItem(feedCacheKey(url));
      return null;
    }
    return envelope.html;
  } catch {
    return null;
  }
}

export function setCachedFeedHTML(url: string, html: string) {
  if (!html) return;
  try {
    const normalized = new URL(url, window.location.origin);
    normalized.hash = "";
    const envelope: FeedCacheEnvelope = {
      version: CACHE_VERSION,
      timestamp: Date.now(),
      did: currentDID(),
      app: currentApp(),
      url: `${normalized.pathname}${normalized.search}`,
      html,
    };
    sessionStorage.setItem(feedCacheKey(url), JSON.stringify(envelope));
  } catch {
    // Ignore storage quota and privacy-mode failures.
  }
}

export function clearFeedCache() {
  try {
    for (let index = sessionStorage.length - 1; index >= 0; index -= 1) {
      const key = sessionStorage.key(index);
      if (key?.startsWith(CACHE_PREFIX)) {
        sessionStorage.removeItem(key);
      }
    }
  } catch {
    // Ignore storage failures.
  }
}

export function dispatchFeedMutation(detail: FeedMutationDetail = {}) {
  clearFeedCache();
  document.body?.dispatchEvent(
    new CustomEvent<FeedMutationDetail>(FEED_MUTATION_EVENT, {
      bubbles: true,
      detail,
    }),
  );
}
