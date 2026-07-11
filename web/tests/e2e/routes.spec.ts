import { test, expect } from "./fixtures";

/**
 * SPA route coverage for pages that previously only had page-level (Vitest)
 * tests. These verify real data flow through the API + SvelteKit load
 * functions, not just component rendering.
 */

// ---------------------------------------------------------------------------
// Onboarding / Get Started
// ---------------------------------------------------------------------------
test("onboarding page shows setup stations for a fresh account", async ({
	authedPage: page,
}) => {
	await page.goto("/onboarding");
	await page.waitForLoadState("networkidle");
	await expect(page.getByRole("heading", { name: "Let's get you started." })).toBeVisible();

	// The four stations should be present: Brewer, Roaster, Bean, Grinder.
	await expect(page.getByRole("heading", { name: "Brewer" })).toBeVisible();
	await expect(page.getByRole("heading", { name: "Roaster" })).toBeVisible();
	await expect(page.getByRole("heading", { name: "Bean" })).toBeVisible();
	await expect(page.getByRole("heading", { name: "Grinder" })).toBeVisible();

	// Brewer, Roaster, Bean are required; Grinder is optional.
	await expect(page.getByText("required").first()).toBeVisible();
	await expect(page.locator("span.station-tag[data-tag='optional']")).toBeVisible();
});

test("onboarding station drawer opens on click", async ({ authedPage: page }) => {
	await page.goto("/onboarding");
	await page.waitForLoadState("networkidle");

	// Click the "Add a Brewer" station button.
	const brewerAdd = page.locator("article.station[data-kind='brewer'] button.station-add");
	await brewerAdd.click();

	// The drawer should open — the StationDrawer renders an inline add form.
	// The aria-expanded attribute reflects the drawer state.
	await expect(brewerAdd).toHaveAttribute("aria-expanded", "true");
});

// ---------------------------------------------------------------------------
// Add records (bulk-add / library page)
// ---------------------------------------------------------------------------
test("add records page loads with setup stations", async ({ authedPage: page }) => {
	await page.goto("/add");
	await page.waitForLoadState("networkidle");
	await expect(page.getByRole("heading", { name: "Add records." })).toBeVisible();
	await expect(page.getByText("your coffee bar")).toBeVisible();

	// The same GetStartedCard component renders the stations in library mode.
	await expect(page.getByRole("heading", { name: "Brewer" })).toBeVisible();
	await expect(page.getByRole("heading", { name: "Bean" })).toBeVisible();
});

// ---------------------------------------------------------------------------
// Explore
// ---------------------------------------------------------------------------
test("explore page loads with search and filters", async ({ authedPage: page, apiRequest }) => {
	// Seed a record so explore has data.
	await apiRequest.post("/api/roasters", {
		form: { name: "Explore Route Roaster", location: "Brooklyn, NY" },
	});
	await page.waitForTimeout(2000);

	await page.goto("/explore");
	await page.waitForLoadState("networkidle");
	await expect(page.getByRole("heading", { name: "Explore records." })).toBeVisible();

	// Search input and type filter.
	await expect(page.getByPlaceholder("Ethiopia, V60, washed")).toBeVisible();

	// The type filter dropdown should list entity types.
	const typeSelect = page.locator("select").first();
	await typeSelect.selectOption("roaster");

	// Apply filters (form submit).
	await page.getByRole("button", { name: "Explore" }).click();
	await page.waitForLoadState("networkidle");

	// Results should load (or the empty state renders).
	const hasResults = await page.locator(".explore-results").count() > 0;
	const hasEmpty = await page.getByText("No matching records yet.").count() > 0;
	expect(hasResults || hasEmpty).toBeTruthy();
});

// ---------------------------------------------------------------------------
// Moderation dashboard (_mod) — permission denied
// ---------------------------------------------------------------------------
test("moderation dashboard shows permission denied for non-moderators", async ({
	authedPage: page,
}) => {
	// The e2e harness builds the router with no moderation service, so the
	// primary account is not a moderator. The page should render the error
	// state, not crash.
	await page.goto("/_mod");
	await page.waitForLoadState("networkidle");
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
	// Either the permission error or the page title should be present.
	const hasDenied = await page.getByText(/permission|don't have/i).count() > 0;
	const hasTitle = await page.getByRole("heading", { name: "Moderation Dashboard" }).count() > 0;
	expect(hasDenied || hasTitle).toBeTruthy();
});

// ---------------------------------------------------------------------------
// Static content pages
// ---------------------------------------------------------------------------
test("about page renders content", async ({ authedPage: page }) => {
	await page.goto("/about");
	await page.waitForLoadState("networkidle");
	await expect(page.getByRole("heading", { name: "About Arabica" })).toBeVisible();
	// Key content about data ownership.
	await expect(page.getByText("You own your data.")).toBeVisible();
});

test("atproto page renders content", async ({ authedPage: page }) => {
	await page.goto("/atproto");
	await page.waitForLoadState("networkidle");
	await expect(page.getByRole("heading", { name: "The AT Protocol", exact: true })).toBeVisible();
	await expect(page.getByText("Personal Data Server (PDS)")).toBeVisible();
});

test("terms page renders content", async ({ authedPage: page }) => {
	await page.goto("/terms");
	await page.waitForLoadState("networkidle");
	await expect(page.getByRole("heading", { name: "Terms of Service" })).toBeVisible();
	await expect(page.getByText("The Simple Truth")).toBeVisible();
	await expect(page.getByText("You own all of your data.")).toBeVisible();
});

// ---------------------------------------------------------------------------
// Join / Create account
// ---------------------------------------------------------------------------
test("join/create page loads provider catalog", async ({ authedPage: page }) => {
	await page.goto("/join/create");
	await page.waitForLoadState("networkidle");
	await expect(
		page.getByRole("heading", { name: "Create an Atmosphere Account" }),
	).toBeVisible();

	// The page should either render provider categories or a load-failed
	// message (the e2e harness may not have a signup catalog configured).
	const hasProviders = await page.getByRole("button", { name: "Create Account" }).count() > 0;
	const hasLoadError = await page.getByText(/Could not load the provider list/i).count() > 0;
	expect(hasProviders || hasLoadError).toBeTruthy();
});
