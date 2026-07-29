import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { get } from "svelte/store";
import {
	clearToasts,
	dismissToast,
	extractNotifyMessage,
	pushToast,
	toasts,
} from "../../src/lib/stores/toasts";

describe("toasts store", () => {
	beforeEach(() => {
		vi.useFakeTimers();
		clearToasts();
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	describe("pushToast", () => {
		it("adds a toast to the store", () => {
			pushToast("Saved!");
			expect(get(toasts)).toHaveLength(1);
			expect(get(toasts)[0].message).toBe("Saved!");
			expect(get(toasts)[0].id).toBeTypeOf("number");
		});

		it("appends to existing toasts and increments ids", () => {
			pushToast("first");
			pushToast("second");
			const list = get(toasts);
			expect(list).toHaveLength(2);
			expect(list[0].message).toBe("first");
			expect(list[1].message).toBe("second");
			expect(list[1].id).toBeGreaterThan(list[0].id);
		});

		it("is a no-op for an empty message", () => {
			pushToast("");
			expect(get(toasts)).toHaveLength(0);
		});

		it("auto-dismisses after the default duration (2900ms)", () => {
			pushToast("hello");
			expect(get(toasts)).toHaveLength(1);

			vi.advanceTimersByTime(2899);
			expect(get(toasts)).toHaveLength(1);

			vi.advanceTimersByTime(1);
			expect(get(toasts)).toHaveLength(0);
		});

		it("respects a custom duration", () => {
			pushToast("custom", 1000);
			expect(get(toasts)).toHaveLength(1);

			vi.advanceTimersByTime(999);
			expect(get(toasts)).toHaveLength(1);

			vi.advanceTimersByTime(1);
			expect(get(toasts)).toHaveLength(0);
		});
	});

	describe("dismissToast", () => {
		it("removes the toast with the given id", () => {
			pushToast("a");
			pushToast("b");
			const id = get(toasts)[0].id;

			dismissToast(id);

			expect(get(toasts)).toHaveLength(1);
			expect(get(toasts)[0].message).toBe("b");
		});

		it("is a no-op for an unknown id", () => {
			pushToast("a");
			dismissToast(99999);
			expect(get(toasts)).toHaveLength(1);
		});

		it("cancels the auto-dismiss timer so a manually dismissed toast stays gone", () => {
			pushToast("temp");
			const id = get(toasts)[0].id;
			dismissToast(id);

			vi.advanceTimersByTime(5000);
			expect(get(toasts)).toHaveLength(0);
		});
	});

	describe("clearToasts", () => {
		it("empties the store", () => {
			pushToast("a");
			pushToast("b");
			pushToast("c");
			expect(get(toasts)).toHaveLength(3);

			clearToasts();

			expect(get(toasts)).toHaveLength(0);
		});

		it("cancels pending auto-dismiss timers", () => {
			pushToast("a");
			pushToast("b");
			clearToasts();

			vi.advanceTimersByTime(10000);
			expect(get(toasts)).toHaveLength(0);
		});

		it("is safe to call on an empty store", () => {
			clearToasts();
			expect(get(toasts)).toHaveLength(0);
		});
	});

	describe("extractNotifyMessage", () => {
		it("returns the string when detail is a string", () => {
			expect(extractNotifyMessage("Saved!")).toBe("Saved!");
		});

		it("extracts message from {value: {message: 'x'}}", () => {
			expect(extractNotifyMessage({ value: { message: "Liked!" } })).toBe("Liked!");
		});

		it("extracts message from {value: 'x'}", () => {
			expect(extractNotifyMessage({ value: "Done" })).toBe("Done");
		});

		it("extracts message from {message: 'x'}", () => {
			expect(extractNotifyMessage({ message: "Created" })).toBe("Created");
		});

		it("returns empty for null", () => {
			expect(extractNotifyMessage(null)).toBe("");
		});

		it("returns empty for undefined", () => {
			expect(extractNotifyMessage(undefined)).toBe("");
		});

		it("prefers the nested value.message over a top-level message", () => {
			expect(
				extractNotifyMessage({ value: { message: "nested" }, message: "top-level" }),
			).toBe("nested");
		});

		it("falls back to top-level message when value is present but not a string/object", () => {
			expect(extractNotifyMessage({ value: 42, message: "fallback" })).toBe("fallback");
		});

		it.each([
			["number", 42],
			["boolean", true],
			["array", [1, 2, 3]],
			["object without value/message", { foo: "bar" }],
			["object with non-string value.message", { value: { message: 123 } }],
			["object with non-string value", { value: 123 }],
			["object with non-string message", { message: 456 }],
		])("returns empty for non-matching shape: %s", (_label, detail) => {
			expect(extractNotifyMessage(detail)).toBe("");
		});
	});
});
