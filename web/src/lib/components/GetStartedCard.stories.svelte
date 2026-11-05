<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import type { OnboardingResponse } from "../types/api";
  import GetStartedCard from "./GetStartedCard.svelte";

  const { Story } = defineMeta({
    title: "Components/GetStartedCard",
    component: GetStartedCard,
    tags: ["autodocs"],
  });

  function onboarding(state: {
    bean?: boolean;
    brewer?: boolean;
    roaster?: boolean;
    beans?: string[];
    brewers?: string[];
    grinders?: string[];
    roasters?: string[];
  }): OnboardingResponse {
    return {
      readiness: {
        HasBean: state.bean ?? false,
        HasBrewer: state.brewer ?? false,
        HasRoaster: state.roaster ?? false,
      },
      beans: (state.beans ?? []).map((name) => ({
        rkey: name,
        name,
        origin: "",
        variety: "",
        roast_level: "",
        process: "",
        description: "",
        notes: "",
        link: "",
        closed: false,
        created_at: "",
      })),
      brewers: (state.brewers ?? []).map((name) => ({
        rkey: name,
        name,
        brewer_type: "",
        description: "",
        link: "",
        created_at: "",
      })),
      grinders: (state.grinders ?? []).map((name) => ({
        rkey: name,
        name,
        grinder_type: "",
        burr_type: "",
        notes: "",
        link: "",
        created_at: "",
      })),
      roasters: (state.roasters ?? []).map((name) => ({
        rkey: name,
        name,
        location: "",
        website: "",
        created_at: "",
      })),
    };
  }

  const fresh = onboarding({});
  const midway = onboarding({
    brewer: true,
    bean: true,
    brewers: ["Flair 58"],
    beans: ["Kiamugumo AB"],
  });
  const stocked = onboarding({
    bean: true,
    brewer: true,
    roaster: true,
    beans: [
      "Kiamugumo AB",
      "Yirgacheffe G1",
      "La Palma Gesha",
      "Huila Pink Bourbon",
    ],
    brewers: ["Flair 58", "V60"],
    grinders: ["1Zpresso J-Max"],
    roasters: ["Karak Mornings"],
  });
</script>

<Story name="Fresh start" args={{ onboarding: fresh }} />
<Story name="Two of three done" args={{ onboarding: midway }} />
<Story name="Ready to brew" args={{ onboarding: stocked }} />
<Story name="Library mode" args={{ onboarding: stocked, mode: "library" }} />
