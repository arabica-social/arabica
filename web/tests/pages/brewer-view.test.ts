import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import BrewerView from "../../src/routes/brewers/[actor]/[id]/+page.svelte";

const brewerData = {
	data: {
		record: {
			rkey: "brewer-1",
			name: "Hario V60",
			brewer_type: "Pour Over",
			description: "The classic pour-over dripper.",
			link: "https://hario.com",
			created_at: "2026-01-15T10:00:00Z",
		},
		subject_uri: "at://did:plc:abc/social.arabica.alpha.brewer/brewer-1",
		subject_cid: "bafy123",
		author: {
			did: "did:plc:abc",
			handle: "alice.test",
			display_name: "Alice",
			avatar: "",
		},
		social: {
			is_liked: false,
			like_count: 0,
			comment_count: 0,
			comments: [],
			is_moderator: false,
			can_hide_record: false,
			can_block_user: false,
			is_record_hidden: false,
		},
		backlinks: null,
		is_own_profile: true,
		is_authenticated: true,
		share_url: "/brewers/alice.test/brewer-1",
		entity_type: "brewer",
		entity_count: 4,
	},
	error: undefined,
	status: 200,
};

const noOwnerData = {
	...brewerData,
	data: {
		...brewerData.data,
		is_own_profile: false,
		is_authenticated: false,
	},
};

describe("Brewer view", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the brewer name and type", () => {
		render(BrewerView, { data: brewerData });
		expect(screen.getByText("Hario V60")).toBeTruthy();
		expect(screen.getByText("Pour Over")).toBeTruthy();
	});

	it("renders the brew count stat line", () => {
		render(BrewerView, { data: brewerData });
		expect(screen.getByText(/4 brews/)).toBeTruthy();
	});

	it("renders the description", () => {
		render(BrewerView, { data: brewerData });
		expect(screen.getByText("The classic pour-over dripper.")).toBeTruthy();
	});

	it("renders the website link", () => {
		render(BrewerView, { data: brewerData });
		const link = screen.getByText("https://hario.com");
		expect(link).toHaveAttribute("href", "https://hario.com");
	});

	it("renders an error state", () => {
		render(BrewerView, { data: { error: "Record not found", status: 404 } });
		expect(screen.getByText("Record not found")).toBeTruthy();
	});

	it("does not show edit menu for non-owners", () => {
		render(BrewerView, { data: noOwnerData });
		expect(screen.queryByText("Edit")).toBeNull();
	});
});
