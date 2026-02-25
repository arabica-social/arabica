package suggestions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"arabica/internal/atproto"
	"arabica/internal/firehose"

	"github.com/stretchr/testify/assert"
)

func newTestFeedIndex(t *testing.T) *firehose.FeedIndex {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test-index.db")
	idx, err := firehose.NewFeedIndex(path, 1*time.Hour)
	assert.NoError(t, err)
	t.Cleanup(func() {
		idx.Close()
		os.Remove(path)
	})
	return idx
}

func insertRecord(t *testing.T, idx *firehose.FeedIndex, did, collection, rkey string, fields map[string]interface{}) {
	t.Helper()
	fields["$type"] = collection
	fields["createdAt"] = time.Now().Format(time.RFC3339)
	data, err := json.Marshal(fields)
	assert.NoError(t, err)
	err = idx.UpsertRecord(did, collection, rkey, "cid-"+rkey, data, 0)
	assert.NoError(t, err)
}

// --- Helper unit tests ---

func TestFuzzyName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Counter Culture Coffee", "counter culture"},
		{"Counter Culture", "counter culture"},
		{"counter culture coffee roasters", "counter culture"},
		{"Stumptown Coffee Roasters", "stumptown"},
		{"Stumptown", "stumptown"},
		{"Black & White Coffee", "black white"},
		{"  Some  Roasting Company  ", "some"},
		{"Heart Coffee Roasters", "heart"},
		{"Heart Roasters", "heart"},
		{"Heart", "heart"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, fuzzyName(tt.input), "fuzzyName(%q)", tt.input)
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://www.counterculturecoffee.com/shop", "counterculturecoffee.com"},
		{"http://example.com", "example.com"},
		{"https://example.com:8080/path", "example.com"},
		{"www.example.com", "example.com"},
		{"example.com", "example.com"},
		{"", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, extractDomain(tt.input), "extractDomain(%q)", tt.input)
	}
}

func TestNormalize(t *testing.T) {
	assert.Equal(t, "durham, nc", normalize("  Durham,  NC  "))
	assert.Equal(t, "oakland ca", normalize("Oakland CA"))
	assert.Equal(t, "", normalize(""))
}

// --- Roaster dedup tests ---

func TestRoasterDedup_SameNameDifferentLocation(t *testing.T) {
	idx := newTestFeedIndex(t)

	insertRecord(t, idx, "did:plc:alice", atproto.NSIDRoaster, "r1", map[string]interface{}{
		"name":     "Stumptown Coffee",
		"location": "Portland, OR",
	})
	insertRecord(t, idx, "did:plc:bob", atproto.NSIDRoaster, "r2", map[string]interface{}{
		"name":     "Stumptown Coffee",
		"location": "New York, NY",
	})

	results, err := Search(idx, atproto.NSIDRoaster, "stumptown", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2, "different locations should produce separate suggestions")
}

func TestRoasterDedup_FuzzyNameMerge(t *testing.T) {
	idx := newTestFeedIndex(t)

	// Same roaster, different name variations, same location
	insertRecord(t, idx, "did:plc:alice", atproto.NSIDRoaster, "r1", map[string]interface{}{
		"name":     "Counter Culture Coffee",
		"location": "Durham, NC",
		"website":  "https://counterculturecoffee.com",
	})
	insertRecord(t, idx, "did:plc:bob", atproto.NSIDRoaster, "r2", map[string]interface{}{
		"name":     "Counter Culture Coffee Roasters",
		"location": "Durham, NC",
	})

	results, err := Search(idx, atproto.NSIDRoaster, "counter", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1, "fuzzy name + same location should merge")
	assert.Equal(t, 2, results[0].Count)
}

func TestRoasterDedup_NoLocationMerges(t *testing.T) {
	idx := newTestFeedIndex(t)

	// Both have no location — should merge on fuzzy name alone
	insertRecord(t, idx, "did:plc:alice", atproto.NSIDRoaster, "r1", map[string]interface{}{
		"name": "Blue Bottle Coffee",
	})
	insertRecord(t, idx, "did:plc:bob", atproto.NSIDRoaster, "r2", map[string]interface{}{
		"name": "Blue Bottle",
	})

	results, err := Search(idx, atproto.NSIDRoaster, "blue", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1, "same fuzzy name with no location should merge")
	assert.Equal(t, 2, results[0].Count)
}

// --- Grinder dedup tests ---

func TestGrinderDedup_SameNameDifferentType(t *testing.T) {
	idx := newTestFeedIndex(t)

	insertRecord(t, idx, "did:plc:alice", atproto.NSIDGrinder, "g1", map[string]interface{}{
		"name":        "Baratza Encore",
		"grinderType": "electric",
		"burrType":    "conical",
	})
	insertRecord(t, idx, "did:plc:bob", atproto.NSIDGrinder, "g2", map[string]interface{}{
		"name":        "Baratza Encore",
		"grinderType": "electric",
		"burrType":    "flat",
	})

	results, err := Search(idx, atproto.NSIDGrinder, "baratza", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2, "different burr types should produce separate suggestions")
}

func TestGrinderDedup_SameEverythingMerges(t *testing.T) {
	idx := newTestFeedIndex(t)

	insertRecord(t, idx, "did:plc:alice", atproto.NSIDGrinder, "g1", map[string]interface{}{
		"name":        "1Zpresso JX Pro",
		"grinderType": "hand",
		"burrType":    "conical",
	})
	insertRecord(t, idx, "did:plc:bob", atproto.NSIDGrinder, "g2", map[string]interface{}{
		"name":        "1Zpresso JX Pro",
		"grinderType": "hand",
		"burrType":    "conical",
	})

	results, err := Search(idx, atproto.NSIDGrinder, "1zp", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 2, results[0].Count)
}

// --- Brewer dedup tests ---

func TestBrewerDedup_SameNameDifferentType(t *testing.T) {
	idx := newTestFeedIndex(t)

	insertRecord(t, idx, "did:plc:alice", atproto.NSIDBrewer, "br1", map[string]interface{}{
		"name":       "Hario V60",
		"brewerType": "pour-over",
	})
	insertRecord(t, idx, "did:plc:bob", atproto.NSIDBrewer, "br2", map[string]interface{}{
		"name":       "Hario V60",
		"brewerType": "dripper",
	})

	results, err := Search(idx, atproto.NSIDBrewer, "hario", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2, "different brewer types should produce separate suggestions")
}

func TestBrewerDedup_SameNameSameTypeMerges(t *testing.T) {
	idx := newTestFeedIndex(t)

	insertRecord(t, idx, "did:plc:alice", atproto.NSIDBrewer, "br1", map[string]interface{}{
		"name":       "AeroPress",
		"brewerType": "immersion",
	})
	insertRecord(t, idx, "did:plc:bob", atproto.NSIDBrewer, "br2", map[string]interface{}{
		"name":       "AeroPress",
		"brewerType": "immersion",
	})

	results, err := Search(idx, atproto.NSIDBrewer, "aero", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 2, results[0].Count)
}

// --- Bean dedup tests ---

func TestBeanDedup_SameNameDifferentProcess(t *testing.T) {
	idx := newTestFeedIndex(t)

	insertRecord(t, idx, "did:plc:alice", atproto.NSIDBean, "b1", map[string]interface{}{
		"name":    "Yirgacheffe",
		"origin":  "Ethiopia",
		"process": "Washed",
	})
	insertRecord(t, idx, "did:plc:bob", atproto.NSIDBean, "b2", map[string]interface{}{
		"name":    "Yirgacheffe",
		"origin":  "Ethiopia",
		"process": "Natural",
	})

	results, err := Search(idx, atproto.NSIDBean, "yirga", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2, "different processes should produce separate suggestions")
}

func TestBeanDedup_SameNameDifferentOrigin(t *testing.T) {
	idx := newTestFeedIndex(t)

	insertRecord(t, idx, "did:plc:alice", atproto.NSIDBean, "b1", map[string]interface{}{
		"name":   "Gesha",
		"origin": "Panama",
	})
	insertRecord(t, idx, "did:plc:bob", atproto.NSIDBean, "b2", map[string]interface{}{
		"name":   "Gesha",
		"origin": "Ethiopia",
	})

	results, err := Search(idx, atproto.NSIDBean, "gesha", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2, "different origins should produce separate suggestions")
}

func TestBeanDedup_SameEverythingMerges(t *testing.T) {
	idx := newTestFeedIndex(t)

	insertRecord(t, idx, "did:plc:alice", atproto.NSIDBean, "b1", map[string]interface{}{
		"name":       "Ethiopian Yirgacheffe",
		"origin":     "Ethiopia",
		"roastLevel": "Light",
		"process":    "Washed",
	})
	insertRecord(t, idx, "did:plc:bob", atproto.NSIDBean, "b2", map[string]interface{}{
		"name":    "Ethiopian Yirgacheffe",
		"origin":  "Ethiopia",
		"process": "Washed",
	})

	results, err := Search(idx, atproto.NSIDBean, "ethiopia", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 2, results[0].Count)
}

// --- General search tests ---

func TestSearch_PrefixMatch(t *testing.T) {
	idx := newTestFeedIndex(t)

	insertRecord(t, idx, "did:plc:alice", atproto.NSIDRoaster, "r1", map[string]interface{}{
		"name":     "Black & White Coffee",
		"location": "Raleigh, NC",
	})
	insertRecord(t, idx, "did:plc:bob", atproto.NSIDRoaster, "r2", map[string]interface{}{
		"name":     "Blue Bottle",
		"location": "Oakland, CA",
	})

	results, err := Search(idx, atproto.NSIDRoaster, "bl", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Black & White Coffee", results[0].Name)
	assert.Equal(t, "Blue Bottle", results[1].Name)
}

func TestSearch_CaseInsensitive(t *testing.T) {
	idx := newTestFeedIndex(t)

	insertRecord(t, idx, "did:plc:alice", atproto.NSIDRoaster, "r1", map[string]interface{}{
		"name": "Stumptown Coffee",
	})

	results, err := Search(idx, atproto.NSIDRoaster, "STUMP", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Stumptown Coffee", results[0].Name)
}

func TestSearch_SubstringMatch(t *testing.T) {
	idx := newTestFeedIndex(t)

	insertRecord(t, idx, "did:plc:alice", atproto.NSIDRoaster, "r1", map[string]interface{}{
		"name":     "Red Rooster Coffee",
		"location": "Floyd, VA",
	})

	results, err := Search(idx, atproto.NSIDRoaster, "floyd", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Red Rooster Coffee", results[0].Name)
}

func TestSearch_Deduplication(t *testing.T) {
	idx := newTestFeedIndex(t)

	// Two users have the same roaster, same location (one with website, one without)
	insertRecord(t, idx, "did:plc:alice", atproto.NSIDRoaster, "r1", map[string]interface{}{
		"name":     "Counter Culture Coffee",
		"location": "Durham, NC",
		"website":  "https://counterculturecoffee.com",
	})
	insertRecord(t, idx, "did:plc:bob", atproto.NSIDRoaster, "r2", map[string]interface{}{
		"name":     "Counter Culture",
		"location": "Durham, NC",
	})

	results, err := Search(idx, atproto.NSIDRoaster, "counter", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 2, results[0].Count)
	assert.Equal(t, "Durham, NC", results[0].Fields["location"])
}

func TestSearch_Limit(t *testing.T) {
	idx := newTestFeedIndex(t)

	for i := 0; i < 5; i++ {
		rkey := "r" + string(rune('0'+i))
		insertRecord(t, idx, "did:plc:alice", atproto.NSIDGrinder, rkey, map[string]interface{}{
			"name":        "Grinder " + string(rune('A'+i)),
			"grinderType": "hand",
		})
	}

	results, err := Search(idx, atproto.NSIDGrinder, "grinder", 3)
	assert.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestSearch_ShortQuery(t *testing.T) {
	idx := newTestFeedIndex(t)

	insertRecord(t, idx, "did:plc:alice", atproto.NSIDRoaster, "r1", map[string]interface{}{
		"name": "ABC",
	})

	results, err := Search(idx, atproto.NSIDRoaster, "a", 10)
	assert.NoError(t, err)
	assert.Empty(t, results)

	results, err = Search(idx, atproto.NSIDRoaster, "ab", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestSearch_EmptyQuery(t *testing.T) {
	idx := newTestFeedIndex(t)

	results, err := Search(idx, atproto.NSIDRoaster, "", 10)
	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearch_UnknownCollection(t *testing.T) {
	idx := newTestFeedIndex(t)

	results, err := Search(idx, "unknown.collection", "test", 10)
	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearch_GrinderFields(t *testing.T) {
	idx := newTestFeedIndex(t)

	insertRecord(t, idx, "did:plc:alice", atproto.NSIDGrinder, "g1", map[string]interface{}{
		"name":        "1Zpresso JX Pro",
		"grinderType": "hand",
		"burrType":    "conical",
	})

	results, err := Search(idx, atproto.NSIDGrinder, "1zp", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "hand", results[0].Fields["grinderType"])
	assert.Equal(t, "conical", results[0].Fields["burrType"])
}

func TestSearch_BeanFields(t *testing.T) {
	idx := newTestFeedIndex(t)

	insertRecord(t, idx, "did:plc:alice", atproto.NSIDBean, "b1", map[string]interface{}{
		"name":       "Ethiopian Yirgacheffe",
		"origin":     "Ethiopia",
		"roastLevel": "Light",
		"process":    "Washed",
	})

	results, err := Search(idx, atproto.NSIDBean, "ethiopia", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Ethiopian Yirgacheffe", results[0].Name)
	assert.Equal(t, "Light", results[0].Fields["roastLevel"])
}

func TestSearch_BrewerFields(t *testing.T) {
	idx := newTestFeedIndex(t)

	insertRecord(t, idx, "did:plc:alice", atproto.NSIDBrewer, "br1", map[string]interface{}{
		"name":       "Hario V60",
		"brewerType": "Pour-Over",
	})

	results, err := Search(idx, atproto.NSIDBrewer, "hario", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Pour-Over", results[0].Fields["brewerType"])
}

func TestSearch_SortOrder(t *testing.T) {
	idx := newTestFeedIndex(t)

	// "Alpha Roasters" used by 3 people (same location so they merge)
	insertRecord(t, idx, "did:plc:alice", atproto.NSIDRoaster, "r1", map[string]interface{}{"name": "Alpha Roasters"})
	insertRecord(t, idx, "did:plc:bob", atproto.NSIDRoaster, "r2", map[string]interface{}{"name": "Alpha Roasters"})
	insertRecord(t, idx, "did:plc:charlie", atproto.NSIDRoaster, "r3", map[string]interface{}{"name": "Alpha Roasters"})

	// "Alpha Beta" used by 1 person
	insertRecord(t, idx, "did:plc:dave", atproto.NSIDRoaster, "r4", map[string]interface{}{"name": "Alpha Beta"})

	results, err := Search(idx, atproto.NSIDRoaster, "alpha", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Alpha Roasters", results[0].Name)
	assert.Equal(t, 3, results[0].Count)
	assert.Equal(t, "Alpha Beta", results[1].Name)
	assert.Equal(t, 1, results[1].Count)
}
