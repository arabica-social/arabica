import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import ProfilePage from "../../src/routes/profile/[actor]/+page.svelte";

const profileData = {
	profile: {
		notifications: null,
		error: "",
		actor: "alice.test",
		isAuthenticated: true,
		profile: {
			profile: { handle: "alice.test", display_name: "Alice", avatar: "" },
			did: "did:plc:abc",
			is_own_profile: true,
			is_authenticated: true,
			is_app_user: true,
			brews: [
				{
					rkey: "brew-1",
					bean_rkey: "bean-1",
					recipe_rkey: "",
					temperature: 205,
					water_amount: 250,
					coffee_amount: 15,
					time_seconds: 180,
					grind_size: "Medium",
					grinder_rkey: "",
					brewer_rkey: "",
					tasting_notes: "Bright and sweet",
					rating: 8,
					created_at: "2026-01-20T10:00:00Z",
					bean: { rkey: "bean-1", name: "Finca Buena Vista", origin: "Colombia", variety: "", roast_level: "Light", roast_date: "", process: "", description: "", notes: "", link: "", closed: false, created_at: "", roaster: { name: "Heart", location: "", rkey: "r1" } },
					brewer_obj: { rkey: "brewer-1", name: "V60", brewer_type: "pourover", description: "", link: "", created_at: "" },
				},
			],
			total_brews: 1,
			brews_has_more: false,
			brews_next_offset: 25,
			beans: [
				{ rkey: "bean-1", name: "Finca Buena Vista", origin: "Colombia", variety: "", roast_level: "Light", roast_date: "", process: "Washed", description: "", notes: "", link: "", rating: 8, closed: false, created_at: "", roaster: { name: "Heart", location: "", rkey: "r1" } },
				{ rkey: "bean-2", name: "Old Bag", origin: "Brazil", variety: "", roast_level: "Dark", roast_date: "", process: "", description: "", notes: "", link: "", closed: true, created_at: "" },
			],
			roasters: [
				{ rkey: "roaster-1", name: "Heart Coffee", location: "Portland, OR", website: "", created_at: "" },
			],
			grinders: [
				{ rkey: "grinder-1", name: "Ode 2", grinder_type: "Electric", burr_type: "Flat", notes: "", link: "", created_at: "" },
			],
			brewers: [
				{ rkey: "brewer-1", name: "V60", brewer_type: "pourover", description: "", link: "", created_at: "" },
			],
			brew_like_counts: {},
			brew_comment_counts: {},
			brew_liked_by_user: {},
			brew_cids: {},
			bean_brew_counts: {},
			grinder_brew_counts: {},
			brewer_brew_counts: {},
			roaster_bean_counts: {},
			bean_avg_brew_ratings: {},
			roaster_avg_brew_ratings: {},
		},
	},
};

const testData = {
	profile: profileData.profile.profile,
	error: "",
	actor: "alice.test",
	isAuthenticated: true,
};

describe("Profile page", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the profile header with display name and handle", () => {
		render(ProfilePage, { data: testData });
		expect(screen.getByText("Alice")).toBeTruthy();
		expect(screen.getByText("@alice.test")).toBeTruthy();
	});

	it("renders the stats grid", () => {
		render(ProfilePage, { data: testData });
		expect(screen.getAllByText("Brews").length).toBeGreaterThan(0);
		expect(screen.getAllByText("Beans").length).toBeGreaterThan(0);
		expect(screen.getAllByText("Roasters").length).toBeGreaterThan(0);
		expect(screen.getAllByText("Grinders").length).toBeGreaterThan(0);
		expect(screen.getAllByText("Brewers").length).toBeGreaterThan(0);
	});

	it("shows the brews tab by default with brew cards", () => {
		render(ProfilePage, { data: testData });
		expect(screen.getByText("Finca Buena Vista")).toBeTruthy();
		expect(screen.getByText("Bright and sweet")).toBeTruthy();
	});

	it("renders the 404 error state", () => {
		render(ProfilePage, { data: { profile: null, error: "Profile not found", actor: "x", isAuthenticated: false } });
		expect(screen.getByText("404")).toBeTruthy();
		expect(screen.getByText("Profile not found")).toBeTruthy();
	});

	it("links brew cards to the brew view", () => {
		render(ProfilePage, { data: testData });
		const link = screen.getByText("Finca Buena Vista").closest("a");
		expect(link).toHaveAttribute("href", "/brews/did:plc:abc/brew-1");
	});
});
