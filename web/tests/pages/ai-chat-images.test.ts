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
import ChatWorkspace from "../../src/lib/components/ai/ChatWorkspace.svelte";
import { createSession, DraftEngine } from "../../src/lib/ai/draft";
import { prepareImage } from "../../src/lib/ai/images";
import {
  loadSession,
  saveSession,
  storageKey,
} from "../../src/lib/ai/persistence";
import { installWebLocks } from "../helpers/webLocks";
vi.mock("../../src/lib/ai/images", () => ({
  prepareImage: vi.fn(),
  MAX_IMAGES: 4,
}));
vi.mock("../../src/lib/stores/appCache", () => ({
  appCache: { invalidateCache: vi.fn() },
}));
const did = "did:plc:image-test";
const image = {
  image: {
    mime_type: "image/jpeg" as const,
    data: "TRANSIENT_ONLY_IMAGE_BYTES",
  },
  preview: "data:image/jpeg;base64,TRANSIENT_ONLY_IMAGE_BYTES",
};
const reply = {
  reply: "What else is on the label?",
  kind: "brew",
  patch: {},
  choices: [],
};
const response = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
let requests: {
  messages: {
    role: string;
    content: string;
    images?: unknown[];
    images_unavailable?: number;
  }[];
}[];
let rejectNext = false;
let currentReply = reply;
function open() {
  return render(ChatWorkspace, { accountDid: did, enabled: true });
}
async function attach(count = 1) {
  await fireEvent.change(await screen.findByLabelText("Choose images"), {
    target: {
      files: Array.from(
        { length: count },
        () => new File(["raw"], "label.png", { type: "image/png" }),
      ),
    },
  });
  await waitFor(() =>
    expect(screen.queryByText("Preparing images…")).toBeNull(),
  );
}
async function send() {
  await userEvent.click(
    screen.getByRole("button", { name: "Send", exact: true }),
  );
  await waitFor(() =>
    expect(
      screen.queryByRole("button", { name: "Stop", exact: true }),
    ).toBeNull(),
  );
}
beforeEach(() => {
  installWebLocks();
  localStorage.clear();
  sessionStorage.clear();
  requests = [];
  rejectNext = false;
  currentReply = reply;
  vi.mocked(prepareImage).mockResolvedValue(image);
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
      if (String(url) === "/api/data")
        return response({ beans: [], brewers: [], grinders: [], roasters: [] });
      if (String(url) === "/api/beans")
        return response({ error: "Name rejected" }, 400);
      requests.push(JSON.parse(String(init?.body)));
      if (rejectNext) {
        rejectNext = false;
        return response({ error: "Try again" }, 400);
      }
      return response(currentReply);
    }),
  );
});
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  vi.mocked(prepareImage).mockReset();
});
describe("transient chat images", () => {
  it("sends image-only input, retains images for follow-ups, and persists only metadata", async () => {
    open();
    await attach();
    expect(screen.getByAltText("Image 1 to send")).toHaveAttribute(
      "src",
      image.preview,
    );
    expect(localStorage.getItem(storageKey(did))).not.toContain(
      image.image.data,
    );
    await send();
    expect(requests[0].messages).toEqual([
      { role: "user", content: "", images: [image.image] },
    ]);
    expect(screen.getByAltText("Attached image 1")).toBeVisible();
    await userEvent.type(screen.getByLabelText("Message"), "What origin?");
    await send();
    expect(requests[1].messages[0].images).toEqual([image.image]);
    const stored = localStorage.getItem(storageKey(did))!;
    expect(stored).toContain('"imageCount":1');
    expect(stored).not.toContain(image.image.data);
    expect(sessionStorage.length).toBe(0);
  });
  it("retains chat images when a record save is rejected without changing conversation", async () => {
    const e = new DraftEngine(createSession(did, "bean-chat", "bean"));
    e.edit({ name: "Ethiopia", origin: "Gedeb" });
    expect(saveSession(localStorage, did, e.snapshot()).ok).toBe(true);
    currentReply = { ...reply, kind: "bean" };
    open();
    await attach();
    await send();
    await userEvent.click(
      screen.getByRole("button", { name: "Save bean", exact: true }),
    );
    await screen.findByText("Name rejected");
    expect(screen.getByAltText("Attached image 1")).toBeVisible();
    await userEvent.type(screen.getByLabelText("Message"), "Correct that name");
    await send();
    expect(requests[1].messages[0].images).toEqual([image.image]);
  });
  it("removes unsent images and restores disabled empty send", async () => {
    open();
    await attach();
    await userEvent.click(
      screen.getByRole("button", { name: "Remove image 1" }),
    );
    expect(screen.queryByAltText("Image 1 to send")).toBeNull();
    expect(
      screen.getByRole("button", { name: "Send", exact: true }),
    ).toBeDisabled();
    expect(requests).toHaveLength(0);
  });
  it("retries exactly the same image message without duplicating it", async () => {
    rejectNext = true;
    open();
    await attach();
    await send();
    await userEvent.click(
      screen.getByRole("button", { name: "Retry", exact: true }),
    );
    await waitFor(() => expect(requests).toHaveLength(2));
    expect(requests[1].messages).toEqual(requests[0].messages);
    expect(requests[1].messages).toHaveLength(1);
  });
  it("does not pretend restored images are available and requires reattachment for Retry", async () => {
    rejectNext = true;
    const view = open();
    await attach();
    await send();
    view.unmount();
    open();
    await screen.findByText(/Image no longer available/);
    await userEvent.click(
      screen.getByRole("button", { name: "Retry", exact: true }),
    );
    expect(
      await screen.findByText(
        /Images from that message are no longer available/,
      ),
    ).toBeVisible();
    expect(requests).toHaveLength(1);
    await attach();
    await send();
    expect(requests[1].messages[0]).toEqual({
      role: "user",
      content: "",
      images_unavailable: 1,
    });
    expect(requests[1].messages[1].images).toEqual([image.image]);
  });
  it("pastes images and text together but leaves ordinary text paste to the browser", async () => {
    open();
    const box = await screen.findByLabelText("Message");
    const plain = new Event("paste", { bubbles: true, cancelable: true });
    Object.defineProperty(plain, "clipboardData", {
      value: { items: [], getData: () => "plain" },
    });
    await fireEvent(box, plain);
    expect(plain.defaultPrevented).toBe(false);
    const mixed = new Event("paste", { bubbles: true, cancelable: true });
    Object.defineProperty(mixed, "clipboardData", {
      value: {
        items: [
          {
            kind: "file",
            type: "image/png",
            getAsFile: () =>
              new File(["raw"], "paste.png", { type: "image/png" }),
          },
        ],
        getData: () => "Coffee label",
      },
    });
    await fireEvent(box, mixed);
    await screen.findByAltText("Image 1 to send");
    expect(mixed.defaultPrevented).toBe(true);
    expect(box).toHaveValue("Coffee label");
    await send();
    expect(requests[0].messages[0]).toMatchObject({
      content: "Coffee label",
      images: [image.image],
    });
  });
  it("keeps existing attachments when a batch is invalid or exceeds the cap", async () => {
    open();
    await attach();
    await attach(4);
    expect(await screen.findByText(/Up to 4 images/)).toBeVisible();
    expect(prepareImage).toHaveBeenCalledTimes(1);
    vi.mocked(prepareImage).mockRejectedValueOnce(
      new Error("Choose a JPEG, PNG, or WebP image."),
    );
    await attach();
    expect(screen.getByRole("alert")).toHaveTextContent("Choose a JPEG");
    expect(screen.getAllByAltText(/Image .* to send/)).toHaveLength(1);
  });
  it("drops delayed preprocessing when a new chat is activated", async () => {
    let finish!: (result: typeof image) => void;
    vi.mocked(prepareImage).mockReturnValueOnce(
      new Promise((resolve) => {
        finish = resolve;
      }),
    );
    open();
    await screen.findByLabelText("Message");
    await fireEvent.change(screen.getByLabelText("Choose images"), {
      target: {
        files: [new File(["raw"], "label.png", { type: "image/png" })],
      },
    });
    await screen.findByText("Preparing images…");
    await userEvent.click(
      screen.getByRole("button", { name: "New chat", exact: true }),
    );
    await act(() => finish(image));
    expect(screen.queryByAltText("Image 1 to send")).toBeNull();
    expect(
      screen.getByRole("button", { name: "Send", exact: true }),
    ).toBeDisabled();
  });
  it("clears sent image data on cross-tab reconciliation", async () => {
    open();
    await attach();
    await send();
    const loaded = loadSession(localStorage, did);
    if (!loaded.ok || !loaded.value) throw Error("fixture");
    const newer = { ...loaded.value, composerText: "Another tab" };
    expect(
      saveSession(localStorage, did, newer, { expected: loaded.value }).ok,
    ).toBe(true);
    await fireEvent(
      window,
      new StorageEvent("storage", { key: storageKey(did) }),
    );
    await screen.findByText(/Image no longer available/);
    expect(screen.queryByAltText("Attached image 1")).toBeNull();
    await userEvent.type(screen.getByLabelText("Message"), " follow-up");
    await send();
    expect(requests[1].messages[0]).toEqual({
      role: "user",
      content: "",
      images_unavailable: 1,
    });
  });
  it("clears pending images and ignores late preparation on account changes", async () => {
    const view = open();
    await attach();
    await view.rerender({ accountDid: "did:plc:other", enabled: true });
    await waitFor(() =>
      expect(screen.queryByAltText("Image 1 to send")).toBeNull(),
    );
    expect(
      screen.getByRole("button", { name: "Send", exact: true }),
    ).toBeDisabled();
    expect(localStorage.getItem(storageKey("did:plc:other"))).not.toContain(
      image.image.data,
    );
  });
  it("does not attach old-account pixels when decoding finishes after an account switch", async () => {
    let finish!: (result: typeof image) => void;
    vi.mocked(prepareImage).mockReturnValueOnce(
      new Promise((resolve) => {
        finish = resolve;
      }),
    );
    const view = open();
    await screen.findByLabelText("Message");
    await fireEvent.change(screen.getByLabelText("Choose images"), {
      target: {
        files: [new File(["raw"], "old-account.png", { type: "image/png" })],
      },
    });
    await screen.findByText("Preparing images…");
    await view.rerender({ accountDid: "did:plc:new-account", enabled: true });
    await act(() => finish(image));
    expect(screen.queryByAltText("Image 1 to send")).toBeNull();
    expect(
      screen.getByRole("button", { name: "Send", exact: true }),
    ).toBeDisabled();
    expect(requests).toHaveLength(0);
  });
  it("clears images on pagehide even if the page is restored from the back-forward cache", async () => {
    open();
    await attach();
    await send();
    await fireEvent(window, new Event("pagehide"));
    await screen.findByText(/Image no longer available/);
    expect(screen.queryByAltText("Attached image 1")).toBeNull();
  });
});
