import { defineConfig, devices } from "@playwright/test";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

/**
 * Read the server URL written by the Go e2e-server binary.
 * Falls back to a localhost URL if the file doesn't exist.
 */
function getServerURL(): string {
	// The path is relative to the repo root (parent of web/).
	const urlFile = resolve(__dirname, "../tests/e2e/.server-url");
	try {
		const url = readFileSync(urlFile, "utf-8").trim();
		if (url) return url;
	} catch {
		// File doesn't exist — use env var or default.
	}
	return process.env.ARABICA_E2E_BASE_URL ?? "http://127.0.0.1:8080";
}

/**
 * Playwright E2E config for arabica.
 *
 * The test server is booted by the Go binary (cmd/e2e-server) which writes
 * its URL to tests/e2e/.server-url. Playwright reads this file to know
 * where to point the browser.
 *
 * Auth is injected via route interception: every request the browser makes
 * gets the X-Test-Auth-DID and X-Test-Auth-Session headers added, so the
 * Go test harness authenticates the request without a real OAuth flow.
 *
 * Run with: just e2e
 */
export default defineConfig({
	testDir: "./tests/e2e",
	testMatch: "*.spec.ts",
	timeout: 30_000,
	expect: {
		timeout: 10_000,
	},
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	workers: 1,
	reporter: "list",
	use: {
		baseURL: getServerURL(),
		trace: "on-first-retry",
		headless: true,
	},
	projects: [
		{
			name: "chromium",
			use: {
				...devices["Desktop Chrome"],
				// Use the system Chromium if available (e.g. via Nix), otherwise
				// fall back to Playwright's bundled browser.
				...(process.env.CHROMIUM_PATH
					? { launchOptions: { executablePath: process.env.CHROMIUM_PATH } }
					: {}),
			},
		},
	],
});
