package arabica_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/arabica.social/arabica/internal/suggestions"
)

// fakeSource is an in-memory suggestions.RecordSource for testing the
// dedup/search logic against arabica's registered configs. Tests live
// in this package (not the suggestions package) so the integration
// tests sit alongside the configs they exercise — and so the
// suggestions package itself can stay free of any app-specific
// imports.
type fakeSource struct {
	records map[string][]suggestions.IndexedRecord // collection -> records
	refs    map[string]int                         // URI -> reference count
}

func newFakeSource() *fakeSource {
	return &fakeSource{
		records: map[string][]suggestions.IndexedRecord{},
		refs:    map[string]int{},
	}
}

func (f *fakeSource) ListRecordsByCollectionOldest(_ context.Context, collection string) ([]suggestions.IndexedRecord, error) {
	return f.records[collection], nil
}

func (f *fakeSource) CountReferencesToURI(_ context.Context, uri string) (int, error) {
	return f.refs[uri], nil
}

func (f *fakeSource) insert(t *testing.T, did, collection, rkey string, fields map[string]any) {
	t.Helper()
	fields["$type"] = collection
	data, err := json.Marshal(fields)
	assert.NoError(t, err)
	uri := "at://" + did + "/" + collection + "/" + rkey
	f.records[collection] = append(f.records[collection], suggestions.IndexedRecord{
		URI:    uri,
		DID:    did,
		Record: data,
	})
}

// --- Roaster dedup tests ---

func TestRoasterDedup_SameNameDifferentLocation(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDRoaster, "r1", map[string]any{
		"name":     "Stumptown Coffee",
		"location": "Portland, OR",
	})
	idx.insert(t, "did:plc:bob", arabica.NSIDRoaster, "r2", map[string]any{
		"name":     "Stumptown Coffee",
		"location": "New York, NY",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDRoaster, "stumptown", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2, "different locations should produce separate suggestions")
}

func TestRoasterDedup_FuzzyNameMerge(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDRoaster, "r1", map[string]any{
		"name":     "Counter Culture Coffee",
		"location": "Durham, NC",
		"website":  "https://counterculturecoffee.com",
	})
	idx.insert(t, "did:plc:bob", arabica.NSIDRoaster, "r2", map[string]any{
		"name":     "Counter Culture Coffee Roasters",
		"location": "Durham, NC",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDRoaster, "counter", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1, "fuzzy name + same location should merge")
	assert.Equal(t, 2, results[0].Count)
}

func TestRoasterDedup_NoLocationMerges(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDRoaster, "r1", map[string]any{"name": "Blue Bottle Coffee"})
	idx.insert(t, "did:plc:bob", arabica.NSIDRoaster, "r2", map[string]any{"name": "Blue Bottle"})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDRoaster, "blue", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1, "same fuzzy name with no location should merge")
	assert.Equal(t, 2, results[0].Count)
}

// --- Grinder dedup tests ---

func TestGrinderDedup_SameNameDifferentType(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDGrinder, "g1", map[string]any{
		"name": "Baratza Encore", "grinderType": "electric", "burrType": "conical",
	})
	idx.insert(t, "did:plc:bob", arabica.NSIDGrinder, "g2", map[string]any{
		"name": "Baratza Encore", "grinderType": "electric", "burrType": "flat",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDGrinder, "baratza", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestGrinderDedup_SameEverythingMerges(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDGrinder, "g1", map[string]any{
		"name": "1Zpresso JX Pro", "grinderType": "hand", "burrType": "conical",
	})
	idx.insert(t, "did:plc:bob", arabica.NSIDGrinder, "g2", map[string]any{
		"name": "1Zpresso JX Pro", "grinderType": "hand", "burrType": "conical",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDGrinder, "1zp", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 2, results[0].Count)
}

// --- Brewer dedup tests ---

func TestBrewerDedup_SameNameDifferentType(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDBrewer, "br1", map[string]any{
		"name": "Hario V60", "brewerType": "pour-over",
	})
	idx.insert(t, "did:plc:bob", arabica.NSIDBrewer, "br2", map[string]any{
		"name": "Hario V60", "brewerType": "dripper",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDBrewer, "hario", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestBrewerDedup_SameNameSameTypeMerges(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDBrewer, "br1", map[string]any{
		"name": "AeroPress", "brewerType": "immersion",
	})
	idx.insert(t, "did:plc:bob", arabica.NSIDBrewer, "br2", map[string]any{
		"name": "AeroPress", "brewerType": "immersion",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDBrewer, "aero", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 2, results[0].Count)
}

// --- Bean dedup tests ---

func TestBeanDedup_SameNameDifferentProcess(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDBean, "b1", map[string]any{
		"name": "Yirgacheffe", "origin": "Ethiopia", "process": "Washed",
	})
	idx.insert(t, "did:plc:bob", arabica.NSIDBean, "b2", map[string]any{
		"name": "Yirgacheffe", "origin": "Ethiopia", "process": "Natural",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDBean, "yirga", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestBeanDedup_SameNameDifferentOrigin(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDBean, "b1", map[string]any{"name": "Gesha", "origin": "Panama"})
	idx.insert(t, "did:plc:bob", arabica.NSIDBean, "b2", map[string]any{"name": "Gesha", "origin": "Ethiopia"})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDBean, "gesha", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestBeanDedup_SameEverythingMerges(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDBean, "b1", map[string]any{
		"name": "Ethiopian Yirgacheffe", "origin": "Ethiopia", "roastLevel": "Light", "process": "Washed",
	})
	idx.insert(t, "did:plc:bob", arabica.NSIDBean, "b2", map[string]any{
		"name": "Ethiopian Yirgacheffe", "origin": "Ethiopia", "process": "Washed",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDBean, "ethiopia", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 2, results[0].Count)
}

// --- General search tests ---

func TestSearch_PrefixMatch(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDRoaster, "r1", map[string]any{
		"name": "Black & White Coffee", "location": "Raleigh, NC",
	})
	idx.insert(t, "did:plc:bob", arabica.NSIDRoaster, "r2", map[string]any{
		"name": "Blue Bottle", "location": "Oakland, CA",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDRoaster, "bl", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Black & White Coffee", results[0].Name)
	assert.Equal(t, "Blue Bottle", results[1].Name)
}

func TestSearch_CaseInsensitive(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDRoaster, "r1", map[string]any{"name": "Stumptown Coffee"})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDRoaster, "STUMP", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Stumptown Coffee", results[0].Name)
}

func TestSearch_SubstringMatch(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDRoaster, "r1", map[string]any{
		"name": "Red Rooster Coffee", "location": "Floyd, VA",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDRoaster, "floyd", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestSearch_Deduplication(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDRoaster, "r1", map[string]any{
		"name": "Counter Culture Coffee", "location": "Durham, NC", "website": "https://counterculturecoffee.com",
	})
	idx.insert(t, "did:plc:bob", arabica.NSIDRoaster, "r2", map[string]any{
		"name": "Counter Culture", "location": "Durham, NC",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDRoaster, "counter", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 2, results[0].Count)
	assert.Equal(t, "Durham, NC", results[0].Fields["location"])
}

func TestSearch_Limit(t *testing.T) {
	idx := newFakeSource()
	for i := range 5 {
		rkey := "r" + string(rune('0'+i))
		idx.insert(t, "did:plc:alice", arabica.NSIDGrinder, rkey, map[string]any{
			"name": "Grinder " + string(rune('A'+i)), "grinderType": "hand",
		})
	}

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDGrinder, "grinder", 3)
	assert.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestSearch_ShortQuery(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDRoaster, "r1", map[string]any{"name": "ABC"})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDRoaster, "a", 10)
	assert.NoError(t, err)
	assert.Empty(t, results)

	results, err = suggestions.Search(context.Background(), idx, arabica.NSIDRoaster, "ab", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestSearch_EmptyQuery(t *testing.T) {
	idx := newFakeSource()
	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDRoaster, "", 10)
	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearch_UnknownCollection(t *testing.T) {
	idx := newFakeSource()
	results, err := suggestions.Search(context.Background(), idx, "unknown.collection", "test", 10)
	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearch_GrinderFields(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDGrinder, "g1", map[string]any{
		"name": "1Zpresso JX Pro", "grinderType": "hand", "burrType": "conical",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDGrinder, "1zp", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "hand", results[0].Fields["grinderType"])
	assert.Equal(t, "conical", results[0].Fields["burrType"])
}

func TestSearch_BeanFields(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDBean, "b1", map[string]any{
		"name": "Ethiopian Yirgacheffe", "origin": "Ethiopia", "roastLevel": "Light", "process": "Washed",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDBean, "ethiopia", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Light", results[0].Fields["roastLevel"])
}

func TestSearch_BeanRoasterNameResolved(t *testing.T) {
	idx := newFakeSource()
	roasterURI := "at://did:plc:alice/" + arabica.NSIDRoaster + "/r1"
	idx.insert(t, "did:plc:alice", arabica.NSIDRoaster, "r1", map[string]any{"name": "Counter Culture"})
	idx.insert(t, "did:plc:alice", arabica.NSIDBean, "b1", map[string]any{
		"name": "Hologram", "origin": "Blend", "roasterRef": roasterURI,
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDBean, "hologram", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Counter Culture", results[0].Fields["roasterName"])
}

func TestSearch_BeanNoRoasterRef(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDBean, "b1", map[string]any{
		"name": "Mystery Bean", "origin": "Unknown",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDBean, "mystery", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Empty(t, results[0].Fields["roasterName"])
}

func TestSearch_BrewerFields(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDBrewer, "br1", map[string]any{
		"name": "Hario V60", "brewerType": "Pour-Over",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDBrewer, "hario", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Pour-Over", results[0].Fields["brewerType"])
}

func TestRecipeDedup_SameNameDifferentBrewerType(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDRecipe, "rec1", map[string]any{
		"name": "V60 Standard", "brewerType": "pourover",
	})
	idx.insert(t, "did:plc:bob", arabica.NSIDRecipe, "rec2", map[string]any{
		"name": "V60 Standard", "brewerType": "immersion",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDRecipe, "v60", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestRecipeDedup_SameNameSameTypeMerges(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDRecipe, "rec1", map[string]any{
		"name": "AeroPress Standard", "brewerType": "immersion",
	})
	idx.insert(t, "did:plc:bob", arabica.NSIDRecipe, "rec2", map[string]any{
		"name": "AeroPress Standard", "brewerType": "immersion",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDRecipe, "aero", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 2, results[0].Count)
}

func TestSearch_RecipeFields(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDRecipe, "rec1", map[string]any{
		"name": "James Hoffmann V60", "brewerType": "pourover",
	})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDRecipe, "hoffmann", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "pourover", results[0].Fields["brewerType"])
}

func TestSearch_SortOrder(t *testing.T) {
	idx := newFakeSource()
	idx.insert(t, "did:plc:alice", arabica.NSIDRoaster, "r1", map[string]any{"name": "Alpha Roasters"})
	idx.insert(t, "did:plc:bob", arabica.NSIDRoaster, "r2", map[string]any{"name": "Alpha Roasters"})
	idx.insert(t, "did:plc:charlie", arabica.NSIDRoaster, "r3", map[string]any{"name": "Alpha Roasters"})
	idx.insert(t, "did:plc:dave", arabica.NSIDRoaster, "r4", map[string]any{"name": "Alpha Beta"})

	results, err := suggestions.Search(context.Background(), idx, arabica.NSIDRoaster, "alpha", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Alpha Roasters", results[0].Name)
	assert.Equal(t, 3, results[0].Count)
	assert.Equal(t, "Alpha Beta", results[1].Name)
	assert.Equal(t, 1, results[1].Count)
}
