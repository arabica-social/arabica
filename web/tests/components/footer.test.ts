import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import Footer from "../../src/lib/components/Footer.svelte";

describe("Footer", () => {
  afterEach(cleanup);

  it("keeps feedback available without a promotional panel", () => {
    render(Footer);
    const feedback = screen.getByRole("link", { name: "Feedback" });
    expect(feedback).toHaveAttribute(
      "href",
      "https://userinput.app/s/did:plc:chqc2ockzmyvlrasfb66x64a/3mrgh3b4f722p",
    );
    expect(feedback).toHaveAttribute("target", "_blank");
    expect(feedback).toHaveAttribute("rel", "noopener noreferrer");
    expect(screen.getByRole("link", { name: "About" })).toHaveAttribute(
      "href",
      "/about",
    );
    expect(screen.getByRole("link", { name: "AT Protocol" })).toHaveAttribute(
      "href",
      "/atproto",
    );
  });
});
