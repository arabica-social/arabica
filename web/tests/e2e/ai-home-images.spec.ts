import { test, expect } from "./fixtures";

test.use({ aiAccess: true });
import type { Page } from "@playwright/test";
const entry = (page: Page) =>
  page.getByRole("textbox", { name: "Describe a brew or bean", exact: true });
async function openHome(page: Page) {
  const requests: {
    messages: {
      content: string;
      images?: { mime_type: string; data: string }[];
    }[];
  }[] = [];
  await page.route("**/api/ai/status", (route) =>
    route.fulfill({ json: { enabled: true, allowed: true } }),
  );
  await page.route("**/api/ai/chat", (route) => {
    requests.push(route.request().postDataJSON());
    return route.fulfill({
      json: {
        reply: "Homepage image received",
        kind: "brew",
        patch: {},
        choices: [],
      },
    });
  });
  await page.goto("/");
  await expect(entry(page)).toBeVisible();
  return requests;
}
async function nativeImagePaste(page: Page) {
  await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.evaluate(async () => {
    const canvas = document.createElement("canvas");
    canvas.width = 3200;
    canvas.height = 1600;
    const ctx = canvas.getContext("2d")!;
    ctx.fillStyle = "#975a35";
    ctx.fillRect(0, 0, 3200, 1600);
    const blob = await new Promise<Blob>((resolve) =>
      canvas.toBlob((blob) => resolve(blob!), "image/png"),
    );
    await navigator.clipboard.write([new ClipboardItem({ "image/png": blob })]);
  });
  await entry(page).focus();
  await page.keyboard.press("Control+V");
  await expect(
    page.getByAltText("Image 1 to send", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Send", exact: true }),
  ).toBeEnabled();
}
async function assertNoPixelsStored(page: Page) {
  const stored = await page.evaluate(() =>
    JSON.stringify({ ...localStorage, ...sessionStorage }),
  );
  expect(stored).not.toContain("data:image/");
  expect(stored).not.toContain("base64");
}
test("native homepage image paste reaches chat and remains available for follow-ups without storage", async ({
  authedPage: page,
}) => {
  const requests = await openHome(page);
  await nativeImagePaste(page);
  await expect(
    page.getByRole("button", { name: "Attach images", exact: true }),
  ).toHaveCount(0);
  const preview = page.getByAltText("Image 1 to send", { exact: true });
  await expect
    .poll(() => preview.evaluate((el) => (el as HTMLImageElement).naturalWidth))
    .toBe(2048);
  await assertNoPixelsStored(page);
  await page.getByRole("button", { name: "Send", exact: true }).click();
  await expect(page).toHaveURL(/\/chat$/);
  await expect(page.getByText("Homepage image received")).toBeVisible();
  expect(requests).toHaveLength(1);
  expect(requests[0].messages[0].content).toBe("");
  const image = requests[0].messages[0].images![0];
  expect(image.mime_type).toBe("image/jpeg");
  expect(Buffer.from(image.data, "base64").length).toBeLessThanOrEqual(
    1024 * 1024,
  );
  await page
    .getByRole("textbox", { name: "Message", exact: true })
    .fill("What origin?");
  await page.getByRole("button", { name: "Send", exact: true }).click();
  await expect.poll(() => requests.length).toBe(2);
  expect(requests[1].messages[0].images).toEqual([image]);
  await assertNoPixelsStored(page);
  await page.reload();
  await expect(page.getByText(/Image no longer available/)).toBeVisible();
});
test("homepage mixed paste retains selected-text insertion and allows image removal", async ({
  authedPage: page,
}) => {
  await openHome(page);
  await entry(page).fill("before replace after");
  await entry(page).evaluate((el) => {
    const input = el as HTMLTextAreaElement;
    input.setSelectionRange(7, 14);
    const canvas = document.createElement("canvas");
    canvas.width = 50;
    canvas.height = 50;
    const bytes = Uint8Array.from(
      atob(canvas.toDataURL("image/png").split(",")[1]),
      (c) => c.charCodeAt(0),
    );
    const data = new DataTransfer();
    data.items.add(new File([bytes], "paste.png", { type: "image/png" }));
    data.setData("text/plain", "coffee");
    input.dispatchEvent(
      new ClipboardEvent("paste", {
        clipboardData: data,
        bubbles: true,
        cancelable: true,
      }),
    );
  });
  await expect(entry(page)).toHaveValue("before coffee after");
  await expect(
    page.getByAltText("Image 1 to send", { exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Remove image 1", exact: true })
    .click();
  await expect(
    page.getByAltText("Image 1 to send", { exact: true }),
  ).toHaveCount(0);
  await entry(page).fill("");
  await expect(
    page.getByRole("button", { name: "Send", exact: true }),
  ).toBeDisabled();
});
for (const theme of ["light", "dark"]) {
  for (const width of [320, 1440]) {
    test(`homepage image previews fit ${theme} at ${width}px`, async ({
      authedPage: page,
    }) => {
      await page.setViewportSize({ width, height: 900 });
      await openHome(page);
      await page.evaluate(
        (theme) => document.documentElement.setAttribute("data-theme", theme),
        theme,
      );
      await nativeImagePaste(page);
      await expect(
        page.getByRole("heading", { name: "Community", exact: true }),
      ).toBeVisible();
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
      const remove = await page
        .getByRole("button", { name: "Remove image 1", exact: true })
        .boundingBox();
      expect(remove!.width).toBeGreaterThanOrEqual(44);
      expect(remove!.height).toBeGreaterThanOrEqual(44);
      if (theme === "dark" && width === 320)
        await page.screenshot({
          path: "/tmp/arabica-home-image-paste.png",
          fullPage: true,
        });
      await assertNoPixelsStored(page);
    });
  }
}
