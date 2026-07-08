// JSON feed cache for the SvelteKit SPA.
//
// This is the SvelteKit port of `internal/web/assets/svelte/src/feedCache.ts`.
// The legacy version cached HTML fragments returned by HTMX partials. The SPA
// consumes JSON from `GET /api/feed`, so this cache stores parsed JSON
// payloads keyed by the request URL (path + query).
//
// The invalidation contract is unchanged: mutations dispatch the
// `arabica:feed-mutation` event on `document.body`, which clears the cache
// and lets feed components refetch. The TTL is short (60s) to match the
// legacy behavior.

type FeedCacheEnvelope = {
  version: number;
  timestamp: number;
  did: string | null;
  app: string | null;
  url: string;
  json: unknown;
};

const CACHE_PREFIX = "arabica_feed_cache:";
const CACHE_VERSION = 2;
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

/** Returns the cached JSON payload for `url`, or `null` if missing/stale. */
export function getCachedFeedJSON<T = unknown>(url: string): T | null {
  try {
    const raw = sessionStorage.getItem(feedCacheKey(url));
    if (!raw) return null;
    const envelope = JSON.parse(raw) as FeedCacheEnvelope;
    if (!isEnvelopeValid(envelope, url)) {
      sessionStorage.removeItem(feedCacheKey(url));
      return null;
    }
    return envelope.json as T;
  } catch {
    return null;
  }
}

/** Stores a parsed JSON payload for `url`. No-ops on empty payloads. */
export function setCachedFeedJSON(url: string, json: unknown) {
  if (json === null || json === undefined) return;
  try {
    const normalized = new URL(url, window.location.origin);
    normalized.hash = "";
    const envelope: FeedCacheEnvelope = {
      version: CACHE_VERSION,
      timestamp: Date.now(),
      did: currentDID(),
      app: currentApp(),
      url: `${normalized.pathname}${normalized.search}`,
      json,
    };
    sessionStorage.setItem(feedCacheKey(url), JSON.stringify(envelope));
  } catch {
    // Ignore storage quota and privacy-mode failures.
  }
}

/** Removes every feed cache entry for the current app/user. */
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

/**
 * Clears the feed cache and dispatches `arabica:feed-mutation` on
 * `document.body`. Feed components listen for this event to refetch.
 */
export function dispatchFeedMutation(detail: FeedMutationDetail = {}) {
  clearFeedCache();
  document.body?.dispatchEvent(
    new CustomEvent<FeedMutationDetail>(FEED_MUTATION_EVENT, {
      bubbles: true,
      detail,
    }),
  );
}
