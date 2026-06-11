import { cleanup, render } from "@testing-library/svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import ModalContainerIsland from "./ModalContainerIsland.svelte";

describe("ModalContainerIsland", () => {
  afterEach(() => {
    cleanup();
    document.body.innerHTML = "";
    vi.useRealTimers();
  });

  it("does not call showModal for a dialog removed before the deferred open", async () => {
    vi.useFakeTimers();
    const target = document.createElement("div");
    target.id = "modal-container";
    target.innerHTML = `<dialog id="entity-modal"></dialog>`;
    document.body.appendChild(target);
    const dialog = target.querySelector<HTMLDialogElement>("dialog")!;
    dialog.showModal = vi.fn();

    render(ModalContainerIsland, { props: { target } });
    document.body.dispatchEvent(
      new CustomEvent("htmx:afterSwap", {
        detail: { target },
      }),
    );
    dialog.remove();

    await vi.runAllTimersAsync();

    expect(dialog.showModal).not.toHaveBeenCalled();
  });
});
