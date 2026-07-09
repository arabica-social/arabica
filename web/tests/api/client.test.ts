import { describe, expect, it, vi } from "vitest";
import { APIError, createAPIClient } from "../../src/lib/api/client";

describe("API client", () => {
	it("decodes successful JSON responses with shared request defaults", async () => {
		const fetchFn = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ value: "ok" }), {
				status: 200,
				headers: { "Content-Type": "application/json; charset=utf-8" },
			}),
		);
		const client = createAPIClient(fetchFn as typeof fetch);

		await expect(client.get<{ value: string }>("/api/test")).resolves.toEqual({ value: "ok" });
		expect(fetchFn).toHaveBeenCalledWith(
			"/api/test",
			expect.objectContaining({
				credentials: "same-origin",
				headers: expect.any(Headers),
			}),
		);
		const options = fetchFn.mock.calls[0][1] as RequestInit;
		expect((options.headers as Headers).get("Accept")).toBe("application/json");
	});

	it("converts JSON error envelopes into typed API errors", async () => {
		const fetchFn = vi.fn().mockResolvedValue(
			new Response(
				JSON.stringify({
					error: "Please correct the highlighted fields.",
					code: "validation_failed",
					fields: { name: "Name is required" },
				}),
				{ status: 400, headers: { "Content-Type": "application/json" } },
			),
		);
		const client = createAPIClient(fetchFn as typeof fetch);

		const error = await client.get("/api/test").catch((caught) => caught);
		expect(error).toBeInstanceOf(APIError);
		expect(error).toMatchObject({
			message: "Please correct the highlighted fields.",
			status: 400,
			code: "validation_failed",
			kind: "http",
			fields: { name: "Name is required" },
		});
	});

	it("classifies malformed successful JSON as an invalid response", async () => {
		const fetchFn = vi.fn().mockResolvedValue(
			new Response("{broken", { status: 200, headers: { "Content-Type": "application/json" } }),
		);
		const client = createAPIClient(fetchFn as typeof fetch);

		await expect(client.get("/api/test")).rejects.toMatchObject({
			status: 200,
			code: "invalid_json",
			kind: "invalid-response",
		});
	});

	it("classifies successful non-JSON responses as invalid", async () => {
		const fetchFn = vi.fn().mockResolvedValue(
			new Response("<html>legacy</html>", { status: 200, headers: { "Content-Type": "text/html" } }),
		);
		const client = createAPIClient(fetchFn as typeof fetch);

		await expect(client.get("/api/test")).rejects.toMatchObject({
			status: 200,
			code: "unexpected_content_type",
			kind: "invalid-response",
		});
	});

	it("uses a safe plain-text fallback for legacy error responses", async () => {
		const fetchFn = vi.fn().mockResolvedValue(
			new Response("Failed to save", { status: 500, headers: { "Content-Type": "text/plain" } }),
		);
		const client = createAPIClient(fetchFn as typeof fetch);

		await expect(client.get("/api/test")).rejects.toMatchObject({
			message: "Failed to save",
			status: 500,
			code: "internal_error",
			kind: "http",
		});
	});

	it.each([
		[401, "authentication_required"],
		[403, "permission_denied"],
		[404, "not_found"],
		[409, "conflict"],
		[503, "service_unavailable"],
	])("maps legacy HTTP %s responses to stable fallback codes", async (status, code) => {
		const fetchFn = vi.fn().mockResolvedValue(
			new Response("Request failed", { status, headers: { "Content-Type": "text/plain" } }),
		);
		const client = createAPIClient(fetchFn as typeof fetch);

		await expect(client.get("/api/test")).rejects.toMatchObject({ status, code, kind: "http" });
	});

	it("classifies fetch rejection as a network error", async () => {
		const fetchFn = vi.fn().mockRejectedValue(new TypeError("offline"));
		const client = createAPIClient(fetchFn as typeof fetch);

		await expect(client.get("/api/test")).rejects.toMatchObject({
			status: 0,
			code: "network_error",
			kind: "network",
		});
	});

	it("signals the shared session-expiry UI for session_expired responses", async () => {
		const showSessionExpired = vi.fn();
		window.__showSessionExpiredModal = showSessionExpired;
		const fetchFn = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ error: "Session expired", code: "session_expired" }), {
				status: 401,
				headers: { "Content-Type": "application/json" },
			}),
		);
		const client = createAPIClient(fetchFn as typeof fetch);

		await expect(client.get("/api/test")).rejects.toMatchObject({ status: 401, code: "session_expired" });
		expect(showSessionExpired).toHaveBeenCalledOnce();
		delete window.__showSessionExpiredModal;
	});

	it("encodes form requests without overriding the browser content type", async () => {
		const fetchFn = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ saved: true }), {
				status: 200,
				headers: { "Content-Type": "application/json" },
			}),
		);
		const client = createAPIClient(fetchFn as typeof fetch);
		const form = new FormData();
		form.set("name", "Heart");

		await client.postForm("/api/roasters", form);

		const options = fetchFn.mock.calls[0][1] as RequestInit;
		expect(options.method).toBe("POST");
		expect(options.body).toBe(form);
		expect((options.headers as Headers).has("Content-Type")).toBe(false);
	});

	it("supports PUT form mutations", async () => {
		const fetchFn = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ saved: true }), {
				status: 200,
				headers: { "Content-Type": "application/json" },
			}),
		);
		const client = createAPIClient(fetchFn as typeof fetch);
		const form = new URLSearchParams({ name: "Heart" });

		await client.putForm("/api/roasters/r1", form);

		expect(fetchFn).toHaveBeenCalledWith(
			"/api/roasters/r1",
			expect.objectContaining({ method: "PUT", body: form }),
		);
	});

	it("supports empty 204 success responses", async () => {
		const fetchFn = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		const client = createAPIClient(fetchFn as typeof fetch);

		await expect(client.deleteJSON("/api/test")).resolves.toBeUndefined();
	});
});
