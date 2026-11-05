import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { installWebLocks } from "../helpers/webLocks";
import ChatWorkspace from "../../src/lib/components/ai/ChatWorkspace.svelte";
import DraftEditor from "../../src/lib/components/ai/DraftEditor.svelte";
import HomeEntry from "../../src/lib/components/ai/HomeEntry.svelte";
import { createSession, DraftEngine } from "../../src/lib/ai/draft";
import {
  saveSession,
  loadSession,
  storageKey,
} from "../../src/lib/ai/persistence";
import { entryKey } from "../../src/lib/api/ai";
import { attemptKey } from "../../src/lib/components/ai/saves";
import { goto } from "$app/navigation";
vi.mock("$app/navigation", () => ({ goto: vi.fn() }));
vi.mock("../../src/lib/stores/appCache", () => ({
  appCache: { invalidateCache: vi.fn() },
}));
const did = "did:plc:alice";
const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
const library = {
  beans: [{ rkey: "bean-1", name: "Ethiopia" }],
  brewers: [
    { rkey: "v60", name: "V60", brewer_type: "pourover" },
    { rkey: "espresso", name: "Espresso", brewer_type: "espresso" },
  ],
  grinders: [
    { rkey: "c40", name: "C40" },
    { rkey: "ode", name: "Ode" },
  ],
  roasters: [],
};
const reply = {
  reply: "Review your bean.",
  kind: "bean",
  patch: { name: "Gedeb", origin: "Ethiopia" },
  choices: [],
};
function install(
  handler?: (url: string, init?: RequestInit) => Promise<Response>,
) {
  const fetcher = vi.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
    if (String(url) === "/api/data") return json(library);
    if (handler) return handler(String(url), init);
    return json(reply);
  });
  vi.stubGlobal("fetch", fetcher);
  return fetcher;
}
function seed(
  kind: "brew" | "bean" | "grinder" | "brewer",
  patch: Record<string, unknown>,
) {
  const e = new DraftEngine(createSession(did, "session", kind));
  e.edit(patch);
  saveSession(localStorage, did, e.snapshot());
}
async function storedDraftButton(name: string) {
  const options = screen.getByLabelText("Chat options");
  if (!options.closest("details")!.open) await userEvent.click(options);
  const history = screen.getByText("Previous drafts", { exact: true });
  if (!history.closest("details")!.open) await userEvent.click(history);
  return screen.getByRole("button", { name });
}
async function message(text = "A bean from Ethiopia") {
  await userEvent.type(await screen.findByLabelText("Message"), text);
  await userEvent.click(
    screen.getByRole("button", { name: "Send", exact: true }),
  );
}
beforeEach(() => {
  installWebLocks();
  localStorage.clear();
  sessionStorage.clear();
  vi.clearAllMocks();
});
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});
describe("AI chat", () => {
  it("rejects a stale edit/pagehide even before the other tab's storage event is delivered", async () => {
    seed("bean", { name: "Gedeb", origin: "Ethiopia" });
    install();
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await screen.findByLabelText("Message");
    await waitFor(() =>
      expect(localStorage.getItem(storageKey(did))).toContain("composerText"),
    );
    const previous = loadSession(localStorage, did);
    if (!previous.ok || !previous.value) throw Error("fixture");
    const newer = {
      ...previous.value,
      composerText: "Newer unsent work",
      messages: [{ role: "user" as const, content: "Another tab's message" }],
    };
    saveSession(localStorage, did, newer, { expected: previous.value });
    await fireEvent.input(screen.getByLabelText("Message"), {
      target: { value: "Stale edit" },
    });
    await fireEvent(window, new Event("pagehide"));
    await waitFor(() =>
      expect(screen.getByLabelText("Message")).toHaveValue("Newer unsent work"),
    );
    expect(screen.getByRole("log")).toHaveTextContent("Another tab's message");
    const loaded = loadSession(localStorage, did);
    expect(loaded.ok && loaded.value).toEqual(newer);
  });
  it("reconciles a terminal cross-tab save and never restores the stale active session on pagehide", async () => {
    seed("bean", { name: "Gedeb", origin: "Ethiopia" });
    const fetcher = install();
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await screen.findByRole("button", { name: "Save bean" });
    await waitFor(() =>
      expect(localStorage.getItem(storageKey(did))).toContain("composerText"),
    );
    const previous = loadSession(localStorage, did);
    if (!previous.ok || !previous.value) throw Error("fixture");
    const other = new DraftEngine(previous.value);
    other.markSaved("/beans/other-tab");
    saveSession(localStorage, did, other.snapshot(), {
      expected: previous.value,
    });
    await fireEvent(
      window,
      new StorageEvent("storage", { key: storageKey(did) }),
    );
    await screen.findByRole("link", { name: "View saved bean" });
    await fireEvent(window, new Event("pagehide"));
    expect(screen.queryByRole("button", { name: "Save bean" })).toBeNull();
    expect(loadSession(localStorage, did)).toEqual({
      ok: true,
      value: other.snapshot(),
    });
    expect(fetcher.mock.calls.some(([, init]) => init?.method === "POST")).toBe(
      false,
    );
  });
  it("invalidates an in-flight AI engine on same-revision cross-tab changes before late callbacks apply", async () => {
    seed("bean", { name: "Gedeb", origin: "Ethiopia" });
    let finish!: (response: Response) => void;
    install(
      () =>
        new Promise<Response>((resolve) => {
          finish = resolve;
        }),
    );
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await message("Update this coffee");
    const previous = loadSession(localStorage, did, "session", {
      keepPending: true,
    });
    if (!previous.ok || !previous.value) throw Error("fixture");
    const newer = { ...previous.value, composerText: "Cross-tab composer" };
    delete newer.pending;
    saveSession(localStorage, did, newer, { expected: previous.value });
    await fireEvent(
      window,
      new StorageEvent("storage", { key: storageKey(did) }),
    );
    await act(() => finish(json({ ...reply, reply: "Obsolete response" })));
    expect(screen.getByLabelText("Message")).toHaveValue("Cross-tab composer");
    expect(screen.getByRole("log")).not.toHaveTextContent("Obsolete response");
    expect(loadSession(localStorage, did)).toEqual({ ok: true, value: newer });
  });
  it("keeps live chat usable when a previously writable store runs out of quota", async () => {
    seed("bean", { name: "Original", origin: "Kenya" });
    install();
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await screen.findByLabelText("Message");
    await waitFor(() =>
      expect(localStorage.getItem(storageKey(did))).toContain("composerText"),
    );
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw Error("quota");
    });
    await message("Change the name");
    await waitFor(() =>
      expect(screen.getByRole("log")).toHaveTextContent(reply.reply),
    );
    expect(screen.getByLabelText(/Name/)).toHaveValue("Gedeb");
    expect(screen.getByRole("log")).toHaveAttribute("aria-busy", "false");
  });
  it("does not submit old-account data when an account changes while Save waits for another tab's lock", async () => {
    seed("bean", { name: "Gedeb", origin: "Ethiopia" });
    const fetcher = install();
    let release!: () => void;
    const held = navigator.locks.request(
      `arabica.ai.save:${JSON.stringify([did, "session"])}`,
      () =>
        new Promise<void>((resolve) => {
          release = resolve;
        }),
    );
    await waitFor(() => expect(release).toBeTypeOf("function"));
    const view = render(ChatWorkspace, { accountDid: did, enabled: true });
    await userEvent.click(
      await screen.findByRole("button", { name: "Save bean" }),
    );
    await view.rerender({ accountDid: "did:plc:bob", enabled: true });
    release();
    await held;
    await waitFor(() =>
      expect(localStorage.getItem(attemptKey(did, "session"))).toContain(
        "pending",
      ),
    );
    expect(fetcher.mock.calls.some(([url]) => url === "/api/beans")).toBe(
      false,
    );
    expect(screen.queryByLabelText(/Name/)).toBeNull();
  });
  it("explains why saving is unsafe in browsers without Web Locks", async () => {
    seed("bean", { name: "Gedeb", origin: "Ethiopia" });
    vi.stubGlobal("navigator", {});
    const fetcher = install();
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await userEvent.click(
      await screen.findByRole("button", { name: "Save bean" }),
    );
    await waitFor(() =>
      expect(
        screen
          .getAllByRole("alert")
          .some((el) => el.textContent?.includes("Web Locks support")),
      ).toBe(true),
    );
    expect(fetcher.mock.calls.some(([, init]) => init?.method === "POST")).toBe(
      false,
    );
  });
  it.each(["home", "chat"])("%s composer sends with Enter", async (kind) => {
    const fetcher = install();
    if (kind === "home") render(HomeEntry, { accountDid: did });
    else render(ChatWorkspace, { accountDid: did, enabled: true });
    const composer = await screen.findByLabelText(
      kind === "home" ? "Describe a brew or bean" : "Message",
    );
    await userEvent.type(composer, "A bean");
    await userEvent.keyboard("{Enter}");
    if (kind === "home") {
      expect(goto).toHaveBeenCalledWith("/chat");
      expect(sessionStorage.getItem(entryKey(did))).toBe("A bean");
    } else {
      await screen.findByRole("button", { name: "Save bean" });
      expect(
        fetcher.mock.calls.filter(([u]) => u === "/api/ai/chat"),
      ).toHaveLength(1);
    }
  });
  it.each(["home", "chat"])(
    "%s composer keeps Shift+Enter as a newline",
    async (kind) => {
      const fetcher = install();
      if (kind === "home") render(HomeEntry, { accountDid: did });
      else render(ChatWorkspace, { accountDid: did, enabled: true });
      const composer = await screen.findByLabelText(
        kind === "home" ? "Describe a brew or bean" : "Message",
      );
      await userEvent.type(composer, "First");
      await userEvent.keyboard("{Shift>}{Enter}{/Shift}");
      await userEvent.type(composer, "Second");
      expect(composer).toHaveValue("First\nSecond");
      expect(goto).not.toHaveBeenCalled();
      expect(
        fetcher.mock.calls.filter(([u]) => u === "/api/ai/chat"),
      ).toHaveLength(0);
    },
  );
  it.each(["home", "chat"])(
    "%s composer never sends a composing Enter",
    async (kind) => {
      const fetcher = install();
      if (kind === "home") render(HomeEntry, { accountDid: did });
      else render(ChatWorkspace, { accountDid: did, enabled: true });
      const composer = await screen.findByLabelText(
        kind === "home" ? "Describe a brew or bean" : "Message",
      );
      await userEvent.type(composer, "Composing");
      await fireEvent.keyDown(composer, { key: "Enter", isComposing: true });
      await fireEvent.keyDown(composer, { key: "Enter", keyCode: 229 });
      expect(composer).toHaveValue("Composing");
      expect(goto).not.toHaveBeenCalled();
      expect(
        fetcher.mock.calls.filter(([u]) => u === "/api/ai/chat"),
      ).toHaveLength(0);
    },
  );

  it("cleans the old attempt marker only on explicit Delete after successful session removal", async () => {
    seed("bean", { name: "Gedeb", origin: "Ethiopia" });
    localStorage.setItem(
      attemptKey(did, "session"),
      JSON.stringify({ revision: 1, status: "pending" }),
    );
    install();
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await screen.findByRole("link", { name: "Check My Coffee" });
    await userEvent.click(await storedDraftButton("Delete chat: Gedeb"));
    expect(confirm).toHaveBeenCalledWith(
      expect.stringContaining("may already exist"),
    );
    expect(localStorage.getItem(attemptKey(did, "session"))).toBeNull();
    const loaded = loadSession(localStorage, did);
    expect(loaded.ok && loaded.value?.id).not.toBe("session");
  });
  it("retains the attempt marker when explicit session removal fails", async () => {
    seed("bean", { name: "Gedeb", origin: "Ethiopia" });
    const marker = JSON.stringify({ revision: 1, status: "pending" });
    localStorage.setItem(attemptKey(did, "session"), marker);
    install();
    vi.spyOn(window, "confirm").mockReturnValue(true);
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await screen.findByRole("link", { name: "Check My Coffee" });
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("unavailable");
    });
    await userEvent.click(await storedDraftButton("Delete chat: Gedeb"));
    expect(localStorage.getItem(attemptKey(did, "session"))).toBe(marker);
    expect(screen.getByLabelText(/Name/)).toHaveValue("Gedeb");
  });

  it.each(["", "{broken"])(
    "treats corrupt save marker %j as an unresolved outcome",
    async (marker) => {
      seed("bean", { name: "Gedeb", origin: "Ethiopia" });
      localStorage.setItem(attemptKey(did, "session"), marker);
      const fetcher = install();
      render(ChatWorkspace, { accountDid: did, enabled: true });
      await screen.findByRole("link", { name: "Check My Coffee" });
      expect(screen.getByRole("button", { name: "Save bean" })).toBeDisabled();
      expect(fetcher.mock.calls.some(([, i]) => i?.method === "POST")).toBe(
        false,
      );
    },
  );
  it("preserves uncertain marker for HTTP500 rather than offering blind save retry", async () => {
    seed("bean", { name: "Gedeb", origin: "Ethiopia" });
    install(async () => json({ error: "PDS timed out" }, 500));
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await userEvent.click(
      await screen.findByRole("button", { name: "Save bean" }),
    );
    await screen.findByRole("link", { name: "Check My Coffee" });
    expect(screen.getByRole("button", { name: "Save bean" })).toBeDisabled();
    expect(localStorage.getItem(attemptKey(did, "session"))).toContain(
      "pending",
    );
  });
  it("retains a usable live engine and known saved link after quota failure at completion", async () => {
    seed("bean", { name: "Gedeb", origin: "Ethiopia" });
    let resolve!: (r: Response) => void;
    install(
      () =>
        new Promise((r) => {
          resolve = r;
        }),
    );
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await userEvent.click(
      await screen.findByRole("button", { name: "Save bean" }),
    );
    const set = vi
      .spyOn(Storage.prototype, "setItem")
      .mockImplementation(() => {
        throw new Error("Quota exceeded");
      });
    resolve(json({ rkey: "saved" }));
    expect(
      await screen.findByRole("link", { name: "View saved version" }),
    ).toHaveAttribute("href", "/beans/did%3Aplc%3Aalice/saved");
    expect(
      screen.queryByRole("button", { name: "I checked; it was not saved" }),
    ).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save bean" })).toBeDisabled();
    set.mockRestore();
    await userEvent.type(screen.getByLabelText(/Name/), " retained");
    const live = loadSession(localStorage, did);
    expect(live.ok && live.value?.status).toBe("active");
    expect(live.ok && live.value?.draft.fields).toMatchObject({
      name: "Gedeb retained",
    });
  });

  it.each(["account-cycle", "remount"])(
    "keeps pending save frozen across %s",
    async (mode) => {
      seed("bean", { name: "Gedeb", origin: "Ethiopia" });
      let resolve!: (r: Response) => void;
      const fetcher = install(
        () =>
          new Promise((r) => {
            resolve = r;
          }),
      );
      let view = render(ChatWorkspace, { accountDid: did, enabled: true });
      await userEvent.click(
        await screen.findByRole("button", { name: "Save bean" }),
      );
      expect(localStorage.getItem(attemptKey(did, "session"))).toContain(
        "pending",
      );
      if (mode === "account-cycle") {
        await view.rerender({ accountDid: "did:plc:bob", enabled: true });
        await view.rerender({ accountDid: did, enabled: true });
      } else {
        view.unmount();
        view = render(ChatWorkspace, { accountDid: did, enabled: true });
      }
      expect(await screen.findByLabelText(/Name/)).toBeDisabled();
      expect(screen.getByLabelText("Message")).toBeDisabled();
      resolve(json({ rkey: "saved" }));
      await screen.findByRole("link", { name: "View saved bean" });
      expect(
        fetcher.mock.calls.filter(([, init]) => init?.method === "POST"),
      ).toHaveLength(1);
    },
  );
  it("does not overwrite a newer persisted revision when a late save completes", async () => {
    seed("bean", { name: "Gedeb", origin: "Ethiopia" });
    let resolve!: (r: Response) => void;
    install(
      () =>
        new Promise((r) => {
          resolve = r;
        }),
    );
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await userEvent.click(
      await screen.findByRole("button", { name: "Save bean" }),
    );
    const loaded = loadSession(localStorage, did);
    if (!loaded.ok || !loaded.value) throw Error("fixture");
    const newer = new DraftEngine(loaded.value);
    newer.edit({ name: "Newer work" });
    saveSession(localStorage, did, newer.snapshot(), {
      expected: loaded.value,
    });
    resolve(json({ rkey: "saved" }));
    await screen.findByRole("link", { name: "Check My Coffee" });
    const stored = loadSession(localStorage, did);
    expect(stored.ok && stored.value?.draft.fields).toMatchObject({
      name: "Newer work",
    });
    expect(screen.getByLabelText(/Name/)).toHaveValue("Newer work");
    await userEvent.type(screen.getByLabelText(/Name/), " retained");
    const edited = loadSession(localStorage, did);
    expect(edited.ok && edited.value?.draft.fields).toMatchObject({
      name: "Newer work retained",
    });
    expect(screen.getByRole("button", { name: "Save bean" })).toBeDisabled();
    expect(
      screen.getByRole("link", { name: "View saved version" }),
    ).toHaveAttribute("href", "/beans/did%3Aplc%3Aalice/saved");
  });
  it("blocks blind resubmit after a hard reload with an unresolved save marker", async () => {
    seed("bean", { name: "Gedeb", origin: "Ethiopia" });
    localStorage.setItem(
      attemptKey(did, "session"),
      JSON.stringify({ revision: 1, status: "pending" }),
    );
    const fetcher = install();
    render(ChatWorkspace, { accountDid: did, enabled: true });
    expect(
      await screen.findByRole("link", { name: "Check My Coffee" }),
    ).toHaveAttribute("href", "/my-coffee");
    expect(screen.getByRole("button", { name: "Save bean" })).toBeDisabled();
    expect(fetcher.mock.calls.some(([, init]) => init?.method === "POST")).toBe(
      false,
    );
  });
  it("offers live chat when storage reads are unavailable without overwriting inaccessible work", async () => {
    const get = vi
      .spyOn(Storage.prototype, "getItem")
      .mockImplementation(() => {
        throw new Error("unavailable");
      });
    install();
    render(ChatWorkspace, { accountDid: did, enabled: true });
    expect(await screen.findByLabelText("Message")).toBeEnabled();
    expect(screen.getByRole("alert")).toHaveTextContent("live only");
    get.mockRestore();
  });
  it("restores visible record type after declining draft discard", async () => {
    seed("bean", { name: "Gedeb", origin: "Ethiopia" });
    install();
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await userEvent.selectOptions(
      await screen.findByLabelText("Record type"),
      "brew",
    );
    expect(screen.getByLabelText("Record type")).toHaveValue("bean");
    expect(screen.getByRole("button", { name: "Save bean" })).toBeEnabled();
    confirm.mockRestore();
  });
  it("clears unsent home input when account changes", async () => {
    const view = render(HomeEntry, { accountDid: did });
    await userEvent.type(
      screen.getByLabelText("Describe a brew or bean"),
      "Private draft",
    );
    await view.rerender({ accountDid: "did:plc:bob" });
    expect(screen.getByLabelText("Describe a brew or bean")).toHaveValue("");
  });

  it("starts without a blank editor and never writes until Save; failure retains editable data", async () => {
    let attempts = 0;
    const fetcher = install(async (url) =>
      url === "/api/ai/chat"
        ? json(reply)
        : ++attempts === 1
          ? json({ error: "Try again", code: "validation_failed" }, 400)
          : json({ name: "Gedeb", rkey: "new-bean" }),
    );
    render(ChatWorkspace, { accountDid: did, enabled: true });
    expect(screen.queryByLabelText(/Name/)).not.toBeInTheDocument();
    await message();
    expect(await screen.findByLabelText(/Name/)).toHaveValue("Gedeb");
    expect(
      fetcher.mock.calls
        .filter(([, i]) => i?.method === "POST")
        .map(([u]) => u),
    ).toEqual(["/api/ai/chat"]);
    await userEvent.click(screen.getByRole("button", { name: "Save bean" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Try again");
    expect(screen.getByLabelText(/Name/)).toHaveValue("Gedeb");
    await userEvent.click(screen.getByRole("button", { name: "Save bean" }));
    expect(
      await screen.findByRole("link", { name: "View saved bean" }),
    ).toHaveAttribute("href", "/beans/did%3Aplc%3Aalice/new-bean");
  });
  it("opens manual completion below threshold and saves references only without invented rating", async () => {
    const fetcher = install(async () =>
      json({ brew: { rkey: "brew-1" }, author_did: did }),
    );
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await userEvent.click(
      await screen.findByRole("button", { name: "Add details manually" }),
    );
    await userEvent.selectOptions(
      await screen.findByLabelText(/Bean/),
      "bean-1",
    );
    await userEvent.selectOptions(screen.getByLabelText(/Brewer/), "v60");
    expect(fetcher.mock.calls.some(([, i]) => i?.method === "POST")).toBe(
      false,
    );
    await userEvent.click(screen.getByRole("button", { name: "Save brew" }));
    await screen.findByRole("link", { name: "View saved brew" });
    const call = fetcher.mock.calls.find(([u]) => u === "/api/brews");
    expect(JSON.parse(String(call?.[1]?.body))).toEqual({
      bean_rkey: "bean-1",
      brewer_rkey: "v60",
    });
  });
  it("applies ambiguous library choices locally without another POST and keeps sticky reveal", async () => {
    const fetcher = install(async () =>
      json({
        reply: "Which brewer?",
        kind: "brew",
        patch: { bean_rkey: "bean-1", coffee_amount: 20, water_amount: 300 },
        choices: [
          {
            label: "Use V60",
            patch: { brewer_rkey: "v60" },
            brewer_type: "pourover",
          },
        ],
      }),
    );
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await message("20g to 300g with Ethiopia");
    await userEvent.click(
      await screen.findByRole("button", { name: "Use V60" }),
    );
    expect(screen.getByLabelText(/Brewer/)).toHaveValue("v60");
    await userEvent.clear(screen.getByLabelText("Coffee (g)"));
    await userEvent.clear(screen.getByLabelText("Water (g)"));
    await userEvent.selectOptions(screen.getByLabelText(/Bean/), "");
    expect(screen.getByLabelText("Coffee (g)")).toBeInTheDocument();
    expect(
      fetcher.mock.calls.filter(([, i]) => i?.method === "POST"),
    ).toHaveLength(1);
  });
  it("requires confirmation for grinder replacement and preserves setting on cancel", async () => {
    seed("brew", {
      bean_rkey: "bean-1",
      brewer_rkey: "v60",
      grinder_rkey: "c40",
      grind_size: "24 clicks",
    });
    install();
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await screen.findByRole("option", { name: "Ode" });
    await userEvent.selectOptions(
      await screen.findByLabelText("Grinder"),
      "ode",
    );
    expect(confirm).toHaveBeenCalled();
    expect(screen.getByLabelText("Grinder")).toHaveValue("c40");
    expect(screen.getByLabelText("Grind setting")).toHaveValue("24 clicks");
    confirm.mockReturnValue(true);
    await userEvent.selectOptions(screen.getByLabelText("Grinder"), "ode");
    expect(screen.getByLabelText("Grind setting")).toHaveValue("");
    confirm.mockRestore();
  });
  it("rejects stopped and old-account responses without losing persisted manual edits", async () => {
    let resolve!: (response: Response) => void;
    install(
      () =>
        new Promise((r) => {
          resolve = r;
        }),
    );
    const view = render(ChatWorkspace, { accountDid: did, enabled: true });
    await message();
    await userEvent.click(screen.getByRole("button", { name: "Stop" }));
    resolve(json(reply));
    await waitFor(() =>
      expect(
        screen.queryByRole("button", { name: "Save bean" }),
      ).not.toBeInTheDocument(),
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Add details manually" }),
    );
    await userEvent.selectOptions(screen.getByLabelText(/Bean/), "bean-1");
    await view.rerender({ accountDid: "did:plc:bob", enabled: true });
    expect(screen.queryByLabelText(/Bean/)).not.toBeInTheDocument();
    const saved = loadSession(localStorage, did);
    expect(saved.ok && saved.value?.draft.fields).toMatchObject({
      bean_rkey: "bean-1",
    });
  });
  it("preserves later direct edits against AI proposals and exposes Undo", async () => {
    seed("bean", { name: "Original", origin: "Ethiopia" });
    let resolve!: (r: Response) => void;
    install(
      () =>
        new Promise((r) => {
          resolve = r;
        }),
    );
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await message("make it Kenya");
    const origin = await screen.findByLabelText(/Origin/);
    await userEvent.clear(origin);
    await userEvent.type(origin, "Colombia");
    resolve(json(reply));
    await screen.findByRole("button", { name: "Undo last AI change" });
    expect(origin).toHaveValue("Colombia");
    await userEvent.click(
      screen.getByRole("button", { name: "Undo last AI change" }),
    );
    expect(origin).toHaveValue("Colombia");
  });
  it("blocks all editing during save and marks only the originating session saved", async () => {
    seed("bean", { name: "Gedeb", origin: "Ethiopia" });
    let resolve!: (r: Response) => void;
    install(
      () =>
        new Promise((r) => {
          resolve = r;
        }),
    );
    const view = render(ChatWorkspace, { accountDid: did, enabled: true });
    await userEvent.click(
      await screen.findByRole("button", { name: "Save bean" }),
    );
    expect(screen.getByLabelText(/Name/)).toBeDisabled();
    expect(screen.getByLabelText("Message")).toBeDisabled();
    expect(screen.getByRole("button", { name: "New chat" })).toBeDisabled();
    await view.rerender({ accountDid: "did:plc:bob", enabled: true });
    resolve(json({ rkey: "new-bean" }));
    await waitFor(() => {
      const saved = loadSession(localStorage, did);
      expect(saved.ok && saved.value?.status).toBe("saved");
    });
    expect(
      screen.queryByRole("link", { name: "View saved bean" }),
    ).not.toBeInTheDocument();
  });
  it("retains corrupt storage until explicit discard and handles disabled direct loads", async () => {
    localStorage.setItem(storageKey(did), "broken");
    install();
    const view = render(ChatWorkspace, { accountDid: did, enabled: true });
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "has not been overwritten",
    );
    expect(localStorage.getItem(storageKey(did))).toBe("broken");
    await view.rerender({ accountDid: did, enabled: false });
    const heading = screen.getByRole("heading", {
      name: "AI entry is unavailable.",
    });
    expect(heading.closest(".card")).not.toBeNull();
    expect(
      screen.getByText("You can still add a brew or bean manually."),
    ).toBeVisible();
    expect(screen.getByRole("link", { name: "Add manually" })).toHaveAttribute(
      "href",
      "/add",
    );
    expect(screen.getByRole("link", { name: "Back to Home" })).toHaveAttribute(
      "href",
      "/",
    );
    expect(screen.queryByLabelText("Message")).not.toBeInTheDocument();
    expect(localStorage.getItem(storageKey(did))).toBe("broken");
  });
  it.each([
    [
      "bean",
      { name: "Gedeb", origin: "Ethiopia", rating: 8, closed: true },
      "/api/beans",
    ],
    [
      "grinder",
      { name: "C40", grinder_type: "Hand", burr_type: "Conical" },
      "/api/grinders",
    ],
    [
      "brewer",
      { name: "V60", brewer_type: "pourover", description: "Glass" },
      "/api/brewers",
    ],
    [
      "brew",
      {
        bean_rkey: "bean-1",
        brewer_rkey: "espresso",
        coffee_amount: 18,
        "espresso_params.yield_weight": 36,
        "espresso_params.pressure": 9,
        pours: [{ water_amount: 50, time_seconds: 30 }],
      },
      "/api/brews",
    ],
  ] as const)(
    "serializes %s through its existing create API only",
    async (kind, patch, path) => {
      seed(kind, patch);
      const fetcher = install(async () =>
        json(
          kind === "brew"
            ? { brew: { rkey: "saved" }, author_did: did }
            : { rkey: "saved" },
        ),
      );
      render(ChatWorkspace, { accountDid: did, enabled: true });
      if (kind === "brew")
        await screen.findByRole("option", { name: "Espresso" });
      await userEvent.click(
        await screen.findByRole("button", { name: `Save ${kind}` }),
      );
      await screen.findByRole("link", { name: `View saved ${kind}` });
      const writes = fetcher.mock.calls.filter(
        ([, init]) => init?.method === "POST",
      );
      expect(writes).toHaveLength(1);
      expect(writes[0][0]).toBe(path);
      const body = JSON.parse(String(writes[0][1]?.body));
      if (kind === "brew") {
        expect(body.espresso_params).toEqual({ yield_weight: 36, pressure: 9 });
        expect(body.pours).toEqual([{ water_amount: 50, time_seconds: 30 }]);
        expect(Object.keys(body)).not.toContain("espresso_params.pressure");
      } else expect(body).toEqual(patch);
    },
  );
  it("persists explicit editor opening and incomplete edits across reload", async () => {
    install();
    const view = render(ChatWorkspace, { accountDid: did, enabled: true });
    await userEvent.click(
      await screen.findByRole("button", { name: "Add details manually" }),
    );
    await userEvent.type(screen.getByLabelText("Coffee (g)"), "20");
    view.unmount();
    render(ChatWorkspace, { accountDid: did, enabled: true });
    expect(await screen.findByLabelText("Coffee (g)")).toHaveValue(20);
  });
  it("asks before stale library choice replaces a newer pick", async () => {
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
    install(async () =>
      json({
        reply: "Choose brewer",
        kind: "brew",
        patch: { bean_rkey: "bean-1", coffee_amount: 20, water_amount: 300 },
        choices: [
          {
            label: "Use V60",
            patch: { brewer_rkey: "v60" },
            brewer_type: "pourover",
          },
        ],
      }),
    );
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await message();
    await screen.findByRole("button", { name: "Use V60" });
    await userEvent.selectOptions(screen.getByLabelText(/Brewer/), "espresso");
    await userEvent.click(screen.getByRole("button", { name: "Use V60" }));
    expect(confirm).toHaveBeenCalled();
    expect(screen.getByLabelText(/Brewer/)).toHaveValue("espresso");
    confirm.mockReturnValue(true);
    await userEvent.click(screen.getByRole("button", { name: "Use V60" }));
    expect(screen.getByLabelText(/Brewer/)).toHaveValue("v60");
    confirm.mockRestore();
  });
  it("retries provider failure using current draft without duplicating the user turn", async () => {
    let calls = 0;
    const fetcher = install(async () =>
      ++calls === 1
        ? json({ error: "Provider unavailable" }, 503)
        : json(reply),
    );
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await message();
    await userEvent.click(
      await screen.findByRole("button", { name: "Retry", exact: true }),
    );
    await screen.findByRole("button", { name: "Save bean" });
    const requests = fetcher.mock.calls.filter(
      ([url]) => url === "/api/ai/chat",
    );
    expect(requests).toHaveLength(2);
    expect(JSON.parse(String(requests[1][1]?.body)).messages).toHaveLength(1);
  });
  it.each(["failed", "interrupted"])(
    "restores Retry after reloading a %s request without duplicating the user turn",
    async (outcome) => {
      let resolveOld!: (response: Response) => void;
      let resolveRetry!: (response: Response) => void;
      let calls = 0;
      const fetcher = install(() => {
        if (++calls === 1) {
          if (outcome === "failed")
            return Promise.resolve(
              json({ error: "Provider unavailable" }, 503),
            );
          return new Promise<Response>((resolve) => {
            resolveOld = resolve;
          });
        }
        return new Promise<Response>((resolve) => {
          resolveRetry = resolve;
        });
      });
      const view = render(ChatWorkspace, { accountDid: did, enabled: true });
      await message();
      if (outcome === "failed")
        await screen.findByRole("button", { name: "Retry", exact: true });
      else await screen.findByRole("button", { name: "Stop" });
      // Preserve the in-flight snapshot as a hard reload would, without leave cleanup.
      const persisted = localStorage.getItem(storageKey(did))!;
      view.unmount();
      localStorage.setItem(storageKey(did), persisted);
      render(ChatWorkspace, { accountDid: did, enabled: true });
      const retry = await screen.findByRole("button", {
        name: "Retry",
        exact: true,
      });
      expect(screen.getByRole("log")).toHaveAttribute("aria-busy", "false");
      expect(calls).toBe(1); // Restoration must not automatically resubmit.
      await userEvent.click(retry);
      const requests = fetcher.mock.calls.filter(
        ([url]) => url === "/api/ai/chat",
      );
      expect(requests).toHaveLength(2);
      const original = JSON.parse(String(requests[0][1]?.body));
      const retried = JSON.parse(String(requests[1][1]?.body));
      expect(retried.messages).toEqual([
        { role: "user", content: "A bean from Ethiopia" },
      ]);
      expect(retried.session_id).toBe(original.session_id);
      expect(retried.request_id).not.toBe(original.request_id);
      expect(
        screen.getByRole("log").querySelectorAll(".message.user"),
      ).toHaveLength(1);
      if (outcome === "interrupted") {
        await act(() =>
          resolveOld(json({ ...reply, reply: "Obsolete reply" })),
        );
        expect(screen.getByRole("log")).not.toHaveTextContent("Obsolete reply");
        expect(screen.getByRole("log")).toHaveAttribute("aria-busy", "true");
        expect(screen.queryByRole("button", { name: "Save bean" })).toBeNull();
      }
      await act(() => resolveRetry(json(reply)));
      await screen.findByRole("button", { name: "Save bean" });
      expect(screen.getByRole("log")).toHaveTextContent(reply.reply);
      await waitFor(() => {
        const loaded = loadSession(localStorage, did);
        expect(loaded.ok && loaded.value?.messages).toEqual([
          { role: "user", content: "A bean from Ethiopia" },
          { role: "assistant", content: reply.reply },
        ]);
      });
      expect(calls).toBe(2);
    },
  );
  it.each(["saved", "assistant"])(
    "does not restore Retry for a chat with %s as its final state",
    async (ending) => {
      const engine = new DraftEngine(createSession(did, "finished", "bean"));
      engine.appendMessage({ role: "user", content: "A bean from Ethiopia" });
      if (ending === "saved") engine.markSaved("/beans/alice/saved");
      else engine.appendMessage({ role: "assistant", content: "All done." });
      saveSession(localStorage, did, engine.snapshot());
      const fetcher = install();
      render(ChatWorkspace, { accountDid: did, enabled: true });
      expect(await screen.findByRole("log")).toHaveTextContent(
        "A bean from Ethiopia",
      );
      expect(
        screen.queryByRole("button", { name: "Retry", exact: true }),
      ).toBeNull();
      expect(
        fetcher.mock.calls.filter(([url]) => url === "/api/ai/chat"),
      ).toHaveLength(0);
    },
  );
  it("renders equipment with native controlled fields and emits numeric leaf paths", async () => {
    const onchange = vi.fn();
    const view = render(DraftEditor, {
      draft: { kind: "grinder", fields: { name: "C40" } },
      onchange,
    });
    await userEvent.selectOptions(screen.getByLabelText("Type"), "Hand");
    expect(onchange).toHaveBeenLastCalledWith(
      { grinder_type: "Hand" },
      undefined,
    );
    await view.rerender({
      draft: { kind: "brew", fields: {} },
      onchange,
      brewerType: "espresso",
    });
    expect(screen.queryByLabelText("Bloom water (g)")).not.toBeInTheDocument();
    await userEvent.click(
      screen.getByRole("button", { name: "Detailed", exact: true }),
    );
    await userEvent.click(screen.getByText("More brewing details"));
    await fireEvent.input(screen.getByLabelText("Pressure (bar)"), {
      target: { value: "9" },
    });
    expect(onchange).toHaveBeenLastCalledWith(
      { "espresso_params.pressure": 9 },
      undefined,
    );
  });
  it("home entry contains only composer controls, transfers text scoped to account, and opens chat", async () => {
    render(HomeEntry, { accountDid: did });
    expect(screen.queryByRole("heading")).not.toBeInTheDocument();
    await userEvent.type(
      screen.getByLabelText("Describe a brew or bean"),
      "A bean",
    );
    await userEvent.click(screen.getByRole("button", { name: "Send" }));
    expect(sessionStorage.getItem(entryKey(did))).toBe("A bean");
    expect(goto).toHaveBeenCalledWith("/chat");
  });
  it("automatically sends a scoped homepage handoff and reloads saved draft fields", async () => {
    sessionStorage.setItem(entryKey(did), "A bean");
    install();
    const view = render(ChatWorkspace, { accountDid: did, enabled: true });
    expect(await screen.findByLabelText(/Origin/)).toHaveValue("Ethiopia");
    expect(sessionStorage.getItem(entryKey(did))).toBeNull();
    view.unmount();
    render(ChatWorkspace, { accountDid: did, enabled: true });
    expect(await screen.findByLabelText(/Name/)).toHaveValue("Gedeb");
  });
});

it("New chat preserves the old draft and unresolved save marker in clickable history", async () => {
  seed("bean", { name: "Gedeb", origin: "Ethiopia" });
  const marker = JSON.stringify({ revision: 1, status: "pending" });
  localStorage.setItem(attemptKey(did, "session"), marker);
  install();
  const confirm = vi.spyOn(window, "confirm");
  render(ChatWorkspace, { accountDid: did, enabled: true });
  await screen.findByRole("link", { name: "Check My Coffee" });
  await userEvent.click(
    screen.getByRole("button", { name: "New chat", exact: true }),
  );
  expect(confirm).not.toHaveBeenCalled();
  expect(localStorage.getItem(attemptKey(did, "session"))).toBe(marker);
  expect(screen.queryByLabelText(/Name/)).not.toBeInTheDocument();
  await userEvent.click(await storedDraftButton("Open chat: Gedeb"));
  expect(screen.getByLabelText(/Name/)).toHaveValue("Gedeb");
  expect(screen.getByRole("button", { name: "Save bean" })).toBeDisabled();
  expect(
    screen.getByRole("link", { name: "Check My Coffee" }),
  ).toBeInTheDocument();
});

it("separates conversations and restores manual edits plus unsent input when switching", async () => {
  const fetcher = install();
  render(ChatWorkspace, { accountDid: did, enabled: true });
  await message("First coffee");
  await screen.findByRole("button", { name: "Save bean" });
  const origin = screen.getByLabelText(/Origin/);
  await userEvent.clear(origin);
  await userEvent.type(origin, "Kenya");
  await userEvent.type(screen.getByLabelText("Message"), "Unsent follow up");
  await userEvent.click(
    screen.getByRole("button", { name: "New chat", exact: true }),
  );
  expect(screen.getByRole("log")).not.toHaveTextContent("First coffee");
  expect(screen.getByLabelText("Message")).toHaveValue("");
  await message("Second coffee");
  await screen.findByRole("button", { name: "Save bean" });
  expect(screen.getByRole("log")).not.toHaveTextContent("First coffee");
  await userEvent.click(await storedDraftButton("Open chat: First coffee"));
  expect(screen.getByRole("log")).toHaveTextContent("First coffee");
  expect(screen.getByRole("log")).not.toHaveTextContent("Second coffee");
  expect(screen.getByLabelText(/Origin/)).toHaveValue("Kenya");
  expect(screen.getByLabelText("Message")).toHaveValue("Unsent follow up");
  expect(
    fetcher.mock.calls.filter(([url]) => url === "/api/ai/chat"),
  ).toHaveLength(2);
});

it("late responses from a previous chat cannot populate a new conversation", async () => {
  let resolve!: (response: Response) => void;
  install(
    () =>
      new Promise<Response>((r) => {
        resolve = r;
      }),
  );
  render(ChatWorkspace, { accountDid: did, enabled: true });
  await message("First pending coffee");
  await screen.findByRole("button", { name: "Stop" });
  await userEvent.click(
    screen.getByRole("button", { name: "New chat", exact: true }),
  );
  resolve(json(reply));
  await waitFor(() =>
    expect(
      screen.queryByRole("button", { name: "Save bean" }),
    ).not.toBeInTheDocument(),
  );
  expect(screen.getByRole("log")).not.toHaveTextContent("Review your bean.");
  expect(
    await storedDraftButton("Open chat: First pending coffee"),
  ).toBeInTheDocument();
});

it("a save completing in another chat never steals selection or overwrites its draft", async () => {
  seed("bean", { name: "Gedeb", origin: "Ethiopia" });
  const first = loadSession(localStorage, did);
  if (!first.ok || !first.value) throw new Error("seed failed");
  const second = new DraftEngine(createSession(did, "second", "bean"));
  second.edit({ name: "Second bean", origin: "Kenya" });
  saveSession(localStorage, did, second.snapshot());
  saveSession(localStorage, did, first.value);
  let resolve!: (response: Response) => void;
  install(
    () =>
      new Promise<Response>((r) => {
        resolve = r;
      }),
  );
  render(ChatWorkspace, { accountDid: did, enabled: true });
  await userEvent.click(
    await screen.findByRole("button", { name: "Save bean" }),
  );
  await userEvent.click(await storedDraftButton("Open chat: Second bean"));
  expect(screen.getByLabelText(/Name/)).toHaveValue("Second bean");
  resolve(json({ rkey: "saved-first" }));
  await waitFor(() => {
    const saved = loadSession(localStorage, did, "session");
    expect(saved.ok && saved.value?.status).toBe("saved");
  });
  const active = loadSession(localStorage, did);
  expect(active.ok && active.value?.id).toBe("second");
  expect(screen.getByLabelText(/Name/)).toHaveValue("Second bean");
  expect(
    screen.queryByRole("link", { name: "View saved bean" }),
  ).not.toBeInTheDocument();
  await userEvent.click(await storedDraftButton("Open chat: Gedeb"));
  expect(screen.getByRole("link", { name: "View saved bean" })).toHaveAttribute(
    "href",
    expect.stringContaining("saved-first"),
  );
});

it("refuses a twenty-first chat without discarding the selected draft", async () => {
  for (let i = 1; i <= 20; i++) {
    const engine = new DraftEngine(createSession(did, `bounded-${i}`, "bean"));
    engine.edit({ name: `Bean ${i}`, origin: "Kenya" });
    expect(saveSession(localStorage, did, engine.snapshot()).ok).toBe(true);
  }
  install();
  render(ChatWorkspace, { accountDid: did, enabled: true });
  await screen.findByRole("button", { name: "Save bean" });
  await userEvent.click(
    screen.getByRole("button", { name: "New chat", exact: true }),
  );
  expect(screen.getByRole("alert")).toHaveTextContent("Chat storage is full");
  expect(screen.getByLabelText(/Name/)).toHaveValue("Bean 20");
});

it("a detached save observer cannot steal selection from a remounted workspace", async () => {
  seed("bean", { name: "Gedeb", origin: "Ethiopia" });
  let resolve!: (response: Response) => void;
  install(
    () =>
      new Promise<Response>((r) => {
        resolve = r;
      }),
  );
  const firstView = render(ChatWorkspace, { accountDid: did, enabled: true });
  await userEvent.click(
    await screen.findByRole("button", { name: "Save bean" }),
  );
  firstView.unmount();
  const second = new DraftEngine(createSession(did, "remounted", "bean"));
  second.edit({ name: "Remounted bean", origin: "Kenya" });
  saveSession(localStorage, did, second.snapshot());
  render(ChatWorkspace, { accountDid: did, enabled: true });
  await screen.findByRole("button", { name: "Save bean" });
  resolve(json({ rkey: "saved-before-navigation" }));
  await waitFor(() => {
    const saved = loadSession(localStorage, did, "session");
    expect(saved.ok && saved.value?.status).toBe("saved");
  });
  const active = loadSession(localStorage, did);
  expect(active.ok && active.value?.id).toBe("remounted");
  expect(screen.getByLabelText(/Name/)).toHaveValue("Remounted bean");
});

it("preserves unsent composer text when a save finishes in an inactive chat", async () => {
  seed("bean", { name: "Gedeb", origin: "Ethiopia" });
  const first = loadSession(localStorage, did);
  if (!first.ok || !first.value) throw new Error("seed failed");
  const second = new DraftEngine(createSession(did, "other-composer", "bean"));
  second.edit({ name: "Another bean", origin: "Kenya" });
  saveSession(localStorage, did, second.snapshot());
  saveSession(localStorage, did, first.value);
  let resolve!: (response: Response) => void;
  install(
    () =>
      new Promise<Response>((r) => {
        resolve = r;
      }),
  );
  render(ChatWorkspace, { accountDid: did, enabled: true });
  await userEvent.type(
    await screen.findByLabelText("Message"),
    "Keep this unsent note",
  );
  await userEvent.click(screen.getByRole("button", { name: "Save bean" }));
  await userEvent.click(await storedDraftButton("Open chat: Another bean"));
  resolve(json({ rkey: "saved-with-composer" }));
  await waitFor(() => {
    const saved = loadSession(localStorage, did, "session");
    expect(saved.ok && saved.value?.status).toBe("saved");
    expect(saved.ok && saved.value?.composerText).toBe("Keep this unsent note");
  });
});

it("never silently discards a nonempty live-only conversation on New chat", async () => {
  vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
    throw new Error("unavailable");
  });
  install();
  const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
  render(ChatWorkspace, { accountDid: did, enabled: true });
  await userEvent.type(await screen.findByLabelText("Message"), "My only copy");
  await userEvent.click(
    screen.getByRole("button", { name: "New chat", exact: true }),
  );
  expect(confirm).toHaveBeenCalledWith(
    expect.stringContaining("lose this live-only conversation"),
  );
  expect(screen.getByLabelText("Message")).toHaveValue("My only copy");
});

it("starts a home handoff in a separate chat while an old save is pending", async () => {
  seed("bean", { name: "Gedeb", origin: "Ethiopia" });
  let resolve!: (response: Response) => void;
  install(async (url) =>
    url === "/api/ai/chat"
      ? json(reply)
      : new Promise<Response>((r) => {
          resolve = r;
        }),
  );
  const oldView = render(ChatWorkspace, { accountDid: did, enabled: true });
  await userEvent.click(
    await screen.findByRole("button", { name: "Save bean" }),
  );
  oldView.unmount();
  sessionStorage.setItem(entryKey(did), "New coffee from Home");
  render(ChatWorkspace, { accountDid: did, enabled: true });
  await screen.findByText("Review your bean.");
  expect(screen.getByRole("log")).toHaveTextContent("New coffee from Home");
  expect(sessionStorage.getItem(entryKey(did))).toBeNull();
  const active = loadSession(localStorage, did);
  expect(active.ok && active.value?.id).not.toBe("session");
  resolve(json({ rkey: "previous-save" }));
  await waitFor(() => {
    const old = loadSession(localStorage, did, "session");
    expect(old.ok && old.value?.status).toBe("saved");
  });
  const after = loadSession(localStorage, did);
  expect(after.ok && after.value?.id).toBe(active.ok && active.value?.id);
});

it("uses the same accessible arrow control in Community and chat composers", async () => {
  install();
  render(HomeEntry, { accountDid: did });
  render(ChatWorkspace, { accountDid: did, enabled: true });
  await screen.findByLabelText("Message");
  const buttons = screen.getAllByRole("button", { name: "Send", exact: true });
  expect(buttons).toHaveLength(2);
  for (const button of buttons) {
    expect(button).toHaveTextContent("↑");
    expect(button).toHaveAttribute("title", "Send");
    expect(button).toBeDisabled();
  }
});

describe("Chat-first workspace", () => {
  it("keeps useful drafts read-only until optional editing and saves directly from chat", async () => {
    const fetcher = install(async (url) =>
      url === "/api/ai/chat" ? json(reply) : json({ rkey: "from-chat" }),
    );
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await message();
    await screen.findByRole("button", { name: "Save bean" });
    const disclosure = screen.getByText("Edit details").closest("details")!;
    expect(disclosure).not.toHaveAttribute("open");
    expect(screen.getByRole("heading", { name: "Gedeb" })).toBeVisible();
    expect(screen.getByLabelText(/^Name/)).not.toBeVisible();
    expect(
      screen.queryByRole("button", { name: "Quick", exact: true }),
    ).toBeNull();
    expect(
      screen.queryByRole("button", { name: "Detailed", exact: true }),
    ).toBeNull();
    expect(screen.getByLabelText("Record type")).not.toBeVisible();
    expect(fetcher.mock.calls.some(([u]) => u === "/api/beans")).toBe(false);
    await userEvent.click(screen.getByRole("button", { name: "Save bean" }));
    await screen.findByRole("link", { name: "View saved bean" });
    expect(
      JSON.parse(
        String(fetcher.mock.calls.find(([u]) => u === "/api/beans")?.[1]?.body),
      ),
    ).toEqual({ name: "Gedeb", origin: "Ethiopia" });
    expect(screen.getByRole("heading", { name: "Gedeb" })).toBeVisible();
    expect(screen.queryByText("Edit details")).toBeNull();
  });
  it("shows actual working feedback immediately and removes it on failure", async () => {
    let reject!: (error: Error) => void;
    seed("bean", { name: "Gedeb", origin: "Ethiopia" });
    install(
      () =>
        new Promise<Response>((_, fail) => {
          reject = fail;
        }),
    );
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await message("Change the origin");
    expect(await screen.findByRole("status")).toHaveTextContent(
      "Working on your draft…",
    );
    expect(screen.getByRole("log")).toHaveAttribute("aria-busy", "true");
    expect(screen.getByRole("button", { name: "Save bean" })).toBeDisabled();
    reject(new Error("Provider unavailable"));
    await screen.findByRole("alert");
    expect(screen.queryByText("Working on your draft…")).toBeNull();
    expect(screen.getByRole("log")).toHaveAttribute("aria-busy", "false");
    expect(screen.getByRole("heading", { name: "Gedeb" })).toBeVisible();
    expect(
      screen.getByRole("button", { name: "Retry", exact: true }),
    ).toBeEnabled();
  });
  it("opens optional editing for an invalid hidden field instead of silently blocking Save", async () => {
    const fetcher = install(async () =>
      json({ ...reply, patch: { origin: null } }),
    );
    seed("bean", { name: "Gedeb", origin: "Ethiopia" });
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await message("Remove the origin");
    await waitFor(() =>
      expect(screen.getByLabelText(/^Origin/)).toHaveValue(""),
    );
    const disclosure = screen.getByText("Edit details").closest("details")!;
    expect(disclosure).not.toHaveAttribute("open");
    await userEvent.click(screen.getByRole("button", { name: "Save bean" }));
    await waitFor(() => expect(disclosure).toHaveAttribute("open"));
    expect(screen.getByLabelText(/^Origin/)).toBeVisible();
    expect(fetcher.mock.calls.some(([u]) => u === "/api/beans")).toBe(false);
  });
});

it("removes resolved bean alternatives without hiding an independent equipment ambiguity", async () => {
  install(async () =>
    json({
      reply: "Which bean and brewer?",
      kind: "brew",
      patch: {},
      choices: [
        { label: "Bean A", patch: { bean_rkey: "bean-1" } },
        { label: "Bean B", patch: { bean_rkey: "bean-2" } },
        {
          label: "Use V60",
          patch: { brewer_rkey: "v60" },
          brewer_type: "pourover",
        },
      ],
    }),
  );
  render(ChatWorkspace, { accountDid: did, enabled: true });
  await message("A brew");
  await userEvent.click(
    await screen.findByRole("button", { name: "Bean A", exact: true }),
  );
  expect(
    screen.queryByRole("button", { name: "Bean A", exact: true }),
  ).toBeNull();
  expect(
    screen.queryByRole("button", { name: "Bean B", exact: true }),
  ).toBeNull();
  expect(
    screen.getByRole("button", { name: "Use V60", exact: true }),
  ).toBeEnabled();
});

it("keeps rating visible outside details, editable and clearable, and saves only its chosen value", async () => {
  const fetcher = install(async (url) =>
    url === "/api/ai/chat"
      ? json({
          reply: "Ready",
          kind: "brew",
          patch: {
            bean_rkey: "bean-1",
            brewer_rkey: "v60",
            coffee_amount: 15,
            water_amount: 250,
          },
          choices: [],
          brewer_type: "pourover",
        })
      : json({ brew: { rkey: "rated-brew" }, author_did: did }),
  );
  const view = render(ChatWorkspace, { accountDid: did, enabled: true });
  await message("A V60 brew");
  const rating = await screen.findByLabelText("Rating (1–10)");
  expect(rating).toBeVisible();
  expect(rating).toHaveValue(null);
  expect(screen.queryByRole("button", { name: "Add rating" })).toBeNull();
  expect(
    screen.getByText("Edit details").closest("details"),
  ).not.toHaveAttribute("open");
  await userEvent.type(rating, "8");
  expect(rating).toHaveValue(8);
  await userEvent.clear(rating);
  expect(rating).toHaveValue(null);
  await userEvent.type(rating, "9");
  expect(
    fetcher.mock.calls.filter(([url]) => url === "/api/brews"),
  ).toHaveLength(0);
  view.unmount();
  render(ChatWorkspace, { accountDid: did, enabled: true });
  expect(await screen.findByLabelText("Rating (1–10)")).toHaveValue(9);
  await userEvent.click(screen.getByText("Edit details"));
  expect(screen.getAllByLabelText("Rating (1–10)")).toHaveLength(1);
  expect(screen.queryByRole("button", { name: "Add rating" })).toBeNull();
  await userEvent.click(screen.getByRole("button", { name: "Save brew" }));
  await screen.findByRole("link", { name: "View saved brew" });
  const writes = fetcher.mock.calls.filter(([url]) => url === "/api/brews");
  expect(writes).toHaveLength(1);
  expect(JSON.parse(String(writes[0][1]?.body)).rating).toBe(9);
});
