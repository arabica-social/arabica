package moderation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFilterSource implements FilterSource for testing
type mockFilterSource struct {
	hiddenURIs      []string
	blacklistedDIDs []string
	hiddenErr       error
	blacklistErr    error
}

func (m *mockFilterSource) ListHiddenURIs(ctx context.Context) ([]string, error) {
	return m.hiddenURIs, m.hiddenErr
}

func (m *mockFilterSource) ListBlacklistedDIDs(ctx context.Context) ([]string, error) {
	return m.blacklistedDIDs, m.blacklistErr
}

func TestLoadFilter(t *testing.T) {
	ctx := context.Background()
	src := &mockFilterSource{
		hiddenURIs:      []string{"at://did:plc:a/col/1", "at://did:plc:b/col/2"},
		blacklistedDIDs: []string{"did:plc:bad"},
	}

	f, err := LoadFilter(ctx, src)
	require.NoError(t, err)
	assert.NotNil(t, f)
}

func TestLoadFilter_NilSource(t *testing.T) {
	f, err := LoadFilter(context.Background(), nil)
	require.NoError(t, err)
	assert.NotNil(t, f)
	// Empty filter should hide nothing
	assert.False(t, f.ShouldHide("at://anything", "did:plc:anyone"))
}

func TestShouldHide_HiddenURI(t *testing.T) {
	ctx := context.Background()
	f, _ := LoadFilter(ctx, &mockFilterSource{
		hiddenURIs: []string{"at://did:plc:a/col/1"},
	})

	assert.True(t, f.ShouldHide("at://did:plc:a/col/1", ""))
	assert.False(t, f.ShouldHide("at://did:plc:a/col/2", ""))
}

func TestShouldHide_BlacklistedAuthor(t *testing.T) {
	ctx := context.Background()
	f, _ := LoadFilter(ctx, &mockFilterSource{
		blacklistedDIDs: []string{"did:plc:bad"},
	})

	assert.True(t, f.ShouldHide("", "did:plc:bad"))
	assert.False(t, f.ShouldHide("", "did:plc:good"))
}

func TestShouldHide_BothEmpty(t *testing.T) {
	ctx := context.Background()
	f, _ := LoadFilter(ctx, &mockFilterSource{})

	assert.False(t, f.ShouldHide("at://anything", "did:plc:anyone"))
}

func TestIsBlocked(t *testing.T) {
	ctx := context.Background()
	f, _ := LoadFilter(ctx, &mockFilterSource{
		blacklistedDIDs: []string{"did:plc:bad"},
	})

	assert.True(t, f.IsBlocked("did:plc:bad"))
	assert.False(t, f.IsBlocked("did:plc:good"))
}

func TestFilterSlice(t *testing.T) {
	type item struct {
		uri       string
		authorDID string
		name      string
	}

	ctx := context.Background()
	f, _ := LoadFilter(ctx, &mockFilterSource{
		hiddenURIs:      []string{"at://did:plc:a/col/hidden"},
		blacklistedDIDs: []string{"did:plc:bad"},
	})

	items := []*item{
		{uri: "at://did:plc:a/col/ok", authorDID: "did:plc:good", name: "visible"},
		{uri: "at://did:plc:a/col/hidden", authorDID: "did:plc:good", name: "hidden-record"},
		{uri: "at://did:plc:b/col/ok", authorDID: "did:plc:bad", name: "blocked-author"},
		{uri: "at://did:plc:c/col/ok", authorDID: "did:plc:nice", name: "also-visible"},
	}

	result := FilterSlice(f, items, func(i *item) (string, string) {
		return i.uri, i.authorDID
	})

	assert.Len(t, result, 2)
	assert.Equal(t, "visible", result[0].name)
	assert.Equal(t, "also-visible", result[1].name)
}

func TestFilterSlice_NilFilter(t *testing.T) {
	type item struct{ name string }
	items := []*item{{name: "a"}, {name: "b"}}

	result := FilterSlice[*item](nil, items, func(i *item) (string, string) {
		return "", ""
	})

	assert.Len(t, result, 2)
}

func TestLoadFilter_SourceErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("hidden URIs error returns partial filter", func(t *testing.T) {
		f, err := LoadFilter(ctx, &mockFilterSource{
			hiddenErr:       assert.AnError,
			blacklistedDIDs: []string{"did:plc:bad"},
		})
		require.NoError(t, err)
		// Blacklist still works
		assert.True(t, f.IsBlocked("did:plc:bad"))
		// Hidden URIs degraded gracefully
		assert.False(t, f.ShouldHide("at://anything", ""))
	})

	t.Run("blacklist error returns partial filter", func(t *testing.T) {
		f, err := LoadFilter(ctx, &mockFilterSource{
			hiddenURIs:   []string{"at://did:plc:a/col/1"},
			blacklistErr: assert.AnError,
		})
		require.NoError(t, err)
		// Hidden URIs still work
		assert.True(t, f.ShouldHide("at://did:plc:a/col/1", ""))
		// Blacklist degraded gracefully
		assert.False(t, f.IsBlocked("did:plc:bad"))
	})
}
