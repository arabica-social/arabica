import { cleanup, render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import BrewFormIsland from "./BrewFormIsland.svelte";

function mountTarget() {
  const form = document.createElement("form");
  const target = document.createElement("div");
  target.dataset.svelteBrewForm = "";
  target.dataset.submitLabel = "Save Brew";
  target.dataset.beanRkey = "bean-old";
  target.dataset.beanLabel = "Old Bean";
  target.dataset.coffeeAmount = "18";
  target.dataset.waterAmount = "250";
  target.dataset.grindSize = "medium";
  target.dataset.temperature = "93";
  target.dataset.timeSeconds = "180";
  target.dataset.tastingNotes = "sweet and bright";
  target.dataset.rating = "7";
  target.dataset.pours = '[{"water":50,"time":30}]';
  form.appendChild(target);
  document.body.appendChild(form);
  return { form, target };
}

function installAppCache() {
  const data = {
    beans: [
      {
        rkey: "bean-new",
        name: "Nueva Esperanza",
        origin: "Guatemala",
        roast_level: "Medium",
      },
    ],
    grinders: [{ rkey: "grinder-1", name: "Comandante" }],
    brewers: [
      { rkey: "brewer-1", name: "V60", brewer_type: "pourover" },
      { rkey: "brewer-2", name: "Linea Mini", brewer_type: "espresso" },
    ],
    recipes: [],
  };
  window.AppCache = {
    getCachedData: vi.fn(() => data),
    getData: vi.fn(() => Promise.resolve(data)),
    refreshCache: vi.fn(() => Promise.resolve(data)),
    invalidateCache: vi.fn(),
    init: vi.fn(() => Promise.resolve()),
    preload: vi.fn(() => Promise.resolve(data)),
    isCacheValid: vi.fn(() => true),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    invalidateAndRefresh: vi.fn(() => Promise.resolve(data)),
  };
}

describe("BrewFormIsland", () => {
  afterEach(() => {
    cleanup();
    document.body.innerHTML = "";
    delete window.AppCache;
    vi.restoreAllMocks();
  });

  it("renders form controls from the server dataset and submits selected entity rkeys", async () => {
    const user = userEvent.setup();
    installAppCache();
    const { form, target } = mountTarget();

    render(BrewFormIsland, { target, props: { target } });

    expect(target.querySelector("noscript")).toBeNull();
    expect(screen.getByRole("button", { name: "Save Brew" })).toBeVisible();
    expect(screen.getByDisplayValue("18")).toHaveAttribute(
      "name",
      "coffee_amount",
    );
    expect(screen.getByDisplayValue("sweet and bright")).toHaveAttribute(
      "name",
      "tasting_notes",
    );

    await user.clear(
      screen.getByRole("combobox", { name: "Search coffee beans" }),
    );
    await user.type(
      screen.getByRole("combobox", { name: "Search coffee beans" }),
      "Nueva",
    );
    await user.click(
      await screen.findByRole("option", {
        name: "Nueva Esperanza (Guatemala - Medium)",
      }),
    );

    await user.type(
      screen.getByRole("combobox", { name: "Search grinders" }),
      "Com",
    );
    await user.click(await screen.findByRole("option", { name: "Comandante" }));

    await user.type(
      screen.getByRole("combobox", { name: "Search brew methods" }),
      "V60",
    );
    await user.click(await screen.findByRole("option", { name: "V60" }));

    const formData = new FormData(form);
    expect(formData.get("bean_rkey")).toBe("bean-new");
    expect(formData.get("grinder_rkey")).toBe("grinder-1");
    expect(formData.get("brewer_rkey")).toBe("brewer-1");
    expect(formData.get("coffee_amount")).toBe("18");
    expect(formData.get("water_amount")).toBe("250");
    expect(formData.get("pour_water_0")).toBe("50");
    expect(formData.get("pour_time_0")).toBe("30");
    expect(formData.get("rating")).toBe("7");
  });

  it("shows derived pour totals and validation warnings", async () => {
    const user = userEvent.setup();
    installAppCache();
    const { target } = mountTarget();

    render(BrewFormIsland, { target, props: { target } });

    expect(screen.getByTestId("pour-summary")).toHaveTextContent(
      "1 pour · 50g total · last at 30s",
    );
    expect(screen.getByRole("status")).toHaveTextContent(
      "Pour water totals 50g, which does not match total water 250g.",
    );

    await user.clear(screen.getByLabelText("Coffee Amount (grams)"));
    await user.type(screen.getByLabelText("Coffee Amount (grams)"), "0");
    await user.clear(screen.getByLabelText("Brew Time (seconds)"));
    await user.type(screen.getByLabelText("Brew Time (seconds)"), "-1");

    expect(
      screen.getByText("Coffee amount must be greater than 0."),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Brew time must be greater than 0."),
    ).toBeInTheDocument();
  });

  it("shows method-specific fields after selecting an espresso brewer", async () => {
    const user = userEvent.setup();
    installAppCache();
    const { target } = mountTarget();

    render(BrewFormIsland, { target, props: { target } });

    expect(screen.queryByText("Espresso")).not.toBeInTheDocument();

    await user.type(
      screen.getByRole("combobox", { name: "Search brew methods" }),
      "Linea",
    );
    await user.click(await screen.findByRole("option", { name: "Linea Mini" }));

    expect(screen.getByText("Espresso")).toBeInTheDocument();
    expect(screen.getByLabelText("Yield Weight (grams)")).toHaveAttribute(
      "name",
      "espresso_yield_weight",
    );
  });
});
