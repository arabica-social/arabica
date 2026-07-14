<script lang="ts">
  import { afterNavigate } from "$app/navigation";
  import { page } from "$app/state";
  import { definitionFor } from "$lib/app/definitions";
  import {
    app,
    displayHandle,
    formatNotificationCount,
    openLoginModal,
    profileIdentifier,
    session,
  } from "$lib/stores/session";
  import Avatar from "./Avatar.svelte";
  import Icon, { type IconName } from "./Icon.svelte";

  let { brandName = "Arabica" }: { brandName?: string } = $props();

  let createOpen = $state(false);
  let userOpen = $state(false);

  let isOolong = $derived($app === "oolong");
  let definition = $derived(definitionFor($app));
  let s = $derived($session);
  let unread = $derived(s.unreadNotifications);
  let pathname = $derived(page.url.pathname);

  type NavItem = {
    href: string;
    label: string;
    shortLabel: string;
    icon: IconName;
  };
  let primaryItems = $derived.by<NavItem[]>(() => {
    const items: NavItem[] = [
      {
        href: "/",
        label: "Community",
        shortLabel: "Community",
        icon: "coffee",
      },
      {
        href: "/explore",
        label: "Explore",
        shortLabel: "Explore",
        icon: "globe",
      },
    ];
    if (s.isAuthenticated) {
      items.push({
        href: "/brews",
        label: isOolong ? "Steeps" : "Brews",
        shortLabel: isOolong ? "Steeps" : "Brews",
        icon: isOolong ? "droplet" : "coffee",
      });
      items.push({
        href: definition.libraryPath,
        label: definition.libraryLabel,
        shortLabel: isOolong ? "My Tea" : "My Coffee",
        icon: isOolong ? "leaf" : "bean",
      });
      if (!isOolong) {
        items.push({
          href: "/recipes",
          label: "Recipes",
          shortLabel: "Recipes",
          icon: "fileText",
        });
      }
    }
    return items;
  });

  function isActive(href: string): boolean {
    if (href === "/") return pathname === "/";
    return pathname === href || pathname.startsWith(`${href}/`);
  }

  function closeMenus() {
    createOpen = false;
    userOpen = false;
  }

  function toggleCreate() {
    userOpen = false;
    createOpen = !createOpen;
  }

  function toggleUser() {
    createOpen = false;
    userOpen = !userOpen;
  }

  function handleOutsideClick(event: MouseEvent) {
    const target = event.target;
    if (target instanceof Element && !target.closest("[data-menu-root]"))
      closeMenus();
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") closeMenus();
  }

  $effect(() => {
    document.addEventListener("click", handleOutsideClick);
    document.addEventListener("keydown", handleKeydown);
    return () => {
      document.removeEventListener("click", handleOutsideClick);
      document.removeEventListener("keydown", handleKeydown);
    };
  });

  afterNavigate(closeMenus);
</script>

<header class="site-header">
  <div class="site-header__rule" aria-hidden="true"></div>
  <nav class="site-header__bar" aria-label="Primary navigation">
    <a href="/" class="site-brand" aria-label={`${brandName} community home`}>
      <span class="site-brand__mark" aria-hidden="true">
        <Icon name={isOolong ? "teapot" : "coffee"} />
      </span>
      <span class="site-brand__name">{brandName}</span>
      <span class="badge-alpha">ALPHA</span>
    </a>

    <div class="desktop-nav" aria-label="Ledger destinations">
      {#each primaryItems as item (item.href)}
        <a
          href={item.href}
          class:active={isActive(item.href)}
          aria-current={isActive(item.href) ? "page" : undefined}
        >
          {item.label}
        </a>
      {/each}
    </div>

    <div class="site-actions">
      {#if !s.isAuthenticated}
        <button type="button" onclick={openLoginModal} class="login-action"
          >Log In</button
        >
      {:else}
        <div class="create-control" data-menu-root>
          <button
            type="button"
            onclick={toggleCreate}
            class="create-action"
            aria-label="Create new"
            aria-expanded={createOpen}
          >
            <Icon name={isOolong ? "droplet" : "coffee"} />
            <span class="create-action__long">{definition.sessionAction}</span>
            <span class="create-action__short">Log</span>
            <Icon name="chevronDown" class="create-action__chevron" />
          </button>
          {#if createOpen}
            <div class="ledger-menu ledger-menu--create" role="menu">
              <div class="ledger-menu__heading"><span>01</span>Log</div>
              <a href="/brews/new" class="ledger-menu__primary" role="menuitem">
                <Icon name={isOolong ? "droplet" : "coffee"} />
                <span
                  ><strong>{isOolong ? "New steep" : "New brew"}</strong><small
                    >Add a session to your journal</small
                  ></span
                >
              </a>
              <div class="ledger-menu__heading ledger-menu__heading--secondary">
                <span>02</span>Add to your ledger
              </div>
              {#if isOolong}
                <a href="/teas/new" class="ledger-menu__item" role="menuitem"
                  ><Icon name="leaf" />Tea</a
                >
                <a href="/vendors/new" class="ledger-menu__item" role="menuitem"
                  ><Icon name="store" />Vendor</a
                >
                <a href="/vessels/new" class="ledger-menu__item" role="menuitem"
                  ><Icon name="brewer" />Vessel</a
                >
                <a
                  href="/infusers/new"
                  class="ledger-menu__item"
                  role="menuitem"><Icon name="disc" />Infuser</a
                >
              {:else}
                <a href="/beans/new" class="ledger-menu__item" role="menuitem"
                  ><Icon name="bean" />Bean</a
                >
                <a
                  href="/roasters/new"
                  class="ledger-menu__item"
                  role="menuitem"><Icon name="store" />Roaster</a
                >
                <a
                  href="/grinders/new"
                  class="ledger-menu__item"
                  role="menuitem"><Icon name="disc" />Grinder</a
                >
                <a href="/brewers/new" class="ledger-menu__item" role="menuitem"
                  ><Icon name="brewer" />Brewer</a
                >
                <a href="/recipes/new" class="ledger-menu__item" role="menuitem"
                  ><Icon name="fileText" />Recipe</a
                >
              {/if}
            </div>
          {/if}
        </div>

        <a
          href="/notifications"
          class="notification-action"
          class:active={isActive("/notifications")}
          aria-label={unread > 0
            ? `Notifications, ${unread} unread`
            : "Notifications"}
          aria-current={isActive("/notifications") ? "page" : undefined}
        >
          <Icon name="bell" />
          {#if unread > 0}<span>{formatNotificationCount(unread)}</span>{/if}
        </a>

        <div class="account-control" data-menu-root>
          <button
            type="button"
            onclick={toggleUser}
            class="account-action"
            aria-label="User menu"
            aria-expanded={userOpen}
          >
            <Avatar
              avatarURL={s.avatar}
              displayName={s.displayName}
              size="sm"
            />
            <Icon name="chevronDown" />
          </button>
          {#if userOpen}
            <div class="ledger-menu ledger-menu--account" role="menu">
              {#if s.handle}
                <div class="account-slip">
                  <p>Your ledger</p>
                  <strong>{s.displayName || displayHandle(s.handle)}</strong>
                  <small>@{displayHandle(s.handle)}</small>
                </div>
              {/if}
              <a
                href={`/profile/${profileIdentifier(s)}`}
                class="ledger-menu__item"
                role="menuitem">View profile</a
              >
              <a
                href={definition.libraryPath}
                class="ledger-menu__item"
                role="menuitem">{definition.libraryLabel}</a
              >
              <a href="/settings" class="ledger-menu__item" role="menuitem"
                >Settings</a
              >
              {#if s.isModerator}
                <a
                  href="/_mod"
                  class="ledger-menu__item ledger-menu__item--mod"
                  role="menuitem"><Icon name="shieldCheck" />Moderation</a
                >
              {/if}
              <form action="/logout" method="POST" class="ledger-menu__logout">
                <button type="submit" class="ledger-menu__item" role="menuitem"
                  >Log out</button
                >
              </form>
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </nav>
</header>

<nav class="mobile-ledger-nav" aria-label="Mobile navigation">
  {#each primaryItems as item (item.href)}
    <a
      href={item.href}
      class:active={isActive(item.href)}
      aria-current={isActive(item.href) ? "page" : undefined}
    >
      <Icon name={item.icon} />
      <span>{item.shortLabel}</span>
    </a>
  {/each}
</nav>

<style>
  .site-header {
    position: sticky;
    top: 0;
    z-index: 200;
    color: var(--header-text);
    background: var(--header-bg);
    box-shadow: var(--shadow-sm);
  }
  .site-header__rule {
    height: 2px;
    background: color-mix(
      in oklch,
      var(--brand-amber-500) 68%,
      var(--header-bg)
    );
  }
  .site-header__bar {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr) auto;
    align-items: center;
    gap: clamp(0.75rem, 2vw, 1.5rem);
    width: min(100%, 90rem);
    min-height: 4.25rem;
    margin-inline: auto;
    padding: 0.55rem clamp(0.75rem, 2.5vw, 2rem);
    border-bottom: 1px solid var(--header-border);
  }
  .site-brand {
    display: inline-flex;
    align-items: center;
    gap: 0.55rem;
    min-height: 2.75rem;
    color: inherit;
    text-decoration: none;
  }
  .site-brand__mark {
    display: grid;
    place-items: center;
    width: 1.75rem;
    height: 1.75rem;
    border: 1px solid color-mix(in oklch, currentColor 35%, transparent);
    border-radius: 50%;
  }
  .site-brand__mark :global(svg) {
    width: 1rem;
    height: 1rem;
    color: currentColor;
  }
  .site-brand__name {
    font-family: var(--font-display);
    font-size: 1.45rem;
    font-weight: 600;
    letter-spacing: -0.015em;
  }
  .desktop-nav {
    display: none;
    justify-content: center;
    align-self: stretch;
  }
  .desktop-nav a {
    position: relative;
    display: inline-flex;
    align-items: center;
    min-height: 100%;
    padding: 0 0.7rem;
    color: color-mix(in oklch, var(--header-text) 72%, transparent);
    font-size: 0.77rem;
    font-weight: 600;
    text-decoration: none;
  }
  .desktop-nav a::after {
    content: "";
    position: absolute;
    inset: auto 0.7rem -0.55rem;
    height: 3px;
    background: var(--brand-amber-400);
    transform: scaleX(0);
    transform-origin: center;
    transition: transform 160ms cubic-bezier(0.16, 1, 0.3, 1);
  }
  .desktop-nav a:hover,
  .desktop-nav a:focus-visible,
  .desktop-nav a.active {
    color: var(--header-text);
  }
  .desktop-nav a.active::after {
    transform: scaleX(1);
  }
  .site-actions {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 0.35rem;
  }
  .login-action,
  .create-action,
  .notification-action,
  .account-action {
    min-height: 2.75rem;
    border-radius: 0.45rem;
    color: var(--header-text);
  }
  .login-action {
    padding: 0.55rem 1rem;
    border: 1px solid color-mix(in oklch, var(--header-text) 32%, transparent);
    font-size: 0.8rem;
    font-weight: 600;
  }
  .create-control,
  .account-control {
    position: relative;
  }
  .create-action {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    padding: 0.5rem 0.7rem;
    border: 1px solid
      color-mix(in oklch, var(--brand-amber-400) 68%, var(--header-bg));
    background: color-mix(
      in oklch,
      var(--brand-amber-500) 18%,
      var(--header-bg)
    );
    font-size: 0.78rem;
    font-weight: 600;
  }
  .create-action :global(svg) {
    width: 1rem;
    height: 1rem;
    color: currentColor;
  }
  .create-action__long {
    display: none;
  }
  :global(.create-action__chevron) {
    width: 0.75rem !important;
    height: 0.75rem !important;
    opacity: 0.72;
  }
  .notification-action {
    position: relative;
    display: grid;
    place-items: center;
    width: 2.75rem;
    text-decoration: none;
  }
  .notification-action :global(svg) {
    width: 1.25rem;
    height: 1.25rem;
  }
  .notification-action > span {
    position: absolute;
    top: 0.15rem;
    right: 0.05rem;
    display: grid;
    place-items: center;
    min-width: 1.1rem;
    height: 1.1rem;
    padding-inline: 0.2rem;
    border: 2px solid var(--header-bg);
    border-radius: 999px;
    color: var(--espresso-deep, #3d2319);
    background: var(--brand-amber-400);
    font-size: 0.58rem;
    font-weight: 700;
  }
  .account-action {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.2rem 0.25rem;
  }
  .account-action :global(svg) {
    width: 0.75rem;
    height: 0.75rem;
  }
  .login-action:hover,
  .create-action:hover,
  .notification-action:hover,
  .account-action:hover,
  .notification-action.active {
    background: color-mix(in oklch, var(--header-text) 10%, transparent);
  }
  .login-action:focus-visible,
  .create-action:focus-visible,
  .notification-action:focus-visible,
  .account-action:focus-visible,
  .desktop-nav a:focus-visible,
  .mobile-ledger-nav a:focus-visible {
    outline: 2px solid var(--brand-amber-400);
    outline-offset: 2px;
  }
  .ledger-menu {
    position: absolute;
    top: calc(100% + 0.65rem);
    right: 0;
    width: min(20rem, calc(100vw - 1.5rem));
    padding: 0.35rem;
    border: 1px solid var(--card-border);
    border-top: 3px double var(--text-primary);
    border-radius: 0.45rem;
    color: var(--text-primary);
    background: var(--card-bg);
    background-image: var(--texture-kraft);
    background-blend-mode: multiply;
    box-shadow: var(--shadow-lg);
  }
  .ledger-menu__heading {
    display: flex;
    gap: 0.45rem;
    padding: 0.45rem 0.65rem 0.35rem;
    color: var(--text-faint);
    font-size: 0.63rem;
    font-weight: 600;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  .ledger-menu__heading span {
    font-variant-numeric: tabular-nums;
  }
  .ledger-menu__heading--secondary {
    margin-top: 0.35rem;
    border-top: 1px solid var(--surface-border);
    padding-top: 0.7rem;
  }
  .ledger-menu__primary,
  .ledger-menu__item {
    display: flex;
    align-items: center;
    gap: 0.7rem;
    min-height: 2.75rem;
    padding: 0.55rem 0.65rem;
    border-radius: 0.3rem;
    color: var(--text-primary);
    text-decoration: none;
  }
  .ledger-menu__primary {
    border: 1px solid var(--card-border);
    background: color-mix(in oklch, var(--type-brew-tint) 42%, var(--card-bg));
  }
  .ledger-menu__primary :global(svg),
  .ledger-menu__item :global(svg) {
    width: 1rem;
    height: 1rem;
    color: currentColor;
  }
  .ledger-menu__primary strong,
  .ledger-menu__primary small {
    display: block;
  }
  .ledger-menu__primary strong {
    font-family: var(--font-display);
    font-size: 1rem;
  }
  .ledger-menu__primary small {
    margin-top: 0.1rem;
    color: var(--text-muted);
    font-size: 0.67rem;
  }
  .ledger-menu__item {
    border-top: 1px solid var(--surface-border);
    font-size: 0.8rem;
  }
  .ledger-menu__item:hover,
  .ledger-menu__item:focus-visible,
  .ledger-menu__primary:hover,
  .ledger-menu__primary:focus-visible {
    outline: none;
    background: var(--surface-bg);
  }
  .ledger-menu__item--mod {
    color: var(--brand-amber-700);
  }
  .account-slip {
    padding: 0.7rem 0.65rem 0.8rem;
    border-bottom: 1px solid var(--card-border);
  }
  .account-slip p,
  .account-slip strong,
  .account-slip small {
    display: block;
    margin: 0;
  }
  .account-slip p {
    color: var(--text-faint);
    font-size: 0.62rem;
    font-weight: 600;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  .account-slip strong {
    margin-top: 0.35rem;
    font-family: var(--font-display);
    font-size: 1rem;
  }
  .account-slip small {
    margin-top: 0.1rem;
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .ledger-menu__logout {
    margin-top: 0.35rem;
    border-top: 1px solid var(--card-border);
    padding-top: 0.35rem;
  }
  .ledger-menu__logout button {
    width: 100%;
    border: 0;
    background: transparent;
  }
  .mobile-ledger-nav {
    position: fixed;
    inset: auto 0 0;
    z-index: 190;
    display: grid;
    grid-auto-flow: column;
    grid-auto-columns: 1fr;
    padding: 0.35rem max(0.35rem, env(safe-area-inset-right))
      max(0.35rem, env(safe-area-inset-bottom))
      max(0.35rem, env(safe-area-inset-left));
    border-top: 1px solid var(--card-border);
    background: color-mix(in oklch, var(--card-bg) 96%, transparent);
    box-shadow: 0 -4px 16px
      color-mix(in oklch, var(--header-bg) 12%, transparent);
  }
  .mobile-ledger-nav a {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.2rem;
    min-width: 0;
    min-height: 3.25rem;
    padding: 0.25rem;
    color: var(--text-muted);
    font-size: 0.58rem;
    font-weight: 600;
    text-decoration: none;
  }
  .mobile-ledger-nav a::before {
    content: "";
    position: absolute;
    inset: 0.05rem 28% auto;
    height: 2px;
    background: var(--text-primary);
    transform: scaleX(0);
  }
  .mobile-ledger-nav a.active {
    color: var(--text-primary);
  }
  .mobile-ledger-nav a.active::before {
    transform: scaleX(1);
  }
  .mobile-ledger-nav :global(svg) {
    width: 1rem;
    height: 1rem;
    color: currentColor;
  }
  :global(main) {
    padding-bottom: 5.5rem !important;
  }
  @media (min-width: 760px) {
    .site-header__bar {
      min-height: 4.5rem;
    }
    .create-action__long {
      display: inline;
    }
    .create-action__short {
      display: none;
    }
  }
  @media (min-width: 1040px) {
    .desktop-nav {
      display: flex;
    }
    .mobile-ledger-nav {
      display: none;
    }
    :global(main) {
      padding-bottom: 2rem !important;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .desktop-nav a::after {
      transition: none;
    }
  }
</style>
