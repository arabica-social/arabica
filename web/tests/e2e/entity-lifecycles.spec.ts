import { test, expect } from "./fixtures";

test("bean lifecycle: create, reload, edit, delete", async ({
	authedPage: page,
	did,
	waitForIndex,
}) => {
	const suffix = Date.now().toString(36);
	const name = `E2E Bean ${suffix}`;
	const origin = "Colombia";
	const variety = "Bourbon";
	const updatedNotes = `Updated personal notes ${suffix}`;

	await page.goto("/beans/new");
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();

	// Validation: name + origin required (bean form shows both errors).
	await page.getByRole("button", { name: "Add bean" }).click();
	await expect(page.getByText("Name is required")).toBeVisible();
	await expect(page.getByText("Origin is required")).toBeVisible();

	await page.getByLabel("Name").fill(name);
	await page.getByRole("textbox", { name: "Origin" }).fill(origin);
	await page.getByLabel("Variety").fill(variety);
	await page.getByLabel("Roast level").selectOption("Light");
	await page.getByLabel("Process").fill("Washed");
	await page.getByText("Your notes and rating (optional)").click();
	await page.getByLabel("Notes").fill("Initial notes");
	await page.getByRole("button", { name: "Add bean" }).click();

	await page.waitForURL(/\/beans\/[^/]+\/[^/]+$/);
	const detailURL = new URL(page.url());
	const rkey = detailURL.pathname.split("/").at(-1);
	expect(rkey).toBeTruthy();

	await expect(page.getByRole("heading", { name })).toBeVisible();
	await expect(page.getByText(origin)).toBeVisible();
	await expect(page.getByText(variety)).toBeVisible();
	await expect(page.getByText("Light")).toBeVisible();
	await expect(page.getByText("Washed")).toBeVisible();

	await page.reload();
	await expect(page.getByRole("heading", { name })).toBeVisible();

	await page.goto(`/beans/${rkey}/edit`);
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();
	await expect(page.getByLabel("Name")).toHaveValue(name);
	const notesField = page.getByLabel("Notes");
	await notesField.fill(updatedNotes);
	await page.getByRole("button", { name: "Save changes" }).click();

	await page.waitForURL(/\/beans\/[^/]+\/[^/]+$/);
	await expect(page.getByText(updatedNotes)).toBeVisible();

	const uri = `at://${did}/social.arabica.alpha.bean/${rkey}`;
	await waitForIndex(uri);
	await page.goto("/my-coffee");
	await page.getByRole("button", { name: "Beans" }).click();
	await expect(page.getByRole("link", { name })).toBeVisible();

	await page.goto(detailURL.pathname);
	await page.getByRole("button", { name: "More options" }).click();
	page.once("dialog", (dialog) => dialog.accept());
	await page.getByRole("menuitem", { name: "Delete" }).click();
	await expect(page).toHaveURL(/\/my-coffee$/);
	await waitForIndex(uri, false);

	await page.goto(detailURL.pathname);
	await expect(page.getByText("Record not found")).toBeVisible();
});

test("grinder lifecycle: create, reload, edit, delete", async ({
	authedPage: page,
	did,
	waitForIndex,
}) => {
	const suffix = Date.now().toString(36);
	const name = `E2E Grinder ${suffix}`;
	const updatedNotes = `Dial-in notes ${suffix}`;

	await page.goto("/grinders/new");
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();

	await page.getByRole("button", { name: "Add Grinder" }).click();
	await expect(page.getByRole("alert")).toContainText("Name is required");

	await page.getByLabel("Name").fill(name);
	await page.locator("#grinder-type").selectOption("Electric");
	await page.locator("#grinder-burr-type").selectOption("Flat");
	await page.getByLabel("Notes").fill("Stock burrs");
	await page.getByRole("button", { name: "Add Grinder" }).click();

	await page.waitForURL(/\/grinders\/[^/]+\/[^/]+$/);
	const detailURL = new URL(page.url());
	const rkey = detailURL.pathname.split("/").at(-1);
	expect(rkey).toBeTruthy();

	await expect(page.getByRole("heading", { name })).toBeVisible();
	await expect(page.getByText("Electric")).toBeVisible();
	await expect(page.getByText("Flat")).toBeVisible();
	await expect(page.getByText("Stock burrs")).toBeVisible();

	await page.reload();
	await expect(page.getByRole("heading", { name })).toBeVisible();

	await page.goto(`/grinders/${rkey}/edit`);
	await expect(page.getByLabel("Name")).toHaveValue(name);
	await page.getByLabel("Notes").fill(updatedNotes);
	await page.getByRole("button", { name: "Save Changes" }).click();

	await page.waitForURL(/\/grinders\/[^/]+\/[^/]+$/);
	await expect(page.getByText(updatedNotes)).toBeVisible();

	const uri = `at://${did}/social.arabica.alpha.grinder/${rkey}`;
	await waitForIndex(uri);
	await page.goto("/my-coffee");
	await page.getByRole("button", { name: "Grinders" }).click();
	await expect(page.getByRole("link", { name })).toBeVisible();

	await page.goto(detailURL.pathname);
	await page.getByRole("button", { name: "More options" }).click();
	page.once("dialog", (dialog) => dialog.accept());
	await page.getByRole("menuitem", { name: "Delete" }).click();
	await expect(page).toHaveURL(/\/my-coffee$/);
	await waitForIndex(uri, false);

	await page.goto(detailURL.pathname);
	await expect(page.getByText("Record not found")).toBeVisible();
});

test("brewer lifecycle: create, reload, edit, delete", async ({
	authedPage: page,
	did,
	waitForIndex,
}) => {
	const suffix = Date.now().toString(36);
	const name = `E2E Brewer ${suffix}`;
	const updatedDescription = `Updated description ${suffix}`;

	await page.goto("/brewers/new");
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();

	await page.getByRole("button", { name: "Add Brewer" }).click();
	await expect(page.getByRole("alert")).toContainText("Name is required");

	await page.getByLabel("Name").fill(name);
	await page.getByLabel("Type").selectOption("pourover");
	await page.getByLabel("Description").fill("02 size dripper");
	await page.getByRole("button", { name: "Add Brewer" }).click();

	await page.waitForURL(/\/brewers\/[^/]+\/[^/]+$/);
	const detailURL = new URL(page.url());
	const rkey = detailURL.pathname.split("/").at(-1);
	expect(rkey).toBeTruthy();

	await expect(page.getByRole("heading", { name })).toBeVisible();
	await expect(page.getByText("pourover")).toBeVisible();
	await expect(page.getByText("02 size dripper")).toBeVisible();

	await page.reload();
	await expect(page.getByRole("heading", { name })).toBeVisible();

	await page.goto(`/brewers/${rkey}/edit`);
	await expect(page.getByLabel("Name")).toHaveValue(name);
	await page.getByLabel("Description").fill(updatedDescription);
	await page.getByRole("button", { name: "Save Changes" }).click();

	await page.waitForURL(/\/brewers\/[^/]+\/[^/]+$/);
	await expect(page.getByText(updatedDescription)).toBeVisible();

	const uri = `at://${did}/social.arabica.alpha.brewer/${rkey}`;
	await waitForIndex(uri);
	await page.goto("/my-coffee");
	await page.getByRole("button", { name: "Brewers" }).click();
	await expect(page.getByRole("link", { name })).toBeVisible();

	await page.goto(detailURL.pathname);
	await page.getByRole("button", { name: "More options" }).click();
	page.once("dialog", (dialog) => dialog.accept());
	await page.getByRole("menuitem", { name: "Delete" }).click();
	await expect(page).toHaveURL(/\/my-coffee$/);
	await waitForIndex(uri, false);

	await page.goto(detailURL.pathname);
	await expect(page.getByText("Record not found")).toBeVisible();
});

test("recipe lifecycle: create, reload, edit, delete", async ({
	authedPage: page,
	did,
	waitForIndex,
}) => {
	const suffix = Date.now().toString(36);
	const name = `E2E Recipe ${suffix}`;
	const updatedNotes = `Updated recipe notes ${suffix}`;

	await page.goto("/recipes/new");
	await expect(page.locator('body[data-frontend="sveltekit"]')).toBeAttached();

	await page.getByRole("button", { name: "Add Recipe" }).click();
	await expect(page.getByRole("alert")).toContainText("Name is required");

	await page.getByLabel("Name").fill(name);
	await page.getByLabel("Brewer type").selectOption("pourover");
	await page.getByLabel("Coffee amount in grams").fill("15");
	await page.getByLabel("Water amount in grams").fill("250");
	await page.getByRole("textbox", { name: "Notes" }).fill("Standard 1:16 ratio");
	await page.getByRole("button", { name: "Add Recipe" }).click();

	await page.waitForURL(/\/recipes\/[^/]+\/[^/]+$/);
	const detailURL = new URL(page.url());
	const rkey = detailURL.pathname.split("/").at(-1);
	expect(rkey).toBeTruthy();

	await expect(page.getByRole("heading", { name })).toBeVisible();
	// Recipe view renders amounts with one decimal: "15.0g" / "250.0g".
	await expect(page.getByText("15.0g")).toBeVisible();
	await expect(page.getByText("250.0g")).toBeVisible();
	await expect(page.getByText("Standard 1:16 ratio")).toBeVisible();

	await page.reload();
	await expect(page.getByRole("heading", { name })).toBeVisible();

	await page.goto(`/recipes/${rkey}/edit`);
	await expect(page.getByLabel("Name")).toHaveValue(name);
	await page.getByRole("textbox", { name: "Notes" }).fill(updatedNotes);
	await page.getByRole("button", { name: "Save Changes" }).click();

	await page.waitForURL(/\/recipes\/[^/]+\/[^/]+$/);
	await expect(page.getByText(updatedNotes)).toBeVisible();

	const uri = `at://${did}/social.arabica.alpha.recipe/${rkey}`;
	await waitForIndex(uri);
	await page.goto("/my-coffee");
	await page.getByRole("button", { name: "Recipes" }).click();
	await expect(page.getByRole("link", { name })).toBeVisible();

	await page.goto(detailURL.pathname);
	await page.getByRole("button", { name: "More options" }).click();
	page.once("dialog", (dialog) => dialog.accept());
	await page.getByRole("menuitem", { name: "Delete" }).click();
	await expect(page).toHaveURL(/\/my-coffee$/);
	await waitForIndex(uri, false);

	await page.goto(detailURL.pathname);
	await expect(page.getByText("Record not found")).toBeVisible();
});
