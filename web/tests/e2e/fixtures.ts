import { test as base, expect, type Page, type APIRequestContext } from "@playwright/test";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

export type E2EAccount = {
	did: string;
	handle: string;
	session_id: string;
};

function readMarker(name: string, fallback = ""): string {
	const file = resolve(__dirname, `../../../tests/e2e/${name}`);
	try {
		return readFileSync(file, "utf-8").trim();
	} catch {
		return fallback;
	}
}

function readServerURL(): string {
	return process.env.ARABICA_E2E_BASE_URL ?? readMarker(".server-url", "http://127.0.0.1:8080");
}

function readControlURL(): string {
	return process.env.ARABICA_E2E_CONTROL_URL ?? readMarker(".control-url", "");
}

function readServerDID(): string {
	return readMarker(".server-did", process.env.ARABICA_E2E_DID ?? "");
}

export const test = base.extend<{
	account: E2EAccount;
	authedPage: Page;
	apiRequest: APIRequestContext;
	did: string;
	waitForIndex: (uri: string, present?: boolean) => Promise<void>;
}>({
	account: async ({ playwright }, use) => {
		const controlURL = readControlURL();
		if (!controlURL) throw new Error("No E2E control URL found. Is cmd/e2e-server running?");
		const request = await playwright.request.newContext({ baseURL: controlURL });
		const response = await request.post("/accounts");
		if (!response.ok()) {
			throw new Error(`Failed to provision isolated E2E account: ${response.status()} ${await response.text()}`);
		}
		const account = (await response.json()) as E2EAccount;
		await request.dispose();
		await use(account);
	},

	authedPage: async ({ browser, account }, use) => {
		const baseURL = readServerURL();
		const authHeaders = {
			"X-Test-Auth-DID": account.did,
			"X-Test-Auth-Session": account.session_id,
			Origin: baseURL,
		};
		const context = await browser.newContext({ baseURL, extraHTTPHeaders: authHeaders });
		await context.route("**/*", async (route) => {
			await route.continue({ headers: { ...route.request().headers(), ...authHeaders } });
		});

		const failures: string[] = [];
		const page = await context.newPage();
		page.on("pageerror", (error) => failures.push(`page error: ${error.stack ?? error.message}`));
		page.on("console", (message) => {
			if (message.type() === "error" && !message.text().startsWith("Failed to load resource:")) {
				failures.push(`console error: ${message.text()}`);
			}
		});
		page.on("requestfailed", (request) => {
			if (new URL(request.url()).pathname.startsWith("/api/")) {
				failures.push(`failed API request: ${request.method()} ${request.url()} ${request.failure()?.errorText ?? ""}`);
			}
		});
		page.on("response", (response) => {
			const pathname = new URL(response.url()).pathname;
			if ((pathname.startsWith("/_app/") || pathname.startsWith("/static/")) && response.status() >= 400) {
				failures.push(`asset ${response.status()}: ${pathname}`);
			}
			if (pathname.startsWith("/api/") && response.status() >= 500) {
				failures.push(`API ${response.status()}: ${response.request().method()} ${pathname}`);
			}
		});

		await use(page);
		await context.close();
		if (failures.length > 0) throw new Error(failures.join("\n"));
	},

	apiRequest: async ({ playwright, account }, use) => {
		const baseURL = readServerURL();
		const request = await playwright.request.newContext({
			baseURL,
			extraHTTPHeaders: {
				"X-Test-Auth-DID": account.did,
				"X-Test-Auth-Session": account.session_id,
				Origin: baseURL,
				Accept: "application/json",
			},
		});
		await use(request);
		await request.dispose();
	},

	did: async ({ account }, use) => {
		await use(account.did);
	},

	waitForIndex: async ({ playwright }, use) => {
		const controlURL = readControlURL();
		await use(async (uri: string, present = true) => {
			const request = await playwright.request.newContext({ baseURL: controlURL });
			const response = await request.get("/wait-index", {
				params: { uri, present: String(present) },
			});
			const body = await response.text();
			await request.dispose();
			expect(response.ok(), body).toBeTruthy();
		});
	},
});

export { expect, readServerURL, readServerDID, readControlURL };
