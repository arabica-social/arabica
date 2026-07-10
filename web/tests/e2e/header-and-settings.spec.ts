import { test, expect, readControlURL, readServerURL } from "./fixtures";
import { request as playwrightRequest } from "@playwright/test";

/**
 * Critical path: Header dropdown navigation.
 *
 * The header has two dropdown menus (Create + User). These were broken
 * (invisible) because the Svelte Header rendered `.dropdown-menu` without
 * the `.is-open` class the CSS requires to reveal. This spec locks in the
 * fix by actually opening the Create menu and navigating.
 */
test("header Create dropdown navigates to entity forms", async ({
	authedPage: page,
}) => {
	await page.goto("/");
	await page.waitForLoadState("networkidle");

	// Open the "Create new" dropdown.
	const createBtn = page.getByRole("button", { name: "Create new" });
	await expect(createBtn).toBeVisible();
	await createBtn.click();

	// The dropdown menu should now be visible (the is-open fix). Click the
	// "Bean" item and confirm we land on the bean form.
	await page.getByRole("menuitem", { name: /Bean/ }).click();
	await expect(page).toHaveURL(/\/beans\/new$/);
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
});

test("header User dropdown navigates to settings", async ({ authedPage: page }) => {
	await page.goto("/");
	await page.waitForLoadState("networkidle");

	// Open the user profile dropdown via its aria-label.
	const userBtn = page.getByRole("button", { name: "User menu" });
	await expect(userBtn).toBeVisible();
	await userBtn.click();

	await page.getByRole("menuitem", { name: "Settings" }).click();
	await expect(page).toHaveURL(/\/settings$/);
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
});

/**
 * Critical path: Settings page mutations.
 *
 * Verifies the brewing preferences and profile visibility forms save via
 * the JSON API and surface a toast.
 */
test("settings preferences and visibility save", async ({ authedPage: page }) => {
	await page.goto("/settings");
	await page.waitForLoadState("networkidle");

	// Change the temperature unit preference and save.
	const tempSelect = page.getByLabel("Preferred temperature unit");
	await tempSelect.selectOption("celsius");
	await page.getByRole("button", { name: "Save" }).first().click();
	await expect(page.locator("#toast-region")).toContainText(/Preferences saved/i);

	// Change profile visibility and save.
	const beanVis = page.getByLabel("Bean average brew rating");
	await beanVis.selectOption("private");
	await page.getByRole("button", { name: "Save" }).nth(1).click();
	await expect(page.locator("#toast-region")).toContainText(/Visibility saved/i);
});

/**
 * Critical path: Settings theme toggle.
 *
 * The theme buttons had no click handler in the SPA (the legacy
 * SettingsControlsIsland wasn't loaded). Verifies the toggle now writes
 * localStorage and applies the data-theme attribute.
 */
test("settings theme toggle applies theme", async ({ authedPage: page }) => {
	await page.goto("/settings");
	await page.waitForLoadState("networkidle");

	// Select the Dark theme.
	await page.getByRole("button", { name: "Dark" }).click();

	// The document should now have data-theme="dark".
	await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
	// And localStorage should persist the choice.
	const stored = await page.evaluate(() => localStorage.getItem("arabica-theme"));
	expect(stored).toBe("dark");

	// Selecting System clears it.
	await page.getByRole("button", { name: "System" }).click();
	await expect(page.locator("html")).not.toHaveAttribute("data-theme", "dark");
	const storedAfter = await page.evaluate(() => localStorage.getItem("arabica-theme"));
	expect(storedAfter).toBeNull();
});

/**
 * Critical path: Report modal (non-owner social action).
 *
 * Provisions a second account, creates a record as that account, then views
 * it as the primary account and opens the report modal via the action bar's
 * "More options" menu. Verifies the modal renders and submission succeeds.
 */
test("report modal opens and submits from another user's record", async ({
	authedPage: page,
	did,
}) => {
	// Provision a second account and create a roaster as that account.
	const controlURL = readControlURL();
	const controlReq = await playwrightRequest.newContext({ baseURL: controlURL });
	const acctResp = await controlReq.post("/accounts");
	expect(acctResp.ok()).toBeTruthy();
	const other = (await acctResp.json()) as { did: string; session_id: string };
	await controlReq.dispose();

	// Create a roaster as the other account via the app API.
	const appReq = await playwrightRequest.newContext({
		baseURL: readServerURL(),
		extraHTTPHeaders: {
			"X-Test-Auth-DID": other.did,
			"X-Test-Auth-Session": other.session_id,
			Accept: "application/json",
		},
	});
	const createResp = await appReq.post("/api/roasters", {
		form: { name: "E2E Report Target Roaster" },
	});
	expect(createResp.ok()).toBeTruthy();
	const roaster = await createResp.json();
	await appReq.dispose();

	// View the other account's roaster as the primary (authed) user.
	await page.goto(`/roasters/${other.did}/${roaster.rkey}`);
	await page.waitForLoadState("networkidle");
	await expect(page.getByText("E2E Report Target Roaster")).toBeVisible();

	// Open the "More options" menu and click Report.
	await page.getByRole("button", { name: "More options" }).click();
	await page.getByRole("menuitem", { name: "Report" }).click();

	// The report modal should appear.
	await expect(page.getByText("Report Content").first()).toBeVisible();

	// Submit the report form.
	await page.getByLabel("Report reason").fill("Spam content");
	await page.getByRole("button", { name: "Submit Report" }).click();
	await expect(page.getByText("Report Submitted")).toBeVisible();
});

/**
 * Critical path: 404 / not-found rendering on a SPA-owned entity route.
 *
 * A never-existed entity view should render the "Record not found" state
 * (served by the SPA shell + client-side load), not a raw Go 404.
 */
test("nonexistent entity view renders not-found state", async ({
	authedPage: page,
	did,
}) => {
	await page.goto(`/roasters/${did}/nonexistent-rkey-12345`);
	await page.waitForLoadState("networkidle");

	// The SPA shell should still be served.
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
	// And the not-found message should render.
	await expect(page.getByText("Record not found")).toBeVisible();
});
