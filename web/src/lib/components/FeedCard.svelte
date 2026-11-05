<script lang="ts">
  import type { Snippet } from "svelte";
  import ActionBar from "./ActionBar.svelte";
  import PourStrip from "./PourStrip.svelte";
  import BrewParams from "./BrewParams.svelte";
  import Avatar from "./Avatar.svelte";
  import TypeBadge from "./TypeBadge.svelte";
  import Icon from "./Icon.svelte";
  import { safeWebsiteURL } from "../utils/format";
  import { goto } from "$app/navigation";
  import { displayHandle } from "../stores/session";
  import type { FeedItem } from "../types/feed";
  import type {
    Bean,
    Brew,
    Brewer,
    Grinder,
    Recipe,
    Roaster,
  } from "../types/entity_view";

  type Props = {
    item: FeedItem;
    isAuthenticated: boolean;
    footer?: Snippet;
  };

  let { item, isAuthenticated, footer }: Props = $props();

  function shareURL(i: FeedItem): string {
    const actor = i.author?.handle || i.author?.did || "";
    const rkey = rkeyOf(i);
    if (!actor || !rkey) return `/profile/${i.author?.did ?? ""}`;
    return `/${nounPath(i.record_type)}/${actor}/${rkey}`;
  }

  function nounPath(rt: string): string {
    switch (rt) {
      case "brew":
        return "brews";
      case "bean":
        return "beans";
      case "roaster":
        return "roasters";
      case "grinder":
        return "grinders";
      case "brewer":
        return "brewers";
      case "recipe":
        return "recipes";
      default:
        return "";
    }
  }

  function rkeyOf(i: FeedItem): string {
    const r = i.record as { rkey?: string };
    return r?.rkey ?? "";
  }

  function cardClass(): string {
    const noun = item.record_type;
    const compact = ["roaster", "grinder", "brewer"].includes(noun)
      ? " feed-card-compact"
      : "";
    return `feed-card feed-card-${noun}${compact}`;
  }

  // Per-entity typed records (derived so the template can narrow safely).
  let brew = $derived(
    item.record_type === "brew" ? (item.record as Brew) : null,
  );
  let bean = $derived(
    item.record_type === "bean" ? (item.record as Bean) : null,
  );
  let roaster = $derived(
    item.record_type === "roaster" ? (item.record as Roaster) : null,
  );
  let grinder = $derived(
    item.record_type === "grinder" ? (item.record as Grinder) : null,
  );
  let brewer = $derived(
    item.record_type === "brewer" ? (item.record as Brewer) : null,
  );
  let recipe = $derived(
    item.record_type === "recipe" ? (item.record as Recipe) : null,
  );
  function actionNoun(): string {
    return item.record_type;
  }

  function shareTitle(): string {
    if (brew?.bean) return brew.bean.name || brew.bean.origin || "Brew";
    if (brew) return "Brew";
    if (bean) return bean.name || bean.origin || "Bean";
    if (roaster) return roaster.name || "Roaster";
    if (grinder) return grinder.name || "Grinder";
    if (brewer) return brewer.name || "Brewer";
    if (recipe) return recipe.name || "Recipe";
    return "Arabica";
  }

  let url = $derived(shareURL(item));
  let title = $derived(shareTitle());
  let editURL = $derived(
    item.is_owner ? `/${nounPath(item.record_type)}/${rkeyOf(item)}/edit` : "",
  );
  let deleteURL = $derived(
    item.is_owner
      ? item.record_type === "brew"
        ? `/${nounPath(item.record_type)}/${rkeyOf(item)}`
        : `/api/${nounPath(item.record_type)}/${rkeyOf(item)}`
      : "",
  );

  function shortDID(did: string): string {
    if (!did.startsWith("did:")) return did;
    return `${did.slice(0, 11)}…${did.slice(-4)}`;
  }
</script>

<div class={cardClass()}>
  <div class="mb-3">
    <div class="flex items-center gap-2">
      <a href={`/profile/${item.author.did}`}>
        <Avatar
          avatarURL={item.author.avatar}
          displayName={item.author.display_name || item.author.handle}
          size="md"
        />
      </a>
      <div class="flex-1 min-w-0">
        <a
          href={`/profile/${item.author.did}`}
          class="font-medium text-primary hover:underline text-sm block truncate"
          title={(
            item.author.display_name ||
            item.author.handle ||
            ""
          ).startsWith("did:")
            ? item.author.display_name || item.author.handle
            : undefined}
        >
          {shortDID(
            item.author.display_name || displayHandle(item.author.handle),
          )}
        </a>
        <div class="text-xs text-muted truncate">
          {#if item.author.display_name && !item.author.display_name.startsWith("did:") && displayHandle(item.author.handle) !== item.author.display_name}@{displayHandle(
              item.author.handle,
            )} ·
          {/if}{item.time_ago}
        </div>
      </div>
    </div>
  </div>

  <div class="mb-2 flex items-center justify-between gap-2">
    <div class="text-sm text-emphasis">
      added a
      <a href={url} class="underline hover:text-primary">new {actionNoun()}</a>
      <TypeBadge recordType={actionNoun()} />
    </div>
    {#if brew && brew.rating > 0}
      <span class="badge-rating flex items-center gap-1">
        <Icon name="star" class="w-3 h-3 text-amber-500" />
        {brew.rating}/10
      </span>
    {/if}
  </div>

  <div
    role="link"
    tabindex="0"
    class="block hover:opacity-90 transition-opacity cursor-pointer"
    onclick={() => goto(url)}
    onkeydown={(e) => {
      if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        goto(url);
      }
    }}
  >
    {#if brew}
      <div class="feed-content-box">
        <div class="mb-3">
          {#if brew.bean}
            <div class="record-title">
              {brew.bean.name || brew.bean.origin}
            </div>
            {#if brew.bean.roaster?.name}
              <div class="text-sm text-emphasis mt-0.5 flex items-center gap-1">
                <Icon name="store" class="w-3 h-3" />
                {brew.bean.roaster.name}
              </div>
            {/if}
            <div class="text-xs text-muted mt-1 flex flex-wrap gap-x-2 gap-y-1">
              {#if brew.bean.origin}<span class="flex items-center gap-1"
                  ><Icon name="mapPin" class="w-3 h-3" />{brew.bean
                    .origin}</span
                >{/if}
              {#if brew.bean.roast_level}<span class="flex items-center gap-1"
                  ><Icon name="flame" class="w-3 h-3" />{brew.bean
                    .roast_level}</span
                >{/if}
            </div>
          {:else}
            <div class="record-title">Brew</div>
          {/if}
        </div>
        <BrewParams {brew} />
        <PourStrip pours={brew.pours} />
        {#if brew.tasting_notes}
          <div
            class="mt-3 text-sm text-secondary italic border-t border-brown-200 pt-2"
          >
            "{brew.tasting_notes}"
          </div>
        {/if}
      </div>
    {:else if bean}
      <div class="feed-content-box-sm">
        <div class="record-title">{bean.name || bean.origin}</div>
        {#if bean.roaster?.name}
          <div class="text-sm text-emphasis mt-0.5 flex items-center gap-1">
            <Icon name="store" class="w-3 h-3" />{bean.roaster.name}
          </div>
        {/if}
        <div class="text-xs text-muted mt-1 flex flex-wrap gap-x-2 gap-y-1">
          {#if bean.origin}<span class="flex items-center gap-1"
              ><Icon name="mapPin" class="w-3 h-3" />{bean.origin}</span
            >{/if}
          {#if bean.roast_level}<span class="flex items-center gap-1"
              ><Icon name="flame" class="w-3 h-3" />{bean.roast_level}</span
            >{/if}
          {#if bean.variety}<span class="flex items-center gap-1"
              ><Icon name="leaf" class="w-3 h-3" />{bean.variety}</span
            >{/if}
          {#if bean.process}<span class="flex items-center gap-1"
              ><Icon name="sprout" class="w-3 h-3" />{bean.process}</span
            >{/if}
          {#if bean.rating}<span class="flex items-center gap-1"
              ><Icon
                name="star"
                class="w-3 h-3 text-amber-500"
              />{bean.rating}/10</span
            >{/if}
        </div>
        {#if bean.description}
          <div class="mt-2 text-sm text-secondary italic line-clamp-2">
            "{bean.description}"
          </div>
        {/if}
        {#if bean.link && safeWebsiteURL(bean.link)}
          <div class="mt-2 text-xs text-muted flex items-center gap-1">
            <Icon name="link" class="w-3 h-3" />
            <a
              href={safeWebsiteURL(bean.link)}
              target="_blank"
              rel="noopener noreferrer"
              class="text-secondary hover:underline"
              >{safeWebsiteURL(bean.link)}</a
            >
          </div>
        {/if}
      </div>
    {:else if roaster}
      <div class="feed-content-box-sm">
        <div class="record-title">{roaster.name}</div>
        <div class="text-xs text-muted mt-1 flex flex-wrap gap-x-2 gap-y-1">
          {#if roaster.location}<span class="flex items-center gap-1"
              ><Icon name="mapPin" class="w-3 h-3" />{roaster.location}</span
            >{/if}
          {#if roaster.website && safeWebsiteURL(roaster.website)}
            <span class="flex items-center gap-1"
              ><Icon name="globe" class="w-3 h-3" /><a
                href={safeWebsiteURL(roaster.website)}
                target="_blank"
                rel="noopener noreferrer"
                class="text-secondary hover:underline"
                >{safeWebsiteURL(roaster.website)}</a
              ></span
            >
          {/if}
        </div>
      </div>
    {:else if grinder}
      <div class="feed-content-box-sm">
        <div class="record-title">{grinder.name}</div>
        <div class="text-xs text-muted mt-1 flex flex-wrap gap-x-2 gap-y-1">
          {#if grinder.grinder_type}<span class="flex items-center gap-1"
              ><Icon name="tag" class="w-3 h-3" />{grinder.grinder_type}</span
            >{/if}
          {#if grinder.burr_type}<span class="flex items-center gap-1"
              ><Icon name="disc" class="w-3 h-3" />{grinder.burr_type}</span
            >{/if}
        </div>
        {#if grinder.notes}<div class="mt-2 text-sm text-secondary italic">
            "{grinder.notes}"
          </div>{/if}
      </div>
    {:else if brewer}
      <div class="feed-content-box-sm">
        <div class="record-title">{brewer.name}</div>
        {#if brewer.brewer_type}
          <div class="text-xs text-muted mt-1">
            <span class="flex items-center gap-1"
              ><Icon name="brewer" class="w-3 h-3" />{brewer.brewer_type}</span
            >
          </div>
        {/if}
        {#if brewer.description}<div class="mt-2 text-sm text-secondary italic">
            "{brewer.description}"
          </div>{/if}
      </div>
    {:else if recipe}
      <div class="feed-content-box-sm">
        <div class="text-base mb-2">
          <span class="record-title">{recipe.name}</span>
        </div>
        <div class="text-xs text-muted mt-1 flex flex-wrap gap-x-2 gap-y-1">
          {#if recipe.coffee_amount > 0}<span class="flex items-center gap-1"
              ><Icon
                name="coffee"
                class="w-3 h-3"
              />{recipe.coffee_amount.toFixed(1)}g</span
            >{/if}
          {#if recipe.water_amount > 0}<span class="flex items-center gap-1"
              ><Icon
                name="droplet"
                class="w-3 h-3"
              />{recipe.water_amount.toFixed(1)}g</span
            >{/if}
          {#if recipe.brewer_obj?.name}<span class="flex items-center gap-1"
              ><Icon name="brewer" class="w-3 h-3" />{recipe.brewer_obj
                .name}</span
            >{/if}
          {#if !recipe.brewer_obj?.name && recipe.brewer_type}<span
              class="flex items-center gap-1"
              ><Icon name="brewer" class="w-3 h-3" />{recipe.brewer_type}</span
            >{/if}
        </div>
        {#if recipe.notes}<div class="mt-2 text-sm text-secondary italic">
            "{recipe.notes}"
          </div>{/if}
      </div>
    {/if}
  </div>

  {#if item.subject_uri && item.subject_cid}
    <ActionBar
      subjectURI={item.subject_uri}
      subjectCID={item.subject_cid}
      isLiked={item.is_liked_by_viewer}
      likeCount={item.like_count}
      commentCount={item.comment_count}
      shareURL={url}
      shareTitle={title}
      shareText={`Check out this ${actionNoun()} by ${item.author.display_name || displayHandle(item.author.handle)} on Arabica`}
      isOwner={item.is_owner}
      {editURL}
      {deleteURL}
      deleteRedirect="/my-coffee"
      viewURL={url}
      {isAuthenticated}
      isModerator={item.is_moderator}
      canHideRecord={item.can_hide_record}
      canBlockUser={item.can_block_user}
      isRecordHidden={item.is_record_hidden}
      authorDID={item.author_did}
    />
  {/if}

  {#if footer}
    <div class="mt-1">
      {@render footer()}
    </div>
  {/if}
</div>
