import { cleanup, render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import EntityCombo from "../../src/lib/components/EntityCombo.svelte";

describe("EntityCombo", () => {
	afterEach(() => {
		cleanup();
		localStorage.clear();
		vi.unstubAllGlobals();
	});

	it("loads matching user entities from an already-populated app cache", async () => {
		document.body.dataset.userDid = "did:plc:alice";
		document.body.dataset.app = "arabica";
		localStorage.setItem(
			"arabica_data_cache",
			JSON.stringify({
				version: 1,
				timestamp: Date.now(),
				did: "did:plc:alice",
				app: "arabica",
				data: {
					did: "did:plc:alice",
					beans: [{ rkey: "bean-1", name: "E2E Brew Bean", origin: "Ethiopia" }],
				},
			}),
		);
		vi.stubGlobal("fetch", vi.fn().mockResolvedValue(
			new Response("[]", { status: 200, headers: { "Content-Type": "application/json" } }),
		));
		render(EntityCombo, {
			entityType: "bean",
			apiEndpoint: "/api/beans",
			suggestEndpoint: "/api/suggestions/beans",
			inputName: "bean_rkey",
			sectionLabel: "Your beans",
			ariaLabel: "Search coffee beans",
		});

		await userEvent.type(screen.getByRole("combobox", { name: "Search coffee beans" }), "E2E Brew Bean");

		expect(await screen.findByRole("option", { name: "E2E Brew Bean (Ethiopia)" })).toBeVisible();
	});
});
