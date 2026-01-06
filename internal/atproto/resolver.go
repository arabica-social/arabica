package atproto

import (
	"context"
	"fmt"

	"arabica/internal/models"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// ResolveATURI parses an AT-URI and returns its components
// AT-URI format: at://did:plc:abc123/com.arabica.brew/3jxyabc
func ResolveATURI(uri string) (did string, collection string, rkey string, err error) {
	atURI, err := syntax.ParseATURI(uri)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid AT-URI: %w", err)
	}

	did = atURI.Authority().String()
	collection = atURI.Collection().String()
	rkey = atURI.RecordKey().String()

	return did, collection, rkey, nil
}

// ResolveBeanRef fetches a bean record from an AT-URI
// This performs a network call to the user's PDS to fetch the referenced bean
func ResolveBeanRef(ctx context.Context, client *Client, atURI string, sessionID string) (*models.Bean, error) {
	if atURI == "" {
		return nil, nil // No reference to resolve
	}

	// Parse the AT-URI to extract components
	did, collection, rkey, err := ResolveATURI(atURI)
	if err != nil {
		return nil, err
	}

	// Validate it's the right collection type
	if collection != "com.arabica.bean" {
		return nil, fmt.Errorf("expected com.arabica.bean collection, got %s", collection)
	}

	// Fetch the record from the PDS
	didObj, err := syntax.ParseDID(did)
	if err != nil {
		return nil, fmt.Errorf("invalid DID: %w", err)
	}

	output, err := client.GetRecord(ctx, didObj, sessionID, &GetRecordInput{
		Collection: collection,
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bean record: %w", err)
	}

	// Convert the record to a Bean model
	bean, err := RecordToBean(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert bean record: %w", err)
	}

	return bean, nil
}

// ResolveRoasterRef fetches a roaster record from an AT-URI
func ResolveRoasterRef(ctx context.Context, client *Client, atURI string, sessionID string) (*models.Roaster, error) {
	if atURI == "" {
		return nil, nil
	}

	did, collection, rkey, err := ResolveATURI(atURI)
	if err != nil {
		return nil, err
	}

	if collection != "com.arabica.roaster" {
		return nil, fmt.Errorf("expected com.arabica.roaster collection, got %s", collection)
	}

	didObj, err := syntax.ParseDID(did)
	if err != nil {
		return nil, fmt.Errorf("invalid DID: %w", err)
	}

	output, err := client.GetRecord(ctx, didObj, sessionID, &GetRecordInput{
		Collection: collection,
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch roaster record: %w", err)
	}

	roaster, err := RecordToRoaster(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert roaster record: %w", err)
	}

	return roaster, nil
}

// ResolveGrinderRef fetches a grinder record from an AT-URI
func ResolveGrinderRef(ctx context.Context, client *Client, atURI string, sessionID string) (*models.Grinder, error) {
	if atURI == "" {
		return nil, nil
	}

	did, collection, rkey, err := ResolveATURI(atURI)
	if err != nil {
		return nil, err
	}

	if collection != "com.arabica.grinder" {
		return nil, fmt.Errorf("expected com.arabica.grinder collection, got %s", collection)
	}

	didObj, err := syntax.ParseDID(did)
	if err != nil {
		return nil, fmt.Errorf("invalid DID: %w", err)
	}

	output, err := client.GetRecord(ctx, didObj, sessionID, &GetRecordInput{
		Collection: collection,
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grinder record: %w", err)
	}

	grinder, err := RecordToGrinder(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert grinder record: %w", err)
	}

	return grinder, nil
}

// ResolveBrewerRef fetches a brewer record from an AT-URI
func ResolveBrewerRef(ctx context.Context, client *Client, atURI string, sessionID string) (*models.Brewer, error) {
	if atURI == "" {
		return nil, nil
	}

	did, collection, rkey, err := ResolveATURI(atURI)
	if err != nil {
		return nil, err
	}

	if collection != "com.arabica.brewer" {
		return nil, fmt.Errorf("expected com.arabica.brewer collection, got %s", collection)
	}

	didObj, err := syntax.ParseDID(did)
	if err != nil {
		return nil, fmt.Errorf("invalid DID: %w", err)
	}

	output, err := client.GetRecord(ctx, didObj, sessionID, &GetRecordInput{
		Collection: collection,
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch brewer record: %w", err)
	}

	brewer, err := RecordToBrewer(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert brewer record: %w", err)
	}

	return brewer, nil
}

// ResolveBrewRefs resolves all references within a brew record
// This is a convenience function that resolves bean, grinder, and brewer refs in one call
func ResolveBrewRefs(ctx context.Context, client *Client, brew *models.Brew, beanRef, grinderRef, brewerRef, sessionID string) error {
	var err error

	// Resolve bean reference (required)
	if beanRef != "" {
		brew.Bean, err = ResolveBeanRef(ctx, client, beanRef, sessionID)
		if err != nil {
			return fmt.Errorf("failed to resolve bean reference: %w", err)
		}

		// If bean has a roaster reference, resolve it too
		if brew.Bean != nil && brew.Bean.RoasterID != nil {
			// Note: We need to get the roasterRef from the bean record
			// This requires storing the raw record data or fetching it again
			// For now, we'll skip nested resolution and handle it in store.go
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
