import { cleanup, fireEvent, render, screen } from "@testing-library/svelte";
import { afterEach, expect, it, vi } from "vitest";
import DraftRating from "./DraftRating.svelte";
afterEach(cleanup);
it("starts blank and editable without an Add rating button", () => {
  const onchange = vi.fn();
  render(DraftRating, { onchange });
  expect(screen.getByLabelText("Rating (1–10)")).toHaveValue(null);
  expect(screen.getByLabelText("Rating (1–10)")).not.toBeRequired();
  expect(onchange).not.toHaveBeenCalled();
  expect(screen.queryByRole("button")).toBeNull();
});
it("emits numeric edits and explicit clearing, then restores the canonical value", async () => {
  const onchange = vi.fn();
  const { rerender } = render(DraftRating, { value: 8, onchange });
  const rating = screen.getByLabelText("Rating (1–10)");
  await fireEvent.input(rating, { target: { value: "9" } });
  expect(onchange).toHaveBeenLastCalledWith(9);
  expect(rating).toHaveValue(8);
  await rerender({ value: 9, onchange });
  expect(rating).toHaveValue(9);
  await fireEvent.input(rating, { target: { value: "" } });
  expect(onchange).toHaveBeenLastCalledWith(null);
  await rerender({ value: null, onchange });
  expect(rating).toHaveValue(null);
});
it("locks the visible rating during Save", () => {
  render(DraftRating, { value: 7, disabled: true, onchange: vi.fn() });
  expect(screen.getByLabelText("Rating (1–10)")).toHaveValue(7);
  expect(screen.getByLabelText("Rating (1–10)")).toBeDisabled();
});
