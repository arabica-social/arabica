import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import HandleAutocompleteIsland from "./HandleAutocompleteIsland.svelte";

const ACTOR = {
  handle: "alice.bsky.social",
  displayName: "Alice",
  avatar: "",
};

function actorsResponse(actors: unknown[]) {
  return new Response(JSON.stringify({ actors }), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

// The island renders its suggestion list into the Svelte mount container, not
// into the `target` prop element. We pass real input/target elements so the
// component can bind listeners, then query the rendered output via screen.
function setup() {
  document.body.innerHTML = `
    <div class="relative" data-handle-autocomplete-root>
      <input type="text" id="handle" name="handle" autocomplete="off" />
      <div class="handle-dropdown" data-svelte-handle-autocomplete></div>
    </div>
  `;
  const root = document.querySelector<HTMLElement>("[data-handle-autocomplete-root]")!;
  const input = root.querySelector<HTMLInputElement>('input[name="handle"]')!;
  const target =
    root.querySelector<HTMLElement>("[data-svelte-handle-autocomplete]")!;
  return { root, input, target };
}

async function waitForSuggestion(handle: string) {
  return waitFor(() => {
    const el = screen.queryByText(`@${handle}`);
    if (!el) throw new Error("suggestion not rendered yet");
    return el.closest<HTMLButtonElement>("button")!;
  });
}

describe("HandleAutocompleteIsland", () => {
  afterEach(() => {
    cleanup();
    document.body.innerHTML = "";
    vi.restoreAllMocks();
  });

  it("does not re-search after selecting a suggestion", async () => {
    const user = userEvent.setup();
    const { input, target } = setup();

    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(actorsResponse([ACTOR]));

    render(HandleAutocompleteIsland, { props: { input, target } });

    // Type enough to trigger the debounced search.
    await user.type(input, "alic");
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));

    const option = await waitForSuggestion(ACTOR.handle);

    await user.click(option);

    // The input should hold the selected handle.
    expect(input).toHaveValue(ACTOR.handle);

    // The bug: selecting a suggestion used to dispatch an "input" event that
    // re-triggered the typeahead search, reopening the dropdown shortly after.
    // Wait past the debounce window and confirm no additional search happened.
    await new Promise((r) => setTimeout(r, 450));
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(screen.queryByText(`@${ACTOR.handle}`)).toBeNull();
  });

  it("searches again when the user edits the input after selecting", async () => {
    const user = userEvent.setup();
    const { input, target } = setup();

    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(actorsResponse([ACTOR]));

    render(HandleAutocompleteIsland, { props: { input, target } });

    await user.type(input, "alic");
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));

    const option = await waitForSuggestion(ACTOR.handle);
    await user.click(option);
    expect(input).toHaveValue(ACTOR.handle);

    // Genuine typing/deletion after selection must still trigger a search.
    await user.type(input, "x");
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    expect(fetchMock).toHaveBeenLastCalledWith(
      expect.stringContaining(
        `/api/search-actors?q=${encodeURIComponent("alice.bsky.socialx")}`,
      ),
      expect.anything(),
    );
  });
});
