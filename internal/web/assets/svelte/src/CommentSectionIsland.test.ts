import { cleanup, render, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import CommentSectionIsland from "./CommentSectionIsland.svelte";
import {
  FEED_MUTATION_EVENT,
  getCachedFeedHTML,
  setCachedFeedHTML,
} from "./feedCache";

function commentTarget() {
  const target = document.createElement("section");
  target.id = "comment-section";
  target.dataset.svelteCommentSection = "";
  target.dataset.commentSubjectUri =
    "at://did:plc:alice/social.arabica.alpha.brew/brew-1";
  target.dataset.commentSubjectCid = "cid-1";
  target.innerHTML = `
    <form action="/api/comments" method="post" data-svelte-comment-form>
      <textarea name="text" aria-label="Write a comment"></textarea>
      <span data-comment-form-status></span>
      <button type="submit">Post</button>
    </form>
  `;
  document.body.appendChild(target);
  return target;
}

describe("CommentSectionIsland", () => {
  afterEach(() => {
    cleanup();
    document.body.innerHTML = "";
    sessionStorage.clear();
    vi.restoreAllMocks();
  });

  it("dispatches a feed mutation and clears cached feed html after successful submit", async () => {
    const user = userEvent.setup();
    document.body.dataset.userDid = "did:plc:alice";
    document.body.dataset.app = "arabica";
    setCachedFeedHTML("/api/feed", '<div id="feed-items">stale</div>');

    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(new Response("", { status: 200 }))
      .mockResolvedValueOnce(
        new Response(
          '<section id="comment-section" data-comment-subject-uri="at://did:plc:alice/social.arabica.alpha.brew/brew-1" data-comment-subject-cid="cid-1"><p>Fresh comment</p></section>',
          { status: 200 },
        ),
      );
    const mutationListener = vi.fn();
    document.body.addEventListener(FEED_MUTATION_EVENT, mutationListener);

    const target = commentTarget();
    render(CommentSectionIsland, { props: { target } });

    await user.type(target.querySelector("textarea")!, "Nice cup");
    await user.click(target.querySelector("button")!);

    await waitFor(() => expect(mutationListener).toHaveBeenCalledTimes(1));
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(getCachedFeedHTML("/api/feed")).toBeNull();
    expect(mutationListener.mock.calls[0][0]).toMatchObject({
      detail: {
        source: "comment",
        action: "create",
        subjectURI: "at://did:plc:alice/social.arabica.alpha.brew/brew-1",
      },
    });
    expect(target).toHaveTextContent("Fresh comment");
  });
});
