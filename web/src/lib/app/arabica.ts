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
  heroHeading: "A coffee journal",
  heroDescription:
    "Keep your brew notes and recipes, and see what other people are making.",
  metaDescription:
    "Record your brews, beans, recipes, and equipment. Built on AT Protocol, so your records stay in your own data store.",
  readinessEntityTypes: ["bean", "brewer", "roaster"],
  readinessNudge: "Add a bean, brewer, and roaster to start logging brews.",
  aboutHeading: "About Arabica",
  aboutBody:
    "Arabica is a coffee brew tracking app built on AT Protocol. Your brewing data is stored in your own Personal Data Server, giving you full ownership and portability.",
  feedbackUrl:
    "https://userinput.app/s/did:plc:chqc2ockzmyvlrasfb66x64a/3mrgh3b4f722p",
};
