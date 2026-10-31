import { cleanup, fireEvent, render, screen } from "@testing-library/svelte";
import { afterEach, expect, it, vi } from "vitest";
import DraftPreview from "../../src/lib/components/ai/DraftPreview.svelte";
import DraftEditor from "../../src/lib/components/ai/DraftEditor.svelte";
import type { Draft, DraftPatch } from "../../src/lib/ai/types";

const library = {
  beans: [{ rkey: "bean-secret", name: "October Fix" }],
  brewers: [{ rkey: "brewer-secret", name: "Orea", brewer_type: "pourover" }],
  recipes: [
    {
      rkey: "recipe-secret",
      name: "Morning recipe",
      notes: "Swirl gently <script>alert(1)</script>",
    },
  ],
  grinders: [{ rkey: "grinder-secret", name: "K6" }],
};
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});
function preview(fields: DraftPatch = {}) {
  return render(DraftPreview, {
    draft: { kind: "brew", fields },
    library,
    brewerType: "pourover",
  });
}
it("uses the actual bean title, library names and supplied metrics without mutation or network", () => {
  const fetch = vi.fn();
  vi.stubGlobal("fetch", fetch);
  const fields: DraftPatch = Object.freeze({
    bean_rkey: "bean-secret",
    brewer_rkey: "brewer-secret",
    recipe_rkey: "recipe-secret",
    coffee_amount: 18.5,
    water_amount: 296,
    time_seconds: 165,
    temperature: 93.5,
  });
  const { container } = preview(fields);
  expect(screen.getByRole("heading", { name: "October Fix" })).toBeVisible();
  expect(screen.getByText("Orea")).toBeVisible();
  expect(screen.getByText("Morning recipe")).toBeVisible();
  for (const text of ["18.5 g", "296 g", "165 s", "93.5°", "1 : 16"])
    expect(screen.getByText(text, { exact: false })).toBeVisible();
  expect(screen.getByText(/recorded unit/)).toBeInTheDocument();
  expect(container.textContent).not.toContain("secret");
  expect(container.querySelector("button,input,textarea,select")).toBeNull();
  expect(fetch).not.toHaveBeenCalled();
  expect(fields.coffee_amount).toBe(18.5);
});
it("does not invent missing metrics, preparation, results, or completion claims", () => {
  const { container } = preview();
  expect(screen.getByRole("heading", { name: "Brew draft" })).toBeVisible();
  expect(container.querySelector("dl")).toBeNull();
  expect(container.textContent).not.toMatch(
    /0 g|0 s|ratio|Ready|Tasting notes|Pours/,
  );
});
it("preserves explicit zero and fractional values, and never divides by zero", () => {
  const { container } = preview({
    coffee_amount: 0,
    water_amount: 0,
    temperature: 0,
    time_seconds: 0,
    "espresso_params.yield_weight": 0.5,
    "pourover_params.bloom_seconds": 0,
  });
  expect(screen.getAllByText("0 g")).toHaveLength(2);
  expect(screen.getByText("0 s")).toBeVisible();
  expect(screen.getByText("0.5")).toBeVisible();
  expect(container.textContent).not.toMatch(/Coffee : water|Infinity|NaN/);
});
it("uses human fallbacks for unavailable references and does not join external recipe keys", () => {
  const { container } = preview({
    bean_rkey: "private-bean-key",
    brewer_rkey: "private-brewer-key",
    recipe_rkey: "recipe-secret",
    recipe_owner_did: "did:plc:external-secret",
  });
  expect(screen.getByText("Bean selected (unavailable)")).toBeVisible();
  expect(screen.getByText("Brewer selected (unavailable)")).toBeVisible();
  expect(screen.getByText("Recipe selected (unavailable)")).toBeVisible();
  expect(screen.queryByText("Morning recipe")).not.toBeInTheDocument();
  expect(container.textContent).not.toMatch(/private-|secret|did:plc/);
});
it("keeps ordered pours and their actual relative start times without inventing bloom", () => {
  const { container } = preview({
    pours: [
      { water_amount: 50, time_seconds: 0 },
      { water_amount: 120.5, time_seconds: 45 },
    ],
  });
  expect(screen.getAllByRole("listitem").map((li) => li.textContent)).toEqual([
    "50 gat 0 s",
    "120.5 gat 45 s",
  ]);
  expect(
    screen.getByText("Times relative to the start of brewing."),
  ).toBeVisible();
  expect(container.textContent).not.toMatch(/Bloom/);
});
it("discloses all 100 pours rather than truncating longer timelines", async () => {
  preview({
    pours: Array.from({ length: 100 }, (_, i) => ({
      water_amount: i,
      time_seconds: i * 5,
    })),
  });
  expect(screen.getAllByRole("listitem", { hidden: true })).toHaveLength(100);
  expect(screen.getByText("100 pours")).toBeVisible();
  await fireEvent.click(screen.getByText("100 pours"));
  expect(screen.getByText("at 495 s")).toBeVisible();
});
it("renders tasting text safely and omits source recipe instructions", () => {
  const { container } = preview({
    recipe_rkey: "recipe-secret",
    tasting_notes: "Peach <img src=x onerror=alert(1)>",
  });
  const tasting = screen.getByText("Tasting notes").parentElement!;
  expect(tasting).toHaveTextContent("Peach <img src=x onerror=alert(1)>");
  expect(tasting).not.toHaveTextContent("Swirl gently");
  expect(screen.queryByText("Recipe instructions")).not.toBeInTheDocument();
  expect(container.textContent).not.toContain("Swirl gently");
  expect(screen.getByText("Morning recipe")).toBeVisible();
  expect(container.querySelector("script,img")).toBeNull();
});
it.each([
  [
    "bean",
    { name: "Finca", origin: "Colombia", closed: false },
    "Finca",
    "Colombia",
  ],
  ["grinder", { name: "K6", burr_type: "Conical" }, "K6", "Conical"],
  [
    "brewer",
    { name: "Switch", brewer_type: "immersion" },
    "Switch",
    "Immersion",
  ],
] as const)("shows supplied %s facts", (kind, fields, title, fact) => {
  render(DraftPreview, { draft: { kind, fields } as Draft });
  expect(screen.getByRole("heading", { name: title })).toBeVisible();
  expect(screen.getByText(fact)).toBeVisible();
});
it("chat editing has flat common fields without mode controls, defaults, or recipe controls", () => {
  const onchange = vi.fn();
  render(DraftEditor, {
    draft: { kind: "brew", fields: {} },
    brewerType: "pourover",
    presentation: "chat",
    ratingInEditor: false,
    onchange,
  });
  expect(
    screen.queryByRole("group", { name: "Brew logging mode" }),
  ).not.toBeInTheDocument();
  expect(screen.queryByText("Brewing details added")).not.toBeInTheDocument();
  expect(screen.getByLabelText("Coffee (g)")).toBeVisible();
  expect(screen.getByLabelText("Coffee (g)")).toHaveValue(null);
  for (const label of [
    "Bean *",
    "Brewer *",
    "Grinder",
    "Grind setting",
    "Water (g)",
    "Temperature (°F/°C)",
    "Brew time (s)",
    "Tasting notes",
  ])
    expect(screen.getByLabelText(label)).toBeVisible();
  expect(screen.queryByLabelText("Recipe")).not.toBeInTheDocument();
  expect(screen.queryByLabelText("Rating (1–10)")).not.toBeInTheDocument();
  expect(screen.queryByText("Add rating")).not.toBeInTheDocument();
  expect(screen.queryByText("More brewing details")).not.toBeInTheDocument();
  expect(screen.queryByText("Add pour")).not.toBeInTheDocument();
  expect(document.querySelector("details")).toBeNull();
  expect(screen.queryByLabelText(/Method/)).not.toBeInTheDocument();
  expect(onchange).not.toHaveBeenCalled();
});
it("chat editing keeps supplied common values and edits controlled", async () => {
  const onchange = vi.fn();
  render(DraftEditor, {
    draft: { kind: "brew", fields: { temperature: 93.5, rating: 8 } },
    presentation: "chat",
    onchange,
  });
  expect(screen.getByLabelText("Temperature (°F/°C)")).toBeVisible();
  expect(screen.getByLabelText("Temperature (°F/°C)")).toHaveValue(93.5);
  expect(screen.getByLabelText("Rating (1–10)")).toHaveValue(8);
  expect(onchange).not.toHaveBeenCalled();
  await fireEvent.input(screen.getByLabelText("Temperature (°F/°C)"), {
    target: { value: "94" },
  });
  expect(onchange).toHaveBeenLastCalledWith({ temperature: 94 }, undefined);
});

it("does not round a positive derived ratio down to zero", () => {
  preview({ coffee_amount: 1000, water_amount: 1 });
  expect(screen.getByText("1 : 0.001")).toBeVisible();
});

it("chat correction preserves hidden recipe and advanced preparation values", async () => {
  const onchange = vi.fn();
  const fields: DraftPatch = Object.freeze({
    recipe_rkey: "recipe-secret",
    coffee_amount: 18,
    method: "Legacy V60",
    "pourover_params.filter": "paper",
    "pourover_params.bloom_seconds": 30,
    pours: [{ water_amount: 50, time_seconds: 0 }],
  });
  const { container } = render(DraftEditor, {
    draft: { kind: "brew", fields },
    library,
    brewerType: "pourover",
    presentation: "chat",
    ratingInEditor: false,
    onchange,
  });
  expect(container.querySelector("details")).toBeNull();
  expect(container.textContent).not.toMatch(
    /recipe details|Recipe instructions|Swirl gently|More brewing details/,
  );
  for (const label of [
    "Recipe",
    "Filter",
    "Bloom time (s)",
    "Pour 1 water (g)",
    "Pressure (bar)",
  ])
    expect(screen.queryByLabelText(label)).not.toBeInTheDocument();
  expect(
    screen.queryByRole("button", {
      name: /Add pour|Remove pour|Quick|Detailed|rating/i,
    }),
  ).not.toBeInTheDocument();
  expect(onchange).not.toHaveBeenCalled();
  await fireEvent.input(screen.getByLabelText("Coffee (g)"), {
    target: { value: "20" },
  });
  expect(onchange).toHaveBeenCalledExactlyOnceWith(
    { coffee_amount: 20 },
    undefined,
  );
  expect({ ...fields, ...onchange.mock.calls[0][0] }).toEqual({
    ...fields,
    coffee_amount: 20,
  });
  expect(library.recipes[0].notes).toContain("Swirl gently");
});

it.each([
  ["espresso", "Shot time (s)", "Espresso yield (g)"],
  ["immersion", "Steep time (s)", "Water (g)"],
  ["mokapot", "Brew time (s)", "Water (g)"],
])(
  "chat keeps common fields visible with %s category labels",
  (brewerType, timeLabel, waterLabel) => {
    const onchange = vi.fn();
    render(DraftEditor, {
      draft: { kind: "brew", fields: {} },
      presentation: "chat",
      brewerType,
      onchange,
    });
    expect(screen.getByLabelText(timeLabel)).toBeVisible();
    expect(screen.getByLabelText(waterLabel)).toBeVisible();
    expect(screen.getByLabelText("Grinder")).toBeVisible();
    expect(screen.getByLabelText("Grind setting")).toBeVisible();
    expect(screen.getByLabelText("Temperature (°F/°C)")).toBeVisible();
    expect(screen.getByLabelText("Rating (1–10)")).toHaveValue(null);
    expect(screen.queryByText("Add rating")).not.toBeInTheDocument();
    expect(onchange).not.toHaveBeenCalled();
  },
);
