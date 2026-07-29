import { test, expect } from "./fixtures";

test("roaster lifecycle: create, reload, edit, manage, delete", async ({
	authedPage: page,
	did,
	waitForIndex,
}) => {
	const suffix = Date.now().toString(36);
	const name = `E2E Roaster ${suffix}`;
	const originalLocation = "Portland, OR";
	const originalWebsite = `https://original-${suffix}.example`;
	const updatedLocation = "Seattle, WA";
	const updatedWebsite = `https://updated-${suffix}.example`;

	await page.goto("/roasters/new");
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();

	await page.getByRole("button", { name: "Add Roaster" }).click();
	await expect(page.getByRole("alert")).toContainText("Name is required");

	await page.getByLabel("Name").fill(name);
	await page.getByLabel("Location").fill(originalLocation);
	await page.getByLabel("Website").fill(originalWebsite);
	await page.getByRole("button", { name: "Add Roaster" }).click();

	await page.waitForURL(/\/roasters\/[^/]+\/[^/]+$/);
	const detailURL = new URL(page.url());
	const rkey = detailURL.pathname.split("/").at(-1);
	expect(rkey).toBeTruthy();
	await expect(page.getByRole("heading", { name })).toBeVisible();
	await expect(page.getByText(originalLocation)).toBeVisible();
	await expect(page.getByRole("link", { name: originalWebsite })).toBeVisible();

	await page.reload();
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
	await expect(page.getByRole("heading", { name })).toBeVisible();

	await page.getByRole("button", { name: "More options" }).click();
	await page.getByRole("menuitem", { name: "Edit" }).click();
	await expect(page).toHaveURL(`/roasters/${rkey}/edit`);
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
	await expect(page.getByLabel("Name")).toHaveValue(name);
	await page.getByLabel("Location").fill(updatedLocation);
	await page.getByLabel("Website").fill(updatedWebsite);
	await page.getByRole("button", { name: "Save Changes" }).click();

	await page.waitForURL(/\/roasters\/[^/]+\/[^/]+$/);
	await expect(page.getByText(updatedLocation)).toBeVisible();
	await expect(page.getByRole("link", { name: updatedWebsite })).toBeVisible();

	const uri = `at://${did}/social.arabica.alpha.roaster/${rkey}`;
	await waitForIndex(uri);
	await page.goto("/my-coffee");
	await page.getByRole("button", { name: "Roasters" }).click();
	await expect(page.getByRole("link", { name })).toBeVisible();
	await expect(page.getByText(updatedLocation)).toBeVisible();

	await page.goto("/");
	await expect(page.getByText(name).first()).toBeVisible();

	await page.goto(detailURL.pathname);
	await page.getByRole("button", { name: "More options" }).click();
	page.once("dialog", (dialog) => dialog.accept());
	await page.getByRole("menuitem", { name: "Delete" }).click();
	await expect(page).toHaveURL(/\/my-coffee$/);
	await waitForIndex(uri, false);
	await page.getByRole("button", { name: "Roasters" }).click();
	await expect(page.getByRole("link", { name })).toHaveCount(0);

	await page.goto(detailURL.pathname);
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
	await expect(page.getByText("Record not found")).toBeVisible();
});

/**
 * Critical path: Entity view.
 *
 * Flow: create roaster → view roaster detail page → verify content.
 */
test("view entity detail page", async ({ authedPage: page, apiRequest, did, waitForIndex }) => {
	const resp = await apiRequest.post("/api/roasters", {
		form: {
			name: "E2E View Roaster",
			location: "Austin, TX",
			website: "https://example.com",
		},
	});
	const roaster = await resp.json();

	await waitForIndex(`at://${did}/social.arabica.alpha.roaster/${roaster.rkey}`);

	await page.goto(`/roasters/${did}/${roaster.rkey}`);
	await page.waitForLoadState("networkidle");

	await expect(page.getByText("E2E View Roaster")).toBeVisible();
	await expect(page.getByText("Austin, TX")).toBeVisible();
});
