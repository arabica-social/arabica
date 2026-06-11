import { cleanup, render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import RecipeFormIsland from "./RecipeFormIsland.svelte";

function mountTarget() {
  const target = document.createElement("div");
  target.dataset.name = "Morning V60";
  target.dataset.brewerRkey = "brewer-1";
  target.dataset.brewerType = "pourover";
  target.dataset.coffeeAmount = "18";
  target.dataset.waterAmount = "250";
  target.dataset.pours = '[{"water":50,"time":45},{"water":180,"time":120}]';
  target.dataset.brewers =
    '[{"rkey":"brewer-1","name":"V60","brewer_type":"pourover"}]';
  document.body.appendChild(target);
  return target;
}

function installAppCache() {
  const data = {
    brewers: [{ rkey: "brewer-1", name: "V60", brewer_type: "pourover" }],
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

describe("RecipeFormIsland", () => {
  afterEach(() => {
    cleanup();
    document.body.innerHTML = "";
    delete window.AppCache;
    vi.restoreAllMocks();
  });

  it("shows derived pour totals and water mismatch warnings", () => {
    installAppCache();
    const target = mountTarget();

    render(RecipeFormIsland, { target, props: { target } });

    expect(screen.getByTestId("pour-summary")).toHaveTextContent(
      "2 pours · 230g total · last at 120s",
    );
    expect(screen.getByRole("status")).toHaveTextContent(
      "Pour water totals 230g, which does not match total water 250g.",
    );
  });

  it("shows inline required and numeric validation without changing field names", async () => {
    const user = userEvent.setup();
    installAppCache();
    const target = mountTarget();
    target.dataset.name = "";
    target.dataset.brewerRkey = "";
    target.dataset.coffeeAmount = "0";

    render(RecipeFormIsland, { target, props: { target } });

    expect(screen.getByText("Recipe name is required.")).toBeInTheDocument();
    expect(screen.getByText("Brewer is required.")).toBeInTheDocument();
    expect(
      screen.getByText("Coffee amount must be greater than 0."),
    ).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Name")).toHaveAttribute("name", "name");

    const waterInput = document.querySelector<HTMLInputElement>(
      'input[name="water_amount"]',
    );
    expect(waterInput).not.toBeNull();
    await user.clear(waterInput!);
    await user.type(waterInput!, "-1");

    expect(
      screen.getByText("Water amount must be greater than 0."),
    ).toBeInTheDocument();
  });
});
