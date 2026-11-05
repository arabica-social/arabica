import type { PageLoad } from "./$types";
import { fetchEntityView } from "#lib/api/entityView.js";
import type { Recipe } from "#lib/types/entity_view.js";

export const load: PageLoad = async ({ fetch, params }) => {
  const result = await fetchEntityView<Recipe>(
    fetch,
    "recipes",
    params.actor,
    params.id,
  );
  return result;
};
