import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { get } from "svelte/store";
import {
	appCache,
	appCacheData,
	appCacheError,
	appCacheLoading,
	installAppCacheGlobal,
} from "../../src/lib/stores/appCache";
import { dataCacheKey } from "../../src/lib/stores/storageKeys";

const CACHE_VERSION = 1;

type Envelope = {
	version: number;
	timestamp: number;
	did: string | null;
	app: string | null;
	data: Record<string, unknown>;
};

function writeCache(envelope: Partial<Envelope> & { data?: Record<string, unknown> }) {
	localStorage.setItem(
		dataCacheKey(),
		JSON.stringify({
			version: envelope.version ?? CACHE_VERSION,
			timestamp: envelope.timestamp ?? Date.now(),
			did: envelope.did ?? null,
			app: envelope.app ?? null,
			data: envelope.data ?? {},
		} satisfies Envelope),
	);
}

function okResponse(data: Record<string, unknown>, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		headers: { "Content-Type": "application/json" },
	});
}

function setBody(did: string, app: string) {
	document.body.dataset.userDid = did;
	document.body.dataset.app = app;
}

function clearBody() {
	delete document.body.dataset.userDid;
	delete document.body.dataset.app;
}

describe("appCache store", () => {
	beforeEach(() => {
		localStorage.clear();
		clearBody();
		vi.useFakeTimers();
		vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
	});

	afterEach(() => {
		vi.useRealTimers();
		vi.unstubAllGlobals();
		clearBody();
		appCache.invalidateCache();
	});

	describe("getData — cache miss", () => {
		it("fetches /api/data when no cache exists", async () => {
			setBody("did:plc:alice", "arabica");
			const data = { did: "did:plc:alice", beans: [{ rkey: "b1" }] };
			const fetchFn = vi.fn().mockResolvedValue(okResponse(data));
			vi.stubGlobal("fetch", fetchFn);

			const result = await appCache.getData();

			expect(fetchFn).toHaveBeenCalledTimes(1);
			expect(fetchFn).toHaveBeenCalledWith(
				"/api/data",
				expect.objectContaining({ credentials: "same-origin" }),
			);
			expect(result).toEqual(data);
			expect(get(appCacheData)).toEqual(data);
			expect(get(appCacheLoading)).toBe(false);
			expect(get(appCacheError)).toBeNull();
		});

		it("sends the current pathname as X-Page-Context header", async () => {
			setBody("did:plc:alice", "arabica");
			const fetchFn = vi.fn().mockResolvedValue(okResponse({ ok: true }));
			vi.stubGlobal("fetch", fetchFn);

			await appCache.getData();

			const options = fetchFn.mock.calls[0][1] as RequestInit;
			expect((options.headers as Record<string, string>)["X-Page-Context"]).toBe(
				window.location.pathname,
			);
		});

		it("throws and sets appCacheError when fetch returns non-ok", async () => {
			setBody("did:plc:alice", "arabica");
			const fetchFn = vi.fn().mockResolvedValue(okResponse({ error: "boom" }, 500));
			vi.stubGlobal("fetch", fetchFn);

			await expect(appCache.getData()).rejects.toThrow("Failed to fetch data: 500");
			expect(get(appCacheError)).toBe("Failed to fetch data: 500");
			expect(get(appCacheLoading)).toBe(false);
		});

		it("persists the fetched data to localStorage", async () => {
			setBody("did:plc:alice", "arabica");
			const data = { did: "did:plc:alice", beans: [] };
			vi.stubGlobal("fetch", vi.fn().mockResolvedValue(okResponse(data)));

			await appCache.getData();

			const raw = localStorage.getItem(dataCacheKey());
			expect(raw).not.toBeNull();
			const stored = JSON.parse(raw as string) as Envelope;
			expect(stored.version).toBe(CACHE_VERSION);
			expect(stored.did).toBe("did:plc:alice");
			expect(stored.app).toBe("arabica");
			expect(stored.data).toEqual(data);
		});
	});

	describe("getData — cache hit within TTL", () => {
		it("does not fetch when a fresh valid cache exists", async () => {
			setBody("did:plc:alice", "arabica");
			writeCache({
				did: "did:plc:alice",
				app: "arabica",
				data: { did: "did:plc:alice", cached: true },
			});
			const fetchFn = vi.fn();
			vi.stubGlobal("fetch", fetchFn);

			const result = await appCache.getData();

			expect(fetchFn).not.toHaveBeenCalled();
			expect(result).toEqual({ did: "did:plc:alice", cached: true });
			expect(get(appCacheData)).toEqual({ did: "did:plc:alice", cached: true });
		});
	});

	describe("DID mismatch", () => {
		it("treats the cache as invalid when the current DID differs", async () => {
			setBody("did:plc:bob", "arabica");
			writeCache({
				did: "did:plc:alice",
				app: "arabica",
				data: { did: "did:plc:alice" },
			});
			const fetchFn = vi.fn().mockResolvedValue(okResponse({ did: "did:plc:bob" }));
			vi.stubGlobal("fetch", fetchFn);

			await appCache.getData();

			expect(fetchFn).toHaveBeenCalledTimes(1);
		});

		it("getCachedData returns null and removes the stale entry", () => {
			setBody("did:plc:bob", "arabica");
			writeCache({
				did: "did:plc:alice",
				app: "arabica",
				data: { did: "did:plc:alice" },
			});

			expect(appCache.getCachedData()).toBeNull();
			expect(localStorage.getItem(dataCacheKey())).toBeNull();
		});
	});

	describe("app mismatch", () => {
		it("treats the cache as invalid when the current app differs", async () => {
			setBody("did:plc:alice", "other");
			writeCache({
				did: "did:plc:alice",
				app: "arabica",
				data: { did: "did:plc:alice" },
			});
			const fetchFn = vi.fn().mockResolvedValue(okResponse({ did: "did:plc:alice" }));
			vi.stubGlobal("fetch", fetchFn);

			await appCache.getData();

			expect(fetchFn).toHaveBeenCalledTimes(1);
		});
	});

	describe("TTL expiry", () => {
		it("refetches after the 5-minute TTL expires", async () => {
			setBody("did:plc:alice", "arabica");
			writeCache({
				did: "did:plc:alice",
				app: "arabica",
				data: { did: "did:plc:alice", stale: true },
			});
			const fetchFn = vi.fn().mockResolvedValue(okResponse({ did: "did:plc:alice", fresh: true }));
			vi.stubGlobal("fetch", fetchFn);

			vi.advanceTimersByTime(5 * 60 * 1000 - 1);
			await appCache.getData();
			expect(fetchFn).not.toHaveBeenCalled();

			vi.advanceTimersByTime(2);
			await appCache.getData();
			expect(fetchFn).toHaveBeenCalledTimes(1);
		});
	});

	describe("invalidateCache", () => {
		it("removes the cache entry from localStorage and clears the store", async () => {
			setBody("did:plc:alice", "arabica");
			vi.stubGlobal("fetch", vi.fn().mockResolvedValue(okResponse({ did: "did:plc:alice" })));
			await appCache.getData();
			expect(localStorage.getItem(dataCacheKey())).not.toBeNull();
			expect(get(appCacheData)).not.toBeNull();

			appCache.invalidateCache();

			expect(localStorage.getItem(dataCacheKey())).toBeNull();
			expect(get(appCacheData)).toBeNull();
		});
	});

	describe("invalidateAndRefresh", () => {
		it("clears storage then fetches fresh data", async () => {
			setBody("did:plc:alice", "arabica");
			writeCache({
				did: "did:plc:alice",
				app: "arabica",
				data: { did: "did:plc:alice", old: true },
			});
			const fetchFn = vi
				.fn()
				.mockResolvedValue(okResponse({ did: "did:plc:alice", fresh: true }));
			vi.stubGlobal("fetch", fetchFn);

			const result = await appCache.invalidateAndRefresh();

			expect(fetchFn).toHaveBeenCalledTimes(1);
			expect(result).toEqual({ did: "did:plc:alice", fresh: true });
			const stored = JSON.parse(localStorage.getItem(dataCacheKey()) as string) as Envelope;
			expect(stored.data).toEqual({ did: "did:plc:alice", fresh: true });
		});
	});

	describe("refresh coalescing", () => {
		it("does not double-fetch when refreshCache is called concurrently", async () => {
			setBody("did:plc:alice", "arabica");
			const fetchFn = vi.fn().mockResolvedValue(okResponse({ did: "did:plc:alice" }));
			vi.stubGlobal("fetch", fetchFn);

			const [a, b] = await Promise.all([
				appCache.refreshCache(),
				appCache.refreshCache(),
			]);

			expect(fetchFn).toHaveBeenCalledTimes(1);
			expect(a).toEqual({ did: "did:plc:alice" });
			expect(b).toEqual({ did: "did:plc:alice" });
		});
	});

	describe("isCacheValid", () => {
		it("returns false when no cache exists", () => {
			setBody("did:plc:alice", "arabica");
			expect(appCache.isCacheValid()).toBe(false);
		});

		it("returns true for a fresh matching cache", () => {
			setBody("did:plc:alice", "arabica");
			writeCache({ did: "did:plc:alice", app: "arabica", data: { did: "did:plc:alice" } });
			expect(appCache.isCacheValid()).toBe(true);
		});

		it("returns false for a wrong-version cache", () => {
			setBody("did:plc:alice", "arabica");
			writeCache({ version: 999, did: "did:plc:alice", app: "arabica", data: {} });
			// getCache drops it on version mismatch; isCacheValid() reads through getCache
			expect(appCache.isCacheValid()).toBe(false);
			expect(localStorage.getItem(dataCacheKey())).toBeNull();
		});
	});

	describe("installAppCacheGlobal", () => {
		it("installs the singleton onto window.AppCache", () => {
			installAppCacheGlobal();
			expect((window as unknown as { AppCache: unknown }).AppCache).toBe(appCache);
		});

		it("is idempotent", () => {
			installAppCacheGlobal();
			const first = (window as unknown as { AppCache: unknown }).AppCache;
			installAppCacheGlobal();
			expect((window as unknown as { AppCache: unknown }).AppCache).toBe(first);
		});
	});

	describe("listeners", () => {
		it("notifies registered listeners after a successful refresh", async () => {
			setBody("did:plc:alice", "arabica");
			const data = { did: "did:plc:alice", beans: [{ rkey: "b1" }] };
			// Response body can only be consumed once, so return a fresh one per call.
			const fetchFn = vi.fn(() =>
				Promise.resolve(okResponse(data)),
			);
			vi.stubGlobal("fetch", fetchFn);
			const cb = vi.fn();
			appCache.addListener(cb);

			await appCache.refreshCache();

			expect(cb).toHaveBeenCalledWith(data);

			appCache.removeListener(cb);
			await appCache.refreshCache();
			expect(cb).toHaveBeenCalledTimes(1);
		});
	});
});
