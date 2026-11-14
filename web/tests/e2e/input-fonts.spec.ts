import { test, expect } from "./fixtures";
import type { Page } from "@playwright/test";

test.use({ aiAccess: true });

async function expectReadableInputs(page: Page) {
  const controls = page.locator(
    'input:not([type="hidden"]):not([type="checkbox"]):not([type="radio"]):not([type="range"]):not([type="file"]):visible, textarea:visible, select:visible',
  );
  expect(await controls.count()).toBeGreaterThan(0);
  for (const control of await controls.all()) {
    expect(
      await control.evaluate((el) => parseFloat(getComputedStyle(el).fontSize)),
    ).toBeGreaterThanOrEqual(16);
  }
  expect(
    await page.evaluate(() => document.documentElement.scrollWidth),
  ).toBeLessThanOrEqual(page.viewportSize()!.width + 2);
}

for (const width of [320, 390, 768]) {
  test(`editable controls stay readable at ${width}px`, async ({
    authedPage: page,
  }, testInfo) => {
    await page.setViewportSize({ width, height: 900 });
    for (const route of ["/beans/new", "/brewers/new", "/brews/new"]) {
      await page.goto(route);
      await expect(page.locator("form").first()).toBeVisible();
      if (route === "/brews/new") {
        await page
          .getByRole("button", { name: "Detailed", exact: true })
          .click();
      }
      await expectReadableInputs(page);
    }
    await page.screenshot({ path: testInfo.outputPath("brew-inputs.png") });
    await page.route("**/api/ai/status", (route) =>
      route.fulfill({ json: { enabled: true, allowed: true } }),
    );
    await page.goto("/");
    await expect(
      page.getByRole("textbox", { name: "Describe a brew or bean" }),
    ).toBeVisible();
    await expectReadableInputs(page);
    await page.goto("/chat");
    await expect(page.locator(".composer textarea")).toBeVisible();
    await expectReadableInputs(page);
    await page.screenshot({ path: testInfo.outputPath("chat-inputs.png") });
  });
}

test("desktop composer keeps its compact font", async ({
  authedPage: page,
}) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.route("**/api/ai/status", (route) =>
    route.fulfill({ json: { enabled: true, allowed: true } }),
  );
  await page.goto("/");
  const input = page.getByRole("textbox", { name: "Describe a brew or bean" });
  await expect(input).toBeVisible();
  expect(
    await input.evaluate((el) => parseFloat(getComputedStyle(el).fontSize)),
  ).toBeCloseTo(14.4, 1);
});

test("touch landscape uses readable inputs and preserves user zoom", async ({
  browser,
  authedPage,
  account,
}) => {
  await authedPage.goto("/");
  const origin = new URL(authedPage.url()).origin;
  const context = await browser.newContext({
    baseURL: origin,
    viewport: { width: 1024, height: 768 },
    hasTouch: true,
    isMobile: true,
    extraHTTPHeaders: {
      "X-Test-Auth-DID": account.did,
      "X-Test-Auth-Session": account.session_id,
      Origin: origin,
    },
  });
  try {
    const page = await context.newPage();
    await page.route("**/api/ai/status", (route) =>
      route.fulfill({ json: { enabled: true, allowed: true } }),
    );
    await page.goto("/");
    await expect(
      page.getByRole("textbox", { name: "Describe a brew or bean" }),
    ).toBeVisible();
    expect(
      await page.evaluate(
        () => matchMedia("(hover: none) and (pointer: coarse)").matches,
      ),
    ).toBe(true);
    await expectReadableInputs(page);
    const viewport = await page
      .locator('meta[name="viewport"]')
      .getAttribute("content");
    expect(viewport).not.toMatch(
      /user-scalable\s*=\s*(no|0)|maximum-scale\s*=/i,
    );
  } finally {
    await context.close();
  }
});
