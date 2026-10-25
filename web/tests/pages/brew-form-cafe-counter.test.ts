import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import BrewForm from "../../src/lib/components/BrewForm.svelte";
import { userPrefs } from "../../src/lib/stores/userPrefs";
import type { Brew } from "../../src/lib/types/entity_view";

vi.mock("$app/navigation", () => ({ goto: vi.fn(), beforeNavigate: vi.fn() }));
vi.mock("../../src/lib/stores/appCache", () => ({
  appCache: {
    getData: vi
      .fn()
      .mockResolvedValue({ beans: [{}], brewers: [], recipes: [] }),
    refreshCache: vi.fn().mockResolvedValue({
      beans: [{}],
      brewers: [{ rkey: "brewer-v60", name: "V60", brewer_type: "pourover" }],
      recipes: [],
    }),
    invalidateCache: vi.fn(),
  },
}));
vi.mock("../../src/lib/stores/userPrefs", async () => {
  const { writable } = await import("svelte/store");
  return {
    userPrefs: writable({
      loading: false,
      smartAutofillEnabled: false,
      brew_form_mode: "quick",
    }),
    loadUserPrefs: vi.fn().mockResolvedValue(undefined),
    setBrewFormMode: vi.fn().mockResolvedValue(undefined),
  };
});
vi.mock("../../src/lib/stores/session", async () => {
  const { writable } = await import("svelte/store");
  return {
    session: writable({ did: "did:plc:alice", handle: "alice.test" }),
    warnIfSessionExpired: vi.fn(),
  };
});

const editBrew = {
  rkey: "brew-1",
  bean_rkey: "bean-1",
  recipe_rkey: "",
  method: "Pour Over",
  temperature: 93.5,
  water_amount: 250,
  coffee_amount: 15,
  time_seconds: 180,
  grind_size: "Medium",
  grinder_rkey: "grinder-1",
  brewer_rkey: "brewer-1",
  tasting_notes: "Bright and sweet",
  rating: 8,
  brewer_obj: { rkey: "brewer-1", name: "V60", brewer_type: "pourover" },
  pourover_params: {
    bloom_water: 50,
    bloom_seconds: 30,
    drawdown_seconds: 45,
    bypass_water: 0,
    filter: "paper",
  },
} as Brew;
const recipeResponse = (category = "pourover") =>
  new Response(
    JSON.stringify({
      rkey: "daily-v60",
      name: "Daily V60",
      coffee_amount: 18,
      water_amount: 300,
      brewer_rkey: "brewer-v60",
      brewer_type: category,
      brewer_obj: { rkey: "brewer-v60", name: "V60", brewer_type: category },
      pours: [
        { water_amount: 50, time_seconds: 30 },
        { water_amount: 250, time_seconds: 90 },
      ],
    }),
    { status: 200, headers: { "Content-Type": "application/json" } },
  );

describe("BrewForm live journal and preparation layout", () => {
  beforeEach(() => {
    userPrefs.set({
      loading: false,
      smartAutofillEnabled: false,
      brew_form_mode: "quick",
    });
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(recipeResponse()));
  });
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("shows the ratio with the measurements and updates it with the shared draft", async () => {
    const user = userEvent.setup();
    render(BrewForm, { brew: editBrew, isEdit: true });
    expect(screen.getByText(/Ratio 1:16\.7/)).toBeTruthy();
    expect(
      screen.getByRole("combobox", { name: "Search brew methods" }),
    ).toHaveValue("V60");
    expect(
      screen.queryByRole("complementary", { name: "Form context" }),
    ).not.toBeInTheDocument();
    await user.clear(screen.getByLabelText("Coffee amount (g)"));
    await user.type(screen.getByLabelText("Coffee amount (g)"), "25");
    expect(screen.getByText(/Ratio 1:10\.0/)).toBeTruthy();
  });

  it("scales recipe dose, ratio, water, pours, and bloom after opening Detailed", async () => {
    const user = userEvent.setup();
    render(BrewForm, { brew: null, recipeRKey: "daily-v60", isEdit: false });
    await waitFor(() =>
      expect(
        screen.getAllByRole("button", { name: "Edit brewing details" }).length,
      ).toBeGreaterThan(0),
    );
    await user.click(
      screen.getAllByRole("button", { name: "Edit brewing details" })[0],
    );
    await user.click(screen.getByText("More brewing details"));
    const coffee = screen.getByLabelText("Coffee (g)");
    const ratio = screen.getByLabelText("Ratio (1:X)");
    const water = screen.getByLabelText("Water (g)");
    expect(coffee).toHaveValue(18);
    expect(ratio).toHaveValue(16.67);
    expect(water).toHaveValue(300);
    expect(screen.getByLabelText("Bloom water (g)")).toHaveValue(50);
    expect(screen.getByLabelText("Bloom time (s)")).toHaveValue(30);
    expect(
      screen.getByRole("combobox", { name: "Search brew methods" }),
    ).toHaveValue("V60");
    await user.clear(coffee);
    await user.type(coffee, "20");
    expect(water).toHaveValue(333);
    expect(screen.getByLabelText("Pour 1")).toHaveValue(56);
    expect(screen.getByLabelText("Pour 2")).toHaveValue(277);
    await user.clear(ratio);
    await user.type(ratio, "15");
    expect(water).toHaveValue(300);
    await user.clear(water);
    await user.type(water, "240");
    expect(coffee).toHaveValue(16);
    expect(screen.getByLabelText("Pour 1")).toHaveValue(40);
    expect(screen.getByLabelText("Pour 2")).toHaveValue(200);
    expect(screen.getByLabelText("Bloom water (g)")).toHaveValue(40);
  });

  it.each([
    ["espresso", "Shot time (s)", true],
    ["pourover", "Brew time (s)", true],
    ["French Press", "Steep time (s)", false],
    ["Cold Brew", "Steep time (s)", false],
    ["cupping", "Steep time (s)", false],
    ["unrecognized", "Brew time (s)", false],
  ])(
    "places %s preparation and grinder in their matrix sections",
    (category, timeLabel, mainGrinder) => {
      const record = {
        ...editBrew,
        method: "",
        grinder_rkey: "",
        grind_size: "",
        water_amount: 0,
        pourover_params: undefined,
        brewer_obj: { ...editBrew.brewer_obj!, brewer_type: category },
      };
      render(BrewForm, { brew: record, isEdit: true });
      const more = screen.getByText("More brewing details").parentElement!;
      expect(screen.getByLabelText(timeLabel).closest("details")).toBeNull();
      expect(
        screen
          .getByRole("combobox", { name: "Search grinders", hidden: true })
          .closest("details"),
      ).toBe(mainGrinder ? null : more);
      expect(
        screen.getByLabelText("Temperature (°F/°C)").closest("details"),
      ).toBe(more);
      if (category === "espresso") {
        expect(screen.queryByLabelText("Water amount (g)")).toBeNull();
        expect(
          screen.getByLabelText("Yield weight (g)").closest("details"),
        ).toBeNull();
        expect(screen.getByLabelText("Pressure (bar)").closest("details")).toBe(
          more,
        );
      } else
        expect(
          screen.getByLabelText("Water amount (g)").closest("details"),
        ).toBeNull();
    },
  );

  it("places Moka pot time under More brewing details and accepts long steeps in seconds", async () => {
    const user = userEvent.setup();
    render(BrewForm, {
      brew: {
        ...editBrew,
        brewer_obj: { ...editBrew.brewer_obj!, brewer_type: "Moka Pot" },
      },
      isEdit: true,
    });
    const more = screen.getByText("More brewing details").parentElement!;
    expect(screen.getByLabelText("Brew time (s)").closest("details")).toBe(
      more,
    );
    await user.click(screen.getByText("More brewing details"));
    await user.clear(screen.getByLabelText("Brew time (s)"));
    await user.type(screen.getByLabelText("Brew time (s)"), "86400");
    expect(screen.getByLabelText("Brew time (s)")).toHaveValue(86400);
  });

  it("converts a long steep from hours to the existing seconds contract", async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ brew: { rkey: "brew-1" } }), {
        status: 200,
      }),
    );
    render(BrewForm, {
      brew: {
        ...editBrew,
        brewer_obj: { ...editBrew.brewer_obj!, brewer_type: "coldbrew" },
      },
      isEdit: true,
    });
    await user.selectOptions(screen.getByLabelText("Time unit"), "3600");
    await user.clear(screen.getByLabelText("Steep time (h)"));
    await user.type(screen.getByLabelText("Steep time (h)"), "12.5");
    await user.click(screen.getByRole("button", { name: "Update brew" }));
    await waitFor(() =>
      expect(fetch).toHaveBeenCalledWith(
        "/api/brews/brew-1",
        expect.objectContaining({ method: "PUT" }),
      ),
    );
    const input = JSON.parse(vi.mocked(fetch).mock.calls[0][1]!.body as string);
    expect(input.time_seconds).toBe(45000);
  });

  it("keeps old out-of-category water, pours, and mixed parameters reachable", async () => {
    const user = userEvent.setup();
    render(BrewForm, {
      brew: {
        ...editBrew,
        brewer_obj: { ...editBrew.brewer_obj!, brewer_type: "espresso" },
        pours: [{ water_amount: 50, time_seconds: 30 }],
      },
      isEdit: true,
    });
    await user.click(screen.getByText("More brewing details"));
    expect(screen.getByLabelText("Water amount (g)")).toHaveValue(250);
    expect(screen.getByLabelText("Pour 1")).toHaveValue(50);
    expect(screen.getByLabelText("Filter")).toHaveValue("paper");
  });

  it("points Cancel at the brew view on edit and my-coffee on creation", () => {
    const edit = render(BrewForm, { brew: editBrew, isEdit: true });
    expect(screen.getByRole("link", { name: "Cancel" })).toHaveAttribute(
      "href",
      "/brews/did%3Aplc%3Aalice/brew-1",
    );
    edit.unmount();
    render(BrewForm, { brew: null, isEdit: false });
    expect(screen.getByRole("link", { name: "Cancel" })).toHaveAttribute(
      "href",
      "/my-coffee",
    );
  });
});
