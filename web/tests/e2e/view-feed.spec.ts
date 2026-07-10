import { test, expect } from "./fixtures";

/**
 * Critical path: View the feed.
 *
 * Flow: home → feed loads → verify feed items render → use filter tabs.
 *
 * The SPA home page fetches /api/feed as JSON and re-renders Svelte
 * client-side. Filter pills trigger a refetch; the URL doesn't change.
 */
test("view feed and filter", async ({ authedPage: page, apiRequest }) => {
	// Create a roaster via the API so the feed has data.
	await apiRequest.post("/api/roasters", {
		form: { name: "E2E Feed Roaster", location: "Portland, OR" },
	});

	// Wait for the firehose to index the record.
	await page.waitForTimeout(2000);

	// Load the home page and wait for hydration.
	await page.goto("/");
	await page.waitForLoadState("networkidle");
	await expect(page.getByText("Community Activity")).toBeVisible();

	// Verify the feed items container exists.
	await expect(page.locator("#feed-items")).toBeAttached();

	// Verify the filter pills are present. The SPA feed filters refetch
	// /api/feed with a type parameter when clicked.
	await expect(page.getByRole("button", { name: "All" })).toBeVisible();
	await expect(page.getByRole("button", { name: "Brews" })).toBeVisible();
	await expect(page.getByRole("button", { name: "Beans" })).toBeVisible();
	await expect(page.getByRole("button", { name: "Recipes" })).toBeVisible();

	// Click the "Brews" filter and wait for the feed refetch to settle.
	const brewsFilter = page.getByRole("button", { name: "Brews" });
	await brewsFilter.click();
	await expect(page.locator("#feed-items")).toBeVisible();

	// Click "All" to reset.
	await page.getByRole("button", { name: "All" }).click();
	await expect(page.locator("#feed-items")).toBeVisible();
});

/**
 * Critical path: Feed pagination.
 *
 * Flow: home → feed loads → load more if pagination is available.
 */
test("feed load more pagination", async ({ authedPage: page, apiRequest }) => {
	// Create multiple roasters so the feed has enough items for pagination.
	for (let i = 0; i < 3; i++) {
		await apiRequest.post("/api/roasters", {
			form: { name: `E2E Pagination Roaster ${i}` },
		});
	}

	await page.waitForTimeout(2000);
	await page.goto("/");
	await page.waitForLoadState("networkidle");

	// If the "Load more" button is present, click it.
	const loadMoreBtn = page.getByRole("button", { name: /load more/i });
	if (await loadMoreBtn.isVisible().catch(() => false)) {
		await loadMoreBtn.click();
		await page.waitForTimeout(1000);
	}
});
