<script lang="ts">
	import { safeAvatarURL } from "../stores/session";

	let {
		avatarURL = "",
		displayName = "",
		size = "md",
	}: {
		avatarURL?: string;
		displayName?: string;
		size?: "sm" | "md" | "lg";
	} = $props();

	let imgClass = $derived(
		size === "sm" ? "avatar-sm" : size === "lg" ? "avatar-lg" : "avatar-md",
	);
	let placeholderClass = $derived(
		size === "sm"
			? "avatar-placeholder-sm"
			: size === "lg"
				? "avatar-placeholder-lg"
				: "avatar-placeholder-md",
	);
	let textClass = $derived(
		size === "sm"
			? "avatar-text-sm"
			: size === "lg"
				? "avatar-text-lg"
				: "avatar-text-md",
	);
	let dimension = $derived(size === "sm" ? "32" : size === "lg" ? "80" : "48");
	let safe = $derived(safeAvatarURL(avatarURL));
	let initial = $derived(displayName && displayName.length > 0 ? displayName[0] : "?");
</script>

{#if safe}
	<img src={safe} alt="" class={imgClass} loading="lazy" width={dimension} height={dimension} />
{:else}
	<div class={placeholderClass}>
		<span class={textClass}>{initial}</span>
	</div>
{/if}
