import { test, expect } from "./fixtures";

test("view feed and filter", async ({ authedPage: page, apiRequest }) => {
  await apiRequest.post("/api/roasters", {
    form: { name: "E2E Feed Roaster", location: "Portland, OR" },
  });

  await page.waitForTimeout(2000);

  await page.goto("/");
  await page.waitForLoadState("networkidle");
  await expect(
    page.getByRole("heading", { level: 1, name: "Community", exact: true }),
  ).toBeVisible();

  await expect(page.locator("#feed-items")).toBeAttached();

  await expect(page.getByRole("button", { name: "All" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Brews" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Beans" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Roasters" })).toBeVisible();

  const brewsFilter = page.getByRole("button", { name: "Brews" });
  await brewsFilter.click();
  await expect(page.locator("#feed-items")).toBeVisible();

  await page.getByRole("button", { name: "All" }).click();
  await expect(page.locator("#feed-items")).toBeVisible();
});

test("feed load more pagination", async ({ authedPage: page, apiRequest }) => {
  for (let i = 0; i < 3; i++) {
    await apiRequest.post("/api/roasters", {
      form: { name: `E2E Pagination Roaster ${i}` },
    });
  }

  await page.waitForTimeout(2000);
  await page.goto("/");
  await page.waitForLoadState("networkidle");

  const loadMoreBtn = page.getByRole("button", { name: /load more/i });
  if (await loadMoreBtn.isVisible().catch(() => false)) {
    await loadMoreBtn.click();
    await page.waitForTimeout(1000);
  }
});

for (const width of [320, 375, 719, 1280]) {
  test(`feed controls fit at ${width}px`, async ({ authedPage: page }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.goto("/");
    const filters = page.locator('[aria-label="Feed filters"]');
    await expect(filters.locator("button")).toHaveText([
      "All",
      "Beans",
      "Brews",
      "Roasters",
    ]);
    const bar = page.locator(".feed-filters");
    await expect(bar).toBeVisible();
    expect(await bar.evaluate((el) => el.scrollWidth <= el.clientWidth)).toBe(
      true,
    );
    for (const button of await bar
      .locator("button:visible, select:visible")
      .all()) {
      await expect(button).toBeInViewport({ ratio: 0.95 });
      const bounds = await button.boundingBox();
      expect(bounds).not.toBeNull();
      expect(bounds!.x).toBeGreaterThanOrEqual(0);
      expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(width);
    }
    await expect(bar.getByRole("combobox")).toHaveCount(0);
    if (width <= 719) {
      await expect(
        page.getByRole("button", { name: "New", exact: true }),
      ).toBeHidden();
      await expect(
        page.getByRole("button", { name: "Popular", exact: true }),
      ).toBeHidden();
      const rows = await filters
        .locator("button")
        .evaluateAll((buttons) =>
          buttons.map((button) => button.getBoundingClientRect().y),
        );
      expect(Math.max(...rows) - Math.min(...rows)).toBeLessThan(2);
    }
    await filters
      .getByRole("button", { name: "Roasters", exact: true })
      .click();
    await expect(
      filters.getByRole("button", { name: "Roasters" }),
    ).toHaveAttribute("aria-pressed", "true");
    await expect(page).toHaveURL(/type=roaster/);
    if (width <= 719) return;
    await page.getByRole("button", { name: "Popular", exact: true }).click();
    await expect(page).toHaveURL(/sort=popular/);
    await expect(
      page.getByRole("button", { name: "Popular", exact: true }),
    ).toHaveAttribute("aria-pressed", "true");
    await page.getByRole("button", { name: "New", exact: true }).click();
    await expect(
      page.getByRole("button", { name: "New", exact: true }),
    ).toHaveAttribute("aria-pressed", "true");
  });
}
