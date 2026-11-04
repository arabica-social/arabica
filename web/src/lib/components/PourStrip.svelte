<script lang="ts">
	import { formatTime } from "../utils/format";
	import type { Pour } from "../types/entity_view";

	type Props = { pours: Pour[] | undefined };
	let { pours }: Props = $props();
</script>

{#if pours && pours.length > 0}
	<div class="pour-strip">
		<span class="text-label pour-strip-label">Pours</span>
		<div class="pour-strip-grid">
			{#each pours as pour, i (pour.pour_number)}
				<span class="pour-cell">
					<span class="pour-num">{i + 1}</span>
					<span class="pour-water">{pour.water_amount}g</span>
					{#if pour.time_seconds > 0}
						<span class="pour-time">{formatTime(pour.time_seconds)}</span>
					{/if}
				</span>
			{/each}
		</div>
	</div>
{/if}
