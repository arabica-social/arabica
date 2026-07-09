import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import FeedCard from "../../src/lib/components/FeedCard.svelte";
import type { FeedItem } from "../../src/lib/types/feed";

const baseAuthor = {
	did: "did:plc:abc",
	handle: "alice.test",
	display_name: "Alice",
	avatar: "",
};

function makeItem(overrides: Partial<FeedItem> & { record: Record<string, unknown> }): FeedItem {
	return {
		record_type: "brew",
		action: "added a new brew",
		record: {},
		author: baseAuthor,
		timestamp: "2026-01-20T10:00:00Z",
		time_ago: "2 hours ago",
		like_count: 3,
		comment_count: 1,
		subject_uri: "at://did:plc:abc/social.arabica.alpha.brew/brew-1",
		subject_cid: "bafy123",
		is_liked_by_viewer: false,
		is_owner: true,
		...overrides,
	};
}

describe("FeedCard", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the author name and time ago", () => {
		const item = makeItem({ record: { rkey: "brew-1" } });
		render(FeedCard, { item, isAuthenticated: true });
		expect(screen.getByText("Alice")).toBeTruthy();
		expect(screen.getByText(/2 hours ago/)).toBeTruthy();
	});

	it("renders the action header with a link to the record", () => {
		const item = makeItem({ record: { rkey: "brew-1" } });
		render(FeedCard, { item, isAuthenticated: true });
		const link = screen.getByText("new brew");
		expect(link.closest("a")).toHaveAttribute("href", "/brews/alice.test/brew-1");
	});

	it("renders brew content with bean name and rating", () => {
		const item = makeItem({
			record: {
				rkey: "brew-1",
				rating: 8,
				coffee_amount: 15,
				water_amount: 250,
				bean: { rkey: "bean-1", name: "Finca Buena Vista", origin: "Colombia", roast_level: "Light" },
			},
		});
		render(FeedCard, { item, isAuthenticated: true });
		expect(screen.getByText("Finca Buena Vista")).toBeTruthy();
		expect(screen.getByText("8/10")).toBeTruthy();
	});

	it("renders bean content with origin and roast level", () => {
		const item = makeItem({
			record_type: "bean",
			record: {
				rkey: "bean-1",
				name: "Test Bean",
				origin: "Ethiopia",
				roast_level: "Medium",
				variety: "Heirloom",
				process: "Natural",
			},
		});
		render(FeedCard, { item, isAuthenticated: false });
		expect(screen.getByText("Test Bean")).toBeTruthy();
		expect(screen.getByText("Ethiopia")).toBeTruthy();
		expect(screen.getByText("Medium")).toBeTruthy();
	});

	it("renders roaster content with location", () => {
		const item = makeItem({
			record_type: "roaster",
			record: { rkey: "roaster-1", name: "Heart Coffee", location: "Portland, OR" },
		});
		render(FeedCard, { item, isAuthenticated: false });
		expect(screen.getByText("Heart Coffee")).toBeTruthy();
		expect(screen.getByText("Portland, OR")).toBeTruthy();
	});

	it("renders grinder content with type and burr", () => {
		const item = makeItem({
			record_type: "grinder",
			record: { rkey: "grinder-1", name: "Ode 2", grinder_type: "Electric", burr_type: "Flat" },
		});
		render(FeedCard, { item, isAuthenticated: false });
		expect(screen.getByText("Ode 2")).toBeTruthy();
		expect(screen.getByText("Electric")).toBeTruthy();
		expect(screen.getByText("Flat")).toBeTruthy();
	});

	it("renders brewer content with type", () => {
		const item = makeItem({
			record_type: "brewer",
			record: { rkey: "brewer-1", name: "V60", brewer_type: "pourover" },
		});
		render(FeedCard, { item, isAuthenticated: false });
		expect(screen.getByText("V60")).toBeTruthy();
		expect(screen.getByText("pourover")).toBeTruthy();
	});

	it("renders recipe content with amounts", () => {
		const item = makeItem({
			record_type: "recipe",
			record: {
				rkey: "recipe-1",
				name: "V60 Standard",
				coffee_amount: 15,
				water_amount: 250,
				brewer_type: "pourover",
			},
		});
		render(FeedCard, { item, isAuthenticated: false });
		expect(screen.getByText("V60 Standard")).toBeTruthy();
		expect(screen.getByText("15.0g")).toBeTruthy();
		expect(screen.getByText("250.0g")).toBeTruthy();
	});

	it("renders pours as pills", () => {
		const item = makeItem({
			record: {
				rkey: "brew-1",
				pours: [
					{ pour_number: 1, water_amount: 50, time_seconds: 30 },
					{ pour_number: 2, water_amount: 100, time_seconds: 60 },
				],
			},
		});
		render(FeedCard, { item, isAuthenticated: true });
		expect(screen.getByText("Pours:")).toBeTruthy();
		expect(screen.getByText("50g")).toBeTruthy();
		expect(screen.getByText("100g")).toBeTruthy();
	});

	it("renders tasting notes", () => {
		const item = makeItem({
			record: { rkey: "brew-1", tasting_notes: "Bright and sweet" },
		});
		render(FeedCard, { item, isAuthenticated: true });
		expect(screen.getByText(/Bright and sweet/)).toBeTruthy();
	});

	it("applies the compact class for roasters", () => {
		const item = makeItem({
			record_type: "roaster",
			record: { rkey: "r1", name: "Roaster" },
		});
		const { container } = render(FeedCard, { item, isAuthenticated: false });
		expect(container.querySelector(".feed-card-roaster.feed-card-compact")).toBeTruthy();
	});
});
