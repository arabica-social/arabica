import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createSession, DraftEngine } from "#lib/ai/draft.js";
import {
  listSessions,
  loadSession,
  removeSession,
  saveSession,
  storageKey,
} from "#lib/ai/persistence.js";
import type { DraftSession } from "#lib/ai/types.js";
import { installWebLocks } from "../../../../tests/helpers/webLocks";
import { APIError } from "#lib/api/client.js";
import {
  attemptKey,
  clearAttempt,
  resolveAttempt,
  pendingSave,
  readAttempt,
  SavedRecordError,
  trackSave,
} from "./saves";

const did = "did:plc:a";
const make = (id = "original") => createSession(did, id, "bean");
const saved = (session: DraftSession): DraftSession => ({
  ...session,
  status: "saved",
  savedRecordURI: "/beans/saved",
  revision: session.revision + 1,
});
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}
beforeEach(() => {
  localStorage.clear();
  installWebLocks();
});
afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("chat save registry", () => {
  it.each([
    "missing",
    "corrupt",
    "terminal",
    "newer-fields",
    "same-revision-composer",
    "same-revision-messages",
    "pending-request",
  ])("does not POST when canonical state is %s", async (state) => {
    const original = make();
    if (state !== "missing") saveSession(localStorage, did, original);
    if (state === "corrupt") localStorage.setItem(storageKey(did), "{");
    else if (state !== "missing") {
      let next = { ...original };
      if (state === "terminal") next = saved(original);
      if (state === "newer-fields")
        next = {
          ...original,
          revision: 1,
          draft: { kind: "bean", fields: { name: "Newer" } },
        };
      if (state === "same-revision-composer")
        next.composerText = "Newer unsent work";
      if (state === "same-revision-messages")
        next.messages = [{ role: "user", content: "Newer conversation" }];
      if (state === "pending-request") {
        const engine = new DraftEngine(original);
        engine.beginRequest("pending");
        next = engine.snapshot();
      }
      saveSession(localStorage, did, next, { expected: original });
    }
    const write = vi.fn(async () => saved(original));
    await expect(trackSave(original, write)).rejects.toThrow(
      "could not be verified",
    );
    expect(write).not.toHaveBeenCalled();
    expect(readAttempt(did, original.id)).toBeNull();
  });
  it("fails safely without cross-tab locks, before any marker or POST", async () => {
    vi.stubGlobal("navigator", {});
    const original = make();
    saveSession(localStorage, did, original);
    const write = vi.fn(async () => saved(original));
    await expect(trackSave(original, write)).rejects.toThrow(
      "Web Locks support",
    );
    expect(write).not.toHaveBeenCalled();
    expect(readAttempt(did, original.id)).toBeNull();
  });
  it("serializes independently loaded tab modules and keeps success durable if stale data reappears", async () => {
    const original = make();
    saveSession(localStorage, did, original);
    const staleEnvelope = localStorage.getItem(storageKey(did))!;
    const gate = deferred<DraftSession>();
    const write = vi.fn(() => gate.promise);
    const first = trackSave(original, write);
    await vi.waitFor(() => expect(write).toHaveBeenCalledTimes(1));
    vi.resetModules();
    const tabB = await import("./saves");
    const secondWrite = vi.fn(async () => saved(original));
    const second = tabB.trackSave(original, secondWrite);
    const secondRejected = expect(second).rejects.toThrow("Check My Coffee");
    await expect(tabB.resolveAttempt(did, original.id)).rejects.toThrow(
      "still saving",
    );
    gate.resolve(saved(original));
    await first;
    await secondRejected;
    expect(secondWrite).not.toHaveBeenCalled();
    removeSession(localStorage, did, original.id);
    clearAttempt(did, original.id);
    await resolveAttempt(did, original.id);
    localStorage.setItem(storageKey(did), staleEnvelope);
    await expect(tabB.trackSave(original, secondWrite)).rejects.toThrow(
      "Check My Coffee",
    );
    expect(secondWrite).not.toHaveBeenCalled();
    expect(readAttempt(did, original.id)?.status).toBe("saved");
  });
  it("does not replace same-revision work arriving while a save is in flight", async () => {
    const original = make();
    saveSession(localStorage, did, original);
    const gate = deferred<DraftSession>();
    const operation = trackSave(original, () => gate.promise);
    await vi.waitFor(() =>
      expect(readAttempt(did, original.id)?.status).toBe("pending"),
    );
    const newer = {
      ...original,
      composerText: "Newer work, same field revision",
    };
    saveSession(localStorage, did, newer, { expected: original });
    gate.resolve(saved(original));
    await expect(operation).rejects.toBeInstanceOf(SavedRecordError);
    expect(
      loadSession(localStorage, did).ok && loadSession(localStorage, did),
    ).toEqual({ ok: true, value: newer });
    expect(readAttempt(did, original.id)?.status).toBe("saved");
  });
  it("completes the originating inactive session without selecting it or reordering history", async () => {
    const original = make();
    saveSession(localStorage, did, original);
    const gate = deferred<DraftSession>();
    const write = vi.fn(() => gate.promise);
    const operation = trackSave(original, write);
    expect(pendingSave(did, original.id)).toBe(operation);
    expect(trackSave(original, write)).toBe(operation);
    await vi.waitFor(() =>
      expect(readAttempt(did, original.id)?.status).toBe("pending"),
    );
    saveSession(localStorage, did, make("new"));
    gate.resolve(saved(original));
    await expect(operation).resolves.toEqual(saved(original));
    expect(write).toHaveBeenCalledTimes(1);
    expect(loadSession(localStorage, did)).toEqual({
      ok: true,
      value: make("new"),
    });
    expect(loadSession(localStorage, did, original.id)).toEqual({
      ok: true,
      value: saved(original),
    });
    const listed = listSessions(localStorage, did);
    expect(listed.ok && listed.value.map((item) => item.id)).toEqual([
      "new",
      "original",
    ]);
    expect(readAttempt(did, original.id)?.status).toBe("saved");
    expect(pendingSave(did, original.id)).toBeUndefined();
  });
  it("does not overwrite a newer originating revision and retains its saved outcome marker", async () => {
    const original = make();
    saveSession(localStorage, did, original);
    const gate = deferred<DraftSession>();
    const operation = trackSave(original, () => gate.promise);
    await vi.waitFor(() =>
      expect(readAttempt(did, original.id)?.status).toBe("pending"),
    );
    const newer = new DraftEngine(original);
    newer.edit({ name: "Newer manual work" });
    saveSession(localStorage, did, newer.snapshot(), { expected: original });
    saveSession(localStorage, did, make("active"));
    gate.resolve(saved(original));
    await expect(operation).rejects.toThrow(
      "Newer stored work was not replaced",
    );
    expect(loadSession(localStorage, did, original.id)).toEqual({
      ok: true,
      value: newer.snapshot(),
    });
    expect(loadSession(localStorage, did)).toEqual({
      ok: true,
      value: make("active"),
    });
    expect(readAttempt(did, original.id)?.status).toBe("saved");
    await expect(
      trackSave(original, async () => saved(original)),
    ).rejects.toThrow("Check My Coffee");
  });
  it("never resurrects a removed session or substitutes another account's same-ID session", async () => {
    const original = make();
    saveSession(localStorage, did, original);
    const other = { ...original, accountDid: "other" };
    saveSession(localStorage, "other", other);
    const gate = deferred<DraftSession>();
    const operation = trackSave(original, () => gate.promise);
    await vi.waitFor(() =>
      expect(readAttempt(did, original.id)?.status).toBe("pending"),
    );
    removeSession(localStorage, did, original.id);
    saveSession(localStorage, did, make("active"));
    gate.resolve(saved(original));
    await expect(operation).rejects.toThrow(
      "Newer stored work was not replaced",
    );
    expect(loadSession(localStorage, did, original.id)).toEqual({
      ok: true,
      value: null,
    });
    expect(loadSession(localStorage, "other", original.id)).toEqual({
      ok: true,
      value: other,
    });
    expect(readAttempt(did, original.id)?.status).toBe("saved");
  });
  it.each([400, 401, 409, 429])(
    "allows retry after a definitive %s rejection",
    async (status) => {
      const original = make();
      saveSession(localStorage, did, original);
      const error = new APIError("rejected", { kind: "http", status });
      await expect(
        trackSave(original, async () => {
          throw error;
        }),
      ).rejects.toBe(error);
      expect(readAttempt(did, original.id)).toBeNull();
      expect(pendingSave(did, original.id)).toBeUndefined();
    },
  );
  it.each([
    new APIError("failed", { kind: "http", status: 503 }),
    new APIError("offline", { kind: "network" }),
  ])("blocks duplicate retry after uncertain write failure", async (error) => {
    const original = make();
    saveSession(localStorage, did, original);
    await expect(
      trackSave(original, async () => {
        throw error;
      }),
    ).rejects.toBe(error);
    expect(readAttempt(did, original.id)?.status).toBe("pending");
    await expect(
      trackSave(original, async () => saved(original)),
    ).rejects.toThrow("Check My Coffee");
  });
  it("requires a writable attempt marker before beginning the mutation", async () => {
    saveSession(localStorage, did, make());
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("quota");
    });
    const write = vi.fn();
    await expect(trackSave(make(), write)).rejects.toThrow(
      "Cannot safely start saving",
    );
    expect(write).not.toHaveBeenCalled();
  });
  it("retains unknown markers when saved-result persistence fails, without losing the envelope", async () => {
    const original = make();
    saveSession(localStorage, did, original);
    const before = localStorage.getItem(storageKey(did));
    const realSet = Storage.prototype.setItem;
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(function (
      this: Storage,
      key,
      value,
    ) {
      if (key === storageKey(did)) throw new Error("quota");
      realSet.call(this, key, value);
    });
    await expect(
      trackSave(original, async () => saved(original)),
    ).rejects.toBeInstanceOf(SavedRecordError);
    expect(localStorage.getItem(storageKey(did))).toBe(before);
    expect(readAttempt(did, original.id)?.status).toBe("saved");
  });
  it("corrupt attempt markers fail closed and keys separate accounts and sessions", async () => {
    localStorage.setItem(attemptKey(did, "original"), "{");
    await expect(trackSave(make(), async () => saved(make()))).rejects.toThrow(
      "Check My Coffee",
    );
    expect(readAttempt("other", "original")).toBeNull();
    expect(readAttempt(did, "different")).toBeNull();
    expect(attemptKey("a:b", "c")).not.toBe(attemptKey("a", "b:c"));
  });
});
