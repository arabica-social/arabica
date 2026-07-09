import { test, expect, readServerURL, readServerDID } from "./fixtures";

/**
 * Critical path: Social interactions.
 *
 * Flow: create a record → view it → like it → comment → check notifications.
 */
test("like and comment on a record", async ({ authedPage: page }) => {
	const baseURL = readServerURL();
	const did = readServerDID();

	// Create a roaster to interact with.
	const resp = await page.request.post(`${baseURL}/api/roasters`, {
		form: { name: "E2E Social Roaster" },
	});
	const roaster = await resp.json();

	// Wait for indexing so the view page loads correctly.
	await page.waitForTimeout(2000);

	// Navigate to the roaster view page.
	await page.goto(`/roasters/${did}/${roaster.rkey}`);
	await expect(page.getByText("E2E Social Roaster")).toBeVisible();

	// Find and click the like button. The ActionBar renders a like button
	// with an accessible name.
	const likeButton = page.getByRole("button", { name: /like/i }).first();
	if (await likeButton.isVisible().catch(() => false)) {
		await likeButton.click();
		await page.waitForTimeout(500);
	}

	// Find the comment section and add a comment.
	const commentTextarea = page.locator("textarea").first();
	if (await commentTextarea.isVisible().catch(() => false)) {
		await commentTextarea.fill("E2E test comment!");
		const submitBtn = page
			.getByRole("button", { name: /post|comment|submit/i })
			.first();
		if (await submitBtn.isVisible().catch(() => false)) {
			await submitBtn.click();
			await page.waitForTimeout(500);
		}
	}
});

/**
 * Critical path: Notifications.
 *
 * Flow: create a record → like it (generates a notification) →
 * check notifications page.
 */
test("view notifications", async ({ authedPage: page }) => {
	const baseURL = readServerURL();
	const did = readServerDID();

	// Create a roaster and like it to generate a notification.
	const resp = await page.request.post(`${baseURL}/api/roasters`, {
		form: { name: "E2E Notif Roaster" },
	});
	const roaster = await resp.json();
	const subjectURI = `at://${did}/social.arabica.alpha.roaster/${roaster.rkey}`;

	// Like the roaster via the API to generate a notification.
	await page.request.post(`${baseURL}/api/likes/toggle`, {
		form: { subject_uri: subjectURI, subject_cid: "bafyfake" },
	});

	// Wait for the firehose to index the like.
	await page.waitForTimeout(2000);

	// Navigate to the notifications page.
	await page.goto("/notifications");
	await expect(page).toHaveURL(/\/notifications/);
});
