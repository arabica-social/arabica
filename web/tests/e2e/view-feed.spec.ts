import { test, expect } from "./fixtures";

test("view feed and filter", async ({ authedPage: page, apiRequest }) => {
	await apiRequest.post("/api/roasters", {
		form: { name: "E2E Feed Roaster", location: "Portland, OR" },
	});

	await page.waitForTimeout(2000);

	await page.goto("/");
	await page.waitForLoadState("networkidle");
	await expect(page.getByText("Community Activity")).toBeVisible();

	await expect(page.locator("#feed-items")).toBeAttached();

	// /api/feed with a type parameter when clicked.
	await expect(page.getByRole("button", { name: "All" })).toBeVisible();
	await expect(page.getByRole("button", { name: "Brews" })).toBeVisible();
	await expect(page.getByRole("button", { name: "Beans" })).toBeVisible();
	await expect(page.getByRole("button", { name: "Recipes" })).toBeVisible();

	const brewsFilter = page.getByRole("button", { name: "Brews" });
	await brewsFilter.click();
	await expect(page.locator("#feed-items")).toBeVisible();

	await page.getByRole("button", { name: "All" }).click();
	await expect(page.locator("#feed-items")).toBeVisible();
});

test("feed load more pagination", async ({ authedPage: page, apiRequest }) => {
	for (let i = 0; i < 3; i++) {
		await apiRequest.post("/api/roasters", {
			form: { name: `E2E Pagination Roaster ${i}` },
		});
	}

	await page.waitForTimeout(2000);
	await page.goto("/");
	await page.waitForLoadState("networkidle");

	const loadMoreBtn = page.getByRole("button", { name: /load more/i });
	if (await loadMoreBtn.isVisible().catch(() => false)) {
		await loadMoreBtn.click();
		await page.waitForTimeout(1000);
	}
});
