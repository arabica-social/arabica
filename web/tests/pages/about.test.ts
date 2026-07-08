import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import About from "../../src/routes/about/+page.svelte";

describe("About page", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the page heading and hero", () => {
		render(About);

		expect(screen.getByText("About Arabica")).toBeTruthy();
		expect(
			screen.getByText(/Alpha Software:/),
		).toBeTruthy();
	});

	it("renders the support links", () => {
		render(About);

		const kofi = screen.getByText("Ko-fi").closest("a");
		expect(kofi).toHaveAttribute(
			"href",
			"https://ko-fi.com/pdewey",
		);
		const sponsors = screen.getByText("GitHub Sponsors").closest("a");
		expect(sponsors).toHaveAttribute(
			"href",
			"https://github.com/sponsors/ptdewey",
		);
	});

	it("links to bsky.app in the getting started section", () => {
		render(About);

		const link = screen.getByText("bsky.app").closest("a");
		expect(link).toHaveAttribute("href", "https://bsky.app");
	});

	it("shows a back button", () => {
		render(About);
		expect(screen.getByRole("button", { name: "Go back" })).toBeTruthy();
	});
});
