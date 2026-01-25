<script>
  export let item;
  import { navigate } from "../lib/router.js";

  function safeAvatarURL(url) {
    if (!url) return null;
    if (url.startsWith("https://") || url.startsWith("/static/")) {
      return url;
    }
    return null;
  }

  function hasValue(val) {
    return val !== null && val !== undefined && val !== "";
  }

  function formatTemperature(temp) {
    if (!hasValue(temp)) return null;
    const unit = temp <= 100 ? "C" : "F";
    return `${temp}°${unit}`;
  }

  function formatTime(seconds) {
    if (!hasValue(seconds)) return null;
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    if (mins > 0) {
      return `${mins}:${secs.toString().padStart(2, "0")}`;
    }
    return `${seconds}s`;
  }

  function safeWebsiteURL(url) {
    if (!url) return null;
    if (url.startsWith("https://") || url.startsWith("http://")) {
      return url;
    }
    return null;
  }
</script>

<div
  class="bg-gradient-to-br from-brown-50 to-brown-100 rounded-lg shadow-md border border-brown-200 p-3 md:p-4 hover:shadow-lg transition-shadow"
>
  <!-- Author row -->
  <div class="flex items-center gap-3 mb-3">
    <a
      href="/profile/{item.Author.handle}"
      on:click|preventDefault={() => navigate(`/profile/${item.Author.handle}`)}
      class="flex-shrink-0"
    >
      {#if safeAvatarURL(item.Author.avatar)}
        <img
          src={safeAvatarURL(item.Author.avatar)}
          alt=""
          class="w-10 h-10 rounded-full object-cover hover:ring-2 hover:ring-brown-600 transition"
        />
      {:else}
        <div
          class="w-10 h-10 rounded-full bg-brown-300 flex items-center justify-center hover:ring-2 hover:ring-brown-600 transition"
        >
          <span class="text-brown-600 text-sm">?</span>
        </div>
      {/if}
    </a>
    <div class="flex-1 min-w-0">
      <div class="flex items-center gap-2">
        {#if item.Author.displayName}
          <a
            href="/profile/{item.Author.handle}"
            on:click|preventDefault={() =>
              navigate(`/profile/${item.Author.handle}`)}
            class="font-medium text-brown-900 truncate hover:text-brown-700 hover:underline"
            >{item.Author.displayName}</a
          >
        {/if}
        <a
          href="/profile/{item.Author.handle}"
          on:click|preventDefault={() =>
            navigate(`/profile/${item.Author.handle}`)}
          class="text-brown-600 text-sm truncate hover:text-brown-700 hover:underline"
          >@{item.Author.handle}</a
        >
      </div>
      <span class="text-brown-500 text-sm">{item.TimeAgo}</span>
    </div>
  </div>

  <!-- Action header -->
  <div class="mb-2 text-sm text-brown-700">
    {#if item.RecordType === "brew" && item.Brew}
      <span>added a </span>
      <a
        href="/brews/{item.Author.did}/{item.Brew.rkey}"
        on:click|preventDefault={() =>
          navigate(`/brews/${item.Author.did}/${item.Brew.rkey}`)}
        class="font-semibold text-brown-800 hover:text-brown-900 hover:underline cursor-pointer"
      >
        new brew
      </a>
    {:else}
      {item.Action}
    {/if}
  </div>

  <!-- Record content -->
  {#if item.RecordType === "brew" && item.Brew}
    <div
      class="bg-white/60 backdrop-blur rounded-lg p-3 md:p-4 border border-brown-200"
    >
      <!-- Bean info with rating -->
      <div class="flex items-start justify-between gap-3 mb-3">
        <div class="flex-1 min-w-0">
          {#if item.Brew.bean}
            <div class="font-bold text-brown-900 text-base">
              {item.Brew.bean.name || item.Brew.bean.origin}
            </div>
            {#if item.Brew.bean.roaster?.name}
              <div class="text-sm text-brown-700 mt-0.5">
                <span class="font-medium">🏭 {item.Brew.bean.roaster.name}</span
                >
              </div>
            {/if}
            <div
              class="text-xs text-brown-600 mt-1 flex flex-wrap gap-x-2 gap-y-0.5"
            >
              {#if item.Brew.bean.origin}<span
                  class="inline-flex items-center gap-0.5"
                  >📍 {item.Brew.bean.origin}</span
                >{/if}
              {#if item.Brew.bean.roast_level}<span
                  class="inline-flex items-center gap-0.5"
                  >🔥 {item.Brew.bean.roast_level}</span
                >{/if}
              {#if item.Brew.bean.process}<span
                  class="inline-flex items-center gap-0.5"
                  >🌱 {item.Brew.bean.process}</span
                >{/if}
              {#if hasValue(item.Brew.coffee_amount)}<span
                  class="inline-flex items-center gap-0.5"
                  >⚖️ {item.Brew.coffee_amount}g</span
                >{/if}
            </div>
          {/if}
        </div>
        {#if hasValue(item.Brew.rating)}
          <span
            class="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-amber-100 text-amber-900 flex-shrink-0"
          >
            ⭐ {item.Brew.rating}/10
          </span>
        {/if}
      </div>

      <!-- Brewer -->
      {#if item.Brew.brewer_obj || item.Brew.method}
        <div class="mb-2">
          <span class="text-xs text-brown-600">Brewer:</span>
          <span class="text-sm font-semibold text-brown-900">
            {item.Brew.brewer_obj?.name || item.Brew.method}
          </span>
        </div>
      {/if}

      <!-- Brew parameters in compact grid -->
      <div class="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-brown-700">
        {#if item.Brew.grinder_obj}
          <div>
            <span class="text-brown-600">Grinder:</span>
            {item.Brew.grinder_obj.name}{#if item.Brew.grind_size}
              ({item.Brew.grind_size}){/if}
          </div>
        {:else if item.Brew.grind_size}
          <div>
            <span class="text-brown-600">Grind:</span>
            {item.Brew.grind_size}
          </div>
        {/if}
        {#if item.Brew.pours && item.Brew.pours.length > 0}
          <div class="col-span-2">
            <span class="text-brown-600">Pours:</span>
            {#each item.Brew.pours as pour}
              <div class="pl-2 text-brown-600">
                • {pour.water_amount}g @ {formatTime(pour.time_seconds)}
              </div>
            {/each}
          </div>
        {:else if hasValue(item.Brew.water_amount)}
          <div>
            <span class="text-brown-600">Water:</span>
            {item.Brew.water_amount}g
          </div>
        {/if}
        {#if formatTemperature(item.Brew.temperature)}
          <div>
            <span class="text-brown-600">Temp:</span>
            {formatTemperature(item.Brew.temperature)}
          </div>
        {/if}
        {#if hasValue(item.Brew.time_seconds)}
          <div>
            <span class="text-brown-600">Time:</span>
            {formatTime(item.Brew.time_seconds)}
          </div>
        {/if}
      </div>

      <!-- Notes -->
      {#if item.Brew.tasting_notes}
        <div
          class="mt-3 text-sm text-brown-800 italic border-t border-brown-200 pt-2"
        >
          "{item.Brew.tasting_notes}"
        </div>
      {/if}

      <!-- View button -->
      <div class="mt-3 border-t border-brown-200 pt-3">
        <a
          href="/brews/{item.Author.did}/{item.Brew.rkey}"
          on:click|preventDefault={() =>
            navigate(`/brews/${item.Author.did}/${item.Brew.rkey}`)}
          class="inline-flex items-center text-sm font-medium text-brown-700 hover:text-brown-900 hover:underline"
        >
          View full details →
        </a>
      </div>
    </div>
  {:else if item.RecordType === "bean" && item.Bean}
    <div
      class="bg-white/60 backdrop-blur rounded-lg p-3 border border-brown-200"
    >
      <div class="text-base mb-2">
        <span class="font-bold text-brown-900">
          {item.Bean.name || item.Bean.origin}
        </span>
        {#if item.Bean.roaster?.name}
          <span class="text-brown-700"> from {item.Bean.roaster.name}</span>
        {/if}
      </div>
      <div class="text-sm text-brown-700 space-y-1">
        {#if item.Bean.origin}
          <div><span class="text-brown-600">Origin:</span> {item.Bean.origin}</div>
        {/if}
        {#if item.Bean.roast_level}
          <div>
            <span class="text-brown-600">Roast:</span>
            {item.Bean.roast_level}
          </div>
        {/if}
        {#if item.Bean.process}
          <div><span class="text-brown-600">Process:</span> {item.Bean.process}</div>
        {/if}
        {#if item.Bean.description}
          <div class="mt-2 text-brown-800 italic">
            "{item.Bean.description}"
          </div>
        {/if}
      </div>
    </div>
  {:else if item.RecordType === "roaster" && item.Roaster}
    <div
      class="bg-white/60 backdrop-blur rounded-lg p-3 border border-brown-200"
    >
      <div class="text-base mb-2">
        <span class="font-bold text-brown-900">{item.Roaster.name}</span>
      </div>
      <div class="text-sm text-brown-700 space-y-1">
        {#if item.Roaster.location}
          <div>
            <span class="text-brown-600">Location:</span>
            {item.Roaster.location}
          </div>
        {/if}
        {#if safeWebsiteURL(item.Roaster.website)}
          <div>
            <span class="text-brown-600">Website:</span>
            <a
              href={safeWebsiteURL(item.Roaster.website)}
              target="_blank"
              rel="noopener noreferrer"
              class="text-brown-800 hover:underline"
              >{safeWebsiteURL(item.Roaster.website)}</a
            >
          </div>
        {/if}
      </div>
    </div>
  {:else if item.RecordType === "grinder" && item.Grinder}
    <div
      class="bg-white/60 backdrop-blur rounded-lg p-3 border border-brown-200"
    >
      <div class="text-base mb-2">
        <span class="font-bold text-brown-900">{item.Grinder.name}</span>
      </div>
      <div class="text-sm text-brown-700 space-y-1">
        {#if item.Grinder.grinder_type}
          <div>
            <span class="text-brown-600">Type:</span>
            {item.Grinder.grinder_type}
          </div>
        {/if}
        {#if item.Grinder.burr_type}
          <div>
            <span class="text-brown-600">Burr:</span>
            {item.Grinder.burr_type}
          </div>
        {/if}
        {#if item.Grinder.notes}
          <div class="mt-2 text-brown-800 italic">"{item.Grinder.notes}"</div>
        {/if}
      </div>
    </div>
  {:else if item.RecordType === "brewer" && item.Brewer}
    <div
      class="bg-white/60 backdrop-blur rounded-lg p-3 border border-brown-200"
    >
      <div class="text-base mb-2">
        <span class="font-bold text-brown-900">{item.Brewer.name}</span>
      </div>
      {#if item.Brewer.description}
        <div class="text-sm text-brown-800 italic">
          "{item.Brewer.description}"
        </div>
      {/if}
    </div>
  {/if}
</div>
