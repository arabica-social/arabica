import { test, expect } from "./fixtures";
import type { Page } from "@playwright/test";

const grantedDID = `did:plc:${"a".repeat(24)}`;
const staffDID = `did:plc:${"s".repeat(24)}`;

async function management(page: Page) {
  await page.route("**/api/_mod", (route) =>
    route.fulfill({
      json: {
        HiddenRecords: [],
        AuditLog: [],
        Reports: [],
        BlockedUsers: [],
        Labels: [],
        Backups: [],
        Stats: {},
        CanHide: true,
        CanUnhide: true,
        IsAdmin: true,
      },
    }),
  );
  let entries: { did: string; granted_by: string; created_at: string }[] = [];
  let failWrite = false;
  const writes: { method: string; did: string }[] = [];
  await page.route("**/api/_mod/ai-access", async (route) => {
    const request = route.request();
    if (request.method() === "GET") {
      await route.fulfill({
        json: { entries, moderators: [{ did: staffDID }] },
      });
      return;
    }
    const { did } = request.postDataJSON();
    writes.push({ method: request.method(), did });
    if (failWrite) {
      await route.fulfill({
        status: 403,
        json: { error: "Permission denied" },
      });
      return;
    }
    if (request.method() === "PUT") {
      entries = [
        { did, granted_by: staffDID, created_at: "2026-10-31T12:00:00Z" },
      ];
    } else {
      entries = entries.filter((entry) => entry.did !== did);
    }
    await route.fulfill({ json: { success: true } });
  });
  await page.goto("/_mod");
  await page.getByRole("button", { name: "AI access", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "AI chat access" }),
  ).toBeVisible();
  await expect(page.getByText(staffDID, { exact: true })).toBeVisible();
  return {
    writes,
    fail: (value: boolean) => {
      failWrite = value;
    },
  };
}

test("moderator AI access grants and revokes explicit DIDs without removing staff access", async ({
  authedPage: page,
}) => {
  const state = await management(page);
  await expect(page.getByText("No explicit grants.")).toBeVisible();
  await page.getByLabel("Account DID", { exact: true }).fill(` ${grantedDID} `);
  await page.getByRole("button", { name: "Grant access", exact: true }).click();
  await expect(page.getByRole("status")).toHaveText("Access granted.");
  await expect(page.getByText(grantedDID, { exact: true })).toBeVisible();
  await expect(page.getByLabel("Account DID", { exact: true })).toHaveValue("");
  await page
    .getByRole("button", {
      name: `Remove access for ${grantedDID}`,
      exact: true,
    })
    .click();
  await expect(page.getByRole("status")).toHaveText("Explicit grant removed.");
  await expect(page.getByText(grantedDID, { exact: true })).toHaveCount(0);
  await expect(page.getByText(staffDID, { exact: true })).toBeVisible();
  await expect(
    page.getByRole("button", {
      name: `Remove access for ${staffDID}`,
      exact: true,
    }),
  ).toHaveCount(0);
  expect(state.writes).toEqual([
    { method: "PUT", did: grantedDID },
    { method: "DELETE", did: grantedDID },
  ]);
});

test("narrow AI access form retains input and grants after failed writes in both themes", async ({
  authedPage: page,
}) => {
  await page.setViewportSize({ width: 320, height: 812 });
  const state = await management(page);
  const field = page.getByLabel("Account DID", { exact: true });
  await field.fill(grantedDID);
  await page.getByRole("button", { name: "Grant access", exact: true }).click();
  await expect(page.getByText(grantedDID, { exact: true })).toBeVisible();
  state.fail(true);
  for (const colorScheme of ["light", "dark"] as const) {
    await page.emulateMedia({ colorScheme });
    await field.fill("did:plc:anotheraccount");
    await page
      .getByRole("button", { name: "Grant access", exact: true })
      .click();
    await expect(page.getByRole("alert")).toHaveText("Permission denied");
    await expect(field).toHaveValue("did:plc:anotheraccount");
    await page
      .getByRole("button", {
        name: `Remove access for ${grantedDID}`,
        exact: true,
      })
      .click();
    await expect(page.getByRole("alert")).toHaveText("Permission denied");
    await expect(page.getByText(grantedDID, { exact: true })).toBeVisible();
    await expect(page.getByRole("status")).toHaveText("");
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    const panel = page.getByRole("region", { name: "AI chat access" });
    await panel.screenshot({
      path: test.info().outputPath(`ai-access-${colorScheme}-320.png`),
    });
  }
});

test("denied status hides signed-in home composer and client navigation renders 404", async ({
  authedPage: page,
}) => {
  await page.route("**/api/ai/status", (route) =>
    route.fulfill({ json: { enabled: false, allowed: false } }),
  );
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Community", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("textbox", { name: "Describe a brew or bean", exact: true }),
  ).toHaveCount(0);
  await page.evaluate(() => {
    const link = document.createElement("a");
    link.href = "/chat";
    link.textContent = "Open direct chat URL";
    document.querySelector("main")!.append(link);
  });
  await page
    .getByRole("link", { name: "Open direct chat URL", exact: true })
    .click();
  await expect(page).toHaveURL(/\/chat$/);
  await expect(
    page.getByRole("heading", { name: "Page Not Found", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("The page you're looking for could not be found.", {
      exact: true,
    }),
  ).toBeVisible();
  await expect(
    page.getByRole("textbox", { name: "Describe a brew or bean", exact: true }),
  ).toHaveCount(0);
});

test("fresh signed-in accounts are denied before chat parsing and have no home composer", async ({
  authedPage: page,
  apiRequest,
}) => {
  const status = await apiRequest.get("/api/ai/status");
  expect(status.ok()).toBe(true);
  expect(await status.json()).toEqual({ enabled: false, allowed: false });
  const direct = await page.goto("/chat");
  expect(direct?.status()).toBe(404);
  expect(direct?.headers()["content-type"]).toContain("text/html");
  expect(direct?.headers()["cache-control"]).toBe("no-store");
  await expect(
    page.getByRole("heading", { name: "Page Not Found", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("The page you're looking for could not be found.", {
      exact: true,
    }),
  ).toBeVisible();
  await expect(page.locator(".chat-shell")).toHaveCount(0);
  await expect(
    page
      .locator(".card")
      .filter({
        has: page.getByRole("heading", { name: "Page Not Found", exact: true }),
      }),
  ).toBeVisible();
  await page.screenshot({
    path: test.info().outputPath("designed-404.png"),
    fullPage: true,
  });
  await page.getByRole("link", { name: "Back to Home", exact: true }).click();
  await expect(page).toHaveURL(/\/$/);
  const head = await apiRequest.head("/chat");
  expect(head.status()).toBe(404);
  expect(head.headers()["content-type"]).toContain("text/html");
  expect((await head.body()).length).toBe(0);
  const malformed = await apiRequest.post("/api/ai/chat", {
    headers: { "Content-Type": "application/json" },
    data: "{",
  });
  expect(malformed.status()).toBe(404);
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Community", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("textbox", { name: "Describe a brew or bean", exact: true }),
  ).toHaveCount(0);
});

test("anonymous status is disabled and direct chat URL returns 404", async ({
  page,
}) => {
  const status = await page.request.get("/api/ai/status");
  expect(status.ok()).toBe(true);
  expect(await status.json()).toEqual({ enabled: false, allowed: false });
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Community", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("textbox", { name: "Describe a brew or bean", exact: true }),
  ).toHaveCount(0);
  const response = await page.goto("/chat");
  expect(response?.status()).toBe(404);
  expect(response?.headers()["content-type"]).toContain("text/html");
  expect(response?.headers()["cache-control"]).toBe("no-store");
  await expect(
    page.getByRole("heading", { name: "Page Not Found", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("The page you're looking for could not be found.", {
      exact: true,
    }),
  ).toBeVisible();
  await expect(page.locator(".chat-shell")).toHaveCount(0);
  await expect(
    page
      .locator(".card")
      .filter({
        has: page.getByRole("heading", { name: "Page Not Found", exact: true }),
      }),
  ).toBeVisible();
  await page.screenshot({
    path: test.info().outputPath("designed-404.png"),
    fullPage: true,
  });
  await expect(
    page.getByRole("textbox", { name: "Describe a brew or bean", exact: true }),
  ).toHaveCount(0);
  await page.getByRole("link", { name: "Back to Home", exact: true }).click();
  await expect(page).toHaveURL(/\/$/);
});
