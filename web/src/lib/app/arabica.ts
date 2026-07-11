import type { AppDefinition } from "./definitions";

export const arabica: AppDefinition = {
	name: "arabica",
	displayName: "Arabica",
	tagline: "Your brew, your data",
	libraryPath: "/my-coffee",
	sessionNoun: "brew",
	sessionAction: "Log Brew",
	commentCollection: "social.arabica.alpha.comment",
	entityRoutes: {
		bean: "beans",
		roaster: "roasters",
		grinder: "grinders",
		brewer: "brewers",
		recipe: "recipes",
		brew: "brews",
	},
	feedRecordTypes: ["bean", "roaster", "grinder", "brewer", "recipe", "brew"],
};
