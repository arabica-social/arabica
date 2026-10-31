import { cleanup, fireEvent, render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vitest";
import DraftEditor from "../../src/lib/components/ai/DraftEditor.svelte";
import type { DraftPatch } from "../../src/lib/ai/types";

const library = {
  beans: [{ rkey: "fix", name: "October Fix" }],
  brewers: [{ rkey: "orea", name: "Orea", brewer_type: "pourover" }],
  grinders: [{ rkey: "k6", name: "K6" }],
};
afterEach(cleanup);
function editor(fields: DraftPatch = {}, brewerType = "pourover") {
  const onchange = vi.fn();
  const props = {
    draft: { kind: "brew" as const, fields },
    brewerType,
    library,
    onchange,
  };
  return { ...render(DraftEditor, props), onchange, props };
}
it("starts an empty brew in Quick with the essentials and optional results only", () => {
  const { onchange } = editor();
  expect(
    screen.getByRole("button", { name: "Quick", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  expect(screen.getByLabelText("Bean *")).toBeVisible();
  expect(screen.getByLabelText("Brewer *")).toBeVisible();
  expect(screen.getByLabelText("Recipe")).toBeVisible();
  expect(screen.getByLabelText("Tasting notes")).toHaveAttribute(
    "placeholder",
    "Describe the flavors, aroma, and your thoughts...",
  );
  expect(screen.queryByLabelText("Coffee (g)")).not.toBeInTheDocument();
  expect(screen.queryByText("More brewing details")).not.toBeInTheDocument();
  expect(screen.queryByLabelText("Rating (1–10)")).not.toBeInTheDocument();
  expect(onchange).not.toHaveBeenCalled();
});
it("does not render Method even for a legacy populated draft and does not erase it", async () => {
  const fields = { method: "Legacy V60" };
  const { onchange } = editor(fields);
  await userEvent.click(
    screen.getByRole("button", { name: "Detailed", exact: true }),
  );
  expect(screen.queryByLabelText(/method/i)).not.toBeInTheDocument();
  expect(screen.queryByDisplayValue("Legacy V60")).not.toBeInTheDocument();
  expect(fields.method).toBe("Legacy V60");
  expect(onchange).not.toHaveBeenCalled();
});
it("reveals Detailed with ghost examples rather than inserting defaults", async () => {
  const { onchange } = editor();
  await userEvent.click(
    screen.getByRole("button", { name: "Detailed", exact: true }),
  );
  expect(screen.getByLabelText("Coffee (g)")).toHaveAttribute(
    "placeholder",
    "e.g. 18",
  );
  expect(screen.getByLabelText("Coffee (g)")).toHaveValue(null);
  expect(screen.getByLabelText("Water (g)")).toHaveAttribute(
    "placeholder",
    "e.g. 250",
  );
  expect(screen.getByLabelText("Water (g)")).toHaveValue(null);
  expect(screen.getByLabelText("Brew time (s)")).toBeVisible();
  expect(screen.getByLabelText("Grinder")).toBeVisible();
  expect(screen.queryByLabelText("Grind setting")).not.toBeInTheDocument();
  expect(screen.getByLabelText("Temperature (°F/°C)")).not.toBeVisible();
  await userEvent.click(screen.getByText("More brewing details"));
  expect(screen.getByLabelText("Temperature (°F/°C)")).toBeVisible();
  expect(screen.getByLabelText("Temperature (°F/°C)")).toHaveAttribute(
    "placeholder",
    "e.g. 93.5",
  );
  expect(screen.getByLabelText("Filter")).toHaveAttribute(
    "placeholder",
    "e.g. paper, metal, cloth",
  );
  expect(onchange).not.toHaveBeenCalled();
});
it("uses the existing espresso primary layout and leaves the empty water field out", async () => {
  editor({}, "espresso");
  await userEvent.click(
    screen.getByRole("button", { name: "Detailed", exact: true }),
  );
  expect(screen.getByLabelText("Espresso yield (g)")).toBeVisible();
  expect(screen.getByLabelText("Shot time (s)")).toBeVisible();
  expect(screen.queryByLabelText("Water (g)")).not.toBeInTheDocument();
  expect(screen.queryByLabelText("Bloom water (g)")).not.toBeInTheDocument();
  expect(screen.getByLabelText("Pressure (bar)")).not.toBeVisible();
});
it("preserves existing espresso water in automatically expanded advanced details", () => {
  editor({ water_amount: 100 }, "espresso");
  expect(screen.getByLabelText("Water (g)")).toHaveValue(100);
  expect(screen.getByLabelText("Water (g)")).toBeVisible();
});
it("automatically reveals populated preparation and advanced details for review", () => {
  editor({
    coffee_amount: 18,
    temperature: 93,
    "pourover_params.filter": "paper",
    pours: [{ water_amount: 50, time_seconds: 0 }],
  });
  expect(
    screen.getByRole("button", { name: "Detailed", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  expect(screen.getByLabelText("Coffee (g)")).toHaveValue(18);
  expect(screen.getByLabelText("Temperature (°F/°C)")).toBeVisible();
  expect(screen.getByLabelText("Filter")).toBeVisible();
  expect(screen.getByLabelText("Pour 1 water (g)")).toBeVisible();
});
it("keeps values in a Quick summary and restores the same controls without edits", async () => {
  const { onchange } = editor({
    coffee_amount: 18,
    water_amount: 300,
    temperature: 93,
    grind_size: "24 clicks",
  });
  await userEvent.click(
    screen.getByRole("button", { name: "Quick", exact: true }),
  );
  expect(screen.queryByLabelText("Coffee (g)")).not.toBeInTheDocument();
  expect(screen.getByLabelText("Brewing details added")).toHaveTextContent(
    "18g coffee · 300g water",
  );
  expect(screen.getByLabelText("Brewing details added")).toHaveTextContent(
    "24 clicks",
  );
  await userEvent.click(
    screen.getByRole("button", { name: "Edit brewing details", exact: true }),
  );
  expect(screen.getByLabelText("Coffee (g)")).toHaveValue(18);
  expect(screen.getByLabelText("Grind setting")).toHaveValue("24 clicks");
  expect(onchange).not.toHaveBeenCalled();
});
it("keeps an explicitly chosen Quick view when new preparation arrives and summarizes it", async () => {
  const view = editor();
  await userEvent.click(
    screen.getByRole("button", { name: "Quick", exact: true }),
  );
  await view.rerender({
    ...view.props,
    draft: { kind: "brew", fields: { coffee_amount: 21, temperature: 94 } },
  });
  expect(
    screen.getByRole("button", { name: "Quick", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  expect(screen.getByLabelText("Brewing details added")).toHaveTextContent(
    "21g coffee",
  );
  expect(screen.getByLabelText("Brewing details added")).toHaveTextContent(
    "94°",
  );
  expect(view.onchange).not.toHaveBeenCalled();
});
it("automatically expands a previously empty draft when AI preparation arrives", async () => {
  const view = editor();
  await view.rerender({
    ...view.props,
    draft: { kind: "brew", fields: { coffee_amount: 19, temperature: 94 } },
  });
  expect(screen.getByLabelText("Coffee (g)")).toBeVisible();
  expect(screen.getByLabelText("Temperature (°F/°C)")).toBeVisible();
  expect(view.onchange).not.toHaveBeenCalled();
});
it("does not collapse Detailed when the final preparation value is cleared", async () => {
  const view = editor({ coffee_amount: 18 });
  await view.rerender({ ...view.props, draft: { kind: "brew", fields: {} } });
  expect(
    screen.getByRole("button", { name: "Detailed", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  expect(screen.getByLabelText("Coffee (g)")).toBeVisible();
});
it("reveals an empty rating on demand without inventing a rating", async () => {
  const { onchange } = editor();
  await userEvent.click(
    screen.getByRole("button", { name: "Add rating", exact: true }),
  );
  expect(screen.getByLabelText("Rating (1–10)")).toBeVisible();
  expect(screen.getByLabelText("Rating (1–10)")).toHaveValue(null);
  expect(screen.getByLabelText("Rating (1–10)")).toHaveFocus();
  expect(onchange).not.toHaveBeenCalled();
  await fireEvent.input(screen.getByLabelText("Rating (1–10)"), {
    target: { value: "8" },
  });
  expect(onchange).toHaveBeenLastCalledWith({ rating: 8 }, undefined);
});
it("shows an existing rating without replacing its value", () => {
  const { onchange } = editor({ rating: 7 });
  expect(screen.getByLabelText("Rating (1–10)")).toHaveValue(7);
  expect(onchange).not.toHaveBeenCalled();
});
it("freezes disclosure buttons with the rest of the form during Save", () => {
  const onchange = vi.fn();
  render(DraftEditor, {
    draft: { kind: "brew", fields: {} },
    onchange,
    disabled: true,
  });
  expect(
    screen.getByRole("button", { name: "Detailed", exact: true }),
  ).toBeDisabled();
  expect(
    screen.getByRole("button", { name: "Add rating", exact: true }),
  ).toBeDisabled();
});
it("does not add brew mode controls to other entity editors", () => {
  render(DraftEditor, {
    draft: { kind: "bean", fields: { name: "Fix" } },
    onchange: vi.fn(),
  });
  expect(screen.getByLabelText("Name *")).toHaveValue("Fix");
  expect(
    screen.queryByRole("group", { name: "Brew logging mode" }),
  ).not.toBeInTheDocument();
});

it.each(["immersion", "coldbrew", "cupping", "mokapot"])(
  "uses the existing %s disclosure policy",
  async (category) => {
    editor({}, category);
    await userEvent.click(
      screen.getByRole("button", { name: "Detailed", exact: true }),
    );
    expect(screen.getByLabelText("Coffee (g)")).toBeVisible();
    expect(screen.getByLabelText("Water (g)")).toBeVisible();
    expect(screen.getByLabelText("Grinder")).not.toBeVisible();
    const time = screen.getByLabelText(
      category === "mokapot" ? "Brew time (s)" : "Steep time (s)",
    );
    if (category === "mokapot") expect(time).not.toBeVisible();
    else expect(time).toBeVisible();
  },
);
