Pulled three real specimens from the codebase — simple, middling, gross — and
rewrote each in petite-vue. Single shared `<script>` at the bottom.

## 1. Simple: header dropdown (`header.templ:73`)

**Alpine today:**

```html
<div x-data="{ open: false }" class="relative">
  <button @click="open = !open" @click.outside="open = false">Menu</button>
  <div x-show="open" x-cloak x-transition:enter="...">…items…</div>
</div>
```

**petite-vue equivalent:**

```html
<div v-scope="Disclosure()" class="relative">
  <button @click="toggle" @click.outside="close">Menu</button>
  <div v-show="open" v-cloak>…items…</div>
</div>
```

```js
// in arabica.js
function Disclosure() {
  return {
    open: false,
    toggle() {
      this.open = !this.open;
    },
    close() {
      this.open = false;
    },
  };
}
```

Verdict: ~lateral. Alpine's `{ open: false }` inline is honestly fine here; the
petite-vue version is only nicer because it names the pattern. Reusable factory
pays off if you have 10 disclosures (you do).

The bigger win is no `x-transition:enter="transition ease-out duration-150 ..."`
string soup — petite-vue has no transition directive, so you'd move that to a
CSS class toggle, which is cleaner anyway.

## 2. Middling: share button (`action_bar.templ:267`)

**Alpine today** — the offender is the `fmt.Sprintf` injecting URL/title/text
into a giant `x-data` string:

```go
x-data={ fmt.Sprintf("{ copied: false, share() { const fullUrl = window.location.origin + '%s'; if (navigator.share) { navigator.share({ title: '%s', text: '%s', url: fullUrl })... } else { navigator.clipboard.writeText(fullUrl).then(() => { this.copied = true; setTimeout(() => this.copied = false, 2000); }); } } }", escapeForJS(props.ShareURL), escapeForJS(props.ShareTitle), escapeForJS(props.ShareText)) }
@click="share()"
```

**petite-vue:**

```go
<button
  type="button"
  v-scope={ fmt.Sprintf(`ShareButton(%q, %q, %q)`, props.ShareURL, props.ShareTitle, props.ShareText) }
  @click="share"
  class="action-btn"
>
  <svg v-show="!copied">…</svg>
  <svg v-show="copied" v-cloak>…</svg>
  <span v-show="copied" v-cloak>Copied!</span>
</button>
```

```js
function ShareButton(url, title, text) {
  return {
    copied: false,
    async share() {
      const fullUrl = window.location.origin + url;
      if (navigator.share) {
        try {
          await navigator.share({ title, text, url: fullUrl });
        } catch {}
      } else {
        await navigator.clipboard.writeText(fullUrl);
        this.copied = true;
        setTimeout(() => (this.copied = false), 2000);
      }
    },
  };
}
```

Verdict: **clear win**. The whole 500-char `Sprintf` collapses to a 3-arg
function call; logic lives in a real JS file where you can lint it, the templ
only has to escape strings for JS literals (or you JSON-marshal the args).
`escapeForJS` largely goes away.

## 3. Gross: report modal (`action_bar.templ:291`)

**Alpine today** has a multi-line `@submit.prevent={ fmt.Sprintf(` ` ...`
30-line backtick string of inline JS, plus six pieces of state in
`x-data="{ reason: '', charCount: 0, submitting: false, error: '', success: false }"`,
plus `x-text`, `x-bind:disabled`, two `<template x-if>` branches.

**petite-vue:**

```html
<dialog id={ props.ID } class="modal-dialog" v-scope={ fmt.Sprintf("ReportModal(%q, %q, %q)", props.ID, props.SubjectURI, props.SubjectCID) }>
  <div class="modal-content">
    <h3 class="modal-title">Report Content</h3>

    <form v-if="!success" @submit.prevent="submit" class="space-y-4">
      <p class="text-sm text-emphasis">Please describe why you're reporting…</p>
      <div>
        <textarea v-model="reason" rows="4" maxlength="500" class="w-full form-textarea"></textarea>
        <div class="flex justify-between text-xs text-faint mt-1">
          <span>Optional, but helpful for moderators</span>
          <span>{{ reason.length }}/500</span>
        </div>
      </div>
      <div v-if="error" class="bg-red-100 ..." >{{ error }}</div>
      <div class="flex gap-2">
        <button type="submit" :disabled="submitting" class="flex-1 btn-primary">
          {{ submitting ? 'Submitting...' : 'Submit Report' }}
        </button>
        <button type="button" @click="close" :disabled="submitting" class="flex-1 btn-secondary">Cancel</button>
      </div>
    </form>

    <div v-if="success" class="text-center py-4">
      <div class="text-green-600 mb-2"><svg>…</svg></div>
      <p class="font-medium text-primary">Report Submitted</p>
      <p class="text-sm text-muted mt-1">Thank you for helping keep our community safe.</p>
    </div>
  </div>
</dialog>
```

```js
function ReportModal(id, subjectURI, subjectCID) {
  return {
    reason: "",
    submitting: false,
    error: "",
    success: false,
    async submit() {
      this.submitting = true;
      this.error = "";
      try {
        const res = await fetch("/api/report", {
          method: "POST",
          headers: { "Content-Type": "application/x-www-form-urlencoded" },
          body: new URLSearchParams({
            subject_uri: subjectURI,
            subject_cid: subjectCID,
            reason: this.reason,
          }),
        });
        const data = await res.json();
        if (res.ok) {
          this.success = true;
          setTimeout(() => document.getElementById(id).close(), 2000);
        } else {
          this.error = data.message || "Failed to submit report";
        }
      } catch {
        this.error = "Network error. Please try again.";
      } finally {
        this.submitting = false;
      }
    },
    close() {
      document.getElementById(id).close();
    },
  };
}
```

Verdict: **big win**. The 30-line inline `fmt.Sprintf` fetch becomes a normal
`async submit()` you can read, debug, and breakpoint. `charCount` state vanishes
because `{{ reason.length }}` is just a real expression. `<template x-if>`
becomes `v-if` directly on the element (no wrapper). The templ side is markup,
period.

---

## What this means for arabica

- **Replace `escapeForJS` + giant `Sprintf` x-data strings → factory functions
  taking real args.** That alone removes a class of bugs (quote escaping,
  multi-line backtick formatting drifting from Go indentation).
- **Logic moves into `arabica.js` (or split files).** Lintable, testable,
  debuggable. Templ becomes pure markup with declarative bindings.
- **htmx interop**: petite-vue's `createApp(...).mount()` scans once. For htmx
  swaps you'd call `createApp(...).mount(swappedRoot)` on `htmx:afterSwap`, or
  use `v-scope` re-init helpers — basically the same pattern you'd need with
  Alpine's `Alpine.initTree()`.
- **What you lose vs Alpine**: built-in transitions (use CSS), the
  `$dispatch`/`$el`/`$watch` magicals (petite-vue has fewer; you'd use plain
  `dispatchEvent`/refs/`$watch`-as-method).
- **What you'd skip**: `combo-select.js` is already a hand-written ~28kb
  component module — petite-vue wouldn't change much there. The win is
  concentrated in the dozens of inline-string Alpine blocks across
  action_bar/header/buttons/comments/feed.

Honest call: for arabica specifically, petite-vue would clean up ~80% of what
bugs you about Alpine with low migration cost (Alpine and petite-vue can
co-exist on the same page during migration). Preact+htm would clean up 100% but
at higher cost and worse htmx ergonomics on swapped fragments.
