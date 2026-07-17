import { describe, expect, it } from "vitest";
import {
	definitionFor,
	entityRouteForCollection,
} from "../../src/lib/app/definitions";

describe("frontend app definitions", () => {
	it("exposes the Arabica product branding and library destination", () => {
		expect(definitionFor("arabica").libraryPath).toBe("/my-coffee");
		expect(definitionFor("arabica").commentCollection).toBe(
			"social.arabica.alpha.comment",
		);
	});

	it("maps collection tails through the active app", () => {
		expect(
			entityRouteForCollection("arabica", "social.arabica.alpha.roaster"),
		).toBe("roasters");
		expect(
			entityRouteForCollection("arabica", "social.arabica.alpha.bean"),
		).toBe("beans");
	});
});
