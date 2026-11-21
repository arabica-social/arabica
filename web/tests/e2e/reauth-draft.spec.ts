import type { Page } from "@playwright/test";
import { test, expect } from "./fixtures";

const landing = "/?resume_reauth=1";
const storageKey = "arabica:reauth-draft:v1";

async function mockReauth(
  page: Page,
  did: string,
  returnTo = landing,
  viaDocument = false,
) {
  await page.route("**/reauth", async (route) => {
    const data = new URLSearchParams(route.request().postData() ?? "");
    expect(route.request().method()).toBe("POST");
    expect(data.get("handle")).toBe(did);
    expect(data.get("return_to")).toBe(returnTo);
    if (viaDocument) {
      await route.fulfill({
        contentType: "text/html",
        body: `<script>location.replace(${JSON.stringify(returnTo)})</script>`,
      });
    } else {
      await route.fulfill({ status: 303, headers: { Location: returnTo } });
    }
  });
}

async function rejectWrite(page: Page, path: string, method = "POST") {
  let writes = 0;
  await page.route(`**${path}`, async (route) => {
    if (route.request().method() !== method) return route.fallback();
    writes++;
    await route.fulfill({
      status: 401,
      json: { error: "Your session expired", code: "session_expired" },
    });
  });
  return () => writes;
}

test("reauth returns to a dirty brew, restoring references and hidden pours without another write", async ({
  authedPage: page,
  apiRequest,
  did,
}) => {
  const bean = await apiRequest.post("/api/beans", {
    form: { name: "Recovery Bean", origin: "Kenya", roast_level: "Light" },
  });
  expect(bean.ok()).toBeTruthy();
  const brewer = await apiRequest.post("/api/brewers", {
    form: { name: "Recovery V60", brewer_type: "pourover" },
  });
  expect(brewer.ok()).toBeTruthy();
  const writes = await rejectWrite(page, "/api/brews");
  await mockReauth(page, did);

  const path = "/brews/new?from=library#test-anchor";
  await page.goto(path);
  await page
    .getByRole("combobox", { name: "Search coffee beans" })
    .fill("Recovery Bean");
  await page.getByRole("option", { name: /Recovery Bean/ }).click();
  await page
    .getByRole("combobox", { name: "Search brew methods" })
    .fill("Recovery V60");
  await page.getByRole("option", { name: "Recovery V60" }).click();
  await page.getByRole("button", { name: "Detailed", exact: true }).click();
  await page.getByLabel("Coffee amount (g)").fill("18.5");
  await page.getByLabel("Tasting notes").fill("Currant and honey");
  await page.getByText("More brewing details", { exact: true }).click();
  await page.getByRole("button", { name: "Add first pour" }).click();
  await page.getByLabel("Pour 1", { exact: true }).fill("60");
  await page.getByLabel("Time (s)", { exact: true }).fill("30");
  await page.getByRole("button", { name: "Quick", exact: true }).click();
  await page.getByRole("button", { name: "Save brew" }).click();
  await page.getByRole("button", { name: "Log In Again" }).click();

  await expect(page).toHaveURL(
    (url) => url.pathname + url.search + url.hash === path,
  );
  await expect(page.getByLabel("Tasting notes")).toHaveValue(
    "Currant and honey",
  );
  await expect(
    page.getByRole("combobox", { name: "Search coffee beans" }),
  ).toHaveValue(/Recovery Bean/);
  await expect(
    page.getByRole("combobox", { name: "Search brew methods" }),
  ).toHaveValue("Recovery V60");
  await page.getByRole("button", { name: "Edit brewing details" }).click();
  await expect(page.getByLabel("Coffee amount (g)")).toHaveValue("18.5");
  await expect(page.getByLabel("Pour 1", { exact: true })).toHaveValue("60");
  await expect(page.getByLabel("Time (s)", { exact: true })).toHaveValue("30");
  expect(writes()).toBe(1);
  expect(
    await page.evaluate((key) => sessionStorage.getItem(key), storageKey),
  ).toBeNull();
});

test("reauth reopens the add-station drawer and fills its unsaved inputs", async ({
  authedPage: page,
  did,
}) => {
  const writes = await rejectWrite(page, "/api/roasters");
  await mockReauth(page, did);
  await page.goto("/add");
  await page.getByRole("button", { name: "Add a roaster" }).click();
  const drawer = page.locator("[data-drawer]");
  await drawer.getByPlaceholder("Name *").fill("Recovery Roaster");
  await drawer.getByPlaceholder("Location").fill("Portland");
  await drawer.getByRole("button", { name: "Save" }).click();
  await page.getByRole("button", { name: "Log In Again" }).click();
  await expect(drawer.getByPlaceholder("Name *")).toHaveValue(
    "Recovery Roaster",
  );
  await expect(drawer.getByPlaceholder("Location")).toHaveValue("Portland");
  expect(writes()).toBe(1);
});

test("a different account does not restore or retain the expired account's draft", async ({
  authedPage: page,
  did,
}) => {
  await mockReauth(page, did, landing, true);
  await page.route(`**${landing}`, async (route) => {
    const response = await route.fetch();
    const html = await response.text();
    const body = html.replace(
      `data-user-did="${did}"`,
      'data-user-did="did:plc:another-account"',
    );
    expect(body).not.toBe(html);
    await route.fulfill({ response, body });
  });
  await rejectWrite(page, "/api/brewers");
  await page.goto("/brewers/new");
  await page.getByRole("textbox", { name: "Name" }).fill("Private draft");
  await page.locator("main form button[type=submit]").click();
  await page.getByRole("button", { name: "Log In Again" }).click();
  await expect(page).toHaveURL((url) => url.pathname + url.search === "/");
  expect(
    await page.evaluate((key) => sessionStorage.getItem(key), storageKey),
  ).toBeNull();
  await page.goto("/brewers/new");
  await expect(page.getByRole("textbox", { name: "Name" })).toHaveValue("");
});

test("blocked storage leaves the form intact until the user chooses to continue without it", async ({
  authedPage: page,
  did,
}) => {
  const path = "/roasters/new?from=manage#test-anchor";
  await mockReauth(page, did, path);
  await rejectWrite(page, "/api/roasters");
  await page.addInitScript((key) => {
    const setItem = Storage.prototype.setItem;
    Storage.prototype.setItem = function (name, value) {
      if (name === key)
        throw new DOMException("Storage disabled", "QuotaExceededError");
      return setItem.call(this, name, value);
    };
  }, storageKey);
  await page.goto(path);
  const name = page.getByRole("textbox", { name: "Name" });
  await name.fill("Roaster to copy");
  await page.locator("main form button[type=submit]").click();
  await page.getByRole("button", { name: "Log In Again" }).click();
  await expect(
    page.getByText("This browser couldn't keep your form changes.", {
      exact: false,
    }),
  ).toBeVisible();
  await expect(name).toHaveValue("Roaster to copy");
  const response = page.waitForResponse("**/reauth");
  await page
    .getByRole("button", { name: "Log In Without Form Changes" })
    .click();
  await response;
  await expect(name).toHaveValue("");
  await expect(page).toHaveURL(
    (url) => url.pathname + url.search + url.hash === path,
  );
});

test("reauth preserves removing a pour from an existing brew without typing", async ({
  authedPage: page,
  apiRequest,
  did,
}) => {
  const bean = await apiRequest.post("/api/beans", {
    form: { name: "Edit Recovery Bean", origin: "Kenya", roast_level: "Light" },
  });
  const brewer = await apiRequest.post("/api/brewers", {
    form: { name: "Edit Recovery V60", brewer_type: "pourover" },
  });
  expect(bean.ok()).toBeTruthy();
  expect(brewer.ok()).toBeTruthy();
  const created = await apiRequest.post("/api/brews", {
    data: {
      bean_rkey: (await bean.json()).rkey,
      brewer_rkey: (await brewer.json()).rkey,
      pours: [{ water_amount: 50, time_seconds: 30 }],
    },
  });
  expect(created.ok()).toBeTruthy();
  const { brew } = await created.json();
  const writes = await rejectWrite(page, `/api/brews/${brew.rkey}`, "PUT");
  await mockReauth(page, did);
  await page.goto(`/brews/${brew.rkey}/edit`);
  await page.getByText("More brewing details", { exact: true }).click();
  await page.getByRole("button", { name: "Remove pour 1" }).click();
  await page.getByRole("button", { name: "Update brew" }).click();
  await page.getByRole("button", { name: "Log In Again" }).click();
  await expect(
    page.getByRole("button", { name: "Add first pour" }),
  ).toBeVisible();
  expect(writes()).toBe(1);
});
