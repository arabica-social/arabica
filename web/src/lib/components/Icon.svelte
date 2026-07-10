<script lang="ts" module>
	// Icon path data, ported from internal/web/components/icons.templ.
	// Each entry is the inner SVG markup (paths/circles/etc.) — the
	// wrapper <svg> is rendered by Icon.svelte with the shared class.
	export const iconPaths = {
		coffee:
			'<path d="M17 8h1a4 4 0 1 1 0 8h-1"></path><path d="M3 8h14v9a4 4 0 0 1-4 4H7a4 4 0 0 1-4-4Z"></path><line x1="6" y1="2" x2="6" y2="4"></line><line x1="10" y1="2" x2="10" y2="4"></line><line x1="14" y1="2" x2="14" y2="4"></line>',
		leaf: '<path d="M7 20h10"></path><path d="M10 20c5.5-2.5.8-6.4 3-10"></path><path d="M9.5 9.4c1.1.8 1.8 2.2 2.3 3.7-2 .4-3.5.4-4.8-.3-1.2-.6-2.3-1.9-3-4.2 2.8-.5 4.4 0 5.5.8z"></path><path d="M14.1 6a7 7 0 0 0-1.1 4c1.9-.1 3.3-.6 4.3-1.4 1-1 1.6-2.3 1.7-4.6-2.7.1-4 1-4.9 2z"></path>',
		store: '<path d="m2 7 4.41-4.41A2 2 0 0 1 7.83 2h8.34a2 2 0 0 1 1.42.59L22 7"></path><path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"></path><path d="M15 22v-4a2 2 0 0 0-2-2h-2a2 2 0 0 0-2 2v4"></path><path d="M2 7h20"></path><path d="M22 7v3a2 2 0 0 1-2 2v0a2.7 2.7 0 0 1-1.59-.63.7.7 0 0 0-.82 0A2.7 2.7 0 0 1 16 12a2.7 2.7 0 0 1-1.59-.63.7.7 0 0 0-.82 0A2.7 2.7 0 0 1 12 12a2.7 2.7 0 0 1-1.59-.63.7.7 0 0 0-.82 0A2.7 2.7 0 0 1 8 12a2.7 2.7 0 0 1-1.59-.63.7.7 0 0 0-.82 0A2.7 2.7 0 0 1 4 12v0a2 2 0 0 1-2-2V7"></path>',
		disc: '<circle cx="12" cy="12" r="10"></circle><circle cx="12" cy="12" r="2"></circle>',
		brewer: '<path d="M6 4h12l-3 16H9L6 4z"></path><path d="M4 4h16"></path>',
		fileText:
			'<path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"></path><polyline points="14 2 14 8 20 8"></polyline><line x1="16" y1="13" x2="8" y2="13"></line><line x1="16" y1="17" x2="8" y2="17"></line><line x1="10" y1="9" x2="8" y2="9"></line>',
		droplet:
			'<path d="M12 22a7 7 0 0 0 7-7c0-2-1-3.9-3-5.5s-3.5-4-4-6.5c-.5 2.5-2 4.9-4 6.5C6 11.1 5 13 5 15a7 7 0 0 0 7 7z"></path>',
		bell: '<path stroke-linecap="round" stroke-linejoin="round" d="M14.857 17.082a23.848 23.848 0 0 0 5.454-1.31A8.967 8.967 0 0 1 18 9.75V9A6 6 0 0 0 6 9v.75a8.967 8.967 0 0 1-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 0 1-5.714 0m5.714 0a3 3 0 1 1-5.714 0"></path>',
		plus: '<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15"></path>',
		chevronDown:
			'<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>',
		shieldCheck:
			'<path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z"></path>',
		x: '<line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line>',
		arrowLeft:
			'<path stroke-linecap="round" stroke-linejoin="round" d="M10.5 19.5 3 12m0 0 7.5-7.5M3 12h18"></path>',
		bean: '<ellipse cx="12" cy="12" rx="5.5" ry="9" transform="rotate(25 12 12)"></ellipse><path d="M12 3c-1.5 3 1.5 6 0 9s1.5 6 0 9" transform="rotate(25 12 12)"></path>',
		gear: '<circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>',
		mapPin: '<path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z"></path><circle cx="12" cy="10" r="3"></circle>',
		teapot: '<circle cx="12" cy="5" r="0.75"></circle><path d="M7 9c1-2 3-3 5-3s4 1 5 3"></path><path d="M6 10h12"></path><path d="M6 10c0 6 2 10 6 10s6-4 6-10"></path><path d="M6 12c-2 0-3-1-3-3"></path><path d="M18 12c2 0 3 1 3 3s-1 3-3 3"></path>',
		scale: '<path d="m16 16 3-8 3 8c-.87.65-1.92 1-3 1s-2.13-.35-3-1Z"></path><path d="m2 16 3-8 3 8c-.87.65-1.92 1-3 1s-2.13-.35-3-1Z"></path><path d="M7 21h10"></path><path d="M12 3v18"></path><path d="M3 7h2c2 0 5-1 7-2 2 1 5 2 7 2h2"></path>',
		thermometer: '<path d="M14 4v10.54a4 4 0 1 1-4 0V4a2 2 0 0 1 4 0Z"></path>',
		clock: '<circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline>',
		flame: '<path d="M8.5 14.5A2.5 2.5 0 0 0 11 12c0-1.38-.5-2-1-3-1.072-2.143-.224-4.054 2-6 .5 2.5 2 4.9 4 6.5 2 1.6 3 3.5 3 5.5a7 7 0 1 1-14 0c0-1.153.433-2.294 1-3a2.5 2.5 0 0 0 2.5 2.5z"></path>',
		sliders: '<line x1="4" y1="21" x2="4" y2="14"></line><line x1="4" y1="10" x2="4" y2="3"></line><line x1="12" y1="21" x2="12" y2="12"></line><line x1="12" y1="8" x2="12" y2="3"></line><line x1="20" y1="21" x2="20" y2="16"></line><line x1="20" y1="12" x2="20" y2="3"></line><line x1="1" y1="14" x2="7" y2="14"></line><line x1="9" y1="8" x2="15" y2="8"></line><line x1="17" y1="16" x2="23" y2="16"></line>',
		sprout: '<path d="M8.5 2h7"></path><path d="M14 2v6a2 2 0 0 0 .245.96l5.51 10.08A2 2 0 0 1 18 22H6a2 2 0 0 1-1.755-2.96l5.51-10.08A2 2 0 0 0 10 8V2"></path><path d="M6.453 15h11.094"></path>',
		star: '<polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon>',
		link: '<path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path>',
		globe: '<circle cx="12" cy="12" r="10"></circle><path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20"></path><path d="M2 12h20"></path>',
		tag: '<path d="M12.586 2.586A2 2 0 0 0 11.172 2H4a2 2 0 0 0-2 2v7.172a2 2 0 0 0 .586 1.414l8.704 8.704a2.426 2.426 0 0 0 3.42 0l6.58-6.58a2.426 2.426 0 0 0 0-3.42z"></path><circle cx="7.5" cy="7.5" r=".5" fill="currentColor"></circle>',
	} as const;

	export type IconName = keyof typeof iconPaths;

	// Per-icon defaults ported from internal/web/components/icons.templ.
	// Each legacy icon baked in a semantic color class so it read correctly
	// wherever it was dropped; the SPA Icon applies `color` only when the
	// caller hasn't passed its own `text-*` color utility (otherwise both
	// would be present and CSS source order would decide the winner).
	// `fill` is "currentColor" for icons that should render solid (star);
	// the default "none" renders a stroked outline, matching legacy.
	const iconDefaults: Partial<
		Record<IconName, { color?: string; fill?: string }>
	> = {
		coffee: { color: "text-muted" },
		leaf: { color: "text-green-600" },
		store: { color: "text-muted" },
		disc: { color: "text-placeholder" },
		brewer: { color: "text-faint" },
		fileText: { color: "text-placeholder" },
		droplet: { color: "text-blue-400" },
		bell: {},
		plus: {},
		chevronDown: {},
		shieldCheck: {},
		x: {},
		arrowLeft: {},
		bean: { color: "text-muted" },
		gear: { color: "text-placeholder" },
		mapPin: { color: "text-red-400" },
		teapot: { color: "text-muted" },
		scale: { color: "text-faint" },
		thermometer: { color: "text-red-400" },
		clock: { color: "text-placeholder" },
		flame: { color: "text-orange-400" },
		sliders: { color: "text-placeholder" },
		sprout: { color: "text-faint" },
		star: { color: "text-amber-500", fill: "currentColor" },
		link: { color: "text-blue-400" },
		globe: { color: "text-blue-400" },
		tag: { color: "text-placeholder" },
	};
</script>

<script lang="ts">
	let {
		name,
		class: className = "inline-block w-4 h-4 flex-shrink-0",
	}: { name: IconName; class?: string } = $props();

	let paths = $derived(iconPaths[name] ?? "");
	let defaults = $derived(iconDefaults[name] ?? {});

	// Apply the default color only when the caller hasn't supplied a `text-*`
	// color utility of its own. The regex matches the palette and semantic
	// color classes used in this app, but not font-size/alignment utilities
	// like `text-sm` or `text-center`.
	let hasExplicitColor = $derived(
		/\btext-(?:amber|red|green|blue|orange|purple|gray|white|muted|faint|placeholder|primary|secondary|emphasis|danger)[a-z0-9-]*\b/.test(
			className,
		),
	);
	let resolvedClass = $derived(
		!hasExplicitColor && defaults.color
			? `${className} ${defaults.color}`
			: className,
	);
	let resolvedFill = $derived(defaults.fill ?? "none");
</script>

<svg class={resolvedClass} viewBox="0 0 24 24" fill={resolvedFill} stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
	<!-- eslint-disable-next-line svelte/no-at-html-tags -->
	{@html paths}
</svg>
