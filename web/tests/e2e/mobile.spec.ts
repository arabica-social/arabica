import { test, expect } from "./fixtures";

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
	await expect(page.getByText("Recent records")).toBeVisible();

	// The feed masonry JS distributes cards into .feed-masonry-col elements
	// only at >=768px. On mobile (375px) the cards should NOT be in
	// masonry columns — they render directly in the grid's single column.
	const masonryCols = await page.locator(".feed-masonry-col").count();
	expect(masonryCols).toBe(0);

	await expect(page.locator("#feed-items")).toBeAttached();
});

test("brew form is usable on mobile — fields visible and not clipped", async ({
	authedPage: page,
}) => {
	await page.goto("/brews/new");
	await page.waitForLoadState("networkidle");
	await expect(page.getByRole("heading", { name: "New Brew" })).toBeVisible();

	await expect(page.getByRole("region", { name: "Coffee" })).toBeVisible();
	await expect(page.getByRole("region", { name: "Brewing" })).toBeVisible();
	await expect(page.getByRole("region", { name: "Results" })).toBeVisible();

	const submitButton = page.getByRole("button", { name: "Save brew" });
	await expect(submitButton).toBeVisible();
	const box = await submitButton.boundingBox();
	expect(box).not.toBeNull();
	expect(box!.x).toBeGreaterThanOrEqual(0);
	expect(box!.x + box!.width).toBeLessThanOrEqual(375);
});

test("header elements are accessible on mobile", async ({ authedPage: page }) => {
	await page.goto("/");
	await page.waitForLoadState("networkidle");

	await expect(page.getByRole("link", { name: /arabica/i }).first()).toBeVisible();

	await expect(page.getByRole("button", { name: "Create new" })).toBeVisible();

	await expect(page.getByRole("button", { name: "User menu" })).toBeVisible();
});

test("log menu stays centered within the viewport on mobile", async ({
	authedPage: page,
}) => {
	await page.goto("/");
	await page.waitForLoadState("networkidle");

	await page.getByRole("button", { name: "Create new" }).click();
	const menu = page.locator(".ledger-menu--create");
	await expect(menu).toBeVisible();

	const box = await menu.boundingBox();
	expect(box).not.toBeNull();
	const viewportWidth = page.viewportSize()!.width;
	expect(box!.x).toBeGreaterThanOrEqual(0);
	expect(box!.x + box!.width).toBeLessThanOrEqual(viewportWidth);
	expect(box!.x + box!.width / 2).toBeCloseTo(viewportWidth / 2, 0);
});

test("my coffee tabs scroll horizontally on mobile", async ({ authedPage: page }) => {
	await page.goto("/my-coffee");
	await page.waitForLoadState("networkidle");
	await expect(page.getByRole("heading", { name: "My Coffee" })).toBeVisible();

	const tabs = ["Brews", "Beans", "Roasters", "Grinders", "Brewers", "Recipes"];
	for (const tab of tabs) {
		await expect(page.getByRole("button", { name: tab })).toBeVisible();
	}

	// The nav container should allow horizontal scrolling (overflow-x-auto).
	const nav = page.locator("nav").first();
	await expect(nav).toBeVisible();
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
