<script lang="ts">
	import { toasts, pushToast } from "../stores/toasts";
	import { session } from "../stores/session";

	type Props = {
		subjectURI: string;
		subjectCID: string;
		isLiked: boolean;
		likeCount: number;
		commentCount: number;
		shareURL: string;
		shareTitle?: string;
		shareText?: string;
		isOwner: boolean;
		editURL?: string;
		editModalURL?: string;
		deleteURL?: string;
		deleteRedirect?: string;
		viewURL?: string;
		isAuthenticated: boolean;
		isModerator?: boolean;
		canHideRecord?: boolean;
		canBlockUser?: boolean;
		isRecordHidden?: boolean;
		authorDID?: string;
	};

	let {
		subjectURI = "",
		subjectCID = "",
		isLiked = $bindable(false),
		likeCount = $bindable(0),
		commentCount = 0,
		shareURL = "",
		shareTitle = "",
		shareText = "",
		isOwner = false,
		editURL = "",
		editModalURL = "",
		deleteURL = "",
		deleteRedirect = "",
		viewURL = "",
		isAuthenticated = false,
		isModerator = false,
		canHideRecord = false,
		canBlockUser = false,
		isRecordHidden = false,
		authorDID = "",
	}: Props = $props();

	let menuOpen = $state(false);
	let shareCopied = $state(false);
	let likePending = $state(false);
	let reportOpen = $state(false);
	let reportReason = $state("");
	let reportError = $state("");
	let reportSubmitted = $state(false);

	function commentHref() {
		if (viewURL) return `${viewURL}#comment-section`;
		return "#comment-section";
	}

	function hasModActions() {
		return (canHideRecord && subjectURI !== "") || (canBlockUser && authorDID !== "" && !isOwner);
	}

	function hasReportAction() {
		return isAuthenticated && !isOwner;
	}

	async function toggleLike() {
		if (!isAuthenticated) {
			pushToast("Log in to like this");
			return;
		}
		if (likePending) return;
		likePending = true;
		const prevLiked = isLiked;
		const prevCount = likeCount;
		// Optimistic update
		isLiked = !prevLiked;
		likeCount = prevCount + (prevLiked ? -1 : 1);
		try {
			const res = await fetch("/api/likes/toggle", {
				method: "POST",
				credentials: "same-origin",
				headers: { "Content-Type": "application/x-www-form-urlencoded" },
				body: new URLSearchParams({
					subject_uri: subjectURI,
					subject_cid: subjectCID,
				}),
			});
			if (!res.ok) throw new Error(`Like failed: ${res.status}`);
			// The HTMX endpoint returns HTML; we ignore the body and trust the
			// optimistic state. When P1.9 ships a JSON response, parse it here.
		} catch (error) {
			// Revert on failure
			isLiked = prevLiked;
			likeCount = prevCount;
			console.error("Like toggle failed:", error);
			pushToast("Failed to update like");
		} finally {
			likePending = false;
		}
	}

	async function share() {
		if (!shareURL) return;
		const url = window.location.origin + shareURL;
		if (navigator.share) {
			try {
				await navigator.share({ title: shareTitle, text: shareText, url });
			} catch {
				// User cancelled
			}
			return;
		}
		try {
			await navigator.clipboard.writeText(url);
			shareCopied = true;
			setTimeout(() => (shareCopied = false), 2000);
		} catch {
			// Silent
		}
	}

	async function doDelete() {
		if (!deleteURL) return;
		try {
			const res = await fetch(deleteURL, {
				method: "DELETE",
				credentials: "same-origin",
			});
			if (!res.ok) throw new Error(`Delete failed: ${res.status}`);
			if (deleteRedirect) {
				window.location.href = deleteRedirect;
			}
		} catch (error) {
			console.error("Delete failed:", error);
			pushToast("Failed to delete");
		}
	}

	function confirmDelete() {
		menuOpen = false;
		if (confirm("Are you sure you want to delete this?")) {
			void doDelete();
		}
	}

	async function modAction(action: string, url: string, payload: Record<string, string>, confirmMsg?: string) {
		menuOpen = false;
		if (confirmMsg && !confirm(confirmMsg)) return;
		try {
			const res = await fetch(url, {
				method: "POST",
				credentials: "same-origin",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify(payload),
			});
			if (!res.ok) throw new Error(`${action} failed: ${res.status}`);
			pushToast(action === "hide" ? "Record hidden" : action === "unhide" ? "Record unhidden" : "User blocked");
			if (action === "hide") isRecordHidden = true;
			if (action === "unhide") isRecordHidden = false;
		} catch (error) {
			console.error(`${action} failed:`, error);
			pushToast(`Failed to ${action}`);
		}
	}

	async function submitReport() {
		reportError = "";
		try {
			const res = await fetch("/api/report", {
				method: "POST",
				credentials: "same-origin",
				headers: { "Content-Type": "application/x-www-form-urlencoded" },
				body: new URLSearchParams({
					subject_uri: subjectURI,
					subject_cid: subjectCID,
					reason: reportReason,
				}),
			});
			if (!res.ok) throw new Error(`Report failed: ${res.status}`);
			reportSubmitted = true;
		} catch (error) {
			reportError = "Failed to submit report. Please try again.";
			console.error("Report failed:", error);
		}
	}

	function handleOutsideClick(event: MouseEvent) {
		const target = event.target;
		if (!(target instanceof Element)) return;
		if (!target.closest("[data-more-menu-root]")) {
			menuOpen = false;
		}
	}

	$effect(() => {
		document.addEventListener("click", handleOutsideClick);
		return () => document.removeEventListener("click", handleOutsideClick);
	});

	let s = $derived($session);
</script>

<div class="action-bar">
	<!-- Comments -->
	<a href={commentHref()} class="action-btn" title="View comments">
		<svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" d="M8.625 12a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0H8.25m4.125 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0H12m4.125 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 0 1-2.555-.337A5.972 5.972 0 0 1 5.41 20.97a5.969 5.969 0 0 1-.474-.065 4.48 4.48 0 0 0 .978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25Z"></path>
		</svg>
		<span>{commentCount}</span>
	</a>

	<!-- Hidden indicator (visible to moderators) -->
	{#if isModerator && isRecordHidden}
		<span class="hidden-badge" title="This record is hidden from the public feed">
			<svg class="w-3 h-3" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" d="M3.98 8.223A10.477 10.477 0 0 0 1.934 12C3.226 16.338 7.244 19.5 12 19.5c.993 0 1.953-.138 2.863-.395M6.228 6.228A10.451 10.451 0 0 1 12 4.5c4.756 0 8.773 3.162 10.065 7.498a10.522 10.522 0 0 1-4.293 5.774M6.228 6.228 3 3m3.228 3.228 3.65 3.65m7.894 7.894L21 21m-3.228-3.228-3.65-3.65m0 0a3 3 0 1 0-4.243-4.243m4.242 4.242L9.88 9.88"></path>
			</svg>
			Hidden
		</span>
	{/if}

	<!-- Like -->
	{#if subjectURI && subjectCID}
		<button
			type="button"
			onclick={toggleLike}
			disabled={likePending}
			class="action-btn"
			aria-label={isLiked ? "Unlike" : "Like"}
			class:liked={isLiked}
		>
			<svg class="w-4 h-4" fill={isLiked ? "currentColor" : "none"} stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" d="M21 8.25c0-2.485-2.099-4.5-4.688-4.5-1.935 0-3.597 1.126-4.312 2.733-.715-1.607-2.377-2.733-4.312-2.733C5.1 3.75 3 5.765 3 8.25c0 7.22 9 12 9 12s9-4.78 9-12Z"></path>
			</svg>
			<span>{likeCount}</span>
		</button>
	{/if}

	<!-- Share -->
	{#if shareURL}
		<button type="button" onclick={share} class="action-btn" aria-label="Share">
			<svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" d="M7.217 10.907a2.25 2.25 0 1 0 0 2.186m0-2.186c.18.324.283.696.283 1.093s-.103.77-.283 1.093m0-2.186 9.566-5.314m-9.566 7.5 9.566 5.314m0 0a2.25 2.25 0 1 0 3.935 2.186 2.25 2.25 0 0 0-3.935-2.186Zm0-12.814a2.25 2.25 0 1 0 3.933-2.185 2.25 2.25 0 0 0-3.933 2.185Z"></path>
			</svg>
			{#if shareCopied}
				<span class="text-xs text-success" aria-live="polite">Copied!</span>
			{/if}
		</button>
	{/if}

	<!-- More menu -->
	<div class="relative z-10" data-more-menu-root>
		<button
			type="button"
			onclick={() => (menuOpen = !menuOpen)}
			class="action-btn"
			aria-label="More options"
			aria-expanded={menuOpen}
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" d="M6.75 12a.75.75 0 1 1-1.5 0 .75.75 0 0 1 1.5 0ZM12.75 12a.75.75 0 1 1-1.5 0 .75.75 0 0 1 1.5 0ZM18.75 12a.75.75 0 1 1-1.5 0 .75.75 0 0 1 1.5 0Z"></path>
			</svg>
		</button>
		{#if menuOpen}
			<div class="action-menu" role="menu">
				{#if isOwner}
					{#if editURL}
						<a href={editURL} class="action-menu-item" role="menuitem">Edit</a>
					{/if}
					{#if editModalURL}
						<a href={editModalURL} class="action-menu-item" role="menuitem">Edit</a>
					{/if}
					{#if deleteURL}
						<button type="button" onclick={confirmDelete} class="action-menu-item action-menu-item-danger" role="menuitem">
							Delete
						</button>
					{/if}
					{#if hasModActions() || hasReportAction()}
						<div class="action-menu-divider"></div>
					{/if}
				{/if}
				<!-- Moderation actions -->
				{#if canHideRecord && subjectURI}
					{#if isRecordHidden}
						<button
							type="button"
							onclick={() => modAction("unhide", "/_mod/unhide", { uri: subjectURI })}
							class="action-menu-item"
							role="menuitem"
						>
							Unhide from feed
						</button>
					{:else}
						<button
							type="button"
							onclick={() => modAction("hide", "/_mod/hide", { uri: subjectURI }, "Hide this record from the public feed?")}
							class="action-menu-item action-menu-item-warning"
							role="menuitem"
						>
							Hide from feed
						</button>
					{/if}
					{#if (canBlockUser && authorDID && !isOwner) || hasReportAction()}
						<div class="action-menu-divider"></div>
					{/if}
				{/if}
				{#if canBlockUser && authorDID && !isOwner}
					<button
						type="button"
						onclick={() => modAction("block", "/_mod/block", { did: authorDID }, "Block this user? All their content will be hidden from the feed.")}
						class="action-menu-item action-menu-item-danger"
						role="menuitem"
					>
						Block user
					</button>
					{#if hasReportAction()}
						<div class="action-menu-divider"></div>
					{/if}
				{/if}
				<!-- Report -->
				{#if isAuthenticated && !isOwner}
					<button
						type="button"
						onclick={() => { menuOpen = false; reportOpen = true; }}
						class="action-menu-item"
						role="menuitem"
					>
						Report
					</button>
				{/if}
			</div>
		{/if}
	</div>
</div>

<!-- Report modal -->
{#if reportOpen}
	<dialog open class="modal-dialog" aria-labelledby="report-title" data-testid="report-modal">
		<div class="modal-content">
			{#if reportSubmitted}
				<div class="text-center py-4">
					<div class="text-green-600 mb-2">
						<svg class="w-12 h-12 mx-auto" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"></path>
						</svg>
					</div>
					<p class="font-medium text-primary">Report Submitted</p>
					<p class="text-sm text-muted mt-1">Thank you for helping keep our community safe.</p>
					<div class="mt-4 flex justify-center">
						<button type="button" class="btn-secondary px-6" onclick={() => (reportOpen = false)}>Close</button>
					</div>
				</div>
			{:else}
				<h3 id="report-title" class="modal-title">Report Content</h3>
				<form
					onsubmit={(e) => { e.preventDefault(); void submitReport(); }}
					class="space-y-4"
				>
					<p class="text-sm text-emphasis">
						Please describe why you're reporting this content. Reports are reviewed by moderators.
					</p>
					<div>
						<textarea
							bind:value={reportReason}
							placeholder="Describe the issue (optional)"
							rows="4"
							maxlength="500"
							class="w-full form-textarea"
							aria-label="Report reason"
						></textarea>
					</div>
					{#if reportError}
						<div class="bg-red-100 border border-red-300 text-red-800 px-3 py-2 rounded-lg text-sm">
							{reportError}
						</div>
					{/if}
					<div class="flex gap-2">
						<button type="submit" class="flex-1 btn-primary">Submit Report</button>
						<button type="button" class="flex-1 btn-secondary" onclick={() => (reportOpen = false)}>Cancel</button>
					</div>
				</form>
			{/if}
		</div>
	</dialog>
{/if}
