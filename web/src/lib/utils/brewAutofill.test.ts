import { describe, expect, it } from "vitest";
import { computeAutofill } from "./brewAutofill";
import type { AutofillField, AutofillResult } from "./brewAutofill";
import type {
  Brew,
  EspressoParams,
  PouroverParams,
} from "../types/entity_view";

const espresso = (values: Partial<EspressoParams>): EspressoParams => ({
  yield_weight: 0,
  pressure: 0,
  pre_infusion_seconds: 0,
  ...values,
});
const pourover = (values: Partial<PouroverParams>): PouroverParams => ({
  bloom_water: 0,
  bloom_seconds: 0,
  drawdown_seconds: 0,
  bypass_water: 0,
  filter: "",
  ...values,
});

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
function temperatureBrews(
  count: number,
  assign: (rank: number) => number | undefined,
): Brew[] {
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
  return result.fields[field] as NonNullable<
    AutofillResult["fields"][AutofillField]
  >;
}

describe("brewAutofill", () => {
  it("does not suggest conflicting method text from a mixed historical record", () => {
    const history = [t(2), t(1)].map((created_at) =>
      makeBrew({
        created_at,
        method: "Espresso",
        brewer_obj: {
          rkey: "brewer-1",
          name: "V60",
          brewer_type: "pourover",
          description: "",
          link: "",
          created_at: "",
        },
      }),
    );
    const result = computeAutofill(history, {
      beanRKey: "bean-1",
      brewerRKey: "brewer-1",
      brewerCategory: "pourover",
    })!;
    expect(result.fields.coffee_amount?.value).toBe("20");
    expect(result.fields.method).toBeUndefined();
  });

  it("uses only the selected category and ignores mixed historical method parameters", () => {
    const brews = [
      makeBrew({
        created_at: t(4),
        method: "espresso",
        brewer_rkey: "espresso",
        temperature: 92,
        espresso_params: espresso({ yield_weight: 36 }),
        pourover_params: pourover({ bloom_water: 60 }),
      }),
      makeBrew({
        created_at: t(3),
        method: "espresso",
        brewer_rkey: "espresso",
        temperature: 92,
        espresso_params: espresso({ yield_weight: 36 }),
      }),
      makeBrew({
        created_at: t(2),
        method: "pour-over",
        temperature: 96,
        pourover_params: pourover({ bloom_water: 50 }),
        espresso_params: espresso({ pressure: 9 }),
      }),
    ];
    const result = computeAutofill(brews, {
      beanRKey: "bean-1",
      brewerRKey: "brewer-1",
      brewerCategory: "Pour Over",
    })!;
    expect(result.fields.temperature?.value).toBe("96");
    expect(result.fields.pourover_bloom_water?.value).toBe("50");
    expect(result.fields.espresso_pressure).toBeUndefined();
    expect(result.fields.espresso_yield_weight).toBeUndefined();
    expect(result.fields.brewer_rkey).toBeUndefined();
  });

  it("selects the brewer before method-specific suggestions", () => {
    const result = computeAutofill(
      [t(2), t(1)].map((created_at) =>
        makeBrew({
          created_at,
          method: "espresso",
          espresso_params: espresso({ pressure: 9 }),
          pourover_params: pourover({ bloom_water: 50 }),
        }),
      ),
      { beanRKey: "bean-1" },
    )!;
    expect(Object.keys(result.fields)[0]).toBe("brewer_rkey");
    expect(result.fields.espresso_pressure?.value).toBe("9");
    expect(result.fields.pourover_bloom_water).toBeUndefined();
  });

  it("takes grind settings only from the selected or suggested grinder", () => {
    const brews = [
      ...[t(5), t(4), t(3)].map((created_at) =>
        makeBrew({ created_at, grinder_rkey: "a", grind_size: "" }),
      ),
      ...[t(2), t(1)].map((created_at) =>
        makeBrew({ created_at, grinder_rkey: "b", grind_size: "12" }),
      ),
    ];
    const suggested = computeAutofill(brews, { beanRKey: "bean-1" })!;
    expect(suggested.fields.grinder_rkey?.value).toBe("a");
    expect(suggested.fields.grind_size).toBeUndefined();
    const selected = computeAutofill(brews, {
      beanRKey: "bean-1",
      grinderRKey: "b",
    })!;
    expect(selected.fields.grinder_rkey).toBeUndefined();
    expect(selected.fields.grind_size?.value).toBe("12");
    expect(
      computeAutofill(brews, { beanRKey: "bean-1", grinderRKey: "new" })!.fields
        .grind_size,
    ).toBeUndefined();
  });

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

    const withA = computeAutofill(brews, {
      beanRKey: "bean-1",
      recipeRKey: "recipe-A",
    });
    expect(withA).not.toBeNull();
    expect(withA!.basis).toBe("bean+recipe");
    expect(withA!.totalBrews).toBe(3);

    const withB = computeAutofill(brews, {
      beanRKey: "bean-1",
      recipeRKey: "recipe-B",
    });
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
      (r) =>
        [undefined, undefined, 10, 11, undefined, undefined, 11, 10, 11, 10][
          r
        ] as number | undefined,
    );
    const result = computeAutofill(brews, { beanRKey: "bean-1" });
    expect(result).not.toBeNull();
    const suggestion = suggestionOf(result!, "temperature");
    expect(suggestion.value).toBe("10");
    expect(suggestion.matches).toBe(3);
    expect(suggestion.total).toBe(10);
  });

  it("normalizes numeric fields to strings usable by the form", () => {
    const brews = [t(1), t(0)].map((d) =>
      makeBrew({ temperature: 93, created_at: d }),
    );
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
      makeBrew({
        tasting_notes: "berry",
        rating: 4,
        pours: [
          {
            pour_number: 1,
            water_amount: 60,
            time_seconds: 10,
            created_at: t(0),
          },
        ],
        created_at: t(2),
      }),
      makeBrew({
        tasting_notes: "berry",
        rating: 5,
        pours: [
          {
            pour_number: 1,
            water_amount: 80,
            time_seconds: 12,
            created_at: t(0),
          },
        ],
        created_at: t(1),
      }),
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
        grinder_obj: {
          rkey: "grinder-1",
          name: "Comandante C40",
          grinder_type: "",
          burr_type: "",
          notes: "",
          link: "",
          created_at: "",
        },
        created_at: t(1),
      }),
      makeBrew({
        grinder_rkey: "grinder-1",
        grinder_obj: {
          rkey: "grinder-1",
          name: "Comandante C40",
          grinder_type: "",
          burr_type: "",
          notes: "",
          link: "",
          created_at: "",
        },
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
