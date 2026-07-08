import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import BeanView from "../../src/routes/beans/[actor]/[id]/+page.svelte";

const beanData = {
	data: {
		record: {
			rkey: "bean-1",
			name: "Finca Buena Vista",
			origin: "Colombia",
			variety: "Bourbon",
			roast_level: "Light",
			roast_date: "2026-01-10",
			process: "Washed",
			description: "Bright and fruity",
			notes: "Love this one",
			link: "https://heartcoffee.com/beans/buena-vista",
			rating: 8,
			closed: false,
			created_at: "2026-01-15T10:00:00Z",
			roaster: {
				name: "Heart Coffee Roasters",
				location: "Portland, OR",
				rkey: "roaster-1",
			},
		},
		subject_uri: "at://did:plc:abc/social.arabica.alpha.bean/bean-1",
		subject_cid: "bafy123",
		author: {
			did: "did:plc:abc",
			handle: "alice.test",
			display_name: "Alice",
			avatar: "",
		},
		social: {
			is_liked: true,
			like_count: 5,
			comment_count: 2,
			comments: [],
			is_moderator: false,
			can_hide_record: false,
			can_block_user: false,
			is_record_hidden: false,
		},
		backlinks: null,
		is_own_profile: true,
		is_authenticated: true,
		share_url: "/beans/alice.test/bean-1",
		entity_type: "bean",
		entity_count: 3,
	},
	error: undefined,
	status: 200,
};

describe("Bean view", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the bean name as the hero title", () => {
		render(BeanView, { data: beanData });
		expect(screen.getByText("Finca Buena Vista")).toBeTruthy();
	});

	it("renders the roaster link", () => {
		render(BeanView, { data: beanData });
		const roasterLink = screen.getByText("Heart Coffee Roasters");
		expect(roasterLink).toHaveAttribute(
			"href",
			"/roasters/alice.test/roaster-1",
		);
	});

	it("renders the rating hero", () => {
		render(BeanView, { data: beanData });
		expect(screen.getByText("8")).toBeTruthy();
		expect(screen.getByText("/ 10")).toBeTruthy();
	});

	it("renders the tags (origin, variety, roast level, process)", () => {
		render(BeanView, { data: beanData });
		expect(screen.getByText("Colombia")).toBeTruthy();
		expect(screen.getByText("Bourbon")).toBeTruthy();
		expect(screen.getByText("Light")).toBeTruthy();
		expect(screen.getByText("Washed")).toBeTruthy();
	});

	it("renders the roast date", () => {
		render(BeanView, { data: beanData });
		expect(screen.getByText(/Roasted/)).toBeTruthy();
	});

	it("renders the brew count stat line", () => {
		render(BeanView, { data: beanData });
		expect(screen.getByText(/3 brews/)).toBeTruthy();
	});

	it("shows Close Bag and Edit buttons for the owner", () => {
		render(BeanView, { data: beanData });
		expect(screen.getByText("Close Bag")).toBeTruthy();
		expect(screen.getByText("Edit Bean")).toBeTruthy();
	});

	it("renders the description and personal notes", () => {
		render(BeanView, { data: beanData });
		expect(screen.getByText("Bright and fruity")).toBeTruthy();
		expect(screen.getByText("Love this one")).toBeTruthy();
	});

	it("renders an error state", () => {
		render(BeanView, { data: { error: "Record not found", status: 404 } });
		expect(screen.getByText("Record not found")).toBeTruthy();
	});
});
