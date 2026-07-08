import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import JoinCreate from "../../src/routes/join/create/+page.svelte";

const sampleCategories = [
	{
		title: "Recommended",
		description: "A reliable provider.",
		providers: [
			{
				url: "https://selfhosted.social",
				name: "selfhosted.social",
				domain: "selfhosted.social",
				description: "Community provider",
				location: "United States",
				badge: "Open",
				badge_color: "green",
				operator_name: "@baileytownsend.dev",
				operator_url: "https://bsky.app/profile/baileytownsend.dev",
				signup_url: "",
			},
		],
		dev_only: false,
	},
	{
		title: "App Providers",
		description: "Apps that host your account.",
		providers: [
			{
				url: "https://bsky.social",
				name: "Bluesky",
				domain: "bsky.social",
				description: "The largest PDS provider.",
				location: "United States",
				badge: "Open",
				badge_color: "green",
				operator_name: "",
				operator_url: "",
				signup_url: "https://bsky.app",
			},
		],
		dev_only: false,
	},
];

describe("Join/Create page", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the heading and description", () => {
		render(JoinCreate, { data: { error: "", categories: [], loadFailed: false } });
		expect(
			screen.getByText("Create an Atmosphere Account"),
		).toBeTruthy();
	});

	it("renders provider cards from loaded categories", () => {
		render(JoinCreate, {
			data: { error: "", categories: sampleCategories, loadFailed: false },
		});

		expect(screen.getByText("selfhosted.social")).toBeTruthy();
		expect(screen.getByText("Bluesky")).toBeTruthy();
		expect(
			screen.getAllByText((_, node) =>
				!!node?.textContent?.includes("Community provider"),
			).length,
		).toBeGreaterThan(0);
	});

	it("renders a POST form for providers without a signup URL", () => {
		render(JoinCreate, {
			data: { error: "", categories: sampleCategories, loadFailed: false },
		});

		const form = screen
			.getAllByRole("button", { name: "Create Account" })
			.find((btn) => btn.closest("form"));
		expect(form).toBeTruthy();
		const hidden = form
			?.closest("form")
			?.querySelector('input[type="hidden"][name="pds_url"]');
		expect(hidden).toHaveValue("https://selfhosted.social");
	});

	it("renders an external link for providers with a signup URL", () => {
		render(JoinCreate, {
			data: { error: "", categories: sampleCategories, loadFailed: false },
		});

		const link = screen
			.getAllByRole("link", { name: "Create Account" })
			.find((a) => a.getAttribute("target") === "_blank");
		expect(link).toBeTruthy();
		expect(link).toHaveAttribute("href", "https://bsky.app");
	});

	it("shows an error banner when error is present", () => {
		render(JoinCreate, {
			data: { error: "Invalid server selection", categories: [], loadFailed: false },
		});
		expect(screen.getByText("Invalid server selection")).toBeTruthy();
	});

	it("shows a failure banner when the load failed", () => {
		render(JoinCreate, {
			data: { error: "", categories: [], loadFailed: true },
		});
		expect(
			screen.getByText(/Could not load the provider list/),
		).toBeTruthy();
	});
});
