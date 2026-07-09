import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import AddPage from "../../src/routes/add/+page.svelte";

const addData = {
	onboarding: {
		readiness: { HasBean: true, HasBrewer: true, HasRoaster: true },
		beans: [{ rkey: "bean-1", name: "Finca", origin: "Colombia", variety: "", roast_level: "Light", roast_date: "", process: "", description: "", notes: "", link: "", closed: false, created_at: "" }],
		brewers: [{ rkey: "brewer-1", name: "V60", brewer_type: "pourover", description: "", link: "", created_at: "" }],
		grinders: [],
		roasters: [{ rkey: "roaster-1", name: "Heart", location: "", website: "", created_at: "" }],
	},
	error: "",
	mode: "library" as const,
	initialEntity: "",
};

describe("Add records page", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the library hero title", () => {
		render(AddPage, { data: addData });
		expect(screen.getByText("Add records.")).toBeTruthy();
	});

	it("renders the library lede", () => {
		render(AddPage, { data: addData });
		expect(screen.getByText(/anything new on the shelf/)).toBeTruthy();
		expect(screen.getAllByText("brewer").length).toBeGreaterThan(0);
		expect(screen.getAllByText("roaster").length).toBeGreaterThan(0);
	});

	it("renders the stations without progress strip", () => {
		render(AddPage, { data: addData });
		// Station titles
		expect(screen.getAllByText("Brewer").length).toBeGreaterThan(0);
		expect(screen.getAllByText("Roaster").length).toBeGreaterThan(0);
		expect(screen.getAllByText("Bean").length).toBeGreaterThan(0);
		expect(screen.getAllByText("Grinder").length).toBeGreaterThan(0);
	});

	it("renders entity items", () => {
		render(AddPage, { data: addData });
		expect(screen.getByText("Finca")).toBeTruthy();
		expect(screen.getByText("V60")).toBeTruthy();
		expect(screen.getByText("Heart")).toBeTruthy();
	});

	it("renders error state", () => {
		render(AddPage, { data: { onboarding: null, error: "Authentication required", mode: "library", initialEntity: "" } });
		expect(screen.getByText("Authentication required")).toBeTruthy();
	});

	it("does not show the ready panel in library mode", () => {
		render(AddPage, { data: addData });
		expect(screen.queryByText("You're ready to brew!")).toBeNull();
	});
});
