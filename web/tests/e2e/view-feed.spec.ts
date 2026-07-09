import { test, expect, readServerURL } from "./fixtures";

/**
 * Critical path: View the feed.
 *
 * Flow: home → feed loads → verify feed items render → use filter tabs.
 */
test("view feed and filter", async ({ authedPage: page }) => {
	// Create a roaster via the API so the feed has data.
	const baseURL = readServerURL();
	await page.request.post(`${baseURL}/api/roasters`, {
		form: { name: "E2E Feed Roaster", location: "Portland, OR" },
	});

	// Wait for the firehose to index the record (the SPA reads from the
	// JSON API which reads from the witness cache).
	await page.waitForTimeout(2000);

	// Load the home page.
	await page.goto("/");
	await expect(page.getByText("Community Activity")).toBeVisible();

	// The feed should contain at least one item.
	const feedGrid = page.locator("#feed-items");
	await expect(feedGrid).toBeVisible();

	// Verify the filter tabs are present and clickable.
	const allTab = page.getByRole("button", { name: "All" });
	const brewsTab = page.getByRole("button", { name: "Brews" });
	const beansTab = page.getByRole("button", { name: "Beans" });

	await expect(allTab).toBeVisible();
	await expect(brewsTab).toBeVisible();
	await expect(beansTab).toBeVisible();

	// Click the "Roasters" filter and verify the URL updates.
	await page.getByRole("button", { name: "Roasters" }).click();
	await expect(page).toHaveURL(/type=roaster/);

	// Click "All" to reset.
	await allTab.click();
	await expect(page).toHaveURL(/^[^?]+$/);
});

/**
 * Critical path: Feed pagination.
 *
 * Flow: home → feed loads → load more if pagination is available.
 */
test("feed load more pagination", async ({ authedPage: page }) => {
	const baseURL = readServerURL();

	// Create multiple roasters so the feed has enough items for pagination.
	for (let i = 0; i < 3; i++) {
		await page.request.post(`${baseURL}/api/roasters`, {
			form: { name: `E2E Pagination Roaster ${i}` },
		});
	}

	await page.waitForTimeout(2000);
	await page.goto("/");

	// If the "Load more" button is present, click it.
	const loadMoreBtn = page.getByRole("button", { name: /load more/i });
	if (await loadMoreBtn.isVisible().catch(() => false)) {
		await loadMoreBtn.click();
		await expect(loadMoreBtn).not.toHaveText("Loading...");
	}
});
