<script lang="ts">
	import Avatar from "$lib/components/Avatar.svelte";
	import BackButton from "$lib/components/BackButton.svelte";
	import { pushToast } from "$lib/stores/toasts";
	import { displayHandle } from "$lib/stores/session";
	import { markAllNotificationsRead } from "$lib/api/notifications";
	import type { PageData } from "./$types";
	import type { NotificationItem } from "$lib/types/api";

	let { data }: { data: PageData } = $props();

	function actorName(n: NotificationItem): string {
		return n.actor_display_name || displayHandle(n.actor_handle) || n.actor_did;
	}

	function timeAgo(iso: string): string {
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

	// svelte-ignore state_referenced_locally
	let notifications = $state<NotificationItem[]>(data.notifications ?? []);
	let markingRead = $state(false);

	$effect(() => {
		notifications = data.notifications ?? [];
	});

	async function markAllRead() {
		if (markingRead) return;
		markingRead = true;
		try {
			await markAllNotificationsRead(fetch);
			notifications = notifications.map((n) => ({ ...n, read: true }));
			pushToast("Marked all as read");
		} catch {
			pushToast("Failed to mark as read");
		} finally {
			markingRead = false;
		}
	}
</script>

<svelte:head>
	<title>Notifications - Arabica</title>
</svelte:head>

<div class="page-container-md">
	<div class="flex items-center justify-between mb-8">
		<div class="flex items-center gap-3">
			<BackButton />
			<h1 class="text-2xl font-semibold text-primary">Notifications</h1>
		</div>
		{#if notifications.length > 0}
			<button type="button" class="btn-secondary text-sm" onclick={markAllRead} disabled={markingRead}>
				{markingRead ? "Marking..." : "Mark all as read"}
			</button>
		{/if}
	</div>

	{#if data.error}
		<div class="card card-inner text-center py-8">
			<p class="text-secondary mb-4">{data.error}</p>
			<a href="/login" class="btn-primary">Log In</a>
		</div>
	{:else if notifications.length === 0}
		<div class="card card-inner text-center py-12">
			<div class="text-4xl mb-3">🔔</div>
			<p class="text-emphasis font-medium mb-1">No notifications yet</p>
			<p class="text-sm text-muted">When someone likes or comments on your brews, you'll see it here.</p>
		</div>
	{:else}
		<div class="space-y-2">
			{#each notifications as notif (notif.id)}
				<a
					href={notif.link || "/notifications"}
					class="card card-inner flex items-start gap-3 p-4 no-underline hover:bg-brown-50 transition-colors cursor-pointer"
					class:bg-amber-50={!notif.read}
					class:border-amber-200={!notif.read}
				>
					<div class="shrink-0">
						<Avatar avatarURL={notif.actor_avatar} displayName={actorName(notif)} size="sm" />
					</div>
					<div class="flex-1 min-w-0">
						<p class="text-sm text-secondary">
							<span class="font-semibold text-primary">{actorName(notif)}</span>
							{notif.action_text}
						</p>
						<p class="text-xs text-placeholder mt-1">{timeAgo(notif.created_at)}</p>
					</div>
					{#if !notif.read}
						<div class="shrink-0 mt-1">
							<span class="block w-2 h-2 rounded-full bg-amber-400"></span>
						</div>
					{/if}
				</a>
			{/each}
		</div>
		{#if data.nextCursor}
			<div class="mt-6 text-center">
				<a href={`/notifications?cursor=${data.nextCursor}`} class="btn-secondary">Load more</a>
			</div>
		{/if}
	{/if}
</div>
