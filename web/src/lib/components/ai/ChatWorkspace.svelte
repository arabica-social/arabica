<script lang="ts">
  import { onMount, tick, untrack } from "svelte";
  import { DraftEngine, createSession, isUsefulDraft } from "#lib/ai/draft.js";
  import {
    loadSession,
    saveSession,
    removeSession,
    listSessions,
    MAX_SESSIONS,
    sameSession,
    storageKey,
    withAccountLock,
    type SessionSummary,
  } from "#lib/ai/persistence.js";
  import { MAX_MESSAGES, MAX_USER_CHARS } from "#lib/ai/types.js";
  import type {
    ApplyResult,
    ChangeOptions,
    FieldPath,
    DraftKind,
    DraftPatch,
    DraftSession,
    RequestToken,
  } from "#lib/ai/types.js";
  import { chat, entryKey, type ChatReply } from "#lib/api/ai.js";
  import {
    createBrew,
    createBean,
    createGrinder,
    createBrewer,
    type BrewInput,
    type BeanInput,
    type GrinderInput,
    type BrewerInput,
  } from "#lib/api/entities.js";
  import { requestJSON } from "#lib/api/client.js";
  import { appCache } from "#lib/stores/appCache.js";
  import {
    pendingSave,
    trackSave,
    readAttempt,
    resolveAttempt,
    attemptKey,
    SavedRecordError,
  } from "./saves";
  import DraftEditor, { type Library } from "./DraftEditor.svelte";
  import SendButton from "./SendButton.svelte";
  import { autosizeTextarea } from "./autosizeTextarea";
  import DraftPreview from "./DraftPreview.svelte";
  import DraftRating from "./DraftRating.svelte";
  import WorkingIndicator from "./WorkingIndicator.svelte";
  import Icon from "#lib/components/Icon.svelte";
  let { accountDid, enabled }: { accountDid: string; enabled: boolean } =
    $props();
  let engine: DraftEngine | undefined;
  let snapshot = $state<DraftSession>();
  let library = $state<Library>({});
  let libraryError = $state("");
  let error = $state("");
  let persistenceError = $state("");
  let notice = $state("");
  let input = $state("");
  let revealed = $state(false);
  let manualOpen = $state(false);
  let optionsOpen = $state(false);
  let saving = $state(false);
  let retryable = $state(false);
  let recovery = $state(false);
  let liveOnly = false;
  let uncertainSave = $state(false);
  let savedVersionURL = $state("");
  let choices = $state<
    (ChatReply["choices"][number] & { token: RequestToken })[]
  >([]);
  let controller: AbortController | undefined;
  let mounted = $state(false);
  let logElement = $state<HTMLElement>();
  let nearBottom = true;
  let histories = $state<SessionSummary[]>([]);
  let historyOpen = $state(false);
  const id = () => crypto.randomUUID();
  type Writer = {
    expected: DraftSession | null;
    valid: boolean;
    tail: Promise<boolean>;
  };
  let writer: Writer | undefined;
  function reconcile(): boolean {
    if (!engine || !writer || liveOnly) return true;
    const current = engine.snapshot();
    let loaded: ReturnType<typeof loadSession>;
    try {
      loaded = loadSession(localStorage, current.accountDid, current.id, {
        keepPending: true,
      });
    } catch {
      loaded = { ok: false, error: "unavailable" };
    }
    if (loaded.ok && sameSession(loaded.value, writer.expected)) return true;
    writer.valid = false;
    engine.stopRequest();
    controller?.abort();
    controller = undefined;
    choices = [];
    saving = false;
    if (loaded.ok && loaded.value) {
      engine = restoreSession(loaded.value);
      snapshot = engine.snapshot();
      revealed = !!snapshot.editorRevealed || isUsefulDraft(snapshot.draft);
      notice =
        "This chat changed in another tab. The latest stored version is shown.";
      persistenceError = "";
      const attempt = readAttempt(current.accountDid, current.id);
      uncertainSave = snapshot.status === "active" && !!attempt;
      savedVersionURL = attempt?.recordURL ?? "";
    } else {
      engine = undefined;
      snapshot = undefined;
      writer = undefined;
      persistenceError = loaded.ok
        ? "This chat was removed in another tab. Start a new chat to continue."
        : "Stored chats could not be verified. They have not been overwritten. Reload before continuing.";
      recovery = !loaded.ok;
    }
    refreshHistory();
    return false;
  }
  function sync(activate = false): Promise<boolean> {
    snapshot = engine
      ? { ...engine.snapshot(), composerText: input }
      : undefined;
    if (snapshot && (snapshot.editorRevealed || isUsefulDraft(snapshot.draft)))
      revealed = true;
    if (snapshot) {
      if (liveOnly) {
        persistenceError =
          "Storage is unavailable. This chat is live only; keep this tab open.";
        return Promise.resolve(true);
      }
      if (!navigator.locks?.request) {
        persistenceError =
          "This browser cannot safely coordinate saves across tabs. Your draft is live only; use a browser with Web Locks support to save.";
        return Promise.resolve(true);
      }
      const session = snapshot;
      const target = engine;
      const state = writer;
      if (!state) return Promise.resolve(false);
      state.tail = state.tail.then(async () => {
        if (!state.valid) return false;
        try {
          const result = await withAccountLock(session.accountDid, () => {
            if (!state.valid)
              return { ok: false as const, error: "conflict" as const };
            // Even a no-op must compare canonical state, but need not emit a storage event.
            const canonical = loadSession(
              localStorage,
              session.accountDid,
              session.id,
              { keepPending: true },
            );
            if (!canonical.ok) return canonical;
            if (!sameSession(canonical.value, state.expected))
              return { ok: false as const, error: "conflict" as const };
            if (!activate && sameSession(canonical.value, session))
              return { ok: true as const, value: undefined };
            return saveSession(localStorage, session.accountDid, session, {
              activate,
              expected: state.expected,
            });
          });
          if (result.ok) state.expected = session;
          if (engine === target && writer === state) {
            if (!result.ok && result.error === "conflict") reconcile();
            else
              persistenceError = result.ok
                ? ""
                : "Draft is live, but could not be saved on this device. Keep this tab open.";
            if (result.ok) refreshHistory();
          }
          return result.ok;
        } catch (e) {
          if (engine === target && writer === state)
            persistenceError =
              e instanceof Error
                ? e.message
                : "Draft is live, but storage is unavailable. Keep this tab open.";
          return false;
        }
      });
      return state.tail;
    }
    return Promise.resolve(true);
  }
  function stop() {
    if (engine?.snapshot().pending) {
      retryable = true;
      notice = "Stopped.";
    }
    engine?.stopRequest(); // Invalidate before aborting: even a completed fetch must be rejected.
    controller?.abort();
    controller = undefined;
    return sync();
  }
  async function loadLibrary(owner = accountDid) {
    libraryError = "";
    try {
      const data = await requestJSON<Library>(fetch, "/api/data");
      if (owner === accountDid) library = data;
    } catch {
      if (owner === accountDid)
        libraryError =
          "Could not load your library. Retry before selecting references.";
    }
  }
  $effect(() => {
    if (!mounted) return;
    const owner = accountDid;
    return untrack(() => {
      engine?.stopRequest();
      controller?.abort();
      engine = undefined;
      writer = undefined;
      snapshot = undefined;
      library = {};
      histories = [];
      historyOpen = false;
      choices = [];
      input = "";
      error = "";
      notice = "";
      persistenceError = "";
      revealed = false;
      manualOpen = false;
      optionsOpen = false;
      recovery = false;
      liveOnly = false;
      retryable = false;
      saving = false;
      uncertainSave = false;
      savedVersionURL = "";
      if (!owner) return;
      try {
        const loaded = loadSession(localStorage, owner, undefined, {
          keepPending: true,
        });
        if (!loaded.ok && loaded.error === "unavailable") {
          liveOnly = true;
          engine = new DraftEngine(createSession(owner, id(), "brew"));
          sync();
        } else if (!loaded.ok) {
          recovery = true;
          persistenceError =
            "Saved chat could not be loaded. It has not been overwritten. Discard it explicitly to start again.";
          return;
        } else {
          engine = restoreSession(
            loaded.value ?? createSession(owner, id(), "brew"),
            loaded.value,
          );
          refreshHistory();
          sync();
        }
      } catch {
        liveOnly = true;
        engine = new DraftEngine(createSession(owner, id(), "brew"));
        sync();
      }
      const pending = pendingSave(owner, engine.snapshot().id);
      if (pending) void observeSave(engine, pending);
      else if (!liveOnly && engine.snapshot().status === "active") {
        const attempt = readAttempt(owner, engine.snapshot().id);
        uncertainSave = !!attempt;
        savedVersionURL = attempt?.recordURL ?? "";
      }
      void loadLibrary(owner);
      void (async () => {
        if (enabled) {
          try {
            const queued = sessionStorage.getItem(entryKey(owner));
            if (queued) {
              // A home entry starts a new conversation, not an append to an old thread.
              if (
                engine.snapshot().messages.length ||
                Object.keys(engine.snapshot().draft.fields).length ||
                input
              ) {
                if (!(await fresh("brew", true))) return;
              }
              if (accountDid !== owner) return;
              input = queued;
              sessionStorage.removeItem(entryKey(owner));
              void send();
            }
          } catch {
            /* Live input remains available. */
          }
        }
      })();
      void scrollToLatest();
      return () => {
        engine?.stopRequest();
        sync();
        controller?.abort();
      };
    });
  });
  onMount(() => {
    mounted = true;
    const persistOnLeave = () => sync();
    const storageChanged = (event: StorageEvent) => {
      if (
        !engine ||
        (event.key !== null &&
          event.key !== storageKey(accountDid) &&
          event.key !== attemptKey(accountDid, engine.snapshot().id))
      )
        return;
      // Invalidate old engines before abort callbacks or late AI responses can run.
      if (reconcile() && engine?.snapshot().status === "active") {
        const attempt = readAttempt(accountDid, engine.snapshot().id);
        uncertainSave = !!attempt;
        savedVersionURL = attempt?.recordURL ?? "";
        if (attempt) {
          engine.stopRequest();
          controller?.abort();
          choices = [];
          snapshot = { ...engine.snapshot(), composerText: input };
        }
      }
      refreshHistory();
    };
    window.addEventListener("pagehide", persistOnLeave);
    window.addEventListener("storage", storageChanged);
    return () => {
      window.removeEventListener("pagehide", persistOnLeave);
      window.removeEventListener("storage", storageChanged);
      engine?.stopRequest();
      sync();
      controller?.abort();
      engine = undefined; // Detached save/request observers must not reselect this chat.
      mounted = false;
    };
  });
  function guarded(
    action: (options: ChangeOptions) => ApplyResult,
    options: ChangeOptions = {},
  ) {
    let result = action(options);
    if (result.status === "needs-confirmation") {
      if (
        window.confirm(
          `This change will replace or clear ${result.requiredClears.join(", ")}. Continue?`,
        )
      )
        result = action({ ...options, confirmRevision: result.revision });
      else return false;
    }
    notice = result.skipped.length
      ? `Kept newer edits: ${result.skipped.join(", ")}.`
      : result.status === "inactive"
        ? "That choice is no longer current."
        : "";
    if (result.error) error = result.error;
    sync();
    return result.status === "applied" || result.status === "conflict";
  }
  function edit(patch: DraftPatch, brewerType?: string) {
    if (!engine || saving) return;
    guarded((o) => engine!.edit(patch, o), { brewerType });
    // Restore controlled values after a declined destructive selection.
    snapshot = engine.snapshot();
  }
  async function send(retry = false) {
    const current = engine;
    if (
      !current ||
      !enabled ||
      saving ||
      current.snapshot().pending ||
      current.snapshot().status !== "active"
    )
      return;
    error = "";
    notice = "";
    if (!retry) {
      const text = input.trim();
      if (!text) return;
      if (current.snapshot().messages.length > MAX_MESSAGES - 2) {
        error =
          "Conversation limit reached. Save or explicitly discard this draft before starting again.";
        return;
      }
      const result = current.appendMessage({ role: "user", content: text });
      if (!result.ok) {
        error = result.error ?? "Message could not be added";
        return;
      }
      input = "";
    }
    const token = current.beginRequest(id());
    if (!token) {
      error = "Request limit reached. Your draft is still here.";
      sync();
      return;
    }
    retryable = false;
    choices = [];
    const abort = new AbortController();
    controller = abort;
    await sync(); // A quota failure must not disable live chat; a conflict replaces the engine.
    if (
      engine !== current ||
      current.snapshot().pending?.requestId !== token.requestId
    )
      return;
    if (nearBottom) void scrollToLatest();
    try {
      const s = current.snapshot();
      const response = await chat(
        fetch,
        {
          session_id: s.id,
          request_id: token.requestId,
          revision: token.revision,
          messages: s.messages,
          draft: s.draft,
        },
        abort.signal,
      );
      await writer?.tail;
      if (
        engine !== current ||
        !reconcile() ||
        current.snapshot().pending?.requestId !== token.requestId
      )
        return;
      if (
        response.kind !== current.snapshot().draft.kind &&
        !current.setKind(response.kind)
      )
        throw new Error("Start a new draft before changing record type.");
      const applied = guarded(
        (o) => current.applyResponse(token, response.patch, o),
        { brewerType: response.brewer_type },
      );
      if (!applied) {
        current.stopRequest();
        notice = "Draft change was not applied.";
      }
      const appended = current.appendMessage({
        role: "assistant",
        content: response.reply,
      });
      if (!appended.ok)
        error = "Conversation limit reached. Your draft is still here.";
      if (applied)
        choices = (response.choices ?? []).map((choice) => ({
          ...choice,
          token,
        }));
      sync();
      if (nearBottom) {
        await scrollToLatest();
      }
    } catch (e) {
      if (
        engine !== current ||
        current.snapshot().pending?.requestId !== token.requestId
      )
        return;
      current.stopRequest();
      error = e instanceof Error ? e.message : "Could not get a reply.";
      retryable = true;
      sync();
    } finally {
      if (controller === abort) controller = undefined;
    }
  }
  function choose(choice: (typeof choices)[number]) {
    if (!engine || saving) return;
    const before = engine.snapshot().draft.fields as DraftPatch;
    const changedReferences = (Object.keys(choice.patch) as FieldPath[]).filter(
      (key) => key.endsWith("_rkey") && before[key] !== choice.patch[key],
    );
    const accepted = guarded((o) =>
      engine!.applyChoice(
        {
          token: choice.token,
          patch: choice.patch,
          brewerType: choice.brewer_type,
        },
        o,
      ),
    );
    if (!accepted) return;
    const after = engine.snapshot().draft.fields as DraftPatch;
    if (
      Object.entries(choice.patch).every(([key, value]) =>
        value === null
          ? after[key as FieldPath] == null
          : JSON.stringify(after[key as FieldPath]) === JSON.stringify(value),
      )
    ) {
      choices = choices.filter(
        (candidate) =>
          candidate !== choice &&
          !changedReferences.some((key) => Object.hasOwn(candidate.patch, key)),
      );
    }
  }
  function refreshHistory() {
    if (liveOnly) return;
    const loaded = listSessions(localStorage, accountDid);
    if (loaded.ok) histories = loaded.value;
  }
  async function scrollToLatest() {
    await tick();
    const last = logElement?.querySelector<HTMLElement>(".message:last-child");
    if (logElement) {
      const top = last
        ? last.getBoundingClientRect().top -
          logElement.getBoundingClientRect().top +
          logElement.scrollTop
        : 0;
      logElement.scrollTo?.({ top });
    }
  }
  function restoreSession(
    session: DraftSession,
    expected: DraftSession | null = session,
  ): DraftEngine {
    writer = { expected, valid: true, tail: Promise.resolve(true) };
    input = session.composerText ?? "";
    retryable =
      session.status === "active" && session.messages.at(-1)?.role === "user";
    const restored = { ...session };
    delete restored.pending;
    return new DraftEngine(restored);
  }
  function activate(
    session: DraftSession,
    expected: DraftSession | null = session,
  ) {
    engine = restoreSession(session, expected);
    choices = [];
    error = "";
    notice = "";
    revealed = !!session.editorRevealed || isUsefulDraft(session.draft);
    manualOpen = false;
    optionsOpen = false;
    recovery = false;
    saving = false;
    uncertainSave = false;
    savedVersionURL = "";
    historyOpen = false;
    nearBottom = true;
    void sync(true);
    const pending = pendingSave(accountDid, session.id);
    if (pending) void observeSave(engine, pending);
    else if (!liveOnly && session.status === "active") {
      const attempt = readAttempt(accountDid, session.id);
      uncertainSave = !!attempt;
      savedVersionURL = attempt?.recordURL ?? "";
    }
    void scrollToLatest();
  }
  async function fresh(
    kind: DraftKind = "brew",
    allowPendingSave = false,
  ): Promise<boolean> {
    const owner = accountDid;
    const origin = engine;
    if ((saving && !allowPendingSave) || recovery) return false;
    if (
      snapshot?.status === "active" &&
      !snapshot.messages.length &&
      !Object.keys(snapshot.draft.fields).length &&
      !input &&
      !uncertainSave
    ) {
      engine?.setKind(kind);
      optionsOpen = false;
      historyOpen = false;
      sync();
      return true;
    }
    if (
      liveOnly &&
      !window.confirm(
        "Storage is unavailable. Starting a new chat will lose this live-only conversation and unsaved draft. Continue?",
      )
    )
      return false;
    if (!engine) persistenceError = "";
    await stop(); // Archive, do not discard the prior conversation or save marker.
    if (
      accountDid !== owner ||
      engine !== origin ||
      (persistenceError && !liveOnly)
    )
      return false;
    const next = createSession(owner, id(), kind);
    if (!liveOnly) {
      try {
        const result = await withAccountLock(owner, () =>
          saveSession(localStorage, owner, next, { expected: null }),
        );
        if (accountDid !== owner || engine !== origin) return false;
        if (!result.ok) {
          error =
            result.error === "limit"
              ? `Chat storage is full (up to ${MAX_SESSIONS} chats). Delete a chat to make room.`
              : "Could not start a new chat. Your current chat has been kept.";
          return false;
        }
      } catch {
        error = "Storage is unavailable. Your current chat has been kept.";
        return false;
      }
    }
    activate(next);
    return true;
  }
  async function switchChat(sessionId: string) {
    const owner = accountDid;
    const origin = engine;
    if (sessionId === snapshot?.id) {
      optionsOpen = false;
      historyOpen = false;
      return;
    }
    await stop();
    if (accountDid !== owner || engine !== origin || persistenceError) return;
    const loaded = loadSession(localStorage, owner, sessionId, {
      keepPending: true,
    });
    if (!loaded.ok || !loaded.value) {
      error = "Could not open that chat. Your current chat has been kept.";
      return;
    }
    activate(loaded.value);
  }
  async function deleteChat(sessionId: string) {
    const owner = accountDid;
    if (pendingSave(owner, sessionId)) return;
    const attempt = readAttempt(owner, sessionId);
    if (
      !window.confirm(
        attempt
          ? "This record may already exist. Check My Coffee before deleting this chat. Saved coffee records will not be deleted."
          : "Delete this chat and its unsaved draft? Saved coffee records will not be deleted.",
      )
    )
      return;
    if (sessionId === snapshot?.id) await stop();
    if (accountDid !== owner) return;
    const removed = await withAccountLock(owner, () =>
      removeSession(localStorage, owner, sessionId),
    ).catch(() => ({ ok: false as const, error: "unavailable" as const }));
    if (accountDid !== owner) return;
    if (!removed.ok) {
      error = "Could not delete the chat. It has been kept.";
      return;
    }
    try {
      await resolveAttempt(owner, sessionId);
    } catch {
      /* The chat was removed first; never risk a duplicate record. */
    }
    if (accountDid !== owner) return;
    if (sessionId === snapshot?.id) {
      const loaded = loadSession(localStorage, owner, undefined, {
        keepPending: true,
      });
      if (!loaded.ok) {
        recovery = true;
        persistenceError =
          "Stored chats could not be loaded. They have not been overwritten.";
        return;
      }
      activate(
        loaded.value ?? createSession(accountDid, id(), "brew"),
        loaded.value,
      );
    } else refreshHistory();
  }
  async function discardStored() {
    const owner = accountDid;
    if (
      !window.confirm(
        "Discard stored chats and unsaved drafts on this device? Saved coffee records will not be deleted.",
      )
    )
      return;
    const removed = await withAccountLock(owner, () =>
      removeSession(localStorage, owner),
    ).catch(() => ({ ok: false as const, error: "unavailable" as const }));
    if (accountDid !== owner) return;
    if (!removed.ok) {
      persistenceError = "Could not discard stored chats. Keep this tab open.";
      return;
    }
    persistenceError = "";
    activate(createSession(accountDid, id(), "brew"), null);
  }
  function saveInput(s: DraftSession) {
    const result: Record<string, unknown> = {};
    for (const [path, value] of Object.entries(s.draft.fields)) {
      if (value === null || value === undefined) continue;
      const [parent, leaf] = path.split(".");
      if (leaf) {
        const block = (result[parent] ?? {}) as Record<string, unknown>;
        block[leaf] = value;
        result[parent] = block;
      } else result[path] = value;
    }
    return result;
  }
  async function observeSave(
    origin: DraftEngine,
    operation: Promise<DraftSession>,
  ) {
    saving = true;
    try {
      const saved = await operation;
      if (engine === origin) {
        engine = restoreSession(saved);
        choices = [];
        snapshot = engine.snapshot();
      }
    } catch (e) {
      if (engine === origin) {
        error =
          e instanceof Error
            ? e.message
            : "Save failed. Your draft is still here.";
        const attempt = readAttempt(
          origin.snapshot().accountDid,
          origin.snapshot().id,
        );
        uncertainSave = !!attempt;
        savedVersionURL =
          e instanceof SavedRecordError
            ? (e.saved.savedRecordURI ?? "")
            : (attempt?.recordURL ?? "");
        // A different tab may have advanced the stored revision while this write ran.
        // Keep that work canonical here too; never let a later edit re-save the old snapshot.
        try {
          const latest = loadSession(
            localStorage,
            origin.snapshot().accountDid,
            origin.snapshot().id,
            { keepPending: true },
          );
          if (
            latest.ok &&
            latest.value?.id === origin.snapshot().id &&
            latest.value.revision >= origin.snapshot().revision
          ) {
            engine = restoreSession(latest.value);
            snapshot = engine.snapshot();
            if (snapshot.editorRevealed || isUsefulDraft(snapshot.draft))
              revealed = true;
          }
        } catch {
          /* Keep the live draft and unknown-outcome warning. */
        }
      }
    } finally {
      if (
        engine === origin ||
        (engine?.snapshot().id === origin.snapshot().id &&
          engine?.snapshot().accountDid === origin.snapshot().accountDid)
      )
        saving = false;
    }
  }
  async function save(event: SubmitEvent) {
    event.preventDefault();
    const current = engine;
    if (
      !current ||
      saving ||
      uncertainSave ||
      current.snapshot().pending ||
      current.snapshot().status !== "active"
    )
      return;
    const form = event.currentTarget as HTMLFormElement;
    // Hidden controls still participate in validation. Reveal the actual invalid
    // field before reporting it, so Save never leaves an unfocusable error behind.
    if (!form.checkValidity()) {
      manualOpen = true;
      await tick();
      if (engine !== current) return;
      const invalid = form.querySelector<HTMLElement>(
        "input:invalid, select:invalid, textarea:invalid",
      );
      for (const details of form.querySelectorAll<HTMLDetailsElement>(
        "details",
      )) {
        if (invalid && details.contains(invalid)) details.open = true;
      }
      await tick();
      form.reportValidity();
      return;
    }
    saving = true;
    error = "";
    if (!(await sync())) {
      saving = false;
      return;
    }
    if (engine !== current || !reconcile()) {
      saving = false;
      return;
    }
    const s = { ...current.snapshot(), composerText: input };
    const body = saveInput(s);
    try {
      const operation = trackSave(s, async () => {
        if (accountDid !== s.accountDid)
          throw new Error(
            "The account changed before saving began. No record was submitted; check the original account before retrying.",
          );
        let rkey = "";
        let owner = s.accountDid;
        // TODO: future conversational saving requires a separate explicit confirmation contract.
        if (s.draft.kind === "brew") {
          const response = await createBrew(fetch, body as BrewInput);
          rkey = response.brew.rkey;
          owner = response.author_did || owner;
        } else if (s.draft.kind === "bean")
          rkey = (await createBean(fetch, body as BeanInput)).rkey;
        else if (s.draft.kind === "grinder")
          rkey = (await createGrinder(fetch, body as GrinderInput)).rkey;
        else rkey = (await createBrewer(fetch, body as BrewerInput)).rkey;
        if (!rkey)
          throw new Error(
            "The save response did not include a record link. Check My Coffee before retrying.",
          );
        const savedEngine = new DraftEngine(s);
        savedEngine.markSaved(
          `/${s.draft.kind === "brew" ? "brews" : s.draft.kind === "bean" ? "beans" : s.draft.kind === "grinder" ? "grinders" : "brewers"}/${encodeURIComponent(owner)}/${encodeURIComponent(rkey)}`,
        );
        appCache.invalidateCache();
        return savedEngine.snapshot();
      });
      await observeSave(current, operation);
    } catch (e) {
      if (engine === current)
        error =
          e instanceof Error
            ? e.message
            : "Save failed. Your draft is still here.";
    } finally {
      if (engine === current) saving = false;
    }
  }
</script>

{#if !accountDid}<p>Log in to start a chat.</p>
  <a href="/">Home</a>
{:else if !enabled}<p>AI entry is unavailable.</p>
  <a href="/add">Add manually</a>
{:else}
  <div class="chat-workspace">
    <nav class="chat-toolbar" aria-label="Chat actions">
      <a class="back-link" href="/" aria-label="Back" title="Back to Community"
        ><Icon name="arrowLeft" /></a
      >
      <details class="workspace-options" bind:open={optionsOpen}>
        <summary aria-label="Chat options" title="Chat options">···</summary>
        <div class="options-content">
          <a href="/add">Add manually</a>
          {#if histories.length}
            <details class="history" bind:open={historyOpen}>
              <summary>Previous drafts</summary>
              <div class="history-list">
                {#each histories as session (session.id)}
                  <div
                    class="history-row"
                    class:selected={session.id === snapshot?.id}
                  >
                    <button
                      class="history-link"
                      type="button"
                      aria-current={session.id === snapshot?.id
                        ? "page"
                        : undefined}
                      aria-label={`Open chat: ${session.title}`}
                      onclick={() => switchChat(session.id)}
                      >{session.title}</button
                    >
                    <button
                      class="history-delete"
                      type="button"
                      aria-label={`Delete chat: ${session.title}`}
                      disabled={!!pendingSave(accountDid, session.id)}
                      onclick={() => deleteChat(session.id)}>×</button
                    >
                  </div>
                {/each}
              </div>
            </details>
          {/if}

          {#if snapshot?.undo}<button
              type="button"
              disabled={saving}
              onclick={() => {
                guarded((o) => engine!.undo(o));
                optionsOpen = false;
              }}>Undo last AI change</button
            >{/if}
          {#if revealed && snapshot?.status === "active"}<label
              >Record type
              <select
                class="form-select"
                value={snapshot.draft.kind}
                disabled={saving || !!snapshot.pending}
                onchange={async (e) => {
                  const select = e.currentTarget;
                  const kind = select.value as DraftKind;
                  if (engine?.setKind(kind)) sync();
                  else if (
                    window.confirm(
                      "Start a new chat for this record type? Your previous chat and draft will be kept.",
                    )
                  ) {
                    if (await fresh(kind)) {
                      engine?.revealEditor();
                      revealed = true;
                      manualOpen = true;
                      sync();
                    }
                  }
                  await tick();
                  select.value = engine?.snapshot().draft.kind ?? "brew";
                }}
              >
                <option value="brew">Brew</option><option value="bean"
                  >Bean</option
                >
                <option value="grinder">Grinder</option><option value="brewer"
                  >Brewer</option
                >
              </select>
            </label>{/if}
        </div>
      </details>
      <button
        class="new-chat-button"
        type="button"
        disabled={saving || recovery}
        onclick={() => fresh()}><Icon name="plus" />New chat</button
      >
    </nav>
    <div class="chat-body">
      <div
        class="chat-main"
        class:empty-thread={snapshot?.status === "active" &&
          snapshot.messages.length === 0 &&
          !revealed}
      >
        <!-- svelte-ignore a11y_no_noninteractive_tabindex (The named scroll region needs keyboard focus for PageUp/PageDown scrolling.) -->
        <div
          class="chat-scroll"
          role="region"
          aria-label="Chat content"
          tabindex="0"
          bind:this={logElement}
          onscroll={(e) => {
            const el = e.currentTarget;
            nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80;
          }}
        >
          <div class="thread-content">
            {#if persistenceError}<p role="alert">{persistenceError}</p>{/if}
            {#if recovery}<button
                class="btn-secondary"
                type="button"
                onclick={discardStored}>Discard stored chat</button
              >{/if}
            {#if uncertainSave}<p role="alert">
                The previous save outcome needs checking before another save. <a
                  href="/my-coffee">Check My Coffee</a
                >{#if savedVersionURL}
                  · <a href={savedVersionURL}>View saved version</a>{/if}
              </p>
              {#if !savedVersionURL}<button
                  type="button"
                  class="btn-secondary"
                  onclick={async () => {
                    if (
                      window.confirm(
                        "Have you checked My Coffee and confirmed this record was not saved?",
                      )
                    ) {
                      try {
                        const origin = engine;
                        await resolveAttempt(accountDid, snapshot!.id);
                        if (engine !== origin) return;
                        uncertainSave = false;
                        error = "";
                      } catch (e) {
                        error =
                          e instanceof Error
                            ? e.message
                            : "Storage is unavailable.";
                      }
                    }
                  }}>I checked; it was not saved</button
                >{/if}
            {/if}
            {#if snapshot}
              <div
                class="conversation"
                role="log"
                aria-label="Conversation"
                aria-live="polite"
                aria-busy={!!snapshot.pending}
              >
                {#each snapshot.messages as message, index}<div
                    class:user={message.role === "user"}
                    class="message"
                  >
                    <span class="sr-only"
                      >{message.role === "user" ? "You" : "Assistant"}:
                    </span>{message.content}
                  </div>{/each}
              </div>
              {#if snapshot.pending}<WorkingIndicator />{/if}
              {#if notice}<p role="status">{notice}</p>{/if}
              {#if error}<p role="alert">{error}</p>{/if}
              {#if snapshot.status === "saved"}
                <section
                  class="draft-card saved-draft"
                  aria-label="Saved draft"
                >
                  <DraftPreview
                    draft={snapshot.draft}
                    {library}
                    brewerType={snapshot.brewerType}
                  />
                  <a class="saved-link" href={snapshot.savedRecordURI}
                    >View saved {snapshot.draft.kind}</a
                  >
                </section>
              {:else}
                {#if choices.length}<div
                    class="choices"
                    aria-label="Library matches"
                  >
                    {#each choices as choice}<button
                        class="btn-secondary"
                        type="button"
                        disabled={saving}
                        onclick={() => choose(choice)}>{choice.label}</button
                      >{/each}
                  </div>{/if}
                {#if !revealed && snapshot.messages.length && !snapshot.pending}
                  <button
                    class="manual-entry"
                    type="button"
                    onclick={() => {
                      engine?.revealEditor();
                      revealed = true;
                      manualOpen = true;
                      sync();
                    }}>Add details manually</button
                  >
                {/if}
                {#if revealed}
                  {#if libraryError}<p role="alert">
                      {libraryError}
                      <button type="button" onclick={() => loadLibrary()}
                        >Retry library</button
                      >
                    </p>{/if}
                  <form class="draft-card" novalidate onsubmit={save}>
                    <DraftPreview
                      draft={snapshot.draft}
                      showRating={snapshot.draft.kind !== "brew"}
                      {library}
                      brewerType={snapshot.brewerType}
                    />
                    <details class="manual-editor" bind:open={manualOpen}>
                      <summary>Edit details</summary>
                      <div class="editor-fields">
                        {#key snapshot.id + snapshot.draft.kind}<DraftEditor
                            draft={snapshot.draft}
                            brewerType={snapshot.brewerType}
                            {library}
                            presentation="chat"
                            ratingInEditor={false}
                            disabled={saving}
                            onchange={edit}
                          />{/key}
                      </div>
                    </details>
                    <div class="draft-footer">
                      {#if snapshot.draft.kind === "brew"}
                        <DraftRating
                          value={(snapshot.draft.fields as DraftPatch).rating}
                          disabled={saving}
                          onchange={(rating) => edit({ rating })}
                        />
                      {/if}
                      {#if saving}<WorkingIndicator
                          label="Saving your draft…"
                        />{/if}
                      <button
                        type="submit"
                        class="btn-primary save"
                        disabled={saving || uncertainSave || !!snapshot.pending}
                        >{saving
                          ? "Saving…"
                          : `Save ${snapshot.draft.kind}`}</button
                      >
                    </div>
                  </form>
                {/if}
              {/if}
            {/if}
          </div>
        </div>
        {#if snapshot?.status === "active"}
          <div class="composer-dock">
            <form
              class="composer"
              onsubmit={(e) => {
                e.preventDefault();
                void send();
              }}
            >
              <label class="sr-only" for="chat-message">Message</label>
              <textarea
                id="chat-message"
                class="form-textarea"
                placeholder={snapshot.messages.length
                  ? "Message"
                  : "Describe a brew or bean"}
                rows="1"
                maxlength={MAX_USER_CHARS}
                bind:value={input}
                use:autosizeTextarea={input}
                onkeydown={(e) => {
                  if (
                    e.key === "Enter" &&
                    !e.shiftKey &&
                    !e.isComposing &&
                    e.keyCode !== 229
                  ) {
                    e.preventDefault();
                    if (!e.repeat) void send();
                  }
                }}
                disabled={saving || !!snapshot.pending}
                oninput={(e) => {
                  input = e.currentTarget.value;
                  void sync();
                }}
                onblur={() => void sync()}
              ></textarea>
              {#if snapshot.pending}<button
                  class="stop-button"
                  type="button"
                  aria-label="Stop"
                  title="Stop"
                  onclick={stop}
                  ><svg
                    aria-hidden="true"
                    width="14"
                    height="14"
                    viewBox="0 0 14 14"
                    ><rect
                      x="2"
                      y="2"
                      width="10"
                      height="10"
                      rx="1"
                      fill="currentColor"
                    /></svg
                  ></button
                >{:else}<SendButton disabled={saving || !input.trim()} />{/if}
            </form>
            {#if retryable || (!revealed && !snapshot.messages.length)}
              <div class="composer-secondary">
                {#if retryable}<button type="button" onclick={() => send(true)}
                    >Retry</button
                  >{/if}
                {#if !revealed && !snapshot.messages.length}<button
                    type="button"
                    onclick={() => {
                      engine?.revealEditor();
                      revealed = true;
                      manualOpen = true;
                      sync();
                    }}>Add details manually</button
                  >{/if}
              </div>
            {/if}
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  :global([data-theme="dark"]) .chat-workspace {
    color-scheme: dark;
  }
  :global([data-theme="light"]) .chat-workspace {
    color-scheme: light;
  }
  @media (prefers-color-scheme: dark) {
    :global(:root:not([data-theme="light"])) .chat-workspace {
      color-scheme: dark;
    }
  }
  .chat-workspace {
    height: 100%;
    min-height: 0;
    max-width: 96rem;
    margin: auto;
    display: flex;
    flex-direction: column;
    color: var(--text-primary);
  }
  .chat-toolbar {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 1rem;
    flex-shrink: 0;
    min-height: 3.5rem;
    padding: 0.5rem clamp(1rem, 3vw, 2rem);
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .chat-toolbar > button:last-child {
    margin-left: auto;
  }
  .back-link {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 2rem;
    height: 2rem;
    border-radius: 0.5rem;
  }
  .back-link :global(svg),
  .new-chat-button :global(svg) {
    width: 1rem;
    height: 1rem;
  }
  .new-chat-button {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 0.65rem;
    border-radius: 0.5rem;
    color: var(--text-primary);
  }
  .new-chat-button:hover:not(:disabled),
  .back-link:hover {
    background: var(--surface-bg);
  }
  .new-chat-button:disabled {
    opacity: 0.4;
  }
  .chat-body {
    flex: 1;
    min-height: 0;
    display: grid;
    grid-template-columns: minmax(0, 1fr);
  }
  .history {
    min-width: 0;
    border-top: 1px solid var(--surface-border);
    padding-top: 0.5rem;
  }
  .history > summary {
    padding-block: 0.5rem;
    color: var(--text-muted);
    cursor: pointer;
  }
  .history-list {
    max-height: min(18rem, 40dvh);
    overflow-y: auto;
    overscroll-behavior: contain;
    scrollbar-width: thin;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }
  .history-row {
    display: flex;
    align-items: center;
    border-radius: 0.6rem;
    min-height: 2.5rem;
  }
  .history-row.selected {
    background: var(--surface-bg);
  }
  .history-row:hover {
    background: color-mix(in oklch, var(--surface-bg) 65%, transparent);
  }
  .history-row.selected:hover {
    background: var(--surface-bg);
  }
  .history-link {
    flex: 1;
    min-width: 0;
    text-align: left;
    padding: 0.65rem;
    font-size: 0.8rem;
    line-height: 1.5;
    overflow: hidden;
    white-space: normal;
    overflow-wrap: anywhere;
  }
  .history-link[aria-current="page"] {
    font-weight: 600;
  }
  .history-delete {
    flex-shrink: 0;
    width: 2rem;
    align-self: stretch;
    color: var(--text-muted);
    font-size: 1.1rem;
  }
  .history-delete:hover {
    color: var(--text-primary);
  }
  .chat-main {
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }
  .chat-scroll {
    position: relative;
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    overscroll-behavior: contain;
    scrollbar-width: thin;
    scrollbar-color: var(--surface-border) transparent;
  }
  .chat-scroll::-webkit-scrollbar,
  .history::-webkit-scrollbar {
    width: 6px;
  }
  .chat-scroll::-webkit-scrollbar-track,
  .history::-webkit-scrollbar-track {
    background: transparent;
  }
  .chat-scroll::-webkit-scrollbar-thumb,
  .history::-webkit-scrollbar-thumb {
    background: var(--surface-border);
    border-radius: 4px;
  }
  .thread-content {
    width: min(100%, 52rem);
    margin: auto;
    padding: 1.5rem clamp(1.25rem, 3vw, 2rem) 2.5rem;
  }
  .conversation {
    display: flex;
    flex-direction: column;
    gap: 1.75rem;
  }
  .message {
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    max-width: 65ch;
    font-size: 0.95rem;
    line-height: 1.75;
  }
  .message.user {
    align-self: end;
    max-width: min(85%, 36rem);
    padding: 0.65rem 1rem;
    background: var(--surface-bg);
    border-radius: 1rem;
    line-height: 1.6;
  }
  .conversation:not(:empty) {
    margin-bottom: 1.75rem;
  }
  .choices {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 0.75rem;
    margin-block: 1.25rem;
  }
  .options-content label {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .options-content select {
    width: auto;
    min-width: 9rem;
    color: var(--text-primary);
    background: var(--input-bg, var(--surface-bg));
    border: 1px solid var(--surface-border);
    border-radius: 0.4rem;
    padding: 0.55rem 0.75rem;
  }
  .workspace-options {
    position: relative;
  }
  .workspace-options > summary {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2.5rem;
    min-height: 2.5rem;
    border-radius: 0.5rem;
    cursor: pointer;
    list-style: none;
    font-size: 1.15rem;
    letter-spacing: 0.1em;
  }
  .workspace-options > summary::-webkit-details-marker {
    display: none;
  }
  .workspace-options > summary:hover,
  .workspace-options[open] > summary {
    background: var(--surface-bg);
    color: var(--text-primary);
  }
  .options-content {
    position: absolute;
    top: calc(100% + 0.35rem);
    left: 0;
    z-index: 10;
    display: grid;
    gap: 0.85rem;
    width: min(19rem, calc(100vw - 5rem));
    max-height: min(34rem, calc(100dvh - 12rem));
    overflow-y: auto;
    padding: 1rem;
    border: 1px solid var(--surface-border);
    border-radius: 0.75rem;
    background: var(--card-bg);
    color: var(--text-primary);
    box-shadow: 0 0.5rem 1.5rem
      color-mix(in oklch, var(--text-primary) 8%, transparent);
  }
  .options-content button,
  .options-content a {
    text-align: left;
    padding-block: 0.35rem;
  }
  .options-content label {
    flex-wrap: wrap;
  }
  .draft-card {
    margin-top: 1.5rem;
    padding: clamp(1rem, 3vw, 1.5rem);
    background: var(--card-bg);
    border: 1px solid var(--surface-border);
    border-radius: 1rem;
    overflow-wrap: anywhere;
    animation: draft-arrive 220ms ease-out both;
  }
  .manual-editor {
    margin-top: 1.25rem;
    border-top: 1px solid var(--surface-border);
  }
  .manual-editor > summary {
    width: fit-content;
    cursor: pointer;
    padding-block: 0.9rem;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .manual-editor > summary:hover {
    color: var(--text-primary);
  }
  .editor-fields {
    padding-block: 0.25rem 1rem;
  }
  .draft-footer {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
    align-items: center;
    justify-content: flex-end;
    padding-top: 0.5rem;
  }
  .draft-footer :global(.working-indicator) {
    margin: 0 auto 0 0;
  }
  .manual-entry {
    font-size: 0.8rem;
    color: var(--text-muted);
    margin-block: 1rem;
  }
  .saved-draft .saved-link {
    display: inline-block;
    margin-top: 1.25rem;
  }
  .choices button {
    padding: 0.4rem 0.75rem;
    border: 1px solid var(--surface-border);
    border-radius: 1rem;
    background: transparent;
    font-size: 0.8rem;
  }
  .choices button:hover:not(:disabled),
  .choices button:focus-visible {
    background: color-mix(in oklch, var(--btn-primary-bg) 18%, var(--card-bg));
    border-color: var(--btn-primary-bg);
    color: var(--text-primary);
  }
  .choices button:focus-visible,
  .stop-button:focus-visible {
    outline: 2px solid var(--btn-primary-bg);
    outline-offset: 3px;
  }
  .stop-button svg {
    width: 1.75rem;
    height: 1.75rem;
    padding: 0.4375rem;
    border-radius: 50%;
    background: var(--btn-primary-bg);
  }
  .stop-button:hover svg {
    background: var(--btn-primary-bg-hover);
  }
  @keyframes draft-arrive {
    from {
      opacity: 0;
      transform: translateY(0.35rem);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .draft-card {
      animation: none;
    }
  }
  .composer-dock {
    flex-shrink: 0;
    padding: 1rem clamp(1.25rem, 3vw, 2rem);
    background: var(--page-bg, var(--surface-bg));
  }
  .composer {
    view-transition-name: ai-composer;
    position: relative;
    width: min(100%, 48rem);
    margin: auto;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.0625rem 0.15rem 0.0625rem 0.5rem;
    border: 1px solid var(--surface-border);
    border-radius: 1.25rem;
    background: var(--surface-bg);
  }
  .composer:focus-within {
    outline: 2px solid var(--text-muted);
    outline-offset: 2px;
  }
  .composer textarea {
    flex: 1;
    min-width: 0;
    min-height: 1.875rem;
    max-height: min(10rem, 25dvh);
    resize: none;
    background: transparent;
    border: none;
    box-shadow: none;
    outline: none;
    padding: 0.25rem 0.5rem;
    line-height: 1.5;
    font-size: 0.9rem;
  }
  .composer textarea::placeholder {
    color: var(--text-muted);
  }
  .composer textarea:focus {
    box-shadow: none;
    outline: none;
  }
  .stop-button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    width: 2.75rem;
    height: 2.75rem;
    padding: 0;
    border: 0;
    border-radius: 50%;
    background: transparent;
    color: var(--btn-primary-text);
  }
  .composer-secondary {
    width: min(100%, 48rem);
    margin: 0.75rem auto 0;
    display: flex;
    gap: 1rem;
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .composer-secondary button:hover {
    color: var(--text-primary);
  }
  .empty-thread .chat-scroll {
    flex: 0 1 auto;
    max-height: 35%;
  }
  .empty-thread .thread-content {
    padding-block: 0;
  }
  .empty-thread .composer-dock {
    margin-block: auto;
  }
  .saved-link {
    display: inline-block;
    margin-top: 1rem;
  }
  .thread-content > p {
    margin-block: 0.75rem;
    font-size: 0.85rem;
    line-height: 1.6;
  }
  button:focus-visible {
    outline: 2px solid var(--brand-amber-400);
    outline-offset: 2px;
  }
  @media (max-width: 759px) {
    .chat-toolbar {
      gap: 0.5rem;
      min-height: 3.75rem;
      padding: 0.5rem 0.75rem;
      font-size: 0.75rem;
    }
    .chat-toolbar button {
      font-size: 0.75rem;
    }
    .chat-toolbar > button:last-child {
      padding: 0.5rem 0.4rem;
    }
    .back-link {
      width: 1.75rem;
    }
    .thread-content {
      padding: 1.25rem 1rem;
    }
    .composer-dock {
      padding: 0.75rem 1rem;
    }
    .composer {
      gap: 0.35rem;
    }
  }
</style>
