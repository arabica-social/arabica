import { test, expect } from "./fixtures";

test("production shell starts under CSP and supports history and direct reloads", async ({
  authedPage: page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.addInitScript(() => {
    (window as any).__cspViolations = [];
    document.addEventListener("securitypolicyviolation", (event) => {
      (window as any).__cspViolations.push(event.violatedDirective);
    });
  });
  const response = await page.goto("/about");
  expect(response?.headers()["content-security-policy"]).toContain("'nonce-");
  await expect(
    page.getByRole("heading", { name: "About Arabica", exact: true }),
  ).toBeVisible();
  const scripts = await page
    .locator("script:not([src]):not([type='application/json'])")
    .evaluateAll((nodes) =>
      nodes.map((node) => (node as HTMLScriptElement).nonce),
    );
  expect(scripts.length).toBeGreaterThanOrEqual(2);
  expect(scripts.every(Boolean)).toBe(true);
  await page.locator("header a[href='/']").first().click();
  await expect(
    page.getByRole("heading", { name: "Community", exact: true }),
  ).toBeVisible();
  await page.goBack();
  await expect(
    page.getByRole("heading", { name: "About Arabica", exact: true }),
  ).toBeVisible();
  await page.reload();
  await expect(
    page.getByRole("heading", { name: "About Arabica", exact: true }),
  ).toBeVisible();
  expect(await page.evaluate(() => (window as any).__cspViolations)).toEqual(
    [],
  );
  expect(errors).toEqual([]);
});

test("feed filters replace history without resetting focus or scroll", async ({
  authedPage: page,
}) => {
  await page.goto("/about");
  await expect(
    page.getByRole("heading", { name: "About Arabica", exact: true }),
  ).toBeVisible();
  await page.locator("header a[href='/']").first().click();
  const filter = page.getByRole("button", { name: "Brews", exact: true });
  await expect(filter).toBeVisible();
  await page.evaluate(() => {
    document.body.style.minHeight = "4000px";
  });
  const focusTarget = page.locator("header a[href='/']").first();
  await focusTarget.focus();
  await filter.evaluate((button) => {
    window.scrollTo(0, 300);
    button.click();
  });
  await expect(page).toHaveURL(/\?type=brew$/);
  await expect(focusTarget).toBeFocused();
  await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(300);
  await page.goBack();
  await expect(
    page.getByRole("heading", { name: "About Arabica", exact: true }),
  ).toBeVisible();
});
