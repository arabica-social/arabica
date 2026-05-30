package backlinks

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"tangled.org/arabica.social/arabica/internal/suggestions"
	"tangled.org/pdewey.com/atp"
)

const (
	testBean = "social.test.bean"
	testBrew = "social.test.brew"
)

type fakeSource struct{ records []IndexedRecord }

func (f fakeSource) ListSourceRefChain(_ context.Context, uri string, maxDepth, maxRecords int) ([]IndexedRecord, error) {
	seen := map[string]struct{}{uri: {}}
	frontier := []string{uri}
	var out []IndexedRecord
	for depth := 0; depth < maxDepth && len(frontier) > 0 && len(out) < maxRecords; depth++ {
		var next []string
		for _, u := range frontier {
			for _, rec := range f.records {
				if sourceRefOf(rec.Record) != u {
					continue
				}
				if _, ok := seen[rec.URI]; ok {
					continue
				}
				seen[rec.URI] = struct{}{}
				out = append(out, rec)
				next = append(next, rec.URI)
			}
		}
		frontier = next
	}
	return out, nil
}

func (f fakeSource) ListRecordsByCollectionOldest(_ context.Context, collection string) ([]IndexedRecord, error) {
	var out []IndexedRecord
	for _, rec := range f.records {
		if rec.Collection == collection {
			out = append(out, rec)
		}
	}
	return out, nil
}

func (f fakeSource) ListUsageBacklinks(_ context.Context, uri, fromCollection, fieldName string) ([]IndexedRecord, error) {
	var out []IndexedRecord
	for _, rec := range f.records {
		if rec.Collection != fromCollection {
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(rec.Record, &data); err != nil {
			continue
		}
		if data[fieldName] == uri {
			out = append(out, rec)
		}
	}
	return out, nil
}

func (f fakeSource) ListUsageBacklinksPage(ctx context.Context, uri, fromCollection, fieldName string, limit, offset int) ([]IndexedRecord, int, error) {
	all, err := f.ListUsageBacklinks(ctx, uri, fromCollection, fieldName)
	if err != nil {
		return nil, 0, err
	}
	if offset >= len(all) {
		return nil, len(all), nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], len(all), nil
}

func (f fakeSource) GetRecord(_ context.Context, uri string) (IndexedRecord, bool) {
	for _, rec := range f.records {
		if rec.URI == uri {
			return rec, true
		}
	}
	return IndexedRecord{}, false
}

type noProfiles struct{}

func (noProfiles) GetProfile(context.Context, string) (*Profile, error) {
	return nil, assert.AnError
}

func TestLookupFindsSourceRefChainFuzzyMatchesAndUsage(t *testing.T) {
	Register(testBean, EntityConfig{
		AllFields: []string{"name", "origin"},
		DedupKey: func(fields map[string]string) string {
			return suggestions.Normalize(fields["name"]) + "|" + suggestions.Normalize(fields["origin"])
		},
		UsageRefs: []UsageRef{{Collection: testBrew, Field: "beanRef", Label: "brews"}},
	})

	root := testRecord(t, "did:plc:alice", testBean, "b1", 1, map[string]any{"name": "Gesha", "origin": "Panama"})
	copy := testRecord(t, "did:plc:bob", testBean, "b2", 2, map[string]any{"name": "Gesha", "origin": "Panama", "sourceRef": root.URI})
	copyOfCopy := testRecord(t, "did:plc:carol", testBean, "b3", 3, map[string]any{"name": "Gesha", "origin": "Panama", "sourceRef": copy.URI})
	fuzzy := testRecord(t, "did:plc:dana", testBean, "b4", 4, map[string]any{"name": " gesha ", "origin": "panama"})
	brew := testRecord(t, "did:plc:erin", testBrew, "br1", 5, map[string]any{"beanRef": root.URI})

	res, err := NewService(fakeSource{records: []IndexedRecord{root, copy, copyOfCopy, fuzzy, brew}}, noProfiles{}).Lookup(context.Background(), root.URI)

	assert.NoError(t, err)
	assert.Equal(t, 3, res.LibraryCount)
	assert.Equal(t, 1, res.UsageCount)
	assert.Len(t, res.Usage, 1)
	assert.Equal(t, "brews", res.Usage[0].Label)
	assert.Equal(t, "Gesha", res.Usage[0].Entries[0].Title)

	depths := map[string]int{}
	for _, e := range res.LibraryEntries {
		depths[e.DID] = e.ChainDepth
	}
	assert.Equal(t, 1, depths["did:plc:bob"])
	assert.Equal(t, 2, depths["did:plc:carol"])
	assert.Equal(t, 0, depths["did:plc:dana"])
}

func TestLookupPaginatesSelectedUsageGroup(t *testing.T) {
	Register(testBean, EntityConfig{
		AllFields: []string{"name", "origin"},
		DedupKey:  func(fields map[string]string) string { return suggestions.Normalize(fields["name"]) },
		UsageRefs: []UsageRef{{Collection: testBrew, Field: "beanRef", Label: "brews"}},
	})
	root := testRecord(t, "did:plc:alice", testBean, "b1", 1, map[string]any{"name": "Gesha"})
	recs := []IndexedRecord{root}
	for i := 0; i < 30; i++ {
		recs = append(recs, testRecord(t, "did:plc:user", testBrew, fmt.Sprintf("br%d", i), 100-i, map[string]any{"beanRef": root.URI}))
	}

	res, err := NewService(fakeSource{records: recs}, noProfiles{}).LookupWithOptions(context.Background(), root.URI, LookupOptions{UsageKey: "brews", UsagePage: 2, UsagePerPage: 25})

	assert.NoError(t, err)
	assert.Len(t, res.Usage, 1)
	assert.Equal(t, 30, res.Usage[0].Count)
	assert.Len(t, res.Usage[0].Entries, 5)
	assert.True(t, res.Usage[0].HasPrev)
	assert.False(t, res.Usage[0].HasNext)
}

func TestLookupReturnsEmptyForUnknownCollection(t *testing.T) {
	res, err := NewService(fakeSource{}, noProfiles{}).Lookup(context.Background(), "at://did:plc:alice/social.test.unknown/r1")

	assert.NoError(t, err)
	assert.True(t, res.IsEmpty())
}

func testRecord(t *testing.T, did, collection, rkey string, seconds int, fields map[string]any) IndexedRecord {
	t.Helper()
	raw, err := json.Marshal(fields)
	assert.NoError(t, err)
	return IndexedRecord{
		URI:        atp.BuildATURI(did, collection, rkey),
		DID:        did,
		Collection: collection,
		RKey:       rkey,
		Record:     raw,
		CreatedAt:  time.Unix(int64(seconds), 0),
	}
}
