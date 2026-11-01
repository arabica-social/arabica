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
import { goto } from "$app/navigation";
import HomeEntry from "../../src/lib/components/ai/HomeEntry.svelte";
import ChatWorkspace from "../../src/lib/components/ai/ChatWorkspace.svelte";
import { prepareImage } from "../../src/lib/ai/images";
import { clearHomeImages, takeHomeImages } from "../../src/lib/ai/homeImages";
import { createSession, DraftEngine } from "../../src/lib/ai/draft";
import { saveSession, loadSession } from "../../src/lib/ai/persistence";
import { installWebLocks } from "../helpers/webLocks";
vi.mock("$app/navigation", () => ({ goto: vi.fn() }));
vi.mock("../../src/lib/ai/images", () => ({
  prepareImage: vi.fn(),
  MAX_IMAGES: 4,
}));
vi.mock("../../src/lib/stores/appCache", () => ({
  appCache: { invalidateCache: vi.fn() },
}));
const did = "did:plc:home-images";
const image = {
  image: { mime_type: "image/jpeg" as const, data: "HOME_PRIVATE_PIXELS" },
  preview: "data:image/jpeg;base64,HOME_PRIVATE_PIXELS",
};
const response = (body: unknown) =>
  new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
  });
let requests: {
  messages: { content: string; images?: unknown[] }[];
  draft: { fields: object };
}[];
async function paste(text = "", count = 1) {
  const event = new Event("paste", { bubbles: true, cancelable: true });
  Object.defineProperty(event, "clipboardData", {
    value: {
      items: Array.from({ length: count }, () => ({
        kind: "file",
        type: "image/png",
        getAsFile: () => new File(["raw"], "pasted.png", { type: "image/png" }),
      })),
      getData: () => text,
    },
  });
  await fireEvent(
    await screen.findByLabelText("Describe a brew or bean"),
    event,
  );
  return event;
}
async function readyPaste(text = "", count = 1) {
  await paste(text, count);
  await screen.findByAltText("Image 1 to send");
  await waitFor(() =>
    expect(screen.queryByText("Preparing images…")).toBeNull(),
  );
}
beforeEach(() => {
  installWebLocks();
  localStorage.clear();
  sessionStorage.clear();
  clearHomeImages();
  vi.clearAllMocks();
  vi.mocked(goto).mockResolvedValue();
  vi.mocked(prepareImage).mockResolvedValue(image);
  requests = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
      if (String(url) === "/api/data")
        return response({ beans: [], brewers: [], grinders: [], roasters: [] });
      requests.push(JSON.parse(String(init?.body)));
      return response({
        reply: "Image received",
        kind: "brew",
        patch: {},
        choices: [],
      });
    }),
  );
});
afterEach(() => {
  cleanup();
  clearHomeImages();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  vi.mocked(prepareImage).mockReset();
});
describe("homepage image paste", () => {
  it("sends image-only paste through the memory handoff and retains pixels for chat follow-ups", async () => {
    const home = render(HomeEntry, { accountDid: did });
    await readyPaste();
    expect(screen.queryByRole("button", { name: "Attach images" })).toBeNull();
    expect(
      JSON.stringify({ ...localStorage, ...sessionStorage }),
    ).not.toContain(image.image.data);
    await userEvent.click(
      screen.getByRole("button", { name: "Send", exact: true }),
    );
    expect(goto).toHaveBeenCalledExactlyOnceWith("/chat");
    expect(sessionStorage.length).toBe(0);
    home.unmount();
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await screen.findByText("Image received");
    expect(requests[0].messages).toEqual([
      { role: "user", content: "", images: [image.image] },
    ]);
    expect(screen.getByAltText("Attached image 1")).toBeVisible();
    await userEvent.type(screen.getByLabelText("Message"), "What origin?");
    await userEvent.click(
      screen.getByRole("button", { name: "Send", exact: true }),
    );
    await waitFor(() => expect(requests).toHaveLength(2));
    expect(requests[1].messages[0].images).toEqual([image.image]);
    expect(
      JSON.stringify({ ...localStorage, ...sessionStorage }),
    ).not.toContain(image.image.data);
  });
  it("keeps image handoff independent of storage availability", async () => {
    const home = render(HomeEntry, { accountDid: did });
    await readyPaste("Bag label");
    const spy = vi
      .spyOn(Storage.prototype, "getItem")
      .mockImplementation(() => {
        throw Error("denied");
      });
    await userEvent.click(
      screen.getByRole("button", { name: "Send", exact: true }),
    );
    home.unmount();
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await screen.findByText("Image received");
    expect(requests[0].messages[0]).toMatchObject({
      content: "Bag label",
      images: [image.image],
    });
    spy.mockRestore();
  });
  it("opens a fresh draft instead of appending images to an existing conversation", async () => {
    const old = new DraftEngine(createSession(did, "old", "bean"));
    old.edit({ name: "Keep me" });
    expect(saveSession(localStorage, did, old.snapshot()).ok).toBe(true);
    const home = render(HomeEntry, { accountDid: did });
    await readyPaste("New bag");
    await userEvent.click(
      screen.getByRole("button", { name: "Send", exact: true }),
    );
    home.unmount();
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await screen.findByText("Image received");
    expect(requests[0].draft.fields).toEqual({});
    expect(loadSession(localStorage, did, "old")).toEqual({
      ok: true,
      value: { ...old.snapshot(), composerText: "" },
    });
  });
  it("preserves mixed text/image paste and supports removing images", async () => {
    render(HomeEntry, { accountDid: did });
    await fireEvent.input(screen.getByLabelText("Describe a brew or bean"), {
      target: { value: "before replace after" },
    });
    (
      screen.getByLabelText("Describe a brew or bean") as HTMLTextAreaElement
    ).setSelectionRange(7, 14);
    await readyPaste("coffee");
    expect(screen.getByLabelText("Describe a brew or bean")).toHaveValue(
      "before coffee after",
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Remove image 1" }),
    );
    expect(screen.queryByAltText("Image 1 to send")).toBeNull();
    await fireEvent.input(screen.getByLabelText("Describe a brew or bean"), {
      target: { value: "" },
    });
    expect(
      screen.getByRole("button", { name: "Send", exact: true }),
    ).toBeDisabled();
  });
  it("keeps native text paste untouched", async () => {
    render(HomeEntry, { accountDid: did });
    const event = await paste("ordinary text", 0);
    expect(event.defaultPrevented).toBe(false);
    expect(prepareImage).not.toHaveBeenCalled();
  });
  it("keeps images and text for retry after failed navigation", async () => {
    vi.mocked(goto).mockRejectedValueOnce(Error("failed"));
    render(HomeEntry, { accountDid: did });
    await readyPaste("Label");
    await userEvent.click(
      screen.getByRole("button", { name: "Send", exact: true }),
    );
    await screen.findByRole("alert");
    expect(screen.getByAltText("Image 1 to send")).toBeVisible();
    expect(screen.getByLabelText("Describe a brew or bean")).toHaveValue(
      "Label",
    );
    expect(takeHomeImages(did)).toBeUndefined();
    await userEvent.click(
      screen.getByRole("button", { name: "Send", exact: true }),
    );
    expect(takeHomeImages(did)?.images).toEqual([image]);
  });
  it("rejects excess or invalid paste without losing prior images", async () => {
    render(HomeEntry, { accountDid: did });
    await readyPaste();
    await paste("", 4);
    expect(await screen.findByRole("alert")).toHaveTextContent("Up to 4");
    expect(prepareImage).toHaveBeenCalledTimes(1);
    vi.mocked(prepareImage).mockRejectedValueOnce(Error("Invalid image"));
    await paste();
    await screen.findByText("Invalid image");
    expect(screen.getByAltText("Image 1 to send")).toBeVisible();
  });
  it("ignores a late decode after account change", async () => {
    let finish!: (value: typeof image) => void;
    vi.mocked(prepareImage).mockReturnValueOnce(
      new Promise((resolve) => {
        finish = resolve;
      }),
    );
    const home = render(HomeEntry, { accountDid: did });
    await paste();
    await home.rerender({ accountDid: "did:plc:other" });
    await act(() => finish(image));
    expect(screen.queryByAltText("Image 1 to send")).toBeNull();
    expect(
      screen.getByRole("button", { name: "Send", exact: true }),
    ).toBeDisabled();
  });
  it("releases unsent images on pagehide", async () => {
    render(HomeEntry, { accountDid: did });
    await readyPaste();
    await fireEvent(window, new Event("pagehide"));
    expect(screen.queryByAltText("Image 1 to send")).toBeNull();
    expect(takeHomeImages(did)).toBeUndefined();
  });
});
