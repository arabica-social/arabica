<script>
  import { onMount } from "svelte";
  import router from "./lib/router.js";
  import { authStore } from "./stores/auth.js";

  // Import route components
  import Home from "./routes/Home.svelte";
  import Login from "./routes/Login.svelte";
  import Brews from "./routes/Brews.svelte";
  import BrewView from "./routes/BrewView.svelte";
  import BrewForm from "./routes/BrewForm.svelte";
  import Manage from "./routes/Manage.svelte";
  import Profile from "./routes/Profile.svelte";
  import About from "./routes/About.svelte";
  import Terms from "./routes/Terms.svelte";
  import NotFound from "./routes/NotFound.svelte";

  import Header from "./components/Header.svelte";
  import Footer from "./components/Footer.svelte";

  let currentRoute = null;
  let params = {};

  onMount(() => {
    // Check auth status on mount
    authStore.checkAuth();

    // Setup routes
    router
      .on("/", () => {
        currentRoute = Home;
        params = {};
      })
      .on("/login", () => {
        currentRoute = Login;
        params = {};
      })
      .on("/brews", () => {
        currentRoute = Brews;
        params = {};
      })
      .on("/brews/new", () => {
        currentRoute = BrewForm;
        params = { mode: "create" };
      })
      .on("/brews/:id", (routeParams) => {
        currentRoute = BrewView;
        params = routeParams;
      })
      .on("/brews/:id/edit", (routeParams) => {
        currentRoute = BrewForm;
        params = { ...routeParams, mode: "edit" };
      })
      .on("/manage", () => {
        currentRoute = Manage;
        params = {};
      })
      .on("/profile/:actor", (routeParams) => {
        currentRoute = Profile;
        params = routeParams;
      })
      .on("/about", () => {
        currentRoute = About;
        params = {};
      })
      .on("/terms", () => {
        currentRoute = Terms;
        params = {};
      })
      .on("*", () => {
        currentRoute = NotFound;
        params = {};
      });

    // Start router
    router.listen();
  });
</script>

<div class="flex flex-col min-h-screen">
  <Header />

  <main class="flex-1 container mx-auto px-4 py-8">
    {#if currentRoute}
      <svelte:component this={currentRoute} {...params} />
    {/if}
  </main>

  <Footer />
</div>
