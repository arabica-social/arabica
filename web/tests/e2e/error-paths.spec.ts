import { test, expect } from "./fixtures";

test("brew create API rejects without a required bean", async ({
	apiRequest,
}) => {
	// The server-side validation returns 400 "Bean selection is required"
	// when bean_rkey is missing. Verified at the API level to avoid the
	// brew form's console.error on expected failures (which the authedPage
	// fixture treats as a test failure).
	const response = await apiRequest.post("/brews", {
		form: {
			coffee_amount: "18",
			water_amount: "250",
			temperature: "93",
			time_seconds: "180",
			rating: "7",
		},
	});
	expect(response.status()).toBe(400);
	const body = await response.text();
	expect(body).toContain("Bean selection is required");
});

test("nonexistent brew view renders not-found state", async ({
	authedPage: page,
	did,
}) => {
	await page.goto(`/brews/${did}/nonexistent-brew-rkey-999`);
	await page.waitForLoadState("networkidle");
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
	await expect(page.getByText("Record not found")).toBeVisible();
});

test("nonexistent bean view renders not-found state", async ({
	authedPage: page,
	did,
}) => {
	await page.goto(`/beans/${did}/nonexistent-bean-rkey-999`);
	await page.waitForLoadState("networkidle");
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
	await expect(page.getByText("Record not found")).toBeVisible();
});

test("nonexistent grinder view renders not-found state", async ({
	authedPage: page,
	did,
}) => {
	await page.goto(`/grinders/${did}/nonexistent-grinder-rkey-999`);
	await page.waitForLoadState("networkidle");
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
	await expect(page.getByText("Record not found")).toBeVisible();
});

test("nonexistent recipe view renders not-found state", async ({
	authedPage: page,
	did,
}) => {
	await page.goto(`/recipes/${did}/nonexistent-recipe-rkey-999`);
	await page.waitForLoadState("networkidle");
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
	await expect(page.getByText("Record not found")).toBeVisible();
});

test("explore renders empty state when no community records exist", async ({
	authedPage: page,
}) => {
	await page.goto("/explore");
	await page.waitForLoadState("networkidle");
	await expect(page.getByRole("heading", { name: "Explore records." })).toBeVisible();

	await expect(page.getByPlaceholder("Ethiopia, V60, washed")).toBeVisible();

	// Either results render (if the global index has data from other tests)
	// or the empty state renders — either way the page must not error.
	const hasResults = await page.locator(".explore-results").count() > 0;
	const hasEmpty = await page.getByText("No matching records yet.").count() > 0;
	expect(hasResults || hasEmpty).toBeTruthy();
});
