import type { AppDefinition } from "./definitions";

export const oolong: AppDefinition = {
	name: "oolong",
	displayName: "Oolong",
	tagline: "Your tea, your data",
	libraryPath: "/my-tea",
	libraryLabel: "My Tea",
	sessionNoun: "steep",
	sessionAction: "Log Steep",
	commentCollection: "social.oolong.alpha.comment",
	entityRoutes: {
		tea: "teas",
		vendor: "vendors",
		vessel: "vessels",
		infuser: "infusers",
		brew: "brews",
	},
	feedRecordTypes: ["tea", "vendor", "vessel", "infuser", "brew"],
	heroHeading: "Your tea journey, documented.",
	heroDescription:
		"Log every steep, track your teaware and vendors, and share your tea story with the community.",
	metaDescription:
		"Log every steep, track your teaware and vendors, and share your tea story with the community. Built on AT Protocol — you own your data.",
	readinessEntityTypes: ["tea", "vessel", "vendor"],
	readinessNudge: "Add a tea, vessel, and vendor to start logging steeps.",
	aboutHeading: "About Oolong",
	aboutBody:
		"Oolong is a tea tracking app built on AT Protocol. Your steep logs, teas, and teaware are stored in your own Personal Data Server, giving you full ownership and portability.",
};
