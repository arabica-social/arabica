import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { get } from "svelte/store";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { goto } from "$app/navigation";
import ActionBar from "../../src/lib/components/ActionBar.svelte";
import { clearToasts, toasts } from "../../src/lib/stores/toasts";

vi.mock("$app/navigation", () => ({ goto: vi.fn() }));

const props = {
	subjectURI: "at://did:plc:alice/social.arabica.alpha.roaster/r1",
	subjectCID: "cid",
	isLiked: false,
	likeCount: 0,
	commentCount: 0,
	shareURL: "/roasters/alice.test/r1",
	isOwner: true,
	deleteURL: "/api/roasters/r1",
	deleteRedirect: "/my-coffee",
	isAuthenticated: true,
};

describe("ActionBar delete", () => {
	beforeEach(() => {
		vi.spyOn(window, "confirm").mockReturnValue(true);
		vi.mocked(goto).mockReset();
		clearToasts();
	});

	afterEach(() => {
		cleanup();
		vi.restoreAllMocks();
		vi.unstubAllGlobals();
	});

	it("confirms, deletes, and navigates after success", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(
				new Response(JSON.stringify({ deleted: true }), {
					status: 200,
					headers: { "Content-Type": "application/json" },
				}),
			),
		);
		render(ActionBar, props);

		await userEvent.click(screen.getByRole("button", { name: "More options" }));
		await userEvent.click(screen.getByRole("menuitem", { name: "Delete" }));

		await waitFor(() => expect(goto).toHaveBeenCalledWith("/my-coffee"));
		expect(window.confirm).toHaveBeenCalledWith("Are you sure you want to delete this?");
	});

	it("keeps the user on the page and reports a failed delete", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(
				new Response(JSON.stringify({ error: "Failed to delete roaster", code: "internal_error" }), {
					status: 500,
					headers: { "Content-Type": "application/json" },
				}),
			),
		);
		render(ActionBar, props);

		await userEvent.click(screen.getByRole("button", { name: "More options" }));
		await userEvent.click(screen.getByRole("menuitem", { name: "Delete" }));

		await waitFor(() => expect(get(toasts).at(-1)?.message).toBe("Failed to delete"));
		expect(goto).not.toHaveBeenCalled();
	});
});
