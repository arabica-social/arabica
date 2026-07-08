import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import BrewView from "../../src/routes/brews/[actor]/[id]/+page.svelte";

const brewData = {
	data: {
		record: {
			rkey: "brew-1",
			bean_rkey: "bean-1",
			recipe_rkey: "",
			method: "Pour Over",
			temperature: 205,
			water_amount: 250,
			coffee_amount: 15,
			time_seconds: 180,
			grind_size: "Medium",
			grinder_rkey: "grinder-1",
			brewer_rkey: "brewer-1",
			tasting_notes: "Bright and sweet",
			rating: 8,
			created_at: "2026-01-20T10:00:00Z",
			bean: {
				rkey: "bean-1",
				name: "Finca Buena Vista",
				origin: "Colombia",
				roast_level: "Light",
				roaster: { name: "Heart", location: "Portland", rkey: "roaster-1" },
			},
			grinder_obj: { rkey: "grinder-1", name: "Ode 2" },
			brewer_obj: { rkey: "brewer-1", name: "V60" },
			pours: [
				{ pour_number: 1, water_amount: 50, time_seconds: 30 },
				{ pour_number: 2, water_amount: 100, time_seconds: 60 },
			],
			pourover_params: {
				bloom_water: 50,
				bloom_seconds: 30,
				drawdown_seconds: 45,
				bypass_water: 0,
				filter: "paper",
			},
		},
		subject_uri: "at://did:plc:abc/social.arabica.alpha.brew/brew-1",
		subject_cid: "bafy123",
		author: { did: "did:plc:abc", handle: "alice.test", display_name: "Alice" },
		social: {
			is_liked: false, like_count: 2, comment_count: 0, comments: [],
			is_moderator: false, can_hide_record: false, can_block_user: false, is_record_hidden: false,
		},
		backlinks: null,
		is_own_profile: true,
		is_authenticated: true,
		share_url: "/brews/alice.test/brew-1",
		entity_type: "brew",
		entity_count: 0,
	},
	error: undefined,
	status: 200,
};

describe("Brew view", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the bean name as title", () => {
		render(BrewView, { data: brewData });
		expect(screen.getAllByText("Finca Buena Vista").length).toBeGreaterThan(0);
	});

	it("renders the rating hero", () => {
		render(BrewView, { data: brewData });
		expect(screen.getByText("8")).toBeTruthy();
		expect(screen.getByText("/10")).toBeTruthy();
	});

	it("renders the ratio, time, and method summary stats", () => {
		render(BrewView, { data: brewData });
		expect(screen.getByText("1:16.7")).toBeTruthy();
		expect(screen.getAllByText("3m").length).toBeGreaterThan(0);
		expect(screen.getAllByText("V60").length).toBeGreaterThan(0);
	});

	it("renders the bean reference card with roaster link", () => {
		render(BrewView, { data: brewData });
		const beanLink = screen.getAllByText("Finca Buena Vista").find((el) => el.closest("a"));
		expect(beanLink?.closest("a")).toHaveAttribute("href", "/beans/alice.test/bean-1");
		expect(screen.getByText("Heart")).toBeTruthy();
	});

	it("renders the pours section", () => {
		render(BrewView, { data: brewData });
		expect(screen.getByText("Pours")).toBeTruthy();
		expect(screen.getByText("50g")).toBeTruthy();
		expect(screen.getByText("100g")).toBeTruthy();
	});

	it("renders tasting notes", () => {
		render(BrewView, { data: brewData });
		expect(screen.getByText("Bright and sweet")).toBeTruthy();
	});

	it("shows Save as Recipe button for owner when no recipe linked", () => {
		render(BrewView, { data: brewData });
		expect(screen.getByText("Save as Recipe")).toBeTruthy();
	});

	it("renders the bloom and filter", () => {
		render(BrewView, { data: brewData });
		expect(screen.getByText("50g for 30s")).toBeTruthy();
		expect(screen.getByText("paper")).toBeTruthy();
	});

	it("renders an error state", () => {
		render(BrewView, { data: { error: "Record not found", status: 404 } });
		expect(screen.getByText("Record not found")).toBeTruthy();
	});
});
