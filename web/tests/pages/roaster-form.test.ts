import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { get } from "svelte/store";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { goto } from "$app/navigation";
import RoasterForm from "../../src/lib/components/RoasterForm.svelte";
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
		await userEvent.type(screen.getByLabelText("Website"), "https://heart.example");
		await userEvent.click(screen.getByRole("button", { name: "Add Roaster" }));

		await waitFor(() => expect(goto).toHaveBeenCalledWith("/roasters/did%3Aplc%3Aalice/created-rkey"));
		expect(fetchMock).toHaveBeenCalledWith(
			"/api/roasters",
			expect.objectContaining({ method: "POST" }),
		);
	});

	it("prepopulates the edit form", () => {
		render(RoasterForm, { roaster: existing, isEdit: true });

		expect(screen.getByLabelText("Name")).toHaveValue("Heart Coffee");
		expect(screen.getByLabelText("Location")).toHaveValue("Portland, OR");
		expect(screen.getByLabelText("Website")).toHaveValue("https://heart.example");
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

		await waitFor(() => expect(goto).toHaveBeenCalledWith("/roasters/did%3Aplc%3Aalice/r1"));
		expect(fetchMock).toHaveBeenCalledWith(
			"/api/roasters/r1",
			expect.objectContaining({ method: "PUT" }),
		);
	});

	it("shows a server validation error and retains user input", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(
				new Response(JSON.stringify({ error: "Name is already in use", code: "validation_failed" }), {
					status: 400,
					headers: { "Content-Type": "application/json" },
				}),
			),
		);
		render(RoasterForm, { roaster: null, isEdit: false });

		await userEvent.type(screen.getByLabelText("Name"), "Heart Coffee");
		await userEvent.type(screen.getByLabelText("Location"), "Portland, OR");
		await userEvent.click(screen.getByRole("button", { name: "Add Roaster" }));

		await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Name is already in use"));
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

		await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Unable to reach Arabica"));
		expect(website).toHaveValue("https://updated.example");
		expect(get(toasts).at(-1)?.message).toBe("Failed to update roaster");
	});
});
