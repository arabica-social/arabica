export type APIErrorKind = "http" | "network" | "invalid-response";

export type APIErrorEnvelope = {
	error?: string;
	code?: string;
	fields?: Record<string, string>;
};

export class APIError extends Error {
	status: number;
	code: string;
	fields?: Record<string, string>;
	kind: APIErrorKind;

	constructor(
		message: string,
		options: {
			status?: number;
			code?: string;
			fields?: Record<string, string>;
			kind: APIErrorKind;
		},
	) {
		super(message);
		this.name = "APIError";
		this.status = options.status ?? 0;
		this.code = options.code ?? "unknown_error";
		this.fields = options.fields;
		this.kind = options.kind;
	}
}

function notifySessionExpired(error: APIError) {
	if (error.status !== 401 || error.code !== "session_expired") return;
	if (typeof window !== "undefined") window.__showSessionExpiredModal?.();
}

function fallbackErrorCode(status: number): string {
	switch (status) {
		case 400:
			return "invalid_request";
		case 401:
			return "authentication_required";
		case 403:
			return "permission_denied";
		case 404:
			return "not_found";
		case 409:
			return "conflict";
		case 429:
			return "rate_limited";
		case 503:
			return "service_unavailable";
		default:
			return status >= 500 ? "internal_error" : `http_${status}`;
	}
}

export async function requestJSON<T>(
	fetchFn: typeof fetch,
	path: string,
	init: RequestInit = {},
): Promise<T> {
	const headers = new Headers(init.headers);
	if (!headers.has("Accept")) headers.set("Accept", "application/json");

	let response: Response;
	try {
		response = await fetchFn(path, {
			...init,
			headers,
			credentials: init.credentials ?? "same-origin",
		});
	} catch {
		throw new APIError("Unable to reach Arabica. Check your connection and try again.", {
			status: 0,
			code: "network_error",
			kind: "network",
		});
	}

	const text = await response.text();
	const contentType = response.headers.get("content-type") ?? "";
	let payload: unknown;
	if (text !== "") {
		if (!contentType.includes("application/json")) {
			if (response.ok) {
				throw new APIError("Arabica returned an unexpected response.", {
					status: response.status,
					code: "unexpected_content_type",
					kind: "invalid-response",
				});
			}
			payload = { error: text.trim() };
		} else {
			try {
				payload = JSON.parse(text);
			} catch {
				throw new APIError("Arabica returned malformed JSON.", {
					status: response.status,
					code: "invalid_json",
					kind: "invalid-response",
				});
			}
		}
	}

	if (!response.ok) {
		const envelope = (payload ?? {}) as APIErrorEnvelope;
		const error = new APIError(envelope.error || `Request failed (${response.status})`, {
			status: response.status,
			code: envelope.code || fallbackErrorCode(response.status),
			fields: envelope.fields,
			kind: "http",
		});
		notifySessionExpired(error);
		throw error;
	}

	return payload as T;
}

export function postJSON<T>(fetchFn: typeof fetch, path: string, body: unknown): Promise<T> {
	return requestJSON<T>(fetchFn, path, {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(body),
	});
}

export function putJSON<T>(fetchFn: typeof fetch, path: string, body: unknown): Promise<T> {
	return requestJSON<T>(fetchFn, path, {
		method: "PUT",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(body),
	});
}

export function deleteJSON<T = void>(fetchFn: typeof fetch, path: string): Promise<T> {
	return requestJSON<T>(fetchFn, path, { method: "DELETE" });
}

type FormBody = FormData | URLSearchParams;

export function createAPIClient(fetchFn: typeof fetch = globalThis.fetch) {
	return {
		request<T>(path: string, init: RequestInit = {}): Promise<T> {
			return requestJSON<T>(fetchFn, path, init);
		},
		get<T>(path: string): Promise<T> {
			return requestJSON<T>(fetchFn, path);
		},
		postForm<T>(path: string, form: FormBody): Promise<T> {
			return requestJSON<T>(fetchFn, path, { method: "POST", body: form });
		},
		putForm<T>(path: string, form: FormBody): Promise<T> {
			return requestJSON<T>(fetchFn, path, { method: "PUT", body: form });
		},
		postJSON<T>(path: string, body: unknown): Promise<T> {
			return postJSON<T>(fetchFn, path, body);
		},
		putJSON<T>(path: string, body: unknown): Promise<T> {
			return putJSON<T>(fetchFn, path, body);
		},
		deleteJSON<T = void>(path: string): Promise<T> {
			return deleteJSON<T>(fetchFn, path);
		},
	};
}
