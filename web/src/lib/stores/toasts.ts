// Toasts can be pushed directly or through the shared `notify` window event.

import { writable } from "svelte/store";

export type Toast = {
  id: number;
  message: string;
};

export const toasts = writable<Toast[]>([]);

let nextId = 1;
const timers = new Map<number, ReturnType<typeof setTimeout>>();

/** Push a toast message that auto-dismisses after `durationMs` (default 2.9s). */
export function pushToast(message: string, durationMs = 2900) {
  if (!message) return;
  const id = nextId++;
  toasts.update((list) => [...list, { id, message }]);
  const timer = setTimeout(() => dismissToast(id), durationMs);
  timers.set(id, timer);
}

/** Dismiss a toast by id. */
export function dismissToast(id: number) {
  toasts.update((list) => list.filter((t) => t.id !== id));
  const timer = timers.get(id);
  if (timer) {
    clearTimeout(timer);
    timers.delete(id);
  }
}

/** Clear all toasts. */
export function clearToasts() {
  toasts.set([]);
  for (const timer of timers.values()) clearTimeout(timer);
  timers.clear();
}

/** Extracts a toast message from supported `notify` event payloads. */
export function extractNotifyMessage(detail: unknown): string {
  if (!detail) return "";
  if (typeof detail === "string") return detail;
  if (typeof detail !== "object") return "";
  const d = detail as Record<string, unknown>;
  const value = d.value;
  if (value && typeof value === "object") {
    const m = (value as Record<string, unknown>).message;
    if (typeof m === "string") return m;
  }
  if (typeof value === "string") return value;
  if (typeof d.message === "string") return d.message;
  return "";
}
