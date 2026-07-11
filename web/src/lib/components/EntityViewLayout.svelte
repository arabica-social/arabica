<script lang="ts">
	import BackButton from "./BackButton.svelte";
	import RecordViewHeader from "./RecordViewHeader.svelte";
	import ActionBar from "./ActionBar.svelte";
	import CommentSection from "./CommentSection.svelte";
	import BacklinksSection from "./BacklinksSection.svelte";
	import type { Snippet } from "svelte";
	import type { BacklinksResult } from "../types/entity_view";
	import { definitionFor } from "$lib/app/definitions";
	import { app } from "$lib/stores/session";

	type Props = {
		recordType: string;
		title: string;
		createdAt: string;
		authorDID?: string;
		authorHandle?: string;
		authorDisplay?: string;
		authorAvatar?: string;
		children: Snippet;
		statLine?: Snippet;
		backlinks?: BacklinksResult | null;
		backlinksDetailURL?: string;
		// ActionBar props
		subjectURI: string;
		subjectCID: string;
		isLiked: boolean;
		likeCount: number;
		commentCount: number;
		shareURL: string;
		isOwner: boolean;
		editURL?: string;
		deleteURL?: string;
		deleteRedirect?: string;
		isAuthenticated: boolean;
		isModerator?: boolean;
		canHideRecord?: boolean;
		canBlockUser?: boolean;
		isRecordHidden?: boolean;
		// Comments
		comments?: import("../types/entity_view").IndexedComment[];
		currentUserDID?: string;
	};

	let {
		recordType,
		title,
		createdAt,
		authorDID = "",
		authorHandle = "",
		authorDisplay = "",
		authorAvatar = "",
		children,
		statLine,
		backlinks = null,
		backlinksDetailURL = "",
		subjectURI,
		subjectCID,
		isLiked,
		likeCount,
		commentCount,
		shareURL,
		isOwner,
		editURL = "",
		deleteURL = "",
		deleteRedirect = "",
		isAuthenticated,
		isModerator = false,
		canHideRecord = false,
		canBlockUser = false,
		isRecordHidden = false,
		comments = [],
		currentUserDID = "",
	}: Props = $props();

	let timestampISO = $derived(new Date(createdAt).toISOString());
	let timestampDisplay = $derived(
		new Date(createdAt).toLocaleDateString(undefined, {
			year: "numeric",
			month: "long",
			day: "numeric",
			hour: "numeric",
			minute: "2-digit",
		}),
	);
	let appDefinition = $derived(definitionFor($app));
</script>

<div class="page-container-sm">
	<div class="card card-inner">
		<RecordViewHeader
			{recordType}
			{title}
			timestamp={timestampDisplay}
			{timestampISO}
			{authorDID}
			{authorHandle}
			authorDisplay={authorDisplay}
			authorAvatar={authorAvatar}
		/>
		{@render children()}
		{#if statLine}
			{@render statLine()}
		{/if}
		{#if backlinks}
			<BacklinksSection result={backlinks} detailURL={backlinksDetailURL} />
		{/if}
		<div class="record-view-footer">
			<BackButton />
			<ActionBar
				{subjectURI}
				{subjectCID}
				{isLiked}
				{likeCount}
				{commentCount}
				{shareURL}
				shareTitle={title}
				shareText={`Check out this ${recordType} on ${appDefinition.displayName}`}
				{isOwner}
				{editURL}
				{deleteURL}
				{deleteRedirect}
				{isAuthenticated}
				{isModerator}
				{canHideRecord}
				{canBlockUser}
				{isRecordHidden}
				{authorDID}
			/>
		</div>
		<CommentSection
			{subjectURI}
			{subjectCID}
			{comments}
			{isAuthenticated}
			currentUserDID={currentUserDID}
			{isModerator}
			viewURL={shareURL}
		/>
	</div>
</div>
