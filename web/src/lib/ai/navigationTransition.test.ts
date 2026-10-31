import { describe, expect, it, vi } from "vitest";
import { homeToChatTransition } from "./navigationTransition";

function deferred() {
  let resolve!: () => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<void>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}
function fixture(from = "/", to = "/chat") {
  const complete = deferred();
  const finished = deferred();
  let update!: () => Promise<void>;
  const doc = {
    documentElement: { classList: { add: vi.fn(), remove: vi.fn() } },
    startViewTransition: vi.fn((callback: () => Promise<void>) => {
      update = callback;
      return { finished: finished.promise };
    }),
  };
  const nav = {
    from: { url: new URL(from, "https://arabica.test") },
    to: { url: new URL(to, "https://arabica.test") },
    complete: complete.promise,
  };
  return { doc, nav, complete, finished, update: () => update() };
}

describe("Home to chat transition", () => {
  it("releases navigation before waiting for its completion", async () => {
    const f = fixture();
    const ready = homeToChatTransition(f.nav, f.doc, false);
    expect(f.doc.startViewTransition).toHaveBeenCalledOnce();
    expect(f.doc.documentElement.classList.add).toHaveBeenCalledWith(
      "ai-chat-transition",
    );
    const updating = f.update();
    await ready;
    let updated = false;
    void updating.then(() => {
      updated = true;
    });
    await Promise.resolve();
    expect(updated).toBe(false);
    f.complete.resolve();
    await updating;
    f.finished.resolve();
    await f.finished.promise;
    expect(f.doc.documentElement.classList.remove).toHaveBeenCalledWith(
      "ai-chat-transition",
    );
  });

  it.each([
    ["/chat", "/"],
    ["/coffee", "/chat"],
    ["/", "/coffee"],
    ["/", "https://elsewhere.test/chat"],
  ])("does not animate %s → %s", (from, to) => {
    const f = fixture(from, to);
    expect(homeToChatTransition(f.nav, f.doc, false)).toBeUndefined();
    expect(f.doc.startViewTransition).not.toHaveBeenCalled();
  });

  it("respects reduced motion", () => {
    const f = fixture();
    expect(homeToChatTransition(f.nav, f.doc, true)).toBeUndefined();
    expect(f.doc.startViewTransition).not.toHaveBeenCalled();
  });

  it("lets unsupported browsers navigate normally", () => {
    const f = fixture();
    expect(
      homeToChatTransition(
        f.nav,
        { documentElement: f.doc.documentElement },
        false,
      ),
    ).toBeUndefined();
  });

  it("does not animate an initial navigation without a source", () => {
    const f = fixture();
    expect(
      homeToChatTransition({ ...f.nav, from: null }, f.doc, false),
    ).toBeUndefined();
  });

  it("releases navigation if starting a transition throws synchronously", async () => {
    const f = fixture();
    f.doc.startViewTransition.mockImplementation(() => {
      throw new Error("unavailable");
    });
    await homeToChatTransition(f.nav, f.doc, false);
    expect(f.doc.documentElement.classList.remove).toHaveBeenCalled();
  });

  it("releases navigation if the browser rejects before calling update", async () => {
    const f = fixture();
    const ready = homeToChatTransition(f.nav, f.doc, false);
    f.finished.reject(new Error("skipped"));
    await ready;
    expect(f.doc.documentElement.classList.remove).toHaveBeenCalled();
  });

  it("consumes failed navigation and skipped-transition rejections", async () => {
    const f = fixture();
    const ready = deferred();
    const updateDone = deferred();
    f.doc.startViewTransition.mockImplementation((callback) => {
      void callback();
      return {
        finished: f.finished.promise,
        ready: ready.promise,
        updateCallbackDone: updateDone.promise,
      };
    });
    await homeToChatTransition(f.nav, f.doc, false);
    f.complete.reject(new Error("cancelled navigation"));
    ready.reject(new Error("skipped animation"));
    updateDone.reject(new Error("cancelled update"));
    f.finished.reject(new Error("cancelled transition"));
    await Promise.resolve();
    await Promise.resolve();
    expect(f.doc.documentElement.classList.remove).toHaveBeenCalled();
  });
});
