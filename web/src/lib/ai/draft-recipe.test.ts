import { describe, expect, it } from "vitest";
import { createSession, DraftEngine } from "./draft";
const engine = () =>
  new DraftEngine(createSession("did:plc:alice", "chat", "brew"));
describe("recipe draft dependencies", () => {
  it("requires revision-bound confirmation to clear stale recipe when brewer changes", () => {
    const e = engine();
    e.edit(
      {
        recipe_rkey: "recipe",
        brewer_rkey: "orea",
        pours: [{ water_amount: 70, time_seconds: 30 }],
      },
      { brewerType: "pourover" },
    );
    const before = e.snapshot();
    const result = e.edit(
      { brewer_rkey: "espresso" },
      { brewerType: "espresso" },
    );
    expect(result.status).toBe("needs-confirmation");
    expect(result.requiredClears).toEqual([
      "recipe_rkey",
      "recipe_owner_did",
      "pours",
    ]);
    expect(e.snapshot()).toEqual(before);
    expect(
      e.edit(
        { brewer_rkey: "espresso" },
        { brewerType: "espresso", confirmRevision: result.revision },
      ).status,
    ).toBe("applied");
    expect(e.snapshot().draft.fields).toMatchObject({
      recipe_rkey: null,
      recipe_owner_did: null,
      pours: null,
      brewer_rkey: "espresso",
    });
  });
  it("keeps compatible preparation but confirms discarding the old brewer-linked recipe", () => {
    const e = engine();
    const pours = [{ water_amount: 70, time_seconds: 30 }];
    e.edit(
      { recipe_rkey: "old", brewer_rkey: "orea", pours },
      { brewerType: "pourover" },
    );
    const r = e.edit({ brewer_rkey: "v60" }, { brewerType: "pourover" });
    expect(r.requiredClears).toEqual(["recipe_rkey", "recipe_owner_did"]);
    e.edit({ coffee_amount: 21 });
    expect(
      e.edit(
        { brewer_rkey: "v60" },
        { brewerType: "pourover", confirmRevision: r.revision },
      ).status,
    ).toBe("needs-confirmation");
    expect(e.snapshot().draft.fields).toMatchObject({
      recipe_rkey: "old",
      brewer_rkey: "orea",
      pours,
    });
    const latest = e.snapshot().revision;
    e.edit(
      { brewer_rkey: "v60" },
      { brewerType: "pourover", confirmRevision: latest },
    );
    expect(e.snapshot().draft.fields).toMatchObject({
      recipe_rkey: null,
      brewer_rkey: "v60",
      pours,
    });
  });
  it("replaces a recipe atomically while clearing incompatible old method details", () => {
    const e = engine();
    e.edit(
      {
        recipe_rkey: "old",
        brewer_rkey: "espresso",
        "espresso_params.pressure": 9,
      },
      { brewerType: "espresso" },
    );
    const patch = {
      recipe_rkey: "new",
      brewer_rkey: "orea",
      coffee_amount: 18,
      pours: [{ water_amount: 70, time_seconds: 30 }],
    };
    const r = e.edit(patch, { brewerType: "pourover" });
    expect(r.requiredClears).toEqual(["espresso_params.pressure"]);
    e.edit(patch, { brewerType: "pourover", confirmRevision: r.revision });
    expect(e.snapshot().draft.fields).toMatchObject({
      ...patch,
      recipe_owner_did: null,
      "espresso_params.pressure": null,
    });
  });
  it("preserves details until unknown brewer metadata is available", () => {
    const e = engine();
    e.edit(
      { brewer_rkey: "orea", "pourover_params.bloom_water": 70 },
      { brewerType: "pourover" },
    );
    expect(e.edit({ brewer_rkey: "unknown" }).status).toBe("applied");
    expect(e.snapshot().draft.fields).toHaveProperty(
      "pourover_params.bloom_water",
      70,
    );
  });
  it("keeps manual recipe selection and defaults ahead of a late response", () => {
    const e = engine();
    const t = e.beginRequest("request")!;
    e.edit({ recipe_rkey: "manual", coffee_amount: 20 });
    const r = e.applyResponse(t, {
      recipe_rkey: "late",
      coffee_amount: 18,
      water_amount: 300,
    });
    expect(r.status).toBe("conflict");
    expect(e.snapshot().draft.fields).toEqual({
      recipe_rkey: "manual",
      recipe_owner_did: null,
      coffee_amount: 20,
    });
  });
  it("late standalone defaults cannot overwrite manual recipe choice", () => {
    const e = engine();
    const t = e.beginRequest("request")!;
    e.edit({ recipe_rkey: "manual" });
    e.applyResponse(t, {
      coffee_amount: 18,
      pours: [{ water_amount: 70, time_seconds: 30 }],
    });
    expect(e.snapshot().draft.fields).toEqual({
      recipe_rkey: "manual",
      recipe_owner_did: null,
    });
  });
  it("blocks every old recipe preparation default after a manual recipe change", () => {
    const e = engine();
    const token = e.beginRequest("request")!;
    e.edit({ recipe_rkey: "manual" });
    const old = {
      recipe_rkey: "old",
      coffee_amount: 18,
      water_amount: 300,
      temperature: 94,
      time_seconds: 120,
      grind_size: "5",
      method: "pourover",
      brewer_rkey: "old-brewer",
      pours: [{ water_amount: 70, time_seconds: 30 }],
      "espresso_params.pressure": 9,
      "pourover_params.bloom_water": 70,
    };
    expect(e.applyResponse(token, old, { brewerType: "pourover" }).status).toBe(
      "conflict",
    );
    expect(e.snapshot().draft.fields).toEqual({
      recipe_rkey: "manual",
      recipe_owner_did: null,
    });
  });
  it("blocks old preparation-only responses after a manual recipe change", () => {
    const e = engine();
    const token = e.beginRequest("request")!;
    e.edit({ recipe_rkey: "manual" });
    e.applyResponse(token, {
      temperature: 94,
      time_seconds: 120,
      grind_size: "5",
      method: "pourover",
      "espresso_params.pressure": 9,
    });
    expect(e.snapshot().draft.fields).toEqual({
      recipe_rkey: "manual",
      recipe_owner_did: null,
    });
  });
  it("stale choices require renewed confirmation, even after recipe was cleared", () => {
    const e = engine();
    const t = e.beginRequest("request")!;
    e.applyResponse(t, {});
    e.edit({ recipe_rkey: "manual" });
    e.edit({ recipe_rkey: null });
    const choice = {
      token: t,
      patch: { recipe_rkey: "choice", coffee_amount: 18 },
    };
    const r = e.applyChoice(choice);
    expect(r.status).toBe("needs-confirmation");
    expect(e.applyChoice(choice, { confirmRevision: r.revision }).status).toBe(
      "applied",
    );
    expect(e.snapshot().draft.fields).toMatchObject({
      recipe_rkey: "choice",
      recipe_owner_did: null,
      coffee_amount: 18,
    });
  });
});
