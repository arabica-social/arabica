import { requestJSON } from "./client.js";

export type AIAccess = {
  entries: { did: string; granted_by: string; created_at: string }[];
  moderators: { did: string }[];
};

const path = "/api/_mod/ai-access";

export const getAIAccess = (fetchFn: typeof fetch) =>
  requestJSON<AIAccess>(fetchFn, path);

export const changeAIAccess = (
  fetchFn: typeof fetch,
  method: "PUT" | "DELETE",
  did: string,
) =>
  requestJSON<void>(fetchFn, path, {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ did }),
  });
