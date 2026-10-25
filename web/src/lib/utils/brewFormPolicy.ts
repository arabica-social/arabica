import type { Brew } from "../types/entity_view";

export type BrewFormMode = "quick" | "detailed";
export type BrewerCategory =
  | "espresso"
  | "pourover"
  | "immersion"
  | "mokapot"
  | "coldbrew"
  | "cupping"
  | "other";

/** Coffee-specific presentation policy, not a persisted-record constraint. */
export function normalizeBrewerCategory(raw: string = ""): BrewerCategory {
  const key = raw
    .toLowerCase()
    .trim()
    .replace(/[\s_-]+/g, "");
  if (
    [
      "espresso",
      "espressomachine",
      "leverespresso",
      "leverespressomachine",
    ].includes(key)
  )
    return "espresso";
  if (["pourover", "dripper"].includes(key)) return "pourover";
  if (
    [
      "immersion",
      "frenchpress",
      "aeropress",
      "siphon",
      "clever",
      "cleverdripper",
    ].includes(key)
  )
    return "immersion";
  if (["mokapot", "moka", "bialetti"].includes(key)) return "mokapot";
  if (key === "coldbrew" || key === "colddrip") return "coldbrew";
  if (key === "cupping") return "cupping";
  return "other";
}

export function hasPreparationValues(brew: Brew | null): boolean {
  if (!brew) return false;
  return !!(
    brew.method ||
    brew.grinder_rkey ||
    brew.grind_size ||
    brew.coffee_amount ||
    brew.water_amount ||
    brew.temperature ||
    brew.time_seconds ||
    brew.pours?.some((p) => p.water_amount || p.time_seconds) ||
    (brew.espresso_params &&
      Object.values(brew.espresso_params).some(Boolean)) ||
    (brew.pourover_params && Object.values(brew.pourover_params).some(Boolean))
  );
}

export function initialBrewFormMode(
  isEdit: boolean,
  brew: Brew | null,
  preference?: unknown,
): BrewFormMode {
  if (isEdit) return hasPreparationValues(brew) ? "detailed" : "quick";
  return preference === "detailed" ? "detailed" : "quick";
}

export function preparationLayout(raw: string) {
  const category = normalizeBrewerCategory(raw);
  return {
    category,
    mainGrinder: category === "espresso" || category === "pourover",
    mainWater: category !== "espresso",
    mainTime: category !== "mokapot",
    timeLabel:
      category === "espresso"
        ? "Shot time (s)"
        : ["immersion", "coldbrew", "cupping"].includes(category)
          ? "Steep time (s)"
          : "Brew time (s)",
  };
}
