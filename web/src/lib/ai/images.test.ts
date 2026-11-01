import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  MAX_IMAGES,
  MAX_IMAGE_BYTES,
  MAX_IMAGE_EDGE,
  MAX_SOURCE_BYTES,
  MAX_SOURCE_PIXELS,
  prepareImage,
} from "./images";

function png(width = 800, height = 600): Uint8Array {
  const bytes = new Uint8Array(33);
  bytes.set([137, 80, 78, 71, 13, 10, 26, 10]);
  const view = new DataView(bytes.buffer);
  view.setUint32(8, 13);
  bytes.set([73, 72, 68, 82], 12);
  view.setUint32(16, width);
  view.setUint32(20, height);
  return bytes;
}
function file(bytes = png(), type = "image/png", size = bytes.length): File {
  return {
    type,
    size,
    arrayBuffer: vi.fn(async () => bytes.buffer),
  } as unknown as File;
}
function blob(size = 3): Blob {
  return {
    type: "image/jpeg",
    size,
    arrayBuffer: async () => new Uint8Array([1, 2, 3]).buffer,
  } as Blob;
}

describe("ephemeral chat image preparation", () => {
  let bitmap: {
    width: number;
    height: number;
    close: ReturnType<typeof vi.fn>;
  };
  let canvas: HTMLCanvasElement;
  let draw: ReturnType<typeof vi.fn>;
  let fill: ReturnType<typeof vi.fn>;
  let toBlob: ReturnType<typeof vi.fn>;
  beforeEach(() => {
    bitmap = { width: 800, height: 600, close: vi.fn() };
    vi.stubGlobal(
      "createImageBitmap",
      vi.fn(async () => bitmap),
    );
    draw = vi.fn();
    fill = vi.fn();
    toBlob = vi.fn((callback: BlobCallback) => callback(blob()));
    canvas = {
      width: 0,
      height: 0,
      getContext: vi.fn(() => ({ drawImage: draw, fillRect: fill })),
      toBlob,
    } as unknown as HTMLCanvasElement;
    vi.spyOn(document, "createElement").mockReturnValue(canvas);
  });
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("publishes the shared bounds", () => {
    expect([
      MAX_IMAGES,
      MAX_IMAGE_BYTES,
      MAX_IMAGE_EDGE,
      MAX_SOURCE_BYTES,
      MAX_SOURCE_PIXELS,
    ]).toEqual([4, 1048576, 2048, 20971520, 40000000]);
  });
  it.each(["image/svg+xml", "image/gif", "text/plain", ""])(
    "rejects unsupported type %s before reading",
    async (type) => {
      const source = file(png(), type);
      await expect(prepareImage(source)).rejects.toThrow(
        "Choose a JPEG, PNG, or WebP",
      );
      expect(source.arrayBuffer).not.toHaveBeenCalled();
      expect(createImageBitmap).not.toHaveBeenCalled();
    },
  );
  it("rejects empty and oversized sources before reading", async () => {
    for (const size of [0, MAX_SOURCE_BYTES + 1]) {
      const source = file(png(), "image/png", size);
      await expect(prepareImage(source)).rejects.toThrow();
      expect(source.arrayBuffer).not.toHaveBeenCalled();
    }
  });
  it.each([new Uint8Array(), new Uint8Array([1, 2, 3]), png(0, 1)])(
    "rejects invalid headers",
    async (bytes) => {
      await expect(prepareImage(file(bytes))).rejects.toThrow(
        "could not be read",
      );
      expect(createImageBitmap).not.toHaveBeenCalled();
    },
  );
  it("rejects excessive source dimensions before decoding", async () => {
    await expect(prepareImage(file(png(10000, 4001)))).rejects.toThrow(
      "40 megapixels",
    );
    expect(createImageBitmap).not.toHaveBeenCalled();
  });
  it("always reencodes even a small source and previews only sanitized bytes", async () => {
    const prepared = await prepareImage(file());
    expect(prepared).toEqual({
      image: { mime_type: "image/jpeg", data: "AQID" },
      preview: "data:image/jpeg;base64,AQID",
    });
    expect(draw).toHaveBeenCalledWith(bitmap, 0, 0, 800, 600);
    expect(fill).toHaveBeenCalledWith(0, 0, 800, 600);
    expect(toBlob).toHaveBeenCalledWith(
      expect.any(Function),
      "image/jpeg",
      0.9,
    );
    expect(bitmap.close).toHaveBeenCalledOnce();
    expect([canvas.width, canvas.height]).toEqual([0, 0]);
  });
  it("downscales without changing aspect ratio", async () => {
    bitmap.width = 4096;
    bitmap.height = 2048;
    await prepareImage(file(png(4096, 2048)));
    expect(draw).toHaveBeenCalledWith(bitmap, 0, 0, 2048, 1024);
  });
  it("checks decoded dimensions too and closes oversized decoded images", async () => {
    bitmap.width = 40000001;
    bitmap.height = 1;
    await expect(prepareImage(file())).rejects.toThrow("40 megapixels");
    expect(bitmap.close).toHaveBeenCalledOnce();
    expect(draw).not.toHaveBeenCalled();
  });
  it("reduces quality then dimensions until the output fits", async () => {
    toBlob.mockImplementation((callback: BlobCallback) =>
      callback(
        blob(
          toBlob.mock.calls.length <= 4 ? MAX_IMAGE_BYTES + 1 : MAX_IMAGE_BYTES,
        ),
      ),
    );
    await prepareImage(file());
    expect(toBlob).toHaveBeenCalledTimes(5);
    expect(draw.mock.calls.map((call) => call.slice(3))).toEqual([
      [800, 600],
      [600, 450],
    ]);
    expect(bitmap.close).toHaveBeenCalledOnce();
  });
  it("bounds compression attempts and releases resources when no encoding fits", async () => {
    toBlob.mockImplementation((callback: BlobCallback) =>
      callback(blob(MAX_IMAGE_BYTES + 1)),
    );
    await expect(prepareImage(file())).rejects.toThrow("below 1 MiB");
    expect(toBlob).toHaveBeenCalledTimes(16);
    expect(bitmap.close).toHaveBeenCalledOnce();
    expect(canvas.width).toBe(0);
  });
  it.each(["null", "wrong-type", "throw", "context"])(
    "cleans up after canvas failure: %s",
    async (failure) => {
      if (failure === "null")
        toBlob.mockImplementation((callback: BlobCallback) => callback(null));
      if (failure === "wrong-type")
        toBlob.mockImplementation((callback: BlobCallback) =>
          callback({ type: "image/png" } as Blob),
        );
      if (failure === "throw")
        draw.mockImplementation(() => {
          throw new Error("draw failed");
        });
      if (failure === "context")
        vi.mocked(canvas.getContext).mockReturnValue(null);
      await expect(prepareImage(file())).rejects.toThrow();
      expect(bitmap.close).toHaveBeenCalledOnce();
      expect([canvas.width, canvas.height]).toEqual([0, 0]);
    },
  );
  it("reports a safe decoder error", async () => {
    vi.mocked(createImageBitmap).mockRejectedValue(
      new Error("private decoder internals"),
    );
    await expect(prepareImage(file())).rejects.toThrow(
      "This image could not be read.",
    );
  });
  it("accepts JPEG dimensions behind metadata segments", async () => {
    const bytes = new Uint8Array([
      255, 216, 255, 225, 0, 4, 0, 0, 255, 192, 0, 8, 8, 2, 88, 3, 32, 0,
    ]);
    await prepareImage(file(bytes, "image/jpeg"));
    expect(createImageBitmap).toHaveBeenCalledOnce();
  });
  it("rejects malformed JPEG segments without decoding", async () => {
    await expect(
      prepareImage(
        file(new Uint8Array([255, 216, 255, 225, 255, 255]), "image/jpeg"),
      ),
    ).rejects.toThrow("could not be read");
    expect(createImageBitmap).not.toHaveBeenCalled();
  });
  it.each(["VP8X", "VP8 ", "VP8L"])(
    "reads %s WebP dimensions",
    async (chunk) => {
      const bytes = new Uint8Array(30);
      const view = new DataView(bytes.buffer);
      bytes.set([..."RIFF"].map((s) => s.charCodeAt(0)));
      bytes.set(
        [..."WEBP"].map((s) => s.charCodeAt(0)),
        8,
      );
      bytes.set(
        [...chunk].map((s) => s.charCodeAt(0)),
        12,
      );
      view.setUint32(4, 22, true);
      view.setUint32(16, chunk === "VP8L" ? 5 : 10, true);
      if (chunk === "VP8X") {
        bytes[24] = 99;
        bytes[27] = 99;
      }
      if (chunk === "VP8 ") {
        bytes.set([157, 1, 42], 23);
        view.setUint16(26, 100, true);
        view.setUint16(28, 100, true);
      }
      if (chunk === "VP8L") {
        bytes[20] = 47;
        view.setUint32(21, 99 | (99 << 14), true);
      }
      await prepareImage(file(bytes, "image/webp"));
      expect(createImageBitmap).toHaveBeenCalledOnce();
    },
  );
  it.each([true, false])(
    "clears fallback data URLs and event handlers on success=%s",
    async (success) => {
      vi.stubGlobal("createImageBitmap", undefined);
      let reader: {
        onload: (() => void) | null;
        onerror: (() => void) | null;
        onabort: (() => void) | null;
      };
      vi.stubGlobal(
        "FileReader",
        class {
          result = "data:image/png;base64,TEMPORARY_SOURCE";
          onload: (() => void) | null = null;
          onerror: (() => void) | null = null;
          onabort: (() => void) | null = null;
          constructor() {
            reader = this;
          }
          readAsDataURL() {
            queueMicrotask(() => this.onload?.());
          }
        },
      );
      let fallback: {
        onload: (() => void) | null;
        onerror: (() => void) | null;
        src: string;
      };
      vi.stubGlobal(
        "Image",
        class {
          naturalWidth = 800;
          naturalHeight = 600;
          onload: (() => void) | null = null;
          onerror: (() => void) | null = null;
          value = "";
          constructor() {
            fallback = this;
          }
          set src(value: string) {
            this.value = value;
            if (value)
              queueMicrotask(() =>
                success ? this.onload?.() : this.onerror?.(),
              );
          }
          get src() {
            return this.value;
          }
        },
      );
      if (success) await prepareImage(file());
      else
        await expect(prepareImage(file())).rejects.toThrow("could not be read");
      expect(reader!.onload).toBeNull();
      expect(reader!.onerror).toBeNull();
      expect(reader!.onabort).toBeNull();
      expect(fallback!.src).toBe("");
      expect(fallback!.onload).toBeNull();
      expect(fallback!.onerror).toBeNull();
    },
  );
});
