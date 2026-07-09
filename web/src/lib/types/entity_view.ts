// Hand-maintained types for the entity view JSON API.
// Mirrors internal/handlers/entity_view_json.go (EntityViewJSONResponse,
// AuthorSummary, SocialDataJSON) and internal/backlinks/service.go
// (Result, Entry, UsageGroup). Run `just types-generate` for entity
// model types; these envelope types are maintained by hand (see
// docs/api/entities.md).
//
// The `record` field is generic — each page supplies its own record type.
// We define lightweight record types here to avoid importing the generated
// entities.ts (which has some unresolvable cross-package consts).

export type Roaster = {
	rkey: string;
	name: string;
	location: string;
	website: string;
	source_ref?: string;
	created_at: string;
};

export type Grinder = {
	rkey: string;
	name: string;
	grinder_type: string;
	burr_type: string;
	notes: string;
	link: string;
	created_at: string;
};

export type Brewer = {
	rkey: string;
	name: string;
	brewer_type: string;
	description: string;
	link: string;
	created_at: string;
};

export type Bean = {
	rkey: string;
	name: string;
	origin: string;
	variety: string;
	roast_level: string;
	roast_date?: string;
	process: string;
	description: string;
	notes: string;
	link: string;
	rating?: number;
	closed: boolean;
	created_at: string;
	roaster?: { name: string; location: string; rkey: string };
};

export type Pour = {
	pour_number: number;
	water_amount: number;
	time_seconds: number;
	created_at: string;
};

export type EspressoParams = {
	yield_weight: number;
	pressure: number;
	pre_infusion_seconds: number;
};

export type PouroverParams = {
	bloom_water: number;
	bloom_seconds: number;
	drawdown_seconds: number;
	bypass_water: number;
	filter: string;
};

export type Recipe = {
	rkey: string;
	name: string;
	brewer_rkey: string;
	brewer_type: string;
	coffee_amount: number;
	water_amount: number;
	notes: string;
	created_at: string;
	brewer_obj?: Brewer;
	pours?: Pour[];
	ratio?: number;
	author_did?: string;
	author_handle?: string;
	author_avatar?: string;
	author_display?: string;
	source_author_handle?: string;
	source_author_avatar?: string;
	source_author_display?: string;
	fork_count?: number;
	brew_count?: number;
	forker_avatars?: string[];
};

export type Brew = {
	rkey: string;
	bean_rkey: string;
	recipe_rkey: string;
	method?: string;
	temperature: number;
	water_amount: number;
	coffee_amount: number;
	time_seconds: number;
	grind_size: string;
	grinder_rkey: string;
	brewer_rkey: string;
	tasting_notes: string;
	rating: number;
	created_at: string;
	espresso_params?: EspressoParams;
	pourover_params?: PouroverParams;
	bean?: Bean;
	recipe_obj?: Recipe;
	grinder_obj?: Grinder;
	brewer_obj?: Brewer;
	pours?: Pour[];
};

export type AuthorSummary = {
  did: string;
  handle: string;
  display_name?: string;
  avatar?: string;
};

export type IndexedComment = {
  rkey: string;
  subject_uri: string;
  text: string;
  actor_did: string;
  created_at: string;
  parent_uri?: string;
  parent_rkey?: string;
  cid?: string;
  depth: number;
  like_count: number;
  is_liked: boolean;
  // Computed profile fields (populated by the JSON layer).
  handle?: string;
  display_name?: string;
  avatar?: string;
};

export type SocialDataJSON = {
  is_liked: boolean;
  like_count: number;
  comment_count: number;
  comments: IndexedComment[];
  is_moderator: boolean;
  can_hide_record: boolean;
  can_block_user: boolean;
  is_record_hidden: boolean;
};

export type BacklinkEntry = {
  DID: string;
  Handle: string;
  DisplayName: string;
  AvatarURL: string;
  RecordURI: string;
  Collection: string;
  RKey: string;
  CreatedAt: string;
  Title: string;
  Rating: number;
  HasRating: boolean;
  ChainDepth: number;
};

export type UsageGroup = {
  Key: string;
  Label: string;
  Entries: BacklinkEntry[];
  Count: number;
  RatingAverage: number;
  RatingCount: number;
  Page: number;
  PerPage: number;
  HasPrev: boolean;
  HasNext: boolean;
};

export type BacklinksResult = {
  LibraryEntries: BacklinkEntry[];
  LibraryCount: number;
  Usage: UsageGroup[];
  UsageCount: number;
  RatingAverage: number;
  RatingCount: number;
};

// Union of all entity record types returned in the `record` field.
// Pages type-narrow based on entity_type.
export type EntityViewResponse<TRecord = Record<string, unknown>> = {
  record: TRecord;
  subject_uri: string;
  subject_cid: string;
  author?: AuthorSummary;
  social: SocialDataJSON;
  backlinks?: BacklinksResult;
  is_own_profile: boolean;
  is_authenticated: boolean;
  share_url: string;
  entity_type: string;
  entity_count: number;
};
