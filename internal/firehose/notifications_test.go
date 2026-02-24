package firehose

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"arabica/internal/models"

	"github.com/stretchr/testify/assert"
)

func newTestIndex(t *testing.T) *FeedIndex {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test-index.db")
	idx, err := NewFeedIndex(path, time.Hour)
	assert.NoError(t, err)
	t.Cleanup(func() {
		idx.Close()
		os.Remove(path)
	})
	return idx
}

func TestCreateNotification(t *testing.T) {
	idx := newTestIndex(t)

	targetDID := "did:plc:target123"
	actorDID := "did:plc:actor456"
	subjectURI := "at://did:plc:target123/social.arabica.alpha.brew/abc"

	notif := models.Notification{
		Type:       models.NotificationLike,
		ActorDID:   actorDID,
		SubjectURI: subjectURI,
		CreatedAt:  time.Now(),
	}

	err := idx.CreateNotification(targetDID, notif)
	assert.NoError(t, err)

	// Verify it was created
	notifications, _, err := idx.GetNotifications(targetDID, 10, "")
	assert.NoError(t, err)
	assert.Len(t, notifications, 1)
	assert.Equal(t, models.NotificationLike, notifications[0].Type)
	assert.Equal(t, actorDID, notifications[0].ActorDID)
	assert.Equal(t, subjectURI, notifications[0].SubjectURI)
}

func TestCreateNotification_SkipsSelfNotification(t *testing.T) {
	idx := newTestIndex(t)

	selfDID := "did:plc:self123"

	notif := models.Notification{
		Type:       models.NotificationLike,
		ActorDID:   selfDID,
		SubjectURI: "at://did:plc:self123/social.arabica.alpha.brew/abc",
		CreatedAt:  time.Now(),
	}

	err := idx.CreateNotification(selfDID, notif)
	assert.NoError(t, err)

	notifications, _, err := idx.GetNotifications(selfDID, 10, "")
	assert.NoError(t, err)
	assert.Empty(t, notifications)
}

func TestCreateNotification_Deduplication(t *testing.T) {
	idx := newTestIndex(t)

	targetDID := "did:plc:target123"
	notif := models.Notification{
		Type:       models.NotificationLike,
		ActorDID:   "did:plc:actor456",
		SubjectURI: "at://did:plc:target123/social.arabica.alpha.brew/abc",
		CreatedAt:  time.Now(),
	}

	// Create the same notification twice
	assert.NoError(t, idx.CreateNotification(targetDID, notif))
	assert.NoError(t, idx.CreateNotification(targetDID, notif))

	notifications, _, err := idx.GetNotifications(targetDID, 10, "")
	assert.NoError(t, err)
	assert.Len(t, notifications, 1) // should be deduplicated
}

func TestGetUnreadCount(t *testing.T) {
	idx := newTestIndex(t)

	targetDID := "did:plc:target123"
	baseTime := time.Now().Add(-time.Minute)

	// Initially zero
	assert.Equal(t, 0, idx.GetUnreadCount(targetDID))

	// Add some notifications
	for i := 0; i < 3; i++ {
		notif := models.Notification{
			Type:       models.NotificationLike,
			ActorDID:   "did:plc:actor" + string(rune('a'+i)),
			SubjectURI: "at://did:plc:target123/social.arabica.alpha.brew/abc",
			CreatedAt:  baseTime.Add(time.Duration(i) * time.Second),
		}
		assert.NoError(t, idx.CreateNotification(targetDID, notif))
	}

	assert.Equal(t, 3, idx.GetUnreadCount(targetDID))
}

func TestMarkAllRead(t *testing.T) {
	idx := newTestIndex(t)

	targetDID := "did:plc:target123"
	baseTime := time.Now().Add(-time.Minute) // use past times to avoid race

	// Add notifications
	for i := 0; i < 3; i++ {
		notif := models.Notification{
			Type:       models.NotificationLike,
			ActorDID:   "did:plc:actor" + string(rune('a'+i)),
			SubjectURI: "at://did:plc:target123/social.arabica.alpha.brew/abc",
			CreatedAt:  baseTime.Add(time.Duration(i) * time.Second),
		}
		assert.NoError(t, idx.CreateNotification(targetDID, notif))
	}

	assert.Equal(t, 3, idx.GetUnreadCount(targetDID))

	// Mark all as read
	assert.NoError(t, idx.MarkAllRead(targetDID))
	assert.Equal(t, 0, idx.GetUnreadCount(targetDID))

	// Notifications still exist, but are marked as read
	notifications, _, err := idx.GetNotifications(targetDID, 10, "")
	assert.NoError(t, err)
	assert.Len(t, notifications, 3)
	for _, n := range notifications {
		assert.True(t, n.Read)
	}
}

func TestGetNotifications_Pagination(t *testing.T) {
	idx := newTestIndex(t)

	targetDID := "did:plc:target123"
	baseTime := time.Now().Add(-time.Minute)

	// Add 5 notifications
	for i := 0; i < 5; i++ {
		notif := models.Notification{
			Type:       models.NotificationLike,
			ActorDID:   "did:plc:actor" + string(rune('a'+i)),
			SubjectURI: "at://did:plc:target123/social.arabica.alpha.brew/abc",
			CreatedAt:  baseTime.Add(time.Duration(i) * time.Second),
		}
		assert.NoError(t, idx.CreateNotification(targetDID, notif))
	}

	// Get first page of 3
	page1, cursor1, err := idx.GetNotifications(targetDID, 3, "")
	assert.NoError(t, err)
	assert.Len(t, page1, 3)
	assert.NotEmpty(t, cursor1)

	// Get second page
	page2, cursor2, err := idx.GetNotifications(targetDID, 3, cursor1)
	assert.NoError(t, err)
	assert.Len(t, page2, 2)
	assert.Empty(t, cursor2)
}

func TestParseTargetDID(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "valid brew URI",
			uri:      "at://did:plc:abc123/social.arabica.alpha.brew/xyz",
			expected: "did:plc:abc123",
		},
		{
			name:     "valid like URI",
			uri:      "at://did:web:example.com/social.arabica.alpha.like/xyz",
			expected: "did:web:example.com",
		},
		{
			name:     "empty string",
			uri:      "",
			expected: "",
		},
		{
			name:     "not an AT URI",
			uri:      "https://example.com/something",
			expected: "",
		},
		{
			name:     "AT URI without did prefix",
			uri:      "at://handle.example.com/collection/rkey",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTargetDID(tt.uri)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateLikeNotification(t *testing.T) {
	idx := newTestIndex(t)

	actorDID := "did:plc:actor456"
	subjectURI := "at://did:plc:target123/social.arabica.alpha.brew/abc"

	idx.CreateLikeNotification(actorDID, subjectURI)

	notifications, _, err := idx.GetNotifications("did:plc:target123", 10, "")
	assert.NoError(t, err)
	assert.Len(t, notifications, 1)
	assert.Equal(t, models.NotificationLike, notifications[0].Type)
}

func TestCreateLikeNotification_SkipsSelf(t *testing.T) {
	idx := newTestIndex(t)

	selfDID := "did:plc:target123"
	subjectURI := "at://did:plc:target123/social.arabica.alpha.brew/abc"

	idx.CreateLikeNotification(selfDID, subjectURI)

	notifications, _, err := idx.GetNotifications("did:plc:target123", 10, "")
	assert.NoError(t, err)
	assert.Empty(t, notifications)
}

func TestCreateCommentNotification(t *testing.T) {
	idx := newTestIndex(t)

	actorDID := "did:plc:actor456"
	subjectURI := "at://did:plc:target123/social.arabica.alpha.brew/abc"

	idx.CreateCommentNotification(actorDID, subjectURI, "")

	notifications, _, err := idx.GetNotifications("did:plc:target123", 10, "")
	assert.NoError(t, err)
	assert.Len(t, notifications, 1)
	assert.Equal(t, models.NotificationComment, notifications[0].Type)
}

func TestCreateCommentNotification_WithReply(t *testing.T) {
	idx := newTestIndex(t)

	actorDID := "did:plc:actor456"
	subjectURI := "at://did:plc:brewowner/social.arabica.alpha.brew/abc"
	parentURI := "at://did:plc:commenter789/social.arabica.alpha.comment/xyz"

	idx.CreateCommentNotification(actorDID, subjectURI, parentURI)

	// Brew owner gets a comment notification
	brewOwnerNotifs, _, err := idx.GetNotifications("did:plc:brewowner", 10, "")
	assert.NoError(t, err)
	assert.Len(t, brewOwnerNotifs, 1)
	assert.Equal(t, models.NotificationComment, brewOwnerNotifs[0].Type)

	// Parent comment author gets a reply notification with the brew URI (not the parent comment URI)
	commenterNotifs, _, err := idx.GetNotifications("did:plc:commenter789", 10, "")
	assert.NoError(t, err)
	assert.Len(t, commenterNotifs, 1)
	assert.Equal(t, models.NotificationCommentReply, commenterNotifs[0].Type)
	assert.Equal(t, subjectURI, commenterNotifs[0].SubjectURI)
}
