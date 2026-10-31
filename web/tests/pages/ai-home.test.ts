import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import FeedPage from "../../src/routes/+page.svelte";
vi.mock("$app/navigation", () => ({ goto: vi.fn() }));
const data = {
  feed: {
    items: [],
    next_cursor: "",
    is_authenticated: true,
    query: { type: "", sort: "recent" },
    tabs: [
      { label: "All", value: "" },
      { label: "Brews", value: "brew" },
    ],
  },
  error: "",
  typeFilter: "",
  sort: "recent",
  isAuthenticated: true,
  userDID: "did:plc:alice",
  appName: "arabica",
  aiEnabled: true,
};
afterEach(() => {
  cleanup();
  sessionStorage.clear();
});
describe("AI home entry", () => {
  it("adds only the input bar while retaining Community, manual logging and feed filters", () => {
    render(FeedPage, { data });
    expect(
      screen.getByRole("heading", { name: "Community", level: 1 }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Log Brew" })).toHaveAttribute(
      "href",
      "/brews/new",
    );
    expect(
      screen.getByRole("group", { name: "Feed filters" }),
    ).toBeInTheDocument();
    expect(
      screen.getByLabelText("Describe a brew or bean"),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Send", exact: true }),
    ).toBeDisabled();
    expect(screen.getAllByRole("heading", { level: 1 })).toHaveLength(1);
    const prompt = screen.getByLabelText("Describe a brew or bean");
    const filters = screen.getByRole("group", { name: "Feed filters" });
    const masthead = prompt.closest(".cafe-masthead")!;
    expect(masthead).not.toBeNull();
    expect(masthead.contains(filters)).toBe(false);
    expect(
      masthead.compareDocumentPosition(filters) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });
  it("keeps Finish setup alongside input for incomplete accounts", () => {
    render(FeedPage, {
      data: {
        ...data,
        onboarding: {
          readiness: { HasBean: false, HasBrewer: false, HasRoaster: false },
          beans: [],
          brewers: [],
          roasters: [],
        },
      },
    });
    expect(screen.getByRole("link", { name: "Finish setup" })).toHaveAttribute(
      "href",
      "/onboarding",
    );
    expect(
      screen.getByLabelText("Describe a brew or bean"),
    ).toBeInTheDocument();
  });
  it.each([{ aiEnabled: false }, { isAuthenticated: false }])(
    "hides AI entry unless configured and signed in",
    (overrides) => {
      render(FeedPage, { data: { ...data, ...overrides } });
      expect(
        screen.queryByLabelText("Describe a brew or bean"),
      ).not.toBeInTheDocument();
    },
  );
});
