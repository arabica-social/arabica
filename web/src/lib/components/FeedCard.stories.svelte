<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import type { FeedItem, FeedAuthor } from "../types/feed";
  import type { Brew, Bean, Roaster } from "../types/entity_view";
  import FeedCard from "./FeedCard.svelte";

  const { Story } = defineMeta({
    title: "Components/FeedCard",
    component: FeedCard,
    tags: ["autodocs"],
  });

  const author: FeedAuthor = {
    did: "did:plc:author1",
    handle: "mika.arabica.example",
    display_name: "Mika",
    avatar: "",
  };

  function itemFor(
    recordType: FeedItem["record_type"],
    record: object,
    overrides: Partial<FeedItem> = {},
  ): FeedItem {
    return {
      record_type: recordType,
      action: "added",
      record: record as Record<string, unknown>,
      author,
      timestamp: "2026-11-01T08:30:00Z",
      time_ago: "2h",
      like_count: 4,
      comment_count: 1,
      subject_uri: `at://${author.did}/app.arabica.record.${recordType}/3m2abc`,
      subject_cid: "bafyrei-example",
      is_liked_by_viewer: false,
      is_owner: false,
      is_moderator: false,
      can_hide_record: false,
      can_block_user: false,
      is_record_hidden: false,
      author_did: author.did,
      ...overrides,
    };
  }

  const bean: Bean = {
    rkey: "bean-rkey",
    name: "Kiamugumo AB",
    origin: "Nyeri, Kenya",
    variety: "SL28, SL34",
    roast_level: "Light",
    process: "Washed",
    description: "",
    notes: "",
    link: "",
    closed: false,
    created_at: "2026-10-12T21:00:00Z",
    roaster: { name: "Karak Mornings", location: "Oslo", rkey: "roaster-rkey" },
  };

  const espressoBrew: Brew = {
    rkey: "3m2abc",
    bean_rkey: "bean-rkey",
    recipe_rkey: "",
    method: "Espresso",
    temperature: 93,
    water_amount: 36,
    coffee_amount: 18,
    time_seconds: 28,
    grind_size: "fine",
    grinder_rkey: "",
    brewer_rkey: "",
    tasting_notes: "Cocoa, dried fig, and a syrupy finish.",
    rating: 9,
    created_at: "2026-11-01T08:30:00Z",
    espresso_params: {
      yield_weight: 36.2,
      pressure: 9,
      pre_infusion_seconds: 6,
    },
    bean,
    brewer_obj: {
      rkey: "",
      name: "Flair 58",
      brewer_type: "manual",
      description: "",
      link: "",
      created_at: "",
    },
    grinder_obj: {
      rkey: "",
      name: "1Zpresso J-Max",
      grinder_type: "hand",
      burr_type: "",
      notes: "",
      link: "",
      created_at: "",
    },
  };

  const pouroverBrew: Brew = {
    rkey: "3m2def",
    bean_rkey: "bean-rkey",
    recipe_rkey: "recipe-rkey",
    temperature: 94,
    water_amount: 300,
    coffee_amount: 20,
    time_seconds: 210,
    grind_size: "medium-fine",
    grinder_rkey: "",
    brewer_rkey: "",
    tasting_notes: "Juicy acidity, black tea body.",
    rating: 8,
    created_at: "2026-10-30T07:45:00Z",
    pourover_params: {
      bloom_water: 50,
      bloom_seconds: 45,
      drawdown_seconds: 30,
      bypass_water: 0,
      filter: "Sibarist Fast",
    },
    bean,
    pours: [
      { pour_number: 1, water_amount: 60, time_seconds: 10, created_at: "" },
      { pour_number: 2, water_amount: 120, time_seconds: 15, created_at: "" },
      { pour_number: 3, water_amount: 120, time_seconds: 20, created_at: "" },
    ],
  };

  const roaster: Roaster = {
    rkey: "roaster-rkey",
    name: "Karak Mornings",
    location: "Oslo, Norway",
    website: "https://karak.example",
    created_at: "2026-09-01T12:00:00Z",
  };
</script>

<Story
  name="Espresso brew"
  args={{ item: itemFor("brew", espressoBrew), isAuthenticated: true }}
/>
<Story
  name="Pourover with pours"
  args={{ item: itemFor("brew", pouroverBrew), isAuthenticated: true }}
/>
<Story
  name="New bean"
  args={{ item: itemFor("bean", bean), isAuthenticated: true }}
/>
<Story
  name="New roaster (compact)"
  args={{ item: itemFor("roaster", roaster), isAuthenticated: true }}
/>
<Story
  name="Own record"
  args={{
    item: itemFor("brew", espressoBrew, { is_owner: true }),
    isAuthenticated: true,
  }}
/>
