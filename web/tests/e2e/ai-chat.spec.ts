import { test, expect } from "./fixtures";
import type { Page } from "@playwright/test";

// Record the interaction for visual motion review alongside the contract assertions.
test.use({ video: "on" });

async function settleMotion(page: Page) {
  await page.evaluate(async () => {
    await (
      window as typeof window & { __chatTransitionFinished?: Promise<void> }
    ).__chatTransitionFinished;
    await Promise.allSettled(
      Array.from(document.querySelectorAll(".draft-card, .working-indicator"))
        .flatMap((element) => element.getAnimations())
        .filter((animation) => {
          const end = animation.effect?.getComputedTiming().endTime;
          return end !== undefined && end <= 300;
        })
        .map((animation) => animation.finished),
    );
  });
}

async function assertCompactComposer(page: Page, name: string) {
  await page.evaluate(() => document.fonts.ready);
  const input = page.getByRole("textbox", { name, exact: true });
  const send = page.getByRole("button", { name: "Send", exact: true });
  const original = await input.inputValue();
  const base = await input.evaluate((el) => el.getBoundingClientRect().height);
  expect(await input.evaluate((el) => getComputedStyle(el).resize)).toBe(
    "none",
  );
  const inputBox = (await input.boundingBox())!;
  const sendBox = (await send.locator("span").boundingBox())!;
  expect((await send.boundingBox())!.width).toBeGreaterThanOrEqual(44);
  expect((await send.boundingBox())!.height).toBeGreaterThanOrEqual(44);
  expect(sendBox.width).toBe(28);
  expect(sendBox.height).toBe(28);
  expect(
    await send
      .locator("span")
      .evaluate((el) => getComputedStyle(el).borderRadius),
  ).toBe("50%");
  expect(
    Math.abs(sendBox.y + sendBox.height / 2 - inputBox.y - inputBox.height / 2),
  ).toBeLessThan(1);
  const viewport = page.viewportSize()!;
  await input.fill("coffee ".repeat(24));
  await expect
    .poll(() => input.evaluate((el) => el.getBoundingClientRect().height))
    .toBeGreaterThan(base);
  const wrapped = await input.evaluate(
    (el) => el.getBoundingClientRect().height,
  );
  await page.setViewportSize({
    width: viewport.width < 500 ? 1440 : 320,
    height: viewport.height,
  });
  if (viewport.width < 500)
    await expect
      .poll(() => input.evaluate((el) => el.getBoundingClientRect().height))
      .toBeLessThan(wrapped);
  else
    await expect
      .poll(() => input.evaluate((el) => el.getBoundingClientRect().height))
      .toBeGreaterThan(wrapped);
  await page.setViewportSize(viewport);
  await expect
    .poll(() => input.evaluate((el) => el.getBoundingClientRect().height))
    .toBe(wrapped);
  await input.fill("Coffee with peach and black tea. ".repeat(24));
  await expect
    .poll(() => input.evaluate((el) => el.getBoundingClientRect().height))
    .toBeGreaterThan(base);
  await input.fill("");
  await expect
    .poll(() => input.evaluate((el) => el.getBoundingClientRect().height))
    .toBe(base);
  await input.fill(original);
}

async function openEditor(page: Page) {
  const editor = page.locator("details.manual-editor");
  await expect(editor).toBeAttached();
  if (!(await editor.evaluate((el) => (el as HTMLDetailsElement).open)))
    await editor.locator(":scope > summary").click();
}

// AI responses are deterministic here; record creation still goes through the
// actual Go mutation handler and test PDS. Provider validation is tested in Go.
test("home chat draft survives reload and writes only on Save", async ({
  authedPage: page,
}) => {
  const name = `Chat bean ${Date.now().toString(36)}`;
  let writes = 0;
  page.on("request", (request) => {
    if (
      request.method() === "POST" &&
      /\/api\/(beans|brews|grinders|brewers|roasters)$/.test(
        new URL(request.url()).pathname,
      )
    )
      writes++;
  });
  await page.route("**/api/ai/status", (route) =>
    route.fulfill({ json: { enabled: true } }),
  );
  await page.route("**/api/ai/chat", (route) =>
    route.fulfill({
      json: {
        reply: "Bean details added.",
        kind: "bean",
        patch: { name, origin: "Colombia", process: "Washed" },
        choices: [],
      },
    }),
  );
  await page.goto("/");
  const entry = page.getByRole("textbox", { name: "Describe a brew or bean" });
  await expect(entry).toBeVisible();
  await entry.fill(`Add ${name}, Colombia, washed`);
  await entry.press("Enter");
  await expect(page).toHaveURL(/\/chat$/);
  await openEditor(page);
  await expect(
    page.getByRole("textbox", { name: /^Name(?: \*)?$/ }),
  ).toHaveValue(name);
  await page
    .getByRole("textbox", { name: /^Origin(?: \*)?$/ })
    .fill("Colombia, Huila");
  await expect(
    page.getByRole("textbox", { name: /^Origin(?: \*)?$/ }),
  ).toHaveValue("Colombia, Huila");
  expect(writes).toBe(0);
  await page.reload();
  await openEditor(page);
  await expect(
    page.getByRole("textbox", { name: /^Name(?: \*)?$/ }),
  ).toHaveValue(name);
  await expect(
    page.getByRole("textbox", { name: /^Origin(?: \*)?$/ }),
  ).toHaveValue("Colombia, Huila");
  expect(writes).toBe(0);
  await page.setViewportSize({ width: 390, height: 844 });
  await expect
    .poll(() =>
      page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    )
    .toBe(true);
  const saved = page.waitForResponse(
    (response) =>
      new URL(response.url()).pathname === "/api/beans" &&
      response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Save bean", exact: true }).click();
  const response = await saved;
  expect(response.ok(), await response.text()).toBe(true);
  expect(response.request().postDataJSON()).toMatchObject({
    name,
    origin: "Colombia, Huila",
    process: "Washed",
  });
  expect(writes).toBe(1);
});

test("chat direct load is available without provider configuration", async ({
  authedPage: page,
}) => {
  await page.route("**/api/ai/status", (route) =>
    route.fulfill({ json: { enabled: false } }),
  );
  await page.goto("/chat");
  await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
  await expect(
    page.getByText(/not configured|unavailable|not available/i).first(),
  ).toBeVisible();
});

test("library choices fill a brew without writing related records", async ({
  authedPage: page,
  apiRequest,
  did,
  waitForIndex,
}) => {
  const suffix = Date.now().toString(36);
  const beanResponse = await apiRequest.post("/api/beans", {
    data: { name: `Chat Ethiopia ${suffix}`, origin: "Ethiopia" },
  });
  const brewerResponse = await apiRequest.post("/api/brewers", {
    data: { name: `Chat V60 ${suffix}`, brewer_type: "pourover" },
  });
  expect(beanResponse.ok()).toBe(true);
  expect(brewerResponse.ok()).toBe(true);
  const bean = await beanResponse.json();
  const brewer = await brewerResponse.json();
  await Promise.all([
    waitForIndex(`at://${did}/social.arabica.alpha.bean/${bean.rkey}`),
    waitForIndex(`at://${did}/social.arabica.alpha.brewer/${brewer.rkey}`),
  ]);
  const writes: string[] = [];
  page.on("request", (request) => {
    const path = new URL(request.url()).pathname;
    if (
      request.method() === "POST" &&
      /^\/api\/(beans|brews|grinders|brewers|roasters)$/.test(path)
    )
      writes.push(path);
  });
  await page.route("**/api/ai/status", (route) =>
    route.fulfill({ json: { enabled: true } }),
  );
  await page.route("**/api/ai/chat", (route) =>
    route.fulfill({
      json: {
        reply: "Choose the bean and brewer.",
        kind: "brew",
        patch: { coffee_amount: 15, water_amount: 250 },
        choices: [
          {
            label: `${bean.name} · ${brewer.name}`,
            patch: { bean_rkey: bean.rkey, brewer_rkey: brewer.rkey },
            brewer_type: "pourover",
          },
        ],
      },
    }),
  );
  await page.goto("/chat");
  await page
    .getByRole("textbox", { name: "Message", exact: true })
    .fill("V60 with Ethiopian beans, 15g to 250g");
  await page
    .getByRole("textbox", { name: "Message", exact: true })
    .press("Enter");
  const match = page.getByRole("button", {
    name: `${bean.name} · ${brewer.name}`,
    exact: true,
  });
  const rest = await match.evaluate(
    (el) => getComputedStyle(el).backgroundColor,
  );
  await match.hover();
  await expect
    .poll(() => match.evaluate((el) => getComputedStyle(el).backgroundColor))
    .not.toBe(rest);
  await match.focus();
  await page.keyboard.press("Tab");
  await page.keyboard.press("Shift+Tab");
  expect(await match.evaluate((el) => getComputedStyle(el).outlineStyle)).toBe(
    "solid",
  );
  expect(writes).toEqual([]);
  await match.click();
  await expect(match).toHaveCount(0);
  await openEditor(page);
  await expect(
    page.getByRole("combobox", { name: /^Bean(?: \*)?$/ }),
  ).toHaveValue(bean.rkey);
  await expect(
    page.getByRole("combobox", { name: /^Brewer(?: \*)?$/ }),
  ).toHaveValue(brewer.rkey);
  await expect(
    page.getByRole("spinbutton", { name: "Coffee (g)", exact: true }),
  ).toHaveValue("15");
  expect(writes).toEqual([]);
  const saved = page.waitForResponse(
    (response) =>
      new URL(response.url()).pathname === "/api/brews" &&
      response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Save brew", exact: true }).click();
  const response = await saved;
  expect(response.ok(), await response.text()).toBe(true);
  expect(response.request().postDataJSON()).toMatchObject({
    bean_rkey: bean.rkey,
    brewer_rkey: brewer.rkey,
    coffee_amount: 15,
    water_amount: 250,
  });
  expect(writes).toEqual(["/api/brews"]);
  await expect(
    page.getByRole("link", { name: "View saved brew", exact: true }),
  ).toBeVisible();
});

for (const viewport of [
  { width: 1440, height: 1000, name: "desktop" },
  { width: 390, height: 844, name: "mobile" },
  { width: 320, height: 700, name: "small-mobile" },
]) {
  test(`long chat stays centered with stored drafts tucked into options (${viewport.name})`, async ({
    authedPage: page,
  }) => {
    await page.setViewportSize(viewport);
    await page.addInitScript(() => {
      localStorage.setItem("arabica-theme", "dark");
    });
    await page.route("**/api/ai/status", (route) =>
      route.fulfill({ json: { enabled: true } }),
    );
    await page.route("**/api/data", (route) =>
      route.fulfill({
        json: {
          beans: [{ rkey: "october", name: "October Fix" }],
          brewers: [{ rkey: "orea", name: "Orea 01", brewer_type: "pourover" }],
          grinders: [],
          roasters: [],
        },
      }),
    );
    await page.route("**/api/ai/chat", (route) => {
      const request = route.request().postDataJSON();
      const second = request.messages.some(
        (message: { content: string }) =>
          message.content === "A washed bean from Kenya",
      );
      return route.fulfill({
        json: second
          ? {
              reply: "The bean draft is ready to review.",
              kind: "bean",
              patch: { name: "Kenya washed", origin: "Kenya" },
              choices: [],
            }
          : {
              reply:
                "I found October Fix and your Orea 01. The draft uses 18 grams of coffee and 300 grams of water.\n\nReview the measurements and preparation details below. Nothing has been saved yet.",
              kind: "brew",
              brewer_type: "pourover",
              patch: {
                bean_rkey: "october",
                brewer_rkey: "orea",
                coffee_amount: 18,
                water_amount: 300,
                temperature: 198,
                time_seconds: 180,
              },
              choices: [],
            },
      });
    });
    let writes = 0;
    page.on("request", (request) => {
      if (
        request.method() === "POST" &&
        /\/api\/(beans|brews|grinders|brewers|roasters)$/.test(
          new URL(request.url()).pathname,
        )
      )
        writes++;
    });
    await page.goto("/chat");
    await page.evaluate(() =>
      document.documentElement.setAttribute("data-theme", "dark"),
    );
    const composer = page.getByRole("textbox", {
      name: "Message",
      exact: true,
    });
    await composer.fill("October Fix, 3-pour Orea recipe.");
    await composer.press("Enter");
    await expect(
      page.getByRole("button", { name: "Save brew", exact: true }),
    ).toBeAttached();
    for (const text of [
      "Use 18 grams coffee.",
      "Use 300 grams water.",
      "Total time is three minutes.",
    ]) {
      await composer.fill(text);
      await composer.press("Enter");
      await expect(
        page.getByRole("button", { name: "Stop", exact: true }),
      ).toHaveCount(0);
      await expect(composer).toBeEnabled();
    }
    await composer.fill("Keep this unsent follow-up");
    await page.getByRole("button", { name: "New chat", exact: true }).click();
    await expect(page.getByRole("log")).not.toContainText(
      "October Fix, 3-pour Orea recipe.",
    );
    await composer.fill("A washed bean from Kenya");
    await composer.press("Enter");
    await expect(
      page.getByRole("button", { name: "Save bean", exact: true }),
    ).toBeAttached();
    await expect(
      page.getByRole("button", { name: "Chats", exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("complementary", { name: "Chat history", exact: true }),
    ).toHaveCount(0);
    await page.getByLabel("Chat options", { exact: true }).click();
    await page.getByText("Previous drafts", { exact: true }).click();
    const historyMenu = (await page.locator(".options-content").boundingBox())!;
    expect(historyMenu.x).toBeGreaterThanOrEqual(0);
    expect(historyMenu.x + historyMenu.width).toBeLessThanOrEqual(
      viewport.width,
    );
    expect(historyMenu.y + historyMenu.height).toBeLessThanOrEqual(
      viewport.height,
    );
    await settleMotion(page);
    await page.screenshot({
      path: `/tmp/arabica-chat-stored-drafts-${viewport.name}.png`,
    });
    await page
      .getByRole("button", {
        name: "Open chat: October Fix, 3-pour Orea recipe.",
        exact: true,
      })
      .click();
    await expect(composer).toHaveValue("Keep this unsent follow-up");
    await expect(page.getByRole("log")).not.toContainText(
      "A washed bean from Kenya",
    );
    await expect(
      page.getByRole("button", { name: "Save brew", exact: true }),
    ).toBeAttached();
    await page.reload();
    await expect(composer).toHaveValue("Keep this unsent follow-up");
    const scroll = page.getByRole("region", {
      name: "Chat content",
      exact: true,
    });
    const dimensions = await scroll.evaluate((el) => ({
      height: el.clientHeight,
      full: el.scrollHeight,
      width: document.documentElement.scrollWidth,
      pageHeight: document.documentElement.scrollHeight,
      viewport: window.innerHeight,
      transcriptMax: getComputedStyle(el.querySelector(".conversation")!)
        .maxHeight,
      scrollbar: getComputedStyle(el).scrollbarWidth,
    }));
    expect(
      await page
        .getByRole("navigation", { name: "Chat actions" })
        .evaluate((el) => el.getBoundingClientRect().height),
    ).toBeLessThanOrEqual(72);
    expect(dimensions.width).toBeLessThanOrEqual(viewport.width);
    expect(dimensions.pageHeight).toBeLessThanOrEqual(viewport.height + 1);
    expect(dimensions.height).toBeGreaterThan(viewport.height * 0.45);
    expect(dimensions.full).toBeGreaterThan(dimensions.height);
    expect(dimensions.transcriptMax).toBe("none");
    expect(dimensions.scrollbar).toBe("thin");
    const composerBox = await composer.boundingBox();
    expect(composerBox).not.toBeNull();
    const centered = (await page.locator(".composer").boundingBox())!;
    expect(
      Math.abs(centered.x + centered.width / 2 - viewport.width / 2),
    ).toBeLessThan(1);
    const mobileNav = page.getByRole("navigation", {
      name: "Mobile navigation",
    });
    const bottom = (await mobileNav.isVisible())
      ? (await mobileNav.boundingBox())!.y
      : viewport.height;
    expect(composerBox!.y + composerBox!.height).toBeLessThanOrEqual(
      bottom + 1,
    );
    await scroll.evaluate((el) => {
      el.scrollTop = 0;
    });
    await settleMotion(page);
    await page.screenshot({
      path: `/tmp/arabica-chat-${viewport.name}-conversation-dark.png`,
    });
    await scroll.focus();
    await page.keyboard.press("PageDown");
    await expect
      .poll(() => scroll.evaluate((el) => el.scrollTop))
      .toBeGreaterThan(0);
    await page
      .getByRole("button", { name: "Save brew", exact: true })
      .scrollIntoViewIfNeeded();
    await expect(
      page.getByRole("button", { name: "Save brew", exact: true }),
    ).toBeInViewport();
    await expect(composer).toBeInViewport();
    expect(writes).toBe(0);
    await scroll.evaluate((el) => el.blur());
    await settleMotion(page);
    await page.screenshot({
      path: `/tmp/arabica-chat-${viewport.name}-dark.png`,
    });
  });
}

for (const theme of ["light", "dark"] as const) {
  test(`Community and chat share the same compact circular send control (${theme})`, async ({
    authedPage: page,
  }) => {
    await page.setViewportSize({
      width: theme === "dark" ? 320 : 1440,
      height: theme === "dark" ? 700 : 1000,
    });
    await page.addInitScript(
      (theme) => localStorage.setItem("arabica-theme", theme),
      theme,
    );
    await page.route("**/api/ai/status", (route) =>
      route.fulfill({ json: { enabled: true } }),
    );
    await page.route("**/api/ai/chat", (route) =>
      route.fulfill({
        json: {
          reply: "What origin should I use?",
          kind: "bean",
          patch: { name: "October Fix" },
          choices: [],
        },
      }),
    );
    await page.goto("/");
    const send = page.getByRole("button", { name: "Send", exact: true });
    await page
      .getByRole("textbox", { name: "Describe a brew or bean" })
      .fill("October Fix");
    await expect(send).toBeEnabled();
    await expect(send).toHaveText("↑");
    await assertCompactComposer(page, "Describe a brew or bean");
    const masthead = (await page.locator(".cafe-masthead").boundingBox())!;
    const prompt = (await page
      .getByRole("textbox", { name: "Describe a brew or bean" })
      .boundingBox())!;
    const filters = (await page
      .getByRole("group", { name: "Feed filters" })
      .boundingBox())!;
    expect(prompt.y).toBeGreaterThan(masthead.y);
    expect(prompt.y + prompt.height).toBeLessThan(masthead.y + masthead.height);
    expect(filters.y).toBeGreaterThan(masthead.y + masthead.height);
    expect(
      await page
        .locator(".cafe-masthead")
        .evaluate((el) => getComputedStyle(el).borderBottomWidth),
    ).toBe("1px");
    await page.screenshot({
      path: `/tmp/arabica-community-prompt-first-${theme}.png`,
    });
    const communityStyle = await send.locator("span").evaluate((el) => {
      const style = getComputedStyle(el);
      return {
        background: style.backgroundColor,
        color: style.color,
        width: style.width,
        height: style.height,
        radius: style.borderRadius,
      };
    });
    await send.click();
    await expect(page).toHaveURL(/\/chat$/);
    await expect(
      page.getByRole("button", { name: "Stop", exact: true }),
    ).toHaveCount(0);
    await page
      .getByRole("textbox", { name: "Message", exact: true })
      .fill("Follow up");
    await expect(send).toBeEnabled();
    await expect(send).toHaveText("↑");
    await assertCompactComposer(page, "Message");
    const chatStyle = await send.locator("span").evaluate((el) => {
      const style = getComputedStyle(el);
      return {
        background: style.backgroundColor,
        color: style.color,
        width: style.width,
        height: style.height,
        radius: style.borderRadius,
      };
    });
    expect(chatStyle).toEqual(communityStyle);
    await page.getByRole("button", { name: "New chat", exact: true }).click();
    await expect(page.locator(".chat-main.empty-thread")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Add details manually", exact: true }),
    ).toBeVisible();
    await expect(send).toBeDisabled();
    await settleMotion(page);
    await page.screenshot({ path: `/tmp/arabica-chat-empty-${theme}.png` });
  });
}

test("saved recipe choices preserve source details through reload and only Save writes a brew", async ({
  authedPage: page,
  apiRequest,
  did,
  waitForIndex,
}) => {
  const suffix = Date.now().toString(36);
  const [beanResponse, brewerResponse] = await Promise.all([
    apiRequest.post("/api/beans", {
      data: { name: `Recluse bean ${suffix}`, origin: "Colombia" },
    }),
    apiRequest.post("/api/brewers", {
      data: { name: `Orea ${suffix}`, brewer_type: "pourover" },
    }),
  ]);
  expect(beanResponse.ok()).toBe(true);
  expect(brewerResponse.ok()).toBe(true);
  const bean = await beanResponse.json();
  const brewer = await brewerResponse.json();
  const pours = [
    { water_amount: 70, time_seconds: 0 },
    { water_amount: 130, time_seconds: 40 },
    { water_amount: 100, time_seconds: 90 },
  ];
  const recipeResponse = await apiRequest.post("/api/recipes", {
    data: {
      name: `Orea 3 pour ${suffix}`,
      brewer_rkey: brewer.rkey,
      brewer_type: "pourover",
      coffee_amount: 18,
      water_amount: 300,
      pours,
      notes: "Use 93°C water. Do not invent a grinder setting.",
    },
  });
  expect(recipeResponse.ok(), await recipeResponse.text()).toBe(true);
  const recipe = await recipeResponse.json();
  await Promise.all([
    waitForIndex(`at://${did}/social.arabica.alpha.bean/${bean.rkey}`),
    waitForIndex(`at://${did}/social.arabica.alpha.brewer/${brewer.rkey}`),
    waitForIndex(`at://${did}/social.arabica.alpha.recipe/${recipe.rkey}`),
  ]);
  const writes: string[] = [];
  page.on("request", (request) => {
    const path = new URL(request.url()).pathname;
    if (
      request.method() === "POST" &&
      /^\/api\/(beans|brews|grinders|brewers|roasters|recipes)$/.test(path)
    )
      writes.push(path);
  });
  await page.route("**/api/ai/status", (route) =>
    route.fulfill({ json: { enabled: true } }),
  );
  const choiceLabel = `${recipe.name} · ${brewer.name}`;
  await page.route("**/api/ai/chat", (route) =>
    route.fulfill({
      json: {
        reply: "Choose the saved recipe to use its details.",
        kind: "brew",
        patch: { bean_rkey: bean.rkey },
        choices: [
          {
            label: choiceLabel,
            patch: {
              recipe_rkey: recipe.rkey,
              brewer_rkey: brewer.rkey,
              coffee_amount: 18,
              water_amount: 300,
              pours,
            },
            brewer_type: "pourover",
          },
        ],
      },
    }),
  );
  await page.goto("/chat");
  await page
    .getByRole("textbox", { name: "Message", exact: true })
    .fill("brew with bean from Recluse, Orea 3 pour recipe");
  await page.getByRole("button", { name: "Send", exact: true }).click();
  const choice = page.getByRole("button", { name: choiceLabel, exact: true });
  await expect(choice).toBeVisible();
  await expect(
    page.getByRole("spinbutton", { name: "Coffee (g)", exact: true }),
  ).toHaveCount(0);
  expect(writes).toEqual([]);
  await choice.click();
  await openEditor(page);
  await expect(
    page.getByRole("combobox", { name: "Recipe", exact: true }),
  ).toHaveCount(0);
  await expect(page.locator(".draft-preview")).toContainText(recipe.name);
  await expect(
    page.getByRole("combobox", { name: /^Brewer(?: \*)?$/ }),
  ).toHaveValue(brewer.rkey);
  await expect(
    page.getByRole("spinbutton", { name: "Coffee (g)", exact: true }),
  ).toHaveValue("18");
  await expect(
    page.getByRole("spinbutton", { name: "Water (g)", exact: true }),
  ).toHaveValue("300");
  await expect(
    page.getByText("Recipe instructions", { exact: true }),
  ).toHaveCount(0);
  await expect(
    page.getByText("Use 93°C water. Do not invent a grinder setting.", {
      exact: true,
    }),
  ).toHaveCount(0);
  expect(writes).toEqual([]);
  await page.reload();
  await openEditor(page);
  await expect(
    page.getByRole("combobox", { name: "Recipe", exact: true }),
  ).toHaveCount(0);
  await expect(page.locator(".draft-preview")).toContainText(recipe.name);
  await expect(page.locator("details.more-details")).toHaveCount(0);
  await expect(
    page.getByRole("spinbutton", { name: "Pour 1 water (g)", exact: true }),
  ).toHaveCount(0);
  await expect(page.locator(".draft-preview")).toContainText("70 g");
  await expect(page.locator(".draft-preview")).toContainText("130 g");
  await page
    .getByRole("spinbutton", { name: "Coffee (g)", exact: true })
    .fill("19");
  await expect(page.locator(".draft-preview")).toContainText("19 g");
  expect(writes).toEqual([]);
  const saved = page.waitForResponse(
    (response) =>
      new URL(response.url()).pathname === "/api/brews" &&
      response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Save brew", exact: true }).click();
  const response = await saved;
  expect(response.ok(), await response.text()).toBe(true);
  expect(response.request().postDataJSON()).toMatchObject({
    bean_rkey: bean.rkey,
    brewer_rkey: brewer.rkey,
    recipe_rkey: recipe.rkey,
    coffee_amount: 19,
    water_amount: 300,
    pours,
  });
  expect(writes).toEqual(["/api/brews"]);
  await expect(
    page.getByRole("link", { name: "View saved brew", exact: true }),
  ).toBeVisible();
});

for (const viewport of [
  { name: "desktop", width: 1440, height: 1000 },
  { name: "phone", width: 320, height: 700 },
]) {
  test(`optional brew details preserve real values without saving ghost text (${viewport.name})`, async ({
    authedPage: page,
  }) => {
    await page.setViewportSize(viewport);
    await page.addInitScript(() =>
      localStorage.setItem("arabica-theme", "dark"),
    );
    const writes: string[] = [];
    page.on("request", (request) => {
      if (
        request.method() === "POST" &&
        /^\/api\/(beans|brews|grinders|brewers|recipes|roasters)$/.test(
          new URL(request.url()).pathname,
        )
      )
        writes.push(request.url());
    });
    await page.route("**/api/ai/status", (route) =>
      route.fulfill({ json: { enabled: true } }),
    );
    await page.route("**/api/data", (route) =>
      route.fulfill({
        json: {
          beans: [{ rkey: "fix", name: "October Fix" }],
          brewers: [{ rkey: "orea", name: "Orea", brewer_type: "pourover" }],
          grinders: [{ rkey: "k6", name: "K6" }],
          recipes: [],
        },
      }),
    );
    await page.route("**/api/ai/chat", (route) =>
      route.fulfill({
        json: {
          reply: "Dose updated.",
          kind: "brew",
          patch: { coffee_amount: 21 },
          choices: [],
        },
      }),
    );
    await page.goto("/chat");
    await page
      .getByRole("button", { name: "Add details manually", exact: true })
      .click();
    await expect(
      page.getByRole("button", { name: "Quick", exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Detailed", exact: true }),
    ).toHaveCount(0);
    await expect(page.getByLabel("Method", { exact: true })).toHaveCount(0);
    await expect(
      page.getByLabel("Coffee (g)", { exact: true }),
    ).toHaveAttribute("placeholder", "e.g. 18");
    await expect(page.getByLabel("Coffee (g)", { exact: true })).toHaveValue(
      "",
    );
    await expect(page.getByLabel("Water (g)", { exact: true })).toHaveValue("");
    await expect(
      page.getByLabel("Temperature (°F/°C)", { exact: true }),
    ).toBeVisible();
    await page.getByLabel("Brewer *", { exact: true }).selectOption("orea");
    await page.getByLabel("Coffee (g)", { exact: true }).fill("18");
    await page.getByLabel("Grinder", { exact: true }).selectOption("k6");
    await expect(
      page.getByLabel("Grind setting", { exact: true }),
    ).toHaveAttribute("placeholder", "e.g. 18, Medium, 3.5, Fine");
    await expect(
      page.getByText("More brewing details", { exact: true }),
    ).toHaveCount(0);
    await page.getByLabel("Temperature (°F/°C)", { exact: true }).fill("93");
    await page.locator(".chat-scroll").evaluate((el) => {
      el.scrollTop = 0;
    });
    await page.getByText("Edit details", { exact: true }).click();
    await expect(page.locator("details.manual-editor")).not.toHaveAttribute(
      "open",
    );
    await expect(page.locator(".draft-preview")).toContainText("18 g");
    await expect(page.locator(".draft-preview")).toContainText("93°");
    await page
      .getByLabel("Message", { exact: true })
      .fill("Use 21 grams instead");
    await page.getByRole("button", { name: "Send", exact: true }).click();
    await expect(page.locator(".draft-preview")).toContainText("21 g");
    await expect(page.locator("details.manual-editor")).not.toHaveAttribute(
      "open",
    );
    await page.locator(".chat-scroll").evaluate((el) => {
      const target = el.querySelector(".draft-card")!;
      el.scrollTop +=
        target.getBoundingClientRect().top -
        el.getBoundingClientRect().top -
        12;
    });
    await settleMotion(page);
    await page.screenshot({
      path: `/tmp/arabica-chat-preview-${viewport.name}.png`,
    });
    await openEditor(page);
    await expect(page.getByLabel("Coffee (g)", { exact: true })).toHaveValue(
      "21",
    );
    await expect(
      page.getByLabel("Temperature (°F/°C)", { exact: true }),
    ).toHaveValue("93");
    await expect(page.getByLabel("Method", { exact: true })).toHaveCount(0);
    await page.locator(".chat-scroll").evaluate((el) => {
      el.scrollTop = 0;
    });
    if (viewport.name === "phone")
      await page.locator(".chat-scroll").evaluate((el) => {
        const target = el.querySelector(".manual-editor")!;
        el.scrollTop +=
          target.getBoundingClientRect().top -
          el.getBoundingClientRect().top -
          12;
      });
    await settleMotion(page);
    await page.screenshot({
      path: `/tmp/arabica-chat-edit-details-${viewport.name}.png`,
    });
    expect(writes).toEqual([]);
    expect(
      await page.evaluate(
        () =>
          document.documentElement.scrollHeight <= innerHeight + 1 &&
          document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
  });
}

test.describe("Chat-first motion", () => {
  for (const scenario of [
    {
      name: "desktop",
      width: 1440,
      height: 1000,
      reduced: false,
      unsupported: false,
    },
    {
      name: "phone-reduced",
      width: 320,
      height: 700,
      reduced: true,
      unsupported: false,
    },
    {
      name: "fallback",
      width: 900,
      height: 900,
      reduced: false,
      unsupported: true,
    },
  ]) {
    test(`Home to chat shows working feedback and saves without form buttons (${scenario.name})`, async ({
      page,
      account,
      apiRequest,
      did,
      waitForIndex,
    }) => {
      // The shared authedPage fixture creates a manual context without video.
      // Use the video-aware default page for these three motion-review cases.
      const baseURL = new URL((await apiRequest.get("/")).url()).origin;
      const headers = {
        "X-Test-Auth-DID": account.did,
        "X-Test-Auth-Session": account.session_id,
        Origin: baseURL,
      };
      await page.context().setExtraHTTPHeaders(headers);
      await page.context().route("**/*", async (route) => {
        await route.continue({
          headers: { ...route.request().headers(), ...headers },
        });
      });
      const browserErrors: string[] = [];
      page.on("pageerror", (error) => browserErrors.push(error.message));
      page.on("console", (message) => {
        if (
          message.type() === "error" &&
          !message.text().startsWith("Failed to load resource:")
        )
          browserErrors.push(message.text());
      });
      page.on("response", (response) => {
        const path = new URL(response.url()).pathname;
        if (
          ((path.startsWith("/_app/") || path.startsWith("/static/")) &&
            response.status() >= 400) ||
          (path.startsWith("/api/") && response.status() >= 500)
        )
          browserErrors.push(`${response.status()}: ${path}`);
      });
      page.on("requestfailed", (request) => {
        if (new URL(request.url()).pathname.startsWith("/api/"))
          browserErrors.push(`Failed API request: ${request.method()}`);
      });
      await page.setViewportSize(scenario);
      await page.emulateMedia({
        reducedMotion: scenario.reduced ? "reduce" : "no-preference",
      });
      await page.addInitScript(
        ({ unsupported }) => {
          const tracked = window as typeof window & {
            __chatTransitions: number;
          };
          tracked.__chatTransitions = 0;
          if (unsupported) {
            Object.defineProperty(document, "startViewTransition", {
              value: undefined,
              configurable: true,
            });
          } else if (document.startViewTransition) {
            const start = document.startViewTransition.bind(document);
            document.startViewTransition = (update) => {
              tracked.__chatTransitions++;
              const transition = start(update);
              (
                window as typeof window & {
                  __chatTransitionFinished?: Promise<void>;
                }
              ).__chatTransitionFinished = transition.finished.catch(() => {});
              return transition;
            };
          }
        },
        { unsupported: scenario.unsupported },
      );
      const suffix = Date.now().toString(36);
      const [beanResponse, brewerResponse] = await Promise.all([
        apiRequest.post("/api/beans", {
          data: { name: `October Fix ${suffix}`, origin: "Ethiopia" },
        }),
        apiRequest.post("/api/brewers", {
          data: { name: `Orea ${suffix}`, brewer_type: "pourover" },
        }),
      ]);
      expect(beanResponse.ok()).toBe(true);
      expect(brewerResponse.ok()).toBe(true);
      const bean = await beanResponse.json();
      const brewer = await brewerResponse.json();
      await Promise.all([
        waitForIndex(`at://${did}/social.arabica.alpha.bean/${bean.rkey}`),
        waitForIndex(`at://${did}/social.arabica.alpha.brewer/${brewer.rkey}`),
      ]);
      const writes: string[] = [];
      page.on("request", (request) => {
        const path = new URL(request.url()).pathname;
        if (
          request.method() === "POST" &&
          /^\/api\/(beans|brews|grinders|brewers|recipes|roasters)$/.test(path)
        )
          writes.push(path);
      });
      await page.route("**/api/ai/status", (route) =>
        route.fulfill({ json: { enabled: true } }),
      );
      let release!: () => void;
      const waiting = new Promise<void>((resolve) => {
        release = resolve;
      });
      await page.route("**/api/ai/chat", async (route) => {
        await waiting;
        await route.fulfill({
          json: {
            reply:
              "Your brew is drafted. Tell me anything else you’d like to change, or save it below.",
            kind: "brew",
            brewer_type: "pourover",
            patch: {
              bean_rkey: bean.rkey,
              brewer_rkey: brewer.rkey,
              coffee_amount: 18,
              water_amount: 300,
              temperature: 93,
              time_seconds: 180,
              tasting_notes: "Peach and black tea.",
              pours: [
                { water_amount: 50, time_seconds: 0 },
                { water_amount: 150, time_seconds: 45 },
                { water_amount: 100, time_seconds: 90 },
              ],
            },
            choices: [],
          },
        });
      });
      await page.goto("/");
      await page
        .getByRole("textbox", { name: "Describe a brew or bean" })
        .fill(
          "October Fix in Orea: 18g to 300g, 93 degrees, 180 seconds. Pours 50g at 0s, 150g at 45s, 100g at 90s. Peach and black tea.",
        );
      await page
        .getByRole("textbox", { name: "Describe a brew or bean" })
        .press("Enter");
      await expect(page).toHaveURL(/\/chat$/);
      const working = page
        .getByRole("status")
        .filter({ hasText: "Working on your draft…" });
      await expect(working).toBeVisible();
      await expect(page.getByRole("log")).toHaveAttribute("aria-busy", "true");
      await expect(page.locator(".draft-card")).toHaveCount(0);
      expect(writes).toEqual([]);
      const animationName = await working
        .locator('[aria-hidden="true"]')
        .evaluate((el) => getComputedStyle(el).animationName);
      expect(animationName === "none").toBe(scenario.reduced);
      expect(
        await page.evaluate(
          () =>
            (window as typeof window & { __chatTransitions: number })
              .__chatTransitions,
        ),
      ).toBe(scenario.reduced || scenario.unsupported ? 0 : 1);
      await settleMotion(page);
      await page.screenshot({
        path: `/tmp/arabica-chat-working-${scenario.name}.png`,
      });
      release();
      const card = page.locator("form.draft-card");
      await expect(card).toContainText(bean.name);
      await expect(working).toHaveCount(0);
      await expect(page.locator("details.manual-editor")).not.toHaveAttribute(
        "open",
      );
      await expect(
        page.getByLabel("Bean *", { exact: true }),
      ).not.toBeVisible();
      await expect(card.locator("button:visible")).toHaveCount(1);
      const rating = page.getByLabel("Rating (1–10)", { exact: true });
      await expect(rating).toBeVisible();
      await expect(rating).toHaveValue("");
      await expect(
        page.getByRole("button", { name: "Add rating", exact: true }),
      ).toHaveCount(0);
      await rating.fill("8");
      await expect(rating).toHaveValue("8");
      await rating.fill("");
      await expect(rating).toHaveValue("");
      await rating.fill("9");
      await rating.scrollIntoViewIfNeeded();
      await expect(rating).toBeInViewport();
      await settleMotion(page);
      await page.screenshot({
        path: `/tmp/arabica-chat-rating-${scenario.name}.png`,
      });
      expect(writes).toEqual([]);
      await expect(
        card.getByRole("button", { name: "Save brew", exact: true }),
      ).toBeEnabled();
      expect(writes).toEqual([]);
      await page.locator(".chat-scroll").evaluate((el) => {
        const target = el.querySelector(".draft-card")!;
        el.scrollTop +=
          target.getBoundingClientRect().top -
          el.getBoundingClientRect().top -
          12;
      });
      await settleMotion(page);
      await page.screenshot({
        path: `/tmp/arabica-chat-rich-preview-${scenario.name}.png`,
      });
      const saved = page.waitForResponse(
        (response) =>
          new URL(response.url()).pathname === "/api/brews" &&
          response.request().method() === "POST",
      );
      await card
        .getByRole("button", { name: "Save brew", exact: true })
        .click();
      const response = await saved;
      expect(response.ok(), await response.text()).toBe(true);
      expect(response.request().postDataJSON()).toMatchObject({
        bean_rkey: bean.rkey,
        brewer_rkey: brewer.rkey,
        coffee_amount: 18,
        water_amount: 300,
        rating: 9,
      });
      expect(writes).toEqual(["/api/brews"]);
      await expect(
        page.getByRole("link", { name: "View saved brew", exact: true }),
      ).toBeVisible();
      expect(
        await page.evaluate(
          () =>
            document.documentElement.scrollHeight <= innerHeight + 1 &&
            document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
      expect(browserErrors).toEqual([]);
      const video = page.video();
      await page.close();
      if (video)
        await video.saveAs(`/tmp/arabica-chat-flow-${scenario.name}.webm`);
    });
  }
});

test("Save opens and focuses missing optional-editor details without writing", async ({
  authedPage: page,
}) => {
  await page.setViewportSize({ width: 320, height: 700 });
  const writes: string[] = [];
  page.on("request", (request) => {
    const path = new URL(request.url()).pathname;
    if (
      request.method() === "POST" &&
      /^\/api\/(beans|brews|grinders|brewers|recipes|roasters)$/.test(path)
    )
      writes.push(path);
  });
  await page.route("**/api/ai/status", (route) =>
    route.fulfill({ json: { enabled: true } }),
  );
  await page.route("**/api/data", (route) =>
    route.fulfill({
      json: {
        beans: [{ rkey: "fix", name: "October Fix" }],
        brewers: [{ rkey: "orea", name: "Orea", brewer_type: "pourover" }],
      },
    }),
  );
  await page.route("**/api/ai/chat", (route) =>
    route.fulfill({
      json: {
        reply: "Which brewer did you use?",
        kind: "brew",
        patch: { bean_rkey: "fix", coffee_amount: 18, water_amount: 300 },
        choices: [],
      },
    }),
  );
  await page.goto("/chat");
  await page
    .getByLabel("Message", { exact: true })
    .fill("October Fix, 18g to 300g.");
  await page.getByLabel("Message", { exact: true }).press("Enter");
  await expect(page.locator("form.draft-card")).toBeVisible();
  await expect(page.locator("details.manual-editor")).not.toHaveAttribute(
    "open",
  );
  await page.getByRole("button", { name: "Save brew", exact: true }).click();
  await expect(page.locator("details.manual-editor")).toHaveAttribute(
    "open",
    "",
  );
  await expect(page.getByLabel("Brewer *", { exact: true })).toBeFocused();
  expect(writes).toEqual([]);
  expect(
    await page.evaluate(
      () =>
        document.documentElement.scrollHeight <= innerHeight + 1 &&
        document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
});

test("ambiguity options show only bean alternatives, highlight, and dismiss once resolved", async ({
  authedPage: page,
}) => {
  await page.setViewportSize({ width: 1000, height: 900 });
  const writes: string[] = [];
  page.on("request", (request) => {
    if (
      request.method() === "POST" &&
      /^\/api\/(beans|brews|grinders|brewers|recipes|roasters)$/.test(
        new URL(request.url()).pathname,
      )
    )
      writes.push(request.url());
  });
  await page.route("**/api/ai/status", (route) =>
    route.fulfill({ json: { enabled: true } }),
  );
  await page.route("**/api/data", (route) =>
    route.fulfill({
      json: {
        beans: [
          { rkey: "fix", name: "October Fix" },
          { rkey: "luna", name: "La Luna" },
        ],
        brewers: [{ rkey: "orea", name: "Orea", brewer_type: "pourover" }],
      },
    }),
  );
  await page.route("**/api/ai/chat", (route) =>
    route.fulfill({
      json: {
        reply: "Using Orea. Which Recluse bean did you use?",
        kind: "brew",
        brewer_type: "pourover",
        patch: { brewer_rkey: "orea", coffee_amount: 18, water_amount: 300 },
        choices: [
          { label: "October Fix · Recluse", patch: { bean_rkey: "fix" } },
          { label: "La Luna · Recluse", patch: { bean_rkey: "luna" } },
        ],
      },
    }),
  );
  await page.goto("/chat");
  await page
    .getByLabel("Message", { exact: true })
    .fill("Brew with Recluse beans and Orea, 18g to 300g.");
  await page.getByLabel("Message", { exact: true }).press("Enter");
  const choices = page.getByLabel("Library matches", { exact: true });
  await expect(choices.locator("button")).toHaveCount(2);
  await expect(choices).not.toContainText("Orea");
  await expect(page.locator(".draft-preview")).toContainText("Orea");
  const choice = choices.getByRole("button", {
    name: "October Fix · Recluse",
    exact: true,
  });
  const rest = await choice.evaluate(
    (el) => getComputedStyle(el).backgroundColor,
  );
  await choice.hover();
  await expect
    .poll(() => choice.evaluate((el) => getComputedStyle(el).backgroundColor))
    .not.toBe(rest);
  await settleMotion(page);
  await page.screenshot({ path: "/tmp/arabica-chat-match-hover.png" });
  await choice.click();
  await expect(choices).toHaveCount(0);
  await expect(page.locator(".draft-preview")).toContainText("October Fix");
  await expect(page.locator(".draft-preview")).toContainText("Orea");
  expect(writes).toEqual([]);
});
