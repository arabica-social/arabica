import { clone, createSession, DraftEngine, validPatch } from "./draft";
import { MAX_MESSAGES, MAX_STORAGE_BYTES, MAX_USER_CHARS } from "./types";
import type {
  DraftKind,
  DraftPatch,
  DraftSession,
  RequestToken,
} from "./types";

export interface StorageLike {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
}
export type StorageResult<T = undefined> =
  | { ok: true; value: T }
  | {
      ok: false;
      error:
        | "unavailable"
        | "corrupt"
        | "version"
        | "limit"
        | "account"
        | "conflict";
    };
export const SESSION_SCHEMA_VERSION = 2;
export const MAX_SESSIONS = 20;
export const MAX_ACCOUNT_STORAGE_BYTES = 2 * 1024 * 1024;
export interface SessionSummary {
  id: string;
  title: string;
  status: DraftSession["status"];
  kind: DraftKind;
}
type Envelope = {
  version: 2;
  accountDid: string;
  activeId: string | null;
  sessions: DraftSession[];
};
export const storageKey = (accountDid: string) =>
  `arabica.ai.sessions.v1:${encodeURIComponent(accountDid)}`;
const bytes = (value: string) => new TextEncoder().encode(value).byteLength;
const object = (value: unknown): value is Record<string, unknown> =>
  !!value && typeof value === "object" && !Array.isArray(value);
const integer = (value: unknown): value is number =>
  Number.isSafeInteger(value) && (value as number) >= 0;
const text = (value: unknown): value is string =>
  typeof value === "string" && value.length > 0;
const kinds = ["brew", "bean", "grinder", "brewer"];
function validToken(
  value: unknown,
  session: DraftSession,
): value is RequestToken {
  return (
    object(value) &&
    value.accountDid === session.accountDid &&
    value.sessionId === session.id &&
    text(value.requestId) &&
    session.requestIds.includes(value.requestId) &&
    integer(value.revision) &&
    value.revision <= session.revision
  );
}
/** Do not treat parsed JSON as trusted session state. A bad record is retained for explicit recovery. */
export function validSession(
  value: unknown,
  accountDid: string,
): value is DraftSession {
  if (
    !object(value) ||
    value.accountDid !== accountDid ||
    !text(value.id) ||
    typeof value.status !== "string" ||
    !["active", "saved", "discarded"].includes(value.status) ||
    !integer(value.revision) ||
    !object(value.draft) ||
    typeof value.draft.kind !== "string" ||
    !kinds.includes(value.draft.kind) ||
    !validPatch(value.draft.kind as DraftKind, value.draft.fields) ||
    !object(value.fieldRevisions)
  )
    return false;
  const kind = value.draft.kind as DraftKind;
  const revisionLimit = value.revision;
  if (
    !Object.entries(value.fieldRevisions).every(
      ([key, revision]) =>
        validPatch(kind, { [key]: null }) &&
        integer(revision) &&
        revision <= revisionLimit,
    )
  )
    return false;
  if (
    !Array.isArray(value.requestIds) ||
    value.requestIds.length > MAX_MESSAGES ||
    !value.requestIds.every(text) ||
    new Set(value.requestIds).size !== value.requestIds.length ||
    !Array.isArray(value.completedRequestIds) ||
    new Set(value.completedRequestIds).size !==
      value.completedRequestIds.length ||
    !value.completedRequestIds.every(
      (id) =>
        typeof id === "string" && (value.requestIds as string[]).includes(id),
    )
  )
    return false;
  if (
    !Array.isArray(value.messages) ||
    value.messages.length > MAX_MESSAGES ||
    !value.messages.every(
      (message) =>
        object(message) &&
        typeof message.role === "string" &&
        ["user", "assistant"].includes(message.role) &&
        typeof message.content === "string" &&
        (message.role !== "user" || message.content.length <= MAX_USER_CHARS),
    )
  )
    return false;
  if (
    value.editorRevealed !== undefined &&
    typeof value.editorRevealed !== "boolean"
  )
    return false;
  if (
    value.composerText !== undefined &&
    (typeof value.composerText !== "string" ||
      value.composerText.length > MAX_USER_CHARS)
  )
    return false;
  if (value.brewerType !== undefined && typeof value.brewerType !== "string")
    return false;
  const s = value as unknown as DraftSession;
  if (
    s.pending &&
    (!validToken(s.pending, s) ||
      s.status !== "active" ||
      s.completedRequestIds.includes(s.pending.requestId))
  )
    return false;
  if (s.savedRecordURI !== undefined && !text(s.savedRecordURI)) return false;
  if (s.status === "saved" && !text(s.savedRecordURI)) return false;
  if (s.undo !== undefined) {
    const u = s.undo;
    if (
      !object(u) ||
      !integer(u.revision) ||
      u.revision > s.revision ||
      !validPatch(kind, u.before) ||
      !Array.isArray(u.absent) ||
      !u.absent.every(
        (key) => typeof key === "string" && Object.hasOwn(u.before, key),
      ) ||
      (u.brewerType !== undefined && typeof u.brewerType !== "string")
    )
      return false;
  }
  return true;
}
/** Read and validate the whole account before any mutation; never repair by overwriting. */
function readEnvelope(
  storage: StorageLike,
  accountDid: string,
): StorageResult<Envelope> {
  let serialized: string | null;
  try {
    serialized = storage.getItem(storageKey(accountDid));
  } catch {
    return { ok: false, error: "unavailable" };
  }
  if (serialized === null)
    return {
      ok: true,
      value: { version: 2, accountDid, activeId: null, sessions: [] },
    };
  if (bytes(serialized) > MAX_ACCOUNT_STORAGE_BYTES)
    return { ok: false, error: "limit" };
  let record: unknown;
  try {
    record = JSON.parse(serialized);
  } catch {
    return { ok: false, error: "corrupt" };
  }
  if (!object(record)) return { ok: false, error: "corrupt" };
  if (record.version !== 1 && record.version !== SESSION_SCHEMA_VERSION)
    return { ok: false, error: "version" };
  if (record.accountDid !== accountDid) return { ok: false, error: "account" };
  if (record.version === 1) {
    if (bytes(serialized) > MAX_STORAGE_BYTES)
      return { ok: false, error: "limit" };
    if (!validSession(record.session, accountDid))
      return { ok: false, error: "corrupt" };
    return {
      ok: true,
      value: {
        version: 2,
        accountDid,
        activeId: record.session.id,
        sessions: [record.session],
      },
    };
  }
  if (!Array.isArray(record.sessions)) return { ok: false, error: "corrupt" };
  if (record.sessions.length > MAX_SESSIONS)
    return { ok: false, error: "limit" };
  if (!record.sessions.every((session) => validSession(session, accountDid)))
    return { ok: false, error: "corrupt" };
  const sessions = record.sessions as DraftSession[];
  if (
    sessions.some(
      (session) => bytes(JSON.stringify(session)) > MAX_STORAGE_BYTES,
    )
  )
    return { ok: false, error: "limit" };
  if (
    new Set(sessions.map((session) => session.id)).size !== sessions.length ||
    (sessions.length === 0
      ? record.activeId !== null
      : !sessions.some((session) => session.id === record.activeId))
  )
    return { ok: false, error: "corrupt" };
  return {
    ok: true,
    value: {
      version: 2,
      accountDid,
      activeId: record.activeId as string | null,
      sessions,
    },
  };
}
function writeEnvelope(
  storage: StorageLike,
  envelope: Envelope,
): StorageResult {
  const serialized = JSON.stringify(envelope);
  if (bytes(serialized) > MAX_ACCOUNT_STORAGE_BYTES)
    return { ok: false, error: "limit" };
  try {
    storage.setItem(storageKey(envelope.accountDid), serialized);
    return { ok: true, value: undefined };
  } catch {
    return { ok: false, error: "unavailable" };
  }
}
/** Full persisted state, not the draft-field revision (messages/composer need guards too). */
export const sameSession = (a: DraftSession | null, b: DraftSession | null) =>
  JSON.stringify(a) === JSON.stringify(b);

/** Browser callers serialize envelope/attempt mutations across tab module instances.
 * There is deliberately no localStorage check/set or in-memory lock fallback.
 */
export async function withAccountLock<T>(
  accountDid: string,
  action: () => T | Promise<T>,
): Promise<T> {
  if (typeof navigator === "undefined" || !navigator.locks?.request)
    return Promise.reject(
      new Error(
        "This browser cannot safely coordinate saves across tabs. Use a browser with Web Locks support; your live draft is still here.",
      ),
    );
  return await navigator.locks.request(
    `arabica.ai.account:${encodeURIComponent(accountDid)}`,
    async () => await action(),
  );
}

/** Synchronous storage primitive; browser callers must hold withAccountLock.
 * Updating requires the exact expected persisted snapshot; omission permits only
 * creation or identical state (for selection), never an unguarded replacement.
 */
export function saveSession(
  storage: StorageLike,
  accountDid: string,
  session: DraftSession,
  options: { activate?: boolean; expected?: DraftSession | null } = {},
): StorageResult {
  if (session.accountDid !== accountDid) return { ok: false, error: "account" };
  if (!validSession(session, accountDid))
    return { ok: false, error: "corrupt" };
  if (bytes(JSON.stringify(session)) > MAX_STORAGE_BYTES)
    return { ok: false, error: "limit" };
  const loaded = readEnvelope(storage, accountDid);
  if (!loaded.ok) return loaded;
  const envelope = loaded.value;
  const index = envelope.sessions.findIndex((item) => item.id === session.id);
  const canonical = envelope.sessions[index] ?? null;
  if (
    (options.expected !== undefined
      ? !sameSession(canonical, options.expected)
      : canonical !== null && !sameSession(canonical, session)) ||
    (canonical &&
      (session.revision < canonical.revision ||
        (canonical.status !== "active" && !sameSession(canonical, session))))
  )
    return { ok: false, error: "conflict" };
  if (index < 0 && envelope.sessions.length >= MAX_SESSIONS)
    return { ok: false, error: "limit" };
  if (options.activate !== false) {
    envelope.sessions = [
      session,
      ...envelope.sessions.filter((item) => item.id !== session.id),
    ];
    envelope.activeId = session.id;
  } else {
    if (index < 0) envelope.sessions.push(session);
    else envelope.sessions[index] = session;
    envelope.activeId ??= session.id;
  }
  return writeEnvelope(storage, envelope);
}
export function loadSession(
  storage: StorageLike,
  accountDid: string,
  sessionId?: string,
  options: { keepPending?: boolean } = {},
): StorageResult<DraftSession | null> {
  const loaded = readEnvelope(storage, accountDid);
  if (!loaded.ok) return loaded;
  const found = loaded.value.sessions.find(
    (session) => session.id === (sessionId ?? loaded.value.activeId),
  );
  if (!found) return { ok: true, value: null };
  const session = clone(found);
  if (!options.keepPending) delete session.pending; // Reload never resurrects in-flight work; request IDs remain consumed.
  return { ok: true, value: session };
}
export function listSessions(
  storage: StorageLike,
  accountDid: string,
): StorageResult<SessionSummary[]> {
  const loaded = readEnvelope(storage, accountDid);
  if (!loaded.ok) return loaded;
  return {
    ok: true,
    value: loaded.value.sessions.map((session) => {
      const message = session.messages.find(
        (item) => item.role === "user" && item.content.trim(),
      );
      const name = (session.draft.fields as DraftPatch).name;
      const title = (
        message?.content.trim().split(/\r?\n/, 1)[0] ||
        (typeof name === "string" ? name.trim() : "") ||
        "New chat"
      ).slice(0, 64);
      return {
        id: session.id,
        title,
        status: session.status,
        kind: session.draft.kind,
      };
    }),
  };
}
/** Explicit removal only. Omitted ID retains the legacy whole-account recovery action. */
export function removeSession(
  storage: StorageLike,
  accountDid: string,
  sessionId?: string,
): StorageResult {
  if (sessionId !== undefined) {
    const loaded = readEnvelope(storage, accountDid);
    if (!loaded.ok) return loaded;
    const envelope = loaded.value;
    if (!envelope.sessions.some((session) => session.id === sessionId))
      return { ok: true, value: undefined };
    envelope.sessions = envelope.sessions.filter(
      (session) => session.id !== sessionId,
    );
    if (envelope.activeId === sessionId)
      envelope.activeId = envelope.sessions[0]?.id ?? null;
    return writeEnvelope(storage, envelope);
  }
  try {
    storage.removeItem(storageKey(accountDid));
    return { ok: true, value: undefined };
  } catch {
    return { ok: false, error: "unavailable" };
  }
}

/** Browser lifecycle adapter. Caller owns fetch AbortController; switching invalidates state before abort. */
export class AccountSessionStore {
  private current?: DraftEngine;
  private expected: DraftSession | null = null;
  constructor(
    private storage: StorageLike,
    private abortRequest: () => void = () => {},
  ) {}
  get active(): DraftEngine | undefined {
    return this.current;
  }
  persist(): StorageResult {
    if (!this.current) return { ok: true, value: undefined };
    const s = this.current.snapshot();
    const result = saveSession(this.storage, s.accountDid, s, {
      expected: this.expected,
    });
    if (result.ok) this.expected = s;
    return result;
  }
  switchAccount(
    accountDid: string | null,
    newSessionId: string,
    kind: DraftKind = "brew",
  ): { persistence: StorageResult; load: StorageResult<DraftEngine | null> } {
    if (accountDid && this.current?.snapshot().accountDid === accountDid)
      return {
        persistence: { ok: true, value: undefined },
        load: { ok: true, value: this.current },
      };
    this.current?.stopRequest(); // Late callbacks cannot apply while abort listeners run.
    this.abortRequest();
    const persistence = this.persist();
    this.current = undefined;
    if (!accountDid) return { persistence, load: { ok: true, value: null } };
    const loaded = loadSession(this.storage, accountDid, undefined, {
      keepPending: true,
    });
    if (!loaded.ok) return { persistence, load: loaded };
    this.expected = loaded.value;
    const restored = clone(
      loaded.value ?? createSession(accountDid, newSessionId, kind),
    );
    delete restored.pending;
    this.current = new DraftEngine(restored);
    return { persistence, load: { ok: true, value: this.current } };
  }
}
