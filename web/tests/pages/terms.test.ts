import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import Terms from "../../src/routes/terms/+page.svelte";

describe("Terms page", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the page heading", () => {
		render(Terms);
		expect(screen.getByText("Terms of Service")).toBeTruthy();
	});

	it("renders all 16 numbered sections", () => {
		render(Terms);
		for (let i = 1; i <= 16; i += 1) {
			expect(screen.getByText(new RegExp(`^${i}\\.`))).toBeTruthy();
		}
	});

	it("includes the last updated date", () => {
		render(Terms);
		expect(screen.getByText(/March 29, 2026/)).toBeTruthy();
	});

	it("shows the contact email", () => {
		render(Terms);
		const emails = screen.getAllByText("mail@arabica.systems");
		expect(emails.length).toBeGreaterThan(0);
		expect(emails[0]!.closest("a")).toHaveAttribute(
			"href",
			"mailto:mail@arabica.systems",
		);
	});
});
