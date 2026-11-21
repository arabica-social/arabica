import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { get } from "svelte/store";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { goto } from "$app/navigation";
import RoasterForm from "../../src/lib/components/forms/RoasterForm.svelte";
import { session } from "../../src/lib/stores/session";
import { clearToasts, toasts } from "../../src/lib/stores/toasts";
import type { Roaster } from "../../src/lib/types/entity_view";

vi.mock("$app/navigation", () => ({ goto: vi.fn() }));

vi.mock("../../src/lib/stores/appCache", () => ({
  appCache: { invalidateCache: vi.fn() },
}));

const existing: Roaster = {
  rkey: "r1",
  name: "Heart Coffee",
  location: "Portland, OR",
  website: "https://heart.example",
  source_ref: "at://did:plc:bob/social.arabica.alpha.roaster/source-rkey",
  created_at: "2026-07-09T12:00:00Z",
};

describe("RoasterForm", () => {
  beforeEach(() => {
    session.set({
      did: "did:plc:alice",
      handle: "alice.test",
      displayName: "Alice",
      avatar: "",
      isAuthenticated: true,
      isModerator: false,
      unreadNotifications: 0,
    });
    vi.mocked(goto).mockReset();
    clearToasts();
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("keeps the task and fields ahead of redundant form chrome", () => {
    render(RoasterForm, { roaster: null, isEdit: false });
    expect(
      screen.queryByRole("complementary", { name: "Form context" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(
        /record completeness|fields recorded|useful details recorded/i,
      ),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/beans and drinks|attach them to beans and drinks/i),
    ).not.toBeInTheDocument();
    expect(screen.getByLabelText("Name")).toBeRequired();
  });

  it("shows accessible feedback when the required name is empty", async () => {
    render(RoasterForm, { roaster: null, isEdit: false });

    await userEvent.click(screen.getByRole("button", { name: "Add Roaster" }));

    const name = screen.getByLabelText("Name");
    expect(name).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByRole("alert")).toHaveTextContent("Name is required");
  });

  it("creates a roaster with JSON and navigates to its canonical detail route", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ ...existing, rkey: "created-rkey" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);
    render(RoasterForm, { roaster: null, isEdit: false });

    await userEvent.type(screen.getByLabelText("Name"), "Heart Coffee");
    await userEvent.type(screen.getByLabelText("Location"), "Portland, OR");
    await userEvent.type(
      screen.getByLabelText("Website"),
      "https://heart.example",
    );
    await userEvent.click(screen.getByRole("button", { name: "Add Roaster" }));

    await waitFor(() =>
      expect(goto).toHaveBeenCalledWith(
        "/roasters/did%3Aplc%3Aalice/created-rkey",
      ),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/roasters",
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("autofills a selected community roaster and preserves manual overrides", async () => {
    const suggestion = {
      name: "Counter Culture Coffee",
      source_uri:
        "at://did:plc:bob/social.arabica.alpha.roaster/counter-culture",
      fields: {
        name: "Counter Culture Coffee",
        location: "Durham, NC",
        website: "https://counterculturecoffee.com",
      },
      count: 3,
    };
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).startsWith("/api/suggestions/roasters")) {
        return Promise.resolve(
          new Response(JSON.stringify([suggestion]), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      return Promise.resolve(
        new Response(JSON.stringify({ ...existing, rkey: "created-rkey" }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    });
    vi.stubGlobal("fetch", fetchMock);
    render(RoasterForm, { roaster: null, isEdit: false });

    await userEvent.type(screen.getByLabelText("Name"), "Counter");
    await userEvent.click(
      await screen.findByRole("button", {
        name: /Counter Culture Coffee.*Durham, NC.*3 users/i,
      }),
    );

    expect(screen.getByLabelText("Name")).toHaveValue("Counter Culture Coffee");
    expect(screen.getByLabelText("Location")).toHaveValue("Durham, NC");
    expect(screen.getByLabelText("Website")).toHaveValue(
      "https://counterculturecoffee.com",
    );

    const website = screen.getByLabelText("Website");
    await userEvent.clear(website);
    await userEvent.type(website, "https://shop.example");
    await userEvent.click(screen.getByRole("button", { name: "Add Roaster" }));

    await waitFor(() =>
      expect(goto).toHaveBeenCalledWith(
        "/roasters/did%3Aplc%3Aalice/created-rkey",
      ),
    );
    const postCall = fetchMock.mock.calls.find(
      ([, init]) => init?.method === "POST",
    );
    expect(postCall).toBeDefined();
    expect(postCall?.[0]).toBe("/api/roasters");
    expect(JSON.parse(String(postCall?.[1]?.body))).toEqual({
      name: "Counter Culture Coffee",
      location: "Durham, NC",
      website: "https://shop.example",
      source_ref: suggestion.source_uri,
    });
  });

  it("clears prior suggestion autofill when a new result omits that field", async () => {
    const first = {
      name: "First Roaster",
      source_uri: "at://did:plc:bob/social.arabica.alpha.roaster/first",
      fields: {
        name: "First Roaster",
        location: "Portland, OR",
        website: "https://first.example",
      },
      count: 1,
    };
    const second = {
      name: "Second Roaster",
      source_uri: "at://did:plc:carol/social.arabica.alpha.roaster/second",
      fields: {
        name: "Second Roaster",
        location: "Seattle, WA",
      },
      count: 1,
    };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation(() =>
        Promise.resolve(
          new Response(JSON.stringify([first, second]), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        ),
      ),
    );
    render(RoasterForm, { roaster: null, isEdit: false });

    const name = screen.getByLabelText("Name");
    await userEvent.type(name, "First");
    await userEvent.click(
      await screen.findByRole("button", {
        name: /First Roaster.*Portland, OR/i,
      }),
    );
    expect(screen.getByLabelText("Website")).toHaveValue(
      "https://first.example",
    );

    await userEvent.clear(name);
    await userEvent.type(name, "Second");
    await userEvent.click(
      await screen.findByRole("button", {
        name: /Second Roaster.*Seattle, WA/i,
      }),
    );

    expect(screen.getByLabelText("Location")).toHaveValue("Seattle, WA");
    expect(screen.getByLabelText("Website")).toHaveValue("");
  });

  it("drops selected provenance and stale options when the suggested name is edited", async () => {
    const suggestion = {
      name: "Heart Coffee",
      source_uri: "at://did:plc:bob/social.arabica.alpha.roaster/heart",
      fields: {
        name: "Heart Coffee",
        location: "Portland, OR",
        website: "https://heart.example",
      },
      count: 1,
    };
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).startsWith("/api/suggestions/roasters")) {
        return Promise.resolve(
          new Response(JSON.stringify([suggestion]), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      return Promise.resolve(
        new Response(JSON.stringify({ ...existing, rkey: "created-rkey" }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    });
    vi.stubGlobal("fetch", fetchMock);
    render(RoasterForm, { roaster: null, isEdit: false });

    const user = userEvent.setup();
    const name = screen.getByLabelText("Name");
    await user.type(name, "Heart");
    const option = await screen.findByRole("button", {
      name: /Heart Coffee.*Portland, OR/i,
    });
    await user.tab();
    expect(option).toHaveFocus();
    await user.keyboard("{Enter}");
    await user.type(name, " Roasters");

    expect(
      screen.queryByRole("button", { name: /Heart Coffee.*Portland, OR/i }),
    ).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Add Roaster" }));

    await waitFor(() =>
      expect(goto).toHaveBeenCalledWith(
        "/roasters/did%3Aplc%3Aalice/created-rkey",
      ),
    );
    const postCall = fetchMock.mock.calls.find(
      ([, init]) => init?.method === "POST",
    );
    expect(JSON.parse(String(postCall?.[1]?.body))).toEqual({
      name: "Heart Coffee Roasters",
      location: "Portland, OR",
      website: "https://heart.example",
    });
  });

  it("prepopulates the edit form", () => {
    render(RoasterForm, { roaster: existing, isEdit: true });

    expect(screen.getByLabelText("Name")).toHaveValue("Heart Coffee");
    expect(screen.getByLabelText("Location")).toHaveValue("Portland, OR");
    expect(screen.getByLabelText("Website")).toHaveValue(
      "https://heart.example",
    );
  });

  it("updates a roaster and returns to its canonical detail route", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ ...existing, location: "Seattle, WA" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);
    render(RoasterForm, { roaster: existing, isEdit: true });

    const location = screen.getByLabelText("Location");
    await userEvent.clear(location);
    await userEvent.type(location, "Seattle, WA");
    await userEvent.click(screen.getByRole("button", { name: "Save Changes" }));

    await waitFor(() =>
      expect(goto).toHaveBeenCalledWith("/roasters/did%3Aplc%3Aalice/r1"),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/roasters/r1",
      expect.objectContaining({ method: "PUT" }),
    );
    const [, request] = fetchMock.mock.calls[0];
    expect(JSON.parse(request.body)).toMatchObject({
      source_ref: existing.source_ref,
    });
  });

  it("shows a server validation error and retains user input", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: "Name is already in use",
            code: "validation_failed",
          }),
          {
            status: 400,
            headers: { "Content-Type": "application/json" },
          },
        ),
      ),
    );
    render(RoasterForm, { roaster: null, isEdit: false });

    await userEvent.type(screen.getByLabelText("Name"), "Heart Coffee");
    await userEvent.type(screen.getByLabelText("Location"), "Portland, OR");
    await userEvent.click(screen.getByRole("button", { name: "Add Roaster" }));

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(
        "Name is already in use",
      ),
    );
    expect(screen.getByLabelText("Name")).toHaveValue("Heart Coffee");
    expect(screen.getByLabelText("Location")).toHaveValue("Portland, OR");
    expect(goto).not.toHaveBeenCalled();
  });

  it("retains input and reports a failed update", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("offline")));
    render(RoasterForm, { roaster: existing, isEdit: true });

    const website = screen.getByLabelText("Website");
    await userEvent.clear(website);
    await userEvent.type(website, "https://updated.example");
    await userEvent.click(screen.getByRole("button", { name: "Save Changes" }));

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(
        "Unable to reach Arabica",
      ),
    );
    expect(website).toHaveValue("https://updated.example");
    expect(get(toasts).at(-1)?.message).toBe("Failed to update roaster");
  });
});
