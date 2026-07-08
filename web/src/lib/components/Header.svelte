<script lang="ts">
	import Icon from "./Icon.svelte";
	import Avatar from "./Avatar.svelte";
	import {
		session,
		app,
		profileIdentifier,
		displayHandle,
		formatNotificationCount,
	} from "../stores/session";

	let {
		brandName = "Arabica",
	}: { brandName?: string } = $props();

	let createOpen = $state(false);
	let userOpen = $state(false);

	function closeMenus() {
		createOpen = false;
		userOpen = false;
	}

	function toggleCreate() {
		userOpen = false;
		createOpen = !createOpen;
	}

	function toggleUser() {
		createOpen = false;
		userOpen = !userOpen;
	}

	function handleOutsideClick(event: MouseEvent) {
		const target = event.target;
		if (!(target instanceof Element)) return;
		if (!target.closest("[data-menu-root]")) {
			closeMenus();
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === "Escape") closeMenus();
	}

	$effect(() => {
		document.addEventListener("click", handleOutsideClick);
		document.addEventListener("keydown", handleKeydown);
		return () => {
			document.removeEventListener("click", handleOutsideClick);
			document.removeEventListener("keydown", handleKeydown);
		};
	});

	let isOolong = $derived($app === "oolong");
	let s = $derived($session);
	let unread = $derived(s.unreadNotifications);
</script>

<nav
	class="sticky top-0 z-50"
	style="color: var(--header-text); background: linear-gradient(135deg, var(--header-bg-from), var(--header-bg-to)); border-bottom: 1px solid var(--header-border); box-shadow: var(--shadow-sm);"
>
	<div class="container mx-auto px-4 py-3">
		<div class="flex items-center justify-between">
			<!-- Logo -->
			<a href="/" class="flex items-center gap-2 hover:opacity-80 transition">
				<h1 class="text-2xl font-bold">☕ {brandName}</h1>
				<span class="badge-alpha">ALPHA</span>
			</a>

			<div class="flex items-center gap-4">
				{#if !s.isAuthenticated}
					<a href="/login" class="text-sm font-semibold transition-colors hover:opacity-80" style="color: var(--header-text);">
						Log In
					</a>
				{:else}
					<!-- Notification bell -->
					<a href="/notifications" class="relative hover:opacity-80 transition p-2" title="Notifications" aria-label="Notifications">
						<Icon name="bell" class="w-6 h-6" />
						{#if unread > 0}
							<span class="absolute -top-1 -right-1 bg-amber-400 text-primary text-xs font-bold rounded-full min-w-[18px] h-[18px] flex items-center justify-center px-1 shadow-xs">
								{formatNotificationCount(unread)}
							</span>
						{/if}
					</a>

					<!-- Create new dropdown -->
					<div class="relative" data-menu-root>
						<button
							type="button"
							onclick={toggleCreate}
							class="hover:opacity-80 transition p-2 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-amber-400/50 rounded-sm"
							title="Create new"
							aria-label="Create new"
							aria-expanded={createOpen}
						>
							<Icon name="plus" class="w-6 h-6" />
						</button>
						{#if createOpen}
							<div class="dropdown-menu w-52" role="menu">
								<div class="dropdown-header">
									<p class="text-xs font-semibold uppercase tracking-wider text-faint">Log</p>
								</div>
								<a href="/brews/new" class="dropdown-item" role="menuitem">
									{#if isOolong}<Icon name="droplet" />{:else}<Icon name="coffee" />{/if}
									{isOolong ? "New Steep" : "New Brew"}
								</a>
								<div class="dropdown-divider">
									<p class="px-4 pt-1 text-xs font-semibold uppercase tracking-wider text-faint">Add</p>
								</div>
								{#if isOolong}
									<a href="/teas/new" class="dropdown-item" role="menuitem">
										<Icon name="leaf" />Tea
									</a>
									<a href="/vendors/new" class="dropdown-item" role="menuitem">
										<Icon name="store" />Vendor
									</a>
									<a href="/vessels/new" class="dropdown-item" role="menuitem">
										<Icon name="brewer" />Vessel
									</a>
									<a href="/infusers/new" class="dropdown-item" role="menuitem">
										<Icon name="disc" />Infuser
									</a>
								{:else}
									<a href="/beans/new" class="dropdown-item" role="menuitem">
										<Icon name="leaf" />Bean
									</a>
									<a href="/roasters/new" class="dropdown-item" role="menuitem">
										<Icon name="store" />Roaster
									</a>
									<a href="/grinders/new" class="dropdown-item" role="menuitem">
										<Icon name="disc" />Grinder
									</a>
									<a href="/brewers/new" class="dropdown-item" role="menuitem">
										<Icon name="brewer" />Brewer
									</a>
									<a href="/recipes/new" class="dropdown-item" role="menuitem">
										<Icon name="fileText" />Recipe
									</a>
								{/if}
							</div>
						{/if}
					</div>

					<!-- User profile dropdown -->
					<div class="relative" data-menu-root>
						<button
							type="button"
							onclick={toggleUser}
							class="flex items-center gap-2 hover:opacity-80 transition focus:outline-hidden focus-visible:ring-2 focus-visible:ring-amber-400/50 rounded-sm"
							aria-label="User menu"
							aria-expanded={userOpen}
						>
							<Avatar avatarURL={s.avatar} displayName={s.displayName} size="sm" />
							<Icon name="chevronDown" class="w-4 h-4 transition-transform" />
						</button>
						{#if userOpen}
							<div class="dropdown-menu" role="menu">
								{#if s.handle}
									<div class="dropdown-header">
										<p class="text-sm font-medium text-primary truncate">
											{s.displayName || displayHandle(s.handle)}
										</p>
										<p class="text-xs text-faint truncate">@{displayHandle(s.handle)}</p>
									</div>
								{/if}
								<a href={`/profile/${profileIdentifier(s)}`} class="dropdown-item" role="menuitem">
									View Profile
								</a>
								{#if isOolong}
									<a href="/my-tea" class="dropdown-item" role="menuitem">My Tea</a>
								{:else}
									<a href="/explore" class="dropdown-item" role="menuitem">Explore</a>
									<a href="/my-coffee" class="dropdown-item" role="menuitem">My Coffee</a>
									<a href="/recipes" class="dropdown-item" role="menuitem">Recipes</a>
								{/if}
								<a href="/settings" class="dropdown-item" role="menuitem">Settings</a>
								{#if s.isModerator}
									<div class="dropdown-divider"></div>
									<a href="/_mod" class="dropdown-item dropdown-item-mod" role="menuitem">
										<Icon name="shieldCheck" class="w-4 h-4 inline-block mr-1" />
										Moderation
									</a>
								{/if}
								<div class="dropdown-divider">
									<form action="/logout" method="POST">
										<button type="submit" class="dropdown-item w-full text-left" role="menuitem">
											Logout
										</button>
									</form>
								</div>
							</div>
						{/if}
					</div>
				{/if}
			</div>
		</div>
	</div>
</nav>
