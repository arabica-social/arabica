import { cleanup, fireEvent, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import DraftEditor from "../../src/lib/components/ai/DraftEditor.svelte";
import { createSession, DraftEngine } from "../../src/lib/ai/draft";
import type { DraftPatch } from "../../src/lib/ai/types";
const pours = [
  { water_amount: 70, time_seconds: 30 },
  { water_amount: 230, time_seconds: 60 },
];
const recipe = {
  rkey: "orea3",
  name: "Orea 3 pour",
  brewer_rkey: "orea",
  brewer_type: "pourover",
  coffee_amount: 18,
  water_amount: 300,
  pours,
  notes: "Swirl after each pour.",
};
const library = {
  recipes: [recipe],
  brewers: [{ rkey: "orea", name: "Orea", brewer_type: "pourover" }],
};
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});
function editor(fields: DraftPatch = {}, extra = {}) {
  const onchange = vi.fn();
  const view = render(DraftEditor, {
    draft: { kind: "brew", fields },
    library,
    onchange,
    ...extra,
  });
  return { ...view, onchange };
}
describe("own recipe draft editor", () => {
  it("uses native local options and emits one normalized controlled patch without network", async () => {
    const fetcher = vi.fn();
    vi.stubGlobal("fetch", fetcher);
    const { onchange } = editor({ brewer_rkey: "orea" });
    expect(
      screen.getByRole("option", { name: "Orea 3 pour" }),
    ).toBeInTheDocument();
    await fireEvent.change(screen.getByLabelText("Recipe"), {
      target: { value: "orea3" },
    });
    expect(onchange).toHaveBeenCalledExactlyOnceWith({
      recipe_rkey: "orea3",
      recipe_owner_did: null,
      coffee_amount: 18,
      water_amount: 300,
      pours,
    });
    expect(screen.getByLabelText("Brewer *")).toHaveValue("orea");
    expect(fetcher).not.toHaveBeenCalled();
    // Parent did not accept the patch: selection returns to the canonical draft.
    expect(screen.getByLabelText("Recipe")).toHaveValue("");
  });
  it("copies relative pour start times without inventing bloom amounts or durations", async () => {
    const { onchange } = editor(
      {
        brewer_rkey: "orea",
        "pourover_params.bloom_water": 30,
        "pourover_params.bloom_seconds": 20,
      },
      {
        library: {
          ...library,
          recipes: [
            { ...recipe, pours: [{ water_amount: 70, time_seconds: 45 }] },
          ],
        },
      },
    );
    await fireEvent.change(screen.getByLabelText("Recipe"), {
      target: { value: "orea3" },
    });
    expect(onchange.mock.calls[0][0]).toMatchObject({
      pours: [{ water_amount: 70, time_seconds: 45 }],
    });
    expect(onchange.mock.calls[0][0]).not.toHaveProperty(
      "pourover_params.bloom_water",
    );
    expect(onchange.mock.calls[0][0]).not.toHaveProperty(
      "pourover_params.bloom_seconds",
    );
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });
  it.each(["water_amount", "time_seconds"])(
    "does not copy unsafe integer pour %s",
    async (key) => {
      const { onchange } = editor(
        { brewer_rkey: "orea" },
        {
          library: {
            ...library,
            recipes: [
              {
                ...recipe,
                pours: [
                  {
                    water_amount: 70,
                    time_seconds: 45,
                    [key]: Number.MAX_SAFE_INTEGER + 1,
                  },
                ],
              },
            ],
          },
        },
      );
      await fireEvent.change(screen.getByLabelText("Recipe"), {
        target: { value: "orea3" },
      });
      expect(onchange.mock.calls[0][0]).not.toHaveProperty("pours");
      expect(screen.getByRole("status")).toHaveTextContent(
        "Recipe pours were not copied",
      );
    },
  );
  it("shows recipe instructions and actual amounts on reload, never as tasting notes", () => {
    editor({ recipe_rkey: "orea3" });
    expect(screen.getByLabelText("Recipe")).toHaveValue("orea3");
    expect(screen.getByText("Swirl after each pour.")).toBeInTheDocument();
    expect(screen.getByText("70 g at 30 s")).toBeInTheDocument();
    expect(screen.getByText(/18 g coffee · 300 g water/)).toBeInTheDocument();
    expect(screen.getByLabelText("Tasting notes")).toHaveValue("");
  });
  it("keeps a selected unavailable reference visible without fetching", () => {
    editor({ recipe_rkey: "missing" });
    expect(screen.getByLabelText("Recipe")).toHaveValue("missing");
    expect(
      screen.getByRole("option", { name: "missing (unavailable)" }),
    ).toBeInTheDocument();
  });
  it("clears only the paired reference, preserving measured details", async () => {
    const { onchange } = editor({
      recipe_rkey: "orea3",
      coffee_amount: 20,
      pours,
    });
    await fireEvent.change(screen.getByLabelText("Recipe"), {
      target: { value: "" },
    });
    expect(onchange).toHaveBeenCalledExactlyOnceWith({
      recipe_rkey: null,
      recipe_owner_did: null,
    });
  });
  it("does not guess a brewer by name or type when linked own brewer is absent", async () => {
    const { onchange } = editor(
      {},
      {
        library: {
          ...library,
          recipes: [{ ...recipe, brewer_rkey: "foreign" }],
        },
      },
    );
    await fireEvent.change(screen.getByLabelText("Recipe"), {
      target: { value: "orea3" },
    });
    const patch = onchange.mock.calls[0][0];
    expect(patch).not.toHaveProperty("brewer_rkey");
    expect(patch).not.toHaveProperty("pours");
    expect(screen.getByRole("status")).toHaveTextContent(
      "Choose one of your saved brewers",
    );
  });
  it("never substitutes a colliding own brewer for a foreign recipe brewer", async () => {
    const { onchange } = editor(
      {},
      {
        library: {
          ...library,
          recipes: [
            {
              ...recipe,
              brewer_obj: {
                rkey: "orea",
                name: "Foreign Orea",
                brewer_type: "pourover",
              },
            },
          ],
        },
      },
    );
    await fireEvent.change(screen.getByLabelText("Recipe"), {
      target: { value: "orea3" },
    });
    expect(onchange.mock.calls[0][0]).not.toHaveProperty("brewer_rkey");
    expect(onchange.mock.calls[0][0]).not.toHaveProperty("pours");
    expect(screen.getByLabelText("Brewer *")).toHaveValue("");
    expect(screen.getByRole("status")).toHaveTextContent(
      "Choose one of your saved brewers",
    );
  });
  it("keeps existing pours if authoritative recipe category is unavailable", async () => {
    const { onchange } = editor(
      { brewer_rkey: "orea", pours },
      {
        library: {
          ...library,
          recipes: [{ ...recipe, brewer_type: undefined }],
        },
      },
    );
    await fireEvent.change(screen.getByLabelText("Recipe"), {
      target: { value: "orea3" },
    });
    expect(onchange.mock.calls[0][0]).not.toHaveProperty("pours");
    expect(onchange.mock.calls[0][0]).not.toHaveProperty("brewer_rkey");
  });
  it("does not round fractional amounts; fills only safe partial defaults", async () => {
    const { onchange } = editor(
      { coffee_amount: 20, brewer_rkey: "orea" },
      {
        library: { ...library, recipes: [{ ...recipe, coffee_amount: 18.5 }] },
      },
    );
    await fireEvent.change(screen.getByLabelText("Recipe"), {
      target: { value: "orea3" },
    });
    expect(onchange.mock.calls[0][0]).not.toHaveProperty("coffee_amount");
    expect(onchange.mock.calls[0][0]).toMatchObject({
      water_amount: 300,
      pours,
    });
    expect(screen.getByRole("status")).toHaveTextContent("18.5 g");
    expect(screen.getByLabelText("Coffee (g)")).toHaveValue(20);
  });
  it("preserves unknown-category preparation and refuses incompatible pours", async () => {
    const { onchange } = editor(
      { brewer_rkey: "espresso", pours },
      {
        brewerType: "espresso",
        library: { recipes: [{ ...recipe, brewer_rkey: "missing" }] },
      },
    );
    await fireEvent.change(screen.getByLabelText("Recipe"), {
      target: { value: "orea3" },
    });
    expect(onchange.mock.calls[0][0]).not.toHaveProperty("pours");
    expect(screen.getByRole("status")).toHaveTextContent(
      "compatible saved pour-over brewer",
    );
  });
  it("freezes every field while saving", () => {
    const { container } = editor(
      { recipe_rkey: "orea3", pours },
      { disabled: true },
    );
    for (const control of container.querySelectorAll(
      "input, select, textarea, button",
    ))
      expect(control).toBeDisabled();
  });
  it("passes manual brewer changes through the recipe dependency guard", async () => {
    const engine = new DraftEngine(
      createSession("did:plc:alice", "chat", "brew"),
    );
    engine.edit(
      {
        recipe_rkey: "orea3",
        brewer_rkey: "espresso",
        "espresso_params.pressure": 9,
      },
      { brewerType: "espresso" },
    );
    let result;
    const onchange = vi.fn((patch, brewerType) => {
      result = engine.edit(patch, { brewerType });
    });
    render(DraftEditor, {
      draft: engine.snapshot().draft,
      brewerType: "espresso",
      library,
      onchange,
    });
    await fireEvent.change(screen.getByLabelText("Brewer *"), {
      target: { value: "orea" },
    });
    expect(result).toMatchObject({
      status: "needs-confirmation",
      requiredClears: [
        "recipe_rkey",
        "recipe_owner_did",
        "espresso_params.pressure",
      ],
    });
    expect(engine.snapshot().draft.fields).toHaveProperty(
      "recipe_rkey",
      "orea3",
    );
    expect(engine.snapshot().draft.fields).toHaveProperty(
      "brewer_rkey",
      "espresso",
    );
  });
});
