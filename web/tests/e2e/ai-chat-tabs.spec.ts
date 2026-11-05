import { test, expect } from "./fixtures";

test.use({ aiAccess: true });
import type { Page } from "@playwright/test";

async function twoTabs(page: Page) {
  const context = page.context();
  await context.route("**/api/ai/status", (route) =>
    route.fulfill({ json: { enabled: true, allowed: true } }),
  );
  await context.route("**/api/ai/chat", (route) =>
    route.fulfill({
      json: {
        reply: "Review the bean.",
        kind: "bean",
        patch: { name: "Two-tab bean", origin: "Kenya" },
        choices: [],
      },
    }),
  );
  await page.goto("/chat");
  await page
    .getByRole("textbox", { name: "Message", exact: true })
    .fill("A bean from Kenya");
  await page.getByRole("button", { name: "Send", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Save bean", exact: true }),
  ).toBeEnabled();
  const other = await context.newPage();
  // A suspended/background tab may not process storage events before pagehide or a user edit.
  // Keep that exact stale UI while still sharing real browser storage and Web Locks.
  await other.addInitScript(() => {
    window.addEventListener(
      "storage",
      (event) => event.stopImmediatePropagation(),
      true,
    );
  });
  await other.goto("/chat");
  await expect(
    other.getByRole("button", { name: "Save bean", exact: true }),
  ).toBeEnabled();
  return other;
}

for (const staleAction of ["pagehide", "edit"] as const) {
  test(`two tabs: stale ${staleAction} after success cannot resurrect or save a duplicate`, async ({
    authedPage: page,
    did,
  }) => {
    const other = await twoTabs(page);
    let writes = 0;
    page.context().on("request", (request) => {
      if (
        request.method() === "POST" &&
        new URL(request.url()).pathname === "/api/beans"
      )
        writes++;
    });
    await page.getByRole("button", { name: "Save bean", exact: true }).click();
    await expect(
      page.getByRole("link", { name: "View saved bean" }),
    ).toBeVisible();
    // Verify this is genuinely an independently loaded stale tab, not a reconciled one.
    await expect(
      other.getByRole("button", { name: "Save bean", exact: true }),
    ).toBeEnabled();
    if (staleAction === "edit") {
      await other.getByText("Edit details", { exact: true }).click();
      await other
        .getByRole("textbox", { name: /^Name/ })
        .fill("Stale tab overwrite");
    } else
      await other.evaluate(() => window.dispatchEvent(new Event("pagehide")));
    await expect(
      other.getByRole("link", { name: "View saved bean" }),
    ).toBeVisible();
    await other.evaluate(() => window.dispatchEvent(new Event("pagehide")));
    await other.reload();
    await expect(
      other.getByRole("link", { name: "View saved bean" }),
    ).toBeVisible();
    const canonical = await other.evaluate(
      (did) =>
        JSON.parse(
          localStorage.getItem(
            `arabica.ai.sessions.v1:${encodeURIComponent(did)}`,
          )!,
        ),
      did,
    );
    expect(canonical.sessions[0].status).toBe("saved");
    expect(canonical.sessions[0].draft.fields.name).toBe("Two-tab bean");
    expect(writes).toBe(1);
    await other.close();
  });
}

test("two tabs: simultaneous saves wait for a cross-tab lock and issue one POST", async ({
  authedPage: page,
  did,
}) => {
  const other = await twoTabs(page);
  let writes = 0;
  page.context().on("request", (request) => {
    if (
      request.method() === "POST" &&
      new URL(request.url()).pathname === "/api/beans"
    )
      writes++;
  });
  // Hold the actual browser lock before either tab starts. A module-local Map or
  // localStorage marker check/set cannot satisfy this assertion.
  await page.evaluate((did) => {
    const envelope = JSON.parse(
      localStorage.getItem(
        `arabica.ai.sessions.v1:${encodeURIComponent(did)}`,
      )!,
    );
    const name = `arabica.ai.save:${JSON.stringify([did, envelope.activeId])}`;
    const state = window as typeof window & { releaseSaveLock?: () => void };
    void navigator.locks.request(
      name,
      () =>
        new Promise<void>((resolve) => {
          state.releaseSaveLock = resolve;
        }),
    );
  }, did);
  await expect
    .poll(() =>
      page.evaluate(
        () =>
          !!(window as typeof window & { releaseSaveLock?: () => void })
            .releaseSaveLock,
      ),
    )
    .toBe(true);
  await Promise.all(
    [page, other].map((tab) =>
      tab.evaluate(() =>
        document
          .querySelector<HTMLFormElement>("form.draft-card")!
          .requestSubmit(),
      ),
    ),
  );
  await Promise.all(
    [page, other].map((tab) =>
      expect(
        tab.getByRole("button", { name: "Saving…", exact: true }),
      ).toBeDisabled(),
    ),
  );
  // Let both independent module preflights reach the held lock.
  await page.waitForTimeout(200);
  expect(writes).toBe(0);
  await page.evaluate(() =>
    (window as typeof window & { releaseSaveLock?: () => void })
      .releaseSaveLock!(),
  );
  await expect(
    page.getByRole("link", { name: "View saved bean" }),
  ).toBeVisible();
  await expect(
    other.getByRole("link", { name: "View saved bean" }),
  ).toBeVisible();
  expect(writes).toBe(1);
  await other.reload();
  await expect(
    other.getByRole("link", { name: "View saved bean" }),
  ).toBeVisible();
  expect(writes).toBe(1);
  await other.close();
});
