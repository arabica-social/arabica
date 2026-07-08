import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import RoasterView from "../../src/routes/roasters/[actor]/[id]/+page.svelte";

const roasterData = {
	data: {
		record: {
			rkey: "roaster-1",
			name: "Heart Coffee Roasters",
			location: "Portland, OR",
			website: "https://heartcoffee.com",
			created_at: "2026-01-15T10:00:00Z",
		},
		subject_uri: "at://did:plc:abc/social.arabica.alpha.roaster/roaster-1",
		subject_cid: "bafy123",
		author: {
			did: "did:plc:abc",
			handle: "alice.test",
			display_name: "Alice",
			avatar: "",
		},
		social: {
			is_liked: false,
			like_count: 3,
			comment_count: 1,
			comments: [],
			is_moderator: false,
			can_hide_record: false,
			can_block_user: false,
			is_record_hidden: false,
		},
		backlinks: null,
		is_own_profile: true,
		is_authenticated: true,
		share_url: "/roasters/alice.test/roaster-1",
		entity_type: "roaster",
		entity_count: 5,
	},
	error: undefined,
	status: 200,
};

const noOwnerData = {
	...roasterData,
	data: {
		...roasterData.data,
		is_own_profile: false,
		is_authenticated: false,
	},
};

describe("Roaster view", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the roaster name and location", () => {
		render(RoasterView, { data: roasterData });
		expect(screen.getByText("Heart Coffee Roasters")).toBeTruthy();
		expect(screen.getByText("Portland, OR")).toBeTruthy();
	});

	it("renders the bean count stat line", () => {
		render(RoasterView, { data: roasterData });
		expect(screen.getByText(/5 beans/)).toBeTruthy();
	});

	it("renders the website link", () => {
		render(RoasterView, { data: roasterData });
		const link = screen.getByText("https://heartcoffee.com");
		expect(link).toHaveAttribute("href", "https://heartcoffee.com");
	});

	it("renders the author handle", () => {
		render(RoasterView, { data: roasterData });
		expect(screen.getByText(/alice\.test/)).toBeTruthy();
	});

	it("renders an error state", () => {
		render(RoasterView, { data: { error: "Record not found", status: 404 } });
		expect(screen.getByText("Record not found")).toBeTruthy();
	});

	it("does not show edit menu for non-owners", () => {
		render(RoasterView, { data: noOwnerData });
		// The more menu button is still there (for report), but Edit isn't.
		expect(screen.queryByText("Edit")).toBeNull();
	});
});
