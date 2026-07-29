import { describe, expect, it } from "vitest";
import {
	formatAvgRating,
	formatRating,
	formatTemp,
	formatTempForUnit,
	formatTime,
	pluralS,
	pluralWord,
	safeWebsiteURL,
} from "../../src/lib/utils/format";

describe("pluralS", () => {
	it("returns empty string for 1", () => {
		expect(pluralS(1)).toBe("");
	});

	it.each([0, 2, 5, 100])("returns 's' for %i", (n) => {
		expect(pluralS(n)).toBe("s");
	});

	it("returns 's' for negative counts", () => {
		expect(pluralS(-1)).toBe("s");
	});
});

describe("pluralWord", () => {
	it("returns singular for 1", () => {
		expect(pluralWord("bean", "beans", 1)).toBe("bean");
	});

	it.each([0, 2, 3])("returns plural for %i", (n) => {
		expect(pluralWord("bean", "beans", n)).toBe("beans");
	});
});

describe("safeWebsiteURL", () => {
	it("returns empty for empty input", () => {
		expect(safeWebsiteURL("")).toBe("");
	});

	it.each([
		"http://example.com",
		"https://example.com",
		"https://roasters.example.com/path",
		"http://example.com:8080/path?q=1",
		"https://sub.domain.example.com",
	])("allows %s", (url) => {
		expect(safeWebsiteURL(url)).toBe(url);
	});

	it.each([
		["javascript:alert(1)", "disallows javascript scheme"],
		["ftp://example.com", "disallows ftp scheme"],
		["data:text/html,<x>", "disallows data scheme"],
		["file:///etc/passwd", "disallows file scheme"],
		["//example.com", "disallows scheme-relative URLs"],
		["example.com", "disallows bare host without scheme"],
		["not a url at all", "disallows non-URL text"],
		["http://localhost", "disallows hostnames without a dot"],
		["https://localhost", "disallows hostnames without a dot"],
	])("returns empty for %s (%s)", (url) => {
		expect(safeWebsiteURL(url)).toBe("");
	});

	it("is case-insensitive on the scheme", () => {
		expect(safeWebsiteURL("HTTPS://example.com")).toBe("HTTPS://example.com");
	});

	it("returns empty for null-ish input", () => {
		expect(safeWebsiteURL("" as string)).toBe("");
	});
});

describe("formatRating", () => {
	it("formats an integer rating as N/10", () => {
		expect(formatRating(7)).toBe("7/10");
	});

	it("formats zero as 0/10", () => {
		expect(formatRating(0)).toBe("0/10");
	});

	it("formats ten as 10/10", () => {
		expect(formatRating(10)).toBe("10/10");
	});
});

describe("formatAvgRating", () => {
	it("returns empty string for zero", () => {
		expect(formatAvgRating(0)).toBe("");
	});

	it("formats a non-zero average with one decimal", () => {
		expect(formatAvgRating(8.5)).toBe("8.5/10");
	});

	it("pads to one decimal place", () => {
		expect(formatAvgRating(7)).toBe("7.0/10");
	});

	it("rounds to one decimal", () => {
		expect(formatAvgRating(8.46)).toBe("8.5/10");
	});
});

describe("formatTime", () => {
	it("returns N/A for zero", () => {
		expect(formatTime(0)).toBe("N/A");
	});

	it("formats sub-minute seconds", () => {
		expect(formatTime(45)).toBe("45s");
	});

	it("formats exactly one second", () => {
		expect(formatTime(1)).toBe("1s");
	});

	it("formats whole minutes", () => {
		expect(formatTime(120)).toBe("2m");
	});

	it("formats minutes with remaining seconds", () => {
		expect(formatTime(150)).toBe("2m 30s");
	});

	it("formats exactly one minute", () => {
		expect(formatTime(60)).toBe("1m");
	});
});

describe("formatTemp", () => {
	it("returns N/A for zero", () => {
		expect(formatTemp(0)).toBe("N/A");
	});

	it("detects Celsius for temps <= 100", () => {
		expect(formatTemp(93)).toBe("93.0°C");
	});

	it("detects Fahrenheit for temps > 100", () => {
		expect(formatTemp(205)).toBe("205.0°F");
	});

	it("uses exactly 100 as Celsius (boundary)", () => {
		expect(formatTemp(100)).toBe("100.0°C");
	});

	it("formats with one decimal", () => {
		expect(formatTemp(92.5)).toBe("92.5°C");
	});
});

describe("formatTempForUnit", () => {
	it("returns N/A for zero", () => {
		expect(formatTempForUnit(0, "C")).toBe("N/A");
		expect(formatTempForUnit(0, "F")).toBe("N/A");
		expect(formatTempForUnit(0, "recorded")).toBe("N/A");
		expect(formatTempForUnit(0, "")).toBe("N/A");
	});

	describe("recorded preference (default)", () => {
		it("uses recorded unit when preference is empty", () => {
			expect(formatTempForUnit(93, "")).toBe("93.0°C");
			expect(formatTempForUnit(205, "")).toBe("205.0°F");
		});

		it("uses recorded unit when preference is 'recorded'", () => {
			expect(formatTempForUnit(93, "recorded")).toBe("93.0°C");
			expect(formatTempForUnit(205, "recorded")).toBe("205.0°F");
		});
	});

	describe("no-op when preference matches recorded unit", () => {
		it("keeps Celsius when recorded C and preferred C", () => {
			expect(formatTempForUnit(93, "C")).toBe("93.0°C");
		});

		it("keeps Fahrenheit when recorded F and preferred F", () => {
			expect(formatTempForUnit(205, "F")).toBe("205.0°F");
		});
	});

	describe("C→F conversion", () => {
		it("converts a recorded Celsius value to Fahrenheit", () => {
			expect(formatTempForUnit(93, "F")).toBe("199.4°F");
		});

		it("converts 0°C-ish boundary (recorded as C since <=100)", () => {
			expect(formatTempForUnit(100, "F")).toBe("212.0°F");
		});

		it("converts 0°C to 32°F", () => {
			expect(formatTempForUnit(1, "F")).toBe("33.8°F");
		});
	});

	describe("F→C conversion", () => {
		it("converts a recorded Fahrenheit value to Celsius", () => {
			expect(formatTempForUnit(205, "C")).toBe("96.1°C");
		});

		it("converts 212°F to 100°C", () => {
			expect(formatTempForUnit(212, "C")).toBe("100.0°C");
		});

		it("treats a recorded-Fahrenheit value exactly at the F→C conversion boundary", () => {
			// 32 is <= 100, so it is recorded as Celsius; preferred C matches → no-op.
			expect(formatTempForUnit(32, "C")).toBe("32.0°C");
		});
	});

	describe("case-insensitive preference", () => {
		it("treats lowercase 'f' as Fahrenheit", () => {
			expect(formatTempForUnit(93, "f")).toBe("199.4°F");
		});

		it("treats lowercase 'c' as Celsius", () => {
			expect(formatTempForUnit(205, "c")).toBe("96.1°C");
		});

		it("does NOT treat capitalized 'Recorded' as recorded (falls through to conversion)", () => {
			expect(formatTempForUnit(93, "Recorded")).toBe("199.4°F");
		});
	});

	describe("unrecognized preference strings", () => {
		it("converts C to F for an unrecognized preference when recorded as C", () => {
			expect(formatTempForUnit(93, "kelvin")).toBe("199.4°F");
		});

		it("converts F to F (mis-applies C→F formula) for an unrecognized preference when recorded as F", () => {
			expect(formatTempForUnit(205, "kelvin")).toBe("401.0°F");
		});
	});
});
