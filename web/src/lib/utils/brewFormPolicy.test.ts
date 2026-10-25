import { describe, expect, it } from "vitest";
import type { Brew } from "../types/entity_view";
import {
  hasPreparationValues,
  initialBrewFormMode,
  normalizeBrewerCategory,
  preparationLayout,
} from "./brewFormPolicy";

const record = (fields: Partial<Brew>) => fields as Brew;

describe("brew form policy", () => {
  it.each([undefined, null, "invalid", "quick", "", false])(
    "defaults new brews to Quick for %s",
    (preference) => {
      expect(initialBrewFormMode(false, null, preference)).toBe("quick");
    },
  );
  it("honors Detailed only on new brews, not observation-only edits", () => {
    expect(initialBrewFormMode(false, null, "detailed")).toBe("detailed");
    expect(
      initialBrewFormMode(
        true,
        record({
          bean_rkey: "bean",
          brewer_rkey: "brewer",
          rating: 8,
          tasting_notes: "Sweet",
        }),
        "detailed",
      ),
    ).toBe("quick");
  });
  it.each([
    { grind_size: "Medium" },
    { grinder_rkey: "grinder" },
    { method: "Old method" },
    { coffee_amount: 18 },
    { water_amount: 300 },
    { temperature: 93.5 },
    { time_seconds: 86400 },
    {
      pours: [
        { pour_number: 1, water_amount: 50, time_seconds: 30, created_at: "" },
      ],
    },
    {
      espresso_params: {
        yield_weight: 36,
        pressure: 0,
        pre_infusion_seconds: 0,
      },
    },
    {
      pourover_params: {
        bloom_water: 0,
        bloom_seconds: 0,
        drawdown_seconds: 0,
        bypass_water: 0,
        filter: "paper",
      },
    },
  ])("opens legacy preparation values in Detailed: %j", (fields) => {
    expect(initialBrewFormMode(true, record(fields), "quick")).toBe("detailed");
  });
  it("does not treat empty optional objects as preparation", () => {
    expect(
      hasPreparationValues(
        record({
          espresso_params: {
            yield_weight: 0,
            pressure: 0,
            pre_infusion_seconds: 0,
          },
          pours: [],
        }),
      ),
    ).toBe(false);
  });
  it.each([
    ["Pour Over", "pourover"],
    ["pour-over", "pourover"],
    ["dripper", "pourover"],
    ["Lever Espresso Machine", "espresso"],
    ["French Press", "immersion"],
    ["Clever Dripper", "immersion"],
    ["Moka Pot", "mokapot"],
    ["Bialetti", "mokapot"],
    ["Cold Brew", "coldbrew"],
    ["Cold Drip", "coldbrew"],
    ["cupping", "cupping"],
    ["unknown", "other"],
    ["", "other"],
  ])("normalizes %s to %s", (raw, expected) =>
    expect(normalizeBrewerCategory(raw)).toBe(expected),
  );
  it.each([
    ["espresso", true, false, true, "Shot time (s)"],
    ["pourover", true, true, true, "Brew time (s)"],
    ["immersion", false, true, true, "Steep time (s)"],
    ["mokapot", false, true, false, "Brew time (s)"],
    ["coldbrew", false, true, true, "Steep time (s)"],
    ["cupping", false, true, true, "Steep time (s)"],
    ["unknown", false, true, true, "Brew time (s)"],
  ])(
    "uses the %s matrix",
    (raw, mainGrinder, mainWater, mainTime, timeLabel) => {
      expect(preparationLayout(String(raw))).toMatchObject({
        mainGrinder,
        mainWater,
        mainTime,
        timeLabel,
      });
    },
  );
});
