import { cleanup, render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it } from "vitest";
import MyCoffee from "../../src/routes/my-coffee/+page.svelte";

const manageData = {
	did: "did:plc:abc",
	beans: [
		{
			rkey: "bean-1",
			name: "Finca Buena Vista",
			origin: "Colombia",
			variety: "Bourbon",
			roast_level: "Light",
			roast_date: "",
			process: "Washed",
			description: "",
			notes: "",
			link: "",
			rating: 8,
			closed: false,
			created_at: "2026-01-15T10:00:00Z",
			roaster: { name: "Heart", location: "Portland", rkey: "roaster-1" },
		},
	],
	roasters: [
		{ rkey: "roaster-1", name: "Heart Coffee", location: "Portland, OR", website: "", created_at: "2026-01-01T00:00:00Z" },
	],
	grinders: [
		{ rkey: "grinder-1", name: "Ode 2", grinder_type: "Electric", burr_type: "Flat", notes: "", link: "", created_at: "2026-01-01T00:00:00Z" },
	],
	brewers: [
		{ rkey: "brewer-1", name: "V60", brewer_type: "pourover", description: "", link: "", created_at: "2026-01-01T00:00:00Z" },
	],
	recipes: [],
	stats: {
		bean_brew_counts: { "at://did:plc:abc/social.arabica.alpha.bean/bean-1": 3 },
		grinder_brew_counts: { "at://did:plc:abc/social.arabica.alpha.grinder/grinder-1": 5 },
		brewer_brew_counts: { "at://did:plc:abc/social.arabica.alpha.brewer/brewer-1": 5 },
		roaster_bean_counts: { "at://did:plc:abc/social.arabica.alpha.roaster/roaster-1": 1 },
		bean_avg_brew_ratings: { "at://did:plc:abc/social.arabica.alpha.bean/bean-1": 7.5 },
		roaster_avg_brew_ratings: {},
	},
};

const brewsData = {
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
			bean: { rkey: "bean-1", name: "Finca Buena Vista", origin: "Colombia", variety: "", roast_level: "Light", roast_date: "", process: "", description: "", notes: "", link: "", closed: false, created_at: "", roaster: { name: "Heart", location: "", rkey: "roaster-1" } },
			brewer_obj: { rkey: "brewer-1", name: "V60", brewer_type: "pourover", description: "", link: "", created_at: "" },
		},
	],
	has_more: false,
	next_offset: 25,
};

describe("My Coffee page", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
		localStorage.clear();
	});

	it("renders the heading and tab navigation", () => {
		render(MyCoffee, { data: { manage: manageData, brews: brewsData, error: "" } });
		expect(screen.getByText("My Coffee")).toBeTruthy();
		expect(screen.getByText("Brews")).toBeTruthy();
		expect(screen.getByText("Beans")).toBeTruthy();
		expect(screen.getByText("Roasters")).toBeTruthy();
		expect(screen.getByText("Grinders")).toBeTruthy();
	});

	it("shows the brews tab by default", () => {
		render(MyCoffee, { data: { manage: manageData, brews: brewsData, error: "" } });
		expect(screen.getByText("Bright and sweet")).toBeTruthy();
	});

	it("switches to the beans tab and shows bean stats", async () => {
		const user = userEvent.setup();
		render(MyCoffee, { data: { manage: manageData, brews: brewsData, error: "" } });
		await user.click(screen.getByText("Beans", { selector: "button" }));
		expect(screen.getByText("Finca Buena Vista")).toBeTruthy();
		expect(screen.getByText(/3 brews/)).toBeTruthy();
		expect(screen.getByText(/7.5\/10 avg/)).toBeTruthy();
	});

	it("switches to the roasters tab and shows bean count", async () => {
		const user = userEvent.setup();
		render(MyCoffee, { data: { manage: manageData, brews: brewsData, error: "" } });
		await user.click(screen.getByText("Roasters", { selector: "button" }));
		expect(screen.getByText("Heart Coffee")).toBeTruthy();
		expect(screen.getByText(/1 bean/)).toBeTruthy();
	});

	it("switches to the grinders tab and shows brew count", async () => {
		const user = userEvent.setup();
		render(MyCoffee, { data: { manage: manageData, brews: brewsData, error: "" } });
		await user.click(screen.getByText("Grinders", { selector: "button" }));
		expect(screen.getByText("Ode 2")).toBeTruthy();
		expect(screen.getByText(/5 brews/)).toBeTruthy();
	});

	it("renders an empty state when no brews", () => {
		render(MyCoffee, { data: { manage: manageData, brews: { brews: [], has_more: false, next_offset: 0 }, error: "" } });
		expect(screen.getByText("Your brew journal is empty.")).toBeTruthy();
	});

	it("renders an error state", () => {
		render(MyCoffee, { data: { manage: null, brews: null, error: "Authentication required" } });
		expect(screen.getByText("Authentication required")).toBeTruthy();
	});

	it("links brew cards to the view URL", () => {
		render(MyCoffee, { data: { manage: manageData, brews: brewsData, error: "" } });
		const link = screen.getByText("Bright and sweet").closest("a");
		expect(link).toHaveAttribute("href", "/brews/did:plc:abc/brew-1");
	});
});
