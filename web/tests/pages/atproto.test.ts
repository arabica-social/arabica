import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import ATProto from "../../src/routes/atproto/+page.svelte";

describe("AT Protocol page", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the page heading", () => {
		render(ATProto);
		expect(screen.getByText("The AT Protocol")).toBeTruthy();
	});

	it("renders the key concepts section", () => {
		render(ATProto);
		expect(screen.getByText("Personal Data Server (PDS)")).toBeTruthy();
		expect(screen.getByText("Decentralized Identity (DID)")).toBeTruthy();
		expect(screen.getByText("Lexicons")).toBeTruthy();
		expect(screen.getByText("AT-URIs")).toBeTruthy();
	});

	it("links to the official atproto.com site", () => {
		render(ATProto);
		const link = screen
			.getByText("atproto.com")
			.closest("a");
		expect(link).toHaveAttribute("href", "https://atproto.com");
	});
});
