import { test as base, expect, type Page } from "@playwright/test";
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
 * Extended test fixture that injects auth headers on every request so
 * the Go test harness authenticates the browser session.
 *
 * The harness uses X-Test-Auth-DID and X-Test-Auth-Session headers to
 * bypass OAuth. We inject these headers in two places:
 *
 * 1. Browser context extraHTTPHeaders — applies to all browser navigation
 *    and fetch() calls from the SPA.
 * 2. page.route interception — catches any requests that don't pick up
 *    the context headers (e.g. cross-origin or redirected requests).
 *
 * For page.request (Playwright's APIRequestContext used in tests), we
 * pass the headers explicitly on each call via a helper.
 *
 * Usage in a spec:
 *
 *   import { test, expect } from "./fixtures";
 *   test("create brew", async ({ authedPage: page }) => { ... });
 */
export const test = base.extend<{ authedPage: Page }>({
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
});

export { expect, readServerURL, readServerDID };
