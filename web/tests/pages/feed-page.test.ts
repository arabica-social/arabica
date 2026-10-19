import { cleanup, render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import FeedPage from "../../src/routes/+page.svelte";
import type { FeedResponse } from "../../src/lib/types/feed";

vi.mock("$app/navigation", () => ({
  goto: vi.fn(),
}));

const brewItem = {
  record_type: "brew" as const,
  action: "added a new brew",
  record: {
    rkey: "brew-1",
    rating: 8,
    coffee_amount: 15,
    water_amount: 250,
    bean: { rkey: "bean-1", name: "Test Bean" },
  },
  author: {
    did: "did:plc:abc",
    handle: "alice.test",
    display_name: "Alice",
    avatar: "",
  },
  timestamp: "2026-01-20T10:00:00Z",
  time_ago: "2 hours ago",
  like_count: 3,
  comment_count: 1,
  subject_uri: "at://did:plc:abc/social.arabica.alpha.brew/brew-1",
  subject_cid: "bafy123",
  is_liked_by_viewer: false,
  is_owner: true,
};

const feedData: FeedResponse = {
  items: [brewItem],
  next_cursor: "",
  is_authenticated: true,
  query: { type: "", sort: "recent" },
  tabs: [
    { label: "All", value: "" },
    { label: "Brews", value: "brew" },
    { label: "Beans", value: "bean" },
    { label: "Roasters", value: "roaster" },
  ],
};

const authedPageData = {
  feed: feedData,
  error: "",
  typeFilter: "",
  sort: "recent",
  isAuthenticated: true,
  userDID: "did:plc:abc",
  appName: "arabica",
};

const unauthedPageData = {
  feed: feedData,
  error: "",
  typeFilter: "",
  sort: "recent",
  isAuthenticated: false,
  userDID: "",
  appName: "arabica",
};

describe("Feed page (home)", () => {
  afterEach(() => {
    cleanup();
    document.body.innerHTML = "";
    vi.restoreAllMocks();
  });

  it("preserves the original journal header and brew action", () => {
    render(FeedPage, { data: authedPageData });
    expect(
      screen.getByRole("heading", {
        level: 1,
        name: "Good coffee deserves a record.",
      }),
    ).toBeTruthy();
    expect(screen.getByRole("link", { name: "Log Brew" })).toHaveAttribute(
      "href",
      "/brews/new",
    );
    expect(
      screen.getByText(
        "Keep track of what you brewed and how it turned out. Compare notes with other people around the Atmosphere.",
      ),
    ).toBeTruthy();
    expect(screen.getByText("Your home coffee journal")).toBeTruthy();
    expect(
      screen.getByRole("link", { name: "Browse your journal" }),
    ).toHaveAttribute("href", "/my-coffee");
    expect(screen.queryByText("Ready for the next cup.")).toBeNull();
    expect(
      screen.queryByRole("heading", { name: "Unfinished entries" }),
    ).toBeNull();
    expect(
      screen.queryByRole("heading", { name: "Popular recipes" }),
    ).toBeNull();
  });

  it("keeps icon navigation and a concise feedback invitation", () => {
    render(FeedPage, { data: authedPageData });
    expect(
      screen.getByRole("navigation", { name: "Home shortcuts" }),
    ).toBeTruthy();
    for (const [name, href] of [
      ["My Coffee", "/my-coffee"],
      ["Explore", "/explore"],
      ["Recipes", "/recipes"],
      ["Your profile", "/profile/did:plc:abc"],
    ]) {
      const link = screen.getByRole("link", { name, exact: true });
      expect(link).toHaveAttribute("href", href);
      expect(link.querySelector("svg")).toBeTruthy();
    }
    const feedback = screen.getByRole("link", { name: "Feedback" });
    expect(feedback).toHaveAttribute(
      "href",
      "https://userinput.app/s/did:plc:chqc2ockzmyvlrasfb66x64a/3mrgh3b4f722p",
    );
    expect(feedback).toHaveAttribute("target", "_blank");
    expect(feedback).toHaveAttribute("rel", "noopener noreferrer");
    expect(screen.getByRole("link", { name: "About Arabica" })).toHaveAttribute(
      "href",
      "/about",
    );
    expect(
      screen.getByText(
        "Found something confusing, broken, or missing? Let us know.",
      ),
    ).toBeTruthy();
    expect(screen.queryByText("Tell us what needs work.")).toBeNull();
  });

  it("offers one setup action when prerequisites are missing", () => {
    render(FeedPage, {
      data: {
        ...authedPageData,
        onboarding: {
          readiness: { HasBean: false, HasBrewer: false, HasRoaster: false },
          beans: [],
          brewers: [],
          grinders: [],
          roasters: [],
        },
      },
    });
    expect(screen.getAllByRole("link", { name: "Finish setup" })).toHaveLength(
      1,
    );
    expect(screen.getByRole("link", { name: "Finish setup" })).toHaveAttribute(
      "href",
      "/onboarding",
    );
    expect(screen.queryByRole("link", { name: "Log Brew" })).toBeNull();
    expect(screen.queryByText("Open setup")).toBeNull();
    expect(
      screen.queryByText(
        "Keep track of what you brewed and how it turned out. Compare notes with other people around the Atmosphere.",
      ),
    ).toBeNull();
  });

  it("shows only useful sidebar sections when their data is present", () => {
    render(FeedPage, {
      data: {
        ...authedPageData,
        incompleteRecords: {
          records: [
            {
              EntityType: "bean",
              RKey: "bean-1",
              Name: "Current bag",
              MissingFields: ["origin"],
            },
          ],
        },
        popularRecipes: [
          {
            rkey: "recipe-1",
            name: "Morning V60",
            author_handle: "alice.test",
            brewer_rkey: "",
            brewer_type: "V60",
            coffee_amount: 15,
            water_amount: 250,
            notes: "",
            created_at: "",
            brew_count: 2,
            fork_count: 1,
          },
        ],
      },
    });
    expect(
      screen.getByRole("complementary", { name: "Community links" }),
    ).toBeTruthy();
    expect(
      screen.getByRole("heading", { name: "Unfinished entries" }),
    ).toBeTruthy();
    expect(screen.getByRole("link", { name: /Current bag/ })).toHaveAttribute(
      "href",
      "/beans/bean-1/edit",
    );
    expect(
      screen.getByRole("heading", { name: "Popular recipes" }),
    ).toBeTruthy();
    expect(screen.getByRole("link", { name: /Morning V60/ })).toHaveAttribute(
      "href",
      "/recipes/alice.test/recipe-1",
    );
    expect(screen.getByText("2 brews · 1 fork")).toBeTruthy();
    expect(screen.queryByText("Worth a closer look.")).toBeNull();
  });

  it("renders the unauth hero CTA for unauthenticated users", () => {
    render(FeedPage, { data: unauthedPageData });
    expect(screen.queryByPlaceholderText("your-handle.bsky.social")).toBeNull();
    expect(screen.getByText("Log in")).toBeTruthy();
    expect(screen.getByText("Create an account")).toBeTruthy();
    expect(
      screen.getByRole("heading", { level: 1, name: "A coffee journal" }),
    ).toBeTruthy();
    expect(
      screen.queryByRole("complementary", { name: "Your coffee journal" }),
    ).toBeNull();
    expect(screen.queryByText("Sign in")).toBeNull();
    expect(screen.queryByRole("link", { name: "My Coffee" })).toBeNull();
    expect(screen.queryByRole("link", { name: "Your profile" })).toBeNull();
    expect(screen.getByRole("link", { name: "Feedback" })).toBeTruthy();
    expect(screen.getByRole("link", { name: "About Arabica" })).toBeTruthy();
    expect(
      screen.getByRole("link", { name: "AT Protocol", exact: true }),
    ).toHaveAttribute("href", "/atproto");
    expect(
      screen.getByText(
        /Coffee records are saved to your account's server through/,
      ),
    ).toBeTruthy();
  });

  it("renders the recent activity heading", () => {
    render(FeedPage, { data: authedPageData });
    expect(
      screen.getByRole("heading", { name: "Recent activity" }),
    ).toBeTruthy();
  });

  it("renders feed filter tabs", () => {
    render(FeedPage, { data: authedPageData });
    expect(screen.getByText("All")).toBeTruthy();
    expect(screen.getByText("Brews")).toBeTruthy();
    expect(screen.getByText("Beans")).toBeTruthy();
    expect(screen.getByText("Roasters")).toBeTruthy();
  });

  it("renders feed items from load data", () => {
    render(FeedPage, { data: authedPageData });
    expect(screen.getByText("Alice")).toBeTruthy();
    expect(screen.getByText(/2 hours ago/)).toBeTruthy();
  });

  it("renders an empty state when feed has no items", () => {
    const emptyData = {
      ...authedPageData,
      feed: { ...feedData, items: [] },
    };
    render(FeedPage, { data: emptyData });
    expect(screen.getByText("No activity yet")).toBeTruthy();
  });

  it("renders an error state when feed failed to load", () => {
    const errorData = {
      ...authedPageData,
      feed: null,
      error: "Failed to load feed",
    };
    render(FeedPage, { data: errorData });
    expect(screen.getByText("The feed could not be loaded")).toBeTruthy();
    expect(screen.getByText("Failed to load feed")).toBeTruthy();
  });

  it("fetches new items when a filter tab is clicked", async () => {
    const user = userEvent.setup();
    const mockResponse: FeedResponse = {
      items: [
        {
          ...brewItem,
          record: { rkey: "roaster-1", name: "Filtered Roaster" },
          record_type: "roaster",
          action: "added a new roaster",
          subject_uri:
            "at://did:plc:abc/social.arabica.alpha.roaster/roaster-1",
        },
      ],
      next_cursor: "",
      is_authenticated: true,
      query: { type: "roaster", sort: "recent" },
      tabs: [
        { label: "All", value: "" },
        { label: "Brews", value: "brew" },
        { label: "Beans", value: "bean" },
        { label: "Roasters", value: "roaster" },
      ],
    };
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify(mockResponse), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    render(FeedPage, { data: authedPageData });

    const roastersTab = screen.getByText("Roasters");
    await user.click(roastersTab);

    expect(fetchSpy).toHaveBeenCalled();
    const calledURL = fetchSpy.mock.calls[0][0] as string;
    expect(calledURL).toContain("type=roaster");
  });

  it("shows a Load more button when next_cursor is present", () => {
    const paginatedData = {
      ...authedPageData,
      feed: { ...feedData, next_cursor: "cursor-abc" },
    };
    render(FeedPage, { data: paginatedData });
    expect(screen.getByText("Load more")).toBeTruthy();
  });
});
