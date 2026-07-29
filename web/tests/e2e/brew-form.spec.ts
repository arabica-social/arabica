import { test, expect } from "./fixtures";

test("brew form: create, view, edit", async ({
	authedPage: page,
	apiRequest,
	did,
	waitForIndex,
}) => {
	const suffix = Date.now().toString(36);
	const beanName = `E2E Brew Bean ${suffix}`;
	const grinderName = `E2E Grinder ${suffix}`;
	const brewerName = `E2E V60 ${suffix}`;

	const beanResp = await apiRequest.post("/api/beans", {
		form: { name: beanName, origin: "Ethiopia", roast_level: "Light" },
	});
	expect(beanResp.ok()).toBeTruthy();
	const bean = await beanResp.json();

	const grinderResp = await apiRequest.post("/api/grinders", {
		form: { name: grinderName, grinder_type: "Electric", burr_type: "Flat" },
	});
	expect(grinderResp.ok()).toBeTruthy();
	const grinder = await grinderResp.json();

	const brewerResp = await apiRequest.post("/api/brewers", {
		form: { name: brewerName, brewer_type: "pourover" },
	});
	expect(brewerResp.ok()).toBeTruthy();
	const brewer = await brewerResp.json();

	await Promise.all([
		waitForIndex(`at://${did}/social.arabica.alpha.bean/${bean.rkey}`),
		waitForIndex(`at://${did}/social.arabica.alpha.grinder/${grinder.rkey}`),
		waitForIndex(`at://${did}/social.arabica.alpha.brewer/${brewer.rkey}`),
	]);

	const dataLoaded = page.waitForResponse(
		(response) => new URL(response.url()).pathname === "/api/data",
	);
	await page.goto("/brews/new");
	await dataLoaded;
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
	await expect(page.getByRole("heading", { name: "New Brew" })).toBeVisible();

	// Give the app cache a moment to populate EntityCombo results.
	await expect
		.poll(async () =>
			page.evaluate((name) => {
				const data = window.AppCache?.getCachedData?.() as {
					beans?: Array<{ name?: string }>;
				} | null;
				return data?.beans?.some((b) => b.name === name) ?? false;
			}, beanName),
		)
		.toBe(true);

	const beanCombo = page.getByRole("combobox", { name: "Search coffee beans" });
	await beanCombo.fill(beanName);
	await page.getByRole("option", { name: new RegExp(beanName) }).click();

	const grinderCombo = page.getByRole("combobox", { name: "Search grinders" });
	await grinderCombo.fill(grinderName);
	await page.getByRole("option", { name: grinderName }).click();

	// --- Select the brewer (pourover → reveals Pour-over Details section). ---
	const brewerCombo = page.getByRole("combobox", { name: "Search brew methods" });
	await brewerCombo.fill(brewerName);
	await page.getByRole("option", { name: brewerName }).click();

	await expect(page.getByText("Pour-over Details")).toBeVisible();

	await page.getByLabel("Coffee Amount (grams)").fill("18");
	await page.getByLabel("Water Amount (grams)").fill("300");
	await page.getByLabel("Grind Size").fill("Medium");
	await page.getByLabel("Temperature (°F/°C)").fill("94");
	await page.getByLabel("Brew Time (seconds)").fill("210");

	await page.getByLabel("Bloom Water (grams)").fill("50");
	await page.getByLabel("Bloom Time (seconds)").fill("45");
	await page.getByLabel("Drawdown Time (seconds)").fill("30");
	await page.getByLabel("Filter").fill("paper");

	await page.getByLabel("Tasting Notes").fill("Bright, floral, and sweet.");

	// Rating slider (range input) — set via input event so Svelte binds it.
	await page.locator("#brew-rating").evaluate((el, val) => {
		const input = el as HTMLInputElement;
		input.value = String(val);
		input.dispatchEvent(new Event("input", { bubbles: true }));
	}, 8);

	const saveResponse = page.waitForResponse(
		(response) =>
			new URL(response.url()).pathname === "/api/brews" && response.request().method() === "POST",
	);
	await page.getByRole("button", { name: "Save Brew" }).click();
	const brewResp = await saveResponse;
	expect(brewResp.ok()).toBeTruthy();

	await page.waitForURL(/\/brews\/[^/]+\/[^/]+$/);
	const detailURL = new URL(page.url());
	const rkey = detailURL.pathname.split("/").at(-1);
	expect(rkey).toBeTruthy();
	await waitForIndex(`at://${did}/social.arabica.alpha.brew/${rkey}`);

	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
	await expect(page.getByText(beanName).first()).toBeVisible();
	await expect(page.getByText("8").first()).toBeVisible();
	await expect(page.getByText("Bright, floral, and sweet.")).toBeVisible();
	await expect(page.getByText("18g")).toBeVisible();
	await expect(page.getByText("300g")).toBeVisible();
	await expect(page.getByText("50g for 45s")).toBeVisible();
	await expect(page.getByRole("link", { name: brewerName })).toBeVisible();

	await page.goto(`/brews/${rkey}/edit`);
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
	await expect(page.getByRole("heading", { name: "Edit Brew" })).toBeVisible();

	await expect(page.getByText("Pour-over Details")).toBeVisible();

	const notesField = page.getByLabel("Tasting Notes");
	await notesField.fill("Updated: more caramel than expected.");
	await page.locator("#brew-rating").evaluate((el, val) => {
		const input = el as HTMLInputElement;
		input.value = String(val);
		input.dispatchEvent(new Event("input", { bubbles: true }));
	}, 6);

	const updateResponse = page.waitForResponse(
		(response) =>
			new URL(response.url()).pathname === `/api/brews/${rkey}` &&
			response.request().method() === "PUT",
	);
	await page.getByRole("button", { name: "Update Brew" }).click();
	const updatedResp = await updateResponse;
	expect(updatedResp.ok()).toBeTruthy();

	await page.waitForURL(/\/brews\/[^/]+\/[^/]+$/);

	await expect(page.getByText("Updated: more caramel than expected.")).toBeVisible();
	await expect(page.getByText("6").first()).toBeVisible();
});
