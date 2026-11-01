import { afterEach, describe, expect, it, vi } from "vitest";
import {
  cancelHomeImages,
  clearHomeImages,
  stageHomeImages,
  takeHomeImages,
} from "./homeImages";
const value = {
  text: "Label",
  images: [
    {
      image: { mime_type: "image/jpeg" as const, data: "PRIVATE_PIXELS" },
      preview: "data:image/jpeg;base64,PRIVATE_PIXELS",
    },
  ],
};
afterEach(() => {
  clearHomeImages();
  vi.useRealTimers();
});
describe("memory-only homepage image handoff", () => {
  it("is single-use and account-scoped", () => {
    stageHomeImages("alice", value);
    expect(takeHomeImages("alice")).toBe(value);
    expect(takeHomeImages("alice")).toBeUndefined();
    stageHomeImages("alice", value);
    expect(takeHomeImages("bob")).toBeUndefined();
    expect(takeHomeImages("alice")).toBeUndefined();
  });
  it("stale cancellation cannot erase a newer handoff", () => {
    const old = stageHomeImages("alice", value);
    stageHomeImages("bob", value);
    cancelHomeImages(old);
    expect(takeHomeImages("bob")).toBe(value);
  });
  it("expires abandoned image bytes after 30 seconds", () => {
    vi.useFakeTimers();
    stageHomeImages("alice", value);
    vi.advanceTimersByTime(30_000);
    expect(takeHomeImages("alice")).toBeUndefined();
  });
});
