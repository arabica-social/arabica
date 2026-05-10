package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLikeRoundTrip(t *testing.T) {
	original := &Like{
		SubjectURI: "at://did:plc:author/social.oolong.alpha.tea/tea1",
		SubjectCID: "bafyreig...",
		CreatedAt:  time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
	}
	rec, err := LikeToRecord(original)
	require.NoError(t, err)
	shutter.Snap(t, "LikeToRecord/full like", rec)

	round, err := RecordToLike(rec, "at://did:plc:test/social.oolong.alpha.like/like1")
	require.NoError(t, err)
	assert.Equal(t, "like1", round.RKey)
	assert.Equal(t, original.SubjectURI, round.SubjectURI)
	assert.Equal(t, original.SubjectCID, round.SubjectCID)
}

func TestCommentRoundTrip(t *testing.T) {
	original := &Comment{
		SubjectURI: "at://did:plc:author/social.oolong.alpha.tea/tea1",
		SubjectCID: "bafyreig...",
		Text:       "Beautiful first steep",
		CreatedAt:  time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
	}
	rec, err := CommentToRecord(original)
	require.NoError(t, err)
	shutter.Snap(t, "CommentToRecord/full comment", rec)

	round, err := RecordToComment(rec, "at://did:plc:test/social.oolong.alpha.comment/c1")
	require.NoError(t, err)
	assert.Equal(t, "c1", round.RKey)
	assert.Equal(t, original.Text, round.Text)
}

func TestCommentWithParent(t *testing.T) {
	original := &Comment{
		SubjectURI: "at://did:plc:author/social.oolong.alpha.tea/tea1",
		SubjectCID: "bafyreig...",
		Text:       "Reply",
		ParentURI:  "at://did:plc:author/social.oolong.alpha.comment/parent",
		ParentCID:  "bafyreig...",
		CreatedAt:  time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
	}
	rec, err := CommentToRecord(original)
	require.NoError(t, err)
	shutter.Snap(t, "CommentToRecord/with parent", rec)
}
