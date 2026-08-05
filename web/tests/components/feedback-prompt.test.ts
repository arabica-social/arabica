import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import FeedbackPrompt from "../../src/lib/components/FeedbackPrompt.svelte";

describe("FeedbackPrompt", () => {
  afterEach(() => {
    cleanup();
  });

  it("links people to the feedback board in a new tab", () => {
    render(FeedbackPrompt);

    expect(
      screen.getByRole("heading", { name: "Tell us what needs work." }),
    ).toBeTruthy();
    const link = screen.getByRole("link", { name: /share feedback/i });
    expect(link).toHaveAttribute(
      "href",
      "https://userinput.app/s/did:plc:chqc2ockzmyvlrasfb66x64a/3mrgh3b4f722p",
    );
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("accepts copy and destination overrides for reuse", () => {
    render(FeedbackPrompt, {
      title: "A different note",
      actionLabel: "Write to us",
      href: "https://example.com/contact",
    });

    expect(
      screen.getByRole("heading", { name: "A different note" }),
    ).toBeTruthy();
    expect(screen.getByRole("link", { name: /write to us/i })).toHaveAttribute(
      "href",
      "https://example.com/contact",
    );
  });
});
