import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import RecipeView from "../../src/routes/recipes/[actor]/[id]/+page.svelte";

const recipeData = {
	data: {
		record: {
			rkey: "recipe-1",
			name: "V60 Standard",
			brewer_rkey: "brewer-1",
			brewer_type: "pourover",
			coffee_amount: 15,
			water_amount: 250,
			notes: "Bloom for 30s, then pulse pour",
			created_at: "2026-01-10T10:00:00Z",
			brewer_obj: { rkey: "brewer-1", name: "V60" },
			pours: [
				{ pour_number: 1, water_amount: 50, time_seconds: 30 },
			],
			ratio: 16.7,
		},
		subject_uri: "at://did:plc:abc/social.arabica.alpha.recipe/recipe-1",
		subject_cid: "bafy123",
		author: { did: "did:plc:abc", handle: "alice.test", display_name: "Alice" },
		social: {
			is_liked: true, like_count: 4, comment_count: 1, comments: [],
			is_moderator: false, can_hide_record: false, can_block_user: false, is_record_hidden: false,
		},
		backlinks: null,
		is_own_profile: false,
		is_authenticated: true,
		share_url: "/recipes/alice.test/recipe-1",
		entity_type: "recipe",
		entity_count: 0,
	},
	error: undefined,
	status: 200,
};

describe("Recipe view", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the recipe name", () => {
		render(RecipeView, { data: recipeData });
		expect(screen.getByText("V60 Standard")).toBeTruthy();
	});

	it("renders coffee and water amounts", () => {
		render(RecipeView, { data: recipeData });
		expect(screen.getByText("15.0g")).toBeTruthy();
		expect(screen.getByText("250.0g")).toBeTruthy();
	});

	it("renders the ratio", () => {
		render(RecipeView, { data: recipeData });
		expect(screen.getByText("1:16.7")).toBeTruthy();
	});

	it("renders the brewer link", () => {
		render(RecipeView, { data: recipeData });
		const brewerLink = screen.getByText("V60").closest("a");
		expect(brewerLink).toHaveAttribute("href", "/brewers/alice.test/brewer-1");
	});

	it("renders the pours section", () => {
		render(RecipeView, { data: recipeData });
		expect(screen.getByText("Pours")).toBeTruthy();
		expect(screen.getByText("50g")).toBeTruthy();
	});

	it("renders the notes", () => {
		render(RecipeView, { data: recipeData });
		expect(screen.getByText("Bloom for 30s, then pulse pour")).toBeTruthy();
	});

	it("shows Use in Brew button for authenticated users", () => {
		render(RecipeView, { data: recipeData });
		expect(screen.getByText("Use in Brew")).toBeTruthy();
	});

	it("shows Copy Recipe button for non-owners", () => {
		render(RecipeView, { data: recipeData });
		expect(screen.getByText("Copy Recipe")).toBeTruthy();
	});

	it("renders an error state", () => {
		render(RecipeView, { data: { error: "Record not found", status: 404 } });
		expect(screen.getByText("Record not found")).toBeTruthy();
	});
});
