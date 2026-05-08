// Package domain holds the per-binary App configuration shared between
// arabica and any sister app (matcha, etc.). Every code path that needs to
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

func (a *App) OAuthScopes() []string {
	nsids := a.NSIDs()
	out := make([]string, 0, len(nsids)+1)
	out = append(out, "atproto")
	for _, nsid := range nsids {
		out = append(out, "repo:"+nsid)
	}
	return out
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
