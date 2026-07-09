import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { get } from "svelte/store";
import { afterEach, describe, expect, it, vi } from "vitest";
import Notifications from "../../src/routes/notifications/+page.svelte";
import { clearToasts, toasts } from "../../src/lib/stores/toasts";

const notificationsData = {
	notifications: [
		{
			id: "notif-1",
			type: "like",
			actor_did: "did:plc:bob",
			subject_uri: "at://did:plc:abc/social.arabica.alpha.brew/brew-1",
			created_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
			read: false,
			actor_handle: "bob.test",
			actor_display_name: "Bob",
			actor_avatar: "",
			link: "/brews/did:plc:abc/brew-1",
			action_text: "liked your brew",
		},
		{
			id: "notif-2",
			type: "comment",
			actor_did: "did:plc:carol",
			subject_uri: "at://did:plc:abc/social.arabica.alpha.brew/brew-2",
			created_at: new Date(Date.now() - 3 * 60 * 60 * 1000).toISOString(),
			read: true,
			actor_handle: "carol.test",
			actor_display_name: "",
			actor_avatar: "",
			link: "/brews/did:plc:abc/brew-2",
			action_text: "commented on your brew",
		},
	],
	nextCursor: "",
	error: "",
};

describe("Notifications page", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
		clearToasts();
		vi.unstubAllGlobals();
	});

	it("renders the heading", () => {
		render(Notifications, { data: notificationsData });
		expect(screen.getByText("Notifications")).toBeTruthy();
	});

	it("renders notification rows with actor names", () => {
		render(Notifications, { data: notificationsData });
		expect(screen.getByText("Bob")).toBeTruthy();
		expect(screen.getByText("carol.test")).toBeTruthy();
	});

	it("renders action text", () => {
		render(Notifications, { data: notificationsData });
		expect(screen.getByText(/liked your brew/)).toBeTruthy();
		expect(screen.getByText(/commented on your brew/)).toBeTruthy();
	});

	it("shows mark all as read button", () => {
		render(Notifications, { data: notificationsData });
		expect(screen.getByText("Mark all as read")).toBeTruthy();
	});

	it("renders unread indicator for unread notifications", () => {
		const { container } = render(Notifications, { data: notificationsData });
		expect(container.querySelector(".bg-amber-400")).toBeTruthy();
	});

	it("links notification rows to the subject", () => {
		render(Notifications, { data: notificationsData });
		const link = screen.getByText("Bob").closest("a");
		expect(link).toHaveAttribute("href", "/brews/did:plc:abc/brew-1");
	});

	it("renders empty state", () => {
		render(Notifications, { data: { notifications: [], nextCursor: "", error: "" } });
		expect(screen.getByText("No notifications yet")).toBeTruthy();
	});

	it("renders error state", () => {
		render(Notifications, { data: { notifications: [], nextCursor: "", error: "Authentication required" } });
		expect(screen.getByText("Authentication required")).toBeTruthy();
	});

	it("renders load more link when cursor present", () => {
		render(Notifications, { data: { ...notificationsData, nextCursor: "cursor123" } });
		expect(screen.getByText("Load more")).toBeTruthy();
	});

	it("does not mark notifications read when the mutation fails", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(
				new Response(JSON.stringify({ error: "Failed to mark notifications as read", code: "internal_error" }), {
					status: 500,
					headers: { "Content-Type": "application/json" },
				}),
			),
		);
		const { container } = render(Notifications, { data: notificationsData });

		await userEvent.click(screen.getByRole("button", { name: "Mark all as read" }));

		await waitFor(() => expect(get(toasts).at(-1)?.message).toBe("Failed to mark as read"));
		expect(container.querySelector(".bg-amber-400")).toBeTruthy();
	});
});
