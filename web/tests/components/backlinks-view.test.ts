import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import BacklinksView from "../../src/lib/components/BacklinksView.svelte";
import { app } from "../../src/lib/stores/session";
import type { BacklinksResult } from "../../src/lib/types/entity_view";

const result: BacklinksResult = {
	LibraryEntries: [
		{
			DID: "did:plc:abc",
			Handle: "alice.test",
			DisplayName: "Alice",
			AvatarURL: "",
			RecordURI: "at://did:plc:abc/social.arabica.alpha.bean/bean-1",
			Collection: "social.arabica.alpha.bean",
			RKey: "bean-1",
			CreatedAt: "2026-01-15T10:00:00Z",
			Title: "Ethiopian Yirgacheffe",
			Rating: 0,
			HasRating: false,
			ChainDepth: 2,
		},
	],
	LibraryCount: 1,
	Usage: [
		{
			Key: "brew",
			Label: "brews",
			Entries: [
				{
					DID: "did:plc:bob",
					Handle: "bob.test",
					DisplayName: "Bob",
					AvatarURL: "",
					RecordURI: "at://did:plc:bob/social.arabica.alpha.brew/brew-1",
					Collection: "social.arabica.alpha.brew",
					RKey: "brew-1",
					CreatedAt: "2026-02-01T08:00:00Z",
					Title: "Morning cup",
					Rating: 8,
					HasRating: true,
					ChainDepth: 0,
				},
			],
			Count: 1,
			RatingAverage: 8,
			RatingCount: 1,
			Page: 1,
			PerPage: 25,
			HasPrev: false,
			HasNext: false,
		},
	],
	UsageCount: 1,
	RatingAverage: 8,
	RatingCount: 1,
};

describe("BacklinksView component", () => {
	afterEach(() => {
		app.set("arabica");
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the community heading with entity name", () => {
		render(BacklinksView, {
			result,
			entityNoun: "bean",
			entityName: "Ethiopian Yirgacheffe",
			backURL: "/beans/alice.test/bean-1",
		});
		expect(
			screen.getByText("Community around Ethiopian Yirgacheffe"),
		).toBeTruthy();
	});

	it("renders the back link", () => {
		render(BacklinksView, {
			result,
			entityNoun: "bean",
			entityName: "Ethiopian Yirgacheffe",
			backURL: "/beans/alice.test/bean-1",
		});
		expect(screen.getByText("← back to bean")).toBeTruthy();
	});

	it("renders the community brew rating when present", () => {
		render(BacklinksView, {
			result,
			entityNoun: "bean",
			entityName: "Ethiopian Yirgacheffe",
			backURL: "",
		});
		expect(screen.getByText("Community brew rating")).toBeTruthy();
		expect(screen.getByText("8.0")).toBeTruthy();
		expect(screen.getByText(/avg from 1 rating/)).toBeTruthy();
	});

	it("renders library entries with record links", () => {
		render(BacklinksView, {
			result,
			entityNoun: "bean",
			entityName: "Ethiopian Yirgacheffe",
			backURL: "",
		});
		const link = screen.getByText("Ethiopian Yirgacheffe");
		expect(link.getAttribute("href")).toBe("/beans/alice.test/bean-1");
	});

	it("renders usage group entries with record links", () => {
		render(BacklinksView, {
			result,
			entityNoun: "bean",
			entityName: "Ethiopian Yirgacheffe",
			backURL: "",
		});
		expect(screen.getByText("Used in 1 brews")).toBeTruthy();
		const link = screen.getByText("Morning cup");
		expect(link.getAttribute("href")).toBe("/brews/bob.test/brew-1");
	});

	it("uses the active Oolong app route mapping", () => {
		app.set("oolong");
		const oolongResult: BacklinksResult = {
			...result,
			LibraryEntries: [
				{
					...result.LibraryEntries[0],
					Collection: "social.oolong.alpha.vendor",
					RKey: "vendor-1",
				},
			],
		};
		render(BacklinksView, {
			result: oolongResult,
			entityNoun: "vendor",
			entityName: "Tea House",
			backURL: "",
		});
		expect(screen.getByText("Ethiopian Yirgacheffe")).toHaveAttribute(
			"href",
			"/vendors/alice.test/vendor-1",
		);
	});

	it("renders empty state when result is null", () => {
		render(BacklinksView, {
			result: null,
			entityNoun: "bean",
			entityName: "Some Bean",
			backURL: "",
		});
		expect(screen.getByText("No community backlinks yet.")).toBeTruthy();
	});

	it("renders empty state when result has no library or usage", () => {
		render(BacklinksView, {
			result: {
				LibraryEntries: [],
				LibraryCount: 0,
				Usage: [],
				UsageCount: 0,
				RatingAverage: 0,
				RatingCount: 0,
			},
			entityNoun: "bean",
			entityName: "Some Bean",
			backURL: "",
		});
		expect(screen.getByText("No community backlinks yet.")).toBeTruthy();
	});

	it("renders load more link when a usage group has more pages", () => {
		const withMore: BacklinksResult = {
			...result,
			Usage: [
				{
					...result.Usage[0],
					HasNext: true,
					Page: 1,
					PerPage: 25,
				},
			],
		};
		render(BacklinksView, {
			result: withMore,
			entityNoun: "bean",
			entityName: "Ethiopian Yirgacheffe",
			backURL: "",
		});
		const loadMore = screen.getByText("Load more");
		expect(loadMore.getAttribute("href")).toContain("usage=brew");
		expect(loadMore.getAttribute("href")).toContain("page=2");
	});
});
