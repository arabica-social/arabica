import {
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import SettingsPage from "../../src/routes/settings/+page.svelte";
import { pushToast } from "$lib/stores/toasts";

vi.mock("$lib/stores/toasts", () => ({ pushToast: vi.fn() }));
vi.mock("$lib/stores/session", async (importOriginal) => ({
  ...(await importOriginal<typeof import("$lib/stores/session")>()),
  openLoginModal: vi.fn(),
}));

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

  it("renders settings with section navigation and the shared-profile warning", () => {
    render(SettingsPage, { data: { settings, error: "" } });

    expect(screen.getByRole("heading", { name: "Settings" })).toBeTruthy();
    expect(screen.queryByText("Account ledger")).toBeNull();
    expect(
      screen.getByText(
        /Changes here update your Bluesky profile and appear in other apps/,
      ),
    ).toBeTruthy();
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

  it("explains browser-local and account-wide preference scopes beside the controls", () => {
    render(SettingsPage, { data: { settings, error: "" } });

    expect(
      within(screen.getByRole("region", { name: "Appearance" })).getByText(
        /Saved only in this browser on this device\. System follows your operating system's theme\./,
      ),
    ).toBeTruthy();
    expect(
      within(
        screen.getByRole("region", { name: "Brewing preferences" }),
      ).getByText(
        "Saved by Arabica for your account and used across devices when you sign in.",
      ),
    ).toBeTruthy();
    expect(
      within(screen.getByRole("region", { name: "Developer" })).getByText(
        "Saved only in this browser on this device.",
      ),
    ).toBeTruthy();
    expect(screen.queryByText("Travels with your DID")).toBeNull();
  });

  it("persists a local theme choice", async () => {
    const user = userEvent.setup();
    render(SettingsPage, { data: { settings, error: "" } });

    await user.click(screen.getByRole("button", { name: "Dark", exact: true }));

    expect(localStorage.getItem("arabica-theme")).toBe("dark");
    expect(
      screen.getByRole("button", { name: "Dark", exact: true }),
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
