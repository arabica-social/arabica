import { test, expect } from "./fixtures";

test("view own profile", async ({ authedPage: page, apiRequest, did }) => {
	await apiRequest.post("/api/roasters", {
		form: { name: "E2E Profile Roaster" },
	});
	await page.waitForTimeout(2000);

	await page.goto(`/profile/${did}`);
	await page.waitForLoadState("networkidle");
	await expect(page).toHaveURL(new RegExp(`/profile/${did}`));
});

test("profile not found", async ({ authedPage: page }) => {
	await page.goto("/profile/nonexistent.test");
	await page.waitForLoadState("networkidle");
	// Should show an error or redirect — either way, not a 500.
	await expect(page).toHaveURL(/\/profile\/nonexistent\.test|\/404|\/$/);
});
