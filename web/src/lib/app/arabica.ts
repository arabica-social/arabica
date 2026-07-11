import type { AppDefinition } from "./definitions";

export const arabica: AppDefinition = {
	name: "arabica",
	displayName: "Arabica",
	tagline: "Your brew, your data",
	libraryPath: "/my-coffee",
	libraryLabel: "My Coffee",
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
	heroHeading: "Your coffee journey, documented.",
	heroDescription:
		"Log every brew, track your beans and equipment, and share your coffee story with the community.",
	metaDescription:
		"Log every brew, track your beans and equipment, and share your coffee story with the community. Built on AT Protocol — you own your data.",
	readinessEntityTypes: ["bean", "brewer", "roaster"],
	readinessNudge: "Add a bean, brewer, and roaster to start logging brews.",
	aboutHeading: "About Arabica",
	aboutBody:
		"Arabica is a coffee brew tracking app built on AT Protocol. Your brewing data is stored in your own Personal Data Server, giving you full ownership and portability.",
};
