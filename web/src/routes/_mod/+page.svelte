<script lang="ts">
	import { pushToast } from "$lib/stores/toasts";
	import { displayHandle } from "$lib/stores/session";
	import type { PageData } from "./$types";
	import type { AdminResponse, HiddenRecord, BlacklistedUser, EnrichedReport, AuditEntry, Label, SourceStatus } from "$lib/types/api";

	let { data }: { data: PageData } = $props();

	type Tab = "hidden" | "blocked" | "reports" | "activity" | "labels" | "stats" | "cache";
	let activeTab = $state<Tab>("hidden");

	// svelte-ignore state_referenced_locally
	let admin = $state<AdminResponse | null>(data.admin);

	// svelte-ignore state_referenced_locally
	$effect(() => {
		admin = data.admin;
	});

	function fmtTime(iso: string): string {
		const d = new Date(iso);
		if (isNaN(d.getTime())) return "";
		return d.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric", hour: "numeric", minute: "2-digit" });
	}

	function fmtShort(iso: string): string {
		const d = new Date(iso);
		if (isNaN(d.getTime())) return "";
		return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
	}

	function fmtDuration(ns: number): string {
		if (ns <= 0) return "—";
		const d = ns / 1e9;
		if (d < 1) return `${Math.round(d * 1000)}ms`;
		if (d < 60) return `${d.toFixed(1)}s`;
		return `${Math.floor(d / 60)}m${Math.floor(d) % 60}s`;
	}

	function fmtBytes(n: number): string {
		if (n <= 0) return "—";
		const unit = 1024;
		if (n < unit) return `${n} B`;
		let div = unit;
		let exp = 0;
		for (let x = n / unit; x >= unit; x /= unit) { div *= unit; exp++; }
		return `${(n / div).toFixed(1)} ${"KMGTPE"[exp]}iB`;
	}

	function backupHealthy(b: SourceStatus): boolean {
		const lastRun = new Date(b.LastRun).getTime();
		if (isNaN(lastRun) || lastRun === 0) return false;
		const lastSuccess = new Date(b.LastSuccess).getTime();
		const lastFailure = new Date(b.LastFailure).getTime();
		return !isNaN(lastSuccess) && lastSuccess !== 0 && (!lastSuccess || lastSuccess >= lastFailure);
	}

	function collectionLabel(nsid: string): string {
		const labels: Record<string, string> = {
			"social.arabica.alpha.brew": "Brews",
			"social.arabica.alpha.bean": "Beans",
			"social.arabica.alpha.roaster": "Roasters",
			"social.arabica.alpha.grinder": "Grinders",
			"social.arabica.alpha.brewer": "Brewers",
			"social.arabica.alpha.like": "Likes",
			"social.arabica.alpha.comment": "Comments",
		};
		return labels[nsid] ?? nsid;
	}

	async function adminAction(endpoint: string, body: Record<string, string>, confirmMsg: string, successMsg: string) {
		if (!confirm(confirmMsg)) return;
		try {
			const res = await fetch(endpoint, {
				method: "POST",
				credentials: "same-origin",
				headers: {
					"Content-Type": "application/x-www-form-urlencoded",
					Accept: "application/json",
				},
				body: new URLSearchParams(body),
			});
			if (!res.ok) throw new Error(`Failed: ${res.status}`);
			pushToast(successMsg);
			// Reload admin data to reflect the change.
			const adminRes = await fetch("/api/_mod", { headers: { Accept: "application/json" } });
			if (adminRes.ok) admin = (await adminRes.json()) as AdminResponse;
		} catch (error) {
			console.error("Admin action failed:", error);
			pushToast("Action failed");
		}
	}

	function unhide(uri: string) {
		adminAction("/_mod/unhide", { uri }, "Are you sure you want to unhide this record?", "Record unhidden");
	}
	function unblock(did: string) {
		adminAction("/_mod/unblock", { did }, "Are you sure you want to unblock this user? Their content will reappear in the feed.", "User unblocked");
	}
	function hideRecord(uri: string) {
		adminAction("/_mod/hide", { uri, reason: "Reported by user" }, "Hide this record from the feed?", "Record hidden");
	}
	function blockUser(did: string) {
		adminAction("/_mod/block", { did, reason: "Reported by user" }, `Block user ${did}? All their content will be hidden from the feed.`, "User blocked");
	}
	function resetAutoHide(did: string) {
		adminAction("/_mod/reset-autohide", { did }, `Reset the auto-hide report counter for ${did}?`, "Auto-hide counter reset");
	}
	function dismissReport(id: string) {
		adminAction("/_mod/dismiss-report", { id }, "Dismiss this report?", "Report dismissed");
	}
	function removeLabel(entityType: string, entityId: string, label: string) {
		adminAction("/_mod/label/remove", { entity_type: entityType, entity_id: entityId, label }, `Remove label '${label}'?`, "Label removed");
	}

	let showAddLabel = $state(false);

	async function addLabel(e: SubmitEvent) {
		e.preventDefault();
		const form = e.target as HTMLFormElement;
		const formData = new FormData(form);
		try {
			const res = await fetch("/_mod/label/add", {
				method: "POST",
				credentials: "same-origin",
				headers: { Accept: "application/json" },
				body: formData,
			});
			if (!res.ok) throw new Error(`Failed: ${res.status}`);
			pushToast("Label added");
			showAddLabel = false;
			const adminRes = await fetch("/api/_mod", { headers: { Accept: "application/json" } });
			if (adminRes.ok) admin = (await adminRes.json()) as AdminResponse;
		} catch {
			pushToast("Failed to add label");
		}
	}

	// Cache management actions (admin only). These POST to the existing
	// JSON-returning endpoints and show the result inline.
	let cacheActionLoading = $state("");
	let cacheActionResult = $state("");

	async function cacheAction(endpoint: string, body: Record<string, string>, label: string) {
		cacheActionLoading = label;
		cacheActionResult = "";
		try {
			const res = await fetch(endpoint, {
				method: "POST",
				credentials: "same-origin",
				headers: { Accept: "application/json" },
				body: new URLSearchParams(body),
			});
			if (!res.ok) throw new Error(`Failed: ${res.status}`);
			const data = await res.json();
			cacheActionResult = JSON.stringify(data, null, 2);
			pushToast(`${label} complete`);
		} catch (error) {
			console.error(`${label} failed:`, error);
			cacheActionResult = `Error: ${error}`;
			pushToast(`${label} failed`);
		} finally {
			cacheActionLoading = "";
		}
	}

	function refreshHandles() {
		cacheAction("/_mod/refresh-handles", {}, "Refresh handles");
	}

	let rebuildInput = $state("");
	function rebuildDID() {
		if (!rebuildInput.trim()) return;
		cacheAction("/_mod/rebuild", { did: rebuildInput.trim() }, "Rebuild");
	}

	let purgeInput = $state("");
		function purgeDID() {
		if (!purgeInput.trim()) return;
		if (!confirm(`Purge ALL witness data for ${purgeInput.trim()}? This cannot be undone.`)) return;
		cacheAction("/_mod/purge", { did: purgeInput.trim() }, "Purge");
	}
</script>

<svelte:head>
	<title>Moderation Dashboard - Arabica</title>
</svelte:head>

<div class="page-container-lg">
	<div class="mb-8">
		<h1 class="page-title flex items-center gap-2">
			<svg class="w-8 h-8 text-amber-600" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z" />
			</svg>
			Moderation Dashboard
		</h1>
		<p class="text-muted mt-2">Manage hidden content, review reports, and view moderation activity.</p>
	</div>

	{#if data.error}
		<div class="card card-inner text-center py-8">
			<p class="text-secondary mb-4">{data.error}</p>
			<a href="/" class="btn-primary">Back to Home</a>
		</div>
	{:else if admin}
		<div class="space-y-6">
			<!-- Tab nav -->
			<nav class="flex flex-wrap gap-2">
				{#if admin.CanHide || admin.CanUnhide}
					<button type="button" onclick={() => (activeTab = "hidden")} class="px-3 py-1.5 rounded-lg border font-medium text-sm transition-colors {activeTab === 'hidden' ? 'bg-amber-100 border-amber-300 text-amber-800' : 'border-brown-200 text-muted hover:bg-brown-50'}">
						Hidden Records
						{#if admin.HiddenRecords.length > 0}<span class="ml-1.5 bg-brown-200 text-emphasis py-0.5 px-1.5 rounded-full text-xs">{admin.HiddenRecords.length}</span>{/if}
					</button>
				{/if}
				{#if admin.CanBlock || admin.CanUnblock}
					<button type="button" onclick={() => (activeTab = "blocked")} class="px-3 py-1.5 rounded-lg border font-medium text-sm transition-colors {activeTab === 'blocked' ? 'bg-red-100 border-red-300 text-red-800' : 'border-brown-200 text-muted hover:bg-brown-50'}">
						Blocked Users
						{#if admin.BlockedUsers.length > 0}<span class="ml-1.5 bg-red-100 text-red-700 py-0.5 px-1.5 rounded-full text-xs">{admin.BlockedUsers.length}</span>{/if}
					</button>
				{/if}
				{#if admin.CanViewReports}
					<button type="button" onclick={() => (activeTab = "reports")} class="px-3 py-1.5 rounded-lg border font-medium text-sm transition-colors {activeTab === 'reports' ? 'bg-red-100 border-red-300 text-red-800' : 'border-brown-200 text-muted hover:bg-brown-50'}">
						Reports
						{#if admin.Reports.length > 0}<span class="ml-1.5 bg-red-100 text-red-700 py-0.5 px-1.5 rounded-full text-xs">{admin.Reports.length}</span>{/if}
					</button>
				{/if}
				{#if admin.CanViewLogs}
					<button type="button" onclick={() => (activeTab = "activity")} class="px-3 py-1.5 rounded-lg border font-medium text-sm transition-colors {activeTab === 'activity' ? 'bg-brown-200 border-brown-300 text-primary' : 'border-brown-200 text-muted hover:bg-brown-50'}">
						Activity Log
					</button>
				{/if}
				{#if admin.CanManageLabels}
					<button type="button" onclick={() => (activeTab = "labels")} class="px-3 py-1.5 rounded-lg border font-medium text-sm transition-colors {activeTab === 'labels' ? 'bg-purple-100 border-purple-300 text-purple-800' : 'border-brown-200 text-muted hover:bg-brown-50'}">
						Labels
						{#if admin.Labels.length > 0}<span class="ml-1.5 bg-purple-100 text-purple-700 py-0.5 px-1.5 rounded-full text-xs">{admin.Labels.length}</span>{/if}
					</button>
				{/if}
				{#if admin.IsAdmin}
					<button type="button" onclick={() => (activeTab = "stats")} class="px-3 py-1.5 rounded-lg border font-medium text-sm transition-colors {activeTab === 'stats' ? 'bg-brown-200 border-brown-300 text-primary' : 'border-brown-200 text-muted hover:bg-brown-50'}">Stats</button>
					<button type="button" onclick={() => (activeTab = "cache")} class="px-3 py-1.5 rounded-lg border font-medium text-sm transition-colors {activeTab === 'cache' ? 'bg-brown-200 border-brown-300 text-primary' : 'border-brown-200 text-muted hover:bg-brown-50'}">Cache</button>
				{/if}
			</nav>

			<!-- Hidden Records -->
			{#if activeTab === "hidden" && (admin.CanHide || admin.CanUnhide)}
				<div class="card card-inner">
					<h2 class="section-title">Hidden Records</h2>
					{#if admin.HiddenRecords.length === 0}
						<div class="bg-brown-50 rounded-lg p-4 text-center text-muted"><p>No records are currently hidden.</p></div>
					{:else}
						<div class="space-y-3">
							{#each admin.HiddenRecords as record (record.at_uri)}
								<div class="bg-brown-50 border border-brown-200 rounded-lg p-4">
									<div class="flex flex-col gap-3">
										<div>
											<span class="text-xs font-medium text-faint uppercase tracking-wide">Record URI</span>
											<code class="mt-1 block text-sm bg-brown-100 px-2 py-1 rounded-sm break-all font-mono">{record.at_uri}</code>
										</div>
										<div class="flex flex-wrap gap-x-6 gap-y-2 text-sm">
											<div><span class="text-faint">Hidden:</span> <span class="text-emphasis ml-1">{fmtTime(record.hidden_at)}</span></div>
											<div><span class="text-faint">By:</span> <code class="text-emphasis ml-1 text-xs">{record.hidden_by}</code>
												{#if record.auto_hidden}<span class="ml-1 text-xs bg-amber-100 text-amber-700 px-1.5 py-0.5 rounded-sm">auto</span>{/if}
											</div>
											{#if record.reason}<div><span class="text-faint">Reason:</span> <span class="text-emphasis ml-1">{record.reason}</span></div>{/if}
										</div>
										{#if admin.CanUnhide}
											<div class="pt-2 border-t border-brown-200">
												<button class="text-sm text-amber-600 hover:text-amber-800 font-medium" onclick={() => unhide(record.at_uri)}>Unhide Record</button>
											</div>
										{/if}
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			{/if}

			<!-- Blocked Users -->
			{#if activeTab === "blocked" && (admin.CanBlock || admin.CanUnblock)}
				<div class="card card-inner">
					<h2 class="section-title">Blocked Users</h2>
					{#if admin.BlockedUsers.length === 0}
						<div class="bg-brown-50 rounded-lg p-4 text-center text-muted"><p>No users are currently blocked.</p></div>
					{:else}
						<div class="space-y-3">
							{#each admin.BlockedUsers as user (user.did)}
								<div class="bg-brown-50 border border-brown-200 rounded-lg p-4">
									<div class="flex flex-col gap-3">
										<div>
											<span class="text-xs font-medium text-faint uppercase tracking-wide">User DID</span>
											<code class="mt-1 block text-sm bg-brown-100 px-2 py-1 rounded-sm break-all font-mono">{user.did}</code>
										</div>
										<div class="flex flex-wrap gap-x-6 gap-y-2 text-sm">
											<div><span class="text-faint">Blocked:</span> <span class="text-emphasis ml-1">{fmtTime(user.blacklisted_at)}</span></div>
											<div><span class="text-faint">By:</span> <code class="text-emphasis ml-1 text-xs">{user.blacklisted_by}</code></div>
											{#if user.reason}<div><span class="text-faint">Reason:</span> <span class="text-emphasis ml-1">{user.reason}</span></div>{/if}
										</div>
										{#if admin.CanUnblock}
											<div class="pt-2 border-t border-brown-200">
												<button class="text-sm text-amber-600 hover:text-amber-800 font-medium" onclick={() => unblock(user.did)}>Unblock User</button>
											</div>
										{/if}
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			{/if}

			<!-- Reports -->
			{#if activeTab === "reports" && admin.CanViewReports}
				<div class="card card-inner">
					<h2 class="section-title">Pending Reports</h2>
					{#if admin.Reports.length === 0}
						<div class="bg-brown-50 rounded-lg p-4 text-center text-muted"><p>No pending reports to review.</p></div>
					{:else}
						<div class="space-y-4">
							{#each admin.Reports as report (report.Report.id)}
								{@const r = report.Report}
								<div class="bg-brown-50 border border-brown-200 rounded-lg p-4">
									<div class="flex flex-col gap-4">
										<div class="flex items-center justify-between">
											<span class="inline-flex items-center px-2 py-0.5 rounded-sm text-xs font-medium {r.status === 'pending' ? 'bg-yellow-100 text-yellow-800' : r.status === 'dismissed' ? 'bg-gray-100 text-gray-800' : 'bg-green-100 text-green-800'}">{r.status}</span>
											<time class="text-sm text-faint">{fmtTime(r.created_at)}</time>
										</div>
										<div>
											<span class="text-xs font-medium text-faint uppercase tracking-wide">Record URI</span>
											<code class="mt-1 block text-sm bg-brown-100 px-2 py-1 rounded-sm break-all font-mono">{r.subject_uri}</code>
										</div>
										<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
											<div>
												<span class="text-xs font-medium text-faint uppercase tracking-wide">Content Owner</span>
												<div class="mt-1">
													{#if report.OwnerHandle}<a href={`/profile/${report.OwnerHandle}`} class="text-amber-600 hover:text-amber-700 font-medium">@{displayHandle(report.OwnerHandle)}</a>{:else}<code class="text-sm text-emphasis">{r.subject_did}</code>{/if}
												</div>
											</div>
											<div>
												<span class="text-xs font-medium text-faint uppercase tracking-wide">Reported By</span>
												<div class="mt-1">
													{#if report.ReporterHandle}<a href={`/profile/${report.ReporterHandle}`} class="text-amber-600 hover:text-amber-700 font-medium">@{displayHandle(report.ReporterHandle)}</a>{:else}<code class="text-sm text-emphasis">{r.reporter_did}</code>{/if}
												</div>
											</div>
										</div>
										{#if report.PostContent}
											<div>
												<span class="text-xs font-medium text-faint uppercase tracking-wide">Content Preview</span>
												<div class="mt-1 bg-brown-100 rounded-lg p-3"><p class="text-sm text-emphasis whitespace-pre-wrap">{report.PostContent}</p></div>
											</div>
										{/if}
										{#if r.reason}
											<div>
												<span class="text-xs font-medium text-faint uppercase tracking-wide">Report Reason</span>
												<p class="mt-1 text-sm text-emphasis">{r.reason}</p>
											</div>
										{/if}
										{#if r.status === "pending"}
											<div class="pt-3 border-t border-brown-200 flex flex-wrap gap-3">
												{#if admin.CanHide}<button class="text-sm bg-amber-100 text-amber-700 hover:bg-amber-200 px-3 py-1.5 rounded-sm font-medium transition-colors" onclick={() => hideRecord(r.subject_uri)}>Hide Record</button>{/if}
												{#if admin.CanBlock}<button class="text-sm bg-red-100 text-red-700 hover:bg-red-200 px-3 py-1.5 rounded-sm font-medium transition-colors" onclick={() => blockUser(r.subject_did)}>Block User</button>{/if}
												{#if admin.CanResetAutoHide}<button class="text-sm bg-blue-100 text-blue-700 hover:bg-blue-200 px-3 py-1.5 rounded-sm font-medium transition-colors" onclick={() => resetAutoHide(r.subject_did)}>Reset Auto-Hide</button>{/if}
												<button class="text-sm text-muted hover:text-secondary px-3 py-1.5 rounded-sm font-medium transition-colors" onclick={() => dismissReport(r.id)}>Dismiss</button>
											</div>
										{/if}
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			{/if}

			<!-- Activity Log -->
			{#if activeTab === "activity" && admin.CanViewLogs}
				<div class="card card-inner">
					<h2 class="section-title">Recent Activity</h2>
					{#if admin.AuditLog.length === 0}
						<div class="bg-brown-50 rounded-lg p-4 text-center text-muted"><p>No moderation activity recorded yet.</p></div>
					{:else}
						<div class="space-y-3">
							{#each admin.AuditLog as entry (entry.id)}
								<div class="bg-brown-50 border border-brown-200 rounded-lg p-4">
									<div class="flex flex-col gap-3">
										<div class="flex items-center justify-between">
											<span class="inline-flex items-center px-2 py-0.5 rounded-sm text-xs font-medium bg-brown-100 text-secondary">{entry.action}</span>
											<span class="text-sm text-faint">{fmtTime(entry.timestamp)}</span>
										</div>
										{#if entry.details?.email}<div class="flex items-center justify-between"><span class="font-medium text-primary">{entry.details.email}</span></div>{/if}
										{#if entry.details?.ip}<div class="text-sm"><span class="text-faint">IP:</span> <code class="text-emphasis ml-1 text-xs">{entry.details.ip}</code></div>{/if}
										{#if entry.details?.message}<div><span class="text-xs font-medium text-faint uppercase tracking-wide">Message</span><p class="mt-1 text-sm text-emphasis">{entry.details.message}</p></div>{/if}
										{#if entry.target_uri}
											<div>
												<span class="text-xs font-medium text-faint uppercase tracking-wide">Target</span>
												<code class="mt-1 block text-sm bg-brown-100 px-2 py-1 rounded-sm break-all font-mono">{entry.target_uri}</code>
											</div>
										{/if}
										<div class="flex flex-wrap gap-x-6 gap-y-2 text-sm">
											<div><span class="text-faint">Actor:</span> <code class="text-emphasis ml-1 text-xs">{entry.actor_did}</code>
												{#if entry.auto_mod}<span class="ml-1 text-xs bg-amber-100 text-amber-700 px-1.5 py-0.5 rounded-sm">auto</span>{/if}
											</div>
											{#if entry.reason}<div><span class="text-faint">Reason:</span> <span class="text-emphasis ml-1">{entry.reason}</span></div>{/if}
										</div>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			{/if}

			<!-- Labels -->
			{#if activeTab === "labels" && admin.CanManageLabels}
				<div class="card card-inner">
					<div class="flex items-center justify-between mb-4">
						<h2 class="section-title mb-0">Labels</h2>
						<button type="button" onclick={() => (showAddLabel = !showAddLabel)} class="text-sm bg-purple-100 text-purple-700 hover:bg-purple-200 px-3 py-1.5 rounded-sm font-medium transition-colors">Add Label</button>
					</div>
					{#if admin.Labels.length === 0}
						<div class="bg-brown-50 rounded-lg p-4 text-center text-muted"><p>No labels have been applied yet.</p></div>
					{:else}
						<div class="space-y-3">
							{#each admin.Labels as label (label.id)}
								<div class="bg-brown-50 border border-brown-200 rounded-lg p-4">
									<div class="flex flex-col gap-3">
										<div class="flex items-center justify-between">
											<div class="flex items-center gap-2">
												<span class="inline-flex items-center px-2 py-0.5 rounded-sm text-xs font-medium bg-purple-100 text-purple-800">{label.label}</span>
												<span class="text-xs bg-brown-100 text-muted px-1.5 py-0.5 rounded-sm">{label.entity_type}</span>
												{#if label.value}<span class="text-xs text-faint">= {label.value}</span>{/if}
											</div>
											<time class="text-sm text-faint">{fmtShort(label.created_at)}</time>
										</div>
										<div>
											<span class="text-xs font-medium text-faint uppercase tracking-wide">{label.entity_type === "user" ? "User DID" : "Record URI"}</span>
											<code class="mt-1 block text-sm bg-brown-100 px-2 py-1 rounded-sm break-all font-mono">{label.entity_id}</code>
										</div>
										<div class="flex flex-wrap gap-x-6 gap-y-2 text-sm">
											<div><span class="text-faint">By:</span> <code class="text-emphasis ml-1 text-xs">{label.created_by}</code></div>
											<div><span class="text-faint">Expires:</span> <span class="text-emphasis ml-1">{label.expires_at ? fmtTime(label.expires_at) : "Never"}</span></div>
										</div>
										<div class="pt-2 border-t border-brown-200">
											<button class="text-sm text-red-600 hover:text-red-800 font-medium" onclick={() => removeLabel(label.entity_type, label.entity_id, label.label)}>Remove Label</button>
										</div>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>
				{#if showAddLabel}
					<div class="card card-inner mt-4">
						<h3 class="section-title">Add Label</h3>
						<form onsubmit={addLabel} class="space-y-4">
							<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
								<div>
									<label class="block text-sm font-medium text-emphasis mb-1" for="label-entity-type">Entity Type</label>
									<select id="label-entity-type" name="entity_type" required class="w-full px-3 py-2 border border-brown-300 rounded-lg bg-white text-primary text-sm">
										<option value="user">User (DID)</option>
										<option value="record">Record (AT-URI)</option>
									</select>
								</div>
								<div>
									<label class="block text-sm font-medium text-emphasis mb-1" for="label-entity-id">Entity ID</label>
									<input id="label-entity-id" type="text" name="entity_id" required placeholder="did:plc:... or at://..." class="w-full px-3 py-2 border border-brown-300 rounded-lg bg-white text-primary text-sm" />
								</div>
							</div>
							<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
								<div>
									<label class="block text-sm font-medium text-emphasis mb-1" for="label-name">Label</label>
									<input id="label-name" type="text" name="label" required placeholder="e.g. warned, trusted, spam" class="w-full px-3 py-2 border border-brown-300 rounded-lg bg-white text-primary text-sm" />
								</div>
								<div>
									<label class="block text-sm font-medium text-emphasis mb-1" for="label-value">Value (optional)</label>
									<input id="label-value" type="text" name="value" placeholder="Optional value" class="w-full px-3 py-2 border border-brown-300 rounded-lg bg-white text-primary text-sm" />
								</div>
								<div>
									<label class="block text-sm font-medium text-emphasis mb-1" for="label-expires">Expires (optional)</label>
									<select id="label-expires" name="expires" class="w-full px-3 py-2 border border-brown-300 rounded-lg bg-white text-primary text-sm">
										<option value="">Never</option>
										<option value="24h">24 hours</option>
										<option value="168h">7 days</option>
										<option value="720h">30 days</option>
										<option value="2160h">90 days</option>
									</select>
								</div>
							</div>
							<div class="flex gap-3">
								<button type="submit" class="text-sm bg-purple-100 text-purple-700 hover:bg-purple-200 px-4 py-2 rounded-sm font-medium transition-colors">Add Label</button>
								<button type="button" onclick={() => (showAddLabel = false)} class="text-sm text-muted hover:text-secondary px-4 py-2 rounded-sm font-medium transition-colors">Cancel</button>
							</div>
						</form>
					</div>
				{/if}
			{/if}

			<!-- Stats (admin only) -->
			{#if activeTab === "stats" && admin.IsAdmin}
				<div class="card card-inner">
					<h2 class="section-title">System Stats</h2>
					<div class="grid grid-cols-2 md:grid-cols-4 gap-4">
						<div class="bg-brown-50 border border-brown-200 rounded-lg p-4 text-center"><div class="text-2xl font-bold text-primary">{admin.Stats.KnownUsers}</div><div class="text-sm font-medium text-emphasis mt-1">Known Users</div><div class="text-xs text-faint mt-0.5">Unique DIDs indexed</div></div>
						<div class="bg-brown-50 border border-brown-200 rounded-lg p-4 text-center"><div class="text-2xl font-bold text-primary">{admin.Stats.RegisteredUsers}</div><div class="text-sm font-medium text-emphasis mt-1">Registered Users</div><div class="text-xs text-faint mt-0.5">Feed registry</div></div>
						<div class="bg-brown-50 border border-brown-200 rounded-lg p-4 text-center"><div class="text-2xl font-bold text-primary">{admin.Stats.IndexedRecords}</div><div class="text-sm font-medium text-emphasis mt-1">Indexed Records</div><div class="text-xs text-faint mt-0.5">Total records</div></div>
						<div class="bg-brown-50 border border-brown-200 rounded-lg p-4 text-center"><div class="text-2xl font-bold text-primary">{admin.Stats.TotalLikes}</div><div class="text-sm font-medium text-emphasis mt-1">Total Likes</div><div class="text-xs text-faint mt-0.5">Across all records</div></div>
						<div class="bg-brown-50 border border-brown-200 rounded-lg p-4 text-center"><div class="text-2xl font-bold text-primary">{admin.Stats.TotalComments}</div><div class="text-sm font-medium text-emphasis mt-1">Total Comments</div><div class="text-xs text-faint mt-0.5">Across all records</div></div>
						<div class="rounded-lg border p-4 text-center {admin.Stats.FirehoseConnected ? 'bg-brown-50 border-brown-200' : 'bg-red-50 border-red-200'}">
							<div class="text-2xl font-bold {admin.Stats.FirehoseConnected ? 'text-primary' : 'text-red-700'}">{admin.Stats.FirehoseConnected ? "Connected" : "Disconnected"}</div>
							<div class="text-sm font-medium {admin.Stats.FirehoseConnected ? 'text-emphasis' : 'text-red-600'} mt-1">Firehose</div>
							<div class="text-xs {admin.Stats.FirehoseConnected ? 'text-faint' : 'text-red-500'} mt-0.5">{admin.Stats.FirehoseConnected ? "Real-time events" : "Not receiving events"}</div>
						</div>
					</div>
					{#if Object.keys(admin.Stats.RecordsByCollection).length > 0}
						<h3 class="section-title mt-6">Records by Collection</h3>
						<div class="grid grid-cols-2 md:grid-cols-3 gap-4">
							{#each Object.entries(admin.Stats.RecordsByCollection) as [collection, count]}
								<div class="bg-brown-50 border border-brown-200 rounded-lg p-4 text-center"><div class="text-2xl font-bold text-primary">{count}</div><div class="text-sm font-medium text-emphasis mt-1">{collectionLabel(collection)}</div></div>
							{/each}
						</div>
					{/if}
				</div>
				<!-- Backups -->
				<div class="card card-inner mt-4">
					<h2 class="section-title">Database Backups</h2>
					{#if admin.Backups.length === 0}
						<p class="text-sm text-muted">No backup sources are configured. Set a writable backup directory to enable automated backups.</p>
					{:else}
						<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
							{#each admin.Backups as b (b.Source)}
								<div class="rounded-lg border p-4 {backupHealthy(b) ? 'bg-brown-50 border-brown-200' : 'bg-red-50 border-red-200'}">
									<div class="flex items-center justify-between">
										<div class="font-semibold text-emphasis">{b.Source}</div>
										<span class="px-2 py-0.5 rounded-full text-xs font-medium {new Date(b.LastRun).getTime() === 0 ? 'bg-brown-100 text-muted' : backupHealthy(b) ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}">
											{new Date(b.LastRun).getTime() === 0 ? "Pending" : backupHealthy(b) ? "Healthy" : "Failed"}
										</span>
									</div>
									<dl class="mt-3 grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
										<dt class="text-faint">Last run</dt><dd class="text-emphasis">{new Date(b.LastRun).getTime() ? fmtShort(b.LastRun) : "Never"}</dd>
										<dt class="text-faint">Duration</dt><dd class="text-emphasis">{fmtDuration(b.LastDuration)}</dd>
										<dt class="text-faint">Size</dt><dd class="text-emphasis">{fmtBytes(b.LastSize)}</dd>
										<dt class="text-faint">Retained on disk</dt><dd class="text-emphasis">{b.RetainedCount}</dd>
										<dt class="text-faint">Next run</dt><dd class="text-emphasis">{new Date(b.NextRun).getTime() ? fmtShort(b.NextRun) : "—"}</dd>
									</dl>
									{#if !backupHealthy(b) && b.LastError}<div class="mt-3 text-xs text-red-700 bg-red-50 border border-red-200 rounded p-2 break-words">{b.LastError}</div>{/if}
								</div>
							{/each}
						</div>
					{/if}
				</div>
			{/if}

			<!-- Cache (admin only) -->
			{#if activeTab === "cache" && admin.IsAdmin}
				<div class="space-y-4">
					<div class="card card-inner">
						<h2 class="section-title">Export Witness Cache</h2>
						<p class="text-sm text-muted mb-4">Download every witness-cached record for a DID as a single JSON file. Reads from the local firehose index — does not contact the user's PDS.</p>
						<form action="/_mod/export" method="get" class="flex flex-col gap-3 sm:flex-row sm:items-end">
							<div class="flex-1">
								<label for="export-did" class="block text-sm font-medium text-emphasis mb-1">DID</label>
								<input id="export-did" type="text" name="did" required placeholder="did:plc:..." class="w-full px-3 py-2 border border-brown-300 rounded-lg bg-white text-primary text-sm font-mono" />
							</div>
							<button type="submit" class="text-sm bg-brown-300 text-primary hover:bg-brown-400 px-4 py-2 rounded-sm font-medium transition-colors">Export JSON</button>
						</form>
					</div>
					<div class="card card-inner">
						<h2 class="section-title">Fetch PDS Records</h2>
						<p class="text-sm text-muted mb-4">Fetch every Arabica record for a user directly from their PDS, bypassing the witness cache. Accepts a DID or a handle.</p>
						<form action="/_mod/pds-records" method="get" class="flex flex-col gap-3 sm:flex-row sm:items-end">
							<div class="flex-1">
								<label for="pds-actor" class="block text-sm font-medium text-emphasis mb-1">DID or handle</label>
								<input id="pds-actor" type="text" name="actor" required placeholder="did:plc:... or alice.example.com" class="w-full px-3 py-2 border border-brown-300 rounded-lg bg-white text-primary text-sm font-mono" />
							</div>
							<button type="submit" class="text-sm bg-brown-300 text-primary hover:bg-brown-400 px-4 py-2 rounded-sm font-medium transition-colors">Fetch JSON</button>
						</form>
					</div>
					<div class="card card-inner">
						<h2 class="section-title">Refresh All Handles</h2>
						<p class="text-sm text-muted mb-4">Re-fetch every cached profile from the AppView so stale handles get corrected. A less-destructive alternative to purge+rebuild.</p>
						<button type="button" class="text-sm bg-brown-300 text-primary hover:bg-brown-400 px-4 py-2 rounded-sm font-medium transition-colors" disabled={cacheActionLoading === "Refresh handles"} onclick={refreshHandles}>
							{cacheActionLoading === "Refresh handles" ? "Refreshing..." : "Refresh All Handles"}
						</button>
					</div>
					<div class="card card-inner">
						<h2 class="section-title">Rebuild Witness Cache from PDS</h2>
						<p class="text-sm text-muted mb-4">Re-pull every record for a user from their PDS into the witness cache. Pair with purge to fully recycle a user's witness data. Accepts a DID or handle.</p>
						<div class="flex flex-col gap-3 sm:flex-row sm:items-end">
							<div class="flex-1">
								<label for="rebuild-did" class="block text-sm font-medium text-emphasis mb-1">DID or handle</label>
								<input id="rebuild-did" type="text" bind:value={rebuildInput} placeholder="did:plc:... or alice.example.com" class="w-full px-3 py-2 border border-brown-300 rounded-lg bg-white text-primary text-sm font-mono" />
							</div>
							<button type="button" class="text-sm bg-brown-300 text-primary hover:bg-brown-400 px-4 py-2 rounded-sm font-medium transition-colors" disabled={cacheActionLoading === "Rebuild" || !rebuildInput.trim()} onclick={rebuildDID}>
								{cacheActionLoading === "Rebuild" ? "Rebuilding..." : "Rebuild"}
							</button>
						</div>
					</div>
					<div class="card card-inner">
						<h2 class="section-title text-red-900">Purge DID from Witness Cache</h2>
						<p class="text-sm text-muted mb-4">Remove ALL witness data for a user. This clears their records, profiles, and backfill marker. Pair with rebuild to fully recycle. Accepts a DID.</p>
						<div class="flex flex-col gap-3 sm:flex-row sm:items-end">
							<div class="flex-1">
								<label for="purge-did" class="block text-sm font-medium text-emphasis mb-1">DID</label>
								<input id="purge-did" type="text" bind:value={purgeInput} placeholder="did:plc:..." class="w-full px-3 py-2 border border-brown-300 rounded-lg bg-white text-primary text-sm font-mono" />
							</div>
							<button type="button" class="text-sm bg-red-600 text-white hover:bg-red-700 px-4 py-2 rounded-sm font-medium transition-colors" disabled={cacheActionLoading === "Purge" || !purgeInput.trim()} onclick={purgeDID}>
								{cacheActionLoading === "Purge" ? "Purging..." : "Purge"}
							</button>
						</div>
					</div>
					{#if cacheActionResult}
						<div class="card card-inner">
							<h2 class="section-title">Result</h2>
							<pre class="text-sm text-emphasis font-mono whitespace-pre-wrap break-all">{cacheActionResult}</pre>
						</div>
					{/if}
				</div>
			{/if}
		</div>
	{/if}
</div>
