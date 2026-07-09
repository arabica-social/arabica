import { deleteJSON, postJSON, putJSON } from "./client";
import type { Roaster } from "$lib/types/entity_view";

export type RoasterInput = {
	name: string;
	location: string;
	website: string;
	source_ref?: string;
};

export function createRoaster(fetchFn: typeof fetch, input: RoasterInput): Promise<Roaster> {
	return postJSON<Roaster>(fetchFn, "/api/roasters", input);
}

export function updateRoaster(fetchFn: typeof fetch, rkey: string, input: RoasterInput): Promise<Roaster> {
	return putJSON<Roaster>(fetchFn, `/api/roasters/${encodeURIComponent(rkey)}`, input);
}

export function deleteEntity(fetchFn: typeof fetch, path: string): Promise<void> {
	return deleteJSON<void>(fetchFn, path);
}
