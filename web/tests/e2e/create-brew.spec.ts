import { test, expect } from "./fixtures";

test("new brew form loads indexed prerequisites", async ({ authedPage: page, apiRequest, did, waitForIndex }) => {
	const suffix = Date.now().toString(36);
	const beanName = `E2E Brew Bean ${suffix}`;
	const brewerName = `E2E Brewer ${suffix}`;

	const beanResponse = await apiRequest.post("/api/beans", {
		form: { name: beanName, origin: "Ethiopia" },
	});
	expect(beanResponse.ok()).toBeTruthy();
	const bean = await beanResponse.json();
	const brewerResponse = await apiRequest.post("/api/brewers", {
		form: { name: brewerName, brewer_type: "pourover" },
	});
	expect(brewerResponse.ok()).toBeTruthy();
	const brewer = await brewerResponse.json();
	await Promise.all([
		waitForIndex(`at://${did}/social.arabica.alpha.bean/${bean.rkey}`),
		waitForIndex(`at://${did}/social.arabica.alpha.brewer/${brewer.rkey}`),
	]);

	const dataLoaded = page.waitForResponse((response) =>
		new URL(response.url()).pathname === "/api/data",
	);
	await page.goto("/brews/new");
	const dataResponse = await dataLoaded;
	expect(dataResponse.ok()).toBeTruthy();
	await expect.poll(async () =>
		page.evaluate((expectedName) => {
			const data = window.AppCache?.getCachedData?.() as { beans?: Array<{ name?: string }> } | null;
			return data?.beans?.some((bean) => bean.name === expectedName) ?? false;
		}, beanName),
	).toBe(true);
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
	await expect(page.getByRole("heading", { name: "New Brew" })).toBeVisible();
	await expect(page.getByRole("region", { name: "Coffee" })).toBeVisible();
	await expect(page.getByRole("button", { name: "Save Brew" })).toBeVisible();
	await page.goto("/my-coffee");
	await page.getByRole("button", { name: "Beans" }).click();
	await expect(page.getByText(beanName)).toBeVisible();
});
