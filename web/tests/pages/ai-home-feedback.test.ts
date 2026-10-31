import { cleanup, fireEvent, render, screen } from "@testing-library/svelte";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { goto } from "$app/navigation";
import HomeEntry from "../../src/lib/components/ai/HomeEntry.svelte";
import WorkingIndicator from "../../src/lib/components/ai/WorkingIndicator.svelte";
import { entryKey } from "../../src/lib/api/ai";

vi.mock("$app/navigation", () => ({ goto: vi.fn() }));
const did = "did:plc:alice";
function deferred() {
  let resolve!: () => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<void>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}
async function enter(text = "  My V60 brew  ") {
  const input = screen.getByRole("textbox", {
    name: "Describe a brew or bean",
  }) as HTMLTextAreaElement;
  expect(input).toHaveAttribute("placeholder", "Describe a brew or bean...");
  await fireEvent.input(input, { target: { value: text } });
  await fireEvent.submit(input.form!);
  return input;
}
beforeEach(() => {
  vi.resetAllMocks();
  sessionStorage.clear();
});
afterEach(cleanup);

describe("Home chat feedback", () => {
  it("shows immediate truthful navigation status and suppresses duplicate submission", async () => {
    const nav = deferred();
    vi.mocked(goto).mockReturnValue(nav.promise);
    render(HomeEntry, { accountDid: did });
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    const input = await enter();
    expect(screen.getByRole("status")).toHaveTextContent("Opening chat…");
    expect(screen.getByRole("status")).toHaveAttribute("aria-live", "polite");
    expect(input.form).toHaveAttribute("aria-busy", "true");
    expect(input).toBeDisabled();
    expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();
    await fireEvent.submit(input.form!);
    expect(goto).toHaveBeenCalledExactlyOnceWith("/chat");
    expect(sessionStorage.getItem(entryKey(did))).toBe("My V60 brew");
    nav.resolve();
    await nav.promise;
  });

  it("preserves the exact input and restores controls when navigation fails", async () => {
    const nav = deferred();
    vi.mocked(goto).mockReturnValue(nav.promise);
    render(HomeEntry, { accountDid: did });
    const input = await enter();
    nav.reject(new Error("navigation failed"));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Your message is still here.",
    );
    expect(input).toHaveValue("  My V60 brew  ");
    expect(input).not.toBeDisabled();
    expect(input.form).toHaveAttribute("aria-busy", "false");
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    vi.mocked(goto).mockResolvedValue();
    await fireEvent.submit(input.form!);
    expect(goto).toHaveBeenCalledTimes(2);
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("keeps handoffs scoped and ignores an old account's late failure", async () => {
    const old = deferred();
    const current = deferred();
    vi.mocked(goto)
      .mockReturnValueOnce(old.promise)
      .mockReturnValueOnce(current.promise);
    const view = render(HomeEntry, { accountDid: did });
    await enter("Alice's brew");
    const otherDid = "did:plc:bob";
    await view.rerender({ accountDid: otherDid });
    const input = screen.getByLabelText("Describe a brew or bean");
    expect(input).toHaveValue("");
    expect(input).not.toBeDisabled();
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    await enter("Bob's bean");
    old.reject(new Error("old failure"));
    await old.promise.catch(() => {});
    expect(input).toHaveValue("Bob's bean");
    expect(input).toBeDisabled();
    expect(screen.getByRole("status")).toHaveTextContent("Opening chat…");
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(sessionStorage.getItem(entryKey(did))).toBe("Alice's brew");
    expect(sessionStorage.getItem(entryKey(otherDid))).toBe("Bob's bean");
    current.resolve();
    await current.promise;
  });

  it("reports storage failure without navigating or losing the input", async () => {
    const spy = vi
      .spyOn(Storage.prototype, "setItem")
      .mockImplementation(() => {
        throw new Error("denied");
      });
    render(HomeEntry, { accountDid: did });
    const input = await enter();
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(goto).not.toHaveBeenCalled();
    expect(input).toHaveValue("  My V60 brew  ");
    expect(input).not.toBeDisabled();
    spy.mockRestore();
  });

  it("provides one atomic live status with a decorative spinner", () => {
    render(WorkingIndicator);
    const status = screen.getByRole("status");
    expect(status).toHaveTextContent("Working on your draft…");
    expect(status).toHaveAttribute("aria-atomic", "true");
    expect(status.querySelector(".spinner")).toHaveAttribute(
      "aria-hidden",
      "true",
    );
  });
});
