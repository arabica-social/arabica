export type ChatImage = {
  mime_type: "image/jpeg" | "image/png" | "image/webp";
  data: string;
};
export type PreparedImage = { image: ChatImage; preview: string };

export const MAX_IMAGES = 4;
export const MAX_IMAGE_BYTES = 1024 * 1024;
export const MAX_IMAGE_EDGE = 2048;
export const MAX_SOURCE_BYTES = 20 * 1024 * 1024;
export const MAX_SOURCE_PIXELS = 40_000_000;

const INVALID =
  "This image could not be read. Choose a valid JPEG, PNG, or WebP image.";
const TOO_LARGE =
  "This image is too large. Choose an image under 20 MiB and 40 megapixels.";

type Dimensions = { width: number; height: number };

function checkDimensions({ width, height }: Dimensions): void {
  if (!width || !height) throw new Error(INVALID);
  if (width * height > MAX_SOURCE_PIXELS) throw new Error(TOO_LARGE);
}

// Read dimensions before asking the browser to allocate decoded pixel memory.
// The source buffer itself is capped at 20 MiB, including JPEG metadata segments.
function dimensions(bytes: Uint8Array, mime: string): Dimensions {
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
  const text = (start: number, length: number) =>
    String.fromCharCode(...bytes.subarray(start, start + length));
  if (
    mime === "image/png" &&
    bytes.length >= 33 &&
    bytes[0] === 137 &&
    text(1, 7) === "PNG\r\n\x1a\n" &&
    view.getUint32(8) === 13 &&
    text(12, 4) === "IHDR"
  ) {
    return { width: view.getUint32(16), height: view.getUint32(20) };
  }
  if (mime === "image/jpeg" && bytes[0] === 255 && bytes[1] === 216) {
    let offset = 2;
    while (offset + 3 < bytes.length) {
      if (bytes[offset++] !== 255) break;
      while (bytes[offset] === 255) offset++;
      const marker = bytes[offset++];
      if (marker === 217 || marker === 218) break;
      if (marker === 1 || (marker >= 208 && marker <= 215)) continue;
      if (offset + 2 > bytes.length) break;
      const length = view.getUint16(offset);
      if (length < 2 || offset + length > bytes.length) break;
      if (
        [
          192, 193, 194, 195, 197, 198, 199, 201, 202, 203, 205, 206, 207,
        ].includes(marker)
      ) {
        if (length < 8) break;
        return {
          height: view.getUint16(offset + 3),
          width: view.getUint16(offset + 5),
        };
      }
      offset += length;
    }
  }
  if (
    mime === "image/webp" &&
    bytes.length >= 30 &&
    text(0, 4) === "RIFF" &&
    text(8, 4) === "WEBP" &&
    view.getUint32(4, true) + 8 === bytes.length
  ) {
    const chunk = text(12, 4);
    const length = view.getUint32(16, true);
    if (length + 20 <= bytes.length) {
      if (chunk === "VP8X" && length === 10) {
        const uint24 = (offset: number) =>
          bytes[offset] + bytes[offset + 1] * 256 + bytes[offset + 2] * 65536;
        return { width: uint24(24) + 1, height: uint24(27) + 1 };
      }
      if (
        chunk === "VP8 " &&
        length >= 10 &&
        bytes[23] === 157 &&
        bytes[24] === 1 &&
        bytes[25] === 42
      ) {
        return {
          width: view.getUint16(26, true) & 16383,
          height: view.getUint16(28, true) & 16383,
        };
      }
      if (chunk === "VP8L" && length >= 5 && bytes[20] === 47) {
        const bits = view.getUint32(21, true);
        return {
          width: (bits & 16383) + 1,
          height: ((bits >>> 14) & 16383) + 1,
        };
      }
    }
  }
  throw new Error(INVALID);
}

async function decode(file: File): Promise<{
  source: CanvasImageSource;
  width: number;
  height: number;
  release: () => void;
}> {
  if (typeof createImageBitmap === "function") {
    const bitmap = await createImageBitmap(file);
    return {
      source: bitmap,
      width: bitmap.width,
      height: bitmap.height,
      release: () => bitmap.close(),
    };
  }
  // data: images are already permitted by Arabica's CSP; blob: is deliberately
  // not. Keep the source only in memory and clear it as soon as reencoding ends.
  const url = await new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    const finish = () => {
      reader.onload = null;
      reader.onerror = null;
      reader.onabort = null;
    };
    reader.onload = () => {
      const result = reader.result;
      finish();
      if (typeof result === "string") resolve(result);
      else reject(new Error(INVALID));
    };
    reader.onerror = reader.onabort = () => {
      finish();
      reject(new Error(INVALID));
    };
    reader.readAsDataURL(file);
  });
  const image = new Image();
  try {
    await new Promise<void>((resolve, reject) => {
      image.onload = () => resolve();
      image.onerror = () => reject(new Error(INVALID));
      image.src = url;
    });
    return {
      source: image,
      width: image.naturalWidth,
      height: image.naturalHeight,
      release: () => {
        image.src = "";
      },
    };
  } catch (error) {
    image.src = "";

    throw error;
  } finally {
    image.onload = null;
    image.onerror = null;
  }
}

function encode(canvas: HTMLCanvasElement, quality: number): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (!blob || blob.type !== "image/jpeg")
          reject(new Error("Your browser could not prepare this image."));
        else resolve(blob);
      },
      "image/jpeg",
      quality,
    );
  });
}

/** Ephemeral, metadata-free image data. Never writes storage or sends a request. */
export async function prepareImage(file: File): Promise<PreparedImage> {
  if (!["image/jpeg", "image/png", "image/webp"].includes(file.type)) {
    throw new Error("Choose a JPEG, PNG, or WebP image.");
  }
  if (file.size > MAX_SOURCE_BYTES) throw new Error(TOO_LARGE);
  if (!file.size) throw new Error(INVALID);
  const header = dimensions(
    new Uint8Array(await file.arrayBuffer()),
    file.type,
  );
  checkDimensions(header);
  let decoded: Awaited<ReturnType<typeof decode>>;
  try {
    decoded = await decode(file);
  } catch {
    throw new Error(INVALID);
  }
  let canvas: HTMLCanvasElement | undefined;
  try {
    checkDimensions(decoded);
    canvas = document.createElement("canvas");
    const scale = Math.min(
      1,
      MAX_IMAGE_EDGE / Math.max(decoded.width, decoded.height),
    );
    let width = Math.max(1, Math.round(decoded.width * scale));
    let height = Math.max(1, Math.round(decoded.height * scale));
    const context = canvas.getContext("2d");
    if (!context) throw new Error("Your browser could not prepare this image.");
    // Four dimension levels, each with four quality attempts: strictly bounded work.
    for (let level = 0; level < 4; level++) {
      canvas.width = width;
      canvas.height = height;
      context.fillStyle = "#fff";
      context.fillRect(0, 0, width, height);
      context.drawImage(decoded.source, 0, 0, width, height);
      for (const quality of [0.9, 0.75, 0.6, 0.45]) {
        const blob = await encode(canvas, quality);
        if (blob.size > MAX_IMAGE_BYTES) continue;
        const bytes = new Uint8Array(await blob.arrayBuffer());
        let binary = "";
        for (let offset = 0; offset < bytes.length; offset += 8192) {
          binary += String.fromCharCode(
            ...bytes.subarray(offset, offset + 8192),
          );
        }
        const data = btoa(binary);
        return {
          image: { mime_type: "image/jpeg", data },
          preview: `data:image/jpeg;base64,${data}`,
        };
      }
      width = Math.max(1, Math.floor(width * 0.75));
      height = Math.max(1, Math.floor(height * 0.75));
    }
    throw new Error(
      "This image could not be reduced below 1 MiB. Choose a smaller image.",
    );
  } finally {
    decoded.release();
    if (canvas) {
      canvas.width = 0;
      canvas.height = 0;
    }
  }
}
