import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it } from "vitest";
import ManageCollectionsIsland from "./ManageCollectionsIsland.svelte";

function setupManageDOM() {
  const root = document.createElement("section");
  root.dataset.svelteManageTabs = "";
  root.dataset.activeTab = "beans";
  root.innerHTML = `
    <div data-svelte-manage-collections data-content-selector="#manage-content"></div>
    <div id="manage-content">
      <div data-tab-panel="beans">
        <div data-manage-section>
          <div data-manage-collection>
            <article data-manage-card data-manage-name="Zebra" data-manage-search="zebra kenya" data-manage-created="3">Zebra</article>
            <article data-manage-card data-manage-name="Alpha" data-manage-search="alpha colombia" data-manage-created="1">Alpha</article>
            <article data-manage-card data-manage-name="Bravo" data-manage-search="bravo ethiopia" data-manage-created="2">Bravo</article>
          </div>
        </div>
      </div>
      <div data-tab-panel="roasters">
        <div data-manage-collection>
          <article data-manage-card data-manage-name="Hidden Roaster" data-manage-search="hidden" data-manage-created="4">Hidden Roaster</article>
        </div>
      </div>
    </div>
  `;
  document.body.appendChild(root);
  const target = root.querySelector<HTMLElement>(
    "[data-svelte-manage-collections]",
  )!;
  return { root, target };
}

function visibleCards() {
  return Array.from(
    document.querySelectorAll<HTMLElement>(
      '#manage-content [data-tab-panel="beans"] [data-manage-card]:not([hidden])',
    ),
  ).map((card) => card.dataset.manageName);
}

describe("ManageCollectionsIsland", () => {
  afterEach(() => {
    cleanup();
    document.body.innerHTML = "";
  });

  it("filters the active manage collection without replacing cards", async () => {
    const user = userEvent.setup();
    const { target } = setupManageDOM();
    const original = document.querySelector('[data-manage-name="Alpha"]');

    render(ManageCollectionsIsland, { props: { target } });
    await user.type(screen.getByRole("searchbox"), "colombia");

    expect(document.querySelector('[data-manage-name="Alpha"]')).toBe(original);
    expect(visibleCards()).toEqual(["Alpha"]);
    expect(
      screen.getByText("Showing 1 of 3 records in this tab."),
    ).toBeTruthy();
  });

  it("sorts active collection by name and date metadata", async () => {
    const user = userEvent.setup();
    const { target } = setupManageDOM();

    render(ManageCollectionsIsland, { props: { target } });
    expect(visibleCards()).toEqual(["Zebra", "Bravo", "Alpha"]);

    await user.selectOptions(
      screen.getByLabelText("Sort current manage collection"),
      "name",
    );
    await waitFor(() =>
      expect(visibleCards()).toEqual(["Alpha", "Bravo", "Zebra"]),
    );

    await user.selectOptions(
      screen.getByLabelText("Sort current manage collection"),
      "oldest",
    );
    await waitFor(() =>
      expect(visibleCards()).toEqual(["Alpha", "Bravo", "Zebra"]),
    );
  });
});
