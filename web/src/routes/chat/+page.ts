import type { PageLoad } from "./$types";
import { aiStatus } from "#lib/api/ai.js";
export const load: PageLoad = async ({ fetch }) => ({
  enabled: (await aiStatus(fetch).catch(() => ({ enabled: false }))).enabled,
});
