<script lang="ts">
  import {
    extractFragment,
    fetchHTMXPartial,
    formToURLSearchParams,
    postURLEncodedForm,
    responseTextOrThrow,
    showSessionExpiredOn401,
  } from "./domContracts";

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

    const response = await fetchHTMXPartial(
      `/api/comments?${params.toString()}`,
    );
    const html = await responseTextOrThrow(response, "Failed to load comments");
    const nextSection = extractFragment(html, "#comment-section");
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

  function ensureSubjectFields(form: HTMLFormElement) {
    const section = form.closest<HTMLElement>("[data-svelte-comment-section]");
    const subjectURIInput = form.elements.namedItem("subject_uri");
    const subjectCIDInput = form.elements.namedItem("subject_cid");
    const subjectURI =
      (subjectURIInput instanceof HTMLInputElement
        ? subjectURIInput.value
        : "") ||
      section?.dataset.commentSubjectUri ||
      target.dataset.commentSubjectUri ||
      "";
    const subjectCID =
      (subjectCIDInput instanceof HTMLInputElement
        ? subjectCIDInput.value
        : "") ||
      section?.dataset.commentSubjectCid ||
      target.dataset.commentSubjectCid ||
      "";

    upsertHiddenField(form, "subject_uri", subjectURI);
    upsertHiddenField(form, "subject_cid", subjectCID);

    return subjectURI !== "" && subjectCID !== "";
  }

  function commentFormBody(form: HTMLFormElement) {
    const textarea = form.querySelector<HTMLTextAreaElement>(
      'textarea[name="text"]',
    );
    const params = formToURLSearchParams(form);
    params.set("text", textarea?.value || "");
    return params;
  }

  function upsertHiddenField(
    form: HTMLFormElement,
    name: string,
    value: string,
  ) {
    const existing = form.elements.namedItem(name);
    let input: HTMLInputElement;
    if (existing instanceof HTMLInputElement) {
      input = existing;
    } else {
      const hidden = document.createElement("input");
      hidden.type = "hidden";
      hidden.name = name;
      form.prepend(hidden);
      input = hidden;
    }
    input.value = value;
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
    const submitButtons = [
      ...form.querySelectorAll<HTMLButtonElement>('button[type="submit"]'),
    ];
    const localStatus = form.querySelector<HTMLElement>(
      "[data-comment-form-status]",
    );

    const previousStatusNode = statusNode;
    statusNode = localStatus;
    isSubmitting = true;
    if (!ensureSubjectFields(form)) {
      setStatus("Comments are still loading. Refresh and try again.", "error");
      isSubmitting = false;
      statusNode = previousStatusNode;
      return;
    }
    setStatus("Posting...", "info");
    submitButtons.forEach((button) => {
      button.disabled = true;
    });

    try {
      const response = await postURLEncodedForm(form, commentFormBody(form));
      const responseText = await response.text();

      if (!response.ok) {
        setStatus(responseText || "Failed to save comment", "error");
        showSessionExpiredOn401(response);
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
