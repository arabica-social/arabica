import type {
  BeanInput,
  BrewerInput,
  BrewInput,
  GrinderInput,
} from "../api/entities";

export type DraftKind = "brew" | "bean" | "grinder" | "brewer";
type NullableLeaves<T> = {
  [K in keyof T as T[K] extends object | undefined ? never : K]?: T[K] | null;
};
type BrewFields = Omit<
  NullableLeaves<BrewInput>,
  "espresso_params" | "pourover_params" | "pours"
> & {
  [K in keyof NonNullable<
    BrewInput["espresso_params"]
  > as `espresso_params.${K}`]?: number | null;
} & {
  [K in keyof NonNullable<
    BrewInput["pourover_params"]
  > as `pourover_params.${K}`]?:
    | NonNullable<BrewInput["pourover_params"]>[K]
    | null;
} & { pours?: BrewInput["pours"] | null };
export type FieldsByKind = {
  brew: BrewFields;
  bean: NullableLeaves<BeanInput>;
  grinder: NullableLeaves<GrinderInput>;
  brewer: NullableLeaves<BrewerInput>;
};
export type Draft = {
  [K in DraftKind]: { kind: K; fields: FieldsByKind[K] };
}[DraftKind];
export type DraftPatch = FieldsByKind["brew"] &
  FieldsByKind["bean"] &
  FieldsByKind["grinder"] &
  FieldsByKind["brewer"];
export type FieldPath = keyof DraftPatch;
export type FieldValue = Exclude<DraftPatch[FieldPath], undefined>;
export type RequestToken = {
  accountDid: string;
  sessionId: string;
  requestId: string;
  revision: number;
};
export type Message = { role: "user" | "assistant"; content: string };
export type ChangeOptions = {
  /** From a validated own-library brewer, never a model-invented category. */
  brewerType?: string;
  /** Explicit UI confirmation. Safety and revision checks still run on retry. */
  confirmRevision?: number;
};
export type LibraryChoice = {
  token: RequestToken;
  patch: DraftPatch;
  brewerType?: string;
};
export type ApplyResult = {
  status:
    | "applied"
    | "conflict"
    | "needs-confirmation"
    | "invalid"
    | "inactive";
  applied: FieldPath[];
  skipped: FieldPath[];
  requiredClears: FieldPath[];
  revision: number;
  error?: string;
};
export type UndoEntry = {
  revision: number;
  before: DraftPatch;
  absent: FieldPath[];
  brewerType?: string;
};
export type DraftSession = {
  accountDid: string;
  id: string;
  status: "active" | "saved" | "discarded";
  draft: Draft;
  revision: number;
  fieldRevisions: Partial<Record<FieldPath, number>>;
  brewerType?: string;
  pending?: RequestToken;
  /** IDs remain consumed after Stop: old responses must never be retargeted. */
  requestIds: string[];
  completedRequestIds: string[];
  messages: Message[];
  undo?: UndoEntry;
  savedRecordURI?: string;
  editorRevealed?: boolean;
  /** Unsent input stays local to this conversation. */
  composerText?: string;
};
export const MAX_MESSAGES = 40;
export const MAX_USER_CHARS = 4000;
export const MAX_STORAGE_BYTES = 256 * 1024;
