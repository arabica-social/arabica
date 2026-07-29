import { cleanup, render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import BrewForm from "../../src/lib/components/BrewForm.svelte";
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

// Mock the session store so warnIfSessionExpired is observable on mount and
// does not issue a real fetch in jsdom. vi.hoisted keeps the spy reference
// available inside the hoisted vi.mock factory.
const { warnIfSessionExpired } = vi.hoisted(() => ({
	warnIfSessionExpired: vi.fn(),
}));
vi.mock("../../src/lib/stores/session", async () => {
	const { writable } = await import("svelte/store");
	const session = writable({
		did: "did:plc:alice",
		handle: "alice.test",
		displayName: "Alice",
		avatar: "",
		isAuthenticated: true,
		isModerator: false,
		unreadNotifications: 0,
		temperatureUnit: "recorded",
	});
	return { session, warnIfSessionExpired };
});

const editBrew: Brew = {
	rkey: "brew-1",
	bean_rkey: "bean-1",
	recipe_rkey: "",
	method: "Pour Over",
	temperature: 93.5,
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
	pourover_params: { bloom_water: 50, bloom_seconds: 30, drawdown_seconds: 45, bypass_water: 0, filter: "paper" },
};

describe("BrewForm", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders 'New Brew' title for create mode", () => {
		render(BrewForm, { brew: null, isEdit: false });
		expect(screen.getByText("New Brew")).toBeTruthy();
	});

	it("renders 'Edit Brew' title for edit mode", () => {
		render(BrewForm, { brew: editBrew, isEdit: true });
		expect(screen.getByText("Edit Brew")).toBeTruthy();
	});

	it("prefills fields from the brew in edit mode", () => {
		render(BrewForm, { brew: editBrew, isEdit: true });
		expect(screen.getByDisplayValue("15")).toBeTruthy(); // coffee_amount
		expect(screen.getByDisplayValue("250")).toBeTruthy(); // water_amount
		expect(screen.getByDisplayValue("Medium")).toBeTruthy(); // grind_size
		expect(screen.getByDisplayValue("Bright and sweet")).toBeTruthy(); // tasting_notes
	});

	it("shows the pourover params section for pourover brewers", () => {
		render(BrewForm, { brew: editBrew, isEdit: true });
		expect(screen.getByText("Pour-over Details")).toBeTruthy();
		expect(screen.getByPlaceholderText("e.g. 50")).toBeTruthy(); // bloom water
		expect(screen.getByPlaceholderText("e.g. paper, metal, cloth")).toBeTruthy(); // filter
	});

	it("does not show espresso params for pourover brewers", () => {
		render(BrewForm, { brew: editBrew, isEdit: true });
		expect(screen.queryByText("Espresso")).toBeNull();
	});

	it("shows espresso params for espresso brewers", () => {
		const espressoBrew: Brew = {
			...editBrew,
			pourover_params: undefined,
			espresso_params: { yield_weight: 36, pressure: 9, pre_infusion_seconds: 5 },
			brewer_obj: { rkey: "brewer-2", name: "La Marzocco", brewer_type: "espresso", description: "", link: "", created_at: "" },
		};
		render(BrewForm, { brew: espressoBrew, isEdit: true });
		expect(screen.getByText("Espresso")).toBeTruthy();
		expect(screen.getByPlaceholderText("e.g. 36")).toBeTruthy(); // yield weight
		expect(screen.getByPlaceholderText("e.g. 9")).toBeTruthy(); // pressure
	});

	it("renders the rating slider with correct value", () => {
		render(BrewForm, { brew: editBrew, isEdit: true });
		expect(screen.getByText("8/10")).toBeTruthy();
	});

	it("renders the submit button with correct label", () => {
		render(BrewForm, { brew: null, isEdit: false });
		expect(screen.getByText("Save Brew")).toBeTruthy();
	});

	it("renders the submit button with update label in edit mode", () => {
		render(BrewForm, { brew: editBrew, isEdit: true });
		expect(screen.getByText("Update Brew")).toBeTruthy();
	});

	it("renders all required form sections", () => {
		render(BrewForm, { brew: null, isEdit: false });
		expect(screen.getByText("Coffee")).toBeTruthy();
		expect(screen.getByText("Brewing")).toBeTruthy();
		expect(screen.getByText("Results")).toBeTruthy();
		expect(screen.getByText("Recipe (Optional)")).toBeTruthy();
	});

	it("proactively checks session validity on mount", () => {
		render(BrewForm, { brew: null, isEdit: false });
		expect(warnIfSessionExpired).toHaveBeenCalled();
	});
});
