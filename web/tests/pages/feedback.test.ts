import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import FeedbackPage from "../../src/routes/feedback/+page.svelte";

describe("Feedback page", () => {
  afterEach(() => {
    cleanup();
    document.body.innerHTML = "";
  });

  it("collects a required title plus optional contact and details", () => {
    render(FeedbackPage);

    expect(
      screen.getByRole("heading", { name: "Submit feedback" }),
    ).toBeTruthy();
    expect(screen.getByLabelText(/^Title/)).toBeRequired();
    expect(screen.getByLabelText("Contact (optional)")).toHaveAttribute(
      "placeholder",
      "you@example.com or @you.bsky.social",
    );
    expect(screen.getByLabelText("Description")).toBeTruthy();
  });

  it("opens a pre-addressed plain-text email form", () => {
    render(FeedbackPage);

    const form = screen
      .getByRole("button", { name: /open feedback email/i })
      .closest("form");
    expect(form).toHaveAttribute(
      "action",
      "mailto:mail@arabica.systems?subject=Arabica%20feedback",
    );
    expect(form).toHaveAttribute("method", "post");
    expect(form).toHaveAttribute("enctype", "text/plain");
  });
});
