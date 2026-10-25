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

  await page.getByRole("button", { name: "Detailed", exact: true }).click();
  await expect(page.getByLabel("Grind setting")).toHaveCount(0);
  const grinderCombo = page.getByRole("combobox", { name: "Search grinders" });

  // --- Select the brewer (pourover → reveals Pour-over details section). ---
  const brewerCombo = page.getByRole("combobox", {
    name: "Search brew methods",
  });
  await brewerCombo.fill(brewerName);
  await page.getByRole("option", { name: brewerName }).click();
  await grinderCombo.fill(grinderName);
  await page.getByRole("option", { name: grinderName }).click();

  await page.getByText("More brewing details", { exact: true }).click();
  await expect(
    page.getByText("Pour-over details", { exact: true }),
  ).toBeVisible();

  const brewingSection = page.getByRole("region", {
    name: "Brewing (optional)",
    exact: true,
  });
  expect(
    await brewingSection.evaluate(
      (element) => getComputedStyle(element).borderTopWidth,
    ),
  ).toBe("1px");
  await expect
    .poll(async () =>
      page
        .locator(".autofill-status")
        .evaluateAll((elements) =>
          elements.every(
            (element) =>
              element.childElementCount > 0 ||
              element.getBoundingClientRect().height === 0,
          ),
        ),
    )
    .toBe(true);
  await page.getByLabel("Coffee amount (g)").fill("18");
  await page.getByLabel("Water amount (g)").fill("300");
  await page.getByLabel("Grind setting").fill("Medium");
  // Cancel a destructive grinder removal: both the association and setting survive.
  page.once("dialog", (dialog) => dialog.dismiss());
  await grinderCombo
    .locator("..")
    .getByRole("button", { name: "Clear selection" })
    .click();
  await expect(grinderCombo).toHaveValue(grinderName);
  await expect(page.getByLabel("Grind setting")).toHaveValue("Medium");
  await page.getByRole("button", { name: "Quick", exact: true }).click();
  await expect(
    page.getByText("Brewing details added", { exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Edit brewing details" }).click();
  await expect(page.getByLabel("Grind setting")).toHaveValue("Medium");
  await page.getByLabel("Temperature (°F/°C)").fill("94");
  await page.getByLabel("Brew time (s)").fill("210");

  await page.getByLabel("Bloom water (g)").fill("50");
  await page.getByLabel("Bloom time (s)").fill("45");
  await page.getByLabel("Drawdown time (s)").fill("30");
  await page.getByLabel("Filter").fill("paper");

  await page.getByLabel("Tasting notes").fill("Bright, floral, and sweet.");

  await page.getByRole("button", { name: "Add rating" }).click();
  const ratingSlider = page.getByRole("slider", {
    name: "Rating",
    exact: true,
  });
  await expect(ratingSlider).toBeFocused();
  await expect(ratingSlider).toHaveValue("5");
  await ratingSlider.press("End");
  await ratingSlider.press("ArrowLeft");
  await ratingSlider.press("ArrowLeft");
  await expect(ratingSlider).toHaveValue("8");

  const saveResponse = page.waitForResponse(
    (response) =>
      new URL(response.url()).pathname === "/api/brews" &&
      response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Save brew" }).click();
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

  await expect(
    page.getByRole("button", { name: "Detailed", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  await page.getByText("More brewing details", { exact: true }).click();
  await expect(
    page.getByText("Pour-over details", { exact: true }),
  ).toBeVisible();

  const notesField = page.getByLabel("Tasting notes");
  await notesField.fill("Updated: more caramel than expected.");
  await page.getByRole("button", { name: "Remove rating" }).click();
  await expect(
    page.getByRole("slider", { name: "Rating", exact: true }),
  ).toHaveCount(0);

  const clearBrewer = brewerCombo.locator("..").getByRole("button", {
    name: "Clear selection",
  });
  page.once("dialog", (dialog) => dialog.dismiss());
  await clearBrewer.click();
  await expect(brewerCombo).toHaveValue(brewerName);
  await expect(page.getByLabel("Bloom water (g)")).toHaveValue("50");
  page.once("dialog", (dialog) => dialog.accept());
  await clearBrewer.click();
  await expect(brewerCombo).toHaveValue("");
  await brewerCombo.fill(brewerName);
  await page.getByRole("option", { name: brewerName }).click();
  await expect(page.getByLabel("Bloom water (g)")).toHaveValue("");
  await expect(page.getByLabel("Coffee amount (g)")).toHaveValue("18");
  await expect(page.getByLabel("Grind setting")).toHaveValue("Medium");
  await page.getByRole("button", { name: "Detailed", exact: true }).click();

  const updateResponse = page.waitForResponse(
    (response) =>
      new URL(response.url()).pathname === `/api/brews/${rkey}` &&
      response.request().method() === "PUT",
  );
  await page.getByRole("button", { name: "Update brew" }).click();
  const updatedResp = await updateResponse;
  expect(updatedResp.ok()).toBeTruthy();

  await page.waitForURL(/\/brews\/[^/]+\/[^/]+$/);

  await expect(
    page.getByText("Updated: more caramel than expected."),
  ).toBeVisible();
  expect(updatedResp.request().postDataJSON()).not.toHaveProperty("rating");
  await page.goto(`/brews/${rkey}/edit`);
  await expect(
    page.getByRole("slider", { name: "Rating", exact: true }),
  ).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Add rating" })).toBeVisible();
  await page.goto("/brews/new");
  await expect(
    page.getByRole("button", { name: "Quick", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
});
