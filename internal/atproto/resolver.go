package atproto

import (
	"context"
	"fmt"

	"tangled.org/arabica.social/arabica/internal/entities/arabica"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"tangled.org/pdewey.com/atp"
)

// ATURIComponents holds the parsed components of an AT-URI
type ATURIComponents struct {
	DID        string
	Collection string
	RKey       string
}

// ResolveATURI parses an AT-URI and returns its components.
// AT-URI format: at://did:plc:abc123/social.arabica.brew/3jxyabc
func ResolveATURI(uri string) (*ATURIComponents, error) {
	did, collection, rkey, err := atp.ParseATURI(uri)
	if err != nil {
		return nil, err
	}

	return &ATURIComponents{
		DID:        did,
		Collection: collection,
		RKey:       rkey,
	}, nil
}

// resolveRef is a generic helper that fetches and converts a record from an AT-URI
func resolveRef[T any](
	ctx context.Context,
	client *Client,
	atURI string,
	sessionID string,
	expectedCollection string,
	convert func(map[string]any, string) (*T, error),
) (*T, error) {
	if atURI == "" {
		return nil, nil
	}

	components, err := ResolveATURI(atURI)
	if err != nil {
		return nil, err
	}

	if components.Collection != expectedCollection {
		return nil, fmt.Errorf("expected %s collection, got %s", expectedCollection, components.Collection)
	}

	didObj, err := syntax.ParseDID(components.DID)
	if err != nil {
		return nil, fmt.Errorf("invalid DID: %w", err)
	}

	output, err := client.GetRecord(ctx, didObj, sessionID, &GetRecordInput{
		Collection: components.Collection,
		RKey:       components.RKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s record: %w", expectedCollection, err)
	}

	result, err := convert(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert %s record: %w", expectedCollection, err)
	}

	return result, nil
}

// ResolveBeanRefWithRoaster fetches a bean record and also resolves its roaster reference
func ResolveBeanRefWithRoaster(ctx context.Context, client *Client, atURI string, sessionID string) (*arabica.Bean, error) {
	if atURI == "" {
		return nil, nil
	}

	components, err := ResolveATURI(atURI)
	if err != nil {
		return nil, err
	}

	if components.Collection != arabica.NSIDBean {
		return nil, fmt.Errorf("expected %s collection, got %s", arabica.NSIDBean, components.Collection)
	}

	didObj, err := syntax.ParseDID(components.DID)
	if err != nil {
		return nil, fmt.Errorf("invalid DID: %w", err)
	}

	output, err := client.GetRecord(ctx, didObj, sessionID, &GetRecordInput{
		Collection: components.Collection,
		RKey:       components.RKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bean record: %w", err)
	}

	bean, err := arabica.RecordToBean(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert bean record: %w", err)
	}

	// Extract and resolve roaster reference if present
	if roasterRef, ok := output.Value["roasterRef"].(string); ok && roasterRef != "" {
		// Extract rkey
		if roasterComponents, err := ResolveATURI(roasterRef); err == nil {
			bean.RoasterRKey = roasterComponents.RKey
		}
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
	return resolveRef(ctx, client, atURI, sessionID, arabica.NSIDRoaster, arabica.RecordToRoaster)
}

// ResolveGrinderRef fetches a grinder record from an AT-URI
func ResolveGrinderRef(ctx context.Context, client *Client, atURI string, sessionID string) (*arabica.Grinder, error) {
	return resolveRef(ctx, client, atURI, sessionID, arabica.NSIDGrinder, arabica.RecordToGrinder)
}

// ResolveBrewerRef fetches a brewer record from an AT-URI
func ResolveBrewerRef(ctx context.Context, client *Client, atURI string, sessionID string) (*arabica.Brewer, error) {
	return resolveRef(ctx, client, atURI, sessionID, arabica.NSIDBrewer, arabica.RecordToBrewer)
}

// ResolveRecipeRef fetches a recipe record from an AT-URI
func ResolveRecipeRef(ctx context.Context, client *Client, atURI string, sessionID string) (*arabica.Recipe, error) {
	return resolveRef(ctx, client, atURI, sessionID, arabica.NSIDRecipe, arabica.RecordToRecipe)
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
