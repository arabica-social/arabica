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
async function choose(count = 1) {
  const picker = screen.getByLabelText("Choose images");
  await fireEvent.change(picker, {
    target: {
      files: Array.from(
        { length: count },
        (_, i) => new File(["raw"], `coffee-${i}.png`, { type: "image/png" }),
      ),
    },
  });
}
async function readyChoose(count = 1) {
  await choose(count);
  await screen.findByAltText("Image 1 to send");
  await waitFor(() =>
    expect(screen.queryByText("Preparing images…")).toBeNull(),
  );
}
describe("homepage images", () => {
  it("opens the native multiple-image picker from Add images", async () => {
    render(HomeEntry, { accountDid: did });
    const picker = screen.getByLabelText("Choose images");
    expect(picker).toHaveAttribute("multiple");
    expect(picker).toHaveAttribute("accept", "image/jpeg,image/png,image/webp");
    const click = vi.spyOn(picker, "click");
    await userEvent.click(screen.getByRole("button", { name: "Add images" }));
    expect(click).toHaveBeenCalledOnce();
  });
  it("hands multiple picked images to a fresh chat without storing pixels", async () => {
    const home = render(HomeEntry, { accountDid: did });
    await readyChoose(2);
    expect(screen.getByLabelText("Choose images")).toHaveValue("");
    await userEvent.click(
      screen.getByRole("button", { name: "Send", exact: true }),
    );
    expect(goto).toHaveBeenCalledExactlyOnceWith("/chat");
    home.unmount();
    render(ChatWorkspace, { accountDid: did, enabled: true });
    await screen.findByText("Image received");
    expect(requests[0].messages[0]).toMatchObject({
      content: "",
      images: [image.image, image.image],
    });
    expect(
      JSON.stringify({ ...localStorage, ...sessionStorage }),
    ).not.toContain(image.image.data);
  });
  it("supports choosing the same file again after removal and cancelling the picker", async () => {
    render(HomeEntry, { accountDid: did });
    await readyChoose();
    await choose(0);
    expect(prepareImage).toHaveBeenCalledTimes(1);
    expect(screen.getByAltText("Image 1 to send")).toBeVisible();
    await userEvent.click(
      screen.getByRole("button", { name: "Remove image 1" }),
    );
    await readyChoose();
    expect(prepareImage).toHaveBeenCalledTimes(2);
  });
  it("sends image-only paste through the memory handoff and retains pixels for chat follow-ups", async () => {
    const home = render(HomeEntry, { accountDid: did });
    await readyPaste();
    expect(screen.getByRole("button", { name: "Add images" })).toBeEnabled();
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
  it.each(["paste", "picker"])(
    "keeps %s images and text for retry after failed navigation",
    async (source) => {
      vi.mocked(goto).mockRejectedValueOnce(Error("failed"));
      render(HomeEntry, { accountDid: did });
      if (source === "paste") await readyPaste("Label");
      else {
        await fireEvent.input(
          screen.getByLabelText("Describe a brew or bean"),
          {
            target: { value: "Label" },
          },
        );
        await readyChoose();
      }
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
    },
  );
  it.each(["paste", "picker"])(
    "rejects excess or invalid %s without losing prior images",
    async (source) => {
      render(HomeEntry, { accountDid: did });
      await readyPaste();
      if (source === "paste") await paste("", 4);
      else await choose(4);
      expect(await screen.findByRole("alert")).toHaveTextContent("Up to 4");
      expect(prepareImage).toHaveBeenCalledTimes(1);
      vi.mocked(prepareImage).mockRejectedValueOnce(Error("Invalid image"));
      if (source === "paste") await paste();
      else await choose();
      await screen.findByText("Invalid image");
      expect(screen.getByAltText("Image 1 to send")).toBeVisible();
    },
  );
  it.each(["paste", "picker"])(
    "ignores a late %s decode after account change",
    async (source) => {
      let finish!: (value: typeof image) => void;
      vi.mocked(prepareImage).mockReturnValueOnce(
        new Promise((resolve) => {
          finish = resolve;
        }),
      );
      const home = render(HomeEntry, { accountDid: did });
      if (source === "paste") await paste();
      else await choose();
      expect(screen.getByRole("button", { name: "Add images" })).toBeDisabled();
      expect(
        screen.getByRole("button", { name: "Send", exact: true }),
      ).toBeDisabled();
      await home.rerender({ accountDid: "did:plc:other" });
      await act(() => finish(image));
      expect(screen.queryByAltText("Image 1 to send")).toBeNull();
      expect(
        screen.getByRole("button", { name: "Send", exact: true }),
      ).toBeDisabled();
    },
  );
  it("releases unsent images on pagehide", async () => {
    render(HomeEntry, { accountDid: did });
    await readyPaste();
    await fireEvent(window, new Event("pagehide"));
    expect(screen.queryByAltText("Image 1 to send")).toBeNull();
    expect(takeHomeImages(did)).toBeUndefined();
  });
});
