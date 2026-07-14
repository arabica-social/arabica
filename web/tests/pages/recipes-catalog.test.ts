import { cleanup, render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import RecipesCatalog from "../../src/routes/recipes/+page.svelte";
import type { Recipe } from "$lib/types/entity_view";

const mockRecipe1: Recipe = {
	rkey: "recipe-1",
	name: "V60 Standard",
	brewer_rkey: "brewer-1",
	brewer_type: "pourover",
	coffee_amount: 15,
	water_amount: 250,
	notes: "Bloom for 30s, then pulse pour.",
	created_at: "2026-01-10T10:00:00Z",
	brewer_obj: { rkey: "brewer-1", name: "V60", brewer_type: "pourover", description: "", link: "", created_at: "" },
	pours: [{ pour_number: 1, water_amount: 50, time_seconds: 30 }],
	ratio: 16.7,
	author_did: "did:plc:abc",
	author_handle: "alice.test",
	author_display: "Alice",
	brew_count: 3,
	fork_count: 1,
};

const mockRecipe2: Recipe = {
	rkey: "recipe-2",
	name: "French Press Bold",
	brewer_rkey: "brewer-2",
	brewer_type: "immersion",
	coffee_amount: 22,
	water_amount: 350,
	notes: "Steep for 4 minutes, then plunge.",
	created_at: "2026-01-12T10:00:00Z",
	brewer_obj: { rkey: "brewer-2", name: "French Press", brewer_type: "immersion", description: "", link: "", created_at: "" },
	ratio: 15.9,
	author_did: "did:plc:def",
	author_handle: "bob.test",
	author_display: "Bob",
	brew_count: 0,
	fork_count: 0,
};

const catalogData = {
	recipes: [mockRecipe1, mockRecipe2],
	error: "",
	isAuthenticated: true,
};

describe("Recipes catalog page", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
		vi.unstubAllGlobals();
	});

	it("renders LedgerHeader with h1, eyebrow, description, and New Recipe action", () => {
		render(RecipesCatalog, { data: catalogData });
		const heading = screen.getByRole("heading", { level: 1, name: "Explore Recipes" });
		expect(heading).toBeTruthy();
		expect(screen.getByText("Community catalog")).toBeTruthy();
		expect(
			screen.getByText("Browse, compare, and fork brewing recipes shared by the Arabica community."),
		).toBeTruthy();
		const newRecipe = screen.getByRole("link", { name: /New Recipe/i });
		expect(newRecipe).toHaveAttribute("href", "/recipes/new");
	});

	it("shows the alpha disclosure as a native details/summary with expanded explanation", async () => {
		const user = userEvent.setup();
		render(RecipesCatalog, { data: catalogData });
		const disclosure = document.querySelector("details.recipe-alpha-notice") as HTMLDetailsElement | null;
		expect(disclosure).toBeTruthy();
		const summary = screen.getByText("Recipes are in early alpha").closest("summary");
		expect(summary).toBeTruthy();
		expect(disclosure?.open).toBe(false);
		await user.click(summary!);
		expect(disclosure?.open).toBe(true);
		expect(screen.getByText(/recipe format may change significantly/)).toBeTruthy();
	});

	it("renders search, category filters, brewer type, and amount controls", () => {
		render(RecipesCatalog, { data: catalogData });
		expect(screen.getByLabelText("Search recipes")).toBeTruthy();
		expect(screen.getByRole("group", { name: "Recipe size filters" })).toBeTruthy();
		expect(screen.getByLabelText("Brewer Type")).toBeTruthy();
		expect(screen.getByLabelText("Min coffee (g)")).toBeTruthy();
		expect(screen.getByLabelText("Max coffee (g)")).toBeTruthy();
	});

	it("renders the sort control group", () => {
		render(RecipesCatalog, { data: catalogData });
		expect(screen.getByRole("group", { name: "Sort recipes" })).toBeTruthy();
		expect(screen.getByRole("button", { name: "Popular" })).toBeTruthy();
		expect(screen.getByRole("button", { name: "Newest" })).toBeTruthy();
		expect(screen.getByRole("button", { name: "Most Forked" })).toBeTruthy();
	});

	it("displays the result count and recipe cards", () => {
		render(RecipesCatalog, { data: catalogData });
		expect(screen.getByText("2 recipes found")).toBeTruthy();
		expect(screen.getByText("V60 Standard")).toBeTruthy();
		expect(screen.getByText("French Press Bold")).toBeTruthy();
	});

	it("selects a recipe card and shows the detail panel", async () => {
		const user = userEvent.setup();
		render(RecipesCatalog, { data: catalogData });
		const card = screen.getByText("V60 Standard").closest("[role='button']");
		expect(card).toBeTruthy();
		await user.click(card!);
		expect(screen.getByText("Bloom for 30s, then pulse pour.")).toBeTruthy();
		expect(screen.getByRole("button", { name: "Close recipe details" })).toBeTruthy();
		expect(screen.getByText("Use in Brew")).toBeTruthy();
	});

	it("renders authentication required state", () => {
		render(RecipesCatalog, { data: { recipes: [], error: "Authentication required", isAuthenticated: false } });
		expect(screen.getByText("Authentication required")).toBeTruthy();
		expect(screen.getByRole("button", { name: "Log In" })).toBeTruthy();
		expect(screen.queryByRole("heading", { level: 1 })).toBeNull();
	});

	it("shows empty-state messaging when no recipes match", () => {
		render(RecipesCatalog, { data: { recipes: [], error: "", isAuthenticated: true } });
		expect(screen.getByText("No recipes match these filters")).toBeTruthy();
		expect(screen.getByText(/Try widening the search/)).toBeTruthy();
	});
});
