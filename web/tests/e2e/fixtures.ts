import { test as base, expect, type Page, type APIRequestContext } from "@playwright/test";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

/**
 * Read the server URL from the file written by the Go e2e-server binary.
 */
function readServerURL(): string {
	const urlFile = resolve(__dirname, "../../../tests/e2e/.server-url");
	try {
		return readFileSync(urlFile, "utf-8").trim();
	} catch {
		// File doesn't exist.
	}
	return process.env.ARABICA_E2E_BASE_URL ?? "http://127.0.0.1:8080";
}

/**
 * Read the primary test account DID from the file written by the Go
 * e2e-server binary.
 */
function readServerDID(): string {
	const didFile = resolve(__dirname, "../../../tests/e2e/.server-did");
	try {
		return readFileSync(didFile, "utf-8").trim();
	} catch {
		// File doesn't exist.
	}
	return process.env.ARABICA_E2E_DID ?? "";
}

/**
 * Test fixtures providing an authenticated browser page and API request
 * context.
 *
 * The Go test harness authenticates requests via X-Test-Auth-DID and
 * X-Test-Auth-Session headers (bypassing OAuth). We inject these headers
 * in three places:
 *
 * 1. Browser context extraHTTPHeaders — applies to all browser navigation
 *    and fetch() calls from the SPA.
 * 2. Context route interception — catches any requests that don't pick up
 *    the context headers (e.g. cross-origin or redirected requests).
 * 3. apiRequest fixture — a Playwright APIRequestContext with the auth
 *    headers set, for making API calls directly from tests (page.request
 *    doesn't inherit the browser context's extraHTTPHeaders).
 *
 * Usage in a spec:
 *
 *   import { test, expect } from "./fixtures";
 *   test("create brew", async ({ authedPage: page, apiRequest, did }) => {
 *     // Navigate in the browser:
 *     await page.goto("/");
 *     // Make an authenticated API call:
 *     const resp = await apiRequest.post("/api/roasters", { form: {...} });
 *   });
 */
export const test = base.extend<{
	authedPage: Page;
	apiRequest: APIRequestContext;
	did: string;
}>({
	authedPage: async ({ browser }, use) => {
		const baseURL = readServerURL();
		const did = readServerDID();

		if (!did) {
			throw new Error(
				"No primary DID found. Is the e2e-server running? " +
					"Expected tests/e2e/.server-did file.",
			);
		}

		const authHeaders = {
			"X-Test-Auth-DID": did,
			"X-Test-Auth-Session": `test-session-${did}`,
			Origin: baseURL,
		};

		// Create a browser context with auth headers set on every request.
		const context = await browser.newContext({
			baseURL,
			extraHTTPHeaders: authHeaders,
		});

		// Also use route interception as a fallback for any requests
		// that don't pick up the context headers.
		await context.route("**/*", async (route) => {
			const headers = {
				...route.request().headers(),
				...authHeaders,
			};
			await route.continue({ headers });
		});

		const page = await context.newPage();
		await use(page);
		await context.close();
	},

	apiRequest: async ({ playwright }, use) => {
		const baseURL = readServerURL();
		const did = readServerDID();

		if (!did) {
			throw new Error(
				"No primary DID found. Is the e2e-server running? " +
					"Expected tests/e2e/.server-did file.",
			);
		}

		// Create an APIRequestContext with auth headers for direct API calls
		// from tests (page.request doesn't inherit browser context headers).
		const request = await playwright.request.newContext({
			baseURL,
			extraHTTPHeaders: {
				"X-Test-Auth-DID": did,
				"X-Test-Auth-Session": `test-session-${did}`,
				Origin: baseURL,
				Accept: "application/json",
			},
		});

		await use(request);
		await request.dispose();
	},

	did: async ({}, use) => {
		const did = readServerDID();
		if (!did) {
			throw new Error("No primary DID found. Is the e2e-server running?");
		}
		await use(did);
	},
});

export { expect, readServerURL, readServerDID };
