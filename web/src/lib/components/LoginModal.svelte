<script lang="ts">
	import Icon from "./Icon.svelte";
	import Avatar from "./Avatar.svelte";

	let {
		open = $bindable(false),
	}: {
		open?: boolean;
	} = $props();

	let dialog = $state<HTMLDialogElement>();
	let handleInput = $state<HTMLInputElement>();
	let handle = $state("");

	type Actor = {
		handle: string;
		displayName?: string;
		avatar?: string;
	};
	let actors = $state<Actor[]>([]);
	let autocompleteOpen = $state(false);
	let loading = $state(false);
	let searched = $state(false);
	let debounceTimer: ReturnType<typeof setTimeout> | undefined;
	let abortController: AbortController | undefined;
	let popoverEl = $state<HTMLDivElement>();
	let dropdownStyle = $state("");
	let suppressSearch = $state(false);
	let activeIndex = $state(-1);
	let justSelected = $state(false);
	let justSelectedTimer: ReturnType<typeof setTimeout> | undefined;

	function safeAvatar(actor: Actor) {
		const avatar = actor.avatar || "";
		if (avatar.startsWith("https://") || avatar.startsWith("/static/")) {
			return avatar;
		}
		return "/static/icon-placeholder.svg";
	}

	function displayName(actor: Actor) {
		return actor.displayName || actor.handle;
	}

	function updateDropdownPosition() {
		if (!autocompleteOpen || !handleInput) return;
		const rect = handleInput.getBoundingClientRect();
		const below = window.innerHeight - rect.bottom;
		const above = rect.top;
		const minHeight = 8 * 16;
		const preferAbove = below < minHeight && above > below;
		const available = preferAbove ? above : below;
		const maxH = Math.min(15 * 16, available - 8);
		const top = preferAbove ? rect.top - maxH - 4 : rect.bottom + 4;
		dropdownStyle =
			`position:fixed;left:${rect.left}px;width:${rect.width}px;` +
			`top:${top}px;max-height:${maxH}px;`;
	}

	function openDropdown() {
		if (!popoverEl) return;
		if (typeof popoverEl.showPopover !== "function") return;
		try {
			popoverEl.showPopover();
		} catch {
			// ignore
		}
		updateDropdownPosition();
	}

	function closeDropdown() {
		if (popoverEl && typeof popoverEl.hidePopover === "function") {
			try {
				popoverEl.hidePopover();
			} catch {
				// ignore
			}
		}
	}

	function clearResults() {
		window.clearTimeout(debounceTimer);
		actors = [];
		autocompleteOpen = false;
		searched = false;
	}

	async function searchActors(query: string) {
		const trimmed = query.trim();
		if (trimmed.length < 3) {
			clearResults();
			return;
		}

		abortController?.abort();
		abortController = new AbortController();
		loading = true;
		searched = true;

		try {
			const response = await fetch(
				`/api/search-actors?q=${encodeURIComponent(trimmed)}`,
				{ signal: abortController.signal },
			);
			if (!response.ok) {
				actors = [];
				autocompleteOpen = false;
				return;
			}
			const data = await response.json();
			actors = Array.isArray(data?.actors) ? data.actors : [];
			autocompleteOpen = true;
		} catch (error) {
			if ((error as Error).name !== "AbortError") {
				console.error("Error searching actors:", error);
			}
		} finally {
			loading = false;
		}
	}

	function scheduleSearch() {
		if (suppressSearch) return;
		window.clearTimeout(debounceTimer);
		debounceTimer = window.setTimeout(() => {
			void searchActors(handle);
		}, 300);
	}

	function selectActor(actor: Actor, deferClose = false) {
		handle = actor.handle;
		suppressSearch = true;
		activeIndex = -1;
		// Briefly flag that a suggestion was just picked. This lets the
		// form's submit handler ignore any spurious submit events that some
		// browsers synthesize when a popover button is removed during a click.
		justSelected = true;
		window.clearTimeout(justSelectedTimer);
		justSelectedTimer = window.setTimeout(() => {
			justSelected = false;
		}, 500);

		const finish = () => {
			clearResults();
			// Refocus the input after selection so the user can submit.
			setTimeout(() => {
				handleInput?.focus();
				suppressSearch = false;
			}, 0);
		};

		if (deferClose) {
			// Defer closing the dropdown until after the current click sequence
			// completes so the browser doesn't retarget the click to the submit
			// button underneath the popover.
			window.setTimeout(finish, 0);
		} else {
			finish();
		}
	}

	function handleSuggestionClick(event: MouseEvent, actor: Actor) {
		// Prevent the suggestion click from accidentally submitting the form.
		// The button is type="button", but explicit prevention keeps the
		// behavior robust across browsers and focus changes.
		event.preventDefault();
		event.stopPropagation();
		selectActor(actor, true);
	}

	function handleFormSubmit(event: SubmitEvent) {
		// If the user just picked a suggestion, ignore the submit event. This
		// catches spurious submissions that some browsers emit when the
		// popover button disappears during the click.
		if (justSelected) {
			event.preventDefault();
		}
	}

	function handleInputKeydown(event: KeyboardEvent) {
		if (!autocompleteOpen) return;

		switch (event.key) {
			case "ArrowDown":
				event.preventDefault();
				activeIndex =
					activeIndex < actors.length - 1 ? activeIndex + 1 : 0;
				break;
			case "ArrowUp":
				event.preventDefault();
				activeIndex =
					activeIndex > 0 ? activeIndex - 1 : actors.length - 1;
				break;
			case "Enter":
				event.preventDefault();
				if (actors.length > 0) {
					selectActor(actors[activeIndex >= 0 ? activeIndex : 0]);
				}
				break;
			case "Escape":
				event.preventDefault();
				clearResults();
				activeIndex = -1;
				break;
		}
	}

	$effect(() => {
		if (actors.length === 0) {
			activeIndex = -1;
		} else if (activeIndex >= actors.length) {
			activeIndex = actors.length - 1;
		}
	});

	$effect(() => {
		if (autocompleteOpen) {
			openDropdown();
		} else {
			closeDropdown();
		}
	});

	$effect(() => {
		if (autocompleteOpen) updateDropdownPosition();
	});

	// Drive the native dialog from the open prop so callers can show/hide it
	// programmatically (e.g. from the header or page CTAs).
	$effect(() => {
		const d = dialog;
		if (!d) return;
		if (open && !d.open) {
			d.showModal();
			handle = "";
			clearResults();
		} else if (!open && d.open) {
			d.close();
		}
	});

	function handleDialogClose() {
		open = false;
		handle = "";
		clearResults();
	}

	function handleBackdropClick(event: MouseEvent) {
		if (event.target === dialog) {
			open = false;
		}
	}

	function handleDocumentClick(event: MouseEvent) {
		const clicked = event.target;
		if (!(clicked instanceof Node)) return;
		if (
			!handleInput?.contains(clicked) &&
			!popoverEl?.contains(clicked)
		) {
			autocompleteOpen = false;
		}
	}

	$effect(() => {
		document.addEventListener("click", handleDocumentClick);
		window.addEventListener("resize", updateDropdownPosition);
		window.addEventListener("scroll", updateDropdownPosition, true);
		return () => {
			document.removeEventListener("click", handleDocumentClick);
			window.removeEventListener("resize", updateDropdownPosition);
			window.removeEventListener("scroll", updateDropdownPosition, true);
			closeDropdown();
			window.clearTimeout(debounceTimer);
			window.clearTimeout(justSelectedTimer);
			abortController?.abort();
		};
	});
</script>

<dialog
	bind:this={dialog}
	class="modal-dialog"
	aria-labelledby="login-title"
	data-testid="login-modal"
	onclose={handleDialogClose}
	onclick={handleBackdropClick}
>
	<div class="modal-content">
		<div class="flex items-center justify-between mb-4">
			<h2 id="login-title" class="modal-title mb-0">
				Log in with your Atmosphere account
			</h2>
			<button
				type="button"
				onclick={() => (open = false)}
				class="text-placeholder hover:text-muted transition-colors"
				aria-label="Close"
			>
				<Icon name="x" class="w-5 h-5" />
			</button>
		</div>
		<form
			method="POST"
			action="/auth/login"
			class="space-y-4"
			data-testid="login-form"
			onsubmit={handleFormSubmit}
		>
			<div class="relative">
				<label for="login-handle" class="block text-sm font-medium text-primary mb-2"
					>Handle</label
				>
				<input
					bind:this={handleInput}
					type="text"
					id="login-handle"
					name="handle"
					placeholder="your-handle.bsky.social"
					autocomplete="off"
					required
					class="w-full form-input-lg"
					bind:value={handle}
					oninput={scheduleSearch}
					onkeydown={handleInputKeydown}
					onfocus={() => {
						if (searched && handle.trim().length >= 3) autocompleteOpen = true;
					}}
				/>
				<!--
					The dropdown is rendered as a manual popover so it paints above
					the dialog's ::backdrop. We keep it inside the form's relative
					container so clicks on suggestion buttons are part of the same
					interaction tree and work reliably across browsers.
				-->
				<div
					bind:this={popoverEl}
					popover="manual"
					class="handle-dropdown"
					style={dropdownStyle}
				>
					{#if loading && actors.length === 0}
						<div class="handle-no-results">Searching...</div>
					{:else if actors.length === 0}
						<div class="handle-no-results">No accounts found</div>
					{:else}
						{#each actors as actor, i (actor.handle)}
							<button
								type="button"
								class="handle-result"
								class:active={activeIndex === i}
								data-handle={actor.handle}
								onclick={(event) => handleSuggestionClick(event, actor)}
							>
								<Avatar
									avatarURL={safeAvatar(actor)}
									displayName={displayName(actor)}
									size="sm"
								/>
								<span class="handle-result-text">
									<span class="handle-name">{displayName(actor)}</span>
									<span class="handle-at">@{actor.handle}</span>
								</span>
							</button>
						{/each}
					{/if}
				</div>
			</div>
			<button
				type="submit"
				disabled={autocompleteOpen}
				class="btn-primary w-full py-3 font-semibold"
				class:opacity-50={autocompleteOpen}
				class:cursor-not-allowed={autocompleteOpen}
			>
				Log In
			</button>
		</form>
		<div class="mt-4 text-sm text-muted text-center">
			<a
				href="/join/create"
				class="font-medium text-secondary hover:text-primary transition-colors hover:underline"
			>
				Create an account
			</a>
			<span class="mx-1.5 text-placeholder">&middot;</span>
			<a
				href="/about"
				class="text-muted hover:text-secondary transition-colors hover:underline"
			>
				Learn more
			</a>
		</div>
		<details class="mt-4">
			<summary
				class="text-faint text-sm cursor-pointer hover:text-emphasis transition-colors"
			>
				What's an Atmosphere account?
			</summary>
			<p class="text-muted mt-2 text-sm leading-relaxed">
				One account for the entire
				<a href="/atproto" class="link">AT Protocol</a>
				ecosystem. Sign up once and use it across Arabica,
				<a href="https://bsky.app" class="link" target="_blank" rel="noopener noreferrer"
					>Bluesky</a
				>,
				<a href="https://leaflet.pub" class="link" target="_blank" rel="noopener noreferrer"
					>Leaflet</a
				>, and more. Your data is portable — you own it.
			</p>
		</details>
	</div>
</dialog>
