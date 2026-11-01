import type { PreparedImage } from "./images";

export type HomeImages = { text: string; images: PreparedImage[] };
type Ticket = { accountDid: string };
let pending:
  | { ticket: Ticket; value: HomeImages; timer: ReturnType<typeof setTimeout> }
  | undefined;

/** A single short-lived navigation handoff; never serialized to either browser store. */
export function stageHomeImages(accountDid: string, value: HomeImages): Ticket {
  clearHomeImages();
  const ticket = { accountDid };
  const timer = setTimeout(() => cancelHomeImages(ticket), 30_000);
  pending = { ticket, value, timer };
  return ticket;
}
export function cancelHomeImages(ticket: Ticket) {
  if (pending?.ticket === ticket) clearHomeImages();
}
export function clearHomeImages() {
  if (pending) clearTimeout(pending.timer);
  pending = undefined;
}
export function takeHomeImages(accountDid: string): HomeImages | undefined {
  const current = pending;
  clearHomeImages();
  return current?.ticket.accountDid === accountDid ? current.value : undefined;
}
