import { test, expect } from "./fixtures";

/**
 * Mobile viewport tests.
 *
 * Verifies key responsive layouts at 375px width (iPhone-class):
 *   - Feed masonry flattens to a single column below the 768px breakpoint
 *   - Entity forms are usable (fields not clipped, submit visible)
 *   - Header elements remain accessible
 *   - My Coffee tabs scroll horizontally without breaking
 *
 * These run under the "mobile" project (375x812). The desktop project
 * covers full-width assertions; these focus on responsive breakage only.
 */

test("feed renders in a single column on mobile", async ({
	authedPage: page,
	apiRequest,
}) => {
	await apiRequest.post("/api/roasters", {
		form: { name: "Mobile Feed Roaster" },
	});
	await page.waitForTimeout(2000);

	await page.goto("/");
	await page.waitForLoadState("networkidle");
	await expect(page.getByText("Community Activity")).toBeVisible();

	// The feed masonry JS distributes cards into .feed-masonry-col elements
	// only at >=768px. On mobile (375px) the cards should NOT be in
	// masonry columns — they render directly in the grid's single column.
	const masonryCols = await page.locator(".feed-masonry-col").count();
	expect(masonryCols).toBe(0);

	// The feed-items container should still be present.
	await expect(page.locator("#feed-items")).toBeAttached();
});

test("brew form is usable on mobile — fields visible and not clipped", async ({
	authedPage: page,
}) => {
	await page.goto("/brews/new");
	await page.waitForLoadState("networkidle");
	await expect(page.getByRole("heading", { name: "New Brew" })).toBeVisible();

	// The form sections should be visible (not clipped off-screen).
	await expect(page.getByRole("region", { name: "Coffee" })).toBeVisible();
	await expect(page.getByRole("region", { name: "Brewing" })).toBeVisible();
	await expect(page.getByRole("region", { name: "Results" })).toBeVisible();

	// The submit button should be visible without horizontal scrolling.
	const submitButton = page.getByRole("button", { name: "Save Brew" });
	await expect(submitButton).toBeVisible();
	const box = await submitButton.boundingBox();
	expect(box).not.toBeNull();
	expect(box!.x).toBeGreaterThanOrEqual(0);
	expect(box!.x + box!.width).toBeLessThanOrEqual(375);
});

test("header elements are accessible on mobile", async ({ authedPage: page }) => {
	await page.goto("/");
	await page.waitForLoadState("networkidle");

	// The brand/logo link should be visible.
	await expect(page.getByRole("link", { name: /arabica/i }).first()).toBeVisible();

	// The "Create new" button should still be accessible (not hidden).
	await expect(page.getByRole("button", { name: "Create new" })).toBeVisible();

	// The user menu button should be accessible.
	await expect(page.getByRole("button", { name: "User menu" })).toBeVisible();
});

test("my coffee tabs scroll horizontally on mobile", async ({ authedPage: page }) => {
	await page.goto("/my-coffee");
	await page.waitForLoadState("networkidle");
	await expect(page.getByRole("heading", { name: "My Coffee" })).toBeVisible();

	// The tab nav should be present with all tabs.
	const tabs = ["Brews", "Beans", "Roasters", "Grinders", "Brewers", "Recipes"];
	for (const tab of tabs) {
		await expect(page.getByRole("button", { name: tab })).toBeVisible();
	}

	// The nav container should allow horizontal scrolling (overflow-x-auto).
	const nav = page.locator("nav").first();
	await expect(nav).toBeVisible();
	// Clicking a tab further down the list should work even on narrow screens.
	await page.getByRole("button", { name: "Recipes" }).click();
	await expect(page.getByRole("button", { name: "Recipes" })).toHaveAttribute("aria-current", "page");
});

test("entity view page renders without horizontal overflow on mobile", async ({
	authedPage: page,
	apiRequest,
	did,
	waitForIndex,
}) => {
	const roaster = await (
		await apiRequest.post("/api/roasters", {
			form: { name: "Mobile View Roaster", location: "Portland, OR" },
		})
	).json();
	await waitForIndex(`at://${did}/social.arabica.alpha.roaster/${roaster.rkey}`);

	await page.goto(`/roasters/${did}/${roaster.rkey}`);
	await page.waitForLoadState("networkidle");
	await expect(page.getByRole("heading", { name: "Mobile View Roaster" })).toBeVisible();

	// The page should not have horizontal scroll (body wider than viewport).
	const scrollWidth = await page.evaluate(() => document.documentElement.scrollWidth);
	expect(scrollWidth).toBeLessThanOrEqual(375 + 2); // allow 2px tolerance for borders
});
