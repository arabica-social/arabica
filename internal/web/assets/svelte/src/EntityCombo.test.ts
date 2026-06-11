import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import EntityCombo from "./EntityCombo.svelte";

function installAppCache(data: Record<string, unknown>) {
  const listener = vi.fn();
  window.AppCache = {
    getCachedData: vi.fn(() => data),
    getData: vi.fn(() => Promise.resolve(data)),
    refreshCache: vi.fn(() => Promise.resolve(data)),
    invalidateCache: vi.fn(),
    init: vi.fn(() => Promise.resolve()),
    preload: vi.fn(() => Promise.resolve(data)),
    isCacheValid: vi.fn(() => true),
    addListener: vi.fn((fn) => listener.mockImplementation(fn)),
    removeListener: vi.fn(),
    invalidateAndRefresh: vi.fn(() => Promise.resolve(data)),
  };
  return window.AppCache;
}

describe("EntityCombo", () => {
  afterEach(() => {
    cleanup();
    document.body.innerHTML = "";
    delete window.AppCache;
    vi.restoreAllMocks();
  });

  it("renders as a Svelte-owned combo and selects a cached entity", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    installAppCache({
      beans: [
        {
          rkey: "bean-1",
          name: "Finca Buena Vista",
          origin: "Colombia",
          roast_level: "Light",
        },
      ],
    });

    const { container } = render(EntityCombo, {
      props: {
        entityType: "bean",
        apiEndpoint: "/api/beans",
        suggestEndpoint: "",
        inputName: "bean_rkey",
        placeholder: "Search beans...",
        sectionLabel: "Your beans",
        required: true,
        allowCreate: false,
        ariaLabel: "Search coffee beans",
        onChange,
      },
    });

    const hidden = container.querySelector<HTMLInputElement>(
      'input[type="hidden"][name="bean_rkey"]',
    );
    expect(hidden).toBeTruthy();
    expect(hidden).toHaveAttribute("required");
    expect(hidden).toHaveValue("");

    const search = screen.getByRole("combobox", {
      name: "Search coffee beans",
    });
    await user.type(search, "buena");
    await user.click(
      await screen.findByRole("option", {
        name: "Finca Buena Vista (Colombia - Light)",
      }),
    );

    expect(hidden).toHaveValue("bean-1");
    expect(search).toHaveValue("Finca Buena Vista (Colombia - Light)");
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ entityType: "bean", rkey: "bean-1" }),
    );
  });

  it("creates an entity with JSON and refreshes the app cache", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ rkey: "grinder-1", name: "Ode" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const appCache = installAppCache({ grinders: [] });

    const { container } = render(EntityCombo, {
      props: {
        entityType: "grinder",
        apiEndpoint: "/api/grinders",
        suggestEndpoint: "",
        inputName: "grinder_rkey",
        placeholder: "Search grinders...",
        sectionLabel: "Your grinders",
      },
    });

    await user.type(screen.getByRole("combobox"), "Ode");
    await user.click(screen.getByRole("option", { name: 'Create "Ode"' }));
    await user.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/grinders",
      expect.objectContaining({
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: "Ode",
          grinder_type: "",
          burr_type: "",
          link: "",
        }),
      }),
    );
    await waitFor(() =>
      expect(appCache?.invalidateAndRefresh).toHaveBeenCalledOnce(),
    );
    expect(
      container.querySelector<HTMLInputElement>('input[name="grinder_rkey"]'),
    ).toHaveValue("grinder-1");
  });
});
