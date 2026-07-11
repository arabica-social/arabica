import { describe, expect, it } from "vitest";
import {
	definitionFor,
	entityRouteForCollection,
} from "../../src/lib/app/definitions";

describe("frontend app definitions", () => {
	it("keeps product branding and library destinations separate", () => {
		expect(definitionFor("arabica").libraryPath).toBe("/my-coffee");
		expect(definitionFor("oolong").libraryPath).toBe("/my-tea");
		expect(definitionFor("oolong").commentCollection).toBe(
			"social.oolong.alpha.comment",
		);
	});

	it("maps collection tails through the active app only", () => {
		expect(
			entityRouteForCollection("arabica", "social.arabica.alpha.roaster"),
		).toBe("roasters");
		expect(
			entityRouteForCollection("oolong", "social.oolong.alpha.vendor"),
		).toBe("vendors");
		expect(
			entityRouteForCollection("oolong", "social.arabica.alpha.roaster"),
		).toBe("");
	});
});
