import { cleanup, render, waitFor } from "@testing-library/svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import ModalShellIsland from "./ModalShellIsland.svelte";

function setupModal() {
  document.body.innerHTML = `
    <div id="manage-content" hx-get="/api/manage"></div>
    <dialog id="entity-modal">
      <form data-svelte-modal-shell hx-put="/api/beans/abc">
        <div data-modal-shell-error hidden></div>
      </form>
    </dialog>
  `;
  const dialog = document.querySelector<HTMLDialogElement>("dialog")!;
  const form = document.querySelector<HTMLFormElement>("form")!;
  const calls: string[] = [];
  dialog.close = vi.fn(() => {
    calls.push("close");
    dialog.dispatchEvent(new Event("close"));
  });
  window.htmx = {
    trigger: vi.fn(() => {
      calls.push("refreshManage");
    }),
  } as typeof window.htmx;
  return { dialog, form, calls };
}

function setupDrawer() {
  document.body.innerHTML = `
    <section id="get-started" hx-get="/api/get-started-card"></section>
    <div class="station-drawer-row">
      <div class="station-drawer" data-drawer data-kind="vendor">
        <form data-svelte-modal-shell hx-post="/api/vendors">
          <div data-modal-shell-error hidden></div>
        </form>
      </div>
    </div>
  `;
  const form = document.querySelector<HTMLFormElement>("form")!;
  const calls: string[] = [];
  window.htmx = {
    trigger: vi.fn(() => {
      calls.push("refreshManage");
    }),
  } as typeof window.htmx;
  return { form, calls };
}

describe("ModalShellIsland", () => {
  afterEach(() => {
    cleanup();
    document.body.innerHTML = "";
    delete window.htmx;
  });

  it("closes the edit modal before triggering manage refresh", async () => {
    const { dialog, form, calls } = setupModal();
    render(ModalShellIsland, { props: { target: form } });

    form.dispatchEvent(
      new CustomEvent("htmx:afterRequest", {
        bubbles: true,
        detail: { successful: true, xhr: { responseText: "{}" } },
      }),
    );

    await waitFor(() => expect(window.htmx?.trigger).toHaveBeenCalled());
    expect(dialog.close).toHaveBeenCalled();
    expect(document.querySelector("dialog#entity-modal")).toBeNull();
    expect(calls).toEqual(["close", "refreshManage"]);
  });

  it("removes onboarding drawers and refreshes the get-started card after save", async () => {
    const { form, calls } = setupDrawer();
    render(ModalShellIsland, { props: { target: form } });

    form.dispatchEvent(
      new CustomEvent("htmx:afterRequest", {
        bubbles: true,
        detail: { successful: true, xhr: { responseText: "{}" } },
      }),
    );

    await waitFor(() => expect(window.htmx?.trigger).toHaveBeenCalled());
    expect(document.querySelector("[data-drawer]")).toBeNull();
    expect(calls).toEqual(["refreshManage"]);
  });
});
