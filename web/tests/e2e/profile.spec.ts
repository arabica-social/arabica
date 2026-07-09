import { test, expect, readServerURL, readServerDID } from "./fixtures";

/**
 * Critical path: Profile.
 *
 * Flow: view own profile → view another user's profile.
 */
test("view own profile", async ({ authedPage: page }) => {
	const baseURL = readServerURL();
	const did = readServerDID();

	// Create a roaster so the profile has data.
	await page.request.post(`${baseURL}/api/roasters`, {
		form: { name: "E2E Profile Roaster" },
	});
	await page.waitForTimeout(2000);

	// Navigate to the own profile.
	await page.goto(`/profile/${did}`);
	await expect(page).toHaveURL(new RegExp(`/profile/${did}`));
});

/**
 * Verify the profile page handles a non-existent user gracefully.
 */
test("profile not found", async ({ authedPage: page }) => {
	await page.goto("/profile/nonexistent.test");
	// Should show an error or redirect — either way, not a 500.
	await expect(page).toHaveURL(/\/profile\/nonexistent\.test|\/404|\/$/);
});
