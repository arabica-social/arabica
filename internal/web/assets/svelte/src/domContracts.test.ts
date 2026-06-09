import { afterEach, describe, expect, it, vi } from "vitest";
import {
  extractFragment,
  fetchHTMXPartial,
  formToURLSearchParams,
  postURLEncodedForm,
  responseTextOrThrow,
  showSessionExpiredOn401,
} from "./domContracts";

describe("domContracts", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("serializes only string form fields as URLSearchParams", () => {
    const form = document.createElement("form");
    form.innerHTML = `
      <input name="name" value="Colombia" />
      <input name="name" value="Ethiopia" />
      <input name="empty" value="" />
      <input type="file" name="photo" />
    `;

    const params = formToURLSearchParams(form);

    expect(params.getAll("name")).toEqual(["Colombia", "Ethiopia"]);
    expect(params.get("empty")).toBe("");
    expect(params.has("photo")).toBe(false);
  });

  it("posts forms using Go handler friendly URL encoding", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(new Response("ok"));
    const form = document.createElement("form");
    form.method = "post";
    form.action = "/api/beans";
    form.innerHTML = `<input name="name" value="Test Bean" />`;

    await postURLEncodedForm(form);

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/api/beans",
      expect.objectContaining({
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: expect.any(URLSearchParams),
      }),
    );
    const options = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect((options.body as URLSearchParams).toString()).toBe("name=Test+Bean");
  });

  it("sets the HTMX header when fetching protected partials", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(new Response("<div></div>"));

    await fetchHTMXPartial("/manage/partial?tab=beans");

    expect(fetchMock).toHaveBeenCalledWith("/manage/partial?tab=beans", {
      method: "GET",
      credentials: "same-origin",
      headers: { "HX-Request": "true" },
    });
  });

  it("extracts server fragments by selector", () => {
    const fragment = extractFragment(
      `<main><section id="target">Updated</section></main>`,
      "#target",
    );

    expect(fragment).toBeInstanceOf(HTMLElement);
    expect(fragment?.textContent).toBe("Updated");
  });

  it("returns response text for successful responses and throws useful errors", async () => {
    await expect(
      responseTextOrThrow(new Response("hello", { status: 200 }), "fallback"),
    ).resolves.toBe("hello");

    await expect(
      responseTextOrThrow(new Response("", { status: 500 }), "fallback"),
    ).rejects.toThrow("fallback");
  });

  it("opens the session expired modal only for 401 responses", () => {
    const showModal = vi.fn();
    window.__showSessionExpiredModal = showModal;

    showSessionExpiredOn401(new Response("", { status: 403 }));
    expect(showModal).not.toHaveBeenCalled();

    showSessionExpiredOn401(new Response("", { status: 401 }));
    expect(showModal).toHaveBeenCalledOnce();

    delete window.__showSessionExpiredModal;
  });
});
