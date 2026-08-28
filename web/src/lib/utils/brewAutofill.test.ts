import { describe, expect, it } from "vitest";
import { computeAutofill } from "./brewAutofill";
import type { AutofillField, AutofillResult } from "./brewAutofill";
import type { Brew } from "../types/entity_view";

const DAY = 86_400_000;
const EPOCH = 1_700_000_000_000;

/** ISO timestamp n days after a fixed epoch, newest = largest n. */
function t(n: number): string {
  return new Date(EPOCH + n * DAY).toISOString();
}

let seq = 0;

function makeBrew(overrides: Partial<Brew> & { created_at: string }): Brew {
  return {
    rkey: `b${seq++}`,
    bean_rkey: "bean-1",
    recipe_rkey: "",
    method: "pourover",
    temperature: 93,
    water_amount: 300,
    coffee_amount: 20,
    time_seconds: 180,
    grind_size: "medium-coarse",
    grinder_rkey: "grinder-1",
    brewer_rkey: "brewer-1",
    tasting_notes: "",
    rating: 0,
    pours: [],
    ...overrides,
  };
}

/** Builds brews at rank 0..count-1 (0 = newest) with the given temperature. */
function temperatureBrews(count: number, assign: (rank: number) => number | undefined): Brew[] {
  const brews: Brew[] = [];
  for (let rank = 0; rank < count; rank += 1) {
    const value = assign(rank);
    brews.push(
      makeBrew({
        temperature: value ?? 0,
        created_at: t(count - rank),
      }),
    );
  }
  return brews;
}

function suggestionOf(result: AutofillResult, field: AutofillField) {
  return result.fields[field] as NonNullable<AutofillResult["fields"][AutofillField]>;
}

describe("brewAutofill", () => {
  it("returns null with fewer than 2 matching brews", () => {
    expect(computeAutofill([], { beanRKey: "bean-1" })).toBeNull();
    const one = makeBrew({ bean_rkey: "bean-1", created_at: t(0) });
    expect(computeAutofill([one], { beanRKey: "bean-1" })).toBeNull();
  });

  it("uses the pair tier when a recipe matches 2+ brews, else falls back to bean-only", () => {
    const beanOnly = [
      makeBrew({ recipe_rkey: "", created_at: t(10) }),
      makeBrew({ recipe_rkey: "", created_at: t(9) }),
      makeBrew({ recipe_rkey: "", created_at: t(8) }),
      makeBrew({ recipe_rkey: "", created_at: t(7) }),
    ];
    const recipeA = [t(6), t(5), t(4)].map((d) =>
      makeBrew({ recipe_rkey: "recipe-A", created_at: d }),
    );
    const recipeB = [makeBrew({ recipe_rkey: "recipe-B", created_at: t(3) })];
    const brews = [...beanOnly, ...recipeA, ...recipeB];

    const withA = computeAutofill(brews, { beanRKey: "bean-1", recipeRKey: "recipe-A" });
    expect(withA).not.toBeNull();
    expect(withA!.basis).toBe("bean+recipe");
    expect(withA!.totalBrews).toBe(3);

    const withB = computeAutofill(brews, { beanRKey: "bean-1", recipeRKey: "recipe-B" });
    expect(withB).not.toBeNull();
    expect(withB!.basis).toBe("bean");
    expect(withB!.totalBrews).toBe(8);
  });

  it("prefers a value repeated in the recent brews over an older majority", () => {
    // Newest 5 brews at 96, older 5 at 95.
    const brews = temperatureBrews(10, (r) => (r < 5 ? 96 : 95));
    const result = computeAutofill(brews, { beanRKey: "bean-1" });
    expect(result).not.toBeNull();
    const suggestion = suggestionOf(result!, "temperature");
    expect(suggestion.value).toBe("96");
    expect(suggestion.matches).toBe(5);
    expect(suggestion.total).toBe(10);
  });

  it("does not let a single newest deviation flip the winner", () => {
    // Newest brew deviates to 94; the run of 9 at 95 outweighs it.
    const brews = temperatureBrews(10, (r) => (r === 0 ? 94 : 95));
    const result = computeAutofill(brews, { beanRKey: "bean-1" });
    expect(result).not.toBeNull();
    const suggestion = suggestionOf(result!, "temperature");
    expect(suggestion.value).toBe("95");
    expect(suggestion.matches).toBe(9);
    expect(suggestion.total).toBe(10);
  });

  it("breaks weight ties toward the value in the newest brew", () => {
    // Distribute 10 and 11 so their decayed weights are effectively tied;
    // 10 appears in the newer brew (rank 2 vs 3) and should win.
    const brews = temperatureBrews(
      10,
      (r) => [undefined, undefined, 10, 11, undefined, undefined, 11, 10, 11, 10][r] as number | undefined,
    );
    const result = computeAutofill(brews, { beanRKey: "bean-1" });
    expect(result).not.toBeNull();
    const suggestion = suggestionOf(result!, "temperature");
    expect(suggestion.value).toBe("10");
    expect(suggestion.matches).toBe(3);
    expect(suggestion.total).toBe(10);
  });

  it("normalizes numeric fields to strings usable by the form", () => {
    const brews = [t(1), t(0)].map((d) => makeBrew({ temperature: 93, created_at: d }));
    const result = computeAutofill(brews, { beanRKey: "bean-1" });
    expect(result).not.toBeNull();
    expect(suggestionOf(result!, "temperature").value).toBe("93");
  });

  it("matches recipes cross-user only by the selected owner DID", () => {
    const owner1 = [t(3), t(2), t(1)].map((d) =>
      makeBrew({
        recipe_rkey: "recipe-A",
        recipe_owner_did: "did:owner1",
        temperature: 90,
        created_at: d,
      }),
    );
    const owner2 = [t(7), t(6)].map((d) =>
      makeBrew({
        recipe_rkey: "recipe-A",
        recipe_owner_did: "did:owner2",
        temperature: 99,
        created_at: d,
      }),
    );
    const brews = [...owner1, ...owner2];

    const result = computeAutofill(brews, {
      beanRKey: "bean-1",
      recipeRKey: "recipe-A",
      recipeOwnerDID: "did:owner1",
    });
    expect(result).not.toBeNull();
    expect(result!.basis).toBe("bean+recipe");
    expect(result!.totalBrews).toBe(3);
    expect(suggestionOf(result!, "temperature").value).toBe("90");
  });

  it("never suggests tasting_notes, rating, or pours", () => {
    const brews = [
      makeBrew({ tasting_notes: "berry", rating: 4, pours: [{ pour_number: 1, water_amount: 60, time_seconds: 10, created_at: t(0) }], created_at: t(2) }),
      makeBrew({ tasting_notes: "berry", rating: 5, pours: [{ pour_number: 1, water_amount: 80, time_seconds: 12, created_at: t(0) }], created_at: t(1) }),
    ];
    const result = computeAutofill(brews, { beanRKey: "bean-1" });
    expect(result).not.toBeNull();
    const keys = Object.keys(result!.fields) as AutofillField[];
    expect(keys).not.toContain("tasting_notes");
    expect(keys).not.toContain("rating");
    expect(keys).not.toContain("pours");
  });

  it("resolves ref fields to a label and rkey from the hydrated objects", () => {
    const brews = [
      makeBrew({
        grinder_rkey: "grinder-1",
        grinder_obj: { rkey: "grinder-1", name: "Comandante C40", grinder_type: "", burr_type: "", notes: "", link: "", created_at: "" },
        created_at: t(1),
      }),
      makeBrew({
        grinder_rkey: "grinder-1",
        grinder_obj: { rkey: "grinder-1", name: "Comandante C40", grinder_type: "", burr_type: "", notes: "", link: "", created_at: "" },
        created_at: t(0),
      }),
    ];
    const result = computeAutofill(brews, { beanRKey: "bean-1" });
    expect(result).not.toBeNull();
    const suggestion = suggestionOf(result!, "grinder_rkey");
    expect(suggestion.value).toBe("grinder-1");
    expect(suggestion.rkey).toBe("grinder-1");
    expect(suggestion.label).toBe("Comandante C40");
    expect(suggestion.matches).toBe(2);
    expect(suggestion.total).toBe(2);
  });

  it("omits a field entirely when it has no non-empty value across matching brews", () => {
    const brews = [
      makeBrew({ water_amount: 0, created_at: t(1) }),
      makeBrew({ water_amount: 0, created_at: t(0) }),
    ];
    const result = computeAutofill(brews, { beanRKey: "bean-1" });
    expect(result).not.toBeNull();
    expect(result!.fields).not.toHaveProperty("water_amount");
  });

  it("does not return suggestions when the bean matches nothing", () => {
    const brews = [makeBrew({ bean_rkey: "other-bean", created_at: t(1) })];
    expect(computeAutofill(brews, { beanRKey: "bean-1" })).toBeNull();
  });
});
