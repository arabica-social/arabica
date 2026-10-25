import { beforeEach, afterEach, describe, expect, it, vi } from "vitest";
import { get } from "svelte/store";
import { session } from "./session";
import {
  loadUserPrefs,
  setBrewFormMode,
  setSmartAutofillEnabled,
  userPrefs,
} from "./userPrefs";

function login(did: string) {
  session.update((s) => ({ ...s, did, isAuthenticated: !!did }));
}
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((r) => {
    resolve = r;
  });
  return { promise, resolve };
}
function response(preferences: unknown = {}) {
  return {
    ok: true,
    json: async () => ({ user_preferences: preferences }),
  } as Response;
}
const fetchMock = vi.fn<typeof fetch>();
beforeEach(() => {
  login("");
  login("did:plc:alice");
  fetchMock.mockReset();
  vi.stubGlobal("fetch", fetchMock);
  vi.spyOn(console, "warn").mockImplementation(() => {});
});
afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("account brew form preferences", () => {
  it.each([{}, { brew_form_mode: "invalid" }, { brew_form_mode: null }])(
    "defaults missing/invalid mode to Quick: %j",
    async (prefs) => {
      fetchMock.mockResolvedValue(response(prefs));
      await loadUserPrefs();
      expect(get(userPrefs)).toMatchObject({
        brew_form_mode: "quick",
        loading: false,
      });
    },
  );

  it("loads once per DID and resets on account change and logout", async () => {
    fetchMock.mockResolvedValue(
      response({ brew_form_mode: "detailed", smart_autofill: "off" }),
    );
    await Promise.all([loadUserPrefs(), loadUserPrefs()]);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(get(userPrefs)).toMatchObject({
      brew_form_mode: "detailed",
      smartAutofillEnabled: false,
    });
    login("did:plc:bob");
    expect(get(userPrefs)).toMatchObject({
      brew_form_mode: "quick",
      smartAutofillEnabled: true,
    });
    await loadUserPrefs();
    expect(fetchMock).toHaveBeenCalledTimes(2);
    login("");
    await loadUserPrefs();
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(get(userPrefs).brew_form_mode).toBe("quick");
  });

  it.each(["network", "http"])(
    "falls back on %s load failure",
    async (failure) => {
      if (failure === "network")
        fetchMock.mockRejectedValue(new Error("offline"));
      else fetchMock.mockResolvedValue({ ok: false, status: 500 } as Response);
      await loadUserPrefs();
      expect(get(userPrefs)).toEqual({
        brew_form_mode: "quick",
        smartAutofillEnabled: true,
        loading: false,
      });
    },
  );

  it("does not let initial reads overwrite explicit mode or autofill changes", async () => {
    const read = deferred<Response>();
    fetchMock.mockReturnValueOnce(read.promise).mockResolvedValue(response());
    const loading = loadUserPrefs();
    await setBrewFormMode("detailed");
    setSmartAutofillEnabled(false);
    read.resolve(response({ brew_form_mode: "quick", smart_autofill: "on" }));
    await loading;
    expect(get(userPrefs)).toEqual({
      brew_form_mode: "detailed",
      smartAutofillEnabled: false,
      loading: false,
    });
  });

  it("preserves selections made before loading, even when loading fails", async () => {
    fetchMock
      .mockResolvedValueOnce(response())
      .mockRejectedValueOnce(new Error("offline"));
    await setBrewFormMode("detailed");
    await loadUserPrefs();
    expect(get(userPrefs).brew_form_mode).toBe("detailed");
  });

  it("serializes writes and coalesces queued toggles, sending only mode", async () => {
    const first = deferred<Response>();
    fetchMock.mockReturnValueOnce(first.promise).mockResolvedValue(response());
    setSmartAutofillEnabled(false);
    const writes = [
      setBrewFormMode("detailed"),
      setBrewFormMode("quick"),
      setBrewFormMode("detailed"),
      setBrewFormMode("quick"),
    ];
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(get(userPrefs).brew_form_mode).toBe("quick");
    first.resolve(response());
    await Promise.all(writes);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    for (const [url, init] of fetchMock.mock.calls) {
      expect(url).toBe("/api/settings/preferences");
      expect(init).toMatchObject({
        method: "POST",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
      });
      expect(new Headers(init!.headers).has("Content-Type")).toBe(false);
      expect(init!.body).toBeInstanceOf(URLSearchParams);
    }
    expect(
      fetchMock.mock.calls.map(([, init]) => init!.body!.toString()),
    ).toEqual(["brew_form_mode=detailed", "brew_form_mode=quick"]);
    expect(get(userPrefs).smartAutofillEnabled).toBe(false);
  });

  it("keeps optimistic state after failed writes and allows later saves", async () => {
    fetchMock
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ ok: false, status: 500 } as Response)
      .mockResolvedValue(response());
    await setBrewFormMode("detailed");
    expect(get(userPrefs).brew_form_mode).toBe("detailed");
    await setBrewFormMode("quick");
    expect(get(userPrefs).brew_form_mode).toBe("quick");
    await setBrewFormMode("detailed");
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("ignores old account reads and cancels queued writes when DID changes", async () => {
    const read = deferred<Response>();
    const write = deferred<Response>();
    fetchMock
      .mockReturnValueOnce(read.promise)
      .mockReturnValueOnce(write.promise)
      .mockResolvedValue(response());
    const loading = loadUserPrefs();
    const writes = [setBrewFormMode("detailed"), setBrewFormMode("quick")];
    const signal = fetchMock.mock.calls[1][1]!.signal!;
    login("did:plc:bob");
    expect(signal.aborted).toBe(true);
    await loadUserPrefs();
    read.resolve(
      response({ brew_form_mode: "detailed", smart_autofill: "off" }),
    );
    write.resolve(response());
    await Promise.all([loading, ...writes]);
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(get(userPrefs)).toEqual({
      brew_form_mode: "quick",
      smartAutofillEnabled: true,
      loading: false,
    });
  });
});
