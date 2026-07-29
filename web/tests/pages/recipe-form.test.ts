import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { get } from "svelte/store";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { goto } from "$app/navigation";
import RecipeForm from "../../src/lib/components/RecipeForm.svelte";
import { session } from "../../src/lib/stores/session";
import { clearToasts, toasts } from "../../src/lib/stores/toasts";
import type { Recipe } from "../../src/lib/types/entity_view";

vi.mock("$app/navigation", () => ({ goto: vi.fn() }));

vi.mock("../../src/lib/stores/appCache", () => ({
	appCache: { invalidateCache: vi.fn() },
}));

const existing: Recipe = {
	rkey: "r1",
	name: "V60 1:16 standard",
	brewer_rkey: "brewer-1",
	brewer_type: "pourover",
	coffee_amount: 15,
	water_amount: 240,
	notes: "Bright and sweet",
	created_at: "2026-07-09T12:00:00Z",
	brewer_obj: { rkey: "brewer-1", name: "V60", brewer_type: "pourover", description: "", link: "", created_at: "" },
};

describe("RecipeForm", () => {
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
		render(RecipeForm, { recipe: null, isEdit: false });

		await userEvent.click(screen.getByRole("button", { name: "Add Recipe" }));

		const name = screen.getByLabelText("Name");
		expect(name).toHaveAttribute("aria-invalid", "true");
		expect(screen.getByRole("alert")).toHaveTextContent("Name is required");
	});

	it("creates a recipe with JSON and navigates to its canonical detail route", async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ ...existing, rkey: "created-rkey" }), {
				status: 200,
				headers: { "Content-Type": "application/json" },
			}),
		);
		vi.stubGlobal("fetch", fetchMock);
		render(RecipeForm, { recipe: null, isEdit: false });

		await userEvent.type(screen.getByLabelText("Name"), "V60 1:16 standard");
		await userEvent.click(screen.getByRole("button", { name: "Add Recipe" }));

		await waitFor(() => expect(goto).toHaveBeenCalledWith("/recipes/did%3Aplc%3Aalice/created-rkey"));
		expect(fetchMock).toHaveBeenCalledWith(
			"/api/recipes",
			expect.objectContaining({ method: "POST" }),
		);
	});

	it("prepopulates the edit form", () => {
		render(RecipeForm, { recipe: existing, isEdit: true });

		expect(screen.getByLabelText("Name")).toHaveValue("V60 1:16 standard");
		expect(screen.getByLabelText("Coffee amount in grams")).toHaveValue(15);
		expect(screen.getByLabelText("Water amount in grams")).toHaveValue(240);
		expect(screen.getByLabelText("Notes")).toHaveValue("Bright and sweet");
		expect(screen.getByRole("combobox", { name: "Brewer" })).toHaveValue("V60");
		expect(document.querySelector('input[name="brewer_rkey"]')?.getAttribute("value")).toBe("brewer-1");
	});

	it("renders all brewer type options", () => {
		render(RecipeForm, { recipe: existing, isEdit: true });

		const brewerType = screen.getByLabelText("Brewer type");
		expect(brewerType).toBeInTheDocument();
		for (const label of [
			"Pour-over",
			"Espresso",
			"Immersion",
			"Moka Pot",
			"Cold Brew",
			"Cupping",
			"Other",
		]) {
			expect(brewerType).toContainHTML(label);
		}
	});

	it("updates a recipe and returns to its canonical detail route", async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ ...existing, notes: "Updated notes" }), {
				status: 200,
				headers: { "Content-Type": "application/json" },
			}),
		);
		vi.stubGlobal("fetch", fetchMock);
		render(RecipeForm, { recipe: existing, isEdit: true });

		const notes = screen.getByLabelText("Notes");
		await userEvent.clear(notes);
		await userEvent.type(notes, "Updated notes");
		await userEvent.click(screen.getByRole("button", { name: "Save Changes" }));

		await waitFor(() => expect(goto).toHaveBeenCalledWith("/recipes/did%3Aplc%3Aalice/r1"));
		expect(fetchMock).toHaveBeenCalledWith(
			"/api/recipes/r1",
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
		render(RecipeForm, { recipe: null, isEdit: false });

		await userEvent.type(screen.getByLabelText("Name"), "V60 1:16 standard");
		await userEvent.click(screen.getByRole("button", { name: "Add Recipe" }));

		await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Name is already in use"));
		expect(screen.getByLabelText("Name")).toHaveValue("V60 1:16 standard");
		expect(goto).not.toHaveBeenCalled();
	});

	it("retains input and reports a failed update", async () => {
		vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("offline")));
		render(RecipeForm, { recipe: existing, isEdit: true });

		const notes = screen.getByLabelText("Notes");
		await userEvent.clear(notes);
		await userEvent.type(notes, "Updated notes");
		await userEvent.click(screen.getByRole("button", { name: "Save Changes" }));

		await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Unable to reach Arabica"));
		expect(notes).toHaveValue("Updated notes");
		expect(get(toasts).at(-1)?.message).toBe("Failed to update recipe");
	});
});
