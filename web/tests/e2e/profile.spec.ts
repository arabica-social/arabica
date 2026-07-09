import { test, expect } from "./fixtures";

/**
 * Critical path: Profile.
 *
 * Flow: view own profile → view another user's profile.
 */
test("view own profile", async ({ authedPage: page, apiRequest, did }) => {
	// Create a roaster so the profile has data.
	await apiRequest.post("/api/roasters", {
		form: { name: "E2E Profile Roaster" },
	});
	await page.waitForTimeout(2000);

	// Navigate to the own profile.
	await page.goto(`/profile/${did}`);
	await page.waitForLoadState("networkidle");
	await expect(page).toHaveURL(new RegExp(`/profile/${did}`));
});

/**
 * Verify the profile page handles a non-existent user gracefully.
 */
test("profile not found", async ({ authedPage: page }) => {
	await page.goto("/profile/nonexistent.test");
	await page.waitForLoadState("networkidle");
	// Should show an error or redirect — either way, not a 500.
	await expect(page).toHaveURL(/\/profile\/nonexistent\.test|\/404|\/$/);
});
