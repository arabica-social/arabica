import { test, expect } from "./fixtures";
import type { Page, TestInfo } from "@playwright/test";

async function capture(page: Page, info: TestInfo, name: string) {
  await page.evaluate(() => document.fonts.ready);
  expect(await page.evaluate(() => document.documentElement.scrollWidth))
    .toBeLessThanOrEqual(page.viewportSize()!.width);
  await page.screenshot({ path: info.outputPath(`${name}.png`), fullPage: true, animations: "disabled" });
}

for (const colorScheme of ["light", "dark"] as const) {
  for (const width of [390, 1280]) {
    test.describe(`${width}px ${colorScheme} cleanup`, () => {
      test.use({ viewport: { width, height: 900 }, colorScheme });

      test("collection tools stay with navigation in empty and populated states", async ({ authedPage: page, apiRequest, did, waitForIndex }, info) => {
        await page.goto("/my-coffee");
        await expect(page.getByText("No brews recorded yet.")).toBeVisible();
        await expect(page.locator(".coffee-library-rail")).toHaveCount(0);
        const refresh = page.getByRole("button", { name: "Refresh", exact: true });
        await expect(refresh).toBeVisible();
        await expect(page.getByRole("link", { name: "Log your first brew" })).toBeVisible();
        await capture(page, info, "collection-empty");
        const response = await apiRequest.post("/api/beans", { form: { name: "Gedeb washed", origin: "Ethiopia" } });
        expect(response.ok()).toBeTruthy();
        const bean = await response.json();
        await waitForIndex(`at://${did}/social.arabica.alpha.bean/${bean.rkey}`);
        await refresh.click();
        await page.getByRole("button", { name: /^Beans\s/ }).click();
        await expect(page.getByRole("link", { name: "Gedeb washed", exact: true })).toBeVisible();
        await capture(page, info, "collection-populated");
        await page.route("**/api/manage/refresh", route => route.fulfill({ status: 400, body: "Unavailable" }));
        await refresh.click();
        await expect(page.getByText("Refresh failed", { exact: true })).toBeVisible();
        await expect(page.getByRole("link", { name: "Gedeb washed", exact: true })).toBeVisible();
      });

      test("Explore leaves room for results and discloses additional filters", async ({ authedPage: page }, info) => {
        await page.goto("/explore");
        const disclosure = page.locator("details.explore-advanced-filters");
        await expect(disclosure).not.toHaveAttribute("open", "");
        const controls = await page.locator(".explore-controls").boundingBox();
        expect(controls!.y + controls!.height).toBeLessThan(700);
        await capture(page, info, "explore");
        await disclosure.locator("summary").click();
        await expect(disclosure).toHaveAttribute("open", "");
        await page.getByLabel("Origin", { exact: true }).fill("Ethiopia");
        await page.getByRole("button", { name: "Explore", exact: true }).click();
        await expect(page).toHaveURL(/origin=Ethiopia/);
        await capture(page, info, "explore-filtered");
        await page.getByRole("button", { name: "Clear", exact: true }).click();
        await expect(page).toHaveURL(/\/explore$/);
      });

      test("record forms keep controls and ratio without explanatory sidebars", async ({ authedPage: page }, info) => {
        for (const type of ["beans", "roasters", "brewers", "grinders", "recipes"]) {
          await page.goto(`/${type}/new`);
          await expect(page.locator(".form-workspace__rail")).toHaveCount(0);
          await expect(page.getByLabel("Name", { exact: true })).toBeVisible();
          await capture(page, info, type);
        }
        const coffee = await page.getByLabel("Coffee amount in grams").boundingBox();
        const water = await page.getByLabel("Water amount in grams").boundingBox();
        if (width >= 640) expect(coffee!.y).toBe(water!.y);
        else expect(water!.y).toBeGreaterThan(coffee!.y);
        await page.getByLabel("Coffee amount in grams").fill("15");
        await page.getByLabel("Water amount in grams").fill("250");
        await expect(page.getByText(/Brew ratio 1:16.7/)).toBeVisible();
        await capture(page, info, "recipe-ratio");
      });

      test("Add is a chooser while onboarding keeps prerequisites", async ({ authedPage: page }, info) => {
        await page.goto("/add");
        await expect(page.getByRole("heading", { name: "Brewer", exact: true })).toBeVisible();
        await expect(page.locator(".station-no")).toHaveCount(0);
        await expect(page.locator(".station-tag")).toHaveCount(0);
        if (width >= 640) {
          const choices = await page.locator("article.station").evaluateAll(elements => elements.map(element => element.getBoundingClientRect().width));
          expect(Math.max(...choices) - Math.min(...choices)).toBeLessThan(2);
        }
        await capture(page, info, "add");
        await page.goto("/onboarding");
        await expect(page.getByText("required", { exact: true }).first()).toBeVisible();
        await capture(page, info, "onboarding");
        await page.goto("/about");
        await expect(page.getByRole("heading", { level: 1, name: "About Arabica" })).toBeVisible();
        await capture(page, info, "about");
        await page.goto("/join/create");
        await expect(page.getByRole("button", { name: "Log in", exact: true })).toHaveCount(1);
        await expect(page.getByRole("button", { name: "Log in here" })).toHaveCount(0);
        await capture(page, info, "join");
      });
    });
  }
}
