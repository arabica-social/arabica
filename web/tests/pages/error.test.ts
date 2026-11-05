import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it, vi } from "vitest";

const { page } = vi.hoisted(() => ({
  page: { status: 404, error: { message: "Not found" } },
}));
vi.mock("$app/state", () => ({ page }));
import ErrorPage from "../../src/routes/+error.svelte";

afterEach(cleanup);

describe("app error page", () => {
  it("renders the designed missing-page card and home action", () => {
    page.status = 404;
    page.error = { message: "Not found" };
    render(ErrorPage);
    const heading = screen.getByRole("heading", {
      name: "Page Not Found",
      level: 1,
    });
    expect(heading.closest(".card")).not.toBeNull();
    expect(screen.getByText("404")).toHaveClass("text-6xl");
    expect(
      screen.getByText("The page you're looking for could not be found."),
    ).toBeVisible();
    expect(screen.getByRole("link", { name: "Back to Home" })).toHaveAttribute(
      "href",
      "/",
    );
  });

  it("does not reveal access-denial details in the missing-page body", () => {
    page.status = 404;
    page.error = { message: "AI access is not permitted" };
    render(ErrorPage);
    expect(screen.queryByText("AI access is not permitted")).toBeNull();
    expect(
      screen.getByRole("heading", { name: "Page Not Found" }),
    ).toBeVisible();
  });

  it("preserves non-404 error status and message", () => {
    page.status = 500;
    page.error = { message: "Something went wrong" };
    render(ErrorPage);
    expect(screen.getByRole("heading", { name: "500" })).toBeVisible();
    expect(screen.getByText("Something went wrong")).toBeVisible();
    expect(
      screen.queryByRole("heading", { name: "Page Not Found" }),
    ).toBeNull();
  });
});
