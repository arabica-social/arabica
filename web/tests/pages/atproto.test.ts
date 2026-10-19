import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import ATProto from "../../src/routes/atproto/+page.svelte";

describe("AT Protocol page", () => {
  afterEach(() => {
    cleanup();
    document.body.innerHTML = "";
  });

  it("renders the page heading", () => {
    render(ATProto);
    expect(screen.getByText("The AT Protocol")).toBeTruthy();
  });

  it("explains account storage, public records, and references", () => {
    render(ATProto);
    expect(
      screen.getByRole("heading", { name: "Your account and server" }),
    ).toBeTruthy();
    expect(
      screen.getByText(/AT Protocol repository records are public/),
    ).toBeTruthy();
    expect(screen.getByText(/Arabica never sees your password/)).toBeTruthy();
    expect(screen.getByText(/at:\/\/did:plc:xxx/)).toBeTruthy();
  });

  it("links to the official atproto.com site", () => {
    render(ATProto);
    const link = screen.getByText("atproto.com").closest("a");
    expect(link).toHaveAttribute("href", "https://atproto.com");
  });
});
