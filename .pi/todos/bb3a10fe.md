{
  "id": "bb3a10fe",
  "title": "Port /recipes explore page to SvelteKit",
  "tags": [
    "sveltekit-migration",
    "recipes"
  ],
  "status": "completed",
  "created_at": "2026-07-10T11:27:40.280Z",
  "assigned_to_session": "019f4bc4-cbcc-7206-93ed-3c66624bbd4c"
}

The /recipes index page (Explore Recipes) is still served by the legacy templ handler (HandleRecipeExplore → RecipeExplorePage) which loads RecipeExploreIsland.svelte as a Svelte island. Port it to a native SvelteKit route.

Steps:
1. Add `"GET /recipes"` to SPAOwnedRoutes() in internal/arabica/handlers/routes.go
2. Create web/src/routes/recipes/+page.ts — auth check + initial fetch from /api/recipes/suggestions
3. Create web/src/routes/recipes/+page.svelte — port RecipeExploreIsland.svelte UI (search, filters, card grid, detail panel, fork, alpha warning, New Recipe link to /recipes/new)
4. Verify: svelte-check clean, go build, go test
