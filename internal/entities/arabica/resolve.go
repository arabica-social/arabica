package arabica

import (
	"context"
	"fmt"

	"tangled.org/pdewey.com/atp"
)

// ResolveBean fetches a Bean record from its AT-URI and resolves its nested
// roaster reference. The roaster resolution is best-effort; a missing or
// unresolvable roaster does not fail the bean fetch.
func ResolveBean(ctx context.Context, client *atp.Client, atURI string) (*Bean, error) {
	if atURI == "" {
		return nil, nil
	}

	// Fetch raw record so we can extract the roasterRef before converting.
	// RecordToBean doesn't store the raw roasterRef URI, only the parsed RKey.
	uri, err := atp.ParseATURI(atURI)
	if err != nil {
		return nil, err
	}
	if uri.Collection != NSIDBean {
		return nil, fmt.Errorf("expected %s collection, got %s", NSIDBean, uri.Collection)
	}

	rec, err := client.GetRecord(ctx, uri.Collection, uri.RKey)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bean record: %w", err)
	}

	bean, err := RecordToBean(rec.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert bean record: %w", err)
	}

	// Extract and resolve roaster reference if present.
	if roasterRef, ok := rec.Value["roasterRef"].(string); ok && roasterRef != "" {
		bean.RoasterRKey = atp.RKeyFromURI(roasterRef)
		bean.Roaster, err = ResolveRoaster(ctx, client, roasterRef)
		if err != nil {
			// Log but don't fail — roaster resolution is optional
			return bean, nil
		}
	}

	return bean, nil
}

// ResolveRoaster fetches a Roaster record from its AT-URI.
func ResolveRoaster(ctx context.Context, client *atp.Client, atURI string) (*Roaster, error) {
	return atp.ResolveRecord(ctx, client, atURI, NSIDRoaster, RecordToRoaster)
}

// ResolveGrinder fetches a Grinder record from its AT-URI.
func ResolveGrinder(ctx context.Context, client *atp.Client, atURI string) (*Grinder, error) {
	return atp.ResolveRecord(ctx, client, atURI, NSIDGrinder, RecordToGrinder)
}

// ResolveBrewer fetches a Brewer record from its AT-URI.
func ResolveBrewer(ctx context.Context, client *atp.Client, atURI string) (*Brewer, error) {
	return atp.ResolveRecord(ctx, client, atURI, NSIDBrewer, RecordToBrewer)
}

// ResolveRecipe fetches a Recipe record from its AT-URI.
func ResolveRecipe(ctx context.Context, client *atp.Client, atURI string) (*Recipe, error) {
	return atp.ResolveRecord(ctx, client, atURI, NSIDRecipe, RecordToRecipe)
}

// ResolveBrewRefs resolves the bean, grinder, and brewer references on a Brew.
// Recipe resolution is handled separately by the caller (it may live on a
// different PDS and requires different client scoping).
func ResolveBrewRefs(ctx context.Context, client *atp.Client, brew *Brew, beanRef, grinderRef, brewerRef string) error {
	var err error

	// Resolve bean reference (required) — also resolves nested roaster
	if beanRef != "" {
		brew.Bean, err = ResolveBean(ctx, client, beanRef)
		if err != nil {
			return fmt.Errorf("failed to resolve bean reference: %w", err)
		}
	}

	// Resolve grinder reference (optional)
	if grinderRef != "" {
		brew.GrinderObj, err = ResolveGrinder(ctx, client, grinderRef)
		if err != nil {
			return fmt.Errorf("failed to resolve grinder reference: %w", err)
		}
	}

	// Resolve brewer reference (optional)
	if brewerRef != "" {
		brew.BrewerObj, err = ResolveBrewer(ctx, client, brewerRef)
		if err != nil {
			return fmt.Errorf("failed to resolve brewer reference: %w", err)
		}
	}

	return nil
}
