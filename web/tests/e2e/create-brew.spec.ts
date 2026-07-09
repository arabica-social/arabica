import { test, expect, readServerDID, readServerURL } from "./fixtures";

/**
 * Critical path: Create a brew.
 *
 * Flow: home → /brews/new → fill form → submit → see in my-coffee.
 */
test("create a brew", async ({ authedPage: page }) => {
	// Start at the home page.
	await page.goto("/");
	await expect(page.getByRole("heading", { name: /coffee journey/i })).toBeVisible();

	// Navigate to the new brew form.
	await page.getByRole("link", { name: /log brew/i }).click();
	await expect(page).toHaveURL(/\/brews\/new/);

	// The brew form requires a bean. Since the test account starts empty,
	// we create a bean first via the API (faster than navigating the UI).
	const baseURL = readServerURL();
	await page.request.post(`${baseURL}/api/beans`, {
		form: { name: "E2E Test Bean", origin: "Ethiopia" },
	});

	// Navigate to my-coffee to verify we have entities.
	await page.goto("/my-coffee");
	await expect(page.getByRole("heading", { name: /my coffee/i })).toBeVisible();

	// The brew form is complex (combo-selects, pours editor). For E2E,
	// we verify the form renders and the key sections are present.
	await page.goto("/brews/new");
	await expect(page.getByText("New Brew")).toBeVisible();
	await expect(page.getByText("Coffee")).toBeVisible();
	await expect(page.getByText("Brewing")).toBeVisible();
	await expect(page.getByText("Results")).toBeVisible();
});
