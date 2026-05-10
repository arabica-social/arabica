package atproto

import (
	"context"
	"fmt"

	"tangled.org/arabica.social/arabica/internal/entities/arabica"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"tangled.org/pdewey.com/atp"
)

// atpClientForURI gets an *atp.Client scoped to the DID in the given AT-URI.
func atpClientForURI(ctx context.Context, client *Client, atURI string, sessionID string) (*atp.Client, error) {
	u, err := atp.ParseATURI(atURI)
	if err != nil {
		return nil, err
	}
	didObj, err := syntax.ParseDID(u.DID)
	if err != nil {
		return nil, fmt.Errorf("invalid DID: %w", err)
	}
	return client.AtpClient(ctx, didObj, sessionID)
}

// ResolveBeanRefWithRoaster fetches a bean record and also resolves its roaster reference
func ResolveBeanRefWithRoaster(ctx context.Context, client *Client, atURI string, sessionID string) (*arabica.Bean, error) {
	if atURI == "" {
		return nil, nil
	}

	atpClient, err := atpClientForURI(ctx, client, atURI, sessionID)
	if err != nil {
		return nil, err
	}

	// Fetch raw record so we can extract the roasterRef before converting.
	// RecordToBean doesn't store the raw roasterRef URI, only the parsed RKey.
	beanURI, err := atp.ParseATURI(atURI)
	if err != nil {
		return nil, err
	}
	if beanURI.Collection != arabica.NSIDBean {
		return nil, fmt.Errorf("expected %s collection, got %s", arabica.NSIDBean, beanURI.Collection)
	}

	rec, err := atpClient.GetRecord(ctx, beanURI.Collection, beanURI.RKey)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bean record: %w", err)
	}

	// Convert raw record to typed model
	bean, err := arabica.RecordToBean(rec.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert bean record: %w", err)
	}

	// Extract and resolve roaster reference if present.
	// Needs to come from the raw record since the Bean model doesn't store
	// the full roasterRef URI, only its parsed RKey (RoasterRKey).
	if roasterRef, ok := rec.Value["roasterRef"].(string); ok && roasterRef != "" {
		bean.RoasterRKey = atp.RKeyFromURI(roasterRef)
		// Resolve the roaster
		bean.Roaster, err = ResolveRoasterRef(ctx, client, roasterRef, sessionID)
		if err != nil {
			// Log but don't fail - roaster resolution is optional
			return bean, nil
		}
	}

	return bean, nil
}

// ResolveRoasterRef fetches a roaster record from an AT-URI
func ResolveRoasterRef(ctx context.Context, client *Client, atURI string, sessionID string) (*arabica.Roaster, error) {
	atpClient, err := atpClientForURI(ctx, client, atURI, sessionID)
	if err != nil {
		return nil, err
	}
	return atp.ResolveRecord(ctx, atpClient, atURI, arabica.NSIDRoaster, arabica.RecordToRoaster)
}

// ResolveGrinderRef fetches a grinder record from an AT-URI
func ResolveGrinderRef(ctx context.Context, client *Client, atURI string, sessionID string) (*arabica.Grinder, error) {
	atpClient, err := atpClientForURI(ctx, client, atURI, sessionID)
	if err != nil {
		return nil, err
	}
	return atp.ResolveRecord(ctx, atpClient, atURI, arabica.NSIDGrinder, arabica.RecordToGrinder)
}

// ResolveBrewerRef fetches a brewer record from an AT-URI
func ResolveBrewerRef(ctx context.Context, client *Client, atURI string, sessionID string) (*arabica.Brewer, error) {
	atpClient, err := atpClientForURI(ctx, client, atURI, sessionID)
	if err != nil {
		return nil, err
	}
	return atp.ResolveRecord(ctx, atpClient, atURI, arabica.NSIDBrewer, arabica.RecordToBrewer)
}

// ResolveRecipeRef fetches a recipe record from an AT-URI
func ResolveRecipeRef(ctx context.Context, client *Client, atURI string, sessionID string) (*arabica.Recipe, error) {
	atpClient, err := atpClientForURI(ctx, client, atURI, sessionID)
	if err != nil {
		return nil, err
	}
	return atp.ResolveRecord(ctx, atpClient, atURI, arabica.NSIDRecipe, arabica.RecordToRecipe)
}

// ResolveBrewRefs resolves all references within a brew record
// This is a convenience function that resolves bean, grinder, brewer, and recipe refs in one call
func ResolveBrewRefs(ctx context.Context, client *Client, brew *arabica.Brew, beanRef, grinderRef, brewerRef, sessionID string) error {
	var err error

	// Resolve bean reference (required) - also resolves nested roaster
	if beanRef != "" {
		brew.Bean, err = ResolveBeanRefWithRoaster(ctx, client, beanRef, sessionID)
		if err != nil {
			return fmt.Errorf("failed to resolve bean reference: %w", err)
		}
	}

	// Resolve grinder reference (optional)
	if grinderRef != "" {
		brew.GrinderObj, err = ResolveGrinderRef(ctx, client, grinderRef, sessionID)
		if err != nil {
			return fmt.Errorf("failed to resolve grinder reference: %w", err)
		}
	}

	// Resolve brewer reference (optional)
	if brewerRef != "" {
		brew.BrewerObj, err = ResolveBrewerRef(ctx, client, brewerRef, sessionID)
		if err != nil {
			return fmt.Errorf("failed to resolve brewer reference: %w", err)
		}
	}

	return nil
}
