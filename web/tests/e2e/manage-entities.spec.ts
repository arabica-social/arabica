import { test, expect, readServerURL, readServerDID } from "./fixtures";

/**
 * Critical path: Manage entities.
 *
 * Flow: my-coffee → create roaster → create bean referencing roaster →
 * verify both appear in my-coffee.
 */
test("create roaster and bean via API, verify in my-coffee", async ({
	authedPage: page,
}) => {
	const baseURL = readServerURL();

	// Create a roaster via the API.
	const roasterResp = await page.request.post(`${baseURL}/api/roasters`, {
		form: { name: "E2E Manage Roaster", location: "Seattle, WA" },
	});
	expect(roasterResp.ok()).toBeTruthy();
	const roaster = await roasterResp.json();
	const roasterRKey = roaster.rkey;

	// Create a bean referencing the roaster.
	const beanResp = await page.request.post(`${baseURL}/api/beans`, {
		form: {
			name: "E2E Manage Bean",
			origin: "Colombia",
			roast_level: "Medium",
			roaster_rkey: roasterRKey,
		},
	});
	expect(beanResp.ok()).toBeTruthy();

	// Navigate to my-coffee and verify both entities appear.
	await page.goto("/my-coffee");
	await expect(page.getByRole("heading", { name: /my coffee/i })).toBeVisible();

	// Switch to the Roasters tab.
	await page.getByRole("button", { name: "Roasters" }).click();
	await expect(page.getByText("E2E Manage Roaster")).toBeVisible();

	// Switch to the Beans tab.
	await page.getByRole("button", { name: "Beans" }).click();
	await expect(page.getByText("E2E Manage Bean")).toBeVisible();
});

/**
 * Critical path: Entity view.
 *
 * Flow: create roaster → view roaster detail page → verify content.
 */
test("view entity detail page", async ({ authedPage: page }) => {
	const baseURL = readServerURL();
	const did = readServerDID();

	// Create a roaster.
	const resp = await page.request.post(`${baseURL}/api/roasters`, {
		form: {
			name: "E2E View Roaster",
			location: "Austin, TX",
			website: "https://example.com",
		},
	});
	const roaster = await resp.json();

	// Wait for indexing.
	await page.waitForTimeout(1500);

	// Navigate to the roaster view page using the DID + rkey.
	await page.goto(`/roasters/${did}/${roaster.rkey}`);

	// Verify the roaster details render.
	await expect(page.getByText("E2E View Roaster")).toBeVisible();
	await expect(page.getByText("Austin, TX")).toBeVisible();
});
