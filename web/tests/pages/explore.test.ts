import {
  cleanup,
  render,
  screen,
  fireEvent,
  waitFor,
} from "@testing-library/svelte";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import Explore from "../../src/routes/explore/+page.svelte";
import { goto } from "$app/navigation";

vi.mock("$app/navigation", () => ({
  goto: vi.fn(),
}));

const exploreData = {
  explore: {
    items: [
      {
        record_type: "bean",
        action: "added a new bean",
        record: {
          rkey: "bean-1",
          name: "Ethiopian Yirgacheffe",
          origin: "Ethiopia",
          variety: "",
          roast_level: "Light",
          roast_date: "",
          process: "Washed",
          description: "",
          notes: "",
          link: "",
          closed: false,
          created_at: "2026-01-01T00:00:00Z",
        },
        author: {
          did: "did:plc:abc",
          handle: "alice.test",
          display_name: "Alice",
          avatar: "",
        },
        timestamp: "2026-01-01T00:00:00Z",
        time_ago: "2 days ago",
        like_count: 5,
        comment_count: 2,
        subject_uri: "at://did:plc:abc/social.arabica.alpha.bean/bean-1",
        subject_cid: "bafy123",
        is_liked_by_viewer: false,
        is_owner: false,
      },
    ],
    documents: {},
    facet_counts: [],
    next_cursor: "",
    health: { Ready: true, Dirty: false, TotalDocuments: 100 },
  },
  error: "",
  isAuthenticated: true,
};

describe("Explore page", () => {
  beforeEach(() => {
    window.history.replaceState({}, "", "/explore");
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
    document.body.innerHTML = "";
    vi.unstubAllGlobals();
  });

  it("renders the hero title", () => {
    render(Explore, { data: exploreData });
    expect(
      screen.getByRole("heading", { name: "Explore", exact: true }),
    ).toBeTruthy();
  });

  it("renders the search input", () => {
    render(Explore, { data: exploreData });
    expect(screen.getByPlaceholderText("Ethiopia, V60, washed")).toBeTruthy();
  });

  it("renders the type filter dropdown", () => {
    render(Explore, { data: exploreData });
    expect(screen.getByText("All", { selector: "option" })).toBeTruthy();
  });

  it("renders results", () => {
    render(Explore, { data: exploreData });
    expect(screen.getByText("Ethiopian Yirgacheffe")).toBeTruthy();
  });

  it("renders the Explore button", () => {
    render(Explore, { data: exploreData });
    expect(
      screen.getByRole("button", { name: "Explore", exact: true }),
    ).toBeTruthy();
  });

  it("renders empty state when no results", () => {
    render(Explore, {
      data: {
        explore: {
          items: [],
          documents: {},
          facet_counts: [],
          next_cursor: "",
          health: { Ready: true, Dirty: false, TotalDocuments: 0 },
        },
        error: "",
        isAuthenticated: false,
      },
    });
    expect(screen.getByText("No matching records yet.")).toBeTruthy();
  });

  it("keeps primary controls visible and expands optional filters on request", async () => {
    render(Explore, { data: exploreData });
    const more = screen
      .getByText("More filters")
      .closest("details") as HTMLDetailsElement;
    expect(more.open).toBe(false);
    expect(screen.getByRole("searchbox", { name: "Search" })).toBeTruthy();
    expect(screen.getByRole("combobox", { name: "Type" })).toBeTruthy();
    expect(screen.getByRole("combobox", { name: "Sort" })).toBeTruthy();
    await fireEvent.click(screen.getByText("More filters"));
    expect(more.open).toBe(true);
    expect(screen.getByRole("textbox", { name: "Origin" })).toBeTruthy();
  });

  it("loads URL facets into visible controls and summarizes the applied result", async () => {
    window.history.replaceState(
      {},
      "",
      "/explore?q=washed&type=bean&origin=Kenya&sort=popular",
    );
    render(Explore, { data: exploreData });
    await waitFor(() =>
      expect(
        screen.getByRole<HTMLInputElement>("searchbox", { name: "Search" })
          .value,
      ).toBe("washed"),
    );
    expect(
      screen.getByRole<HTMLSelectElement>("combobox", { name: "Type" }).value,
    ).toBe("bean");
    expect(
      screen.getByRole<HTMLSelectElement>("combobox", { name: "Sort" }).value,
    ).toBe("popular");
    expect(
      screen.getByRole<HTMLInputElement>("textbox", { name: "Origin" }).value,
    ).toBe("Kenya");
    expect(
      screen.getByText("More filters (selected)").closest("details")?.open,
    ).toBe(true);
    expect(
      screen.getByText(
        /Filters: Search: washed · Type: Beans · Origin: Kenya · Sort: Popular/,
      ),
    ).toBeTruthy();
  });

  it("resets all filter controls and returns to the unfiltered route", async () => {
    window.history.replaceState(
      {},
      "",
      "/explore?q=washed&type=bean&origin=Kenya",
    );
    render(Explore, { data: exploreData });
    await waitFor(() =>
      expect(
        screen.getByRole<HTMLInputElement>("searchbox", { name: "Search" })
          .value,
      ).toBe("washed"),
    );
    await fireEvent.click(screen.getByRole("button", { name: "Clear" }));
    expect(
      screen.getByRole<HTMLInputElement>("searchbox", { name: "Search" }).value,
    ).toBe("");
    expect(
      screen.getByRole<HTMLSelectElement>("combobox", { name: "Type" }).value,
    ).toBe("");
    expect(
      screen.getByRole<HTMLInputElement>("textbox", { name: "Origin" }).value,
    ).toBe("");
    expect(goto).toHaveBeenCalledWith("/explore");
  });

  it("applies filters only after a successful search", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ ...exploreData.explore, items: [] }),
    });
    vi.stubGlobal("fetch", fetchMock);
    render(Explore, { data: exploreData });
    await fireEvent.input(screen.getByRole("searchbox", { name: "Search" }), {
      target: { value: "Kenya" },
    });
    expect(screen.queryByText(/Filters: Search: Kenya/)).toBeNull();
    await fireEvent.click(
      screen.getByRole("button", { name: "Explore", exact: true }),
    );
    await waitFor(() =>
      expect(screen.getByText(/Filters: Search: Kenya/)).toBeTruthy(),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/explore?q=Kenya",
      expect.anything(),
    );
    expect(goto).toHaveBeenCalledWith(
      "/explore?q=Kenya",
      expect.objectContaining({ replaceState: true }),
    );
    expect(screen.getByText("No matching records yet.")).toBeTruthy();
  });

  it("loads the next page using applied filters, not unsubmitted edits", async () => {
    window.history.replaceState({}, "", "/explore?type=bean");
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        ...exploreData.explore,
        items: [],
        next_cursor: "",
      }),
    });
    vi.stubGlobal("fetch", fetchMock);
    render(Explore, {
      data: {
        ...exploreData,
        explore: { ...exploreData.explore, next_cursor: "next" },
      },
    });
    await fireEvent.input(screen.getByRole("searchbox", { name: "Search" }), {
      target: { value: "unsent" },
    });
    await fireEvent.click(screen.getByRole("button", { name: "More results" }));
    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/explore?type=bean&cursor=next",
        expect.anything(),
      ),
    );
  });

  it("ignores an old next page after clearing the filters", async () => {
    let finish!: (response: unknown) => void;
    vi.stubGlobal("fetch", vi.fn().mockReturnValue(new Promise(resolve => { finish = resolve; })));
    render(Explore, { data: { ...exploreData, explore: { ...exploreData.explore, next_cursor: "next" } } });
    await fireEvent.click(screen.getByRole("button", { name: "More results" }));
    await fireEvent.click(screen.getByRole("button", { name: "Clear" }));
    finish({ ok: true, json: async () => ({ ...exploreData.explore, items: [{
      ...exploreData.explore.items[0], subject_uri: "at://did:plc:abc/social.arabica.alpha.bean/stale",
      record: { ...exploreData.explore.items[0].record, rkey: "stale", name: "Stale result" },
    }] }) });
    await waitFor(() => expect(screen.getByRole("button", { name: "More results" })).not.toBeDisabled());
    expect(screen.queryByText("Stale result")).toBeNull();
  });

  it("resets pending state when navigation supplies another result set", async () => {
    let finish!: (response: unknown) => void;
    vi.stubGlobal("fetch", vi.fn().mockReturnValue(new Promise(resolve => { finish = resolve; })));
    const { rerender } = render(Explore, { data: exploreData });
    await fireEvent.click(screen.getByRole("button", { name: "Explore", exact: true }));
    expect(screen.getByRole("button", { name: "Searching..." })).toBeDisabled();
    await rerender({ data: { ...exploreData, explore: { ...exploreData.explore, items: [] } } });
    expect(screen.getByRole("button", { name: "Explore", exact: true })).not.toBeDisabled();
    finish({ ok: true, json: async () => exploreData.explore });
    await waitFor(() => expect(screen.getByText("No matching records yet.")).toBeTruthy());
    expect(screen.queryByText("Ethiopian Yirgacheffe")).toBeNull();
  });

  it("keeps results and URL matched to the query sent before further typing", async () => {
    let finish!: (response: unknown) => void;
    const pending = new Promise((resolve) => {
      finish = resolve;
    });
    vi.stubGlobal("fetch", vi.fn().mockReturnValue(pending));
    render(Explore, { data: exploreData });
    const search = screen.getByRole<HTMLInputElement>("searchbox", {
      name: "Search",
    });
    await fireEvent.input(search, { target: { value: "Kenya" } });
    await fireEvent.click(
      screen.getByRole("button", { name: "Explore", exact: true }),
    );
    await fireEvent.input(search, { target: { value: "Colombia" } });
    finish({
      ok: true,
      json: async () => ({ ...exploreData.explore, items: [] }),
    });
    await waitFor(() =>
      expect(screen.getByText(/Filters: Search: Kenya/)).toBeTruthy(),
    );
    expect(search.value).toBe("Colombia");
    expect(goto).toHaveBeenCalledWith("/explore?q=Kenya", expect.anything());
    expect(screen.queryByText(/Filters: Search: Colombia/)).toBeNull();
  });

  it("does not apply a pending search after clearing", async () => {
    let finish!: (response: unknown) => void;
    vi.stubGlobal(
      "fetch",
      vi.fn().mockReturnValue(
        new Promise((resolve) => {
          finish = resolve;
        }),
      ),
    );
    render(Explore, { data: exploreData });
    await fireEvent.input(screen.getByRole("searchbox", { name: "Search" }), {
      target: { value: "Kenya" },
    });
    await fireEvent.click(
      screen.getByRole("button", { name: "Explore", exact: true }),
    );
    await fireEvent.click(screen.getByRole("button", { name: "Clear" }));
    finish({
      ok: true,
      json: async () => ({ ...exploreData.explore, items: [] }),
    });
    await waitFor(() =>
      expect(
        screen.getByRole<HTMLInputElement>("searchbox", { name: "Search" })
          .value,
      ).toBe(""),
    );
    expect(goto).toHaveBeenCalledTimes(1);
    expect(goto).toHaveBeenCalledWith("/explore");
    expect(screen.getByText("Ethiopian Yirgacheffe")).toBeTruthy();
  });

  it("shows direct-load API failure instead of an empty-state suggestion", () => {
    render(Explore, {
      data: {
        explore: null,
        error: "Failed to load explore results",
        isAuthenticated: true,
      },
    });
    expect(screen.getByText("Failed to load explore results")).toBeTruthy();
    expect(screen.queryByText("No matching records yet.")).toBeNull();
  });

  it("prompts unauthenticated visitors to log in, without showing empty results", () => {
    render(Explore, {
      data: {
        explore: null,
        error: "Authentication required",
        isAuthenticated: false,
      },
    });
    expect(screen.getByText("Authentication required")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Log In" })).toBeTruthy();
    expect(screen.queryByText("No matching records yet.")).toBeNull();
  });

  it("shows a search failure without replacing previous results or changing URL", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false }));
    render(Explore, { data: exploreData });
    await fireEvent.input(screen.getByRole("searchbox", { name: "Search" }), {
      target: { value: "Kenya" },
    });
    await fireEvent.click(
      screen.getByRole("button", { name: "Explore", exact: true }),
    );
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Search failed. Showing previous results.",
    );
    expect(screen.getByText("Ethiopian Yirgacheffe")).toBeTruthy();
    expect(screen.queryByText("No matching records yet.")).toBeNull();
    expect(goto).not.toHaveBeenCalled();
  });

  it("does not present failed empty searches as empty results", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("offline")));
    render(Explore, {
      data: { ...exploreData, explore: { ...exploreData.explore, items: [] } },
    });
    await fireEvent.click(
      screen.getByRole("button", { name: "Explore", exact: true }),
    );
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Search failed. Please try again.",
    );
    expect(screen.queryByText("No matching records yet.")).toBeNull();
  });

  it("shows the stale note when health is dirty", () => {
    render(Explore, {
      data: {
        ...exploreData,
        explore: {
          ...exploreData.explore,
          health: { Ready: true, Dirty: true, TotalDocuments: 100 },
        },
      },
    });
    expect(screen.getByText(/Explore is catching up/)).toBeTruthy();
  });
});
