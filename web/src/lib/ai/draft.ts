import { normalizeBrewerCategory } from "../utils/brewFormPolicy";
import { MAX_MESSAGES, MAX_USER_CHARS } from "./types";
import type {
  ApplyResult,
  ChangeOptions,
  DraftKind,
  DraftPatch,
  DraftSession,
  FieldPath,
  FieldValue,
  LibraryChoice,
  Message,
  RequestToken,
} from "./types";

const stringFields: Record<DraftKind, string[]> = {
  brew: [
    "bean_rkey",
    "recipe_rkey",
    "recipe_owner_did",
    "method",
    "grind_size",
    "grinder_rkey",
    "brewer_rkey",
    "tasting_notes",
    "pourover_params.filter",
  ],
  bean: [
    "name",
    "origin",
    "variety",
    "roast_level",
    "roast_date",
    "process",
    "description",
    "notes",
    "link",
    "roaster_rkey",
    "source_ref",
  ],
  grinder: ["name", "grinder_type", "burr_type", "notes", "link", "source_ref"],
  brewer: ["name", "brewer_type", "description", "link", "source_ref"],
};
const numberFields: Record<DraftKind, string[]> = {
  brew: [
    "temperature",
    "water_amount",
    "coffee_amount",
    "time_seconds",
    "rating",
    "espresso_params.yield_weight",
    "espresso_params.pressure",
    "espresso_params.pre_infusion_seconds",
    "pourover_params.bloom_water",
    "pourover_params.bloom_seconds",
    "pourover_params.drawdown_seconds",
    "pourover_params.bypass_water",
  ],
  bean: ["rating"],
  grinder: [],
  brewer: [],
};
export const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value));
const entries = (patch: DraftPatch) =>
  Object.entries(patch) as [FieldPath, FieldValue][];
const meaningful = (value: unknown): boolean =>
  typeof value === "string"
    ? !!value.trim()
    : Array.isArray(value)
      ? value.length > 0
      : typeof value === "number"
        ? value > 0
        : value === true;
const same = (a: unknown, b: unknown) =>
  JSON.stringify(a) === JSON.stringify(b);

/** Structural/finite-value validation only; existing mutation APIs own save validation. */
export function validPatch(
  kind: DraftKind,
  patch: unknown,
): patch is DraftPatch {
  if (
    !Object.hasOwn(stringFields, kind) ||
    !patch ||
    typeof patch !== "object" ||
    Array.isArray(patch)
  )
    return false;
  return Object.entries(patch).every(([key, value]) => {
    const string = stringFields[kind].includes(key);
    const number = numberFields[kind].includes(key);
    const closed = kind === "bean" && key === "closed";
    const pours = kind === "brew" && key === "pours";
    if (!string && !number && !closed && !pours) return false;
    if (value === null) return true;
    if (string) return typeof value === "string";
    if (number) return typeof value === "number" && Number.isFinite(value);
    if (closed) return typeof value === "boolean";
    return (
      Array.isArray(value) &&
      value.length <= 100 &&
      value.every(
        (p) =>
          p &&
          typeof p === "object" &&
          !Array.isArray(p) &&
          Object.keys(p).length === 2 &&
          typeof p.water_amount === "number" &&
          Number.isFinite(p.water_amount) &&
          typeof p.time_seconds === "number" &&
          Number.isFinite(p.time_seconds),
      )
    );
  });
}

export function createSession(
  accountDid: string,
  sessionId: string,
  kind: DraftKind,
): DraftSession {
  if (!accountDid || !sessionId || !Object.hasOwn(stringFields, kind))
    throw new Error("Invalid session identity or kind");
  return {
    accountDid,
    id: sessionId,
    status: "active",
    draft: { kind, fields: {} },
    revision: 0,
    fieldRevisions: {},
    messages: [],
    requestIds: [],
    completedRequestIds: [],
  };
}

/** Display threshold, deliberately separate from save validity. Never blocks manual editing. */
export function isUsefulDraft(draft: DraftSession["draft"]): boolean {
  const f = draft.fields as DraftPatch;
  if (draft.kind === "bean") return meaningful(f.name) && meaningful(f.origin);
  if (draft.kind === "grinder") return meaningful(f.name);
  if (draft.kind === "brewer") return meaningful(f.name);
  const bean = meaningful(f.bean_rkey);
  const brewer = meaningful(f.brewer_rkey);
  const substantive = entries(f).filter(
    ([key, value]) =>
      ![
        "bean_rkey",
        "brewer_rkey",
        "grinder_rkey",
        "recipe_rkey",
        "recipe_owner_did",
      ].includes(key) && meaningful(value),
  ).length;
  return (bean && brewer) || ((bean || brewer) && substantive >= 2);
}

const brewerDetail = (key: string) =>
  key === "method" ||
  key === "pours" ||
  key.startsWith("espresso_params.") ||
  key.startsWith("pourover_params.");
function dependentClears(
  session: DraftSession,
  patch: DraftPatch,
  options: ChangeOptions,
): FieldPath[] {
  if (session.draft.kind !== "brew") return [];
  const f = session.draft.fields as DraftPatch;
  const clears: FieldPath[] = [];
  if (
    Object.hasOwn(patch, "grinder_rkey") &&
    !same(f.grinder_rkey, patch.grinder_rkey) &&
    meaningful(f.grind_size)
  )
    clears.push("grind_size");
  if (
    Object.hasOwn(patch, "brewer_rkey") &&
    !same(f.brewer_rkey, patch.brewer_rkey)
  ) {
    if (meaningful(f.recipe_rkey) && !Object.hasOwn(patch, "recipe_rkey")) {
      clears.push("recipe_rkey", "recipe_owner_did");
    }
    const category = normalizeBrewerCategory(options.brewerType);
    for (const [key, value] of entries(f)) {
      if (!meaningful(value)) continue;
      // Missing metadata is not proof that existing preparation is incompatible.
      if (patch.brewer_rkey && !options.brewerType) continue;
      if (key.startsWith("espresso_params.") && category !== "espresso")
        clears.push(key);
      if (
        (key.startsWith("pourover_params.") || key === "pours") &&
        category !== "pourover"
      )
        clears.push(key);
      if (
        key === "method" &&
        normalizeBrewerCategory(String(value)) !== "other" &&
        normalizeBrewerCategory(String(value)) !== category
      )
        clears.push(key);
    }
  }
  return clears;
}
const recipePaths: FieldPath[] = ["recipe_rkey", "recipe_owner_did"];
function guardedPaths(key: FieldPath): FieldPath[] {
  if (recipePaths.includes(key)) return [...recipePaths, "brewer_rkey"];
  if (
    [
      "coffee_amount",
      "water_amount",
      "temperature",
      "time_seconds",
      "brewer_rkey",
    ].includes(key)
  )
    return [key, ...recipePaths];
  if (key === "grind_size") return [key, "grinder_rkey", ...recipePaths];
  if (brewerDetail(key)) return [key, "brewer_rkey", ...recipePaths];
  return [key];
}

/** Owns local draft state only. Never calls fetch, localStorage, dialogs, or record mutations. */
export class DraftEngine {
  private state: DraftSession;
  constructor(session: DraftSession) {
    this.state = clone(session);
  }
  snapshot(): DraftSession {
    return clone(this.state);
  }
  revealEditor(): void {
    this.state.editorRevealed = true;
  }
  private result(
    status: ApplyResult["status"],
    extra: Partial<ApplyResult> = {},
  ): ApplyResult {
    return {
      status,
      applied: [],
      skipped: [],
      requiredClears: [],
      revision: this.state.revision,
      ...extra,
    };
  }
  beginRequest(requestId: string): RequestToken | null {
    const s = this.state;
    if (
      s.status !== "active" ||
      s.pending ||
      !requestId ||
      s.requestIds.includes(requestId) ||
      s.requestIds.length >= MAX_MESSAGES
    )
      return null;
    const token = {
      accountDid: s.accountDid,
      sessionId: s.id,
      requestId,
      revision: s.revision,
    };
    s.pending = token;
    s.requestIds.push(requestId);
    return clone(token);
  }
  /** Only a provisional empty draft can accept a kind discovered by the assistant. */
  setKind(kind: DraftKind): boolean {
    if (
      this.state.status !== "active" ||
      !Object.hasOwn(stringFields, kind) ||
      entries(this.state.draft.fields).some(([, value]) => meaningful(value))
    )
      return false;
    if (this.state.draft.kind === kind) return true;
    this.state.draft = { kind, fields: {} };
    this.state.fieldRevisions = {};
    delete this.state.undo;
    delete this.state.brewerType;
    return true;
  }
  stopRequest(): void {
    delete this.state.pending;
  }
  private owns(token: RequestToken): boolean {
    return (
      token.accountDid === this.state.accountDid &&
      token.sessionId === this.state.id &&
      token.revision <= this.state.revision &&
      this.state.status === "active"
    );
  }
  applyResponse(
    token: RequestToken,
    patch: DraftPatch,
    options: ChangeOptions = {},
  ): ApplyResult {
    if (!this.owns(token) || !same(this.state.pending, token))
      return this.result("inactive");
    const result = this.apply(patch, options, token.revision, true);
    if (result.status !== "needs-confirmation" && result.status !== "invalid") {
      this.state.completedRequestIds.push(token.requestId);
      delete this.state.pending;
    }
    return result;
  }
  edit(patch: DraftPatch, options: ChangeOptions = {}): ApplyResult {
    return this.apply(patch, options);
  }
  applyChoice(choice: LibraryChoice, options: ChangeOptions = {}): ApplyResult {
    if (
      !this.owns(choice.token) ||
      !this.state.completedRequestIds.includes(choice.token.requestId)
    )
      return this.result("inactive");
    if (!validPatch(this.state.draft.kind, choice.patch))
      return this.result("invalid", {
        error: "Invalid draft fields or values",
      });
    const changeOptions = { ...options, brewerType: choice.brewerType };
    const stale = entries(choice.patch)
      .filter(
        ([key]) =>
          guardedPaths(key).some(
            (path) =>
              (this.state.fieldRevisions[path] ?? 0) > choice.token.revision,
          ) ||
          ((key === "grinder_rkey" || key === "brewer_rkey") &&
            Object.entries(this.state.fieldRevisions).some(
              ([path, revision]) =>
                revision! > choice.token.revision &&
                (key === "grinder_rkey"
                  ? path === "grind_size"
                  : brewerDetail(path)),
            )),
      )
      .map(([key]) => key);
    if (stale.length) {
      if (options.confirmRevision !== this.state.revision)
        return this.result("needs-confirmation", {
          skipped: stale,
          requiredClears: [
            ...new Set([
              ...stale,
              ...dependentClears(this.state, choice.patch, changeOptions),
            ]),
          ],
        });
      // A reviewed choice is a new local instruction, never an exemption for a late response.
      return this.apply(choice.patch, changeOptions);
    }
    return this.apply(choice.patch, changeOptions, choice.token.revision);
  }
  private apply(
    patch: DraftPatch,
    options: ChangeOptions = {},
    base?: number,
    ai = false,
  ): ApplyResult {
    const s = this.state;
    if (s.status !== "active") return this.result("inactive");
    if (!validPatch(s.draft.kind, patch))
      return this.result("invalid", {
        error: "Invalid draft fields or values",
      });
    // Recipe owner and key are one reference. Own-only selectors omit the owner.
    if (
      Object.hasOwn(patch, "recipe_rkey") &&
      !Object.hasOwn(patch, "recipe_owner_did")
    )
      patch = { ...patch, recipe_owner_did: null };
    const f = s.draft.fields as DraftPatch;
    const skipped: FieldPath[] = [];
    const accepted: DraftPatch = {};
    for (const [key, value] of entries(patch)) {
      if (
        base !== undefined &&
        guardedPaths(key).some((path) => (s.fieldRevisions[path] ?? 0) > base)
      )
        skipped.push(key);
      else Object.assign(accepted, { [key]: clone(value) });
    }
    // Never apply a partial recipe selection after a newer manual edit.
    if (
      base !== undefined &&
      Object.hasOwn(patch, "recipe_rkey") &&
      skipped.length
    )
      return this.result("conflict", {
        skipped: entries(patch).map(([key]) => key),
      });
    // Changing a reference must not erase a newer edit in its dependency group.
    if (base !== undefined) {
      for (const reference of ["grinder_rkey", "brewer_rkey"] as const) {
        if (
          !Object.hasOwn(accepted, reference) ||
          same(accepted[reference], f[reference])
        )
          continue;
        const newer = Object.entries(s.fieldRevisions).some(
          ([path, rev]) =>
            rev! > base &&
            (reference === "grinder_rkey"
              ? path === "grind_size"
              : brewerDetail(path)),
        );
        if (newer || skipped.length)
          return this.result("conflict", {
            skipped: entries(patch).map(([key]) => key),
          });
      }
    }
    const clears = dependentClears(s, accepted, options);
    if (clears.length && options.confirmRevision !== s.revision)
      return this.result("needs-confirmation", {
        skipped,
        requiredClears: clears,
      });
    for (const key of clears) {
      // A confirmed new setting may replace the old one; otherwise clear explicitly.
      if (!Object.hasOwn(accepted, key))
        Object.assign(accepted, { [key]: null });
    }
    const changed = entries(accepted).filter(
      ([key, value]) => !same(f[key], value),
    );
    if (!changed.length)
      return this.result(skipped.length ? "conflict" : "applied", { skipped });
    const before: DraftPatch = {};
    const absent: FieldPath[] = [];
    for (const [key] of changed) {
      if (Object.hasOwn(f, key))
        Object.assign(before, { [key]: clone(f[key]) });
      else {
        absent.push(key);
        Object.assign(before, { [key]: null });
      }
    }
    const oldBrewerType = s.brewerType;
    s.revision++;
    for (const [key, value] of changed) {
      Object.assign(f, { [key]: value });
      s.fieldRevisions[key] = s.revision;
    }
    if (changed.some(([key]) => key === "brewer_rkey"))
      s.brewerType = options.brewerType;
    if (isUsefulDraft(s.draft)) s.editorRevealed = true;
    if (ai)
      s.undo = {
        revision: s.revision,
        before,
        absent,
        brewerType: oldBrewerType,
      };
    return this.result(skipped.length ? "conflict" : "applied", {
      applied: changed.map(([key]) => key),
      skipped,
    });
  }
  undo(options: ChangeOptions = {}): ApplyResult {
    const s = this.state;
    const undo = s.undo;
    if (!undo || s.status !== "active") return this.result("inactive");
    const patch: DraftPatch = {};
    const skipped: FieldPath[] = [];
    for (const [key, value] of entries(undo.before)) {
      if (
        s.fieldRevisions[key] !== undo.revision ||
        guardedPaths(key).some(
          (path) => (s.fieldRevisions[path] ?? 0) > undo.revision,
        )
      )
        skipped.push(key);
      else Object.assign(patch, { [key]: value });
    }
    // Undo cannot clear a dependency manually edited after the AI batch.
    for (const ref of ["grinder_rkey", "brewer_rkey"] as const) {
      if (!Object.hasOwn(patch, ref)) continue;
      if (
        Object.entries(s.fieldRevisions).some(
          ([key, rev]) =>
            rev! > undo.revision &&
            (ref === "grinder_rkey" ? key === "grind_size" : brewerDetail(key)),
        )
      ) {
        delete patch[ref];
        skipped.push(ref);
      }
    }
    const result = this.apply(patch, {
      ...options,
      brewerType: undo.brewerType,
    });
    if (result.status === "needs-confirmation" || result.status === "invalid")
      return result;
    for (const key of undo.absent)
      if (Object.hasOwn(patch, key)) delete (s.draft.fields as DraftPatch)[key];
    delete s.undo;
    return {
      ...result,
      skipped,
      status: skipped.length ? "conflict" : result.status,
    };
  }
  appendMessage(message: Message): { ok: boolean; error?: string } {
    if (this.state.status !== "active")
      return { ok: false, error: "Session is not active" };
    if (
      !["user", "assistant"].includes(message.role) ||
      typeof message.content !== "string" ||
      this.state.messages.length >= MAX_MESSAGES ||
      (message.role === "user" && message.content.length > MAX_USER_CHARS) ||
      (message.imageCount !== undefined &&
        (message.role !== "user" ||
          !Number.isInteger(message.imageCount) ||
          message.imageCount < 1 ||
          message.imageCount > 4))
    )
      return {
        ok: false,
        error: "Conversation limit reached or invalid message",
      };
    // Never let transport-only image bytes enter the canonical/persisted draft.
    this.state.messages.push({
      role: message.role,
      content: message.content,
      ...(message.imageCount ? { imageCount: message.imageCount } : {}),
    });
    return { ok: true };
  }
  markSaved(recordURI: string): boolean {
    if (this.state.status !== "active" || this.state.pending || !recordURI)
      return false;
    this.state.status = "saved";
    this.state.savedRecordURI = recordURI;
    delete this.state.undo;
    return true;
  }
  discard(): void {
    this.stopRequest();
    this.state.status = "discarded";
    delete this.state.undo;
  }
}
