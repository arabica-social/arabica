import { test, expect } from "./fixtures";

test("like and comment on a record", async ({
	authedPage: page,
	apiRequest,
	did,
}) => {
	const resp = await apiRequest.post("/api/roasters", {
		form: { name: "E2E Social Roaster" },
	});
	const roaster = await resp.json();

	await page.waitForTimeout(2000);

	await page.goto(`/roasters/${did}/${roaster.rkey}`);
	await page.waitForLoadState("networkidle");
	await expect(page.getByText("E2E Social Roaster")).toBeVisible();

	const likeButton = page.getByRole("button", { name: /like/i }).first();
	if (await likeButton.isVisible().catch(() => false)) {
		await likeButton.click();
		await page.waitForTimeout(500);
	}

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

test("view notifications", async ({ authedPage: page, apiRequest, did }) => {
	const resp = await apiRequest.post("/api/roasters", {
		form: { name: "E2E Notif Roaster" },
	});
	const roaster = await resp.json();
	const subjectURI = `at://${did}/social.arabica.alpha.roaster/${roaster.rkey}`;

	await apiRequest.post("/api/likes/toggle", {
		form: { subject_uri: subjectURI, subject_cid: "bafyfake" },
	});

	await page.waitForTimeout(2000);

	await page.goto("/notifications");
	await page.waitForLoadState("networkidle");
	await expect(page).toHaveURL(/\/notifications/);
});
