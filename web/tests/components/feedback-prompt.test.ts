import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import FeedbackPrompt from "../../src/lib/components/FeedbackPrompt.svelte";

describe("FeedbackPrompt", () => {
  afterEach(() => {
    cleanup();
  });

  it("links people to the feedback page", () => {
    render(FeedbackPrompt);

    expect(
      screen.getByRole("heading", { name: "Help shape Arabica." }),
    ).toBeTruthy();
    expect(
      screen.getByRole("link", { name: /share feedback/i }),
    ).toHaveAttribute("href", "/feedback");
  });

  it("accepts copy and destination overrides for reuse", () => {
    render(FeedbackPrompt, {
      title: "A different note",
      actionLabel: "Write to us",
      href: "/contact",
    });

    expect(
      screen.getByRole("heading", { name: "A different note" }),
    ).toBeTruthy();
    expect(screen.getByRole("link", { name: /write to us/i })).toHaveAttribute(
      "href",
      "/contact",
    );
  });
});
