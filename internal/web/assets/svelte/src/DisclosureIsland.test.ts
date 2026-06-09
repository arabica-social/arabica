import { cleanup, render } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it } from "vitest";
import DisclosureIsland from "./DisclosureIsland.svelte";

function disclosureTarget() {
  const target = document.createElement("section");
  target.innerHTML = `
    <button data-disclosure-button aria-expanded="true">Toggle</button>
    <div data-disclosure-menu class="menu is-open">Menu</div>
    <span data-disclosure-rotate class="chevron rotate-180"></span>
    <button data-disclosure-close data-disclosure-open="true">Close</button>
  `;
  document.body.appendChild(target);
  return target;
}

describe("DisclosureIsland", () => {
  afterEach(() => {
    cleanup();
    document.body.innerHTML = "";
  });

  it("attaches behavior without replacing server-rendered disclosure markup", async () => {
    const user = userEvent.setup();
    const target = disclosureTarget();
    const originalButton = target.querySelector("[data-disclosure-button]");
    const originalMenu = target.querySelector("[data-disclosure-menu]");

    render(DisclosureIsland, { props: { target } });

    expect(target.querySelector("[data-disclosure-button]")).toBe(
      originalButton,
    );
    expect(target.querySelector("[data-disclosure-menu]")).toBe(originalMenu);
    expect(originalButton).toHaveAttribute("aria-expanded", "false");
    expect(originalMenu).not.toHaveClass("is-open");

    await user.click(originalButton as HTMLElement);

    expect(originalButton).toHaveAttribute("aria-expanded", "true");
    expect(originalMenu).toHaveClass("is-open");
    expect(target.querySelector("[data-disclosure-rotate]")).toHaveClass(
      "rotate-180",
    );
  });

  it("closes on close triggers, outside clicks, and Escape", async () => {
    const user = userEvent.setup();
    const target = disclosureTarget();
    render(DisclosureIsland, { props: { target } });

    const button = target.querySelector<HTMLElement>(
      "[data-disclosure-button]",
    )!;
    const close = target.querySelector<HTMLElement>("[data-disclosure-close]")!;
    const menu = target.querySelector<HTMLElement>("[data-disclosure-menu]")!;

    await user.click(button);
    expect(menu).toHaveClass("is-open");

    await user.click(close);
    expect(menu).not.toHaveClass("is-open");
    expect(close.dataset.disclosureOpen).toBe("false");

    await user.click(button);
    expect(menu).toHaveClass("is-open");
    await user.click(document.body);
    expect(menu).not.toHaveClass("is-open");

    await user.click(button);
    expect(menu).toHaveClass("is-open");
    await user.keyboard("{Escape}");
    expect(menu).not.toHaveClass("is-open");
  });
});
