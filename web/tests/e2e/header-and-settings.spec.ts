import { test, expect, readControlURL, readServerURL } from "./fixtures";
import { request as playwrightRequest } from "@playwright/test";

test("header Create dropdown navigates to entity forms", async ({
  authedPage: page,
}) => {
  await page.goto("/");
  await page.waitForLoadState("networkidle");

  await expect(
    page.getByRole("link", { name: "Community" }).first(),
  ).toBeVisible();
  await expect(page.getByRole("link", { name: "Brews" }).first()).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Recipes" }).first(),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: "My Coffee" }).first(),
  ).toBeVisible();

  const createBtn = page.getByRole("button", { name: "Create new" });
  await expect(createBtn).toBeVisible();
  await createBtn.click();

  await page.getByRole("menuitem", { name: /Bean/ }).click();
  await expect(page).toHaveURL(/\/beans\/new$/);
  await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
});

test("header marks the active ledger destination", async ({
  authedPage: page,
}) => {
  await page.goto("/recipes");
  await page.waitForLoadState("networkidle");

  await expect(
    page.getByRole("link", { name: "Recipes" }).first(),
  ).toHaveAttribute("aria-current", "page");
});

test("header User dropdown navigates to settings", async ({
  authedPage: page,
}) => {
  await page.goto("/");
  await page.waitForLoadState("networkidle");

  const userBtn = page.getByRole("button", { name: "User menu" });
  await expect(userBtn).toBeVisible();
  await userBtn.click();

  await page.getByRole("menuitem", { name: "Settings" }).click();
  await expect(page).toHaveURL(/\/settings$/);
  await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
});

test("settings preferences and visibility save", async ({
  authedPage: page,
}) => {
  await page.goto("/settings");
  await page.waitForLoadState("networkidle");

  const tempSelect = page.getByLabel("Preferred temperature unit");
  await tempSelect.selectOption("celsius");
  await page.getByRole("button", { name: "Save" }).first().click();
  await expect(page.locator("#toast-region")).toContainText(
    /Preferences saved/i,
  );

  const beanVis = page.getByLabel("Bean average brew rating");
  await beanVis.selectOption("private");
  await page.getByRole("button", { name: "Save" }).nth(1).click();
  await expect(page.locator("#toast-region")).toContainText(
    /Visibility saved/i,
  );
});

test("settings theme toggle applies theme", async ({ authedPage: page }) => {
  await page.goto("/settings");
  await page.waitForLoadState("networkidle");

  await page.getByRole("button", { name: "Dark" }).click();

  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  const stored = await page.evaluate(() =>
    localStorage.getItem("arabica-theme"),
  );
  expect(stored).toBe("dark");

  await page.getByRole("button", { name: "System" }).click();
  await expect(page.locator("html")).not.toHaveAttribute("data-theme", "dark");
  const storedAfter = await page.evaluate(() =>
    localStorage.getItem("arabica-theme"),
  );
  expect(storedAfter).toBeNull();
});

test("report modal opens and submits from another user's record", async ({
  authedPage: page,
  did,
}) => {
  const controlURL = readControlURL();
  const controlReq = await playwrightRequest.newContext({
    baseURL: controlURL,
  });
  const acctResp = await controlReq.post("/accounts");
  expect(acctResp.ok()).toBeTruthy();
  const other = (await acctResp.json()) as { did: string; session_id: string };
  await controlReq.dispose();

  const appReq = await playwrightRequest.newContext({
    baseURL: readServerURL(),
    extraHTTPHeaders: {
      "X-Test-Auth-DID": other.did,
      "X-Test-Auth-Session": other.session_id,
      Accept: "application/json",
    },
  });
  const createResp = await appReq.post("/api/roasters", {
    form: { name: "E2E Report Target Roaster" },
  });
  expect(createResp.ok()).toBeTruthy();
  const roaster = await createResp.json();
  await appReq.dispose();

  await page.goto(`/roasters/${other.did}/${roaster.rkey}`);
  await page.waitForLoadState("networkidle");
  await expect(page.getByText("E2E Report Target Roaster")).toBeVisible();

  await page.getByRole("button", { name: "More options" }).click();
  await page.getByRole("menuitem", { name: "Report" }).click();

  await expect(page.getByText("Report Content").first()).toBeVisible();

  await page.getByLabel("Report reason").fill("Spam content");
  await page.getByRole("button", { name: "Submit Report" }).click();
  await expect(page.getByText("Report Submitted")).toBeVisible();
});

test("nonexistent entity view renders not-found state", async ({
  authedPage: page,
  did,
}) => {
  await page.goto(`/roasters/${did}/nonexistent-rkey-12345`);
  await page.waitForLoadState("networkidle");

  await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
  await expect(page.getByText("Record not found")).toBeVisible();
});
