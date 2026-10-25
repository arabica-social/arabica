import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import BrewForm from "../../src/lib/components/BrewForm.svelte";
import { userPrefs } from "../../src/lib/stores/userPrefs";
import type { Brew, Recipe } from "../../src/lib/types/entity_view";

const mocks = vi.hoisted(() => ({
  getData: vi.fn(),
  loadPrefs: vi.fn(),
  saveMode: vi.fn(),
  warn: vi.fn(),
  navigate: vi.fn(),
  beforeNavigate: vi.fn(),
  computeAutofill: vi.fn(),
}));
vi.mock("$app/navigation", () => ({
  goto: mocks.navigate,
  beforeNavigate: mocks.beforeNavigate,
}));
vi.mock("../../src/lib/stores/appCache", () => ({
  appCache: {
    getData: mocks.getData,
    refreshCache: mocks.getData,
    invalidateCache: vi.fn(),
  },
}));
vi.mock("../../src/lib/stores/userPrefs", async () => {
  const { writable } = await import("svelte/store");
  return {
    userPrefs: writable({
      loading: false,
      smartAutofillEnabled: true,
      brew_form_mode: "quick",
    }),
    loadUserPrefs: mocks.loadPrefs,
    setBrewFormMode: mocks.saveMode,
  };
});
vi.mock("../../src/lib/utils/brewAutofill", () => ({
  computeAutofill: mocks.computeAutofill,
}));
vi.mock("../../src/lib/stores/session", async () => {
  const { writable } = await import("svelte/store");
  return {
    session: writable({
      did: "did:plc:alice",
      handle: "alice.test",
      isAuthenticated: true,
    }),
    warnIfSessionExpired: mocks.warn,
  };
});

const bean = {
  rkey: "bean-1",
  name: "Colombia",
  origin: "Colombia",
  roast_level: "Light",
};
const v60 = { rkey: "v60", name: "V60", brewer_type: "Pour Over" };
const espresso = {
  rkey: "espresso",
  name: "Espresso machine",
  brewer_type: "espresso",
};
const grinders = [
  { rkey: "ode", name: "Ode" },
  { rkey: "niche", name: "Niche" },
];
const data = {
  beans: [bean],
  brewers: [v60, espresso],
  grinders,
  recipes: [],
  brews: [],
};
const editBrew = {
  rkey: "brew-1",
  bean_rkey: "bean-1",
  bean,
  brewer_rkey: "v60",
  brewer_obj: v60,
  grinder_rkey: "ode",
  grinder_obj: grinders[0],
  grind_size: "Medium",
  coffee_amount: 18,
  water_amount: 300,
  temperature: 93.5,
  time_seconds: 180,
  tasting_notes: "Sweet",
  rating: 8,
  pourover_params: {
    bloom_water: 50,
    bloom_seconds: 30,
    drawdown_seconds: 0,
    bypass_water: 0,
    filter: "paper",
  },
} as Brew;
const recipe = (rkey: string, fields: Partial<Recipe> = {}) => ({
  rkey,
  name: rkey,
  coffee_amount: 18,
  water_amount: 300,
  brewer_rkey: "v60",
  brewer_obj: v60,
  brewer_type: "pourover",
  pours: [
    { water_amount: 50, time_seconds: 30 },
    { water_amount: 250, time_seconds: 90 },
  ],
  ...fields,
});
const response = (body: unknown) =>
  new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
const deferred = <T>() => {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((r) => (resolve = r));
  return { promise, resolve };
};
const hidden = (name: string) =>
  document.querySelector<HTMLInputElement>(`input[name="${name}"]`)?.value;
async function select(
  user: ReturnType<typeof userEvent.setup>,
  label: string,
  option: string | RegExp,
) {
  const combo = await screen.findByRole("combobox", { name: label });
  await user.click(combo);
  await user.clear(combo);
  await user.keyboard("{ArrowDown}");
  await user.click(await screen.findByRole("option", { name: option }));
}
async function openMore(user: ReturnType<typeof userEvent.setup>) {
  const summary = screen.getByText("More brewing details");
  if (!(summary.parentElement as HTMLDetailsElement).open)
    await user.click(summary);
}
function savedInput(fetchMock: ReturnType<typeof vi.fn>) {
  const call = fetchMock.mock.calls.find(
    ([, init]) => init?.method === "POST" || init?.method === "PUT",
  );
  return JSON.parse(call?.[1].body ?? "{}");
}

describe("BrewForm Quick and Detailed", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getData.mockResolvedValue(data);
    mocks.loadPrefs.mockResolvedValue(undefined);
    mocks.saveMode.mockResolvedValue(undefined);
    mocks.computeAutofill.mockReturnValue(null);
    userPrefs.set({
      loading: false,
      smartAutofillEnabled: true,
      brew_form_mode: "quick",
    });
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          response({ brew: { rkey: "saved" }, author_did: "did:plc:alice" }),
        ),
    );
    vi.spyOn(window, "confirm").mockReturnValue(true);
  });
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("starts Quick with no preparation, recipe selector, or default rating and saves references only", async () => {
    const user = userEvent.setup();
    render(BrewForm, { brew: null, isEdit: false });
    expect(screen.getByRole("button", { name: "Quick" })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(screen.queryByLabelText("Coffee amount (g)")).toBeNull();
    expect(
      screen.queryByRole("combobox", { name: "Search recipes" }),
    ).toBeNull();
    expect(
      screen.queryByRole("slider", { name: "Rating" }),
    ).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add rating" })).toHaveAttribute(
      "aria-expanded",
      "false",
    );
    await select(user, "Search coffee beans", /Colombia/);
    await select(user, "Search brew methods", "V60");
    await user.click(screen.getByRole("button", { name: "Save brew" }));
    await waitFor(() => expect(mocks.navigate).toHaveBeenCalled());
    expect(savedInput(vi.mocked(fetch))).toEqual({
      bean_rkey: "bean-1",
      brewer_rkey: "v60",
    });
    expect(mocks.computeAutofill).not.toHaveBeenCalled();
    expect(mocks.warn).toHaveBeenCalled();
  });

  it("requires a brewer only for creation and focuses the missing control", async () => {
    const user = userEvent.setup();
    render(BrewForm, { brew: null, isEdit: false });
    await select(user, "Search coffee beans", /Colombia/);
    await user.click(screen.getByRole("button", { name: "Save brew" }));
    expect(screen.getByRole("alert")).toHaveTextContent("Choose a brewer");
    expect(
      screen.getByRole("combobox", { name: "Search brew methods" }),
    ).toHaveFocus();
    expect(fetch).not.toHaveBeenCalled();
  });

  it("marks the actual missing reference controls invalid and clears their errors after selection", async () => {
    const user = userEvent.setup();
    render(BrewForm, { brew: null, isEdit: false });
    await user.click(screen.getByRole("button", { name: "Save brew" }));
    const beanControl = screen.getByRole("combobox", {
      name: "Search coffee beans",
    });
    const brewerControl = screen.getByRole("combobox", {
      name: "Search brew methods",
    });
    expect(beanControl).toHaveFocus();
    expect(beanControl).toHaveAttribute("aria-invalid", "true");
    expect(beanControl).toHaveAccessibleDescription(
      "Choose a coffee bean before saving.",
    );
    expect(brewerControl).toHaveAttribute("aria-invalid", "true");
    expect(brewerControl).toHaveAttribute("aria-required", "true");
    await select(user, "Search coffee beans", /Colombia/);
    await select(user, "Search brew methods", "V60");
    expect(beanControl).toHaveAttribute("aria-invalid", "false");
    expect(brewerControl).toHaveAttribute("aria-invalid", "false");
  });

  it.each([
    ["coffee_amount", "Coffee amount (g)"],
    ["water_amount", "Water amount (g)"],
    ["temperature", "Temperature (°F/°C)"],
    ["time_seconds", "Brew time (s)"],
  ])(
    "reveals and focuses an invalid %s from Quick on save",
    async (field, label) => {
      const user = userEvent.setup();
      render(BrewForm, { brew: editBrew, isEdit: true });
      if (field === "temperature") await openMore(user);
      const control = screen.getByLabelText(label);
      await fireEvent.input(control, { target: { value: "-1" } });
      await user.click(screen.getByRole("button", { name: "Quick" }));
      await user.click(screen.getByRole("button", { name: "Update brew" }));
      const invalid = document.getElementById(`brew-${field}`)!;
      await waitFor(() => expect(invalid).toHaveFocus());
      expect(invalid).toHaveAttribute("aria-invalid", "true");
      expect(invalid).toHaveAccessibleDescription(
        `${label} must be greater than 0.`,
      );
      expect(
        screen.getByText("More brewing details").parentElement,
      ).toHaveAttribute("open");
      expect(fetch).not.toHaveBeenCalled();
    },
  );

  it("round-trips legacy absent brewer/rating, unassociated setting, and mixed parameters without cleanup", async () => {
    const user = userEvent.setup();
    const legacy = {
      ...editBrew,
      brewer_rkey: "",
      brewer_obj: undefined,
      grinder_rkey: "",
      grinder_obj: undefined,
      rating: 0,
      espresso_params: {
        yield_weight: 36,
        pressure: 9,
        pre_infusion_seconds: 5,
      },
    };
    render(BrewForm, { brew: legacy, isEdit: true });
    expect(screen.getByRole("button", { name: "Detailed" })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    await openMore(user);
    expect(screen.getByLabelText("Grind setting")).toHaveValue("Medium");
    expect(screen.getByLabelText("Yield weight (g)")).toHaveValue(36);
    expect(screen.getByLabelText("Filter")).toHaveValue("paper");
    await user.click(screen.getByRole("button", { name: "Update brew" }));
    await waitFor(() => expect(mocks.navigate).toHaveBeenCalled());
    const input = savedInput(vi.mocked(fetch));
    expect(input).toMatchObject({
      grind_size: "Medium",
      espresso_params: legacy.espresso_params,
      pourover_params: legacy.pourover_params,
    });
    expect(input).not.toHaveProperty("rating");
    expect(input).not.toHaveProperty("brewer_rkey");
    expect(window.confirm).not.toHaveBeenCalled();
    expect(mocks.saveMode).not.toHaveBeenCalled();
  });

  it("keeps the form focused on inputs, with the brew ratio beside the measurements", async () => {
    const user = userEvent.setup();
    const { container } = render(BrewForm, { brew: null, isEdit: false });
    expect(
      screen.queryByRole("complementary", { name: "Form context" }),
    ).not.toBeInTheDocument();
    expect(container.querySelector(".form-workspace")).not.toHaveClass(
      "has-rail",
    );
    expect(screen.queryByText("Brew session")).not.toBeInTheDocument();
    expect(screen.queryByText("No recipe selected.")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Tasting notes")).toHaveAttribute(
      "placeholder",
      "Describe the flavors, aroma, and your thoughts...",
    );
    expect(
      screen.queryByText("Optional", { exact: true }),
    ).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Detailed" }));
    await user.type(screen.getByLabelText("Coffee amount (g)"), "20");
    await user.type(screen.getByLabelText("Water amount (g)"), "300");
    expect(
      within(
        screen.getByRole("region", { name: "Brewing (optional)" }),
      ).getByText("Ratio 1:15.0"),
    ).toBeInTheDocument();
  });

  it("adds an opt-in slider, updates its value, and preserves it across modes", async () => {
    const user = userEvent.setup();
    render(BrewForm, { brew: null, isEdit: false });
    await select(user, "Search coffee beans", /Colombia/);
    await select(user, "Search brew methods", "V60");
    await user.click(screen.getByRole("button", { name: "Add rating" }));
    const slider = screen.getByRole("slider", { name: "Rating" });
    await waitFor(() => expect(slider).toHaveFocus());
    expect(slider).toHaveValue("5");
    expect(slider).toHaveAttribute("min", "1");
    expect(slider).toHaveAttribute("max", "10");
    await fireEvent.input(slider, { target: { value: "8" } });
    expect(slider).toHaveAttribute("aria-valuetext", "8 out of 10");
    expect(screen.getByText("8/10")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Detailed" }));
    await user.click(screen.getByRole("button", { name: "Quick" }));
    expect(screen.getByRole("slider", { name: "Rating" })).toHaveValue("8");
    await user.click(screen.getByRole("button", { name: "Save brew" }));
    await waitFor(() => expect(mocks.navigate).toHaveBeenCalled());
    expect(savedInput(vi.mocked(fetch))).toMatchObject({ rating: 8 });
  });

  it("removes a rating and can add it again without retaining the previous value", async () => {
    const user = userEvent.setup();
    render(BrewForm, { brew: editBrew, isEdit: true });
    expect(screen.getByRole("slider", { name: "Rating" })).toHaveValue("8");
    await user.click(screen.getByRole("button", { name: "Remove rating" }));
    expect(screen.queryByRole("slider")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add rating" })).toHaveFocus();
    await user.click(screen.getByRole("button", { name: "Add rating" }));
    expect(screen.getByRole("slider", { name: "Rating" })).toHaveValue("5");
  });

  it("clears a saved rating by omitting it from the replacement PUT", async () => {
    const user = userEvent.setup();
    render(BrewForm, { brew: editBrew, isEdit: true });
    await user.click(screen.getByRole("button", { name: "Remove rating" }));
    await user.click(screen.getByRole("button", { name: "Update brew" }));
    await waitFor(() => expect(mocks.navigate).toHaveBeenCalled());
    expect(savedInput(vi.mocked(fetch))).not.toHaveProperty("rating");
  });

  it("keeps a single draft across modes, with optional grinder-dependent setting and a details summary", async () => {
    const user = userEvent.setup();
    render(BrewForm, { brew: null, isEdit: false });
    await select(user, "Search brew methods", "V60");
    await user.click(screen.getByRole("button", { name: "Detailed" }));
    expect(screen.queryByLabelText("Grind setting")).toBeNull();
    await user.type(screen.getByLabelText("Coffee amount (g)"), "18");
    await select(user, "Search grinders", "Ode");
    await user.type(screen.getByLabelText("Grind setting"), "3.5");
    await user.click(screen.getByRole("button", { name: "Quick" }));
    expect(screen.queryByLabelText("Grind setting")).toBeNull();
    expect(screen.getByLabelText("Brewing details added")).toHaveTextContent(
      "3.5",
    );
    await user.click(
      screen.getByRole("button", { name: "Edit brewing details" }),
    );
    expect(screen.getByLabelText("Grind setting")).toHaveValue("3.5");
    expect(screen.getByLabelText("Coffee amount (g)")).toHaveFocus();
    expect(mocks.saveMode.mock.calls).toEqual([["detailed"], ["quick"]]);
  });

  it("confirms grinder changes, restores both controls on cancel, and clears setting on confirmation", async () => {
    const user = userEvent.setup();
    render(BrewForm, { brew: editBrew, isEdit: true });
    vi.mocked(window.confirm).mockReturnValueOnce(false);
    await select(user, "Search grinders", "Niche");
    expect(hidden("grinder_rkey")).toBe("ode");
    expect(
      screen.getByRole("combobox", { name: "Search grinders" }),
    ).toHaveValue("Ode");
    expect(screen.getByLabelText("Grind setting")).toHaveValue("Medium");
    await select(user, "Search grinders", "Niche");
    expect(hidden("grinder_rkey")).toBe("niche");
    expect(screen.getByLabelText("Grind setting")).toHaveValue("");
  });

  it("does not silently associate an old uncalibrated setting with a grinder", async () => {
    const user = userEvent.setup();
    render(BrewForm, {
      brew: { ...editBrew, grinder_rkey: "", grinder_obj: undefined },
      isEdit: true,
    });
    vi.mocked(window.confirm).mockReturnValueOnce(false);
    await select(user, "Search grinders", "Ode");
    expect(hidden("grinder_rkey")).toBe("");
    expect(screen.getByLabelText("Grind setting")).toHaveValue("Medium");
    expect(window.confirm).toHaveBeenCalledWith(
      expect.stringContaining("no grinder"),
    );
  });

  it("clears a stale history method label when changing brewer category", async () => {
    const user = userEvent.setup();
    mocks.computeAutofill.mockImplementation((_brews, key) => ({
      fields: {
        method: {
          value: key.brewerCategory === "pourover" ? "Pour Over" : "Espresso",
          matches: 2,
          total: 2,
        },
      },
    }));
    render(BrewForm, { brew: null, isEdit: false });
    await user.click(screen.getByRole("button", { name: "Detailed" }));
    await select(user, "Search brew methods", "Espresso machine");
    await select(user, "Search coffee beans", /Colombia/);
    await select(user, "Search brew methods", "V60");
    expect(window.confirm).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Save brew" }));
    await waitFor(() => expect(mocks.navigate).toHaveBeenCalled());
    expect(savedInput(vi.mocked(fetch))).toMatchObject({
      brewer_rkey: "v60",
      method: "Pour Over",
    });
  });

  it("preserves saved method text until a confirmed category change, including recipe changes", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn().mockImplementation((url, init) =>
      response(
        String(url).includes("/suggestions/")
          ? []
          : init?.method
            ? { brew: { rkey: "saved" } }
            : recipe("espresso-recipe", {
                brewer_rkey: "espresso",
                brewer_obj: espresso as Recipe["brewer_obj"],
                brewer_type: "espresso",
                pours: [],
              }),
      ),
    );
    vi.stubGlobal("fetch", fetchMock);
    mocks.getData.mockResolvedValue({
      ...data,
      recipes: [recipe("espresso-recipe")],
    });
    render(BrewForm, {
      brew: { ...editBrew, method: "Pour Over", pourover_params: undefined },
      isEdit: true,
    });
    vi.mocked(window.confirm).mockReturnValueOnce(false);
    await select(user, "Search recipes", /espresso-recipe/);
    await waitFor(() => expect(window.confirm).toHaveBeenCalled());
    expect(hidden("brewer_rkey")).toBe("v60");
    await openMore(user);
    expect(screen.getByLabelText("Saved method")).toHaveValue("Pour Over");
    await select(user, "Search recipes", /espresso-recipe/);
    await waitFor(() => expect(hidden("brewer_rkey")).toBe("espresso"));
    await user.click(screen.getByRole("button", { name: "Update brew" }));
    await waitFor(() => expect(mocks.navigate).toHaveBeenCalled());
    expect(savedInput(fetchMock)).toMatchObject({ brewer_rkey: "espresso" });
    expect(savedInput(fetchMock)).not.toHaveProperty("method");
  });

  it("confirms method transitions, preserving common values and clearing only incompatible data", async () => {
    const user = userEvent.setup();
    render(BrewForm, { brew: editBrew, isEdit: true });
    vi.mocked(window.confirm).mockReturnValueOnce(false);
    await select(user, "Search brew methods", "Espresso machine");
    expect(hidden("brewer_rkey")).toBe("v60");
    expect(
      screen.getByRole("combobox", { name: "Search brew methods" }),
    ).toHaveValue("V60");
    await select(user, "Search brew methods", "Espresso machine");
    expect(screen.getByLabelText("Coffee amount (g)")).toHaveValue(18);
    await user.click(screen.getByRole("button", { name: "Quick" }));
    await user.click(screen.getByRole("button", { name: "Update brew" }));
    await waitFor(() => expect(mocks.navigate).toHaveBeenCalled());
    expect(savedInput(vi.mocked(fetch))).toMatchObject({
      coffee_amount: 18,
      water_amount: 300,
      brewer_rkey: "espresso",
    });
    expect(savedInput(vi.mocked(fetch))).not.toHaveProperty("pourover_params");
  });

  it("never autofills Quick after delayed preference/history loads; preserves provenance when returning to Quick", async () => {
    const user = userEvent.setup();
    const pending = deferred<typeof data>();
    mocks.getData.mockReturnValue(pending.promise);
    mocks.computeAutofill.mockReturnValue({
      fields: { coffee_amount: { value: "18", matches: 3, total: 3 } },
    });
    render(BrewForm, {
      brew: { ...editBrew, coffee_amount: 0 } as Brew,
      isEdit: false,
    });
    await user.click(screen.getByRole("button", { name: "Detailed" }));
    await user.click(screen.getByRole("button", { name: "Quick" }));
    mocks.computeAutofill.mockClear();
    pending.resolve(data);
    await waitFor(() =>
      expect(
        screen.getByRole("combobox", { name: "Search coffee beans" }),
      ).toBeTruthy(),
    );
    expect(mocks.computeAutofill).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Detailed" }));
    expect(screen.getByLabelText("Coffee amount (g)")).toHaveValue(18);
    await user.click(screen.getByRole("button", { name: "Quick" }));
    expect(screen.getByRole("button", { name: "Clear autofill" })).toBeTruthy();
    expect(screen.getByLabelText("Brewing details added")).toHaveTextContent(
      "18g coffee",
    );
    await user.click(screen.getByRole("button", { name: "Clear autofill" }));
    expect(
      screen.getByLabelText("Brewing details added"),
    ).not.toHaveTextContent("18g coffee");
  });

  it("does not let late preferences reset an explicit Quick choice, and preference save failures leave the draft usable", async () => {
    const user = userEvent.setup();
    const prefs = deferred<void>();
    mocks.loadPrefs.mockReturnValue(prefs.promise);
    mocks.saveMode.mockRejectedValue(new Error("offline"));
    userPrefs.set({
      loading: true,
      smartAutofillEnabled: true,
      brew_form_mode: "quick",
    });
    render(BrewForm, { brew: null, isEdit: false });
    await select(user, "Search coffee beans", /Colombia/);
    await user.click(screen.getByRole("button", { name: "Detailed" }));
    await user.type(screen.getByLabelText("Coffee amount (g)"), "18");
    await user.click(screen.getByRole("button", { name: "Quick" }));
    userPrefs.set({
      loading: false,
      smartAutofillEnabled: true,
      brew_form_mode: "detailed",
    });
    prefs.resolve();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Quick" })).toHaveAttribute(
        "aria-pressed",
        "true",
      ),
    );
    expect(screen.getByLabelText("Brewing details added")).toHaveTextContent(
      "18g coffee",
    );
    expect(mocks.computeAutofill).not.toHaveBeenCalled();
  });

  it("keeps the current mode when late preferences arrive after editing without a mode choice", async () => {
    const user = userEvent.setup();
    const prefs = deferred<void>();
    mocks.loadPrefs.mockReturnValue(prefs.promise);
    userPrefs.set({
      loading: true,
      smartAutofillEnabled: true,
      brew_form_mode: "quick",
    });
    render(BrewForm, { brew: null, isEdit: false });
    await user.type(screen.getByLabelText("Tasting notes"), "Keep this draft");
    userPrefs.set({
      loading: false,
      smartAutofillEnabled: true,
      brew_form_mode: "detailed",
    });
    prefs.resolve();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Quick" })).toHaveAttribute(
        "aria-pressed",
        "true",
      ),
    );
    expect(screen.getByLabelText("Tasting notes")).toHaveValue("Keep this draft");
    expect(mocks.computeAutofill).not.toHaveBeenCalled();
  });

  it("opens new forms in the loaded preference without persisting an initialization", async () => {
    const prefs = deferred<void>();
    mocks.loadPrefs.mockReturnValue(prefs.promise);
    render(BrewForm, { brew: null, isEdit: false });
    userPrefs.set({
      loading: false,
      smartAutofillEnabled: false,
      brew_form_mode: "detailed",
    });
    prefs.resolve();
    expect(await screen.findByLabelText("Coffee amount (g)")).toBeTruthy();
    expect(mocks.saveMode).not.toHaveBeenCalled();
  });

  it("allows unedited history grind suggestions to clear without destructive confirmation", async () => {
    const user = userEvent.setup();
    userPrefs.set({
      loading: false,
      smartAutofillEnabled: true,
      brew_form_mode: "detailed",
    });
    mocks.computeAutofill.mockReturnValue({
      fields: {
        grinder_rkey: { value: "ode", label: "Ode", matches: 2, total: 2 },
        grind_size: { value: "3", matches: 2, total: 2 },
      },
    });
    render(BrewForm, { brew: null, isEdit: false });
    await select(user, "Search brew methods", "V60");
    await select(user, "Search coffee beans", /Colombia/);
    expect(screen.getByLabelText("Grind setting")).toHaveValue("3");
    mocks.computeAutofill.mockReturnValue(null);
    await select(user, "Search grinders", "Niche");
    expect(screen.getByLabelText("Grind setting")).toHaveValue("");
    expect(window.confirm).not.toHaveBeenCalled();
  });

  it("does not autofill a new Quick draft after a recipe requested in Detailed completes", async () => {
    const user = userEvent.setup();
    const pending = deferred<Response>();
    mocks.getData.mockResolvedValue({
      ...data,
      recipes: [{ rkey: "daily", name: "Daily" }],
    });
    vi.mocked(fetch).mockReturnValue(pending.promise);
    render(BrewForm, { brew: null, isEdit: false });
    await select(user, "Search coffee beans", /Colombia/);
    await user.click(screen.getByRole("button", { name: "Detailed" }));
    await select(user, "Search recipes", "Daily");
    await user.click(screen.getByRole("button", { name: "Quick" }));
    mocks.computeAutofill.mockClear();
    pending.resolve(response(recipe("daily")));
    await screen.findByText("18g coffee · 300g water · 2 pours");
    expect(mocks.computeAutofill).not.toHaveBeenCalled();
    expect(screen.queryByLabelText("Temperature (°F/°C)")).toBeNull();
  });

  it("a recipe cannot bypass destructive method confirmation even in Quick", async () => {
    const user = userEvent.setup();
    const pending = deferred<Response>();
    mocks.getData.mockResolvedValue({
      ...data,
      recipes: [{ rkey: "shot", name: "Shot" }],
    });
    vi.mocked(fetch).mockReturnValue(pending.promise);
    render(BrewForm, { brew: editBrew, isEdit: true });
    await select(user, "Search recipes", "Shot");
    await user.click(screen.getByRole("button", { name: "Quick" }));
    vi.mocked(window.confirm).mockReturnValueOnce(false);
    pending.resolve(
      response(
        recipe("shot", {
          brewer_rkey: "espresso",
          brewer_obj: espresso as Recipe["brewer_obj"],
          coffee_amount: 99,
        }),
      ),
    );
    await waitFor(() => expect(screen.queryByRole("status")).toBeNull());
    expect(hidden("brewer_rkey")).toBe("v60");
    expect(hidden("recipe_rkey")).toBe("");
    expect(screen.getByLabelText("Brewing details added")).toHaveTextContent(
      "18g coffee",
    );
    expect(mocks.computeAutofill).not.toHaveBeenCalled();
  });

  it("keeps a newer explicit brewer when a pending recipe resolves, including during later recipe scaling", async () => {
    const user = userEvent.setup();
    const pending = deferred<Response>();
    vi.mocked(fetch).mockImplementation(async (_url, init) =>
      init?.method ? response({ brew: { rkey: "saved" } }) : pending.promise,
    );
    render(BrewForm, { brew: null, recipeRKey: "daily", isEdit: false });
    await select(user, "Search coffee beans", /Colombia/);
    await select(user, "Search brew methods", "Espresso machine");
    await user.click(screen.getByRole("button", { name: "Detailed" }));
    await user.type(screen.getByLabelText("Yield weight (g)"), "36");
    await user.click(screen.getByRole("button", { name: "Quick" }));
    pending.resolve(response(recipe("daily")));
    await screen.findByText(/Recipe pours were not copied/);
    expect(hidden("brewer_rkey")).toBe("espresso");
    expect(window.confirm).not.toHaveBeenCalled();
    await user.click(
      screen.getAllByRole("button", { name: "Edit brewing details" })[0],
    );
    expect(screen.getByLabelText("Coffee (g)")).toHaveFocus();
    await fireEvent.input(screen.getByLabelText("Water (g)"), {
      target: { value: "450" },
    });
    await user.click(screen.getByRole("button", { name: "Save brew" }));
    await waitFor(() => expect(mocks.navigate).toHaveBeenCalled());
    expect(savedInput(vi.mocked(fetch))).toMatchObject({
      brewer_rkey: "espresso",
      water_amount: 450,
      espresso_params: { yield_weight: 36 },
    });
    expect(savedInput(vi.mocked(fetch))).not.toHaveProperty("pours");
    expect(savedInput(vi.mocked(fetch))).not.toHaveProperty("pourover_params");
  });

  it("does not let a recipe without a saved brewer change method or copy incompatible pours", async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockImplementation(async (_url, init) =>
      init?.method
        ? response({ brew: { rkey: "saved" } })
        : response(
            recipe("unassociated", {
              brewer_rkey: "",
              brewer_obj: undefined,
              brewer_type: "pourover",
            }),
          ),
    );
    mocks.getData.mockResolvedValue({
      ...data,
      recipes: [{ rkey: "unassociated", name: "Unassociated" }],
    });
    const shot = {
      ...editBrew,
      brewer_rkey: "espresso",
      brewer_obj: espresso,
      pourover_params: undefined,
      espresso_params: {
        yield_weight: 36,
        pressure: 9,
        pre_infusion_seconds: 5,
      },
    };
    render(BrewForm, { brew: shot, isEdit: true });
    await select(user, "Search recipes", "Unassociated");
    await screen.findByText(/Recipe pours were not copied/);
    expect(hidden("brewer_rkey")).toBe("espresso");
    expect(window.confirm).not.toHaveBeenCalled(); // No method transition or destructive clearing.
    await user.click(screen.getByRole("button", { name: "Update brew" }));
    await waitFor(() => expect(mocks.navigate).toHaveBeenCalled());
    expect(savedInput(vi.mocked(fetch))).toMatchObject({
      brewer_rkey: "espresso",
      espresso_params: shot.espresso_params,
    });
    expect(savedInput(vi.mocked(fetch))).not.toHaveProperty("pours");
  });

  it("distinguishes collection authentication expiry from failures and empty collections, preserving the draft", async () => {
    const user = userEvent.setup();
    const modal = vi.fn();
    window.__showSessionExpiredModal = modal;
    mocks.getData.mockImplementation((force: boolean) =>
      force
        ? Promise.reject(new Error("Failed to fetch data: 401"))
        : Promise.resolve(data),
    );
    render(BrewForm, { brew: editBrew, isEdit: true });
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Your session expired",
    );
    expect(screen.queryByLabelText("Before your first brew")).toBeNull();
    expect(modal).toHaveBeenCalledTimes(1);
    expect(mocks.warn).toHaveBeenCalled();
    expect(screen.getByLabelText("Tasting notes")).toHaveValue("Sweet");
    mocks.getData.mockResolvedValue(data);
    await user.click(screen.getByRole("button", { name: "Retry loading" }));
    await waitFor(() => expect(screen.queryByRole("alert")).toBeNull());
    delete window.__showSessionExpiredModal;
  });

  it("surfaces recipe authentication expiry through the shared session prompt without erasing the draft", async () => {
    const user = userEvent.setup();
    const modal = vi.fn();
    window.__showSessionExpiredModal = modal;
    mocks.getData.mockResolvedValue({
      ...data,
      recipes: [{ rkey: "expired", name: "Expired" }],
    });
    vi.mocked(fetch).mockResolvedValue(new Response("", { status: 401 }));
    render(BrewForm, { brew: editBrew, isEdit: true });
    await select(user, "Search recipes", "Expired");
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "details have been kept",
    );
    expect(modal).toHaveBeenCalledTimes(1);
    expect(hidden("brewer_rkey")).toBe("v60");
    expect(screen.getByLabelText("Coffee amount (g)")).toHaveValue(18);
    delete window.__showSessionExpiredModal;
  });

  it("offers separate prerequisite creation without leaving the draft, and distinguishes failed loads", async () => {
    mocks.getData.mockResolvedValue({ ...data, beans: [], brewers: [] });
    render(BrewForm, {
      brew: null,
      recipeRKey: "linked",
      recipeOwnerDID: "did:plc:bob",
      isEdit: false,
    });
    const panel = await screen.findByLabelText("Before your first brew");
    expect(
      within(panel).getByRole("link", { name: "Add a bean" }),
    ).toHaveAttribute("target", "_blank");
    expect(
      within(panel).getByRole("link", { name: "Add a brewer" }),
    ).toHaveAttribute("href", "/brewers/new");
    cleanup();
    mocks.getData.mockImplementation((force: boolean) =>
      force ? Promise.reject(new Error("offline")) : Promise.resolve(null),
    );
    render(BrewForm, { brew: null, isEdit: false });
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Could not load",
    );
    expect(screen.queryByLabelText("Before your first brew")).toBeNull();
  });

  it("warns before abandoning a changed draft and retains it after save errors", async () => {
    const user = userEvent.setup();
    render(BrewForm, { brew: editBrew, isEdit: true });
    await user.type(screen.getByLabelText("Tasting notes"), "!");
    const cancel = vi.fn();
    vi.mocked(window.confirm).mockReturnValueOnce(false);
    mocks.beforeNavigate.mock.calls[0][0]({ cancel });
    expect(cancel).toHaveBeenCalled();
    const event = new Event("beforeunload", { cancelable: true });
    window.dispatchEvent(event);
    expect(event.defaultPrevented).toBe(true);
    vi.mocked(fetch).mockRejectedValue(new Error("offline"));
    await user.click(screen.getByRole("button", { name: "Update brew" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Your draft is still here",
    );
    expect(screen.getByLabelText("Tasting notes")).toHaveValue("Sweet!");
  });

  it("retains foreign recipe ownership, copies in Quick, and summarizes without exposing measurements", async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockImplementation(async (url, init) =>
      init?.method
        ? response({ brew: { rkey: "saved" } })
        : response(recipe("foreign")),
    );
    render(BrewForm, {
      brew: null,
      recipeRKey: "foreign",
      recipeOwnerDID: "did:plc:bob",
      isEdit: false,
    });
    await screen.findByText("18g coffee · 300g water · 2 pours");
    expect(fetch).toHaveBeenCalledWith(
      "/api/recipes/foreign?owner=did%3Aplc%3Abob",
      expect.anything(),
    );
    expect(
      screen.getByRole("combobox", { name: "Search recipes" }),
    ).toBeTruthy();
    expect(screen.queryByLabelText("Coffee amount (g)")).toBeNull();
    await select(user, "Search coffee beans", /Colombia/);
    await user.click(screen.getByRole("button", { name: "Save brew" }));
    await waitFor(() => expect(mocks.navigate).toHaveBeenCalled());
    expect(savedInput(vi.mocked(fetch))).toMatchObject({
      recipe_rkey: "foreign",
      recipe_owner_did: "did:plc:bob",
      brewer_rkey: "v60",
      coffee_amount: 18,
    });
    expect(mocks.computeAutofill).not.toHaveBeenCalled();
  });

  it("never accepts an unmatched foreign brewer rkey, even when it collides with an own rkey", async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockResolvedValue(
      response(
        recipe("foreign", {
          brewer_rkey: "v60",
          brewer_obj: {
            ...v60,
            name: "Foreign press",
            brewer_type: "immersion",
          } as Recipe["brewer_obj"],
          brewer_type: "immersion",
        }),
      ),
    );
    render(BrewForm, {
      brew: null,
      recipeRKey: "foreign",
      recipeOwnerDID: "did:plc:bob",
      isEdit: false,
    });
    await screen.findByText(/This recipe has no matching brewer/);
    expect(hidden("brewer_rkey")).toBe("");
    await select(user, "Search coffee beans", /Colombia/);
    await user.click(screen.getByRole("button", { name: "Save brew" }));
    expect(screen.getByRole("alert")).toHaveTextContent("Choose a brewer");
    expect(
      screen.getByRole("combobox", { name: "Search brew methods" }),
    ).toHaveFocus();
    expect(mocks.navigate).not.toHaveBeenCalled();
  });

  it.each(["name", "type", "reference"])(
    "resolves a foreign brewer by the existing %s policy to an own saved record",
    async (policy) => {
      const user = userEvent.setup();
      const own = {
        ...v60,
        rkey: "own-v60",
        brewer_type: policy === "reference" ? "Pour Over" : "pourover",
      };
      const source = {
        ...v60,
        rkey: "foreign-v60",
        brewer_type: "pourover",
        name: policy === "name" ? "v60" : "Different name",
      };
      mocks.getData.mockResolvedValue({
        ...data,
        brewers: [
          {
            ...own,
            source_ref:
              policy === "reference"
                ? "at://did:plc:bob/social.arabica.alpha.brewer/foreign-v60"
                : undefined,
          },
        ],
      });
      vi.mocked(fetch).mockImplementation(async (_url, init) =>
        init?.method
          ? response({ brew: { rkey: "saved" } })
          : response(
              recipe("foreign", {
                brewer_rkey: source.rkey,
                brewer_obj: source as Recipe["brewer_obj"],
              }),
            ),
      );
      render(BrewForm, {
        brew: null,
        recipeRKey: "foreign",
        recipeOwnerDID: "did:plc:bob",
        isEdit: false,
      });
      await screen.findByText("18g coffee · 300g water · 2 pours");
      expect(hidden("brewer_rkey")).toBe("own-v60");
      await select(user, "Search coffee beans", /Colombia/);
      await user.click(screen.getByRole("button", { name: "Save brew" }));
      await waitFor(() => expect(mocks.navigate).toHaveBeenCalled());
      expect(savedInput(vi.mocked(fetch))).toMatchObject({
        brewer_rkey: "own-v60",
        recipe_owner_did: "did:plc:bob",
      });
    },
  );

  it("waits for the own collection before accepting a fast linked-recipe response", async () => {
    const pending = deferred<typeof data>();
    mocks.getData.mockReturnValue(pending.promise);
    vi.mocked(fetch).mockResolvedValue(response(recipe("linked")));
    render(BrewForm, { brew: null, recipeRKey: "linked", isEdit: false });
    expect(hidden("brewer_rkey")).toBe("");
    pending.resolve(data);
    await screen.findByText("18g coffee · 300g water · 2 pours");
    expect(hidden("brewer_rkey")).toBe("v60");
  });

  it("ignores recipe responses after clearing or superseding selection, including failures", async () => {
    const user = userEvent.setup();
    const first = deferred<Response>();
    const second = deferred<Response>();
    mocks.getData.mockResolvedValue({
      ...data,
      recipes: [
        { rkey: "one", name: "One" },
        { rkey: "two", name: "Two" },
      ],
    });
    vi.mocked(fetch).mockImplementation((url) =>
      String(url).includes("/one") ? first.promise : second.promise,
    );
    render(BrewForm, { brew: null, isEdit: false });
    await select(user, "Search recipes", "One");
    await select(user, "Search recipes", "Two");
    second.resolve(response(recipe("two", { coffee_amount: 22 })));
    await screen.findByText("22g coffee · 300g water · 2 pours");
    first.resolve(response(recipe("one", { coffee_amount: 99 })));
    await fireEvent.blur(
      screen.getByRole("combobox", { name: "Search recipes" }),
    );
    expect(screen.queryByText(/99g coffee/)).toBeNull();
    const third = deferred<Response>();
    vi.mocked(fetch).mockReturnValue(third.promise);
    await select(user, "Search recipes", "One");
    const combo = screen
      .getByRole("combobox", { name: "Search recipes" })
      .closest(".combo-select")!;
    await user.click(
      within(combo as HTMLElement).getByRole("button", {
        name: "Clear selection",
      }),
    );
    third.resolve(response(recipe("one", { coffee_amount: 99 })));
    await user.click(screen.getByRole("button", { name: "Detailed" }));
    expect(screen.getByLabelText("Coffee amount (g)")).toHaveValue(22);
  });

  it("does not erase draft values or the prior recipe when a fetch fails", async () => {
    const user = userEvent.setup();
    mocks.getData.mockResolvedValue({
      ...data,
      recipes: [{ rkey: "broken", name: "Broken" }],
    });
    vi.mocked(fetch).mockResolvedValue(new Response("", { status: 500 }));
    render(BrewForm, { brew: editBrew, isEdit: true });
    await select(user, "Search recipes", "Broken");
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "details have been kept",
    );
    expect(screen.getByLabelText("Coffee amount (g)")).toHaveValue(18);
    expect(hidden("recipe_rkey")).toBe("");
  });
});
