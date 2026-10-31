import { describe, expect, it } from "vitest";
import { createSession, DraftEngine, isUsefulDraft, validPatch } from "./draft";
import type { DraftPatch } from "./types";
const engine = () =>
  new DraftEngine(createSession("did:plc:a", "session-a", "brew"));

describe("draft request reconciliation", () => {
  it("applies untouched leaf fields but protects later manual edits", () => {
    const e = engine();
    const token = e.beginRequest("r1")!;
    e.edit({ "espresso_params.pressure": 8 });
    const r = e.applyResponse(token, {
      "espresso_params.pressure": 9,
      "espresso_params.yield_weight": 36,
    });
    expect(r.status).toBe("conflict");
    expect(r.skipped).toEqual(["espresso_params.pressure"]);
    expect(e.snapshot().draft.fields).toEqual({
      "espresso_params.pressure": 8,
      "espresso_params.yield_weight": 36,
    });
    const next = e.beginRequest("r2")!;
    expect(
      e.applyResponse(next, { "espresso_params.pressure": 9 }).status,
    ).toBe("applied");
    expect(e.snapshot().draft.fields).toHaveProperty(
      "espresso_params.pressure",
      9,
    );
  });
  it("treats null as clear, absence as unchanged, and arrays atomically", () => {
    const e = engine();
    e.edit({
      tasting_notes: "sweet",
      temperature: 93,
      pours: [{ water_amount: 100, time_seconds: 20 }],
    });
    const t = e.beginRequest("r")!;
    expect(e.applyResponse(t, { tasting_notes: null, pours: [] }).status).toBe(
      "applied",
    );
    expect(e.snapshot().draft.fields).toEqual({
      tasting_notes: null,
      temperature: 93,
      pours: [],
    });
  });
  it("rejects stopped, foreign, reused, saved and discarded request responses", () => {
    const e = engine();
    const t = e.beginRequest("r")!;
    expect(e.markSaved("at://record")).toBe(false);
    expect(
      e.applyResponse({ ...t, sessionId: "other" }, { temperature: 90 }).status,
    ).toBe("inactive");
    expect(
      e.applyResponse({ ...t, accountDid: "did:plc:b" }, { temperature: 90 })
        .status,
    ).toBe("inactive");
    e.stopRequest();
    expect(e.applyResponse(t, { temperature: 90 }).status).toBe("inactive");
    expect(e.applyChoice({ token: t, patch: { bean_rkey: "x" } }).status).toBe(
      "inactive",
    );
    expect(e.beginRequest("r")).toBeNull();
    const t2 = e.beginRequest("r2")!;
    e.applyResponse(t2, {});
    expect(e.markSaved("at://record")).toBe(true);
    expect(e.applyResponse(t2, { temperature: 90 }).status).toBe("inactive");
    expect(e.edit({ temperature: 90 }).status).toBe("inactive");
    const d = engine();
    const td = d.beginRequest("d")!;
    d.discard();
    expect(d.applyResponse(td, {}).status).toBe("inactive");
  });
  it("rejects stale match clicks without a model round trip", () => {
    const e = engine();
    const t = e.beginRequest("r")!;
    e.applyResponse(t, {});
    expect(
      e.applyChoice({ token: t, patch: { bean_rkey: "old" } }).status,
    ).toBe("applied");
    e.edit({ bean_rkey: "new" });
    expect(
      e.applyChoice({ token: t, patch: { bean_rkey: "old" } }).status,
    ).toBe("needs-confirmation");
    expect(e.snapshot().draft.fields).toHaveProperty("bean_rkey", "new");
  });
  it("does not expose mutable references", () => {
    const e = engine();
    const s = e.snapshot();
    s.draft.fields = { temperature: 99 };
    const t = e.beginRequest("r")!;
    t.requestId = "changed";
    expect(e.snapshot().pending?.requestId).toBe("r");
    expect(e.snapshot().draft.fields).toEqual({});
  });
});

describe("choice confirmation and editor visibility", () => {
  it("rechecks stale choices after edits and applies only the currently reviewed revision", () => {
    const e = engine();
    const t = e.beginRequest("r")!;
    e.applyResponse(t, {});
    const choice = { token: t, patch: { bean_rkey: "old" } };
    e.edit({ bean_rkey: "new" });
    const review = e.applyChoice(choice);
    expect(review.status).toBe("needs-confirmation");
    expect(review.requiredClears).toEqual(["bean_rkey"]);
    e.edit({ tasting_notes: "manual" });
    const fresh = e.applyChoice(choice, { confirmRevision: review.revision });
    expect(fresh.status).toBe("needs-confirmation");
    expect(
      e.applyChoice(choice, { confirmRevision: fresh.revision }).status,
    ).toBe("applied");
    expect(e.snapshot().draft.fields).toHaveProperty("bean_rkey", "old");
  });
  it("includes dependent clears in stale choice review", () => {
    const e = engine();
    const t = e.beginRequest("r")!;
    e.applyResponse(t, {});
    e.edit({ grinder_rkey: "new", grind_size: "20" });
    const choice = { token: t, patch: { grinder_rkey: "old" } };
    const review = e.applyChoice(choice);
    expect(review.requiredClears).toEqual(["grinder_rkey", "grind_size"]);
    expect(
      e.applyChoice(choice, { confirmRevision: review.revision }).status,
    ).toBe("applied");
    expect(e.snapshot().draft.fields).toMatchObject({
      grinder_rkey: "old",
      grind_size: null,
    });
  });
  it("confirmation never authorizes a stale model response to override manual edits", () => {
    const e = engine();
    const t = e.beginRequest("r")!;
    e.edit({ bean_rkey: "manual" });
    const r = e.applyResponse(
      t,
      { bean_rkey: "model" },
      { confirmRevision: e.snapshot().revision },
    );
    expect(r.status).toBe("conflict");
    expect(e.snapshot().draft.fields).toHaveProperty("bean_rkey", "manual");
  });
  it("editor reveal stays sticky after useful fields are cleared and supports manual reveal", () => {
    const e = engine();
    expect(e.snapshot().editorRevealed).toBeUndefined();
    e.edit({ bean_rkey: "b", brewer_rkey: "v" });
    expect(e.snapshot().editorRevealed).toBe(true);
    e.edit({ bean_rkey: null, brewer_rkey: null });
    expect(e.snapshot().editorRevealed).toBe(true);
    const manual = engine();
    manual.revealEditor();
    expect(manual.snapshot().editorRevealed).toBe(true);
    expect(manual.snapshot().draft.fields).toEqual({});
  });
});

describe("dependent fields", () => {
  it("requires revision-bound confirmation and rechecks intervening direct edits", () => {
    const e = engine();
    e.edit({ grinder_rkey: "one", grind_size: "20" });
    const r = e.edit({ grinder_rkey: "two" });
    expect(r.status).toBe("needs-confirmation");
    expect(r.requiredClears).toEqual(["grind_size"]);
    e.edit({ grind_size: "21" });
    expect(
      e.edit({ grinder_rkey: "two" }, { confirmRevision: r.revision }).status,
    ).toBe("needs-confirmation");
    expect(e.snapshot().draft.fields).toHaveProperty("grinder_rkey", "one");
    const review = e.edit({ grinder_rkey: "two" });
    expect(
      e.edit({ grinder_rkey: "two" }, { confirmRevision: review.revision })
        .status,
    ).toBe("applied");
    expect(e.snapshot().draft.fields).toMatchObject({
      grinder_rkey: "two",
      grind_size: null,
    });
  });
  it("normalizes brewer categories and asks before clearing incompatible details", () => {
    const e = engine();
    e.edit(
      {
        brewer_rkey: "espresso",
        method: "Espresso",
        "espresso_params.pressure": 9,
      },
      { brewerType: "espresso machine" },
    );
    const r = e.edit({ brewer_rkey: "v60" }, { brewerType: "Pour-over" });
    expect(r.requiredClears).toEqual(["method", "espresso_params.pressure"]);
    e.edit(
      { brewer_rkey: "v60" },
      { brewerType: "Pour-over", confirmRevision: r.revision },
    );
    expect(e.snapshot().draft.fields).toMatchObject({
      brewer_rkey: "v60",
      method: null,
      "espresso_params.pressure": null,
    });
    expect(e.snapshot().brewerType).toBe("Pour-over");
  });
  it("keeps compatible details and unknown free-text method labels", () => {
    const e = engine();
    e.edit(
      {
        brewer_rkey: "v60",
        method: "My technique",
        "pourover_params.bloom_water": 50,
      },
      { brewerType: "dripper" },
    );
    expect(
      e.edit({ brewer_rkey: "kalita" }, { brewerType: "pourover" }).status,
    ).toBe("applied");
    expect(e.snapshot().draft.fields).toHaveProperty(
      "pourover_params.bloom_water",
      50,
    );
  });
  it("protects reference-dependent responses in both directions", () => {
    const e = engine();
    e.edit({ grinder_rkey: "one" });
    const t = e.beginRequest("r")!;
    e.edit({ grinder_rkey: "two" });
    expect(e.applyResponse(t, { grind_size: "20" }).skipped).toEqual([
      "grind_size",
    ]);
    const t2 = e.beginRequest("r2")!;
    e.edit({ grind_size: "25" });
    expect(e.applyResponse(t2, { grinder_rkey: "three" }).status).toBe(
      "conflict",
    );
    expect(e.snapshot().draft.fields).toMatchObject({
      grinder_rkey: "two",
      grind_size: "25",
    });
  });
  it("leaves confirmation requests pending until confirmed or stopped", () => {
    const e = engine();
    e.edit({ grinder_rkey: "one", grind_size: "20" });
    const t = e.beginRequest("r")!;
    const r = e.applyResponse(t, { grinder_rkey: "two" });
    expect(e.snapshot().pending).toEqual(t);
    expect(
      e.applyResponse(
        t,
        { grinder_rkey: "two" },
        { confirmRevision: r.revision },
      ).status,
    ).toBe("applied");
    expect(e.snapshot().pending).toBeUndefined();
  });
});

describe("undo and bounded session", () => {
  it("undo preserves subsequent manual edits and restores absent fields", () => {
    const e = engine();
    e.edit({ temperature: 90 });
    const t = e.beginRequest("r")!;
    e.applyResponse(t, {
      temperature: 93,
      tasting_notes: "sweet",
      "espresso_params.pressure": 9,
    });
    e.edit({ tasting_notes: "chocolate" });
    const result = e.undo();
    expect(result.status).toBe("conflict");
    expect(e.snapshot().draft.fields).toEqual({
      temperature: 90,
      tasting_notes: "chocolate",
    });
    expect(e.undo().status).toBe("inactive");
  });
  it("undo cannot destroy later manually edited dependency values", () => {
    const e = engine();
    const t = e.beginRequest("r")!;
    e.applyResponse(t, { grinder_rkey: "one" });
    e.edit({ grind_size: "20" });
    expect(e.undo().skipped).toContain("grinder_rkey");
    expect(e.snapshot().draft.fields).toMatchObject({
      grinder_rkey: "one",
      grind_size: "20",
    });
  });
  it("undo skips details whose reference was manually changed after the AI batch", () => {
    const e = engine();
    e.edit(
      { brewer_rkey: "v60", "pourover_params.bloom_water": 40 },
      { brewerType: "pourover" },
    );
    const t = e.beginRequest("r")!;
    e.applyResponse(t, { "pourover_params.bloom_water": 50 });
    e.edit({ brewer_rkey: "kalita" }, { brewerType: "pourover" });
    expect(e.undo().skipped).toContain("pourover_params.bloom_water");
    expect(e.snapshot().draft.fields).toHaveProperty(
      "pourover_params.bloom_water",
      50,
    );
  });
  it("runs dependency confirmation for undo too", () => {
    const e = engine();
    e.edit(
      { brewer_rkey: "v60", "pourover_params.bloom_water": 50 },
      { brewerType: "pourover" },
    );
    const t = e.beginRequest("r")!;
    let r = e.applyResponse(
      t,
      { brewer_rkey: "espresso", "espresso_params.pressure": 9 },
      { brewerType: "espresso" },
    );
    r = e.applyResponse(
      t,
      { brewer_rkey: "espresso", "espresso_params.pressure": 9 },
      { brewerType: "espresso", confirmRevision: r.revision },
    );
    expect(r.status).toBe("applied");
    const undo = e.undo();
    expect(undo.status).toBe("needs-confirmation");
    expect(e.undo({ confirmRevision: undo.revision }).status).toBe("applied");
    expect(e.snapshot().draft.fields).toEqual({
      brewer_rkey: "v60",
      "pourover_params.bloom_water": 50,
    });
  });
  it("does not silently truncate message text or evict history", () => {
    const e = engine();
    expect(
      e.appendMessage({ role: "user", content: "a".repeat(4001) }).ok,
    ).toBe(false);
    for (let i = 0; i < 40; i++)
      expect(e.appendMessage({ role: "user", content: String(i) }).ok).toBe(
        true,
      );
    expect(e.appendMessage({ role: "assistant", content: "41" }).ok).toBe(
      false,
    );
    expect(e.snapshot().messages).toHaveLength(40);
  });
  it("kind discovery cannot replace a useful or partially filled draft", () => {
    const e = engine();
    expect(e.setKind("bean")).toBe(true);
    e.edit({ name: "coffee" });
    expect(e.setKind("brew")).toBe(false);
  });
  it("rejects unknown/nested fields, undefined, nonfinite numbers and malformed arrays", () => {
    for (const p of [
      { nope: 1 },
      { temperature: Infinity },
      { temperature: undefined },
      { espresso_params: { pressure: 9 } },
      { pours: [{ water_amount: 1 }] },
      { name: "wrong kind" },
    ])
      expect(validPatch("brew", p)).toBe(false);
    expect(engine().edit({ temperature: Infinity }).status).toBe("invalid");
  });
  it.each([
    [
      "brew",
      { bean_rkey: "b", brewer_rkey: "v", tasting_notes: "sweet" },
      true,
    ],
    ["brew", { bean_rkey: "b", brewer_rkey: "v" }, true],
    ["brew", { bean_rkey: "b", brewer_rkey: "v", coffee_amount: 18 }, true],
    [
      "brew",
      { bean_rkey: "b", coffee_amount: 18, tasting_notes: "sweet" },
      true,
    ],
    ["brew", { bean_rkey: "b", coffee_amount: 18 }, false],
    ["brew", { bean_rkey: "b", coffee_amount: 0, tasting_notes: " " }, false],
    [
      "brew",
      { coffee_amount: 18, temperature: 93, tasting_notes: "sweet" },
      false,
    ],
    ["bean", { name: "Coffee", origin: "Kenya" }, true],
    ["bean", { name: "Coffee" }, false],
    ["grinder", { name: "C40", grinder_type: "manual" }, true],
    ["brewer", { name: "V60", brewer_type: "pourover" }, true],
    ["brewer", { name: "V60", brewer_type: " " }, true],
  ] as const)("useful %s draft threshold", (kind, fields, expected) => {
    expect(
      isUsefulDraft({ kind, fields } as ReturnType<
        typeof createSession
      >["draft"]),
    ).toBe(expected);
  });
});
