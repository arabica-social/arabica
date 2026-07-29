import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import LoginModal from "../../src/lib/components/LoginModal.svelte";

const ACTOR = {
	handle: "alice.bsky.social",
	displayName: "Alice",
	avatar: "",
};

function actorsResponse(actors: unknown[]): Response {
	return new Response(JSON.stringify({ actors }), {
		status: 200,
		headers: { "Content-Type": "application/json" },
	});
}

describe("LoginModal component", () => {
	beforeEach(() => {
		// jsdom doesn't support the Dialog or Popover APIs, so stub them out.
		if (!HTMLDialogElement.prototype.showModal) {
			HTMLDialogElement.prototype.showModal = vi.fn();
		}
		if (!HTMLDialogElement.prototype.close) {
			HTMLDialogElement.prototype.close = vi.fn();
		}
		if (!HTMLElement.prototype.showPopover) {
			HTMLElement.prototype.showPopover = vi.fn();
		}
		if (!HTMLElement.prototype.hidePopover) {
			HTMLElement.prototype.hidePopover = vi.fn();
		}
	});

	afterEach(() => {
		cleanup();
		vi.unstubAllGlobals();
		vi.restoreAllMocks();
	});

	it("fills the input but does not submit the form when a suggestion is clicked", async () => {
		const user = userEvent.setup();
		const fetchMock = vi
			.spyOn(globalThis, "fetch")
			.mockResolvedValue(actorsResponse([ACTOR]));
		const submitSpy = vi.fn();

		render(LoginModal, { props: { open: true } });

		const input = screen.getByPlaceholderText(
			"your-handle.bsky.social",
		) as HTMLInputElement;
		const form = input.closest("form") as HTMLFormElement;
		form.addEventListener("submit", (event) => {
			event.preventDefault();
			submitSpy();
		});

		await user.type(input, "alic");
		await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));

		const option = await waitFor(() =>
			screen.getByText(`@${ACTOR.handle}`),
		);
		await user.click(option);

		expect(input).toHaveValue(ACTOR.handle);
		expect(submitSpy).not.toHaveBeenCalled();
	});

	it("selects the first suggestion instead of submitting when Enter is pressed with the dropdown open", async () => {
		const user = userEvent.setup();
		const fetchMock = vi
			.spyOn(globalThis, "fetch")
			.mockResolvedValue(actorsResponse([ACTOR]));
		const submitSpy = vi.fn();

		render(LoginModal, { props: { open: true } });

		const input = screen.getByPlaceholderText(
			"your-handle.bsky.social",
		) as HTMLInputElement;
		const form = input.closest("form") as HTMLFormElement;
		form.addEventListener("submit", (event) => {
			event.preventDefault();
			submitSpy();
		});

		await user.type(input, "alic");
		await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
		await waitFor(() => screen.getByText(`@${ACTOR.handle}`));

		await user.keyboard("{Enter}");

		expect(input).toHaveValue(ACTOR.handle);
		expect(submitSpy).not.toHaveBeenCalled();
	});

	it("disables the submit button while the dropdown is open", async () => {
		const user = userEvent.setup();
		const fetchMock = vi
			.spyOn(globalThis, "fetch")
			.mockResolvedValue(actorsResponse([ACTOR]));

		render(LoginModal, { props: { open: true } });

		const input = screen.getByPlaceholderText(
			"your-handle.bsky.social",
		) as HTMLInputElement;
		const submitButton = screen.getByText("Log In") as HTMLButtonElement;

		expect(submitButton).not.toBeDisabled();

		await user.type(input, "alic");
		await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
		await waitFor(() => screen.getByText(`@${ACTOR.handle}`));

		expect(submitButton).toBeDisabled();

		const option = screen.getByText(`@${ACTOR.handle}`);
		await user.click(option);
		await waitFor(() => expect(submitButton).not.toBeDisabled());
	});

	it("submits the form normally when Enter is pressed with the dropdown closed", async () => {
		const user = userEvent.setup();
		const submitSpy = vi.fn();

		render(LoginModal, { props: { open: true } });

		const input = screen.getByPlaceholderText(
			"your-handle.bsky.social",
		) as HTMLInputElement;
		const form = input.closest("form") as HTMLFormElement;
		form.addEventListener("submit", (event) => {
			event.preventDefault();
			submitSpy();
		});

		await user.type(input, "manual.handle.bsky.social");
		await user.keyboard("{Enter}");

		expect(submitSpy).toHaveBeenCalledTimes(1);
	});

	it("prevents a spurious submit event right after selecting a suggestion", async () => {
		const user = userEvent.setup();
		const fetchMock = vi
			.spyOn(globalThis, "fetch")
			.mockResolvedValue(actorsResponse([ACTOR]));
		let lastDefaultPrevented = false;

		render(LoginModal, { props: { open: true } });

		const input = screen.getByPlaceholderText(
			"your-handle.bsky.social",
		) as HTMLInputElement;
		const form = input.closest("form") as HTMLFormElement;
		form.addEventListener("submit", (event) => {
			lastDefaultPrevented = event.defaultPrevented;
			event.preventDefault();
		});

		await user.type(input, "alic");
		await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));

		const option = await waitFor(() =>
			screen.getByText(`@${ACTOR.handle}`),
		);
		await user.click(option);
		expect(input).toHaveValue(ACTOR.handle);

		// Simulate the browser synthesizing a submit event immediately after
		// the popover button disappears during the click.
		form.dispatchEvent(new SubmitEvent("submit", { bubbles: true, cancelable: true }));

		expect(lastDefaultPrevented).toBe(true);
	});
});
