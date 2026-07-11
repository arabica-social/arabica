import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { get } from "svelte/store";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { goto } from "$app/navigation";
import BrewerForm from "../../src/lib/components/BrewerForm.svelte";
import { session } from "../../src/lib/stores/session";
import { clearToasts, toasts } from "../../src/lib/stores/toasts";
import type { Brewer } from "../../src/lib/types/entity_view";

vi.mock("$app/navigation", () => ({ goto: vi.fn() }));

vi.mock("../../src/lib/stores/appCache", () => ({
	appCache: { invalidateCache: vi.fn() },
}));

const existing: Brewer = {
	rkey: "r1",
	name: "Hario V60-02",
	brewer_type: "pourover",
	description: "Pour-over dripper",
	link: "https://hario.example",
	created_at: "2026-07-09T12:00:00Z",
};

describe("BrewerForm", () => {
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
		render(BrewerForm, { brewer: null, isEdit: false });

		await userEvent.click(screen.getByRole("button", { name: "Add Brewer" }));

		const name = screen.getByLabelText("Name");
		expect(name).toHaveAttribute("aria-invalid", "true");
		expect(screen.getByRole("alert")).toHaveTextContent("Name is required");
	});

	it("creates a brewer with JSON and navigates to its canonical detail route", async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ ...existing, rkey: "created-rkey" }), {
				status: 200,
				headers: { "Content-Type": "application/json" },
			}),
		);
		vi.stubGlobal("fetch", fetchMock);
		render(BrewerForm, { brewer: null, isEdit: false });

		await userEvent.type(screen.getByLabelText("Name"), "Hario V60-02");
		await userEvent.type(screen.getByLabelText("Description"), "Pour-over dripper");
		await userEvent.type(screen.getByLabelText("Link"), "https://hario.example");
		await userEvent.click(screen.getByRole("button", { name: "Add Brewer" }));

		await waitFor(() => expect(goto).toHaveBeenCalledWith("/brewers/did%3Aplc%3Aalice/created-rkey"));
		expect(fetchMock).toHaveBeenCalledWith(
			"/api/brewers",
			expect.objectContaining({ method: "POST" }),
		);
	});

	it("prepopulates the edit form", () => {
		render(BrewerForm, { brewer: existing, isEdit: true });

		expect(screen.getByLabelText("Name")).toHaveValue("Hario V60-02");
		expect(screen.getByLabelText("Type")).toHaveValue("pourover");
		expect(screen.getByLabelText("Description")).toHaveValue("Pour-over dripper");
		expect(screen.getByLabelText("Link")).toHaveValue("https://hario.example");
	});

	it("updates a brewer and returns to its canonical detail route", async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ ...existing, description: "Updated description" }), {
				status: 200,
				headers: { "Content-Type": "application/json" },
			}),
		);
		vi.stubGlobal("fetch", fetchMock);
		render(BrewerForm, { brewer: existing, isEdit: true });

		const description = screen.getByLabelText("Description");
		await userEvent.clear(description);
		await userEvent.type(description, "Updated description");
		await userEvent.click(screen.getByRole("button", { name: "Save Changes" }));

		await waitFor(() => expect(goto).toHaveBeenCalledWith("/brewers/did%3Aplc%3Aalice/r1"));
		expect(fetchMock).toHaveBeenCalledWith(
			"/api/brewers/r1",
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
		render(BrewerForm, { brewer: null, isEdit: false });

		await userEvent.type(screen.getByLabelText("Name"), "Hario V60-02");
		await userEvent.type(screen.getByLabelText("Description"), "Pour-over dripper");
		await userEvent.click(screen.getByRole("button", { name: "Add Brewer" }));

		await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Name is already in use"));
		expect(screen.getByLabelText("Name")).toHaveValue("Hario V60-02");
		expect(screen.getByLabelText("Description")).toHaveValue("Pour-over dripper");
		expect(goto).not.toHaveBeenCalled();
	});

	it("retains input and reports a failed update", async () => {
		vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("offline")));
		render(BrewerForm, { brewer: existing, isEdit: true });

		const link = screen.getByLabelText("Link");
		await userEvent.clear(link);
		await userEvent.type(link, "https://updated.example");
		await userEvent.click(screen.getByRole("button", { name: "Save Changes" }));

		await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Unable to reach Arabica"));
		expect(link).toHaveValue("https://updated.example");
		expect(get(toasts).at(-1)?.message).toBe("Failed to update brewer");
	});
});
