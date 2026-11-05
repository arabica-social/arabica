import { requestJSON } from "./client";
import type { Draft, DraftKind, DraftPatch } from "#lib/ai/types.js";
import type { ChatImage } from "#lib/ai/images.js";
export type ChatRequest = {
  session_id: string;
  request_id: string;
  revision: number;
  messages: {
    role: "user" | "assistant";
    content: string;
    images?: ChatImage[];
    images_unavailable?: number;
  }[];
  draft: Draft;
};
export type ChatReply = {
  reply: string;
  kind: DraftKind;
  patch: DraftPatch;
  choices: { label: string; patch: DraftPatch; brewer_type?: string }[];
  brewer_type?: string;
};
export const aiStatus = (fetchFn: typeof fetch) =>
  requestJSON<{ enabled: boolean; allowed: boolean }>(
    fetchFn,
    "/api/ai/status",
  );
export const chat = (
  fetchFn: typeof fetch,
  body: ChatRequest,
  signal: AbortSignal,
) =>
  requestJSON<ChatReply>(fetchFn, "/api/ai/chat", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    signal,
  });
export const entryKey = (did: string) =>
  `arabica.ai.entry:${encodeURIComponent(did)}`;
