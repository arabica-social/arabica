import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import AIAccess from "../../src/lib/components/ai/AIAccess.svelte";

const entry = {
  did: "did:plc:alice",
  granted_by: "did:plc:mod",
  created_at: "2026-10-31T12:00:00Z",
};
const list = (entries = [entry]) => ({
  entries,
  moderators: [{ did: "did:plc:staff" }],
});
const json = (data: unknown, status = 200) =>
  new Response(JSON.stringify(data), {
    status,
    headers: { "Content-Type": "application/json" },
  });

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("AI access management", () => {
  it("loads explicit grants and implicit staff access", async () => {
    const fetcher = vi.fn().mockResolvedValue(json(list()));
    vi.stubGlobal("fetch", fetcher);
    render(AIAccess);
    expect(await screen.findByText(entry.did)).toBeInTheDocument();
    expect(screen.getByText("did:plc:staff")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Remove access for did:plc:staff" }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByText(/Admins and moderators have access automatically/),
    ).toBeInTheDocument();
    expect(fetcher.mock.calls[0][0]).toBe("/api/_mod/ai-access");
  });

  it("grants and revokes using JSON and refreshes authoritative entries", async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(json(list([])))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(json(list()))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(json(list([])));
    vi.stubGlobal("fetch", fetcher);
    const user = userEvent.setup();
    render(AIAccess);
    await screen.findByText("No explicit grants.");
    await user.type(screen.getByLabelText("Account DID"), ` ${entry.did} `);
    await user.click(screen.getByRole("button", { name: "Grant access" }));
    expect(await screen.findByText(entry.did)).toBeInTheDocument();
    expect(fetcher.mock.calls[1][1]).toMatchObject({
      method: "PUT",
      body: JSON.stringify({ did: entry.did }),
      credentials: "same-origin",
    });
    expect(
      new Headers(fetcher.mock.calls[1][1].headers).get("Content-Type"),
    ).toBe("application/json");
    expect(screen.getByLabelText("Account DID")).toHaveValue("");
    await user.click(
      screen.getByRole("button", { name: `Remove access for ${entry.did}` }),
    );
    await screen.findByText("Explicit grant removed.");
    expect(screen.queryByText(entry.did)).not.toBeInTheDocument();
    expect(fetcher.mock.calls[3][1]).toMatchObject({
      method: "DELETE",
      body: JSON.stringify({ did: entry.did }),
    });
  });

  it.each(["PUT", "DELETE"])(
    "preserves grants and input on failed %s without reporting success",
    async (method) => {
      const fetcher = vi
        .fn()
        .mockResolvedValueOnce(json(list()))
        .mockResolvedValueOnce(json({ error: "Permission denied" }, 403));
      vi.stubGlobal("fetch", fetcher);
      const user = userEvent.setup();
      render(AIAccess);
      await screen.findByText(entry.did);
      await user.type(screen.getByLabelText("Account DID"), "did:plc:new");
      await user.click(
        screen.getByRole("button", {
          name:
            method === "PUT"
              ? "Grant access"
              : `Remove access for ${entry.did}`,
        }),
      );
      expect(await screen.findByRole("alert")).toHaveTextContent(
        "Permission denied",
      );
      expect(screen.getByText(entry.did)).toBeInTheDocument();
      expect(screen.getByLabelText("Account DID")).toHaveValue("did:plc:new");
      expect(screen.getByRole("status").textContent).toBe("");
      expect(fetcher).toHaveBeenCalledTimes(2);
    },
  );

  it("distinguishes load failure from an empty list and allows retry", async () => {
    const fetcher = vi
      .fn()
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce(json(list([])));
    vi.stubGlobal("fetch", fetcher);
    const user = userEvent.setup();
    render(AIAccess);
    await screen.findByRole("alert");
    expect(screen.queryByText("No explicit grants.")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Grant access" }),
    ).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Reload access" }));
    expect(await screen.findByText("No explicit grants.")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("reports a successful write with failed refresh truthfully instead of showing stale grants", async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(json(list()))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockRejectedValueOnce(new Error("offline"));
    vi.stubGlobal("fetch", fetcher);
    const user = userEvent.setup();
    render(AIAccess);
    await screen.findByText(entry.did);
    await user.click(
      screen.getByRole("button", { name: `Remove access for ${entry.did}` }),
    );
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Access changed, but the list could not be refreshed.",
    );
    expect(screen.queryByText(entry.did)).not.toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByRole("status").textContent).toBe(""),
    );
  });

  it("prevents duplicate writes while a mutation is pending", async () => {
    let resolve!: (response: Response) => void;
    const pending = new Promise<Response>((done) => {
      resolve = done;
    });
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(json(list()))
      .mockReturnValueOnce(pending)
      .mockResolvedValueOnce(json(list([])));
    vi.stubGlobal("fetch", fetcher);
    const user = userEvent.setup();
    render(AIAccess);
    await screen.findByText(entry.did);
    const remove = screen.getByRole("button", {
      name: `Remove access for ${entry.did}`,
    });
    await user.click(remove);
    expect(remove).toBeDisabled();
    expect(
      screen.getByRole("button", { name: "Reload access" }),
    ).toBeDisabled();
    await user.click(remove);
    expect(fetcher).toHaveBeenCalledTimes(2);
    resolve(new Response(null, { status: 204 }));
    await screen.findByText("Explicit grant removed.");
  });
});
