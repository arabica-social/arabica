<script lang="ts">
  interface Props {
    target: HTMLElement;
  }

  let { target }: Props = $props();

  let isSubmitting = $state(false);
  let statusNode = $state<HTMLElement | null>(null);
  const statusClasses = {
    info: "text-xs text-placeholder",
    success: "text-xs text-green-700 dark:text-green-400",
    error: "text-xs text-danger",
  };

  async function refreshCommentSection() {
    const subjectURI = target.dataset.commentSubjectUri || "";
    const subjectCID = target.dataset.commentSubjectCid || "";
    if (!subjectURI || !subjectCID) {
      return;
    }

    const params = new URLSearchParams({
      subject_uri: subjectURI,
      subject_cid: subjectCID,
    });

    const response = await fetch(`/api/comments?${params.toString()}`, {
      method: "GET",
      credentials: "same-origin",
    });

    const html = await response.text();
    if (!response.ok) {
      throw new Error(html.trim() || "Failed to load comments");
    }

    const parser = new DOMParser();
    const doc = parser.parseFromString(html, "text/html");
    const nextSection = doc.querySelector("#comment-section");
    if (nextSection) {
      target.innerHTML = nextSection.innerHTML;
    } else {
      target.innerHTML = html;
    }
  }

  function setStatus(
    message: string,
    type: "info" | "success" | "error" = "info",
  ) {
    if (!statusNode) {
      return;
    }
    statusNode.textContent = message;
    statusNode.className = statusClasses[type];
  }

  async function handleSubmit(event: SubmitEvent) {
    const eventTarget = event.target;
    if (!(eventTarget instanceof HTMLFormElement)) {
      return;
    }
    if (
      !target.contains(eventTarget) ||
      !eventTarget.matches("form[data-svelte-comment-form]")
    ) {
      return;
    }

    event.preventDefault();

    if (isSubmitting) {
      return;
    }

    const form = eventTarget;
    const endpoint = form.getAttribute("action") || "/api/comments";
    const method = (form.method || "POST").toUpperCase();
    const submitButtons = [
      ...form.querySelectorAll<HTMLButtonElement>('button[type="submit"]'),
    ];
    const localStatus = form.querySelector<HTMLElement>(
      "[data-comment-form-status]",
    );

    const previousStatusNode = statusNode;
    statusNode = localStatus;
    isSubmitting = true;
    setStatus("Posting...", "info");
    submitButtons.forEach((button) => {
      button.disabled = true;
    });

    try {
      const response = await fetch(endpoint, {
        method,
        credentials: "same-origin",
        body: new FormData(form),
      });
      const responseText = await response.text();

      if (!response.ok) {
        setStatus(responseText || "Failed to save comment", "error");
        if (response.status === 401) {
          window.__showSessionExpiredModal?.();
        }
        return;
      }

      await refreshCommentSection();
      setStatus("Posted", "success");
      form.reset();
      window.__arabicaSvelteIslands?.mountAll();
    } catch (error) {
      console.error("Failed to submit comment form:", error);
      setStatus("Failed to save comment", "error");
    } finally {
      isSubmitting = false;
      statusNode = previousStatusNode;
      submitButtons.forEach((button) => {
        button.disabled = false;
      });
    }
  }

  $effect(() => {
    target.addEventListener("submit", handleSubmit);
    return () => {
      target.removeEventListener("submit", handleSubmit);
    };
  });
</script>
