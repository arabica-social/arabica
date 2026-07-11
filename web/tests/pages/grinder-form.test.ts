import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { get } from "svelte/store";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { goto } from "$app/navigation";
import GrinderForm from "../../src/lib/components/GrinderForm.svelte";
import { session } from "../../src/lib/stores/session";
import { clearToasts, toasts } from "../../src/lib/stores/toasts";
import type { Grinder } from "../../src/lib/types/entity_view";

vi.mock("$app/navigation", () => ({ goto: vi.fn() }));

vi.mock("../../src/lib/stores/appCache", () => ({
	appCache: { invalidateCache: vi.fn() },
}));

const existing: Grinder = {
	rkey: "r1",
	name: "Comandante C40",
	grinder_type: "Hand",
	burr_type: "Conical",
	notes: "Great grinder",
	link: "https://comandante.example",
	created_at: "2026-07-09T12:00:00Z",
};

describe("GrinderForm", () => {
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
		render(GrinderForm, { grinder: null, isEdit: false });

		await userEvent.click(screen.getByRole("button", { name: "Add Grinder" }));

		const name = screen.getByLabelText("Name");
		expect(name).toHaveAttribute("aria-invalid", "true");
		expect(screen.getByRole("alert")).toHaveTextContent("Name is required");
	});

	it("creates a grinder with JSON and navigates to its canonical detail route", async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ ...existing, rkey: "created-rkey" }), {
				status: 200,
				headers: { "Content-Type": "application/json" },
			}),
		);
		vi.stubGlobal("fetch", fetchMock);
		render(GrinderForm, { grinder: null, isEdit: false });

		await userEvent.type(screen.getByLabelText("Name"), "Comandante C40");
		await userEvent.type(screen.getByLabelText("Notes"), "Great grinder");
		await userEvent.type(screen.getByLabelText("Link"), "https://comandante.example");
		await userEvent.click(screen.getByRole("button", { name: "Add Grinder" }));

		await waitFor(() => expect(goto).toHaveBeenCalledWith("/grinders/did%3Aplc%3Aalice/created-rkey"));
		expect(fetchMock).toHaveBeenCalledWith(
			"/api/grinders",
			expect.objectContaining({ method: "POST" }),
		);
	});

	it("prepopulates the edit form", () => {
		render(GrinderForm, { grinder: existing, isEdit: true });

		expect(screen.getByLabelText("Name")).toHaveValue("Comandante C40");
		expect(screen.getByLabelText("Type")).toHaveValue("Hand");
		expect(screen.getByLabelText("Burr type")).toHaveValue("Conical");
		expect(screen.getByLabelText("Notes")).toHaveValue("Great grinder");
		expect(screen.getByLabelText("Link")).toHaveValue("https://comandante.example");
	});

	it("updates a grinder and returns to its canonical detail route", async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ ...existing, notes: "Updated notes" }), {
				status: 200,
				headers: { "Content-Type": "application/json" },
			}),
		);
		vi.stubGlobal("fetch", fetchMock);
		render(GrinderForm, { grinder: existing, isEdit: true });

		const notes = screen.getByLabelText("Notes");
		await userEvent.clear(notes);
		await userEvent.type(notes, "Updated notes");
		await userEvent.click(screen.getByRole("button", { name: "Save Changes" }));

		await waitFor(() => expect(goto).toHaveBeenCalledWith("/grinders/did%3Aplc%3Aalice/r1"));
		expect(fetchMock).toHaveBeenCalledWith(
			"/api/grinders/r1",
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
		render(GrinderForm, { grinder: null, isEdit: false });

		await userEvent.type(screen.getByLabelText("Name"), "Comandante C40");
		await userEvent.type(screen.getByLabelText("Notes"), "Great grinder");
		await userEvent.click(screen.getByRole("button", { name: "Add Grinder" }));

		await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Name is already in use"));
		expect(screen.getByLabelText("Name")).toHaveValue("Comandante C40");
		expect(screen.getByLabelText("Notes")).toHaveValue("Great grinder");
		expect(goto).not.toHaveBeenCalled();
	});

	it("retains input and reports a failed update", async () => {
		vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("offline")));
		render(GrinderForm, { grinder: existing, isEdit: true });

		const link = screen.getByLabelText("Link");
		await userEvent.clear(link);
		await userEvent.type(link, "https://updated.example");
		await userEvent.click(screen.getByRole("button", { name: "Save Changes" }));

		await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Unable to reach Arabica"));
		expect(link).toHaveValue("https://updated.example");
		expect(get(toasts).at(-1)?.message).toBe("Failed to update grinder");
	});
});
