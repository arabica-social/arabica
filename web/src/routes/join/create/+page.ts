import type { PageLoad } from "./$types";
import type { SignupCategoriesResponse } from "$lib/types/signup";

// Fetch the PDS provider catalog from the JSON API. The endpoint returns
// dev-only providers when dev mode is enabled on the server.
export const load: PageLoad = async ({ fetch, url }) => {
  const error = url.searchParams.get("error") ?? "";
  try {
    const response = await fetch("/api/signup/categories", {
      headers: { Accept: "application/json" },
    });
    if (!response.ok) {
      return { error, categories: [], loadFailed: true };
    }
    const data = (await response.json()) as SignupCategoriesResponse;
    return {
      error,
      categories: data.categories ?? [],
      loadFailed: false,
    };
  } catch {
    return { error, categories: [], loadFailed: true };
  }
};
