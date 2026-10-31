import { afterEach, expect, it, vi } from "vitest";
import { autosizeTextarea } from "./autosizeTextarea";
afterEach(() => vi.unstubAllGlobals());
function textarea() {
  const node = document.createElement("textarea");
  node.style.borderTopWidth = "1px";
  node.style.borderBottomWidth = "1px";
  let content = 40;
  Object.defineProperty(node, "scrollHeight", { get: () => content });
  Object.defineProperty(node, "clientHeight", {
    get: () => Math.min(content, 100),
  });
  return {
    node,
    content: (next: number) => {
      content = next;
    },
  };
}
it("fits initial content, grows for wrapping, and shrinks after a controlled clear", async () => {
  vi.stubGlobal("ResizeObserver", undefined);
  const area = textarea();
  const action = autosizeTextarea(area.node, "start");
  await Promise.resolve();
  expect(area.node.style.height).toBe("42px");
  expect(area.node.style.overflowY).toBe("hidden");
  area.content(140);
  action.update();
  expect(area.node.style.height).toBe("142px");
  expect(area.node.style.overflowY).toBe("auto");
  area.content(20);
  action.update();
  expect(area.node.style.height).toBe("22px");
  expect(area.node.style.overflowY).toBe("hidden");
  action.destroy();
});
it("reflows on width changes but ignores its own height-only resize notifications", async () => {
  let notify!: () => void;
  const disconnect = vi.fn();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      constructor(callback: () => void) {
        notify = callback;
      }
      observe = vi.fn();
      disconnect = disconnect;
    },
  );
  const area = textarea();
  let width = 400;
  area.node.getBoundingClientRect = () => ({ width }) as DOMRect;
  const action = autosizeTextarea(area.node, "start");
  await Promise.resolve();
  area.content(70);
  notify();
  expect(area.node.style.height).toBe("42px");
  width = 200;
  notify();
  expect(area.node.style.height).toBe("72px");
  action.destroy();
  expect(disconnect).toHaveBeenCalledOnce();
  width = 100;
  area.content(90);
  notify();
  expect(area.node.style.height).toBe("72px");
});
it("removes fallback listeners and ignores queued work after unmount", async () => {
  vi.stubGlobal("ResizeObserver", undefined);
  const area = textarea();
  const action = autosizeTextarea(area.node, "start");
  action.destroy();
  await Promise.resolve();
  window.dispatchEvent(new Event("resize"));
  expect(area.node.style.height).toBe("");
});
