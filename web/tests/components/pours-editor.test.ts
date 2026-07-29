import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it } from "vitest";
import PoursEditor from "../../src/lib/components/PoursEditor.svelte";

// Collapse whitespace so rendered textContent (newlines/spaces from markup)
// matches plain string assertions.
function summaryText(): string {
	const el = screen.getByTestId("pour-summary");
	return (el.textContent ?? "").replace(/\s+/g, " ").trim();
}

describe("PoursEditor component", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders empty-state button with default label when pours is empty", () => {
		render(PoursEditor, { pours: [] });
		expect(screen.getByText("+ Add pours")).toBeTruthy();
	});

	it("renders empty-state button with custom emptyLabel when provided", () => {
		render(PoursEditor, { pours: [], emptyLabel: "+ Add a pour" });
		expect(screen.getByText("+ Add a pour")).toBeTruthy();
		expect(screen.queryByText("+ Add pours")).toBeNull();
	});

	it("clicking add-pour in empty state adds a pour row", async () => {
		const user = userEvent.setup();
		render(PoursEditor, { pours: [] });

		await user.click(screen.getByText("+ Add pours"));

		expect(screen.getByText("Pours")).toBeTruthy();
		expect(screen.getByPlaceholderText("Water (g)")).toBeTruthy();
		expect(screen.getByPlaceholderText("e.g. 45")).toBeTruthy();
	});

	it("renders summary with correct count and total for pre-filled pours", () => {
		render(PoursEditor, {
			pours: [
				{ water: 50, time: "" },
				{ water: 100, time: "" },
			],
		});
		expect(summaryText()).toBe("2 pours · 150g total");
	});

	it('summary shows "pour" (singular) for 1 pour and "pours" (plural) otherwise', () => {
		const { unmount } = render(PoursEditor, {
			pours: [{ water: 60, time: "" }],
		});
		expect(summaryText()).toBe("1 pour · 60g total");
		unmount();

		render(PoursEditor, {
			pours: [
				{ water: 60, time: "" },
				{ water: 40, time: "" },
			],
		});
		expect(summaryText()).toBe("2 pours · 100g total");
	});

	it("summary shows last pour time when time values present", () => {
		render(PoursEditor, {
			pours: [
				{ water: 50, time: 30 },
				{ water: 100, time: 45 },
			],
		});
		expect(summaryText()).toBe("2 pours · 150g total · last at 45s");
	});

	it("summary omits time when no time values", () => {
		render(PoursEditor, {
			pours: [
				{ water: 50, time: "" },
				{ water: 100, time: "" },
			],
		});
		const text = summaryText();
		expect(text).not.toContain("last at");
		expect(text).toBe("2 pours · 150g total");
	});

	it("shows water-mismatch warning when pourTotal differs from expectedWater", () => {
		render(PoursEditor, {
			pours: [
				{ water: 50, time: "" },
				{ water: 100, time: "" },
			],
			expectedWater: 250,
		});
		const warning = screen.getByRole("status");
		const text = (warning.textContent ?? "").replace(/\s+/g, " ").trim();
		expect(text).toBe(
			"Pour water totals 150g, which does not match total water 250g.",
		);
	});

	it("does NOT show warning when pourTotal matches expectedWater within 0.01", () => {
		render(PoursEditor, {
			pours: [{ water: 150, time: "" }],
			expectedWater: 150,
		});
		expect(screen.queryByRole("status")).toBeNull();
	});

	it("does NOT show warning when expectedWater is empty/unset", () => {
		render(PoursEditor, {
			pours: [{ water: 150, time: "" }],
		});
		expect(screen.queryByRole("status")).toBeNull();
	});

	it('clicking "+ Add Pour" appends a new empty pour row', async () => {
		const user = userEvent.setup();
		render(PoursEditor, {
			pours: [{ water: 50, time: 30 }],
		});

		expect(screen.getAllByPlaceholderText("Water (g)")).toHaveLength(1);

		await user.click(screen.getByText("+ Add Pour"));

		expect(screen.getAllByPlaceholderText("Water (g)")).toHaveLength(2);
		const waterInputs = screen.getAllByPlaceholderText("Water (g)");
		expect((waterInputs[1] as HTMLInputElement).value).toBe("");
	});

	it("clicking remove button removes the correct pour", async () => {
		const user = userEvent.setup();
		render(PoursEditor, {
			pours: [
				{ water: 50, time: 30 },
				{ water: 100, time: 45 },
			],
		});

		await user.click(screen.getByLabelText("Remove pour 1"));

		expect(screen.getAllByPlaceholderText("Water (g)")).toHaveLength(1);
		const remaining = screen.getByPlaceholderText("Water (g)") as HTMLInputElement;
		expect(remaining.value).toBe("100");
	});

	it("typing into water/time inputs updates the bound pours array", async () => {
		const user = userEvent.setup();
		render(PoursEditor, {
			pours: [{ water: 50, time: 30 }],
		});

		await user.click(screen.getByText("+ Add Pour"));

		const waterInputs = screen.getAllByPlaceholderText("Water (g)");
		const timeInputs = screen.getAllByPlaceholderText("e.g. 45");
		await user.type(waterInputs[1], "100");
		await user.type(timeInputs[1], "45");

		await waitFor(() => {
			expect(summaryText()).toBe("2 pours · 150g total · last at 45s");
		});
	});

	it("non-numeric water values are treated as 0 in the total", () => {
		render(PoursEditor, {
			pours: [
				{ water: "abc", time: "" },
				{ water: 100, time: "" },
			],
		});
		expect(summaryText()).toBe("2 pours · 100g total");
	});
});
