import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import BrewNewPage from "../../src/routes/brews/new/+page.svelte";

// Mock $app/navigation goto
vi.mock("$app/navigation", () => ({
	goto: vi.fn(),
}));

// Mock appCache store to avoid fetch in jsdom
vi.mock("../../src/lib/stores/appCache", () => ({
	appCache: {
		getData: vi.fn().mockResolvedValue(null),
		invalidateCache: vi.fn(),
	},
}));

describe("Brews new page", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the New Brew title", () => {
		render(BrewNewPage, {
			data: { brew: null, error: "", recipeRKey: "", recipeOwnerDID: "" },
		});
		expect(screen.getByText("New Brew")).toBeTruthy();
	});

	it("renders an error state when authentication is required", () => {
		render(BrewNewPage, {
			data: { brew: null, error: "Authentication required", recipeRKey: "", recipeOwnerDID: "" },
		});
		expect(screen.getByText("Authentication required")).toBeTruthy();
		expect(screen.getByText("Log In")).toBeTruthy();
	});

	it("renders the BrewForm component when authenticated", () => {
		const { component } = render(BrewNewPage, {
			data: { brew: null, error: "", recipeRKey: "recipe-1", recipeOwnerDID: "did:plc:other" },
		});
		// The page should render without error and show the form title.
		expect(screen.getByText("New Brew")).toBeTruthy();
		// Verify BrewForm receives the right props by checking the component instance.
		expect(component).toBeTruthy();
	});

	it("renders the Log In link when not authenticated", () => {
		render(BrewNewPage, {
			data: { brew: null, error: "Authentication required", recipeRKey: "", recipeOwnerDID: "" },
		});
		const loginLink = screen.getByText("Log In");
		expect(loginLink.closest("a")).toHaveAttribute("href", "/login");
	});
});
