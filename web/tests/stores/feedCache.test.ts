import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
	FEED_MUTATION_EVENT,
	clearFeedCache,
	dispatchFeedMutation,
	feedCacheKey,
	getCachedFeedJSON,
	setCachedFeedJSON,
} from "../../src/lib/stores/feedCache";

const CACHE_PREFIX = "arabica_feed_cache:";

function setBody(did: string, app: string) {
	document.body.dataset.userDid = did;
	document.body.dataset.app = app;
}

function clearBody() {
	delete document.body.dataset.userDid;
	delete document.body.dataset.app;
}

describe("feedCache store", () => {
	beforeEach(() => {
		sessionStorage.clear();
		clearBody();
		setBody("did:plc:alice", "arabica");
	});

	afterEach(() => {
		sessionStorage.clear();
		clearBody();
		vi.useRealTimers();
	});

	describe("feedCacheKey", () => {
		it("produces the {app}_feed_cache:{did}:{path}{search} format", () => {
			expect(feedCacheKey("/api/feed")).toBe(
				`${CACHE_PREFIX}did:plc:alice:/api/feed`,
			);
		});

		it("includes the query string", () => {
			expect(feedCacheKey("/api/feed?cursor=abc&limit=20")).toBe(
				`${CACHE_PREFIX}did:plc:alice:/api/feed?cursor=abc&limit=20`,
			);
		});

		it("strips the hash fragment", () => {
			expect(feedCacheKey("/api/feed#section")).toBe(
				`${CACHE_PREFIX}did:plc:alice:/api/feed`,
			);
			expect(feedCacheKey("/api/feed?x=1#frag")).toBe(
				`${CACHE_PREFIX}did:plc:alice:/api/feed?x=1`,
			);
		});

		it("falls back to default app and 'anon' when body dataset is unset", () => {
			clearBody();
			expect(feedCacheKey("/api/feed")).toBe(`${CACHE_PREFIX}anon:/api/feed`);
		});

		it("normalizes a bare relative URL against window.location.origin", () => {
			// relative URL resolves against origin; pathname/search preserved
			expect(feedCacheKey("feed")).toBe(`${CACHE_PREFIX}did:plc:alice:/feed`);
		});
	});

	describe("set / get round-trip", () => {
		it("stores and returns a JSON payload", () => {
			const payload = { items: [{ id: "brew-1" }], cursor: "next" };
			setCachedFeedJSON("/api/feed", payload);

			expect(getCachedFeedJSON("/api/feed")).toEqual(payload);
		});

		it("returns null when no entry exists", () => {
			expect(getCachedFeedJSON("/api/feed")).toBeNull();
		});

		it("round-trips a payload with a query string", () => {
			const payload = { items: [] };
			setCachedFeedJSON("/api/feed?cursor=abc", payload);
			expect(getCachedFeedJSON("/api/feed?cursor=abc")).toEqual(payload);
		});

		it("does not store null or undefined payloads", () => {
			setCachedFeedJSON("/api/feed", null);
			setCachedFeedJSON("/api/feed", undefined);
			expect(getCachedFeedJSON("/api/feed")).toBeNull();
			expect(sessionStorage.length).toBe(0);
		});
	});

	describe("staleness — invalidation", () => {
		it("returns null and removes the entry when DID changed", () => {
			setCachedFeedJSON("/api/feed", { items: [] });
			setBody("did:plc:bob", "arabica");

			expect(getCachedFeedJSON("/api/feed")).toBeNull();
			// entry under the new DID key never existed
			expect(sessionStorage.getItem(feedCacheKey("/api/feed"))).toBeNull();
		});

		it("returns null and removes the entry when app changed", () => {
			setCachedFeedJSON("/api/feed", { items: [] });
			setBody("did:plc:alice", "other");

			expect(getCachedFeedJSON("/api/feed")).toBeNull();
			expect(sessionStorage.getItem(feedCacheKey("/api/feed"))).toBeNull();
		});

		it("returns null and removes the entry when the version is wrong", () => {
			setCachedFeedJSON("/api/feed", { items: [] });
			// mutate the stored envelope's version in place
			const key = feedCacheKey("/api/feed");
			const envelope = JSON.parse(sessionStorage.getItem(key) as string);
			envelope.version = 999;
			sessionStorage.setItem(key, JSON.stringify(envelope));

			expect(getCachedFeedJSON("/api/feed")).toBeNull();
			expect(sessionStorage.getItem(key)).toBeNull();
		});

		it("returns null and removes the entry after the TTL expires", () => {
			vi.useFakeTimers();
			vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
			setCachedFeedJSON("/api/feed", { items: [] });

			vi.advanceTimersByTime(60 * 1000 - 1);
			expect(getCachedFeedJSON("/api/feed")).toEqual({ items: [] });

			vi.advanceTimersByTime(2);
			expect(getCachedFeedJSON("/api/feed")).toBeNull();
			expect(sessionStorage.getItem(feedCacheKey("/api/feed"))).toBeNull();
		});

		it("returns null when the stored url no longer matches", () => {
			setCachedFeedJSON("/api/feed", { items: [] });
			const key = feedCacheKey("/api/feed");
			const envelope = JSON.parse(sessionStorage.getItem(key) as string);
			envelope.url = "/api/feed?hijack=1";
			sessionStorage.setItem(key, JSON.stringify(envelope));

			expect(getCachedFeedJSON("/api/feed")).toBeNull();
		});
	});

	describe("clearFeedCache", () => {
		it("removes only feed-prefixed keys", () => {
			setCachedFeedJSON("/api/feed", { a: 1 });
			setCachedFeedJSON("/api/feed?cursor=2", { b: 2 });
			sessionStorage.setItem("arabica_data_cache", JSON.stringify({ keep: true }));
			sessionStorage.setItem("unrelated_key", "stay");

			clearFeedCache();

			expect(sessionStorage.getItem("arabica_data_cache")).not.toBeNull();
			expect(sessionStorage.getItem("unrelated_key")).toBe("stay");
			expect(sessionStorage.getItem(feedCacheKey("/api/feed"))).toBeNull();
			expect(sessionStorage.getItem(feedCacheKey("/api/feed?cursor=2"))).toBeNull();
		});
	});

	describe("dispatchFeedMutation", () => {
		it("clears the feed cache", () => {
			setCachedFeedJSON("/api/feed", { items: [] });
			dispatchFeedMutation();
			expect(getCachedFeedJSON("/api/feed")).toBeNull();
		});

		it("dispatches {app}:feed-mutation CustomEvent on document.body", () => {
			const handler = vi.fn();
			document.body.addEventListener(FEED_MUTATION_EVENT, handler);

			dispatchFeedMutation({ source: "comment", action: "create", subjectURI: "at://x/y/z" });

			expect(handler).toHaveBeenCalledTimes(1);
			const event = handler.mock.calls[0][0] as CustomEvent;
			expect(event).toBeInstanceOf(CustomEvent);
			expect(event.bubbles).toBe(true);
			expect(event.detail).toEqual({
				source: "comment",
				action: "create",
				subjectURI: "at://x/y/z",
			});

			document.body.removeEventListener(FEED_MUTATION_EVENT, handler);
		});

		it("defaults to an empty detail object", () => {
			const handler = vi.fn();
			document.body.addEventListener(FEED_MUTATION_EVENT, handler);

			dispatchFeedMutation();

			const event = handler.mock.calls[0][0] as CustomEvent;
			expect(event.detail).toEqual({});

			document.body.removeEventListener(FEED_MUTATION_EVENT, handler);
		});
	});
});
