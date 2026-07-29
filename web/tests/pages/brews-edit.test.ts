import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import BrewEditPage from "../../src/routes/brews/[id]/edit/+page.svelte";
import type { Brew } from "../../src/lib/types/entity_view";

vi.mock("$app/navigation", () => ({
	goto: vi.fn(),
}));

vi.mock("../../src/lib/stores/appCache", () => ({
	appCache: {
		getData: vi.fn().mockResolvedValue(null),
		invalidateCache: vi.fn(),
	},
}));

const editBrew: Brew = {
	rkey: "brew-1",
	bean_rkey: "bean-1",
	recipe_rkey: "",
	method: "Pour Over",
	temperature: 93,
	water_amount: 250,
	coffee_amount: 15,
	time_seconds: 180,
	grind_size: "Medium",
	grinder_rkey: "grinder-1",
	brewer_rkey: "brewer-1",
	tasting_notes: "Bright and sweet",
	rating: 8,
	created_at: "2026-01-20T10:00:00Z",
	bean: { rkey: "bean-1", name: "Finca Buena Vista", origin: "Colombia", roast_level: "Light", variety: "", process: "", description: "", notes: "", link: "", closed: false, created_at: "" },
	grinder_obj: { rkey: "grinder-1", name: "Ode 2", grinder_type: "Electric", burr_type: "Flat", notes: "", link: "", created_at: "" },
	brewer_obj: { rkey: "brewer-1", name: "V60", brewer_type: "pourover", description: "", link: "", created_at: "" },
};

describe("Brews edit page", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the Edit Brew title", () => {
		render(BrewEditPage, { data: { brew: editBrew, error: "" } });
		expect(screen.getByText("Edit Brew")).toBeTruthy();
	});

	it("renders an error state when authentication is required", () => {
		render(BrewEditPage, { data: { brew: null, error: "Authentication required" } });
		expect(screen.getByText("Authentication required")).toBeTruthy();
		expect(screen.getByText("Back to My Coffee")).toBeTruthy();
	});

	it("renders the Back to My Coffee link when there is an error", () => {
		render(BrewEditPage, { data: { brew: null, error: "Record not found" } });
		const link = screen.getByText("Back to My Coffee");
		expect(link.closest("a")).toHaveAttribute("href", "/my-coffee");
	});

	it("renders a loading state when brew is null and no error", () => {
		render(BrewEditPage, { data: { brew: null, error: "" } });
		expect(screen.getByText("Loading...")).toBeTruthy();
	});
});
