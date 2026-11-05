<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import type { IndexedComment } from "../types/entity_view";
  import CommentSection from "./CommentSection.svelte";

  const { Story } = defineMeta({
    title: "Components/CommentSection",
    component: CommentSection,
    tags: ["autodocs"],
  });

  const base = {
    subject_uri: "at://did:plc:author1/app.arabica.record.brew/3m2abc",
    created_at: "2026-11-01T09:00:00Z",
    like_count: 0,
    is_liked: false,
  };

  const comments: IndexedComment[] = [
    {
      ...base,
      rkey: "c1",
      text: "That yield looks spot on. How does it compare to your usual 1:2?",
      actor_did: "did:plc:c1",
      handle: "mika.arabica.example",
      display_name: "Mika",
      depth: 0,
    },
    {
      ...base,
      rkey: "c2",
      text: "Bit tighter than usual — pulling for more body. Full review soon.",
      actor_did: "did:plc:viewer",
      handle: "viewer.arabica.example",
      display_name: "Pat",
      depth: 1,
      parent_rkey: "c1",
      parent_uri: base.subject_uri,
    },
    {
      ...base,
      rkey: "c3",
      text: "Adding this to my to-brew list.",
      actor_did: "did:plc:c3",
      handle: "sven.arabica.example",
      display_name: "Sven",
      depth: 0,
      created_at: "2026-11-01T11:30:00Z",
    },
  ];

  const shared = {
    comments,
    subjectURI: base.subject_uri,
    subjectCID: "bafyrei-example",
    viewURL: "/brews/mika.arabica.example/3m2abc",
  };
</script>

<Story
  name="Signed in"
  args={{ ...shared, isAuthenticated: true, currentUserDID: "did:plc:viewer" }}
/>
<Story name="Visitor" args={{ ...shared, isAuthenticated: false }} />
<Story
  name="Empty"
  args={{
    comments: [],
    isAuthenticated: true,
    currentUserDID: "did:plc:viewer",
    subjectURI: base.subject_uri,
    subjectCID: "bafyrei-example",
  }}
/>
