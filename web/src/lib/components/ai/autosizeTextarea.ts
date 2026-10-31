/** Fit content, not an interaction history; CSS keeps the maximum height bounded. */
export function autosizeTextarea(node: HTMLTextAreaElement, _value: string) {
  let alive = true;
  let width = node.getBoundingClientRect().width;
  function resize() {
    if (!alive) return;
    const style = getComputedStyle(node);
    const borders =
      parseFloat(style.borderTopWidth || "0") +
      parseFloat(style.borderBottomWidth || "0");
    node.style.height = "0px";
    node.style.height = `${node.scrollHeight + borders}px`;
    node.style.overflowY =
      node.scrollHeight > node.clientHeight ? "auto" : "hidden";
  }
  const observer =
    typeof ResizeObserver === "undefined"
      ? undefined
      : new ResizeObserver(() => {
          const next = node.getBoundingClientRect().width;
          if (next !== width) {
            width = next;
            resize();
          }
        });
  observer?.observe(node);
  if (!observer) window.addEventListener("resize", resize);
  queueMicrotask(resize);
  void document.fonts?.ready.then(resize);
  return {
    update: resize,
    destroy() {
      alive = false;
      observer?.disconnect();
      if (!observer) window.removeEventListener("resize", resize);
    },
  };
}
