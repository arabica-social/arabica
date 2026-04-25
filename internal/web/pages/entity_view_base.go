package pages

import "tangled.org/arabica.social/arabica/internal/firehose"

// EntityViewBase holds the social and auth fields shared by all simple entity
// view pages (bean, roaster, grinder, brewer). Embed this in XxxViewProps.
// Fields are promoted so templ pages access them as props.IsAuthenticated etc.
type EntityViewBase struct {
	IsOwnProfile      bool
	IsAuthenticated   bool
	SubjectURI        string
	SubjectCID        string
	IsLiked           bool
	LikeCount         int
	CommentCount      int
	Comments          []firehose.IndexedComment
	CurrentUserDID    string
	ShareURL          string
	IsModerator       bool
	CanHideRecord     bool
	CanBlockUser      bool
	IsRecordHidden    bool
	AuthorDID         string
	AuthorHandle      string
	AuthorDisplayName string
	AuthorAvatar      string
}
