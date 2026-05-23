package coffeehandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/firehose"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/moderation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type exploreFilterSource struct{ hidden []string }

func (s exploreFilterSource) ListHiddenURIs(ctx context.Context) ([]string, error) {
	return s.hidden, nil
}

func (s exploreFilterSource) ListBlacklistedDIDs(ctx context.Context) ([]string, error) {
	return nil, nil
}

func TestHandleExploreRequiresAuthentication(t *testing.T) {
	tc := NewTestContext()
	req := NewUnauthenticatedRequest(http.MethodGet, "/explore")
	rec := httptest.NewRecorder()

	tc.Handler.HandleExplore(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestParseExploreQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/explore?type=bean&q=ethiopia&sort=popular&cursor=abc&origin=Ethiopia&min_rating=8&unknown=x", nil)

	query := parseExploreQuery(req)

	assert.Equal(t, "arabica", query.App)
	assert.Equal(t, lexicons.RecordTypeBean, query.Type)
	assert.Equal(t, "ethiopia", query.Q)
	assert.Equal(t, "popular", query.Sort)
	assert.Equal(t, "abc", query.Cursor)
	assert.Equal(t, 20, query.Limit)
	assert.Equal(t, "Ethiopia", query.Filters["origin"])
	assert.Equal(t, "8", query.Filters["min_rating"])
	assert.Empty(t, query.Filters["unknown"])
}

func TestGetModeratedExploreOverfetchesHiddenRecords(t *testing.T) {
	idx, err := firehose.NewFeedIndex(t.TempDir()+"/test.db", time.Hour)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, idx.Close()) })

	did := "did:plc:explore"
	display := "Explore User"
	idx.StoreProfile(context.Background(), did, &atproto.Profile{DID: did, Handle: "explore.test", DisplayName: &display})
	var hidden []string
	for i := 0; i < 40; i++ {
		createdAt := time.Date(2026, 5, 23, 12, 0, i, 0, time.UTC).Format(time.RFC3339)
		record, err := json.Marshal(map[string]any{"$type": arabica.NSIDBean, "name": fmt.Sprintf("Bean %02d", i), "createdAt": createdAt})
		require.NoError(t, err)
		rkey := fmt.Sprintf("b%02d", i)
		require.NoError(t, idx.UpsertRecord(context.Background(), did, arabica.NSIDBean, rkey, "cid", record, time.Now().Unix()))
		if i >= 20 {
			hidden = append(hidden, fmt.Sprintf("at://%s/%s/%s", did, arabica.NSIDBean, rkey))
		}
	}

	cf, err := moderation.LoadFilter(context.Background(), exploreFilterSource{hidden: hidden})
	require.NoError(t, err)
	tc := NewTestContext()
	tc.Handler.SetFeedIndex(idx)
	req := httptest.NewRequest(http.MethodGet, "/explore", nil)

	res, err := tc.Handler.getModeratedExplore(req, firehose.ExploreQuery{App: "arabica", Type: lexicons.RecordTypeBean, Limit: 20}, cf)
	require.NoError(t, err)
	require.Len(t, res.Items, 20)
	for _, item := range res.Items {
		assert.False(t, cf.ShouldHide(item.SubjectURI, item.Author.DID))
	}
}
