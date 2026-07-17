<script lang="ts">
	import Icon, { type IconName } from "./Icon.svelte";

	let { recordType }: { recordType: string } = $props();

	const config: Record<string, { icon: IconName; label: string }> = {
		brew: { icon: "coffee", label: "Brew" },
		bean: { icon: "bean", label: "Bean" },
		recipe: { icon: "fileText", label: "Recipe" },
		roaster: { icon: "store", label: "Roaster" },
		grinder: { icon: "gear", label: "Grinder" },
		brewer: { icon: "brewer", label: "Brewer" },
		cafe: { icon: "mapPin", label: "Cafe" },
		drink: { icon: "coffee", label: "Drink" },
	};

	let entry = $derived(config[recordType] ?? { icon: "fileText" as IconName, label: recordType });
</script>

<span class={`type-badge type-badge-${recordType}`}>
	<Icon name={entry.icon} class="inline-block w-4 h-4 flex-shrink-0" />
	{entry.label}
</span>
