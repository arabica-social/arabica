import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import Explore from "../../src/routes/explore/+page.svelte";

// Mock $app/navigation
vi.mock("$app/navigation", () => ({
	goto: vi.fn(),
}));

const exploreData = {
	explore: {
		items: [
			{
				record_type: "bean",
				action: "added a new bean",
				record: { rkey: "bean-1", name: "Ethiopian Yirgacheffe", origin: "Ethiopia", variety: "", roast_level: "Light", roast_date: "", process: "Washed", description: "", notes: "", link: "", closed: false, created_at: "2026-01-01T00:00:00Z" },
				author: { did: "did:plc:abc", handle: "alice.test", display_name: "Alice", avatar: "" },
				timestamp: "2026-01-01T00:00:00Z",
				time_ago: "2 days ago",
				like_count: 5,
				comment_count: 2,
				subject_uri: "at://did:plc:abc/social.arabica.alpha.bean/bean-1",
				subject_cid: "bafy123",
				is_liked_by_viewer: false,
				is_owner: false,
			},
		],
		documents: {},
		facet_counts: [],
		next_cursor: "",
		health: { Ready: true, Dirty: false, TotalDocuments: 100 },
	},
	error: "",
	isAuthenticated: true,
};

describe("Explore page", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the hero title", () => {
		render(Explore, { data: exploreData });
		expect(screen.getByText("Explore records.")).toBeTruthy();
	});

	it("renders the search input", () => {
		render(Explore, { data: exploreData });
		expect(screen.getByPlaceholderText("Ethiopia, V60, washed")).toBeTruthy();
	});

	it("renders the type filter dropdown", () => {
		render(Explore, { data: exploreData });
		expect(screen.getByText("All records")).toBeTruthy();
	});

	it("renders results", () => {
		render(Explore, { data: exploreData });
		expect(screen.getByText("Ethiopian Yirgacheffe")).toBeTruthy();
	});

	it("renders the Explore button", () => {
		render(Explore, { data: exploreData });
		expect(screen.getByText("Explore")).toBeTruthy();
	});

	it("renders empty state when no results", () => {
		render(Explore, { data: { explore: { items: [], documents: {}, facet_counts: [], next_cursor: "", health: { Ready: true, Dirty: false, TotalDocuments: 0 } }, error: "", isAuthenticated: false } });
		expect(screen.getByText("No matching records yet.")).toBeTruthy();
	});

	it("shows the stale note when health is dirty", () => {
		render(Explore, { data: { ...exploreData, explore: { ...exploreData.explore, health: { Ready: true, Dirty: true, TotalDocuments: 100 } } } });
		expect(screen.getByText(/Explore is catching up/)).toBeTruthy();
	});
});
