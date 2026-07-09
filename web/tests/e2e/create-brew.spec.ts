import { test, expect } from "./fixtures";

/**
 * Critical path: Create a brew.
 *
 * Flow: create prerequisites → /brews/new → verify form renders →
 * verify bean in my-coffee.
 *
 * Note: the Go handler for /brews/new redirects to /onboarding if the
 * user doesn't have a bean, brewer, AND roaster. We create all three
 * via the API first so the brew form page renders.
 *
 * The brew form is a Svelte island (data-svelte-brew-form) that hydrates
 * client-side from svelte-islands.js. We wait for hydration before
 * asserting on form content.
 */
test("create a brew", async ({ authedPage: page, apiRequest }) => {
	// Create prerequisite entities via the API so the brew form renders
	// (the server redirects to /onboarding if the user isn't "ready").
	await apiRequest.post("/api/roasters", {
		form: { name: "E2E Test Roaster" },
	});
	await apiRequest.post("/api/brewers", {
		form: { name: "E2E Test Brewer" },
	});
	await apiRequest.post("/api/beans", {
		form: { name: "E2E Test Bean", origin: "Ethiopia" },
	});

	// Wait for indexing.
	await page.waitForTimeout(1500);

	// Navigate to the new brew form.
	await page.goto("/brews/new");
	await page.waitForLoadState("networkidle");

	// Verify the page title is present (server-rendered).
	await expect(page.getByRole("heading", { name: "New Brew" })).toBeVisible();

	// Wait for the Svelte island to hydrate (the brew form mount point
	// gets populated with form fields by svelte-islands.js).
	await expect(page.locator("[data-svelte-brew-form]")).toBeVisible();
	await expect(page.locator("[data-svelte-brew-form] legend").first()).toBeVisible({ timeout: 15000 });

	// Navigate to my-coffee to verify the bean was created.
	await page.goto("/my-coffee");
	// Wait for the HTMX-loaded manage partial to show the bean.
	await expect(page.getByText("E2E Test Bean")).toBeVisible({ timeout: 15000 });
});
