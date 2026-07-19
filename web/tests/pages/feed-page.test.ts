import { cleanup, render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import FeedPage from "../../src/routes/+page.svelte";
import type { FeedResponse } from "../../src/lib/types/feed";

// Mock $app/navigation goto so updateURL doesn't throw in jsdom.
vi.mock("$app/navigation", () => ({
	goto: vi.fn(),
}));

const brewItem = {
	record_type: "brew" as const,
	action: "added a new brew",
	record: {
		rkey: "brew-1",
		rating: 8,
		coffee_amount: 15,
		water_amount: 250,
		bean: { rkey: "bean-1", name: "Test Bean" },
	},
	author: {
		did: "did:plc:abc",
		handle: "alice.test",
		display_name: "Alice",
		avatar: "",
	},
	timestamp: "2026-01-20T10:00:00Z",
	time_ago: "2 hours ago",
	like_count: 3,
	comment_count: 1,
	subject_uri: "at://did:plc:abc/social.arabica.alpha.brew/brew-1",
	subject_cid: "bafy123",
	is_liked_by_viewer: false,
	is_owner: true,
};

const feedData: FeedResponse = {
	items: [brewItem],
	next_cursor: "",
	is_authenticated: true,
	query: { type: "", sort: "recent" },
	tabs: [
		{ label: "All", value: "" },
		{ label: "Brews", value: "brew" },
		{ label: "Beans", value: "bean" },
		{ label: "Roasters", value: "roaster" },
	],
};

const authedPageData = {
	feed: feedData,
	error: "",
	typeFilter: "",
	sort: "recent",
	isAuthenticated: true,
	userDID: "did:plc:abc",
	appName: "arabica",
};

const unauthedPageData = {
	feed: feedData,
	error: "",
	typeFilter: "",
	sort: "recent",
	isAuthenticated: false,
	userDID: "",
	appName: "arabica",
};

describe("Feed page (home)", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
		vi.restoreAllMocks();
	});

	it("renders the welcome heading for authenticated users", () => {
		render(FeedPage, { data: authedPageData });
		expect(screen.getByText("Good coffee deserves a record.")).toBeTruthy();
	});

	it("renders quick action links for authenticated users", () => {
		render(FeedPage, { data: authedPageData });
		expect(screen.getByText("Log Brew →").closest("a")).toHaveAttribute("href", "/brews/new");
		expect(screen.getByText("Explore").closest("a")).toHaveAttribute("href", "/explore");
		expect(screen.getByText("My Coffee").closest("a")).toHaveAttribute("href", "/my-coffee");
	});

	it("renders the Cafe Counter side rails for authenticated Arabica users", () => {
		render(FeedPage, { data: authedPageData });
		expect(screen.getByRole("complementary", { name: "Your coffee journal" })).toBeTruthy();
		expect(screen.getByRole("complementary", { name: "Around the café" })).toBeTruthy();
		expect(screen.getByRole("heading", { name: "Help shape Arabica." })).toBeTruthy();
		expect(screen.getByRole("link", { name: /share feedback/i })).toHaveAttribute("href", "/feedback");
	});

	it("renders the unauth hero CTA for unauthenticated users", () => {
		render(FeedPage, { data: unauthedPageData });
		expect(screen.queryByPlaceholderText("your-handle.bsky.social")).toBeNull();
		expect(screen.getByText("Log In")).toBeTruthy();
		expect(screen.getByText("Create an account")).toBeTruthy();
		expect(screen.getByText("Learn more")).toBeTruthy();
	});

	it("renders the Community Activity heading", () => {
		render(FeedPage, { data: authedPageData });
		expect(screen.getByText("Community Activity")).toBeTruthy();
	});

	it("renders feed filter tabs", () => {
		render(FeedPage, { data: authedPageData });
		expect(screen.getByText("All")).toBeTruthy();
		expect(screen.getByText("Brews")).toBeTruthy();
		expect(screen.getByText("Beans")).toBeTruthy();
		expect(screen.getByText("Roasters")).toBeTruthy();
	});

	it("renders feed items from load data", () => {
		render(FeedPage, { data: authedPageData });
		expect(screen.getByText("Alice")).toBeTruthy();
		expect(screen.getByText(/2 hours ago/)).toBeTruthy();
	});

	it("renders an empty state when feed has no items", () => {
		const emptyData = {
			...authedPageData,
			feed: { ...feedData, items: [] },
		};
		render(FeedPage, { data: emptyData });
		expect(screen.getByText("The feed is quiet today")).toBeTruthy();
	});

	it("renders an error state when feed failed to load", () => {
		const errorData = {
			...authedPageData,
			feed: null,
			error: "Failed to load feed",
		};
		render(FeedPage, { data: errorData });
		expect(screen.getByText("The feed is quiet today")).toBeTruthy();
		expect(screen.getByText("Failed to load feed")).toBeTruthy();
	});

	it("fetches new items when a filter tab is clicked", async () => {
		const user = userEvent.setup();
		const mockResponse: FeedResponse = {
			items: [
				{
					...brewItem,
					record: { rkey: "roaster-1", name: "Filtered Roaster" },
					record_type: "roaster",
					action: "added a new roaster",
					subject_uri: "at://did:plc:abc/social.arabica.alpha.roaster/roaster-1",
				},
			],
			next_cursor: "",
			is_authenticated: true,
			query: { type: "roaster", sort: "recent" },
			tabs: [
				{ label: "All", value: "" },
				{ label: "Brews", value: "brew" },
				{ label: "Beans", value: "bean" },
				{ label: "Roasters", value: "roaster" },
			],
		};
		const fetchSpy = vi
			.spyOn(globalThis, "fetch")
			.mockResolvedValue(
				new Response(JSON.stringify(mockResponse), {
					status: 200,
					headers: { "Content-Type": "application/json" },
				}),
			);

		render(FeedPage, { data: authedPageData });

		// Click the "Roasters" filter tab.
		const roastersTab = screen.getByText("Roasters");
		await user.click(roastersTab);

		// The fetch should have been called with the filter.
		expect(fetchSpy).toHaveBeenCalled();
		const calledURL = fetchSpy.mock.calls[0][0] as string;
		expect(calledURL).toContain("type=roaster");
	});

	it("shows a Load more button when next_cursor is present", () => {
		const paginatedData = {
			...authedPageData,
			feed: { ...feedData, next_cursor: "cursor-abc" },
		};
		render(FeedPage, { data: paginatedData });
		expect(screen.getByText("Load more")).toBeTruthy();
	});
});
