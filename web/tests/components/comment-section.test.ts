import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { get } from "svelte/store";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import CommentSection from "../../src/lib/components/CommentSection.svelte";
import { session } from "../../src/lib/stores/session";
import { clearToasts, toasts } from "../../src/lib/stores/toasts";
import type { IndexedComment } from "../../src/lib/types/entity_view";

const subjectURI = "at://did:plc:alice/social.arabica.alpha.brew/brew-1";
const subjectCID = "cid-brew-1";
const currentUserDID = "did:plc:alice";

function makeComment(overrides: Partial<IndexedComment> = {}): IndexedComment {
	return {
		rkey: "comment-1",
		subject_uri: subjectURI,
		text: "Nice brew!",
		actor_did: "did:plc:bob",
		created_at: new Date(Date.now() - 30 * 1000).toISOString(),
		depth: 0,
		like_count: 0,
		is_liked: false,
		handle: "bob.test",
		display_name: "Bob",
		avatar: "",
		cid: "cid-comment-1",
		...overrides,
	};
}

function jsonOk(data: unknown): Response {
	return new Response(JSON.stringify(data), {
		status: 200,
		headers: { "Content-Type": "application/json" },
	});
}

describe("CommentSection component", () => {
	beforeEach(() => {
		session.set({
			did: currentUserDID,
			handle: "alice.test",
			displayName: "Alice",
			avatar: "",
			isAuthenticated: true,
			isModerator: false,
			unreadNotifications: 0,
			temperatureUnit: "recorded",
		});
		clearToasts();
	});

	afterEach(() => {
		cleanup();
		vi.unstubAllGlobals();
		vi.restoreAllMocks();
	});

	it("shows the indexed placeholder when subjectURI/subjectCID are empty", () => {
		render(CommentSection, {
			subjectURI: "",
			subjectCID: "",
			comments: null,
			isAuthenticated: true,
			currentUserDID,
		});
		expect(
			screen.getByText("Comments will be available once this record is indexed."),
		).toBeTruthy();
	});

	it("shows a login prompt with a button that opens the modal when canComment but not authenticated", async () => {
		const user = userEvent.setup();
		const showModal = vi.fn();
		window.__showLoginModal = showModal;
		render(CommentSection, {
			subjectURI,
			subjectCID,
			comments: null,
			isAuthenticated: false,
			currentUserDID: "",
		});
		const loginButton = screen.getByText("Log in");
		expect(loginButton.tagName).toBe("BUTTON");
		expect(screen.getByText(/to join the conversation/)).toBeTruthy();
		await user.click(loginButton);
		expect(showModal).toHaveBeenCalled();
	});

	it("shows the compose form when canComment and authenticated", () => {
		render(CommentSection, {
			subjectURI,
			subjectCID,
			comments: null,
			isAuthenticated: true,
			currentUserDID,
		});
		expect(screen.getByLabelText("Write a comment")).toBeTruthy();
		expect(screen.getByRole("button", { name: "Post" })).toBeTruthy();
	});

	it("shows the 'No comments yet' empty state when comments is null and canComment", () => {
		render(CommentSection, {
			subjectURI,
			subjectCID,
			comments: null,
			isAuthenticated: true,
			currentUserDID,
		});
		expect(screen.getByText("No comments yet")).toBeTruthy();
	});

	it("renders a comment with display name, handle, text, and time-ago", () => {
		render(CommentSection, {
			subjectURI,
			subjectCID,
			comments: [makeComment()],
			isAuthenticated: true,
			currentUserDID,
		});
		expect(screen.getByText("Bob")).toBeTruthy();
		expect(screen.getByText("@bob.test")).toBeTruthy();
		expect(screen.getByText("Nice brew!")).toBeTruthy();
		expect(screen.getByText("just now")).toBeTruthy();
	});

	it("renders the like button with a count when like_count > 0", () => {
		render(CommentSection, {
			subjectURI,
			subjectCID,
			comments: [makeComment({ like_count: 3, is_liked: false })],
			isAuthenticated: true,
			currentUserDID,
		});
		expect(screen.getByRole("button", { name: "Like comment" })).toHaveTextContent(
			"♥ Like (3)",
		);
	});

	it("shows the delete button for the current user's own comment", () => {
		render(CommentSection, {
			subjectURI,
			subjectCID,
			comments: [makeComment({ actor_did: currentUserDID })],
			isAuthenticated: true,
			currentUserDID,
		});
		expect(screen.getByRole("button", { name: "Delete" })).toBeTruthy();
	});

	it("does NOT show the delete button for another user's comment", () => {
		render(CommentSection, {
			subjectURI,
			subjectCID,
			comments: [makeComment({ actor_did: "did:plc:somebody-else" })],
			isAuthenticated: true,
			currentUserDID,
		});
		expect(screen.queryByRole("button", { name: "Delete" })).toBeNull();
	});

	it("shows the reply button when authenticated, depth < 2, and cid is present", () => {
		render(CommentSection, {
			subjectURI,
			subjectCID,
			comments: [makeComment({ depth: 0, cid: "cid-comment-1" })],
			isAuthenticated: true,
			currentUserDID,
		});
		expect(screen.getByRole("button", { name: "Reply to comment" })).toBeTruthy();
	});

	it("hides the reply button when comment depth >= 2", () => {
		render(CommentSection, {
			subjectURI,
			subjectCID,
			comments: [makeComment({ depth: 2, cid: "cid-comment-1" })],
			isAuthenticated: true,
			currentUserDID,
		});
		expect(screen.queryByRole("button", { name: "Reply to comment" })).toBeNull();
	});

	it("posts a comment via POST /api/comments with a URLSearchParams body", async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonOk({ comments: [] }));
		vi.stubGlobal("fetch", fetchMock);
		const user = userEvent.setup();
		render(CommentSection, {
			subjectURI,
			subjectCID,
			comments: [],
			isAuthenticated: true,
			currentUserDID,
		});

		await user.type(screen.getByLabelText("Write a comment"), "Great shot!");
		await user.click(screen.getByRole("button", { name: "Post" }));

		await waitFor(() => expect(fetchMock).toHaveBeenCalled());
		const [url, opts] = fetchMock.mock.calls[0];
		expect(url).toBe("/api/comments");
		expect(opts.method).toBe("POST");
		expect(opts.headers["Content-Type"]).toBe("application/x-www-form-urlencoded");
		const body = new URLSearchParams(opts.body);
		expect(body.get("subject_uri")).toBe(subjectURI);
		expect(body.get("subject_cid")).toBe(subjectCID);
		expect(body.get("text")).toBe("Great shot!");
		expect(body.has("parent_rkey")).toBe(false);
	});

	it("replaces localComments with the JSON {comments: [...]} response after a post", async () => {
		const returned = [makeComment({ text: "From server", display_name: "Server", handle: "server.test" })];
		vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonOk({ comments: returned })));
		const user = userEvent.setup();
		render(CommentSection, {
			subjectURI,
			subjectCID,
			comments: [],
			isAuthenticated: true,
			currentUserDID,
		});

		await user.type(screen.getByLabelText("Write a comment"), "hi");
		await user.click(screen.getByRole("button", { name: "Post" }));

		await waitFor(() => expect(screen.getByText("From server")).toBeTruthy());
		expect(screen.queryByText("No comments yet")).toBeNull();
	});

	it("shows 'Failed to post comment' toast when the post request fails", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(
				new Response(JSON.stringify({ error: "boom" }), {
					status: 500,
					headers: { "Content-Type": "application/json" },
				}),
			),
		);
		const user = userEvent.setup();
		render(CommentSection, {
			subjectURI,
			subjectCID,
			comments: [],
			isAuthenticated: true,
			currentUserDID,
		});

		await user.type(screen.getByLabelText("Write a comment"), "hi");
		await user.click(screen.getByRole("button", { name: "Post" }));

		await waitFor(() =>
			expect(get(toasts).at(-1)?.message).toBe("Failed to post comment"),
		);
	});

	it("toggles a like via POST /api/likes/toggle with the comment AT-URI", async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonOk({ is_liked: true, like_count: 1 }),
		);
		vi.stubGlobal("fetch", fetchMock);
		const user = userEvent.setup();
		const comment = makeComment({ actor_did: "did:plc:bob", like_count: 0 });
		render(CommentSection, {
			subjectURI,
			subjectCID,
			comments: [comment],
			isAuthenticated: true,
			currentUserDID,
		});

		await user.click(screen.getByRole("button", { name: "Like comment" }));

		await waitFor(() => expect(fetchMock).toHaveBeenCalled());
		const [url, opts] = fetchMock.mock.calls[0];
		expect(url).toBe("/api/likes/toggle");
		expect(opts.method).toBe("POST");
		const body = new URLSearchParams(opts.body);
		expect(body.get("subject_uri")).toBe(
			`at://${comment.actor_did}/social.arabica.alpha.comment/${comment.rkey}`,
		);
		expect(body.get("subject_cid")).toBe(comment.cid);
	});

	it("deletes a comment after confirm and shows a 'Comment deleted' toast", async () => {
		vi.spyOn(window, "confirm").mockReturnValue(true);
		const fetchMock = vi.fn().mockResolvedValue(jsonOk({ comments: [] }));
		vi.stubGlobal("fetch", fetchMock);
		const user = userEvent.setup();
		const comment = makeComment({ actor_did: currentUserDID });
		render(CommentSection, {
			subjectURI,
			subjectCID,
			comments: [comment],
			isAuthenticated: true,
			currentUserDID,
		});

		await user.click(screen.getByRole("button", { name: "Delete" }));

		await waitFor(() => expect(fetchMock).toHaveBeenCalled());
		const [url, opts] = fetchMock.mock.calls[0];
		expect(url).toBe(`/api/comments/${comment.rkey}`);
		expect(opts.method).toBe("DELETE");
		await waitFor(() =>
			expect(get(toasts).at(-1)?.message).toBe("Comment deleted"),
		);
	});

	it("does NOT call fetch when delete is cancelled (confirm=false)", async () => {
		vi.spyOn(window, "confirm").mockReturnValue(false);
		const fetchMock = vi.fn();
		vi.stubGlobal("fetch", fetchMock);
		const user = userEvent.setup();
		render(CommentSection, {
			subjectURI,
			subjectCID,
			comments: [makeComment({ actor_did: currentUserDID })],
			isAuthenticated: true,
			currentUserDID,
		});

		await user.click(screen.getByRole("button", { name: "Delete" }));

		expect(window.confirm).toHaveBeenCalledWith("Delete this comment?");
		expect(fetchMock).not.toHaveBeenCalled();
	});

	describe("formatTimeAgo", () => {
		it.each([
			[10, "just now"],
			[120, "2m ago"],
			[3 * 3600, "3h ago"],
			[2 * 86400, "2d ago"],
		])("renders %ss-ago timestamps as %s", (secondsAgo, expected) => {
			render(CommentSection, {
				subjectURI,
				subjectCID,
				comments: [
					makeComment({
						created_at: new Date(Date.now() - secondsAgo * 1000).toISOString(),
					}),
				],
				isAuthenticated: true,
				currentUserDID,
			});
			expect(screen.getByText(expected)).toBeTruthy();
		});
	});
});
