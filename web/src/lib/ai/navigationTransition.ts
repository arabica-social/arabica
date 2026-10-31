/** The only animated route handoff: the Home composer becoming the chat composer. */
interface ChatNavigation {
  from: { url: URL } | null;
  to: { url: URL } | null;
  complete: Promise<void>;
}
interface TransitionDocument {
  startViewTransition?: (update: () => Promise<void>) => {
    finished: Promise<void>;
    ready?: Promise<void>;
    updateCallbackDone?: Promise<void>;
  };
  documentElement: { classList: Pick<DOMTokenList, "add" | "remove"> };
}

export function homeToChatTransition(
  navigation: ChatNavigation,
  doc: TransitionDocument,
  reducedMotion: boolean,
): Promise<void> | undefined {
  const { from, to } = navigation;
  const startViewTransition = doc.startViewTransition;
  if (
    !from ||
    !to ||
    from.url.origin !== to.url.origin ||
    from.url.pathname !== "/" ||
    to.url.pathname !== "/chat" ||
    reducedMotion ||
    !startViewTransition
  )
    return;

  // Resolve onNavigate as soon as the browser captures the old composer. The
  // update callback then waits for SvelteKit, never the other way around.
  const complete = navigation.complete.then(
    () => {},
    () => {},
  );
  return new Promise<void>((release) => {
    const cleanup = () => {
      doc.documentElement.classList.remove("ai-chat-transition");
      release();
    };
    try {
      doc.documentElement.classList.add("ai-chat-transition");
      const transition = startViewTransition.call(doc, () => {
        release();
        return complete;
      });
      // Browser transitions can be skipped or rejected (including before the
      // update callback runs). None of their promises may block navigation.
      void transition.finished.then(cleanup, cleanup);
      void transition.ready?.catch(() => {});
      void transition.updateCallbackDone?.catch(() => {});
    } catch {
      cleanup();
    }
  });
}
