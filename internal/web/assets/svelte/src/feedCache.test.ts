import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  FEED_MUTATION_EVENT,
  clearFeedCache,
  dispatchFeedMutation,
  feedCacheKey,
  getCachedFeedHTML,
  setCachedFeedHTML,
} from "./feedCache";

describe("feedCache", () => {
  beforeEach(() => {
    document.body.dataset.userDid = "did:plc:alice";
    document.body.dataset.app = "arabica";
    sessionStorage.clear();
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-06-10T12:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
    document.body.innerHTML = "";
    sessionStorage.clear();
  });

  it("stores feed html per normalized URL, app, and user", () => {
    setCachedFeedHTML(
      "/api/feed?type=brew#ignored",
      '<div id="feed-items">Brews</div>',
    );

    expect(getCachedFeedHTML("/api/feed?type=brew")).toContain("Brews");
    expect(
      sessionStorage.getItem(feedCacheKey("/api/feed?type=brew")),
    ).toContain("Brews");
  });

  it("rejects stale or cross-user entries", () => {
    setCachedFeedHTML("/api/feed", '<div id="feed-items">All</div>');
    document.body.dataset.userDid = "did:plc:bob";

    expect(getCachedFeedHTML("/api/feed")).toBeNull();

    document.body.dataset.userDid = "did:plc:alice";
    setCachedFeedHTML("/api/feed", '<div id="feed-items">All</div>');
    vi.advanceTimersByTime(61_000);

    expect(getCachedFeedHTML("/api/feed")).toBeNull();
  });

  it("clears only feed cache entries", () => {
    setCachedFeedHTML("/api/feed", '<div id="feed-items">All</div>');
    sessionStorage.setItem("other", "keep");

    clearFeedCache();

    expect(getCachedFeedHTML("/api/feed")).toBeNull();
    expect(sessionStorage.getItem("other")).toBe("keep");
  });

  it("dispatches feed mutation events after clearing cached feed html", () => {
    const listener = vi.fn();
    document.body.addEventListener(FEED_MUTATION_EVENT, listener);
    setCachedFeedHTML("/api/feed", '<div id="feed-items">All</div>');

    dispatchFeedMutation({ source: "comment", action: "create" });

    expect(getCachedFeedHTML("/api/feed")).toBeNull();
    expect(listener).toHaveBeenCalledTimes(1);
    expect(listener.mock.calls[0][0]).toMatchObject({
      detail: { source: "comment", action: "create" },
    });
  });
});
