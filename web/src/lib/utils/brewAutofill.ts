// Pure suggestion logic for Smart Autofill (see
// docs/plans/2026-07-24-brew-smart-autofill.md). Derives per-field values
// from the authenticated user's own brew history, recency-biased toward the
// most recent habit. Framework-free so it is unit-testable in isolation.

import type { Brew } from "../types/entity_view";

/** Field ids the form can autofill. Never-suggested fields (tasting_notes,
 *  rating, pours) are intentionally absent. */
export type AutofillField =
  | "grinder_rkey"
  | "brewer_rkey"
  | "method"
  | "grind_size"
  | "temperature"
  | "time_seconds"
  | "water_amount"
  | "coffee_amount"
  | "pourover_filter"
  | "pourover_bloom_water"
  | "pourover_bloom_seconds"
  | "pourover_drawdown_seconds"
  | "pourover_bypass_water"
  | "espresso_yield_weight"
  | "espresso_pressure"
  | "espresso_pre_infusion_seconds";

export type AutofillSuggestion = {
	/** Normalized form-usable value. */
	value: string;
	/** Display label for ref fields (grinder/brewer). */
	label?: string;
	/** Record rkey for ref fields. */
	rkey?: string;
	/** Brews in the matching set carrying this value. */
	matches: number;
	/** Total matching brews. */
	total: number;
};

export type AutofillResult = {
	basis: "bean+recipe" | "bean";
	totalBrews: number;
	fields: Partial<Record<AutofillField, AutofillSuggestion>>;
};

export type AutofillKey = {
	beanRKey: string;
	recipeRKey?: string;
	recipeOwnerDID?: string;
};

/** Half-life of the recency decay, in brews (Decision 2, H = 5). */
const HALF_LIFE = 5;

/** Weight for the brew at recency rank `rank` (newest = 0). */
function weightAt(rank: number): number {
	return 0.5 ** (rank / HALF_LIFE);
}

type Extracted = { value: string; label?: string } | null;

function text(v: string | undefined | null): Extracted {
	return v ? { value: v } : null;
}

/** A numeric field counts as empty when it is absent or 0 (unset sentinel). */
function numberAsText(v: number | undefined): Extracted {
	return v ? { value: String(v) } : null;
}

type FieldSpec = {
	extract: (brew: Brew) => Extracted;
	ref?: boolean;
};

const FIELDS: [AutofillField, FieldSpec][] = [
	[
		"grinder_rkey",
		{
			ref: true,
			extract: (b) => (b.grinder_rkey ? { value: b.grinder_rkey, label: b.grinder_obj?.name } : null),
		},
	],
	[
		"brewer_rkey",
		{
			ref: true,
			extract: (b) => (b.brewer_rkey ? { value: b.brewer_rkey, label: b.brewer_obj?.name } : null),
		},
	],
	["method", { extract: (b) => text(b.method) }],
	["grind_size", { extract: (b) => text(b.grind_size) }],
	["temperature", { extract: (b) => numberAsText(b.temperature) }],
	["time_seconds", { extract: (b) => numberAsText(b.time_seconds) }],
	["water_amount", { extract: (b) => numberAsText(b.water_amount) }],
	["coffee_amount", { extract: (b) => numberAsText(b.coffee_amount) }],
	["pourover_filter", { extract: (b) => text(b.pourover_params?.filter) }],
	["pourover_bloom_water", { extract: (b) => numberAsText(b.pourover_params?.bloom_water) }],
	["pourover_bloom_seconds", { extract: (b) => numberAsText(b.pourover_params?.bloom_seconds) }],
	["pourover_drawdown_seconds", { extract: (b) => numberAsText(b.pourover_params?.drawdown_seconds) }],
	["pourover_bypass_water", { extract: (b) => numberAsText(b.pourover_params?.bypass_water) }],
	["espresso_yield_weight", { extract: (b) => numberAsText(b.espresso_params?.yield_weight) }],
	["espresso_pressure", { extract: (b) => numberAsText(b.espresso_params?.pressure) }],
	["espresso_pre_infusion_seconds", { extract: (b) => numberAsText(b.espresso_params?.pre_infusion_seconds) }],
];

/** Picks the winning suggestion for one field. Expects `brews` sorted newest
 *  first. Returns undefined when the field has no non-empty value across the
 *  matching set. */
function pickSuggestion(
	brews: Brew[],
	spec: FieldSpec,
): AutofillSuggestion | undefined {
	const seen = new Map<string, { weight: number; matches: number; newestIndex: number; label?: string }>();
	brews.forEach((brew, index) => {
		const got = spec.extract(brew);
		if (!got) return;
		const existing = seen.get(got.value);
		if (existing) {
			existing.weight += weightAt(index);
			existing.matches += 1;
			if (index < existing.newestIndex) {
				existing.newestIndex = index;
				existing.label = got.label ?? existing.label;
			}
		} else {
			seen.set(got.value, {
				weight: weightAt(index),
				matches: 1,
				newestIndex: index,
				label: got.label,
			});
		}
	});

	let bestValue: string | undefined;
	let bestScore: { weight: number; index: number } | undefined;
	// Effectively-tied weights (within an epsilon) break toward the value
	// appearing in the newest brew (Decision 2). Competing values in real
	// history usually differ by far more than this, so the tie-break only
	// decides genuinely close candidates.
	const EPSILON = 1e-3;
	for (const [value, entry] of seen) {
		const tied = bestScore !== undefined && Math.abs(entry.weight - bestScore.weight) <= EPSILON;
		if (
			!bestScore ||
			entry.weight > bestScore.weight + EPSILON ||
			(tied && entry.newestIndex < bestScore.index)
		) {
			bestValue = value;
			bestScore = { weight: entry.weight, index: entry.newestIndex };
		}
	}
	if (bestValue === undefined || !bestScore) return undefined;

	const winner = seen.get(bestValue)!;
	const suggestion: AutofillSuggestion = {
		value: bestValue,
		matches: winner.matches,
		total: brews.length,
	};
	if (spec.ref) {
		suggestion.rkey = bestValue;
		if (winner.label) suggestion.label = winner.label;
	}
	return suggestion;
}

/**
 * Computes autofill suggestions for a new brew from the user's brew history.
 * Matching tiers (Decision 4): with a recipe selected, first try the
 * (bean, recipe + owner DID) pair; fall back to bean-only if that yields
 * fewer than 2 brews. Without a recipe, use bean-only. Returns null when the
 * final matching set has fewer than 2 brews.
 */
export function computeAutofill(brews: Brew[], key: AutofillKey): AutofillResult | null {
	const newestFirst = [...brews].sort((a, b) => b.created_at.localeCompare(a.created_at));
	const beanOnly = newestFirst.filter((b) => b.bean_rkey === key.beanRKey);

	let basis: "bean+recipe" | "bean";
	let matching: Brew[];
	if (key.recipeRKey) {
		const pair = newestFirst.filter(
			(b) =>
				b.bean_rkey === key.beanRKey &&
				b.recipe_rkey === key.recipeRKey &&
				(key.recipeOwnerDID === undefined || b.recipe_owner_did === key.recipeOwnerDID),
		);
		if (pair.length >= 2) {
			basis = "bean+recipe";
			matching = pair;
		} else {
			basis = "bean";
			matching = beanOnly;
		}
	} else {
		basis = "bean";
		matching = beanOnly;
	}

	if (matching.length < 2) return null;

	const fields: Partial<Record<AutofillField, AutofillSuggestion>> = {};
	for (const [field, spec] of FIELDS) {
		const suggestion = pickSuggestion(matching, spec);
		if (suggestion) fields[field] = suggestion;
	}
	return { basis, totalBrews: matching.length, fields };
}
