import { test, expect } from "./fixtures";

for (const colorScheme of ["light", "dark"] as const) {
  for (const width of [390, 1280]) {
    for (const [path, title] of [
      ["/about", "About Arabica"],
      ["/atproto", "The AT Protocol"],
      ["/terms", "Terms of Service"],
    ]) {
      test(`${path} keeps its journal framing at ${width}px in ${colorScheme} mode`, async ({
        page,
      }) => {
        await page.setViewportSize({ width, height: 900 });
        await page.emulateMedia({ colorScheme });
        await page.goto(path);
        await expect(
          page.getByRole("heading", { level: 1, name: title }),
        ).toBeVisible();
        await page.evaluate(() => document.fonts.ready);
        const hero = await page.locator(".info-hero").evaluate((element) => {
          const style = getComputedStyle(element);
          return {
            border: style.borderTopWidth,
            radius: parseFloat(style.borderTopLeftRadius),
            background: style.backgroundImage,
          };
        });
        expect(hero.border).toBe("1px");
        expect(hero.radius).toBeGreaterThanOrEqual(16);
        expect(hero.background).toContain("data:image/svg+xml");
        expect(
          await page.evaluate(() => document.documentElement.scrollWidth),
        ).toBeLessThanOrEqual(width);
        if (path === "/atproto") {
          await expect(
            page.getByText(
              "Do not put private information in coffee records.",
              { exact: false },
            ),
          ).toBeVisible();
        }
      });
    }
  }
}
