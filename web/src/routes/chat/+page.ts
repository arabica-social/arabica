import { error } from "@sveltejs/kit";
import type { PageLoad } from "./$types";
import { aiStatus } from "#lib/api/ai.js";

export const load: PageLoad = async ({ fetch }) => {
  const status = await aiStatus(fetch).catch(() => ({
    enabled: false,
    allowed: false,
  }));
  if (status.allowed !== true) error(404, "Not found");
  return { enabled: status.enabled === true };
};
