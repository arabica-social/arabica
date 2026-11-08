import { test, expect } from "./fixtures";

for (const path of ["/chatt", "/unknown/deep/path", "/apiary"]) {
  test(`unknown page ${path} loads the app 404 and recovers home`, async ({
    page,
  }) => {
    const response = await page.goto(path);
    expect(response?.status()).toBe(404);
    expect(response?.headers()["content-type"]).toContain("text/html");
    await expect(
      page.locator('body[data-frontend="sveltekit"]'),
    ).toBeAttached();
    await expect(
      page.getByRole("heading", { name: "Page Not Found" }),
    ).toBeVisible();
    await expect(page).toHaveTitle("Page Not Found - Arabica");
    await page.getByRole("link", { name: "Back to Home" }).click();
    await expect(page).toHaveURL(/\/$/);
    await expect(
      page.getByRole("heading", { name: "Page Not Found" }),
    ).toHaveCount(0);
  });
}

test("unknown API and asset URLs do not return the app shell", async ({
  request,
}) => {
  for (const path of [
    "/api/unknown",
    "/_app/missing.js",
    "/static/missing.css",
  ]) {
    const response = await request.get(path);
    expect(response.status()).toBe(404);
    expect(response.headers()["content-type"]).not.toContain("text/html");
    expect(await response.text()).not.toContain('data-frontend="sveltekit"');
  }
});
