import type { PageLoad } from "./$types";
import type { SignupCategoriesResponse } from "$lib/types/signup";

// The server includes development-only providers when dev mode is enabled.
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
