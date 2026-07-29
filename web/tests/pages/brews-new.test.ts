import { cleanup, render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import BrewNewPage from "../../src/routes/brews/new/+page.svelte";

vi.mock("$app/navigation", () => ({
	goto: vi.fn(),
}));

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
		expect(screen.getByText("New Brew")).toBeTruthy();
		expect(component).toBeTruthy();
	});

	it("renders the Log In button when not authenticated", () => {
		render(BrewNewPage, {
			data: { brew: null, error: "Authentication required", recipeRKey: "", recipeOwnerDID: "" },
		});
		const loginButton = screen.getByText("Log In");
		expect(loginButton.tagName).toBe("BUTTON");
	});

	it("opens the login modal when the Log In button is clicked", async () => {
		const user = userEvent.setup();
		const showModal = vi.fn();
		window.__showLoginModal = showModal;
		render(BrewNewPage, {
			data: { brew: null, error: "Authentication required", recipeRKey: "", recipeOwnerDID: "" },
		});
		const loginButton = screen.getByText("Log In");
		await user.click(loginButton);
		expect(showModal).toHaveBeenCalled();
	});
});
