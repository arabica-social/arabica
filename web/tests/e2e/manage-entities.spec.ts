import { test, expect } from "./fixtures";

/**
 * Critical path: Manage entities.
 *
 * Flow: create roaster → create bean referencing roaster → navigate to
 * my-coffee → verify both entities appear in their respective tabs.
 *
 * Note: the my-coffee page loads data via HTMX (hx-get="/api/manage").
 * We wait for the HTMX content to load before asserting. Tab switching
 * uses data-manage-tab attributes with JS-driven visibility.
 */
test("create roaster and bean via API, verify in my-coffee", async ({
	authedPage: page,
	apiRequest,
}) => {
	// Create a roaster via the API.
	const roasterResp = await apiRequest.post("/api/roasters", {
		form: { name: "E2E Manage Roaster", location: "Seattle, WA" },
	});
	expect(roasterResp.ok()).toBeTruthy();
	const roaster = await roasterResp.json();
	const roasterRKey = roaster.rkey;

	// Create a bean referencing the roaster.
	const beanResp = await apiRequest.post("/api/beans", {
		form: {
			name: "E2E Manage Bean",
			origin: "Colombia",
			roast_level: "Medium",
			roaster_rkey: roasterRKey,
		},
	});
	expect(beanResp.ok()).toBeTruthy();

	// Wait for the records to be indexed by the firehose.
	await page.waitForTimeout(2000);

	// Navigate to my-coffee. The page loads entity data via HTMX
	// (hx-get="/api/manage" on load). We wait for the bean name to
	// appear in the DOM, which means the HTMX swap completed.
	await page.goto("/my-coffee");
	// Wait for the manage partial to load via HTMX.
	await expect(page.getByText("E2E Manage Bean")).toBeVisible({ timeout: 15000 });

	// Switch to the Roasters tab.
	await page.getByRole("button", { name: "Roasters" }).click();
	await expect(page.getByText("E2E Manage Roaster").first()).toBeVisible();
});

/**
 * Critical path: Entity view.
 *
 * Flow: create roaster → view roaster detail page → verify content.
 */
test("view entity detail page", async ({ authedPage: page, apiRequest, did }) => {
	// Create a roaster.
	const resp = await apiRequest.post("/api/roasters", {
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
	await page.waitForLoadState("networkidle");

	// Verify the roaster details render.
	await expect(page.getByText("E2E View Roaster")).toBeVisible();
	await expect(page.getByText("Austin, TX")).toBeVisible();
});
