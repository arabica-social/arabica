import type { AppName } from "$lib/stores/session";
import { arabica } from "./arabica";
import { oolong } from "./oolong";

export type EntityRouteDefinition = Record<string, string>;

export type AppDefinition = {
	name: AppName;
	displayName: string;
	tagline: string;
	libraryPath: string;
	sessionNoun: string;
	sessionAction: string;
	commentCollection: string;
	entityRoutes: EntityRouteDefinition;
	feedRecordTypes: string[];
};

export const appDefinitions: Record<AppName, AppDefinition> = { arabica, oolong };

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
