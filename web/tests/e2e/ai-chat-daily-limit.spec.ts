import { test, expect } from "./fixtures";

test.use({ aiAccess: true });

// The server owns the allowance; the client must surface its refusal, keep the
// typed message and draft, and still offer Retry.
test("daily allowance refusal keeps the draft and offers retry", async ({
  authedPage: page,
}) => {
  let requests = 0;
  await page.route("**/api/ai/status", (route) =>
    route.fulfill({ json: { enabled: true, allowed: true } }),
  );
  await page.route("**/api/ai/chat", (route) => {
    requests += 1;
    return route.fulfill({
      status: 429,
      headers: { "Content-Type": "application/json", "Retry-After": "120" },
      json: {
        error:
          "Daily assistant limit reached; try again tomorrow or start a new chat",
        code: "ai_daily_limit",
      },
    });
  });

  await page.goto("/chat");
  const composer = page.getByRole("textbox", { name: "Message", exact: true });
  await composer.fill("18 grams coffee, 250g water");
  await composer.press("Enter");

  await expect(
    page
      .getByRole("alert")
      .filter({ hasText: "Daily assistant limit reached" }),
  ).toBeVisible();
  const conversation = page.getByRole("log", { name: "Conversation" });
  await expect(conversation.getByText("18 grams coffee, 250g water")).toBeVisible();

  const retry = page.getByRole("button", { name: "Retry", exact: true });
  await expect(retry).toBeVisible();
  await retry.click();
  await expect.poll(() => requests).toBe(2);
  await expect(
    page
      .getByRole("alert")
      .filter({ hasText: "Daily assistant limit reached" }),
  ).toBeVisible();
});
