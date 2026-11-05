import { describe, expect, it, vi } from "vitest";
import { load } from "../../src/routes/chat/+page";

const event = (fetcher: typeof fetch) =>
  ({ fetch: fetcher }) as Parameters<typeof load>[0];
const json = (data: unknown, status = 200) =>
  new Response(JSON.stringify(data), {
    status,
    headers: { "Content-Type": "application/json" },
  });

describe("chat client navigation access", () => {
  it("allows authorized status", async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValue(json({ enabled: true, allowed: true }));
    await expect(load(event(fetcher))).resolves.toEqual({ enabled: true });
    expect(fetcher.mock.calls[0][0]).toBe("/api/ai/status");
  });
  it.each([
    json({ enabled: false, allowed: false }),
    json({ enabled: true, allowed: true }, 403),
    json({}),
    json({ enabled: true }),
    json({ enabled: true, allowed: "true" }),
  ])("throws SvelteKit 404 on denial or invalid status", async (response) => {
    const fetcher = vi.fn().mockResolvedValue(response);
    await expect(load(event(fetcher))).rejects.toMatchObject({
      status: 404,
      body: { message: "Not found" },
    });
  });
  it("keeps local drafts accessible for allowed users when the provider is disabled", async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValue(json({ enabled: false, allowed: true }));
    await expect(load(event(fetcher))).resolves.toEqual({ enabled: false });
  });
  it("fails closed on network failure", async () => {
    const fetcher = vi.fn().mockRejectedValue(new Error("offline"));
    await expect(load(event(fetcher))).rejects.toMatchObject({ status: 404 });
  });
});
