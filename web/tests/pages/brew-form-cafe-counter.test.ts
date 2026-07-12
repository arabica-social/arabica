import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { goto } from "$app/navigation";
import BrewForm from "../../src/lib/components/BrewForm.svelte";
import { session } from "../../src/lib/stores/session";
import type { Brew } from "../../src/lib/types/entity_view";

vi.mock("$app/navigation", () => ({ goto: vi.fn() }));

vi.mock("../../src/lib/stores/appCache", () => ({
	appCache: {
		getData: vi.fn().mockResolvedValue(null),
		invalidateCache: vi.fn(),
	},
}));

const editBrew: Brew = {
	rkey: "brew-1",
	bean_rkey: "bean-1",
	recipe_rkey: "",
	method: "Pour Over",
	temperature: 93.5,
	water_amount: 250,
	coffee_amount: 15,
	time_seconds: 180,
	grind_size: "Medium",
	grinder_rkey: "grinder-1",
	brewer_rkey: "brewer-1",
	tasting_notes: "Bright and sweet",
	rating: 8,
	created_at: "2026-01-20T10:00:00Z",
	bean: { rkey: "bean-1", name: "Finca Buena Vista", origin: "Colombia", roast_level: "Light", variety: "", process: "", description: "", notes: "", link: "", closed: false, created_at: "" },
	grinder_obj: { rkey: "grinder-1", name: "Ode 2", grinder_type: "Electric", burr_type: "Flat", notes: "", link: "", created_at: "" },
	brewer_obj: { rkey: "brewer-1", name: "V60", brewer_type: "pourover", description: "", link: "", created_at: "" },
	pourover_params: { bloom_water: 50, bloom_seconds: 30, drawdown_seconds: 45, bypass_water: 0, filter: "paper" },
};

describe("BrewForm cafe-counter migration", () => {
	beforeEach(() => {
		session.set({
			did: "did:plc:alice",
			handle: "alice.test",
			displayName: "Alice",
			avatar: "",
			isAuthenticated: true,
			isModerator: false,
			unreadNotifications: 0,
			temperatureUnit: "celsius",
		});
		vi.mocked(goto).mockReset();
	});

	afterEach(() => {
		cleanup();
		vi.unstubAllGlobals();
	});

	it("renders the live brew context rail with a ratio derived from coffee and water", async () => {
		render(BrewForm, { brew: editBrew, isEdit: true });
		// 250g water / 15g coffee = 1:16.7
		expect(screen.getByText(/Ratio 1:16\.7/)).toBeTruthy();
	});

	it("shows the brewer category label in the rail for a pourover brew", () => {
		render(BrewForm, { brew: editBrew, isEdit: true });
		expect(screen.getByText("Method: Pour-over")).toBeTruthy();
	});

	it("updates the rail ratio live as coffee and water change", async () => {
		const user = userEvent.setup();
		render(BrewForm, { brew: null, isEdit: false });

		// No ratio until both values are present.
		expect(screen.queryByText(/Ratio 1:/)).toBeNull();

		await user.type(screen.getByLabelText("Coffee Amount (grams)"), "18");
		await user.type(screen.getByLabelText("Water Amount (grams)"), "300");

		// 300/18 = 16.7
		expect(screen.getByText(/Ratio 1:16\.7/)).toBeTruthy();
	});

	it("counts recorded details in the completeness rail section", () => {
		render(BrewForm, { brew: editBrew, isEdit: true });
		// editBrew has bean, brewer, grinder, coffee, water, time, temperature,
		// tasting notes, and rating set — all 9 useful details.
		expect(screen.getByText("9 of 9 useful details recorded.")).toBeTruthy();
	});

	it("points the Cancel link at the brew view in edit mode and my-coffee in create mode", () => {
		const edit = render(BrewForm, { brew: editBrew, isEdit: true });
		const editCancel = edit.getByRole("link", { name: "Cancel" });
		expect(editCancel.getAttribute("href")).toBe(
			"/brews/did%3Aplc%3Aalice/brew-1",
		);
		edit.unmount();

		const create = render(BrewForm, { brew: null, isEdit: false });
		const createCancel = create.getByRole("link", { name: "Cancel" });
		expect(createCancel.getAttribute("href")).toBe("/my-coffee");
	});

	it("posts raw FormData with the expected brew field names and redirects to the brew view", async () => {
		const user = userEvent.setup();
		let capturedBody: FormData | null = null;
		const fetchMock = vi.fn().mockImplementation((_url: string, init: RequestInit) => {
			capturedBody = init.body as FormData;
			return Promise.resolve(
				new Response(
					JSON.stringify({
						brew: { rkey: "new-brew-1" },
						author_did: "did:plc:alice",
					}),
					{ status: 200, headers: { "Content-Type": "application/json" } },
				),
			);
		});
		vi.stubGlobal("fetch", fetchMock);
		render(BrewForm, { brew: null, isEdit: false });

		await user.type(screen.getByLabelText("Coffee Amount (grams)"), "18");
		await user.type(screen.getByLabelText("Water Amount (grams)"), "300");
		await user.type(screen.getByLabelText("Temperature (°F/°C)"), "94");
		await user.type(screen.getByLabelText("Brew Time (seconds)"), "210");
		await user.type(screen.getByLabelText("Tasting Notes"), "Bright and floral.");
		await user.click(screen.getByRole("button", { name: "Save Brew" }));

		await waitFor(() => expect(goto).toHaveBeenCalledWith("/brews/did:plc:alice/new-brew-1"));

		expect(fetchMock).toHaveBeenCalledWith(
			"/brews",
			expect.objectContaining({ method: "POST" }),
		);
		expect(capturedBody).not.toBeNull();
		const fd = capturedBody as FormData;
		expect(fd.get("coffee_amount")).toBe("18");
		expect(fd.get("water_amount")).toBe("300");
		expect(fd.get("temperature")).toBe("94");
		expect(fd.get("time_seconds")).toBe("210");
		expect(fd.get("tasting_notes")).toBe("Bright and floral.");
		expect(fd.get("rating")).toBe("5"); // default rating
	});

	it("PUTs raw FormData to the brew's own route on update", async () => {
		const user = userEvent.setup();
		let capturedMethod: string | undefined;
		const fetchMock = vi.fn().mockImplementation((url: string, init: RequestInit) => {
			capturedMethod = init.method;
			return Promise.resolve(
				new Response(
					JSON.stringify({
						brew: { rkey: "brew-1" },
						author_did: "did:plc:alice",
					}),
					{ status: 200, headers: { "Content-Type": "application/json" } },
				),
			);
		});
		vi.stubGlobal("fetch", fetchMock);
		render(BrewForm, { brew: editBrew, isEdit: true });

		await user.click(screen.getByRole("button", { name: "Update Brew" }));

		await waitFor(() => expect(goto).toHaveBeenCalledWith("/brews/did:plc:alice/brew-1"));
		expect(fetchMock).toHaveBeenCalledWith(
			"/brews/brew-1",
			expect.objectContaining({ method: "PUT" }),
		);
		expect(capturedMethod).toBe("PUT");
	});
});
