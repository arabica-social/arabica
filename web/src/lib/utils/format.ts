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

/** Formats a duration in seconds as "45s", "2m", or "2m 30s".. */
export function formatTime(seconds: number): string {
	if (seconds === 0) return "N/A";
	if (seconds < 60) return `${seconds}s`;
	const minutes = Math.floor(seconds / 60);
	const remaining = seconds % 60;
	if (remaining === 0) return `${minutes}m`;
	return `${minutes}m ${remaining}s`;
}

/**
 * Formats a temperature with unit detection (>100 = F, else C). Mirrors
 * bff.FormatTemp (no preference — uses the recorded unit).
 */
export function formatTemp(temp: number): string {
	if (temp === 0) return "N/A";
	const unit = temp > 100 ? "F" : "C";
	return `${temp.toFixed(1)}°${unit}`;
}

/**
 * Formats a temperature respecting the user's preferred unit. Mirrors
 * bff.FormatTempForUnit. When the preference is "recorded" (or empty), the
 * recorded unit is used as-is. Otherwise the value is converted between
 * Celsius and Fahrenheit to match the preference.
 */
export function formatTempForUnit(temp: number, preferred: string): string {
	if (temp === 0) return "N/A";
	const recordedUnit = temp > 100 ? "F" : "C";
	if (!preferred || preferred === "recorded") {
		return `${temp.toFixed(1)}°${recordedUnit}`;
	}
	let value = temp;
	let unit = recordedUnit;
	const prefUpper = preferred.charAt(0).toUpperCase() + preferred.slice(1);
	if (prefUpper !== unit) {
		if (prefUpper === "C") {
			value = (temp - 32) * 5 / 9;
			unit = "C";
		} else {
			value = temp * 9 / 5 + 32;
			unit = "F";
		}
	}
	return `${value.toFixed(1)}°${unit}`;
}
