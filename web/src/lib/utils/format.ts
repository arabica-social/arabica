// Shared formatting helpers for entity view pages. Ported from
// internal/web/components/shared.templ and internal/arabica/web/pages/*.templ.

/** Returns "s" for non-1 counts, "" for 1. */
export function pluralS(n: number): string {
	return n === 1 ? "" : "s";
}

/** Returns the singular or plural word based on count. */
export function pluralWord(singular: string, plural: string, n: number): string {
	return n === 1 ? singular : plural;
}

/**
 * Sanitizes website URLs client-side, mirroring bff.SafeWebsiteURL on the
 * server. Only allows HTTP/HTTPS URLs with a hostname containing a dot.
 */
export function safeWebsiteURL(url: string): string {
	if (!url) return "";
	let parsed: URL;
	try {
		parsed = new URL(url);
	} catch {
		return "";
	}
	const scheme = parsed.protocol.toLowerCase();
	if (scheme !== "http:" && scheme !== "https:") return "";
	if (!parsed.hostname.includes(".")) return "";
	return url;
}

/** Formats a rating integer as "N/10". */
export function formatRating(rating: number): string {
	return `${rating}/10`;
}

/** Formats an average rating as "X.X/10", or "" if zero. */
export function formatAvgRating(avg: number): string {
	if (avg === 0) return "";
	return `${avg.toFixed(1)}/10`;
}

/** Formats a duration in seconds as "45s", "2m", or "2m 30s". Mirrors bff.FormatTime. */
export function formatTime(seconds: number): string {
	if (seconds === 0) return "N/A";
	if (seconds < 60) return `${seconds}s`;
	const minutes = Math.floor(seconds / 60);
	const remaining = seconds % 60;
	if (remaining === 0) return `${minutes}m`;
	return `${minutes}m ${remaining}s`;
}

/** Formats a temperature with unit detection (>100 = F, else C). Mirrors bff.FormatTemp. */
export function formatTemp(temp: number): string {
	if (temp === 0) return "N/A";
	const unit = temp > 100 ? "F" : "C";
	return `${temp.toFixed(1)}°${unit}`;
}
