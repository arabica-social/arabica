import { test, expect } from "./fixtures";

test("Quick brew saves without a rating or preparation values", async ({
  authedPage: page,
  apiRequest,
  did,
  waitForIndex,
}) => {
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

  expect(
    (
      await apiRequest.post("/api/settings/preferences", {
        form: { brew_form_mode: "quick" },
        headers: { Accept: "application/json" },
      })
    ).ok(),
  ).toBeTruthy();
  const dataLoaded = page.waitForResponse(
    (response) => new URL(response.url()).pathname === "/api/data",
  );
  await page.goto("/brews/new");
  const dataResponse = await dataLoaded;
  expect(dataResponse.ok()).toBeTruthy();
  await expect
    .poll(async () =>
      page.evaluate((expectedName) => {
        const data = window.AppCache?.getCachedData?.() as {
          beans?: Array<{ name?: string }>;
        } | null;
        return data?.beans?.some((bean) => bean.name === expectedName) ?? false;
      }, beanName),
    )
    .toBe(true);
  await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
  await expect(page.getByRole("heading", { name: "New Brew" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Coffee" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Save brew" })).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Quick", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  for (const mode of ["Detailed", "Quick"]) {
    const preferenceSaved = page.waitForResponse(
      (response) =>
        new URL(response.url()).pathname === "/api/settings/preferences" &&
        response.request().method() === "POST",
    );
    await page.getByRole("button", { name: mode, exact: true }).click();
    const preferenceResponse = await preferenceSaved;
    expect(preferenceResponse.ok()).toBeTruthy();
    await preferenceResponse.finished();
    await page.reload();
    await expect(
      page.getByRole("button", { name: mode, exact: true }),
    ).toHaveAttribute("aria-pressed", "true");
  }
  await expect(
    page.getByRole("slider", { name: "Rating", exact: true }),
  ).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Add rating" })).toBeVisible();
  await expect(page.getByLabel("Coffee amount (g)")).toHaveCount(0);
  await page
    .getByRole("combobox", { name: "Search coffee beans" })
    .fill(beanName);
  await page.getByRole("option", { name: new RegExp(beanName) }).click();
  await page
    .getByRole("combobox", { name: "Search brew methods" })
    .fill(brewerName);
  await page.getByRole("option", { name: brewerName }).click();
  const notes = page.getByLabel("Tasting notes");
  await expect(notes).toHaveAttribute(
    "placeholder",
    "Describe the flavors, aroma, and your thoughts...",
  );
  const notesBox = await notes.boundingBox();
  const ratingButtonBox = await page
    .getByRole("button", { name: "Add rating" })
    .boundingBox();
  expect(notesBox).not.toBeNull();
  expect(ratingButtonBox).not.toBeNull();
  const notesToRatingGap =
    ratingButtonBox!.y - (notesBox!.y + notesBox!.height);
  expect(notesToRatingGap).toBeGreaterThanOrEqual(16);
  expect(notesToRatingGap).toBeLessThanOrEqual(28);
  await page.setViewportSize({ width: 390, height: 844 });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await notes.fill("Quick and sweet");
  const saved = page.waitForResponse(
    (response) =>
      new URL(response.url()).pathname === "/api/brews" &&
      response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Save brew" }).click();
  const response = await saved;
  expect(response.ok()).toBeTruthy();
  const submitted = response.request().postDataJSON();
  expect(submitted).not.toHaveProperty("rating");
  expect(submitted).not.toHaveProperty("coffee_amount");
  expect(submitted).not.toHaveProperty("grind_size");
  await page.waitForURL(/\/brews\/[^/]+\/[^/]+$/);
  await expect(page.getByText("Quick and sweet")).toBeVisible();
  await page.goto("/my-coffee");
  await page.getByRole("button", { name: "Beans" }).click();
  await expect(page.getByText(beanName)).toBeVisible();
});
