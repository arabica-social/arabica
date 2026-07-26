import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { get } from "svelte/store";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { goto } from "$app/navigation";
import BeanForm from "../../src/lib/components/BeanForm.svelte";
import { session } from "../../src/lib/stores/session";
import { clearToasts, toasts } from "../../src/lib/stores/toasts";
import type { Bean } from "../../src/lib/types/entity_view";

vi.mock("$app/navigation", () => ({ goto: vi.fn() }));

vi.mock("../../src/lib/stores/appCache", () => ({
	appCache: { invalidateCache: vi.fn() },
}));

const existing: Bean = {
	rkey: "r1",
	name: "Ethiopia Gedeb",
	origin: "Ethiopia, Gedeb",
	variety: "Gesha",
	roast_level: "Light",
	roast_date: "2026-07-01",
	process: "Washed",
	description: "Floral and bright",
	notes: "A favorite",
	link: "https://roaster.example/beans/gedeb",
	rating: 8,
	closed: false,
	created_at: "2026-07-09T12:00:00Z",
	roaster: { name: "Heart Coffee", location: "Portland, OR", rkey: "roaster1" },
};

describe("BeanForm", () => {
	beforeEach(() => {
		session.set({
			did: "did:plc:alice",
			handle: "alice.test",
			displayName: "Alice",
			avatar: "",
			isAuthenticated: true,
			isModerator: false,
			unreadNotifications: 0,
			temperatureUnit: "celsius",
		});
		vi.mocked(goto).mockReset();
		clearToasts();
	});

	afterEach(() => {
		cleanup();
		vi.unstubAllGlobals();
	});

	it("shows accessible feedback when the required name is empty", async () => {
		// Edit mode renders a plain Name input (create mode uses NameSuggest).
		render(BeanForm, { bean: existing, isEdit: true });

		const name = screen.getByLabelText("Name");
		await userEvent.clear(name);
		await userEvent.click(screen.getByRole("button", { name: "Save Changes" }));

		expect(name).toHaveAttribute("aria-invalid", "true");
		expect(screen.getByRole("alert")).toHaveTextContent("Name is required");
	});

	it("shows accessible feedback when the required origin is empty", async () => {
		render(BeanForm, { bean: existing, isEdit: true });

		const origin = screen.getByLabelText("Origin");
		await userEvent.clear(origin);
		await userEvent.click(screen.getByRole("button", { name: "Save Changes" }));

		expect(origin).toHaveAttribute("aria-invalid", "true");
		expect(screen.getByRole("alert")).toHaveTextContent("Origin is required");
	});

	it("creates a bean with JSON and navigates to its canonical detail route", async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ ...existing, rkey: "created-rkey" }), {
				status: 200,
				headers: { "Content-Type": "application/json" },
			}),
		);
		vi.stubGlobal("fetch", fetchMock);
		render(BeanForm, { bean: null, isEdit: false });

		expect(screen.queryByLabelText("Rating")).not.toBeInTheDocument();
		await userEvent.type(screen.getByLabelText("Name"), "Ethiopia Gedeb");
		await userEvent.type(screen.getByLabelText("Origin"), "Ethiopia, Gedeb");
		await userEvent.click(screen.getByRole("button", { name: "Add Bean" }));

		await waitFor(() => expect(goto).toHaveBeenCalledWith("/beans/did%3Aplc%3Aalice/created-rkey"));
		expect(fetchMock).toHaveBeenCalledWith(
			"/api/beans",
			expect.objectContaining({ method: "POST" }),
		);
		const [, request] = fetchMock.mock.calls[0];
		expect(JSON.parse(request.body)).not.toHaveProperty("rating");
	});

	it("only includes a rating after the user adds one", async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ ...existing, rkey: "created-rkey" }), {
				status: 200,
				headers: { "Content-Type": "application/json" },
			}),
		);
		vi.stubGlobal("fetch", fetchMock);
		render(BeanForm, { bean: null, isEdit: false });

		await userEvent.type(screen.getByLabelText("Name"), "Ethiopia Gedeb");
		await userEvent.type(screen.getByLabelText("Origin"), "Ethiopia, Gedeb");
		await userEvent.click(
			screen.getByText(/^Personal details/, { selector: "summary span" }),
		);
		await userEvent.click(screen.getByRole("button", { name: "Add rating" }));
		expect(screen.getByLabelText("Rating")).toHaveValue("5");
		await userEvent.click(screen.getByRole("button", { name: "Add Bean" }));

		await waitFor(() => expect(goto).toHaveBeenCalledWith("/beans/did%3Aplc%3Aalice/created-rkey"));
		const [, request] = fetchMock.mock.calls[0];
		expect(JSON.parse(request.body)).toMatchObject({ rating: 5 });
	});

	it("prepopulates the edit form", () => {
		render(BeanForm, { bean: existing, isEdit: true });

		expect(screen.getByLabelText("Name")).toHaveValue("Ethiopia Gedeb");
		expect(screen.getByLabelText("Origin")).toHaveValue("Ethiopia, Gedeb");
		expect(screen.getByLabelText("Variety")).toHaveValue("Gesha");
		expect(screen.getByLabelText("Process")).toHaveValue("Washed");
		expect(screen.getByLabelText("Roast level")).toHaveValue("Light");
		expect(screen.getByLabelText("Roast date")).toHaveValue("2026-07-01");
		expect(screen.getByLabelText("Link")).toHaveValue("https://roaster.example/beans/gedeb");
	});

	it("keeps the roaster selected when editing an existing bean", () => {
		render(BeanForm, { bean: existing, isEdit: true });

		// The visible search field shows the roaster name, and the hidden
		// roaster_rkey input retains the selected roaster's rkey.
		expect(screen.getByLabelText("Roaster")).toHaveValue("Heart Coffee");
		expect(screen.getByDisplayValue("roaster1")).toHaveAttribute(
			"name",
			"roaster_rkey",
		);
	});

	it("renders all roast level options and the closed checkbox", () => {
		render(BeanForm, { bean: existing, isEdit: true });

		const roastLevel = screen.getByLabelText("Roast level");
		expect(roastLevel).toBeInTheDocument();
		for (const level of [
			"Ultra-Light",
			"Light",
			"Medium-Light",
			"Medium",
			"Medium-Dark",
			"Dark",
		]) {
			expect(roastLevel).toContainHTML(level);
		}

		const closed = screen.getByLabelText("Bag is closed/finished");
		expect(closed).toHaveAttribute("type", "checkbox");
		expect(closed).not.toBeChecked();
	});

	it("updates a bean and returns to its canonical detail route", async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ ...existing, notes: "Updated notes" }), {
				status: 200,
				headers: { "Content-Type": "application/json" },
			}),
		);
		vi.stubGlobal("fetch", fetchMock);
		render(BeanForm, { bean: existing, isEdit: true });

		const notes = screen.getByLabelText("Personal notes");
		await userEvent.clear(notes);
		await userEvent.type(notes, "Updated notes");
		await userEvent.click(screen.getByRole("button", { name: "Save Changes" }));

		await waitFor(() => expect(goto).toHaveBeenCalledWith("/beans/did%3Aplc%3Aalice/r1"));
		expect(fetchMock).toHaveBeenCalledWith(
			"/api/beans/r1",
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
		render(BeanForm, { bean: null, isEdit: false });

		await userEvent.type(screen.getByLabelText("Name"), "Ethiopia Gedeb");
		await userEvent.type(screen.getByLabelText("Origin"), "Ethiopia, Gedeb");
		await userEvent.click(screen.getByRole("button", { name: "Add Bean" }));

		await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Name is already in use"));
		expect(screen.getByLabelText("Origin")).toHaveValue("Ethiopia, Gedeb");
		expect(goto).not.toHaveBeenCalled();
	});

	it("retains input and reports a failed update", async () => {
		vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("offline")));
		render(BeanForm, { bean: existing, isEdit: true });

		const link = screen.getByLabelText("Link");
		await userEvent.clear(link);
		await userEvent.type(link, "https://updated.example");
		await userEvent.click(screen.getByRole("button", { name: "Save Changes" }));

		await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Unable to reach Arabica"));
		expect(link).toHaveValue("https://updated.example");
		expect(get(toasts).at(-1)?.message).toBe("Failed to update bean");
	});
});
