// Package signup holds the PDS provider catalog shown on the account
// creation page and the derived allowlist used by the signup handler.
package signup

// Provider describes a single PDS hosting option.
type Provider struct {
	URL          string // Full URL for the signup form (e.g. "https://arabica.systems")
	Name         string // Display name (e.g. "Arabica")
	Domain       string // Display domain (e.g. "arabica.systems")
	Description  string // Short description shown under the name
	Location     string // Geographic location (e.g. "United States")
	Badge        string // Status badge text (e.g. "Invite Only", "Open")
	BadgeColor   string // Tailwind color prefix for badge (e.g. "amber", "green")
	OperatorName string // Optional: handle of community operator (e.g. "@baileytownsend.dev")
	OperatorURL  string // Optional: link to operator profile
	SignupURL    string // Optional: if set, link directly to this URL instead of using prompt=create
}

// Category groups providers under a heading.
type Category struct {
	Title       string
	Description string
	Providers   []Provider
	DevOnly     bool // If true, only shown to clients with devMode enabled
}

// Categories returns the list of PDS provider categories shown on the
// create account page. This is the single source of truth for both the
// rendered view and the server-side allowlist. When devMode is false,
// categories flagged DevOnly are excluded.
func Categories(devMode bool) []Category {
	all := allCategories()
	if devMode {
		return all
	}
	out := make([]Category, 0, len(all))
	for _, c := range all {
		if c.DevOnly {
			continue
		}
		out = append(out, c)
	}
	return out
}

func allCategories() []Category {
	return []Category{
		{
			Title:       "Recommended",
			Description: "A reliable, open community provider — a great default if you're unsure where to start.",
			Providers: []Provider{
				{
					URL:          "https://selfhosted.social",
					Name:         "selfhosted.social",
					Domain:       "selfhosted.social",
					Description:  "Community provider",
					Location:     "United States",
					Badge:        "Open",
					BadgeColor:   "green",
					OperatorName: "@baileytownsend.dev",
					OperatorURL:  "https://bsky.app/profile/baileytownsend.dev",
				},
			},
		},
		{
			Title:       "App Providers",
			Description: "These apps host your account and data for you.",
			Providers: []Provider{
				{
					URL:         "https://arabica.systems",
					Name:        "Arabica",
					Domain:      "arabica.systems",
					Description: "The official Arabica provider.",
					Location:    "United States",
					Badge:       "Invite Only",
					BadgeColor:  "amber",
				},
				{
					URL:         "https://bsky.social",
					Name:        "Bluesky",
					Domain:      "bsky.social",
					Description: "The largest PDS provider.",
					Location:    "United States",
					Badge:       "Open",
					BadgeColor:  "green",
					SignupURL:   "https://bsky.app",
				},
				{
					URL:         "https://npmx.social",
					Name:        "npmx",
					Domain:      "npmx.social",
					Description: "Developer-focused Community provider.",
					Location:    "Europe",
					Badge:       "Open",
					BadgeColor:  "green",
					SignupURL:   "https://npmx.social",
				},
			},
		},
		{
			Title:       "Developer",
			Description: "Experimental providers for testing. Only shown when developer mode is enabled.",
			DevOnly:     true,
			Providers: []Provider{
				{
					URL:         "https://pds.rip",
					Name:        "pds.rip",
					Domain:      "pds.rip",
					Description: "Experimental PDS for developers.",
					Location:    "United States",
					Badge:       "Dev",
					BadgeColor:  "amber",
				},
			},
		},
		{
			Title:       "Independent Providers",
			Description: "Account hosts run by the community. They hold your account and data, independent of any single app.",
			Providers: []Provider{
				{
					URL:          "https://selfhosted.social",
					Name:         "selfhosted.social",
					Domain:       "selfhosted.social",
					Description:  "Community provider",
					Location:     "United States",
					Badge:        "Open",
					BadgeColor:   "green",
					OperatorName: "@baileytownsend.dev",
					OperatorURL:  "https://bsky.app/profile/baileytownsend.dev",
				},
				{
					URL:         "https://eurosky.social",
					Name:        "Eurosky",
					Domain:      "eurosky.social",
					Description: "Sovereign European PDS hosting.",
					Location:    "Europe",
					Badge:       "Open",
					BadgeColor:  "green",
					SignupURL:   "https://portal.eurosky.tech/create-account",
				},
			},
		},
	}
}

// IsAllowedPDSURL reports whether the given PDS URL is in the catalog as
// a prompt=create destination (i.e. a provider without an external
// SignupURL override). External-redirect providers are excluded because
// they never POST to the signup handler. DevOnly categories are only
// considered when devMode is true.
func IsAllowedPDSURL(url string, devMode bool) bool {
	for _, cat := range Categories(devMode) {
		for _, p := range cat.Providers {
			if p.SignupURL == "" && p.URL == url {
				return true
			}
		}
	}
	return false
}
