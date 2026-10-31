import type { DraftSession } from "#lib/ai/types.js";
import {
  loadSession,
  sameSession,
  saveSession,
  withAccountLock,
} from "#lib/ai/persistence.js";
import { APIError } from "#lib/api/client.js";
export class SavedRecordError extends Error {
  constructor(
    message: string,
    public saved: DraftSession,
  ) {
    super(message);
  }
}
// Convenience for route remounts only. Web Locks, not this map, exclude other tabs.
const pending = new Map<string, Promise<DraftSession>>();
const key = (did: string, sessionId: string) =>
  JSON.stringify([did, sessionId]);
export const attemptKey = (did: string, sessionId: string) =>
  `arabica.ai.save:${encodeURIComponent(did)}:${encodeURIComponent(sessionId)}`;
const lockKey = (did: string, sessionId: string) =>
  `arabica.ai.save:${key(did, sessionId)}`;
export type SaveAttempt = {
  revision: number;
  status: "pending" | "saved";
  recordURL?: string;
};
export function readAttempt(
  did: string,
  sessionId: string,
): SaveAttempt | null {
  try {
    const raw = localStorage.getItem(attemptKey(did, sessionId));
    if (raw === null) return null;
    const value = JSON.parse(raw);
    return value?.status === "saved" &&
      typeof value.recordURL === "string" &&
      value.recordURL
      ? value
      : { revision: 0, status: "pending" };
  } catch {
    return { revision: 0, status: "pending" };
  }
}
/** Successful outcomes are durable tombstones, including after chat deletion. */
export function clearAttempt(did: string, sessionId: string): void {
  if (readAttempt(did, sessionId)?.status !== "saved")
    localStorage.removeItem(attemptKey(did, sessionId));
}
/** Explicit unknown-outcome recovery cannot clear another tab's in-flight save. */
export async function resolveAttempt(
  did: string,
  sessionId: string,
): Promise<void> {
  await withAccountLock(did, () => {}); // Fail closed without Web Locks.
  await navigator.locks.request(
    lockKey(did, sessionId),
    { ifAvailable: true },
    async (lock) => {
      if (!lock)
        throw new Error(
          "This chat is still saving in another tab. Wait before continuing.",
        );
      await withAccountLock(did, () => clearAttempt(did, sessionId));
    },
  );
}
export const pendingSave = (did: string, sessionId: string) =>
  pending.get(key(did, sessionId));
export function trackSave(
  session: DraftSession,
  save: () => Promise<DraftSession>,
): Promise<DraftSession> {
  const id = key(session.accountDid, session.id);
  const existing = pending.get(id);
  if (existing) return existing;
  if (pending.size >= 8)
    return Promise.reject(
      new Error(
        "Other saves are still pending. Wait before saving another draft.",
      ),
    );
  const operation = (async () => {
    // Feature detection and all canonical preflight failures occur before invoking save/POST.
    await withAccountLock(session.accountDid, () => {});
    return navigator.locks.request(
      lockKey(session.accountDid, session.id),
      async () => {
        let request!: Promise<DraftSession>;
        await withAccountLock(session.accountDid, () => {
          if (readAttempt(session.accountDid, session.id))
            throw new Error(
              "Check My Coffee to resolve the previous save before saving again.",
            );
          const stored = loadSession(
            localStorage,
            session.accountDid,
            session.id,
            { keepPending: true },
          );
          if (
            !stored.ok ||
            !stored.value ||
            stored.value.status !== "active" ||
            stored.value.pending ||
            !sameSession(stored.value, session)
          )
            throw new Error(
              "This chat changed in another tab or could not be verified. Reopen the latest draft before saving.",
            );
          try {
            localStorage.setItem(
              attemptKey(session.accountDid, session.id),
              JSON.stringify({ revision: session.revision, status: "pending" }),
            );
          } catch {
            throw new Error(
              "Cannot safely start saving while device storage is unavailable. Your live draft is still here.",
            );
          }
          // Invoke POST under the same lock as canonical preflight, but release
          // the account lock while awaiting HTTP so other chats remain editable.
          try {
            request = save();
          } catch (error) {
            request = Promise.reject(error);
          }
          void request.catch(() => {});
        });
        let saved: DraftSession;
        try {
          saved = await request;
        } catch (error) {
          if (
            error instanceof APIError &&
            error.kind === "http" &&
            error.status >= 400 &&
            error.status < 500
          )
            await withAccountLock(session.accountDid, () =>
              clearAttempt(session.accountDid, session.id),
            );
          throw error; // Transport failures retain the unknown-outcome marker.
        }
        return withAccountLock(session.accountDid, () => {
          try {
            localStorage.setItem(
              attemptKey(session.accountDid, session.id),
              JSON.stringify({
                revision: session.revision,
                status: "saved",
                recordURL: saved.savedRecordURI,
              }),
            );
          } catch {
            throw new SavedRecordError(
              "The record was saved, but device storage could not be updated. Check My Coffee before continuing.",
              saved,
            );
          }
          const stored = loadSession(
            localStorage,
            session.accountDid,
            session.id,
            { keepPending: true },
          );
          if (!stored.ok || !sameSession(stored.value, session))
            throw new SavedRecordError(
              "An earlier draft version was saved. Newer stored work was not replaced. Check My Coffee before continuing.",
              saved,
            );
          const result = saveSession(localStorage, session.accountDid, saved, {
            activate: false,
            expected: session,
          });
          if (!result.ok)
            throw new SavedRecordError(
              "The record was saved, but device storage could not be updated. Check My Coffee before continuing.",
              saved,
            );
          // Never remove the successful marker: it protects even against reappearing stale data.
          return saved;
        });
      },
    );
  })();
  pending.set(id, operation);
  const remove = () => {
    if (pending.get(id) === operation) pending.delete(id);
  };
  void operation.then(remove, remove);
  return operation;
}
