import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import GrinderView from "../../src/routes/grinders/[actor]/[id]/+page.svelte";

const grinderData = {
	data: {
		record: {
			rkey: "grinder-1",
			name: "Comandante C40",
			grinder_type: "Manual",
			burr_type: "Conical",
			notes: "Red clix installed",
			link: "https://comandante.com",
			created_at: "2026-01-15T10:00:00Z",
		},
		subject_uri: "at://did:plc:abc/social.arabica.alpha.grinder/grinder-1",
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
		share_url: "/grinders/alice.test/grinder-1",
		entity_type: "grinder",
		entity_count: 3,
	},
	error: undefined,
	status: 200,
};

const noOwnerData = {
	...grinderData,
	data: {
		...grinderData.data,
		is_own_profile: false,
		is_authenticated: false,
	},
};

describe("Grinder view", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the grinder name and type", () => {
		render(GrinderView, { data: grinderData });
		expect(screen.getByText("Comandante C40")).toBeTruthy();
		expect(screen.getByText("Manual")).toBeTruthy();
		expect(screen.getByText("Conical")).toBeTruthy();
	});

	it("renders the brew count stat line", () => {
		render(GrinderView, { data: grinderData });
		expect(screen.getByText(/3 brews/)).toBeTruthy();
	});

	it("renders the website link", () => {
		render(GrinderView, { data: grinderData });
		const link = screen.getByText("https://comandante.com");
		expect(link).toHaveAttribute("href", "https://comandante.com");
	});

	it("renders the notes", () => {
		render(GrinderView, { data: grinderData });
		expect(screen.getByText("Red clix installed")).toBeTruthy();
	});

	it("renders an error state", () => {
		render(GrinderView, { data: { error: "Record not found", status: 404 } });
		expect(screen.getByText("Record not found")).toBeTruthy();
	});

	it("does not show edit menu for non-owners", () => {
		render(GrinderView, { data: noOwnerData });
		expect(screen.queryByText("Edit")).toBeNull();
	});
});
