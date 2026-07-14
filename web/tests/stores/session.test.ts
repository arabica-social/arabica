import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { get } from "svelte/store";
import {
	app,
	checkSessionExpired,
	displayHandle,
	formatNotificationCount,
	profileIdentifier,
	refreshSession,
	safeAvatarURL,
	session,
	warnIfSessionExpired,
} from "../../src/lib/stores/session";

const DATASET_KEYS = [
	"userDid",
	"userHandle",
	"userDisplay",
	"userAvatar",
	"isModerator",
	"unreadNotifications",
	"temperatureUnit",
	"app",
] as const;

function setBodyDataset(overrides: Partial<Record<(typeof DATASET_KEYS)[number], string>>) {
	const body = document.body;
	for (const key of DATASET_KEYS) {
		delete body.dataset[key];
	}
	for (const [key, value] of Object.entries(overrides)) {
		body.dataset[key] = value;
	}
}

describe("session store", () => {
	beforeEach(() => {
		setBodyDataset({});
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue({
				ok: true,
				json: async () => ({
					is_authenticated: false,
					temperature_unit: "recorded",
					app: "arabica",
				}),
			}),
		);
	});

	afterEach(() => {
		setBodyDataset({});
		vi.unstubAllGlobals();
	});

	describe("refreshSession", () => {
		it("reads an authenticated session from body dataset", async () => {
			setBodyDataset({
				userDid: "did:plc:alice",
				userHandle: "alice.test",
				userDisplay: "Alice",
				userAvatar: "https://cdn.bsky.app/img/avatar/feed.jpg",
				isModerator: "true",
				unreadNotifications: "3",
				temperatureUnit: "celsius",
				app: "arabica",
			});
			await refreshSession();

			expect(get(session)).toEqual({
				did: "did:plc:alice",
				handle: "alice.test",
				displayName: "Alice",
				avatar: "https://cdn.bsky.app/img/avatar/feed.jpg",
				isAuthenticated: true,
				isModerator: true,
				unreadNotifications: 3,
				temperatureUnit: "celsius",
			});
			expect(get(app)).toBe("arabica");
		});

		it("loads an unauthenticated session from the development API when body data is absent", async () => {
			await refreshSession();

			expect(get(session)).toEqual({
				did: "",
				handle: "",
				displayName: "",
				avatar: "",
				isAuthenticated: false,
				isModerator: false,
				unreadNotifications: 0,
				temperatureUnit: "recorded",
			});
		});

		it("reflects app oolong when set on body", async () => {
			setBodyDataset({ app: "oolong" });
			await refreshSession();
			expect(get(app)).toBe("oolong");
		});

		it("treats isModerator !== 'true' as false", async () => {
			setBodyDataset({ userDid: "did:plc:x", isModerator: "false" });
			await refreshSession();
			expect(get(session).isModerator).toBe(false);
		});

		it("coerces unreadNotifications to a number, falling back to 0 on NaN", async () => {
			setBodyDataset({ userDid: "did:plc:x", unreadNotifications: "not-a-number" });
			await refreshSession();
			expect(get(session).unreadNotifications).toBe(0);
		});

		it("preserves last-read session until refreshSession is called again", async () => {
			setBodyDataset({ userDid: "did:plc:alice" });
			await refreshSession();
			expect(get(session).did).toBe("did:plc:alice");

			// mutate body without refresh; store is stale by design
			setBodyDataset({ userDid: "did:plc:bob" });
			expect(get(session).did).toBe("did:plc:alice");

			await refreshSession();
			expect(get(session).did).toBe("did:plc:bob");
		});
	});

	describe("profileIdentifier", () => {
		it("prefers the handle when set", async () => {
			setBodyDataset({ userDid: "did:plc:alice", userHandle: "alice.test" });
			await refreshSession();
			expect(profileIdentifier(get(session))).toBe("alice.test");
		});

		it("falls back to the DID when handle is empty", async () => {
			setBodyDataset({ userDid: "did:plc:alice" });
			await refreshSession();
			expect(profileIdentifier(get(session))).toBe("did:plc:alice");
		});

		it("falls back to empty string for an unauthenticated session", async () => {
			await refreshSession();
			expect(profileIdentifier(get(session))).toBe("");
		});

		it("accepts an explicit session argument without reading the store", () => {
			const explicit = {
				did: "did:plc:carol",
				handle: "carol.test",
				displayName: "Carol",
				avatar: "",
				isAuthenticated: true,
				isModerator: false,
				unreadNotifications: 0,
				temperatureUnit: "recorded",
			};
			expect(profileIdentifier(explicit)).toBe("carol.test");
		});
	});

	describe("safeAvatarURL", () => {
		it("returns empty string for empty input", () => {
			expect(safeAvatarURL("")).toBe("");
		});

		it.each([
			"https://cdn.bsky.app/img/avatar/feed.jpg",
			"https://av-cdn.bsky.app/img/avatar/alice.jpg",
			"https://sub.cdn.bsky.app/avatar.png",
			"https://deep.sub.av-cdn.bsky.app/x.png",
		])("allows trusted HTTPS CDN URL %s", (url) => {
			expect(safeAvatarURL(url)).toBe(url);
		});

		it("allows /static/ paths", () => {
			expect(safeAvatarURL("/static/avatars/default.png")).toBe(
				"/static/avatars/default.png",
			);
		});

		it.each([
			["bare root path", "/"],
			["non-static absolute path", "/api/data"],
			["other absolute path", "/avatars/x.png"],
		])("rejects %s", (_label, url) => {
			expect(safeAvatarURL(url)).toBe("");
		});

		it.each([
			["http CDN url (wrong protocol)", "http://cdn.bsky.app/x.png"],
			["ftp url", "ftp://cdn.bsky.app/x.png"],
			["untrusted domain", "https://evil.example.com/x.png"],
			["lookalike domain", "https://cdn.bsky.app.evil.com/x.png"],
			["malformed url", "not-a-url"],
		])("rejects %s", (_label, url) => {
			expect(safeAvatarURL(url)).toBe("");
		});

		it("is case-insensitive on host", () => {
			expect(safeAvatarURL("https://CDN.BSKY.APP/x.png")).toBe(
				"https://CDN.BSKY.APP/x.png",
			);
		});
	});

	describe("displayHandle", () => {
		it("returns empty string for empty input", () => {
			expect(displayHandle("")).toBe("");
		});

		it("strips a leading @", () => {
			expect(displayHandle("@alice.test")).toBe("alice.test");
		});

		it("leaves a handle without @ unchanged", () => {
			expect(displayHandle("bob.test")).toBe("bob.test");
		});

		it("strips only one leading @", () => {
			expect(displayHandle("@@double.test")).toBe("@double.test");
		});
	});

	describe("formatNotificationCount", () => {
		it.each([
			["0", 0, "0"],
			["1", 1, "1"],
			["single digit", 5, "5"],
			["double digit", 42, "42"],
			["boundary 99", 99, "99"],
		])("returns the count string for %s", (_label, count, expected) => {
			expect(formatNotificationCount(count)).toBe(expected);
		});

		it("returns '99+' for counts greater than 99", () => {
			expect(formatNotificationCount(100)).toBe("99+");
			expect(formatNotificationCount(9999)).toBe("99+");
		});
	});

	describe("checkSessionExpired", () => {
		it("returns true when the status endpoint reports session_expired", async () => {
			vi.stubGlobal(
				"fetch",
				vi.fn().mockResolvedValue({
					ok: true,
					json: async () => ({ is_authenticated: false, session_expired: true }),
				}),
			);

			await expect(checkSessionExpired()).resolves.toBe(true);
		});

		it("returns false when the session is authenticated", async () => {
			vi.stubGlobal(
				"fetch",
				vi.fn().mockResolvedValue({
					ok: true,
					json: async () => ({ is_authenticated: true, session_expired: false }),
				}),
			);

			await expect(checkSessionExpired()).resolves.toBe(false);
		});

		it("returns false when the endpoint is unreachable", async () => {
			vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("offline")));

			await expect(checkSessionExpired()).resolves.toBe(false);
		});

		it("returns false on a non-OK response", async () => {
			vi.stubGlobal(
				"fetch",
				vi.fn().mockResolvedValue({ ok: false, status: 500 }),
			);

			await expect(checkSessionExpired()).resolves.toBe(false);
		});
	});

	describe("warnIfSessionExpired", () => {
		it("opens the session-expired modal when the session is expired", async () => {
			vi.stubGlobal(
				"fetch",
				vi.fn().mockResolvedValue({
					ok: true,
					json: async () => ({ is_authenticated: false, session_expired: true }),
				}),
			);
			const showModal = vi.fn();
			window.__showSessionExpiredModal = showModal;

			await warnIfSessionExpired();

			expect(showModal).toHaveBeenCalledOnce();
			delete window.__showSessionExpiredModal;
		});

		it("does not open the modal when the session is healthy", async () => {
			vi.stubGlobal(
				"fetch",
				vi.fn().mockResolvedValue({
					ok: true,
					json: async () => ({ is_authenticated: true, session_expired: false }),
				}),
			);
			const showModal = vi.fn();
			window.__showSessionExpiredModal = showModal;

			await warnIfSessionExpired();

			expect(showModal).not.toHaveBeenCalled();
			delete window.__showSessionExpiredModal;
		});
	});
});
