import { cleanup, render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

const { pushToast } = vi.hoisted(() => ({ pushToast: vi.fn() }));
vi.mock("$lib/stores/toasts", () => ({ pushToast }));
import MyCoffee from "../../src/routes/my-coffee/+page.svelte";

const manageData = {
  did: "did:plc:abc",
  beans: [
    {
      rkey: "bean-1",
      name: "Finca Buena Vista",
      origin: "Colombia",
      variety: "Bourbon",
      roast_level: "Light",
      roast_date: "",
      process: "Washed",
      description: "",
      notes: "",
      link: "",
      rating: 8,
      closed: false,
      created_at: "2026-01-15T10:00:00Z",
      roaster: { name: "Heart", location: "Portland", rkey: "roaster-1" },
    },
  ],
  roasters: [
    {
      rkey: "roaster-1",
      name: "Heart Coffee",
      location: "Portland, OR",
      website: "",
      created_at: "2026-01-01T00:00:00Z",
    },
  ],
  grinders: [
    {
      rkey: "grinder-1",
      name: "Ode 2",
      grinder_type: "Electric",
      burr_type: "Flat",
      notes: "",
      link: "",
      created_at: "2026-01-01T00:00:00Z",
    },
  ],
  brewers: [
    {
      rkey: "brewer-1",
      name: "V60",
      brewer_type: "pourover",
      description: "",
      link: "",
      created_at: "2026-01-01T00:00:00Z",
    },
  ],
  recipes: [],
  stats: {
    bean_brew_counts: {
      "at://did:plc:abc/social.arabica.alpha.bean/bean-1": 3,
    },
    grinder_brew_counts: {
      "at://did:plc:abc/social.arabica.alpha.grinder/grinder-1": 5,
    },
    brewer_brew_counts: {
      "at://did:plc:abc/social.arabica.alpha.brewer/brewer-1": 5,
    },
    roaster_bean_counts: {
      "at://did:plc:abc/social.arabica.alpha.roaster/roaster-1": 1,
    },
    bean_avg_brew_ratings: {
      "at://did:plc:abc/social.arabica.alpha.bean/bean-1": 7.5,
    },
    roaster_avg_brew_ratings: {},
  },
};

const brewsData = {
  brews: [
    {
      rkey: "brew-1",
      bean_rkey: "bean-1",
      recipe_rkey: "",
      temperature: 205,
      water_amount: 250,
      coffee_amount: 15,
      time_seconds: 180,
      grind_size: "Medium",
      grinder_rkey: "",
      brewer_rkey: "",
      tasting_notes: "Bright and sweet",
      rating: 8,
      created_at: "2026-01-20T10:00:00Z",
      bean: {
        rkey: "bean-1",
        name: "Finca Buena Vista",
        origin: "Colombia",
        variety: "",
        roast_level: "Light",
        roast_date: "",
        process: "",
        description: "",
        notes: "",
        link: "",
        closed: false,
        created_at: "",
        roaster: { name: "Heart", location: "", rkey: "roaster-1" },
      },
      brewer_obj: {
        rkey: "brewer-1",
        name: "V60",
        brewer_type: "pourover",
        description: "",
        link: "",
        created_at: "",
      },
    },
  ],
  has_more: false,
  next_offset: 25,
};

describe("My Coffee page", () => {
  afterEach(() => {
    cleanup();
    document.body.innerHTML = "";
    localStorage.clear();
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it("renders the heading and tab navigation", () => {
    render(MyCoffee, {
      data: { manage: manageData, brews: brewsData, error: "" },
    });
    expect(screen.getByText("My Coffee")).toBeTruthy();
    expect(screen.getByText("Brews")).toBeTruthy();
    expect(screen.getByText("Beans")).toBeTruthy();
    expect(screen.getByText("Roasters")).toBeTruthy();
    expect(screen.getByText("Grinders")).toBeTruthy();
    expect(screen.queryByText("Personal collection")).toBeNull();
    expect(screen.queryByText("1 brewer and 1 grinder saved.")).toBeNull();
    expect(screen.queryByRole("complementary", { name: "Collection tools" })).toBeNull();
    expect(screen.getByRole("button", { name: "Refresh", exact: true })).toBeTruthy();
  });

  it("shows the brews tab by default", () => {
    render(MyCoffee, {
      data: { manage: manageData, brews: brewsData, error: "" },
    });
    expect(screen.getByText("Bright and sweet")).toBeTruthy();
  });

  it("switches to the beans tab and shows bean stats", async () => {
    const user = userEvent.setup();
    render(MyCoffee, {
      data: { manage: manageData, brews: brewsData, error: "" },
    });
    await user.click(screen.getByText("Beans", { selector: "button" }));
    expect(screen.getByText("Finca Buena Vista")).toBeTruthy();
    expect(screen.getByText(/3 brews/)).toBeTruthy();
    expect(screen.getByText(/7.5\/10 avg/)).toBeTruthy();
  });

  it("opens the beans collection and separates open and finished bags", async () => {
    const user = userEvent.setup();
    render(MyCoffee, {
      data: {
        manage: {
          ...manageData,
          beans: [
            ...manageData.beans,
            {
              ...manageData.beans[0],
              rkey: "bean-finished",
              name: "Finished coffee",
              closed: true,
            },
          ],
        },
        brews: brewsData,
        error: "",
      },
    });
    expect(
      screen.getByRole("button", { name: "Beans", exact: true }),
    ).toHaveAttribute("data-count", "2");
    await user.click(
      screen.getByRole("button", { name: "Beans", exact: true }),
    );
    expect(
      screen.getByRole("button", { name: "Beans", exact: true }),
    ).toHaveAttribute("aria-current", "page");
    expect(localStorage.getItem("manageTab")).toBe("beans");
    expect(screen.getByText("Finca Buena Vista")).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Open bags" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Finished bags" })).toBeTruthy();
    expect(screen.getByText("Finished coffee")).toBeTruthy();
  });

  it("shows the actionable empty bean state without redundant equipment summaries", async () => {
    const user = userEvent.setup();
    render(MyCoffee, {
      data: {
        manage: {
          ...manageData,
          beans: [],
          brewers: [],
          grinders: [
            ...manageData.grinders,
            { ...manageData.grinders[0], rkey: "grinder-2" },
          ],
        },
        brews: brewsData,
        error: "",
      },
    });

    expect(screen.queryByText("0 brewers and 2 grinders saved.")).toBeNull();
    await user.click(
      screen.getByRole("button", { name: "Beans", exact: true }),
    );
    expect(screen.getByText(/No open bags\./)).toBeTruthy();
    expect(
      screen.getByRole("link", { name: "Add the bag you're drinking now" }),
    ).toHaveAttribute("href", "/beans/new");
  });

  it("switches to the roasters tab and shows bean count", async () => {
    const user = userEvent.setup();
    render(MyCoffee, {
      data: { manage: manageData, brews: brewsData, error: "" },
    });
    await user.click(screen.getByText("Roasters", { selector: "button" }));
    expect(screen.getByText("Heart Coffee")).toBeTruthy();
    expect(screen.getByText(/1 bean/)).toBeTruthy();
  });

  it("switches to the grinders tab and shows brew count", async () => {
    const user = userEvent.setup();
    render(MyCoffee, {
      data: { manage: manageData, brews: brewsData, error: "" },
    });
    await user.click(screen.getByText("Grinders", { selector: "button" }));
    expect(screen.getByText("Ode 2")).toBeTruthy();
    expect(screen.getByText(/5 brews/)).toBeTruthy();
  });

  it("renders an empty state when no brews", () => {
    render(MyCoffee, {
      data: {
        manage: manageData,
        brews: { brews: [], has_more: false, next_offset: 0 },
        error: "",
      },
    });
    expect(screen.getByText("No brews recorded yet.")).toBeTruthy();
  });

  it("renders an error state", () => {
    render(MyCoffee, {
      data: { manage: null, brews: null, error: "Authentication required" },
    });
    expect(screen.getByText("Authentication required")).toBeTruthy();
  });

  it("refreshes the collection without switching tabs", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn()
      .mockResolvedValueOnce({ ok: true })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ ...manageData, beans: [] }) })
      .mockResolvedValueOnce({ ok: true, json: async () => brewsData });
    vi.stubGlobal("fetch", fetchMock);
    render(MyCoffee, { data: { manage: manageData, brews: brewsData, error: "" } });
    await user.click(screen.getByRole("button", { name: "Beans", exact: true }));
    await user.click(screen.getByRole("button", { name: "Refresh", exact: true }));
    expect(fetchMock).toHaveBeenCalledWith("/api/manage/refresh", {
      method: "POST", credentials: "same-origin",
    });
    expect(screen.getByText(/No open bags\./)).toBeTruthy();
    expect(localStorage.getItem("manageTab")).toBe("beans");
    expect(pushToast).toHaveBeenCalledWith("Refreshed");
  });

  it("disables refresh while the account is reloading", async () => {
    const user = userEvent.setup();
    let finish!: (value: { ok: boolean }) => void;
    const pending = new Promise<{ ok: boolean }>((resolve) => { finish = resolve; });
    vi.stubGlobal("fetch", vi.fn().mockReturnValue(pending));
    render(MyCoffee, { data: { manage: manageData, brews: brewsData, error: "" } });
    await user.click(screen.getByRole("button", { name: "Refresh", exact: true }));
    expect(screen.getByRole("button", { name: "Refreshing…" })).toBeDisabled();
    finish({ ok: false });
    expect(await screen.findByRole("button", { name: "Refresh", exact: true })).not.toBeDisabled();
  });

  it.each(["refresh", "reload"])("reports %s failure and retains visible records", async (stage) => {
    const user = userEvent.setup();
    const fetchMock = vi.fn();
    if (stage === "refresh") fetchMock.mockResolvedValue({ ok: false });
    else fetchMock.mockResolvedValueOnce({ ok: true })
      .mockResolvedValueOnce({ ok: false })
      .mockResolvedValueOnce({ ok: true, json: async () => brewsData });
    vi.stubGlobal("fetch", fetchMock);
    render(MyCoffee, { data: { manage: manageData, brews: brewsData, error: "" } });
    await user.click(screen.getByRole("button", { name: "Refresh", exact: true }));
    expect(screen.getByText("Bright and sweet")).toBeTruthy();
    expect(pushToast).toHaveBeenCalledWith("Refresh failed");
    expect(pushToast).not.toHaveBeenCalledWith("Refreshed");
    expect(screen.getByRole("button", { name: "Refresh", exact: true })).not.toBeDisabled();
  });

  it("links brew cards to the view URL", () => {
    render(MyCoffee, {
      data: { manage: manageData, brews: brewsData, error: "" },
    });
    const link = screen.getByText("Bright and sweet").closest("a");
    expect(link).toHaveAttribute("href", "/brews/did:plc:abc/brew-1");
  });
});
