import { test, expect } from "./fixtures";

test.use({ aiAccess: true });
import type { Page } from "@playwright/test";

type Image = { mime_type: string; data: string };
type ChatRequest = {
  messages: {
    role: string;
    content: string;
    images?: Image[];
    images_unavailable?: number;
  }[];
};
const reply = {
  reply: "Image understood.",
  kind: "brew",
  patch: {},
  choices: [],
};
const composer = (page: Page) =>
  page.getByRole("textbox", { name: "Message", exact: true });
const pending = (page: Page) =>
  page.getByRole("img", { name: /^Image \d+ to send$/ });
const sent = (page: Page) =>
  page.getByRole("img", { name: /^Attached image \d+$/ });

async function openChat(page: Page, failures = 0) {
  const requests: ChatRequest[] = [];
  await page.route("**/api/ai/status", (route) =>
    route.fulfill({ json: { enabled: true, allowed: true } }),
  );
  await page.route("**/api/ai/chat", (route) => {
    requests.push(route.request().postDataJSON() as ChatRequest);
    if (failures-- > 0)
      return route.fulfill({
        status: 400,
        json: { error: "Please retry this request." },
      });
    return route.fulfill({ json: reply });
  });
  await page.goto("/chat");
  await expect(composer(page)).toBeEnabled();
  return requests;
}

// Browser-generated pixels exercise real PNG decoding and JPEG canvas encoding.
async function pngFile(page: Page, width = 80, height = 40) {
  const data = await page.evaluate(
    ({ width, height }) => {
      const canvas = document.createElement("canvas");
      canvas.width = width;
      canvas.height = height;
      const context = canvas.getContext("2d")!;
      context.fillStyle = "#975a35";
      context.fillRect(0, 0, width, height);
      context.fillStyle = "#fff";
      context.fillRect(width / 4, height / 4, width / 2, height / 2);
      return canvas.toDataURL("image/png").split(",")[1];
    },
    { width, height },
  );
  return {
    name: "coffee.png",
    mimeType: "image/png",
    buffer: Buffer.from(data, "base64"),
  };
}
async function attach(page: Page, count = 1) {
  const file = await pngFile(page);
  await page.getByLabel("Choose images", { exact: true }).setInputFiles(
    Array.from({ length: count }, (_, i) => ({
      ...file,
      name: `coffee-${i}.png`,
    })),
  );
  await expect(pending(page)).toHaveCount(count);
  await expect(
    page.getByRole("button", { name: "Attach images", exact: true }),
  ).toBeEnabled();
}
async function storage(page: Page) {
  return page.evaluate(() =>
    JSON.stringify({
      local: { ...localStorage },
      session: { ...sessionStorage },
    }),
  );
}
async function expectNoPixelsStored(page: Page, imageData?: string) {
  const stored = await storage(page);
  expect(stored).not.toContain("data:image/");
  expect(stored).not.toContain("base64");
  if (imageData) expect(stored).not.toContain(imageData);
}

for (const width of [320, 1280]) {
  for (const theme of ["light", "dark"]) {
    test(`Home Add images picker works at ${width}px in ${theme} mode`, async ({
      authedPage: page,
    }) => {
      await page.setViewportSize({ width, height: 900 });
      await page.emulateMedia({ colorScheme: theme as "light" | "dark" });
      await page.addInitScript(
        (theme) => localStorage.setItem("arabica-theme", theme),
        theme,
      );
      const requests: ChatRequest[] = [];
      await page.route("**/api/ai/status", (route) =>
        route.fulfill({ json: { enabled: true, allowed: true } }),
      );
      await page.route("**/api/ai/chat", (route) => {
        requests.push(route.request().postDataJSON() as ChatRequest);
        return route.fulfill({ json: reply });
      });
      await page.goto("/");
      const add = page.getByRole("button", { name: "Add images", exact: true });
      await expect(add).toBeVisible();
      const box = await add.boundingBox();
      expect(box!.width).toBeGreaterThanOrEqual(44);
      expect(box!.height).toBeGreaterThanOrEqual(44);
      const input = page.getByLabel("Describe a brew or bean", { exact: true });
      const padding = await input.evaluate((el) =>
        parseFloat(getComputedStyle(el).paddingLeft),
      );
      expect(padding).toBeGreaterThanOrEqual(box!.width);
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= window.innerWidth,
        ),
      ).toBe(true);
      const chooser = page.waitForEvent("filechooser");
      await add.click();
      const native = await chooser;
      expect(native.isMultiple()).toBe(true);
      const file = await pngFile(page);
      await native.setFiles([file, { ...file, name: "coffee-two.png" }]);
      await expect(pending(page)).toHaveCount(2);
      await expect(add).toBeEnabled();
      await expectNoPixelsStored(page);
      await page.getByRole("button", { name: "Send", exact: true }).click();
      await expect(page).toHaveURL(/\/chat$/);
      await expect(page.getByRole("log")).toContainText(reply.reply);
      expect(requests[0].messages[0].content).toBe("");
      expect(requests[0].messages[0].images).toHaveLength(2);
      await expectNoPixelsStored(page, requests[0].messages[0].images![0].data);
    });
  }
}

test("file picker downscales real pixels and supports image-only send without storing bytes", async ({
  authedPage: page,
}) => {
  const requests = await openChat(page);
  const file = await pngFile(page, 3200, 1600);
  const chooser = page.waitForEvent("filechooser");
  await page
    .getByRole("button", { name: "Attach images", exact: true })
    .click();
  await (await chooser).setFiles(file);
  await expect(pending(page)).toHaveCount(1);
  const preview = pending(page).first();
  await expect
    .poll(() => preview.evaluate((el) => (el as HTMLImageElement).naturalWidth))
    .toBe(2048);
  expect(
    await preview.evaluate((el) => (el as HTMLImageElement).naturalHeight),
  ).toBe(1024);
  expect(await preview.getAttribute("src")).toMatch(
    /^data:image\/jpeg;base64,/,
  );
  await expectNoPixelsStored(page);
  await expect(
    page.getByRole("button", { name: "Send", exact: true }),
  ).toBeEnabled();
  await page.getByRole("button", { name: "Send", exact: true }).click();
  await expect(sent(page)).toHaveCount(1);
  await expect(page.getByRole("log")).toContainText(reply.reply);
  expect(requests).toHaveLength(1);
  expect(requests[0].messages[0].content).toBe("");
  const image = requests[0].messages[0].images![0];
  expect(image.mime_type).toBe("image/jpeg");
  expect(Buffer.from(image.data, "base64").byteLength).toBeLessThanOrEqual(
    1024 * 1024,
  );
  await expect.poll(() => storage(page)).toContain("imageCount");
  await expectNoPixelsStored(page, image.data);
});

test("fallback image decoding works under the real page content security policy", async ({
  authedPage: page,
}) => {
  await page.addInitScript(() => {
    Object.defineProperty(window, "createImageBitmap", { value: undefined });
  });
  const requests = await openChat(page);
  await attach(page);
  await page.getByRole("button", { name: "Send", exact: true }).click();
  await expect(page.getByRole("log")).toContainText(reply.reply);
  expect(requests[0].messages[0].images).toHaveLength(1);
});

test("ordinary text paste keeps native insertion and undo", async ({
  authedPage: page,
}) => {
  await openChat(page);
  await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
  await composer(page).fill("before after");
  await composer(page).evaluate((el) =>
    (el as HTMLTextAreaElement).setSelectionRange(7, 7),
  );
  await page.evaluate(() => navigator.clipboard.writeText("coffee "));
  await composer(page).focus();
  await page.keyboard.press("Control+V");
  await expect(composer(page)).toHaveValue("before coffee after");
  await expect(pending(page)).toHaveCount(0);
  await page.keyboard.press("Control+Z");
  await expect(composer(page)).toHaveValue("before after");
});

test("mixed image and text paste preserves selected-text insertion and allows removal", async ({
  authedPage: page,
}) => {
  await openChat(page);
  const file = await pngFile(page);
  await composer(page).fill("before replace after");
  await composer(page).evaluate((el, base64) => {
    const input = el as HTMLTextAreaElement;
    input.setSelectionRange(7, 14);
    const bytes = Uint8Array.from(atob(base64), (character) =>
      character.charCodeAt(0),
    );
    const transfer = new DataTransfer();
    transfer.items.add(new File([bytes], "pasted.png", { type: "image/png" }));
    transfer.setData("text/plain", "coffee");
    input.dispatchEvent(
      new ClipboardEvent("paste", {
        clipboardData: transfer,
        bubbles: true,
        cancelable: true,
      }),
    );
  }, file.buffer.toString("base64"));
  await expect(composer(page)).toHaveValue("before coffee after");
  await expect(pending(page)).toHaveCount(1);
  await page
    .getByRole("button", { name: "Remove image 1", exact: true })
    .click();
  await expect(pending(page)).toHaveCount(0);
  await composer(page).fill("");
  await expect(
    page.getByRole("button", { name: "Send", exact: true }),
  ).toBeDisabled();
});

test("sent image bytes survive follow-up and Retry only in the live chat", async ({
  authedPage: page,
}) => {
  const requests = await openChat(page, 1);
  await attach(page);
  await page.getByRole("button", { name: "Send", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Retry", exact: true }),
  ).toBeVisible();
  const first = requests[0].messages[0].images;
  await page.getByRole("button", { name: "Retry", exact: true }).click();
  await expect(page.getByRole("log")).toContainText(reply.reply);
  expect(requests).toHaveLength(2);
  expect(requests[1].messages[0].images).toEqual(first);
  await composer(page).fill("What else is on the label?");
  await composer(page).press("Enter");
  await expect.poll(() => requests.length).toBe(3);
  expect(requests[2].messages[0].images).toEqual(first);
  await expectNoPixelsStored(page, first![0].data);
});

test("reload marks images unavailable, blocks image-only Retry, and accepts new images", async ({
  authedPage: page,
}) => {
  const requests = await openChat(page, 1);
  await attach(page);
  await page.getByRole("button", { name: "Send", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Retry", exact: true }),
  ).toBeVisible();
  await page.reload();
  await expect(
    page.getByText("Image no longer available. Attach again if needed.", {
      exact: true,
    }),
  ).toBeVisible();
  await expect(sent(page)).toHaveCount(0);
  await page.getByRole("button", { name: "Retry", exact: true }).click();
  await expect(
    page
      .getByRole("alert")
      .filter({ hasText: "Images from that message are no longer available" }),
  ).toBeVisible();
  expect(requests).toHaveLength(1);
  await attach(page);
  await page.getByRole("button", { name: "Send", exact: true }).click();
  await expect.poll(() => requests.length).toBe(2);
  expect(requests[1].messages[0]).toMatchObject({ images_unavailable: 1 });
  expect(requests[1].messages[0].images).toBeUndefined();
  expect(requests[1].messages.at(-1)!.images).toHaveLength(1);
  await expectNoPixelsStored(page);
});

test("switching chats clears pending and sent images, including when reopening the old chat", async ({
  authedPage: page,
}) => {
  const requests = await openChat(page);
  await attach(page);
  await composer(page).fill("First label");
  await composer(page).press("Enter");
  await expect(page.getByRole("log")).toContainText(reply.reply);
  await attach(page);
  await page.getByRole("button", { name: "New chat", exact: true }).click();
  await expect(pending(page)).toHaveCount(0);
  await expect(sent(page)).toHaveCount(0);
  await page.getByLabel("Chat options", { exact: true }).click();
  await page.getByText("Previous drafts", { exact: true }).click();
  await page
    .getByRole("button", { name: "Open chat: First label", exact: true })
    .click();
  await expect(sent(page)).toHaveCount(0);
  await expect(pending(page)).toHaveCount(0);
  await expect(
    page.getByText("Image no longer available. Attach again if needed.", {
      exact: true,
    }),
  ).toBeVisible();
  await composer(page).fill("Continue without the picture");
  await composer(page).press("Enter");
  await expect.poll(() => requests.length).toBe(2);
  expect(requests[1].messages[0]).toMatchObject({ images_unavailable: 1 });
  expect(requests[1].messages[0].images).toBeUndefined();
});

test("rejects unsupported images and enforces four total retained images", async ({
  authedPage: page,
}) => {
  const requests = await openChat(page);
  await page.getByLabel("Choose images", { exact: true }).setInputFiles({
    name: "unsafe.svg",
    mimeType: "image/svg+xml",
    buffer: Buffer.from('<svg xmlns="http://www.w3.org/2000/svg"/>'),
  });
  await expect(
    page
      .getByRole("alert")
      .filter({ hasText: "Choose a JPEG, PNG, or WebP image" }),
  ).toBeVisible();
  await expect(pending(page)).toHaveCount(0);
  await attach(page, 4);
  const file = await pngFile(page);
  await page.getByLabel("Choose images", { exact: true }).setInputFiles(file);
  await expect(
    page.getByRole("alert").filter({ hasText: "Up to 4 images" }),
  ).toBeVisible();
  await expect(pending(page)).toHaveCount(4);
  await page.getByRole("button", { name: "Send", exact: true }).click();
  await expect(page.getByRole("log")).toContainText(reply.reply);
  expect(requests[0].messages[0].images).toHaveLength(4);
  await page.getByLabel("Choose images", { exact: true }).setInputFiles(file);
  await expect(
    page.getByRole("alert").filter({ hasText: "Up to 4 images" }),
  ).toBeVisible();
  await expect(pending(page)).toHaveCount(0);
});

for (const theme of ["light", "dark"]) {
  for (const width of [320, 1440]) {
    test(`image composer stays inside viewport in ${theme} at ${width}px`, async ({
      authedPage: page,
    }) => {
      await page.setViewportSize({ width, height: 900 });
      await page.addInitScript(
        (theme) => localStorage.setItem("arabica-theme", theme),
        theme,
      );
      await openChat(page);
      await attach(page, 4);
      await expect
        .poll(() =>
          page.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth,
          ),
        )
        .toBe(true);
      for (const control of [
        page.getByRole("button", { name: "Attach images", exact: true }),
        page.getByRole("button", { name: "Send", exact: true }),
      ]) {
        const box = (await control.boundingBox())!;
        expect(box.x).toBeGreaterThanOrEqual(0);
        expect(box.x + box.width).toBeLessThanOrEqual(width);
        expect(box.height).toBeGreaterThanOrEqual(44);
      }
      if (theme === "dark" && width === 320)
        await page.screenshot({ path: "/tmp/arabica-image-chat.png" });
      await page.getByRole("button", { name: "Send", exact: true }).click();
      await expect(sent(page)).toHaveCount(4);
      await expect
        .poll(() =>
          page.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth,
          ),
        )
        .toBe(true);
    });
  }
}
