import { test, expect } from "./fixtures";
import type { Page } from "@playwright/test";

async function expectHomeComposition(page: Page) {
  await page.evaluate(() => document.fonts.ready);
  const width = page.viewportSize()!.width;
  const artwork = await page.locator(".cafe-journal-head").evaluate((el) => {
    const style = getComputedStyle(el, "::before");
    const header = el.getBoundingClientRect();
    const headerStyle = getComputedStyle(el);
    const intro = el.firstElementChild!.getBoundingClientRect();
    const actions = el.querySelector(".cafe-actions")!.getBoundingClientRect();
    return {
      width: parseFloat(style.width),
      height: parseFloat(style.height),
      bottom: parseFloat(style.bottom),
      opacity: style.opacity,
      mask: decodeURIComponent(style.maskImage),
      minHeight: headerStyle.minHeight,
      headerHeight: header.height,
      contentHeight: Math.max(intro.height, actions.height),
      verticalPadding:
        parseFloat(headerStyle.paddingTop) +
        parseFloat(headerStyle.paddingBottom),
      border: parseFloat(headerStyle.borderBottomWidth),
      introBottom: intro.bottom,
      actionsBottom: actions.bottom,
    };
  });
  expect(artwork.width).toBeCloseTo(
    Math.min(400, Math.max(272, width * 0.3)),
    0,
  );
  expect(artwork.height).toBe(artwork.width);
  expect(artwork.opacity).toBe("0.045");
  expect(artwork.mask).toContain("rx='5.5' ry='9'");
  expect(artwork.mask).toContain("rotate(25 12 12)");
  expect(artwork.mask).toContain("M12 3c-1.5 3 1.5 6 0 9s1.5 6 0 9");

  // Main's original crop and content-driven height. Do not push the bean
  // below the headline or add empty height to make that placement fit.
  expect(artwork.bottom).toBe(-120);
  expect(artwork.minHeight).toBe("0px");
  if (width >= 720) {
    expect(artwork.headerHeight).toBeCloseTo(
      artwork.contentHeight + artwork.verticalPadding + artwork.border,
      0,
    );
    expect(artwork.actionsBottom).toBeCloseTo(artwork.introBottom, 0);
  }

  if (width >= 1040) {
    const left = await page.locator(".cafe-rail-left").boundingBox();
    const main = await page.locator(".cafe-main").boundingBox();
    const right = await page.locator(".cafe-rail-right").boundingBox();
    expect(left).not.toBeNull();
    expect(main).not.toBeNull();
    expect(right).not.toBeNull();
    expect(left!.x + left!.width).toBeLessThan(main!.x);
    expect(main!.x + main!.width).toBeLessThan(right!.x);
    expect(main!.width).toBeLessThan(width * 0.67);
  }
  const feedHeading = await page
    .getByRole("heading", { name: "Recent activity" })
    .evaluate((el) => ({
      height: el.getBoundingClientRect().height,
      lineHeight: parseFloat(getComputedStyle(el).lineHeight),
    }));
  expect(feedHeading.height).toBeCloseTo(feedHeading.lineHeight, 0);
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
    await expect(page.locator(".cafe-journal-head")).toHaveScreenshot(
      `home-${state}-${width}-${theme}.png`,
      { animations: "disabled" },
    );
  }
}

for (const colorScheme of ["light", "dark"] as const) {
  for (const width of [390, 1040, 1440, 1920]) {
    test(`visitor home at ${width}px in ${colorScheme} mode`, async ({
      page,
    }) => {
      await page.setViewportSize({ width, height: 900 });
      await page.emulateMedia({ colorScheme });
      await page.goto("/");
      await expect(
        page.getByRole("heading", { level: 1, name: "A coffee journal" }),
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
        page
          .locator(".cafe-journal-head")
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
          name: "Good coffee deserves a record.",
        }),
      ).toBeVisible();
      await expectHomeComposition(page);
      await expectHeaderScreenshot(page, "fresh", colorScheme);
      const setup = page.getByRole("link", { name: "Finish setup" });
      await expect(setup).toHaveCount(1);
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
      if (width === 1920) {
        // Measured from main's original ready-account header, which matches
        // the reference screenshot. The former 384px minimum was too tall.
        const header = await page.locator(".cafe-journal-head").boundingBox();
        expect(header!.height).toBeCloseTo(236.95, 0);
      }
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
