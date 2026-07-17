import type { AppName } from "$lib/stores/session";
import { arabica } from "./arabica";

export type EntityRouteDefinition = Record<string, string>;

export type AppDefinition = {
	name: AppName;
	displayName: string;
	tagline: string;
	libraryPath: string;
	libraryLabel: string;
	sessionNoun: string;
	sessionAction: string;
	commentCollection: string;
	entityRoutes: EntityRouteDefinition;
	feedRecordTypes: string[];
	// Home page content
	heroHeading: string;
	heroDescription: string;
	metaDescription: string;
	// Readiness: the entity types a user must have before they can log a session
	readinessEntityTypes: string[];
	readinessNudge: string;
	aboutHeading: string;
	aboutBody: string;
};

export const appDefinitions: Record<AppName, AppDefinition> = { arabica };

export function definitionFor(app: AppName): AppDefinition {
	return appDefinitions[app];
}

// Collection names have the same terminal record name as Go's app entity
// descriptors. Keeping that lookup here prevents shared views from importing
// either product's NSIDs or route table.
export function entityRouteForCollection(app: AppName, collection: string): string {
	const tail = collection.includes(".")
		? collection.slice(collection.lastIndexOf(".") + 1)
		: collection;
	return definitionFor(app).entityRoutes[tail] ?? "";
}
