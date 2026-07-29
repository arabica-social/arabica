import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import AdminPage from "../../src/routes/_mod/+page.svelte";

vi.mock("$app/navigation", () => ({
	goto: vi.fn(),
}));

const adminData = {
	admin: {
		HiddenRecords: [
			{ at_uri: "at://did:plc:abc/social.arabica.alpha.brew/xyz", hidden_at: "2026-01-01T00:00:00Z", hidden_by: "did:plc:mod", reason: "spam", auto_hidden: false },
		],
		BlockedUsers: [
			{ did: "did:plc:bad", blacklisted_at: "2026-01-01T00:00:00Z", blacklisted_by: "did:plc:mod", reason: "harassment" },
		],
		Reports: [
			{
				Report: { id: "r1", subject_uri: "at://did:plc:bad/social.arabica.alpha.brew/1", subject_did: "did:plc:bad", reporter_did: "did:plc:user", reason: "spam", created_at: "2026-01-01T00:00:00Z", status: "pending" },
				OwnerHandle: "bad.test",
				ReporterHandle: "user.test",
				PostContent: "spam content",
			},
		],
		AuditLog: [
			{ id: "a1", action: "hide_record", actor_did: "did:plc:mod", target_uri: "at://did:plc:abc/brew/1", reason: "", timestamp: "2026-01-01T00:00:00Z", auto_mod: false },
		],
		Labels: [
			{ id: "l1", entity_type: "user", entity_id: "did:plc:abc", label: "warned", value: "", created_at: "2026-01-01T00:00:00Z", created_by: "did:plc:mod" },
		],
		Stats: { KnownUsers: 42, RegisteredUsers: 38, IndexedRecords: 500, TotalLikes: 100, TotalComments: 30, FirehoseConnected: true, RecordsByCollection: { "social.arabica.alpha.brew": 200 } },
		Backups: [{ Source: "main", LastRun: "2026-01-01T00:00:00Z", LastSuccess: "2026-01-01T00:00:00Z", LastFailure: "", LastError: "", LastDuration: 500000000, LastSize: 1024, RetainedCount: 5, NextRun: "" }],
		CanHide: true, CanUnhide: true, CanViewLogs: true, CanViewReports: true,
		CanBlock: true, CanUnblock: true, CanResetAutoHide: true, CanManageLabels: true,
		IsAdmin: true,
	},
	error: "",
};

describe("Admin dashboard", () => {
	afterEach(() => {
		cleanup();
		document.body.innerHTML = "";
	});

	it("renders the dashboard title", () => {
		render(AdminPage, { data: adminData });
		expect(screen.getByText("Moderation Dashboard")).toBeTruthy();
	});

	it("renders all tab buttons", () => {
		render(AdminPage, { data: adminData });
		expect(screen.getAllByText("Hidden Records").length).toBeGreaterThan(0);
		expect(screen.getAllByText("Blocked Users").length).toBeGreaterThan(0);
		expect(screen.getAllByText("Reports").length).toBeGreaterThan(0);
		expect(screen.getAllByText("Activity Log").length).toBeGreaterThan(0);
		expect(screen.getAllByText("Labels").length).toBeGreaterThan(0);
		expect(screen.getAllByText("Stats").length).toBeGreaterThan(0);
		expect(screen.getAllByText("Cache").length).toBeGreaterThan(0);
	});

	it("shows hidden records count badge", () => {
		render(AdminPage, { data: adminData });
		const badges = screen.getAllByText("1");
		expect(badges.length).toBeGreaterThan(0);
	});

	it("renders the hidden records tab by default", () => {
		render(AdminPage, { data: adminData });
		expect(screen.getByText("at://did:plc:abc/social.arabica.alpha.brew/xyz")).toBeTruthy();
		expect(screen.getByText("Unhide Record")).toBeTruthy();
	});

	it("renders error state", () => {
		render(AdminPage, { data: { admin: null, error: "Authentication required" } });
		expect(screen.getByText("Authentication required")).toBeTruthy();
	});

	it("renders forbidden state", () => {
		render(AdminPage, { data: { admin: null, error: "You don't have permission to access the moderation dashboard." } });
		expect(screen.getByText(/permission/)).toBeTruthy();
	});
});
