import { cleanup, render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import FeedFilters from "../../src/lib/components/FeedFilters.svelte";
import type { FeedFilterTab } from "../../src/lib/types/feed";

const baseProps = {
	typeFilter: "",
	sort: "recent",
	loading: false,
	onType: vi.fn(),
	onSort: vi.fn(),
};

function renderFeedFilters(overrides: Partial<typeof baseProps> = {}) {
	const props = { ...baseProps, ...overrides };
	props.onType = vi.fn();
	props.onSort = vi.fn();
	const result = render(FeedFilters, props);
	return { ...result, onType: props.onType, onSort: props.onSort };
}

describe("FeedFilters component", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders fallback 'All' tab when no tabs prop provided", () => {
		renderFeedFilters();
		const group = screen.getByRole("group", { name: "Feed filters" });
		expect(group).toBeTruthy();
		expect(screen.getByText("All")).toBeTruthy();
	});

	it("renders provided tabs with correct labels", () => {
		const tabs: FeedFilterTab[] = [
			{ label: "Brews", value: "brew" },
			{ label: "Beans", value: "bean" },
		];
		render(FeedFilters, { ...baseProps, onType: vi.fn(), onSort: vi.fn(), tabs });
		expect(screen.getByText("Brews")).toBeTruthy();
		expect(screen.getByText("Beans")).toBeTruthy();
	});

	it("marks the active type tab with aria-pressed='true' and filter-pill-active class", () => {
		const tabs: FeedFilterTab[] = [
			{ label: "Brews", value: "brew" },
			{ label: "Beans", value: "bean" },
		];
		render(FeedFilters, {
			...baseProps,
			onType: vi.fn(),
			onSort: vi.fn(),
			typeFilter: "brew",
			tabs,
		});
		const group = screen.getByRole("group", { name: "Feed filters" });
		const brewsButton = group.querySelector('[data-tab="brew"]') as HTMLButtonElement;
		expect(brewsButton.getAttribute("aria-pressed")).toBe("true");
		expect(brewsButton.className).toContain("filter-pill-active");
	});

	it("marks inactive tabs with aria-pressed='false'", () => {
		const tabs: FeedFilterTab[] = [
			{ label: "Brews", value: "brew" },
			{ label: "Beans", value: "bean" },
		];
		render(FeedFilters, {
			...baseProps,
			onType: vi.fn(),
			onSort: vi.fn(),
			typeFilter: "brew",
			tabs,
		});
		const group = screen.getByRole("group", { name: "Feed filters" });
		const beansButton = group.querySelector('[data-tab="bean"]') as HTMLButtonElement;
		expect(beansButton.getAttribute("aria-pressed")).toBe("false");
		expect(beansButton.className).toContain("filter-pill");
		expect(beansButton.className).not.toContain("filter-pill-active");
	});

	it("calls onType with the tab value when a type tab is clicked", async () => {
		const onType = vi.fn();
		const tabs: FeedFilterTab[] = [
			{ label: "Brews", value: "brew" },
			{ label: "Beans", value: "bean" },
		];
		render(FeedFilters, { ...baseProps, onType, onSort: vi.fn(), tabs });
		await userEvent.click(screen.getByText("Brews"));
		expect(onType).toHaveBeenCalledWith("brew");
	});

	it("marks 'New' sort button active when sort is 'recent'", () => {
		render(FeedFilters, {
			...baseProps,
			onType: vi.fn(),
			onSort: vi.fn(),
			sort: "recent",
		});
		const newButton = screen.getByRole("button", { name: "New" });
		expect(newButton.getAttribute("aria-pressed")).toBe("true");
		expect(newButton.className).toContain("filter-pill-active");
	});

	it("marks 'New' sort button active when sort is '' (empty)", () => {
		render(FeedFilters, {
			...baseProps,
			onType: vi.fn(),
			onSort: vi.fn(),
			sort: "",
		});
		const newButton = screen.getByRole("button", { name: "New" });
		expect(newButton.getAttribute("aria-pressed")).toBe("true");
		expect(newButton.className).toContain("filter-pill-active");
	});

	it("marks 'Popular' sort button active when sort is 'popular'", () => {
		render(FeedFilters, {
			...baseProps,
			onType: vi.fn(),
			onSort: vi.fn(),
			sort: "popular",
		});
		const popularButton = screen.getByRole("button", { name: "Popular" });
		expect(popularButton.getAttribute("aria-pressed")).toBe("true");
		expect(popularButton.className).toContain("filter-pill-active");
		const newButton = screen.getByRole("button", { name: "New" });
		expect(newButton.getAttribute("aria-pressed")).toBe("false");
	});

	it("calls onSort('recent') when New is clicked and onSort('popular') when Popular is clicked", async () => {
		const onSort = vi.fn();
		render(FeedFilters, { ...baseProps, onType: vi.fn(), onSort });
		await userEvent.click(screen.getByRole("button", { name: "New" }));
		expect(onSort).toHaveBeenCalledWith("recent");
		await userEvent.click(screen.getByRole("button", { name: "Popular" }));
		expect(onSort).toHaveBeenCalledWith("popular");
	});

	it("disables all buttons when loading is true", () => {
		const tabs: FeedFilterTab[] = [
			{ label: "Brews", value: "brew" },
			{ label: "Beans", value: "bean" },
		];
		render(FeedFilters, {
			...baseProps,
			onType: vi.fn(),
			onSort: vi.fn(),
			loading: true,
			tabs,
		});
		const group = screen.getByRole("group", { name: "Feed filters" });
		const typeButtons = group.querySelectorAll("button");
		typeButtons.forEach((btn) => {
			expect((btn as HTMLButtonElement).disabled).toBe(true);
		});
		expect((screen.getByRole("button", { name: "New" }) as HTMLButtonElement).disabled).toBe(true);
		expect(
			(screen.getByRole("button", { name: "Popular" }) as HTMLButtonElement).disabled,
		).toBe(true);
	});

	it("sets aria-busy='true' on container when loading", () => {
		const { container } = render(FeedFilters, {
			...baseProps,
			onType: vi.fn(),
			onSort: vi.fn(),
			loading: true,
		});
		const busyEl = container.querySelector('[aria-busy="true"]');
		expect(busyEl).toBeTruthy();
	});

	it("each type tab has data-tab attribute matching its value", () => {
		const tabs: FeedFilterTab[] = [
			{ label: "Brews", value: "brew" },
			{ label: "Beans", value: "bean" },
			{ label: "Roasters", value: "roaster" },
		];
		render(FeedFilters, { ...baseProps, onType: vi.fn(), onSort: vi.fn(), tabs });
		const group = screen.getByRole("group", { name: "Feed filters" });
		const values = Array.from(group.querySelectorAll("[data-tab]")).map((el) =>
			el.getAttribute("data-tab"),
		);
		expect(values).toEqual(["brew", "bean", "roaster"]);
	});
});
