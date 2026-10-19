import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import AddPage from "../../src/routes/add/+page.svelte";
import OnboardingPage from "../../src/routes/onboarding/+page.svelte";

const addData = {
  onboarding: {
    readiness: { HasBean: true, HasBrewer: true, HasRoaster: true },
    beans: [
      {
        rkey: "bean-1",
        name: "Finca",
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
      },
    ],
    brewers: [
      {
        rkey: "brewer-1",
        name: "V60",
        brewer_type: "pourover",
        description: "",
        link: "",
        created_at: "",
      },
    ],
    grinders: [],
    roasters: [
      {
        rkey: "roaster-1",
        name: "Heart",
        location: "",
        website: "",
        created_at: "",
      },
    ],
  },
  error: "",
  mode: "library" as const,
  initialEntity: "",
};

describe("Add records page", () => {
  afterEach(() => {
    cleanup();
    document.body.innerHTML = "";
  });

  it("renders the library hero title", () => {
    render(AddPage, { data: addData });
    expect(
      screen.getByRole("heading", { name: "What would you like to add?" }),
    ).toBeTruthy();
  });

  it("keeps station guidance without duplicated instructions", () => {
    render(AddPage, { data: addData });
    expect(screen.queryByText(/to use in your brew records/)).toBeNull();
    expect(screen.getByText("optional")).toBeTruthy();
    expect(screen.getByRole("button", { name: /Add a grinder/ })).toBeTruthy();
  });

  it("renders the stations without progress strip", () => {
    render(AddPage, { data: addData });
    expect(screen.getAllByText("Brewer").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Roaster").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Bean").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Grinder").length).toBeGreaterThan(0);
  });

  it("preserves onboarding readiness and the first-brew action", () => {
    render(OnboardingPage, { data: addData });
    expect(
      screen.getByRole("group", {
        name: "Setup progress: 3 of 3 required items added",
      }),
    ).toBeTruthy();
    expect(
      screen.getByRole("link", { name: /Log your first brew/ }),
    ).toHaveAttribute("href", "/brews/new");
  });

  it("names missing prerequisites and keeps the grinder optional", () => {
    render(OnboardingPage, {
      data: {
        ...addData,
        onboarding: {
          ...addData.onboarding,
          brewers: [],
          beans: [],
          roasters: [],
          readiness: { HasBean: false, HasBrewer: false, HasRoaster: false },
        },
      },
    });
    expect(screen.getAllByText("required")).toHaveLength(3);
    expect(screen.getByText("optional")).toBeTruthy();
    expect(
      screen.getByText("Add a brewer, roaster, and bean to log a brew."),
    ).toBeTruthy();
    expect(
      screen.queryByRole("link", { name: /Log your first brew/ }),
    ).toBeNull();
  });

  it("renders entity items", () => {
    render(AddPage, { data: addData });
    expect(screen.getByText("Finca")).toBeTruthy();
    expect(screen.getByText("V60")).toBeTruthy();
    expect(screen.getByText("Heart")).toBeTruthy();
  });

  it("renders error state", () => {
    render(AddPage, {
      data: {
        onboarding: null,
        error: "Authentication required",
        mode: "library",
        initialEntity: "",
      },
    });
    expect(screen.getByText("Authentication required")).toBeTruthy();
  });

  it("does not show the ready panel in library mode", () => {
    render(AddPage, { data: addData });
    expect(screen.queryByText("You're ready to brew!")).toBeNull();
  });
});
