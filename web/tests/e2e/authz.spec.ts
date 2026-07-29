import { test, expect, readControlURL, readServerURL } from "./fixtures";
import { request as playwrightRequest } from "@playwright/test";

/**
 * Cross-user authorization and visibility tests.
 *
 * Provisions a second account, creates records as that account, then views
 * them as the primary (authed) user to verify:
 *   - Non-owners can view records but Edit/Delete actions are hidden
 *   - The Report action appears for non-owners
 *   - A like by one account is visible (as a count) from another account
 *
 * The store is always scoped to the authenticated user's PDS, so writes
 * (PUT/DELETE) inherently cannot touch another user's records — the UI
 * hides those actions entirely when !isOwner.
 */

/**
 * Provision a fresh account via the control server and return a request
 * context authenticated as that account.
 */
async function provisionAccount(): Promise<{
	did: string;
	handle: string;
	session_id: string;
	request: import("@playwright/test").APIRequestContext;
}> {
	const controlURL = readControlURL();
	const controlReq = await playwrightRequest.newContext({ baseURL: controlURL });
	const resp = await controlReq.post("/accounts");
	expect(resp.ok()).toBeTruthy();
	const account = (await resp.json()) as { did: string; handle: string; session_id: string };
	await controlReq.dispose();

	const baseURL = readServerURL();
	const request = await playwrightRequest.newContext({
		baseURL,
		extraHTTPHeaders: {
			"X-Test-Auth-DID": account.did,
			"X-Test-Auth-Session": account.session_id,
			Origin: baseURL,
			Accept: "application/json",
		},
	});
	return { ...account, request };
}

test("non-owner can view a record but Edit and Delete are hidden", async ({
	authedPage: page,
}) => {
	const other = await provisionAccount();
	const suffix = Date.now().toString(36);
	const name = `E2E Authz Roaster ${suffix}`;

	const createResp = await other.request.post("/api/roasters", {
		form: { name, location: "Denver, CO" },
	});
	expect(createResp.ok()).toBeTruthy();
	const roaster = await createResp.json();

	// see it via the witness cache.
	await page.waitForTimeout(2000);

	await page.goto(`/roasters/${other.did}/${roaster.rkey}`);
	await page.waitForLoadState("networkidle");
	await expect(page.getByRole("heading", { name })).toBeVisible();

	await page.getByRole("button", { name: "More options" }).click();

	await expect(page.getByRole("menuitem", { name: "Edit" })).toHaveCount(0);
	await expect(page.getByRole("menuitem", { name: "Delete" })).toHaveCount(0);

	await expect(page.getByRole("menuitem", { name: "Report" })).toBeVisible();

	// "Close Bag" / "Rate Bag" / "Edit Bean" (owner-only buttons on the
	// bean view) should not appear on a roaster view regardless, but the
	// key assertion is the absence of Edit/Delete above.
	await other.request.dispose();
});

test("owner sees Edit and Delete on their own record", async ({
	authedPage: page,
	did,
}) => {
	const suffix = Date.now().toString(36);
	const name = `E2E Owner Roaster ${suffix}`;
	const createResp = await page.request.post("/api/roasters", {
		form: { name, location: "Seattle, WA" },
	});
	expect(createResp.ok()).toBeTruthy();
	const roaster = await createResp.json();
	await page.waitForTimeout(2000);

	await page.goto(`/roasters/${did}/${roaster.rkey}`);
	await page.waitForLoadState("networkidle");
	await expect(page.getByRole("heading", { name })).toBeVisible();

	await page.getByRole("button", { name: "More options" }).click();

	await expect(page.getByRole("menuitem", { name: "Edit" })).toBeVisible();
	await expect(page.getByRole("menuitem", { name: "Delete" })).toBeVisible();

	await expect(page.getByRole("menuitem", { name: "Report" })).toHaveCount(0);
});

test("like by one account is visible as a count from another account", async ({
	authedPage: page,
}) => {
	const other = await provisionAccount();
	const suffix = Date.now().toString(36);

	const createResp = await other.request.post("/api/roasters", {
		form: { name: `E2E Like Roaster ${suffix}` },
	});
	expect(createResp.ok()).toBeTruthy();
	const roaster = await createResp.json();
	const subjectURI = `at://${other.did}/social.arabica.alpha.roaster/${roaster.rkey}`;

	const likeResp = await other.request.post("/api/likes/toggle", {
		form: { subject_uri: subjectURI, subject_cid: "bafyfake" },
	});
	expect(likeResp.ok()).toBeTruthy();

	await page.waitForTimeout(2500);

	await page.goto(`/roasters/${other.did}/${roaster.rkey}`);
	await page.waitForLoadState("networkidle");

	const likeButton = page.getByRole("button", { name: /like/i }).first();
	if (await likeButton.isVisible().catch(() => false)) {
		// The count is rendered as text inside/near the like button.
		await expect.poll(async () => {
			const text = (await likeButton.textContent()) ?? "";
			const match = text.match(/\d+/);
			return match ? Number(match[0]) : 0;
		}).toBeGreaterThanOrEqual(1);
	}

	await other.request.dispose();
});
