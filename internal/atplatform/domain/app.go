// Package domain holds the per-binary App configuration shared between
// arabica and any sister app (oolong, etc.). Every code path that needs to
// know what entities or NSIDs the app cares about reads them from App.
package domain

import (
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

type App struct {
	Name        string
	NSIDBase    string
	Descriptors []*entities.Descriptor
	Brand       BrandConfig
}

type BrandConfig struct {
	DisplayName string
	Tagline     string
}

func (a *App) NSIDs() []string {
	out := make([]string, 0, len(a.Descriptors)+2)
	for _, d := range a.Descriptors {
		out = append(out, d.NSID)
	}
	out = append(out, a.NSIDBase+".like")
	out = append(out, a.NSIDBase+".comment")
	return out
}

// LikeNSID returns the like collection NSID for this app.
func (a *App) LikeNSID() string {
	return a.NSIDBase + ".like"
}

// CommentNSID returns the comment collection NSID for this app.
func (a *App) CommentNSID() string {
	return a.NSIDBase + ".comment"
}

func (a *App) OAuthScopes() []string {
	nsids := a.NSIDs()
	out := make([]string, 0, len(nsids)+1)
	out = append(out, "atproto")
	for _, nsid := range nsids {
		out = append(out, "repo:"+nsid)
	}
	return out
}

// BlueskyProfileScopes are the extra OAuth scopes required to read and
// update the user's Bluesky profile record (display name, description,
// avatar, banner) on their PDS. Requested incrementally via a scope-upgrade
// reauth flow when a user opts in to editing their Bluesky profile from
// arabica — not asked for at initial login.
func BlueskyProfileScopes() []string {
	return []string{
		"repo:app.bsky.actor.profile",
		"blob:image/*",
	}
}

// OAuthScopesWithProfile returns the union of OAuthScopes and
// BlueskyProfileScopes. This is the full superset declared in client
// metadata and requested by the scope-upgrade flow.
func (a *App) OAuthScopesWithProfile() []string {
	base := a.OAuthScopes()
	extra := BlueskyProfileScopes()
	out := make([]string, 0, len(base)+len(extra))
	out = append(out, base...)
	out = append(out, extra...)
	return out
}

// HasBlueskyProfileScopes reports whether the given scope list includes the
// scopes needed to edit the Bluesky profile record.
func HasBlueskyProfileScopes(scopes []string) bool {
	have := make(map[string]struct{}, len(scopes))
	for _, s := range scopes {
		have[s] = struct{}{}
	}
	for _, need := range BlueskyProfileScopes() {
		if _, ok := have[need]; !ok {
			return false
		}
	}
	return true
}

func (a *App) DescriptorByNSID(nsid string) *entities.Descriptor {
	for _, d := range a.Descriptors {
		if d.NSID == nsid {
			return d
		}
	}
	return nil
}

// DescriptorByType returns the descriptor whose RecordType matches, or
// nil if the app doesn't run that entity. Used by route registration
// and any handler that wants to gate behaviour on per-app entity
// availability.
func (a *App) DescriptorByType(rt lexicons.RecordType) *entities.Descriptor {
	for _, d := range a.Descriptors {
		if d.Type == rt {
			return d
		}
	}
	return nil
}
