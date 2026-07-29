import { test, expect } from "./fixtures";

const VIEWPORT = { width: 1280, height: 900 } as const;

test.describe("visual regression", () => {
	test.use({ viewport: VIEWPORT });

	test("new brew form", async ({ authedPage: page }) => {
		await page.goto("/brews/new");
		await page.waitForLoadState("networkidle");
		await expect(page.getByRole("heading", { name: "New Brew" })).toBeVisible();
		await expect(page).toHaveScreenshot("brew-form.png", {
			fullPage: true,
			animations: "disabled",
		});
	});

	test("new bean form", async ({ authedPage: page }) => {
		await page.goto("/beans/new");
		await page.waitForLoadState("networkidle");
		await expect(page.getByRole("heading", { name: "Add a Bean" })).toBeVisible();
		await expect(page).toHaveScreenshot("bean-form.png", {
			fullPage: true,
			animations: "disabled",
		});
	});

	test("my coffee page", async ({ authedPage: page }) => {
		await page.goto("/my-coffee");
		await page.waitForLoadState("networkidle");
		await expect(page.getByRole("heading", { name: "My Coffee" })).toBeVisible();
		await expect(page).toHaveScreenshot("my-coffee.png", {
			fullPage: true,
			animations: "disabled",
		});
	});

	test("settings page", async ({ authedPage: page }) => {
		await page.goto("/settings");
		await page.waitForLoadState("networkidle");
		await expect(page).toHaveScreenshot("settings.png", {
			fullPage: true,
			animations: "disabled",
		});
	});

	test("brew logbook page", async ({ authedPage: page }) => {
		await page.goto("/brews");
		await page.waitForLoadState("networkidle");
		await expect(page.getByRole("heading", { name: "Brew Logbook" })).toBeVisible();
		await expect(page).toHaveScreenshot("brew-logbook.png", {
			fullPage: true,
			animations: "disabled",
		});
	});

	test("recipes catalog page", async ({ authedPage: page }) => {
		await page.goto("/recipes");
		await page.waitForLoadState("networkidle");
		await expect(page.getByRole("heading", { name: "Explore Recipes" })).toBeVisible();
		await expect(page).toHaveScreenshot("recipes-catalog.png", {
			fullPage: true,
			animations: "disabled",
		});
	});

	// fullPage is unreliable here because variable-height sections (feed
	// items, community backlinks) shift the total page height per run.

	test("home feed chrome", async ({ authedPage: page }) => {
		await page.goto("/");
		await page.waitForLoadState("networkidle");
		await expect(page.getByText("Community Activity")).toBeVisible();
		await page.waitForTimeout(500);
		await expect(page).toHaveScreenshot("home-feed.png", {
			animations: "disabled",
			maxDiffPixelRatio: 0.02,
			mask: [page.locator("#feed-items")],
		});
	});

	test("bean view page", async ({ authedPage: page, apiRequest, did, waitForIndex }) => {
		const bean = await (
			await apiRequest.post("/api/beans", {
				form: {
					name: "Visual Bean",
					origin: "Ethiopia",
					variety: "Gesha",
					roast_level: "Light",
					process: "Washed",
					description: "A visually appealing bean.",
				},
			})
		).json();
		await waitForIndex(`at://${did}/social.arabica.alpha.bean/${bean.rkey}`);

		await page.goto(`/beans/${did}/${bean.rkey}`);
		await page.waitForLoadState("networkidle");
		await expect(page.getByRole("heading", { name: "Visual Bean" })).toBeVisible();
		await expect(page).toHaveScreenshot("bean-view.png", {
			animations: "disabled",
			maxDiffPixelRatio: 0.02,
			mask: [
				// Author handle (DID) + timestamp vary per run.
				page.locator(".record-view-author-handle"),
				page.locator(".record-view-meta"),
			],
		});
	});

	test("roaster view page", async ({ authedPage: page, apiRequest, did, waitForIndex }) => {
		const roaster = await (
			await apiRequest.post("/api/roasters", {
				form: {
					name: "Visual Roasters Co",
					location: "Austin, TX",
					website: "https://visual.example",
				},
			})
		).json();
		await waitForIndex(`at://${did}/social.arabica.alpha.roaster/${roaster.rkey}`);

		await page.goto(`/roasters/${did}/${roaster.rkey}`);
		await page.waitForLoadState("networkidle");
		await expect(page.getByRole("heading", { name: "Visual Roasters Co" })).toBeVisible();
		await expect(page).toHaveScreenshot("roaster-view.png", {
			animations: "disabled",
			maxDiffPixelRatio: 0.02,
			mask: [
				page.locator(".record-view-author-handle"),
				page.locator(".record-view-meta"),
			],
		});
	});
});
