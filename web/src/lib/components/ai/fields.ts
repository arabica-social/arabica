import type { DraftKind, FieldPath } from "#lib/ai/types.js";
export type EditorField = {
  key: FieldPath;
  label: string;
  type?: "number" | "date" | "url" | "textarea";
  required?: boolean;
  options?: string[];
  library?: string;
  step?: string;
  min?: number;
  max?: number;
};
const name: EditorField = { key: "name", label: "Name", required: true };
export const brewerTypes = [
  "pourover",
  "espresso",
  "immersion",
  "mokapot",
  "coldbrew",
  "cupping",
  "other",
];
export const fields: Record<DraftKind, EditorField[]> = {
  brew: [
    { key: "bean_rkey", label: "Bean", required: true, library: "beans" },
    { key: "brewer_rkey", label: "Brewer", required: true, library: "brewers" },
    { key: "recipe_rkey", label: "Recipe", library: "recipes" },
    {
      key: "coffee_amount",
      label: "Coffee (g)",
      type: "number",
      min: 1,
      max: 1000,
    },
    {
      key: "water_amount",
      label: "Water (g)",
      type: "number",
      min: 1,
      max: 10000,
    },
    {
      key: "temperature",
      label: "Temperature (°F/°C)",
      type: "number",
      step: "any",
      min: 0.01,
      max: 212,
    },
    { key: "time_seconds", label: "Brew time (s)", type: "number", min: 1 },
    { key: "grinder_rkey", label: "Grinder", library: "grinders" },
    { key: "grind_size", label: "Grind setting" },
    { key: "tasting_notes", label: "Tasting notes", type: "textarea" },
    { key: "rating", label: "Rating (1–10)", type: "number", min: 1, max: 10 },
    {
      key: "espresso_params.yield_weight",
      label: "Espresso yield (g)",
      type: "number",
      step: "any",
      min: 0,
    },
    {
      key: "espresso_params.pressure",
      label: "Pressure (bar)",
      type: "number",
      step: "any",
      min: 0,
    },
    {
      key: "espresso_params.pre_infusion_seconds",
      label: "Pre-infusion (s)",
      type: "number",
      min: 0,
    },
    {
      key: "pourover_params.bloom_water",
      label: "Bloom water (g)",
      type: "number",
      min: 0,
    },
    {
      key: "pourover_params.bloom_seconds",
      label: "Bloom time (s)",
      type: "number",
      min: 0,
    },
    {
      key: "pourover_params.drawdown_seconds",
      label: "Drawdown (s)",
      type: "number",
      min: 0,
    },
    {
      key: "pourover_params.bypass_water",
      label: "Bypass water (g)",
      type: "number",
      min: 0,
    },
    { key: "pourover_params.filter", label: "Filter" },
  ],
  bean: [
    name,
    { key: "origin", label: "Origin", required: true },
    { key: "roaster_rkey", label: "Roaster", library: "roasters" },
    { key: "variety", label: "Variety" },
    {
      key: "roast_level",
      label: "Roast level",
      options: [
        "Ultra-Light",
        "Light",
        "Medium-Light",
        "Medium",
        "Medium-Dark",
        "Dark",
      ],
    },
    { key: "roast_date", label: "Roast date", type: "date" },
    { key: "process", label: "Process" },
    { key: "description", label: "Description", type: "textarea" },
    { key: "notes", label: "Notes", type: "textarea" },
    { key: "link", label: "Link", type: "url" },
    { key: "rating", label: "Rating (1–10)", type: "number", min: 1, max: 10 },
  ],
  grinder: [
    name,
    {
      key: "grinder_type",
      label: "Type",
      options: ["Hand", "Electric", "Portable Electric"],
    },
    { key: "burr_type", label: "Burr type", options: ["Conical", "Flat"] },
    { key: "notes", label: "Notes", type: "textarea" },
    { key: "link", label: "Link", type: "url" },
  ],
  brewer: [
    name,
    { key: "brewer_type", label: "Type", options: brewerTypes },
    { key: "description", label: "Description", type: "textarea" },
    { key: "link", label: "Link", type: "url" },
  ],
};

// Examples are placeholders, never default values or saved suggestions.
export const brewGhostText: Partial<Record<FieldPath, string>> = {
  coffee_amount: "e.g. 18",
  water_amount: "e.g. 250",
  temperature: "e.g. 93.5",
  time_seconds: "e.g. 180",
  grind_size: "e.g. 18, Medium, 3.5, Fine",
  tasting_notes: "Describe the flavors, aroma, and your thoughts...",
  rating: "e.g. 8",
  "espresso_params.yield_weight": "e.g. 36",
  "espresso_params.pressure": "e.g. 9",
  "espresso_params.pre_infusion_seconds": "e.g. 5",
  "pourover_params.bloom_water": "e.g. 50",
  "pourover_params.bloom_seconds": "e.g. 45",
  "pourover_params.drawdown_seconds": "e.g. 30",
  "pourover_params.bypass_water": "e.g. 100",
  "pourover_params.filter": "e.g. paper, metal, cloth",
};
