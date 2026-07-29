// Cache GET /api/feed JSON by app, viewer, and request URL for 60 seconds.
// Mutations dispatch `{app}:feed-mutation` on document.body to invalidate it.

import { feedCachePrefix, feedMutationEvent } from "./storageKeys";

type FeedCacheEnvelope = {
  version: number;
  timestamp: number;
  did: string | null;
  app: string | null;
  url: string;
  json: unknown;
};

const CACHE_VERSION = 2;
const CACHE_TTL_MS = 60 * 1000;
export const FEED_MUTATION_EVENT = feedMutationEvent();

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
  return `${feedCachePrefix()}${currentDID() || "anon"}:${normalized.pathname}${normalized.search}`;
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
      if (key?.startsWith(feedCachePrefix())) {
        sessionStorage.removeItem(key);
      }
    }
  } catch {
    // Ignore storage failures.
  }
}

/**
 * Clears the feed cache and dispatches `{app}:feed-mutation` on
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
