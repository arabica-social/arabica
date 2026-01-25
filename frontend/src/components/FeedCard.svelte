<script>
  export let item;
  import { navigate } from '../lib/router.js';
  
  function safeAvatarURL(url) {
    if (!url) return null;
    if (url.startsWith('https://') || url.startsWith('/static/')) {
      return url;
    }
    return null;
  }
  
  function hasValue(val) {
    return val !== null && val !== undefined && val !== '';
  }
</script>

<div class="bg-gradient-to-br from-brown-50 to-brown-100 rounded-lg shadow-md border border-brown-200 p-4 hover:shadow-lg transition-shadow">
  <!-- Author row -->
  <div class="flex items-center gap-3 mb-3">
    <a href="/profile/{item.Author.handle}" on:click|preventDefault={() => navigate(`/profile/${item.Author.handle}`)} class="flex-shrink-0">
      {#if safeAvatarURL(item.Author.avatar)}
        <img src={safeAvatarURL(item.Author.avatar)} alt="" class="w-10 h-10 rounded-full object-cover hover:ring-2 hover:ring-brown-600 transition" />
      {:else}
        <div class="w-10 h-10 rounded-full bg-brown-300 flex items-center justify-center hover:ring-2 hover:ring-brown-600 transition">
          <span class="text-brown-600 text-sm">?</span>
        </div>
      {/if}
    </a>
    <div class="flex-1 min-w-0">
      <div class="flex items-center gap-2">
        {#if item.Author.displayName}
          <a href="/profile/{item.Author.handle}" on:click|preventDefault={() => navigate(`/profile/${item.Author.handle}`)} class="font-medium text-brown-900 truncate hover:text-brown-700 hover:underline">{item.Author.displayName}</a>
        {/if}
        <a href="/profile/{item.Author.handle}" on:click|preventDefault={() => navigate(`/profile/${item.Author.handle}`)} class="text-brown-600 text-sm truncate hover:text-brown-700 hover:underline">@{item.Author.handle}</a>
      </div>
      <span class="text-brown-500 text-sm">{item.TimeAgo}</span>
    </div>
  </div>

  <!-- Action header -->
  <div class="mb-2 text-sm text-brown-700">
    {#if item.RecordType === 'brew' && item.Brew}
      <span>added a </span>
      <a 
        href="/brews/{item.Author.did}/{item.Brew.rkey}"
        on:click|preventDefault={() => navigate(`/brews/${item.Author.did}/${item.Brew.rkey}`)}
        class="font-semibold text-brown-800 hover:text-brown-900 hover:underline cursor-pointer"
      >
        new brew
      </a>
    {:else}
      {item.Action}
    {/if}
  </div>

  <!-- Record content -->
  {#if item.RecordType === 'brew' && item.Brew}
    <div class="bg-white/60 backdrop-blur rounded-lg p-4 border border-brown-200">
      <!-- Bean info with rating -->
      <div class="flex items-start justify-between gap-3 mb-3">
        <div class="flex-1 min-w-0">
          {#if item.Brew.bean}
            <div class="font-bold text-brown-900 text-base">
              {item.Brew.bean.name || item.Brew.bean.origin}
            </div>
            {#if item.Brew.bean.roaster?.name}
              <div class="text-sm text-brown-700 mt-0.5">
                <span class="font-medium">🏭 {item.Brew.bean.roaster.name}</span>
              </div>
            {/if}
            <div class="text-xs text-brown-600 mt-1 flex flex-wrap gap-x-2 gap-y-0.5">
              {#if item.Brew.bean.origin}<span class="inline-flex items-center gap-0.5">📍 {item.Brew.bean.origin}</span>{/if}
              {#if item.Brew.bean.roast_level}<span class="inline-flex items-center gap-0.5">🔥 {item.Brew.bean.roast_level}</span>{/if}
              {#if item.Brew.bean.process}<span class="inline-flex items-center gap-0.5">🌱 {item.Brew.bean.process}</span>{/if}
              {#if hasValue(item.Brew.coffee_amount)}<span class="inline-flex items-center gap-0.5">⚖️ {item.Brew.coffee_amount}g</span>{/if}
            </div>
          {/if}
        </div>
        {#if hasValue(item.Brew.rating)}
          <span class="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-amber-100 text-amber-900 flex-shrink-0">
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
      
      <!-- Notes -->
      {#if item.Brew.tasting_notes}
        <div class="mt-2 text-sm text-brown-800 italic border-l-2 border-brown-300 pl-3">
          "{item.Brew.tasting_notes}"
        </div>
      {/if}
    </div>
  {:else if item.RecordType === 'bean' && item.Bean}
    <div class="bg-white/60 backdrop-blur rounded-lg p-3 border border-brown-200">
      <div class="font-semibold text-brown-900">{item.Bean.name || item.Bean.origin}</div>
      {#if item.Bean.origin}<div class="text-sm text-brown-700">📍 {item.Bean.origin}</div>{/if}
    </div>
  {:else if item.RecordType === 'roaster' && item.Roaster}
    <div class="bg-white/60 backdrop-blur rounded-lg p-3 border border-brown-200">
      <div class="font-semibold text-brown-900">🏭 {item.Roaster.name}</div>
      {#if item.Roaster.location}<div class="text-sm text-brown-700">📍 {item.Roaster.location}</div>{/if}
    </div>
  {:else if item.RecordType === 'grinder' && item.Grinder}
    <div class="bg-white/60 backdrop-blur rounded-lg p-3 border border-brown-200">
      <div class="font-semibold text-brown-900">⚙️ {item.Grinder.name}</div>
      {#if item.Grinder.grinder_type}<div class="text-sm text-brown-700">{item.Grinder.grinder_type}</div>{/if}
    </div>
  {:else if item.RecordType === 'brewer' && item.Brewer}
    <div class="bg-white/60 backdrop-blur rounded-lg p-3 border border-brown-200">
      <div class="font-semibold text-brown-900">☕ {item.Brewer.name}</div>
      {#if item.Brewer.brewer_type}<div class="text-sm text-brown-700">{item.Brewer.brewer_type}</div>{/if}
    </div>
  {/if}
</div>
