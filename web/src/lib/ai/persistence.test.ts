import { describe, expect, it } from "vitest";
import { createSession, DraftEngine } from "./draft";
import {
  AccountSessionStore,
  listSessions,
  MAX_SESSIONS,
  MAX_ACCOUNT_STORAGE_BYTES,
  loadSession,
  removeSession,
  saveSession,
  storageKey,
} from "./persistence";
import type { StorageLike } from "./persistence";
import { MAX_STORAGE_BYTES, MAX_USER_CHARS } from "./types";
class MemoryStorage implements StorageLike {
  values = new Map<string, string>();
  fail = false;
  getItem(key: string) {
    if (this.fail) throw new Error("unavailable");
    return this.values.get(key) ?? null;
  }
  setItem(key: string, value: string) {
    if (this.fail) throw new Error("quota");
    this.values.set(key, value);
  }
  removeItem(key: string) {
    if (this.fail) throw new Error("unavailable");
    this.values.delete(key);
  }
}
const session = () => createSession("did:plc:a", "a-session", "brew");

describe("versioned account persistence", () => {
  it.each(["composer", "messages", "request"])(
    "rejects same-revision stale %s work using the full expected state",
    (change) => {
      const storage = new MemoryStorage();
      const original = session();
      saveSession(storage, original.accountDid, original);
      const newer = new DraftEngine(original);
      if (change === "messages")
        newer.appendMessage({ role: "user", content: "Newer conversation" });
      if (change === "request") newer.beginRequest("new-request");
      const next =
        change === "composer"
          ? { ...original, composerText: "Newer unsent work" }
          : newer.snapshot();
      expect(next.revision).toBe(original.revision);
      expect(
        saveSession(storage, original.accountDid, next, { expected: original })
          .ok,
      ).toBe(true);
      const before = storage.getItem(storageKey(original.accountDid));
      expect(
        saveSession(
          storage,
          original.accountDid,
          { ...original, composerText: "Stale tab" },
          { expected: original },
        ),
      ).toEqual({ ok: false, error: "conflict" });
      expect(saveSession(storage, original.accountDid, original)).toEqual({
        ok: false,
        error: "conflict",
      });
      expect(storage.getItem(storageKey(original.accountDid))).toBe(before);
    },
  );
  it.each(["saved", "discarded"] as const)(
    "never regresses a terminal %s session, even with its current expected state",
    (status) => {
      const storage = new MemoryStorage();
      const original = session();
      const terminal = {
        ...original,
        status,
        ...(status === "saved" ? { savedRecordURI: "/brews/saved" } : {}),
      };
      saveSession(storage, original.accountDid, terminal);
      expect(
        saveSession(
          storage,
          original.accountDid,
          { ...original, revision: 100 },
          { expected: terminal },
        ),
      ).toEqual({ ok: false, error: "conflict" });
      expect(loadSession(storage, original.accountDid)).toEqual({
        ok: true,
        value: terminal,
      });
    },
  );
  it("does not recreate a removed chat from a previously persisted snapshot", () => {
    const storage = new MemoryStorage();
    const original = session();
    saveSession(storage, original.accountDid, original);
    removeSession(storage, original.accountDid, original.id);
    expect(
      saveSession(storage, original.accountDid, original, {
        expected: original,
      }),
    ).toEqual({ ok: false, error: "conflict" });
  });
  it("persists sticky editor visibility even after incomplete edits or manual reveal", () => {
    const storage = new MemoryStorage();
    const e = new DraftEngine(session());
    e.edit({ bean_rkey: "b", brewer_rkey: "v" });
    e.edit({ bean_rkey: null, brewer_rkey: null });
    saveSession(storage, "did:plc:a", e.snapshot());
    let loaded = loadSession(storage, "did:plc:a");
    expect(loaded.ok && loaded.value?.editorRevealed).toBe(true);
    const manual = new DraftEngine(session());
    manual.revealEditor();
    saveSession(storage, "did:plc:a", manual.snapshot(), {
      expected: e.snapshot(),
    });
    loaded = loadSession(storage, "did:plc:a");
    expect(loaded.ok && loaded.value?.editorRevealed).toBe(true);
    const malformed = { ...session(), editorRevealed: "true" };
    storage.values.set(
      storageKey("did:plc:a"),
      JSON.stringify({
        version: 1,
        accountDid: "did:plc:a",
        session: malformed,
      }),
    );
    expect(loadSession(storage, "did:plc:a")).toEqual({
      ok: false,
      error: "corrupt",
    });
  });
  it("roundtrips nested leaves, explicit clears, conversation, revisions, and undo", () => {
    const storage = new MemoryStorage();
    const e = new DraftEngine(session());
    e.appendMessage({ role: "user", content: "18g coffee" });
    const t = e.beginRequest("r")!;
    e.applyResponse(t, {
      coffee_amount: 18,
      tasting_notes: null,
      "espresso_params.pressure": 9,
    });
    expect(saveSession(storage, "did:plc:a", e.snapshot()).ok).toBe(true);
    const result = loadSession(storage, "did:plc:a");
    expect(result).toEqual({ ok: true, value: e.snapshot() });
    if (result.ok && result.value) {
      const restored = new DraftEngine(result.value);
      restored.undo();
      expect(restored.snapshot().draft.fields).toEqual({});
    }
    expect(loadSession(storage, "did:plc:b")).toEqual({
      ok: true,
      value: null,
    });
  });
  it("marks interrupted requests stopped without allowing ID reuse", () => {
    const storage = new MemoryStorage();
    const e = new DraftEngine(session());
    const t = e.beginRequest("r")!;
    saveSession(storage, "did:plc:a", e.snapshot());
    const result = loadSession(storage, "did:plc:a");
    if (!result.ok || !result.value)
      throw new Error("Missing persisted session");
    const restored = new DraftEngine(result.value);
    expect(restored.snapshot().pending).toBeUndefined();
    expect(restored.applyResponse(t, { coffee_amount: 18 }).status).toBe(
      "inactive",
    );
    expect(
      restored.applyChoice({ token: t, patch: { bean_rkey: "late" } }).status,
    ).toBe("inactive");
    expect(restored.beginRequest("r")).toBeNull();
  });
  it("refuses corrupt JSON, prototype kinds, malformed revisions, fields and unknown schemas without deleting them", () => {
    const storage = new MemoryStorage();
    const s = session();
    const records = [
      "{",
      "null",
      JSON.stringify({ version: 3 }),
      ...[
        { ...s, draft: { kind: "__proto__", fields: {} } },
        { ...s, draft: { kind: ["brew"], fields: {} } },
        { ...s, messages: [{ role: ["user"], content: "x" }] },
        {
          ...s,
          draft: { kind: "brew", fields: { espresso_params: { pressure: 9 } } },
        },
        { ...s, fieldRevisions: { coffee_amount: 10 } },
        { ...s, fieldRevisions: { unknown: 0 } },
        { ...s, messages: [{ role: "system", content: "x" }] },
        { ...s, completedRequestIds: ["unknown"] },
        { ...s, composerText: 1 },
        { ...s, composerText: "x".repeat(MAX_USER_CHARS + 1) },
        {
          ...s,
          pending: {
            accountDid: "did:plc:b",
            sessionId: "a-session",
            requestId: "r",
            revision: 0,
          },
        },
      ].map((value) =>
        JSON.stringify({ version: 1, accountDid: "did:plc:a", session: value }),
      ),
    ];
    for (const value of records) {
      storage.values.set(storageKey("did:plc:a"), value);
      expect(loadSession(storage, "did:plc:a").ok).toBe(false);
      expect(storage.values.get(storageKey("did:plc:a"))).toBe(value);
    }
    expect(() => createSession("a", "b", "__proto__" as "brew")).toThrow();
  });
  it("refuses cross-account serialization and copied storage data", () => {
    const storage = new MemoryStorage();
    expect(saveSession(storage, "did:plc:b", session())).toEqual({
      ok: false,
      error: "account",
    });
    saveSession(storage, "did:plc:a", session());
    storage.values.set(
      storageKey("did:plc:b"),
      storage.values.get(storageKey("did:plc:a"))!,
    );
    expect(loadSession(storage, "did:plc:b")).toEqual({
      ok: false,
      error: "account",
    });
  });
  it("leaves live draft and previous persisted work intact on quota failure", () => {
    const storage = new MemoryStorage();
    const e = new DraftEngine(session());
    e.edit({ tasting_notes: "first" });
    saveSession(storage, "did:plc:a", e.snapshot());
    const previous = storage.values.get(storageKey("did:plc:a"));
    e.edit({ tasting_notes: "second" });
    storage.fail = true;
    expect(saveSession(storage, "did:plc:a", e.snapshot())).toEqual({
      ok: false,
      error: "unavailable",
    });
    expect(loadSession(storage, "did:plc:a")).toEqual({
      ok: false,
      error: "unavailable",
    });
    expect(removeSession(storage, "did:plc:a")).toEqual({
      ok: false,
      error: "unavailable",
    });
    expect(e.snapshot().draft.fields).toHaveProperty("tasting_notes", "second");
    expect(storage.values.get(storageKey("did:plc:a"))).toBe(previous);
  });
  it("enforces UTF-8 byte cap without silent truncation or replacing previous work", () => {
    const storage = new MemoryStorage();
    const e = new DraftEngine(session());
    saveSession(storage, "did:plc:a", e.snapshot());
    const previous = storage.values.get(storageKey("did:plc:a"));
    e.edit({ tasting_notes: "☕".repeat(MAX_STORAGE_BYTES / 2) });
    expect(saveSession(storage, "did:plc:a", e.snapshot())).toEqual({
      ok: false,
      error: "limit",
    });
    expect(storage.values.get(storageKey("did:plc:a"))).toBe(previous);
    expect(
      (e.snapshot().draft.fields as { tasting_notes: string }).tasting_notes
        .length,
    ).toBe(MAX_STORAGE_BYTES / 2);
    storage.values.set(
      storageKey("did:plc:a"),
      " ".repeat(MAX_ACCOUNT_STORAGE_BYTES + 1),
    );
    expect(loadSession(storage, "did:plc:a")).toEqual({
      ok: false,
      error: "limit",
    });
  });
  it("only removes persisted work on explicit discard", () => {
    const storage = new MemoryStorage();
    saveSession(storage, "did:plc:a", session());
    expect(removeSession(storage, "did:plc:a").ok).toBe(true);
    expect(loadSession(storage, "did:plc:a")).toEqual({
      ok: true,
      value: null,
    });
  });
});

describe("account lifecycle", () => {
  it("stops before aborting, saves previous account, and never retargets old callbacks", () => {
    const storage = new MemoryStorage();
    let old: DraftEngine | undefined;
    let abortedWithPending = true;
    const store = new AccountSessionStore(storage, () => {
      abortedWithPending = !!old?.snapshot().pending;
    });
    store.switchAccount("did:plc:a", "session-a");
    old = store.active!;
    old.edit({ tasting_notes: "account a" });
    const token = old.beginRequest("r")!;
    store.switchAccount("did:plc:b", "session-b");
    expect(abortedWithPending).toBe(false);
    expect(store.active?.snapshot().accountDid).toBe("did:plc:b");
    expect(store.active?.snapshot().draft.fields).toEqual({});
    expect(
      store.active?.applyResponse(token, { tasting_notes: "wrong account" })
        .status,
    ).toBe("inactive");
    expect(old.applyResponse(token, { tasting_notes: "late" }).status).toBe(
      "inactive",
    );
    store.switchAccount("did:plc:a", "unused");
    expect(store.active?.snapshot().id).toBe("session-a");
    expect(store.active?.snapshot().draft.fields).toHaveProperty(
      "tasting_notes",
      "account a",
    );
    store.switchAccount(null, "unused");
    expect(store.active).toBeUndefined();
  });
  it("same-account activation retains engine and pending request", () => {
    const store = new AccountSessionStore(new MemoryStorage());
    store.switchAccount("a", "s");
    const e = store.active!;
    const t = e.beginRequest("r")!;
    store.switchAccount("a", "new");
    expect(store.active).toBe(e);
    expect(e.snapshot().pending).toEqual(t);
  });
  it("does not overwrite corrupt work on switching; reports both persistence and load failures", () => {
    const storage = new MemoryStorage();
    const store = new AccountSessionStore(storage);
    store.switchAccount("a", "s");
    store.active?.edit({ tasting_notes: "live" });
    storage.values.set(storageKey("b"), "broken");
    const switched = store.switchAccount("b", "new");
    expect(switched.load).toEqual({ ok: false, error: "corrupt" });
    expect(store.active).toBeUndefined();
    expect(storage.values.get(storageKey("b"))).toBe("broken");
    store.switchAccount("a", "s");
    storage.fail = true;
    const fail = store.switchAccount("c", "new");
    expect(fail.persistence.ok).toBe(false);
    expect(fail.load.ok).toBe(false);
    expect(store.active).toBeUndefined();
  });
});

describe("multiple local sessions", () => {
  it("roundtrips bounded unsent input per chat and rejects invalid metadata without overwriting", () => {
    const storage = new MemoryStorage();
    const original = {
      ...session(),
      composerText: "Unsent coffee question",
      editorRevealed: true,
    };
    saveSession(storage, original.accountDid, original);
    saveSession(storage, original.accountDid, {
      ...session(),
      id: "other",
      composerText: "Other input",
    });
    expect(loadSession(storage, original.accountDid, original.id)).toEqual({
      ok: true,
      value: original,
    });
    expect(loadSession(storage, original.accountDid)).toEqual({
      ok: true,
      value: { ...session(), id: "other", composerText: "Other input" },
    });
    const before = storage.getItem(storageKey(original.accountDid));
    for (const composerText of [
      MAX_USER_CHARS,
      "x".repeat(MAX_USER_CHARS + 1),
    ]) {
      expect(
        saveSession(storage, original.accountDid, {
          ...original,
          composerText,
        } as typeof original),
      ).toEqual({ ok: false, error: "corrupt" });
      expect(storage.getItem(storageKey(original.accountDid))).toBe(before);
    }
    expect(
      saveSession(
        storage,
        original.accountDid,
        {
          ...original,
          composerText: "x".repeat(MAX_USER_CHARS),
        },
        { expected: original },
      ).ok,
    ).toBe(true);
  });
  const did = "did:plc:a";
  const make = (id: string) => createSession(did, id, "bean");
  const read = (storage: MemoryStorage, id?: string) => {
    const loaded = loadSession(storage, did, id);
    if (!loaded.ok) throw new Error(loaded.error);
    return loaded.value;
  };
  const ids = (storage: MemoryStorage) => {
    const listed = listSessions(storage, did);
    if (!listed.ok) throw new Error(listed.error);
    return listed.value.map((item) => item.id);
  };
  it("reads legacy data losslessly without migrating until a successful mutation", () => {
    const storage = new MemoryStorage();
    const engine = new DraftEngine(make("old"));
    engine.edit({ name: "Legacy", origin: "Kenya" });
    engine.appendMessage({ role: "user", content: "my coffee" });
    const token = engine.beginRequest("consumed")!;
    const original = engine.snapshot();
    const raw = JSON.stringify({
      version: 1,
      accountDid: did,
      session: original,
    });
    storage.values.set(storageKey(did), raw);
    const { pending: _, ...stopped } = original;
    expect(read(storage)).toEqual(stopped);
    expect(listSessions(storage, did)).toEqual({
      ok: true,
      value: [
        { id: "old", title: "my coffee", kind: "bean", status: "active" },
      ],
    });
    expect(storage.getItem(storageKey(did))).toBe(raw);
    expect(
      new DraftEngine(read(storage)!).applyResponse(token, { name: "late" })
        .status,
    ).toBe("inactive");
    expect(saveSession(storage, did, make("new")).ok).toBe(true);
    expect(JSON.parse(storage.getItem(storageKey(did))!).version).toBe(2);
    expect(read(storage, "old")).toEqual(stopped);
    expect(ids(storage)).toEqual(["new", "old"]);
  });
  it("preserves histories and selects exact IDs; activation alone changes recency", () => {
    const storage = new MemoryStorage();
    const first = make("first");
    const saved = {
      ...make("saved"),
      status: "saved" as const,
      savedRecordURI: "/beans/saved",
    };
    saveSession(storage, did, first);
    saveSession(storage, did, saved);
    expect(read(storage)?.id).toBe("saved");
    expect(read(storage, "first")).toEqual(first);
    expect(read(storage, "missing")).toBeNull();
    expect(loadSession(storage, "other", "saved")).toEqual({
      ok: true,
      value: null,
    });
    saveSession(
      storage,
      did,
      { ...first, revision: 1 },
      { activate: false, expected: first },
    );
    expect(read(storage)?.id).toBe("saved");
    expect(ids(storage)).toEqual(["saved", "first"]);
    saveSession(storage, did, read(storage, "first")!);
    expect(read(storage)?.id).toBe("first");
    expect(ids(storage)).toEqual(["first", "saved"]);
    expect(read(storage, "saved")).toEqual(saved);
  });
  it("initial inactive persistence selects the only session and retains missing-ID reads", () => {
    const storage = new MemoryStorage();
    expect(
      saveSession(storage, did, make("only"), { activate: false }).ok,
    ).toBe(true);
    expect(read(storage)?.id).toBe("only");
    expect(read(storage, "")).toBeNull();
    expect(read(storage, "missing")).toBeNull();
  });
  it("titles use the first nonempty user message, bounded first line, then name", () => {
    const storage = new MemoryStorage();
    const named = make("name");
    named.draft.fields = { name: "  Draft name  " };
    saveSession(storage, did, named);
    const conversation = make("message");
    conversation.messages = [
      { role: "assistant", content: "ignore" },
      { role: "user", content: "  " },
      { role: "user", content: "  " + "x".repeat(80) + "\nsecond line" },
    ];
    saveSession(storage, did, conversation);
    saveSession(storage, did, make("empty"));
    const result = listSessions(storage, did);
    expect(result.ok && result.value.map((item) => item.title)).toEqual([
      "New chat",
      "x".repeat(64),
      "Draft name",
    ]);
  });
  it("removes only the named session, uses retained first fallback, and supports legacy recovery", () => {
    const storage = new MemoryStorage();
    saveSession(storage, did, make("a"));
    saveSession(storage, did, make("b"));
    saveSession(storage, did, make("c"));
    expect(removeSession(storage, did, "a").ok).toBe(true);
    expect(ids(storage)).toEqual(["c", "b"]);
    expect(read(storage)?.id).toBe("c");
    const raw = storage.getItem(storageKey(did));
    removeSession(storage, did, "missing");
    expect(storage.getItem(storageKey(did))).toBe(raw);
    removeSession(storage, did, "c");
    expect(read(storage)?.id).toBe("b");
    removeSession(storage, did, "b");
    expect(read(storage)).toBeNull();
    storage.values.set(storageKey(did), "corrupt");
    expect(removeSession(storage, did, "b")).toEqual({
      ok: false,
      error: "corrupt",
    });
    expect(storage.getItem(storageKey(did))).toBe("corrupt");
    expect(removeSession(storage, did).ok).toBe(true);
    expect(storage.getItem(storageKey(did))).toBeNull();
  });
  it("refuses session count overflow without eviction but permits updates at the count cap", () => {
    const storage = new MemoryStorage();
    for (let i = 0; i < MAX_SESSIONS; i++)
      expect(saveSession(storage, did, make(String(i))).ok).toBe(true);
    const raw = storage.getItem(storageKey(did));
    expect(saveSession(storage, did, make("overflow"))).toEqual({
      ok: false,
      error: "limit",
    });
    expect(storage.getItem(storageKey(did))).toBe(raw);
    expect(
      saveSession(
        storage,
        did,
        { ...make("0"), revision: 1 },
        { activate: false, expected: make("0") },
      ).ok,
    ).toBe(true);
    expect(ids(storage)).toHaveLength(MAX_SESSIONS);
    expect(read(storage)?.id).toBe(String(MAX_SESSIONS - 1));
  });
  it("refuses aggregate byte overflow and quota errors atomically", () => {
    const storage = new MemoryStorage();
    const large = (id: string) => {
      const value = make(id);
      value.messages = [{ role: "assistant", content: "☕".repeat(70_000) }];
      return value;
    };
    for (let i = 0; i < 9; i++)
      expect(saveSession(storage, did, large(String(i))).ok).toBe(true);
    const raw = storage.getItem(storageKey(did));
    expect(saveSession(storage, did, large("overflow"))).toEqual({
      ok: false,
      error: "limit",
    });
    expect(storage.getItem(storageKey(did))).toBe(raw);
    storage.setItem = () => {
      throw new Error("quota");
    };
    expect(saveSession(storage, did, make("small"))).toEqual({
      ok: false,
      error: "unavailable",
    });
    expect(removeSession(storage, did, "0")).toEqual({
      ok: false,
      error: "unavailable",
    });
    expect(storage.getItem(storageKey(did))).toBe(raw);
  });
  it("fails closed on every malformed envelope without losing or replacing any bytes", () => {
    const storage = new MemoryStorage();
    const base = {
      version: 2,
      accountDid: did,
      activeId: "a",
      sessions: [make("a")],
    };
    const invalid = [
      ["corrupt", "{"],
      ["version", JSON.stringify({ ...base, version: 99 })],
      ["account", JSON.stringify({ ...base, accountDid: "other" })],
      ["corrupt", JSON.stringify({ ...base, activeId: "missing" })],
      [
        "corrupt",
        JSON.stringify({ ...base, sessions: [make("a"), make("a")] }),
      ],
      [
        "corrupt",
        JSON.stringify({
          ...base,
          sessions: [{ ...make("a"), accountDid: "other" }],
        }),
      ],
      [
        "limit",
        JSON.stringify({
          ...base,
          sessions: Array.from({ length: MAX_SESSIONS + 1 }, (_, i) =>
            make(String(i)),
          ),
        }),
      ],
      [
        "limit",
        JSON.stringify({
          ...base,
          sessions: [
            {
              ...make("a"),
              messages: [
                { role: "assistant", content: "x".repeat(MAX_STORAGE_BYTES) },
              ],
            },
          ],
        }),
      ],
      ["limit", " ".repeat(MAX_ACCOUNT_STORAGE_BYTES + 1)],
    ];
    for (const [error, raw] of invalid) {
      storage.values.set(storageKey(did), raw);
      expect(loadSession(storage, did)).toEqual({ ok: false, error });
      expect(listSessions(storage, did)).toEqual({ ok: false, error });
      expect(saveSession(storage, did, make("replacement"))).toEqual({
        ok: false,
        error,
      });
      expect(removeSession(storage, did, "a")).toEqual({ ok: false, error });
      expect(storage.getItem(storageKey(did))).toBe(raw);
    }
  });
});
