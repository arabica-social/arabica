import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import SettingsPage from "../../src/routes/settings/+page.svelte";
import { pushToast } from "$lib/stores/toasts";

vi.mock("$lib/stores/toasts", () => ({ pushToast: vi.fn() }));
vi.mock("$lib/stores/session", () => ({ openLoginModal: vi.fn() }));

const settings = {
  user_preferences: { temperature_unit: "recorded" },
  profile_stats_visibility: {
    bean_avg_rating: "public",
    roaster_avg_rating: "private",
  },
  bluesky_profile: {
    has_scopes: true,
    display_name: "Ledger Brewer",
    avatar_url: "",
    load_error: "",
    needs_auth_again: false,
  },
};

describe("Settings page", () => {
  afterEach(() => {
    cleanup();
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it("renders the account ledger and section index", () => {
    render(SettingsPage, { data: { settings, error: "" } });

    expect(screen.getByRole("heading", { name: "Settings" })).toBeTruthy();
    expect(screen.getByText("Account ledger")).toBeTruthy();
    expect(
      screen.getByRole("navigation", { name: "Settings sections" }),
    ).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Appearance" })).toBeTruthy();
    expect(
      screen.getByRole("heading", { name: "Brewing preferences" }),
    ).toBeTruthy();
    expect(
      screen.getByRole("heading", { name: "Profile visibility" }),
    ).toBeTruthy();
    expect(
      screen.getByRole("heading", { name: "Bluesky profile" }),
    ).toBeTruthy();
  });

  it("persists a local theme choice", async () => {
    const user = userEvent.setup();
    render(SettingsPage, { data: { settings, error: "" } });

    await user.click(
      screen.getByRole("button", { name: /DarkEspresso paper/ }),
    );

    expect(localStorage.getItem("arabica-theme")).toBe("dark");
    expect(
      screen.getByRole("button", { name: /DarkEspresso paper/ }),
    ).toHaveAttribute("aria-pressed", "true");
  });

  it("saves the temperature preference through the existing endpoint", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn().mockResolvedValue({ ok: true } as Response);
    globalThis.fetch = fetchMock as unknown as typeof fetch;
    render(SettingsPage, { data: { settings, error: "" } });

    await user.selectOptions(
      screen.getByLabelText("Preferred temperature unit"),
      "celsius",
    );
    await user.click(screen.getAllByRole("button", { name: "Save" })[0]);

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/settings/preferences",
        expect.objectContaining({ method: "POST" }),
      );
      expect(pushToast).toHaveBeenCalledWith("Preferences saved");
    });
  });
});
