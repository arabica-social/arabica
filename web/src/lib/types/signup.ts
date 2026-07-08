// Hand-maintained types for the signup JSON API.
// Mirrors internal/signup.Provider and internal/signup.Category.
// Run `just types-generate` for entity types; these envelope types are
// maintained by hand since they're API-specific (see docs/api/README.md).

export type SignupProvider = {
  url: string;
  name: string;
  domain: string;
  description: string;
  location: string;
  badge: string;
  badge_color: string;
  operator_name: string;
  operator_url: string;
  signup_url: string;
};

export type SignupCategory = {
  title: string;
  description: string;
  providers: SignupProvider[];
  dev_only: boolean;
};

export type SignupCategoriesResponse = {
  categories: SignupCategory[];
};
