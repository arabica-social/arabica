import type { AppDefinition } from "./definitions";

export const oolong: AppDefinition = {
	name: "oolong",
	displayName: "Oolong",
	tagline: "Your tea, your data",
	libraryPath: "/my-tea",
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
};
