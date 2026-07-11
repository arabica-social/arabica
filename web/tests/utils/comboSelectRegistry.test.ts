import { describe, expect, it } from "vitest";
import { comboSelectEntities } from "../../src/lib/comboSelectRegistry";
import type { EntityConfig, Suggestion } from "../../src/lib/comboSelectRegistry";

const ENTITY_NAMES = [
	"bean",
	"brewer",
	"grinder",
	"recipe",
	"roaster",
	"cafe",
	"tea",
	"oolongBrewer",
	"oolongVessel",
	"oolongInfuser",
	"oolongRecipe",
	"vendor",
] as const;

function config(name: string): EntityConfig {
	const cfg = comboSelectEntities[name];
	if (!cfg) throw new Error(`missing entity config: ${name}`);
	return cfg;
}

describe("comboSelectEntities", () => {
	it("exports a config for every expected entity", () => {
		for (const name of ENTITY_NAMES) {
			expect(comboSelectEntities[name]).toBeDefined();
		}
		expect(Object.keys(comboSelectEntities).sort()).toEqual([...ENTITY_NAMES].sort());
	});

	it("every config defines all three behaviors", () => {
		for (const name of ENTITY_NAMES) {
			const cfg = config(name);
			expect(typeof cfg.formatLabel).toBe("function");
			expect(typeof cfg.formatCreateData).toBe("function");
			expect(Array.isArray(cfg.extraFields)).toBe(true);
		}
	});
});

describe("formatLabel", () => {
	it.each(ENTITY_NAMES)("returns empty string for an empty record (%s)", (name) => {
		expect(config(name).formatLabel!({})).toBe("");
	});

	describe("bean", () => {
		const formatLabel = config("bean").formatLabel!;
		it("formats name + origin + roast_level", () => {
			expect(
				formatLabel({ name: "Heart", origin: "Ethiopia", roast_level: "Light" }),
			).toBe("Heart (Ethiopia - Light)");
		});
		it("formats name + origin only", () => {
			expect(formatLabel({ name: "Heart", origin: "Ethiopia" })).toBe("Heart (Ethiopia)");
		});
		it("returns name only when origin is absent", () => {
			expect(formatLabel({ name: "Heart", roast_level: "Light" })).toBe("Heart");
		});
		it("returns name only when just the name is present", () => {
			expect(formatLabel({ name: "Heart" })).toBe("Heart");
		});
		it("accepts capitalized keys", () => {
			expect(
				formatLabel({ Name: "Heart", Origin: "Ethiopia", RoastLevel: "Light" }),
			).toBe("Heart (Ethiopia - Light)");
			expect(formatLabel({ Name: "Heart", Origin: "Ethiopia" })).toBe("Heart (Ethiopia)");
			expect(formatLabel({ Name: "Heart" })).toBe("Heart");
		});
	});

	describe("recipe", () => {
		const formatLabel = config("recipe").formatLabel!;
		it("formats name + brewer_type", () => {
			expect(formatLabel({ name: "V60 recipe", brewer_type: "pourover" })).toBe(
				"V60 recipe (pourover)",
			);
		});
		it("reads brewerType from a nested fields object", () => {
			expect(formatLabel({ name: "V60 recipe", fields: { brewerType: "espresso" } })).toBe(
				"V60 recipe (espresso)",
			);
		});
		it("accepts capitalized BrewerType", () => {
			expect(formatLabel({ name: "V60 recipe", BrewerType: "immersion" })).toBe(
				"V60 recipe (immersion)",
			);
		});
		it("returns name only when no brewer type is set", () => {
			expect(formatLabel({ name: "V60 recipe" })).toBe("V60 recipe");
		});
	});

	describe("cafe / vendor", () => {
		it.each(["cafe", "vendor"] as const)("formats name + location (%s)", (name) => {
			expect(config(name).formatLabel!({ name: "Sey", location: "Brooklyn, NY" })).toBe(
				"Sey (Brooklyn, NY)",
			);
		});
		it.each(["cafe", "vendor"] as const)("accepts capitalized Location (%s)", (name) => {
			expect(config(name).formatLabel!({ name: "Sey", Location: "Brooklyn, NY" })).toBe(
				"Sey (Brooklyn, NY)",
			);
		});
		it.each(["cafe", "vendor"] as const)("returns name only without location (%s)", (name) => {
			expect(config(name).formatLabel!({ name: "Sey" })).toBe("Sey");
		});
	});

	describe("roaster", () => {
		const formatLabel = config("roaster").formatLabel!;
		it("returns the name only", () => {
			expect(formatLabel({ name: "Heart" })).toBe("Heart");
		});
		it("ignores location", () => {
			expect(formatLabel({ name: "Heart", location: "Portland, OR" })).toBe("Heart");
		});
		it("accepts capitalized Name", () => {
			expect(formatLabel({ Name: "Heart" })).toBe("Heart");
		});
	});

	describe("brewer / grinder", () => {
		it.each(["brewer", "grinder"] as const)("returns the name only (%s)", (name) => {
			expect(config(name).formatLabel!({ name: "V60" })).toBe("V60");
		});
		it.each(["brewer", "grinder"] as const)("accepts capitalized Name (%s)", (name) => {
			expect(config(name).formatLabel!({ Name: "V60" })).toBe("V60");
		});
	});

	describe("tea", () => {
		const formatLabel = config("tea").formatLabel!;
		it("formats name + category + origin", () => {
			expect(formatLabel({ name: "Dong Ding", category: "oolong", origin: "Taiwan" })).toBe(
				"Dong Ding (oolong · Taiwan)",
			);
		});
		it("formats name + category only", () => {
			expect(formatLabel({ name: "Dong Ding", category: "oolong" })).toBe(
				"Dong Ding (oolong)",
			);
		});
		it("formats name + origin only", () => {
			expect(formatLabel({ name: "Dong Ding", origin: "Taiwan" })).toBe("Dong Ding (Taiwan)");
		});
		it("returns name only", () => {
			expect(formatLabel({ name: "Dong Ding" })).toBe("Dong Ding");
		});
		it("accepts capitalized keys", () => {
			expect(
				formatLabel({ Name: "Dong Ding", Category: "oolong", Origin: "Taiwan" }),
			).toBe("Dong Ding (oolong · Taiwan)");
		});
	});

	describe("oolong brewer / vessel / infuser / recipe", () => {
		it.each(["oolongBrewer", "oolongVessel", "oolongInfuser", "oolongRecipe"] as const)(
			"formats name + style (%s)",
			(name) => {
				expect(config(name).formatLabel!({ name: "Gaiwan", style: "gaiwan" })).toBe(
					"Gaiwan (gaiwan)",
				);
			},
		);
		it.each(["oolongBrewer", "oolongVessel", "oolongInfuser", "oolongRecipe"] as const)(
			"accepts capitalized Style (%s)",
			(name) => {
				expect(config(name).formatLabel!({ name: "Gaiwan", Style: "gaiwan" })).toBe(
					"Gaiwan (gaiwan)",
				);
			},
		);
		it.each(["oolongBrewer", "oolongVessel", "oolongInfuser", "oolongRecipe"] as const)(
			"returns name only without style (%s)",
			(name) => {
				expect(config(name).formatLabel!({ name: "Gaiwan" })).toBe("Gaiwan");
			},
		);
	});
});

describe("formatCreateData", () => {
	const NAME = "Heart";

	it.each(ENTITY_NAMES)("returns { name } with no suggestion (%s)", (name) => {
		expect(config(name).formatCreateData!(NAME)).toEqual({ name: NAME });
	});

	it.each(ENTITY_NAMES)("returns { name } when suggestion has empty fields (%s)", (name) => {
		const suggestion: Suggestion = { name: NAME, fields: {} };
		expect(config(name).formatCreateData!(NAME, suggestion)).toEqual({ name: NAME });
	});

	describe("bean", () => {
		const formatCreateData = config("bean").formatCreateData!;
		it("maps suggestion fields to the expected output keys", () => {
			const suggestion: Suggestion = {
				fields: {
					origin: "Ethiopia",
					roastLevel: "Light",
					process: "Washed",
					link: "https://heart.com",
					roasterName: "Heart",
				},
			};
			expect(formatCreateData(NAME, suggestion)).toEqual({
				name: NAME,
				origin: "Ethiopia",
				roast_level: "Light",
				process: "Washed",
				link: "https://heart.com",
				_source_roaster_name: "Heart",
			});
		});
		it("only sets keys that are present in the suggestion", () => {
			expect(formatCreateData(NAME, { fields: { origin: "Ethiopia" } })).toEqual({
				name: NAME,
				origin: "Ethiopia",
			});
		});
		it("ignores falsy suggestion field values", () => {
			expect(
				formatCreateData(NAME, {
					fields: { origin: "", roastLevel: undefined, process: null },
				}),
			).toEqual({ name: NAME });
		});
	});

	describe("brewer", () => {
		const formatCreateData = config("brewer").formatCreateData!;
		it("maps brewerType and link", () => {
			expect(
				formatCreateData(NAME, {
					fields: { brewerType: "pourover", link: "https://hario.com" },
				}),
			).toEqual({ name: NAME, brewer_type: "pourover", link: "https://hario.com" });
		});
		it("returns { name } when neither field is present", () => {
			expect(formatCreateData(NAME, { fields: { brewerType: "" } })).toEqual({ name: NAME });
		});
	});

	describe("grinder", () => {
		const formatCreateData = config("grinder").formatCreateData!;
		it("maps grinderType, burrType, and link", () => {
			expect(
				formatCreateData(NAME, {
					fields: { grinderType: "Hand", burrType: "Conical", link: "https://1zpresso.com" },
				}),
			).toEqual({
				name: NAME,
				grinder_type: "Hand",
				burr_type: "Conical",
				link: "https://1zpresso.com",
			});
		});
		it("only sets present keys", () => {
			expect(formatCreateData(NAME, { fields: { burrType: "Flat" } })).toEqual({
				name: NAME,
				burr_type: "Flat",
			});
		});
	});

	describe("recipe", () => {
		const formatCreateData = config("recipe").formatCreateData!;
		it("ignores suggestion fields and returns { name }", () => {
			expect(
				formatCreateData(NAME, { fields: { brewerType: "pourover", origin: "x" } }),
			).toEqual({ name: NAME });
		});
	});

	describe("roaster / cafe / vendor", () => {
		it.each(["roaster", "cafe", "vendor"] as const)(
			"maps location and website (%s)",
			(name) => {
				expect(
					config(name).formatCreateData!(NAME, {
						fields: { location: "Portland, OR", website: "https://heart.com" },
					}),
				).toEqual({ name: NAME, location: "Portland, OR", website: "https://heart.com" });
			},
		);
		it.each(["roaster", "cafe", "vendor"] as const)(
			"only sets present keys (%s)",
			(name) => {
				expect(config(name).formatCreateData!(NAME, { fields: { location: "Portland, OR" } })).toEqual(
					{ name: NAME, location: "Portland, OR" },
				);
			},
		);
	});

	describe("tea", () => {
		const formatCreateData = config("tea").formatCreateData!;
		it("maps category, subStyle, origin, and cultivar", () => {
			expect(
				formatCreateData(NAME, {
					fields: {
						category: "oolong",
						subStyle: "Wuyi",
						origin: "Taiwan",
						cultivar: "Qing Xin",
					},
				}),
			).toEqual({
				name: NAME,
				category: "oolong",
				sub_style: "Wuyi",
				origin: "Taiwan",
				cultivar: "Qing Xin",
			});
		});
	});

	describe("oolongBrewer", () => {
		const formatCreateData = config("oolongBrewer").formatCreateData!;
		it("maps style, material, and link", () => {
			expect(
				formatCreateData(NAME, {
					fields: { style: "gaiwan", material: "porcelain", link: "https://x.com" },
				}),
			).toEqual({
				name: NAME,
				style: "gaiwan",
				material: "porcelain",
				link: "https://x.com",
			});
		});
	});

	describe("oolongVessel", () => {
		const formatCreateData = config("oolongVessel").formatCreateData!;
		it("maps style and material", () => {
			expect(
				formatCreateData(NAME, { fields: { style: "teapot", material: "clay" } }),
			).toEqual({ name: NAME, style: "teapot", material: "clay" });
		});
		it("does not map link", () => {
			expect(
				formatCreateData(NAME, { fields: { style: "teapot", link: "https://x.com" } }),
			).toEqual({ name: NAME, style: "teapot" });
		});
	});

	describe("oolongInfuser", () => {
		const formatCreateData = config("oolongInfuser").formatCreateData!;
		it("maps style and link", () => {
			expect(
				formatCreateData(NAME, { fields: { style: "basket", link: "https://x.com" } }),
			).toEqual({ name: NAME, style: "basket", link: "https://x.com" });
		});
		it("does not map material", () => {
			expect(
				formatCreateData(NAME, { fields: { style: "basket", material: "steel" } }),
			).toEqual({ name: NAME, style: "basket" });
		});
	});

	describe("oolongRecipe", () => {
		const formatCreateData = config("oolongRecipe").formatCreateData!;
		it("ignores suggestion fields and returns { name }", () => {
			expect(formatCreateData(NAME, { fields: { style: "gaiwan" } })).toEqual({ name: NAME });
		});
	});
});

describe("extraFields", () => {
	it.each([
		["bean", 4, ["origin", "roast_level", "process", "link"]],
		["brewer", 2, ["brewer_type", "link"]],
		["grinder", 3, ["grinder_type", "burr_type", "link"]],
		["recipe", 0, []],
		["roaster", 2, ["location", "website"]],
		["cafe", 2, ["location", "website"]],
		["tea", 3, ["category", "origin", "cultivar"]],
		["oolongBrewer", 3, ["style", "material", "link"]],
		["oolongVessel", 2, ["style", "material"]],
		["oolongInfuser", 2, ["style", "link"]],
		["oolongRecipe", 0, []],
		["vendor", 2, ["location", "website"]],
	] as const)("has the expected length and field names (%s)", (name, length, fieldNames) => {
		const fields = config(name).extraFields!;
		expect(fields).toHaveLength(length);
		expect(fields.map((f) => f.name)).toEqual(fieldNames);
	});

	it("every extra field has a name, label, and type", () => {
		for (const name of ENTITY_NAMES) {
			for (const field of config(name).extraFields!) {
				expect(typeof field.name).toBe("string");
				expect(field.name.length).toBeGreaterThan(0);
				expect(typeof field.label).toBe("string");
				expect(field.label.length).toBeGreaterThan(0);
				expect(typeof field.type).toBe("string");
				expect(field.type.length).toBeGreaterThan(0);
			}
		}
	});

	it.each([
		["bean", "roast_level", ["Ultra-Light", "Light", "Medium-Light", "Medium", "Medium-Dark", "Dark"]],
		["brewer", "brewer_type", ["pourover", "espresso", "immersion", "mokapot", "coldbrew", "cupping", "other"]],
		["grinder", "grinder_type", ["Hand", "Electric", "Portable Electric"]],
		["grinder", "burr_type", ["Conical", "Flat"]],
		["tea", "category", ["green", "white", "yellow", "oolong", "black", "puerh-sheng", "puerh-shou", "herbal", "blend", "other"]],
		["oolongBrewer", "style", ["gaiwan", "yixing", "kyusu", "teapot", "glass", "french-press", "tetsubin", "other"]],
		["oolongVessel", "style", ["teapot", "mug", "jar", "matcha-bowl", "other"]],
		["oolongInfuser", "style", ["basket", "ball", "sock", "other"]],
	] as const)("select field %s.%s has the expected options", (entity, fieldName, options) => {
		const field = config(entity).extraFields!.find((f) => f.name === fieldName);
		expect(field, `expected ${entity} to define ${fieldName}`).toBeDefined();
		expect(field!.type).toBe("select");
		expect(field!.options).toEqual(options);
	});

	it("text and url fields do not define options", () => {
		for (const name of ENTITY_NAMES) {
			for (const field of config(name).extraFields!) {
				if (field.type !== "select") {
					expect(field.options).toBeUndefined();
				}
			}
		}
	});
});
