import { test, expect } from "./fixtures";
import type { Page } from "@playwright/test";

async function expectHomeComposition(page: Page) {
  await page.evaluate(() => document.fonts.ready);
  const width = page.viewportSize()!.width;
  const masthead = page.locator(".cafe-masthead");
  await expect(
    masthead.getByRole("heading", { level: 1, name: "Community", exact: true }),
  ).toHaveCount(1);
  await expect(
    page.getByRole("heading", { name: "Recent activity" }),
  ).toHaveCount(0);
  const artwork = await masthead.evaluate((el) => {
    const style = getComputedStyle(el, "::before");
    const headerStyle = getComputedStyle(el);
    return {
      width: parseFloat(style.width),
      height: parseFloat(style.height),
      top: parseFloat(style.top),
      bottom: parseFloat(style.bottom),
      right: parseFloat(style.right),
      opacity: style.opacity,
      mask: decodeURIComponent(style.maskImage),
      overflow: headerStyle.overflow,
      minHeight: headerStyle.minHeight,
      border: headerStyle.borderBottomWidth,
    };
  });
  expect(artwork.width).toBe(width < 720 ? 240 : 320);
  expect(artwork.height).toBe(artwork.width);
  expect(artwork.opacity).toBe("0.045");
  expect(artwork.mask).toContain("rx='5.5' ry='9'");
  expect(artwork.mask).toContain("rotate(25 12 12)");
  expect(artwork.mask).toContain("M12 3c-1.5 3 1.5 6 0 9s1.5 6 0 9");
  expect(artwork.overflow).toBe("hidden");
  expect(artwork.minHeight).toBe("0px");
  expect(artwork.border).toBe("1px");
  if (width < 720) {
    // The approved mobile bean is larger, farther right, and buried below
    // the divider. Its crop must survive extra setup or visitor guidance.
    expect(artwork.bottom).toBe(-104);
    expect(artwork.right).toBe(0);
  } else {
    expect(artwork.top).toBeCloseTo(-11.2, 1);
    expect(artwork.right).toBe(48);
  }

  const title = await masthead.getByRole("heading", { level: 1 }).boundingBox();
  const action = await masthead.locator(".home-action-primary").boundingBox();
  expect(action!.height).toBeGreaterThanOrEqual(44);
  expect(
    title!.x + title!.width <= action!.x ||
      title!.y + title!.height <= action!.y,
  ).toBeTruthy();
  const header = await masthead.boundingBox();
  const feed = await page.locator(".cafe-feed-board").boundingBox();
  expect(feed!.y).toBeGreaterThanOrEqual(header!.y + header!.height);
  expect(feed!.y - (header!.y + header!.height)).toBeLessThan(32);

  if (width >= 1040) {
    const left = await page.locator(".cafe-rail-left").boundingBox();
    const main = await page.locator(".cafe-main").boundingBox();
    const right = await page.locator(".cafe-rail-right").boundingBox();
    expect(left!.x + left!.width).toBeLessThan(main!.x);
    expect(main!.x + main!.width).toBeLessThan(right!.x);
    expect(main!.width).toBeLessThan(width * 0.67);
  }
  expect(
    await page.evaluate(() => document.documentElement.scrollWidth),
  ).toBeLessThanOrEqual(width);
}

async function expectHeaderScreenshot(
  page: Page,
  state: string,
  theme: string,
) {
  const width = page.viewportSize()!.width;
  if (
    width === 390 ||
    width === 1440 ||
    (width === 1920 && state === "ready")
  ) {
    await expect(page.locator(".cafe-masthead")).toHaveScreenshot(
      `home-${state}-${width}-${theme}.png`,
      { animations: "disabled" },
    );
  }
}

for (const colorScheme of ["light", "dark"] as const) {
  for (const width of [320, 390, 719, 720, 1040, 1440, 1920]) {
    test(`visitor home at ${width}px in ${colorScheme} mode`, async ({
      page,
    }) => {
      await page.setViewportSize({ width, height: 900 });
      await page.emulateMedia({ colorScheme });
      await page.goto("/");
      await expect(
        page.getByRole("heading", { level: 1, name: "Community", exact: true }),
      ).toBeVisible();
      await expectHomeComposition(page);
      await expectHeaderScreenshot(page, "visitor", colorScheme);
      await expect(
        page.getByRole("navigation", { name: "Home shortcuts" }),
      ).toHaveCount(0);
      const links = page.getByRole("complementary", {
        name: "Community links",
      });
      await expect(links.getByRole("link", { name: "Feedback" })).toBeVisible();
      await expect(
        links.getByText(
          "Found something confusing, broken, or missing? Let us know.",
        ),
      ).toBeVisible();
      await expect(
        links.getByRole("link", { name: "About Arabica" }),
      ).toHaveAttribute("href", "/about");
      await expect(
        page.getByText("Track brews, beans, and recipes."),
      ).toBeVisible();
      await expect(
        page.getByRole("group", { name: "Feed filters" }),
      ).toHaveCount(0);
      await expect(
        page
          .getByRole("contentinfo")
          .getByRole("link", { name: "AT Protocol" }),
      ).toHaveAttribute("href", "/atproto");
      await expect(
        page.locator(".cafe-counter").getByRole("button", { name: "Log in" }),
      ).toHaveCount(1);
      await expect(
        page.getByRole("link", { name: "Create an account" }),
      ).toHaveAttribute("href", "/join/create");
      await page
        .locator(".cafe-counter")
        .getByRole("button", { name: "Log in" })
        .click();
      await expect(page.getByRole("dialog")).toBeVisible();
    });

    test(`fresh account home at ${width}px in ${colorScheme} mode`, async ({
      authedPage: page,
    }) => {
      await page.route("**/api/popular-recipes", (route) =>
        route.fulfill({ json: [] }),
      );
      await page.setViewportSize({ width, height: 900 });
      await page.emulateMedia({ colorScheme });
      await page.goto("/");
      await expect(
        page.getByRole("heading", {
          level: 1,
          name: "Community",
          exact: true,
        }),
      ).toBeVisible();
      await expectHomeComposition(page);
      await expectHeaderScreenshot(page, "fresh", colorScheme);
      const setup = page.getByRole("link", { name: "Finish setup" });
      await expect(setup).toHaveCount(1);
      await expect(page.locator(".cafe-masthead-note")).toHaveText(
        "Add a bean, brewer, and roaster to start logging brews.",
      );
      await expect(
        page.getByRole("heading", { name: "Popular recipes" }),
      ).toHaveCount(0);
      const shortcuts = page.getByRole("navigation", {
        name: "Home shortcuts",
      });
      await expect(
        shortcuts.getByRole("link", { name: "My Coffee" }),
      ).toHaveAttribute("href", "/my-coffee");
      await expect(
        shortcuts.getByRole("link", { name: "Recipes" }),
      ).toHaveAttribute("href", "/recipes");
      await setup.click();
      await expect(
        page.getByRole("heading", { name: "Set up your first brew." }),
      ).toBeVisible();
    });

    test(`ready account home at ${width}px in ${colorScheme} mode`, async ({
      authedPage: page,
    }) => {
      await page.route("**/api/onboarding", (route) =>
        route.fulfill({
          json: {
            readiness: { HasBean: true, HasBrewer: true, HasRoaster: true },
            beans: [],
            brewers: [],
            roasters: [],
            grinders: [],
          },
        }),
      );
      await page.route("**/api/incomplete-records", (route) =>
        route.fulfill({
          json: {
            records: [
              {
                EntityType: "bean",
                RKey: "bag",
                Name: "Current bag",
                MissingFields: ["origin"],
              },
            ],
          },
        }),
      );
      await page.route("**/api/popular-recipes", (route) =>
        route.fulfill({
          json: [
            {
              rkey: "v60",
              name: "Morning V60",
              author_handle: "brewer.test",
              brewer_type: "V60",
              brew_count: 3,
            },
          ],
        }),
      );
      await page.setViewportSize({ width, height: 900 });
      await page.emulateMedia({ colorScheme });
      await page.goto("/");
      await expect(
        page.locator(".cafe-counter").getByRole("link", { name: "Log Brew" }),
      ).toHaveAttribute("href", "/brews/new");
      await expectHomeComposition(page);
      await expectHeaderScreenshot(page, "ready", colorScheme);
      await expect(page.locator(".cafe-masthead-note")).toHaveCount(0);
      await expect(
        page
          .locator(".cafe-masthead")
          .getByRole("group", { name: "Feed filters" }),
      ).toBeVisible();
      await expect(
        page.getByRole("heading", { name: "Popular recipes" }),
      ).toBeVisible();
      await expect(
        page.getByRole("link", { name: /Current bag/ }),
      ).toHaveAttribute("href", "/beans/bag/edit");
      await expect(
        page.getByRole("link", { name: /Morning V60/ }),
      ).toHaveAttribute("href", "/recipes/brewer.test/v60");
      if (width < 1040) {
        const feed = await page.locator(".cafe-feed").boundingBox();
        for (const selector of [".cafe-rail-left", ".cafe-rail-right"]) {
          const rail = await page.locator(selector).boundingBox();
          expect(rail!.y).toBeGreaterThanOrEqual(feed!.y + feed!.height);
        }
      }
    });
  }
}
