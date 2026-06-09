export function formToURLSearchParams(form: HTMLFormElement) {
  const formData = new FormData(form);
  const params = new URLSearchParams();
  formData.forEach((value, key) => {
    if (typeof value === "string") {
      params.append(key, value);
    }
  });
  return params;
}

export async function postURLEncodedForm(
  form: HTMLFormElement,
  body: URLSearchParams = formToURLSearchParams(form),
) {
  const method = (form.method || "POST").toUpperCase();
  return fetch(form.action, {
    method,
    credentials: "same-origin",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body,
  });
}

export async function fetchHTMXPartial(url: string) {
  return fetch(url, {
    method: "GET",
    credentials: "same-origin",
    headers: {
      "HX-Request": "true",
    },
  });
}

export async function responseTextOrThrow(
  response: Response,
  fallbackMessage: string,
) {
  const text = await response.text();
  if (!response.ok) {
    throw new Error(text.trim() || fallbackMessage);
  }
  return text;
}

export function extractFragment(html: string, selector: string) {
  const parser = new DOMParser();
  const doc = parser.parseFromString(html, "text/html");
  return doc.querySelector(selector);
}

export function showSessionExpiredOn401(response: Response) {
  if (response.status === 401) {
    window.__showSessionExpiredModal?.();
  }
}
