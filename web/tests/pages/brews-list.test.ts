import { cleanup, render, screen, fireEvent, waitFor } from "@testing-library/svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import Brews from "../../src/routes/brews/+page.svelte";

// Mock $app/navigation and session store so the page can resolve the owner DID.
vi.mock("$app/navigation", () => ({
	goto: vi.fn(),
}));
vi.mock("$lib/stores/session", () => ({
	session: {
		subscribe: (fn: (v: { did: string }) => void) => {
			fn({ did: "did:plc:abc" });
			return () => {};
		},
	},
}));

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
			bean: {
				rkey: "bean-1",
				name: "Finca Buena Vista",
				origin: "Colombia",
				variety: "",
				roast_level: "Light",
				roast_date: "",
				process: "",
				description: "",
				notes: "",
				link: "",
				closed: false,
				created_at: "",
				roaster: { name: "Heart", location: "", rkey: "roaster-1" },
			},
			brewer_obj: {
				rkey: "brewer-1",
				name: "V60",
				brewer_type: "pourover",
				description: "",
				link: "",
				created_at: "",
			},
		},
	],
	has_more: false,
	next_offset: 25,
};

describe("Brews list page", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the heading and new brew link", () => {
		render(Brews, { data: { brews: brewsData, error: "" } });
		expect(screen.getByText("Your Brews")).toBeTruthy();
		expect(screen.getByText("+ New Brew")).toBeTruthy();
	});

	it("renders brew cards", () => {
		render(Brews, { data: { brews: brewsData, error: "" } });
		expect(screen.getByText("Finca Buena Vista")).toBeTruthy();
		expect(screen.getByText("Bright and sweet")).toBeTruthy();
		expect(screen.getByText("8/10")).toBeTruthy();
	});

	it("renders view/edit/delete actions for own brews", () => {
		render(Brews, { data: { brews: brewsData, error: "" } });
		expect(screen.getByText("View")).toBeTruthy();
		expect(screen.getByText("Edit")).toBeTruthy();
		expect(screen.getByText("Delete")).toBeTruthy();
	});

	it("renders empty state when no brews", () => {
		render(Brews, {
			data: { brews: { brews: [], has_more: false, next_offset: 0 }, error: "" },
		});
		expect(screen.getByText("Your brew journal is empty.")).toBeTruthy();
		expect(screen.getByText("Log Your First Brew")).toBeTruthy();
	});

	it("renders auth error with login link", () => {
		render(Brews, { data: { brews: null, error: "Authentication required" } });
		expect(screen.getByText("Authentication required")).toBeTruthy();
		expect(screen.getByText("Log In")).toBeTruthy();
	});

	it("renders load more button when has_more is true", () => {
		render(Brews, {
			data: { brews: { ...brewsData, has_more: true }, error: "" },
		});
		expect(screen.getByText("Load More")).toBeTruthy();
	});

	it("deletes a brew and removes it from the list", async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue({ ok: true } as Response);
		globalThis.fetch = fetchMock as unknown as typeof fetch;

		render(Brews, { data: { brews: brewsData, error: "" } });
		const deleteBtn = screen.getByText("Delete");
		await fireEvent.click(deleteBtn);

		await waitFor(() => {
			expect(fetchMock).toHaveBeenCalledWith(
				"/api/brews/brew-1",
				expect.objectContaining({ method: "DELETE" }),
			);
		});
		await waitFor(() => {
			expect(screen.queryByText("Finca Buena Vista")).toBeNull();
		});
	});
});
