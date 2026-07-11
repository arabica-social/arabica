<script lang="ts">
	import Avatar from "./Avatar.svelte";
	import { app, displayHandle, session } from "../stores/session";
	import { definitionFor } from "$lib/app/definitions";
	import { pushToast } from "../stores/toasts";
	import type { IndexedComment } from "../types/entity_view";

	let {
		subjectURI = "",
		subjectCID = "",
		comments,
		isAuthenticated = false,
		currentUserDID = "",
		isModerator = false,
		viewURL = "",
	}: {
		subjectURI?: string;
		subjectCID?: string;
		comments?: IndexedComment[] | null;
		isAuthenticated?: boolean;
		currentUserDID?: string;
		isModerator?: boolean;
		viewURL?: string;
	} = $props();

	let commentText = $state("");
	let posting = $state(false);
	// Local copy so we can optimistically add/remove without mutating props.
	let localComments = $state<IndexedComment[]>([]);
	// Reply state: the rkey of the comment being replied to ("" = no reply
	// form open). The old templ stack allowed replies up to depth 2.
	let replyToRKey = $state("");
	let replyText = $state("");
	let postingReply = $state(false);

	// Effective viewer DID: prefer the explicitly-passed prop (entity view
	// pages may forward it), fall back to the session store DID so the
	// delete button renders on the viewer's own comments even when the
	// page passes an empty string.
	let effectiveCurrentUserDID = $derived(currentUserDID || $session.did || "");

	$effect(() => {
		localComments = [...(comments ?? [])];
	});

	function commentDisplayName(c: IndexedComment): string {
		return c.display_name || c.handle || c.actor_did;
	}

	function commentHandle(c: IndexedComment): string {
		return c.handle || c.actor_did;
	}

	function commentAvatar(c: IndexedComment): string {
		return c.avatar || "";
	}

	function formatTimeAgo(iso: string): string {
		const date = new Date(iso);
		if (isNaN(date.getTime())) return "";
		const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
		if (seconds < 60) return "just now";
		const minutes = Math.floor(seconds / 60);
		if (minutes < 60) return `${minutes}m ago`;
		const hours = Math.floor(minutes / 60);
		if (hours < 24) return `${hours}h ago`;
		const days = Math.floor(hours / 24);
		if (days < 7) return `${days}d ago`;
		return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
	}

	function depthClass(depth: number): string {
		if (depth <= 0) return "";
		return `comment-depth-${Math.min(depth, 3)}`;
	}

	async function postComment(parentRKey = "") {
		if (!subjectURI || !subjectCID) return;
		if (!commentText.trim()) return;
		posting = true;
		try {
			const body = new URLSearchParams({
				subject_uri: subjectURI,
				subject_cid: subjectCID,
				text: commentText,
			});
			if (parentRKey) body.set("parent_rkey", parentRKey);
			const res = await fetch("/api/comments", {
				method: "POST",
				credentials: "same-origin",
				headers: {
					"Content-Type": "application/x-www-form-urlencoded",
					Accept: "application/json",
				},
				body,
			});
			if (!res.ok) throw new Error(`Comment failed: ${res.status}`);
			// Parse the JSON response (P1.9) to sync the updated comment thread.
			const contentType = res.headers.get("content-type") ?? "";
			if (contentType.includes("application/json")) {
				const data = await res.json();
				if (Array.isArray(data.comments)) localComments = data.comments;
			} else {
				await refetchComments();
			}
			commentText = "";
		} catch (error) {
			console.error("Comment failed:", error);
			pushToast("Failed to post comment");
		} finally {
			posting = false;
		}
	}

	async function refetchComments() {
		if (!subjectURI) return;
		try {
			const res = await fetch(
				`/api/comments?subject_uri=${encodeURIComponent(subjectURI)}`,
				{ credentials: "same-origin", headers: { Accept: "application/json" } },
			);
			if (!res.ok) return;
			const contentType = res.headers.get("content-type") ?? "";
			if (contentType.includes("application/json")) {
				const data = await res.json();
				if (Array.isArray(data.comments)) localComments = data.comments;
			}
		} catch {
			// Ignore — keep the optimistic state.
		}
	}

	// Build the AT-URI for a comment so it can be liked via /api/likes/toggle.
	// Mirrors buildCommentURI in the old templ stack.
	function commentURI(c: IndexedComment): string {
		return `at://${c.actor_did}/${definitionFor($app).commentCollection}/${c.rkey}`;
	}

	// Share URL for a comment: the parent record's view URL + #comment-rkey.
	function commentShareURL(c: IndexedComment): string {
		if (!viewURL) return "";
		return `${viewURL}#comment-${c.rkey}`;
	}

	async function toggleCommentLike(c: IndexedComment) {
		const uri = commentURI(c);
		if (!uri || !c.cid) return;
		try {
			const res = await fetch("/api/likes/toggle", {
				method: "POST",
				credentials: "same-origin",
				headers: {
					"Content-Type": "application/x-www-form-urlencoded",
					Accept: "application/json",
				},
				body: new URLSearchParams({
					subject_uri: uri,
					subject_cid: c.cid,
				}),
			});
			if (!res.ok) throw new Error(`Like failed: ${res.status}`);
			const contentType = res.headers.get("content-type") ?? "";
			if (contentType.includes("application/json")) {
				const data = await res.json();
				// Update the local comment in place.
				localComments = localComments.map((lc) =>
					lc.rkey === c.rkey
						? { ...lc, is_liked: !!data.is_liked, like_count: data.like_count ?? lc.like_count }
						: lc,
				);
			}
		} catch (error) {
			console.error("Comment like failed:", error);
			pushToast("Failed to like comment");
		}
	}

	async function postReply(parentRKey: string) {
		if (!subjectURI || !subjectCID) return;
		if (!replyText.trim()) return;
		postingReply = true;
		try {
			const body = new URLSearchParams({
				subject_uri: subjectURI,
				subject_cid: subjectCID,
				text: replyText,
				parent_rkey: parentRKey,
			});
			const res = await fetch("/api/comments", {
				method: "POST",
				credentials: "same-origin",
				headers: {
					"Content-Type": "application/x-www-form-urlencoded",
					Accept: "application/json",
				},
				body,
			});
			if (!res.ok) throw new Error(`Reply failed: ${res.status}`);
			const contentType = res.headers.get("content-type") ?? "";
			if (contentType.includes("application/json")) {
				const data = await res.json();
				if (Array.isArray(data.comments)) localComments = data.comments;
			} else {
				await refetchComments();
			}
			replyText = "";
			replyToRKey = "";
		} catch (error) {
			console.error("Reply failed:", error);
			pushToast("Failed to post reply");
		} finally {
			postingReply = false;
		}
	}

	function startReply(rkey: string) {
		replyToRKey = replyToRKey === rkey ? "" : rkey;
		replyText = "";
	}

	function cancelReply() {
		replyToRKey = "";
		replyText = "";
	}

	async function deleteComment(rkey: string) {
		if (!confirm("Delete this comment?")) return;
		try {
			const res = await fetch(`/api/comments/${rkey}`, {
				method: "DELETE",
				credentials: "same-origin",
				headers: { Accept: "application/json" },
			});
			if (!res.ok) throw new Error(`Delete failed: ${res.status}`);
			// If the endpoint returns the updated thread, sync it.
			const contentType = res.headers.get("content-type") ?? "";
			if (contentType.includes("application/json")) {
				const data = await res.json();
				if (Array.isArray(data.comments)) localComments = data.comments;
			} else {
				localComments = localComments.filter((c) => c.rkey !== rkey);
			}
			pushToast("Comment deleted");
		} catch (error) {
			console.error("Delete failed:", error);
			pushToast("Failed to delete comment");
		}
	}

	let canComment = $derived(subjectURI !== "" && subjectCID !== "");
</script>

<div id="comment-section" class="comment-section" data-svelte-comment-section>
	<div class="comment-section-header">
		<div class="flex items-center gap-2">
			<svg class="w-5 h-5 text-muted" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" d="M20.25 8.511c.884.284 1.5 1.128 1.5 2.097v4.286c0 1.136-.847 2.1-1.98 2.193-.34.027-.68.052-1.02.072v3.091l-3-3c-1.354 0-2.694-.055-4.02-.163a2.115 2.115 0 0 1-.825-.242m9.345-8.334a2.126 2.126 0 0 0-.476-.095 48.64 48.64 0 0 0-8.048 0c-1.131.094-1.976 1.057-1.976 2.192v4.286c0 .837.46 1.58 1.155 1.951m9.345-8.334V6.637c0-1.621-1.152-3.026-2.76-3.235A48.455 48.455 0 0 0 11.25 3c-2.115 0-4.198.137-6.24.402-1.608.209-2.76 1.614-2.76 3.235v6.226c0 1.621 1.152 3.026 2.76 3.235.577.075 1.157.14 1.74.194V21l4.155-4.155"></path>
			</svg>
			<h2 class="text-lg font-semibold text-primary">Discussion</h2>
			{#if localComments.length > 0}
				<span class="comment-count-badge">{localComments.length}</span>
			{/if}
		</div>
	</div>

	{#if !canComment}
		<div class="comment-login-prompt">
			<svg class="w-5 h-5 text-faint flex-shrink-0" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6l4 2m6-2a10 10 0 1 1-20 0 10 10 0 0 1 20 0Z"></path>
			</svg>
			<p class="text-sm text-muted">Comments will be available once this record is indexed.</p>
		</div>
	{:else if isAuthenticated}
		<form
			onsubmit={(e) => { e.preventDefault(); void postComment(); }}
			class="comment-compose"
		>
			<textarea
				bind:value={commentText}
				placeholder="Share your thoughts..."
				class="comment-textarea"
				rows="2"
				maxlength="1000"
				required
				aria-label="Write a comment"
			></textarea>
			<div class="flex justify-between items-center">
				<span class="text-xs text-placeholder tracking-wide">1000 char limit</span>
				<button type="submit" class="btn-primary text-sm py-1.5 px-5" disabled={posting}>
					{posting ? "Posting..." : "Post"}
				</button>
			</div>
		</form>
	{:else}
		<div class="comment-login-prompt">
			<p class="text-sm text-muted">
				<a href="/login" class="font-semibold text-secondary hover:text-brown-950 underline underline-offset-2 decoration-brown-400 hover:decoration-brown-700 transition-colors">Log in</a> to join the conversation.
			</p>
		</div>
	{/if}

	<div class="comment-list">
		{#if localComments.length === 0}
			<div class="comment-empty-state">
				<p class="text-faint text-sm font-medium">No comments yet</p>
				<p class="text-placeholder text-xs mt-1">Be the first to share your thoughts</p>
			</div>
		{:else}
			{#each localComments as comment (comment.rkey)}
				<div class={`comment-item ${depthClass(comment.depth)}`} id={`comment-${comment.rkey}`}>
					{#if comment.depth > 0}
						<div class="comment-thread-line"></div>
					{/if}
					<div class="comment-item-inner">
						<div class="flex items-center justify-between gap-2 mb-1.5">
							<a href={`/profile/${commentHandle(comment)}`} class="flex items-center gap-2">
								<Avatar avatarURL={commentAvatar(comment)} displayName={commentDisplayName(comment)} size="sm" />
								<div>
									<span class="text-sm font-medium text-primary">{commentDisplayName(comment)}</span>
									<span class="text-xs text-muted ml-1">@{displayHandle(commentHandle(comment))}</span>
									<span class="text-xs text-faint ml-1">{formatTimeAgo(comment.created_at)}</span>
								</div>
							</a>
						</div>
						<p class="text-secondary whitespace-pre-wrap wrap-break-word pl-11 text-sm leading-relaxed">{comment.text}</p>
						<div class="pl-11 mt-1 flex items-center gap-3">
							<button
								type="button"
								onclick={() => toggleCommentLike(comment)}
								class={`text-xs ${comment.is_liked ? "text-red-500" : "text-faint hover:text-red-500"}`}
								aria-label={comment.is_liked ? "Unlike comment" : "Like comment"}
							>
								♥ {comment.is_liked ? "Liked" : "Like"}{comment.like_count > 0 ? ` (${comment.like_count})` : ""}
							</button>
							{#if isAuthenticated && comment.depth < 2 && comment.cid}
								<button
									type="button"
									onclick={() => startReply(comment.rkey)}
									class="text-xs text-faint hover:text-secondary"
									aria-label="Reply to comment"
								>
									Reply
								</button>
							{/if}
							{#if commentShareURL(comment)}
								<a
									href={commentShareURL(comment)}
									class="text-xs text-faint hover:text-secondary"
									aria-label="Share comment"
								>
									Share
								</a>
							{/if}
							{#if effectiveCurrentUserDID === comment.actor_did}
								<button
									type="button"
									onclick={() => deleteComment(comment.rkey)}
									class="text-xs text-faint hover:text-red-600"
								>
									Delete
								</button>
							{/if}
						</div>
						{#if replyToRKey === comment.rkey}
							<form
								onsubmit={(e) => { e.preventDefault(); void postReply(comment.rkey); }}
								class="pl-11 mt-2"
							>
								<textarea
									bind:value={replyText}
									placeholder="Write a reply..."
									class="comment-textarea"
									rows="2"
									maxlength="1000"
									required
									aria-label="Write a reply"
								></textarea>
								<div class="flex justify-end items-center gap-2 mt-1">
									<button type="button" class="btn-secondary text-xs py-1 px-3" onclick={cancelReply}>Cancel</button>
									<button type="submit" class="btn-primary text-xs py-1 px-4" disabled={postingReply}>
										{postingReply ? "Posting..." : "Reply"}
									</button>
								</div>
							</form>
						{/if}
					</div>
				</div>
			{/each}
		{/if}
	</div>
</div>
