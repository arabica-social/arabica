import { describe, expect, it, vi } from "vitest";
import { fetchEntityView } from "../../src/lib/api/entityView";
import type { EntityViewResponse } from "../../src/lib/types/entity_view";

function successResponse(record: Record<string, unknown>): Response {
	const payload: EntityViewResponse = {
		record,
		subject_uri: "at://did:example:abc/social.arabica.alpha.bean/r1",
		subject_cid: "cid123",
		author: { did: "did:example:abc", handle: "alice.bsky.social" },
		social: {
			is_liked: false,
			like_count: 0,
			comment_count: 0,
			comments: [],
			is_moderator: false,
			can_hide_record: false,
			can_block_user: false,
			is_record_hidden: false,
		},
		is_own_profile: true,
		is_authenticated: true,
		share_url: "/bean/alice.bsky.social/r1",
		entity_type: "bean",
		entity_count: 1,
	};
	return new Response(JSON.stringify(payload), {
		status: 200,
		headers: { "Content-Type": "application/json" },
	});
}

/** Builds a JSON error response mirroring the server error envelope. */
function errorResponse(status: number, message: string, code: string): Response {
	return new Response(JSON.stringify({ error: message, code }), {
		status,
		headers: { "Content-Type": "application/json" },
	});
}

describe("fetchEntityView", () => {
	it("returns {data, status:200} on success", async () => {
		const fetchFn = vi.fn().mockResolvedValue(successResponse({ rkey: "r1", name: "Heart" }));
		const result = await fetchEntityView(fetchFn as typeof fetch, "bean", "alice.bsky.social", "r1");

		expect(result.error).toBeUndefined();
		expect(result.status).toBe(200);
		expect(result.data?.record).toEqual({ rkey: "r1", name: "Heart" });
		expect(result.data?.entity_type).toBe("bean");
		expect(result.data?.share_url).toBe("/bean/alice.bsky.social/r1");
	});

	it("encodes the URL with entity/actor/id segments", async () => {
		const fetchFn = vi.fn().mockResolvedValue(successResponse({}));
		await fetchEntityView(fetchFn as typeof fetch, "bean", "alice.bsky.social", "r1");

		expect(fetchFn).toHaveBeenCalledWith(
			"/api/bean/alice.bsky.social/r1",
			expect.objectContaining({ credentials: "same-origin" }),
		);
	});

	it("percent-encodes special characters in actor and id", async () => {
		const fetchFn = vi.fn().mockResolvedValue(successResponse({}));
		// A handle won't contain slashes, but id could; ensure encodeURIComponent is applied.
		await fetchEntityView(fetchFn as typeof fetch, "bean", "alice.bsky.social", "weird/rkey");

		const url = fetchFn.mock.calls[0][0] as string;
		expect(url).toBe("/api/bean/alice.bsky.social/weird%2Frkey");
	});

	it("maps 404 to 'Record not found' with status 404", async () => {
		const fetchFn = vi.fn().mockResolvedValue(errorResponse(404, "not found", "not_found"));
		const result = await fetchEntityView(fetchFn as typeof fetch, "bean", "alice", "missing");

		expect(result.data).toBeUndefined();
		expect(result.error).toBe("Record not found");
		expect(result.status).toBe(404);
	});

	it("maps 401 to 'Authentication required' with status 401", async () => {
		const fetchFn = vi.fn().mockResolvedValue(errorResponse(401, "unauthorized", "authentication_required"));
		const result = await fetchEntityView(fetchFn as typeof fetch, "bean", "alice", "r1");

		expect(result.error).toBe("Authentication required");
		expect(result.status).toBe(401);
	});

	it("maps network failures (fetch rejection) to status 0", async () => {
		const fetchFn = vi.fn().mockRejectedValue(new TypeError("offline"));
		const result = await fetchEntityView(fetchFn as typeof fetch, "bean", "alice", "r1");

		expect(result.error).toBe("Network error");
		expect(result.status).toBe(0);
		expect(result.data).toBeUndefined();
	});

	it.each([
		[400, "bad request"],
		[403, "forbidden"],
		[500, "internal"],
		[503, "unavailable"],
	])("maps other HTTP %i to its status with the server message", async (status, message) => {
		const fetchFn = vi.fn().mockResolvedValue(errorResponse(status, message, "some_code"));
		const result = await fetchEntityView(fetchFn as typeof fetch, "bean", "alice", "r1");

		expect(result.status).toBe(status);
		expect(result.error).toBe(message);
	});

	it("falls back to a generic message when the server error envelope is empty", async () => {
		// Plain-text non-JSON error: requestJSON falls back to {error: text.trim()}.
		const fetchFn = vi.fn().mockResolvedValue(
			new Response("Something broke", { status: 500, headers: { "Content-Type": "text/plain" } }),
		);
		const result = await fetchEntityView(fetchFn as typeof fetch, "bean", "alice", "r1");

		expect(result.status).toBe(500);
		expect(result.error).toBe("Something broke");
	});

	it("never throws — catches generic non-APIError throws as network errors", async () => {
		const fetchFn = vi.fn().mockImplementation(() => {
			throw new Error("unexpected boom");
		});
		await expect(
			fetchEntityView(fetchFn as typeof fetch, "bean", "alice", "r1"),
		).resolves.toEqual({ error: "Network error", status: 0 });
	});

	it("treats invalid-response (malformed JSON) errors as generic HTTP errors", async () => {
		const fetchFn = vi.fn().mockResolvedValue(
			new Response("{broken", { status: 200, headers: { "Content-Type": "application/json" } }),
		);
		const result = await fetchEntityView(fetchFn as typeof fetch, "bean", "alice", "r1");

		expect(result.status).toBe(200);
		expect(result.error).toBe("Arabica returned malformed JSON.");
		expect(result.data).toBeUndefined();
	});
});
