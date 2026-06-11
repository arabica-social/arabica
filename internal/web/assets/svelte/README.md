# Svelte Island Boundaries

Arabica uses Templ for server-rendered HTML and Svelte for small islands. Each
island must follow one ownership model.

## Render Islands

Render islands own the DOM inside their mount point.

- Templ renders an empty mount element plus `data-*` props.
- `main.ts` clears the mount element with `target.innerHTML = ""`.
- Svelte mounts into that target and may render, replace, or remove children.

Use this for widgets whose markup is Svelte-owned, such as combo selects and
complex form islands.

## Behavior Islands

Behavior islands do not own the target DOM.

- Templ renders the full HTML.
- Svelte mounts on `document.body`.
- The island receives the server-rendered element as a `target` prop.
- The island may attach listeners, toggle classes/attributes, and request server
  fragments.
- The island must not mount into or clear the server-rendered target.

Use this for action menus, share buttons, comment sections, reply toggles, modal
shells, and other progressive-enhancement behavior.

```ts
mount(CommentSectionIsland, {
  target: document.body,
  props: { target: commentSectionElement },
});
```

## Client/Server Contracts

Use `src/domContracts.ts` instead of ad hoc `fetch` calls when crossing from an
island to Go handlers.

- Handlers that call `r.ParseForm()` expect `application/x-www-form-urlencoded`.
  Use `postURLEncodedForm(form)`.
- HTMX partial endpoints protected by HTMX middleware require
  `HX-Request: true`. Use `fetchHTMXPartial(url)`.
- If a form field needs to be normalized before posting, build a
  `URLSearchParams` body and pass it to `postURLEncodedForm(form, body)`.
- Call `showSessionExpiredOn401(response)` for fetch-based authenticated actions
  so the login-expired modal works outside HTMX.

Avoid mixing HTMX-only route assumptions with plain `fetch` unless the island
sets the same request headers HTMX would send.
